package create

import (
	"os"
	"path/filepath"
	"testing"

	"cube/internal/testfixture"
)

// makeTemplateDir 建一个含占位符目录名、多层文件、二进制文件的模板目录。
func makeTemplateDir(t *testing.T, ws *testfixture.Workspace, name string) string {
	dir := ws.Mkdir(name)
	ws.WriteFile(filepath.Join(name, "template.yaml"), []byte("version: 1\n"))
	ws.WriteFile(filepath.Join(name, "__PROJECT__/main.go"), []byte("package main // __MODULE__"))
	ws.WriteFile(filepath.Join(name, "__PROJECT__/nested/util.go"), []byte("// __PROJECT__"))
	ws.WriteFile(filepath.Join(name, "README.md"), []byte("# __PROJECT__"))
	if err := os.WriteFile(ws.Join(name, "logo.png"), []byte("PN\x00G"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func testTemplateYaml(t *testing.T) *TemplateMeta {
	tpl, err := InitTemplateMeta([]byte(`
version: 1
variables:
  project-name: {prompt: 项目名, required: true}
  module: {prompt: module, required: true}
patterns:
  "**/*.go":
    - {pattern: __MODULE__, replace: "github.com/x/${module}"}
    - {pattern: __PROJECT__, replace: "${project-name}"}
  "**/*.md":
    - {pattern: __PROJECT__, replace: "${project-name}"}
`))
	if err != nil {
		t.Fatal(err)
	}
	//tpl.Patterns_old["**/*.go"][0].Replace = "github.com/x/myapp"
	//tpl.Patterns_old["**/*.go"][1].Replace = "demo"
	//tpl.Patterns_old["**/*.md"][0].Replace = "demo"
	return tpl
}

func TestRender(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	templateDir := makeTemplateDir(t, ws, "tpl")
	target := ws.Join("out")

	count, err := Render(templateDir, target, testTemplateYaml(t), map[string]string{})
	if err != nil {
		t.Fatalf("Render 报错: %v", err)
	}
	// 4 个文件（main.go / nested/util.go / README.md / logo.png），template.yaml 不算
	if count != 4 {
		t.Fatalf("生成文件数 = %d, want 4", count)
	}

	read := func(rel string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(target, rel))
		if err != nil {
			t.Fatalf("读取 %s 失败: %v", rel, err)
		}
		return string(data)
	}

	// 目录名占位符替换（路径含文件名）
	if got := read("demo/main.go"); got != "package main // github.com/x/myapp" {
		t.Fatalf("main.go 内容替换失败: %q", got)
	}
	if got := read("demo/nested/util.go"); got != "// demo" {
		t.Fatalf("nested/util.go 替换失败: %q", got)
	}
	if got := read("README.md"); got != "# demo" {
		t.Fatalf("README 替换失败: %q", got)
	}
	// 二进制内容原样
	if got := read("logo.png"); got != "PN\x00G" {
		t.Fatalf("二进制文件内容被改动: %q", got)
	}
	// template.yaml 不被复制
	if _, err := os.Stat(filepath.Join(target, "template.yaml")); !os.IsNotExist(err) {
		t.Fatal("template.yaml 不应被复制到目标")
	}
}

func TestRenderSkipDotGit(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	templateDir := makeTemplateDir(t, ws, "tpl")
	ws.WriteFile(filepath.Join("tpl", ".git/config"), []byte("[core]"))
	ws.WriteFile(filepath.Join("tpl", ".git/objects/x"), []byte("x"))

	target := ws.Join("out")
	count, err := Render(templateDir, target, testTemplateYaml(t), map[string]string{})
	if err != nil {
		t.Fatalf("Render 报错: %v", err)
	}
	if count != 4 {
		t.Fatalf("生成文件数 = %d, want 4（.git 应被跳过）", count)
	}
	if _, err := os.Stat(filepath.Join(target, ".git")); !os.IsNotExist(err) {
		t.Fatal(".git 不应被生成")
	}
}

func TestIsBinary(t *testing.T) {
	if isBinary([]byte("hello")) {
		t.Fatal("纯文本被误判为二进制")
	}
	if !isBinary([]byte("a\x00b")) {
		t.Fatal("含 NUL 字节未被判定为二进制")
	}
}
