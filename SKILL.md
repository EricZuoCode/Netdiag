---
name: netdiag
description: CDN 场景网络基础排障工具。当需要执行 ping 连通性测试、DNS 解析探测（dig/nslookup）、TCP 端口连通性探测、HTTP/HTTPS 探测（curl）、WebSocket 握手探测（ws/wss）或 tcpdump 抓包时使用此技能。工具封装客户端系统命令，支持多目标并发与文件批量导入，结果按 unix 时间戳输出到独立 txt 文件。
agent_created: true
---

# Netdiag — CDN 网络基础排障工具

## 概述

netdiag 是封装客户端系统命令（ping / dig / nslookup / nc / curl / tcpdump）的 Go 命令行工具，用于 CDN 网络排障。本技能目录即项目根目录：源码、`dist/` 全平台产物、README.md 同在此处。

## 工具位置

- 项目根目录 = 本技能所在目录（SKILL.md 同级）
- 全平台交叉编译产物：`dist/`（Linux/macOS/Windows × amd64/arm64）
- 编译：在技能目录执行 `go build -o netdiag .`

执行方式：

```bash
cd <本技能目录>
go build -o netdiag .   # 若尚未编译，先构建
./netdiag <command> [flags]
```

## 子命令

| 命令 | 用途 |
| --- | --- |
| `ping` | ping 连通性测试，统计丢包率与 RTT |
| `dns` | DNS 解析探测（dig + nslookup + 本地解析摘要） |
| `telnet` | TCP 端口连通性探测 |
| `curl` | HTTP/HTTPS 探测（默认 `-I -k -v`） |
| `ws` | WebSocket 握手探测（ws/wss，仅握手不保持长连接） |
| `all` | ping → dns → telnet → curl；默认自检额外含 ws，带参数时 http/ws 二选一 |
| `batch` | 从文件批量导入任务并并发执行 |
| `tcpdump` | 抓包（仅 Linux/macOS，需指定域名与端口） |

## 关键参数

- `-t, --target`：目标主机/IP，逗号分隔多个，可携带 `http(s)://` 或 `ws(s)://`（默认 `www.qq.com`）
- `-p, --port`：端口（默认 `443`）
- `-u, --url`：curl/ws 探测 URL（默认按端口推导：80→http/ws，其余→https/wss）
- `-c, --count`：ping 发包数 / tcpdump 抓包数（默认 `10`，tcpdump `100`）
- `-o, --output`：结果文件路径（默认 `<unix时间戳>-netdiag-output.txt`）
- `-j, --jobs`：并发任务数上限（默认 `10`）
- 其余参数见 `./netdiag --help` 或同目录 README.md

## 常用示例

```bash
./netdiag all -t www.qq.com -p 443                  # 全部探测（http/ws 按 URL 协议二选一）
./netdiag dns -t www.qq.com                          # DNS 解析
./netdiag curl -u https://example.com                # HTTP 探测
./netdiag ws -u wss://example.com                    # WebSocket 握手
./netdiag all -t www.qq.com,www.baidu.com -j 5       # 多目标并发
./netdiag batch -f tasks.txt -j 10                   # 文件批量
sudo ./netdiag tcpdump -t www.qq.com -p 443          # 抓包（需 root）
```

## 注意事项

- `tcpdump` 仅支持 Linux/macOS，需 root 权限（`sudo`）、串行执行；Windows 用 Wireshark 替代。
- 探测结果写入 `<unix时间戳>-netdiag-output.txt`；抓包产物为 `<unix时间戳>-netdiag-tcpdump.pcap`。
- 完整参数与详细说明见同目录 README.md。
