package logger

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readLastRecord 读取日志文件最后一行并解析为 JSON map。
func readLastRecord(t *testing.T, dir string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, logFileName))
	require.NoError(t, err)
	lines := bytes.Split(bytes.TrimSpace(data), []byte{'\n'})
	require.NotEmpty(t, lines, "日志文件应有内容")
	var rec map[string]any
	require.NoError(t, json.Unmarshal(lines[len(lines)-1], &rec))
	return rec
}

func TestInitFileLog(t *testing.T) {
	dir := t.TempDir()
	Init(dir, false)

	slog.Info("文件日志", "key", "value")
	slog.Debug("Info 级别下应被过滤")

	rec := readLastRecord(t, dir)
	assert.Equal(t, "文件日志", rec["msg"])
	assert.Equal(t, "value", rec["key"])
	assert.Equal(t, "INFO", rec["level"])
	assert.Regexp(t, `^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3}$`, rec["time"])
}

func TestInitDebugMode(t *testing.T) {
	// stdio handler 在构造时取 os.Stderr，先重定向到空文件避免污染测试输出
	origin := os.Stderr
	devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	require.NoError(t, err)
	os.Stderr = devnull
	defer func() { os.Stderr = origin }()

	dir := t.TempDir()
	Init(dir, true)

	slog.Debug("debug 级别可见")

	rec := readLastRecord(t, dir)
	assert.Equal(t, "debug 级别可见", rec["msg"])
	assert.Equal(t, "DEBUG", rec["level"])
}

func TestInitPanicsOnBadDir(t *testing.T) {
	assert.Panics(t, func() { Init("", false) }, "空路径")
	assert.Panics(t, func() { Init("relative/path", false) }, "相对路径")

	file := filepath.Join(t.TempDir(), "a-file")
	require.NoError(t, os.WriteFile(file, nil, 0o644))
	assert.Panics(t, func() { Init(file, false) }, "路径是文件不是目录")
}
