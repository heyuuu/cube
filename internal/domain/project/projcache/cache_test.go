package projcache

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"cube/internal/testfixture"
)

// TestLoad_EmptyDir 目录尚不存在（冷启动）→ 空缓存。Load 只读不写不建目录；
// 目录创建懒化在 Save 侧（store.WriteFileAtomic 自带 MkdirAll）。
func TestLoad_EmptyDir(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	dir := ws.Join("not-exist-yet")

	c := Load(dir)
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("Load 不应创建目录: %v", err)
	}
	// 空缓存：Get 任意路径未命中
	if _, ok := c.Get("/any"); ok {
		t.Fatalf("空缓存 Get 应未命中")
	}
}

// TestLoad_FileMissing projcache.json 不存在 → 空缓存（冷启动降级）。
func TestLoad_FileMissing(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	dir := ws.Mkdir("cache")

	c := Load(dir)
	if _, ok := c.Get("/any"); ok {
		t.Fatalf("无 projcache.json 时应返回空缓存")
	}
}

// TestSaveLoad_RoundTrip Save 后 Load 能读回 entries。
func TestSaveLoad_RoundTrip(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	dir := ws.Mkdir("cache")

	// 写入
	c1 := Load(dir)
	c1.entries["/proj/a"] = &Entry{
		Remotes:       []RemoteInfo{{Name: "origin", URL: "git@x:a"}},
		DefaultBranch: "main",
		Branches:      []LocalBranchInfo{{Name: "refs/heads/main", ShortName: "main", Ahead: 3, Behind: 1}},
		Main:          WorktreeInfo{Path: "/proj/a", Branch: "main"},
	}
	if err := c1.Save(); err != nil {
		t.Fatalf("Save 失败: %v", err)
	}

	// 重新 Load
	c2 := Load(dir)
	got, ok := c2.Get("/proj/a")
	if !ok {
		t.Fatalf("Save 后 Load 应命中")
	}
	if got.OriginURL() != "git@x:a" || got.Main.Branch != "main" ||
		got.DefaultBranchAhead() != 3 || got.DefaultBranchBehind() != 1 {
		t.Fatalf("读回数据异常: %+v", got)
	}
}

// TestLoad_CorruptFile 损坏的 projcache.json 当文件不存在丢弃（降级空缓存，不备份）。
func TestLoad_CorruptFile(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	dir := ws.Mkdir("cache")
	ws.WriteFile(filepath.Join("cache", cacheFileName), []byte("{not valid json"))

	c := Load(dir)
	if _, ok := c.Get("/any"); ok {
		t.Fatalf("损坏文件应返回空缓存")
	}
}

// TestLoad_VersionMismatch 版本不符的 projcache.json 当文件不存在丢弃（不备份、不 lenient 混入）。
// 老数据按新结构反序列化会得到缺字段全零值的部分数据，比空缓存更糟。
func TestLoad_VersionMismatch(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	dir := ws.Mkdir("cache")
	old := `{"version":` + fmt.Sprintf("%d", cacheVersion-1) + `,"entries":{"/p/a":{"repoUrl":"x","currentBranch":"main"}}}`
	ws.WriteFile(filepath.Join("cache", cacheFileName), []byte(old))

	c := Load(dir)
	if _, ok := c.Get("/p/a"); ok {
		t.Fatal("旧版本数据应被整体丢弃")
	}
	// 区别于损坏文件：版本不符不备份（文件是合法的，只是过时）
	matches, _ := filepath.Glob(filepath.Join(dir, cacheFileName+".corrupt-*"))
	if len(matches) != 0 {
		t.Fatal("版本不符不应走损坏备份路径")
	}
}

// TestCache_Get 内存态 Get 命中/未命中。
func TestCache_Get(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	c := Load(ws.Mkdir("cache"))

	c.entries["/p"] = &Entry{DefaultBranch: "x", Branches: []LocalBranchInfo{{ShortName: "x", Ahead: 1}}}
	if e, ok := c.Get("/p"); !ok || e.DefaultBranchAhead() != 1 {
		t.Fatalf("Get 命中异常: %+v ok=%v", e, ok)
	}
	if _, ok := c.Get("/missing"); ok {
		t.Fatalf("未命中 Get 应返回 ok=false")
	}
}

// TestSortLocalBranches master/main/develop 置前，其余按短名字典序。
func TestSortLocalBranches(t *testing.T) {
	branch := func(name string) LocalBranchInfo {
		return LocalBranchInfo{Name: "refs/heads/" + name, ShortName: name}
	}
	got := sortLocalBranches([]LocalBranchInfo{
		branch("zeta"), branch("develop"), branch("main"),
		branch("alpha"), branch("master"), branch("feature/x"),
	})
	want := []string{"master", "main", "develop", "alpha", "feature/x", "zeta"}
	if len(got) != len(want) {
		t.Fatalf("排序结果长度异常: %v", got)
	}
	for i, b := range got {
		if b.ShortName != want[i] {
			t.Fatalf("排序结果 = %v，期望 %v", got, want)
		}
	}
}

// TestCollectEntry_RealRepo 真实仓库采集能拿到 branch（主工作区信息）。
func TestCollectEntry_RealRepo(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	repo := ws.MakeGitRepoWith("repo", testfixture.GitRepoSpec{Branch: "main"})

	e, err := collectEntry(repo)
	if err != nil {
		t.Fatalf("collectEntry 不应返回 error: %v", err)
	}
	if e == nil {
		t.Fatalf("collectEntry 不应返回 nil")
	}
	if main := e.Main; main.Branch != "main" {
		t.Fatalf("主工作区分支不符（Entry.Main）: %+v", main)
	}
}

// TestCollectEntry_Remotes 全部 remote 进快照（多 remote 项目的 forge 匹配依赖此数据），
// origin 地址经 OriginURL 派生。
func TestCollectEntry_Remotes(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	repo := ws.MakeGitRepoWith("repo", testfixture.GitRepoSpec{Branch: "main", RemoteURL: "git@github.com:heyuuu/repo.git"})
	// 补一个非 origin remote（指向另一 host）
	if out, err := exec.Command("git", "-C", repo, "remote", "add", "gitee", "git@gitee.com:heyuuu/repo.git").CombinedOutput(); err != nil {
		t.Fatalf("添加 gitee remote 失败: %v %s", err, out)
	}

	e, err := collectEntry(repo)
	if err != nil || e == nil {
		t.Fatalf("collectEntry 失败: %v", err)
	}
	if e.OriginURL() != "git@github.com:heyuuu/repo.git" {
		t.Fatalf("OriginURL = %q（origin 口径不变）", e.OriginURL())
	}
	hosts := make([]string, 0, len(e.Remotes))
	for _, r := range e.Remotes {
		hosts = append(hosts, r.Name+"="+r.URL)
	}
	if len(e.Remotes) != 2 || e.Remotes[0].Name != "gitee" || e.Remotes[1].Name != "origin" {
		t.Fatalf("应含 gitee 与 origin 两条 remote（按名排序）: %v", hosts)
	}
}

// TestCollectEntry_NonRepo 非仓库目录：任一 git 子命令失败即整条失败（全有或全无，v4）。
func TestCollectEntry_NonRepo(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	dir := ws.Mkdir("empty")

	if e, err := collectEntry(dir); err == nil {
		t.Fatalf("非仓库 collectEntry 应返回 error（不接受残缺数据）, got %+v", e)
	}
}

// TestCollectEntry_Worktrees 工作区枚举含主工作区（Worktrees[0]）+ linked worktree（1032 归并、v4 主区并入）。
func TestCollectEntry_Worktrees(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	repo := ws.MakeGitRepoWith("repo", testfixture.GitRepoSpec{Branch: "main"})
	ws.MakeWorktree(repo, "wt-hot", "hotfix")
	ws.MakeWorktree(repo, "wt-feat", "feat")

	e, err := collectEntry(repo)
	if err != nil {
		t.Fatalf("collectEntry 不应返回 error: %v", err)
	}
	// Main 是主工作区（分支 = 仓库当前分支，路径经符号链接规范化），Worktrees 只含 linked
	if len(e.Worktrees) != 2 {
		t.Fatalf("应含 2 个 linked worktree, got %d (%+v)", len(e.Worktrees), e.Worktrees)
	}
	canonicalRepo, err := filepath.EvalSymlinks(repo)
	if err != nil {
		t.Fatalf("解析真实路径失败: %v", err)
	}
	if e.Main.Path != canonicalRepo || e.Main.Branch != "main" {
		t.Fatalf("Main 应为主工作区: %+v", e.Main)
	}
	// 分支名 + 路径落在快照里；git 输出路径经符号链接规范化，比较前同样求值
	canonicalWt, err := filepath.EvalSymlinks(ws.Join("wt-hot"))
	if err != nil {
		t.Fatalf("解析真实路径失败: %v", err)
	}
	byBranch := map[string]WorktreeInfo{}
	for _, wt := range e.Worktrees {
		byBranch[wt.Branch] = wt
	}
	if wt, ok := byBranch["hotfix"]; !ok || wt.Path != canonicalWt {
		t.Fatalf("hotfix worktree 字段不符: %+v ok=%v", wt, ok)
	}
	if _, ok := byBranch["feat"]; !ok {
		t.Fatalf("缺少 feat worktree: %+v", e.Worktrees)
	}
}

// TestCollectEntry_WorktreeDirty dirty 按工作区各自探测（迁入 WorktreeInfo）：
// 主区与各 linked worktree 的 dirty 互不影响。
func TestCollectEntry_WorktreeDirty(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	repo := ws.MakeGitRepoWith("repo", testfixture.GitRepoSpec{Branch: "main", MakeDirty: true})
	ws.MakeWorktree(repo, "wt-clean", "clean")
	ws.MakeWorktree(repo, "wt-dirty", "dirt")
	if err := os.WriteFile(ws.Join("wt-dirty", "new.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	e, err := collectEntry(repo)
	if err != nil {
		t.Fatalf("collectEntry 不应返回 error: %v", err)
	}
	if !e.Main.Dirty {
		t.Fatalf("主区 MakeDirty 应为 dirty: %+v", e.Main)
	}
	byBranch := map[string]WorktreeInfo{}
	for _, wt := range e.Worktrees {
		byBranch[wt.Branch] = wt
	}
	if byBranch["clean"].Dirty {
		t.Fatalf("clean worktree 不应 dirty: %+v", byBranch["clean"])
	}
	if !byBranch["dirt"].Dirty {
		t.Fatalf("含未跟踪文件的 worktree 应 dirty: %+v", byBranch["dirt"])
	}
}

// TestCollectEntry_WorktreePruned 目录已删但 git 元数据未 prune 的 worktree 不进快照（幽灵目标）。
func TestCollectEntry_WorktreePruned(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	repo := ws.MakeGitRepoWith("repo", testfixture.GitRepoSpec{Branch: "main"})
	ws.MakeWorktree(repo, "wt-hot", "hotfix")
	ws.MakeWorktree(repo, "wt-gone", "gone")

	// 删除 wt-gone 目录但不跑 git worktree prune——git 仍会列出它
	if err := os.RemoveAll(ws.Join("wt-gone")); err != nil {
		t.Fatalf("删除 worktree 目录失败: %v", err)
	}

	e, err := collectEntry(repo)
	if err != nil {
		t.Fatalf("collectEntry 不应返回 error: %v", err)
	}
	if len(e.Worktrees) != 1 {
		t.Fatalf("失联 worktree 应被过滤，只留 hotfix（Main 另计）, got %+v", e.Worktrees)
	}
	if e.Worktrees[0].Branch != "hotfix" {
		t.Fatalf("留下的应是 hotfix: %+v", e.Worktrees[0])
	}
}

// TestRefresh_RealRepo Refresh 真实仓库后能读到采集结果。
func TestRefresh_RealRepo(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	cacheDir := ws.Mkdir("cache")
	repo := ws.MakeGitRepoWith("repo", testfixture.GitRepoSpec{Branch: "develop"})

	c := Load(cacheDir)
	if err := c.Refresh([]string{repo}); err != nil {
		t.Fatalf("Refresh 失败: %v", err)
	}
	e, ok := c.Get(repo)
	if !ok {
		t.Fatalf("Refresh 后 Get 应命中")
	}
	if e.Main.Branch != "develop" {
		t.Fatalf("Refresh 采集主工作区分支 = %q，期望 develop", e.Main.Branch)
	}

	// projcache.json 应已落盘
	if _, err := os.Stat(filepath.Join(cacheDir, cacheFileName)); err != nil {
		t.Fatalf("Refresh 后 projcache.json 应存在: %v", err)
	}
}

// TestRefresh_EmptyPaths 空路径列表也成功（整表清空 + 落盘）。
func TestRefresh_EmptyPaths(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	c := Load(ws.Mkdir("cache"))
	if err := c.Refresh(nil); err != nil {
		t.Fatalf("空 Refresh 不应报错: %v", err)
	}
}

// RefreshOne 定向刷新单项目：新建 worktree 后即时进快照，不影响其他条目。
func TestRefreshOne(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	repo := ws.MakeGitRepo("repo")
	other := ws.MakeGitRepo("other")
	c := Load(ws.Mkdir("cache"))

	if err := c.RefreshOne(repo); err != nil {
		t.Fatalf("RefreshOne 报错: %v", err)
	}
	e, ok := c.Get(repo)
	if !ok || e.Main.Branch == "" {
		t.Fatalf("刷新后应有该项目的 entry: %+v", e)
	}
	if _, ok := c.Get(other); ok {
		t.Fatalf("定向刷新不应影响其他项目")
	}

	ws.MakeWorktree(repo, "wt-hot", "hotfix")
	if err := c.RefreshOne(repo); err != nil {
		t.Fatalf("二次刷新报错: %v", err)
	}
	e, _ = c.Get(repo)
	if len(e.Worktrees) != 1 || e.Worktrees[0].Branch != "hotfix" {
		t.Fatalf("worktree 应已进快照（Worktrees 只含 linked）: %+v", e.Worktrees)
	}

	// 非 git 目录：与整表 Refresh 语义一致，整条失败不落盘（全有或全无）
	if err := c.RefreshOne(ws.Join("plain")); err == nil {
		t.Fatal("非仓库按约定应整条失败（不接受残缺数据）")
	}
	if e, ok := c.Get(repo); !ok || e.Main.Branch == "" {
		t.Fatalf("其他条目不受影响: %+v", e)
	}
}
