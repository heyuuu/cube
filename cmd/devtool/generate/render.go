package generate

import (
	"bytes"
	"embed"
	"fmt"
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
