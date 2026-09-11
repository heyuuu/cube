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

// Load 从 path 读取 JSON 配置。
// path 来自 --config flag（或默认值 ~/.config/cube/config.json）——~ 前缀已由
// cmd 层（root PersistentPreRunE）展开为绝对路径，此处仅做基于 cwd 的相对路径兜底。
func Load(path string) (*Config, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("解析配置路径失败: path=%s err=%w", path, err)
	}

	cfg, err := store.LoadJson[Config](absPath)
	if errors.Is(err, store.ErrFileMissing) {
		cfg = Config{} // 文件不存在 → 降级为默认值，不报错
	} else if err != nil {
		return nil, err // 读失败/解析失败（store 已包装中文错误）
	}

	return applyDefaults(&cfg, absPath), nil
}

func applyDefaults(cfg *Config, path string) *Config {
	if cfg.DataDir == "" {
		cfg.DataDir = filepath.Dir(path)
	}
	return cfg
}
