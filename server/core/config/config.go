package config

import (
	"errors"
	"fmt"
	"path/filepath"

	"cube/util/pathkit"
	"cube/util/store"
)

type Config struct {
	DataDir string       `json:"dataDir"` // 数据目录
	Server  ServerConfig `json:"server"`
	Create  CreateConfig `json:"create"`
}

type CreateConfig struct {
	TemplateSource string `json:"templateSource"` // cube create 未显式传 --tpl 时的默认模板来源（本地目录或 git url）
}

type ServerConfig struct {
	Port int `json:"port"`
}

// Load 从 path 读取 JSON 配置。
// path 通常来自 -c 命令行参数（或默认值 ~/.config/cube/config.json），
// 故按命令行输入解析：支持 ~ 前缀与基于 cwd 的相对路径。
func Load(path string) (*Config, error) {
	absPath, err := pathkit.AbsPath(path)
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
