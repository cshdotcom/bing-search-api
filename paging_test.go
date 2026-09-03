package main

// paging_test.go — 翻页机制测试(v1.4.1 修复):
//   1. SERP 翻页导航解析:当前页标记(sb_pagS)、下一页链接(FORM=PERE)
//   2. SearchPaged:0 基 first 对齐、跟随 Bing 自身翻页链接、
//      跨页去重、offset 精确切片、count 跨页聚合
//   3. 翻页校验:请求页 ≥2 却被服务到第 1 页 → ErrPagingBlocked
//   4. handleSearch(SearXNG 网页路径)count>10 跨页聚合,页间不跳空

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// serpPage 构造一页 SERP fixture:3 条结果(链接含页内序号)+ 当前页标记 + 下一页链接。
// 结果块刻意用最简形态:parseBing 只要求 b_algo + h2>a。
func serpPage(query string, pageNum int, withNext bool) string {
	var b strings.Builder
	b.WriteString(`<ol id="b_results">`)
	for i := 1; i <= 3; i++ {
		global := (pageNum-1)*3 + i
		fmt.Fprintf(&b, `<li class="b_algo"><h2><a href="https://example.com/%d">Result %d</a></h2>`+
			`<div class="b_caption"><p>snippet %d</p></div></li>`, global, global, global)
	}
	b.WriteString(`</ol>`)
	b.WriteString(`<span class="sb_count">About 9,000 results</span>`)

	// 翻页导航:当前页 sb_pagS(无 href)+ 后续页链接(first=0 基、10 的倍数)
	b.WriteString(`<li class="b_pag"><nav role="navigation"><ul class="sb_pagF">`)
	fmt.Fprintf(&b, `<li><a aria-label="Page %d" class="sb_pagS sb_pagS_bp b_widePag sb_bp sb_pag_first">%d</a></li>`, pageNum, pageNum)
	if withNext {
		next := pageNum * 10 // Bing 语义:第 N+1 页 first = N*10(0 基)
		fmt.Fprintf(&b, `<li><a class="b_widePag sb_bp" aria-label="Page %d" href="/search?count=10&amp;q=%s&amp;FPIG=TESTFPIG&amp;first=%d&amp;FORM=PERE">%d</a></li>`,
			pageNum+1, query, next, pageNum+1)
	}
	b.WriteString(`</ul></nav></li>`)
	return b.String()
}

// page1OnlyPage 无翻页导航的 SERP(末页/无 b_pag)
func page1OnlyPage() string {
	return `<ol id="b_results">` +
		`<li class="b_algo"><h2><a href="https://example.com/1">Only Result</a></h2></li>` +
		`</ol><span class="sb_count">About 5 results</span>`
}

// newPagingTestServer 构造 mock Bing:按请求里的 first 参数返回对应 SERP 页。
// pages: pageNum → HTML;无 first 视为第 1 页。记录每次请求的查询串。
func newPagingTestServer(t *testing.T, pages map[int]string) (*httptest.Server, *[]string) {
	var queries []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.RequestURI())
		first := 0
		if s := r.URL.Query().Get("first"); s != "" {
			first, _ = strconv.Atoi(s)
		}
		// mock 语义与 Bing 实测一致:当前页 = first/10 + 1(first 0 基)
		pageNum := first/10 + 1
		page, ok := pages[pageNum]
		if !ok {
			// 请求了不存在的页 → 返回第 1 页(模拟 Bing 风控忽略 first)
			page = pages[1]
		}
		w.Write([]byte(page))
	}))
	t.Cleanup(ts.Close)
	return ts, &queries
}

// ── 导航解析 ──────────────────────────────────────────────

func TestParseServedPageNum(t *testing.T) {
	page := serpPage("golang", 3, true)
	if got := parseServedPageNum(page); got != 3 {
		t.Fatalf("parseServedPageNum = %d, want 3", got)
	}
	// 无翻页导航 → 0
	if got := parseServedPageNum(page1OnlyPage()); got != 0 {
		t.Fatalf("无导航应返回 0, got %d", got)
	}
	// aria-label 缺失时用可见文本兜底
	plain := `<li class="b_pag"><nav><ul class="sb_pagF">` +
		`<li><a class="sb_pagS sb_bp sb_pag_first">4</a></li></ul></nav></li>`
	if got := parseServedPageNum(plain); got != 4 {
		t.Fatalf("文本页码兜底应返回 4, got %d", got)
	}
	// 真实 Bing 页面形态(aria-label 在 class 之前)
	real := `<nav role="navigation" aria-label="More results"><ul class="sb_pagF">` +
		`<li><a aria-label="Page 1" class="sb_pagS sb_pagS_bp b_widePag sb_bp sb_pag_first">1</a></li>` +
		`<li><a class="b_widePag sb_bp" aria-label="Page 2" href="/search?count=10&amp;q=golang&amp;FPIG=X&amp;first=10&amp;FORM=PERE">2</a></li>` +
		`</ul></nav>`
	if got := parseServedPageNum(real); got != 1 {
		t.Fatalf("真实形态当前页应返回 1, got %d", got)
	}
}

func TestExtractNextPageLink(t *testing.T) {
	page := serpPage("golang", 1, true)
	got := extractNextPageLink(page)
	want := "/search?count=10&q=golang&FPIG=TESTFPIG&first=10&FORM=PERE"
	// 提取后 &amp; 应还原为 &(fetchPageLink 内统一处理)
	if !strings.Contains(strings.ReplaceAll(got, "&amp;", "&"), want) {
		t.Fatalf("next link = %q, want 含 %q", got, want)
	}
	if got := extractNextPageLink(serpPage("golang", 9, false)); got != "" {
		t.Fatalf("末页应无下一页链接, got %q", got)
	}
	if got := extractNextPageLink(page1OnlyPage()); got != "" {
		t.Fatalf("无导航页应返回空, got %q", got)
	}
}

// ── SearchPaged:0 基 first 对齐与跨页聚合 ─────────────────

func TestSearchPagedFirstPageNoFirstParam(t *testing.T) {
	ts, queries := newPagingTestServer(t, map[int]string{
		1: serpPage("golang", 1, false),
	})
	e := &BingEngine{Base: ts.URL, Client: ts.Client()}
	res, _, _, err := e.SearchPaged(context.Background(), PagedQuery{Term: "golang", Offset: 0, Count: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 3 {
		t.Fatalf("want 3, got %d", len(res))
	}
	// 首页(offset<10)不应携带 first 参数
	if strings.Contains((*queries)[0], "first=") {
		t.Fatalf("首页请求不应带 first: %s", (*queries)[0])
	}
}

func TestSearchPagedOffsetFollowsBingNextLink(t *testing.T) {
	// 3 页 SERP,每页 3 条;offset=5 → first 对齐 0,skip=5,
	// 需聚合 2 页(6 条)后切片 [5:6+] — 同时校验 offset 精确性
	ts, queries := newPagingTestServer(t, map[int]string{
		1: serpPage("golang", 1, true),
		2: serpPage("golang", 2, true),
		3: serpPage("golang", 3, false),
	})
	e := &BingEngine{Base: ts.URL, Client: ts.Client()}
	res, _, total, err := e.SearchPaged(context.Background(), PagedQuery{Term: "golang", Offset: 5, Count: 2})
	if err != nil {
		t.Fatal(err)
	}
	if total != 9000 {
		t.Errorf("total = %d, want 9000", total)
	}
	// offset=5 → 精确取第 6、7 条(跨第 2/3 页边界)
	if len(res) != 2 || res[0].URL != "https://example.com/6" || res[1].URL != "https://example.com/7" {
		t.Fatalf("offset=5 切片应得 [6,7], got %v", urlsOf(res))
	}
	if res[0].Position != 1 || res[1].Position != 2 {
		t.Errorf("返回后 Position 应重新从 1 编号")
	}
	// 请求序:首页(不带 first)→ 第 2 页(带 first=10)
	if len(*queries) < 2 {
		t.Fatalf("应至少 2 次抓取, got %d: %v", len(*queries), *queries)
	}
	if !strings.Contains((*queries)[1], "first=10") {
		t.Fatalf("第 2 次抓取应走 Bing 翻页链接(first=10): %s", (*queries)[1])
	}
}

func TestSearchPagedDeepOffsetDirectFirst(t *testing.T) {
	// offset=20 → first 直接对齐 20(不从头走),请求第 3 页
	ts, queries := newPagingTestServer(t, map[int]string{
		1: serpPage("golang", 1, true),
		3: serpPage("golang", 3, false),
	})
	e := &BingEngine{Base: ts.URL, Client: ts.Client()}
	res, _, _, err := e.SearchPaged(context.Background(), PagedQuery{Term: "golang", Offset: 20, Count: 3})
	if err != nil {
		t.Fatal(err)
	}
	// 第 3 页结果 #7,8,9;skip=20-20=0 → 全取
	if len(res) != 3 || res[0].URL != "https://example.com/7" {
		t.Fatalf("offset=20 应直接取第 3 页, got %v", urlsOf(res))
	}
	if !strings.Contains((*queries)[0], "first=20") {
		t.Fatalf("首次抓取应带 first=20(0 基对齐): %s", (*queries)[0])
	}
}

func TestSearchPagedCountAggregation(t *testing.T) {
	// count=8 → 需聚合 3 页(每页 3 条)凑 8 条
	ts, _ := newPagingTestServer(t, map[int]string{
		1: serpPage("golang", 1, true),
		2: serpPage("golang", 2, true),
		3: serpPage("golang", 3, false),
	})
	e := &BingEngine{Base: ts.URL, Client: ts.Client()}
	res, _, _, err := e.SearchPaged(context.Background(), PagedQuery{Term: "golang", Offset: 0, Count: 8})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 8 || res[7].URL != "https://example.com/8" {
		t.Fatalf("count=8 应跨 3 页聚合 8 条, got %v", urlsOf(res))
	}
}

func TestSearchPagedDedupAcrossPages(t *testing.T) {
	// 第 2 页与第 1 页内容完全相同(模拟 Bing 重复/风控回流)→ 去重后停止
	ts, _ := newPagingTestServer(t, map[int]string{
		1: serpPage("golang", 1, true),
		2: serpPage("golang", 1, true), // 内容与第 1 页相同
	})
	e := &BingEngine{Base: ts.URL, Client: ts.Client()}
	res, _, _, err := e.SearchPaged(context.Background(), PagedQuery{Term: "golang", Offset: 0, Count: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 3 {
		t.Fatalf("重复页应被去重, want 3, got %v", urlsOf(res))
	}
}

func TestSearchPagedPagingBlocked(t *testing.T) {
	// 请求 offset=10(first=10),但服务端坚持返回第 1 页标记 → ErrPagingBlocked
	ts, _ := newPagingTestServer(t, map[int]string{
		1: serpPage("golang", 1, true),
	})
	e := &BingEngine{Base: ts.URL, Client: ts.Client()}
	_, _, _, err := e.SearchPaged(context.Background(), PagedQuery{Term: "golang", Offset: 10, Count: 10})
	if !errors.Is(err, ErrPagingBlocked) {
		t.Fatalf("应报 ErrPagingBlocked, got %v", err)
	}
}

func TestSearchPagedUnverifiablePageAccepted(t *testing.T) {
	// 页面无翻页导航(解析不到当前页)→ 无法校验,按尽力而为返回结果
	ts, _ := newPagingTestServer(t, map[int]string{
		1: page1OnlyPage(),
	})
	e := &BingEngine{Base: ts.URL, Client: ts.Client()}
	res, _, _, err := e.SearchPaged(context.Background(), PagedQuery{Term: "golang", Offset: 10, Count: 10})
	if err != nil {
		t.Fatalf("无法校验时不应报错: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("尽力而为应返回该页结果, got %d", len(res))
	}
}

// ── handleSearch(SearXNG)网页聚合 ─────────────────────────

func TestHandleSearchWebAggregatesPages(t *testing.T) {
	t.Setenv("ENABLE_SEARXNG", "1")
	ts, _ := newPagingTestServer(t, map[int]string{
		1: serpPage("golang", 1, true),
		2: serpPage("golang", 2, true),
		3: serpPage("golang", 3, false),
	})
	srv := &server{engine: &BingEngine{Base: ts.URL, Client: ts.Client()}}

	// count=8:第 1 页应聚合 8 条(旧实现只回单页 3 条)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/search?q=golang&count=8&page=1", nil)
	srv.handleSearch(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if got := strings.Count(body, `"url":"https://example.com/`); got != 8 {
		t.Fatalf("count=8 应聚合 8 条结果, got %d: %s", got, body)
	}

	// page=2&count=3:offset=3 → 精确取 #4,5,6(不跳空)
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/search?q=golang&count=3&page=2", nil)
	srv.handleSearch(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("HTTP %d: %s", w2.Code, w2.Body.String())
	}
	body2 := w2.Body.String()
	for _, u := range []string{"example.com/4", "example.com/5", "example.com/6"} {
		if !strings.Contains(body2, u) {
			t.Fatalf("page=2&count=3 应包含 %s: %s", u, body2)
		}
	}
	if strings.Contains(body2, "example.com/3") {
		t.Fatalf("page=2 不应包含上一页末条: %s", body2)
	}
}

// ── fetch 首页语义 ─────────────────────────────────────────

func TestFetchFirstParamSemantics(t *testing.T) {
	var gotQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RequestURI()
		w.Write([]byte(serpPage("golang", 1, false)))
	}))
	defer ts.Close()
	e := &BingEngine{Base: ts.URL, Client: ts.Client()}

	// First<10 → 不发送 first(与浏览器首页一致)
	if _, err := e.fetch(context.Background(), QueryParams{Term: "golang", Count: 10, First: 0}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(gotQuery, "first=") {
		t.Fatalf("First=0 不应发送 first 参数: %s", gotQuery)
	}
	// First≥10 → 0 基原样发送
	if _, err := e.fetch(context.Background(), QueryParams{Term: "golang", Count: 10, First: 10}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotQuery, "first=10") {
		t.Fatalf("First=10 应发送 first=10: %s", gotQuery)
	}
	// 语言与安全搜索参数保留
	if _, err := e.fetch(context.Background(), QueryParams{Term: "golang", Count: 10, First: 0, Language: "zh-CN", SafeStrict: true}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"mkt=zh-CN", "setlang=zh", "adlt=strict"} {
		if !strings.Contains(gotQuery, want) {
			t.Fatalf("应包含 %s: %s", want, gotQuery)
		}
	}
}

func urlsOf(rs []Result) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.URL
	}
	return out
}
