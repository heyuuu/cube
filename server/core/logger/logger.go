package logger

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/lmittmann/tint"

	"cube/util/tui"
)

const logFileName = "app.log"
const logTimeFormat = "2006-01-02 15:04:05.000"
const stdioLogTimeFormat = "15:04:05.000"

// Init 初始化日志
// 初始化失败会直接 panic，因为没有日志根本无法记录错误，容易导致静默失败。
func Init(logPath string, debug bool) {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	// 文件日志 Handler，始终启用
	handler := initFileHandler(level, logPath)

	// 在 Debug 模式下时，启用标准输出日志 Handler，颜色看 stderr 是否 TTY
	if debug {
		stdioHandler := initStdioHandler(level)
		handler = slog.NewMultiHandler(handler, stdioHandler)
	}

	// 设置为 slog 默认 handler
	slog.SetDefault(slog.New(handler))
}

func initFileHandler(level slog.Level, logPath string) slog.Handler {
	// 校验 logPath
	if logPath == "" {
		panic("log path 配置不应为空")
	} else if !filepath.IsAbs(logPath) {
		panic("log path 必须为绝对路径")
	} else if st, err := os.Stat(logPath); err == nil && !st.IsDir() {
		panic("log path 必须是目录路径")
	}

	// 构造 logFile 路径
	logFile := filepath.Join(logPath, logFileName)

	// init log file
	if err := os.MkdirAll(filepath.Dir(logFile), 0o755); err != nil {
		panic(fmt.Errorf("创建日志目录失败: dir=%s err=%w", filepath.Dir(logFile), err))
	}
	file, err := os.OpenFile(logFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		panic(fmt.Errorf("无法开始日志文件: file=%s err=%w", logFile, err))
	}

	return slog.NewJSONHandler(file, &slog.HandlerOptions{
		Level: level,
		// 修改日志格式
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// 时间字段：顶层记录的 time（groups 为空，key 为 "time"）
			if len(groups) == 0 && a.Key == slog.TimeKey {
				if t, ok := a.Value.Any().(time.Time); ok {
					a.Value = slog.StringValue(t.Format(logTimeFormat))
				}
			}
			return a
		},
	})
}

func initStdioHandler(level slog.Level) slog.Handler {
	return tint.NewTextHandler(os.Stderr, &tint.Options{
		Level:     level,
		AddSource: true,
		NoColor:   !tui.IsTTY(), // 仅 TTY 模式使用 ANSI 颜色
		// 修改日志格式
		TimeFormat: stdioLogTimeFormat,
	})
}
