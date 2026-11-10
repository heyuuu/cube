package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

// writeCubeFile 在根下写 .cube/cube.json。
func writeCubeFile(t *testing.T, root, content string) {
	t.Helper()
	dir := filepath.Join(root, ".cube")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cube.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestResolveExplicit(t *testing.T) {
	root := makeMonorepo(t, "server", "web")
	writeCubeFile(t, root, `{"workspaces":[{"name":"服务端","path":"server"},{"name":"web 前端","path":"web"}]}`)

	got := Resolve(root)
	want := []Workspace{{Name: "服务端", Path: "server"}, {Name: "web 前端", Path: "web"}}
	if len(got) != len(want) {
		t.Fatalf("Resolve() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Resolve()[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

// 空数组是显式声明「没有任何 workspace」，不回落探测（定稿语义）。
func TestResolveExplicitEmptyBeatsDetection(t *testing.T) {
	root := makeMonorepo(t, "apps/web")
	writeCubeFile(t, root, `{"workspaces":[],"workspaceScanRule":"pnpm"}`)
	writePnpmWorkspace(t, root, "apps/*")

	if got := Resolve(root); len(got) != 0 {
		t.Fatalf("空 workspaces 应等价于无成员，不回落探测，got %v", got)
	}
}

// 坏条目逐个跳过：绝对路径 / 逃逸 / 不存在；好条目保留。
func TestResolveSkipsBadEntries(t *testing.T) {
	root := makeMonorepo(t, "server")
	writeCubeFile(t, root, `{"workspaces":[
		{"name":"好条目","path":"server"},
		{"name":"绝对路径","path":"/etc"},
		{"name":"逃逸","path":"../outside"},
		{"name":"不存在","path":"gone"},
		{"name":"缺路径"}
	]}`)

	got := Resolve(root)
	if len(got) != 1 || got[0].Name != "好条目" || got[0].Path != "server" {
		t.Fatalf("坏条目应逐个跳过，got %v", got)
	}
}

// workspaces 字段不存在 → workspaceScanRule 生效。
func TestResolveScanRuleWhenFieldAbsent(t *testing.T) {
	root := makeMonorepo(t, "apps/web")
	writeCubeFile(t, root, `{"workspaceScanRule":"pnpm"}`)
	writePnpmWorkspace(t, root, "apps/*")

	got := Resolve(root)
	if len(got) != 1 || got[0].Path != "apps/web" {
		t.Fatalf("scanRule 探测应生效，got %v", got)
	}
}

// cube.json 不存在 → 默认规则探测；坏 JSON 等价于无声明。
func TestResolveDefaultAndBadJson(t *testing.T) {
	root := makeMonorepo(t, "apps/web")
	writePnpmWorkspace(t, root, "apps/*")

	got := Resolve(root)
	if len(got) != 1 || got[0].Path != "apps/web" {
		t.Fatalf("无 cube.json 应用默认规则探测，got %v", got)
	}

	writeCubeFile(t, root, `{bad json`)
	if got := Resolve(root); len(got) != 1 || got[0].Path != "apps/web" {
		t.Fatalf("坏 JSON 应等价无声明走探测，got %v", got)
	}
}

// 显式声明 path 支持 glob：展开命中目录（覆盖 pnpm-workspace.yaml 的表达力）。
// 单命中时声明 name 生效；多命中时各取路径末段；无命中等价坏条目跳过。
func TestResolveGlobDeclared(t *testing.T) {
	root := makeMonorepo(t, "apps/web", "apps/api", "packages/ui", "docs")

	writeCubeFile(t, root, `{"workspaces":[{"path":"apps/*"},{"name":"UI 库","path":"packages/*"}]}`)
	got := Resolve(root)
	if len(got) != 3 {
		t.Fatalf("glob 声明应展开 apps/* + packages/*，got %v", got)
	}
	byPath := map[string]string{}
	for _, w := range got {
		byPath[w.Path] = w.Name
	}
	// 多命中（apps/*）取路径末段；单命中（packages/*）用声明的 name
	if byPath["apps/web"] != "web" || byPath["apps/api"] != "api" || byPath["packages/ui"] != "UI 库" {
		t.Fatalf("命名规则不符，got %v", byPath)
	}

	// 无命中的通配条目跳过，不阻断其余条目
	writeCubeFile(t, root, `{"workspaces":[{"path":"nothing/*"},{"path":"docs"}]}`)
	if got := Resolve(root); len(got) != 1 || got[0].Path != "docs" {
		t.Fatalf("无命中 glob 条目应跳过，got %v", got)
	}
}

// glob 排除项与探测同口径：node_modules / 隐藏目录不进显式声明展开结果。
func TestResolveGlobExcludes(t *testing.T) {
	root := makeMonorepo(t, "apps/web", "apps/node_modules/pkg", "apps/.hidden")
	writeCubeFile(t, root, `{"workspaces":[{"path":"apps/*"}]}`)

	got := Resolve(root)
	if len(got) != 1 || got[0].Path != "apps/web" {
		t.Fatalf("glob 声明应排除 node_modules 与隐藏目录，got %v", got)
	}
}
