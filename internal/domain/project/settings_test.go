package project

import (
	"path"
	"slices"
	"strings"
	"testing"

	"cube/internal/testfixture"
)

// newWriteService 构造可写 settings 的 Service（无初始规则）。
func newWriteService(t *testing.T) (*Service, *testfixture.Workspace) {
	t.Helper()
	ws := testfixture.NewWorkspace(t)
	s := newServiceWithRules(t, ws, nil, nil)
	return s, ws
}

// TestSaveScanRule 新增与按键替换：同 path 替换、新 path 追加，保存后项目列表即时反映。
func TestSaveScanRule(t *testing.T) {
	s, ws := newWriteService(t)
	root := ws.Mkdir("root")
	ws.MakeProjectDir(path.Join("root", "p1"))

	if err := s.SaveScanRule(ScanRule{Group: "g", Path: root, MaxDepth: 3}); err != nil {
		t.Fatalf("保存失败: %v", err)
	}
	if rules := s.ScanRules(); len(rules) != 1 || rules[0].Group != "g" {
		t.Fatalf("保存后规则异常: %v", rules)
	}
	// 保存即重扫：无需手动 Refresh，列表立即反映新规则
	if projs := s.Projects(); len(projs) != 1 {
		t.Fatalf("保存后项目列表应即时重扫，实际 %d 个", len(projs))
	}

	// 同 path 替换（改 group）
	if err := s.SaveScanRule(ScanRule{Group: "g2", Path: root, MaxDepth: 3}); err != nil {
		t.Fatalf("替换失败: %v", err)
	}
	if rules := s.ScanRules(); len(rules) != 1 || rules[0].Group != "g2" {
		t.Fatalf("同 path 应替换而非追加: %v", rules)
	}

	// maxDepth 0（仅根下一层）与 -1（不限深）均为合法值
	for _, d := range []int{0, -1} {
		if err := s.SaveScanRule(ScanRule{Group: "g2", Path: root, MaxDepth: d}); err != nil {
			t.Fatalf("maxDepth=%d 应合法: %v", d, err)
		}
	}
}

// TestSaveScanRule_TagsNormalized 保存时清洗 tags（trim/去空/去重），空 tags 落盘后不残留空节。
func TestSaveScanRule_TagsNormalized(t *testing.T) {
	s, ws := newWriteService(t)
	root := ws.Mkdir("root")
	ws.MakeProjectDir(path.Join("root", "p1"))

	if err := s.SaveScanRule(ScanRule{Group: "g", Path: root, MaxDepth: 3, Tags: []string{" 公司 ", "", "开源", "公司"}}); err != nil {
		t.Fatalf("保存失败: %v", err)
	}
	rules := s.ScanRules()
	if len(rules) != 1 || !slices.Equal(rules[0].Tags, []string{"公司", "开源"}) {
		t.Fatalf("tags 应清洗为 [公司 开源]，实际: %v", rules)
	}
	if projs := s.Projects(); len(projs) != 1 || !slices.Equal(projs[0].CollectTags(), []string{"公司", "开源"}) {
		t.Fatalf("项目 tags 应来自规则，实际: %v", projs)
	}

	// 清空 tags 替换：不残留旧值
	if err := s.SaveScanRule(ScanRule{Group: "g", Path: root, MaxDepth: 3, Tags: []string{" ", ""}}); err != nil {
		t.Fatalf("替换失败: %v", err)
	}
	if rules := s.ScanRules(); len(rules) != 1 || rules[0].Tags != nil {
		t.Fatalf("全空 tags 应清洗为 nil，实际: %v", rules)
	}
}

// TestSaveScanRule_Validation 坏数据返回中文错误、落不了文件。
func TestSaveScanRule_Validation(t *testing.T) {
	s, ws := newWriteService(t)
	root := ws.Mkdir("root")
	ws.WriteFile("f.txt", []byte("x"))

	cases := []struct {
		name string
		rule ScanRule
		want string
	}{
		{"group 为空", ScanRule{Path: root, MaxDepth: 3}, "group"},
		{"maxDepth 小于 -1", ScanRule{Group: "g", Path: root, MaxDepth: -2}, "maxDepth"},
		{"路径不存在", ScanRule{Group: "g", Path: "/no/such/dir/xyz", MaxDepth: 3}, "不存在"},
		{"路径是文件", ScanRule{Group: "g", Path: ws.Join("f.txt"), MaxDepth: 3}, "非目录"},
		{"相对路径", ScanRule{Group: "g", Path: "relative/dir", MaxDepth: 3}, "绝对路径"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := s.SaveScanRule(c.rule)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("应报含 %q 的中文错误，实际: %v", c.want, err)
			}
			if rules := s.ScanRules(); len(rules) != 0 {
				t.Fatalf("坏数据不应落文件，实际规则: %v", rules)
			}
		})
	}
}

// TestDeleteScanRule 按 path 删除；不存在返回中文错误。
func TestDeleteScanRule(t *testing.T) {
	s, ws := newWriteService(t)
	root := ws.Mkdir("root")
	ws.MakeProjectDir(path.Join("root", "p1"))
	if err := s.SaveScanRule(ScanRule{Group: "g", Path: root, MaxDepth: 3}); err != nil {
		t.Fatalf("保存失败: %v", err)
	}

	// 整项删除只按 path 匹配，其他字段不参与
	if err := s.DeleteScanRule(ScanRule{Group: "旧组名", Path: root, MaxDepth: 9}); err != nil {
		t.Fatalf("删除失败: %v", err)
	}
	if rules := s.ScanRules(); len(rules) != 0 {
		t.Fatalf("删除后规则应为空: %v", rules)
	}
	// 删除即重扫
	if projs := s.Projects(); len(projs) != 0 {
		t.Fatalf("删除后项目列表应即时重扫为空，实际 %d 个", len(projs))
	}

	if err := s.DeleteScanRule(ScanRule{Group: "g", Path: root}); err == nil || !strings.Contains(err.Error(), "未找到") {
		t.Fatalf("删除不存在的规则应报中文错误，实际: %v", err)
	}
}

// TestSaveAllScanRules 提交清单即落盘结果（顺序 / 删除 / 新增 / 修改）；顺序同时改变项目列表
// 展示序（scan 按规则序遍历），保存后立即重扫。单项非法、path/group 重复整体拒绝不落盘。
func TestSaveAllScanRules(t *testing.T) {
	s, ws := newWriteService(t)
	r1, r2, r3 := ws.Mkdir("r1"), ws.Mkdir("r2"), ws.Mkdir("r3")
	ws.MakeProjectDir(path.Join("r1", "a"))
	ws.MakeProjectDir(path.Join("r3", "b"))
	for i, r := range []string{r1, r2} {
		if err := s.SaveScanRule(ScanRule{Group: string(rune('a' + i)), Path: r, MaxDepth: 3}); err != nil {
			t.Fatalf("保存失败: %v", err)
		}
	}
	_ = s.Projects()

	// r3 新增并置顶、r1 修改 group、r2 未提交即删除
	if err := s.SaveAllScanRules([]ScanRule{
		{Group: "c", Path: r3, MaxDepth: 3},
		{Group: "a2", Path: r1, MaxDepth: 3},
	}); err != nil {
		t.Fatalf("save-all 失败: %v", err)
	}
	rules := s.ScanRules()
	if len(rules) != 2 || rules[0].Path != r3 || rules[1].Path != r1 || rules[1].Group != "a2" {
		t.Fatalf("save-all 后规则不符: %v", rules)
	}
	projs := s.Projects()
	if len(projs) != 2 || !strings.HasSuffix(projs[0].Path(), path.Join("r3", "b")) {
		t.Fatalf("save-all 后项目列表应即时重扫，实际: %v", projs)
	}

	cases := []struct {
		name  string
		rules []ScanRule
		want  string
	}{
		{"path 重复", []ScanRule{{Group: "x", Path: r1, MaxDepth: 3}, {Group: "y", Path: r1, MaxDepth: 3}}, "path 重复"},
		{"group 重复", []ScanRule{{Group: "x", Path: r1, MaxDepth: 3}, {Group: "x", Path: r2, MaxDepth: 3}}, "group 已被其它规则占用"},
		{"单项非法", []ScanRule{{Group: "x", Path: r1, MaxDepth: 3}, {Group: "y", Path: "/no/such/dir/xyz", MaxDepth: 3}}, "不存在"},
	}
	for _, c := range cases {
		if err := s.SaveAllScanRules(c.rules); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Fatalf("%s 应报含 %q 的中文错误，实际: %v", c.name, c.want, err)
		}
	}
	if rules := s.ScanRules(); len(rules) != 2 || rules[0].Path != r3 {
		t.Fatalf("失败的 save-all 不应改动文件: %v", rules)
	}

	if err := s.SaveAllScanRules(nil); err != nil {
		t.Fatalf("空清单 save-all 失败: %v", err)
	}
	if len(s.ScanRules()) != 0 || len(s.Projects()) != 0 {
		t.Fatalf("空清单应清空规则并重扫为空")
	}
}

// TestSaveScanRule_GroupUnique group 是分组与项目名前缀的展示身份，写侧拒绝与其它规则重复。
func TestSaveScanRule_GroupUnique(t *testing.T) {
	s, ws := newWriteService(t)
	root1, root2 := ws.Mkdir("r1"), ws.Mkdir("r2")

	if err := s.SaveScanRule(ScanRule{Group: "g", Path: root1, MaxDepth: 3}); err != nil {
		t.Fatalf("保存失败: %v", err)
	}
	// 其它 path 用同名 group：拒绝且不落文件
	err := s.SaveScanRule(ScanRule{Group: "g", Path: root2, MaxDepth: 3})
	if err == nil || !strings.Contains(err.Error(), "group 已被其它规则占用") {
		t.Fatalf("重复 group 应报中文错误，实际: %v", err)
	}
	// 同 path 替换自身不受影响（改 maxDepth、保留 group）
	if err := s.SaveScanRule(ScanRule{Group: "g", Path: root1, MaxDepth: 1}); err != nil {
		t.Fatalf("同 path 替换不应视为占用: %v", err)
	}
	if rules := s.ScanRules(); len(rules) != 1 {
		t.Fatalf("被拒规则不应落文件，实际: %v", rules)
	}
}

// TestLoadScanRules_DuplicateGroupSkipped 手工编辑产生的重复 group 按坏数据降级：跳过后加载的规则。
func TestLoadScanRules_DuplicateGroupSkipped(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	root1, root2 := ws.Mkdir("r1"), ws.Mkdir("r2")
	ws.MakeProjectDir(path.Join("r1", "a"))
	ws.MakeProjectDir(path.Join("r2", "b"))

	// 直接落盘两条同 group 规则（绕开写侧校验，模拟手工编辑 settings.json）
	s := newServiceWithRules(t, ws, []ScanRule{
		{Group: "g", Path: root1, MaxDepth: 3},
		{Group: "g", Path: root2, MaxDepth: 3},
	}, nil)
	if rules := s.ScanRules(); len(rules) != 1 || rules[0].Path != root1 {
		t.Fatalf("重复 group 应只保留先加载的规则，实际: %v", rules)
	}
	if projs := s.Projects(); len(projs) != 1 || !strings.HasSuffix(projs[0].Path(), path.Join("r1", "a")) {
		t.Fatalf("项目应只来自先加载的规则，实际: %v", projs)
	}
}

// TestSaveCloneRule 新增与按 host+prefix 替换。
func TestSaveCloneRule(t *testing.T) {
	s, ws := newWriteService(t)

	if err := s.SaveCloneRule(CloneRule{RepoHost: "github.com", LocalPath: ws.Dir}); err != nil {
		t.Fatalf("保存失败: %v", err)
	}
	// 同 host 不同 prefix 是新规则
	if err := s.SaveCloneRule(CloneRule{RepoHost: "github.com", RepoPrefix: "/heyuuu", LocalPath: ws.Dir}); err != nil {
		t.Fatalf("保存失败: %v", err)
	}
	// 同 host+prefix 替换 localPath
	if err := s.SaveCloneRule(CloneRule{RepoHost: "github.com", RepoPrefix: "/heyuuu", LocalPath: ws.Mkdir("gh")}); err != nil {
		t.Fatalf("替换失败: %v", err)
	}

	rules := s.CloneRules()
	if len(rules) != 2 {
		t.Fatalf("应保留 2 条规则，实际 %d: %v", len(rules), rules)
	}
	for _, r := range rules {
		if r.RepoHost == "github.com" && r.RepoPrefix == "/heyuuu" && r.LocalPath != ws.Join("gh") {
			t.Fatalf("替换未生效: %+v", r)
		}
	}
}

// TestSaveCloneRule_Validation 坏数据返回中文错误、落不了文件。
func TestSaveCloneRule_Validation(t *testing.T) {
	s, ws := newWriteService(t)

	cases := []struct {
		name string
		rule CloneRule
		want string
	}{
		{"host 为空", CloneRule{LocalPath: ws.Dir}, "repoHost"},
		{"prefix 不以 / 开头", CloneRule{RepoHost: "github.com", RepoPrefix: "heyuuu", LocalPath: ws.Dir}, "repoPrefix"},
		{"localPath 相对路径", CloneRule{RepoHost: "github.com", LocalPath: "relative"}, "localPath"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := s.SaveCloneRule(c.rule)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("应报含 %q 的中文错误，实际: %v", c.want, err)
			}
			if rules := s.CloneRules(); len(rules) != 0 {
				t.Fatalf("坏数据不应落文件，实际规则: %v", rules)
			}
		})
	}
}

// TestDeleteCloneRule 按 host+prefix 删除；不存在返回中文错误。
func TestDeleteCloneRule(t *testing.T) {
	s, ws := newWriteService(t)
	if err := s.SaveCloneRule(CloneRule{RepoHost: "github.com", RepoPrefix: "/heyuuu", LocalPath: ws.Dir}); err != nil {
		t.Fatalf("保存失败: %v", err)
	}

	// 整项删除只按 host+prefix 匹配，localPath 不参与
	rule := CloneRule{RepoHost: "github.com", RepoPrefix: "/heyuuu", LocalPath: "/other"}
	if err := s.DeleteCloneRule(rule); err != nil {
		t.Fatalf("删除失败: %v", err)
	}
	if rules := s.CloneRules(); len(rules) != 0 {
		t.Fatalf("删除后规则应为空: %v", rules)
	}
	if err := s.DeleteCloneRule(rule); err == nil || !strings.Contains(err.Error(), "未找到") {
		t.Fatalf("删除不存在的规则应报中文错误，实际: %v", err)
	}
}

// TestSaveAllCloneRules 提交清单即落盘结果；host+prefix 重复 / 单项非法整体拒绝不落盘。
func TestSaveAllCloneRules(t *testing.T) {
	s, ws := newWriteService(t)
	save := func(host, prefix string) {
		t.Helper()
		if err := s.SaveCloneRule(CloneRule{RepoHost: host, RepoPrefix: prefix, LocalPath: ws.Dir}); err != nil {
			t.Fatalf("保存失败: %v", err)
		}
	}
	save("github.com", "/heyuuu")
	save("github.com", "")

	if err := s.SaveAllCloneRules([]CloneRule{
		{RepoHost: "gitee.com", LocalPath: ws.Dir},
		{RepoHost: "github.com", RepoPrefix: "/heyuuu", LocalPath: ws.Mkdir("gh")},
	}); err != nil {
		t.Fatalf("save-all 失败: %v", err)
	}
	rules := s.CloneRules()
	if len(rules) != 2 || rules[0].RepoHost != "gitee.com" || rules[1].RepoPrefix != "/heyuuu" || rules[1].LocalPath != ws.Join("gh") {
		t.Fatalf("save-all 后规则不符: %v", rules)
	}

	if err := s.SaveAllCloneRules([]CloneRule{{RepoHost: "a.com", LocalPath: ws.Dir}, {RepoHost: "a.com", LocalPath: ws.Dir}}); err == nil || !strings.Contains(err.Error(), "重复") {
		t.Fatalf("重复键应报中文错误，实际: %v", err)
	}
	if err := s.SaveAllCloneRules([]CloneRule{{RepoHost: "a.com", LocalPath: "relative"}}); err == nil || !strings.Contains(err.Error(), "localPath") {
		t.Fatalf("单项非法应报中文错误，实际: %v", err)
	}
	if rules := s.CloneRules(); len(rules) != 2 || rules[0].RepoHost != "gitee.com" {
		t.Fatalf("失败的 save-all 不应改动文件: %v", rules)
	}
}
