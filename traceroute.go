package main

// traceroute.go —— 路由路径追踪（traceroute / tracert）。

import (
	"fmt"
	"io"
	"runtime"
	"strings"
)

// runTraceroute 执行路由追踪，输出源到目标的每一跳路径信息。
func runTraceroute(w io.Writer, host string) {
	section(w, "1.1 TRACEROUTE 路由追踪")
	line(w, "目标主机: %s", host)

	bin, args := buildTracerouteCommand(host)
	line(w, "执行命令: %s %s", bin, strings.Join(args, " "))

	out, err := runCmd(bin, args...)
	if err != nil {
		// traceroute 遇目标不可达或部分跳超时可能返回非零退出码，输出仍具排查价值
		line(w, "[提示] 命令返回非零退出码: %v", err)
	}
	raw(w, out)

	fmt.Printf("[traceroute] 目标 %s 路由追踪完成\n", host)
}

// buildTracerouteCommand 根据操作系统构造路由追踪命令：Windows 用 tracert，其余用 traceroute。
func buildTracerouteCommand(host string) (string, []string) {
	switch runtime.GOOS {
	case "windows":
		return "tracert", []string{host}
	default:
		return "traceroute", []string{host}
	}
}
