package web

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"

	"github.com/heyuuu/cube/version"
)

type H map[string]any

// HandleFunc 请求处理函数
type HandleFunc func(params any) (result any, err error)

// Handler 接口
type Handler interface {
	Register(api huma.API)
}

// Server 服务器，响应 api 请求
type Server struct {
	mux *http.ServeMux
	api huma.API
}

func NewServer(handlers ...Handler) *Server {
	mux := http.NewServeMux()

	cfg := huma.DefaultConfig("Cube API", version.Version)
	cfg.DocsRenderer = huma.DocsRendererScalar // 切换 /docs 页面风格为 Scalar 渲染器
	api := humago.New(mux, cfg)

	// 各 domain 注册自己的路由
	for _, handler := range handlers {
		handler.Register(api)
	}

	return &Server{mux: mux, api: api}
}

func (s *Server) API() huma.API { return s.api }

// Start 启动 server, 收到 SIGINT/SIGTERM 优雅关闭。
func (s *Server) Start(addr string) error {
	server := &http.Server{
		Addr:              addr,
		Handler:           s.mux,
		ReadHeaderTimeout: 5 * time.Second,
		//ReadTimeout:       10 * time.Second,
		//WriteTimeout:      30 * time.Second,
		IdleTimeout: 120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("server starting", "addr", addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	// 接收信号关闭 server
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return fmt.Errorf("server start failed: %w", err)
	case sig := <-sigCh:
		slog.Info("server shutting down", "signal", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(ctx)
	}
}
