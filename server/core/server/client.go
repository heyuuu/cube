package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"cube/core/version"
)

// 端口轮询参数。
const (
	probeInterval = 200 * time.Millisecond
	probeTimeout  = 10 * time.Second
)

type Client struct {
	baseUrl string
}

func NewClient(baseUrl string) *Client {
	if !strings.HasSuffix(baseUrl, "/") {
		baseUrl += "/"
	}
	return &Client{baseUrl: baseUrl}
}

// StatusInfo server 状态查询结果。
type StatusInfo struct {
	Running  bool   // 是否在跑（status 返回 app=="cube"）
	Version  string // status 返回的版本号（Running=false 时为空）
	Instance string // status 返回的实例标识（Running=false 或旧版 server 无此字段时为空）
}

// Status 获取服务端状态
func (c *Client) Status() StatusInfo {
	url := c.baseUrl + "api/system/status"
	resp, err := doJsonRequest[StatusResponse](url, nil)
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

// Stop 触发 server graceful shutdown
func (c *Client) Stop() (stopped bool, replaced bool, err error) {
	// 先探活，没在跑直接返回；同时记下当前实例标识
	st := c.Status()
	if !st.Running {
		return false, false, nil
	}
	instanceID := st.Instance

	// 触发 shutdown 请求
	_, err = doJsonRequest[string](c.baseUrl+"api/system/shutdown", map[string]any{
		"token": GenShutdownToken(instanceID, time.Now()),
	})
	if err != nil {
		return false, false, fmt.Errorf("shutdown 请求失败: %w", err)
	}

	// 轮询确认旧实例下线
	replaced, err = c.waitOldStop(instanceID)
	if err != nil {
		return false, false, fmt.Errorf("shutdown 请求已发送但旧实例未下线: %w", err)
	}
	return true, replaced, nil
}

func (c *Client) waitOldStop(oldInstance string) (replace bool, err error) {
	deadline := time.Now().Add(probeTimeout)
	for time.Now().Before(deadline) {
		st := c.Status()
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

// helpers

func doJsonRequest[T any](url string, body any) (*T, error) {
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
	req, err := http.NewRequest(http.MethodGet, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("构造 request 失败: %w", err)
	}

	// 请求
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送 request 请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("请求失败: status-code=%d", resp.StatusCode)
	}

	// 解析 json 结构
	var out Envelope[T]
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("解析 response 结构失败: %w", err)
	}
	if !out.Ok {
		return nil, fmt.Errorf("请求失败：%s", out.Message)
	}

	// 返回结果
	return out.Data, nil
}
