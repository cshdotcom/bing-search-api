package main

// bing.go 负责抓取 Bing 搜索结果页并解析出结构化结果。
// 解析策略:按 <li class="b_algo"> 起始标记切块,块内提取
// 标题链接(h2>a)、摘要(b_caption>p)、并还原 Bing 的 /ck/a 重定向。
// 结果顺序与 Bing 页面完全一致,不做任何重排序。

import (
	"context"
	"encoding/base64"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

const (
	// bingDefaultBase Bing 国际版入口,可通过 BING_BASE 环境变量覆盖
	bingDefaultBase = "https://www.bing.com"
	// userAgent 模拟常见浏览器,否则 Bing 可能返回异常页面
	userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
)

// QueryParams 一次 Bing 查询的参数
type QueryParams struct {
	Term       string // 查询词
	Count      int    // 每页条数
	First      int    // Bing 的 first 偏移(第一条结果的全局序号,从 1 开始)
	Language   string // mkt 市场,如 zh-CN / en-US,可为空
	SafeStrict bool   // safeSearch=Strict → SERP 追加 adlt=strict
}

// BingEngine 封装对 Bing 的抓取与解析
type BingEngine struct {
	Base   string
	Client *http.Client
}

var (
	// 每条自然搜索结果的起始标记:<li class="b_algo" ...>
	reAlgoStart = regexp.MustCompile(`<li[^>]*class="[^"]*\bb_algo\b[^"]*"`)
	// 标题与真实链接:<h2 ...><a ... href="...">标题</a>
	reTitleLink = regexp.MustCompile(`(?is)<h2[^>]*>.*?<a[^>]+href="([^"]+)"[^>]*>(.*?)</a>`)
	// 摘要容器:<div class="b_caption"> ... <p ...>摘要</p>
	reCaptionP = regexp.MustCompile(`(?is)<div[^>]*class="[^"]*\bb_caption\b[^"]*"[^>]*>.*?<p[^>]*>(.*?)</p>`)
	// 兜底摘要:任意 b_lineclamp 段落
	reLineclampP = regexp.MustCompile(`(?is)<p[^>]*class="[^"]*\bb_lineclamp[^"]*"[^>]*>(.*?)</p>`)
	// 相关搜索块(页面底部 b_rs,尽力而为)
	reRelated    = regexp.MustCompile(`(?is)<li[^>]*class="[^"]*\bb_rs\b[^"]*"[^>]*>.*?</ul>`)
	reAnchorText = regexp.MustCompile(`(?is)<a[^>]*>(.*?)</a>`)
	// Bing 重定向参数 u=a1<base64url>,解码后即目标 URL
	reRedirectU = regexp.MustCompile(`[?&]u=a1([A-Za-z0-9_\-]+)`)
	// 清除内联标签
	reTags = regexp.MustCompile(`<[^>]+>`)
	// 行内强调标签(strong/em 等,Bing 用来标记命中词)直接删除,
	// 避免替换成空格后在中文标题里产生假空格
	reInlineTag = regexp.MustCompile(`(?i)</?(?:strong|em|b|i|u|span|cite|sup|sub|wbr)\b[^>]*>`)
	// 压缩空白
	reSpaces = regexp.MustCompile(`\s+`)
	// SERP 计数条("约 7,140,000 条结果" / "2.340.000 Ergebnisse")
	reSBCount = regexp.MustCompile(`(?is)<span[^>]*class="[^"]*\bsb_count\b[^"]*"[^>]*>(.*?)</span>`)
)

// Search 执行一次 Bing 搜索并返回 SearXNG 风格的响应。
func (e *BingEngine) Search(ctx context.Context, qp QueryParams) (*SearchResponse, error) {
	body, err := e.fetch(ctx, qp)
	if err != nil {
		return nil, err
	}
	results, suggestions := parseBing(string(body))

	if results == nil {
		results = []Result{}
	}
	if suggestions == nil {
		suggestions = []string{}
	}
	return &SearchResponse{
		Query:               qp.Term,
		NumberOfResults:     len(results),
		Results:             results,
		Answers:             []string{},
		Corrections:         []string{},
		Infoboxes:           []string{},
		Suggestions:         suggestions,
		UnresponsiveEngines: []string{},
	}, nil
}

// PagedQuery 官方 API v7 兼容层的分页聚合查询
// (offset/count 语义 → Bing SERP 的 first 多页抓取)
type PagedQuery struct {
	Term       string // 查询词
	Language   string // mkt 市场,可为空
	Offset     int    // 0 基偏移(v7 offset 语义)
	Count      int    // 期望返回条数,1~50
	SafeStrict bool   // safeSearch=Strict
}

// maxFetchPages 多页聚合的最大抓取次数(count≤50,每页约 10 条,6 页足够;
// 同时这也是防止 Bing 风控的硬上限)
const maxFetchPages = 6

// SearchPaged 从 offset 起聚合 count 条结果:跨 SERP 多页抓取、去重、
// 截断,同时返回相关搜索与 SERP 计数条上的估计总数(解析不到为 0)。
func (e *BingEngine) SearchPaged(ctx context.Context, pq PagedQuery) ([]Result, []string, int64, error) {
	want := pq.Count
	if want < 1 {
		want = 10
	}
	if want > 50 {
		want = 50
	}

	var (
		results     []Result
		suggestions []string
		total       int64
		seen        = map[string]bool{}
	)
	// v7 的 offset 是 0 基全局序号;Bing SERP 的 first 是 1 基
	first := pq.Offset + 1

	for page := 0; page < maxFetchPages && len(results) < want; page++ {
		body, err := e.fetch(ctx, QueryParams{
			Term:       pq.Term,
			Count:      want,
			First:      first,
			Language:   pq.Language,
			SafeStrict: pq.SafeStrict,
		})
		if err != nil {
			return nil, nil, 0, err
		}
		pageResults, pageSugg := parseBing(string(body))
		if total == 0 {
			total = parseTotalResults(string(body))
		}
		if len(suggestions) == 0 {
			suggestions = pageSugg
		}
		if len(pageResults) == 0 {
			break // SERP 无更多结果
		}
		added := 0
		for _, res := range pageResults {
			if res.URL == "" || seen[res.URL] {
				continue // 跨页去重(Bing 偶尔重复展示)
			}
			seen[res.URL] = true
			res.Position = len(results) + 1
			results = append(results, res)
			added++
			if len(results) >= want {
				break
			}
		}
		if added == 0 {
			break // 整页都是重复项,继续翻页无意义
		}
		// 下一页从本页实际返回的末尾继续
		first += len(pageResults)
	}
	if len(results) > want {
		results = results[:want]
	}
	return results, suggestions, total, nil
}

// fetch 请求 Bing 搜索页并返回 HTML(最多 5MB)
func (e *BingEngine) fetch(ctx context.Context, qp QueryParams) ([]byte, error) {
	u, err := url.Parse(e.Base)
	if err != nil {
		return nil, fmt.Errorf("BING_BASE 配置无效: %w", err)
	}
	u.Path = strings.TrimSuffix(u.Path, "/") + "/search"

	params := url.Values{}
	params.Set("q", qp.Term)
	params.Set("count", strconv.Itoa(qp.Count))
	params.Set("first", strconv.Itoa(qp.First))
	if qp.Language != "" {
		params.Set("mkt", qp.Language)
		params.Set("setlang", strings.ToLower(strings.SplitN(qp.Language, "-", 2)[0]))
	}
	if qp.SafeStrict {
		// Bing SERP 的严格安全搜索开关
		params.Set("adlt", "strict")
	}
	u.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	acceptLang := "en-US,en;q=0.9"
	if qp.Language != "" {
		acceptLang = qp.Language + ",en;q=0.5"
	}
	req.Header.Set("Accept-Language", acceptLang)

	resp, err := e.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 Bing 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Bing 返回状态码 %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 5<<20))
}

// parseBing 解析 SERP 页面,返回结果与相关搜索。
// 结果顺序与 Bing 页面一致。
func parseBing(page string) ([]Result, []string) {
	var results []Result

	starts := reAlgoStart.FindAllStringIndex(page, -1)
	for i, loc := range starts {
		end := len(page)
		if i+1 < len(starts) {
			end = starts[i+1][0]
		}
		// 最后一个块做长度截断,避免把分页/脚本等内容吞进来
		if end-loc[0] > 40000 {
			end = loc[0] + 40000
		}
		block := page[loc[0]:end]

		m := reTitleLink.FindStringSubmatch(block)
		if m == nil {
			continue // 不是标准网页结果(如纯卡片),跳过
		}
		href := html.UnescapeString(m[1])
		title := cleanText(m[2])
		link := decodeBingRedirect(href)

		snippet := ""
		if cm := reCaptionP.FindStringSubmatch(block); cm != nil {
			snippet = cleanText(cm[1])
		} else if lm := reLineclampP.FindStringSubmatch(block); lm != nil {
			snippet = cleanText(lm[1])
		}

		if title == "" && link == "" {
			continue
		}
		results = append(results, Result{
			Title:    title,
			URL:      link,
			Content:  snippet,
			Engine:   "bing",
			Engines:  []string{"bing"},
			Score:    1.0,
			Position: len(results) + 1,
		})
	}

	var suggestions []string
	if rm := reRelated.FindStringSubmatch(page); rm != nil {
		for _, am := range reAnchorText.FindAllStringSubmatch(rm[0], -1) {
			if t := cleanText(am[1]); t != "" {
				suggestions = append(suggestions, t)
			}
		}
	}
	return results, suggestions
}

// decodeBingRedirect 把 Bing 的 /ck/a?...&u=a1<base64> 重定向
// 还原成真实 URL。解码失败时原样返回。
func decodeBingRedirect(href string) string {
	m := reRedirectU.FindStringSubmatch(href)
	if m == nil {
		return href
	}
	raw := m[1]
	if pad := len(raw) % 4; pad != 0 {
		raw += strings.Repeat("=", 4-pad)
	}
	dec, err := base64.URLEncoding.DecodeString(raw)
	if err != nil {
		// 兼容不带填充的变体
		if dec, err = base64.RawURLEncoding.DecodeString(m[1]); err != nil {
			return href
		}
	}
	s := string(dec)
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return s
	}
	return href
}

// parseTotalResults 从 SERP 计数条提取总结果数(如 "约 1,000 条结果" → 1000)。
// 各地区千分位分隔符不同(逗号/点),先移除分隔符再取最长数字串;
// 无法解析时返回 0,由调用方兜底。
func parseTotalResults(page string) int64 {
	m := reSBCount.FindStringSubmatch(page)
	if m == nil {
		return 0
	}
	text := cleanText(m[1])
	text = strings.Map(func(r rune) rune {
		switch r {
		case ',', '.', '\u00a0', '\u2019', ' ': // 千分位分隔与不可见空白
			return -1
		}
		return r
	}, text)
	var best string
	for _, seg := range strings.FieldsFunc(text, func(r rune) bool {
		return r < '0' || r > '9'
	}) {
		if len(seg) > len(best) {
			best = seg
		}
	}
	if best == "" || len(best) > 15 { // 防御异常超长数字
		return 0
	}
	n, err := strconv.ParseInt(best, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// cleanText 去掉 HTML 标签、还原实体、压缩空白
func cleanText(s string) string {
	s = reInlineTag.ReplaceAllString(s, "")
	s = reTags.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "​", "") // 零宽空格
	s = reSpaces.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
