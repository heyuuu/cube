package project

import (
	"path"
	"slices"
	"testing"

	"cube/internal/testfixture"
)

// newServiceAt 为测试构造一个扫描指定根目录的 Service。
// 就地写而非放进 testfixture，是因为 testfixture 不能 import project（会导致 git 等底层包测试循环依赖）。
func newServiceAt(t *testing.T, scanRoot, group string, maxDepth int) *Service {
	t.Helper()
	return newServiceWithRules(t, testfixture.NewWorkspace(t),
		[]ScanRule{{Group: group, Path: scanRoot, MaxDepth: maxDepth}}, nil)
}

// newServiceWithRules 把 scan/clone 规则写进测试目录的 settings.json 再构造 Service（规则数据源即 settings 节）。
func newServiceWithRules(t *testing.T, ws *testfixture.Workspace, scan []ScanRule, clone []CloneRule) *Service {
	t.Helper()
	saveRules(t, ws, scan, clone)
	return NewService(ws.Cfg)
}

// saveRules 分节写 scanRule / cloneRule 两个 settings 节（nil 的部分跳过不写）。
func saveRules(t *testing.T, ws *testfixture.Workspace, scan []ScanRule, clone []CloneRule) {
	t.Helper()
	if scan != nil {
		ws.SaveSection(scanRulesSection, scan)
	}
	if clone != nil {
		ws.SaveSection(cloneRulesSection, clone)
	}
}

// TestScan_SingleProject 单个 git 项目被识别。
func TestScan_SingleProject(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	root := ws.Mkdir("root")
	repo := ws.MakeGitRepoWith(path.Join("root", "myproj"), testfixture.GitRepoSpec{})

	s := newServiceAt(t, root, "g1", 5)
	projs := s.Projects()

	if len(projs) != 1 {
		t.Fatalf("扫描出 %d 个 project，期望 1：%v", len(projs), projs)
	}
	if projs[0].Path() != repo {
		t.Fatalf("project path = %q，期望 %q", projs[0].Path(), repo)
	}
	if projs[0].Group() != "g1" {
		t.Fatalf("group = %q，期望 g1", projs[0].Group())
	}
}

// TestScan_NestedProjects 多个嵌套 git 项目都被识别。
func TestScan_NestedProjects(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	root := ws.Mkdir("root")
	p1 := ws.MakeProjectDir(path.Join("root", "a"))
	p2 := ws.MakeProjectDir(path.Join("root", "b", "c"))

	s := newServiceAt(t, root, "g", 5)
	projs := s.Projects()

	if len(projs) != 2 {
		t.Fatalf("扫描出 %d 个，期望 2", len(projs))
	}
	paths := map[string]bool{}
	for _, p := range projs {
		paths[p.Path()] = true
	}
	if !paths[p1] || !paths[p2] {
		t.Fatalf("扫描结果不全：%v", paths)
	}
}

// TestScan_MaxDepth 深度剪枝：根下直接子项为深度 0，-1 不限深。
func TestScan_MaxDepth(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	root := ws.Mkdir("root")
	ws.MakeProjectDir(path.Join("root", "proj"))                       // 深度 0
	ws.MakeProjectDir(path.Join("root", "x", "y", "deep"))             // 深度 2
	ws.MakeProjectDir(path.Join("root", "a", "b", "c", "d", "e", "f")) // 深度 5

	for _, c := range []struct {
		maxDepth int
		want     int
	}{
		{0, 1},  // 仅根下一层：只有 proj
		{1, 1},  // deep 在深度 2，不可达
		{2, 2},  // deep 可达，更深的 f 不可达
		{-1, 3}, // 不限深：全部
	} {
		s := newServiceAt(t, root, "g", c.maxDepth)
		if got := len(s.Projects()); got != c.want {
			t.Fatalf("MaxDepth=%d 应扫到 %d 个，实际 %d", c.maxDepth, c.want, got)
		}
	}
}

// TestScan_NonProjectDirIgnored 不含 .git 的目录被忽略。
func TestScan_NonProjectDirIgnored(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	root := ws.Mkdir("root")
	ws.Mkdir(path.Join("root", "plain-dir")) // 无 .git
	ws.MakeProjectDir(path.Join("root", "real-proj"))

	s := newServiceAt(t, root, "g", 5)
	if len(s.Projects()) != 1 {
		t.Fatalf("应只识别 git project，实际 %d", len(s.Projects()))
	}
}

// TestScan_DotUnderscoreDirSkipped 以 . 或 _ 开头的目录被跳过。
func TestScan_DotUnderscoreDirSkipped(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	root := ws.Mkdir("root")
	// .git 命名的「伪 project」目录应被跳过（.git 本身也含 .git？不会，scan.go 直接 SkipDir）
	ws.Mkdir(path.Join("root", ".hidden"))
	ws.MakeProjectDir(path.Join("root", ".hidden", "inner")) // 在 .hidden 下，应被跳过
	ws.MakeProjectDir(path.Join("root", "normal"))

	s := newServiceAt(t, root, "g", 5)
	if len(s.Projects()) != 1 {
		t.Fatalf(". 开头目录应被跳过，实际扫到 %d 个", len(s.Projects()))
	}
}

// TestScan_RootNameNotSkipped 规则根本身以 "." / "_" 开头时不跳过（前缀限制只作用于根下子目录）。
func TestScan_RootNameNotSkipped(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	root := ws.Mkdir(".dotroot")
	ws.MakeProjectDir(path.Join(".dotroot", "proj"))

	s := newServiceAt(t, root, "g", 5)
	if len(s.Projects()) != 1 {
		t.Fatalf("规则根本身不受 ./_ 前缀限制，应扫到 1 个，实际 %d", len(s.Projects()))
	}
}

// TestScan_GodotTag godot 项目打 godot tag。
func TestScan_GodotTag(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	root := ws.Mkdir("root")
	ws.MakeProjectDir(path.Join("root", "game"), testfixture.WithGodot())

	s := newServiceAt(t, root, "g", 5)
	projs := s.Projects()
	if len(projs) != 1 {
		t.Fatalf("应扫到 1 个，实际 %d", len(projs))
	}
	tags := projs[0].CollectTags()
	found := false
	for _, tg := range tags {
		if tg == TagGodot {
			found = true
		}
	}
	if !found {
		t.Fatalf("godot 项目应打 godot tag，实际 tags=%v", tags)
	}
}

// TestScan_RuleTags 规则声明的 tags 追加到命中项目，与探测 tag（godot）合并且清洗（去重/去空）。
func TestScan_RuleTags(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	root := ws.Mkdir("root")
	ws.MakeProjectDir(path.Join("root", "game"), testfixture.WithGodot())
	ws.MakeProjectDir(path.Join("root", "plain"))

	s := newServiceWithRules(t, ws, []ScanRule{
		{Group: "g", Path: root, MaxDepth: 5, Tags: []string{"公司", " "}},
	}, nil)
	projs := s.Projects()
	if len(projs) != 2 {
		t.Fatalf("应扫到 2 个，实际 %d", len(projs))
	}
	for _, p := range projs {
		want := []string{"公司"}
		if path.Base(p.Path()) == "game" {
			want = []string{"公司", TagGodot}
		}
		if got := p.CollectTags(); !slices.Equal(got, want) {
			t.Fatalf("项目 %s tags=%v, want %v", p.Name(), got, want)
		}
	}
}

// TestScan_WorktreeSkipped .git 是文件的 worktree 目录不被收录（1032 归并为项目打开目标），
// 且其子目录不再遍历。
func TestScan_WorktreeSkipped(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	root := ws.Mkdir("root")
	ws.MakeProjectDir(path.Join("root", "wt"), testfixture.WithWorktree())
	ws.MakeProjectDir(path.Join("root", "normal"))

	s := newServiceAt(t, root, "g", 5)
	projs := s.Projects()
	if len(projs) != 1 {
		t.Fatalf("worktree 目录应被跳过，实际扫到 %d 个: %+v", len(projs), projs)
	}
	if projs[0].Path() != ws.Join("root", "normal") {
		t.Fatalf("唯一项目应是 normal, got %s", projs[0].Path())
	}
}

// TestMatchScanRule 判定「git init 后能否被 scan 收录为新项目」（纯函数，表驱动）。
// 判定语义需与 scanOne 遍历一致：直接子目录为深度 0（-1 不限深），
// 且规则根下到目标路径的各级目录名不以 . / _ 开头（规则根本身不受限）。
func TestMatchScanRule(t *testing.T) {
	rules := []ScanRule{
		{Group: "g", Path: "/w/code", MaxDepth: 2},
		{Group: "go", Path: "/w/code/go", MaxDepth: 3},
		{Group: "hid", Path: "/w/.hidden", MaxDepth: 3},
		{Group: "any", Path: "/w/any", MaxDepth: -1},
	}

	cases := []struct {
		name     string
		path     string
		wantOK   bool
		wantName string
	}{
		{"规则根直接子目录", "/w/code/proj", true, "g:proj"},
		{"目标即规则根本身", "/w/code", true, "g:g"},
		{"深度恰好等于 maxDepth", "/w/code/a/b/c", true, "g:a/b/c"},
		{"深度超出全部规则", "/w/code/a/b/c/d", false, ""},
		{"多规则命中取最内层", "/w/code/go/x", true, "go:x"},
		{"外层规则超深但内层命中", "/w/code/go/x/y/z", true, "go:x/y/z"},
		{"maxDepth=-1 不限深", "/w/any/a/b/c/d/e", true, "any:a/b/c/d/e"},
		{"规则根本身以 . 开头不跳过", "/w/.hidden/proj", true, "hid:proj"},
		{"中间目录以 _ 开头", "/w/code/_drafts/proj", false, ""},
		{"目标目录自身以 . 开头", "/w/code/.config", false, ""},
		{"不在任何规则目录下", "/w/other/proj", false, ""},
		{"仅前缀字符串相似的兄弟路径", "/w/code2/proj", false, ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, name, ok := MatchScanRule(c.path, rules)
			if ok != c.wantOK {
				t.Fatalf("ok = %v，期望 %v", ok, c.wantOK)
			}
			if ok && name != c.wantName {
				t.Fatalf("name = %q，期望 %q", name, c.wantName)
			}
		})
	}
}

// TestService_MatchScanRule Service 委托方法用构造时的规则判定。
func TestService_MatchScanRule(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	root := ws.Mkdir("root")
	s := newServiceAt(t, root, "g1", 3)

	rule, name, ok := s.MatchScanRule(ws.Join("root", "proj"))
	if !ok || rule.Group != "g1" || name != "g1:proj" {
		t.Fatalf("MatchScanRule = (group=%q, name=%q, ok=%v)，期望 (g1, g1:proj, true)", rule.Group, name, ok)
	}
	if _, _, ok := s.MatchScanRule("/nonexistent/outside"); ok {
		t.Fatalf("规则外路径不应命中")
	}
}
