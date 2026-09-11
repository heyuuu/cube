package server

// Client 集成测试：用 httptest 起 mock cube server 覆盖 Status 三态与 Stop 流程
// （shutdown 后确认下线 / 被新实例接管）。token 鉴权走真实 Gen/Verify 配对。

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// startMockCubeServer 起一个 httptest server 模拟 cube 的 status 端点。
// status==200 且 app=="cube" 时被 Client.Status 判为在跑。
func startMockCubeServer(t *testing.T, version string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/system/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(Envelope[StatusResponse]{
			Ok:   true,
			Data: &StatusResponse{App: "cube", Version: version, Instance: "mock-inst"},
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestClientStatus_Running(t *testing.T) {
	srv := startMockCubeServer(t, "v9.9.9")
	st := NewClient(srv.URL).Status()
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

func TestClientStatus_NotRunning(t *testing.T) {
	// 端口 1 几乎肯定没监听
	if st := NewClient("http://127.0.0.1:1").Status(); st.Running {
		t.Error("连不上应判 Running=false")
	}
}

func TestClientStatus_NotCube(t *testing.T) {
	// 模拟别的服务（app != "cube"）
	mux := http.NewServeMux()
	mux.HandleFunc("/api/system/status", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"data":{"app":"not-cube","version":"x"}}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	if st := NewClient(srv.URL).Status(); st.Running {
		t.Error("app!=cube 应判 Running=false（端口上的不是 cube）")
	}
}

func TestClientStop_NotRunning(t *testing.T) {
	stopped, _, err := NewClient("http://127.0.0.1:1").Stop()
	if err != nil {
		t.Errorf("停一个没在跑的 server 不应报错: %v", err)
	}
	if stopped {
		t.Error("没在跑应返回 stopped=false")
	}
}

// TestClientStop_ShutdownAndConfirm 验证 Stop 能触发 shutdown 端点并确认旧实例下线。
// 用一个可控的 mock server：收到 shutdown 后停止响应 status（端口空出，无新实例接管）。
func TestClientStop_ShutdownAndConfirm(t *testing.T) {
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
		_ = json.NewDecoder(r.Body).Decode(&req)
		if err := VerifyShutdownToken(req.Token, "old-inst", time.Now()); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true,"message":"","data":"shutting down"}`))
		alive = false // 模拟 shutdown 生效
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	stopped, replaced, err := NewClient(srv.URL).Stop()
	if err != nil {
		t.Fatalf("Stop 应成功: %v", err)
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

// TestClientStop_ReplacedByNewInstance 模拟 launchctl 保活：旧实例 shutdown 后新实例立刻占回端口。
// Stop 应按 instance 换人识别旧实例已下线，返回 stopped=true, replaced=true。
func TestClientStop_ReplacedByNewInstance(t *testing.T) {
	instance := "old-inst"
	mux := http.NewServeMux()
	mux.HandleFunc("/api/system/status", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf(`{"ok":true,"data":{"app":"cube","version":"v1","instance":%q}}`, instance)))
	})
	mux.HandleFunc("/api/system/shutdown", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Token string `json:"token"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if err := VerifyShutdownToken(req.Token, instance, time.Now()); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true,"message":"","data":"shutting down"}`))
		instance = "new-inst" // 模拟新进程接管端口
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	stopped, replaced, err := NewClient(srv.URL).Stop()
	if err != nil {
		t.Fatalf("Stop 应成功: %v", err)
	}
	if !stopped {
		t.Error("旧实例已 shutdown，应返回 stopped=true")
	}
	if !replaced {
		t.Error("端口被新实例接管，应返回 replaced=true")
	}
}
