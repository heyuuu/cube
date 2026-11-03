package opener

import (
	"errors"
	"fmt"
	"os"

	"cube/internal/cmd/cmdutil"
)

// checkOpenPath 解析并校验路径：返回绝对路径及其是否为目录。
func checkOpenPath(path string) (absPath string, isDir bool, err error) {
	// 获取绝对路径
	absPath, err = cmdutil.ExtendPath(path)
	if err != nil {
		return "", false, err
	}
	// 获取文件信息
	info, err := os.Stat(absPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, fmt.Errorf("路径不存在: %s", path)
		}
		return "", false, fmt.Errorf("读取路径失败: %w", err)
	}

	return absPath, info.IsDir(), nil
}
