// domain 子命令：生成业务 domain 脚手架——domain 包三件套 + handler + boot.go 注入。

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// app/boot.go 的注入锚点（与 boot.go 内标记保持一致，改一侧须同步另一侧）
const (
	serviceMarker = "/* <<service-factory inject point>> */"
	handlerMarker = "/* <<handler-factory inject point>> */"
	bootFile      = "internal/app/boot.go"
)

// genTarget 单个生成目标。
type genTarget struct {
	Path string // 输出路径，同一套模板变量（如 {{.domain}}）
	Tpl  string // tpl/ 下模板文件名
}

// 生成目标清单：增删生成文件只改这张表，配套模板放 tpl/ 目录。
var genTargets = []genTarget{
	{Path: "internal/domain/{{.domain}}/{{.domain}}.go", Tpl: "domain.gotmpl"},
	{Path: "internal/domain/{{.domain}}/doc.go", Tpl: "domain_doc.gotmpl"},
	{Path: "internal/domain/{{.domain}}/service.go", Tpl: "domain_service.gotmpl"},
	{Path: "internal/handlers/{{.domain}}_handler.go", Tpl: "handler.gotmpl"},
}

// genDomain 生成 domain 脚手架并注入 boot.go。
func genDomain(module, domain string) error {
	exported, err := checkDomain(domain)
	if err != nil {
		return err
	}

	// boot.go 先于建文件检查：执行目录不对时避免留下半成品
	if _, err := os.Stat(bootFile); err != nil {
		return fmt.Errorf("找不到 %s（请在工具目录下执行）: %w", bootFile, err)
	}

	data := map[string]string{
		"module":   module,
		"domain":   domain,
		"exported": exported,
	}

	// 先展开全部目标路径并逐一检查，全部通过后才动文件系统
	targets := make([]genTarget, 0, len(genTargets))
	for _, g := range genTargets {
		path, err := expandPath(g.Path, data)
		if err != nil {
			return fmt.Errorf("展开目标路径 %q 失败: %w", g.Path, err)
		}
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("目标文件已存在，拒绝覆盖: %s", path)
		}
		targets = append(targets, genTarget{Path: path, Tpl: g.Tpl})
	}
	for _, t := range targets {
		if err := os.MkdirAll(filepath.Dir(t.Path), 0o755); err != nil {
			return err
		}
		src, err := renderTemplate(t.Tpl, data)
		if err != nil {
			return err
		}
		if err := writeFormatted(t.Path, src); err != nil {
			return err
		}
		fmt.Println("已创建", t.Path)
	}

	if err := injectBoot(bootFile, module, domain, exported); err != nil {
		return err
	}
	fmt.Println("完成。模板为 mock 骨架，记得按业务调整 service/types/handler 内容。")
	return nil
}

// injectBoot 把新 domain 的 service 工厂与 handler 工厂注入 boot.go 的两个锚点处。
func injectBoot(path, module, domain, exported string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取 %s 失败（请在工具目录下执行）: %w", path, err)
	}
	text := string(src)

	if strings.Contains(text, domain+".NewService,") {
		fmt.Println("boot.go 已注入", domain, "，跳过注入步骤")
		return nil
	}
	for _, anchor := range []string{"import (", serviceMarker, handlerMarker} {
		if !strings.Contains(text, anchor) {
			return fmt.Errorf("boot.go 缺少锚点 %q，注入中止（检查标记是否被改动）", anchor)
		}
	}

	text = strings.Replace(text, "import (\n",
		"import (\n\t\""+module+"/internal/domain/"+domain+"\"\n", 1)
	text = strings.Replace(text, serviceMarker,
		fmt.Sprintf("\t\t%s.NewService,\n\n", domain)+serviceMarker, 1)
	text = strings.Replace(text, handlerMarker,
		fmt.Sprintf("\t\thandlers.New%sHandler,\n\n", exported)+handlerMarker, 1)

	if err := writeFormatted(path, text); err != nil {
		return err
	}
	fmt.Printf("已注入 boot.go: %s.NewService、handlers.New%sHandler\n", domain, exported)
	return nil
}
