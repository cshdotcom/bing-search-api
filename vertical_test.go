package main

// vertical_test.go — 垂直搜索(图片/视频/新闻/词典)测试:
// 解析器(以真实页面样本裁剪的夹具)、类别归一、v7 垂直端点的
// 路由/鉴权/参数校验/响应组装。全部离线,不触网。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// ── 夹具(裁剪自 Bing 真实响应,结构与线上一致)────────────────

const fxImagesAsync = `<div class="dg_b"><div><a class="iusc" style="height:180px;width:242px" m="{&quot;sid&quot;:&quot;&quot;,&quot;cturl&quot;:&quot;&quot;,&quot;cid&quot;:&quot;uUC&quot;,&quot;purl&quot;:&quot;https://example.com/cats&quot;,&quot;murl&quot;:&quot;https://cdn.example.com/cat.jpg&quot;,&quot;turl&quot;:&quot;https://ts4.mm.bing.net/th?id=OIP.abc&amp;pid=15.1&quot;,&quot;md5&quot;:&quot;x&quot;,&quot;t&quot;:&quot;A <b>Cat</b> Photo&quot;,&quot;desc&quot;:&quot;a cat lying on a table&quot;,&quot;mid&quot;:&quot;MID1&quot;}"><img class="mimg" src="https://th.bing.com/th?id=OIP.abc&amp;w=242&amp;h=180"></a><a class="iusc" style="width:160px;height:120px" m="{&quot;purl&quot;:&quot;https://example.org/dogs&quot;,&quot;murl&quot;:&quot;https://cdn.example.org/dog.png&quot;,&quot;t&quot;:&quot;A Dog&quot;,&quot;mid&quot;:&quot;MID2&quot;}"></a><a class="iusc" m="{&quot;purl&quot;:&quot;&quot;,&quot;murl&quot;:&quot;&quot;}"></a>`

const fxVideosPage = `<div class="mc_vtvc_con"><div class="mc_vtvc_meta" vrhm="{&quot;cid&quot;:&quot;mgzhp&quot;,&quot;smturl&quot;:&quot;https://th.bing.com/th/id/OMB.1?pid=2.1&quot;,&quot;du&quot;:&quot;4:18&quot;,&quot;murl&quot;:&quot;https://www.youtube.com/watch?v=abc&quot;,&quot;thid&quot;:&quot;OVP.1&quot;,&quot;mid&quot;:&quot;VID1&quot;,&quot;vt&quot;:&quot;Cats Being <b>Funny</b>&quot;,&quot;pgurl&quot;:&quot;https://www.youtube.com/watch?v=abc&quot;}"></div><div vrhm="{&quot;du&quot;:&quot;1:02:03&quot;,&quot;murl&quot;:&quot;https://vimeo.com/2&quot;,&quot;smturl&quot;:&quot;https://th.bing.com/th/id/OMB.2&quot;,&quot;vt&quot;:&quot;Long Video&quot;,&quot;pgurl&quot;:&quot;https://vimeo.com/2&quot;,&quot;mid&quot;:&quot;VID2&quot;}"></div><div vrhm="{&quot;du&quot;:&quot;9:10&quot;,&quot;murl&quot;:&quot;https://www.youtube.com/watch?v=abc&quot;,&quot;vt&quot;:&quot;Duplicate&quot;,&quot;pgurl&quot;:&quot;https://www.youtube.com/watch?v=abc&quot;}"></div></div>`

const fxNewsRSS = `<?xml version="1.0" encoding="utf-8" ?><rss version="2.0" xmlns:News="https://www.bing.com/news/search?q=cat&amp;format=rss"><channel><title>cat - BingNews</title><link>https://www.bing.com/news/search?q=cat&amp;format=rss</link><item><title>Cat Scratch Disease</title><link>http://www.bing.com/news/apiclick.aspx?ref=FexRss&amp;aid=1&amp;url=https%3a%2f%2fwww.hopkinsmedicine.org%2fhealth%2fcat-scratch-disease&amp;c=1&amp;mkt=en-ww</link><description><![CDATA[What is cat scratch disease? Cat scratches can cause it.]]></description><pubDate>Thu, 05 Sep 2024 17:01:00 GMT</pubDate><News:Source>Johns Hopkins Medicine</News:Source></item><item><title>Plain title</title><link>https://example.com/direct-link</link><description>No redirect here</description><pubDate>Tue, 04 Feb 2025 02:15:00 GMT</pubDate></item></channel></rss>`

const fxDictEN = `<div class="qdef"><div class="hd_area"><div class="hd_div" id="headword"><h1><strong>hello</strong></h1></div><div class="hd_tf_lh"><div class="hd_p1_1" lang="en"><div class="hd_prUS b_primtxt">美&#160;heˈləʊ] </div><div class="hd_tf"><a id="bigaud_us"></a></div><div class="hd_pr b_primtxt">英国&#160;həˈləʊ] </div></div></div></div><ul><li><span class="pos">int.</span><span class="def b_regtxt"><a>你好</a><span>；</span><span> </span><a>喂</a></span></li></ul><div class="li_sen" id="newLeId"><div class="each_seg"><div class="li_pos"><div class="pos">int.</div></div><div class="se_lis"><table><tr class="def_row"><td><div class="se_d b_primtxt">1.</div></td><td><div class="de_co">（用于问候、接电话或引起注意）哈啰，喂，你好</div></td></tr></table></div><div class="se_lis"><table><tr><td><div class="se_d b_primtxt">2.</div></td><td><div class="de_co">（表示惊讶）嘿</div></td></tr></table></div></div></div><div class="sen_en b_regtxt"><a>You</a><span> </span><a>know</a><span>, </span><a>he</a><span> </span><a>didn</a><span> </span><a>even</a><span> </span><a>bother</a></div><div class="sen_en b_regtxt"><a>short</a></div>`

const fxDictZH = `<div class="qdef"><div class="hd_area"><div class="hd_div" id="headword"><h1><strong>你好</strong></h1></div><div class="hd_tf_lh"><div class="hd_p1_1" lang="en">[nǐ hǎo] </div></div></div><ul><li><span class="pos">na.</span><span class="def b_regtxt"><a>hello</a><span>;</span><span> </span><span>〈</span><a>正式</a><span>,</span><a>口</a><span>〉</span><a>how do you do?</a></span></li></ul><div id="crossid" style="display:block;"><table><tr class="def_row"><td><div class="se_d b_primtxt">1.</div></td><td><div class="df_cr_w">〈正式, 口〉 how do you do ?</div></td></tr></table><table><tr><td><div class="se_d b_primtxt">2.</div></td><td><div class="df_cr_w">〈口〉 how are you ?</div></td></tr></table></div><div id="antoid" style="display:none;"></div>`

// ── 类别归一 ──────────────────────────────────────────────────

func TestNormalizeCategory(t *testing.T) {
	cases := []struct {
		raw  string
		want string
		fail bool
	}{
		{"", "", false},
		{"general", "", false},
		{"web", "", false},
		{"All", "", false},
		{"images", "images", false},
		{"PICTURES", "images", false},
		{"videos", "videos", false},
		{"video", "videos", false},
		{"news", "news", false},
		{"dict", "dict", false},
		{"Dictionary", "dict", false},
		{"define", "dict", false},
		{"images,videos", "images", false}, // categories 多值取首 token
		{"academic", "", true},
		{"science", "", true},
		{"文献", "", true},
		{"shopping", "", true},
		{"maps", "", true},
		{"unknown-cat", "", true},
	}
	for _, c := range cases {
		got, err := normalizeCategory(c.raw)
		if c.fail {
			if err == nil {
				t.Fatalf("normalizeCategory(%q) 应报错,实际 %q", c.raw, got)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Fatalf("normalizeCategory(%q)=%q, err=%v; 期望 %q", c.raw, got, err, c.want)
		}
	}
}

func TestPickCategory(t *testing.T) {
	if got := pickCategory("images", ""); got != "images" {
		t.Fatalf("单数 category 应优先: %q", got)
	}
	if got := pickCategory("", "videos,news"); got != "videos" {
		t.Fatalf("复数 categories 应取首 token: %q", got)
	}
	if got := pickCategory("", "news"); got != "news" {
		t.Fatalf("单个 categories: %q", got)
	}
	if got := pickCategory("", ""); got != "" {
		t.Fatalf("空应返回空: %q", got)
	}
	if got := pickCategory("images", "videos"); got != "images" {
		t.Fatalf("category 与 categories 同时给时单数优先: %q", got)
	}
}

// ── 解析器 ────────────────────────────────────────────────────

func TestParseImagesAsync(t *testing.T) {
	items := parseImagesAsync(fxImagesAsync)
	if len(items) != 2 {
		t.Fatalf("应解析出 2 条(空 murl/purl 的锚点跳过),实际 %d", len(items))
	}
	first := items[0]
	if first.Title != "A Cat Photo" {
		t.Fatalf("标题应清洗行内标签: %q", first.Title)
	}
	if first.PageURL != "https://example.com/cats" || first.ImageURL != "https://cdn.example.com/cat.jpg" {
		t.Fatalf("链接错误: %+v", first)
	}
	if first.ThumbURL != "https://ts4.mm.bing.net/th?id=OIP.abc&pid=15.1" {
		t.Fatalf("缩略图应还原实体: %q", first.ThumbURL)
	}
	if first.Desc != "a cat lying on a table" || first.MID != "MID1" {
		t.Fatalf("desc/mid 错误: %+v", first)
	}
	// style 尺寸(height 在前的顺序)按属性名归位
	if first.Width != 242 || first.Height != 180 {
		t.Fatalf("width/height 应按属性名归位: W=%d H=%d", first.Width, first.Height)
	}
	// 无 turl 时用原图兜底
	if items[1].ThumbURL != items[1].ImageURL {
		t.Fatalf("缺 turl 应用 murl 兜底: %+v", items[1])
	}
	// 顺序颠倒的 style(width 在前)同样归位正确
	if items[1].Width != 160 || items[1].Height != 120 {
		t.Fatalf("第二张尺寸错误: W=%d H=%d", items[1].Width, items[1].Height)
	}
}

func TestParseVideosPage(t *testing.T) {
	items := parseVideosPage(fxVideosPage)
	if len(items) != 2 {
		t.Fatalf("应解析出 2 条(第三条为重复 murl+pgurl 去重),实际 %d", len(items))
	}
	first := items[0]
	if first.Title != "Cats Being Funny" {
		t.Fatalf("标题应清洗标签: %q", first.Title)
	}
	if first.PageURL != "https://www.youtube.com/watch?v=abc" || first.ContentURL != first.PageURL {
		t.Fatalf("链接错误: %+v", first)
	}
	if first.Duration != "4:18" || first.MID != "VID1" || first.ThumbID != "OVP.1" {
		t.Fatalf("时长/ID 错误: %+v", first)
	}
	if items[1].Duration != "1:02:03" {
		t.Fatalf("第二时长错误: %q", items[1].Duration)
	}
}

func TestParseNewsRSS(t *testing.T) {
	items := parseNewsRSS(fxNewsRSS)
	if len(items) != 2 {
		t.Fatalf("应解析出 2 条,实际 %d", len(items))
	}
	first := items[0]
	if first.Title != "Cat Scratch Disease" {
		t.Fatalf("标题错误: %q", first.Title)
	}
	if first.URL != "https://www.hopkinsmedicine.org/health/cat-scratch-disease" {
		t.Fatalf("apiclick 重定向应还原真实链接: %q", first.URL)
	}
	if !strings.Contains(first.Desc, "cat scratch disease") {
		t.Fatalf("CDATA 描述应解析: %q", first.Desc)
	}
	if first.Date != "Thu, 05 Sep 2024 17:01:00 GMT" {
		t.Fatalf("pubDate 错误: %q", first.Date)
	}
	if first.Source != "Johns Hopkins Medicine" {
		t.Fatalf("News:Source 错误: %q", first.Source)
	}
	if items[1].URL != "https://example.com/direct-link" {
		t.Fatalf("非 apiclick 链接应原样保留: %q", items[1].URL)
	}
	if items[1].Source != "" {
		t.Fatalf("无 News:Source 应为空: %q", items[1].Source)
	}
}

func TestParseDictPageEN(t *testing.T) {
	entry := parseDictPage(fxDictEN)
	if entry == nil {
		t.Fatal("英文词条应解析成功")
	}
	if entry.Word != "hello" {
		t.Fatalf("头词错误: %q", entry.Word)
	}
	if entry.PronUS != "heˈləʊ" || entry.PronUK != "həˈləʊ" {
		t.Fatalf("音标清洗错误: US=%q UK=%q", entry.PronUS, entry.PronUK)
	}
	if entry.Pinyin != "" {
		t.Fatalf("英文词不应误报拼音(hd_p1_1 含嵌套标签应跳过): %q", entry.Pinyin)
	}
	if entry.Def.Pos != "int." {
		t.Fatalf("首要释义词性错误: %q", entry.Def.Pos)
	}
	if entry.Def.Text != "你好；喂" {
		t.Fatalf("首要释义应解析并收紧全角标点: %q", entry.Def.Text)
	}
	if len(entry.Senses) != 2 {
		t.Fatalf("se_lis 义项数错误: %d", len(entry.Senses))
	}
	if !strings.HasPrefix(entry.Senses[0].Text, "1.（用于问候、接电话或引起注意）哈啰，喂，你好") {
		t.Fatalf("第一条义项错误: %q", entry.Senses[0].Text)
	}
	if !strings.Contains(entry.Senses[1].Text, "（表示惊讶）嘿") {
		t.Fatalf("第二条义项错误: %q", entry.Senses[1].Text)
	}
	// 过短例句(<20)应过滤,有效例句保留
	if len(entry.Examples) != 1 {
		t.Fatalf("例句应过滤过短项: %d", len(entry.Examples))
	}
	if parseDictPage("<html>no dict</html>") != nil {
		t.Fatal("无头词的页面应返回 nil")
	}
}

func TestParseDictPageZH(t *testing.T) {
	entry := parseDictPage(fxDictZH)
	if entry == nil {
		t.Fatal("中文词条应解析成功")
	}
	if entry.Word != "你好" {
		t.Fatalf("头词错误: %q", entry.Word)
	}
	if entry.Pinyin != "nǐ hǎo" {
		t.Fatalf("拼音应从纯文本 hd_p1_1 解析: %q", entry.Pinyin)
	}
	if entry.PronUS != "" || entry.PronUK != "" {
		t.Fatalf("中文词不应有音标: %+v", entry)
	}
	if entry.Def.Pos != "na." || entry.Def.Text != "hello ;〈正式 , 口〉how do you do?" {
		t.Fatalf("汉英首要释义错误: pos=%q text=%q", entry.Def.Pos, entry.Def.Text)
	}
	if len(entry.Senses) != 2 {
		t.Fatalf("crossid 义项数错误: %d", len(entry.Senses))
	}
	if !strings.HasPrefix(entry.Senses[0].Text, "1.〈正式, 口〉how do you do ?") {
		t.Fatalf("crossid 首义项应去掉标签残片: %q", entry.Senses[0].Text)
	}
	if !strings.Contains(entry.Senses[1].Text, "how are you ?") {
		t.Fatalf("第二条义项错误: %q", entry.Senses[1].Text)
	}
}

// ── 换算函数 ──────────────────────────────────────────────────

func TestDurationToISO8601(t *testing.T) {
	cases := []struct{ in, want string }{
		{"32:52", "PT32M52S"},
		{"9:10", "PT9M10S"},
		{"1:02:03", "PT1H2M3S"},
		{"", ""},
		{"abc", ""},
		{"59", ""},
	}
	for _, c := range cases {
		if got := durationToISO8601(c.in); got != c.want {
			t.Fatalf("durationToISO8601(%q)=%q, 期望 %q", c.in, got, c.want)
		}
	}
}

func TestRSSDateToISO(t *testing.T) {
	if got := rssDateToISO("Thu, 05 Sep 2024 17:01:00 GMT"); got != "2024-09-05T17:01:00Z" {
		t.Fatalf("RFC1123 转换错误: %q", got)
	}
	if got := rssDateToISO("not-a-date"); got != "not-a-date" {
		t.Fatalf("解析失败应原样返回: %q", got)
	}
	if got := rssDateToISO(""); got != "" {
		t.Fatalf("空串应返回空: %q", got)
	}
}

func TestImageExtToFormat(t *testing.T) {
	cases := []struct{ in, want string }{
		{"https://e.com/a/b.jpg", "jpeg"},
		{"https://e.com/a/b.JPG?w=2", "jpeg"},
		{"https://e.com/a/b.png", "png"},
		{"https://e.com/a/b.webp#x", "webp"},
		{"https://e.com/a/noext", ""},
		{"https://e.com/", ""},
	}
	for _, c := range cases {
		if got := imageExtToFormat(c.in); got != c.want {
			t.Fatalf("imageExtToFormat(%q)=%q, 期望 %q", c.in, got, c.want)
		}
	}
}

// ── v7 垂直端点:路由 / 鉴权 / 参数(离线,断在触网之前)────────

// newTestServer 构造测试用 server(引擎指向 bing.com 但所有用例
// 都在发起网络请求之前断言完毕)
func newTestServer() *server {
	return &server{engine: &BingEngine{Base: "https://www.bing.com"}}
}

func TestV7VerticalRoutes(t *testing.T) {
	s := newTestServer()
	mux := s.routes()
	// 4 个前缀别名 × 4 个垂直端点,全部应存在(以缺 q 的 400 证明路由命中)
	for _, prefix := range []string{"/v7/", "/v7.0/", "/bing/v7/", "/bing/v7.0/"} {
		for _, vert := range []string{"images", "videos", "news", "dict"} {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, prefix+vert+"/search", nil)
			mux.ServeHTTP(w, r)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("%s%s/search 路由未命中: %d", prefix, vert, w.Code)
			}
			if !strings.Contains(w.Body.String(), "ParameterMissing") {
				t.Fatalf("%s%s/search 应报缺 q: %s", prefix, vert, w.Body.String())
			}
		}
	}
	// v7 web 端点回归:仍在
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v7/search", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("/v7/search 回归失败: %d", w.Code)
	}
}

func TestV7VerticalMethodAndCount(t *testing.T) {
	s := newTestServer()
	mux := s.routes()
	// 非 GET/POST → 405(官方 InvalidRequest)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/v7/images/search?q=x", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("DELETE 应 405: %d", w.Code)
	}
	// images count 上限 150
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v7/images/search?q=x&count=151", nil))
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "1~150") {
		t.Fatalf("images count=151 应 400 且提示 1~150: %d %s", w.Code, w.Body.String())
	}
	// videos count 上限 100
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v7/videos/search?q=x&count=101", nil))
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "1~100") {
		t.Fatalf("videos count=101 应 400 且提示 1~100: %d %s", w.Code, w.Body.String())
	}
	// 非法 mkt
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v7/news/search?q=x&mkt=xx-XX", nil))
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "mkt") {
		t.Fatalf("非法 mkt 应 400: %d %s", w.Code, w.Body.String())
	}
}

// ── 端到端(本地伪上游,验证分发与上游参数)────────────────

// newUpstream 返回一个固定回放的伪 Bing 上游,并记录收到的请求
func newUpstream(body string) (*httptest.Server, *string, *url.Values) {
	var gotPath string
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(body))
	}))
	return srv, &gotPath, &gotQuery
}

// TestV7VerticalEndToEnd 伪上游全链路:路由→参数→抓取→解析→官方响应组装,
// 并断言传给上游的路径与参数(first/mkt/adlt/mmasync 等)。
func TestV7VerticalEndToEnd(t *testing.T) {
	up, gotPath, gotQuery := newUpstream(fxImagesAsync)
	defer up.Close()
	s := &server{engine: &BingEngine{Base: up.URL, Client: up.Client()}}
	mux := s.routes()

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet,
		"/v7/images/search?q=cat&count=2&offset=20&mkt=en-US&safeSearch=Strict", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("images 全链路应 200: %d %s", w.Code, w.Body.String())
	}
	var d map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &d); err != nil {
		t.Fatalf("响应不是合法 JSON: %v", err)
	}
	if d["_type"] != "Images" || len(d["value"].([]any)) != 2 {
		t.Fatalf("Images 响应错误: %v", d)
	}
	if *gotPath != "/images/async" {
		t.Fatalf("上游路径错误: %q", *gotPath)
	}
	q := *gotQuery
	if q.Get("first") != "21" || q.Get("count") != "2" || q.Get("mkt") != "en-US" ||
		q.Get("adlt") != "strict" || q.Get("mmasync") != "1" {
		t.Fatalf("上游参数错误(offset=20 → first=21): %v", q)
	}
}

// TestV7VerticalNewsEndToEnd 新闻 RSS 全链路(验证 format=rss 且不传 mkt)
func TestV7VerticalNewsEndToEnd(t *testing.T) {
	up, gotPath, gotQuery := newUpstream(fxNewsRSS)
	defer up.Close()
	s := &server{engine: &BingEngine{Base: up.URL, Client: up.Client()}}
	mux := s.routes()

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v7/news/search?q=cat&mkt=en-US", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("news 全链路应 200: %d %s", w.Code, w.Body.String())
	}
	var d map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &d)
	if d["_type"] != "News" || len(d["value"].([]any)) != 2 {
		t.Fatalf("News 响应错误: %v", d)
	}
	if *gotPath != "/news/search" {
		t.Fatalf("news 上游路径错误: %q", *gotPath)
	}
	if (*gotQuery).Get("format") != "rss" || (*gotQuery).Get("mkt") != "" {
		t.Fatalf("news 不应向 RSS 端点传 mkt: %v", *gotQuery)
	}
}

// TestV7VerticalDictEndToEnd 词典全链路(经 BING_DICT_BASE 指向伪上游)
func TestV7VerticalDictEndToEnd(t *testing.T) {
	up, gotPath, gotQuery := newUpstream(fxDictEN)
	defer up.Close()
	t.Setenv("BING_DICT_BASE", up.URL)
	s := &server{engine: &BingEngine{Base: "https://www.bing.com", Client: up.Client()}}
	mux := s.routes()

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v7/dict/search?q=hello", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("dict 全链路应 200: %d %s", w.Code, w.Body.String())
	}
	var d map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &d)
	if d["_type"] != "Dict" || d["word"] != "hello" {
		t.Fatalf("Dict 响应错误: %v", d)
	}
	if *gotPath != "/bilingualdictionary/search" {
		t.Fatalf("dict 上游路径错误: %q", *gotPath)
	}
	if (*gotQuery).Get("setlang") != "zh-hans" {
		t.Fatalf("dict 上游 setlang 错误: %v", *gotQuery)
	}
}

// TestSearXNGVerticalDispatch SearXNG /search categories 分发全链路
func TestSearXNGVerticalDispatch(t *testing.T) {
	t.Setenv("ENABLE_SEARXNG", "1")
	up, gotPath, _ := newUpstream(fxImagesAsync)
	defer up.Close()
	s := &server{engine: &BingEngine{Base: up.URL, Client: up.Client()}}
	mux := s.routes()

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet,
		"/search?q=cat&categories=images&count=2&page=2", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("SearXNG 垂直分发应 200: %d %s", w.Code, w.Body.String())
	}
	var d SearchResponse
	if err := json.Unmarshal(w.Body.Bytes(), &d); err != nil {
		t.Fatalf("响应不是 SearXNG 结构: %v", err)
	}
	if d.Category != "images" || d.NumberOfResults != 2 {
		t.Fatalf("category/条数错误: %+v", d)
	}
	first := d.Results[0]
	if first.Template != "images.html" || first.ImgSrc != "https://cdn.example.com/cat.jpg" ||
		first.ThumbnailSrc != "https://ts4.mm.bing.net/th?id=OIP.abc&pid=15.1" {
		t.Fatalf("图片垂直字段错误: %+v", first)
	}
	if first.Position != 3 { // page=2,count=2 → 起始 position=3
		t.Fatalf("position 应为分页全局序号: %d", first.Position)
	}
	if first.Engines[0] != "bing" || first.Engine != "bing" {
		t.Fatalf("引擎字段错误: %+v", first)
	}
	if *gotPath != "/images/async" {
		t.Fatalf("分发上游路径错误: %q", *gotPath)
	}
}

func TestV7VerticalAuth(t *testing.T) {
	t.Setenv("BING_API_KEY", "secret123")
	s := newTestServer()
	mux := s.routes()
	// 无密钥头 → 401 官方 UnauthorizedAccess
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v7/images/search?q=x", nil))
	if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), "UnauthorizedAccess") {
		t.Fatalf("无密钥应 401: %d %s", w.Code, w.Body.String())
	}
	// 错误密钥 → 401
	w = httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v7/images/search?q=x", nil)
	r.Header.Set("Ocp-Apim-Subscription-Key", "wrong")
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("错误密钥应 401: %d", w.Code)
	}
	// 正确密钥(官方头)→ 通过鉴权,走到参数校验(缺 q → 400)
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "/v7/videos/search", nil)
	r.Header.Set("Ocp-Apim-Subscription-Key", "secret123")
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "ParameterMissing") {
		t.Fatalf("正确密钥应放行到参数校验: %d %s", w.Code, w.Body.String())
	}
	// 正确密钥(subscription-key 查询参数,APIM 方式)
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "/v7/dict/search?subscription-key=secret123", nil)
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "ParameterMissing") {
		t.Fatalf("APIM 参数密钥应放行: %d %s", w.Code, w.Body.String())
	}
}

// ── v7 垂直响应组装 ───────────────────────────────────────────

func TestBuildBingV7ImagesResponse(t *testing.T) {
	p := v7Params{Q: "cat", Count: 2, Offset: 10}
	items := []ImageItem{
		{Title: "A Cat", PageURL: "https://example.com/cats", ImageURL: "https://cdn.example.com/cat.jpg",
			ThumbURL: "https://tse.mm.bing.net/th?id=1", Width: 242, Height: 180, MID: "MID1"},
	}
	resp := buildBingV7ImagesResponse(p, "en-US", items)
	if resp.Type != "Images" {
		t.Fatalf("_type 应为 Images: %q", resp.Type)
	}
	if resp.CurrentOffset != 10 || resp.NextOffset != 11 {
		t.Fatalf("offset 字段错误: cur=%d next=%d", resp.CurrentOffset, resp.NextOffset)
	}
	if resp.TotalEstimatedMatches != 11 {
		t.Fatalf("totalEstimatedMatches 应为 offset+条数兜底: %d", resp.TotalEstimatedMatches)
	}
	if !strings.Contains(resp.ReadLink, "images/search?q=cat") {
		t.Fatalf("readLink 应指向官方形态: %q", resp.ReadLink)
	}
	v := resp.Value[0]
	if v.Name != "A Cat" || v.ContentURL != "https://cdn.example.com/cat.jpg" || v.HostPageURL != "https://example.com/cats" {
		t.Fatalf("图片字段错误: %+v", v)
	}
	if v.EncodingFormat != "jpeg" {
		t.Fatalf("encodingFormat 推断错误: %q", v.EncodingFormat)
	}
	if v.Thumbnail.Width != 242 || v.Thumbnail.Height != 180 || v.Thumbnail.ContentURL == "" {
		t.Fatalf("thumbnail 错误: %+v", v.Thumbnail)
	}
	if !strings.Contains(v.WebSearchURL, "view=detailv2") || !strings.Contains(v.WebSearchURL, "MID1") {
		t.Fatalf("webSearchUrl 应为详情页形态: %q", v.WebSearchURL)
	}
	if v.ImageInsightsToken != "ccid_MID1" {
		t.Fatalf("imageInsightsToken 错误: %q", v.ImageInsightsToken)
	}
	// JSON 键名与官方对齐
	b, _ := json.Marshal(resp)
	s := string(b)
	for _, key := range []string{`"_type":"Images"`, `"queryContext"`, `"readLink"`, `"currentOffset"`,
		`"nextOffset"`, `"totalEstimatedMatches"`, `"value"`, `"webSearchUrl"`, `"thumbnailUrl"`,
		`"contentUrl"`, `"hostPageUrl"`, `"hostPageDisplayUrl"`, `"encodingFormat"`, `"thumbnail"`,
		`"imageInsightsToken"`} {
		if !strings.Contains(s, key) {
			t.Fatalf("官方 Images 响应缺少键 %s", key)
		}
	}
}

func TestBuildBingV7VideosResponse(t *testing.T) {
	p := v7Params{Q: "cat", Count: 2, Offset: 0}
	items := []VideoItem{
		{Title: "Cat Video", PageURL: "https://www.youtube.com/watch?v=abc", ContentURL: "https://www.youtube.com/watch?v=abc",
			Duration: "4:18", ThumbURL: "https://th.bing.com/th?id=1", MID: "VID1", ThumbID: "OVP.1"},
	}
	resp := buildBingV7VideosResponse(p, "", items)
	if resp.Type != "Videos" {
		t.Fatalf("_type 应为 Videos: %q", resp.Type)
	}
	v := resp.Value[0]
	if v.Duration != "PT4M18S" {
		t.Fatalf("时长应为 ISO 8601: %q", v.Duration)
	}
	if len(v.Publisher) != 1 || v.Publisher[0].Type != "Organization" || v.Publisher[0].Name != "www.youtube.com" {
		t.Fatalf("publisher 错误: %+v", v.Publisher)
	}
	if v.VideoID != "OVP.1" || !v.AllowHTTPSEmbed {
		t.Fatalf("videoId/allowHttpsEmbed 错误: %+v", v)
	}
	if !strings.Contains(v.WebSearchURL, "view=detail") {
		t.Fatalf("webSearchUrl 错误: %q", v.WebSearchURL)
	}
	b, _ := json.Marshal(resp)
	for _, key := range []string{`"_type":"Videos"`, `"duration"`, `"publisher"`, `"contentUrl"`,
		`"hostPageUrl"`, `"allowHttpsEmbed"`, `"videoId"`, `"thumbnailUrl"`} {
		if !strings.Contains(string(b), key) {
			t.Fatalf("官方 Videos 响应缺少键 %s", key)
		}
	}
}

func TestBuildBingV7NewsResponse(t *testing.T) {
	p := v7Params{Q: "cat", Count: 10, Offset: 0}
	items := []NewsItem{
		{Title: "Cat Scratch Disease", URL: "https://www.hopkinsmedicine.org/a",
			Desc: "What is cat scratch disease?", Date: "Thu, 05 Sep 2024 17:01:00 GMT", Source: "Johns Hopkins"},
	}
	resp := buildBingV7NewsResponse(p, "en-US", items)
	if resp.Type != "News" {
		t.Fatalf("_type 应为 News: %q", resp.Type)
	}
	n := resp.Value[0]
	if n.Type != "NewsArticle" || n.Name != "Cat Scratch Disease" {
		t.Fatalf("NewsArticle 字段错误: %+v", n)
	}
	if n.DatePublished != "2024-09-05T17:01:00Z" {
		t.Fatalf("datePublished 应为 ISO 8601: %q", n.DatePublished)
	}
	if len(n.Provider) != 1 || n.Provider[0].Name != "Johns Hopkins" {
		t.Fatalf("provider 错误: %+v", n.Provider)
	}
	if n.Headline {
		t.Fatal("headline 默认应为 false")
	}
}

func TestBuildBingV7DictResponse(t *testing.T) {
	p := v7Params{Q: "hello"}
	entry := &DictEntry{
		Word:   "hello",
		PronUS: "heˈləʊ", PronUK: "həˈləʊ",
		Def:      DictSense{Pos: "int.", Text: "你好；喂"},
		Senses:   []DictSense{{Text: "1.（用于问候）哈啰"}, {Text: "2.（表示惊讶）嘿"}},
		Examples: []string{"I walked up to say hello."},
	}
	resp := buildBingV7DictResponse(p, entry)
	if resp.Type != "Dict" {
		t.Fatalf("_type 应为 Dict: %q", resp.Type)
	}
	if resp.Word != "hello" || resp.Pronunciation.US != "heˈləʊ" {
		t.Fatalf("词条/音标错误: %+v", resp)
	}
	// 首要释义 + 编号义项共 3 条;例句挂在首条
	if len(resp.Value) != 3 {
		t.Fatalf("value 应含首要释义与编号义项: %d", len(resp.Value))
	}
	if resp.Value[0].Pos != "int." || resp.Value[0].Def != "你好；喂" {
		t.Fatalf("首条错误: %+v", resp.Value[0])
	}
	if len(resp.Value[0].Examples) != 1 {
		t.Fatalf("例句应挂在首条: %+v", resp.Value[0])
	}
	// 未收录 → 空 value
	empty := buildBingV7DictResponse(p, nil)
	if empty.Word != "" || len(empty.Value) != 0 {
		t.Fatalf("未收录应为空 value: %+v", empty)
	}
}

// ── SearXNG /search 垂直分发(离线,断在触网之前)──────────────

func TestHandleSearchCategoryErrors(t *testing.T) {
	t.Setenv("ENABLE_SEARXNG", "1")
	s := newTestServer()
	mux := s.routes()
	// 不支持的类别 → 400 + 原因说明
	for _, cat := range []string{"academic", "shopping", "maps", "unknown"} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/search?q=x&categories="+cat, nil)
		mux.ServeHTTP(w, r)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("categories=%s 应 400: %d %s", cat, w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "不支持的搜索类别") {
			t.Fatalf("应含类别错误说明: %s", w.Body.String())
		}
	}
	// academic 应说明客户端渲染原因
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/search?q=x&categories=academic", nil))
	if !strings.Contains(w.Body.String(), "客户端") {
		t.Fatalf("academic 错误应说明原因: %s", w.Body.String())
	}
}

func TestHandleSearchCategoryParamSources(t *testing.T) {
	t.Setenv("ENABLE_SEARXNG", "1")
	s := newTestServer()
	// category 与 categories 两个参数名均生效(经"未知类别 400"反向验证:
	// 合法类别会走到触网,这里用非法值验证参数确实被读取)
	w := httptest.NewRecorder()
	mux := s.routes()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/search?q=x&category=nope", nil))
	if !strings.Contains(w.Body.String(), "nope") {
		t.Fatalf("category 参数应被读取并回显在错误中: %s", w.Body.String())
	}
	// POST JSON body 携带 categories
	w = httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(`{"q":"x","categories":"badcat"}`))
	r.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(w, r)
	if !strings.Contains(w.Body.String(), "badcat") {
		t.Fatalf("JSON body 的 categories 应被读取: %s", w.Body.String())
	}
}

// sliceByOffset 单页切片语义
func TestSliceByOffset(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}
	if got := sliceByOffset(items, 0, 3); len(got) != 3 || got[0] != 1 || got[2] != 3 {
		t.Fatalf("[0:3] 错误: %v", got)
	}
	if got := sliceByOffset(items, 2, 10); len(got) != 3 || got[0] != 3 {
		t.Fatalf("[2:12] 应截到末尾: %v", got)
	}
	if got := sliceByOffset(items, 5, 3); len(got) != 0 {
		t.Fatalf("offset 越界应为空: %v", got)
	}
	if got := sliceByOffset([]int{}, 0, 5); len(got) != 0 {
		t.Fatalf("空输入应为空: %v", got)
	}
}
