package main

// bing.go 负责抓取 Bing 搜索结果页并解析出结构化结果。
// 解析策略:按 <li class="b_algo"> 起始标记切块,块内提取
// 标题链接(h2>a)、摘要(b_caption>p)、并还原 Bing 的 /ck/a 重定向。
// 结果顺序与 Bing 页面完全一致,不做任何重排序。

import (
        "context"
        "encoding/base64"
        "errors"
        "fmt"
        "html"
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
        First      int    // Bing 的 first 偏移(0 基、10 的倍数;<10 视为首页不发该参数)
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
        // ── 翻页导航(b_pag 区块)解析 ──
        // 当前页锚点:class 含 sb_pagS 且无 href(Bing 服务端标注的真实当前页)
        reCurPage = regexp.MustCompile(`(?is)<a[^>]*class="[^"]*\bsb_pagS\b[^"]*"[^>]*>(.*?)</a>`)
        reAriaPage = regexp.MustCompile(`aria-label="Page\s+(\d+)"`)
        // Bing 自身的下一页链接:FORM=PERE(主)或 sb_pagN 类(备)
        reNextLink = regexp.MustCompile(`(?is)<a[^>]*href="(/search\?[^"]*FORM=PERE[^"]*)"[^>]*>`)
        reNextLinkAlt = regexp.MustCompile(`(?is)<a[^>]*class="[^"]*\bsb_pagN\b[^"]*"[^>]*href="([^"?]*/search\?[^"]*)"[^>]*>`)
        // 翻页锚点可见文本里的页码(兜底)
        rePageNumText = regexp.MustCompile(`(?is)>\s*([0-9]{1,3})\s*<`)
)

// parseServedPageNum 从 b_pag 区块解析服务端标注的当前页码(1 起)。
// sb_pagS 锚点无 href,是 Bing 服务端渲染的“当前页”标记;
// 解析不到时返回 0(调用方视为无法校验,按尽力而为处理)。
func parseServedPageNum(page string) int {
        // 定位翻页区块,避免误读正文数字
        nav := page
        if i := strings.Index(page, "b_pag"); i >= 0 {
                if end := strings.Index(page[i:], "</nav>"); end > 0 {
                        nav = page[i : i+end]
                } else {
                        nav = page[i : min(i+6000, len(page))]
                }
        }
        m := reCurPage.FindStringSubmatch(nav)
        if m == nil {
                return 0
        }
        anchor := m[0]
        if strings.Contains(anchor, "href=") {
                return 0 // 有 href 的不是当前页标记(容错:正规 sb_pagS 无 href)
        }
        if am := reAriaPage.FindStringSubmatch(anchor); am != nil {
                n, _ := strconv.Atoi(am[1])
                return n
        }
        // 兜底:锚点内容即纯页码文本(如 <a class="sb_pagS">4</a>)
        if n, err := strconv.Atoi(strings.TrimSpace(m[1])); err == nil {
                return n
        }
        if tm := rePageNumText.FindStringSubmatch(anchor); tm != nil {
                n, _ := strconv.Atoi(tm[1])
                return n
        }
        return 0
}

// extractNextPageLink 提取 Bing 自己的“下一页”链接(相对路径+参数,
// 含 FPIG/FORM/first,由服务端生成,与浏览器点“下一页”完全同源)。
// 无翻页(末页/无导航)时返回空串。
func extractNextPageLink(page string) string {
        if m := reNextLink.FindStringSubmatch(page); m != nil {
                return m[1]
        }
        if m := reNextLinkAlt.FindStringSubmatch(page); m != nil {
                return m[1]
        }
        return ""
}

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

// bingPageSize Bing web SERP 每页结果数(first 参数的页步进)。
// Bing 自身翻页链接实测:第 2 页 first=10、第 3 页 first=20(0 基、10 的倍数)。
const bingPageSize = 10

// ErrPagingBlocked Bing 忽略翻页参数(疑似出口 IP 风控)。
// 检测方式:请求 first≥10 时,返回页的当前页标记(sb_pagS)仍是第 1 页。
// 此时不能静默把第 1 页当后续页返回,必须明确报错。
var ErrPagingBlocked = errors.New("Bing 忽略了翻页参数 first(疑似出口 IP 被风控):当前仅能返回第 1 页结果,建议更换出口 IP 或降低访问频率")

// SearchPaged 从 offset 起聚合 count 条结果。
//
// 翻页策略(与浏览器用户点“下一页”所见一致):
//   - 首次抓取按 0 基页边界对齐:first = offset/10*10(offset<10 时首页不带 first);
//     Bing web SERP 的 first 为 0 基、10 的倍数(Bing 自身链接实测)
//   - 后续页跟随响应里 b_pag 区块中 Bing 生成的翻页链接原样参数
//     (含 FPIG/FORM/first),并合并回 mkt/setlang/adlt,最大化“像浏览器”
//   - 每页抓取后校验服务端标注的当前页:请求的页码 ≥ 2 却返回第 1 页
//     → 判定翻页被风控拦截,返回 ErrPagingBlocked 明确报错
//   - 跨页去重、按累计结果序号精确切片,页大小波动(9~10 条)不影响 offset 语义
func (e *BingEngine) SearchPaged(ctx context.Context, pq PagedQuery) ([]Result, []string, int64, error) {
        want := pq.Count
        if want < 1 {
                want = 10
        }
        if want > 50 {
                want = 50
        }

        // 0 基页边界对齐:页内偏移 skip 由后续累计切片精确消化
        first := (pq.Offset / bingPageSize) * bingPageSize
        skip := pq.Offset - first
        need := skip + want // 需累计的结果条数

        var (
                results     []Result
                suggestions []string
                total       int64
                seen        = map[string]bool{}
                nextHref    string // Bing 自己的下一页链接(路径+参数)
        )

        for page := 0; page < maxFetchPages && len(results) < need; page++ {
                var body []byte
                var err error
                if page == 0 {
                        body, err = e.fetch(ctx, QueryParams{
                                Term:       pq.Term,
                                Count:      want,
                                First:      first, // fetch 内部对 <10 不发 first(首页)
                                Language:   pq.Language,
                                SafeStrict: pq.SafeStrict,
                        })
                } else {
                        if nextHref == "" {
                                break // SERP 无下一页链接(到末页)
                        }
                        body, err = e.fetchPageLink(ctx, nextHref, pq)
                }
                if err != nil {
                        return nil, nil, 0, err
                }

                pageHTML := string(body)
                if page == 0 {
                        // 翻页校验:请求页 ≥2 却被服务到第 1 页 → 风控拦截
                        if first >= bingPageSize {
                                if served := parseServedPageNum(pageHTML); served == 1 {
                                        return nil, nil, 0, fmt.Errorf("%w(请求 first=%d,服务端返回第 1 页)", ErrPagingBlocked, first)
                                }
                        }
                        total = parseTotalResults(pageHTML)
                        _, suggestions = parseBing(pageHTML)
                }

                pageResults, _ := parseBing(pageHTML)
                nextHref = extractNextPageLink(pageHTML)

                added := 0
                for _, res := range pageResults {
                        if res.URL == "" || seen[res.URL] {
                                continue // 跨页去重(Bing 偶尔重复展示)
                        }
                        seen[res.URL] = true
                        results = append(results, res)
                        added++
                }
                if len(pageResults) == 0 && page > 0 {
                        break // SERP 无更多结果
                }
                if added == 0 && page > 0 {
                        break // 整页都是重复项,继续翻页无意义
                }
        }

        // 按累计序号精确切片:v7 offset 语义 = 跳过前 offset 条结果
        if skip > 0 {
                if skip >= len(results) {
                        results = nil
                } else {
                        results = results[skip:]
                }
        }
        if len(results) > want {
                results = results[:want]
        }
        if results == nil {
                results = []Result{}
        }
        for i := range results {
                results[i].Position = i + 1
        }
        return results, suggestions, total, nil
}

// fetchPageLink 抓取 Bing 自身生成的翻页链接(相对路径)。
// 链接里已含 q/count/first/FPIG/FORM;此处合并回原始查询的
// mkt/setlang/adlt(翻页时语言与安全搜索不能丢)。
func (e *BingEngine) fetchPageLink(ctx context.Context, href string, pq PagedQuery) ([]byte, error) {
        u, err := url.Parse(strings.ReplaceAll(href, "&amp;", "&"))
        if err != nil || (u.Path == "" && u.RawQuery == "") {
                return nil, fmt.Errorf("翻页链接无效: %q", href)
        }
        params := u.Query()
        if pq.Language != "" {
                params.Set("mkt", pq.Language)
                params.Set("setlang", strings.ToLower(strings.SplitN(pq.Language, "-", 2)[0]))
        }
        if pq.SafeStrict {
                params.Set("adlt", "strict")
        }
        return e.fetchBing(ctx, e.Base, u.Path, params, acceptLanguageFor(pq.Language))
}

// fetch 请求 Bing 搜索页并返回 HTML(最多 5MB)。
// 具体请求逻辑由 vertical.go 的 fetchBing 统一承载。
// first 语义:0 基、10 的倍数(Bing 自身翻页链接实测);first<10 视为首页,
// 不发送 first 参数(与浏览器的首页 URL 形态一致)。
func (e *BingEngine) fetch(ctx context.Context, qp QueryParams) ([]byte, error) {
        params := url.Values{}
        params.Set("q", qp.Term)
        params.Set("count", strconv.Itoa(qp.Count))
        if qp.First >= bingPageSize {
                params.Set("first", strconv.Itoa(qp.First))
        }
        if qp.Language != "" {
                params.Set("mkt", qp.Language)
                params.Set("setlang", strings.ToLower(strings.SplitN(qp.Language, "-", 2)[0]))
        }
        if qp.SafeStrict {
                // Bing SERP 的严格安全搜索开关
                params.Set("adlt", "strict")
        }
        return e.fetchBing(ctx, e.Base, "/search", params, acceptLanguageFor(qp.Language))
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
