package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cube/internal/testfixture"
	"cube/util/git"
	"cube/util/oskit"
)

// makeTplRepo 建一个含单模板集的本地 git 仓库（本地路径直接当 repoUrl 供 clone）。
func makeTplRepo(t *testing.T, ws *testfixture.Workspace) string {
	t.Helper()
	repo := ws.MakeGitRepo("tpl-repo")
	meta := "version: 1\n" +
		"variables:\n" +
		"  - name: name\n" +
		"    prompt: 项目名\n" +
		"    required: true\n" +
		"patterns:\n" +
		"  - pattern: __NAME__\n" +
		"    replace: ${name}\n"
	ws.WriteFile(filepath.Join("tpl-repo", "templates", "hello", MetaFileName), []byte(meta))
	ws.WriteFile(filepath.Join("tpl-repo", "templates", "hello", "hello.txt"), []byte("hello __NAME__\n"))
	if err := git.Add(repo, "."); err != nil {
		t.Fatalf("git add 失败: %v", err)
	}
	if err := git.Commit(repo, "add template"); err != nil {
		t.Fatalf("git commit 失败: %v", err)
	}
	return repo
}

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

// TestLoadSourceDir 三形态解析：本地路径直读 / 名字查注册表临时 clone / 未注册报错。
func TestLoadSourceDir(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	repo := makeTplRepo(t, ws)
	sources := []TplSource{{Name: "core", RepoUrl: repo}}

	// 本地绝对路径：直读，无 cleanup
	dir, cleanup, err := loadSourceDir(repo, sources)
	if err != nil || dir != repo || cleanup != nil {
		t.Fatalf("路径形态应直读原路径无 cleanup: dir=%s cleanup=%v err=%v", dir, cleanup != nil, err)
	}

	// 不存在的本地路径
	if _, _, err := loadSourceDir(filepath.Join(ws.Dir, "missing"), sources); err == nil || !strings.Contains(err.Error(), "模板源路径不存在") {
		t.Fatalf("不存在的路径应报中文错误, got %v", err)
	}

	// 名字：查注册表 → 临时 clone，cleanup 后目录删除
	cloned, cleanup, err := loadSourceDir("core", sources)
	if err != nil {
		t.Fatalf("命名源解析失败: %v", err)
	}
	if cloned == repo || !oskit.IsDir(filepath.Join(cloned, "templates", "hello")) {
		t.Fatalf("应 clone 到新目录且含模板: cloned=%s", cloned)
	}
	cleanup()
	if _, err := os.Stat(cloned); !os.IsNotExist(err) {
		t.Fatalf("cleanup 后临时目录应删除: %s, err=%v", cloned, err)
	}

	// 未注册名：中文错误；缺省源缺配置时附建议地址
	_, _, err = loadSourceDir("work", sources)
	if err == nil || !strings.Contains(err.Error(), "未找到指定模板源: work") || strings.Contains(err.Error(), DefaultSourceRepoUrl) {
		t.Fatalf("普通源未注册应报错且不含建议地址, got %v", err)
	}
	_, _, err = loadSourceDir("core", nil)
	if err == nil || !strings.Contains(err.Error(), DefaultSourceRepoUrl) {
		t.Fatalf("缺省源未注册应附建议地址, got %v", err)
	}
}

// TestServiceCreateFromGitSource E2E：注册表指向本地仓库，按名引用走临时 clone 全链路生成。
func TestServiceCreateFromGitSource(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	repo := makeTplRepo(t, ws)
	s := NewService(ws.Join("settings.json"))
	if err := s.SaveTplSource(TplSource{Name: "core", RepoUrl: repo}); err != nil {
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
	s := NewService(ws.Join("settings.json"))
	// 注册一个不可 clone 的源：若预检没先行，会先撞 clone 失败而不是路径错误
	if err := s.SaveTplSource(TplSource{Name: "core", RepoUrl: ws.Join("no-such-repo")}); err != nil {
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
