package web

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"syscall"
	"time"

	"cube/version"
)

// SystemHandler 提供 /api/system/* 端点：服务自身的内部管理 API。
//
//	GET  /api/system/whoami   服务身份探活（无鉴权，只读）
//	POST /api/system/shutdown  触发服务平滑关闭（HMAC 时间戳鉴权，防 CSRF/重放）
//
// shutdown 是运维端点，刻意不进 OpenAPI 文档。
// SystemHandler 不持有 Server 引用——shutdown 通过给本进程发 SIGTERM 触发，
// 复用 Start 里已注册的信号监听路径。
type SystemHandler struct {
	// instance 进程级随机实例标识，随 whoami 暴露。
	// 服务被重启（如 launchctl 保活拉起新进程）后端口可能立刻被新实例占回，
	// 调用方（serve.Stop）靠它区分「重启前后的不同实例」，只看端口会误判。
	instance string
}

func newSystemHandler() *SystemHandler {
	return &SystemHandler{instance: newInstanceID()}
}

func (h *SystemHandler) Register(r *Routes) {
	r.Get("/api/system/whoami", "服务身份探活", JsonHandler(h.whoami))
	r.Post("/api/system/shutdown", "触发服务平滑关闭", JsonHandler(h.shutdown), WithoutOpenAPI())
}

// WhoamiResponse whoami 返回体。
type WhoamiResponse struct {
	App      string `json:"app"`      // 固定 "cube"，供探活方验证身份
	Version  string `json:"version"`  // cube 版本号
	Instance string `json:"instance"` // 进程启动时随机生成的实例标识，重启后变化
}

func (h *SystemHandler) whoami(_ struct{}) (WhoamiResponse, error) {
	return WhoamiResponse{App: version.AppName, Version: version.Version(), Instance: h.instance}, nil
}

// newInstanceID 生成实例标识（8 字节随机 hex）。
// crypto/rand 失败极罕见，退化为纳秒时间戳——只需保证进程间不同，不要求密码学强度。
func newInstanceID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("t%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// shutdown 校验 token 后给本进程发 SIGTERM 触发 graceful shutdown。
// 信号由 Start 的 signal.Notify 接收，走与用户 Ctrl+C 完全相同的关闭路径。
// 鉴权失败以 envelope ok:false 表达（JsonHandler 惯例），调用方解析 envelope 判定。
func (h *SystemHandler) shutdown(input struct {
	Body struct {
		Token string `json:"token" required:"true"`
	}
}) (string, error) {
	if err := VerifyShutdownToken(input.Body.Token, time.Now()); err != nil {
		slog.Warn("shutdown 鉴权失败", "err", err)
		return "", fmt.Errorf("鉴权失败: %w", err)
	}

	// 异步发信号不阻塞 handler 返回；goroutine 可能先于响应写出触发 SIGTERM，
	// 但 Start 走 server.Shutdown graceful 关闭，会等在途响应写完
	go func() {
		_ = syscall.Kill(os.Getpid(), syscall.SIGTERM)
	}()

	return "shutting down", nil
}
