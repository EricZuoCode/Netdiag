package main

// batch.go —— 任务模型、缺省值补齐、文件解析与 goroutine 并发调度。

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// task 表示一个待执行的探测任务，各字段覆盖 ping/dns/telnet/curl 所需参数。
type task struct {
	command   string // 子命令: ping / dns / telnet / curl / all / tcpdump
	target    string // 目标主机 / IP
	port      string // 目标端口
	url       string // curl 探测 URL
	count     int    // ping 发包数量
	timeout   int    // 超时时间（秒）
	curlFlags string // curl 参数
}

// fillTaskDefaults 补齐任务缺省值：
// 仅 curl 与 all 需要构造 URL——目标已携带 http:// 或 https:// 协议则直接使用（不拼接）；
// 否则按端口推导协议：80/8080 等常见 HTTP 端口使用 http，其余（含 443/8443 与默认 443）使用 https；
// 用户通过 -u 显式指定 URL（可自行携带 http/https）时以用户为准。
// ping / dns / telnet / tcpdump 仅使用域名/IP 与端口，不使用 URL，无需推导。
func fillTaskDefaults(t *task) {
	switch t.command {
	case "curl", "all":
		if t.url == "" {
			t.url = buildURL(t.target, t.port)
		}
	}
}

// buildURL 根据目标与端口构造 URL：
// 目标已携带 http:// 或 https:// 协议前缀时直接返回（不拼接）；否则按端口推导协议拼接。
func buildURL(target, port string) string {
	lower := strings.ToLower(target)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return target
	}
	return schemeForPort(port) + "://" + target
}

// schemeForPort 根据端口推导协议：常见 HTTP 端口返回 http，其余默认 https。
func schemeForPort(port string) string {
	switch port {
	case "80", "8080", "8000", "8888":
		return "http"
	default:
		return "https"
	}
}

// executeTask 执行单个任务，结果写入 w。
func executeTask(w io.Writer, t task) {
	timeoutDur := time.Duration(t.timeout) * time.Second
	switch t.command {
	case "ping":
		runPing(w, t.target, t.count, t.timeout)
	case "dns":
		runDNS(w, t.target)
	case "telnet":
		runTelnetWithFallback(w, t.target, t.port, timeoutDur)
	case "curl":
		runCurl(w, t.url, t.curlFlags)
	case "all":
		runPing(w, t.target, t.count, t.timeout)
		runDNS(w, t.target)
		runTelnetWithFallback(w, t.target, t.port, timeoutDur)
		runCurl(w, t.url, t.curlFlags)
	case "tcpdump":
		runTcpdump(w, t.target, t.port, t.count)
	}
}

// runTasksConcurrently 使用 goroutine 并发执行任务，每个任务先写入独立缓冲区，
// 完成后一次性加锁落盘，避免并发写入交错。jobs 为并发数上限。
func runTasksConcurrently(w *resultWriter, tasks []task, jobs int) {
	if jobs <= 0 {
		jobs = len(tasks)
	}
	total := len(tasks)
	sem := make(chan struct{}, jobs)
	var wg sync.WaitGroup

	for i := range tasks {
		wg.Add(1)
		sem <- struct{}{} // 获取信号量，限制并发数
		go func(t task, idx int) {
			defer wg.Done()
			defer func() { <-sem }() // 释放信号量

			var buf bytes.Buffer
			line(&buf, "")
			line(&buf, "<<<<<< 任务 %d/%d | 命令: %s | 目标: %s >>>>>>", idx+1, total, t.command, t.target)
			executeTask(&buf, t)
			w.writeBlock(buf.Bytes())
		}(tasks[i], i)
	}
	wg.Wait()
}

// buildTasksFromArgs 根据命令行参数构造任务列表，支持 -t 逗号分隔多个目标。
func buildTasksFromArgs(command string) []task {
	targets := splitComma(target)
	tasks := make([]task, 0, len(targets))
	for _, t := range targets {
		tk := task{
			command:   command,
			target:    t,
			port:      port,
			url:       url,
			count:     count,
			timeout:   timeout,
			curlFlags: curlFlags,
		}
		fillTaskDefaults(&tk)
		tasks = append(tasks, tk)
	}
	return tasks
}

// splitComma 按逗号拆分并去除空白，用于支持 -t 传入多个目标。
func splitComma(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// splitFields 简单的 shell 风格分词，支持双引号 / 单引号包裹含空格的参数。
func splitFields(line string) []string {
	var fields []string
	var buf strings.Builder
	var quote byte
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			} else {
				buf.WriteByte(c)
			}
		case c == '"' || c == '\'':
			quote = c
		case c == ' ' || c == '\t':
			if buf.Len() > 0 {
				fields = append(fields, buf.String())
				buf.Reset()
			}
		default:
			buf.WriteByte(c)
		}
	}
	if buf.Len() > 0 {
		fields = append(fields, buf.String())
	}
	return fields
}

// parseTaskLine 解析文件中的一行命令为 task。
// 支持 "netdiag all -t www.qq.com -p 80 -u http://www.qq.com" 或省略 netdiag 前缀。
func parseTaskLine(line string) (task, error) {
	fields := splitFields(strings.TrimSpace(line))
	if len(fields) == 0 {
		return task{}, fmt.Errorf("空行")
	}
	// 跳过可选的 "netdiag" / "./netdiag" 前缀
	if fields[0] == "netdiag" || fields[0] == "./netdiag" {
		fields = fields[1:]
	}
	if len(fields) == 0 {
		return task{}, fmt.Errorf("缺少命令")
	}
	command := fields[0]
	if !validCommands[command] || command == "batch" {
		return task{}, fmt.Errorf("未知命令 %q", command)
	}

	t := task{
		command:   command,
		target:    "www.qq.com",
		port:      "443",
		url:       "",
		count:     10,
		timeout:   5,
		curlFlags: "-I -k -v",
	}

	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(io.Discard) // 静默解析错误，交由返回值处理
	fs.StringVar(&t.target, "t", t.target, "")
	fs.StringVar(&t.target, "target", t.target, "")
	fs.StringVar(&t.port, "p", t.port, "")
	fs.StringVar(&t.port, "port", t.port, "")
	fs.StringVar(&t.url, "u", t.url, "")
	fs.StringVar(&t.url, "url", t.url, "")
	fs.IntVar(&t.count, "c", t.count, "")
	fs.IntVar(&t.count, "count", t.count, "")
	fs.IntVar(&t.timeout, "W", t.timeout, "")
	fs.IntVar(&t.timeout, "timeout", t.timeout, "")
	fs.StringVar(&t.curlFlags, "C", t.curlFlags, "")
	fs.StringVar(&t.curlFlags, "curl-flags", t.curlFlags, "")

	if err := fs.Parse(fields[1:]); err != nil {
		return task{}, err
	}
	// tcpdump 抓包数量默认 100 包（区别于 ping 默认 10）
	if command == "tcpdump" {
		cSet := false
		fs.Visit(func(f *flag.Flag) {
			if f.Name == "c" || f.Name == "count" {
				cSet = true
			}
		})
		if !cSet {
			t.count = 100
		}
	}
	fillTaskDefaults(&t)
	return t, nil
}

// loadTasksFromFile 从任务文件读取任务列表，忽略空行与 # 注释行。
func loadTasksFromFile(path string) ([]task, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var tasks []task
	sc := bufio.NewScanner(f)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		t, err := parseTaskLine(line)
		if err != nil {
			return nil, fmt.Errorf("第 %d 行解析失败: %v", lineNo, err)
		}
		tasks = append(tasks, t)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}
