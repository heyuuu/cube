package handlers

import (
	"github.com/heyuuu/x/xstd/xslices"

	"cube/internal/api"
	"cube/internal/domain/template"
)

// TemplateHandler 模板引擎的 HTTP 出口（模板源配置管理，settings.json tplSources 节）。
type TemplateHandler struct {
	templateService *template.Service
}

// NewTemplateHandler 构造 TemplateHandler。
func NewTemplateHandler(templateService *template.Service) *TemplateHandler {
	return &TemplateHandler{templateService: templateService}
}

// Register 注册 template 域端点（实现 api.Registerer）。
func (h *TemplateHandler) Register(r *api.Routes) {
	r.Get("/api/template/source/list", "获取模板源列表（含默认源名）", api.JSONHandler(h.sourceList))
	r.Post("/api/template/source/save", "新增或按 name 替换模板源", api.JSONHandler(h.sourceSave))
	r.Post("/api/template/source/delete", "删除模板源（提交完整单项，按 name 定位）", api.JSONHandler(h.sourceDelete))
	r.Post("/api/template/source/save-all", "整节覆盖模板源（提交即落盘，含顺序）", api.JSONHandler(h.sourceSaveAll))
}

/**
 * GET /api/template/source/list
 */

type templateSourceListResult struct {
	List          []template.TplSource `json:"list"`
	DefaultSource string               `json:"defaultSource"`
}

func (h *TemplateHandler) sourceList(_ struct{}) (*templateSourceListResult, error) {
	return &templateSourceListResult{
		List:          h.templateService.TplSources(),
		DefaultSource: template.DefaultSourceName,
	}, nil
}

/**
 * POST /api/template/source/save
 */

// template/source/save 接口入参（name 是唯一键）。
func (h *TemplateHandler) sourceSave(input WithBody[TplSourceItem]) (*OpResult, error) {
	if err := h.templateService.SaveTplSource(input.Body.toTplSource()); err != nil {
		return nil, err
	}
	return opSuccess, nil
}

/**
 * POST /api/template/source/delete
 */

// template/source/delete 接口入参（完整单项，唯一键由 domain 计算）。
func (h *TemplateHandler) sourceDelete(input WithBody[TplSourceItem]) (*OpResult, error) {
	if err := h.templateService.DeleteTplSource(input.Body.toTplSource()); err != nil {
		return nil, err
	}
	return opSuccess, nil
}

/**
 * POST /api/template/source/save-all
 */

// template/source/save-all 接口入参（提交清单即落盘结果，含顺序）。
func (h *TemplateHandler) sourceSaveAll(input WithBody[struct {
	Items []TplSourceItem `json:"items" doc:"按目标顺序排列的全部模板源"`
}]) (*OpResult, error) {
	if err := h.templateService.SaveAllTplSources(xslices.Map(input.Body.Items, TplSourceItem.toTplSource)); err != nil {
		return nil, err
	}
	return opSuccess, nil
}
