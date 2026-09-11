package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"

	"cube/core/version"
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

func NewSystemHandler() *SystemHandler {
	return &SystemHandler{
		instance: uuid.NewString(),
	}
}

func (h *SystemHandler) Register(r *Routes) {
	r.Get("/api/system/status", "服务身份探活", JsonHandler(h.status))
	r.Post("/api/system/shutdown", "触发服务平滑关闭", JsonHandler(h.shutdown), WithoutOpenAPI())
}

// StatusResponse Status 返回体
type StatusResponse struct {
	App      string `json:"app"`      // 固定 "cube"，供探活方验证身份
	Version  string `json:"version"`  // cube 版本号
	Instance string `json:"instance"` // 进程启动时随机生成的实例标识，重启后变化
}

func (h *SystemHandler) status(_ struct{}) (StatusResponse, error) {
	return StatusResponse{
		App:      version.AppName,
		Version:  version.Version(),
		Instance: h.instance,
	}, nil
}

// shutdown 校验 token 后给本进程发 SIGTERM 触发 graceful shutdown。
// 信号由 Start 的 signal.Notify 接收，走与用户 Ctrl+C 完全相同的关闭路径。
// 鉴权失败以 envelope ok:false 表达（JsonHandler 惯例），调用方解析 envelope 判定。
func (h *SystemHandler) shutdown(input struct {
	Body struct {
		Token string `json:"token" required:"true"`
	}
}) (string, error) {
	if err := VerifyShutdownToken(input.Body.Token, h.instance, time.Now()); err != nil {
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

// GenShutdownToken 生成 shutdown 鉴权 token 值：时间戳 + HMAC 签名。
func GenShutdownToken(instanceID string, now time.Time) string {
	payload := strconv.FormatInt(now.Unix(), 10)
	sig := hmacHex([]byte(instanceID), []byte(payload))
	return payload + "." + sig
}

// shutdownTokenWindow shutdown 请求的时间戳有效窗口（秒）。
// 本机调用毫秒级往返，5 秒窗口够宽裕又能防重放（抓包过 5 秒就作废）。
const shutdownTokenWindow = 5 * time.Second

// VerifyShutdownToken 校验 shutdown 鉴权 token 值。
//
// 双重校验：
//  1. HMAC 比对（证明调用方持有 token）
//  2. 时间戳在窗口内（防重放）
//
// 任一失败返回 error。供 server 端 shutdown handler 调用。
func VerifyShutdownToken(token string, instanceID string, now time.Time) error {
	payload, sig, ok := strings.Cut(token, ".")
	if !ok {
		return fmt.Errorf("token 格式非法")
	}
	wantSig := hmacHex([]byte(instanceID), []byte(payload))
	if !hmac.Equal([]byte(sig), []byte(wantSig)) {
		return fmt.Errorf("token 签名不匹配")
	}
	ts, err := strconv.ParseInt(payload, 10, 64)
	if err != nil {
		return fmt.Errorf("token 时间戳非法: %w", err)
	}
	delta := now.Sub(time.Unix(ts, 0))
	if delta < 0 {
		delta = -delta
	}
	if delta > shutdownTokenWindow {
		return fmt.Errorf("token 已过期（窗口 %s）", shutdownTokenWindow)
	}
	return nil
}

// hmacHex 计算 HMAC-SHA256 的十六进制编码。
func hmacHex(key, msg []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write(msg)
	return hex.EncodeToString(mac.Sum(nil))
}
