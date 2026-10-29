---
name: netdiag
description: CDN 场景网络基础排障工具。当需要执行 ping 连通性测试、DNS 解析探测（dig/nslookup）、TCP 端口连通性探测、HTTP/HTTPS 探测（curl）、WebSocket 握手探测（ws/wss）或 tcpdump 抓包时使用此技能。工具封装客户端系统命令，支持多目标并发与文件批量导入，结果按 unix 时间戳输出到独立 txt 文件。不适用于带宽测速与业务层排障。
agent_created: true
---

# Netdiag — CDN 网络基础排障工具

netdiag 是封装客户端系统命令（ping / dig / nslookup / nc / curl / tcpdump）的 Go 命令行工具，用于 CDN 网络排障。

## 前置准备

- 获取工具：在技能目录执行 `go build -o netdiag .` 构建，或从 GitHub Release 下载对应平台二进制。
- 系统依赖：`dig`/`nslookup`/`curl`/`nc`/`traceroute`（Windows 为 `tracert`）需系统安装；`nc` 缺失时自动跳过交叉验证，不影响主流程。

## 功能

| 命令 | 用途 |
| --- | --- |
| `ping` | ping 连通性测试（统计丢包率/RTT），并自动携带 traceroute 验证路径 |
| `dns` | DNS 解析探测（dig + nslookup） |
| `telnet` | TCP 端口连通性探测 |
| `curl` | HTTP/HTTPS 探测（默认 `-I -k -v`） |
| `ws` | WebSocket 握手探测（ws/wss，不保持长连接） |
| `all` | ping → dns → telnet → curl；自检含 ws，带参数 http/ws 二选一 |
| `batch` | 从文件批量导入任务并发执行 |
| `traceroute` | 路由路径追踪（traceroute / tracert） |
| `tcpdump` | 抓包（仅 Linux/macOS） |

## 关键参数

- `-t, --target`：目标主机/IP，逗号分隔多个，可携带 `http(s)://` 或 `ws(s)://`（默认 `www.qq.com`）
- `-p, --port`：端口（默认 `443`）
- `-u, --url`：curl/ws 探测 URL（默认按端口推导：80→http/ws，其余→https/wss）
- `-c, --count`：ping 发包数 / tcpdump 抓包数（默认 `10`，tcpdump `100`）
- `-o, --output`：结果文件路径（默认 `<unix时间戳>-netdiag-output.txt`）
- `-j, --jobs`：并发上限（默认 `10`）
- `-T, --clean`：运行前清理 `*-netdiag-output.txt` 历史文件（⚠️ 会删除当前目录全部该文件）

## 常用示例

```bash
./netdiag all -t www.qq.com -p 443                  # 全部探测
./netdiag dns -t www.qq.com                          # DNS 解析
./netdiag curl -u https://example.com                # HTTP 探测
./netdiag ws -u wss://example.com                    # WebSocket 握手
./netdiag all -t www.qq.com,www.baidu.com -j 5       # 多目标并发
./netdiag batch -f tasks.txt -j 10                   # 文件批量
./netdiag traceroute -t www.qq.com                    # 路由追踪
sudo ./netdiag tcpdump -t www.qq.com -p 443          # 抓包（需 root）
```

## 结果解读

- ping 丢失占比：`0%` 正常；`<100%` 部分丢包（存在网络波动）；`100%` 完全不可达。RTT 为往返时延（min/avg/max）。
- 常见失败原因：DNS 解析失败、目标端口未监听（telnet 失败）、服务端不支持 WebSocket（ws 非 101）、curl 非零退出码（4xx/5xx 或证书问题）。

## 注意事项

- ⚠️ `-T/--clean` 会删除当前目录下全部 `*-netdiag-output.txt` 文件，执行前须经用户确认，避免误删。
- `tcpdump` 仅 Linux/macOS，需 root（`sudo`）、串行执行；Windows 用 Wireshark。
- 结果写入 `<unix时间戳>-netdiag-output.txt`；抓包产物 `<unix时间戳>-netdiag-tcpdump.pcap`。
