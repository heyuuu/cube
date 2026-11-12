// ref 查询与解析的测试（被测实现在 refs.go；directGit/directAddRemote 供同包测试共用）。
package git

import (
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

// TestCurrentBranch 当前检出分支短名：attached 剥前缀返回；detached、非仓库为空。
func TestCurrentBranch(t *testing.T) {
	ws := newTestWorkspace(t)
	dir := ws.MakeGitRepoWith("repo", GitRepoSpec{Branch: "develop"})
	if got := CurrentBranch(dir); got != "develop" {
		t.Fatalf("CurrentBranch = %q，期望 develop", got)
	}
	directGit(t, dir, "checkout", "--detach")
	if got := CurrentBranch(dir); got != "" {
		t.Fatalf("detached 时 CurrentBranch 应为空，实际 %q", got)
	}
	if got := CurrentBranch(ws.Mkdir("empty")); got != "" {
		t.Fatalf("非仓库 CurrentBranch 应为空，实际 %q", got)
	}
}

// TestCurrentBranch_HeadOnTag HEAD 被 symbolic-ref 挂到 heads 外（tag）时为空，
// 不得把 refs/tags/* 整串误当分支名（git branch --show-current 同口径输出空）。
func TestCurrentBranch_HeadOnTag(t *testing.T) {
	ws := newTestWorkspace(t)
	dir := ws.MakeGitRepoWith("repo", GitRepoSpec{Tags: []string{"v1.0"}})
	directGit(t, dir, "symbolic-ref", "HEAD", "refs/tags/v1.0")
	if got := CurrentBranch(dir); got != "" {
		t.Fatalf("HEAD 挂在 refs/tags/* 时应为空，实际 %q", got)
	}
}

// TestBuildRef 规范全名 → Ref 值对象：三棵子树解析、谓词判定、
// 符号指针与未知形态报错（BuildRefs 依赖该错误口径做跳过）。
func TestBuildRef(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    Ref
		kind    string // local / remote / tag，断言谓词与字段一致
		wantErr bool
	}{
		{"本地分支", "refs/heads/develop",
			Ref{Name: "refs/heads/develop", ShortName: "develop", Branch: "develop"}, "local", false},
		{"带斜杠的本地分支", "refs/heads/feature/x",
			Ref{Name: "refs/heads/feature/x", ShortName: "feature/x", Branch: "feature/x"}, "local", false},
		{"tag", "refs/tags/v1.0",
			Ref{Name: "refs/tags/v1.0", ShortName: "v1.0"}, "tag", false},
		{"远程跟踪分支", "refs/remotes/origin/master",
			Ref{Name: "refs/remotes/origin/master", ShortName: "origin/master", Remote: "origin", Branch: "master"}, "remote", false},
		{"带斜杠的远程分支", "refs/remotes/origin/feature/x",
			Ref{Name: "refs/remotes/origin/feature/x", ShortName: "origin/feature/x", Remote: "origin", Branch: "feature/x"}, "remote", false},
		{"远端 HEAD 符号指针", "refs/remotes/origin/HEAD", Ref{}, "", true},
		{"只有 remote 无分支", "refs/remotes/origin", Ref{}, "", true},
		{"未知子树", "refs/stash", Ref{}, "", true},
		{"短名不被接受", "origin/master", Ref{}, "", true},
		{"空串", "", Ref{}, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := buildRef(tc.in)
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
				t.Errorf("buildRef = %+v，期望 %+v", got, tc.want)
			}
			switch {
			case tc.kind == "local" && !got.IsLocal():
				t.Errorf("%+v 应判定为 local", got)
			case tc.kind == "remote" && !got.IsRemote():
				t.Errorf("%+v 应判定为 remote", got)
			case tc.kind == "tag" && !got.IsTag():
				t.Errorf("%+v 应判定为 tag", got)
			}
		})
	}
}

// TestRefs 全量 ref 清单：三类 namespace 的 Ref 值对象 + 远端 HEAD 符号指针剔除。
// current 属 HEAD 状态，由 TestHeadRef 单独覆盖。
func TestRefs(t *testing.T) {
	ws := newTestWorkspace(t)
	dir := ws.MakeGitRepoWith("repo", GitRepoSpec{
		Branch: "develop",
		Tags:   []string{"v1.0"},
	})
	directGit(t, dir, "update-ref", "refs/remotes/origin/master", "refs/heads/develop")
	directGit(t, dir, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/master")

	refs, err := Refs(dir)
	if err != nil {
		t.Fatalf("Refs 出错: %v", err)
	}
	if !reflect.DeepEqual(refs.Locals, []Ref{
		{Name: "refs/heads/develop", ShortName: "develop", Branch: "develop"},
	}) {
		t.Fatalf("locals = %+v", refs.Locals)
	}
	if !reflect.DeepEqual(refs.Remotes, []Ref{
		{Name: "refs/remotes/origin/master", ShortName: "origin/master", Remote: "origin", Branch: "master"},
	}) {
		t.Fatalf("remotes = %+v（origin/HEAD 符号指针应被剔除）", refs.Remotes)
	}
	if !reflect.DeepEqual(refs.Tags, []Ref{
		{Name: "refs/tags/v1.0", ShortName: "v1.0"},
	}) {
		t.Fatalf("tags = %+v", refs.Tags)
	}
}

// TestRefs_NonRepo 非仓库目录返回零值不报错（降级约定）。
func TestRefs_NonRepo(t *testing.T) {
	ws := newTestWorkspace(t)
	refs, err := Refs(ws.Mkdir("empty"))
	if err != nil {
		t.Fatalf("非仓库 Refs 不应报错: %v", err)
	}
	if refs.Locals != nil || refs.Remotes != nil || refs.Tags != nil {
		t.Fatalf("非仓库应返回零值: %+v", refs)
	}
}

// TestRemoteTags ls-remote 实查远端 tag：与本地 refs/tags 两套口径——远端独有的
// tag 可见、本地独有的不出现；annotated tag 双行合并（Sha≠Commit），轻量 tag 同值。
func TestRemoteTags(t *testing.T) {
	ws := newTestWorkspace(t)
	// 远端仓库：轻量 v1.0 / v9 + annotated rel-1.0
	src := ws.MakeGitRepoWith("src", GitRepoSpec{Branch: "main", Tags: []string{"v1.0", "v9"}})
	directGit(t, src, "tag", "-a", "rel-1.0", "-m", "annotated")

	repo := ws.MakeGitRepo("repo")
	directAddRemote(t, repo, "origin", src)
	// 本地独有 tag 只进 refs/tags，不应出现在远端查询结果里
	directGit(t, repo, "tag", "local-only")

	tags, err := RemoteTags(repo, "origin")
	if err != nil {
		t.Fatalf("RemoteTags 出错: %v", err)
	}
	var names []string
	byName := make(map[string]RemoteTag)
	for _, tag := range tags {
		names = append(names, tag.Name)
		byName[tag.Name] = tag
	}
	if !reflect.DeepEqual(names, []string{"rel-1.0", "v1.0", "v9"}) {
		t.Fatalf("远端 tag 名 = %v（按名排序，不含本地独有 local-only）", names)
	}
	rel := byName["rel-1.0"]
	if rel.Sha == rel.Commit || len(rel.Sha) != 40 || len(rel.Commit) != 40 {
		t.Fatalf("annotated tag 应 Sha≠Commit（tag 对象 / 剥壳 commit）: %+v", rel)
	}
	if v := byName["v1.0"]; v.Sha != v.Commit || len(v.Sha) != 40 {
		t.Fatalf("轻量 tag 应 Sha==Commit: %+v", v)
	}
}

// TestRemoteTags_NonRepoAndBadRemote 非仓库降级 (nil,nil)；不存在的 remote 名
// 被 git 当作路径解析失败，报真实错误而非静默空清单。
func TestRemoteTags_NonRepoAndBadRemote(t *testing.T) {
	ws := newTestWorkspace(t)
	if tags, err := RemoteTags(ws.Mkdir("empty"), "origin"); err != nil || tags != nil {
		t.Fatalf("非仓库应返回 (nil,nil)，实际 (%v,%v)", tags, err)
	}
	repo := ws.MakeGitRepo("repo")
	if _, err := RemoteTags(repo, "no-such-remote"); err == nil {
		t.Fatal("不存在的 remote 名应报错")
	}
}

// TestParseLsRemoteTags 解析 `ls-remote --tags` 输出：剥壳行回填 Commit、
// 孤剥壳行与坏行跳过、空输出为 nil。
func TestParseLsRemoteTags(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []RemoteTag
	}{
		{"空输出", "", nil},
		{"轻量 tag 单行", "abc\trefs/tags/v1.0\n",
			[]RemoteTag{{Name: "v1.0", Sha: "abc", Commit: "abc"}}},
		{"annotated 双行合并", "tagobj\trefs/tags/r\ncommit\trefs/tags/r^{}\n",
			[]RemoteTag{{Name: "r", Sha: "tagobj", Commit: "commit"}}},
		{"孤剥壳行跳过", "commit\trefs/tags/x^{}\n", nil},
		{"无 tab 坏行跳过", "garbage\n", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := parseLsRemoteTags(c.in); !reflect.DeepEqual(got, c.want) {
				t.Fatalf("parseLsRemoteTags(%q) = %+v，期望 %+v", c.in, got, c.want)
			}
		})
	}
}

// TestRefs_UpstreamTrack Refs 单进程顺带取回本地分支的 upstream 跟踪信息：
// 分叉分支的 ahead/behind、同步分支零值、upstream 指向已删远端分支的 gone。
func TestRefs_UpstreamTrack(t *testing.T) {
	ws := newTestWorkspace(t)
	dir := ws.MakeGitRepoWith("repo", GitRepoSpec{Branch: "main"})
	directAddRemote(t, dir, "origin", "/tmp/nonexistent.git")

	// main 与 origin/main 分叉：本地领先 1（"local ahead"），远端独有 1（"remote ahead"）
	directGit(t, dir, "update-ref", "refs/remotes/origin/main", "refs/heads/main")
	directGit(t, dir, "branch", "-u", "origin/main", "main")
	directGit(t, dir, "branch", "tmpbase")
	directGit(t, dir, "commit", "--allow-empty", "-m", "local ahead")
	directGit(t, dir, "checkout", "tmpbase")
	directGit(t, dir, "commit", "--allow-empty", "-m", "remote ahead")
	directGit(t, dir, "update-ref", "refs/remotes/origin/main", "HEAD")
	directGit(t, dir, "checkout", "main")
	directGit(t, dir, "branch", "-D", "tmpbase")

	// synced：upstream 跟踪 ref 停在自身同一 commit → ahead/behind 全 0
	directGit(t, dir, "branch", "synced")
	directGit(t, dir, "update-ref", "refs/remotes/origin/synced", "refs/heads/synced")
	directGit(t, dir, "branch", "-u", "origin/synced", "synced")

	// orphan：upstream 指向不存在的跟踪 ref——branch -u 对不存在的 ref 会拒绝，只能 config 手工配
	directGit(t, dir, "branch", "orphan")
	directGit(t, dir, "config", "branch.orphan.remote", "origin")
	directGit(t, dir, "config", "branch.orphan.merge", "refs/heads/deleted-on-remote")

	refs, err := Refs(dir)
	if err != nil {
		t.Fatalf("Refs 出错: %v", err)
	}
	byBranch := make(map[string]Ref)
	for _, r := range refs.Locals {
		byBranch[r.Branch] = r
	}

	main := byBranch["main"]
	if main.Upstream != "refs/remotes/origin/main" || main.Ahead != 1 || main.Behind != 1 || main.Gone {
		t.Fatalf("main = %+v，期望 upstream=origin/main 且 ahead/behind = 1/1", main)
	}
	synced := byBranch["synced"]
	if synced.Upstream != "refs/remotes/origin/synced" || synced.Ahead != 0 || synced.Behind != 0 || synced.Gone {
		t.Fatalf("synced = %+v，期望 in-sync（upstream 指向、0/0、非 gone）", synced)
	}
	orphan := byBranch["orphan"]
	if orphan.Upstream != "refs/remotes/origin/deleted-on-remote" || !orphan.Gone || orphan.Ahead != 0 || orphan.Behind != 0 {
		t.Fatalf("orphan = %+v，期望 gone（upstream 指向、ahead/behind 无意义为 0）", orphan)
	}

	// 末行是 refs/remotes/origin/synced（refname 排序在最后）：曾因整体 TrimSpace
	// 咬掉末行尾随分隔符导致其 Upstream 错位，断言 remote ref 一律不带 upstream 信息
	for _, r := range refs.Remotes {
		if r.Upstream != "" || r.Ahead != 0 || r.Behind != 0 || r.Gone {
			t.Fatalf("remote ref %s 不应带 upstream 信息: %+v", r.Name, r)
		}
	}
}

// TestParseUpstreamTrack 解析 %(upstream:track) 的输出形态；坏数据降级零值。
func TestParseUpstreamTrack(t *testing.T) {
	cases := []struct {
		in     string
		ahead  int
		behind int
		gone   bool
	}{
		{"", 0, 0, false}, // 已同步或未配置 upstream
		{"[ahead 2]", 2, 0, false},
		{"[behind 3]", 0, 3, false},
		{"[ahead 2, behind 3]", 2, 3, false},
		{"[gone]", 0, 0, true},
		{"garbage", 0, 0, false},   // 无法识别 → 零值
		{"[ahead x]", 0, 0, false}, // 非数字 → 零值
	}
	for _, c := range cases {
		ahead, behind, gone := parseUpstreamTrack(c.in)
		if ahead != c.ahead || behind != c.behind || gone != c.gone {
			t.Errorf("parseUpstreamTrack(%q) = (%d,%d,%v)，期望 (%d,%d,%v)",
				c.in, ahead, behind, gone, c.ahead, c.behind, c.gone)
		}
	}
}

// TestHeadRef attached → refs/heads 全名；detached / 非仓库 → 空串。
func TestHeadRef(t *testing.T) {
	ws := newTestWorkspace(t)
	dir := ws.MakeGitRepoWith("repo", GitRepoSpec{Branch: "develop"})
	if got := HeadRef(dir); got != "refs/heads/develop" {
		t.Fatalf("HeadRef = %q，期望 refs/heads/develop", got)
	}
	directGit(t, dir, "checkout", "--detach")
	if got := HeadRef(dir); got != "" {
		t.Fatalf("detached 时 HeadRef 应为空，实际 %q", got)
	}
	if got := HeadRef(ws.Mkdir("empty")); got != "" {
		t.Fatalf("非仓库 HeadRef 应为空，实际 %q", got)
	}
}

// TestDefaultBranch_OriginHEAD origin/HEAD 已设置时直接取其指向的分支。
func TestDefaultBranch_OriginHEAD(t *testing.T) {
	ws := newTestWorkspace(t)
	dir := ws.MakeGitRepoWith("repo", GitRepoSpec{Branch: "develop"})
	directGit(t, dir, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/develop")

	db, err := DefaultBranch(dir)
	if err != nil {
		t.Fatalf("DefaultBranch 出错: %v", err)
	}
	if db != "develop" {
		t.Fatalf("DefaultBranch = %q，期望 develop（origin/HEAD 指向）", db)
	}
}

// TestDefaultBranch_NoOriginRemote 无 origin remote 时按本地 master/main 兜底。
func TestDefaultBranch_NoOriginRemote(t *testing.T) {
	ws := newTestWorkspace(t)
	dir := ws.MakeGitRepoWith("repo", GitRepoSpec{Branch: "main"})

	db, err := DefaultBranch(dir)
	if err != nil {
		t.Fatalf("DefaultBranch 出错: %v", err)
	}
	if db != "main" && db != "master" {
		t.Fatalf("DefaultBranch = %q，期望 main 或 master", db)
	}
}

// TestDefaultBranch_NonRepo 非仓库返回空不报错。
func TestDefaultBranch_NonRepo(t *testing.T) {
	ws := newTestWorkspace(t)
	dir := ws.Mkdir("empty")
	db, err := DefaultBranch(dir)
	if err != nil || db != "" {
		t.Fatalf("非仓库 DefaultBranch 应返回空，实际 (%q,%v)", db, err)
	}
}

// TestAheadBehindRemote_NoRemote 无 remote 时（origin/xxx ref 不存在）返回 (0,0,nil)。
func TestAheadBehindRemote_NoRemote(t *testing.T) {
	ws := newTestWorkspace(t)
	dir := ws.MakeGitRepo("repo")
	ahead, behind, err := AheadBehindRemote(dir, "master", "origin", "master")
	if err != nil {
		t.Fatalf("无 remote AheadBehind 不应报错: %v", err)
	}
	if ahead != 0 || behind != 0 {
		t.Fatalf("无 remote AheadBehind 应为 (0,0)，实际 (%d,%d)", ahead, behind)
	}
}

// TestAheadBehindRemote_Diverged 分叉场景：ahead/behind 方向正确（本地独有 / 远端独有）。
func TestAheadBehindRemote_Diverged(t *testing.T) {
	ws := newTestWorkspace(t)
	dir := ws.MakeGitRepoWith("repo", GitRepoSpec{Branch: "master"})

	// 同步点 → 本地 master 加 1 commit；base 分支上加另 1 commit 当作远端状态
	directGit(t, dir, "update-ref", "refs/remotes/origin/master", "refs/heads/master")
	directGit(t, dir, "branch", "base")
	directGit(t, dir, "commit", "--allow-empty", "-m", "local ahead")
	directGit(t, dir, "checkout", "base")
	directGit(t, dir, "commit", "--allow-empty", "-m", "remote ahead")
	directGit(t, dir, "update-ref", "refs/remotes/origin/master", "HEAD")
	directGit(t, dir, "checkout", "master")

	ahead, behind, err := AheadBehindRemote(dir, "master", "origin", "master")
	if err != nil {
		t.Fatalf("AheadBehindRemote 出错: %v", err)
	}
	if ahead != 1 || behind != 1 {
		t.Fatalf("ahead/behind = %d/%d，期望 1/1（双方各独有 1 个 commit）", ahead, behind)
	}
}

// TestAheadBehindRemote_SlashBranch 斜杠分支（feature/fix-bug）端到端：
// 本地领先 remote 2 个 commit；同时验证 Refs 对斜杠远程分支的解析
// （info -v 分支同步宽表按短名交集 + 每格调 AheadBehindRemote，依赖这两个行为）。
func TestAheadBehindRemote_SlashBranch(t *testing.T) {
	ws := newTestWorkspace(t)
	dir := ws.MakeGitRepoWith("repo", GitRepoSpec{Branch: "master"})

	// remote 跟踪引用停在 master 当前位置，本地 feature/fix-bug 在其上追加 2 commit
	directGit(t, dir, "branch", "feature/fix-bug")
	directGit(t, dir, "update-ref", "refs/remotes/origin/feature/fix-bug", "refs/heads/master")
	directGit(t, dir, "checkout", "feature/fix-bug")
	directGit(t, dir, "commit", "--allow-empty", "-m", "ahead 1")
	directGit(t, dir, "commit", "--allow-empty", "-m", "ahead 2")

	ahead, behind, err := AheadBehindRemote(dir, "feature/fix-bug", "origin", "feature/fix-bug")
	if err != nil {
		t.Fatalf("AheadBehindRemote 出错: %v", err)
	}
	if ahead != 2 || behind != 0 {
		t.Fatalf("ahead/behind = %d/%d，期望 2/0", ahead, behind)
	}

	// Refs 解析：refs/remotes/origin/feature/fix-bug → Remote/Branch 拆分
	repoRefs, err := Refs(dir)
	if err != nil {
		t.Fatalf("Refs 出错: %v", err)
	}
	found := false
	for _, ref := range repoRefs.Remotes {
		if ref.Remote == "origin" && ref.Branch == "feature/fix-bug" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Refs.Remotes %v 不含 {origin, feature/fix-bug}", repoRefs.Remotes)
	}
}

// directGit 在 dir 下直接调 git（带测试 user 配置），fixture 未覆盖的场景用。
func directGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git",
		append([]string{"-c", "user.email=test@cube.local", "-c", "user.name=cube-test"}, args...)...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v in %s 失败: %v\n%s", args, dir, err, out)
	}
}

// directAddRemote 直接调 git remote add（fixture 只支持 origin，额外 remote 在此加）。
func directAddRemote(t *testing.T, dir, name, url string) {
	t.Helper()
	cmd := exec.Command("git",
		"-c", "user.email=test@cube.local",
		"-c", "user.name=cube-test",
		"remote", "add", name, url)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git remote add %s 失败: %v\n%s", name, err, out)
	}
}

// TestParseCountPair 解析 rev-list --left-right --count 的 "ahead\tbehind" 输出。
func TestParseCountPair(t *testing.T) {
	cases := []struct {
		in     string
		ahead  int
		behind int
		ok     bool
	}{
		{"2\t0\n", 2, 0, true},
		{"0\t3\n", 0, 3, true},
		{"12\t34", 12, 34, true},
		{"1", 0, 0, false},      // 缺 tab
		{"a\tb\n", 0, 0, false}, // 非数字
		{"", 0, 0, false},
	}
	for _, c := range cases {
		ahead, behind, ok := parseCountPair(c.in)
		if ok != c.ok || ahead != c.ahead || behind != c.behind {
			t.Errorf("parseCountPair(%q) = (%d,%d,%v)，期望 (%d,%d,%v)",
				c.in, ahead, behind, ok, c.ahead, c.behind, c.ok)
		}
	}
}

// TestHeadSha 返回 HEAD 完整 sha；非仓库降级为空值。
func TestHeadSha(t *testing.T) {
	ws := newTestWorkspace(t)
	dir := ws.MakeGitRepoWith("repo", GitRepoSpec{EmptyCommitCount: 2})

	sha, err := HeadSha(dir)
	if err != nil {
		t.Fatalf("HeadSha 出错: %v", err)
	}
	if len(sha) != 40 {
		t.Errorf("HEAD sha 长度 = %d，期望 40（%q）", len(sha), sha)
	}

	if sha, err := HeadSha(ws.Mkdir("not-a-repo")); err != nil || sha != "" {
		t.Errorf("非仓库 HeadSha 应返回 (\"\",nil)，实际 (%q,%v)", sha, err)
	}
}

// TestParentSha 父提交解析正确；根提交无父时报错（调用方决定降级）。
func TestParentSha(t *testing.T) {
	ws := newTestWorkspace(t)
	dir := ws.MakeGitRepoWith("repo", GitRepoSpec{EmptyCommitCount: 2})

	head, err := HeadSha(dir)
	if err != nil {
		t.Fatalf("HeadSha 出错: %v", err)
	}
	parent, err := ParentSha(dir, head)
	if err != nil {
		t.Fatalf("ParentSha 出错: %v", err)
	}
	if len(parent) != 40 || parent == head {
		t.Errorf("父提交 sha 异常: %q", parent)
	}
	// EmptyCommitCount=2 时 parent 已是根提交：再取父应报错
	if _, err := ParentSha(dir, parent); err == nil {
		t.Error("根提交再取父应报错")
	}
}

// Clean：删除未跟踪文件与目录；.gitignore 忽略的文件保留。
func TestClean(t *testing.T) {
	ws := newTestWorkspace(t)
	repo := ws.MakeGitRepoWith("repo", GitRepoSpec{EmptyCommitCount: 1})

	untracked := ws.Join("repo", "new.txt")
	untrackedDir := ws.Join("repo", "newdir", "inner.txt")
	ignored := ws.Join("repo", "ignored.log")
	ws.WriteFile("repo/new.txt", []byte("x"))
	ws.WriteFile("repo/newdir/inner.txt", []byte("x"))
	ws.WriteFile("repo/.gitignore", []byte("ignored.log\n"))
	ws.WriteFile("repo/ignored.log", []byte("x"))
	runGit(t, repo, "add", ".gitignore")
	runGit(t, repo, "commit", "-m", "gitignore")

	if err := Clean(repo); err != nil {
		t.Fatalf("Clean 应成功: %v", err)
	}
	for _, p := range []string{untracked, untrackedDir} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("未跟踪路径应被删除: %s", p)
		}
	}
	if _, err := os.Stat(ignored); err != nil {
		t.Errorf("被 .gitignore 忽略的文件应保留: %v", err)
	}
}

// BranchDelete：未合并非 force 拒绝（中文原因）、force 成功；已合并分支直接删。
func TestBranchDelete(t *testing.T) {
	ws := newTestWorkspace(t)
	repo := ws.MakeGitRepoWith("repo", GitRepoSpec{EmptyCommitCount: 2})

	// 建分支并在其上追加 commit → 相对 HEAD 未合并（祖先方向的分支算已合并）
	runGit(t, repo, "branch", "old")
	runGit(t, repo, "checkout", "old")
	runGit(t, repo, "commit", "--allow-empty", "-m", "on old")
	runGit(t, repo, "checkout", "-")

	if err := BranchDelete(repo, "old", false); err == nil {
		t.Fatalf("未合并分支非 force 删除应被拒绝")
	} else if !strings.Contains(err.Error(), "尚未合并") {
		t.Errorf("错误应为中文未合并提示, got: %v", err)
	}
	if err := BranchDelete(repo, "old", true); err != nil {
		t.Fatalf("未合并分支 force 删除应成功: %v", err)
	}

	// 指向 HEAD 的分支视为已合并，-d 直接删
	runGit(t, repo, "branch", "merged")
	if err := BranchDelete(repo, "merged", false); err != nil {
		t.Fatalf("已合并分支删除应成功: %v", err)
	}
}

// BranchAdd：建分支（HEAD/tag 基点）、重名/非法名/基点不存在的中文拒绝。
func TestBranchAdd(t *testing.T) {
	ws := newTestWorkspace(t)
	repo := ws.MakeGitRepoWith("repo", GitRepoSpec{Tags: []string{"v1.0"}})

	if err := BranchAdd(repo, "feat", ""); err != nil {
		t.Fatalf("以 HEAD 为基点建分支应成功: %v", err)
	}
	if err := BranchAdd(repo, "from-tag", "v1.0"); err != nil {
		t.Fatalf("以 tag 为基点建分支应成功: %v", err)
	}
	refs, err := Refs(repo)
	if err != nil {
		t.Fatalf("Refs 报错: %v", err)
	}
	names := map[string]bool{}
	for _, r := range refs.Locals {
		names[r.Branch] = true
	}
	if !names["feat"] || !names["from-tag"] {
		t.Fatalf("新分支应出现在 refs 中: %v", names)
	}

	if err := BranchAdd(repo, "feat", ""); err == nil || !strings.Contains(err.Error(), "已存在") {
		t.Errorf("重名应报中文错误, got: %v", err)
	}
	if err := BranchAdd(repo, "a b", ""); err == nil || !strings.Contains(err.Error(), "不是合法的分支名") {
		t.Errorf("非法名应报中文错误, got: %v", err)
	}
	if err := BranchAdd(repo, "ok", "no-such-thing"); err == nil || !strings.Contains(err.Error(), "基点不存在") {
		t.Errorf("基点不存在应报中文错误, got: %v", err)
	}
}
