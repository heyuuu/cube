package project

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"cube/internal/testfixture"
)

// TestStartRefreshTicker_FreshCacheSkippedStartupRefresh 磁盘缓存仍新鲜时，
// StartRefreshTicker 启动即刷的一次应被跳过（air 热重载场景不重采）：
// 以 scanUpdatedAt 是否保持零值（未被 refresh 触碰）作为「未刷新」的判据。
func TestStartRefreshTicker_FreshCacheSkippedStartupRefresh(t *testing.T) {
	s := newServiceAt(t, testfixture.NewWorkspace(t).Dir, "g1", 1)
	if err := s.snapshotCache.Save(); err != nil { // 预写一份「刚落盘」的新鲜缓存
		t.Fatalf("预写缓存失败: %v", err)
	}
	s.StartRefreshTicker()
	defer s.StopRefreshTicker()

	if !s.ScanUpdatedAt().IsZero() {
		t.Fatalf("缓存新鲜时应跳过启动刷新，scanUpdatedAt 却被更新: %v", s.ScanUpdatedAt())
	}
}

// TestStartRefreshTicker_StaleCacheTriggersStartupRefresh 缓存过期（或为空）时，
// 启动仍应立即刷新一次，避免冷启动空窗。
func TestStartRefreshTicker_StaleCacheTriggersStartupRefresh(t *testing.T) {
	s := newServiceAt(t, testfixture.NewWorkspace(t).Dir, "g1", 1)
	s.StartRefreshTicker()
	defer s.StopRefreshTicker()

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

	if err := s.RefreshSnapshot(repo); err != nil {
		t.Fatalf("收录项目应可定向刷新: %v", err)
	}
	if s.Snapshot(repo) == nil {
		t.Fatal("收录项目刷新后应有快照")
	}

	if err := s.RefreshSnapshot(foreign); err != nil {
		t.Fatalf("外来仓库跳过不应报错: %v", err)
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
