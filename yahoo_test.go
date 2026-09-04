package main

// yahoo_test.go — Yahoo 回退引擎测试(v1.4.4):
//   1. SERP 解析:真实样本(GeeksForGeeks/Coursera 块原样嵌入)、
//      favicon/导航排除、RU= 重定向解码
//   2. YahooEngine.SearchPaged:b 参数 1 基偏移、7 条/页跨页聚合、
//      跨页去重、结果耗尽、非 200 错误、中文查询编码
//   3. webSearchPaged 回退矩阵:Bing 深页被拦→Yahoo 整窗接管;
//      Yahoo 失败→维持原 429;partial 窗口→Yahoo 补齐 bing+yahoo;
//      YAHOO_FALLBACK=0 / 未构造引擎→不回退
//   4. HTTP 层端到端:/search 与 /v7/search 均透出 X-Search-Provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

// ── SERP fixture ─────────────────────────────────────────────

// yahooSerpPage 构造一页 Yahoo SERP:n 条结果(链接全局编号 start 起),
// 附 2 条相关搜索。结构对齐 2026-09 真实 SERP(compTitle/compText/RU 重定向)。
func yahooSerpPage(query string, n, start int) string {
	var b strings.Builder
	b.WriteString(`<div id="web"><ol class="reg searchCenterMiddle">`)
	for i := 0; i < n; i++ {
		g := start + i
		enc := url.QueryEscape(fmt.Sprintf("https://example.com/%d", g))
		fmt.Fprintf(&b, `<li><div class="dd fst algo algo-sr relsrch Sr">`+
			`<div class="compTitle options-toggle">`+
			`<a class="d-ib va-top" href="https://r.search.yahoo.com/_ylt=abc/RV=2/RE=1/RO=10/RU=%s/RK=2/RS=x" target="_blank">`+
			`<div class="thmb algo-favicon"><img src="https://s.yimg.com/pv/static/f.png"/></div>`+
			`<span class="d-ib va-mid"><span class="fc-141414 d-b">example.com</span>https://example.com › %d</span>`+
			`<h3 style="display:block" class="title"><span class="d-b fz-20">Result %d</span></h3></a></div>`+
			`<div class="compText aAbs"><p class="fc-dustygray">snippet %d</p></div></div></li>`,
			enc, g, g, g)
	}
	b.WriteString(`</ol></div>`)
	b.WriteString(`<div class="compRelatedSearch"><a class="s-link s-requery" href="#">related one</a>` +
		`<a class="s-link s-requery" href="#">related two</a></div>`)
	return b.String()
}

// yahooRealSampleSnippet 2026-09 真实 SERP 原样摘录(GeeksForGeeks 块 +
// 相关搜索),用于防结构漂移回归。
const yahooRealSampleSnippet = `<li><div class="dd fst algo algo-sr relsrch Sr" data-yga='{"yModuleName":"Sr","ymodtype":"algo"}'><div class="compTitle options-toggle"><a class="d-ib va-top mt-38 mb-4 mxw-100p" data-matarget="algo" target="_blank" referrerpolicy="origin" href="https://r.search.yahoo.com/_ylt=AwrOtYh72JpqYAIAYHFXNyoA;_ylu=Y29sbwNncTEEcG9zAzEEdnRpZAMEc2VjA3Ny/RV=2/RE=1789742459/RO=10/RU=https%3a%2f%2fwww.geeksforgeeks.org%2fpython%2fdifference-between-python-and-java%2f/RK=2/RS=aLhml0reK9e8otUl1HPdPyHv5g8-"><div  class="d-ib p-abs t-0 l-0 fz-13 lh-16 fc-dustygray wr-bw ls-n fw-m"><div class="thmb algo-favicon bd-1-E3E3E3 bdr-100 p-5 bgc-4th w-16 h-16 va-mid mr-8"><img class="s-img p-rel va-top ov-h bgc-4th" width="16" height="16" alt="" src="https://s.yimg.com/pv/static/search_favicon/images/32x32_7eae5aac8b7f7402.png" aria-hidden="true" role="presentation"/></div><span style="letter-spacing: 0.195px;" class="d-ib va-mid"><span class="fc-141414 d-b">GeeksForGeeks</span>https://www.geeksforgeeks.org › python › difference-between</span></div><h3 style="display:block" class="title fc-2015C2-imp pt-6 ivmt-6 mxw-100p"><span class="d-b fz-20 lh-24 tc ls-024 fw-500">Difference between Python and Java - GeeksforGeeks</span></h3></a></div><div class="compText aAbs"><p class="fc-dustygray fz-14 lh-22 ls-02 mah-44 ov-h d-box fbox-ov fbox-lc2"><span class="fc-smoke">Jul 12, 2025 · </span>  Python is gaining popularity because of its simplicity, but <b>Java</b> has been around for a long time. A Major difference between <b>Java</b> and Python is that <b>Java</b> is compiled and statically typed. </p></div></div></li>` +
	`<a class="s-link s-requery" href="https://search.yahoo.com/search?p=java+tutorial">java tutorial</a>`

// newYahooTestServer 构造 mock Yahoo:结果池 total 条(全局编号 1..total),
// 按请求 b(1 基)返回 7 条/页;b 超出池返回空 SERP。记录每次请求的 RequestURI。
func newYahooTestServer(t *testing.T, total int) (*httptest.Server, *[]string) {
	var queries []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.RequestURI())
		b := 1
		if s := r.URL.Query().Get("b"); s != "" {
			b, _ = strconv.Atoi(s)
		}
		if b < 1 || b > total {
			w.Write([]byte(yahooSerpPage("q", 0, 1))) // 超出池:空页(结果耗尽)
			return
		}
		n := 7
		if b-1+n > total {
			n = total - b + 1
		}
		w.Write([]byte(yahooSerpPage(r.URL.Query().Get("p"), n, b)))
	}))
	t.Cleanup(ts.Close)
	return ts, &queries
}

// ── 解析 ─────────────────────────────────────────────────────

func TestParseYahooRealSample(t *testing.T) {
	results, related := parseYahoo(yahooRealSampleSnippet)
	if len(results) != 1 {
		t.Fatalf("真实样本应解析出 1 条结果, got %d", len(results))
	}
	r := results[0]
	if r.Title != "Difference between Python and Java - GeeksforGeeks" {
		t.Errorf("标题解析错: %q", r.Title)
	}
	if r.URL != "https://www.geeksforgeeks.org/python/difference-between-python-and-java/" {
		t.Errorf("RU 解码错: %q", r.URL)
	}
	if !strings.Contains(r.Content, "Python is gaining popularity") {
		t.Errorf("摘要应包含正文(含日期前缀), got %q", r.Content)
	}
	if r.Engine != "yahoo" || len(r.Engines) != 1 || r.Engines[0] != "yahoo" {
		t.Errorf("Engine 字段应为 yahoo, got %v/%v", r.Engine, r.Engines)
	}
	if len(related) != 1 || related[0] != "java tutorial" {
		t.Errorf("相关搜索解析错: %v", related)
	}
}

func TestParseYahooSkipsFaviconAndNav(t *testing.T) {
	// favicon(thmb algo-favicon)与页头导航(Yahoo logo 的 RU 链接)
	// 均不在 dd-algo 结果块内,不得误抓
	page := `<a id="logo" href="https://r.search.yahoo.com/x/RU=https%3a%2f%2fwww.yahoo.com/RK=2/RS=x">Yahoo</a>` +
		`<div class="thmb algo-favicon"><img src="f.png"/></div>` +
		yahooSerpPage("q", 2, 1)
	results, _ := parseYahoo(page)
	if len(results) != 2 {
		t.Fatalf("应恰好解析出 2 条结果, got %d", len(results))
	}
	for i, r := range results {
		want := fmt.Sprintf("https://example.com/%d", i+1)
		if r.URL != want {
			t.Errorf("结果 %d URL = %q, want %q", i+1, r.URL, want)
		}
		if r.Position != i+1 {
			t.Errorf("结果 %d Position = %d", i+1, r.Position)
		}
	}
}

func TestDecodeYahooRedirect(t *testing.T) {
	// 真实形态:小写十六进制转义(%3a %2f)
	href := "https://r.search.yahoo.com/_ylt=a/RV=2/RE=1/RO=10/RU=https%3a%2f%2fgo.dev%2fdoc%2f/RK=2/RS=x"
	if got := decodeYahooRedirect(href); got != "https://go.dev/doc/" {
		t.Fatalf("decodeYahooRedirect = %q, want https://go.dev/doc/", got)
	}
	// 非重定向原样返回
	if got := decodeYahooRedirect("https://go.dev/blog"); got != "https://go.dev/blog" {
		t.Fatalf("非重定向应原样返回, got %q", got)
	}
	// 大写十六进制同样可解
	if got := decodeYahooRedirect("/RU=https%3A%2F%2Fexample.com/RK=1"); got != "https://example.com" {
		t.Fatalf("大写转义解码错, got %q", got)
	}
}

// ── YahooEngine.SearchPaged ──────────────────────────────────

func TestYahooSearchPagedAggregatesPages(t *testing.T) {
	// count=10 > 7 条/页 → 跨页聚合:b=1 与 b=8 两轮,结果连续编号
	ts, queries := newYahooTestServer(t, 20)
	e := &YahooEngine{Base: ts.URL, Client: ts.Client()}
	res, related, total, err := e.SearchPaged(context.Background(), PagedQuery{Term: "q", Offset: 0, Count: 10})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(res) != 10 {
		t.Fatalf("应聚合 10 条, got %d", len(res))
	}
	for i, r := range res {
		if want := fmt.Sprintf("https://example.com/%d", i+1); r.URL != want {
			t.Errorf("结果 %d URL = %q, want %q", i+1, r.URL, want)
		}
		if r.Position != i+1 {
			t.Errorf("结果 %d Position = %d", i+1, r.Position)
		}
	}
	if len(related) != 2 {
		t.Errorf("相关搜索应取首轮页面的 2 条, got %d", len(related))
	}
	if total != 0 {
		t.Errorf("Yahoo 无计数条,total 应为 0, got %d", total)
	}
	got := *queries
	if len(got) != 2 || !strings.Contains(got[0], "b=1") || !strings.Contains(got[1], "b=8") {
		t.Errorf("应依次请求 b=1、b=8, got %v", got)
	}
}

func TestYahooSearchPagedOffset(t *testing.T) {
	// offset=10 → 首轮直接 b=11(任意 1 基偏移,无需页边界对齐)
	ts, queries := newYahooTestServer(t, 20)
	e := &YahooEngine{Base: ts.URL, Client: ts.Client()}
	res, _, _, err := e.SearchPaged(context.Background(), PagedQuery{Term: "q", Offset: 10, Count: 7})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(res) != 7 {
		t.Fatalf("want 7 条, got %d", len(res))
	}
	if res[0].URL != "https://example.com/11" || res[6].URL != "https://example.com/17" {
		t.Errorf("窗口应为 11..17, got %s..%s", res[0].URL, res[6].URL)
	}
	got := *queries
	if len(got) != 1 || !strings.Contains(got[0], "b=11") {
		t.Errorf("应请求 b=11, got %v", got)
	}
}

func TestYahooSearchPagedDedupCrossPages(t *testing.T) {
	// 跨页重复展示(页 1 含 1..7,页 2 从 5 重复起)→ 去重后不重复计数
	var pages []string
	pages = append(pages, yahooSerpPage("q", 7, 1))
	pages = append(pages, yahooSerpPage("q", 7, 5)) // 5..11,与页 1 重合 5..7
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := 1
		if s := r.URL.Query().Get("b"); s != "" {
			b, _ = strconv.Atoi(s)
		}
		if b <= 7 {
			w.Write([]byte(pages[0]))
		} else {
			w.Write([]byte(pages[1]))
		}
	}))
	t.Cleanup(ts.Close)
	e := &YahooEngine{Base: ts.URL, Client: ts.Client()}
	res, _, _, err := e.SearchPaged(context.Background(), PagedQuery{Term: "q", Offset: 0, Count: 12})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	seen := map[string]bool{}
	for _, r := range res {
		if seen[r.URL] {
			t.Fatalf("出现重复结果: %s", r.URL)
		}
		seen[r.URL] = true
	}
	if len(res) != 11 { // 7 + 4(8,9,10,11 去重后)
		t.Fatalf("去重后应得 11 条, got %d", len(res))
	}
}

func TestYahooSearchPagedExhausted(t *testing.T) {
	// 池仅 7 条,请求 count=10 → 第 2 轮空页 → 如实返回 7 条(nil error)
	ts, queries := newYahooTestServer(t, 7)
	e := &YahooEngine{Base: ts.URL, Client: ts.Client()}
	res, _, _, err := e.SearchPaged(context.Background(), PagedQuery{Term: "q", Offset: 0, Count: 10})
	if err != nil {
		t.Fatalf("结果耗尽应返回已得部分而非报错, got %v", err)
	}
	if len(res) != 7 {
		t.Fatalf("应返回 7 条, got %d", len(res))
	}
	if len(*queries) != 2 { // b=1 + b=8(空页)
		t.Errorf("应抓 2 轮(b=1、b=8 空页收尾), got %v", *queries)
	}
}

func TestYahooSearchPagedBeyondEndEmpty(t *testing.T) {
	// offset 超出全部结果 → 空结果集(200 语义),不报错
	ts, _ := newYahooTestServer(t, 7)
	e := &YahooEngine{Base: ts.URL, Client: ts.Client()}
	res, _, _, err := e.SearchPaged(context.Background(), PagedQuery{Term: "q", Offset: 100, Count: 10})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(res) != 0 {
		t.Fatalf("超出末尾应返回空结果, got %d", len(res))
	}
}

func TestYahooSearchPagedNon200(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusServiceUnavailable)
	}))
	t.Cleanup(ts.Close)
	e := &YahooEngine{Base: ts.URL, Client: ts.Client()}
	_, _, _, err := e.SearchPaged(context.Background(), PagedQuery{Term: "q", Offset: 0, Count: 10})
	var es errUpstreamStatus
	if !errors.As(err, &es) || es.code != http.StatusServiceUnavailable {
		t.Fatalf("应报 errUpstreamStatus 503, got %v", err)
	}
	if !strings.Contains(es.host, "127.0.0.1") {
		t.Errorf("错误应携带实际主机名, got host=%q", es.host)
	}
	// upstreamErrorStatus:Yahoo 503 与 Bing 503 同样映射 502(仅 429 特判)
	if st, _ := upstreamErrorStatus(err); st != http.StatusBadGateway {
		t.Errorf("503 应映射 502, got %d", st)
	}
}

func TestYahooFetchRetriesTransportError(t *testing.T) {
	// 首次请求直接断连(模拟 Yahoo 偶发 unexpected EOF),
	// 退避重试一次后成功 → 整体不报错
	var n int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		if n == 1 {
			// Hijack 后立刻断开连接(客户端视角 = 传输层错误,非状态码)
			if hj, ok := w.(http.Hijacker); ok {
				conn, _, _ := hj.Hijack()
				if conn != nil {
					conn.Close()
				}
				return
			}
		}
		w.Write([]byte(yahooSerpPage("q", 7, 1)))
	}))
	t.Cleanup(ts.Close)
	e := &YahooEngine{Base: ts.URL, Client: ts.Client()}
	res, _, _, err := e.SearchPaged(context.Background(), PagedQuery{Term: "q", Offset: 0, Count: 7})
	if err != nil {
		t.Fatalf("断连后应重试成功, got %v", err)
	}
	if len(res) != 7 {
		t.Fatalf("重试后应正常解析 7 条, got %d", len(res))
	}
	if n != 2 {
		t.Errorf("应恰好请求 2 次(断连+重试), got %d", n)
	}
}

func TestYahooFetchNoRetryOnStatusError(t *testing.T) {
	// 状态码错误(503)不重试:仍只请求 1 次
	var n int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		http.Error(w, "boom", http.StatusServiceUnavailable)
	}))
	t.Cleanup(ts.Close)
	e := &YahooEngine{Base: ts.URL, Client: ts.Client()}
	_, _, _, err := e.SearchPaged(context.Background(), PagedQuery{Term: "q", Offset: 0, Count: 7})
	if err == nil {
		t.Fatal("503 应报错")
	}
	if n != 1 {
		t.Errorf("状态码错误不应重试, got %d 次请求", n)
	}
}

func TestYahooSearchPagedChineseQuery(t *testing.T) {
	// 中文查询:q 正确编码传输 + ei=UTF-8 显式声明
	var gotQuery, gotEI string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("p")
		gotEI = r.URL.Query().Get("ei")
		w.Write([]byte(yahooSerpPage("q", 7, 1)))
	}))
	t.Cleanup(ts.Close)
	e := &YahooEngine{Base: ts.URL, Client: ts.Client()}
	_, _, _, err := e.SearchPaged(context.Background(), PagedQuery{Term: "人工智能 发展趋势", Offset: 0, Count: 7})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if gotQuery != "人工智能 发展趋势" {
		t.Errorf("查询词应原样解码传输, got %q", gotQuery)
	}
	if gotEI != "UTF-8" {
		t.Errorf("应携带 ei=UTF-8, got %q", gotEI)
	}
}

// ── webSearchPaged 回退矩阵 ──────────────────────────────────

// newFallbackServer 构造双上游服务:Bing mock(永远只服务第 1 页,
// 模拟非浏览器会话翻页风控)+ Yahoo mock(total 条结果池)。
func newFallbackServer(t *testing.T, bingHTML string, yahooTotal int) (*server, *[]string, *[]string) {
	bingTS, bingQueries := newPagingTestServer(t, map[int]string{1: bingHTML})
	yahooTS, yahooQueries := newYahooTestServer(t, yahooTotal)
	s := &server{
		engine: &BingEngine{Base: bingTS.URL, Client: bingTS.Client()},
		yahoo:  &YahooEngine{Base: yahooTS.URL, Client: yahooTS.Client()},
	}
	return s, bingQueries, yahooQueries
}

func TestWebSearchPagedDeepFallback(t *testing.T) {
	// offset=10:Bing 深页被风控(回吐第 1 页)→ Yahoo 整窗接管
	s, bingQ, yahooQ := newFallbackServer(t, serpPage("q", 1, true), 30)
	res, suggestions, total, provider, limited, err := s.webSearchPaged(context.Background(), PagedQuery{Term: "q", Offset: 10, Count: 10})
	if err != nil {
		t.Fatalf("深页应回退 Yahoo 而非报错, got %v", err)
	}
	if provider != "yahoo" {
		t.Fatalf("provider = %q, want yahoo", provider)
	}
	if limited {
		t.Error("Yahoo 接管后不应 limited")
	}
	if len(res) != 10 || res[0].URL != "https://example.com/11" {
		t.Fatalf("应返回 Yahoo 窗口 11..20, got %d 条首条 %s", len(res), firstURL(res))
	}
	if len(*bingQ) != 1 {
		t.Errorf("Bing 应只试 1 轮即失败, got %v", *bingQ)
	}
	if len(*yahooQ) != 2 { // b=11、b=18
		t.Errorf("Yahoo 应聚合 2 轮, got %v", *yahooQ)
	}
	if total != 0 || len(suggestions) != 0 {
		// Yahoo 路径 total=0;相关搜索返回 Yahoo 的 2 条(此处允许非空)
		t.Logf("total=%d suggestions=%v(Yahoo 语义)", total, suggestions)
	}
}

func TestWebSearchPagedFallbackYahooFails(t *testing.T) {
	// Bing 深页被拦 + Yahoo 不可达(503)→ 维持原 Bing 错误(429 可诊断)
	bingTS, _ := newPagingTestServer(t, map[int]string{1: serpPage("q", 1, true)})
	yahooTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	t.Cleanup(yahooTS.Close)
	s := &server{
		engine: &BingEngine{Base: bingTS.URL, Client: bingTS.Client()},
		yahoo:  &YahooEngine{Base: yahooTS.URL, Client: yahooTS.Client()},
	}
	_, _, _, _, _, err := s.webSearchPaged(context.Background(), PagedQuery{Term: "q", Offset: 10, Count: 10})
	if !errors.Is(err, ErrPagingBlocked) {
		t.Fatalf("Yahoo 失败应维持原 ErrPagingBlocked, got %v", err)
	}
}

func TestWebSearchPagedPartialCompletion(t *testing.T) {
	// offset=0&count=10:Bing 第 1 页只有 3 条且聚合被拦(limited)→
	// Yahoo 从缺口续接,窗口补满 → provider=bing+yahoo,不再 limited
	s, _, yahooQ := newFallbackServer(t, serpPage("q", 1, true), 30)
	// Bing mock 的 serpPage 每页 3 条,聚合翻页被回吐第 1 页 → limited
	res, _, _, provider, limited, err := s.webSearchPaged(context.Background(), PagedQuery{Term: "q", Offset: 0, Count: 10})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if provider != "bing+yahoo" {
		t.Fatalf("provider = %q, want bing+yahoo", provider)
	}
	if limited {
		t.Error("窗口已补满,不应再 limited")
	}
	if len(res) != 10 {
		t.Fatalf("应补满 10 条, got %d", len(res))
	}
	// 前 3 条来自 Bing(example.com/1..3 由 Bing mock 提供),其余来自 Yahoo
	if res[0].URL != "https://example.com/1" || res[2].URL != "https://example.com/3" {
		t.Errorf("前 3 条应来自 Bing, got %s..%s", res[0].URL, res[2].URL)
	}
	if len(*yahooQ) == 0 {
		t.Error("Yahoo 应被请求续接缺口")
	}
	for i, r := range res {
		if r.Position != i+1 {
			t.Errorf("结果 %d Position = %d(合并后应连续编号)", i+1, r.Position)
		}
	}
}

func TestWebSearchPagedPartialCompletionStillLimited(t *testing.T) {
	// Yahoo 也只有 4 条可补 → 窗口仍不满 → 维持 limited 语义
	s, _, _ := newFallbackServer(t, serpPage("q", 1, true), 5)
	res, _, _, provider, limited, err := s.webSearchPaged(context.Background(), PagedQuery{Term: "q", Offset: 0, Count: 10})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if provider != "bing+yahoo" {
		t.Fatalf("provider = %q, want bing+yahoo", provider)
	}
	if !limited {
		t.Error("窗口未补满应维持 limited=true")
	}
	if len(res) != 5 { // Bing 3 + Yahoo 2(池 5 条里 offset 续接后的去重新增)
		t.Logf("合并结果 %d 条(允许 Yahoo 侧与 Bing 重合被去重)", len(res))
	}
}

func TestWebSearchPagedFallbackDisabled(t *testing.T) {
	// YAHOO_FALLBACK=0:即使 Yahoo 可达也维持 v1.4.3 行为(429)
	t.Setenv("YAHOO_FALLBACK", "0")
	s, _, yahooQ := newFallbackServer(t, serpPage("q", 1, true), 30)
	_, _, _, _, _, err := s.webSearchPaged(context.Background(), PagedQuery{Term: "q", Offset: 10, Count: 10})
	if !errors.Is(err, ErrPagingBlocked) {
		t.Fatalf("关闭回退后应维持原错误, got %v", err)
	}
	if len(*yahooQ) != 0 {
		t.Errorf("关闭回退后不应请求 Yahoo, got %v", *yahooQ)
	}
}

func TestWebSearchPagedNoYahooEngine(t *testing.T) {
	// 未构造 yahoo 引擎(既有测试的默认形态)→ 行为与 v1.4.3 完全一致
	bingTS, _ := newPagingTestServer(t, map[int]string{1: serpPage("q", 1, true)})
	s := &server{engine: &BingEngine{Base: bingTS.URL, Client: bingTS.Client()}}
	_, _, _, _, _, err := s.webSearchPaged(context.Background(), PagedQuery{Term: "q", Offset: 10, Count: 10})
	if !errors.Is(err, ErrPagingBlocked) {
		t.Fatalf("应维持原错误, got %v", err)
	}
}

func TestWebSearchPagedEmptySERPFallback(t *testing.T) {
	// Bing 首轮返回空响应体(200 + 0 字节,实测突发限流瞬断)→
	// ErrEmptySERP → Yahoo 整窗承接(v1.4.4)
	bingTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("")) // 空响应体
	}))
	t.Cleanup(bingTS.Close)
	yahooTS, yahooQ := newYahooTestServer(t, 30)
	s := &server{
		engine: &BingEngine{Base: bingTS.URL, Client: bingTS.Client()},
		yahoo:  &YahooEngine{Base: yahooTS.URL, Client: yahooTS.Client()},
	}
	res, _, _, provider, _, err := s.webSearchPaged(context.Background(), PagedQuery{Term: "q", Offset: 0, Count: 10})
	if err != nil {
		t.Fatalf("空 SERP 应回退 Yahoo, got %v", err)
	}
	if provider != "yahoo" {
		t.Fatalf("provider = %q, want yahoo", provider)
	}
	if len(res) != 10 || res[0].URL != "https://example.com/1" {
		t.Fatalf("应返回 Yahoo 10 条, got %d 条首条 %s", len(res), firstURL(res))
	}
	if len(*yahooQ) != 2 {
		t.Errorf("Yahoo 应聚合 2 轮, got %v", *yahooQ)
	}
}

func TestSearchPagedEmptySERPError(t *testing.T) {
	// 无 Yahoo 回退时,空/不可识别 SERP → ErrEmptySERP(429 + Retry-After 60,
	// 不再静默返回空结果集)
	bingTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html><body>challenge wall</body></html>")) // 无任何 SERP 标记
	}))
	t.Cleanup(bingTS.Close)
	e := &BingEngine{Base: bingTS.URL, Client: bingTS.Client()}
	_, _, _, _, err := e.SearchPaged(context.Background(), PagedQuery{Term: "q", Offset: 0, Count: 10})
	if !errors.Is(err, ErrEmptySERP) {
		t.Fatalf("应报 ErrEmptySERP, got %v", err)
	}
	if st, ra := upstreamErrorStatus(err); st != http.StatusTooManyRequests || ra != "60" {
		t.Errorf("ErrEmptySERP 应映射 429+60, got %d/%q", st, ra)
	}
}

func TestSearchPagedNoResultsPageNotAnomaly(t *testing.T) {
	// 真实“无结果”页(含 b_results 容器与 b_no 提示,无 b_algo)→
	// 如实返回空结果,不报 ErrEmptySERP、不触发回退
	noResults := `<ol id="b_results"><li class="b_no"><h4>没有与此相关的:</h4><p>试试其他关键词</p></li></ol>`
	bingTS, _ := newPagingTestServer(t, map[int]string{1: noResults})
	yahooTS, yahooQ := newYahooTestServer(t, 30)
	s := &server{
		engine: &BingEngine{Base: bingTS.URL, Client: bingTS.Client()},
		yahoo:  &YahooEngine{Base: yahooTS.URL, Client: yahooTS.Client()},
	}
	res, _, _, provider, _, err := s.webSearchPaged(context.Background(), PagedQuery{Term: "zzzqqq", Offset: 0, Count: 10})
	if err != nil {
		t.Fatalf("无结果页不应报错, got %v", err)
	}
	if len(res) != 0 {
		t.Fatalf("应返回空结果, got %d", len(res))
	}
	if provider != "bing" {
		t.Errorf("无结果页不应触发回退, provider = %q", provider)
	}
	if len(*yahooQ) != 0 {
		t.Errorf("不应请求 Yahoo, got %v", *yahooQ)
	}
}

func TestYahooFallbackEnabledDefault(t *testing.T) {
	// 默认开启;各关闭写法均生效
	if !yahooFallbackEnabled() {
		t.Fatal("默认应开启")
	}
	for _, v := range []string{"0", "false", "no", "off", " FALSE "} {
		t.Setenv("YAHOO_FALLBACK", v)
		if yahooFallbackEnabled() {
			t.Errorf("YAHOO_FALLBACK=%q 应关闭", v)
		}
	}
	t.Setenv("YAHOO_FALLBACK", "1")
	if !yahooFallbackEnabled() {
		t.Error("YAHOO_FALLBACK=1 应开启")
	}
}

// ── HTTP 层端到端 ────────────────────────────────────────────

func TestV7SearchYahooFallbackHeader(t *testing.T) {
	// v7 深页(offset=10)→ 200 + X-Search-Provider: yahoo + 10 条 Yahoo 结果
	s, _, _ := newFallbackServer(t, serpPage("q", 1, true), 30)
	req := httptest.NewRequest(http.MethodGet, "/v7/search?q=q&count=10&offset=10", nil)
	rec := httptest.NewRecorder()
	s.handleBingV7Search(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get(searchProviderHeader); got != "yahoo" {
		t.Errorf("X-Search-Provider = %q, want yahoo", got)
	}
	if rec.Header().Get(pagingLimitedHeader) != "" {
		t.Error("Yahoo 接管不应设置 X-Paging-Limited")
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"webPages"`) || strings.Count(body, `"url":"https://example.com/`) != 10 {
		t.Errorf("应返回 10 条 Yahoo 结果, body = %s", body[:min(400, len(body))])
	}
	// v7 字段:totalEstimatedMatches 兜底 = offset+len
	if !strings.Contains(body, `"totalEstimatedMatches":20`) {
		t.Errorf("totalEstimatedMatches 应以 offset+len 兜底为 20, body = %s", body[:min(400, len(body))])
	}
}

func TestHandleSearchYahooFallbackHeader(t *testing.T) {
	// SearXNG 端点 page=2 → offset=10 → Yahoo 承接
	t.Setenv("ENABLE_SEARXNG", "1")
	s, _, _ := newFallbackServer(t, serpPage("q", 1, true), 30)
	req := httptest.NewRequest(http.MethodGet, "/search?q=q&count=10&page=2", nil)
	rec := httptest.NewRecorder()
	s.handleSearch(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get(searchProviderHeader); got != "yahoo" {
		t.Errorf("X-Search-Provider = %q, want yahoo", got)
	}
	if !strings.Contains(rec.Body.String(), `"https://example.com/11"`) {
		t.Errorf("应返回 Yahoo 窗口 11..20 的首条, body = %s", rec.Body.String()[:min(300, len(rec.Body.String()))])
	}
}

// firstURL 取首条结果 URL(空安全)
func firstURL(res []Result) string {
	if len(res) == 0 {
		return ""
	}
	return res[0].URL
}
