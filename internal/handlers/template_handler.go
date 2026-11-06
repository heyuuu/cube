package handlers

import (
	"github.com/heyuuu/x/xstd/xslices"

	"cube/internal/domain/template"
	"cube/internal/server"
)

// TemplateHandler 模板引擎的 HTTP 出口（模板源配置管理，settings.json tplSources 节）。
type TemplateHandler struct {
	templateService *template.Service
}

// NewTemplateHandler 构造 TemplateHandler。
func NewTemplateHandler(templateService *template.Service) *TemplateHandler {
	return &TemplateHandler{templateService: templateService}
}

// Register 注册 template 域端点（实现 server.Handler）。
func (h *TemplateHandler) Register(r *server.Routes) {
	r.Get("/api/template/source/list", "获取模板源列表（含默认源名）", server.JSONHandler(h.sourceList))
	r.Post("/api/template/source/save", "新增或按 name 替换模板源", server.JSONHandler(h.sourceSave))
	r.Post("/api/template/source/delete", "删除模板源（提交完整单项，按 name 定位）", server.JSONHandler(h.sourceDelete))
	r.Post("/api/template/source/save-all", "整节覆盖模板源（提交即落盘，含顺序）", server.JSONHandler(h.sourceSaveAll))
}

// TplSourceListResult template/source/list 出参：列表 + 默认源名（前端标注「默认」用，
// 事实源是 template.DefaultSource 常量，避免前端硬编码）。
type TplSourceListResult struct {
	List          []template.TplSource `json:"list"`
	DefaultSource string               `json:"defaultSource"`
}

func (h *TemplateHandler) sourceList(_ struct{}) (TplSourceListResult, error) {
	return TplSourceListResult{
		List:          h.templateService.TplSources(),
		DefaultSource: template.DefaultSource,
	}, nil
}

// TplSourceItem 模板源单项（save / delete / save-all 共用，字段与 template.TplSource 对齐）。
type TplSourceItem struct {
	Name    string `json:"name" doc:"source 名（cube create 的引用短名，如 core）"`
	RepoURL string `json:"repoUrl" doc:"git 仓库地址（git@host:path / https://host/path / 本地绝对路径）"`
}

func (it TplSourceItem) toTplSource() template.TplSource {
	return template.TplSource{Name: it.Name, RepoURL: it.RepoURL}
}

// TplSourceSaveInput template/source/save 接口入参（name 是唯一键）。
type TplSourceSaveInput struct {
	Body TplSourceItem
}

func (h *TemplateHandler) sourceSave(input TplSourceSaveInput) (map[string]any, error) {
	if err := h.templateService.SaveTplSource(input.Body.toTplSource()); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

// TplSourceDeleteInput template/source/delete 接口入参（完整单项，唯一键由 domain 计算）。
type TplSourceDeleteInput struct {
	Body TplSourceItem
}

func (h *TemplateHandler) sourceDelete(input TplSourceDeleteInput) (map[string]any, error) {
	if err := h.templateService.DeleteTplSource(input.Body.toTplSource()); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

// TplSourceSaveAllInput template/source/save-all 接口入参（提交清单即落盘结果，含顺序）。
type TplSourceSaveAllInput struct {
	Body struct {
		Items []TplSourceItem `json:"items" doc:"按目标顺序排列的全部模板源"`
	}
}

func (h *TemplateHandler) sourceSaveAll(input TplSourceSaveAllInput) (map[string]any, error) {
	if err := h.templateService.SaveAllTplSources(xslices.Map(input.Body.Items, TplSourceItem.toTplSource)); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}
