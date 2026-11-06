package project

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"cube/internal/domain/icons"
	"cube/pkg/x"

	"github.com/heyuuu/x/store"
)

// settings.json 中 project 域的两个规则节：分节存储，可独立读取与写入。
const (
	scanRulesSection  = "scanRules"
	cloneRulesSection = "cloneRules"
)

// loadScanRules 现读 settings.json 的 scanRule 节并转换为生效规则（直读不缓存，改完即生效）：
// 展开 ~/ 为绝对路径，校验目录存在（不存在的规则降级跳过、打日志，不阻断其它规则）。
// store.LoadSection 已把文件级/节级坏数据降级为零值。
//
// group 唯一是写侧约束（见 saveScanRule）；手工编辑产生的重复 group 按坏数据
// 降级跳过后加载的条目——group 是展示身份，重复会让分组合并语义与项目名产生歧义。
func loadScanRules(settingsFile string) []ScanRule {
	specs := store.LoadSectionOrZero[[]ScanRule](settingsFile, scanRulesSection)

	rules := make([]ScanRule, 0, len(specs))
	seenGroups := make(map[string]bool, len(specs))
	for _, r := range specs {
		if !filepath.IsAbs(r.Path) {
			slog.Warn("scan 规则路径配置错误，path 必须为绝对路径", "group", r.Group, "path", r.Path)
			continue
		}
		path := filepath.Clean(r.Path)

		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			slog.Warn("scan 规则路径不存在或非目录，跳过", "group", r.Group, "path", path, "err", err)
			continue
		}
		if seenGroups[r.Group] {
			slog.Warn("scan 规则 group 重复，跳过后加载的规则", "group", r.Group, "path", path)
			continue
		}
		rules = append(rules, ScanRule{Group: r.Group, Path: path, MaxDepth: r.MaxDepth, Icon: r.Icon, Tags: r.Tags})
		seenGroups[r.Group] = true
	}
	return rules
}

// loadCloneRules 现读 settings.json 的 cloneRule 节并转换为生效规则（直读不缓存，改完即生效）：
// 展开 localPath 的 ~/ 为绝对路径（不校验存在——clone 时会自动创建）；相对路径是配置错误，跳过。
func loadCloneRules(settingsFile string) []CloneRule {
	specs := store.LoadSectionOrZero[[]CloneRule](settingsFile, cloneRulesSection)

	var rules []CloneRule
	for _, r := range specs {
		if !filepath.IsAbs(r.LocalPath) {
			slog.Warn("clone 本地路径配置错误, 必须为绝对路径", "localPath", r.LocalPath)
			continue
		}
		localPath := filepath.Clean(r.LocalPath)

		rules = append(rules, CloneRule{RepoHost: r.RepoHost, RepoPrefix: r.RepoPrefix, LocalPath: localPath})
	}
	return rules
}

// --- 写侧（校验收敛在读写边界：加载即校验、校验后才保存，坏数据返回中文错误、落不了文件） ---

// scanRuleKey / cloneRuleKey 规则在节内的唯一键（domain 私有，不出 API）：
// scan 规则键 = path 原串（编辑 path 等价删旧存新），clone 规则键 = host + prefix。
func scanRuleKey(r ScanRule) string { return r.Path }

type cloneRuleID struct{ repoHost, repoPrefix string }

func cloneRuleKey(r CloneRule) cloneRuleID { return cloneRuleID{r.RepoHost, r.RepoPrefix} }

// normalizeScanRule 校验并归一化一条待写入的 scan 规则（save / save-all 共用）。
func normalizeScanRule(rule ScanRule) (ScanRule, error) {
	if rule.Group == "" {
		return rule, fmt.Errorf("scan 规则 group 不得为空")
	}
	if rule.MaxDepth < -1 {
		return rule, fmt.Errorf("scan 规则 maxDepth 不得小于 -1（-1 不限深、0 仅根下一层）: %d", rule.MaxDepth)
	}
	if !filepath.IsAbs(rule.Path) {
		return rule, fmt.Errorf("scan 规则路径必须为绝对路径: path=%s", rule.Path)
	}
	if info, err := os.Stat(rule.Path); err != nil || !info.IsDir() {
		return rule, fmt.Errorf("scan 规则路径不存在或非目录: %s", rule.Path)
	}
	if err := icons.ValidateIcon(rule.Icon); err != nil {
		return rule, fmt.Errorf("scan 规则 %s", err)
	}
	rule.Tags = normalizeTags(rule.Tags)
	return rule, nil
}

// checkScanRulesUnique 整节写入的唯一约束：path 唯一；group 唯一——group 是分组与项目名前缀的
// 展示身份，重复会让分组合并语义与项目名（group:相对路径）产生歧义，写侧拒绝。
func checkScanRulesUnique(rules []ScanRule) error {
	if dup, ok := x.DuplicateKeyed(rules, scanRuleKey); ok {
		return fmt.Errorf("scan 规则 path 重复: %s", dup)
	}
	owner := make(map[string]string, len(rules))
	for _, r := range rules {
		if path, ok := owner[r.Group]; ok {
			return fmt.Errorf("scan 规则 group 已被其它规则占用: group=%s 已用于 %s", r.Group, path)
		}
		owner[r.Group] = r.Path
	}
	return nil
}

// saveScanRule 新增或按 path 替换一条 scan 规则（path 原串是规则唯一键——
// 编辑 path 等价于删旧存新，前端按此语义提交）。
func saveScanRule(settingsFile string, rule ScanRule) error {
	rule, err := normalizeScanRule(rule)
	if err != nil {
		return err
	}
	specs := store.LoadSectionOrZero[[]ScanRule](settingsFile, scanRulesSection)
	// 单条保存只拦与本条冲突的 group（同 path 替换自身除外）：手工编辑遗留的他人间重复
	// 已由读侧降级，不阻断无关规则的保存
	for _, cur := range specs {
		if cur.Group == rule.Group && cur.Path != rule.Path {
			return fmt.Errorf("scan 规则 group 已被其它规则占用: group=%s 已用于 %s", rule.Group, cur.Path)
		}
	}
	specs = x.UpsertKeyed(specs, scanRuleKey, rule)
	return store.SaveSection(settingsFile, scanRulesSection, specs)
}

// saveAllScanRules 以提交清单整节覆盖 scanRule 节（顺序即项目列表展示序）。
// 逐项校验 + 节级唯一约束，任一失败整体不落盘。
func saveAllScanRules(settingsFile string, rules []ScanRule) error {
	normalized := make([]ScanRule, len(rules))
	for i, r := range rules {
		n, err := normalizeScanRule(r)
		if err != nil {
			return err
		}
		normalized[i] = n
	}
	if err := checkScanRulesUnique(normalized); err != nil {
		return err
	}
	return store.SaveSection(settingsFile, scanRulesSection, normalized)
}

// deleteScanRule 按唯一键（path）删除一条 scan 规则；不存在时返回中文错误。
func deleteScanRule(settingsFile string, rule ScanRule) error {
	specs := store.LoadSectionOrZero[[]ScanRule](settingsFile, scanRulesSection)
	rest, removed := x.RemoveKeyed(specs, scanRuleKey, scanRuleKey(rule))
	if !removed {
		return fmt.Errorf("未找到指定 scan 规则: %s", rule.Path)
	}
	return store.SaveSection(settingsFile, scanRulesSection, rest)
}

// normalizeCloneRule 校验一条待写入的 clone 规则（save / save-all 共用）。
func normalizeCloneRule(rule CloneRule) (CloneRule, error) {
	if rule.RepoHost == "" {
		return rule, fmt.Errorf("clone 规则 repoHost 不得为空")
	}
	if rule.RepoPrefix != "" && rule.RepoPrefix[0] != '/' {
		return rule, fmt.Errorf("clone 规则 repoPrefix 须以 / 开头或为空: %s", rule.RepoPrefix)
	}
	if !filepath.IsAbs(rule.LocalPath) {
		return rule, fmt.Errorf("clone 规则 localPath 不合法（须绝对路径）: %s", rule.LocalPath)
	}
	return rule, nil
}

// saveCloneRule 新增或按 host+prefix 替换一条 clone 规则（二者组合是规则唯一键）。
func saveCloneRule(settingsFile string, rule CloneRule) error {
	rule, err := normalizeCloneRule(rule)
	if err != nil {
		return err
	}
	specs := store.LoadSectionOrZero[[]CloneRule](settingsFile, cloneRulesSection)
	specs = x.UpsertKeyed(specs, cloneRuleKey, rule)
	return store.SaveSection(settingsFile, cloneRulesSection, specs)
}

// saveAllCloneRules 以提交清单整节覆盖 cloneRule 节（顺序即展示序；匹配语义按 prefix
// 最长优先，顺序不影响路由结果）。逐项校验 + host+prefix 查重，任一失败整体不落盘。
func saveAllCloneRules(settingsFile string, rules []CloneRule) error {
	normalized := make([]CloneRule, len(rules))
	for i, r := range rules {
		n, err := normalizeCloneRule(r)
		if err != nil {
			return err
		}
		normalized[i] = n
	}
	if dup, ok := x.DuplicateKeyed(normalized, cloneRuleKey); ok {
		return fmt.Errorf("clone 规则重复: %s%s", dup.repoHost, dup.repoPrefix)
	}
	return store.SaveSection(settingsFile, cloneRulesSection, normalized)
}

// deleteCloneRule 按唯一键（host+prefix）删除一条 clone 规则；不存在时返回中文错误。
func deleteCloneRule(settingsFile string, rule CloneRule) error {
	specs := store.LoadSectionOrZero[[]CloneRule](settingsFile, cloneRulesSection)
	rest, removed := x.RemoveKeyed(specs, cloneRuleKey, cloneRuleKey(rule))
	if !removed {
		return fmt.Errorf("未找到指定 clone 规则: %s%s", rule.RepoHost, rule.RepoPrefix)
	}
	return store.SaveSection(settingsFile, cloneRulesSection, rest)
}
