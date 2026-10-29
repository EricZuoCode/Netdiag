package main

// ping.go —— ping 连通性测试（跨平台命令适配 + 统计信息解析）。

import (
	"fmt"
	"io"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// pingStats 保存从 ping 输出中解析出的关键统计指标。
type pingStats struct {
	Transmitted int
	Received    int
	LossPercent float64
	MinRTT      string
	AvgRTT      string
	MaxRTT      string
}

// 以下正则用于解析 ping 输出中的统计信息，兼容英文与中文系统环境。
var (
	reLossEn = regexp.MustCompile(`(\d+(?:\.\d+)?)%\s*packet loss`)                      // 丢包率（英文）
	reLossZh = regexp.MustCompile(`(\d+(?:\.\d+)?)%\s*丢包`)                               // 丢包率（中文）
	reSent   = regexp.MustCompile(`(\d+)\s+packets transmitted`)                         // 已发送包数（英文）
	reSentZh = regexp.MustCompile(`(\d+)\s+个(?:数据包)?已发送`)                                // 已发送包数（中文）
	reRecv   = regexp.MustCompile(`(\d+)\s+(?:packets )?received`)                       // 已接收包数（英文）
	reRecvZh = regexp.MustCompile(`(\d+)\s+个(?:数据包)?已接收`)                                // 已接收包数（中文）
	reRTT    = regexp.MustCompile(`(?:round-trip|rtt).*?=\s*([\d.]+)/([\d.]+)/([\d.]+)`) // 往返时延 min/avg/max
)

// buildPingCommand 根据当前操作系统构造系统 ping 命令及参数。
// 超时参数 timeoutSec 单位为秒，内部按各平台要求换算为毫秒或秒。
func buildPingCommand(host string, count, timeoutSec int) (string, []string) {
	switch runtime.GOOS {
	case "windows":
		// Windows: ping -n 次数 -w 毫秒 主机
		return "ping", []string{"-n", strconv.Itoa(count), "-w", strconv.Itoa(timeoutSec * 1000), host}
	case "darwin":
		// macOS: ping -c 次数 -W 毫秒 主机
		return "ping", []string{"-c", strconv.Itoa(count), "-W", strconv.Itoa(timeoutSec * 1000), host}
	default:
		// Linux 等: ping -c 次数 -W 秒 主机
		return "ping", []string{"-c", strconv.Itoa(count), "-W", strconv.Itoa(timeoutSec), host}
	}
}

// firstMatch 返回正则 r 在 s 中第一个捕获组的内容，未匹配返回空字符串。
func firstMatch(r *regexp.Regexp, s string) string {
	m := r.FindStringSubmatch(s)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

// parseInt 将字符串解析为整数，解析失败返回 0。
func parseInt(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// parseFloat 将字符串解析为浮点数，解析失败返回 0。
func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// parsePingStats 从 ping 原始输出中提取统计信息，尽力而为，解析失败不报错。
func parsePingStats(out string) pingStats {
	st := pingStats{}

	if v := firstMatch(reLossEn, out); v != "" {
		st.LossPercent = parseFloat(v)
	} else if v := firstMatch(reLossZh, out); v != "" {
		st.LossPercent = parseFloat(v)
	}

	if v := firstMatch(reSent, out); v != "" {
		st.Transmitted = parseInt(v)
	} else if v := firstMatch(reSentZh, out); v != "" {
		st.Transmitted = parseInt(v)
	}

	if v := firstMatch(reRecv, out); v != "" {
		st.Received = parseInt(v)
	} else if v := firstMatch(reRecvZh, out); v != "" {
		st.Received = parseInt(v)
	}

	if m := reRTT.FindStringSubmatch(out); len(m) == 4 {
		st.MinRTT, st.AvgRTT, st.MaxRTT = m[1], m[2], m[3]
	}
	return st
}

// runPing 执行 ping 网络测试并写入结果。
func runPing(w io.Writer, host string, count, timeoutSec int) {
	section(w, "1. PING 连通性测试")
	line(w, "目标主机: %s", host)
	line(w, "发包数量: %d", count)

	bin, args := buildPingCommand(host, count, timeoutSec)
	line(w, "执行命令: %s %s", bin, strings.Join(args, " "))

	out, err := runCmd(bin, args...)
	if err != nil {
		// ping 丢包或目标不可达时会返回非零退出码，但输出仍有价值，故一并记录
		line(w, "[提示] 命令返回非零退出码: %v", err)
	}
	raw(w, out)

	st := parsePingStats(out)
	if st.Transmitted > 0 || st.LossPercent > 0 {
		line(w, "")
		line(w, "---- 统计摘要 ----")
		line(w, "已发送: %d 包", st.Transmitted)
		line(w, "已接收: %d 包", st.Received)
		line(w, "丢失占比: %.1f%%", st.LossPercent)
		if st.AvgRTT != "" {
			line(w, "往返时延 (min/avg/max): %s / %s / %s ms", st.MinRTT, st.AvgRTT, st.MaxRTT)
		}
		if st.LossPercent == 0 {
			line(w, "连通性结论: 正常（无丢包）")
		} else if st.LossPercent < 100 {
			line(w, "连通性结论: 部分丢包（存在网络波动）")
		} else {
			line(w, "连通性结论: 完全不可达（100% 丢包）")
		}
	} else {
		line(w, "")
		line(w, "[提示] 未能从输出中解析出统计信息，请参考上方原始输出。")
	}

	fmt.Printf("[ping] 目标 %s 测试完成\n", host)

	// ping 后自动携带 traceroute 路由追踪，验证到达路径
	runTraceroute(w, host)
}
