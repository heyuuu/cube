// Package pathkit 路径展示与处理小工具（纯函数；~ 展开等环境交互只出现在本包）。
package pathkit

import (
	"os"
	"path/filepath"
	"strings"
)

// PrettyPath 把 home 下的绝对路径美化为 ~ 缩写形式，其余规范化后原样返回。
func PrettyPath(path string) string {
	if path == "" {
		return ""
	}
	// 规范化冗余分隔符 / . / ..，保证后续比较稳定
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	rel, err := filepath.Rel(home, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return path
	}

	// path 即 home 本身（rel 为 "."），标准化为 "~"
	if rel == "." {
		return "~"
	}

	return "~/" + rel
}

// IsUnder 判断 path 是否为 parent 本身或 parent 下的目录/文件（路径语义：两边 Clean，
// 按分隔符边界比较，/a/b 不算在 /a/bb 之下）
func IsUnder(path string, parent string) bool {
	path = filepath.Clean(path)
	parent = filepath.Clean(parent)
	return path == parent || strings.HasPrefix(path, parent+string(filepath.Separator))
}
