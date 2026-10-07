// 公共测试用工具函数
package template

import (
	"path/filepath"
	"testing"

	"cube/util/oskit"
)

// 构建测试目录，返回目录地址
func makeTestDir(t *testing.T, files map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	for name, content := range files {
		err := oskit.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
		if err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// 构建测试 *Template
func makeTestTpl(t *testing.T, files map[string]string) *Template {
	t.Helper()

	dir := makeTestDir(t, files)

	tpl, err := LoadTemplate(dir)
	if err != nil {
		t.Fatalf("构建测试 *Template 失败: %s", err)
	}
	if tpl.Meta() == nil || tpl.Meta().Version != 1 {
		t.Fatalf("meta 解析不符: %+v", tpl.Meta())
	}
	return tpl
}
