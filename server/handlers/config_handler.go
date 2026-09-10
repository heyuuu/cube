package handlers

import (
	"cube/core/config"
	"cube/core/server"
)

type ConfigHandler struct {
	cfg *config.Config
}

// NewConfigHandler 接收 *Config 指针，供需要写回配置的场景使用。
func NewConfigHandler(cfg *config.Config) *ConfigHandler {
	return &ConfigHandler{cfg: cfg}
}

func (h *ConfigHandler) Register(r *server.Routes) {
	r.Get("/api/config", "获取配置信息", server.JsonHandler(h.getConfig))
}

func (h *ConfigHandler) getConfig(_ struct{}) (config.Config, error) {
	return *h.cfg, nil
}
