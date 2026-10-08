package handlers

import (
	"cube/core/server"
	"cube/template"
)

// TemplateHandler 模板引擎的 HTTP 出口（模板源配置管理，settings.json tplSources 节）。
type TemplateHandler struct {
	templateService *template.Service
}

func NewTemplateHandler(templateService *template.Service) *TemplateHandler {
	return &TemplateHandler{templateService: templateService}
}

func (h *TemplateHandler) Register(r *server.Routes) {
	r.Get("/api/template/source/list", "获取模板源列表（含默认源名）", server.JsonHandler(h.sourceList))
	r.Post("/api/template/source/save", "新增或按 name 替换模板源", server.JsonHandler(h.sourceSave))
	r.Post("/api/template/source/delete", "按 name 删除模板源", server.JsonHandler(h.sourceDelete))
	r.Post("/api/template/source/reorder", "按 name 重排模板源顺序", server.JsonHandler(h.sourceReorder))
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

// TplSourceSaveInput template/source/save 接口入参（name 是唯一键）。
type TplSourceSaveInput struct {
	Body struct {
		Name    string `json:"name" doc:"source 名（cube create 的引用短名，如 core）"`
		RepoUrl string `json:"repoUrl" doc:"git 仓库地址（git@host:path / https://host/path / 本地绝对路径）"`
	}
}

func (h *TemplateHandler) sourceSave(input TplSourceSaveInput) (map[string]any, error) {
	src := template.TplSource{Name: input.Body.Name, RepoUrl: input.Body.RepoUrl}
	if err := h.templateService.SaveTplSource(src); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

// TplSourceDeleteInput template/source/delete 接口入参。
type TplSourceDeleteInput struct {
	Body struct {
		Name string `json:"name" doc:"source 名（唯一键）"`
	}
}

func (h *TemplateHandler) sourceDelete(input TplSourceDeleteInput) (map[string]any, error) {
	if err := h.templateService.DeleteTplSource(input.Body.Name); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

// TplSourceReorderInput template/source/reorder 接口入参（整表按目标顺序提交 name 名单）。
type TplSourceReorderInput struct {
	Body struct {
		Names []string `json:"names" doc:"按目标顺序排列的 source 名名单"`
	}
}

func (h *TemplateHandler) sourceReorder(input TplSourceReorderInput) (map[string]any, error) {
	if err := h.templateService.ReorderTplSources(input.Body.Names); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}
