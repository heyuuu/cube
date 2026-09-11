package web

import (
	"embed"
	"io/fs"
	"log/slog"
)

// uiFS 前端构建产物（make build-ui 把 web/dist 的内容拷到这里，go:embed 嵌入）。
// 目录常驻一个提交进仓库的空 .keep（gitignore 例外）——纯后端开发时 ui 无产物也能编译；
//
//go:embed all:ui
var uiFS embed.FS

func StaticFS() fs.FS {
	rootFS, err := fs.Sub(uiFS, "ui")
	if err != nil {
		slog.Error("无法进入 ui 子目录", "err", err)
		return nil
	}

	return rootFS
}
