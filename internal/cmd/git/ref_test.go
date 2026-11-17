// allRefInfos 的测试（被测实现在 ref.go；runGitAt 供本文件建 bare remote 等超出
// GitRepoSpec 形态的仓库，测试代码允许直接 exec git）。
package git

import (
	"os/exec"
	"strings"
	"testing"

	"cube/internal/testfixture"
)

// runGitAt 在 dir 下执行 git 命令，失败时 t.Fatalf。
func runGitAt(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s 在 %s 失败: %v\n输出: %s", strings.Join(args, " "), dir, err, out)
	}
}

// refInfoIndex 把 allRefInfos 结果按 name 索引，方便断言。
func refInfoIndex(infos []refInfo) map[string]refInfo {
	index := make(map[string]refInfo, len(infos))
	for _, info := range infos {
		index[info.name] = info
	}
	return index
}

// assertRefKind 断言 name 的本地/远端 kind；ref 不存在时直接失败。
func assertRefKind(t *testing.T, infos map[string]refInfo, name string, wantLocal, wantRemote RefKind) {
	t.Helper()
	info, ok := infos[name]
	if !ok {
		t.Fatalf("ref 列表中缺少 %s，实际: %v", name, infos)
	}
	if info.localKind != wantLocal || info.remoteKind != wantRemote {
		t.Fatalf("%s 期望 localKind=%d remoteKind=%d，实际 localKind=%d remoteKind=%d",
			name, wantLocal, wantRemote, info.localKind, info.remoteKind)
	}
}

// TestAllRefInfos pin allRefInfos 的本地/远端 ref 配对行为。
// 历史 bug：远端分支以含 remote 前缀的 ShortName（"origin/develop"）作 key，
// 与本地分支名配不上对，push/pull 候选里因此没有任何分支；且未过滤目标
// remote 之外的跟踪分支，多 remote 时同名分支会被误判「远端已存在」。
func TestAllRefInfos(t *testing.T) {
	ws := testfixture.NewWorkspace(t)

	// 布局：origin bare（develop / master / remote-only 分支 + v1.0.0 tag）、
	// upstream bare（仅 upstream-only 分支）；工作仓库本地有 develop / master /
	// local-only 分支与 local-tag tag。
	remoteDir := ws.Mkdir("remote.git")
	runGitAt(t, remoteDir, "init", "--bare", "-b", "develop")

	repoDir := ws.Mkdir("repo")
	testfixture.BuildGitRepo(t, repoDir, testfixture.GitRepoSpec{
		Branch:    "develop",
		Tags:      []string{"local-tag"},
		RemoteURL: remoteDir,
	})
	runGitAt(t, repoDir, "push", "origin", "develop")
	runGitAt(t, repoDir, "checkout", "-b", "master")
	runGitAt(t, repoDir, "push", "origin", "master")
	runGitAt(t, repoDir, "tag", "v1.0.0")
	runGitAt(t, repoDir, "push", "origin", "v1.0.0")
	runGitAt(t, repoDir, "checkout", "develop")
	runGitAt(t, repoDir, "branch", "local-only")
	runGitAt(t, remoteDir, "branch", "remote-only", "develop")

	// 第二个 remote：先推后删本地分支，留下远端副本与本地跟踪 ref
	// （refs/remotes/upstream/upstream-only），验证非目标 remote 被过滤。
	upstreamDir := ws.Mkdir("upstream.git")
	runGitAt(t, upstreamDir, "init", "--bare")
	runGitAt(t, repoDir, "remote", "add", "upstream", upstreamDir)
	runGitAt(t, repoDir, "branch", "upstream-only")
	runGitAt(t, repoDir, "push", "upstream", "upstream-only")
	runGitAt(t, repoDir, "branch", "-D", "upstream-only")

	t.Run("origin 视角", func(t *testing.T) {
		infos, err := allRefInfos(repoDir, "origin")
		if err != nil {
			t.Fatalf("allRefInfos 失败: %v", err)
		}
		index := refInfoIndex(infos)

		// 本地与远端同名分支须配成同一条（bug 时拆成 "develop" +
		// "origin/develop" 两条，push 候选条件永假）
		assertRefKind(t, index, "develop", RefBranch, RefBranch)
		assertRefKind(t, index, "master", RefBranch, RefBranch)

		// 远端独有 / 本地独有分支
		assertRefKind(t, index, "remote-only", RefNotExists, RefBranch)
		assertRefKind(t, index, "local-only", RefBranch, RefNotExists)

		// tag：两侧同名配对；本地独有 tag 远端不存在（push 候选口径）
		assertRefKind(t, index, "v1.0.0", RefTag, RefTag)
		assertRefKind(t, index, "local-tag", RefTag, RefNotExists)

		// 不得出现带 remote 前缀的幽灵条目 / 非 origin 的跟踪分支
		for _, ghost := range []string{"origin/develop", "origin/master", "upstream/upstream-only", "upstream-only"} {
			if _, ok := index[ghost]; ok {
				t.Fatalf("origin 视角不应出现 %s", ghost)
			}
		}
	})

	t.Run("upstream 视角", func(t *testing.T) {
		infos, err := allRefInfos(repoDir, "upstream")
		if err != nil {
			t.Fatalf("allRefInfos 失败: %v", err)
		}
		index := refInfoIndex(infos)

		// 「远端存在」相对目标 remote：upstream-only 属于 upstream；
		// develop 本地存在但 upstream 没有，远端应为不存在
		assertRefKind(t, index, "upstream-only", RefNotExists, RefBranch)
		assertRefKind(t, index, "develop", RefBranch, RefNotExists)
	})
}
