package main

// bing-search-api:一个极简的 SearXNG 风格搜索服务。
// 只抓取 Bing 一个引擎,不重排序、不聚合,把 Bing 的原始结果
// 顺序整理成 JSON 通过 API 返回。
//
// 端点:
//   GET|POST /search   搜索(q 必填;count/page/language 可选)
//   GET      /languages 全部支持的语言/市场列表
//   GET      /healthz  健康检查
//   GET      /         服务信息

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// searchParams 从请求解析出的搜索参数
type searchParams struct {
	Q        string // 查询词
	Count    int    // 每页条数(1~50)
	Page     int    // 页码(从 1 开始)
	Language string // 语言/市场,如 zh-CN,为空则由 Bing 自动判断
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// version 由构建时通过 -ldflags 注入,如:
//
//	go build -ldflags "-X main.version=v1.0.0" .
var version = "dev"

func main() {
	port := flag.String("port", envOr("PORT", "8080"), "HTTP 监听端口")
	bingBase := flag.String("bing", envOr("BING_BASE", bingDefaultBase), "Bing 入口 URL")
	showVer := flag.Bool("version", false, "打印版本号并退出")
	flag.Parse()
	if *showVer {
		fmt.Println("bing-search-api", version)
		return
	}

	srv := &server{
		engine: &BingEngine{
			Base:   *bingBase,
			Client: &http.Client{Timeout: 15 * time.Second},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/search", srv.handleSearch)
	mux.HandleFunc("/languages", srv.handleLanguages)
	mux.HandleFunc("/healthz", srv.handleHealth)
	mux.HandleFunc("/", srv.handleRoot)

	httpSrv := &http.Server{
		Addr:              ":" + *port,
		Handler:           withRecover(withCORS(withLog(mux))),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       20 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	go func() {
		log.Printf("bing-search-api 已启动,监听 :%s (Bing: %s)", *port, *bingBase)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("服务启动失败: %v", err)
		}
	}()

	// 优雅退出
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Println("正在关闭 ...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(ctx)
	log.Println("已退出")
}

// server 持有引擎引用
type server struct {
	engine *BingEngine
}

// handleSearch 搜索入口,支持 GET 与 POST(表单或 JSON)
func (s *server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "仅支持 GET / POST"})
		return
	}

	p := paramsFromRequest(r)
	if strings.TrimSpace(p.Q) == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "缺少查询参数 q"})
		return
	}

	// 语言解析(与 SearXNG 行为一致):
	//   - language=all/any     → 不限语言
	//   - 未指定/auto          → 依次尝试 Accept-Language 头,失败则由 Bing 自动判断
	//   - zh / zh-CN / zh-Hans → 统一映射到 Bing mkt 市场
	market, err := resolveLanguage(p.Language, r.Header.Get("Accept-Language"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: err.Error(),
			Query: p.Q,
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	resp, err := s.engine.Search(ctx, QueryParams{
		Term:     p.Q,
		Count:    p.Count,
		First:    (p.Page-1)*p.Count + 1,
		Language: market,
	})
	if err != nil {
		log.Printf("搜索失败 q=%q lang=%q: %v", p.Q, market, err)
		writeJSON(w, http.StatusBadGateway, ErrorResponse{
			Error: "Bing 查询失败: " + err.Error(),
			Query: p.Q,
		})
		return
	}
	// 响应标注实际生效的语言(空 = 不限)
	if market == "" {
		resp.Language = "all"
	} else {
		resp.Language = market
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleHealth 健康检查
func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

// handleLanguages 列出全部支持的语言/市场
func (s *server) handleLanguages(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"count":     len(languages),
		"special":   []string{"all", "auto"},
		"languages": languages,
		"usage":     "/search?q=example&language=zh-CN",
		"note":      "language 支持 语言代码(zh)、语言-地区代码(zh-CN)、别名(zh-Hans、es-419)与 all(不限语言);未指定时自动使用 Accept-Language 请求头,行为与 SearXNG 一致",
	})
}

// handleRoot 服务信息
func (s *server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "接口不存在: " + r.URL.Path})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"name":        "bing-search-api",
		"version":     version,
		"description": "SearXNG 风格的极简搜索 API:只搜 Bing,结果保持原始顺序,以 JSON 返回",
		"engine":      "bing",
		"languages":   len(languages),
		"endpoints": map[string]string{
			"GET|POST /search": "q(必填)、count(默认10)、page(默认1,兼容 pageno)、language(如 zh-CN,见 /languages)",
			"GET /languages":   "全部支持的语言/市场列表",
			"GET /healthz":     "健康检查",
		},
		"example": "/search?q=golang&count=10&page=1&language=zh-CN",
	})
}

// paramsFromRequest 兼容三种传参方式:URL 查询串、POST 表单、POST JSON
func paramsFromRequest(r *http.Request) searchParams {
	p := searchParams{Count: 10, Page: 1}

	// POST + JSON
	if r.Method == http.MethodPost && strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		var body struct {
			Q        string `json:"q"`
			Count    int    `json:"count"`
			Page     int    `json:"page"`
			Language string `json:"language"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err == nil {
			if body.Q != "" {
				p.Q = body.Q
			}
			if body.Count > 0 {
				p.Count = body.Count
			}
			if body.Page > 0 {
				p.Page = body.Page
			}
			if body.Language != "" {
				p.Language = body.Language
			}
			return clamp(p)
		}
		// JSON 解析失败则继续按表单/查询串处理
	}

	// URL 查询串 / POST 表单(FormValue 会自动合并两者)
	p.Q = r.FormValue("q")
	p.Language = r.FormValue("language")
	if v, err := strconv.Atoi(r.FormValue("count")); err == nil {
		p.Count = v
	}
	// page 与 pageno(SearXNG 兼容)二选一
	for _, name := range []string{"page", "pageno"} {
		if v, err := strconv.Atoi(r.FormValue(name)); err == nil {
			p.Page = v
			break
		}
	}
	return clamp(p)
}

// clamp 把参数收敛到安全范围
func clamp(p searchParams) searchParams {
	if p.Count < 1 {
		p.Count = 10
	}
	if p.Count > 50 {
		p.Count = 50
	}
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Page > 100 {
		p.Page = 100
	}
	return p
}

// writeJSON 统一 JSON 输出
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		log.Printf("写响应失败: %v", err)
	}
}

// statusWriter 记录响应状态码,供日志中间件使用
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
}

// withLog 访问日志
func withLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		log.Printf("%s %s -> %d (%s)", r.Method, r.URL.RequestURI(), sw.status,
			time.Since(start).Round(time.Millisecond))
	})
}

// withCORS 开放跨域,方便浏览器前端直接调用
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// withRecover panic 兜底,避免进程退出
func withRecover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic: %v", err)
				writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "服务器内部错误"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
