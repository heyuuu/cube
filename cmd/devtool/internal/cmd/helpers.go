package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// rootDir 返回仓库根目录：从当前工作目录向上查找，
// 直到找到 .git 目录为止。
func rootDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if fi, err := os.Stat(filepath.Join(dir, ".git")); err == nil && fi.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("从 %s 向上未找到仓库根目录（.git）", dir)
		}
		dir = parent
	}
}

func gitOut(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s：%w", strings.Join(args, " "), err)
	}
	return string(out), nil
}

// isGitDirty 返回工作区是否有未提交变更
func isGitDirty(root string) (bool, error) {
	status, err := gitOut(root, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return status != "", nil
}
