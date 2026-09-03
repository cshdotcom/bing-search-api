package main

// searxng_test.go — SearXNG 兼容接口(/search)默认禁用行为测试:
// 环境变量 ENABLE_SEARXNG 解析、handleSearch 门禁(403 与开启指引)。

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestSearxngEnabled ENABLE_SEARXNG 取值解析(默认关,显式开)
func TestSearxngEnabled(t *testing.T) {
	cases := []struct {
		env  string
		want bool
	}{
		{"", false},       // 未设/空 → 关
		{"0", false},      //
		{"false", false},  //
		{"no", false},     //
		{"off", false},    //
		{"random", false}, // 无意义值 → 关
		{"1", true},       //
		{"true", true},    //
		{"YES", true},     // 大小写不敏感
		{" On ", true},    // 空白容忍
	}
	for _, c := range cases {
		t.Setenv("ENABLE_SEARXNG", c.env)
		if got := searxngEnabled(); got != c.want {
			t.Fatalf("ENABLE_SEARXNG=%q → %v, 期望 %v", c.env, got, c.want)
		}
	}
}

// TestHandleSearchDisabled 默认(未开启)访问 /search → 403 + 开启指引
func TestHandleSearchDisabled(t *testing.T) {
	t.Setenv("ENABLE_SEARXNG", "")
	srv := &server{engine: &BingEngine{Base: "https://www.bing.com"}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/search?q=x", nil)
	srv.handleSearch(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("禁用时应返回 403,实际 %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "已禁用") || !strings.Contains(body, "ENABLE_SEARXNG=1") {
		t.Fatalf("403 响应应包含禁用说明与开启指引: %s", body)
	}
	if !strings.Contains(body, "/v7/search") {
		t.Fatalf("403 响应应提示 v7 接口不受影响: %s", body)
	}
}

// TestHandleSearchEnabledGatePasses 开启后门禁放行(缺 q 走到 400 参数校验,
// 不触发网络请求即证明门禁未拦截)
func TestHandleSearchEnabledGatePasses(t *testing.T) {
	t.Setenv("ENABLE_SEARXNG", "1")
	srv := &server{engine: &BingEngine{Base: "https://www.bing.com"}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/search", nil)
	srv.handleSearch(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("开启后缺 q 应返回 400(证明门禁放行),实际 %d: %s", w.Code, w.Body.String())
	}
}
