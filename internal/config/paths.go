package config

import (
	"path/filepath"
)

// 子项相对根的文件/目录名。
const (
	settingsFileName = "settings.json"
	logDirName       = "log"
	stateDirName     = "state"
	cacheDirName     = "cache"
	usageFileName    = "usage.jsonl"
)

// SettingsFile 用户可管理数据文件 settings.json 的完整路径。
func (c *Config) SettingsFile() string { return filepath.Join(c.DataDir, settingsFileName) }

// LogDir 日志目录
func (c *Config) LogDir() string { return filepath.Join(c.DataDir, logDirName) }

// StateDir 运行期状态目录（如 usage.jsonl）：由日常使用产生、非配置非缓存——
// 丢了可接受但不理想，区别于 cache/ 的「可整体删除且行为不变差」。
func (c *Config) StateDir() string { return filepath.Join(c.DataDir, stateDirName) }

// UsageFile 打开记录文件 usage.jsonl 的完整路径（state/ 下）。
func (c *Config) UsageFile() string { return filepath.Join(c.StateDir(), usageFileName) }

// CacheDir 纯缓存目录（如 git.json）：可整体删除且 app 行为不变差，随时可重建。
func (c *Config) CacheDir() string { return filepath.Join(c.DataDir, cacheDirName) }
