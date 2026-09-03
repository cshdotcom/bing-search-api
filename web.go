package main

// web.go Web 界面路由:
//
//   GET  /            测试界面(浏览器);curl 或 Accept: application/json 时返回服务信息 JSON
//   GET  /help        帮助文档页(端点/参数/CLI/systemd 管理说明)
//   GET  /install     安装向导页(?probe=1 返回状态 JSON)
//   POST /install     在当前进程权限内执行 systemd 安装(需 root)
//
// 所有页面均为内联 HTML(见 pages.go),无外部资源,离线可用。

import (
	"net/http"
	"strings"
)

// wantsJSON 内容协商:浏览器(带 text/html 的 Accept)拿 HTML,
// curl(空 Accept / */* / application/json)拿 JSON,保持既有 API 兼容。
func wantsJSON(r *http.Request) bool {
	if fv := strings.ToLower(r.FormValue("format")); fv == "json" || fv == "api" {
		return true
	}
	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "text/html") {
		return false
	}
	if strings.Contains(accept, "application/json") || accept == "" || accept == "*/*" {
		return true
	}
	return false
}

// writeHTML 统一 HTML 输出
func writeHTML(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

// renderPage 把页面模板中的占位符替换为运行时信息
func renderPage(tpl string) string {
	r := strings.NewReplacer(
		"__VERSION__", version,
		"__LANG_COUNT__", itoa(len(languages)),
		"__PORT__", servePort,
	)
	return r.Replace(tpl)
}

// itoa 局部小工具,避免 web.go 依赖 strconv
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [8]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

// handleRoot 测试界面 / 服务信息
func (s *server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "接口不存在: " + r.URL.Path})
		return
	}
	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{
			"name":        "bing-search-api",
			"version":     version,
			"description": "SearXNG 风格的极简搜索 API:只搜 Bing,结果保持原始顺序,以 JSON 返回",
			"engine":      "bing",
			"languages":   len(languages),
			"ui":          "/          (浏览器打开即测试界面)",
			"endpoints": map[string]string{
				"GET|POST /search": "q(必填)、count(默认10)、page(默认1,兼容 pageno)、language(如 zh-CN,见 /languages)",
				"GET /languages":   "全部支持的语言/市场列表",
				"GET /help":        "帮助文档(浏览器打开)",
				"GET /install":     "systemd 安装向导(浏览器打开)",
				"POST /install":    "以服务进程权限执行安装(需 root,建议用 CLI)",
				"GET /healthz":     "健康检查",
				"CLI":              "help | install | uninstall | version | -port N | -host IP",
			},
			"example": "/search?q=golang&count=10&page=1&language=zh-CN",
		})
		return
	}
	writeHTML(w, http.StatusOK, renderPage(testPageHTML))
}

// handleHelpPage 帮助文档页
func (s *server) handleHelpPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "仅支持 GET"})
		return
	}
	writeHTML(w, http.StatusOK, renderPage(helpPageHTML))
}

// handleInstallPage 安装向导页
//
//	GET  /install          → HTML 向导(状态检测 + 命令展示 + 一键安装按钮)
//	GET  /install?probe=1  → JSON 状态(供页面 JS 与脚本使用)
//	POST /install          → 在服务进程权限内执行安装
func (s *server) handleInstallPage(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		if r.FormValue("probe") != "" {
			writeJSON(w, http.StatusOK, installProbe())
			return
		}
		writeHTML(w, http.StatusOK, renderPage(installPageHTML))
	case http.MethodPost:
		s.handleInstallExec(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "仅支持 GET / POST"})
	}
}

// handleInstallExec Web 端执行安装。
// 服务进程通常并非 root,此时返回 403 + 指引(推荐用 CLI 的 sudo install)。
func (s *server) handleInstallExec(w http.ResponseWriter, r *http.Request) {
	port := strings.TrimSpace(r.FormValue("port"))
	if port == "" {
		port = servePort
	}
	if p, err := normalizePort(port); err == nil {
		port = p
	} else {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "端口无效: " + err.Error()})
		return
	}
	host := strings.TrimSpace(r.FormValue("host"))
	if host == "" {
		host = "0.0.0.0"
	}

	if osGeteuid() != 0 {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"error": "当前服务进程不是 root,无法直接安装 systemd 服务",
			"hint":  "请在服务器终端执行: sudo " + osExecutableHint() + " install -port " + port,
			"note":  "CLI 的 install 子命令会自动通过 sudo 提权,复制二进制到 /usr/local/bin,注册 systemd 服务并设置开机自启动",
		})
		return
	}
	if err := doInstall(port, host, true, true); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": err.Error(),
			"hint":  "可改在终端执行: sudo bing-search-api install -port " + port,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"service": serviceName,
		"unit":    unitPath,
		"port":    port,
		"manage":  "systemctl status|restart|stop " + serviceName,
	})
}
