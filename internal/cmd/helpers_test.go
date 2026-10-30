package cmd

import (
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
					t.Fatalf("AbsPath(%q) 期望报错，实际返回 %q", c.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("AbsPath(%q) 出错: %v", c.in, err)
			}
			if got != c.want {
				t.Fatalf("AbsPath(%q) = %q, want %q", c.in, got, c.want)
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
			if got := prettyTime(tt.t); got != tt.want {
				t.Errorf("prettyTime() = %q, want %q", got, tt.want)
			}
		})
	}
}
