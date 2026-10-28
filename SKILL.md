---
name: netdiag
description: CDN 场景网络基础排障工具。此技能应在需要执行 ping 连通性测试、DNS 解析探测（dig/nslookup）、TCP 端口连通性探测、HTTP/HTTPS 探测（curl）、WebSocket 握手探测（ws/wss）或 tcpdump 抓包时使用。工具封装客户端系统命令，支持多目标并发与文件批量导入，探测结果按 unix 时间戳输出到独立 txt 文件。
agent_created: true
---

# Netdiag — CDN 网络基础排障工具

## Overview

netdiag 是一个编译好的 Go 命令行工具，封装客户端系统自带的 ping / dig / nslookup / nc / curl / tcpdump 命令，用于 CDN 场景下的网络基础排障。一次运行即可完成连通性、域名解析、端口连通性、HTTP 探测与抓包，结果统一写入 txt 文件，便于直接排查。

## 工具位置

- 本机（macOS arm64）编译产物：`/Users/zuoxuchen/Desktop/File/代码/netdiag/netdiag`
- 全平台交叉编译产物（Linux/macOS/Windows × amd64/arm64）：`/Users/zuoxuchen/Desktop/File/代码/netdiag/dist/`
- 若目标机器无产物，源码位于 `/Users/zuoxuchen/Desktop/File/代码/netdiag/`，可 `go build -o netdiag .` 重新编译

执行方式（以本机产物为例）：

```bash
/Users/zuoxuchen/Desktop/File/代码/netdiag/netdiag <command> [flags]
```

## 子命令

| 命令 | 用途 |
| --- | --- |
| `ping` | ping 连通性测试，统计丢包率与 RTT |
| `dns` | DNS 解析探测（dig + nslookup + Go 本地解析摘要） |
| `telnet` | TCP 端口连通性探测（Go 原生拨号 + nc 交叉验证） |
| `curl` | HTTP/HTTPS 探测（默认 `-I -k -v`） |
| `ws` | WebSocket 握手探测（ws/wss，仅握手不保持长连接） |
| `all` | 顺序执行 ping → dns → telnet → curl；默认自检额外含 ws 探测，带参数时 http/ws 二选一 |
| `batch` | 从文件导入任务并并发执行 |
| `tcpdump` | 抓包（仅 Linux/macOS，需指定域名与端口） |

## 常用参数

| 参数 | 说明 | 默认值 |
| --- | --- | --- |
| `-t, --target` | 目标主机/IP，逗号分隔多个，可携带 http(s):// | `www.qq.com` |
| `-p, --port` | 目标端口 | `443` |
| `-u, --url` | curl 探测 URL | 按端口推导（80→http，其余→https） |
| `-c, --count` | ping 发包数量 / tcpdump 抓包数量 | `10`（tcpdump 100） |
| `-W, --timeout` | 超时（秒） | `5` |
| `-o, --output` | 结果输出文件路径 | `<unix时间戳>-netdiag-output.txt` |
| `-C, --curl-flags` | curl 参数 | `-I -k -v` |
| `-T, --clean` | 运行前清理 `*-netdiag-output.txt` 历史文件 | 关闭 |
| `-f, --file` | 任务文件（batch 使用） | 无 |
| `-j, --jobs` | 并发任务数上限 | `10` |

## 使用示例

对指定域名执行全部探测：

```bash
netdiag all -t www.qq.com -p 443
```

单项探测：

```bash
netdiag ping -t 1.2.3.4 -c 20
netdiag dns -t www.qq.com
netdiag telnet -t 1.2.3.4 -p 8080
netdiag curl -u https://example.com
netdiag ws -u wss://example.com
```

多目标并发（逗号分隔）：

```bash
netdiag all -t www.qq.com,www.baidu.com -j 5
```

文件批量（每行一条命令，# 为注释行）：

```bash
netdiag batch -f tasks.txt -j 10
```

抓包（需 root）：

```bash
sudo netdiag tcpdump -t www.qq.com -p 443
```

## 输出说明

- 结果默认追加写入 `<unix时间戳>-netdiag-output.txt`，每次运行生成独立文件，避免覆盖。
- 多任务并发时每个任务结果块完整不交错，以 `<<<<<< 任务 N/M | 命令 | 目标 >>>>>>` 分隔。
- 抓包产物为 `<unix时间戳>-netdiag-tcpdump.pcap`，可用 Wireshark 打开。

## 注意事项

- `tcpdump` 仅支持 Linux/macOS；Windows 需自行使用 Wireshark 抓包。
- tcpdump 抓包需 root 权限（`sudo` 运行），且串行执行（不允许并发）。
- ping 参数已按 macOS/Linux/Windows 自动适配。
- `dig`/`nslookup`/`curl`/`nc` 需系统已安装；`nc` 缺失自动跳过交叉验证。
