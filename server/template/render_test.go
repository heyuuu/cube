package template

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// 收集目标目录下所有文件内容
func collectResultFiles(t *testing.T, dir string) map[string]string {
	result := make(map[string]string)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		// 读取文件
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("读取文件失败 %s: %w", path, err)
		}
		result[rel] = string(data)
		return nil
	})

	if err != nil {
		t.Fatalf("收集结果文件 collectResultFiles 失败: %v", err)
	}
	return result
}

func TestRender(t *testing.T) {
	tplMetaFile := `
version: 1
variables:
  - name: project-name
    prompt: 项目名
    required: true
  - name: module
    prompt: module
    required: true
patterns:
  - pattern: __MODULE__
    replace: "github.com/x/${module}"
    scope: "" # 缺省默认匹配所有
  - pattern: __PROJECT__
    replace: "${project-name}"
    scope: "**/*.go"
  - pattern: __PROJECT__
    replace: "${project-name}"
    scope: "**/*.md"
`
	vars := map[string]string{
		"module":       "myapp",
		"project-name": "demo",
	}
	tplFiles := map[string]string{
		MetaFileName: tplMetaFile,
		// 路径替换
		"__PROJECT__/main.go":        "package main // __MODULE__",
		"__PROJECT__/nested/util.go": "// __PROJECT__",
		// 内容替换
		"README.md": "# __PROJECT__",
		// 二进制
		"logo.png": "PN\x00G",
		// 应忽略的文件
		".git/config":    "[core]",
		".git/objects/x": "x",
	}
	wantFiles := map[string]string{
		"demo/main.go":        "package main // github.com/x/myapp",
		"demo/nested/util.go": "// demo",
		"README.md":           "# demo",
		"logo.png":            "PN\x00G",
	}

	// 构建测试 tpl
	tpl := makeTestTpl(t, tplFiles)
	target := t.TempDir()

	// 执行目标逻辑
	count, err := Render(tpl, target, vars)
	if err != nil {
		t.Fatalf("Render 报错: %v", err)
	}

	// 收集产生的文件
	resultFiles := collectResultFiles(t, target)

	// 验证 count 是否匹配
	if count != len(resultFiles) {
		t.Fatalf("count = %d, 实际写入 %d 个文件", count, len(resultFiles))
	}
	if count != len(wantFiles) {
		t.Fatalf("写入文件数为 %d, 预期文件数为 %d", count, len(wantFiles))
	}

	// 校验目标文件
	assert.Equalf(t, wantFiles, resultFiles, "生成文件与预期不符")
}

func TestIsBinary(t *testing.T) {
	tests := []struct {
		data []byte
		want bool
	}{
		{[]byte("plain text"), false},
		{[]byte(""), false},
		{[]byte("utf-8 中文 无 NUL"), false},
		{[]byte{0x01, 0x00, 0x02}, true},
		{[]byte("a\x00b"), true},
	}
	for _, tt := range tests {
		if got := isBinary(tt.data); got != tt.want {
			t.Fatalf("isBinary(%q) = %v, want %v", tt.data, got, tt.want)
		}
	}
}
