package web

import (
	"github.com/danielgtaylor/huma/v2"

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

func (h *ConfigHandler) Register(api huma.API) {
	apiGet(api, "/api/config", "获取配置信息", h.getConfig)
}

func (h *ConfigHandler) getConfig(_ struct{}) (any, error) {
	return h.conf, nil
}
