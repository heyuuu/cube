package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestDiffLine(t *testing.T) {
	tests := []struct {
		name     string
		old, new string
		wantOld  string // 剥离样式后的期望原文（顺带验证输入样式被剥掉）
		wantNew  string
		oldRed   string // 旧串中期望标红的片段；空 = 应无任何高亮
		newGreen string // 新串中期望标绿的片段；空 = 应无任何高亮
	}{
		{
			name:    "完全相同",
			old:     "server.port = 3000",
			new:     "server.port = 3000",
			wantOld: "server.port = 3000",
			wantNew: "server.port = 3000",
		},
		{
			name:     "中段变更",
			old:      "feature/login",
			new:      "feature/logout",
			wantOld:  "feature/login",
			wantNew:  "feature/logout",
			oldRed:   "in",
			newGreen: "out",
		},
		{
			name:     "纯新增",
			old:      "~/Code/lab/cube",
			new:      "~/Code/lab/cube-web",
			wantOld:  "~/Code/lab/cube",
			wantNew:  "~/Code/lab/cube-web",
			newGreen: "-web",
		},
		{
			name:    "纯删除",
			old:     "~/Code/lab/cube-web",
			new:     "~/Code/lab/cube",
			wantOld: "~/Code/lab/cube-web",
			wantNew: "~/Code/lab/cube",
			oldRed:  "-web",
		},
		{
			name:     "中文按字符对比",
			old:      "路径未同步",
			new:      "路径不同步",
			wantOld:  "路径未同步",
			wantNew:  "路径不同步",
			oldRed:   "未",
			newGreen: "不",
		},
		{
			name:     "空串对比",
			old:      "",
			new:      "新增内容",
			wantOld:  "",
			wantNew:  "新增内容",
			newGreen: "新增内容",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOld, gotNew := DiffLine(tt.old, tt.new)
			assertDiffLine(t, "旧串", gotOld, tt.wantOld, tt.oldRed, diffOldStyle)
			assertDiffLine(t, "新串", gotNew, tt.wantNew, tt.newGreen, diffNewStyle)
		})
	}
}

// assertDiffLine 校验一条渲染结果：剥离样式后与期望原文一致；
// 期望片段恰好被指定样式包住；无期望片段时整串不应含 ANSI。
func assertDiffLine(t *testing.T, label, got, wantPlain, wantChanged string, style lipgloss.Style) {
	t.Helper()
	if stripped := ansi.Strip(got); stripped != wantPlain {
		t.Errorf("%s 剥离样式后 = %q, want %q", label, stripped, wantPlain)
	}
	if wantChanged == "" {
		if strings.Contains(got, "\x1b") {
			t.Errorf("%s 无差异却带样式: %q", label, got)
		}
		return
	}
	if !strings.Contains(got, style.Render(wantChanged)) {
		t.Errorf("%s 应含高亮片段 %q, got %q", label, wantChanged, got)
	}
}

func TestDiffLineStripsStyledInput(t *testing.T) {
	styled := lipgloss.NewStyle().Foreground(lipgloss.Color("51")).Render("~/Code/lab/cube")
	gotOld, gotNew := DiffLine(styled, "~/Code/lab/cube-web")

	if gotOld != "~/Code/lab/cube" {
		t.Errorf("旧串应剥掉输入自带样式 = %q, want ~/Code/lab/cube", gotOld)
	}
	if !strings.Contains(gotNew, diffNewStyle.Render("-web")) {
		t.Errorf("新串应含绿色高亮片段 \"-web\": %q", gotNew)
	}
}

func TestDiffSegments(t *testing.T) {
	tests := []struct {
		name         string
		old, new     string
		wantOldSpans []diffSpan
		wantNewSpans []diffSpan
	}{
		{
			name:         "无公共字符",
			old:          "in",
			new:          "out",
			wantOldSpans: []diffSpan{{"in", true}},
			wantNewSpans: []diffSpan{{"out", true}},
		},
		{
			name:         "纯新增",
			old:          "",
			new:          "-web",
			wantNewSpans: []diffSpan{{"-web", true}},
		},
		{
			name:         "中文按字符对比",
			old:          "未同步",
			new:          "不同步",
			wantOldSpans: []diffSpan{{"未", true}, {"同步", false}},
			wantNewSpans: []diffSpan{{"不", true}, {"同步", false}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldSpans, newSpans := diffSegments([]rune(tt.old), []rune(tt.new))
			if !equalDiffSpans(oldSpans, tt.wantOldSpans) {
				t.Errorf("oldSpans = %+v, want %+v", oldSpans, tt.wantOldSpans)
			}
			if !equalDiffSpans(newSpans, tt.wantNewSpans) {
				t.Errorf("newSpans = %+v, want %+v", newSpans, tt.wantNewSpans)
			}
		})
	}
}

func equalDiffSpans(got, want []diffSpan) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
