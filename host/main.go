// offline-dl-saver —— MoeKoeMusic 离线下载器本地宿主（macOS / Linux 版）
// 协议与原版 Windows 端 Host.cs 完全一致：
//   - 传输：stdin/stdout，JSON Lines（每行一条 JSON，UTF-8，\n 分隔）
//   - 入站可能包一层：{"type":"message","payload":{...}}，取 payload；否则读顶层
//   - 命令：ping / ping_all / download / exists / check
//   - 出站：握手 hello；回包 reply；进度事件 start / progress；exists；check(have)
//
// 编译（Mac M 芯片 arm64）：
//   CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o offline-dl-saver main.go
// 编译（Mac Intel x64）：
//   CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o offline-dl-saver main.go
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ---------- 出站收发 ----------

func send(v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	// JSON Lines：json + 换行
	fmt.Println(string(b))
}

// reply 通用回包。id 原样回显（数字/字符串都保留）。
func reply(id any, ok bool, msg, path string, bytes int64) {
	send(map[string]any{
		"id":    id,
		"ok":    ok,
		"msg":   msg,
		"path":  path,
		"bytes": bytes,
	})
}

func evStart(id any) {
	send(map[string]any{"id": id, "event": "start"})
}

func evProgress(id any, received, total, percent int64) {
	send(map[string]any{
		"id":       id,
		"event":    "progress",
		"received": received,
		"total":    total,
		"percent":  percent,
	})
}

// ---------- 文件名清洗 ----------

func isReservedDeviceName(stem string) bool {
	if stem == "" {
		return false
	}
	s := strings.TrimRight(stem, " ")
	if s == "" {
		return false
	}
	up := strings.ToUpper(s)
	switch up {
	case "CON", "PRN", "AUX", "NUL":
		return true
	}
	if len(s) == 4 && s[3] >= '1' && s[3] <= '9' {
		if (up[0] == 'C' && up[1] == 'O' && up[2] == 'M') ||
			(up[0] == 'L' && up[1] == 'P' && up[2] == 'T') {
			return true
		}
	}
	return false
}

func cleanName(name string) string {
	if name == "" {
		return name
	}
	var sb strings.Builder
	for _, c := range name {
		bad := c < 32 || c == 127 ||
			c == '<' || c == '>' || c == ':' || c == '"' ||
			c == '/' || c == '\\' || c == '|' || c == '?' || c == '*'
		// Unicode 双向覆盖 / 隔离符 / 零宽字符
		if (c >= 0x200B && c <= 0x200F) ||
			(c >= 0x202A && c <= 0x202E) ||
			(c >= 0x2060 && c <= 0x206F) ||
			c == 0xFEFF {
			bad = true
		}
		if bad {
			sb.WriteByte('_')
			continue
		}
		sb.WriteRune(c)
	}
	out := strings.Trim(sb.String(), " .")
	out = strings.TrimPrefix(out, ".")
	if out == "" {
		out = "unknown"
	}
	if len(out) > 180 {
		out = out[:180]
	}
	dot := strings.Index(out, ".")
	stem := out
	if dot >= 0 {
		stem = out[:dot]
	}
	if isReservedDeviceName(stem) {
		out = "_" + out
	}
	return out
}

// ---------- 目录匹配 ----------

func isKeyChar(c rune) bool {
	if c >= 'a' && c <= 'z' {
		return true
	}
	if c >= '0' && c <= '9' {
		return true
	}
	if c == '_' {
		return true
	}
	if c >= 0x4e00 && c <= 0x9fff { // CJK 统一表意
		return true
	}
	if c >= 0x3040 && c <= 0x30ff { // 平假名 / 片假名
		return true
	}
	if c >= 0xac00 && c <= 0xd7af { // 韩文音节
		return true
	}
	return false
}

func looseKey(s string) string {
	var sb strings.Builder
	for _, c := range s {
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		if isKeyChar(c) {
			sb.WriteRune(c)
		}
	}
	return sb.String()
}

// 精确 stem 匹配（忽略扩展名，忽略大小写）。
func hasStem(dir, stem string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.EqualFold(strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())), stem) {
			return true
		}
	}
	return false
}

// 忽略标点 / 空格的宽松匹配，用于 check 批量查重。
func hasLooseStem(dir, want string) bool {
	k := looseKey(want)
	if k == "" {
		return false
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if looseKey(strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))) == k {
			return true
		}
	}
	return false
}

// ---------- 错误转译 ----------

func describe(err error) string {
	if err == nil {
		return ""
	}
	if uerr, ok := err.(*url.Error); ok {
		if uerr.Timeout() {
			return "连接超时"
		}
		return uerr.Err.Error()
	}
	return err.Error()
}

// ---------- HTTP 客户端 ----------

var httpClient = &http.Client{
	Timeout: 20 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 8 {
			return fmt.Errorf("重定向次数过多")
		}
		return nil
	},
}

func newRequest(rawURL string) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Referer", "http://www.kugou.com/")
	return req, nil
}

func httpCodeText(code int) string {
	switch code {
	case 403:
		return "无访问权限（该曲目需要会员或受地区限制）"
	case 404:
		return "地址已失效（重新取流再试）"
	case 416:
		return "地址范围无效"
	case 429:
		return "请求过于频繁，被限流"
	case 503:
		return "服务器繁忙"
	default:
		return http.StatusText(code)
	}
}

// ---------- 业务命令 ----------

func cmdDownload(body map[string]any) {
	id := body["id"]
	urlStr := strOf(body["url"])
	dest := strOf(body["dest"])
	dir := expandHome(strOf(body["dir"]))

	if dest == "" {
		reply(id, false, "dest 不能为空", "", 0)
		return
	}
	dest = cleanName(dest)

	if dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}

	full := dest
	if !filepath.IsAbs(dest) {
		full = filepath.Join(dir, dest)
	}

	// 越界兜底：目标必须落在下载目录内。
	if dir != "" {
		guardRoot, err1 := filepath.Abs(dir)
		target, err2 := filepath.Abs(full)
		if err1 != nil || err2 != nil {
			reply(id, false, "路径无效", "", 0)
			return
		}
		prefix := guardRoot
		if !strings.HasSuffix(prefix, string(filepath.Separator)) {
			prefix += string(filepath.Separator)
		}
		if !strings.HasPrefix(target, prefix) {
			reply(id, false, "目标路径越出了下载目录，已拒绝", "", 0)
			return
		}
		full = target
	}

	if parent := filepath.Dir(full); parent != "" {
		_ = os.MkdirAll(parent, 0o755)
	}

	evStart(id)

	part := full + ".part"
	_ = os.Remove(part)

	var total int64
	ok := false
	lastErr := ""

	for attempt := 1; attempt <= 3 && !ok; attempt++ {
		total = 0
		if attempt > 1 {
			// 保持 bridge 的静默看门狗在重试期间不超时
			evProgress(id, 0, 0, 0)
			time.Sleep(time.Duration(600*attempt) * time.Millisecond)
		}

		req, err := newRequest(urlStr)
		if err != nil {
			lastErr = describe(err)
			continue
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = describe(err)
			if resp != nil {
				resp.Body.Close()
			}
			continue
		}

		declared := resp.ContentLength
		evProgress(id, 0, declared, 0)

		out, err := os.OpenFile(part, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			resp.Body.Close()
			lastErr = err.Error()
			continue
		}

		buf := make([]byte, 65536)
		lastReport := int64(0)
		lastClock := time.Now()
		clock := time.Now()
		readOK := true

		for {
			n, rerr := resp.Body.Read(buf)
			if n > 0 {
				if _, werr := out.Write(buf[:n]); werr != nil {
					lastErr = werr.Error()
					readOK = false
					break
				}
				total += int64(n)
				if total-lastReport >= 262144 || time.Since(lastClock) >= 400*time.Millisecond {
					lastReport = total
					lastClock = time.Now()
					pct := int64(0)
					if declared > 0 {
						pct = total * 100 / declared
					}
					evProgress(id, total, declared, pct)
				}
			}
			if rerr == io.EOF {
				break
			}
			if rerr != nil {
				lastErr = rerr.Error()
				readOK = false
				break
			}
		}

		out.Close()
		resp.Body.Close()
		_ = clock

		// 短读校验：声明长度大于 0 但没读全，按失败处理。
		if readOK && declared > 0 && total < declared {
			lastErr = fmt.Sprintf("连接中断，只收到 %d/%d 字节", total, declared)
			readOK = false
		}

		if readOK {
			ok = true
		} else {
			_ = os.Remove(part)
		}
	}

	if !ok {
		reply(id, false, "下载失败（重试 3 次）: "+lastErr, "", total)
		return
	}
	if total == 0 {
		_ = os.Remove(full)
		reply(id, false, "下载内容为空", "", 0)
		return
	}

	_ = os.Remove(full)
	if err := os.Rename(part, full); err != nil {
		reply(id, false, "落盘失败: "+err.Error(), "", total)
		return
	}
	evProgress(id, total, total, 100)
	reply(id, true, "ok", full, total)
}

func cmdExists(body map[string]any) {
	id := body["id"]
	dir := expandHome(strOf(body["dir"]))
	name := cleanName(strOf(body["dest"]))
	found := false
	if dir != "" && name != "" {
		stem := strings.TrimSuffix(name, filepath.Ext(name))
		if entries, err := os.ReadDir(dir); err == nil {
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				if strings.EqualFold(strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())), stem) {
					found = true
					break
				}
			}
		}
	}
	send(map[string]any{"id": id, "ok": true, "exists": found})
}

func cmdCheck(body map[string]any) {
	id := body["id"]
	cDir := expandHome(strOf(body["dir"]))
	have := []int{}

	okDir := cDir != ""
	if entries, err := os.ReadDir(cDir); err != nil {
		okDir = false
		_ = entries
	}

	if okDir {
		if names, ok := body["names"].([]any); ok {
			for i := 0; i < len(names) && len(have) < 200; i++ {
				want := strOf(names[i])
				if want == "" {
					continue
				}
				if hasLooseStem(cDir, want) {
					have = append(have, i)
				}
			}
		}
	}
	send(map[string]any{"id": id, "ok": true, "have": have})
}

// ---------- 工具 ----------

// expandHome 把 ~ / ~/path 展开成用户主目录下的绝对路径（Mac/Linux）。
func expandHome(p string) string {
	if p == "~" {
		if h, err := os.UserHomeDir(); err == nil {
			return h
		}
		return p
	}
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~\\") {
		if h, err := os.UserHomeDir(); err == nil {
			return filepath.Join(h, p[2:])
		}
	}
	return p
}

func strOf(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return fmt.Sprintf("%v", t)
	case bool:
		return fmt.Sprintf("%v", t)
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

func asMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}

// ---------- 主循环 ----------

func main() {
	// 启动握手
	send(map[string]any{"type": "hello", "host": "offline-dl-saver"})

	scanner := bufio.NewScanner(os.Stdin)
	// 放大缓冲，避免一批中文名被 64KB 默认上限截断
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var root map[string]any
		if err := json.Unmarshal([]byte(line), &root); err != nil {
			continue
		}

		// 客户端退出通知
		if t, _ := root["type"].(string); t == "shutdown" {
			return
		}

		// 消息可能包一层 {"type":"message","payload":{...}}
		body := asMap(root["payload"])
		if body == nil {
			body = root
		}

		id := body["id"]
		cmd := strOf(body["cmd"])

		switch cmd {
		case "ping", "ping_all":
			reply(id, true, "pong", "", 0)
		case "download":
			cmdDownload(body)
		case "exists":
			cmdExists(body)
		case "check":
			cmdCheck(body)
		default:
			reply(id, false, "未知命令: "+cmd, "", 0)
		}
	}
}
