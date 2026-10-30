package pathkit

import (
	"os"
	"path/filepath"
	"testing"
)

// ---------- PrettyPath ----------

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
