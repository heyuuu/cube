package handlers

import (
	"github.com/heyuuu/x/xstd/xslices"

	"cube/internal/domain/forge"
	"cube/internal/domain/icons"
	"cube/internal/domain/project"
	"cube/internal/server"
)

// ForgeHandler forge 配置管理的 HTTP 出口（1040 forge / 1041 account / 1042 forge 页）。
// 1044 起 namespace 模型移除，account 是 forge 下唯一的关联配置。
type ForgeHandler struct {
	forgeService   *forge.Service
	projectService *project.Service // 对账用：本地项目快照投影为 forge.LocalRepo
}

// NewForgeHandler 构造 ForgeHandler。
func NewForgeHandler(forgeService *forge.Service, projectService *project.Service) *ForgeHandler {
	return &ForgeHandler{forgeService: forgeService, projectService: projectService}
}

// Register 注册 forge 域全部路由。
func (h *ForgeHandler) Register(r *server.Routes) {
	r.Get("/api/forge/list", "获取 forge 列表", server.JSONHandler(h.forgeList))
	r.Post("/api/forge/save", "新增或按 host 替换 forge", server.JSONHandler(h.forgeSave))
	r.Post("/api/forge/delete", "删除 forge（提交完整单项，按 host 定位，级联删其 account）", server.JSONHandler(h.forgeDelete))
	r.Post("/api/forge/save-all", "整节覆盖 forge（提交即落盘，含顺序；被移除 forge 的 account 级联删除）", server.JSONHandler(h.forgeSaveAll))

	r.Get("/api/forge/account/list", "获取 forge account 列表（token 打码）", server.JSONHandler(h.accountList))
	r.Post("/api/forge/account/save", "新增或按 forgeHost+username 替换 account", server.JSONHandler(h.accountSave))
	r.Post("/api/forge/account/delete", "删除 account（提交完整单项，按 forgeHost+username 定位）", server.JSONHandler(h.accountDelete))
	r.Post("/api/forge/account/save-all", "整节覆盖 account（提交即落盘，含顺序；掩码 token 沿用旧值）", server.JSONHandler(h.accountSaveAll))
	r.Post("/api/forge/account/fetch", "拉取 account 名下全部远端仓库", server.JSONHandler(h.accountFetch))

	r.Get("/api/forge/overview", "forge 页聚合：全部 account 对账行 + 拉取元信息（只读缓存不外呼）", server.JSONHandler(h.forgeOverview))
}

func (h *ForgeHandler) forgeList(_ struct{}) (ListResult[forge.Forge], error) {
	return listResult(h.forgeService.Forges()), nil
}

// ForgeItem forge 单项（save / delete / save-all 共用，host 归一化后是唯一键）。
type ForgeItem struct {
	Host string      `json:"host" doc:"域名（可带端口），如 github.com"`
	Kind string      `json:"kind" doc:"API 方言：github / gitea / gitee / generic"`
	Icon *icons.Icon `json:"icon,omitempty" doc:"平台图标（可选：lucide 图名或 base64 PNG）"`
}

func (it ForgeItem) toForge() forge.Forge {
	return forge.Forge{Host: it.Host, Kind: it.Kind, Icon: it.Icon}
}

// ForgeSaveInput forge/save 接口入参。
type ForgeSaveInput struct {
	Body ForgeItem
}

func (h *ForgeHandler) forgeSave(input ForgeSaveInput) (map[string]any, error) {
	if err := h.forgeService.SaveForge(input.Body.toForge()); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

// ForgeDeleteInput forge/delete 接口入参（完整单项，唯一键由 domain 计算）。
type ForgeDeleteInput struct {
	Body ForgeItem
}

func (h *ForgeHandler) forgeDelete(input ForgeDeleteInput) (map[string]any, error) {
	if err := h.forgeService.DeleteForge(input.Body.toForge()); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

// ForgeSaveAllInput forge/save-all 接口入参（提交清单即落盘结果，含顺序）。
type ForgeSaveAllInput struct {
	Body struct {
		Items []ForgeItem `json:"items" doc:"按目标顺序排列的全部 forge"`
	}
}

func (h *ForgeHandler) forgeSaveAll(input ForgeSaveAllInput) (map[string]any, error) {
	if err := h.forgeService.SaveAllForges(xslices.Map(input.Body.Items, ForgeItem.toForge)); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

// --- account（token 只以打码形态出 API；save 提交掩码值 = 未修改） ---

func (h *ForgeHandler) accountList(_ struct{}) (ListResult[forge.Account], error) {
	accounts := h.forgeService.Accounts()
	for i := range accounts {
		if accounts[i].Token != "" {
			accounts[i].Token = forge.TokenMasked
		}
	}
	return listResult(accounts), nil
}

// AccountItem account 单项（save / delete / save-all 共用，forgeHost+username 是唯一键）。
type AccountItem struct {
	ForgeHost string `json:"forgeHost" doc:"所属 forge host（唯一键之一）"`
	Username  string `json:"username" doc:"平台用户名（唯一键之一）"`
	Token     string `json:"token,omitempty" doc:"API token；提交掩码值视为未修改"`
}

func (it AccountItem) toAccount() forge.Account {
	return forge.Account{ForgeHost: it.ForgeHost, Username: it.Username, Token: it.Token}
}

// AccountSaveInput account/save 接口入参。
type AccountSaveInput struct {
	Body AccountItem
}

func (h *ForgeHandler) accountSave(input AccountSaveInput) (map[string]any, error) {
	if err := h.forgeService.SaveAccount(input.Body.toAccount()); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

// AccountDeleteInput account/delete 接口入参（完整单项，唯一键由 domain 计算）。
type AccountDeleteInput struct {
	Body AccountItem
}

func (h *ForgeHandler) accountDelete(input AccountDeleteInput) (map[string]any, error) {
	if err := h.forgeService.DeleteAccount(input.Body.toAccount()); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

// AccountFetchInput account/fetch 接口入参（出站 API 调用，同步执行）。
type AccountFetchInput struct {
	Body struct {
		ForgeHost string `json:"forgeHost"`
		Username  string `json:"username"`
		Force     bool   `json:"force,omitempty" doc:"跳过缓存强制重拉"`
	}
}

func (h *ForgeHandler) accountFetch(input AccountFetchInput) (map[string]any, error) {
	b := input.Body
	repos, err := h.forgeService.FetchAccount(b.ForgeHost, b.Username, b.Force)
	if err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "count": len(repos)}, nil
}

// AccountSaveAllInput account/save-all 接口入参（提交清单即落盘结果，含顺序；token 可为 list 回显的掩码值）。
type AccountSaveAllInput struct {
	Body struct {
		Items []AccountItem `json:"items" doc:"按目标顺序排列的全部 account"`
	}
}

func (h *ForgeHandler) accountSaveAll(input AccountSaveAllInput) (map[string]any, error) {
	if err := h.forgeService.SaveAllAccounts(xslices.Map(input.Body.Items, AccountItem.toAccount)); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

// forgeOverview forge 页聚合数据源（只读拉取缓存与本地快照，不触发外呼）。
func (h *ForgeHandler) forgeOverview(_ struct{}) (*forge.Overview, error) {
	return h.forgeService.Overview(h.localRepos()), nil
}

// localRepos 把本地项目快照投影为对账用的 LocalRepo 列表。
func (h *ForgeHandler) localRepos() []forge.LocalRepo {
	locals := make([]forge.LocalRepo, 0)
	for _, p := range h.projectService.Projects() {
		info := forge.LocalRepo{Name: p.Name(), Path: p.Path()}
		if gi, ok := h.projectService.GitInfo(p.Path()); ok {
			info.RepoURL = gi.RepoURL
			info.Dirty = gi.Dirty
			info.Ahead = gi.Ahead
			info.Behind = gi.Behind
		}
		locals = append(locals, info)
	}
	return locals
}
