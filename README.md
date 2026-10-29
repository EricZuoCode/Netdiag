# netdiag — CDN 场景网络基础排障工具

基于 Go 的命令行工具，封装客户端系统命令，用于 CDN 网络排障。支持 ping / dns / telnet / curl / WebSocket（ws/wss）握手探测 / traceroute 路由追踪 / tcpdump 抓包，结果按 unix 时间戳输出到独立 txt 文件。

## 功能

| 功能        | 说明                                       | 依赖                      |
| ----------- | ------------------------------------------ | ------------------------- |
| `ping`    | 发送 ICMP 包统计连通性与丢包率，自动携带 traceroute 验证路径 | 系统 `ping`/`traceroute` |
| `dns`     | `dig` + `nslookup` + 本地解析摘要      | 系统 `dig`/`nslookup` |
| `telnet`  | TCP 端口连通性探测 + `nc` 交叉验证       | Go 标准库 + `nc`        |
| `curl`    | HTTP/HTTPS 探测（默认 `-I -k -v`）       | 系统 `curl`             |
| `ws`      | WebSocket 握手探测（ws/wss，不保持长连接） | Go 标准库                 |
| `tcpdump` | 抓取指定域名端口的包（仅 Linux/macOS）     | 系统 `tcpdump`          |

## 目录结构

```
netdiag/
├── .github/workflows/release.yml    GitHub Actions 自动发版
├── .gitignore
├── SKILL.md                         技能说明（agent 调用入口）
├── go.mod                           模块定义
├── main.go                          入口、参数解析、子命令分发
├── batch.go                         任务模型、文件解析、并发调度
├── output.go                        结果文件写入封装
├── ping.go / dns.go / telnet.go     连通性 / 解析 / 端口探测
├── curl.go / ws.go / capture.go     HTTP / WebSocket / 抓包
└── README.md                        使用说明
```

## 构建

```bash
go build -o netdiag .
```

## 参数说明

| 参数                 | 说明                                         | 默认值                                     |
| -------------------- | -------------------------------------------- | ------------------------------------------ |
| `-t, --target`     | 目标主机/IP，逗号分隔多个，可携带 http(s):// | `www.qq.com`                             |
| `-p, --port`       | 端口                                         | `443`                                    |
| `-u, --url`        | curl/ws 探测 URL                             | 按端口推导（80→http/ws，其余→https/wss） |
| `-c, --count`      | ping 发包数 / tcpdump 抓包数                 | `10`（tcpdump 100）                      |
| `-W, --timeout`    | 超时（秒）                                   | `5`                                      |
| `-o, --output`     | 结果文件路径                                 | `<unix时间戳>-netdiag-output.txt`        |
| `-C, --curl-flags` | curl 参数                                    | `-I -k -v`                               |
| `-T, --clean`      | 运行前清理历史输出文件                       | 关闭                                       |
| `-f, --file`       | 任务文件（batch）                            | 无                                         |
| `-j, --jobs`       | 并发上限                                     | `10`                                     |

## 使用

```bash
./netdiag all                                        # 自检（含 ws）
./netdiag all -t www.qq.com -p 443                   # 指定目标全部探测
./netdiag ping -t 1.2.3.4 -c 20                      # ping
./netdiag dns -t www.qq.com                          # DNS
./netdiag telnet -t 1.2.3.4 -p 8080                  # 端口
./netdiag curl -u https://example.com                # HTTP
./netdiag ws -u wss://example.com                    # WebSocket
./netdiag all -t www.qq.com,www.baidu.com -j 5       # 多目标并发
./netdiag batch -f tasks.txt -j 10                   # 文件批量
sudo ./netdiag tcpdump -t www.qq.com -p 443          # 抓包
```

任务文件每行一条命令（`netdiag` 前缀可省略，`#` 为注释）：

```text
netdiag all -t www.qq.com -p 80 -u http://www.qq.com
netdiag ping -t 1.2.3.4 -c 20
dns -t www.taobao.com
```

## 输出

结果追加写入 `<unix时间戳>-netdiag-output.txt`，多任务以 `<<<<<< 任务 N/M >>>>>>` 分隔；抓包产物为 `<unix时间戳>-netdiag-tcpdump.pcap`。

## 注意事项

- `tcpdump` 仅 Linux/macOS，需 root（`sudo`），串行执行；Windows 用 Wireshark。
- `dig`/`nslookup`/`curl`/`nc`/`traceroute`（Windows 为 `tracert`）需系统安装，`nc` 缺失自动跳过。
- `-T` 会删除当前目录所有 `*-netdiag-output.txt`，请确认后再用。
