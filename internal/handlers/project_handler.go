package handlers

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/heyuuu/x/xstd/xslices"

	"cube/internal/api"
	"cube/internal/domain/opener"
	"cube/internal/domain/project"
	"cube/internal/domain/project/cubefile"
	"cube/internal/domain/project/workspace"
	"cube/internal/domain/usage"
)

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
func (h *ProjectHandler) Register(r *api.Routes) {
	r.Get("/api/project/list", "获取项目列表", api.JSONHandler(h.list))
	r.Post("/api/project/open", "用指定 opener 打开已收录项目", api.JSONHandler(h.open))
	r.Get("/api/project/scan-rules", "获取扫描规则", api.JSONHandler(h.scanRules))
	r.Get("/api/project/clone-rules", "获取 clone 规则", api.JSONHandler(h.cloneRules))
	r.Post("/api/project/scan-rule/save", "新增或按 path 替换扫描规则", api.JSONHandler(h.scanRuleSave))
	r.Post("/api/project/scan-rule/delete", "删除扫描规则（提交完整单项，按 path 定位）", api.JSONHandler(h.scanRuleDelete))
	r.Post("/api/project/scan-rule/save-all", "整节覆盖扫描规则（提交即落盘，含顺序；成功后立即重扫）", api.JSONHandler(h.scanRuleSaveAll))
	r.Post("/api/project/clone-rule/save", "新增或按 host+prefix 替换 clone 规则", api.JSONHandler(h.cloneRuleSave))
	r.Post("/api/project/clone-rule/delete", "删除 clone 规则（提交完整单项，按 host+prefix 定位）", api.JSONHandler(h.cloneRuleDelete))
	r.Post("/api/project/clone-rule/save-all", "整节覆盖 clone 规则（提交即落盘，含顺序）", api.JSONHandler(h.cloneRuleSaveAll))
	r.Get("/api/project/workspace/get", "获取项目 workspace 状态（生效清单读快照 / 规则清单）", api.JSONHandler(h.workspaceGet))
	r.Post("/api/project/workspace/save", "保存 workspace 规则清单（.cube/cube.json）并即时重采集", api.JSONHandler(h.workspaceSave))
	r.Post("/api/project/workspace/reset", "删除 workspaces 节恢复默认规则并即时重采集", api.JSONHandler(h.workspaceReset))
}

/**
 * GET /api/project/list
 */

type projectListResult struct {
	List              []*ProjectDTO `json:"list"`
	ScanUpdatedAt     time.Time     `json:"scanUpdatedAt"`     // 项目列表最近一次重扫完成时间（零值=未刷新过）
	SnapshotUpdatedAt time.Time     `json:"snapshotUpdatedAt"` // 项目快照最近一次落盘时间（零值=无缓存）
}

func (h *ProjectHandler) list(_ struct{}) (*projectListResult, error) {
	// 返回原始扫描序 + lastUsedAt；排序是视图偏好，由前端做（alfred 出口自带 sortByRecentUsage 置顶）
	projects := h.projectService.Projects()
	latest := h.usageService.LatestByProject()
	list := xslices.Map(projects, func(p *project.Project) *ProjectDTO {
		snap := h.projectService.Snapshot(p.Path())
		dto := toProjectDTO(p, snap)
		if t, ok := latest[p.Path()]; ok {
			tt := t
			dto.LastUsedAt = &tt
		}
		return dto
	})
	return &projectListResult{
		List:              list,
		ScanUpdatedAt:     h.projectService.ScanUpdatedAt(),
		SnapshotUpdatedAt: h.projectService.SnapshotUpdatedAt(),
	}, nil
}

/**
 * POST /api/project/open
 */

// open 用指定 opener 打开已收录项目的指定目标目录，成功后记 usage（best-effort：失败不影响打开结果）。
// 区别于 opener/open（打开任意路径，不产生使用信号）。
// 入参：path 是项目绝对路径（项目唯一标识口径）；dir 可选，为目标目录（worktree，1032），
// 缺省/等于项目根 = 打开根目录。
func (h *ProjectHandler) open(input WithBody[struct {
	Path   string `json:"path" doc:"项目绝对路径"`
	Dir    string `json:"dir,omitempty" doc:"目标目录绝对路径（worktree；缺省打开项目根）"`
	Opener string `json:"opener" doc:"opener 名称"`
}]) (*OpResult, error) {
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

	if err := h.openerService.Open(*o, opener.RoleOpenDir, target); err != nil {
		return nil, fmt.Errorf("打开失败: %w", err)
	}

	// project 恒记主项目路径；dir 直接传目标目录（等于根时由 RecordOpen 归一为空）
	if err := h.usageService.RecordOpen(proj.Path(), o.Name, target); err != nil {
		slog.Warn("记录 usage 失败", "err", err)
	}
	return opSuccess, nil
}

/**
 * GET /api/project/scan-rules
 */

func (h *ProjectHandler) scanRules(_ struct{}) (*ListResult[project.ScanRule], error) {
	rules := h.projectService.ScanRules()
	return listResult(rules), nil
}

/**
 * GET /api/project/clone-rules
 */

func (h *ProjectHandler) cloneRules(_ struct{}) (*ListResult[project.CloneRule], error) {
	rules := h.projectService.CloneRules()
	return listResult(rules), nil
}

/**
 * POST /api/project/scan-rule/save
 */

func (h *ProjectHandler) scanRuleSave(input WithBody[ScanRuleItem]) (*OpResult, error) {
	if err := h.projectService.SaveScanRule(input.Body.toScanRule()); err != nil {
		return nil, err
	}
	return opSuccess, nil
}

/**
 * POST /api/project/scan-rule/delete
 */

func (h *ProjectHandler) scanRuleDelete(input WithBody[ScanRuleItem]) (*OpResult, error) {
	if err := h.projectService.DeleteScanRule(input.Body.toScanRule()); err != nil {
		return nil, err
	}
	return opSuccess, nil
}

/**
 * POST /api/project/scan-rule/save-all
 */

func (h *ProjectHandler) scanRuleSaveAll(input WithBody[struct {
	Items []ScanRuleItem `json:"items" doc:"按目标顺序排列的全部扫描规则"`
}]) (*OpResult, error) {
	if err := h.projectService.SaveAllScanRules(xslices.Map(input.Body.Items, ScanRuleItem.toScanRule)); err != nil {
		return nil, err
	}
	return opSuccess, nil
}

/**
 * POST /api/project/clone-rule/save
 */

func (h *ProjectHandler) cloneRuleSave(input WithBody[CloneRuleItem]) (*OpResult, error) {
	if err := h.projectService.SaveCloneRule(input.Body.toCloneRule()); err != nil {
		return nil, err
	}
	return opSuccess, nil
}

/**
 * POST /api/project/clone-rule/delete
 */

func (h *ProjectHandler) cloneRuleDelete(input WithBody[CloneRuleItem]) (*OpResult, error) {
	if err := h.projectService.DeleteCloneRule(input.Body.toCloneRule()); err != nil {
		return nil, err
	}
	return opSuccess, nil
}

/**
 * POST /api/project/clone-rule/save-all
 */

func (h *ProjectHandler) cloneRuleSaveAll(input WithBody[struct {
	Items []CloneRuleItem `json:"items" doc:"按目标顺序排列的全部 clone 规则"`
}]) (*OpResult, error) {
	if err := h.projectService.SaveAllCloneRules(xslices.Map(input.Body.Items, CloneRuleItem.toCloneRule)); err != nil {
		return nil, err
	}
	return opSuccess, nil
}

// --- workspace 声明（1030）---

/**
 * GET /api/project/workspace/get
 */

// ProjectWorkspaceState workspace/get 返回：编辑视图所需信息。
// Effective 读项目快照（读路径不读规则文件、不解析）；Rules 是编辑场景的一次性
// 现算（未声明时返回默认规则——编辑起点即当前生效配置）。
type ProjectWorkspaceState struct {
	Effective []WorkspaceMemberDTO `json:"effective"` // 当前生效成员清单（读快照）
	Declared  bool                 `json:"declared"`  // workspaces 节是否显式声明（false=默认规则生效中）
	Rules     []WorkspaceRuleDTO   `json:"rules"`     // 当前规则清单（未声明时为默认规则）
}

func (h *ProjectHandler) workspaceGet(input struct {
	Path string `query:"path" required:"true"`
}) (*ProjectWorkspaceState, error) {
	proj := h.projectService.FindByPath(input.Path)
	if proj == nil {
		return nil, errors.New("未找到指定项目: " + input.Path)
	}

	root := proj.Path()
	result := ProjectWorkspaceState{Effective: []WorkspaceMemberDTO{}, Rules: []WorkspaceRuleDTO{}}

	// 生效清单读主工作树（Entry.Main）的 workspace 成员
	if info := h.projectService.Snapshot(root); info != nil {
		result.Effective = workspaceMemberDTOs(info.Main.Workspaces)
	}

	result.Declared = cubefile.WorkspaceRulesDeclared(root)
	result.Rules = xslices.Map(cubefile.LoadWorkspaceRules(root), func(r cubefile.WorkspaceRule) WorkspaceRuleDTO {
		return WorkspaceRuleDTO(r)
	})
	return &result, nil
}

/**
 * POST /api/project/workspace/save
 */

// workspaceSave 写侧结构校验严格（坏条目整体拒绝，静默跳过是采集侧的降级语义），
// 成功后定向重采集让打开目标即刻反映。
func (h *ProjectHandler) workspaceSave(input WithBody[struct {
	Path       string             `json:"path" doc:"项目绝对路径"`
	Workspaces []WorkspaceRuleDTO `json:"workspaces" doc:"规则清单（name 可选 + path：相对路径/glob/<npm>/<pnpm>）"`
}]) (*OpResult, error) {
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
	return opSuccess, nil
}

/**
 * POST /api/project/workspace/reset
 */

// workspaceReset 删除 workspaces 节恢复默认规则（<npm>+<pnpm>），成功后定向重采集。
func (h *ProjectHandler) workspaceReset(input WithBody[struct {
	Path string `json:"path" doc:"项目绝对路径"`
}]) (*OpResult, error) {
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
	return opSuccess, nil
}
