package logger

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/lmittmann/tint"
	"golang.org/x/term"
)

const (
	logFileName     = "app.log"
	fileTimeFormat  = "2006-01-02 15:04:05.000"
	stdioTimeFormat = "15:04:05.000"
)

// Init 初始化默认 logger
// - JSON 文件日志始终启用，作为权威记录；
// - debug 为 true 时级别提为 Debug，并附加 stdio 着色输出（非 TTY 自动去色）；
// - dir 必须是绝对路径目录（不存在会自动创建）；
// - 失败直接 panic——没有日志，后续错误只能静默失败。
func Init(dir string, debug bool) {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	// 文件日志 Handler，始终启用
	handler := newFileHandler(level, dir)

	// 在 Debug 模式下时，启用标准输出日志 Handler，颜色看 stderr 是否 TTY
	if debug {
		handler = slog.NewMultiHandler(handler, newStdioHandler(level))
	}

	// 设置为 slog 默认 handler
	slog.SetDefault(slog.New(handler))
}

func newFileHandler(level slog.Level, dir string) slog.Handler {
	// 校验 dir
	if dir == "" {
		panic("log path 配置不应为空")
	} else if !filepath.IsAbs(dir) {
		panic("log path 必须为绝对路径")
	} else if st, err := os.Stat(dir); err == nil && !st.IsDir() {
		panic("log path 必须是目录路径")
	}

	// 构造 logFile 路径
	logFile := filepath.Join(dir, logFileName)

	// init log file
	if err := os.MkdirAll(filepath.Dir(logFile), 0o755); err != nil {
		panic(fmt.Errorf("创建日志目录失败: dir=%s err=%w", filepath.Dir(logFile), err))
	}
	file, err := os.OpenFile(logFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		panic(fmt.Errorf("无法打开日志文件: file=%s err=%w", logFile, err))
	}

	return slog.NewJSONHandler(file, &slog.HandlerOptions{
		Level: level,
		// 顶层 time 改写为本地可读格式（仅顶层，groups 内不动）
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if len(groups) == 0 && a.Key == slog.TimeKey {
				if t, ok := a.Value.Any().(time.Time); ok {
					a.Value = slog.StringValue(t.Format(fileTimeFormat))
				}
			}
			return a
		},
	})
}

func newStdioHandler(level slog.Level) slog.Handler {
	return tint.NewTextHandler(os.Stderr, &tint.Options{
		Level:      level,
		AddSource:  true,
		NoColor:    !isTTY(), // 仅 TTY 模式使用 ANSI 颜色
		TimeFormat: stdioTimeFormat,
	})
}

// 判断 stdin 是否为一个交互式终端（TTY）
func isTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}
