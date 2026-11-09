package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestStdoutWidthNonTTY(t *testing.T) {
	// go test 的 stdout 是管道而非 TTY，此时必须返回 0（= 不限宽），
	// 保证 `cube list | head` 等管道场景拿到完整宽度的输出。
	if got := StdoutWidth(); got != 0 {
		t.Errorf("StdoutWidth() = %d, 非 TTY 下应为 0", got)
	}
}

func TestFitWidthTruncatesOversizedTable(t *testing.T) {
	const limit = 60
	headers := []string{"项目", "Path", "RepoUrl"}
	rows := [][]string{
		{"cube", "~/Code/lab/cube", "git@github.com:heyuuu/cube.git"},
		{"一个名字特别长的中文项目", "~/Code/lab/some/very/long/path/to/project", "https://github.com/heyuuu/some-very-long-repo-name.git"},
	}

	// 前提：不限宽时表格必然超过 limit，截断场景才成立。
	unlimited := buildTable(headers, rows).Render()
	if w := lipgloss.Width(unlimited); w <= limit {
		t.Fatalf("不限宽时表格宽度 %d，应大于 %d", w, limit)
	}

	tbl := buildTable(headers, rows)
	fitWidth(tbl, limit)
	out := tbl.Render()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	// 每行可见宽度不超过 limit：终端无需硬换行，边框不再折断。
	// 中文按显示宽度（占 2 列）计入，顺带守住宽字符截断的宽度计算。
	for i, line := range lines {
		if w := lipgloss.Width(line); w > limit {
			t.Errorf("第 %d 行宽度 %d，超过限宽 %d", i, w, limit)
		}
	}
	// … 截断不折行：行数保持 6（上边框 + 表头 + 分隔线 + 2 数据行 + 下边框）。
	if got := len(lines); got != 6 {
		t.Errorf("限宽后行数 = %d，应保持 6 行（Wrap(false) 不折行）", got)
	}
	if !strings.Contains(out, "…") {
		t.Error("限宽输出应包含 … 截断标记")
	}
}

func TestFitWidthKeepsNarrowTableUnstretched(t *testing.T) {
	headers := []string{"A", "B"}
	rows := [][]string{{"1", "2"}}

	tbl := buildTable(headers, rows)
	fitWidth(tbl, 120)
	out := tbl.Render()

	// 内容远窄于限宽时应保持内容宽，不被拉伸到 120（FitContent 语义，
	// 限宽是上限不是目标）。
	if w := lipgloss.Width(out); w >= 120 {
		t.Errorf("窄表被拉伸到 %d 列，应保持内容宽", w)
	}
}
