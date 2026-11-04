package cmd

import (
	"path"
	"testing"

	"cube/internal/cmd/cmdutil"
	"cube/internal/config"
	"cube/internal/domain/project"
	"cube/internal/testfixture"

	"github.com/heyuuu/x/store"
)

// newCmdServiceAt 为测试构造一个扫描指定根目录的 project.Service：
// settings.json / cache 都从 DataDir=ws.Dir 派生，与真实装配同构。
// （不沉淀进 testfixture 以避免其反向依赖 project。）
func newCmdServiceAt(t *testing.T, scanRoot, group string, maxDepth int) *project.Service {
	t.Helper()
	ws := testfixture.NewWorkspace(t)
	settingsFile := ws.Join("settings.json")
	scanRules := []project.ScanRule{
		{Group: group, Path: scanRoot, MaxDepth: maxDepth},
	}
	if err := store.SaveSection(settingsFile, "scanRules", scanRules); err != nil {
		t.Fatalf("写测试 settings.json 失败: %v", err)
	}
	return project.NewService(&config.Config{DataDir: ws.Dir})
}

// TestSearchProjects_PathQueryCwd 验证路径 query 的 cwd 解析发生在 cmd 层：
// query="." 基于当前目录解析后向下搜索（复现 cube project list . 的完整链路）。
// domain 的 SearchByPath 已不收相对路径，此测试守住出口层的解析职责不回退。
// 子目录向上标定是 PickProject 的职责（见 TestPickProject_LocalMode）。
func TestSearchProjects_PathQueryCwd(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	root := ws.Mkdir("root")
	repo := ws.MakeProjectDir(path.Join("root", "proj"))

	s := newCmdServiceAt(t, root, "g1", 5)

	// 项目根：query="." 命中
	t.Chdir(repo)
	projs, err := cmdutil.SearchProjects(s, ".")
	if err != nil {
		t.Fatalf("searchProjects(\".\") 出错: %v", err)
	}
	if len(projs) != 1 || projs[0].Path() != repo {
		t.Fatalf("searchProjects(\".\") 应命中 %s，实际 %v", repo, projs)
	}

	// 项目子目录：query="." 只向下搜索，落空（不向上标定）
	sub := ws.Mkdir(path.Join("root", "proj", "sub"))
	t.Chdir(sub)
	projs, err = cmdutil.SearchProjects(s, ".")
	if err != nil {
		t.Fatalf("searchProjects(\".\") 出错: %v", err)
	}
	if len(projs) != 0 {
		t.Fatalf("searchProjects(\".\") 在子目录不应向上命中，实际 %v", projs)
	}
}

// TestPickProject_LocalMode 验证 --local（p 入口）模式：
//   - query 缺省以 cwd 为起点定位项目（子目录内向上标定项目根）；
//   - 非 local 模式下 query 空匹配多个项目，非 TTY 无法交互选择而报错（守住原有行为）；
//   - 显式 query 不受 --local 干预，仍按名称搜索。
func TestPickProject_LocalMode(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	root := ws.Mkdir("root")
	repoA := ws.MakeProjectDir(path.Join("root", "projA"))
	repoB := ws.MakeProjectDir(path.Join("root", "projB"))
	s := newCmdServiceAt(t, root, "g1", 5)

	// 非 local：query 空匹配两个项目，走交互选择，非 TTY 下报错
	if _, err := cmdutil.PickProject(s, "", false); err == nil {
		t.Fatalf("非 local 模式下 query 空应因多项匹配无法交互而报错")
	}

	// local：query 缺省以 cwd 定位，在 projA 子目录内向上命中唯一项目
	t.Chdir(ws.Mkdir(path.Join("root", "projA", "sub")))
	proj, err := cmdutil.PickProject(s, "", true)
	if err != nil {
		t.Fatalf("local 模式下 pickProject(\"\") 出错: %v", err)
	}
	if proj.Path() != repoA {
		t.Fatalf("local 模式应命中 %s，实际 %s", repoA, proj.Path())
	}

	// local 下显式 query 不干预：仍按名称搜索命中 projB
	proj, err = cmdutil.PickProject(s, "projB", true)
	if err != nil {
		t.Fatalf("pickProject(\"projB\") 出错: %v", err)
	}
	if proj.Path() != repoB {
		t.Fatalf("显式 query 应命中 %s，实际 %s", repoB, proj.Path())
	}
}

// TestSearchProjects_QueryModes name query 不走路径解析；非路径形态按名字搜索。
func TestSearchProjects_QueryModes(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	root := ws.Mkdir("root")
	ws.MakeProjectDir(path.Join("root", "proj"))

	s := newCmdServiceAt(t, root, "g1", 5)

	// name query：不走路径分支
	projs, err := cmdutil.SearchProjects(s, "proj")
	if err != nil {
		t.Fatalf("searchProjects(name) 出错: %v", err)
	}
	if len(projs) != 1 {
		t.Fatalf("name 搜索应命中 1 个，实际 %d", len(projs))
	}

	// ~user 不满足路径形态：按名字搜索，自然落空为空列表
	projs, err = cmdutil.SearchProjects(s, "~user/x")
	if err != nil {
		t.Fatalf("searchProjects(~user/x) 出错: %v", err)
	}
	if len(projs) != 0 {
		t.Fatalf("~user/x 应按名字搜索落空，实际命中 %d", len(projs))
	}
}
