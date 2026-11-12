package cmdutil

import (
	"errors"
	"fmt"
	"strings"
)

// IsPathArg 判断 arg 是否为路径, 仅以下情况满足
// - ".", "..", "~"
// - 以 "./", "../", "~/", "/" 开头
func IsPathArg(arg string) bool {
	if len(arg) == 0 {
		return false
	}

	prefix, _, _ := strings.Cut(arg, "/")
	return prefix == "." || prefix == ".." || prefix == "~" || prefix == ""
}

// AsPathArg 尝试作为路径参数，成功返回绝对路径，失败时返回错误提示 error
// 空字符串会报错，如果空字符串有特殊处理请在调用方提前处理
func AsPathArg(arg string) (string, error) {
	// 正常路径，提前返回
	if IsPathArg(arg) {
		return ExtendPath(arg)
	}

	// 失败路径，尝试各种提示
	if arg == "" {
		return "", errors.New("空字符串无法作为路径参数，如果想表示当前目录请使用 `.`")
	}

	return "", fmt.Errorf("路径参数必须是绝对路径或以 . / ~ 开头的相对路径，你希望的参数是不是: ./%s", arg)
}
