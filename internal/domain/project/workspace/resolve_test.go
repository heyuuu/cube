package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"cube/internal/testfixture"
)

// makeMonorepo 建一个含给定子目录的 monorepo 骨架（无 git，workspace 解析不需要）。
func makeMonorepo(t *testing.T, subdirs ...string) string {
	t.Helper()
	root := testfixture.NewWorkspace(t).Dir
	for _, sub := range subdirs {
		if err := os.MkdirAll(filepath.Join(root, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func writeRootFile(t *testing.T, root, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writePnpmWorkspace(t *testing.T, root, pattern string) {
	t.Helper()
	writeRootFile(t, root, "pnpm-workspace.yaml", "packages:\n  - '"+pattern+"'\n")
}

func writePackageJson(t *testing.T, root, content string) {
	t.Helper()
	writeRootFile(t, root, "package.json", content)
}

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

func assertWorkspaces(t *testing.T, got, want []Workspace) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("[%d] = %v, want %v（全量 got %v）", i, got[i], want[i], got)
		}
	}
}

// 精确目录规则：声明 name 用声明值，缺省回退相对路径本身；顺序即声明序。
func TestResolveExplicit(t *testing.T) {
	root := makeMonorepo(t, "server", "web", "tools/cli")
	writeCubeFile(t, root, `{"workspaces":[{"name":"服务端","path":"server"},{"name":"web 前端","path":"web"},{"path":"tools/cli"}]}`)

	assertWorkspaces(t, Resolve(root), []Workspace{
		{Name: "服务端", Path: "server"},
		{Name: "web 前端", Path: "web"},
		{Name: "tools/cli", Path: "tools/cli"},
	})
}

// 空数组是显式声明「没有任何 workspace」，不回落默认规则。
func TestResolveEmptyDeclaredBeatsDefault(t *testing.T) {
	root := makeMonorepo(t, "apps/web")
	writeCubeFile(t, root, `{"workspaces":[]}`)
	writePnpmWorkspace(t, root, "apps/*")

	if got := Resolve(root); len(got) != 0 {
		t.Fatalf("空 workspaces 应等价于无成员，不回落默认规则，got %v", got)
	}
}

// 坏条目逐个跳过：绝对路径 / 逃逸 / 不存在 / 缺路径；好条目保留。
func TestResolveSkipsBadEntries(t *testing.T) {
	root := makeMonorepo(t, "server")
	writeCubeFile(t, root, `{"workspaces":[
		{"name":"好条目","path":"server"},
		{"name":"绝对路径","path":"/etc"},
		{"name":"逃逸","path":"../outside"},
		{"name":"不存在","path":"gone"},
		{"name":"缺路径"}
	]}`)

	assertWorkspaces(t, Resolve(root), []Workspace{{Name: "好条目", Path: "server"}})
}

// cube.json 不存在 / workspaces 节缺失 → 默认规则（<npm>+<pnpm>）生效；
// 默认规则不带 name，成员显示名即相对路径（无前导冒号）。
func TestResolveDefaultRules(t *testing.T) {
	root := makeMonorepo(t, "apps/web")
	writePnpmWorkspace(t, root, "apps/*")
	want := []Workspace{{Name: "apps/web", Path: "apps/web"}}

	assertWorkspaces(t, Resolve(root), want)

	writeCubeFile(t, root, `{"futureSection":{"a":1}}`)
	assertWorkspaces(t, Resolve(root), want)
}

// 坏 JSON 降级为无成员：不回落默认规则，避免掩盖手工搞坏的文件。
func TestResolveBrokenCubeFile(t *testing.T) {
	root := makeMonorepo(t, "apps/web")
	writePnpmWorkspace(t, root, "apps/*")
	writeCubeFile(t, root, `{bad json`)

	if got := Resolve(root); len(got) != 0 {
		t.Fatalf("坏 JSON 应降级为无成员，got %v", got)
	}
}

// <npm> 读 package.json 的 workspaces 节：数组与 {packages: []}（nohoist 等场景）两种形态。
func TestResolveNpmRule(t *testing.T) {
	rules := `{"workspaces":[{"path":"<npm>"}]}`

	root := makeMonorepo(t, "apps/web", "packages/ui", "docs")
	writeCubeFile(t, root, rules)
	writePackageJson(t, root, `{"name":"m","workspaces":["apps/*","packages/*"]}`)
	assertWorkspaces(t, Resolve(root), []Workspace{
		{Name: "apps/web", Path: "apps/web"},
		{Name: "packages/ui", Path: "packages/ui"},
	})

	root2 := makeMonorepo(t, "apps/web")
	writeCubeFile(t, root2, rules)
	writePackageJson(t, root2, `{"workspaces":{"packages":["apps/*"]}}`)
	assertWorkspaces(t, Resolve(root2), []Workspace{{Name: "apps/web", Path: "apps/web"}})

	// 无 workspaces 节 / 无 package.json → 无命中
	root3 := makeMonorepo(t, "apps/web")
	writeCubeFile(t, root3, rules)
	if got := Resolve(root3); len(got) != 0 {
		t.Fatalf("无 package.json 应无命中，got %v", got)
	}
	writePackageJson(t, root3, `{"name":"m"}`)
	if got := Resolve(root3); len(got) != 0 {
		t.Fatalf("package.json 无 workspaces 节应无命中，got %v", got)
	}
}

// <pnpm> 读 pnpm-workspace.yaml 的 packages：apps/* 只命中一层，未声明的目录不进结果。
func TestResolvePnpmRule(t *testing.T) {
	root := makeMonorepo(t, "apps/web", "apps/api", "packages/ui", "packages/ui/src", "docs")
	writeCubeFile(t, root, `{"workspaces":[{"path":"<pnpm>"}]}`)
	writePnpmWorkspace(t, root, "apps/*")

	assertWorkspaces(t, Resolve(root), []Workspace{
		{Name: "apps/api", Path: "apps/api"},
		{Name: "apps/web", Path: "apps/web"},
	})
}

// ** 递归命中各层目录，但模式的静态前缀是「容器目录」不是成员。
func TestResolvePnpmDoubleStar(t *testing.T) {
	root := makeMonorepo(t, "libs/a/core", "docs")
	writeCubeFile(t, root, `{"workspaces":[{"path":"<pnpm>"}]}`)
	writePnpmWorkspace(t, root, "libs/**")

	assertWorkspaces(t, Resolve(root), []Workspace{
		{Name: "libs/a", Path: "libs/a"},
		{Name: "libs/a/core", Path: "libs/a/core"},
	})
}

// 通配/特殊规则的 name 是组前缀：成员显示名为 name:相对路径。
func TestResolveNamePrefix(t *testing.T) {
	root := makeMonorepo(t, "apps/web", "packages/ui")
	writePnpmWorkspace(t, root, "apps/*")
	writeCubeFile(t, root, `{"workspaces":[{"name":"js","path":"<pnpm>"},{"name":"UI 库","path":"packages/*"}]}`)

	assertWorkspaces(t, Resolve(root), []Workspace{
		{Name: "js:apps/web", Path: "apps/web"},
		{Name: "UI 库:packages/ui", Path: "packages/ui"},
	})
}

// 规则 path 支持 glob：展开命中目录（覆盖 pnpm-workspace.yaml 的表达力）；
// 无命中的通配条目跳过，不阻断其余条目。
func TestResolveGlobDeclared(t *testing.T) {
	root := makeMonorepo(t, "apps/web", "apps/api", "packages/ui", "docs")

	writeCubeFile(t, root, `{"workspaces":[{"path":"apps/*"},{"path":"packages/*"}]}`)
	assertWorkspaces(t, Resolve(root), []Workspace{
		{Name: "apps/api", Path: "apps/api"},
		{Name: "apps/web", Path: "apps/web"},
		{Name: "packages/ui", Path: "packages/ui"},
	})

	writeCubeFile(t, root, `{"workspaces":[{"path":"nothing/*"},{"path":"docs"}]}`)
	assertWorkspaces(t, Resolve(root), []Workspace{{Name: "docs", Path: "docs"}})
}

// glob 排除项：node_modules / 点前缀 / 下划线前缀目录不进结果（与 scan 的跳过口径一致）。
func TestResolveGlobExcludes(t *testing.T) {
	root := makeMonorepo(t, "apps/web", "apps/node_modules/pkg", "apps/.hidden", "apps/_draft")
	writeCubeFile(t, root, `{"workspaces":[{"path":"apps/*"}]}`)

	assertWorkspaces(t, Resolve(root), []Workspace{{Name: "apps/web", Path: "apps/web"}})
}

// 多条规则命中同一目录只留一份（并集全保留）：默认 <npm>+<pnpm> 双声明是常态。
func TestResolveDedup(t *testing.T) {
	root := makeMonorepo(t, "apps/web", "packages/ui")
	writePnpmWorkspace(t, root, "apps/*")
	writePackageJson(t, root, `{"workspaces":["apps/*","packages/*"]}`)

	assertWorkspaces(t, Resolve(root), []Workspace{
		{Name: "apps/web", Path: "apps/web"},
		{Name: "packages/ui", Path: "packages/ui"},
	})
}

// 去重时先出现的规则命名保留：精确声明排在通配/特殊规则前即可给个别成员起名。
func TestResolveDedupKeepsFirstName(t *testing.T) {
	root := makeMonorepo(t, "apps/web", "apps/api")
	writePnpmWorkspace(t, root, "apps/*")
	writeCubeFile(t, root, `{"workspaces":[{"name":"前端","path":"apps/web"},{"path":"<pnpm>"}]}`)

	assertWorkspaces(t, Resolve(root), []Workspace{
		{Name: "前端", Path: "apps/web"},
		{Name: "apps/api", Path: "apps/api"},
	})
}
