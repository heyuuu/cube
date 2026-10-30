// Package oskit 文件系统小工具（存在性判定 / 带父目录自动创建的写文件）。
package oskit

import (
	"os"
	"path/filepath"
)

// PathExists 判断路径存在（文件或目录）。
func PathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsFile 判断存在且是普通文件。
func IsFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// IsDir 判断存在且是目录。
func IsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// WriteFile 写文件，父目录不存在时自动创建。
func WriteFile(path string, content []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if !IsDir(dir) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return os.WriteFile(path, content, perm)
}
