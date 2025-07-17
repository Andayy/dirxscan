package main

// DirXScan - 高性能多线程目录泄露扫描器
// Author: x1n

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gookit/color"
)

var (
	result []string
	wg     sync.WaitGroup
)

func banner() {
	fmt.Println(`
██████╗ ██╗██████╗ ██╗  ██╗███████╗██╗  ██╗ ██████╗ ███████╗ █████╗ ███╗   ██╗
██╔══██╗██║██╔══██╗██║ ██╔╝██╔════╝██║ ██╔╝██╔═══██╗██╔════╝██╔══██╗████╗  ██║
██████╔╝██║██████╔╝█████╔╝ █████╗  █████╔╝ ██║   ██║█████╗  ███████║██╔██╗ ██║
██╔═══╝ ██║██╔═══╝ ██╔═██╗ ██╔══╝  ██╔═██╗ ██║   ██║██╔══╝  ██╔══██║██║╚██╗██║
██║     ██║██║     ██║  ██╗███████╗██║  ██╗╚██████╔╝██║     ██║  ██║██║ ╚████║
╚═╝     ╚═╝╚═╝     ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝ ╚═════╝ ╚═╝     ╚═╝  ╚═╝╚═╝  ╚═══╝

            - GO语言实现目录扫描工具 DirXScan 
`)
}

func main() {
	banner()

	target := flag.String("u", "", "目标 URL，例如：https://example.com")
	dictFile := flag.String("w", "./common.txt", "目录字典路径")
	threadCount := flag.Int("t", 20, "线程数量（默认 20）")

	flag.Parse()

	if *target == "" {
		fmt.Println("[-] 必须使用 -u 参数指定目标 URL")
		flag.Usage()
		os.Exit(1)
	}

	runtime.GOMAXPROCS(runtime.NumCPU())

	dict, err := readLines(*dictFile)
	if err != nil {
		fmt.Printf("[-] 无法读取字典文件: %v\n", err)
		os.Exit(1)
	}

	fullDict := generateDict(*target, dict)
	runScan(*target, fullDict, *threadCount)
}

// 扫描调度器
func runScan(target string, paths []string, threads int) {
	total := len(paths)
	perThread := total / threads
	if total%threads != 0 {
		perThread++
	}
	fmt.Printf("[*] 扫描目标: %s | 线程: %d | 任务数: %d\n", target, threads, total)

	wg.Add(threads)
	for i := 0; i < threads; i++ {
		start := i * perThread
		end := start + perThread
		if end > total {
			end = total
		}
		go scanWorker(target, paths[start:end])
	}
	wg.Wait()
}

// 子扫描线程
func scanWorker(target string, paths []string) {
	defer wg.Done()
	for _, p := range paths {
		full := strings.TrimRight(target, "/") + "/" + strings.TrimLeft(p, "/")
		status, err := checkPath(full)
		if err != nil {
			continue
		}
		switch status {
		case 200, 301, 403:
			color.Green.Printf("[+] [%s] Found: %s (%d)\n", time.Now().Format("15:04:05"), full, status)
			result = append(result, full)
		default:
			fmt.Printf("[-] Checking: %s        \r", full)
		}
	}
}

// 自动生成路径扩展字典
func generateDict(raw string, base []string) []string {
	out := make([]string, 0, len(base)+20)
	u, err := url.Parse(raw)
	if err != nil {
		return base
	}
	host := u.Hostname()
	parts := strings.Split(host, ".")

	exts := []string{".zip", ".rar", ".tar.gz", ".sql", ".bak", ".log", ".txt"}
	for _, ext := range exts {
		out = append(out, "/"+host+ext)
		if len(parts) >= 2 {
			out = append(out, "/"+parts[len(parts)-2]+"."+parts[len(parts)-1]+ext)
			out = append(out, "/"+parts[len(parts)-2]+ext)
		}
	}
	out = append(out, base...)
	return out
}

func readLines(path string) ([]string, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n"), nil
}

func checkPath(url string) (int, error) {
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}
