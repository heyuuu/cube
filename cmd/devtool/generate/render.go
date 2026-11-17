package generate

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"strings"
	"text/template"
)

//go:embed tpl
var tplFS embed.FS

// renderTplFile 渲染 tpl/ 下指定模板文件，可用键由调用方的 data 决定。
func renderTplFile(tplFile string, data map[string]string) (string, error) {
	tpl, err := template.ParseFS(tplFS, "tpl/"+tplFile)
	if err != nil {
		return "", fmt.Errorf("解析模板 tpl/%s 失败: %w", tplFile, err)
	}

	tpl.Option("missingkey=error") // 模板里打错变量名直接报错，不静默产空
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("渲染模板 tpl/%s 失败: %w", tplFile, err)
	}

	return strings.TrimSpace(buf.String()), nil
}

// renderTplStr 渲染指定模板文本
func renderTplStr(tplText string, data map[string]string) (string, error) {
	tpl, err := template.New("path").Parse(tplText)
	if err != nil {
		return "", fmt.Errorf("解析模板文本失败: %w", err)
	}

	tpl.Option("missingkey=error")
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("渲染模板文本失败: %w", err)
	}

	return buf.String(), nil
}

func renderToFile(tplFile string, target string, data map[string]string) error {
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("目标文件已存在，拒绝覆盖: %s", target)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("读取目标文件 %s 失败: %w", target, err)
	}

	src, err := renderTplFile(tplFile, data)
	if err != nil {
		return err
	}

	if err := writeFormatted(target, src); err != nil {
		return err
	}
	fmt.Println("已创建", target)
	return nil
}
