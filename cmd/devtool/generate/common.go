package generate

import (
	"fmt"
	"go/format"
	"regexp"
	"strings"

	"github.com/heyuuu/x/xstd/xos"
	"github.com/heyuuu/x/xstd/xstrings"
)

// writeFormatted 校验内容是合法 Go 源码（gofmt 归一化，含补尾换行）后写出。
func writeFormatted(path, src string) error {
	formatted, err := format.Source([]byte(src))
	if err != nil {
		return fmt.Errorf("生成内容不是合法 Go 源码（%s）: %w", path, err)
	}
	return xos.WriteFile(path, formatted, 0o644)
}

/* naming */

var (
	domainPattern   = regexp.MustCompile(`^[a-z][a-z0-9]*$`)
	baseNamePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z][a-z0-9]*)*$`)
)

// checkDomain 校验 domain 名（小写单词，作包名与 URL 段），返回导出前缀。
func checkDomain(domain string) error {
	if !domainPattern.MatchString(domain) {
		return fmt.Errorf("domain 只允许小写字母/数字且以字母开头: %q", domain)
	}
	return nil
}

// checkCommandName 校验 command 名（kebab-case，作 cobra Use，文件名经 fileName 转 snake_case），返回导出前缀：opener-search → OpenerSearch。
func checkCommandName(name string) (string, error) {
	if !baseNamePattern.MatchString(name) {
		return "", fmt.Errorf("command 名只允许小写字母/数字与中划线（kebab-case，首尾与连续中划线不允许，如 opener-search）: %q", name)
	}
	return xstrings.PascalCase(name), nil
}

// checkCommandGroup 校验 command 组名（小写单词，作包名与目录名，中划线不合法）。
func checkCommandGroup(group string) error {
	if !domainPattern.MatchString(group) {
		return fmt.Errorf("command 组名只允许小写字母/数字且以字母开头（作包名，不支持中划线）: %q", group)
	}
	return nil
}

// checkMethod 校验 HTTP method 并返回路由注册方法名（Routes 层只提供 Get/Post）。
func checkMethod(method string) error {
	switch strings.ToUpper(method) {
	case "GET":
	case "POST":
	default:
		return fmt.Errorf("method 只支持 GET/POST: %q", method)
	}
	return nil
}

// fileName 把 kebab-case 名转 Go 文件名惯用的 snake_case：opener-search → opener_search。
func fileName(s string) string {
	return strings.ReplaceAll(s, "-", "_")
}
