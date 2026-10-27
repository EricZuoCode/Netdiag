package main

import (
	"fmt"
	"io"
	"net"
	"strings"
)

// runDig 执行 dig 命令并写入结果。缺失命令时给出明确提示。
func runDig(w io.Writer, host string) {
	section(w, "2. DNS 解析探测 - dig")
	line(w, "目标域名: %s", host)

	// dig 完整输出
	line(w, "执行命令: dig %s", host)
	out, err := runCmd("dig", host)
	if err != nil {
		line(w, "[提示] dig 执行失败: %v", err)
	}
	raw(w, out)

	// dig +short 精简输出（仅解析结果）
	line(w, "")
	line(w, "执行命令: dig +short %s", host)
	short, err := runCmd("dig", "+short", host)
	if err != nil {
		line(w, "[提示] dig +short 执行失败: %v", err)
	}
	raw(w, short)
}

// runNslookup 执行 nslookup 命令并写入结果。
func runNslookup(w io.Writer, host string) {
	line(w, "")
	line(w, "---- nslookup ----")
	line(w, "执行命令: nslookup %s", host)
	out, err := runCmd("nslookup", host)
	if err != nil {
		line(w, "[提示] nslookup 执行失败: %v", err)
	}
	raw(w, out)
}

// runNativeResolve 使用 Go 标准库做本地解析，作为系统命令之外的补充摘要。
func runNativeResolve(w io.Writer, host string) {
	line(w, "")
	line(w, "---- 本地解析摘要 (Go net 标准库) ----")

	ips, err := net.LookupIP(host)
	if err != nil {
		line(w, "IP 解析失败: %v", err)
		return
	}

	var v4, v6 []string
	for _, ip := range ips {
		if ip.To4() != nil {
			v4 = append(v4, ip.String())
		} else {
			v6 = append(v6, ip.String())
		}
	}
	if len(v4) > 0 {
		line(w, "A 记录: %s", strings.Join(v4, ", "))
	}
	if len(v6) > 0 {
		line(w, "AAAA 记录: %s", strings.Join(v6, ", "))
	}

	// LookupCNAME 对无 CNAME 记录的域名会原样返回带尾点的规范名，需去除尾点后再比较
	if cname, err := net.LookupCNAME(host); err == nil {
		if canonical := strings.TrimSuffix(cname, "."); canonical != "" && canonical != host {
			line(w, "CNAME: %s", canonical)
		}
	}
}

// runDNS 执行 DNS 解析探测的完整流程。
func runDNS(w io.Writer, host string) {
	runDig(w, host)
	runNslookup(w, host)
	runNativeResolve(w, host)
	fmt.Printf("[dns] 域名 %s 解析探测完成\n", host)
}
