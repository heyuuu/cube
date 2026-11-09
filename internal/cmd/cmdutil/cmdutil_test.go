package cmdutil

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestExtendPath 基于 cwd 的绝对化糖层：相对路径以进程当前目录为基准。
func TestExtendPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir := t.TempDir()
	t.Chdir(dir) // 固定 cwd，测试不依赖外部目录

	cases := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"单点指 cwd", ".", dir, false},
		{"./sub 基于 cwd", "./sub", filepath.Join(dir, "sub"), false},
		{"../x 越过 cwd", "../x", filepath.Clean(filepath.Join(dir, "..", "x")), false},
		{"绝对路径覆盖 cwd", "/abs", "/abs", false},
		{"~ 展开优先于 cwd", "~/code", filepath.Join(home, "code"), false},
		{"空串报错", "", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ExtendPath(c.in)
			if c.wantErr {
				if err == nil {
					t.Fatalf("ExtendPath(%q) 期望报错，实际返回 %q", c.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ExtendPath(%q) 出错: %v", c.in, err)
			}
			if got != c.want {
				t.Fatalf("ExtendPath(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestPrettyPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("os.UserHomeDir() 失败: %v", err)
	}

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"home 目录下转 ~", filepath.Join(home, "code", "app"), "~/code/app"},
		{"home 目录本身转 ~", home, "~"},
		{"home 目录带斜杠转 ~", home + "/", "~"},
		{"home 目录外保持绝对", "/usr/local/bin", "/usr/local/bin"},
		{"相对路径原样返回", "foo/bar", "foo/bar"},
		{"空串原样返回", "", ""},

		// 脏输入：冗余分隔符 / . / .. 按 filepath.Clean 规范化后再转 ~
		{"home 下双斜杠折叠", filepath.Join(home, "code") + "//app", "~/code/app"},
		{"home 下 . 段清理", filepath.Join(home, "code", ".", "app"), "~/code/app"},
		{"home 下 .. 回退", filepath.Join(home, "code", "sub", "..", "app"), "~/code/app"},
		{"home 下 .. 回退", filepath.Join(home, "..", "code"), filepath.Clean(filepath.Join(home, "..", "code"))},
		{"home 外双斜杠折叠", "/usr//local/bin", "/usr/local/bin"},
		{"home 外 . 段清理", "/usr/./local/bin", "/usr/local/bin"},
		{"home 外 .. 回退", "/usr/local/../bin", "/usr/bin"},
		{"相对路径 .. 回退-1", "a/./b", "a/b"},
		{"相对路径 .. 回退-2", "a/../../b", "../b"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := PrettyPath(c.in); got != c.want {
				t.Fatalf("PrettyPath(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestPrettyTime(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		t    time.Time
		want string
	}{
		{"刚刚（0 秒前）", now, "0秒前"},
		{"59 秒前", now.Add(-59 * time.Second), "59秒前"},
		{"1 分钟前", now.Add(-60 * time.Second), "1分钟前"},
		{"59 分钟前", now.Add(-59 * time.Minute), "59分钟前"},
		{"1 小时前", now.Add(-60 * time.Minute), "1小时前"},
		{"23 小时前", now.Add(-23 * time.Hour), "23小时前"},
		{"1 天前", now.Add(-24 * time.Hour), "1天前"},
		{"29 天前", now.Add(-29 * 24 * time.Hour), "29天前"},
		{"1 个月前", now.Add(-30 * 24 * time.Hour), "1月前"},
		{"11 个月前", now.Add(-330 * 24 * time.Hour), "11月前"},
		{"1 年前", now.Add(-365 * 24 * time.Hour), "1年前"},
		// 未来时间多留 100ms 余量：prettyTime 执行晚于 now 若干 μs，
		// 否则正好卡在桶边界会被向下取整（30秒后 → 29秒后）
		{"未来 30 秒", now.Add(30*time.Second + 100*time.Millisecond), "30秒后"},
		{"未来 5 分钟", now.Add(5*time.Minute + 100*time.Millisecond), "5分钟后"},
		{"未来 2 小时", now.Add(2*time.Hour + 100*time.Millisecond), "2小时后"},
		{"未来 3 天", now.Add(72*time.Hour + 100*time.Millisecond), "3天后"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PrettyTime(tt.t); got != tt.want {
				t.Errorf("prettyTime() = %q, want %q", got, tt.want)
			}
		})
	}
}
