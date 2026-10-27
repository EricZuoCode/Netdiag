package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// 全局命令行参数，短名与长名绑定到同一变量。
var (
	target    string // 目标主机 / IP（支持逗号分隔多个）
	port      string // 目标端口
	url       string // curl 探测 URL
	count     int    // ping 发包数量
	timeout   int    // 超时时间（秒）
	output    string // 结果输出文件
	curlFlags string // curl 参数
	cleanup   bool   // 运行前是否清理历史输出文件
	file      string // 任务文件路径
	jobs      int    // 并发任务数上限
)

// validCommands 记录所有合法子命令。
var validCommands = map[string]bool{
	"ping":    true,
	"dns":     true,
	"telnet":  true,
	"curl":    true,
	"all":     true,
	"batch":   true,
	"tcpdump": true,
}

func init() {
	flag.StringVar(&target, "t", "www.qq.com", "目标主机 / IP 地址（支持逗号分隔多个，可携带 http(s)://）")
	flag.StringVar(&target, "target", "www.qq.com", "目标主机 / IP 地址（支持逗号分隔多个，可携带 http(s)://）")

	flag.StringVar(&port, "p", "443", "目标端口")
	flag.StringVar(&port, "port", "443", "目标端口")

	flag.StringVar(&url, "u", "", "curl 探测的目标 URL（默认按端口推导：80→http，其余→https）")
	flag.StringVar(&url, "url", "", "curl 探测的目标 URL（默认按端口推导：80→http，其余→https）")

	flag.IntVar(&count, "c", 10, "ping 发包数量")
	flag.IntVar(&count, "count", 10, "ping 发包数量")

	flag.IntVar(&timeout, "W", 5, "超时时间（秒）")
	flag.IntVar(&timeout, "timeout", 5, "超时时间（秒）")

	flag.StringVar(&output, "o", "", "结果输出文件路径（默认 <unix时间戳>-netdiag-output.txt）")
	flag.StringVar(&output, "output", "", "结果输出文件路径（默认 <unix时间戳>-netdiag-output.txt）")

	flag.StringVar(&curlFlags, "C", "-I -k -v", "curl 探测参数")
	flag.StringVar(&curlFlags, "curl-flags", "-I -k -v", "curl 探测参数")

	flag.BoolVar(&cleanup, "T", false, "运行前清理当前目录下 *-netdiag-output.txt 历史输出文件")
	flag.BoolVar(&cleanup, "clean", false, "运行前清理当前目录下 *-netdiag-output.txt 历史输出文件")

	flag.StringVar(&file, "f", "", "任务文件路径（每行一条 netdiag 命令）")
	flag.StringVar(&file, "file", "", "任务文件路径（每行一条 netdiag 命令）")

	flag.IntVar(&jobs, "j", 10, "并发任务数上限")
	flag.IntVar(&jobs, "jobs", 10, "并发任务数上限")
}

// runCmd 执行命令并返回合并后的标准输出 / 错误输出与错误。
func runCmd(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// lookPath 在 PATH 中查找命令的绝对路径。
func lookPath(name string) (string, error) {
	return exec.LookPath(name)
}

// flagSet 判断指定名称的 flag 是否被用户显式设置。
func flagSet(names ...string) bool {
	set := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		set[f.Name] = true
	})
	for _, n := range names {
		if set[n] {
			return true
		}
	}
	return false
}

// usage 打印帮助信息。
func usage() {
	fmt.Fprintf(os.Stderr, `netdiag - CDN 场景网络基础排障工具

用法:
  netdiag <command> [flags]

命令:
  ping      执行 ping 连通性测试
  dns       执行 DNS 解析探测 (dig + nslookup)
  telnet    执行端口连通性探测
  curl      执行 HTTP/HTTPS 探测
  all       按顺序执行 ping -> dns -> telnet -> curl
  batch     从文件导入任务并并发执行
  tcpdump   执行抓包操作（仅 Linux/macOS，需指定域名与端口）

示例:
  netdiag all
  netdiag ping -t 1.2.3.4 -c 20
  netdiag dns -t www.qq.com
  netdiag telnet -t 1.2.3.4 -p 8080
  netdiag curl -u https://example.com/path
  netdiag all -t www.qq.com -p 80 -u http://www.qq.com
  netdiag tcpdump -t www.qq.com -p 443

  多目标（逗号分隔，并发执行）:
  netdiag all -t www.qq.com,www.baidu.com,www.taobao.com -j 5

  文件批量（每行一条命令，并发执行）:
  netdiag batch -f tasks.txt -j 10

  任务文件格式（netdiag 前缀可省略，# 开头为注释行）:
    netdiag all -t www.qq.com -p 80 -u http://www.qq.com
    netdiag ping -t 1.2.3.4 -c 20
    netdiag telnet -t 1.2.3.4 -p 8080

参数:
  -t, --target string      目标主机 / IP 地址，逗号分隔多个，可携带 http(s):// (默认 www.qq.com)
  -p, --port string        目标端口 (默认 443)
  -u, --url string         curl 探测的目标 URL (默认按端口推导：80→http，其余→https)
  -c, --count int          ping 发包数量 / tcpdump 抓包数量 (默认 10，tcpdump 默认 100)
  -W, --timeout int        超时时间，单位秒 (默认 5)
  -o, --output string      结果输出文件路径 (默认 <unix时间戳>-netdiag-output.txt)
  -C, --curl-flags string  curl 探测参数 (默认 "-I -k -v")
  -T, --clean              运行前清理 *-netdiag-output.txt 历史输出文件
  -f, --file string        任务文件路径（batch 命令使用）
  -j, --jobs int           并发任务数上限 (默认 10)
`)
}

func main() {
	flag.Usage = usage // 设置帮助信息

	if len(os.Args) < 2 { // 如果没有命令行参数，则打印帮助信息
		usage()
		os.Exit(1)
	}
	command := os.Args[1]               // 获取命令行参数中的第一个参数，即命令名称
	flag.CommandLine.Parse(os.Args[2:]) // 解析命令行第二个参数及其后续参数

	if !validCommands[command] { // 检查第一个参数命令是否合法
		fmt.Fprintf(os.Stderr, "错误: 未知命令 %q\n\n", command)
		usage()
		os.Exit(1)
	}

	// 构建任务列表（此时尚未打开结果文件，解析失败可直接退出）
	var tasks []task
	switch command {
	case "batch":
		if file == "" {
			fmt.Fprintf(os.Stderr, "错误: batch 命令需要 -f/--file 指定任务文件\n\n")
			usage()
			os.Exit(1)
		}
		var err error
		tasks, err = loadTasksFromFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: 读取任务文件失败: %v\n", err)
			os.Exit(1)
		}
	case "tcpdump":
		if !flagSet("t", "target") {
			fmt.Fprintf(os.Stderr, "错误: tcpdump 必须显式指定域名 -t/--target\n\n")
			usage()
			os.Exit(1)
		}
		if !flagSet("p", "port") {
			fmt.Fprintf(os.Stderr, "错误: tcpdump 必须显式指定端口 -p/--port\n\n")
			usage()
			os.Exit(1)
		}
		// 抓包数量：未显式指定 -c 时默认 100 包
		if !flagSet("c", "count") {
			count = 100
		}
		tasks = buildTasksFromArgs(command)
	default:
		tasks = buildTasksFromArgs(command)
	}

	if len(tasks) == 0 {
		fmt.Fprintf(os.Stderr, "错误: 未解析到任何任务\n")
		os.Exit(1)
	}

	// 未指定 -o 时，按当前 unix 时间戳生成默认输出文件名，避免多次执行冲突
	if output == "" {
		output = fmt.Sprintf("%d-netdiag-output.txt", time.Now().Unix()) // 生成默认输出文件名
	}

	// 可选：清理历史输出文件
	if cleanup {
		if n, err := cleanHistoryOutputs(); err == nil && n > 0 {
			fmt.Printf("已清理 %d 个历史输出文件 (*-netdiag-output.txt)\n", n) // 清理历史输出文件
		}
	}

	// 打开结果文件
	w, err := newResultWriter(output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: 无法写入结果文件 %s: %v\n", output, err)
		os.Exit(1)
	}
	defer w.Close()

	w.runHeader(strings.Join(os.Args, " "))
	runTasksConcurrently(w, tasks, jobs)

	fmt.Printf("\n结果已写入: %s（共 %d 个任务）\n", output, len(tasks))
}
