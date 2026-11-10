package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"cube/internal/domain/project/cubefile"
)

// Save 把解析出的成员固化为具体规则：写读往返一致，空清单拒绝。
func TestSave(t *testing.T) {
	root := makeMonorepo(t, "apps/web", "apps/api")
	writePnpmWorkspace(t, root, "apps/*")

	picked := Resolve(root)[:1]
	if err := Save(root, picked); err != nil {
		t.Fatal(err)
	}
	if !cubefile.WorkspaceRulesDeclared(root) {
		t.Fatal("Save 后应为显式声明")
	}
	// 固化后不再跟随 pnpm 声明：只剩挑中的成员
	assertWorkspaces(t, Resolve(root), picked)

	if err := Save(root, nil); err == nil {
		t.Fatal("空清单应拒绝")
	}
}

// SaveRules 只做结构校验：glob 与特殊规则即使暂无命中也可保存。
func TestSaveRules(t *testing.T) {
	root := makeMonorepo(t, "apps/web")
	rules := []cubefile.WorkspaceRule{
		{Name: "前端", Path: "apps/*"},
		{Path: cubefile.PnpmRule},
		{Path: "not-yet/*"},
	}

	if err := SaveRules(root, rules); err != nil {
		t.Fatal(err)
	}
	got := cubefile.LoadWorkspaceRules(root)
	if len(got) != len(rules) {
		t.Fatalf("规则应原样落盘, got %+v", got)
	}
	for i := range rules {
		if got[i] != rules[i] {
			t.Fatalf("[%d] = %+v, want %+v", i, got[i], rules[i])
		}
	}
	assertWorkspaces(t, Resolve(root), []Workspace{{Name: "前端:apps/web", Path: "apps/web"}})
}

// 写侧严格：任一坏条目整体拒绝且不落盘（区别于采集侧的逐条静默跳过）。
func TestSaveRulesRejectsBadRule(t *testing.T) {
	root := makeMonorepo(t, "server")

	for _, bad := range []cubefile.WorkspaceRule{
		{Name: "缺路径"},
		{Path: "/etc"},
		{Path: "../outside"},
		{Path: "."},
	} {
		err := SaveRules(root, []cubefile.WorkspaceRule{{Path: "server"}, bad})
		if err == nil {
			t.Fatalf("坏规则 %+v 应被拒绝", bad)
		}
	}
	if _, err := os.Stat(filepath.Join(root, ".cube", "cube.json")); !os.IsNotExist(err) {
		t.Fatalf("校验失败不应落盘, stat err=%v", err)
	}
}

// 空清单是显式「无 workspace」声明（压过默认规则）；nil 同义，不能被当成删节。
func TestSaveRulesEmptyIsExplicit(t *testing.T) {
	for name, rules := range map[string][]cubefile.WorkspaceRule{"empty": {}, "nil": nil} {
		t.Run(name, func(t *testing.T) {
			root := makeMonorepo(t, "apps/web")
			writePnpmWorkspace(t, root, "apps/*")

			if err := SaveRules(root, rules); err != nil {
				t.Fatal(err)
			}
			if !cubefile.WorkspaceRulesDeclared(root) {
				t.Fatal("空清单应落盘为显式声明")
			}
			if got := Resolve(root); len(got) != 0 {
				t.Fatalf("显式空声明应无成员，got %v", got)
			}
		})
	}
}

// Reset 删节后回落默认规则。
func TestReset(t *testing.T) {
	root := makeMonorepo(t, "apps/web", "server")
	writePnpmWorkspace(t, root, "apps/*")

	if err := SaveRules(root, []cubefile.WorkspaceRule{{Path: "server"}}); err != nil {
		t.Fatal(err)
	}
	assertWorkspaces(t, Resolve(root), []Workspace{{Name: "server", Path: "server"}})

	if err := Reset(root); err != nil {
		t.Fatal(err)
	}
	if cubefile.WorkspaceRulesDeclared(root) {
		t.Fatal("Reset 后应回到未声明")
	}
	assertWorkspaces(t, Resolve(root), []Workspace{{Name: "apps/web", Path: "apps/web"}})
}
