package handlers

import (
	"fmt"
	"log/slog"
	"os"
	"syscall"
	"time"

	"github.com/google/uuid"

	"cube/internal/api"
	"cube/internal/version"
)

// SystemHandler 提供 /api/system/* 端点：服务自身的内部管理 API。
//
//	GET  /api/system/status    服务身份探活（无鉴权，只读）
//	POST /api/system/shutdown  触发服务平滑关闭（HMAC 时间戳鉴权，防 CSRF/重放）
//
// shutdown 是运维端点，刻意不进 OpenAPI 文档。
// SystemHandler 不持有 Server 引用——shutdown 通过给本进程发 SIGTERM 触发，
// 复用 Start 里已注册的信号监听路径。
type SystemHandler struct {
	// instance 进程级随机实例标识，随 status 暴露。
	// 服务被重启（如 launchctl 保活拉起新进程）后端口可能立刻被新实例占回，
	// 调用方（Client.Stop）靠它区分「重启前后的不同实例」，只看端口会误判。
	instance string
}

// NewSystemHandler 构造 SystemHandler（instance 每进程随机，识别新旧实例）。
func NewSystemHandler() *SystemHandler {
	return &SystemHandler{
		instance: uuid.NewString(),
	}
}

// Register 注册 system 端点（实现 api.Registerer）。
func (h *SystemHandler) Register(r *api.Routes) {
	r.Get("/api/system/status", "服务身份探活", api.JSONHandler(h.status))
	r.Post("/api/system/shutdown", "触发服务平滑关闭", api.JSONHandler(h.shutdown), api.WithoutOpenAPI())
}

/**
 * GET /api/system/status
 */

type SystemStatusOutput struct {
	App      string `json:"app"`      // 固定 "cube"，供探活方验证身份
	Version  string `json:"version"`  // cube 版本号
	Instance string `json:"instance"` // 进程启动时随机生成的实例标识，重启后变化
}

func (h *SystemHandler) status(_ struct{}) (*SystemStatusOutput, error) {
	return &SystemStatusOutput{
		App:      version.AppName,
		Version:  version.Version(),
		Instance: h.instance,
	}, nil
}

/**
 * POST /api/system/shutdown
 */

// shutdown 校验 token 后给本进程发 SIGTERM 触发 graceful shutdown。
// 信号由 Start 的 signal.Notify 接收，走与用户 Ctrl+C 完全相同的关闭路径。
// 鉴权失败以 envelope ok:false 表达（JSONHandler 惯例），调用方解析 envelope 判定。
func (h *SystemHandler) shutdown(input WithBody[struct {
	Token string `json:"token" required:"true"`
}]) (*string, error) {
	if err := api.VerifyToken(input.Body.Token, h.instance, time.Now()); err != nil {
		slog.Warn("shutdown 鉴权失败", "err", err)
		return nil, fmt.Errorf("鉴权失败: %w", err)
	}

	// 异步发信号不阻塞 handler 返回；goroutine 可能先于响应写出触发 SIGTERM，
	// 但 Start 的 graceful 关闭路径（http.Server.Shutdown）会等在途响应写完
	go func() {
		_ = syscall.Kill(os.Getpid(), syscall.SIGTERM)
	}()

	return new("shutting down"), nil
}
