package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cube/internal/testfixture"
)

// TestServiceCreateFromGitSource E2E：注册表指向本地仓库，按名引用走临时 clone 全链路生成。
func TestServiceCreateFromGitSource(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	repo := makeTplRepo(t, ws)
	s := NewService(ws.Cfg)

	if err := s.SaveTplSource(TplSource{Name: "core", RepoURL: repo}); err != nil {
		t.Fatalf("注册模板源失败: %v", err)
	}

	// 正常生成：变量注入 patterns 替换
	target := ws.Join("out", "app")
	if err := s.Create("core", "hello", target, map[string]string{"name": "app"}); err != nil {
		t.Fatalf("create 失败: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(target, "hello.txt"))
	if err != nil || string(data) != "hello app\n" {
		t.Fatalf("渲染产物不符: content=%q err=%v", data, err)
	}

	// 指定不存在的模板
	if err := s.Create("core", "nope", ws.Join("out", "b"), nil); err == nil || !strings.Contains(err.Error(), "模板") {
		t.Fatalf("不存在的模板应报错, got %v", err)
	}

	// 未注册源
	if err := s.Create("work", "hello", ws.Join("out", "c"), nil); err == nil || !strings.Contains(err.Error(), "未找到指定模板源") {
		t.Fatalf("未注册源应报错, got %v", err)
	}
}

// TestServiceCreateTargetPrecheck 预检先行：目标路径非法时不发起 clone（注册表指向坏地址也不应报拉取错误）。
func TestServiceCreateTargetPrecheck(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	s := NewService(ws.Cfg)
	// 注册一个不可 clone 的源：若预检没先行，会先撞 clone 失败而不是路径错误
	if err := s.SaveTplSource(TplSource{Name: "core", RepoURL: ws.Join("no-such-repo")}); err != nil {
		t.Fatalf("注册模板源失败: %v", err)
	}

	// 目标已存在且非空
	occupied := ws.Mkdir("occupied")
	ws.WriteFile(filepath.Join("occupied", "x.txt"), []byte("x"))
	err := s.Create("core", "hello", occupied, nil)
	if err == nil || !strings.Contains(err.Error(), "目标目录非空") {
		t.Fatalf("非空目标应先报路径错误（而非 clone 失败）, got %v", err)
	}
}
