package workspace

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/heyuuu/x/store"
	"github.com/heyuuu/x/xstd/xos"
	"go.yaml.in/yaml/v3"

	"cube/internal/domain/project/cubefile"
)

// Resolve 解析项目根（或 worktree 根）的 workspace 列表，返回相对该根的成员目录。
// 规则来自 cube.json 的 workspaces 节（节缺失走 cubefile 默认规则）。
// 本函数供采集侧调用（读文件系统），读路径不应调用——读侧走快照。
func Resolve(root string) []Workspace {
	return ResolveRules(root, cubefile.LoadWorkspaceRules(root))
}

// ResolveRules 按规则清单逐条解析合并，按 path 去重（先出现的命名保留——
// 默认 <npm>+<pnpm> 双声明同一目录时只留一份）。
func ResolveRules(root string, rules []cubefile.WorkspaceRule) []Workspace {
	result := make([]Workspace, 0, len(rules))
	seen := map[string]bool{}
	for _, rule := range rules {
		for _, w := range resolveRule(root, rule) {
			if !seen[w.Path] {
				seen[w.Path] = true
				result = append(result, w)
			}
		}
	}
	return result
}

// resolveRule 解析单条规则，坏规则跳过（记日志）：path 为特殊规则（读标准
// monorepo 声明文件）或 glob 通配时展开为多成员，否则按精确目录直查（须存在）。
func resolveRule(root string, rule cubefile.WorkspaceRule) []Workspace {
	rel, ok := cleanRulePath(root, rule.Path)
	if !ok {
		return nil
	}

	// 特殊 rule
	switch rule.Path {
	case cubefile.NpmRule:
		return resolvePatterns(root, rule.Name, npmPatterns(root))
	case cubefile.PnpmRule:
		return resolvePatterns(root, rule.Name, pnpmPatterns(root))
	}

	// 模糊匹配
	if hasGlobMeta(rel) {
		return resolvePatterns(root, rule.Name, []string{rel})
	}

	// 精确匹配
	if !xos.IsDir(filepath.Join(root, rel)) {
		slog.Warn("cube.json workspace 条目目录不存在，跳过", "root", root, "path", rule.Path)
		return nil
	}

	return []Workspace{{Name: displayName(rule.Name, rel), Path: rel}}
}

// cleanRulePath 条目 path 的公共校验：非空、相对、不逃逸出根。
// 返回清理后的相对路径；坏路径返回 ok=false（已记日志）。
func cleanRulePath(root, path string) (string, bool) {
	if path == "" {
		slog.Warn("cube.json workspace 条目缺 path，跳过", "root", root)
		return "", false
	}
	if filepath.IsAbs(path) {
		slog.Warn("cube.json workspace 条目 path 须为相对路径，跳过", "root", root, "path", path)
		return "", false
	}
	rel := filepath.Clean(path)
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		slog.Warn("cube.json workspace 条目 path 逃逸出项目根，跳过", "root", root, "path", path)
		return "", false
	}
	return rel, true
}

// hasGlobMeta 判断是否含 glob 通配符（进入通配展开分支而非单目录直查）。
func hasGlobMeta(path string) bool {
	return strings.ContainsAny(path, "*?[")
}

// displayName 精确目录规则的显示名：空 name 回退相对路径本身。
func displayName(name, rel string) string {
	if name == "" {
		return filepath.ToSlash(rel)
	}
	return name
}

// patternName 通配/特殊规则命中的显示名：规则带 name 时作组前缀（name:rel），
// 缺省用相对路径本身——默认规则（<npm>/<pnpm>）不带 name，前导冒号没有意义。
func patternName(name, rel string) string {
	rel = filepath.ToSlash(rel)
	if name == "" {
		return rel
	}
	return name + ":" + rel
}

// maxGlobWalkDepth glob 展开的遍历深度上限：workspace 声明不可能埋得很深，
// 防御性截断超大目录树的遍历成本（** 模式递归任意深）。
const maxGlobWalkDepth = 6

// resolvePatterns 把 glob 模式（apps/*、packages/** 等）展开为根下实际存在的成员。
// 排除隐藏目录与 node_modules（与 scan 的跳过口径一致），深度超限截断。
func resolvePatterns(root string, name string, patterns []string) []Workspace {
	if len(patterns) == 0 {
		return nil
	}
	rootRel := "."
	var workspaces []Workspace
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // 单目录读失败（权限等）跳过，不中断整次解析
		}
		if !d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil || rel == rootRel {
			return nil
		}
		depth := strings.Count(rel, string(filepath.Separator))
		if depth >= maxGlobWalkDepth {
			return fs.SkipDir
		}
		if isGlobDir(rel) {
			return fs.SkipDir // node_modules / .git 等不进 glob 结果也不下钻
		}
		for _, p := range patterns {
			if p == "" {
				continue
			}
			// doublestar 的 "**" 可匹配零段，libs/** 会连 libs 本身也命中——
			// 模式自身的静态前缀是「容器目录」不是成员，排除
			if prefix, ok := strings.CutSuffix(filepath.ToSlash(p), "/**"); ok && filepath.ToSlash(rel) == prefix {
				continue
			}
			if ok, _ := doublestar.Match(filepath.ToSlash(p), filepath.ToSlash(rel)); ok {
				workspaces = append(workspaces, Workspace{Name: patternName(name, rel), Path: rel})
				break
			}
		}
		return nil
	})
	return workspaces
}

// isGlobDir glob 展开明确排除的目录名前缀（与 scan 的跳过口径一致）。
func isGlobDir(rel string) bool {
	base := filepath.Base(rel)
	return strings.HasPrefix(base, ".") || strings.HasPrefix(base, "_") || base == "node_modules"
}

// npmPatterns 读 package.json 的 workspaces 节（数组或 {packages: []} 两种形态，yarn 同）。
func npmPatterns(root string) []string {
	raw, err := store.LoadSection[jsontext.Value](filepath.Join(root, "package.json"), "workspaces")
	if err != nil || len(raw) == 0 {
		return nil
	}

	// 尝试解析为 []string
	var patterns []string
	if err := json.Unmarshal(raw, &patterns); err == nil {
		return patterns
	}

	// 尝试解析为 {packages: []string}
	var obj struct {
		Packages []string `json:"packages"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil {
		return obj.Packages
	}

	return nil
}

// pnpmPatterns 读 pnpm-workspace.yaml 的 packages 通配声明。
func pnpmPatterns(root string) []string {
	data, err := os.ReadFile(filepath.Join(root, "pnpm-workspace.yaml"))
	if err != nil {
		return nil
	}
	var spec struct {
		Packages []string `yaml:"packages"`
	}
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil
	}
	return spec.Packages
}
