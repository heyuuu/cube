package pathkit

import (
	"os"
	"path/filepath"
	"strings"
)

func RealPath(path string) string {
	// 支持 ~ 前缀
	if strings.HasPrefix(path, "~/") {
		path = filepath.Join(os.Getenv("HOME"), path[2:])
	}
	return path
}

func PrettyPath(path string) string {
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

	return "~/" + rel
}
