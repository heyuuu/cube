package web

import (
	"github.com/heyuuu/cube/config"
)

type ConfigHandler struct {
	conf config.Config
}

func NewConfigHandler(conf config.Config) *ConfigHandler {
	return &ConfigHandler{
		conf: conf,
	}
}

func (h *ConfigHandler) Register(register func(name string, handler HandleFunc)) {
	register("config", h.Get)
}

func (h *ConfigHandler) Get(params any) (result any, err error) {
	return h.conf, nil
}
