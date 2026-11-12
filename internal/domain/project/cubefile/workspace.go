package cubefile

import (
	"log/slog"

	"github.com/heyuuu/x/store"
)

const workspaceSection = "workspaces"

// WorkspaceRule workspaces 节里的单条规则。
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

// LoadWorkspaceRules 读取当前根（项目根或 worktree 根）的 workspace 规则。
// 节/文件缺失时返回默认配置；其他异常返回空结果避免过多干扰影响。
func LoadWorkspaceRules(root string) []WorkspaceRule {
	rules, err := store.LoadSection[[]WorkspaceRule](FilePath(root), workspaceSection)
	if err == nil {
		return rules
	} else if store.IsStoreMissing(err) { // 配置不存在时
		return defaultWorkspaceDeclares
	} else { // 其他错误时
		slog.Warn("读取项目配置失败", "root", root, "section", workspaceSection, "err", err)
		return nil
	}
}

// WorkspaceRulesDeclared 报告 workspaces 节是否已显式声明
// （文件/节缺失或坏 JSON 均为 false——坏 JSON 时规则读侧同样降级为空）。
func WorkspaceRulesDeclared(root string) bool {
	_, err := store.LoadSection[[]WorkspaceRule](FilePath(root), workspaceSection)
	return err == nil
}

// SaveWorkspaceRules 保存当前根的 workspace 规则（整节覆盖，未知节由 store 保留；
// nil 会删节——空清单请传初始化过的切片）。
func SaveWorkspaceRules(root string, declares []WorkspaceRule) error {
	return store.SaveSection(FilePath(root), workspaceSection, declares)
}

// ResetWorkspaceRules 重置当前根的 workspace 配置(删除段以走默认逻辑)
func ResetWorkspaceRules(root string) error {
	return store.DeleteSection(FilePath(root), workspaceSection)
}
