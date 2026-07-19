package web

import (
	"github.com/heyuuu/cube/project"
	"github.com/heyuuu/cube/util/slicekit"
)

type RemoteHandler struct {
	service *project.RemoteService
}

func NewRemoteHandler(service *project.RemoteService) *RemoteHandler {
	return &RemoteHandler{
		service: service,
	}
}

func (h *RemoteHandler) Register(register func(name string, handler HandleFunc)) {
	register("remote/list", h.List)
	register("remote/info", h.Info)
}

func (h *RemoteHandler) List(params any) (result any, err error) {
	remotes := h.service.Remotes()
	list := slicekit.Map(remotes, ToRemoteResponseDto)
	return listResult(list), nil
}

func (h *RemoteHandler) Info(params any) (result any, err error) {
	type infoParams struct {
		Name string `json:"name"`
	}

	// 将 params 转换为结构体
	p, err := parseParam[infoParams](params)
	if err != nil {
		return nil, err
	}

	app := h.service.FindByName(p.Name)
	return itemResult(app, ToRemoteResponseDto)
}
