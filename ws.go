package main

// ws.go —— WebSocket 握手探测（ws/wss，仅探测握手基本信息，不保持长连接）。

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// wsSelfCheckURL 为 all 命令默认自检所用的公开 WebSocket 端点。
const wsSelfCheckURL = "wss://ws.postman-echo.com/raw"

// isWSURL 判断 URL 是否为 ws/wss 协议。
func isWSURL(u string) bool {
	lower := strings.ToLower(u)
	return strings.HasPrefix(lower, "ws://") || strings.HasPrefix(lower, "wss://")
}

// runWebSocket 执行 WebSocket 握手探测。
// 仅完成 HTTP Upgrade 握手、读取 101 响应即结束，不建立长连接。
// 参数 rawURL 支持 ws:// 与 wss://，内部转换为 http/https 发送握手请求。
func runWebSocket(w io.Writer, rawURL string, timeout time.Duration) {
	section(w, "6. WEBSOCKET 握手探测")
	line(w, "目标 URL: %s", rawURL)
	line(w, "超时时间: %s", timeout.String())

	// 将 ws/wss 转换为 http/https（WebSocket 握手本质是携带 Upgrade 头的 HTTP 请求）
	httpURL := rawURL
	lower := strings.ToLower(rawURL)
	switch {
	case strings.HasPrefix(lower, "ws://"):
		httpURL = "http://" + rawURL[len("ws://"):]
	case strings.HasPrefix(lower, "wss://"):
		httpURL = "https://" + rawURL[len("wss://"):]
	}

	// 生成随机 Sec-WebSocket-Key（16 字节 base64），服务端据此计算 Sec-WebSocket-Accept
	keyBytes := make([]byte, 16)
	if _, err := rand.Read(keyBytes); err != nil {
		line(w, "[错误] 生成握手 Key 失败: %v", err)
		return
	}
	key := base64.StdEncoding.EncodeToString(keyBytes)

	// 构造 WebSocket 握手请求（HTTP Upgrade）
	req, err := http.NewRequest(http.MethodGet, httpURL, nil)
	if err != nil {
		line(w, "[错误] 无效的 URL: %v", err)
		return
	}
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", key)

	client := &http.Client{Timeout: timeout}
	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)
	if err != nil {
		line(w, "握手结果: 失败")
		line(w, "耗时: %s", elapsed.String())
		line(w, "错误信息: %v", err)
		fmt.Printf("[ws] %s 握手失败\n", rawURL)
		return
	}
	defer resp.Body.Close()

	// 输出握手基本信息：状态码、关键响应头、耗时
	line(w, "握手结果: %s", resp.Status)
	line(w, "耗时: %s", elapsed.String())
	line(w, "Upgrade: %s", resp.Header.Get("Upgrade"))
	line(w, "Connection: %s", resp.Header.Get("Connection"))
	line(w, "Sec-WebSocket-Accept: %s", resp.Header.Get("Sec-WebSocket-Accept"))

	if resp.StatusCode == http.StatusSwitchingProtocols {
		line(w, "连接结论: WebSocket 握手成功（101 Switching Protocols），服务可用")
	} else {
		line(w, "连接结论: 握手未成功（非 101），请检查服务端是否支持 WebSocket")
	}

	fmt.Printf("[ws] %s 握手探测完成\n", rawURL)
}
