// Package web 嵌入前端构建产物 golang 挂载为页面。
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var embedded embed.FS

// FS 返回前端产物根（dist/ 下内容提升到根，StaticHandler 直接按相对路径读）。
func FS() fs.FS {
	sub, err := fs.Sub(embedded, "dist")
	if err != nil {
		panic(err) // embed 路径是编译期常量，出错即代码错误
	}
	return sub
}
