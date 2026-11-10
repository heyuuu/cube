package project

import (
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"cube/internal/testfixture"
)

// TestStartRefreshTicker_FreshCacheSkippedStartupRefresh 磁盘缓存仍新鲜时，
// startRefreshTicker 启动即刷的一次应被跳过（air 热重载场景不重采）：
// 以 scanUpdatedAt 是否保持零值（未被 tickRefresh 触碰）作为「未刷新」的判据。
func TestStartRefreshTicker_FreshCacheSkippedStartupRefresh(t *testing.T) {
	s := newServiceAt(t, testfixture.NewWorkspace(t).Dir, "g1", 1)
	if err := s.snapshotCache.Save(); err != nil { // 预写一份「刚落盘」的新鲜缓存
		t.Fatalf("预写缓存失败: %v", err)
	}
	s.startRefreshTicker()
	defer s.stopRefreshTicker()

	if !s.ScanUpdatedAt().IsZero() {
		t.Fatalf("缓存新鲜时应跳过启动刷新，scanUpdatedAt 却被更新: %v", s.ScanUpdatedAt())
	}
}

// TestStartRefreshTicker_StaleCacheTriggersStartupRefresh 缓存过期（或为空）时，
// 启动仍应立即刷新一次，避免冷启动空窗。
func TestStartRefreshTicker_StaleCacheTriggersStartupRefresh(t *testing.T) {
	s := newServiceAt(t, testfixture.NewWorkspace(t).Dir, "g1", 1)
	s.startRefreshTicker()
	defer s.stopRefreshTicker()

	deadline := time.Now().Add(2 * time.Second)
	for s.ScanUpdatedAt().IsZero() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if s.ScanUpdatedAt().IsZero() {
		t.Fatal("缓存过期时应启动即刷一次，scanUpdatedAt 保持零值")
	}
}

// TestRefreshSnapshotOwnership 定向刷新只服务收录项目：scan 外的仓库不进快照
// （workbench 以任意 git 目录为输入，外来仓库进了快照也会被整表重建删掉——白采）。
func TestRefreshSnapshotOwnership(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	root := ws.Mkdir("root")
	repo := ws.MakeProjectDir(filepath.Join("root", "repo"))
	foreign := ws.MakeGitRepo("foreign") // scan root 之外的同级仓库

	s := newServiceWithRules(t, ws, []ScanRule{{Group: "g", Path: root, MaxDepth: 3}}, nil)

	e, err := s.RefreshSnapshot(repo)
	if err != nil {
		t.Fatalf("收录项目应可定向刷新: %v", err)
	}
	if e == nil || e.Main.Branch == "" {
		t.Fatalf("刷新应返回新采集的 entry: %+v", e)
	}
	if s.Snapshot(repo) != e {
		t.Fatal("快照应为刷新返回的同一条 entry")
	}

	if e, err := s.RefreshSnapshot(foreign); e != nil || err != nil {
		t.Fatalf("外来仓库跳过应返回 (nil, nil): entry=%+v, err=%v", e, err)
	}
	if s.Snapshot(foreign) != nil {
		t.Fatal("外来仓库不应进快照")
	}
}

// TestResolveProject 项目根与任意子目录直接命中（向上标定当前项目）；
// worktree / 主根 workspace 成员归并主项目；项目外目录与相对路径返回 nil。
func TestResolveProject(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	root := ws.Mkdir("root")
	repo := ws.MakeProjectDir(filepath.Join("root", "repo"))
	wtDir := ws.MakeWorktree(repo, "wt-out", "feat")

	s := newServiceWithRules(t, ws, []ScanRule{{Group: "g", Path: root, MaxDepth: 3}}, nil)

	if p := s.ResolveProject(repo); p == nil || p.Path() != repo {
		t.Fatalf("项目根应直接命中, got %+v", p)
	}
	if p := s.ResolveProject(wtDir); p == nil || p.Path() != repo {
		t.Fatalf("worktree 应归并到主项目 %s, got %+v", repo, p)
	}

	// 项目普通子目录归并（--local 在子目录标定当前项目的链路）
	if p := s.ResolveProject(filepath.Join(repo, "misc")); p == nil || p.Path() != repo {
		t.Fatalf("项目子目录应归并到主项目 %s, got %+v", repo, p)
	}

	// 主根 workspace 成员及其子目录（成员是主根直系子目录，同被覆盖）
	for _, sub := range []string{filepath.Join(".cube"), filepath.Join("server"), filepath.Join("server", "sub")} {
		if err := os.MkdirAll(filepath.Join(repo, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	member := filepath.Join(repo, "server")
	for _, dir := range []string{member, filepath.Join(member, "sub")} {
		if p := s.ResolveProject(dir); p == nil || p.Path() != repo {
			t.Fatalf("workspace 成员 %s 应归并到主项目 %s, got %+v", dir, repo, p)
		}
	}

	// 项目外目录与相对路径：不归并
	if p := s.ResolveProject(ws.Join("plain")); p != nil {
		t.Fatalf("普通目录应返回 nil, got %+v", p)
	}
	if p := s.ResolveProject("relative/sub"); p != nil {
		t.Fatalf("相对路径应按未找到处理, got %+v", p)
	}
}

// TestFindByPath FindByPath 查找（name 不再作查询键，FindByName 已移除）。
func TestFindByPathAndName(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	root := ws.Mkdir("root")
	repo := ws.MakeProjectDir(path.Join("root", "proj"))

	s := newServiceAt(t, root, "g1", 5)

	if p := s.FindByPath(repo); p == nil || p.Path() != repo {
		t.Fatalf("FindByPath 未找到: %v", s.FindByPath(repo))
	}
	if p := s.FindByPath("/nonexistent"); p != nil {
		t.Fatalf("不存在 path 应返回 nil")
	}
	// 相对路径不基于 cwd 猜测（调用方是 web，server 的 cwd 无意义），视为未找到
	if p := s.FindByPath("./proj"); p != nil {
		t.Fatalf("相对路径应返回 nil，实际: %v", p)
	}
}

// TestSearchInPath 验证 SearchInPath 的入参契约与向下搜索行为。
// path 只接受绝对路径；~/ / 相对路径的展开是出口层（cmd）职责，见 cmdutil 包的 SearchProjects 测试。
// 向上标定（子目录找所属项目）是 ResolveProject 的职责，见本文件 TestResolveProject。
func TestSearchInPath(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	root := ws.Mkdir("root")
	repo := ws.MakeProjectDir(path.Join("root", "proj"))

	s := newServiceAt(t, root, "g1", 5)

	// 项目根本身 / 父目录：向下命中
	projs := s.SearchInPath(repo)
	if len(projs) != 1 || projs[0].Path() != repo {
		t.Fatalf("SearchInPath(项目根) 应命中 %s，实际 %v", repo, projs)
	}
	if projs = s.SearchInPath(root); len(projs) != 1 || projs[0].Path() != repo {
		t.Fatalf("SearchInPath(父目录) 应向下命中 %s，实际 %v", repo, projs)
	}

	// 项目子目录：不向上命中（向上标定归 ResolveProject）
	sub := ws.Mkdir(path.Join("root", "proj", "sub"))
	if projs = s.SearchInPath(sub); len(projs) != 0 {
		t.Fatalf("SearchInPath(子目录) 不应命中，实际 %v", projs)
	}

	// 相对路径：拒绝（出口层职责，到这里即调用方传错）
	if projs = s.SearchInPath("."); len(projs) != 0 {
		t.Fatalf("SearchInPath(\".\") 应按未找到处理，实际 %v", projs)
	}
}

// TestSearchInPath_HomePrefix ~/ 前缀不再归一化（展开是出口层职责），按未找到处理。
func TestSearchInPath_HomePrefix(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// 在 home 下建 scan root + 项目，确认绝对路径可命中、~/ 同路径不可
	ws := testfixture.NewWorkspace(t)
	repoDir := home + "/root/proj"
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("建目录失败: %v", err)
	}
	testfixture.BuildGitRepo(ws.TB, repoDir, testfixture.GitRepoSpec{})

	cfg := newServiceAt(t, home+"/root", "g1", 5)
	if projs := cfg.SearchInPath(home + "/root"); len(projs) != 1 {
		t.Fatalf("SearchInPath(绝对路径) 应命中，实际 %v", projs)
	}
	if projs := cfg.SearchInPath("~/root"); len(projs) != 0 {
		t.Fatalf("SearchInPath(~/root) 应按未找到处理，实际 %v", projs)
	}
}
