package main

// web.go Web 界面路由:
//
//      GET  /       测试界面(浏览器);curl 或 Accept: application/json 时返回服务信息 JSON
//      GET  /help   帮助文档页(端点/参数/CLI/systemd 管理说明)
//
// 安全设计:安装/卸载只提供本机 CLI(sudo bing-search-api install),
// 不设任何 Web 端安装入口——HTTP 端口无鉴权,不能暴露系统写权限。
//
// 所有页面均为内联 HTML(见 pages.go / pages_help.go),无外部资源,离线可用。

import (
	"net/http"
	"os"
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
	sx := "0"
	if searxngEnabled() {
		sx = "1"
	}
	key := "0"
	if os.Getenv("BING_API_KEY") != "" {
		key = "1"
	}
	r := strings.NewReplacer(
		"__VERSION__", version,
		"__LANG_COUNT__", itoa(len(languages)),
		"__PORT__", servePort,
		"__SEARXNG__", sx,
		"__V7KEY__", key,
	)
	return r.Replace(tpl)
}

// itoa 局部小工具,避免引入 strconv
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
		searchDesc := "SearXNG 兼容接口【已禁用】:设 ENABLE_SEARXNG=1 后重启开启(开放代理有滥用风险,默认关闭)"
		if searxngEnabled() {
			searchDesc = "q(必填)、count(默认10)、page(默认1,兼容 pageno)、language(如 zh-CN,见 /languages)、category/categories 选垂直:images/videos/news/dict"
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"name":            "bing-search-api",
			"version":         version,
			"description":     "SearXNG 风格 + Bing 官方 API v7 兼容的极简搜索 API:网页/图片/视频/新闻/词典多类搜索,只搜 Bing,结果保持原始顺序,以 JSON 返回",
			"engine":          "bing",
			"languages":       len(languages),
			"searxng_enabled": searxngEnabled(),
			"ui":              "/          (浏览器打开即测试界面)",
			"endpoints": map[string]string{
				"GET|POST /v7/search":        "Bing 官方 Web Search API v7 调用兼容(官方退役 API 的直接替换):q、count、offset、mkt、setLang、safeSearch、responseFilter",
				"GET|POST /v7/images/search": "图片搜索(官方 Image Search API 兼容, count≤150);别名 /v7.0/、/bing/v7(.0)/images/search",
				"GET|POST /v7/videos/search": "视频搜索(官方 Video Search API 兼容);同样支持别名前缀",
				"GET|POST /v7/news/search":   "新闻搜索(官方 News Search API 兼容);同样支持别名前缀",
				"GET|POST /v7/dict/search":   "词典查询(服务扩展,官方无此端点):中英双向释义",
				"GET|POST /search":           searchDesc,
				"GET /languages":             "全部支持的语言/市场列表",
				"GET /help":                  "帮助文档(浏览器打开)",
				"GET /healthz":               "健康检查",
				"CLI":                        "help | install(仅本机) | uninstall | version | -port N | -host IP",
			},
			"example":        "/v7/search?q=golang&count=10&offset=0&mkt=en-US",
			"example_v7":     "/v7/search?q=golang&count=10&offset=0&mkt=en-US",
			"example_images": "/v7/images/search?q=cat&count=20",
			"example_news":   "/v7/news/search?q=golang",
			"example_dict":   "/v7/dict/search?q=hello",
			"bing_api_key":   os.Getenv("BING_API_KEY") != "",
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
