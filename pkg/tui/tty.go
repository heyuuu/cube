package tui

import (
	"os"

	"golang.org/x/term"
)

// IsTTY 报告 stdin 是否为一个交互式终端（TTY）。
func IsTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// StdoutWidth 返回 stdout 所在终端的宽度（列数）；非 TTY 或探测失败返回 0。
//
// 探测 stdout 而非 stdin：表格等渲染结果打到 stdout，`cube list | head` 这类
// 管道场景下 stdout 不是 TTY，此时返回 0 表示「不限宽」，让管道拿到完整内容。
// 返回值只作限宽上限用（0 = 调用方跳过限宽），不区分「无法探测」与「真为 0」。
func StdoutWidth() int {
	fd := int(os.Stdout.Fd())
	if !term.IsTerminal(fd) {
		return 0
	}
	w, _, err := term.GetSize(fd)
	if err != nil || w <= 0 {
		return 0
	}
	return w
}

// mustTTY 校验 stdin 是否为交互式终端；非 TTY 时返回 ErrNotTTY。
//
// 所有交互类函数（Select / MultiSelect / Confirm / Input / PasswordInput）
// 在调用 huh 之前先过这一道关。渲染类函数（RenderTable）无需校验。
//
// 校验用 stdin 的 fd：term.IsTerminal(int(os.Stdin.Fd()))。
func mustTTY() error {
	if !IsTTY() {
		return ErrNotTTY
	}
	return nil
}
