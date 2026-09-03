package main

// vertical.go — Bing 垂直搜索抓取与解析(图片/视频/新闻/词典)。
//
// 各垂直的数据来源(均为服务端可直取的稳定结构,与学术/购物等纯客户端
// 渲染页面不同,详见 README「垂直搜索」一节):
//
//   images  GET {base}/images/async?q=&first=&count=&mmasync=1
//           → <a class="iusc" m="{murl,purl,turl,t,desc…}">(转义 JSON)
//   videos  GET {base}/videos/search?q=&mkt=
//           → <div … vrhm="{vt,du,murl,pgurl,smturl,thid…}">(转义 JSON)
//   news    GET {base}/news/search?q=&format=rss
//           → RSS <item>:title/link/description/pubDate/News:Source
//   dict    GET https://cn.bing.com/bilingualdictionary/search?q=&setlang=zh-hans
//           → 头词 #headword + 音标 hd_prUS/hd_pr(拼音 hd_p1_1)
//             + 首义项 <span class="pos">/<span class="def"> + se_lis/crossid 义项
//
// 分页说明:
//   - images:async 端点支持 first(1 基),offset 换算为 first=offset+1
//   - videos:SERP 单页约 50 条,offset/count 在该页内切片(无跨页翻页)
//   - news:RSS 固定一批(约 11~15 条),不支持分页
//   - dict:词条查询,无分页

import (
	"context"
	"encoding/json"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ── 垂直类别 ──────────────────────────────────────────────────

// 支持的垂直类别(空串 = 网页综合搜索,与现有行为一致)
const (
	categoryImages = "images"
	categoryVideos = "videos"
	categoryNews   = "news"
	categoryDict   = "dict"
)

// dictBaseURL 词典固定走 cn.bing.com(www 域不服务 bilingualdictionary),
// 可用 BING_DICT_BASE 覆盖(自建反代等场景)。
func dictBaseURL() string {
	return envOr("BING_DICT_BASE", "https://cn.bing.com")
}

// normalizeCategory 归一化搜索类别,返回空串表示网页综合搜索。
// 无法服务端抓取的类别(学术/购物/地图)与未知值返回错误并说明原因。
func normalizeCategory(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	// categories=general,images 形式:取第一个 token
	if i := strings.IndexAny(s, ",;"); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	switch s {
	case "", "general", "web", "all":
		return "", nil
	case "images", "image", "photos", "pictures", "pic":
		return categoryImages, nil
	case "videos", "video":
		return categoryVideos, nil
	case "news":
		return categoryNews, nil
	case "dict", "dictionary", "definitions", "define":
		return categoryDict, nil
	case "academic", "scholar", "science", "literature", "论文", "文献":
		return "", errUnsupportedCategory("academic/学术",
			"Bing 学术搜索页面为纯客户端(JS)渲染,服务端抓取拿不到结果数据")
	case "shopping", "shop", "shoppingoffers":
		return "", errUnsupportedCategory("shopping/购物",
			"Bing 购物页面为纯客户端(JS)渲染,服务端抓取拿不到结果数据")
	case "maps", "map":
		return "", errUnsupportedCategory("maps/地图",
			"Bing 地图依赖交互式前端,无静态结果可抓取")
	default:
		return "", errUnsupportedCategory(s,
			"未知搜索类别,支持:general(默认)/images/videos/news/dict")
	}
}

// errUnsupportedCategory 构造带说明的类别错误
func errUnsupportedCategory(cat, why string) error {
	return &categoryError{Category: cat, Reason: why}
}

// categoryError 类别错误(供 handler 组装友好提示)
type categoryError struct {
	Category string
	Reason   string
}

func (e *categoryError) Error() string {
	return "不支持的搜索类别 " + e.Category + ":" + e.Reason
}

// ── 垂直结果结构 ──────────────────────────────────────────────

// pickCategory 优先取单数 category,否则取复数 categories 的第一个 token
// (SearXNG 协议使用 categories=images 形式)
func pickCategory(category, categories string) string {
	if c := strings.TrimSpace(category); c != "" {
		return c
	}
	if c := strings.TrimSpace(categories); c != "" {
		if i := strings.IndexAny(c, ",;"); i >= 0 {
			return strings.TrimSpace(c[:i])
		}
		return c
	}
	return ""
}

// searchVertical SearXNG 风格 /search 的垂直搜索分发:抓取垂直结果并
// 转成 SearXNG Result 形态(带 template/img_src/thumbnail_src 等垂直字段,
// 字段命名与 SearXNG 垂直引擎一致)。
func (s *server) searchVertical(ctx context.Context, p searchParams, category, market string) (*SearchResponse, error) {
	resp := &SearchResponse{
		Query:               p.Q,
		Category:            category,
		Results:             []Result{},
		Answers:             []string{},
		Corrections:         []string{},
		Infoboxes:           []string{},
		Suggestions:         []string{},
		UnresponsiveEngines: []string{},
	}

	switch category {
	case categoryImages:
		items, err := s.engine.SearchImages(ctx, p.Q, market, (p.Page-1)*p.Count+1, p.Count, false)
		if err != nil {
			return nil, err
		}
		offset := (p.Page - 1) * p.Count
		for i, it := range sliceByOffset(items, 0, p.Count) {
			resp.Results = append(resp.Results, Result{
				Title: it.Title, URL: it.PageURL, Content: it.Desc,
				Engine: "bing", Engines: []string{"bing"}, Score: 1.0, Position: offset + i + 1,
				Template: "images.html", ImgSrc: it.ImageURL, ThumbnailSrc: it.ThumbURL,
			})
		}
	case categoryVideos:
		items, err := s.engine.SearchVideos(ctx, p.Q, market, false)
		if err != nil {
			return nil, err
		}
		offset := (p.Page - 1) * p.Count
		for i, it := range sliceByOffset(items, offset, p.Count) {
			resp.Results = append(resp.Results, Result{
				Title: it.Title, URL: it.PageURL, Content: it.ContentURL,
				Engine: "bing", Engines: []string{"bing"}, Score: 1.0, Position: offset + i + 1,
				Template: "videos.html", ThumbnailSrc: it.ThumbURL, Length: it.Duration,
			})
		}
	case categoryNews:
		items, err := s.engine.SearchNews(ctx, p.Q, market)
		if err != nil {
			return nil, err
		}
		for i, it := range items {
			resp.Results = append(resp.Results, Result{
				Title: it.Title, URL: it.URL, Content: it.Desc,
				Engine: "bing", Engines: []string{"bing"}, Score: 1.0, Position: i + 1,
				PublishedDate: rssDateToISO(it.Date), Source: it.Source,
			})
		}
	case categoryDict:
		entry, err := s.engine.SearchDict(ctx, p.Q)
		if err != nil {
			return nil, err
		}
		dictURL := dictBaseURL() + "/dict/search?q=" + url.QueryEscape(p.Q)
		if entry != nil {
			// answers:SearXNG 即时答案位,放词条摘要
			pron := ""
			if entry.PronUS != "" || entry.PronUK != "" || entry.Pinyin != "" {
				pron = " ["
				if entry.Pinyin != "" {
					pron += entry.Pinyin
				} else {
					if entry.PronUS != "" {
						pron += "美 " + entry.PronUS
					}
					if entry.PronUK != "" {
						if entry.PronUS != "" {
							pron += " / "
						}
						pron += "英 " + entry.PronUK
					}
				}
				pron += "]"
			}
			summary := entry.Word + pron
			if entry.Def.Pos != "" || entry.Def.Text != "" {
				summary += " " + entry.Def.Pos + " " + entry.Def.Text
			}
			resp.Answers = append(resp.Answers, summary)
			// results:每个义项一条
			for i, sense := range entry.Senses {
				resp.Results = append(resp.Results, Result{
					Title: entry.Word, URL: dictURL, Content: sense.Text,
					Engine: "bing", Engines: []string{"bing"}, Score: 1.0, Position: i + 1,
				})
			}
			// 没有编号义项时至少返回首要释义
			if len(entry.Senses) == 0 && entry.Def.Text != "" {
				resp.Results = append(resp.Results, Result{
					Title: entry.Word, URL: dictURL, Content: entry.Def.Text,
					Engine: "bing", Engines: []string{"bing"}, Score: 1.0, Position: 1,
				})
			}
		}
	}
	resp.NumberOfResults = len(resp.Results)
	return resp, nil
}

// ImageItem 单条图片结果
type ImageItem struct {
	Title    string // 图片标题
	PageURL  string // 来源页面(purl)
	ImageURL string // 原图直链(murl)
	ThumbURL string // 缩略图(turl)
	Desc     string // 描述
	Width    int    // 缩略图宽(卡片尺寸,尽力解析)
	Height   int    // 缩略图高
	MID      string // Bing 图片媒体 id(供 v7 webSearchUrl)
}

// VideoItem 单条视频结果
type VideoItem struct {
	Title      string // 标题(vt)
	PageURL    string // 宿主页链接(pgurl)
	ContentURL string // 视频地址(murl,常为 youtube 等直链)
	Duration   string // 时长 mm:ss 或 h:mm:ss
	ThumbURL   string // 静态缩略图(smturl)
	ThumbID    string // Bing 缩略图 id(thid)
	MID        string // 媒体 id
}

// NewsItem 单条新闻结果
type NewsItem struct {
	Title  string
	URL    string // 真实文章链接(从 Bing apiclick 重定向还原)
	Desc   string
	Date   string // 发布时间(RFC 1123 原样,如 "Thu, 05 Sep 2024 17:01:00 GMT")
	Source string // 来源媒体(News:Source)
}

// DictSense 词典一个义项
type DictSense struct {
	Pos  string // 词性(int./n./na. …),可为空
	Text string // 释义文本(含编号与中英对照,已清洗)
}

// DictEntry 词典词条
type DictEntry struct {
	Word     string      // 头词
	PronUS   string      // 美式音标(不含方括号,空 = 无)
	PronUK   string      // 英式音标
	Pinyin   string      // 中文词的拼音(不含方括号)
	Def      DictSense   // 首要释义(汉英/英汉主释义)
	Senses   []DictSense // 编号义项(英文词:se_lis;中文词:crossid 区)
	Examples []string    // 例句(尽力解析,最多 4 条)
}

// ── 图片:抓取与解析 ───────────────────────────────────────────

var (
	// iusc 锚点的 class 标记(以此为切块起点)
	reIuscClass = regexp.MustCompile(`class="[^"]*\biusc\b[^"]*"`)
	// m 属性(JSON 元数据,实体转义,值内不含裸引号,
	// 即使含裸标签也不会截断捕获)
	reAttrM = regexp.MustCompile(`\bm="([^"]*)"`)
	// 同一锚点块内的卡片 style 尺寸
	reAttrStyle = regexp.MustCompile(`\bstyle="([^"]*)"`)
	// 卡片 style 尺寸:height:180px;width:242px(顺序不定,需按属性名归位)
	reStyleWxH = regexp.MustCompile(`(?i)(?:^|;)\s*(width|height)\s*:\s*(\d+)px`)
)

// bingImageMeta iusc 锚点 m 属性里的 JSON(键名为 Bing 原始小写)
type bingImageMeta struct {
	Murl string `json:"murl"`
	Purl string `json:"purl"`
	Turl string `json:"turl"`
	T    string `json:"t"`
	Desc string `json:"desc"`
	MID  string `json:"mid"`
}

// SearchImages 抓取 Bing 图片搜索异步端点并解析。
// first 为 1 基偏移(v7 offset 换算 first=offset+1);count 为条数(1~35)。
func (e *BingEngine) SearchImages(ctx context.Context, term, market string, first, count int, safeStrict bool) ([]ImageItem, error) {
	params := url.Values{}
	params.Set("q", term)
	params.Set("first", strconv.Itoa(first))
	if count > 0 {
		params.Set("count", strconv.Itoa(count))
	}
	params.Set("mmasync", "1")
	if market != "" {
		params.Set("mkt", market)
		params.Set("setlang", strings.ToLower(strings.SplitN(market, "-", 2)[0]))
	}
	if safeStrict {
		params.Set("adlt", "strict")
	}
	body, err := e.fetchBing(ctx, e.Base, "/images/async", params, acceptLanguageFor(market))
	if err != nil {
		return nil, err
	}
	return parseImagesAsync(string(body)), nil
}

// parseImagesAsync 解析 images/async 响应中的 iusc 锚点。
// 按锚点切块(本锚点到下一锚点),块内提取 m/style 属性,
// m 值即使含裸标签字符也不会截断捕获。
func parseImagesAsync(page string) []ImageItem {
	var items []ImageItem
	starts := reIuscClass.FindAllStringIndex(page, -1)
	for i, loc := range starts {
		end := len(page)
		if i+1 < len(starts) {
			end = starts[i+1][0]
		}
		block := page[loc[1]:end]
		mm := reAttrM.FindStringSubmatch(block)
		if mm == nil {
			continue
		}
		var meta bingImageMeta
		if err := json.Unmarshal([]byte(html.UnescapeString(mm[1])), &meta); err != nil {
			continue
		}
		if meta.Murl == "" && meta.Purl == "" {
			continue
		}
		item := ImageItem{
			Title:    cleanText(meta.T),
			PageURL:  meta.Purl,
			ImageURL: meta.Murl,
			ThumbURL: meta.Turl,
			Desc:     cleanText(meta.Desc),
			MID:      meta.MID,
		}
		// 卡片 style 尺寸 → 缩略图尺寸(尽力而为,按属性名归位)
		if sm := reAttrStyle.FindStringSubmatch(block); sm != nil {
			for _, d := range reStyleWxH.FindAllStringSubmatch(sm[1], -1) {
				n, _ := strconv.Atoi(d[2])
				switch strings.ToLower(d[1]) {
				case "width":
					item.Width = n
				case "height":
					item.Height = n
				}
			}
		}
		if item.ThumbURL == "" && item.ImageURL != "" {
			item.ThumbURL = item.ImageURL // 兜底:无缩略图用原图
		}
		items = append(items, item)
	}
	return items
}

// ── 视频:抓取与解析 ───────────────────────────────────────────

var reVrhmAttr = regexp.MustCompile(`vrhm="([^"]*)"`)

// bingVideoMeta 视频卡片 vrhm 属性里的 JSON(键名为 Bing 原始小写)
type bingVideoMeta struct {
	VT     string `json:"vt"`
	Murl   string `json:"murl"`
	Pgurl  string `json:"pgurl"`
	Du     string `json:"du"`
	Smturl string `json:"smturl"`
	Thid   string `json:"thid"`
	MID    string `json:"mid"`
}

// SearchVideos 抓取 Bing 视频搜索页并解析。
// SERP 单页约 50 条,offset/count 由调用方在返回切片上做。
func (e *BingEngine) SearchVideos(ctx context.Context, term, market string, safeStrict bool) ([]VideoItem, error) {
	params := url.Values{}
	params.Set("q", term)
	if market != "" {
		params.Set("mkt", market)
		params.Set("setlang", strings.ToLower(strings.SplitN(market, "-", 2)[0]))
	}
	if safeStrict {
		params.Set("adlt", "strict")
	}
	body, err := e.fetchBing(ctx, e.Base, "/videos/search", params, acceptLanguageFor(market))
	if err != nil {
		return nil, err
	}
	return parseVideosPage(string(body)), nil
}

// parseVideosPage 解析视频 SERP 中的 vrhm 元数据块(去重)。
func parseVideosPage(page string) []VideoItem {
	var items []VideoItem
	seen := map[string]bool{}
	for _, vm := range reVrhmAttr.FindAllStringSubmatch(page, -1) {
		var meta bingVideoMeta
		if err := json.Unmarshal([]byte(html.UnescapeString(vm[1])), &meta); err != nil {
			continue
		}
		if (meta.VT == "" && meta.Pgurl == "" && meta.Murl == "") || seen[meta.Murl+meta.Pgurl] {
			continue
		}
		seen[meta.Murl+meta.Pgurl] = true
		items = append(items, VideoItem{
			Title:      cleanText(meta.VT),
			PageURL:    meta.Pgurl,
			ContentURL: meta.Murl,
			Duration:   strings.TrimSpace(meta.Du),
			ThumbURL:   meta.Smturl,
			ThumbID:    meta.Thid,
			MID:        meta.MID,
		})
	}
	return items
}

// ── 新闻:抓取与解析(RSS)──────────────────────────────────────

var (
	reRSSItem   = regexp.MustCompile(`(?s)<item>(.*?)</item>`)
	reRSSTitle  = regexp.MustCompile(`(?s)<title>(.*?)</title>`)
	reRSSLink   = regexp.MustCompile(`(?s)<link>(.*?)</link>`)
	reRSSDesc   = regexp.MustCompile(`(?s)<description>(.*?)</description>`)
	reRSSDate   = regexp.MustCompile(`(?s)<pubDate>(.*?)</pubDate>`)
	reRSSSource = regexp.MustCompile(`(?s)<News:Source>(.*?)</News:Source>`)
	reApiclick  = regexp.MustCompile(`[?&]url=([^&]+)`)
)

// SearchNews 抓取 Bing 新闻搜索的 RSS 输出并解析。
// 注意:RSS 端点对 mkt 参数敏感(传 mkt 可能返回空),语言仅通过
// Accept-Language 头表达,由 Bing 按查询词与出口 IP 自行匹配。
func (e *BingEngine) SearchNews(ctx context.Context, term, language string) ([]NewsItem, error) {
	params := url.Values{}
	params.Set("q", term)
	params.Set("format", "rss")
	body, err := e.fetchBing(ctx, e.Base, "/news/search", params, acceptLanguageFor(language))
	if err != nil {
		return nil, err
	}
	return parseNewsRSS(string(body)), nil
}

// parseNewsRSS 解析新闻 RSS 的 <item> 列表。
func parseNewsRSS(page string) []NewsItem {
	var items []NewsItem
	for _, im := range reRSSItem.FindAllStringSubmatch(page, -1) {
		block := im[1]
		item := NewsItem{
			Title:  cleanText(stripCDATA(firstGroup(reRSSTitle, block))),
			URL:    cleanText(stripCDATA(firstGroup(reRSSLink, block))),
			Desc:   cleanText(stripCDATA(firstGroup(reRSSDesc, block))),
			Date:   cleanText(stripCDATA(firstGroup(reRSSDate, block))),
			Source: cleanText(stripCDATA(firstGroup(reRSSSource, block))),
		}
		item.URL = decodeNewsRedirect(item.URL)
		if item.Title == "" && item.URL == "" {
			continue
		}
		items = append(items, item)
	}
	return items
}

// stripCDATA 去掉 RSS 文本的 CDATA 包裹与转义
func stripCDATA(s string) string {
	if s == "" {
		return s
	}
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "<![CDATA[") && strings.HasSuffix(s, "]]>") {
		s = s[len("<![CDATA[") : len(s)-len("]]>")]
	}
	return html.UnescapeString(s)
}

// decodeNewsRedirect 还原新闻 RSS 链接里的 Bing apiclick 重定向:
// http://www.bing.com/news/apiclick.aspx?...&url=https%3a%2f%2fexample.com%2fa...
func decodeNewsRedirect(link string) string {
	if link == "" || !strings.Contains(link, "apiclick") {
		return link
	}
	m := reApiclick.FindStringSubmatch(link)
	if m == nil {
		return link
	}
	if dec, err := url.QueryUnescape(m[1]); err == nil && strings.HasPrefix(dec, "http") {
		return dec
	}
	return link
}

// firstGroup 取正则第一个捕获组,未匹配返回空串
func firstGroup(re *regexp.Regexp, s string) string {
	if m := re.FindStringSubmatch(s); m != nil {
		return m[1]
	}
	return ""
}

// ── 词典:抓取与解析 ───────────────────────────────────────────

var (
	reDictWord   = regexp.MustCompile(`(?s)id="headword"[^>]*>\s*<h1[^>]*>\s*<strong[^>]*>(.*?)</strong>`)
	reDictPrUS   = regexp.MustCompile(`(?s)<div[^>]*class="[^"]*\bhd_prUS\b[^"]*"[^>]*>(.*?)</div>`)
	reDictPrUK   = regexp.MustCompile(`(?s)<div[^>]*class="[^"]*\bhd_pr\b[^"]*"[^>]*>(.*?)</div>`)
	reDictPinyin = regexp.MustCompile(`(?s)<div[^>]*class="[^"]*\bhd_p1_1\b[^"]*"[^>]*>(.*?)</div>`)
	// 首要释义:hd_area 之后的第一对 <span class="pos">…</span><span class="def…">…</li>
	reDictPrimary = regexp.MustCompile(`(?s)<span class="pos">([^<]*)</span>\s*<span class="def[^"]*">(.*?)</li>`)
	// 英文词布局:编号义项块
	reDictSense = regexp.MustCompile(`(?s)<div class="se_lis">.*?</table>`)
	// 中文词布局:crossid 区内的编号单元格标记(区界由 regionAfter 手工切)
	reDictSeD = regexp.MustCompile(`class="se_d b_primtxt"`)
	// 例句(英文词页面)
	reDictExample = regexp.MustCompile(`(?s)<div[^>]*class="[^"]*\bsen_en\b[^"]*"[^>]*>(.*?)</div>`)
)

// regionAfter 取 startMark 起始、至下一个 <div id= 边界(或 max 字节)的区域。
// 用手工切片而非正则量词(Go regexp 重复计数上限 1000)。
func regionAfter(page, startMark string, max int) (string, bool) {
	i := strings.Index(page, startMark)
	if i < 0 {
		return "", false
	}
	rest := page[i:]
	if len(rest) > max {
		rest = rest[:max]
	}
	if j := strings.Index(rest[len(startMark):], `<div id="`); j >= 0 {
		rest = rest[:len(startMark)+j]
	}
	return rest, true
}

// SearchDict 查询 Bing 词典(中英双向:英文词→中文释义,中文词→英文释义)。
// 词典固定使用 cn.bing.com 的 bilingualdictionary 端点。
func (e *BingEngine) SearchDict(ctx context.Context, term string) (*DictEntry, error) {
	params := url.Values{}
	params.Set("q", term)
	params.Set("setlang", "zh-hans")
	body, err := e.fetchBing(ctx, dictBaseURL(), "/bilingualdictionary/search", params, "zh-CN,zh;q=0.9,en;q=0.5")
	if err != nil {
		return nil, err
	}
	entry := parseDictPage(string(body))
	return entry, nil
}

// parseDictPage 解析词典页(兼容英文词/中文词两种布局),未收录返回 nil。
func parseDictPage(page string) *DictEntry {
	wm := reDictWord.FindStringSubmatch(page)
	if wm == nil {
		return nil
	}
	entry := &DictEntry{Word: cleanText(wm[1])}
	if entry.Word == "" {
		return nil
	}
	entry.PronUS = normalizePron(cleanText(firstGroup(reDictPrUS, page)))
	entry.PronUK = normalizePron(cleanText(firstGroup(reDictPrUK, page)))
	// 拼音仅在 hd_p1_1 为纯文本时成立(英文词页面该容器内嵌的是音标 div,
	// 含子标签时应跳过,避免把美式音标误报为拼音)
	if pm := reDictPinyin.FindStringSubmatch(page); pm != nil && !strings.Contains(pm[1], "<") {
		entry.Pinyin = stripBrackets(cleanText(pm[1]))
	}

	// 首要释义:<span class="pos">词性</span><span class="def…">释义</span>
	if pm := reDictPrimary.FindStringSubmatch(page); pm != nil {
		entry.Def = DictSense{Pos: cleanText(pm[1]), Text: tidyDictText(cleanText(pm[2]))}
	}

	// 编号义项:
	// 1) 英文词布局:se_lis 块(含中文释义+英文例解)
	for _, sm := range reDictSense.FindAllStringSubmatch(page, -1) {
		if t := tidyDictText(cleanText(sm[0])); t != "" {
			entry.Senses = append(entry.Senses, DictSense{Text: t})
		}
	}
	// 2) 中文词布局:crossid 区内按 se_d 编号项切分
	if len(entry.Senses) == 0 {
		if region, ok := regionAfter(page, `id="crossid"`, 12000); ok {
			marks := reDictSeD.FindAllStringIndex(region, -1)
			for i, mk := range marks {
				end := len(region)
				if i+1 < len(marks) {
					end = marks[i+1][0]
				}
				if end-mk[1] > 1600 {
					end = mk[1] + 1600
				}
				// 去掉起始残留的标签闭合符与尾部被截断的半个标签
				seg := trimPartialTag(strings.TrimLeft(region[mk[1]:end], ">"))
				if t := tidyDictText(cleanText(seg)); t != "" {
					entry.Senses = append(entry.Senses, DictSense{Text: t})
				}
			}
		}
	}

	// 例句(尽力而为,最多 4 条)
	for _, em := range reDictExample.FindAllStringSubmatch(page, -1) {
		if t := cleanText(em[1]); len(t) >= 20 && len(entry.Examples) < 4 {
			entry.Examples = append(entry.Examples, t)
		}
	}
	return entry
}

// normalizePron 清洗音标:去掉"美/英国"前缀与残缺的右方括号。
// Bing 页面音标源本即残缺(缺左括号),统一输出不带括号的核心音标。
func normalizePron(s string) string {
	s = strings.TrimSpace(s)
	for _, prefix := range []string{"美式", "美", "英国", "英", "US", "UK"} {
		if strings.HasPrefix(s, prefix) {
			s = strings.TrimSpace(strings.TrimPrefix(s, prefix))
			break
		}
	}
	s = strings.TrimSpace(strings.TrimSuffix(s, "]"))
	return s
}

// stripBrackets 去掉首尾方括号
func stripBrackets(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	return strings.TrimSpace(s)
}

// trimPartialTag 去掉尾部被截断的半个 HTML 标签(切片窗口可能切在标签中间)
func trimPartialTag(s string) string {
	if lt, gt := strings.LastIndex(s, "<"), strings.LastIndex(s, ">"); lt > gt && lt >= 0 {
		return s[:lt]
	}
	return s
}

// reCJKPunct 全角标点(词典页每个标点都是独立节点,清标签后产生假空格)
var reCJKPunct = regexp.MustCompile(`\s*([、。，；：！？（）《》〈〉【】])\s*`)

// tidyDictText 收紧词典文本中全角标点周围的假空格
// (英文半角标点不受影响,保留正常空格)
func tidyDictText(s string) string {
	return reCJKPunct.ReplaceAllString(s, "$1")
}

// ── 通用抓取 ──────────────────────────────────────────────────

// acceptLanguageFor 生成 Accept-Language 头(市场优先,回退英文)
func acceptLanguageFor(market string) string {
	if market == "" {
		return "en-US,en;q=0.9"
	}
	return market + ",en;q=0.5"
}

// fetchBing 向指定 Bing 主机请求任意路径(垂直搜索共用),
// 最多读取 5MB,与 bing.go fetch 保持一致的浏览器伪装头。
func (e *BingEngine) fetchBing(ctx context.Context, baseURL, path string, params url.Values, acceptLang string) ([]byte, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	u.Path = strings.TrimSuffix(u.Path, "/") + path
	u.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", acceptLang)

	resp, err := e.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errBingStatus{path: path, code: resp.StatusCode}
	}
	return io.ReadAll(io.LimitReader(resp.Body, 5<<20))
}

// errBingStatus Bing 返回非 200 的结构化错误(fetch 共用)
type errBingStatus struct {
	path string
	code int
}

func (e errBingStatus) Error() string {
	return "Bing 返回状态码 " + strconv.Itoa(e.code) + " (" + e.path + ")"
}

// ── 时长/日期换算(v7 兼容层用)──────────────────────────────

// durationToISO8601 把 "32:52" / "1:02:03" 转成官方 v7 的 ISO 8601 时长(PT32M52S)。
func durationToISO8601(d string) string {
	parts := strings.Split(strings.TrimSpace(d), ":")
	if len(parts) < 2 {
		return ""
	}
	var nums []int
	for _, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return ""
		}
		nums = append(nums, n)
	}
	var b strings.Builder
	b.WriteString("PT")
	if len(nums) == 3 {
		b.WriteString(strconv.Itoa(nums[0]) + "H")
		nums = nums[1:]
	}
	b.WriteString(strconv.Itoa(nums[0]) + "M" + strconv.Itoa(nums[1]) + "S")
	return b.String()
}

// rssDateToISO 把 RFC 1123 日期转成 ISO 8601(官方 v7 datePublished 形态),
// 解析失败返回原串。
func rssDateToISO(s string) string {
	if s == "" {
		return s
	}
	for _, layout := range []string{time.RFC1123, time.RFC1123Z, "Mon, 2 Jan 2006 15:04:05 -0700"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC().Format("2006-01-02T15:04:05Z")
		}
	}
	return s
}
