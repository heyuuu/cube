package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// GenerateToken 生成鉴权 token 值：时间戳 + HMAC 签名。
func GenerateToken(instanceID string, now time.Time) string {
	payload := strconv.FormatInt(now.Unix(), 10)
	sig := hmacHex([]byte(instanceID), []byte(payload))
	return payload + "." + sig
}

// tokenWindow token 请求的时间戳有效窗口（秒）。
// 本机调用毫秒级往返，5 秒窗口够宽裕又能防重放（抓包过 5 秒就作废）。
const tokenWindow = 5 * time.Second

// VerifyToken 校验 shutdown 鉴权 token 值。
//
// 双重校验：
//  1. HMAC 比对（证明调用方持有 token）
//  2. 时间戳在窗口内（防重放）
//
// 任一失败返回 error。供 system/shutdown 端点校验（GenerateToken 在 CLI 停服侧生成）。
func VerifyToken(token string, instanceID string, now time.Time) error {
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
	if delta > tokenWindow {
		return fmt.Errorf("token 已过期（窗口 %s）", tokenWindow)
	}

	return nil
}

// hmacHex 计算 HMAC-SHA256 的十六进制编码。
func hmacHex(key, msg []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write(msg)
	return hex.EncodeToString(mac.Sum(nil))
}
