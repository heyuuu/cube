package server

// server 探活/关停 helper 集成测试：用 httptest 起 mock cube server 覆盖
// serverStatus 三态与 serverStop 流程（shutdown 后确认下线 / 被新实例接管）。
// token 鉴权走真实 Gen/Verify 配对。

import (
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cube/internal/handlers"
	"cube/internal/server"
)

// startMockCubeServer 起一个 httptest server 模拟 cube 的 status 端点。
// status==200 且 app=="cube" 时被 serverStatus 判为在跑。
func startMockCubeServer(t *testing.T, version string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/system/status", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.MarshalWrite(w, server.Envelope[handlers.StatusResponse]{
			Ok:   true,
			Data: &handlers.StatusResponse{App: "cube", Version: version, Instance: "mock-inst"},
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestServerStatus_Running(t *testing.T) {
	srv := startMockCubeServer(t, "v9.9.9")
	st := serverStatus(srv.URL)
	if !st.Running {
		t.Error("cube server 在跑应判 Running=true")
	}
	if st.Version != "v9.9.9" {
		t.Errorf("version 应为 v9.9.9，got %s", st.Version)
	}
	if st.Instance != "mock-inst" {
		t.Errorf("instance 应为 mock-inst，got %s", st.Instance)
	}
}

func TestServerStatus_NotRunning(t *testing.T) {
	// 端口 1 几乎肯定没监听
	if st := serverStatus("http://127.0.0.1:1"); st.Running {
		t.Error("连不上应判 Running=false")
	}
}

func TestServerStatus_NotCube(t *testing.T) {
	// 模拟别的服务（app != "cube"）
	mux := http.NewServeMux()
	mux.HandleFunc("/api/system/status", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"data":{"app":"not-cube","version":"x"}}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	if st := serverStatus(srv.URL); st.Running {
		t.Error("app!=cube 应判 Running=false（端口上的不是 cube）")
	}
}

func TestServerStop_NotRunning(t *testing.T) {
	stopped, _, err := serverStop("http://127.0.0.1:1")
	if err != nil {
		t.Errorf("停一个没在跑的 server 不应报错: %v", err)
	}
	if stopped {
		t.Error("没在跑应返回 stopped=false")
	}
}

// TestServerStop_ShutdownAndConfirm 验证 serverStop 能触发 shutdown 端点并确认旧实例下线。
// 用一个可控的 mock server：收到 shutdown 后停止响应 status（端口空出，无新实例接管）。
func TestServerStop_ShutdownAndConfirm(t *testing.T) {
	alive := true
	mux := http.NewServeMux()
	mux.HandleFunc("/api/system/status", func(w http.ResponseWriter, r *http.Request) {
		if !alive {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true,"data":{"app":"cube","version":"v1","instance":"old-inst"}}`))
	})
	mux.HandleFunc("/api/system/shutdown", func(w http.ResponseWriter, r *http.Request) {
		// shutdown 端点是 POST 注册的，用 GET 会拿到 405——钉死请求 method 防回归
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Token string `json:"token"`
		}
		_ = json.UnmarshalRead(r.Body, &req)
		if err := handlers.VerifyShutdownToken(req.Token, "old-inst", time.Now()); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true,"message":"","data":"shutting down"}`))
		alive = false // 模拟 shutdown 生效
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	stopped, replaced, err := serverStop(srv.URL)
	if err != nil {
		t.Fatalf("serverStop 应成功: %v", err)
	}
	if !stopped {
		t.Error("应返回 stopped=true")
	}
	if replaced {
		t.Error("端口空出无接管，应返回 replaced=false")
	}
	if alive {
		t.Error("mock server 应已被 shutdown（alive 应为 false）")
	}
}

// TestServerStop_ReplacedByNewInstance 模拟 launchctl 保活：旧实例 shutdown 后新实例立刻占回端口。
// serverStop 应按 instance 换人识别旧实例已下线，返回 stopped=true, replaced=true。
func TestServerStop_ReplacedByNewInstance(t *testing.T) {
	instance := "old-inst"
	mux := http.NewServeMux()
	mux.HandleFunc("/api/system/status", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, `{"ok":true,"data":{"app":"cube","version":"v1","instance":%q}}`, instance)
	})
	mux.HandleFunc("/api/system/shutdown", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Token string `json:"token"`
		}
		_ = json.UnmarshalRead(r.Body, &req)
		if err := handlers.VerifyShutdownToken(req.Token, instance, time.Now()); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true,"message":"","data":"shutting down"}`))
		instance = "new-inst" // 模拟新进程接管端口
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	stopped, replaced, err := serverStop(srv.URL)
	if err != nil {
		t.Fatalf("serverStop 应成功: %v", err)
	}
	if !stopped {
		t.Error("旧实例已 shutdown，应返回 stopped=true")
	}
	if !replaced {
		t.Error("端口被新实例接管，应返回 replaced=true")
	}
}
