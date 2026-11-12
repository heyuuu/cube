package tui

import (
	"strings"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// 差异高亮沿用 git diff 的红绿惯例：旧串中被删除的部分标红，
// 新串中新增的部分标绿；两串共同的部分保持原样。
var (
	diffOldStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("203"))
	diffNewStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("35"))
)

// DiffLine 对比两个单行字符串，返回带差异高亮的 (旧串, 新串)。
//
// 输入会先剥离 ANSI 样式与控制字符再对比——调用方传入的串可能已带样式
// （如着色过的路径），残留的转义序列既会干扰对比，也会破坏高亮拼接。
func DiffLine(old, new string) (string, string) {
	oldRunes := []rune(stripControl(ansi.Strip(old)))
	newRunes := []rune(stripControl(ansi.Strip(new)))

	// 先截掉公共前后缀，把 LCS 的 DP 矩阵缩小到真正差异的中段
	// （单行对比的两串通常共享大段前缀，如路径）。
	start := 0
	for start < len(oldRunes) && start < len(newRunes) && oldRunes[start] == newRunes[start] {
		start++
	}
	end := 0
	for end < len(oldRunes)-start && end < len(newRunes)-start &&
		oldRunes[len(oldRunes)-1-end] == newRunes[len(newRunes)-1-end] {
		end++
	}
	prefix, suffix := string(oldRunes[:start]), string(oldRunes[len(oldRunes)-end:])
	oldSpans, newSpans := diffSegments(oldRunes[start:len(oldRunes)-end], newRunes[start:len(newRunes)-end])

	return renderDiffLine(prefix, oldSpans, suffix, diffOldStyle),
		renderDiffLine(prefix, newSpans, suffix, diffNewStyle)
}

// diffSpan 是 diff 结果中的一段文本，changed 标记它是否属于差异部分。
type diffSpan struct {
	text    string
	changed bool
}

// diffSegments 用 LCS（最长公共子序列）对比两个 rune 序列，
// 把各自拆成「共同部分 / 差异部分」交替的段落序列。
func diffSegments(oldRunes, newRunes []rune) (oldSpans, newSpans []diffSpan) {
	n, m := len(oldRunes), len(newRunes)
	// lcs[i][j] = oldRunes[i:] 与 newRunes[j:] 的 LCS 长度，
	// 回溯时据此决定当前位置取删除（走 i）还是新增（走 j）。
	lcs := make([][]int32, n+1)
	for i := range lcs {
		lcs[i] = make([]int32, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			switch {
			case oldRunes[i] == newRunes[j]:
				lcs[i][j] = lcs[i+1][j+1] + 1
			case lcs[i+1][j] >= lcs[i][j+1]:
				lcs[i][j] = lcs[i+1][j]
			default:
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}

	i, j := 0, 0
	for i < n && j < m {
		switch {
		case oldRunes[i] == newRunes[j]:
			appendDiffRune(&oldSpans, oldRunes[i], false)
			appendDiffRune(&newSpans, newRunes[j], false)
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			appendDiffRune(&oldSpans, oldRunes[i], true)
			i++
		default:
			appendDiffRune(&newSpans, newRunes[j], true)
			j++
		}
	}
	for ; i < n; i++ {
		appendDiffRune(&oldSpans, oldRunes[i], true)
	}
	for ; j < m; j++ {
		appendDiffRune(&newSpans, newRunes[j], true)
	}
	return oldSpans, newSpans
}

// appendDiffRune 追加一个字符，与上一段同属共同/差异时并入上一段，避免碎片段落。
func appendDiffRune(spans *[]diffSpan, r rune, changed bool) {
	if last := len(*spans) - 1; last >= 0 && (*spans)[last].changed == changed {
		(*spans)[last].text += string(r)
		return
	}
	*spans = append(*spans, diffSpan{text: string(r), changed: changed})
}

// renderDiffLine 把前后缀与段落序列拼回单个字符串，差异段落套用高亮样式。
func renderDiffLine(prefix string, spans []diffSpan, suffix string, changedStyle lipgloss.Style) string {
	var b strings.Builder
	b.WriteString(prefix)
	for _, span := range spans {
		if span.changed {
			b.WriteString(changedStyle.Render(span.text))
		} else {
			b.WriteString(span.text)
		}
	}
	b.WriteString(suffix)
	return b.String()
}

// stripControl 移除除 tab 外的控制字符：ANSI 序列之外的零散控制符
// （\r、\n、孤立的 ESC 等）没有展示意义，混进高亮片段只会干扰渲染。
func stripControl(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\t' {
			return -1
		}
		return r
	}, s)
}
