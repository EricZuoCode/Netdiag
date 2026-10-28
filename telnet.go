package main

// telnet.go —— TCP 端口连通性探测（Go 原生拨号 + nc 交叉验证）。

import (
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// runTelnet 探测目标主机端口的 TCP 连通性。
// 使用 Go 标准库 net.DialTimeout 实现，不依赖系统 telnet/nc 命令，跨平台且无需特权。
func runTelnet(w io.Writer, host, port string, timeout time.Duration) {
	section(w, "3. TELNET 端口连通性探测")
	line(w, "目标地址: %s:%s", host, port)
	line(w, "超时时间: %s", timeout.String())

	target := net.JoinHostPort(host, port)
	start := time.Now()

	conn, err := net.DialTimeout("tcp", target, timeout)
	elapsed := time.Since(start)
	if err != nil {
		line(w, "连接结果: 失败")
		line(w, "耗时: %s", elapsed.String())
		line(w, "错误信息: %v", err)
		fmt.Printf("[telnet] %s 连接失败\n", target)
		return
	}
	defer conn.Close()

	line(w, "连接结果: 成功")
	line(w, "耗时: %s", elapsed.String())
	line(w, "本地地址: %s", conn.LocalAddr().String())
	line(w, "远端地址: %s", conn.RemoteAddr().String())
	fmt.Printf("[telnet] %s 连接成功\n", target)
}

// runTelnetWithFallback 先尝试 Go 原生 TCP 探测，再调用系统 nc 作为交叉验证。
// 系统 nc 不可用时自动跳过，不影响主流程结果。
func runTelnetWithFallback(w io.Writer, host, port string, timeout time.Duration) {
	runTelnet(w, host, port, timeout)

	// 交叉验证：nc -vz -w 超时秒 主机 端口
	if nc, err := lookPath("nc"); err == nil {
		line(w, "")
		line(w, "---- nc 交叉验证 ----")
		line(w, "执行命令: nc -vz -w %d %s %s", int(timeout.Seconds()), host, port)
		out, err := runCmd(nc, "-vz", "-w", fmt.Sprintf("%d", int(timeout.Seconds())), host, port)
		if err != nil {
			line(w, "[提示] nc 返回非零退出码: %v", err)
		}
		raw(w, strings.TrimRight(out, "\n"))
	}
}
