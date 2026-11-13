package server

import (
	"fmt"
	"net/url"
	"os/exec"
	"strings"

	"github.com/heyuuu/x/xstd/xmaps"
)

type Q = map[string]string

// 构建 url 字符串
func buildURL(baseURL string, path string, query Q) string {
	var buf strings.Builder

	buf.WriteString(strings.TrimRight(baseURL, "/"))
	buf.WriteByte('/')
	buf.WriteString(strings.TrimLeft(path, "/"))

	if len(query) > 0 {
		for i, k := range xmaps.SortedKeys(query) {
			if i == 0 {
				buf.WriteByte('?')
			} else {
				buf.WriteByte('&')
			}
			buf.WriteString(url.QueryEscape(k))
			buf.WriteByte('=')
			buf.WriteString(url.QueryEscape(query[k]))
		}
	}
	return buf.String()
}

// openInBrowser 用系统 open 打开 URL。失败不阻断：打出 URL 让用户手动访问。
func openInBrowser(url string) {
	if err := exec.Command("open", url).Run(); err != nil {
		fmt.Printf("自动打开浏览器失败，请手动访问：%s\n", url)
	} else {
		fmt.Printf("%s\n", url)
	}
}
