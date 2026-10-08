package cmd

import (
	"strings"
	"testing"
)

// TestParseCliTpl <模板源> 三形态解析表（提案 1045：@ 前缀 / 本地路径 / 裸名三分）。
func TestParseCliTpl(t *testing.T) {
	cases := []struct {
		name         string
		in           string
		wantSource   string // 期望的 sourceRef（路径形态为展开后的绝对路径前缀校验）
		wantTpl      string
		wantErr      string // 空 = 应成功
		absPathCheck bool   // true 时 sourceRef 应为绝对路径
	}{
		// @ 前缀：命名源引用
		{"@source/tpl", "@core/go-cli", "core", "go-cli", "", false},
		{"@source 交互选模板", "@core", "core", "", "", false},
		{"@ 后缺 source", "@", "", "", "@ 后缺 source 名", false},
		{"@source/tpl 多余段", "@core/go-cli/x", "", "", "模板名不能包含 /", false},
		{"@/tpl 缺 source", "@/go-cli", "", "", "@ 后缺 source 名", false},

		// 本地路径：./ ~/ / 开头
		{"相对路径 ./", "./tpl", "", "", "", true},
		{"家目录 ~/tpl", "~/tpl", "", "", "", true},
		{"绝对路径", "/tmp/tpl", "/tmp/tpl", "", "", false},

		// 裸名：默认源
		{"裸名 → 默认源", "go-cli", "core", "go-cli", "", false},
		{"裸名带 / 提前拦截", "core/go-cli", "", "", "@source/模板名", false},
		{"空串", "", "", "", "模板源不能为空", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			source, tpl, err := parseCliTpl(c.in)
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("期望错误含 %q, got %v", c.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("应成功, got %v", err)
			}
			if c.absPathCheck {
				if !strings.HasPrefix(source, "/") {
					t.Fatalf("路径形态应展开为绝对路径, got %q", source)
				}
			} else if source != c.wantSource {
				t.Fatalf("source = %q, want %q", source, c.wantSource)
			}
			if tpl != c.wantTpl {
				t.Fatalf("tpl = %q, want %q", tpl, c.wantTpl)
			}
		})
	}
}
