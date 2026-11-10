package cubefile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cube/internal/testfixture"
)

func writeFile(t *testing.T, root, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ".cube"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".cube", "cube.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustRead(t *testing.T, root string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, ".cube", "cube.json"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func isDefaultRules(rules []WorkspaceRule) bool {
	return len(rules) == 2 && rules[0] == WorkspaceRule{Path: NpmRule} && rules[1] == WorkspaceRule{Path: PnpmRule}
}

// 存在性语义：文件缺失 / 节缺失都回落默认规则且视为未声明；节存在才是显式声明。
func TestLoadWorkspaceRulesPresence(t *testing.T) {
	root := testfixture.NewWorkspace(t).Dir

	if rules := LoadWorkspaceRules(root); !isDefaultRules(rules) {
		t.Fatalf("文件缺失应返回默认规则, got %+v", rules)
	}
	if WorkspaceRulesDeclared(root) {
		t.Fatal("文件缺失不应视为已声明")
	}

	writeFile(t, root, `{"futureSection":{"a":1}}`)
	if rules := LoadWorkspaceRules(root); !isDefaultRules(rules) {
		t.Fatalf("节缺失应返回默认规则, got %+v", rules)
	}
	if WorkspaceRulesDeclared(root) {
		t.Fatal("节缺失不应视为已声明")
	}

	writeFile(t, root, `{"workspaces":[{"name":"前端","path":"apps/*"},{"path":"<npm>"}]}`)
	rules := LoadWorkspaceRules(root)
	want := []WorkspaceRule{{Name: "前端", Path: "apps/*"}, {Path: NpmRule}}
	if len(rules) != len(want) || rules[0] != want[0] || rules[1] != want[1] {
		t.Fatalf("显式规则应原样读出, got %+v", rules)
	}
	if !WorkspaceRulesDeclared(root) {
		t.Fatal("节存在应视为已声明")
	}
}

// 空数组是显式声明（没有任何 workspace），不回落默认规则。
func TestLoadWorkspaceRulesEmptyIsDeclared(t *testing.T) {
	root := testfixture.NewWorkspace(t).Dir
	writeFile(t, root, `{"workspaces":[]}`)

	rules := LoadWorkspaceRules(root)
	if rules == nil || len(rules) != 0 {
		t.Fatalf("空数组应读回非 nil 空清单, got %#v", rules)
	}
	if !WorkspaceRulesDeclared(root) {
		t.Fatal("空数组应视为已声明")
	}
}

// 坏 JSON / 节格式坏属可恢复降级：规则为空（不回落默认，避免掩盖手工搞坏的文件）且视为未声明。
func TestLoadWorkspaceRulesBroken(t *testing.T) {
	root := testfixture.NewWorkspace(t).Dir

	for _, content := range []string{`{bad`, `{"workspaces":"oops"}`} {
		writeFile(t, root, content)
		if rules := LoadWorkspaceRules(root); len(rules) != 0 {
			t.Fatalf("坏内容 %q 应降级为空, got %+v", content, rules)
		}
		if WorkspaceRulesDeclared(root) {
			t.Fatalf("坏内容 %q 不应视为已声明", content)
		}
	}
}

// Save 整节覆盖且保留未知节：cube.json 是多节容器，不认识的节不许被写丢。
func TestSaveWorkspaceRulesPreservesUnknownSections(t *testing.T) {
	root := testfixture.NewWorkspace(t).Dir
	writeFile(t, root, `{"workspaces":[{"path":"old"}],"futureSection":{"a":1}}`)

	if err := SaveWorkspaceRules(root, []WorkspaceRule{{Name: "s", Path: "server"}}); err != nil {
		t.Fatal(err)
	}
	out := mustRead(t, root)
	if !strings.Contains(out, `"futureSection"`) {
		t.Fatalf("未知节应保留: %s", out)
	}
	if strings.Contains(out, `"old"`) {
		t.Fatalf("workspaces 节应整体覆盖: %s", out)
	}
	rules := LoadWorkspaceRules(root)
	if len(rules) != 1 || rules[0] != (WorkspaceRule{Name: "s", Path: "server"}) {
		t.Fatalf("写读往返不符: %+v", rules)
	}
}

// 文件不存在时 Save 直接创建（含 .cube 目录）。
func TestSaveWorkspaceRulesCreatesFile(t *testing.T) {
	root := testfixture.NewWorkspace(t).Dir

	if err := SaveWorkspaceRules(root, []WorkspaceRule{{Path: "server"}}); err != nil {
		t.Fatal(err)
	}
	if rules := LoadWorkspaceRules(root); len(rules) != 1 || rules[0].Path != "server" {
		t.Fatalf("新建文件写读往返不符: %+v", rules)
	}
	// name 为空不落盘（omitempty），避免空串噪音进 git
	if out := mustRead(t, root); strings.Contains(out, `"name"`) {
		t.Fatalf("空 name 不应写出: %s", out)
	}
}

// 空清单与 nil 语义不同：空清单落盘为显式空声明，nil 等价删节。
func TestSaveWorkspaceRulesEmptyVersusNil(t *testing.T) {
	root := testfixture.NewWorkspace(t).Dir

	if err := SaveWorkspaceRules(root, []WorkspaceRule{}); err != nil {
		t.Fatal(err)
	}
	if !WorkspaceRulesDeclared(root) || len(LoadWorkspaceRules(root)) != 0 {
		t.Fatalf("空清单应落盘为显式空声明: %s", mustRead(t, root))
	}

	if err := SaveWorkspaceRules(root, nil); err != nil {
		t.Fatal(err)
	}
	if WorkspaceRulesDeclared(root) || !isDefaultRules(LoadWorkspaceRules(root)) {
		t.Fatalf("nil 应删节回落默认规则: %s", mustRead(t, root))
	}
}

// Reset 只删 workspaces 节：回落默认规则，其余节原样保留；文件/节不存在视为成功。
func TestResetWorkspaceRules(t *testing.T) {
	root := testfixture.NewWorkspace(t).Dir

	if err := ResetWorkspaceRules(root); err != nil {
		t.Fatalf("文件不存在时 reset 应成功: %v", err)
	}

	writeFile(t, root, `{"workspaces":[{"path":"server"}],"futureSection":{"a":1}}`)
	if err := ResetWorkspaceRules(root); err != nil {
		t.Fatal(err)
	}
	if WorkspaceRulesDeclared(root) || !isDefaultRules(LoadWorkspaceRules(root)) {
		t.Fatalf("reset 后应回落默认规则: %s", mustRead(t, root))
	}
	if out := mustRead(t, root); !strings.Contains(out, `"futureSection"`) {
		t.Fatalf("reset 只应删 workspaces 节: %s", out)
	}
}

// 坏 JSON 时拒写：防止静默覆盖掉手工搞坏的其他节。
func TestSaveWorkspaceRulesRefusesBrokenFile(t *testing.T) {
	root := testfixture.NewWorkspace(t).Dir
	writeFile(t, root, `{bad`)

	if err := SaveWorkspaceRules(root, []WorkspaceRule{{Path: "server"}}); err == nil {
		t.Fatal("坏 JSON 时 Save 应报错")
	}
	if out := mustRead(t, root); out != `{bad` {
		t.Fatalf("坏文件不应被改写: %s", out)
	}
}
