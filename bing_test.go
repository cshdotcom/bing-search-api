package main

import "testing"

func TestDecodeBingRedirect(t *testing.T) {
	// u=a1aHR0cHM6Ly9nby5kZXYv 是 "https://go.dev/" 的 base64url
	href := "https://www.bing.com/ck/a?!&&p=abc&u=a1aHR0cHM6Ly9nby5kZXYv&ntb=1"
	if got := decodeBingRedirect(href); got != "https://go.dev/" {
		t.Fatalf("decodeBingRedirect = %q, want https://go.dev/", got)
	}
	if got := decodeBingRedirect("https://go.dev/doc"); got != "https://go.dev/doc" {
		t.Fatalf("普通链接不应被改写, got %q", got)
	}
	// 非法 base64 不应导致崩溃
	if got := decodeBingRedirect("https://www.bing.com/ck/a?&u=a1!!!!&ntb=1"); got == "" {
		t.Fatal("非法重定向应原样返回而非空串")
	}
}

func TestCleanText(t *testing.T) {
	cases := map[string]string{
		"<strong>阿里</strong>云-计算":           "阿里云-计算",
		"a &amp; b":                         "a & b",
		"  x \n y  ":                        "x y",
		"The Go <wbr>Programming Language":  "The Go Programming Language",
		"<em>Go</em> is <b>open</b> source": "Go is open source",
	}
	for in, want := range cases {
		if got := cleanText(in); got != want {
			t.Errorf("cleanText(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFindLanguage(t *testing.T) {
	cases := []struct{ in, want string }{
		{"zh", "zh-CN"},      // 语言代码 -> 默认市场
		{"ZH_cn", "zh-CN"},   // 大小写不敏感
		{"zh_CN", "zh-CN"},   // 下划线分隔
		{"zh-Hans", "zh-CN"}, // 别名
		{"pt", "pt-BR"},      // 葡语默认巴西
		{"de-XX", "de-DE"},   // 未知地区 -> 语言默认市场
		{"no", "nb-NO"},      // 挪威语旧代码
		{"es-419", "es-MX"},  // 拉美西语
		{"in", "id-ID"},      // 印尼语旧代码
		{"iw", "he-IL"},      // 希伯来语旧代码
		{"sr-Cyrl-RS", "sr-Cyrl-RS"},
		{"sr", "sr-Latn-RS"}, // 塞尔维亚语默认拉丁
		{"fil", "tl-PH"},
	}
	for _, c := range cases {
		market, _, ok := findLanguage(c.in)
		if !ok || market != c.want {
			t.Errorf("findLanguage(%q) = %q, %v; want %q", c.in, market, ok, c.want)
		}
	}
	if _, _, ok := findLanguage("klingon"); ok {
		t.Error("klingon 不应被识别")
	}
}

func TestResolveLanguage(t *testing.T) {
	if m, err := resolveLanguage("all", "zh-CN"); err != nil || m != "" {
		t.Errorf("all 应不限语言, got %q, %v", m, err)
	}
	if m, err := resolveLanguage("", "zh-CN,zh;q=0.9,en;q=0.8"); err != nil || m != "zh-CN" {
		t.Errorf("空参数应回退 Accept-Language, got %q, %v", m, err)
	}
	if m, err := resolveLanguage("", "zh"); err != nil || m != "zh-CN" {
		t.Errorf("Accept-Language 语言代码, got %q, %v", m, err)
	}
	if m, err := resolveLanguage("", ""); err != nil || m != "" {
		t.Errorf("无任何语言信息应返回空, got %q, %v", m, err)
	}
	if m, err := resolveLanguage("JA-jp", ""); err != nil || m != "ja-JP" {
		t.Errorf("大小写不敏感, got %q, %v", m, err)
	}
	if _, err := resolveLanguage("klingon", ""); err == nil {
		t.Error("未知语言应报错")
	}
}

func TestPreferredFromAcceptLanguage(t *testing.T) {
	if got := preferredFromAcceptLanguage("fr-CH,fr;q=0.9,en;q=0.8"); got != "fr-CH" {
		t.Errorf("got %q, want fr-CH", got)
	}
	if got := preferredFromAcceptLanguage("*"); got != "" {
		t.Errorf("通配符应跳过, got %q", got)
	}
	if got := preferredFromAcceptLanguage("xx-YY,de;q=0.5"); got != "de-DE" {
		t.Errorf("应跳过不可识别项, got %q", got)
	}
	if got := preferredFromAcceptLanguage(""); got != "" {
		t.Errorf("空头应返回空, got %q", got)
	}
}

func TestParseBing(t *testing.T) {
	page := `<ol id="b_results">` +
		`<li class="b_algo"><h2><a href="https://www.bing.com/ck/a?&u=a1aHR0cHM6Ly9nby5kZXYv">The Go <strong>Programming</strong> Language</a></h2>` +
		`<div class="b_caption"><p class="b_lineclamp2">An open-source language</p></div></li>` +
		`<li class="b_algo"><h2><a href="https://example.com/2">Second Result</a></h2>` +
		`<div class="b_caption"><p>Second snippet</p></div></li>` +
		`</ol>`

	res, sug := parseBing(page)
	if len(res) != 2 {
		t.Fatalf("want 2 results, got %d", len(res))
	}
	if res[0].Title != "The Go Programming Language" {
		t.Errorf("title = %q", res[0].Title)
	}
	if res[0].URL != "https://go.dev/" {
		t.Errorf("url = %q", res[0].URL)
	}
	if res[0].Content != "An open-source language" {
		t.Errorf("content = %q", res[0].Content)
	}
	if res[0].Position != 1 || res[1].Position != 2 {
		t.Errorf("position 应保持原始顺序: %d, %d", res[0].Position, res[1].Position)
	}
	if len(sug) != 0 {
		t.Errorf("不应有相关搜索, got %v", sug)
	}
}

func TestParseBingSuggestions(t *testing.T) {
	page := `<li class="b_rs"><ul><li><a href="#">golang tutorial</a></li><li><a href="#">golang vs rust</a></li></ul></li>`
	_, sug := parseBing(page)
	if len(sug) != 2 || sug[0] != "golang tutorial" {
		t.Fatalf("suggestions = %v", sug)
	}
}

func TestClamp(t *testing.T) {
	if got := clamp(searchParams{Count: 0, Page: 0}); got.Count != 10 || got.Page != 1 {
		t.Errorf("下限 clamp 失败: %+v", got)
	}
	if got := clamp(searchParams{Count: 999, Page: 999}); got.Count != 50 || got.Page != 100 {
		t.Errorf("上限 clamp 失败: %+v", got)
	}
}
