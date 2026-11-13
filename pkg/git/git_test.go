package git

import (
	"reflect"
	"strings"
	"testing"
)

func TestFindGitRoot_AtRoot(t *testing.T) {
	ws := newTestWorkspace(t)
	repo := ws.MakeGitRepo("repo")

	root, ok := FindGitRoot(repo)
	if !ok {
		t.Fatalf("仓库根目录 FindGitRoot 应返回 ok=true")
	}
	if root != repo {
		t.Fatalf("FindGitRoot = %q，期望 %q", root, repo)
	}
}

func TestFindGitRoot_InSubdir(t *testing.T) {
	ws := newTestWorkspace(t)
	repo := ws.MakeGitRepo("repo")
	sub := ws.Mkdir("repo/src/deep/nested")

	root, ok := FindGitRoot(sub)
	if !ok {
		t.Fatalf("子目录 FindGitRoot 应返回 ok=true")
	}
	if root != repo {
		t.Fatalf("FindGitRoot = %q，期望 %q", root, repo)
	}
}

func TestFindGitRoot_NotARepo(t *testing.T) {
	// 注意：此 case 必须用系统临时目录而非 testfixture 的 runtime/test/，
	// 因为 runtime/test/ 在仓库内，FindGitRoot 向上探测会命中仓库根的 .git。
	dir := t.TempDir()
	_, ok := FindGitRoot(dir)
	if ok {
		t.Fatalf("非仓库目录 FindGitRoot 应返回 ok=false")
	}
}

func TestFindGitRoot_WorktreeStyleDotGitFile(t *testing.T) {
	// .git 是文件（worktree），FindGitRoot 只看 .git 存在与否，应能识别
	ws := newTestWorkspace(t)
	dir := ws.Mkdir("wt")
	ws.WriteFile("wt/.git", []byte("gitdir: /somewhere"))

	root, ok := FindGitRoot(dir)
	if !ok || root != dir {
		t.Fatalf("worktree 风格 .git 文件 FindGitRoot 异常：root=%q ok=%v", root, ok)
	}
}

// TestFetchTag 单 tag 拉取：落到本地 refs/tags/<tag> 且指向与远端一致；
// 指向已拉历史之外（游离 commit）的 tag 不被 auto-follow 带下。
func TestFetchTag(t *testing.T) {
	ws := newTestWorkspace(t)
	src := ws.MakeGitRepoWith("src", GitRepoSpec{Branch: "main", Tags: []string{"v1.0"}})
	// v9 打在游离 commit 上（不在 main 历史内，tag 保住对象）——指向已拉取
	// 历史的 tag 会被 auto-follow 顺带拉下，故用游离 commit 隔离
	directGit(t, src, "checkout", "--detach")
	directGit(t, src, "commit", "--allow-empty", "-m", "dangling")
	directGit(t, src, "tag", "v9")
	directGit(t, src, "checkout", "main")
	repo := ws.MakeGitRepo("repo")
	directAddRemote(t, repo, "origin", src)

	if err := FetchTag(repo, "origin", "v1.0"); err != nil {
		t.Fatalf("FetchTag 出错: %v", err)
	}

	var names []string
	for _, ref := range refTags(t, repo) {
		names = append(names, ref.ShortName)
	}
	if !reflect.DeepEqual(names, []string{"v1.0"}) {
		t.Fatalf("本地 tag = %v，应只有 v1.0", names)
	}
	if want, got := tagSha(t, src, "v1.0"), tagSha(t, repo, "v1.0"); want != got {
		t.Fatalf("本地 v1.0 = %s，应与远端一致 %s", got, want)
	}
}

// TestFetchAllTags 全量 tag 拉取：远端全部 tag 落本地，本地独有 tag 不受影响。
func TestFetchAllTags(t *testing.T) {
	ws := newTestWorkspace(t)
	src := ws.MakeGitRepoWith("src", GitRepoSpec{Branch: "main", Tags: []string{"v1.0", "v9"}})
	repo := ws.MakeGitRepo("repo")
	directAddRemote(t, repo, "origin", src)
	directGit(t, repo, "tag", "local-only")

	if err := FetchAllTags(repo, "origin"); err != nil {
		t.Fatalf("FetchAllTags 出错: %v", err)
	}

	var names []string
	for _, ref := range refTags(t, repo) {
		names = append(names, ref.ShortName)
	}
	if !reflect.DeepEqual(names, []string{"local-only", "v1.0", "v9"}) {
		t.Fatalf("本地 tag = %v，应 = 远端全部 + 本地独有（for-each-ref 按字母序）", names)
	}
}

// TestFetchTag_ClobberRejected 远端移动了本地已有的同名 tag 时，两种拉取都被
// clobber 拒绝（would clobber existing tag，非 0 退出码）且本地指向保持不变。
func TestFetchTag_ClobberRejected(t *testing.T) {
	ws := newTestWorkspace(t)
	src := ws.MakeGitRepoWith("src", GitRepoSpec{Branch: "main", EmptyCommitCount: 2, Tags: []string{"v1.0"}})
	repo := ws.MakeGitRepo("repo")
	directAddRemote(t, repo, "origin", src)

	if err := FetchTag(repo, "origin", "v1.0"); err != nil {
		t.Fatalf("首次 FetchTag 出错: %v", err)
	}
	before := tagSha(t, repo, "v1.0")

	// 远端把 v1.0 从 HEAD 移到前一个 commit（Tags 打在最后一个 commit 上）
	directGit(t, src, "tag", "-f", "v1.0", "HEAD~1")
	if err := FetchTag(repo, "origin", "v1.0"); err == nil {
		t.Fatalf("远端 tag 已移动时 FetchTag 应被 clobber 拒绝，实际成功")
	}
	if err := FetchAllTags(repo, "origin"); err == nil {
		t.Fatalf("远端 tag 已移动时 FetchAllTags 应被 clobber 拒绝，实际成功")
	}
	if after := tagSha(t, repo, "v1.0"); after != before {
		t.Fatalf("被拒绝后本地 v1.0 不应变动: %s -> %s", before, after)
	}
}

// refTags 读本地 tag 清单（Ref.ShortName 列表），供拉取测试断言。
func refTags(t *testing.T, dir string) []Ref {
	t.Helper()
	refs, err := Refs(dir)
	if err != nil {
		t.Fatalf("Refs 出错: %v", err)
	}
	return refs.Tags
}

// tagSha 读 refs/tags/<tag> 的指向，供拉取测试断言。
func tagSha(t *testing.T, dir string, tag string) string {
	t.Helper()
	out, err := runOut(dir, "rev-parse", "refs/tags/"+tag)
	if err != nil {
		t.Fatalf("rev-parse %s 出错: %v", tag, err)
	}
	return strings.TrimSpace(out)
}
