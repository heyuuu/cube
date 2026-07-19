package web

import (
	"github.com/heyuuu/cube/opener"
	"github.com/heyuuu/cube/util/slicekit"
)

type OpenerHandler struct {
	service *opener.Service
}

func NewOpenerHandler(service *opener.Service) *OpenerHandler {
	return &OpenerHandler{
		service: service,
	}
}

func (h *OpenerHandler) Register(register func(name string, handler HandleFunc)) {
	register("opener/list", h.List)
	register("opener/info", h.Info)
}

func (h *OpenerHandler) List(params any) (result any, err error) {
	apps := h.service.Openers()
	list := slicekit.Map(apps, ToOpenerDTO)
	return listResult(list), nil
}

func (h *OpenerHandler) Info(params any) (result any, err error) {
	type infoParams struct {
		Name string `json:"name"`
	}

	// 将 params 转换为结构体
	p, err := parseParam[infoParams](params)
	if err != nil {
		return nil, err
	}

	app := h.service.FindByName(p.Name)
	return itemResult(app, ToOpenerDTO)
}
