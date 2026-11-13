package server

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/heyuuu/x/xstd/xmaps"

	"cube/internal/handlers"
	"cube/internal/server"
	"cube/internal/version"
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

// 构造并发出 json 请求，并按 cube 标准信封格式解析返回结果
func doServerRequest[T any](method string, url string, body any) (*T, error) {
	// 构造 body reader
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("构造 request body 失败: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	// 构造 request
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("构造 request 失败: %w", err)
	}

	// 请求
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送 request 请求失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("请求失败: status-code=%d", resp.StatusCode)
	}

	// 解析 json 结构
	var out server.Envelope[T]
	if err := json.UnmarshalRead(resp.Body, &out); err != nil {
		return nil, fmt.Errorf("解析 response 结构失败: %w", err)
	}
	if !out.Ok {
		return nil, fmt.Errorf("请求失败：%s", out.Message)
	}

	// 返回结果
	return out.Data, nil
}

// StatusInfo server 状态查询结果。
type StatusInfo struct {
	Running  bool   // 是否在跑（status 返回 app=="cube"）
	Version  string // status 返回的版本号（Running=false 时为空）
	Instance string // status 返回的实例标识（Running=false 或旧版 server 无此字段时为空）
}

// 获取服务端状态
func serverStatus(baseURL string) StatusInfo {
	api := buildURL(baseURL, "api/system/status", nil)
	resp, err := doServerRequest[handlers.StatusResponse](http.MethodGet, api, nil)
	if err != nil {
		return StatusInfo{}
	}
	if resp.App != version.AppName {
		return StatusInfo{} // 端口上的服务不是本服务
	}
	return StatusInfo{
		Running:  true,
		Version:  resp.Version,
		Instance: resp.Instance,
	}
}

// 触发 server graceful shutdown
func serverStop(baseURL string) (stopped bool, replaced bool, err error) {
	// 先探活，没在跑直接返回；同时记下当前实例标识
	st := serverStatus(baseURL)
	if !st.Running {
		return false, false, nil
	}
	instanceID := st.Instance

	// 触发 shutdown 请求
	api := buildURL(baseURL, "api/system/shutdown", nil)
	_, err = doServerRequest[string](http.MethodPost, api, map[string]any{
		"token": handlers.GenShutdownToken(instanceID, time.Now()),
	})
	if err != nil {
		return false, false, fmt.Errorf("shutdown 请求失败: %w", err)
	}

	// 轮询确认旧实例下线
	replaced, err = waitOldStop(baseURL, instanceID)
	if err != nil {
		return false, false, fmt.Errorf("shutdown 请求已发送但旧实例未下线: %w", err)
	}
	return true, replaced, nil
}

// 端口轮询参数。
const (
	probeInterval = 200 * time.Millisecond
	probeTimeout  = 10 * time.Second
)

func waitOldStop(baseURL string, oldInstance string) (replace bool, err error) {
	deadline := time.Now().Add(probeTimeout)
	for time.Now().Before(deadline) {
		st := serverStatus(baseURL)
		if !st.Running {
			return false, nil
		}
		if st.Instance != oldInstance {
			return true, nil
		}
		time.Sleep(probeInterval)
	}
	return false, errors.New("等待端口旧实例下线超时")
}
