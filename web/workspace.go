package web

import (
	"github.com/heyuuu/cube/project"
	"github.com/heyuuu/cube/util/slicekit"
)

type WorkspaceHandler struct {
	service *project.WorkspaceService
}

func NewWorkspaceHandler(service *project.WorkspaceService) *WorkspaceHandler {
	return &WorkspaceHandler{
		service: service,
	}
}

func (h *WorkspaceHandler) Register(register func(name string, handler HandleFunc)) {
	register("workspace/list", h.List)
	register("workspace/info", h.Info)
}

func (h *WorkspaceHandler) List(params any) (result any, err error) {
	workspaces := h.service.Workspaces()
	list := slicekit.Map(workspaces, ToWorkspaceResponseDto)
	return listResult(list), nil
}

func (h *WorkspaceHandler) Info(params any) (result any, err error) {
	type infoParams struct {
		Name string `json:"name"`
	}

	// 将 params 转换为结构体
	p, err := parseParam[infoParams](params)
	if err != nil {
		return nil, err
	}

	app := h.service.FindByName(p.Name)
	return itemResult(app, ToWorkspaceResponseDto)
}
