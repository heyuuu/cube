package tui

import "testing"

func TestStdoutWidthNonTTY(t *testing.T) {
	// go test 的 stdout 是管道而非 TTY，此时必须返回 0（= 不限宽），
	// 保证 `cube list | head` 等管道场景拿到完整宽度的输出。
	if got := StdoutWidth(); got != 0 {
		t.Errorf("StdoutWidth() = %d, 非 TTY 下应为 0", got)
	}
}
