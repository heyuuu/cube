package handlers

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/heyuuu/x/xstd/xslices"

	"cube/internal/domain/icons"
	"cube/internal/domain/opener"
	"cube/internal/domain/project"
	"cube/internal/domain/project/cubefile"
	"cube/internal/domain/project/snapshot"
	"cube/internal/domain/project/workspace"
	"cube/internal/domain/usage"
	"cube/internal/server"
)

// --- dto ---

// ProjectDTO project/list 返回的单个项目快照（含 git 状态与最近使用时间）。
type ProjectDTO struct {
	Name       string              `json:"name"`
	Group      string              `json:"group"`
	Path       string              `json:"path"`
	Tags       []string            `json:"tags"`
	Snapshot   *ProjectSnapshotDTO `json:"snapshot"`             // 项目快照，可能为 nil（未采集）
	LastUsedAt *time.Time          `json:"lastUsedAt,omitempty"` // 最近使用时间（置顶排序信号，未用过为 nil）
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

// newProjectSnapshotDTO 把内部快照转为对外形态；nil（未采集）透传为 nil。
// 主工作树信息（Entry.Main）升位到 DTO 顶级字段；快照 Worktrees
// 只含 linked worktree，作为 worktree 目标下发（前端「主目录」目标由固定首位承载）。
func newProjectSnapshotDTO(e *snapshot.Entry) *ProjectSnapshotDTO {
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

// ProjectListResult 列表接口返回结构：含项目列表 + 两类刷新时间。
type ProjectListResult struct {
	List              []*ProjectDTO `json:"list"`
	ScanUpdatedAt     time.Time     `json:"scanUpdatedAt"`     // 项目列表最近一次重扫完成时间（零值=未刷新过）
	SnapshotUpdatedAt time.Time     `json:"snapshotUpdatedAt"` // 项目快照最近一次落盘时间（零值=无缓存）
}

// --- handler ---

// ProjectHandler project 域的 HTTP 出口。
type ProjectHandler struct {
	projectService *project.Service
	openerService  *opener.Service
	usageService   *usage.Service
}

// NewProjectHandler 构造 ProjectHandler。
func NewProjectHandler(projectService *project.Service, openerService *opener.Service, usageService *usage.Service) *ProjectHandler {
	return &ProjectHandler{
		projectService: projectService,
		openerService:  openerService,
		usageService:   usageService,
	}
}

// Register 注册 project 域全部路由。
func (h *ProjectHandler) Register(r *server.Routes) {
	r.Get("/api/project/list", "获取项目列表", server.JSONHandler(h.projectList))
	r.Post("/api/project/open", "用指定 opener 打开已收录项目（记 usage；目标目录选择为 1030/1032 预留）", server.JSONHandler(h.projectOpen))
	r.Get("/api/project/workspace/get", "获取项目 workspace 状态（生效清单读快照 / 规则清单）", server.JSONHandler(h.workspaceGet))
	r.Post("/api/project/workspace/save", "保存 workspace 规则清单（.cube/cube.json）并即时重采集", server.JSONHandler(h.workspaceSave))
	r.Post("/api/project/workspace/reset", "删除 workspaces 节恢复默认规则并即时重采集", server.JSONHandler(h.workspaceReset))
	r.Get("/api/project/scan-rules", "获取扫描规则", server.JSONHandler(h.scanRules))
	r.Get("/api/project/clone-rules", "获取 clone 规则", server.JSONHandler(h.cloneRules))
	r.Post("/api/project/scan-rule/save", "新增或按 path 替换扫描规则", server.JSONHandler(h.scanRuleSave))
	r.Post("/api/project/scan-rule/delete", "删除扫描规则（提交完整单项，按 path 定位）", server.JSONHandler(h.scanRuleDelete))
	r.Post("/api/project/scan-rule/save-all", "整节覆盖扫描规则（提交即落盘，含顺序；成功后立即重扫）", server.JSONHandler(h.scanRuleSaveAll))
	r.Post("/api/project/clone-rule/save", "新增或按 host+prefix 替换 clone 规则", server.JSONHandler(h.cloneRuleSave))
	r.Post("/api/project/clone-rule/delete", "删除 clone 规则（提交完整单项，按 host+prefix 定位）", server.JSONHandler(h.cloneRuleDelete))
	r.Post("/api/project/clone-rule/save-all", "整节覆盖 clone 规则（提交即落盘，含顺序）", server.JSONHandler(h.cloneRuleSaveAll))
}

func (h *ProjectHandler) projectList(_ struct{}) (ProjectListResult, error) {
	// 返回原始扫描序 + lastUsedAt；排序是视图偏好，由前端做（alfred 出口自带 sortByRecentUsage 置顶）
	projects := h.projectService.Projects()
	latest := h.usageService.LatestByProject()
	list := xslices.Map(projects, func(p *project.Project) *ProjectDTO {
		dto := h.toProjectDTO(p)
		if t, ok := latest[p.Path()]; ok {
			tt := t
			dto.LastUsedAt = &tt
		}
		return dto
	})
	return ProjectListResult{
		List:              list,
		ScanUpdatedAt:     h.projectService.ScanUpdatedAt(),
		SnapshotUpdatedAt: h.projectService.SnapshotUpdatedAt(),
	}, nil
}

// ProjectOpenInput open 接口入参。huma 约定：请求体字段须挂在名为 Body 的子结构上。
// path 是项目绝对路径（项目唯一标识口径）；dir 可选，为目标目录（worktree，1032），
// 缺省/等于项目根 = 打开根目录。
type ProjectOpenInput struct {
	Body struct {
		Path   string `json:"path" doc:"项目绝对路径"`
		Dir    string `json:"dir,omitempty" doc:"目标目录绝对路径（worktree；缺省打开项目根）"`
		Opener string `json:"opener" doc:"opener 名称"`
	}
}

// projectOpen 用指定 opener 打开已收录项目的指定目标目录，成功后记 usage（best-effort：失败不影响打开结果）。
// 区别于 opener/open（打开任意路径，不产生使用信号）。
func (h *ProjectHandler) projectOpen(input ProjectOpenInput) (map[string]any, error) {
	proj := h.projectService.FindByPath(input.Body.Path)
	if proj == nil {
		return nil, errors.New("未找到指定项目: " + input.Body.Path)
	}

	// 目标目录校验：存在性是常规校验；归属只要求「在项目领地内」——主根或其子目录、
	// 任一 worktree 或其子目录（前端目录树可从任意子目录发起打开，不限于 OpenTarget 集合）
	target := proj.Path()
	if input.Body.Dir != "" && input.Body.Dir != proj.Path() {
		if _, err := os.Stat(input.Body.Dir); err != nil {
			return nil, errors.New("目标目录不存在: " + input.Body.Dir)
		}
		if dirProj := h.projectService.ResolveProject(input.Body.Dir); dirProj == nil || dirProj.Path() != proj.Path() {
			return nil, errors.New("目标目录不属于该项目（须位于项目根或其 worktree 内）: " + input.Body.Dir)
		}
		target = input.Body.Dir
	}

	o := h.openerService.FindByName(input.Body.Opener)
	if o == nil {
		return nil, errors.New("未找到指定 opener: " + input.Body.Opener)
	}

	if err := o.Open(opener.RoleOpenDir, target); err != nil {
		return nil, fmt.Errorf("打开失败: %w", err)
	}

	// project 恒记主项目路径；dir 直接传目标目录（等于根时由 RecordOpen 归一为空）
	if err := h.usageService.RecordOpen(proj.Path(), o.Name(), target); err != nil {
		slog.Warn("记录 usage 失败", "err", err)
	}
	return map[string]any{"ok": true}, nil
}

func (h *ProjectHandler) toProjectDTO(entity *project.Project) *ProjectDTO {
	if entity == nil {
		return nil
	}

	info := h.projectService.Snapshot(entity.Path())
	return &ProjectDTO{
		Name:     entity.Name(),
		Group:    entity.Group(),
		Path:     entity.Path(),
		Tags:     entity.CollectTags(),
		Snapshot: newProjectSnapshotDTO(info),
	}
}

func (h *ProjectHandler) scanRules(_ struct{}) (ListResult[project.ScanRule], error) {
	rules := h.projectService.ScanRules()
	return listResult(rules), nil
}

func (h *ProjectHandler) cloneRules(_ struct{}) (ListResult[project.CloneRule], error) {
	rules := h.projectService.CloneRules()
	return listResult(rules), nil
}

// --- 规则写侧（校验在 Service 写侧，坏数据返回中文错误、不落文件） ---

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

// ScanRuleSaveInput scan-rule/save 接口入参。
type ScanRuleSaveInput struct {
	Body ScanRuleItem
}

func (h *ProjectHandler) scanRuleSave(input ScanRuleSaveInput) (map[string]any, error) {
	if err := h.projectService.SaveScanRule(input.Body.toScanRule()); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

// ScanRuleDeleteInput scan-rule/delete 接口入参（完整单项，唯一键由 domain 计算）。
type ScanRuleDeleteInput struct {
	Body ScanRuleItem
}

func (h *ProjectHandler) scanRuleDelete(input ScanRuleDeleteInput) (map[string]any, error) {
	if err := h.projectService.DeleteScanRule(input.Body.toScanRule()); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

// ScanRuleSaveAllInput scan-rule/save-all 接口入参（提交清单即落盘结果，含顺序）。
type ScanRuleSaveAllInput struct {
	Body struct {
		Items []ScanRuleItem `json:"items" doc:"按目标顺序排列的全部扫描规则"`
	}
}

func (h *ProjectHandler) scanRuleSaveAll(input ScanRuleSaveAllInput) (map[string]any, error) {
	if err := h.projectService.SaveAllScanRules(xslices.Map(input.Body.Items, ScanRuleItem.toScanRule)); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
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

// CloneRuleSaveInput clone-rule/save 接口入参。
type CloneRuleSaveInput struct {
	Body CloneRuleItem
}

func (h *ProjectHandler) cloneRuleSave(input CloneRuleSaveInput) (map[string]any, error) {
	if err := h.projectService.SaveCloneRule(input.Body.toCloneRule()); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

// CloneRuleDeleteInput clone-rule/delete 接口入参（完整单项，唯一键由 domain 计算）。
type CloneRuleDeleteInput struct {
	Body CloneRuleItem
}

func (h *ProjectHandler) cloneRuleDelete(input CloneRuleDeleteInput) (map[string]any, error) {
	if err := h.projectService.DeleteCloneRule(input.Body.toCloneRule()); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

// CloneRuleSaveAllInput clone-rule/save-all 接口入参（提交清单即落盘结果，含顺序）。
type CloneRuleSaveAllInput struct {
	Body struct {
		Items []CloneRuleItem `json:"items" doc:"按目标顺序排列的全部 clone 规则"`
	}
}

func (h *ProjectHandler) cloneRuleSaveAll(input CloneRuleSaveAllInput) (map[string]any, error) {
	if err := h.projectService.SaveAllCloneRules(xslices.Map(input.Body.Items, CloneRuleItem.toCloneRule)); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

// --- workspace 声明（1030）---

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

// WorkspaceStateResult workspace/get 返回：编辑视图所需信息。
// Effective 读项目快照（读路径不读规则文件、不解析）；Rules 是编辑场景的一次性
// 现算（未声明时返回默认规则——编辑起点即当前生效配置）。
type WorkspaceStateResult struct {
	Effective []WorkspaceMemberDTO `json:"effective"` // 当前生效成员清单（读快照）
	Declared  bool                 `json:"declared"`  // workspaces 节是否显式声明（false=默认规则生效中）
	Rules     []WorkspaceRuleDTO   `json:"rules"`     // 当前规则清单（未声明时为默认规则）
}

func (h *ProjectHandler) workspaceGet(input struct {
	Path string `query:"path" required:"true"`
}) (WorkspaceStateResult, error) {
	proj := h.projectService.FindByPath(input.Path)
	if proj == nil {
		return WorkspaceStateResult{}, errors.New("未找到指定项目: " + input.Path)
	}
	root := proj.Path()
	result := WorkspaceStateResult{Effective: []WorkspaceMemberDTO{}, Rules: []WorkspaceRuleDTO{}}
	// 生效清单读主工作树（Entry.Main）的 workspace 成员
	if info := h.projectService.Snapshot(root); info != nil {
		result.Effective = workspaceMemberDTOs(info.Main.Workspaces)
	}
	result.Declared = cubefile.WorkspaceRulesDeclared(root)
	result.Rules = xslices.Map(cubefile.LoadWorkspaceRules(root), func(r cubefile.WorkspaceRule) WorkspaceRuleDTO {
		return WorkspaceRuleDTO(r)
	})
	return result, nil
}

// WorkspaceSaveInput workspace/save 接口入参。workspaces 为规则清单整节覆盖
// （空清单 = 显式「无 workspace」；恢复默认规则走 workspace/reset）。
type WorkspaceSaveInput struct {
	Body struct {
		Path       string             `json:"path" doc:"项目绝对路径"`
		Workspaces []WorkspaceRuleDTO `json:"workspaces" doc:"规则清单（name 可选 + path：相对路径/glob/<npm>/<pnpm>）"`
	}
}

// workspaceSave 写侧结构校验严格（坏条目整体拒绝，静默跳过是采集侧的降级语义），
// 成功后定向重采集让打开目标即刻反映。
func (h *ProjectHandler) workspaceSave(input WorkspaceSaveInput) (map[string]any, error) {
	proj := h.projectService.FindByPath(input.Body.Path)
	if proj == nil {
		return nil, errors.New("未找到指定项目: " + input.Body.Path)
	}
	root := proj.Path()
	rules := xslices.Map(input.Body.Workspaces, func(m WorkspaceRuleDTO) cubefile.WorkspaceRule {
		return cubefile.WorkspaceRule(m)
	})
	if err := workspace.SaveRules(root, rules); err != nil {
		return nil, err
	}
	if _, err := h.projectService.RefreshSnapshot(root); err != nil {
		slog.Warn("workspace 保存后刷新快照失败（等下次采集自愈）", "err", err)
	}
	return map[string]any{"ok": true}, nil
}

// workspaceReset 删除 workspaces 节恢复默认规则（<npm>+<pnpm>），成功后定向重采集。
func (h *ProjectHandler) workspaceReset(input struct {
	Body struct {
		Path string `json:"path" doc:"项目绝对路径"`
	}
}) (map[string]any, error) {
	proj := h.projectService.FindByPath(input.Body.Path)
	if proj == nil {
		return nil, errors.New("未找到指定项目: " + input.Body.Path)
	}
	root := proj.Path()
	if err := workspace.Reset(root); err != nil {
		return nil, err
	}
	if _, err := h.projectService.RefreshSnapshot(root); err != nil {
		slog.Warn("workspace 重置后刷新快照失败（等下次采集自愈）", "err", err)
	}
	return map[string]any{"ok": true}, nil
}
