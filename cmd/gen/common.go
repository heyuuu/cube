package main

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"os"
	"regexp"
	"strings"
	"text/template"
	"unicode"
)

// resolveModule 从当前目录 go.mod 读 module 声明。
func resolveModule() (string, error) {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return "", fmt.Errorf("读取 go.mod 失败（请在工具目录下执行）: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if after, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			return strings.TrimSpace(after), nil
		}
	}
	return "", fmt.Errorf("go.mod 里找不到 module 声明")
}

// writeFormatted 校验内容是合法 Go 源码（gofmt 归一化，含补尾换行）后写出。
func writeFormatted(path, src string) error {
	formatted, err := format.Source([]byte(src))
	if err != nil {
		return fmt.Errorf("生成内容不是合法 Go 源码（%s）: %w", path, err)
	}
	return os.WriteFile(path, formatted, 0o644)
}

/* naming */

var (
	domainPattern   = regexp.MustCompile(`^[a-z][a-z0-9]*$`)
	baseNamePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z][a-z0-9]*)*$`)
)

// checkDomain 校验 domain 名（小写单词，作包名与 URL 段），返回导出前缀。
func checkDomain(domain string) (string, error) {
	if !domainPattern.MatchString(domain) {
		return "", fmt.Errorf("domain 只允许小写字母/数字且以字母开头: %q", domain)
	}
	return exportName(domain), nil
}

// checkBaseName 校验 api base-name（kebab-case，作 URL 段），返回导出前缀：run-diff → RunDiff。
func checkBaseName(base string) (string, error) {
	if !baseNamePattern.MatchString(base) {
		return "", fmt.Errorf("base-name 只允许小写字母/数字与中划线（kebab-case，如 run-diff）: %q", base)
	}
	return exportName(base), nil
}

// checkMethod 校验 HTTP method 并返回路由注册方法名（Routes 层只提供 Get/Post）。
func checkMethod(method string) (string, error) {
	switch strings.ToUpper(method) {
	case "GET":
		return "Get", nil
	case "POST":
		return "Post", nil
	default:
		return "", fmt.Errorf("method 只支持 GET/POST: %q", method)
	}
}

// exportName 把小写/kebab-case 名转导出名：compare → Compare、run-diff → RunDiff。
func exportName(s string) string {
	var b strings.Builder
	upper := true
	for _, r := range s {
		switch {
		case r == '-':
			upper = true
		case upper:
			b.WriteRune(unicode.ToUpper(r))
			upper = false
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

/* template */

//go:embed tpl
var tplFS embed.FS

// renderTemplate 渲染 tpl/ 下指定模板为源码片段（去首尾空白），可用键由调用方的 data 决定。
func renderTemplate(name string, data map[string]string) (string, error) {
	tpl, err := template.ParseFS(tplFS, "tpl/"+name)
	if err != nil {
		return "", fmt.Errorf("解析模板 tpl/%s 失败: %w", name, err)
	}
	tpl.Option("missingkey=error") // 模板里打错变量名直接报错，不静默产空
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("渲染模板 tpl/%s 失败: %w", name, err)
	}
	return strings.TrimSpace(buf.String()), nil
}

// expandPath 用同一套模板变量展开目标路径模板。
func expandPath(pattern string, data map[string]string) (string, error) {
	tpl, err := template.New("path").Parse(pattern)
	if err != nil {
		return "", err
	}
	tpl.Option("missingkey=error")
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
