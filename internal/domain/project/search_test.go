package project

import (
	"os"
	"path"
	"testing"

	"cube/internal/testfixture"
)

// TestSearchInPath 验证 SearchInPath 的入参契约与向下搜索行为。
// path 只接受绝对路径；~/ / 相对路径的展开是出口层（cmd）职责，见 cmd 包的 searchProjects 测试。
// 向上标定（子目录找所属项目）是 ResolveProject 的职责，见 targets_test。
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
