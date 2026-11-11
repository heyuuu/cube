package opener

import (
	"reflect"
	"testing"

	"cube/internal/domain/icons"
	"cube/internal/testfixture"
)

// newServiceAt 建一个指向测试专属 settings.json 的 Service（注入 fakeExecutor）。
func newServiceAt(t *testing.T, specs []Opener) (*Service, *testfixture.Workspace) {
	t.Helper()
	ws := testfixture.NewWorkspace(t)
	if specs != nil {
		ws.SaveSection(openersSection, specs)
	}
	fake := &fakeExecutor{}
	return NewServiceWithExecutor(ws.Cfg, fake), ws
}

func TestServiceDirectRead(t *testing.T) {
	t.Run("文件不存在返回空列表", func(t *testing.T) {
		s, _ := newServiceAt(t, nil)
		if got := s.AllOpeners(); len(got) != 0 {
			t.Fatalf("期望空列表, got %v", got)
		}
	})

	t.Run("读取全部与按名查找", func(t *testing.T) {
		s, _ := newServiceAt(t, []Opener{
			{Name: "code", Actions: map[Role]string{"open-dir": `exec: code`}},
			{Name: "bcompare", Actions: map[Role]string{"diff-dir": `exec: bcompare $0 $1`}},
		})
		if got := s.AllOpeners(); len(got) != 2 {
			t.Fatalf("期望 2 个, got %d", len(got))
		}
		if o := s.FindByName("bcompare"); o == nil {
			t.Fatal("FindByName(bcompare) 应命中")
		}
		if o := s.FindByName("nope"); o != nil {
			t.Fatalf("FindByName(nope) 不应命中, got %v", o)
		}
	})

	t.Run("坏条目跳过不阻断", func(t *testing.T) {
		s, _ := newServiceAt(t, []Opener{
			{Name: "bad", Actions: map[Role]string{"open-dir": `exec:`}},       // 缺 cmd，条目级坏
			{Name: "code", Actions: map[Role]string{"open-dir": `exec: code`}}, // 正常
			{Name: "bad2", Actions: map[Role]string{"open-dir": `exec: c $9`}}, // 占位符越界，条目级坏
		})
		got := s.AllOpeners()
		if len(got) != 1 || got[0].Name != "code" {
			t.Fatalf("期望只剩 code, got %v", got)
		}
	})

	t.Run("RoleOpeners 过滤", func(t *testing.T) {
		s, _ := newServiceAt(t, []Opener{
			{Name: "code", Actions: map[Role]string{"open-dir": `exec: code`}},
			{Name: "bcompare", Actions: map[Role]string{"diff-dir": `exec: bcompare $0 $1`}},
		})
		if got := s.RoleOpeners(RoleDiffDir); len(got) != 1 || got[0].Name != "bcompare" {
			t.Fatalf("diff-dir 应只有 bcompare, got %v", got)
		}
	})

	t.Run("SearchAll 模糊匹配", func(t *testing.T) {
		s, _ := newServiceAt(t, []Opener{
			{Name: "code", Actions: map[Role]string{"open-dir": `exec: code`}},
			{Name: "vscode", Actions: map[Role]string{"open-dir": `exec: vscode`}},
		})
		if got := s.SearchAll("cod"); len(got) != 2 {
			t.Fatalf("cod 应匹配 code+vscode, got %v", got)
		}
	})

	t.Run("保存后无需重建 Service 即生效（直读不缓存）", func(t *testing.T) {
		s, ws := newServiceAt(t, []Opener{{Name: "code", Actions: map[Role]string{"open-dir": `exec: code`}}})
		ws.SaveSection(openersSection, []Opener{
			{Name: "newone", Actions: map[Role]string{"open-dir": `exec: newone`}},
		})
		if o := s.FindByName("newone"); o == nil {
			t.Fatal("新条目应立即可见（直读）")
		}
		if o := s.FindByName("code"); o != nil {
			t.Fatalf("旧条目应已消失, got %v", o)
		}
	})

	t.Run("其他节不受影响", func(t *testing.T) {
		s, ws := newServiceAt(t, []Opener{{Name: "code", Actions: map[Role]string{"open-dir": `exec: code`}}})
		ws.SaveSection("other", map[string]int{"a": 1})
		if got := s.AllOpeners(); len(got) != 1 || got[0].Name != "code" {
			t.Fatalf("写其他节不应影响 openers, got %v", got)
		}
	})
}

func TestServiceIconEndToEnd(t *testing.T) {
	// icon 随 Opener 存进 settings.json，读出后原样透传；未配置为 nil（默认 icon 兜底归前端）
	s, _ := newServiceAt(t, []Opener{
		{Name: "code", Actions: map[Role]string{"open-dir": `exec: code`}, Icon: &icons.Icon{Type: icons.IconTypeLucide, Value: "app-window"}},
		{Name: "plain", Actions: map[Role]string{"open-dir": `exec: plain`}},
	})
	o := s.FindByName("code")
	if o == nil || o.Icon == nil || o.Icon.Value != "app-window" {
		t.Fatalf("icon 未透传: %+v", o)
	}
	if p := s.FindByName("plain"); p.Icon != nil {
		t.Fatalf("无 icon 应保持 nil（默认 icon 归前端）, got %+v", p.Icon)
	}
}

func TestServiceSaveDelete(t *testing.T) {
	t.Run("新增后可读回", func(t *testing.T) {
		s, _ := newServiceAt(t, nil)
		if err := s.SaveOpener(Opener{Name: "code", Actions: map[Role]string{"open-dir": `exec: code`}}); err != nil {
			t.Fatalf("保存失败: %v", err)
		}
		if o := s.FindByName("code"); o == nil {
			t.Fatal("保存后应可读回")
		}
	})

	t.Run("按名替换不重复", func(t *testing.T) {
		s, _ := newServiceAt(t, []Opener{{Name: "code", Actions: map[Role]string{"open-dir": `exec: code`}}})
		if err := s.SaveOpener(Opener{Name: "code", Actions: map[Role]string{"open-dir": `exec: code $0`}}); err != nil {
			t.Fatalf("保存失败: %v", err)
		}
		if got := s.AllOpeners(); len(got) != 1 {
			t.Fatalf("应仍为 1 条, got %d", len(got))
		}
		if o := s.FindByName("code"); o.Summary() != "open-dir:exec: code $0" {
			t.Fatalf("内容应被替换, got %q", o.Summary())
		}
	})

	t.Run("坏数据返回中文错误且不落文件", func(t *testing.T) {
		s, _ := newServiceAt(t, nil)
		if err := s.SaveOpener(Opener{Name: "bad", Actions: map[Role]string{"open-dir": `exec:`}}); err == nil {
			t.Fatal("缺 cmd 应报错")
		}
		if err := s.SaveOpener(Opener{Name: "", Actions: map[Role]string{"open-dir": `exec: x`}}); err == nil {
			t.Fatal("空 name 应报错")
		}
		if got := s.AllOpeners(); len(got) != 0 {
			t.Fatalf("坏数据不应落文件, got %d", len(got))
		}
	})

	t.Run("删除与不存在幂等", func(t *testing.T) {
		s, _ := newServiceAt(t, []Opener{{Name: "code", Actions: map[Role]string{"open-dir": `exec: code`}}})
		// 整项删除只按 name 匹配，其他字段不参与
		if err := s.DeleteOpener(Opener{Name: "code", Title: "旧展示文案"}); err != nil {
			t.Fatalf("删除失败: %v", err)
		}
		if got := s.AllOpeners(); len(got) != 0 {
			t.Fatalf("删除后应为空, got %d", len(got))
		}
		if err := s.DeleteOpener(Opener{Name: "nope"}); err != nil {
			t.Fatalf("删除不存在项应幂等成功: %v", err)
		}
	})
}

func TestServiceSaveAll(t *testing.T) {
	newSpecs := []Opener{
		{Name: "finder", Actions: map[Role]string{"open-dir": `exec: open -a Finder`}},
		{Name: "code", Actions: map[Role]string{"open-dir": `exec: code`}},
		{Name: "stree", Actions: map[Role]string{"open-dir": `exec: stree`}},
	}

	t.Run("提交清单即落盘结果（顺序 / 删除 / 新增 / 修改）", func(t *testing.T) {
		s, _ := newServiceAt(t, newSpecs)
		if err := s.SaveAllOpeners([]Opener{
			{Name: "stree", Actions: map[Role]string{"open-dir": `exec: stree $0`}},
			{Name: "ghostty", Actions: map[Role]string{"open-dir": `exec: ghostty`}},
			{Name: "finder", Actions: map[Role]string{"open-dir": `exec: open -a Finder`}},
		}); err != nil {
			t.Fatalf("save-all 失败: %v", err)
		}
		got := s.AllOpeners()
		if len(got) != 3 || got[0].Name != "stree" || got[1].Name != "ghostty" || got[2].Name != "finder" {
			t.Fatalf("顺序/成员不符: %v", got)
		}
		if got[0].Summary() != "open-dir:exec: stree $0" {
			t.Fatalf("内容应被覆盖, got %q", got[0].Summary())
		}
	})

	t.Run("单项非法与重复名整体拒绝不落盘", func(t *testing.T) {
		s, _ := newServiceAt(t, newSpecs)
		if err := s.SaveAllOpeners([]Opener{{Name: "code", Actions: map[Role]string{"open-dir": `exec: code`}}, {Name: "bad", Actions: map[Role]string{"open-dir": `exec:`}}}); err == nil {
			t.Fatal("单项非法应报错")
		}
		if err := s.SaveAllOpeners([]Opener{newSpecs[1], newSpecs[1]}); err == nil {
			t.Fatal("重复名应报错")
		}
		got := s.AllOpeners()
		if len(got) != 3 || got[0].Name != "finder" {
			t.Fatalf("失败的 save-all 不应改动文件: %v", got)
		}
	})

	t.Run("空清单清空节", func(t *testing.T) {
		s, _ := newServiceAt(t, newSpecs)
		if err := s.SaveAllOpeners(nil); err != nil {
			t.Fatalf("save-all 失败: %v", err)
		}
		if got := s.AllOpeners(); len(got) != 0 {
			t.Fatalf("应清空, got %v", got)
		}
	})

	t.Run("被移除的 opener 连带清理 intents 引用", func(t *testing.T) {
		s, ws := newServiceAt(t, newSpecs)
		ws.SaveSection(intentsSection, map[Intent]IntentSpec{
			IntentDir:      {DefaultOpener: "finder", Openers: []string{"finder", "code"}},
			IntentTerminal: {Openers: []string{"stree"}},
		})

		if err := s.SaveAllOpeners([]Opener{newSpecs[1]}); err != nil {
			t.Fatalf("save-all 失败: %v", err)
		}
		specs := s.loadIntents()
		if d, ok := specs[IntentDir]; !ok || d.DefaultOpener != "" || len(d.Openers) != 1 || d.Openers[0] != "code" {
			t.Fatalf("dir 引用应清理为 {无默认, [code]}, got %+v", d)
		}
		if _, ok := specs[IntentTerminal]; ok {
			t.Fatalf("terminal 应整键删除, got %+v", specs[IntentTerminal])
		}
	})
}

// ---------- intents（openerIntents 节，1038） ----------

func mkSpecs(pairs ...any) []Opener {
	specs := make([]Opener, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		specs = append(specs, Opener{Name: pairs[i].(string), Actions: pairs[i+1].(map[Role]string)})
	}
	return specs
}

func TestIntentsSynthesis(t *testing.T) {
	dirOnly := mkSpecs("finder", mkmk(RoleOpenDir, "finder"), "code", mkmk(RoleOpenDir, "code $0", RoleOpenFile, "code $0"))
	s, ws := newServiceAt(t, dirOnly)
	ws.SaveSection(intentsSection, map[Intent]IntentSpec{
		IntentDir:       {DefaultOpener: "finder"},
		IntentGit:       {DefaultOpener: "stree"},               // opener 不存在 → 读侧忽略
		IntentDoc:       {DefaultOpener: "finder"},              // role 不符（doc→open-file）→ 读侧忽略
		IntentTerminal:  {Openers: []string{"code", "ghostty"}}, // ghostty 不存在 → 跳过，保留 code
		Intent("bogus"): {DefaultOpener: "finder"},              // 未知 intent 键 → 跳过
	})

	got := map[Intent]IntentInfo{}
	for _, info := range s.Intents() {
		got[info.Intent] = info
	}

	if d := got[IntentDir]; d.DefaultOpener != "finder" {
		t.Fatalf("dir 默认应为 finder, got %+v", d)
	}
	if d := got[IntentGit]; d.DefaultOpener != "" {
		t.Fatalf("不存在的默认 opener 应被忽略, got %+v", d)
	}
	if d := got[IntentDoc]; d.DefaultOpener != "" {
		t.Fatalf("role 不符的默认 opener 应被忽略, got %+v", d)
	}
	if d := got[IntentTerminal]; len(d.Openers) != 1 || d.Openers[0] != "code" {
		t.Fatalf("terminal 候选应只剩 code, got %+v", d)
	}
	// 缺省候选 = 声明对应 role 的全部 opener（openers 节顺序）
	if d := got[IntentFile]; len(d.Openers) != 1 || d.Openers[0] != "code" {
		t.Fatalf("file 缺省候选应只有 code, got %+v", d)
	}
}

func TestSaveDeleteIntentDefault(t *testing.T) {
	s, _ := newServiceAt(t, mkSpecs("finder", mkmk(RoleOpenDir, "finder"), "code", mkmk(RoleOpenDir, "code $0", RoleOpenFile, "code $0")))

	// 写侧校验：role 不符 / 不存在 报错不落盘
	if err := s.SaveIntentDefault(IntentDoc, "finder"); err == nil {
		t.Fatal("finder 无 open-file，应报错")
	}
	if err := s.SaveIntentDefault(IntentDir, "nope"); err == nil {
		t.Fatal("不存在的 opener 应报错")
	}
	if _, err := s.DefaultOpener(IntentDir); err == nil {
		t.Fatal("未配置默认应报错")
	}

	if err := s.SaveIntentDefault(IntentDir, "code"); err != nil {
		t.Fatalf("保存默认失败: %v", err)
	}
	o, err := s.DefaultOpener(IntentDir)
	if err != nil || o.Name != "code" {
		t.Fatalf("DefaultOpener(dir) = (%v,%v)", o, err)
	}

	if err := s.DeleteIntentDefault(IntentDir); err != nil {
		t.Fatalf("删除默认失败: %v", err)
	}
	if _, err := s.DefaultOpener(IntentDir); err == nil {
		t.Fatal("删除后应视为未配置")
	}
	if err := s.DeleteIntentDefault(IntentDir); err == nil {
		t.Fatal("重复删除应报错")
	}
}

// DeleteOpener 连带清理 intents 引用（默认 + 候选）
func TestDeleteOpenerCleansIntents(t *testing.T) {
	s, ws := newServiceAt(t, mkSpecs("finder", mkmk(RoleOpenDir, "finder"), "code", mkmk(RoleOpenDir, "code $0")))
	ws.SaveSection(intentsSection, map[Intent]IntentSpec{
		IntentDir:      {DefaultOpener: "finder", Openers: []string{"finder", "code"}},
		IntentTerminal: {Openers: []string{"finder"}},
	})

	if err := s.DeleteOpener(Opener{Name: "finder"}); err != nil {
		t.Fatalf("删除 opener 失败: %v", err)
	}
	specs := s.loadIntents()
	if d, ok := specs[IntentDir]; !ok || d.DefaultOpener != "" || len(d.Openers) != 1 || d.Openers[0] != "code" {
		t.Fatalf("dir 引用应清理为 {无默认, [code]}, got %+v", d)
	}
	// terminal 只剩 finder 引用，清空后整键删除
	if _, ok := specs[IntentTerminal]; ok {
		t.Fatalf("terminal 应整键删除, got %+v", specs[IntentTerminal])
	}
}

// ---------- Open 全链路（Service.Open，注入 fakeExecutor）----------

func TestOpenInvokesExecutor(t *testing.T) {
	cases := []struct {
		name     string
		actions  map[Role]string
		role     Role
		paths    []string
		wantBin  string
		wantArgs []string
	}{
		{
			name:     "单槽占位符",
			actions:  mkmk(RoleOpenDir, "code $0"),
			role:     RoleOpenDir,
			paths:    []string{"/proj"},
			wantBin:  "code",
			wantArgs: []string{"/proj"},
		},
		{
			name:     "无占位符路径追加末尾",
			actions:  mkmk(RoleOpenDir, "code"),
			role:     RoleOpenDir,
			paths:    []string{"/proj"},
			wantBin:  "code",
			wantArgs: []string{"/proj"},
		},
		{
			name:     "双槽 diff 工具",
			actions:  mkmk(RoleDiffDir, "bcompare $0 $1"),
			role:     RoleDiffDir,
			paths:    []string{"/a", "/b"},
			wantBin:  "bcompare",
			wantArgs: []string{"/a", "/b"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fake := &fakeExecutor{}
			o, err := validateOpener(Opener{Name: "t", Actions: c.actions})
			if err != nil {
				t.Fatalf("validateOpener 失败: %v", err)
			}
			svc := &Service{executor: fake}
			if err := svc.Open(o, c.role, c.paths...); err != nil {
				t.Fatalf("Open 失败: %v", err)
			}
			if len(fake.calls) != 1 {
				t.Fatalf("期望 executor 被调用 1 次，实际 %d 次", len(fake.calls))
			}
			call := fake.calls[0]
			if call.bin != c.wantBin || !reflect.DeepEqual(call.args, c.wantArgs) {
				t.Fatalf("Open(%v) 执行 (%q,%v), 期望 (%q,%v)", c.paths, call.bin, call.args, c.wantBin, c.wantArgs)
			}
		})
	}
}

func TestOpenSlotCountMismatch(t *testing.T) {
	fake := &fakeExecutor{}
	o, err := validateOpener(Opener{Name: "t", Actions: mkmk(RoleOpenDir, "code $0")})
	if err != nil {
		t.Fatalf("validateOpener 失败: %v", err)
	}
	// open-dir 1 槽但传 2 个路径，应在 buildActionArgs 阶段报错，executor 不被调用
	svc := &Service{executor: fake}
	err = svc.Open(o, RoleOpenDir, "/a", "/b")
	if err == nil {
		t.Fatalf("期望槽个数不匹配报错，实际 nil")
	}
	if len(fake.calls) != 0 {
		t.Fatalf("期望 executor 未被调用，实际调用 %d 次", len(fake.calls))
	}
}

// ---------- Open role 前置校验（收敛在实现内） ----------

func TestOpenRoleUnsupported(t *testing.T) {
	fake := &fakeExecutor{}
	o, err := validateOpener(Opener{Name: "code", Actions: mkmk(RoleOpenDir, "code")})
	if err != nil {
		t.Fatalf("validateOpener 失败: %v", err)
	}
	// 用未声明的 role 调 Open 应在实现内报错，且不触达 executor
	svc := &Service{executor: fake}
	if err := svc.Open(o, RoleDiffDir, "/a"); err == nil {
		t.Fatal("期望 role 不支持报错")
	}
	if len(fake.calls) != 0 {
		t.Fatalf("期望 executor 未被调用，实际调用 %d 次", len(fake.calls))
	}
}
