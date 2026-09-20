package project

import (
	"os"
	"path"
	"slices"
	"testing"

	"cube/core/config"
	"cube/internal/testfixture"
)

// newServiceAt 为测试构造一个扫描指定根目录的 Service。
// 就地写而非放进 testfixture，是因为 testfixture 不能 import project（会导致 git 等底层包测试循环依赖）。
func newServiceAt(t *testing.T, scanRoot, group string, maxDepth int) *Service {
	t.Helper()
	return newServiceWithRules(t, testfixture.NewWorkspace(t),
		[]ScanRule{{Group: group, Path: scanRoot, MaxDepth: maxDepth}}, nil)
}

// newServiceWithRules 把 scan/clone 规则写进工作区的 settings.json 再构造 Service（规则数据源即 settings 节）。
func newServiceWithRules(t *testing.T, ws *testfixture.Workspace, scan []ScanRule, clone []CloneRule) *Service {
	t.Helper()
	settingsFile := ws.Join("settings.json")
	saveRules(t, settingsFile, scan, clone)
	return NewService(settingsFile, ws.Mkdir("cache"))
}

// saveRules 分节写 scanRule / cloneRule 两个 settings 节（nil 的部分跳过不写）。
func saveRules(t *testing.T, settingsFile string, scan []ScanRule, clone []CloneRule) {
	t.Helper()
	if scan != nil {
		if err := config.SaveSection(settingsFile, scanRulesSection, scan); err != nil {
			t.Fatalf("写测试 settings.json scanRule 节失败: %v", err)
		}
	}
	if clone != nil {
		if err := config.SaveSection(settingsFile, cloneRulesSection, clone); err != nil {
			t.Fatalf("写测试 settings.json cloneRule 节失败: %v", err)
		}
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

// TestScan_MaxDepth 深度剪枝：超 MaxDepth 的项目不被扫到。
func TestScan_MaxDepth(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	root := ws.Mkdir("root")
	// 浅层项目（root/proj）
	ws.MakeProjectDir(path.Join("root", "proj"))
	// 深层项目（root/x/y/deep）—— root 到 deep 跨 3 层目录
	ws.MakeProjectDir(path.Join("root", "x", "y", "deep"))

	// MaxDepth=2：只扫到浅层
	s := newServiceAt(t, root, "g", 2)
	if len(s.Projects()) != 1 {
		t.Fatalf("MaxDepth=2 应只扫到 1 个，实际 %d", len(s.Projects()))
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
	tags := projs[0].Tags()
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
		if got := p.Tags(); !slices.Equal(got, want) {
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

// TestScanRules_Getter ScanRules() 从 settings.json 读出规则（含路径校验）。
func TestScanRules_Getter(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	s := newServiceWithRules(t, ws, []ScanRule{{Group: "g", Path: ws.Mkdir("root"), MaxDepth: 3}}, nil)
	rules := s.ScanRules()
	if len(rules) != 1 || rules[0].Group != "g" {
		t.Fatalf("ScanRules 异常: %v", rules)
	}
}

// TestMatchScanRule 判定「git init 后能否被 scan 收录为新项目」（纯函数，表驱动）。
// 判定语义需与 scanOne 遍历一致：位于规则根下 maxDepth 层级内，
// 且从规则根到目标路径的各级目录名不以 . / _ 开头。
func TestMatchScanRule(t *testing.T) {
	rules := []ScanRule{
		{Group: "g", Path: "/w/code", MaxDepth: 2},
		{Group: "go", Path: "/w/code/go", MaxDepth: 3},
		{Group: "hid", Path: "/w/.hidden", MaxDepth: 3},
	}

	cases := []struct {
		name     string
		path     string
		wantOK   bool
		wantName string
	}{
		{"规则根直接子目录", "/w/code/proj", true, "g:proj"},
		{"目标即规则根本身", "/w/code", true, "g:g"},
		{"深度恰好等于 maxDepth", "/w/code/a/b", true, "g:a/b"},
		{"深度超出全部规则", "/w/code/a/b/c", false, ""},
		{"多规则命中取最内层", "/w/code/go/x", true, "go:x"},
		{"外层规则超深但内层命中", "/w/code/go/x/y", true, "go:x/y"},
		{"规则根本身以 . 开头", "/w/.hidden/proj", false, ""},
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

// TestScan_HomePrefixSkipped ~/ 前缀不再由 domain 展开（展开职责在出口层），
// 配置里写 ~/ 视为配置错误，降级跳过、不阻断其它规则。
func TestScan_HomePrefixSkipped(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repoDir := home + "/Code/proj"
	os.MkdirAll(repoDir, 0755)
	ws := testfixture.NewWorkspace(t)
	testfixture.BuildGitRepo(ws.TB, repoDir, testfixture.GitRepoSpec{})

	s := newServiceWithRules(t, ws, []ScanRule{
		{Group: "bad", Path: "~/Code", MaxDepth: 3},
		{Group: "good", Path: home + "/Code", MaxDepth: 3},
	}, nil)

	rules := s.ScanRules()
	if len(rules) != 1 || rules[0].Group != "good" {
		t.Fatalf("~/ 前缀应被跳过，只保留绝对路径规则，实际 %v", rules)
	}
}

// TestScan_InvalidPathSkipped 不存在的 scan 路径被降级跳过（不阻断其它规则）。
func TestScan_InvalidPathSkipped(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	goodRoot := ws.Mkdir("real-root")
	ws.MakeProjectDir(path.Join("real-root", "proj"))

	s := newServiceWithRules(t, ws, []ScanRule{
		{Group: "bad", Path: "/this/does/not/exist/xyz", MaxDepth: 3},
		{Group: "good", Path: goodRoot, MaxDepth: 3},
	}, nil)

	rules := s.ScanRules()
	if len(rules) != 1 {
		t.Fatalf("无效路径应被跳过，保留 1 条规则，实际 %d", len(rules))
	}
	if rules[0].Group != "good" {
		t.Fatalf("应保留 good 规则，实际 %v", rules)
	}
	// 扫描仍能命中有效规则下的项目
	if len(s.Projects()) != 1 {
		t.Fatalf("应扫到 1 个项目，实际 %d", len(s.Projects()))
	}
}

// TestCloneRule_HomePrefixSkipped clone 规则 LocalPath 的 ~/ 前缀不再展开，按配置错误跳过。
func TestCloneRule_HomePrefixSkipped(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	ws := testfixture.NewWorkspace(t)
	s := newServiceWithRules(t, ws, nil, []CloneRule{
		{RepoHost: "github.com", RepoPrefix: "/heyuuu", LocalPath: "~/src"},
	})

	if rules := s.CloneRules(); len(rules) != 0 {
		t.Fatalf("~/ 前缀应被跳过，实际 %v", rules)
	}
}

// TestCloneRule_InvalidLocalPathSkipped clone 规则 LocalPath 为相对路径时降级跳过（不阻断其它规则）。
// 配置路径不得依赖执行目录，相对路径是配置错误。
func TestCloneRule_InvalidLocalPathSkipped(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	s := newServiceWithRules(t, ws, nil, []CloneRule{
		{RepoHost: "github.com", RepoPrefix: "/heyuuu", LocalPath: "relative/src"}, // 相对路径，跳过
		{RepoHost: "gitee.com", RepoPrefix: "/heyuuu", LocalPath: ws.Dir},          // 绝对路径，保留
	})

	rules := s.CloneRules()
	if len(rules) != 1 {
		t.Fatalf("应只保留 1 条合法 clone 规则，实际 %d", len(rules))
	}
	if rules[0].RepoHost != "gitee.com" {
		t.Fatalf("保留的应是绝对路径规则，实际 host=%s", rules[0].RepoHost)
	}
}

// TestService_SettingsDirectRead 规则直读 settings.json：不重建 Service，
// 重写 settings.json 后规则立即更新，重扫（Reload）后项目列表随之变化。
func TestService_SettingsDirectRead(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	root1 := ws.Mkdir("root1")
	ws.MakeProjectDir(path.Join("root1", "p1"))

	// 初始：只 root1 一条规则
	s := newServiceWithRules(t, ws, []ScanRule{{Group: "g1", Path: root1, MaxDepth: 5}}, nil)
	if len(s.Projects()) != 1 {
		t.Fatalf("初始应扫到 1 个项目，实际 %d", len(s.Projects()))
	}

	// 重写 settings.json 为 root2 规则（模拟外部修改 / Web 保存）
	root2 := ws.Mkdir("root2")
	ws.MakeProjectDir(path.Join("root2", "a"))
	ws.MakeProjectDir(path.Join("root2", "b"))
	saveRules(t, ws.Join("settings.json"), []ScanRule{{Group: "g2", Path: root2, MaxDepth: 5}}, nil)

	// 规则立即更新（直读不缓存）
	rules := s.ScanRules()
	if len(rules) != 1 || rules[0].Path != root2 {
		t.Fatalf("规则应直读更新为 root2：%v", rules)
	}
	// 项目列表在重扫后反映新规则
	if _, _, err := s.Refresh(); err != nil {
		t.Fatalf("Refresh 失败: %v", err)
	}
	if projs := s.Projects(); len(projs) != 2 {
		t.Fatalf("重扫后应扫到 root2 的 2 个项目，实际 %d：%v", len(projs), projs)
	}
}
