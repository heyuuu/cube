package server

import (
	"strconv"

	"cube/core/config"
)

// BaseURL 按 port 拼 server 的基地址（无尾斜杠），全仓 http 地址拼接收敛于此。
func BaseURL(cfg config.ServerConfig) string {
	host := cfg.Host
	if host == "" {
		host = "localhost"
	}
	return "http://" + host + ":" + strconv.Itoa(cfg.Port)
}
