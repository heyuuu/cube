package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cube/internal/testfixture"
)

// TestServiceOpenSourceAndGenerate E2E：注册表指向本地仓库，按名引用走临时 clone 全链路生成。
func TestServiceOpenSourceAndGenerate(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	repo := makeTplRepo(t, ws)
	s := NewService(ws.Cfg)

	if err := s.SaveTplSource(TplSource{Name: "core", RepoURL: repo}); err != nil {
		t.Fatalf("注册模板源失败: %v", err)
	}

	src, err := s.OpenSource("core")
	if err != nil {
		t.Fatalf("open source 失败: %v", err)
	}
	defer src.Cleanup()

	// 模板清单
	if names := src.Names(); len(names) != 1 || names[0] != "hello" {
		t.Fatalf("模板清单不符: %v", names)
	}

	// 正常生成：变量注入 patterns 替换
	tpl, err := src.Template("hello")
	if err != nil {
		t.Fatalf("加载模板失败: %v", err)
	}
	target := ws.Join("out", "app")
	if count, err := s.Generate(tpl, target, map[string]string{"name": "app"}); err != nil || count != 1 {
		t.Fatalf("generate 失败: count=%d err=%v", count, err)
	}
	data, err := os.ReadFile(filepath.Join(target, "hello.txt"))
	if err != nil || string(data) != "hello app\n" {
		t.Fatalf("渲染产物不符: content=%q err=%v", data, err)
	}

	// 指定不存在的模板
	if _, err := src.Template("nope"); err == nil || !strings.Contains(err.Error(), "模板") {
		t.Fatalf("不存在的模板应报错, got %v", err)
	}

	// 未注册源
	if _, err := s.OpenSource("work"); err == nil || !strings.Contains(err.Error(), "未找到指定模板源") {
		t.Fatalf("未注册源应报错, got %v", err)
	}
}

// TestServiceCheckTarget 预检：目标已存在且非空时报路径错误（cmd 层先于 clone/交互调用）。
func TestServiceCheckTarget(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	s := NewService(ws.Cfg)

	occupied := ws.Mkdir("occupied")
	ws.WriteFile(filepath.Join("occupied", "x.txt"), []byte("x"))
	if err := s.CheckTarget(occupied); err == nil || !strings.Contains(err.Error(), "目标目录非空") {
		t.Fatalf("非空目标应报路径错误, got %v", err)
	}

	if err := s.CheckTarget(ws.Join("out", "free")); err != nil {
		t.Fatalf("不存在的目标路径应通过预检, got %v", err)
	}
}
