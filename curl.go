package main

import (
	"fmt"
	"io"
	"strings"
)

// runCurl 执行 curl 探测，默认参数为 -I -k -v，收集整体返回信息。
func runCurl(w io.Writer, url string, flags string) {
	section(w, "4. CURL HTTP/HTTPS 探测")
	line(w, "目标 URL: %s", url)

	args := append(strings.Fields(flags), url)
	line(w, "执行命令: curl %s", strings.Join(args, " "))

	out, err := runCmd("curl", args...)
	if err != nil {
		// curl 遇到 4xx/5xx 或证书问题会返回非零退出码，但返回内容仍有排查价值
		line(w, "[提示] curl 返回非零退出码: %v", err)
	}
	raw(w, out)

	fmt.Printf("[curl] URL %s 探测完成\n", url)
}
