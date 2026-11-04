package project

import (
	"os"
	"path"
	"testing"

	"cube/internal/testfixture"
)

// TestSearchByPath_SearchByPath 验证 SearchByPath 的入参契约与向下搜索行为。
// path 只接受绝对路径；~/ / 相对路径的展开是出口层（cmd）职责，见 cmd 包的 searchProjects 测试。
// 向上标定（子目录找所属项目）是 ResolveProject 的职责，见 targets_test。
func TestSearchByPath_SearchByPath(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	root := ws.Mkdir("root")
	repo := ws.MakeProjectDir(path.Join("root", "proj"))

	s := newServiceAt(t, root, "g1", 5)

	// 项目根本身 / 父目录：向下命中
	projs := s.SearchByPath(repo)
	if len(projs) != 1 || projs[0].Path() != repo {
		t.Fatalf("SearchByPath(项目根) 应命中 %s，实际 %v", repo, projs)
	}
	if projs = s.SearchByPath(root); len(projs) != 1 || projs[0].Path() != repo {
		t.Fatalf("SearchByPath(父目录) 应向下命中 %s，实际 %v", repo, projs)
	}

	// 项目子目录：不向上命中（向上标定归 ResolveProject）
	sub := ws.Mkdir(path.Join("root", "proj", "sub"))
	if projs = s.SearchByPath(sub); len(projs) != 0 {
		t.Fatalf("SearchByPath(子目录) 不应命中，实际 %v", projs)
	}

	// 相对路径：拒绝（出口层职责，到这里即调用方传错）
	if projs = s.SearchByPath("."); len(projs) != 0 {
		t.Fatalf("SearchByPath(\".\") 应按未找到处理，实际 %v", projs)
	}
}

// TestSearchByPath_HomePrefix ~/ 前缀不再归一化（展开是出口层职责），按未找到处理。
func TestSearchByPath_HomePrefix(t *testing.T) {
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
	if projs := cfg.SearchByPath(home + "/root"); len(projs) != 1 {
		t.Fatalf("SearchByPath(绝对路径) 应命中，实际 %v", projs)
	}
	if projs := cfg.SearchByPath("~/root"); len(projs) != 0 {
		t.Fatalf("SearchByPath(~/root) 应按未找到处理，实际 %v", projs)
	}
}
