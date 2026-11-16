package handlers

import (
	"time"

	"github.com/heyuuu/x/xstd/xslices"

	"cube/internal/domain/icons"
	"cube/internal/domain/project"
	"cube/internal/domain/project/snapshot"
	"cube/internal/domain/project/workspace"
)

// ProjectDTO project/list 返回的单个项目快照（含 git 状态与最近使用时间）。
type ProjectDTO struct {
	Name       string              `json:"name"`
	Group      string              `json:"group"`
	Path       string              `json:"path"`
	Tags       []string            `json:"tags"`
	Snapshot   *ProjectSnapshotDTO `json:"snapshot"`             // 项目快照，可能为 nil（未采集）
	LastUsedAt *time.Time          `json:"lastUsedAt,omitempty"` // 最近使用时间（置顶排序信号，未用过为 nil）
}

func toProjectDTO(p *project.Project, e *snapshot.Entry) *ProjectDTO {
	if p == nil {
		return nil
	}

	return &ProjectDTO{
		Name:     p.Name(),
		Group:    p.Group(),
		Path:     p.Path(),
		Tags:     p.CollectTags(),
		Snapshot: toProjectSnapshotDTO(e),
	}
}

// ProjectSnapshotDTO 项目快照的对外形态：隔离内部 snapshot.Entry（存储格式与 API
// 从此解耦，snapshot 字段演化不再自动冲击前端）。字段是前端消费盘点后的子集，
// 省略未消费项（branches 仅 CLI 用、worktree 的 detached 经 branch 为空表达）。
type ProjectSnapshotDTO struct {
	RepoURL       string               `json:"repoUrl"`       // origin remote URL
	Remotes       []RemoteDTO          `json:"remotes"`       // 全部 remote（forge 匹配需全部 host）
	CurrentBranch string               `json:"currentBranch"` // 当前检出分支短名，detached 为空
	DefaultBranch string               `json:"defaultBranch"` // 默认主分支名（master/main/...）
	Ahead         int                  `json:"ahead"`         // 默认分支相对 origin 的领先 commit 数
	Behind        int                  `json:"behind"`        // 落后的 commit 数
	Dirty         bool                 `json:"dirty"`         // 工作区是否有改动
	Worktrees     []WorktreeDTO        `json:"worktrees"`     // linked worktree 列表（无则空数组；主目录本身不含在内）
	Workspaces    []WorkspaceMemberDTO `json:"workspaces"`    // 主项目根的 workspace 成员（无则空数组）
	CollectedAt   time.Time            `json:"collectedAt"`   // 本次采集时间
}

// RemoteDTO 单个 remote 配置。
type RemoteDTO struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// WorktreeDTO 单个 linked worktree（path 相对主仓库独立）。
type WorktreeDTO struct {
	Path       string               `json:"path"`       // worktree 绝对路径
	Branch     string               `json:"branch"`     // 检出分支短名；detached 为空
	Workspaces []WorkspaceMemberDTO `json:"workspaces"` // 该 worktree 根的 workspace 成员
}

// toProjectSnapshotDTO 把内部快照转为对外形态；nil（未采集）透传为 nil。
// 主工作树信息（Entry.Main）升位到 DTO 顶级字段；快照 Worktrees
// 只含 linked worktree，作为 worktree 目标下发（前端「主目录」目标由固定首位承载）。
func toProjectSnapshotDTO(e *snapshot.Entry) *ProjectSnapshotDTO {
	if e == nil {
		return nil
	}
	return &ProjectSnapshotDTO{
		RepoURL:       e.OriginURL(),
		Remotes:       xslices.Map(e.Remotes, func(r snapshot.RemoteInfo) RemoteDTO { return RemoteDTO{Name: r.Name, URL: r.URL} }),
		CurrentBranch: e.Main.Branch,
		DefaultBranch: e.DefaultBranch,
		Ahead:         e.DefaultBranchAhead(),
		Behind:        e.DefaultBranchBehind(),
		Dirty:         e.Main.Dirty,
		Worktrees: xslices.Map(e.Worktrees, func(wt snapshot.WorktreeInfo) WorktreeDTO {
			return WorktreeDTO{Path: wt.Path, Branch: wt.Branch, Workspaces: workspaceMemberDTOs(wt.Workspaces)}
		}),
		Workspaces:  workspaceMemberDTOs(e.Main.Workspaces),
		CollectedAt: e.UpdatedAt,
	}
}

// ScanRuleItem scan 规则单项（save / delete / save-all 共用，字段与 project.ScanRule 对齐，path 是唯一键）。
type ScanRuleItem struct {
	Group    string      `json:"group" doc:"扫描出的项目组名"`
	Path     string      `json:"path" doc:"扫描根目录（绝对路径，规则唯一键）"`
	MaxDepth int         `json:"maxDepth" doc:"扫描最大深度"`
	Icon     *icons.Icon `json:"icon,omitempty" doc:"组图标（可选：lucide 图名或 base64 PNG）"`
	Tags     []string    `json:"tags,omitempty" doc:"命中该规则的项目统一追加的标签（如 个人/公司/开源，可选）"`
}

func (it ScanRuleItem) toScanRule() project.ScanRule {
	return project.ScanRule{Group: it.Group, Path: it.Path, MaxDepth: it.MaxDepth, Icon: it.Icon, Tags: it.Tags}
}

// CloneRuleItem clone 规则单项（save / delete / save-all 共用，host+prefix 是唯一键）。
type CloneRuleItem struct {
	RepoHost   string `json:"repoHost" doc:"源域名，无协议，如 github.com"`
	RepoPrefix string `json:"repoPrefix,omitempty" doc:"uri 前缀，须以 / 开头或为空，如 /heyuuu"`
	LocalPath  string `json:"localPath" doc:"对应本地目录（绝对路径）"`
}

func (it CloneRuleItem) toCloneRule() project.CloneRule {
	return project.CloneRule{RepoHost: it.RepoHost, RepoPrefix: it.RepoPrefix, LocalPath: it.LocalPath}
}

// WorkspaceMemberDTO workspace 成员（显示名 + 相对所属根路径）。
type WorkspaceMemberDTO struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// workspaceMemberDTOs workspace.Workspace → WorkspaceMemberDTO（字段同构，类型直转）。
func workspaceMemberDTOs(ws []workspace.Workspace) []WorkspaceMemberDTO {
	return xslices.Map(ws, func(w workspace.Workspace) WorkspaceMemberDTO { return WorkspaceMemberDTO(w) })
}

// WorkspaceRuleDTO workspace 规则（name 可选——通配/特殊规则的组前缀；
// path 为相对路径 / glob / 特殊规则 <npm>/<pnpm>）。
type WorkspaceRuleDTO struct {
	Name string `json:"name"`
	Path string `json:"path"`
}
