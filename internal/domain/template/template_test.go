package template

import (
	"path/filepath"
	"strings"
	"testing"

	"cube/internal/testfixture"
	"cube/pkg/git"
)

func loadTestTemplate(t *testing.T) *Template {
	t.Helper()
	testDir := makeTestDir(t, map[string]string{
		"tpl/template.yaml":   "version: 1\n",
		"tpl/main.go":         "package main",
		"tpl/.git/config":     "git config",
		"tpl/sub/dir/util.go": "package util",
		"tpl/sub/__note__.md": "note",
		"tpl/other.txt":       "hello",
		"tpl/资产.bin":          "\x00\x01\x02",
		"unrelated/readme.md": "out of template",
	})

	tpl, err := LoadTemplate(filepath.Join(testDir, "tpl"))
	if err != nil {
		t.Fatalf("LoadTemplate 报错: %v", err)
	}
	if tpl.Meta() == nil || tpl.Meta().Version != 1 {
		t.Fatalf("meta 解析不符: %+v", tpl.Meta())
	}
	return tpl
}

func TestLoadTemplate(t *testing.T) {
	tests := []struct {
		name    string
		tplPath string
		wantErr string
	}{
		{"目录不存在", "/__missing__", "模板目录不存在"},
		{
			"meta 文件不存在",
			makeTestDir(t, map[string]string{
				"other.txt": "x",
			}),
			"文件不存在或非文件",
		},
		{
			"meta 解析失败",
			makeTestDir(t, map[string]string{
				MetaFileName: "version: 99\n",
			}),
			"解析模板 template.yaml 文件失败",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadTemplate(tt.tplPath)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("期望错误含 %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestTemplateWalkFile(t *testing.T) {
	tpl := loadTestTemplate(t)

	var files []string
	err := tpl.WalkFile(func(rel string) error {
		files = append(files, rel)
		return nil
	})
	if err != nil {
		t.Fatalf("WalkFile 报错: %v", err)
	}

	// 排序后对比，避免依赖遍历顺序
	want := []string{"main.go", "other.txt", "sub/__note__.md", "sub/dir/util.go", "资产.bin"}
	got := append([]string(nil), files...)
	sortStrings(got)
	if !equalStrings(got, want) {
		t.Fatalf("WalkFile 结果不符:\ngot  %v\nwant %v\nraw %v", got, want, files)
	}
}

// 仓库内模板目录按 git 视角遍历：忽略项（目录级/文件级）不进结果，
// 未跟踪未忽略的新文件可见（开发态吃未提交改动），与 clone 口径对齐。
// 非仓库目录的降级遍历由 TestTemplateWalkFile 覆盖（t.TempDir 不在仓库内）。
func TestTemplateWalkFileInGitRepo(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	repo := ws.MakeGitRepo("tpl-repo")
	ws.WriteFile(filepath.Join("tpl-repo", MetaFileName), []byte("version: 1\n"))
	ws.WriteFile(filepath.Join("tpl-repo", "main.go"), []byte("package main"))
	ws.WriteFile(filepath.Join("tpl-repo", ".gitignore"), []byte("node_modules/\nscratch.log\n"))
	// 忽略项：目录整体忽略 + 未跟踪的文件级忽略（tracked 文件不受 .gitignore 影响，
	// ls-files 与 clone 都会包含，此处不测该形态）
	ws.WriteFile(filepath.Join("tpl-repo", "node_modules", "dep", "index.js"), []byte("junk"))
	ws.WriteFile(filepath.Join("tpl-repo", "scratch.log"), []byte("junk"))
	if err := git.Add(repo, MetaFileName, "main.go", ".gitignore"); err != nil {
		t.Fatalf("git add 失败: %v", err)
	}
	if err := git.Commit(repo, "add template"); err != nil {
		t.Fatalf("git commit 失败: %v", err)
	}
	// commit 后再写：保持未跟踪未忽略
	ws.WriteFile(filepath.Join("tpl-repo", "draft.md"), []byte("draft"))

	tpl, err := LoadTemplate(repo)
	if err != nil {
		t.Fatalf("LoadTemplate 报错: %v", err)
	}

	var files []string
	err = tpl.WalkFile(func(rel string) error {
		files = append(files, rel)
		return nil
	})
	if err != nil {
		t.Fatalf("WalkFile 报错: %v", err)
	}

	sortStrings(files)
	want := []string{".gitignore", "draft.md", "main.go"}
	if !equalStrings(files, want) {
		t.Fatalf("WalkFile git 视角结果不符:\ngot  %v\nwant %v", files, want)
	}
}

func TestTemplateReadFile(t *testing.T) {
	tpl := loadTestTemplate(t)

	data, err := tpl.ReadFile("main.go")
	if err != nil {
		t.Fatalf("ReadFile 报错: %v", err)
	}
	if string(data) != "package main" {
		t.Fatalf("ReadFile 内容不符: %q", data)
	}

	if _, err := tpl.ReadFile("missing.txt"); err == nil {
		t.Fatal("读取不存在文件应报错")
	}
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
