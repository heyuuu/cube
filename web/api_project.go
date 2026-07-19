package web

import (
	"github.com/heyuuu/cube/project"
	"github.com/heyuuu/cube/util/slicekit"
)

type ProjectHandler struct {
	service *project.Service
}

func NewProjectHandler(service *project.Service) *ProjectHandler {
	return &ProjectHandler{
		service: service,
	}
}

func (h *ProjectHandler) Register(register func(name string, handler HandleFunc)) {
	register("project/list", h.projectList)
	register("project/info", h.projectInfo)
	register("workspace/list", h.workspaceList)
	register("workspace/info", h.workspaceInfo)
	register("remote/list", h.remoteList)
	register("remote/info", h.remoteInfo)
}

func (h *ProjectHandler) projectList(params any) (result any, err error) {
	projects := h.service.Projects()
	list := slicekit.Map(projects, ToProjectDTO)
	return listResult(list), nil
}

func (h *ProjectHandler) projectInfo(params any) (result any, err error) {
	type infoParams struct {
		Name string `json:"name"`
	}

	// 将 params 转换为结构体
	p, err := parseParam[infoParams](params)
	if err != nil {
		return nil, err
	}

	app := h.service.FindByName(p.Name)
	return itemResult(app, ToProjectDTO)
}

func (h *ProjectHandler) workspaceList(params any) (result any, err error) {
	workspaces := h.service.Workspaces()
	list := slicekit.Map(workspaces, ToWorkspaceDTO)
	return listResult(list), nil
}

func (h *ProjectHandler) workspaceInfo(params any) (result any, err error) {
	type infoParams struct {
		Name string `json:"name"`
	}

	// 将 params 转换为结构体
	p, err := parseParam[infoParams](params)
	if err != nil {
		return nil, err
	}

	app := h.service.FindWorkspaceByName(p.Name)
	return itemResult(app, ToWorkspaceDTO)
}

func (h *ProjectHandler) remoteList(params any) (result any, err error) {
	remotes := h.service.Remotes()
	list := slicekit.Map(remotes, ToRemoteDTO)
	return listResult(list), nil
}

func (h *ProjectHandler) remoteInfo(params any) (result any, err error) {
	type infoParams struct {
		Name string `json:"name"`
	}

	// 将 params 转换为结构体
	p, err := parseParam[infoParams](params)
	if err != nil {
		return nil, err
	}

	app := h.service.FindRemoteByName(p.Name)
	return itemResult(app, ToRemoteDTO)
}
