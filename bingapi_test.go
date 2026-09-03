package main

// bingapi_test.go — Bing 官方 Search API v7 兼容层测试:
// 参数解析/校验、官方响应结构组装、错误格式、SERP 总数解析。

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newV7Req 快速构造带查询串的请求
func newV7Req(method, rawQuery string, body string) *http.Request {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, "/v7/search?"+rawQuery, bytes.NewReader([]byte(body)))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, "/v7/search?"+rawQuery, nil)
	}
	return r
}

// TestV7ParamsDefaults 默认值:count=10、offset=0、safeSearch=Moderate
func TestV7ParamsDefaults(t *testing.T) {
	p, perr := v7ParamsFromRequest(newV7Req(http.MethodGet, "q=golang", ""))
	if perr != nil {
		t.Fatalf("意外错误: %+v", perr)
	}
	if p.Count != 10 || p.Offset != 0 || p.SafeSearch != "Moderate" {
		t.Fatalf("默认值不符: %+v", p)
	}
	if p.Mkt != "" || p.SetLang != "" || len(p.RespFilter) != 0 {
		t.Fatalf("可选参数应为空: %+v", p)
	}
}

// TestV7ParamsFromJSONBody POST JSON body 优先级最高
func TestV7ParamsFromJSONBody(t *testing.T) {
	r := newV7Req(http.MethodPost, "q=urlquery&count=7", `{"q":"jsonquery","count":25,"offset":40,"mkt":"zh-CN","safeSearch":"STRICT"}`)
	p, perr := v7ParamsFromRequest(r)
	if perr != nil {
		t.Fatalf("意外错误: %+v", perr)
	}
	if p.Q != "jsonquery" {
		t.Fatalf("JSON body 应优先: q=%q", p.Q)
	}
	if p.Count != 25 || p.Offset != 40 {
		t.Fatalf("count/offset 解析错误: %+v", p)
	}
	if p.Mkt != "zh-CN" || p.SafeSearch != "Strict" {
		t.Fatalf("mkt/safeSearch 错误: %+v", p)
	}
}

// TestV7ParamsErrors 官方格式参数校验
func TestV7ParamsErrors(t *testing.T) {
	// q 缺失
	if _, perr := v7ParamsFromRequest(newV7Req(http.MethodGet, "count=10", "")); perr == nil || perr.SubCode != "ParameterMissing" || perr.Param != "q" {
		t.Fatalf("缺 q 应报 ParameterMissing/q: %+v", perr)
	}
	// count 越界
	for _, bad := range []string{"0", "51", "abc", "-3"} {
		if _, perr := v7ParamsFromRequest(newV7Req(http.MethodGet, "q=x&count="+bad, "")); perr == nil || perr.SubCode != "ParameterInvalid" || perr.Param != "count" {
			t.Fatalf("count=%q 应报 ParameterInvalid/count: %+v", bad, perr)
		}
	}
	// count 合法边界
	for _, ok := range []string{"1", "50"} {
		if _, perr := v7ParamsFromRequest(newV7Req(http.MethodGet, "q=x&count="+ok, "")); perr != nil {
			t.Fatalf("count=%q 不应报错: %+v", ok, perr)
		}
	}
	// offset 越界
	for _, bad := range []string{"-1", "9001", "zz"} {
		if _, perr := v7ParamsFromRequest(newV7Req(http.MethodGet, "q=x&offset="+bad, "")); perr == nil || perr.Param != "offset" {
			t.Fatalf("offset=%q 应报错: %+v", bad, perr)
		}
	}
	// offset 合法边界
	if _, perr := v7ParamsFromRequest(newV7Req(http.MethodGet, "q=x&offset=9000", "")); perr != nil {
		t.Fatalf("offset=9000 不应报错: %+v", perr)
	}
	// safeSearch 非法
	if _, perr := v7ParamsFromRequest(newV7Req(http.MethodGet, "q=x&safeSearch=medium", "")); perr == nil || perr.Param != "safeSearch" {
		t.Fatalf("safeSearch 非法值应报错: %+v", perr)
	}
}

// TestV7ParamsResponseFilter responseFilter 解析与归一化
func TestV7ParamsResponseFilter(t *testing.T) {
	p, perr := v7ParamsFromRequest(newV7Req(http.MethodGet, "q=x&responseFilter=Webpages,+RelatedSearches", ""))
	if perr != nil {
		t.Fatalf("意外错误: %+v", perr)
	}
	if len(p.RespFilter) != 2 || p.RespFilter[0] != "webpages" || p.RespFilter[1] != "relatedsearches" {
		t.Fatalf("responseFilter 归一化错误: %v", p.RespFilter)
	}
}

// TestBuildBingV7Response 官方响应结构逐字段校验
func TestBuildBingV7Response(t *testing.T) {
	p := v7Params{Q: "golang", Count: 10, Offset: 0, SafeSearch: "Moderate"}
	results := []Result{
		{Title: "Go", URL: "https://go.dev/", Content: "The Go Programming Language", Position: 1},
		{Title: "Go Blog", URL: "https://go.dev/blog", Content: "The Go Blog", Position: 2},
	}
	resp := buildBingV7Response(p, "en-US", results, []string{"golang tutorial"}, 12345, true, true)

	if resp.Type != "SearchResponse" {
		t.Fatalf("_type 应为 SearchResponse: %q", resp.Type)
	}
	if resp.QueryContext.OriginalQuery != "golang" {
		t.Fatalf("queryContext.originalQuery 错误: %q", resp.QueryContext.OriginalQuery)
	}
	if resp.WebPages == nil {
		t.Fatal("webPages 不应为 nil")
	}
	if resp.WebPages.TotalEstimatedMatches != 12345 {
		t.Fatalf("totalEstimatedMatches 错误: %d", resp.WebPages.TotalEstimatedMatches)
	}
	if len(resp.WebPages.Value) != 2 {
		t.Fatalf("应返回 2 条结果: %d", len(resp.WebPages.Value))
	}
	first := resp.WebPages.Value[0]
	if first.ID != bingV7IDPrefix+"#WebPages.0" {
		t.Fatalf("id 格式错误: %q", first.ID)
	}
	if first.Name != "Go" || first.URL != "https://go.dev/" || first.DisplayURL != "https://go.dev/" {
		t.Fatalf("结果字段错误: %+v", first)
	}
	if first.Snippet != "The Go Programming Language" {
		t.Fatalf("snippet 错误: %q", first.Snippet)
	}
	if first.Language != "en-US" || !first.IsFamilyFriendly {
		t.Fatalf("language/isFamilyFriendly 错误: %+v", first)
	}
	if !strings.Contains(resp.WebPages.WebSearchURL, "q=golang") || !strings.Contains(resp.WebPages.WebSearchURL, "mkt=en-US") {
		t.Fatalf("webSearchUrl 缺少 q/mkt: %q", resp.WebPages.WebSearchURL)
	}
	if resp.RelatedSearches == nil || len(resp.RelatedSearches.Value) != 1 {
		t.Fatalf("relatedSearches 错误: %+v", resp.RelatedSearches)
	}
	if resp.RelatedSearches.Value[0].Text != "golang tutorial" || resp.RelatedSearches.Value[0].DisplayText != "golang tutorial" {
		t.Fatalf("相关搜索字段错误: %+v", resp.RelatedSearches.Value[0])
	}

	// JSON 序列化:官方键名必须逐一对齐
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}
	s := string(b)
	for _, key := range []string{`"_type":"SearchResponse"`, `"queryContext"`, `"originalQuery"`, `"adultIntent"`,
		`"webPages"`, `"webSearchUrl"`, `"totalEstimatedMatches"`, `"value"`, `"id"`, `"name"`, `"url"`,
		`"displayUrl"`, `"snippet"`, `"isFamilyFriendly"`, `"isNavigational"`, `"relatedSearches"`, `"displayText"`} {
		if !strings.Contains(s, key) {
			t.Fatalf("官方响应缺少键 %s: %s", key, s)
		}
	}
}

// TestBuildBingV7ResponseFilter responseFilter 过滤语义
func TestBuildBingV7ResponseFilter(t *testing.T) {
	p := v7Params{Q: "x", Count: 10, RespFilter: []string{"images"}}
	results := []Result{{Title: "t", URL: "https://e.com/", Content: "c"}}
	resp := buildBingV7Response(p, "", results, []string{"s"}, 0, false, false)
	if resp.WebPages != nil {
		t.Fatal("过滤掉 webpages 后不应有 webPages 答案")
	}
	if resp.RelatedSearches != nil {
		t.Fatal("过滤掉 relatedsearches 后不应有 relatedSearches 答案")
	}
}

// TestBuildBingV7ResponseTotalFloor totalEstimatedMatches 兜底:offset+结果数
func TestBuildBingV7ResponseTotalFloor(t *testing.T) {
	p := v7Params{Q: "x", Count: 10, Offset: 100}
	results := []Result{{Title: "t", URL: "https://e.com/", Content: "c"}}
	resp := buildBingV7Response(p, "", results, nil, 0, true, false)
	if resp.WebPages.TotalEstimatedMatches != 101 {
		t.Fatalf("兜底值应为 offset+结果数=101: %d", resp.WebPages.TotalEstimatedMatches)
	}
}

// TestWriteBingV7Error 官方 ErrorResponse 格式
func TestWriteBingV7Error(t *testing.T) {
	w := httptest.NewRecorder()
	writeBingV7Error(w, http.StatusBadRequest, bingV7Error{
		Code: "InvalidRequest", SubCode: "ParameterInvalid",
		Message: "bad", Parameter: "count", Value: "99",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("状态码错误: %d", w.Code)
	}
	var got bingV7ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("响应不是合法 JSON: %v", err)
	}
	if got.Type != "ErrorResponse" || len(got.Errors) != 1 {
		t.Fatalf("ErrorResponse 结构错误: %+v", got)
	}
	e := got.Errors[0]
	if e.Code != "InvalidRequest" || e.SubCode != "ParameterInvalid" || e.Parameter != "count" || e.Value != "99" {
		t.Fatalf("错误字段错误: %+v", e)
	}
}

// TestParseTotalResults 各地区 SERP 计数条解析
func TestParseTotalResults(t *testing.T) {
	cases := []struct {
		page string
		want int64
	}{
		{`<span class="sb_count">约 7,140,000 条结果</span>`, 7140000},
		{`<span class="sb_count">2.340.000 Ergebnisse</span>`, 2340000},
		{`<span class="sb_count">1,234 results</span>`, 1234},
		{`<span class="sb_count">约 15 条结果</span>`, 15},
		{`<div>无计数条</div>`, 0},
		{``, 0},
	}
	for _, c := range cases {
		if got := parseTotalResults(c.page); got != c.want {
			t.Fatalf("parseTotalResults(%q)=%d, 期望 %d", c.page, got, c.want)
		}
	}
}

// TestContainsFold token 归一化匹配
func TestContainsFold(t *testing.T) {
	if !containsFold([]string{" WebPages ", "images"}, "webpages") {
		t.Fatal("大小写/空白应忽略")
	}
	if containsFold([]string{"images"}, "webpages") {
		t.Fatal("不应误匹配")
	}
}
