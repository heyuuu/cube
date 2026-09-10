package server

import (
	"bytes"
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// uiFS 前端构建产物（make build-ui 把 web/dist 的内容拷到这里，go:embed 嵌入）。
// 目录常驻一个提交进仓库的空 .keep（gitignore 例外）——纯后端开发时 ui 无产物也能编译；
//
//go:embed all:ui
var uiFS embed.FS

type staticHandler struct{}

func newStaticHandler() *staticHandler {
	return &staticHandler{}
}

// Register 挂载前端静态资源（Raw 路由，不进 OpenAPI），页面请求按回退链线性解析：
//   - 路径在 ui 产物中存在 → 原样返回（含 /assets/* 构建产物与 favicon.svg 等根级文件）
//   - 不存在 → index.html（history 路由 fallback，支持 /projects 直达/刷新）
//   - index.html 未嵌入（纯后端模式）→ 404
//   - /api/*、/docs、/openapi.json 的未命中**不走 fallback**，按 404 处理——
//     否则 API 打错路径会拿到 HTML 200，错误被吞成莫名的解析失败
//
// pattern 用 "GET ..." 前缀限定方法，非 GET/HEAD 请求由 mux 直接回 405。
func (h *staticHandler) Register(r *Routes) {
	rootFS, err := fs.Sub(uiFS, "ui")
	if err != nil {
		slog.Error("无法进入 ui 子目录", "err", err)
		return
	}

	// index.html 启动时读一次即可（embed 内容恒定）；读不到即纯后端模式，页面路径按 404 处理
	indexFile, _ := fs.ReadFile(rootFS, "index.html")
	if indexFile == nil {
		slog.Info("web/ui 未构建，前端未嵌入，仅提供 API（需要前端执行 make build-ui 后重启）")
	}

	r.Raw("GET /", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if isNonFallbackPath(req.URL.Path) {
			http.NotFound(w, req)
			return
		}
		if name := strings.TrimPrefix(req.URL.Path, "/"); name != "" {
			if data, err := fs.ReadFile(rootFS, name); err == nil {
				http.ServeContent(w, req, name, time.Time{}, bytes.NewReader(data))
				return
			}
		}
		if indexFile == nil {
			http.NotFound(w, req)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(indexFile)
	}))
}

// isNonFallbackPath 判定不参与 SPA fallback 的路径：API 与文档端点（未命中应 404 而非回退页面）
func isNonFallbackPath(p string) bool {
	return strings.HasPrefix(p, "/api/") ||
		p == "/docs" || strings.HasPrefix(p, "/docs/") ||
		p == "/openapi.json"
}
