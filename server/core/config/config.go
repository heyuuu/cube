package config

import (
	"errors"
	"fmt"
	"path/filepath"

	"cube/util/store"
)

type Config struct {
	DataDir string       `json:"dataDir"` // 数据目录
	Server  ServerConfig `json:"server"`
}

type ServerConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// Load 从 path 读取 JSON 配置。path 必须是绝对路径—— ~ 与相对路径的展开归 cmd 层，底层不做基于 cwd 的隐式解析。
func Load(path string) (*Config, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("配置路径必须为绝对路径: %s", path)
	}

	cfg, err := store.LoadJson[Config](path)
	if errors.Is(err, store.ErrFileMissing) {
		cfg = Config{} // 文件不存在 → 降级为默认值，不报错
	} else if err != nil {
		return nil, err // 读失败/解析失败（store 已包装中文错误）
	}

	return applyDefaults(&cfg, path), nil
}

func applyDefaults(cfg *Config, path string) *Config {
	if cfg.DataDir == "" {
		cfg.DataDir = filepath.Dir(path)
	}
	return cfg
}
