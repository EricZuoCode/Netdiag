package main

// output.go —— 结果文件写入封装（追加模式、并发安全）与历史输出清理。

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// resultWriter 封装结果文件的写入，追加模式，并发安全。
type resultWriter struct {
	path string     // 结果文件路径
	file *os.File   // 底层文件句柄
	mu   sync.Mutex // 互斥锁，保证并发写入安全
}

// newResultWriter 以追加模式打开（必要时创建）结果文件。
func newResultWriter(path string) (*resultWriter, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &resultWriter{path: path, file: f}, nil
}

// cleanHistoryOutputs 清理当前目录下所有匹配 *-netdiag-output.txt 的历史输出文件，
// 返回成功删除的文件数量。
func cleanHistoryOutputs() (int, error) {
	entries, err := os.ReadDir(".")
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), "-netdiag-output.txt") {
			if err := os.Remove(e.Name()); err == nil {
				removed++
			}
		}
	}
	return removed, nil
}

// Close 关闭结果文件。
func (w *resultWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Close()
}

// Write 实现 io.Writer，单次写入原子，保证并发安全。
func (w *resultWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Write(p)
}

// writeBlock 加锁写入一个完整结果块，保证块内内容不被其他任务交错。
func (w *resultWriter) writeBlock(b []byte) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.file.Write(b)
}

// runHeader 写入本次运行的总头信息，包含执行时间与原始命令行。
func (w *resultWriter) runHeader(cmdLine string) {
	sep := strings.Repeat("#", 72)
	fmt.Fprintf(w, "\n%s\n# 执行时间: %s\n# 执行命令: %s\n%s\n",
		sep, time.Now().Format("2006-01-02 15:04:05"), cmdLine, sep)
}

// section 向 w 写入一个带标题的分隔区块。
func section(w io.Writer, title string) {
	sep := strings.Repeat("=", 72)
	fmt.Fprintf(w, "\n%s\n%s\n%s\n", sep, title, sep)
}

// line 向 w 写入一行格式化文本。
func line(w io.Writer, format string, args ...interface{}) {
	fmt.Fprintf(w, format+"\n", args...)
}

// raw 向 w 写入原始输出块，自动去除尾部多余空行。
func raw(w io.Writer, s string) {
	fmt.Fprintln(w, strings.TrimRight(s, "\n"))
}
