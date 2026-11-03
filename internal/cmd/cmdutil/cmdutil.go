// Package cmdutil 收敛 CLI 命令层共享的助手（路径解析、项目搜索/选择、输出渲染），
// 供根命令与各命令组子包复用。
package cmdutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ExtendPath 展开路径为绝对路径，支持相对路径、~ 开头的路径
func ExtendPath(p string) (string, error) {
	// 判空
	if p == "" {
		return "", errors.New("路径不可为空")
	}

	// 绝对路径
	if filepath.IsAbs(p) {
		return filepath.Clean(p), nil
	}

	// 支持 ~ 前缀
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("获取 home 路径失败: %w", err)
		} else if home == "" {
			return "", errors.New("home 路径为空")
		}
		// TrimPrefix 后 "~" 得 ""（Join 返回 home 本身），两种形态统一处理
		return filepath.Join(home, strings.TrimPrefix(p, "~")), nil
	}

	// ~user 形态（他人 home）不支持展开，也不得当作相对路径静默拼 cwd——显式报错
	if strings.HasPrefix(p, "~") {
		return "", fmt.Errorf("不支持 ~user 形式的路径（仅支持 ~/ 指向 home）: %s", p)
	}

	// 支持相对路径
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("获取当前目录失败: %w", err)
	}
	return filepath.Join(wd, p), nil
}

// PrettyTime 把时间格式化为「x秒前 / x分钟前 / x小时前 ...」等相对形式。
//
// 过去时间以「前」结尾，未来时间以「后」结尾（如「30秒后」）。
// 单位按秒/分钟/小时/天/月/年逐级放大，不保留零头（3 小时 20 分 → 「3小时前」）。
func PrettyTime(t time.Time) string {
	d := time.Since(t)
	suffix := "前"
	if d < 0 {
		d = -d
		suffix = "后"
	}

	secs := int64(d.Seconds())
	switch {
	case secs < 60:
		return fmt.Sprintf("%d秒%s", secs, suffix)
	case secs < 3600:
		return fmt.Sprintf("%d分钟%s", secs/60, suffix)
	case secs < 86400:
		return fmt.Sprintf("%d小时%s", secs/3600, suffix)
	case secs < 30*86400:
		return fmt.Sprintf("%d天%s", secs/86400, suffix)
	case secs < 365*86400:
		return fmt.Sprintf("%d月%s", secs/(30*86400), suffix)
	default:
		return fmt.Sprintf("%d年%s", secs/(365*86400), suffix)
	}
}
