# netdiag — CDN 场景网络基础排障工具

一个基于 Go 编写的命令行小工具，封装客户端系统自带命令，用于 CDN 场景下的网络基础排障。支持 goroutine 并发，可一次对多个目标并发探测，也可从文件批量导入任务；除 ping/dns/telnet/curl 探测外，还支持 WebSocket 握手探测（ws/wss）与 tcpdump 抓包（仅 Linux/macOS）。每次执行的探测结果按当前 unix 时间戳生成独立输出文件（`<unix时间戳>-netdiag-output.txt`），避免多次执行相互覆盖。

## 功能

| 功能   | 说明                                                                        | 依赖命令                    |
| ------ | --------------------------------------------------------------------------- | --------------------------- |
| ping   | 发送指定数量（默认 10 个，可配置）ICMP 包，统计连通性与丢失占比             | 系统 `ping`               |
| dns    | 执行 `dig` 与 `nslookup` 探测，验证域名解析是否正常，并补充本地解析摘要 | 系统 `dig` / `nslookup` |
| telnet | 探测指定 IP 与端口的 TCP 连通性，并做 `nc` 交叉验证                       | Go 标准库 + 系统 `nc`     |
| curl   | 执行 HTTP/HTTPS 探测（默认 `-I -k -v`），收集完整返回信息                 | 系统 `curl`               |
| ws     | 执行 WebSocket 握手探测（ws/wss），仅握手不保持长连接                     | Go 标准库                |
| tcpdump | 抓取指定域名与端口的网络包（仅 Linux/macOS），输出 pcap 文件               | 系统 `tcpdump`           |
| 并发   | goroutine 并发执行，支持命令行多目标与文件批量导入                          | —                          |
| 导出   | 结果按 unix 时间戳生成独立文件，并发落盘不交错                              | —                          |

## 目录结构

```
netdiag/
├── .github/workflows/release.yml    GitHub Actions 自动发版
├── .gitignore                       忽略编译产物与结果文件
├── go.mod                           模块定义
├── main.go                          入口、命令行参数解析、子命令分发
├── batch.go                         任务模型、文件解析、并发调度
├── output.go                        结果文件写入封装、历史输出清理
├── ping.go                          ping 连通性测试
├── dns.go                           DNS 解析探测
├── telnet.go                        端口连通性探测
├── curl.go                          HTTP/HTTPS 探测
├── ws.go                            WebSocket 握手探测
├── capture.go                       tcpdump 抓包
└── README.md                        使用说明
```

## 构建

若目录下已存在可直接运行的 `netdiag` 编译后产物，则无需重新构建，直接执行 `netdiag` 命令即可。仅当需要从源码重新编译时才需执行：

```bash
cd netdiag
go build -o netdiag .
# Golang代码版本 1.27.3
```

## 参数说明

| 参数                 | 说明                                             | 默认值                              |
| -------------------- | ------------------------------------------------ | ----------------------------------- |
| `-t, --target`     | 目标主机 / IP，逗号分隔多个，可携带 http(s)://    | `www.qq.com`                      |
| `-p, --port`       | 目标端口                                         | `443`                             |
| `-u, --url`        | curl/ws 探测的目标 URL                          | 按端口推导（80→http/ws，其余→https/wss） |
| `-c, --count`      | ping 发包数量 / tcpdump 抓包数量               | `10`（tcpdump 默认 100）          |
| `-W, --timeout`    | 超时时间（秒）                                   | `5`                               |
| `-o, --output`     | 结果输出文件路径                                 | `<unix时间戳>-netdiag-output.txt` |
| `-C, --curl-flags` | curl 探测参数                                    | `-I -k -v`                        |
| `-T, --clean`      | 运行前清理 `*-netdiag-output.txt` 历史输出文件 | 关闭                                |
| `-f, --file`       | 任务文件路径（`batch` 命令使用）               | 无（推荐使用绝对路径）              |
| `-j, --jobs`       | 并发任务数上限                                   | `10`                              |

## 使用

### 单项 / 全量探测

> 首次使用或不确定工具是否正常时，可先执行 `./netdiag all`（不带参数，以默认值探测 www.qq.com 及公开 ws 端点）验证工具可正常运行、无报错；该命令无实际排障价值，仅用于可用性自检。

`all` 不带参数为自检（ping → dns → telnet → curl + ws 全量探测）；带上参数时，按 URL 协议二选一执行 http 或 ws 探测（ping → dns → telnet → curl 或 ws）：

```bash
./netdiag all                                        # 自检：以默认值探测 www.qq.com，验证工具可用性
./netdiag all -t www.qq.com -p 443                   # 指定目标与端口，执行全部探测
./netdiag ping -t 1.2.3.4 -c 20                      # ping 20 个包
./netdiag dns -t www.qq.com                          # DNS 解析探测
./netdiag telnet -t 1.2.3.4 -p 8080                  # 端口连通性探测
./netdiag curl -u https://example.com                # HTTP/HTTPS 探测
./netdiag ws -u wss://example.com                    # WebSocket 握手探测
```

### 抓包（tcpdump）

```bash
./netdiag tcpdump -t www.qq.com -p 443                # 抓取 www.qq.com:443 的包（默认 100 个）
./netdiag tcpdump -t www.qq.com -p 443 -c 500         # 抓取 500 个包
```

抓包需指定域名（`-t`）与端口（`-p`），结果输出为 `<unix时间戳>-netdiag-tcpdump.pcap` 文件，可用 Wireshark 打开分析。

### 多目标并发（逗号分隔）

```bash
./netdiag all -t www.qq.com,www.baidu.com,www.taobao.com -j 5
./netdiag dns -t a.com,b.com,c.com
```

`-t` 传入逗号分隔的多个目标，工具会对每个目标并发执行同一探测命令。

### 文件批量导入

```bash
./netdiag batch -f tasks.txt -j 10
```

任务文件每行一条命令，格式与命令行一致（`netdiag` 前缀可省略，`#` 开头为注释行）：

```text
netdiag all -t www.qq.com -p 80 -u http://www.qq.com
netdiag ping -t 1.2.3.4 -c 20
netdiag telnet -t 1.2.3.4 -p 8080
dns -t www.taobao.com
# 这是注释行，会被忽略
```

## 输出说明

默认输出文件名按当前 unix 时间戳生成（形如 `1793073057-netdiag-output.txt`），每次执行得到独立文件，天然避免多次执行的覆盖冲突；`-o` 可指定自定义输出路径。

多任务并发时，每个任务先写入独立缓冲区、完成后一次性落盘，因此每个任务的结果块完整不交错，并以 `<<<<<< 任务 N/M | 命令 | 目标 >>>>>>` 标识分隔，便于在多目标场景下定位问题。每项探测保留系统命令的原始输出，并附带统计摘要（ping 丢包率、RTT、连通性结论等）。

当历史输出文件累积较多时，可附加 `-T`（`--clean`）在本次运行前清理当前目录下所有 `*-netdiag-output.txt` 历史文件，再生成本次的新输出文件。

## 注意事项

1. **平台差异**：ping 参数已按 macOS / Linux / Windows 自动适配（`-c`/`-n`、超时单位毫秒/秒）。
2. **命令依赖**：`dig`、`nslookup`、`curl`、`nc` 需系统已安装；`nc` 缺失时自动跳过交叉验证，不影响主流程。
3. **curl 进度表**：`-v` 输出在重定向到文件时会携带进度表，属 curl 原生行为；如需隐藏可追加 `-sS`，例如 `-C "-I -k -v -sS"`。
4. **历史输出清理**：`-T` 会删除当前目录下所有 `*-netdiag-output.txt` 文件，请确认无其他需保留的同名文件后再使用。
5. **并发数**：`-j` 控制并发上限，默认 10；任务数较少时实际并发即为任务数。
6. **抓包平台**：`tcpdump` 仅支持 Linux / macOS；Windows 请使用 Wireshark 等工具手动抓包。
7. **抓包权限与范围**：tcpdump 抓包需 root 权限（建议 `sudo` 运行）；须指定域名与端口以缩小抓包范围、避免抓包过大；默认抓取全量包数据，暂不支持 TLS 解密。
8. **抓包串行**：tcpdump 抓包操作不允许并发，多个抓包任务会按顺序串行执行。
