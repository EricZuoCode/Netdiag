---
name: netdiag
description: CDN 场景网络基础排障工具。当需要执行 ping 连通性测试、DNS 解析探测（dig/nslookup）、TCP 端口连通性探测、HTTP/HTTPS 探测（curl）、WebSocket 握手探测（ws/wss）或 tcpdump 抓包时使用此技能。工具封装客户端系统命令，支持多目标并发与文件批量导入，结果按 unix 时间戳输出到独立 txt 文件。
agent_created: true
---

# Netdiag — CDN 网络基础排障工具

netdiag 是封装客户端系统命令（ping / dig / nslookup / nc / curl / tcpdump）的 Go 命令行工具，用于 CDN 网络排障。

## 功能

| 命令 | 用途 |
| --- | --- |
| `ping` | ping 连通性测试，统计丢包率与 RTT |
| `dns` | DNS 解析探测（dig + nslookup） |
| `telnet` | TCP 端口连通性探测 |
| `curl` | HTTP/HTTPS 探测（默认 `-I -k -v`） |
| `ws` | WebSocket 握手探测（ws/wss，不保持长连接） |
| `all` | ping → dns → telnet → curl；自检含 ws，带参数 http/ws 二选一 |
| `batch` | 从文件批量导入任务并发执行 |
| `tcpdump` | 抓包（仅 Linux/macOS） |

## 关键参数

- `-t, --target`：目标主机/IP，逗号分隔多个，可携带 `http(s)://` 或 `ws(s)://`（默认 `www.qq.com`）
- `-p, --port`：端口（默认 `443`）
- `-u, --url`：curl/ws 探测 URL（默认按端口推导：80→http/ws，其余→https/wss）
- `-c, --count`：ping 发包数 / tcpdump 抓包数（默认 `10`，tcpdump `100`）
- `-o, --output`：结果文件路径（默认 `<unix时间戳>-netdiag-output.txt`）
- `-j, --jobs`：并发上限（默认 `10`）

## 常用示例

```bash
./netdiag all -t www.qq.com -p 443                  # 全部探测
./netdiag dns -t www.qq.com                          # DNS 解析
./netdiag curl -u https://example.com                # HTTP 探测
./netdiag ws -u wss://example.com                    # WebSocket 握手
./netdiag all -t www.qq.com,www.baidu.com -j 5       # 多目标并发
./netdiag batch -f tasks.txt -j 10                   # 文件批量
sudo ./netdiag tcpdump -t www.qq.com -p 443          # 抓包（需 root）
```

## 注意事项

- `tcpdump` 仅 Linux/macOS，需 root（`sudo`）、串行执行；Windows 用 Wireshark。
- 结果写入 `<unix时间戳>-netdiag-output.txt`；抓包产物 `<unix时间戳>-netdiag-tcpdump.pcap`。
