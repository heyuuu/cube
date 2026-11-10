package workspace

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"cube/internal/domain/project/cubefile"
)

// Save 把解析出的成员清单固化为具体规则写入 root/.cube/cube.json（init 的固化
// 语义：机器推导结果转为人的声明）。需至少一条——空清单的显式声明走 SaveRules。
func Save(root string, ws []Workspace) error {
	if len(ws) == 0 {
		return fmt.Errorf("workspaces 为空，如需清除 workspace 声明请删除 .cube/cube.json: root=%s", root)
	}
	rules := make([]cubefile.WorkspaceRule, 0, len(ws))
	for _, w := range ws {
		rules = append(rules, cubefile.WorkspaceRule(w))
	}
	return cubefile.SaveWorkspaceRules(root, rules)
}

// SaveRules 校验并写入显式规则清单（Web 编辑入口）。写侧结构校验严格——
// 坏条目整体拒绝（静默跳过是采集侧的降级语义）；存在性不查，glob 与特殊规则
// 本就允许多命中/暂无命中。空清单是合法声明（显式「无 workspace」，压过默认
// 规则；恢复默认用 Reset）。nil 归一为空清单——节级 API 的 nil 语义是删节。
func SaveRules(root string, rules []cubefile.WorkspaceRule) error {
	if rules == nil {
		rules = []cubefile.WorkspaceRule{}
	}
	for _, rule := range rules {
		if err := validateRule(rule); err != nil {
			return err
		}
	}
	return cubefile.SaveWorkspaceRules(root, rules)
}

// Reset 删除 workspaces 节，恢复程序默认规则（<npm>+<pnpm>）。
func Reset(root string) error {
	return cubefile.ResetWorkspaceRules(root)
}

// validateRule 规则结构校验：path 必填、相对、不逃逸出根。
func validateRule(rule cubefile.WorkspaceRule) error {
	if rule.Path == "" {
		return errors.New("workspace 规则缺 path")
	}
	if filepath.IsAbs(rule.Path) {
		return fmt.Errorf("workspace 规则 path 须为相对路径: %s", rule.Path)
	}
	rel := filepath.Clean(rule.Path)
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("workspace 规则 path 逃逸出项目根: %s", rule.Path)
	}
	return nil
}
