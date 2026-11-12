package tui

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// 差异高亮沿用 git diff 的红绿惯例：旧串中被删除的部分标红，
// 新串中新增的部分标绿；两串共同的部分保持原样。
var (
	diffOldStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("203"))
	diffNewStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("35"))
)

const (
	// diffCoalesceRunes 是公共段并入变更的长度上限：夹在两段变更之间的
	// 公共段不超过这么多字符时并入两侧变更，整段标红/标绿。
	diffCoalesceRunes = 2

	// diffTrimLimit 是 LCS 的 DP 矩阵规模上限（len(old)*len(new)）。超过则
	// 先截公共前后缀把矩阵缩到差异中段；单行输入几乎总是远低于上限，
	// 走完整对比——首尾的短公共段也要参与合并（见 diffSegments）。
	diffTrimLimit = 1 << 20
)

// DiffLine 对比两个单行字符串，返回带差异高亮的 (旧串, 新串)。
//
// 输入会先剥离 ANSI 样式与控制字符再对比——调用方传入的串可能已带样式
// （如着色过的路径），残留的转义序列既会干扰对比，也会破坏高亮拼接。
func DiffLine(old, new string) (string, string) {
	oldRunes := []rune(stripControl(ansi.Strip(old)))
	newRunes := []rune(stripControl(ansi.Strip(new)))
	oldSpans, newSpans := diffSegments(oldRunes, newRunes)
	return renderDiffSpans(oldSpans, diffOldStyle), renderDiffSpans(newSpans, diffNewStyle)
}

// diffSpan 是 diff 结果中的一段文本，changed 标记它是否属于差异部分。
type diffSpan struct {
	text    string
	changed bool
}

// diffOpKind 是编辑动作的类型。
type diffOpKind uint8

const (
	diffOpMatch  diffOpKind = iota // 两串共同
	diffOpDelete                   // 仅旧串有
	diffOpInsert                   // 仅新串有
)

// diffOp 是对比过程中的一次编辑动作，ops 序列按原顺序拼起两串的全部字符。
type diffOp struct {
	kind diffOpKind
	text string
}

// diffSegments 用 LCS 对比两个 rune 序列，拆成「共同/差异」交替的段落序列。
//
// 裸 LCS 追求字符数最优，会把 token 中间的重复字符当锚点拆出碎块
// （如 3000→8080 拆成 红30/白0/红0），因此回溯出编辑序列后先做一轮
// 合并（coalesceDiffOps）再落成段落。
func diffSegments(oldRunes, newRunes []rune) (oldSpans, newSpans []diffSpan) {
	prefixText, suffixText := "", ""
	if len(oldRunes)*len(newRunes) > diffTrimLimit {
		prefix, suffix := trimCommonEnds(oldRunes, newRunes)
		prefixText, suffixText = string(oldRunes[:prefix]), string(oldRunes[len(oldRunes)-suffix:])
		oldRunes, newRunes = oldRunes[prefix:len(oldRunes)-suffix], newRunes[prefix:len(newRunes)-suffix]
	}
	ops := coalesceDiffOps(diffWalk(oldRunes, newRunes))
	return spansFromOps(ops, prefixText, suffixText)
}

// trimCommonEnds 返回公共前缀与公共后缀的长度。
func trimCommonEnds(oldRunes, newRunes []rune) (prefix, suffix int) {
	for prefix < len(oldRunes) && prefix < len(newRunes) && oldRunes[prefix] == newRunes[prefix] {
		prefix++
	}
	for suffix < len(oldRunes)-prefix && suffix < len(newRunes)-prefix &&
		oldRunes[len(oldRunes)-1-suffix] == newRunes[len(newRunes)-1-suffix] {
		suffix++
	}
	return prefix, suffix
}

// diffWalk 用 DP 求 LCS 并回溯出编辑动作序列。
func diffWalk(oldRunes, newRunes []rune) []diffOp {
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

	var ops []diffOp
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case oldRunes[i] == newRunes[j]:
			appendDiffOp(&ops, diffOpMatch, string(oldRunes[i]))
			i, j = i+1, j+1
		case lcs[i+1][j] >= lcs[i][j+1]:
			appendDiffOp(&ops, diffOpDelete, string(oldRunes[i]))
			i++
		default:
			appendDiffOp(&ops, diffOpInsert, string(newRunes[j]))
			j++
		}
	}
	for ; i < n; i++ {
		appendDiffOp(&ops, diffOpDelete, string(oldRunes[i]))
	}
	for ; j < m; j++ {
		appendDiffOp(&ops, diffOpInsert, string(newRunes[j]))
	}
	return ops
}

// coalesceDiffOps 把夹在变更之间的短公共段并入变更组。公共段只有前后都是
// 变更时才并入——贴着长公共段（真正的锚点，如路径的公共目录）的边界不吞。
func coalesceDiffOps(ops []diffOp) []diffOp {
	var result []diffOp
	for i := 0; i < len(ops); {
		if ops[i].kind == diffOpMatch {
			result = append(result, ops[i])
			i++
			continue
		}
		var delText, insText strings.Builder
		hasDel, hasIns := false, false
		for i < len(ops) {
			op := ops[i]
			if op.kind == diffOpMatch {
				if utf8.RuneCountInString(op.text) <= diffCoalesceRunes && hasChangeAhead(ops, i+1) {
					delText.WriteString(op.text)
					insText.WriteString(op.text)
					i++
					continue
				}
				break
			}
			if op.kind == diffOpDelete {
				delText.WriteString(op.text)
				hasDel = true
			} else {
				insText.WriteString(op.text)
				hasIns = true
			}
			i++
		}
		if hasDel {
			appendDiffOp(&result, diffOpDelete, delText.String())
		}
		if hasIns {
			appendDiffOp(&result, diffOpInsert, insText.String())
		}
	}
	return result
}

// hasChangeAhead 报告 ops[from:] 中是否还有变更动作。
func hasChangeAhead(ops []diffOp, from int) bool {
	for _, op := range ops[from:] {
		if op.kind != diffOpMatch {
			return true
		}
	}
	return false
}

// spansFromOps 把编辑动作序列落成两串各自的段落序列；
// 前后缀是超长输入截掉后回接的两端公共文本。
func spansFromOps(ops []diffOp, prefix, suffix string) (oldSpans, newSpans []diffSpan) {
	appendDiffSpan(&oldSpans, prefix, false)
	appendDiffSpan(&newSpans, prefix, false)
	for _, op := range ops {
		switch op.kind {
		case diffOpMatch:
			appendDiffSpan(&oldSpans, op.text, false)
			appendDiffSpan(&newSpans, op.text, false)
		case diffOpDelete:
			appendDiffSpan(&oldSpans, op.text, true)
		case diffOpInsert:
			appendDiffSpan(&newSpans, op.text, true)
		}
	}
	appendDiffSpan(&oldSpans, suffix, false)
	appendDiffSpan(&newSpans, suffix, false)
	return oldSpans, newSpans
}

// appendDiffOp 追加一个编辑动作，与上一动作同类时并入，避免碎片段落。
func appendDiffOp(ops *[]diffOp, kind diffOpKind, text string) {
	if last := len(*ops) - 1; last >= 0 && (*ops)[last].kind == kind {
		(*ops)[last].text += text
		return
	}
	*ops = append(*ops, diffOp{kind: kind, text: text})
}

// appendDiffSpan 追加一段文本，与上一段同属共同/差异时并入上一段。
func appendDiffSpan(spans *[]diffSpan, text string, changed bool) {
	if text == "" {
		return
	}
	if last := len(*spans) - 1; last >= 0 && (*spans)[last].changed == changed {
		(*spans)[last].text += text
		return
	}
	*spans = append(*spans, diffSpan{text: text, changed: changed})
}

// renderDiffSpans 把段落序列拼回单个字符串，差异段落套用高亮样式。
func renderDiffSpans(spans []diffSpan, changedStyle lipgloss.Style) string {
	var b strings.Builder
	for _, span := range spans {
		if span.changed {
			b.WriteString(changedStyle.Render(span.text))
		} else {
			b.WriteString(span.text)
		}
	}
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
