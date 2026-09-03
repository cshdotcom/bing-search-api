package main

// bing-search-api:一个极简的 SearXNG 风格搜索服务。
// 只抓取 Bing 一个引擎,不重排序、不聚合,把 Bing 的原始结果
// 顺序整理成 JSON 通过 API 返回。
//
// CLI 子命令:
//   (默认)              前台启动 HTTP 服务
//   install             安装为 systemd 服务并设开机自启动(自动 sudo 提权,仅限本机终端)
//   uninstall           卸载 systemd 服务与二进制
//   help                终端打印帮助
//   version             打印版本号
//
// 端点:
//   GET|POST /search         SearXNG 兼容接口【默认禁用,设 ENABLE_SEARXNG=1 开启】
//                             (q 必填;count/page/language 可选;category/categories 选垂直)
//   GET|POST /v7/search       Bing 官方 Web Search API v7 调用兼容(默认启用;q/count/offset/mkt/safeSearch…)
//                             (别名 /v7.0/search、/bing/v7/search、/bing/v7.0/search)
//   GET|POST /v7/images/search  图片搜索(官方 Image Search API 兼容,同样支持上述别名前缀)
//   GET|POST /v7/videos/search  视频搜索(官方 Video Search API 兼容)
//   GET|POST /v7/news/search    新闻搜索(官方 News Search API 兼容)
//   GET|POST /v7/dict/search    词典查询(服务扩展,官方无此端点)
//   GET      /languages 全部支持的语言/市场列表
//   GET      /help      帮助文档页
//   GET      /healthz   健康检查
//   GET      /          测试界面(浏览器)/ 服务信息 JSON(curl)
//
// 可选鉴权:设 BING_API_KEY 环境变量后,/v7/* 需要
// Ocp-Apim-Subscription-Key 头(或 subscription-key 参数)与之匹配。
//
// 安全:①安装/卸载仅限本机 CLI(sudo bing-search-api install),
// 不提供任何 Web 端安装入口;②SearXNG 兼容接口 /search 默认禁用
// (开放代理易被滥用),须设 ENABLE_SEARXNG=1 显式开启。

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
        Category string // 垂直类别:空=网页综合;images/videos/news/dict
}

func envOr(key, def string) string {
        if v := os.Getenv(key); v != "" {
                return v
        }
        return def
}

// searxngEnabled 判断 SearXNG 兼容接口(/search)是否开启。
// 出于安全考虑默认关闭:开放搜索代理会被陌生流量滥用(白嫖出口 IP 抓取、
// 触发 Bing 风控),须显式设 ENABLE_SEARXNG=1/true/yes/on 才启用。
func searxngEnabled() bool {
        switch strings.ToLower(strings.TrimSpace(os.Getenv("ENABLE_SEARXNG"))) {
        case "1", "true", "yes", "on":
                return true
        }
        return false
}

// version 由构建时通过 -ldflags 注入,如:
//
//      go build -ldflags "-X main.version=v1.1.0" .
var version = "dev"

// servePort 当前服务的监听端口(页面渲染用),启动时赋值
var servePort = "8080"

const defaultPort = "8080"

// main CLI 入口:支持 "子命令在前、flag 在后"(install -port 9000)
// 与 "flag 在前、子命令在后"(-port 9000 install)两种写法。
func main() {
        head, sub, rest := splitArgs(os.Args[1:])

        // 全局 flag(出现在子命令之前的部分)
        global := flag.NewFlagSet("bing-search-api", flag.ExitOnError)
        global.SetOutput(os.Stderr)
        global.Usage = func() { printHelp(os.Stdout) }
        gPort := global.String("port", envOr("PORT", defaultPort), "HTTP 监听端口")
        gHost := global.String("host", envOr("HOST", "0.0.0.0"), "HTTP 监听地址")
        gBing := global.String("bing", envOr("BING_BASE", bingDefaultBase), "Bing 入口 URL")
        gVer := global.Bool("version", false, "打印版本号并退出")
        _ = global.Parse(head)

        if *gVer {
                fmt.Println("bing-search-api", version)
                return
        }

        switch sub {
        case "", "serve", "run":
                // 服务子命令:允许 flag 出现在子命令之后,覆盖全局默认
                fs := flag.NewFlagSet("serve", flag.ExitOnError)
                fs.SetOutput(os.Stderr)
                fs.Usage = func() { printHelp(os.Stderr) }
                p := fs.String("port", *gPort, "HTTP 监听端口")
                h := fs.String("host", *gHost, "HTTP 监听地址")
                b := fs.String("bing", *gBing, "Bing 入口 URL")
                if len(rest) > 0 {
                        _ = fs.Parse(rest)
                }
                serve(*p, *h, *b)

        case "help", "-h", "--help":
                printHelp(os.Stdout)

        case "version", "-v", "--version":
                fmt.Println("bing-search-api", version)

        case "install":
                runInstallCLI(*gPort, *gHost, rest)

        case "uninstall", "remove":
                runUninstallCLI(rest)

        default:
                fmt.Fprintf(os.Stderr, "未知子命令: %q\n\n", sub)
                printHelp(os.Stderr)
                os.Exit(2)
        }
}

// splitArgs 把命令行参数拆为:
// head = 子命令之前的全局 flag 部分;sub = 第一个位置参数(子命令);
// rest = 子命令之后的参数。没有子命令时 head = 全部参数。
func splitArgs(args []string) (head []string, sub string, rest []string) {
        valueFlags := map[string]bool{
                "-port": true, "-host": true, "-bing": true,
        }
        i := 0
        for i < len(args) {
                a := args[i]
                if a == "--" {
                        // 之后视为 flag 区域(罕见用法)
                        head = append(head, args[i:]...)
                        return head, "", nil
                }
                if strings.HasPrefix(a, "-") && a != "-" {
                        if strings.Contains(a, "=") {
                                head = append(head, a)
                                i++
                                continue
                        }
                        head = append(head, a)
                        if valueFlags[a] && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
                                // 已知值型 flag:连同取值一起归入 head
                                head = append(head, args[i+1])
                                i += 2
                                continue
                        }
                        // 布尔 flag 或未知 flag(交给 flag.Parse 报错):单 token 归入 head
                        i++
                        continue
                }
                // 第一个位置参数 = 子命令
                return head, a, args[i+1:]
        }
        return head, "", nil
}

// printHelp 终端帮助文本
func printHelp(w *os.File) {
        fmt.Fprintf(w, `bing-search-api %s — SearXNG 风格的极简 Bing 搜索服务

用法:
  bing-search-api [子命令] [参数]

子命令:
  (默认)       前台启动 HTTP 服务
  install      安装为 systemd 服务并设开机自启动(非 root 自动 sudo 提权;仅限本机终端执行,Web 端无安装入口)
  uninstall    卸载 systemd 服务与二进制
  help         显示本帮助
  version      打印版本号

通用参数(写在子命令前或后均可):
  -port N      HTTP 监听端口(默认 8080,或环境变量 PORT)
  -host IP     监听地址(默认 0.0.0.0)
  -bing URL    Bing 入口(默认 https://www.bing.com,可设 https://cn.bing.com)

install 专属参数:
  -no-start    只注册服务,不立即启动

示例:
  bing-search-api                        前台启动,默认 8080
  bing-search-api -port 9000             指定端口启动
  bing-search-api install -port 9000     安装 systemd 服务,端口 9000
  sudo bing-search-api install           安装(install 内部会自动提权,可不加 sudo)
  sudo bing-search-api uninstall         卸载

服务安装后管理:
  systemctl status|start|stop|restart bing-search-api
  journalctl -u bing-search-api -f

Web 界面(服务启动后浏览器访问):
  http://localhost:%s/          测试界面(直接搜索,语言/分页可选)
  http://localhost:%s/help      帮助文档(端点/参数/管理命令)

API:
  GET|POST /v7/search Bing 官方 Web Search API v7 兼容(默认启用):q、count、offset、mkt、safeSearch
                       (官方 API 已退役,存量代码改 base URL 即可继续用;
                        设 BING_API_KEY 后需 Ocp-Apim-Subscription-Key 头)
  GET|POST /v7/images/search  图片搜索(官方 Image Search API 兼容,count≤150)
  GET|POST /v7/videos/search  视频搜索(官方 Video Search API 兼容)
  GET|POST /v7/news/search    新闻搜索(官方 News Search API 兼容)
  GET|POST /v7/dict/search    词典查询(服务扩展,官方无此端点)
                       (以上 v7 端点均支持 /v7.0/、/bing/v7(.0)/ 别名前缀,同样受 BING_API_KEY 鉴权)
  GET|POST /search    SearXNG 兼容接口【默认禁用】:设环境变量 ENABLE_SEARXNG=1 后重启开启
                       (q、count、page、language;category/categories 选垂直 images/videos/news/dict;
                        开放代理有滥用风险,故默认关闭)
  GET      /languages 全部 %d 个语言/市场
  GET      /healthz   健康检查
  https://github.com/cshdotcom/bing-search-api
`, version, defaultPort, defaultPort, len(languages))
}

// printSubUsage 子命令的 -h 说明
func printSubUsage(fs *flag.FlagSet, name string) {
        fmt.Fprintf(os.Stderr, "用法: bing-search-api %s [参数]\n\n", name)
        fs.PrintDefaults()
}

// serve 前台启动 HTTP 服务
func serve(port, host, bingBase string) {
        portN, err := normalizePort(port)
        if err != nil {
                fatalf("端口参数无效: %v", err)
        }
        servePort = portN

        srv := &server{
                engine: &BingEngine{
                        Base:   bingBase,
                        Client: &http.Client{Timeout: 15 * time.Second},
                },
        }

        mux := srv.routes()

        addr := host + ":" + portN
        httpSrv := &http.Server{
                Addr:              addr,
                Handler:           withRecover(withCORS(withLog(mux))),
                ReadHeaderTimeout: 5 * time.Second,
                ReadTimeout:       20 * time.Second,
                WriteTimeout:      60 * time.Second, // v7 大 count 需多页聚合,放宽
        }

        go func() {
                log.Printf("bing-search-api %s 已启动: http://localhost:%s/ (测试界面)", version, portN)
                if searxngEnabled() {
                        log.Printf("帮助文档: /help   API: /search(SearXNG 兼容,已启用,含 images/videos/news/dict 垂直) + /v7/{,images/,videos/,news/,dict/}search(官方 API 兼容)   语言: /languages (Bing: %s)", bingBase)
                } else {
                        log.Printf("帮助文档: /help   API: /v7/{,images/,videos/,news/,dict/}search(Bing 官方 API 兼容)   语言: /languages (Bing: %s)", bingBase)
                        log.Printf("SearXNG 兼容接口 /search 已禁用(默认):设 ENABLE_SEARXNG=1 后重启可开启")
                }
                if os.Getenv("BING_API_KEY") != "" {
                        log.Printf("v7 兼容端点已启用密钥鉴权(BING_API_KEY 已设置)")
                }
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

// routes 注册全部 HTTP 路由(serve 与单元测试共用)。
// Bing 官方 Search API v7 调用兼容层:web + 垂直(images/videos/news/dict),
// 每个端点均覆盖官方两代路径 + 常见变体(4 个前缀别名)。
func (s *server) routes() *http.ServeMux {
        mux := http.NewServeMux()
        mux.HandleFunc("/search", s.handleSearch)
        v7Route := func(path string, h http.HandlerFunc) {
                for _, prefix := range []string{"/v7/", "/v7.0/", "/bing/v7/", "/bing/v7.0/"} {
                        mux.HandleFunc(prefix+path, h)
                }
        }
        v7Route("search", http.HandlerFunc(s.handleBingV7Search))
        v7Route("images/search", http.HandlerFunc(s.handleBingV7Images))
        v7Route("videos/search", http.HandlerFunc(s.handleBingV7Videos))
        v7Route("news/search", http.HandlerFunc(s.handleBingV7News))
        v7Route("dict/search", http.HandlerFunc(s.handleBingV7Dict))
        mux.HandleFunc("/languages", s.handleLanguages)
        mux.HandleFunc("/healthz", s.handleHealth)
        mux.HandleFunc("/help", s.handleHelpPage)
        mux.HandleFunc("/", s.handleRoot)
        return mux
}

// handleSearch 搜索入口,支持 GET 与 POST(表单或 JSON)。
// 默认禁用:仅当环境变量 ENABLE_SEARXNG 开启时可用(见 searxngEnabled)。
func (s *server) handleSearch(w http.ResponseWriter, r *http.Request) {
        if !searxngEnabled() {
                writeJSON(w, http.StatusForbidden, ErrorResponse{
                        Error: "SearXNG 兼容接口已禁用(默认):如需开启,设环境变量 ENABLE_SEARXNG=1 后重启服务;Bing 官方 API v7 兼容接口 /v7/search 不受影响",
                })
                return
        }
        if r.Method != http.MethodGet && r.Method != http.MethodPost {
                writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "仅支持 GET / POST"})
                return
        }

        p := paramsFromRequest(r)
        if strings.TrimSpace(p.Q) == "" {
                writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "缺少查询参数 q"})
                return
        }

        // 垂直类别归一:images/videos/news/dict(学术/购物/地图等
        // 纯客户端渲染的类别会在这里得到明确错误说明)
        category, err := normalizeCategory(p.Category)
        if err != nil {
                writeJSON(w, http.StatusBadRequest, ErrorResponse{
                        Error: err.Error(),
                        Query: p.Q,
                })
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

        // 垂直搜索分发(非网页综合)
        if category != "" {
                resp, err := s.searchVertical(ctx, p, category, market)
                if err != nil {
                        log.Printf("垂直搜索失败 cat=%q q=%q lang=%q: %v", category, p.Q, market, err)
                        writeJSON(w, http.StatusBadGateway, ErrorResponse{
                                Error: "Bing 查询失败: " + err.Error(),
                                Query: p.Q,
                        })
                        return
                }
                if market == "" {
                        resp.Language = "all"
                } else {
                        resp.Language = market
                }
                writeJSON(w, http.StatusOK, resp)
                return
        }

        // 网页综合搜索:多页聚合(与 v7 同源引擎),
        // offset = (page-1)*count → 精确对齐 SearXNG 页语义,
        // count>10 时自动跨 SERP 页补齐,不再出现“页间跳空”
        offset := (p.Page - 1) * p.Count
        results, suggestions, _, err := s.engine.SearchPaged(ctx, PagedQuery{
                Term:     p.Q,
                Language: market,
                Offset:   offset,
                Count:    p.Count,
        })
        if err != nil {
                // 翻页被 Bing 风控拦截:明确报错(不再静默返回第 1 页内容冒充后续页)
                if errors.Is(err, ErrPagingBlocked) {
                        log.Printf("搜索翻页被风控拦截 q=%q page=%d: %v", p.Q, p.Page, err)
                        writeJSON(w, http.StatusBadGateway, ErrorResponse{
                                Error: err.Error(),
                                Query: p.Q,
                        })
                        return
                }
                log.Printf("搜索失败 q=%q lang=%q: %v", p.Q, market, err)
                writeJSON(w, http.StatusBadGateway, ErrorResponse{
                        Error: "Bing 查询失败: " + err.Error(),
                        Query: p.Q,
                })
                return
        }
        resp := &SearchResponse{
                Query:               p.Q,
                NumberOfResults:     len(results),
                Results:             results,
                Answers:             []string{},
                Corrections:         []string{},
                Infoboxes:           []string{},
                Suggestions:         suggestions,
                UnresponsiveEngines: []string{},
        }
        if suggestions == nil {
                resp.Suggestions = []string{}
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

// paramsFromRequest 兼容三种传参方式:URL 查询串、POST 表单、POST JSON
func paramsFromRequest(r *http.Request) searchParams {
        p := searchParams{Count: 10, Page: 1}

        // POST + JSON
        if r.Method == http.MethodPost && strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
                var body struct {
                        Q          string `json:"q"`
                        Count      int    `json:"count"`
                        Page       int    `json:"page"`
                        Language   string `json:"language"`
                        Category   string `json:"category"`
                        Categories string `json:"categories"`
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
                        p.Category = pickCategory(body.Category, body.Categories)
                        return clamp(p)
                }
                // JSON 解析失败则继续按表单/查询串处理
        }

        // URL 查询串 / POST 表单(FormValue 会自动合并两者)
        p.Q = r.FormValue("q")
        p.Language = r.FormValue("language")
        p.Category = pickCategory(r.FormValue("category"), r.FormValue("categories"))
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

// withCORS 开放跨域,方便浏览器前端直接调用;
// 放行 Ocp-Apim-Subscription-Key / X-MSEdge-ClientID 头供 v7 兼容客户端预检
func withCORS(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.Header().Set("Access-Control-Allow-Origin", "*")
                w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
                w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Ocp-Apim-Subscription-Key, X-MSEdge-ClientID")
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
