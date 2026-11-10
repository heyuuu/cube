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

func writePnpmWorkspace(t *testing.T, root, pattern string) {
	t.Helper()
	content := "packages:\n  - '" + pattern + "'\n"
	if err := os.WriteFile(filepath.Join(root, "pnpm-workspace.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDetectPnpm(t *testing.T) {
	root := makeMonorepo(t, "apps/web", "apps/api", "packages/ui", "packages/ui/src", "docs")
	writePnpmWorkspace(t, root, "apps/*")

	got := Detect(root, "pnpm")
	// apps/* 只命中一层：apps/web、apps/api；packages 未声明不进结果
	if len(got) != 2 || got[0].Path != "apps/api" || got[1].Path != "apps/web" {
		t.Fatalf("Detect(pnpm) = %v", got)
	}
}

func TestDetectPnpmDoubleStar(t *testing.T) {
	root := makeMonorepo(t, "libs/a/core", "docs")
	writePnpmWorkspace(t, root, "libs/**")

	got := Detect(root, "pnpm")
	if len(got) != 2 { // libs/a 与 libs/a/core（** 递归）
		t.Fatalf("Detect(pnpm **) = %v", got)
	}
}

func TestDetectNpmListAndObject(t *testing.T) {
	// 数组形态
	root := makeMonorepo(t, "apps/web", "packages/ui")
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"m","workspaces":["apps/*","packages/*"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Detect(root, "npm"); len(got) != 2 {
		t.Fatalf("Detect(npm list) = %v", got)
	}

	// 对象形态（nohoist 等场景）
	root2 := makeMonorepo(t, "apps/web")
	if err := os.WriteFile(filepath.Join(root2, "package.json"), []byte(`{"workspaces":{"packages":["apps/*"]}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Detect(root2, "npm"); len(got) != 1 || got[0].Path != "apps/web" {
		t.Fatalf("Detect(npm object) = %v", got)
	}
}

// 规则按序生效：第一个命中的探测器赢，后续不再看。
func TestDetectOrder(t *testing.T) {
	root := makeMonorepo(t, "apps/web")
	writePnpmWorkspace(t, root, "apps/*")
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"workspaces":["apps/*"]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// pnpm 命名来源优先：两处都展开 apps/web，名字应同为 web——用来源区分不可行，
	// 这里验证的是「pnpm|npm 与 npm+pnpm 结果一致（都命中同一目录集）」以及未知规则跳过
	for _, rule := range []string{"pnpm|npm", "npm|pnpm", "ghost|pnpm", "ghost+pnpm"} {
		if got := Detect(root, rule); len(got) != 1 {
			t.Fatalf("Detect(%s) = %v, want 1 个成员", rule, got)
		}
	}
	if got := Detect(root, "ghost"); got != nil {
		t.Fatalf("全部未知规则应无结果，got %v", got)
	}
}

// 组内 + 合并去重：两份声明展开结果有交集时只留一份，并集全保留。
func TestDetectGroupMerge(t *testing.T) {
	root := makeMonorepo(t, "apps/web", "packages/ui")
	writePnpmWorkspace(t, root, "apps/*")
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"workspaces":["apps/*","packages/*"]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	got := Detect(root, "pnpm+npm")
	if len(got) != 2 || got[0].Path != "apps/web" || got[1].Path != "packages/ui" {
		t.Fatalf("pnpm+npm 应合并去重，got %v", got)
	}
}

// 规则组按序回落：第一组无命中时落到下一组；逗号不再是分隔符（整段视为一个规则名 → 未知跳过）。
func TestDetectGroupFallback(t *testing.T) {
	root := makeMonorepo(t, "apps/web")
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"workspaces":["apps/*"]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := Detect(root, "pnpm|npm"); len(got) != 1 || got[0].Path != "apps/web" {
		t.Fatalf("第一组未命中应回落 npm 组，got %v", got)
	}
	if got := Detect(root, "pnpm,npm"); got != nil {
		t.Fatalf("逗号不再是分隔符，整段为未知规则应无结果，got %v", got)
	}
}

// glob 排除项：node_modules / 点前缀目录不进结果。
func TestDetectExcludes(t *testing.T) {
	root := makeMonorepo(t, "apps/web", "apps/node_modules/pkg", "apps/.hidden")
	writePnpmWorkspace(t, root, "apps/*")

	got := Detect(root, "pnpm")
	if len(got) != 1 || got[0].Path != "apps/web" {
		t.Fatalf("应排除 node_modules 与隐藏目录，got %v", got)
	}
}
