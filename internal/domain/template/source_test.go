package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/heyuuu/x/xstd/xos"

	"cube/internal/testfixture"
	"cube/pkg/git"
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

// TestOpenDirSource 目录源两形态加载：根模板（template.yaml 在源根，唯一模板名固定）
// 与模板库（templates/ 下含 meta 的子目录，过滤无 meta/.开头/非目录项）。
func TestOpenDirSource(t *testing.T) {
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

	src, err := openDirSource(sourcePath)
	if err != nil {
		t.Fatalf("openDirSource 报错: %v", err)
	}

	// 模板库形态：无根模板名，清单只含合法模板目录
	if src.RootTemplateName() != "" {
		t.Fatalf("模板库形态 RootTemplateName 应为空, got %q", src.RootTemplateName())
	}
	if names := src.Names(); len(names) != 2 || names[0] != "go-cli" || names[1] != "lib-a" {
		t.Fatalf("模板清单不符: %v", names)
	}
	if got := src.templates["go-cli"]; got != filepath.Join(sourcePath, "templates/go-cli") {
		t.Fatalf("go-cli 模板目录不符: %q", got)
	}

	// 根模板形态：源根即模板目录（值曾误存 meta 文件名，端到端必挂）
	rootPath := makeTestDir(t, map[string]string{
		MetaFileName: "version: 1\n",
		"main.go":    "package main",
	})
	rootSrc, err := openDirSource(rootPath)
	if err != nil {
		t.Fatalf("根模板形态加载报错: %v", err)
	}
	if name := rootSrc.RootTemplateName(); name != RootTplName {
		t.Fatalf("根模板名 = %q, want %q", name, RootTplName)
	}
	if got := rootSrc.templates[RootTplName]; got != rootPath {
		t.Fatalf("根模板目录应为源根: %q", got)
	}
	if _, err := rootSrc.Template(RootTplName); err != nil {
		t.Fatalf("根模板应可加载: %v", err)
	}

	// 不存在的路径 / 不合法结构
	if _, err := openDirSource(filepath.Join(t.TempDir(), "missing")); err == nil || !strings.Contains(err.Error(), "模板源路径不存在") {
		t.Fatalf("不存在的路径应报中文错误, got %v", err)
	}
	if _, err := openDirSource(makeTestDir(t, map[string]string{"readme.md": "无结构"})); err == nil || !strings.Contains(err.Error(), "不是合法模板来源") {
		t.Fatalf("期望“不是合法模板来源”错误, got %v", err)
	}
}

// TestOpenSource 源引用两形态解析：本地路径直读（无 cleanup）/ 注册名查注册表临时 clone
// （Cleanup 删临时目录）；未注册名中文报错。
func TestOpenSource(t *testing.T) {
	ws := testfixture.NewWorkspace(t)
	repo := makeTplRepo(t, ws)
	sources := []TplSource{{Name: "lib", RepoURL: repo}}

	// 本地绝对路径：直读原路径，Cleanup 空操作
	local, err := openSource(repo, sources)
	if err != nil || local.dir != repo {
		t.Fatalf("路径形态应直读原路径: dir=%s err=%v", local.dir, err)
	}
	local.Cleanup()
	if _, err := os.Stat(repo); err != nil {
		t.Fatalf("本地路径形态 Cleanup 不应删除原目录: %v", err)
	}

	// 不存在的本地路径
	if _, err := openSource(filepath.Join(ws.Dir, "missing"), sources); err == nil || !strings.Contains(err.Error(), "模板源路径不存在") {
		t.Fatalf("不存在的路径应报中文错误, got %v", err)
	}

	// 注册名：查注册表 → 临时 clone，Cleanup 后目录删除
	cloned, err := openSource("lib", sources)
	if err != nil {
		t.Fatalf("命名源解析失败: %v", err)
	}
	if cloned.dir == repo || !xos.IsDir(filepath.Join(cloned.dir, "templates", "hello")) {
		t.Fatalf("应 clone 到新目录且含模板: %s", cloned.dir)
	}
	cloned.Cleanup()
	if _, err := os.Stat(cloned.dir); !os.IsNotExist(err) {
		t.Fatalf("Cleanup 后临时目录应删除: %s, err=%v", cloned.dir, err)
	}

	// 未注册名：中文错误
	if _, err := openSource("work", sources); err == nil || !strings.Contains(err.Error(), "未找到指定模板源: work") {
		t.Fatalf("未注册源应报错, got %v", err)
	}
}
