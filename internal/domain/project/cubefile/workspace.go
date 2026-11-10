package cubefile

import (
	"log/slog"

	"github.com/heyuuu/x/store"
)

const workspaceSection = "workspaces"

type WorkspaceRule struct {
	// 显示名, 当前规则为通配规则时作为组前缀
	Name string `json:"name,omitempty"`
	// 相对所属根的路径
	// - 支持 glob 通配（apps/*、packages/**，语义同 pnpm-workspace.yaml 的 packages）
	// - 支持特殊规则，目前支持 NpmRule / PnpmRule
	Path string `json:"path,omitempty"`
}

const NpmRule = "<npm>"
const PnpmRule = "<pnpm>"

var defaultWorkspaceDeclares = []WorkspaceRule{
	{Path: NpmRule},
	{Path: PnpmRule},
}

// LoadWorkspaceRules 读取当前 worktree 的 workspace 配置
// 配置不存在时返回默认配置；其他异常时返回空结果避免过多干扰影响
func LoadWorkspaceRules(root string) []WorkspaceRule {
	rules, err := store.LoadSection[[]WorkspaceRule](path(root), workspaceSection)
	if err == nil {
		return rules
	} else if store.IsStoreMissing(err) { // 配置不存在时
		return defaultWorkspaceDeclares
	} else { // 其他错误时
		slog.Warn("读取项目配置失败", "root", root, "section", workspaceSection, "err", err)
		return nil
	}
}

// SaveWorkspaceRules 保存置当前 worktree 的 workspace 配置
func SaveWorkspaceRules(root string, declares []WorkspaceRule) error {
	return store.SaveSection(path(root), workspaceSection, declares)
}

// ResetWorkspaceRules 重置当前 worktree 的 workspace 配置(删除段以走默认逻辑)
func ResetWorkspaceRules(root string) error {
	return store.DeleteSection(path(root), workspaceSection)
}
