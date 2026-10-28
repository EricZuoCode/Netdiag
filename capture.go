package main

// capture.go —— tcpdump 抓包（仅 Linux/macOS，全局串行执行）。

import (
	"fmt"
	"io"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// tcpdumpMu 保证抓包操作全局串行执行，不允许并发抓包。
var tcpdumpMu sync.Mutex

// runTcpdump 执行 tcpdump 抓包操作。
// 仅支持 Linux / macOS，Windows 需自行使用 Wireshark 等工具。
// 必须提供域名与端口以缩小抓包范围，避免抓包过大影响存储空间。
// 默认抓取全量包数据（snaplen 0），暂不做 TLS 解密配置。
// 抓包操作不允许并发：无论调用方如何调度，均全局串行执行。
func runTcpdump(w io.Writer, host, port string, count int) {
	tcpdumpMu.Lock()
	defer tcpdumpMu.Unlock()

	section(w, "5. TCPDUMP 抓包")

	// Windows 不支持 tcpdump
	if runtime.GOOS == "windows" {
		line(w, "[错误] tcpdump 不支持 Windows 系统，请使用 Wireshark 等工具手动抓包")
		fmt.Printf("[tcpdump] 当前系统为 Windows，请使用 Wireshark 手动抓包\n")
		return
	}

	// 域名与端口兜底校验
	if host == "" || port == "" {
		line(w, "[错误] tcpdump 必须指定域名(-t)和端口(-p)")
		return
	}

	// 检查 tcpdump 命令是否存在
	if _, err := lookPath("tcpdump"); err != nil {
		line(w, "[错误] 未检测到 tcpdump 命令，请先安装")
		fmt.Printf("[tcpdump] 未检测到 tcpdump 命令\n")
		return
	}

	// 抓包数量兜底
	if count <= 0 {
		count = 100
	}

	// pcap 输出文件（按 unix 时间戳命名，避免覆盖）
	pcapFile := fmt.Sprintf("%d-netdiag-tcpdump.pcap", time.Now().Unix())

	args := []string{
		"-nn",     // 不解析主机名与端口名
		"-s", "0", // snaplen 0：抓取全量包数据
		"-c", strconv.Itoa(count), // 抓包数量上限
		"-w", pcapFile, // 输出 pcap 文件
		"host", host, "and", "port", port, // 过滤条件
	}

	line(w, "目标主机: %s", host)
	line(w, "目标端口: %s", port)
	line(w, "抓包数量: %d 包", count)
	line(w, "输出文件: %s", pcapFile)
	line(w, "执行命令: tcpdump %s", strings.Join(args, " "))
	line(w, "说明: 默认抓取全量包数据，暂不做 TLS 解密")

	out, err := runCmd("tcpdump", args...)
	if err != nil {
		// 权限不足是最常见失败原因
		line(w, "[提示] tcpdump 执行异常: %v", err)
		line(w, "[提示] 抓包通常需要 root 权限，请使用 sudo 运行，例如: sudo ./netdiag tcpdump -t %s -p %s", host, port)
	}
	raw(w, out)

	fmt.Printf("[tcpdump] 目标 %s:%s 抓包完成（输出 %s）\n", host, port, pcapFile)
}
