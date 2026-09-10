package app

import (
	"fmt"
	"os"
	"path/filepath"
)

// 子项相对根的文件/目录名。
const (
	settingsFileName = "settings.json"
	logDirName       = "log"
	stateDirName     = "state"
	cacheDirName     = "cache"
)

// Paths 数据目录的路径
// 零值不可用，必须用 NewPaths() 构造。
type Paths struct {
	dataDir string // 展开后的绝对路径（已 MkdirAll）
}

func NewPaths(dataDir string) *Paths {
	// guard 数据路径必须是绝对路径
	if !filepath.IsAbs(dataDir) {
		panic(fmt.Errorf("数据目录必须是绝对路径: dir=%s", dataDir))
	}
	// 尝试创建目录
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		panic(fmt.Errorf("创建数据目录失败: dir=%s err=%w", dataDir, err))
	}
	return &Paths{dataDir: dataDir}
}

func (p *Paths) SettingsFile() string { return filepath.Join(p.dataDir, settingsFileName) }

// LogDir 日志目录
func (p *Paths) LogDir() string { return filepath.Join(p.dataDir, logDirName) }

// StateDir 运行期状态目录（如 usage.jsonl）：由日常使用产生、非配置非缓存——
// 丢了可接受但不理想，区别于 cache/ 的「可整体删除且行为不变差」。
func (p *Paths) StateDir() string { return filepath.Join(p.dataDir, stateDirName) }

// CacheDir 纯缓存目录（如 git.json）：可整体删除且 app 行为不变差，随时可重建。
func (p *Paths) CacheDir() string { return filepath.Join(p.dataDir, cacheDirName) }
