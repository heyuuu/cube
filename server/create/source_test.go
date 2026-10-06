package create

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSourceTemplates(t *testing.T) {
	sourcePath := makeTestDir(t, map[string]string{
		// 正常项
		"templates/go-cli/" + MetaFileName: "version: 1\n",
		"templates/go-cli/main.go":         "package main",
		"templates/lib-a/" + MetaFileName:  "version: 1\n",
		"templates/lib-b/readme.md":        "无 meta",
		// 干扰项：templates/ 内的非模板子目录、隐藏目录不算
		"templates/docs/readme.md":          "非模板目录",
		"templates/.hidden/" + MetaFileName: "version: 1\n",
		// 干扰项：非直接子目录
		"templates/drafts/wip" + MetaFileName: "version: 1\n",
		// 根目录其他内容不参与判定：README、docs、草稿目录（即使含 template.yaml）均不可见
		"templates/afile.txt": "文件非目录",
		"docs/readme.md":      "无关目录",
	})

	paths, err := LoadSourceTemplates(sourcePath)
	if err != nil {
		t.Fatalf("LoadSourceTemplates 报错: %v", err)
	}

	if len(paths) != 2 {
		t.Fatalf("模板数 = %d, want 2 (过滤无 meta/.开头/文件), got %v", len(paths), paths)
	}
	if paths["go-cli"] != filepath.Join(sourcePath, "templates/go-cli") {
		t.Fatalf("go-cli 路径不符: %q", paths["go-cli"])
	}
	if _, ok := paths["lib-b"]; ok {
		t.Fatal("无 template.yaml 的目录不应被识别为模板")
	}
	if _, ok := paths[".hidden"]; ok {
		t.Fatal(". 开头目录应被跳过")
	}

	// 不存在的 source 目录
	if _, err := LoadSourceTemplates(filepath.Join(t.TempDir(), "missing")); err == nil || !strings.Contains(err.Error(), "不是合法模板来源") {
		t.Fatalf("期望“不是合法模板来源”错误, got %v", err)
	}
}
