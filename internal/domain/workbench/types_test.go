package workbench

import (
	"strings"
	"testing"
)

// TestWorkbenchSourceParse 校验 TreeSource 解析纯函数（HTTP 契约的一部分）。
// 合法用例同时断言 String() 往返一致（写方序列化与解析成对）。
func TestWorkbenchSourceParse(t *testing.T) {
	sha40 := strings.Repeat("a1", 20)
	sha64 := strings.Repeat("b2", 32)
	cases := []struct {
		name    string
		in      string
		want    TreeSource
		wantErr bool
	}{
		{"commit 40 位 sha", "commit://" + sha40, TreeSource{Type: SourceTypeCommit, ID: sha40}, false},
		{"commit 64 位 sha（sha256 仓库）", "commit://" + sha64, TreeSource{Type: SourceTypeCommit, ID: sha64}, false},
		{"ref 全名-分支", "ref://refs/heads/master", TreeSource{Type: SourceTypeRef, ID: "refs/heads/master"}, false},
		{"ref 全名-tag", "ref://refs/tags/v1.0", TreeSource{Type: SourceTypeRef, ID: "refs/tags/v1.0"}, false},
		{"ref 全名-远程跟踪", "ref://refs/remotes/origin/dev", TreeSource{Type: SourceTypeRef, ID: "refs/remotes/origin/dev"}, false},
		{"ref 全名-开放子树", "ref://refs/pull/123/head", TreeSource{Type: SourceTypeRef, ID: "refs/pull/123/head"}, false},
		{"ref 相对路径短名", "ref://heads/master", TreeSource{Type: SourceTypeRef, ID: "heads/master"}, false},
		{"ref 短名-HEAD", "ref://HEAD", TreeSource{Type: SourceTypeRef, ID: "HEAD"}, false},
		{"ref 短名-远程叶名", "ref://origin/dev", TreeSource{Type: SourceTypeRef, ID: "origin/dev"}, false},
		{"worktree 绝对路径", "worktree:///tmp/x", TreeSource{Type: SourceTypeWorktree, ID: "/tmp/x"}, false},
		{"worktree 路径含冒号不歧义", "worktree:///tmp/a://b", TreeSource{Type: SourceTypeWorktree, ID: "/tmp/a://b"}, false},
		{"缺 scheme", "master", TreeSource{}, true},
		{"未知 scheme", "branch://main", TreeSource{}, true},
		{"空 id", "ref://", TreeSource{}, true},
		{"commit 非 hex", "commit://" + strings.Repeat("z", 40), TreeSource{}, true},
		{"commit 长度不符", "commit://abc123", TreeSource{}, true},
		{"rev 表达式-波浪号", "ref://HEAD~2", TreeSource{}, true},
		{"rev 表达式-upstream", "ref://@{u}", TreeSource{}, true},
		{"rev 表达式-插入符", "ref://master^", TreeSource{}, true},
		{"ref 短名含空格", "ref://a b", TreeSource{}, true},
		{"ref 短名含冒号", "ref://a:b", TreeSource{}, true},
		{"ref 短名连续点", "ref://master..dev", TreeSource{}, true},
		{"ref 短名 lock 结尾", "ref://x.lock", TreeSource{}, true},
		{"ref 短名前导点", "ref://.hidden", TreeSource{}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseTreeSource(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("应报错: %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("不应报错: %v", err)
			}
			if got != tc.want {
				t.Errorf("解析不符: got %+v, want %+v", got, tc.want)
			}
			if got.String() != tc.in {
				t.Errorf("String 往返不一致: got %q, want %q", got.String(), tc.in)
			}
		})
	}
}
