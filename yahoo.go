package main

// yahoo.go — Yahoo web 搜索回退引擎(深翻页备胎)。
//
// 背景:Bing web SERP 对非浏览器会话忽略 first 参数(见 bing.go ErrPagingBlocked),
// 服务端深翻页(offset≥10)不可达。Yahoo 网页搜索同为 Bing 索引供给,
// 而 SERP 的 b 参数(1 基结果偏移)对非浏览器会话开放翻页——
// 2026-09 实测(数据中心出口):b=1/11/15/64 均返回互不重合的真实结果,
// 任意 b 值(非步进对齐)同样生效;中文查询返回中文结果。
// 因此当 Bing 深翻页被风控拦截时,改由 Yahoo 承接同一窗口(offset 语义一致)。
//
// 翻页策略:
//   - b = offset+1 直接请求(Yahoo 接受任意 1 基偏移,无需页边界对齐)
//   - 每页固定 ~7 条,续页 b += 7,累计去重直至凑满 count
//   - 单窗口最多抓 8 页(≈56 条,覆盖 count≤50)
//
// 限制(诚实声明,README / /help 同步):
//   - 不支持 mkt/setlang 市场过滤(中文查询仍可返回中文结果)
//   - safeSearch=Strict 在该路径不生效(Yahoo 侧无对应参数映射)
//   - SERP 无总结果计数条,total 恒为 0(由 v7 层以已知条数兜底)
//   - 大陆出口无法直连 search.yahoo.com:该环境自动退回纯 Bing 行为
//     (YAHOO_FALLBACK=0 或网络不可达时均不影响原有语义)
//   - 结果排名为 Yahoo 侧的 Bing 索引排序,与 bing.com SERP 不逐条对齐

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	// yahooDefaultBase Yahoo 搜索入口,可通过 YAHOO_BASE 环境变量覆盖
	yahooDefaultBase = "https://search.yahoo.com"
	// yahooPageSize Yahoo SERP 每页结果数(实测固定 7 条,翻页链接步进同为 7)
	yahooPageSize = 7
	// maxYahooPages 单窗口最大抓取页数(7×8=56 ≥ count 上限 50)
	maxYahooPages = 8
)

// YahooEngine 封装对 Yahoo 搜索的抓取与解析
type YahooEngine struct {
	Base   string
	Client *http.Client
}

// yahooFallbackEnabled Yahoo 回退开关:默认开启,设 YAHOO_FALLBACK=0 关闭。
// 关闭后 web 深翻页维持 v1.4.3 行为(429 或部分结果+X-Paging-Limited)。
func yahooFallbackEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("YAHOO_FALLBACK"))) {
	case "0", "false", "no", "off":
		return false
	}
	return true
}

// yahooOK 该服务实例是否具备 Yahoo 回退能力(引擎已构造且开关开启)
func (s *server) yahooOK() bool {
	return s.yahoo != nil && s.yahoo.Client != nil && yahooFallbackEnabled()
}

// searchProviderHeader 结果来源标记头:web 翻页窗口的实际服务上游。
// "bing" = 纯 Bing;"yahoo" = 深页窗口整体由 Yahoo 承接(Bing 翻页被风控);
// "bing+yahoo" = 窗口前段来自 Bing 第 1 页、后段由 Yahoo 补齐。
// SearXNG / v7 两套接口通用,前端据此提示结果来源。
const searchProviderHeader = "X-Search-Provider"

// webSearchPaged 统一 web 翻页策略(v1.4.4,handleSearch 与 v7 共用):
//
//	Bing 优先 → 深翻页被风控拦截时自动回退 Yahoo(Bing 索引镜像,
//	b 参数对非浏览器会话开放翻页,offset 语义一致)。
//
// 行为矩阵:
//   - Bing 正常服务窗口           → provider="bing"
//   - offset≥10 且 Bing 风控拦截  → Yahoo 整窗接管 → provider="yahoo";
//     Yahoo 也失败(如大陆出口不可达)→ 返回原 Bing 错误(维持 429 可诊断语义)
//   - 窗口与第 1 页相交、聚合补齐被拦(v1.4.3 limited)→
//     Yahoo 从缺口续接补齐 → provider="bing+yahoo",补满则不再 limited;
//     补不齐仍保留 limited 语义(部分结果)
func (s *server) webSearchPaged(ctx context.Context, pq PagedQuery) ([]Result, []string, int64, string, bool, error) {
	results, suggestions, total, limited, err := s.engine.SearchPaged(ctx, pq)
	if err != nil {
		// 两类可回退错误:深页风控(ErrPagingBlocked)与空/不可解析 SERP
		// (ErrEmptySERP,突发限流瞬断)——Yahoo 同为 Bing 索引供给,均整窗承接
		if s.yahooOK() && (errors.Is(err, ErrPagingBlocked) || errors.Is(err, ErrEmptySERP)) {
			// 深页窗口整体不可达:Yahoo 以相同 offset/count 承接
			yr, ys, yt, yerr := s.yahoo.SearchPaged(ctx, pq)
			if yerr == nil {
				return yr, ys, yt, "yahoo", false, nil
			}
			log.Printf("Yahoo 回退失败(维持 Bing 报错) q=%q offset=%d: %v", pq.Term, pq.Offset, yerr)
		}
		return nil, nil, 0, "", false, err
	}
	// Bing 成功但窗口未满(翻页受限 partial):Yahoo 从缺口续接补齐
	if limited && s.yahooOK() && len(results) < pq.Count {
		rest := pq.Count - len(results)
		yr, _, _, yerr := s.yahoo.SearchPaged(ctx, PagedQuery{
			Term:  pq.Term,
			Count: rest,
			// 从 Bing 已得部分末尾续接(Yahoo 排名与 Bing 不逐条对齐,
			// 此处只保证窗口补满 + 去重,排名混合如实标记 bing+yahoo)
			Offset: pq.Offset + len(results),
		})
		if yerr == nil {
			seen := map[string]bool{}
			for _, r := range results {
				seen[r.URL] = true
			}
			for _, r := range yr {
				if r.URL == "" || seen[r.URL] {
					continue
				}
				seen[r.URL] = true
				results = append(results, r)
			}
			for i := range results {
				results[i].Position = i + 1
			}
			if len(results) >= pq.Count {
				limited = false // 窗口已补满
			}
			log.Printf("Yahoo 补齐翻页受限窗口 q=%q offset=%d bing=%d yahoo=%d", pq.Term, pq.Offset, len(results)-len(yr), len(yr))
			return results, suggestions, total, "bing+yahoo", limited, nil
		}
		log.Printf("Yahoo 补齐失败(维持 Bing 部分结果) q=%q offset=%d: %v", pq.Term, pq.Offset, yerr)
	}
	return results, suggestions, total, "bing", limited, nil
}

// SearchPaged 从 offset 起返回 count 条结果(v7 offset 语义,与 BingEngine 一致)。
// total 恒为 0(Yahoo SERP 无计数条);offset 超出可用结果时返回空切片而非报错。
func (e *YahooEngine) SearchPaged(ctx context.Context, pq PagedQuery) ([]Result, []string, int64, error) {
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
		seen        = map[string]bool{}
	)

	// b 为 1 基偏移:offset=0 → b=1;直接请求,无需页边界对齐
	b := pq.Offset + 1
	for page := 0; page < maxYahooPages && len(results) < want; page++ {
		body, err := e.fetch(ctx, b, pq.Term)
		if err != nil {
			return nil, nil, 0, err
		}
		pageHTML := string(body)
		pageResults, pageRelated := parseYahoo(pageHTML)
		if page == 0 {
			suggestions = pageRelated
		}
		added := 0
		for _, res := range pageResults {
			if res.URL == "" || seen[res.URL] {
				continue // 跨页去重(Yahoo 偶尔跨页重复展示)
			}
			seen[res.URL] = true
			results = append(results, res)
			added++
		}
		// 整页为空(结果耗尽):如实返回已得部分(可能为空 = 无更多结果)
		if len(pageResults) == 0 {
			break
		}
		if added == 0 {
			break // 整页全是重复项,继续翻页无意义
		}
		b += yahooPageSize
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
	return results, suggestions, 0, nil
}

// fetch 请求 Yahoo SERP 并返回 HTML(最多 5MB)。
// b 为 1 基结果偏移;ei=UTF-8 显式声明查询编码(中文查询必需)。
// 网络类错误(非状态码错误)重试一次:Yahoo 对突发连续抓取
// 偶发直接断连(实测 count=50 聚合连抛 8 页后可见),
// 300ms 退避后重试可显著提高回退成功率;状态码错误不重试。
func (e *YahooEngine) fetch(ctx context.Context, b int, term string) ([]byte, error) {
	body, err := e.fetchOnce(ctx, b, term)
	if err == nil {
		return body, nil
	}
	var es errUpstreamStatus
	if errors.As(err, &es) {
		return nil, err // 上游明确返回非 200,重试无意义
	}
	select {
	case <-ctx.Done():
		return nil, err
	case <-time.After(300 * time.Millisecond):
	}
	return e.fetchOnce(ctx, b, term)
}

// fetchOnce 单次请求(重试由 fetch 外层承载)
func (e *YahooEngine) fetchOnce(ctx context.Context, b int, term string) ([]byte, error) {
	params := url.Values{}
	params.Set("p", term)
	params.Set("b", strconv.Itoa(b))
	params.Set("ei", "UTF-8")
	return fetchHTTP(ctx, e.Client, e.Base, "/search", params, "en-US,en;q=0.9", "Yahoo")
}

// fetchHTTP 向任意上游主机发起 GET(与 BingEngine.fetchBing 同一套
// 浏览器伪装头与限额)。upstream 名用于错误信息区分来源。
// Yahoo 与 Bing 的抓取逻辑一致,抽出共用以免两份漂移。
func fetchHTTP(ctx context.Context, client *http.Client, baseURL, path string, params url.Values, acceptLang, upstream string) ([]byte, error) {
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

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errUpstreamStatus{host: u.Host, path: path, code: resp.StatusCode}
	}
	return io.ReadAll(io.LimitReader(resp.Body, 5<<20))
}

// parseYahoo 解析 Yahoo SERP,返回结果与相关搜索。
// 结果块:<div class="dd … algo …">(独立 algo 词,首 token 为 dd;
// favicon 等辅助块 class 首 token 为 thmb,天然被排除)。
// 标题:h3(内嵌 span);真实链接:重定向参数 RU=<urlencoded>/RK;
// 摘要:compText 容器内首个 <p>;相关搜索:class 含 s-requery 的锚点。
func parseYahoo(page string) ([]Result, []string) {
	var results []Result
	starts := reYahooAlgoStart.FindAllStringIndex(page, -1)
	for i, loc := range starts {
		end := len(page)
		if i+1 < len(starts) {
			end = starts[i+1][0]
		}
		if end-loc[0] > 40000 {
			end = loc[0] + 40000
		}
		block := page[loc[0]:end]

		am := reYahooAnchor.FindStringSubmatch(block)
		if am == nil {
			continue
		}
		link := decodeYahooRedirect(am[1])
		if !strings.HasPrefix(link, "http://") && !strings.HasPrefix(link, "https://") {
			continue // 解析不出真实目标链接,跳过该块
		}

		title := ""
		if tm := reYahooH3.FindStringSubmatch(block); tm != nil {
			title = cleanText(tm[1])
		}
		snippet := ""
		if sm := reYahooSnippet.FindStringSubmatch(block); sm != nil {
			snippet = cleanText(sm[1])
		}
		if title == "" && link == "" {
			continue
		}
		results = append(results, Result{
			Title:    title,
			URL:      link,
			Content:  snippet,
			Engine:   "yahoo",
			Engines:  []string{"yahoo"},
			Score:    1.0,
			Position: len(results) + 1,
		})
	}

	var suggestions []string
	for _, rm := range reYahooRelated.FindAllStringSubmatch(page, -1) {
		if t := cleanText(rm[1]); t != "" {
			suggestions = append(suggestions, t)
		}
	}
	return results, suggestions
}

// decodeYahooRedirect 还原 Yahoo 的 r.search.yahoo.com 重定向:
// 路径段 /RU=<urlencoded 真实URL>/RK=…,解码即目标。
// 非重定向链接原样返回。
func decodeYahooRedirect(href string) string {
	m := reYahooRU.FindStringSubmatch(href)
	if m == nil {
		return href
	}
	if dec, err := url.QueryUnescape(m[1]); err == nil {
		return dec
	}
	return m[1] // 解码失败时尽量保留原始段
}

var (
	// 结果块起始:class 首 token 为 dd 且含独立 algo 词
	// (thmb algo-favicon 等辅助块不以 dd 起始,不会被误抓)
	reYahooAlgoStart = regexp.MustCompile(`<div[^>]*\bclass="dd[^"]*\balgo\b[^"]*"`)
	// 块内首个锚点(即标题锚,favicon 区无 <a>)
	reYahooAnchor = regexp.MustCompile(`(?is)<a\b[^>]+href="([^"]+)"`)
	// 标题:<h3 …>(.*?)</h3>(内嵌 span,由 cleanText 展平)
	reYahooH3 = regexp.MustCompile(`(?is)<h3\b[^>]*>(.*?)</h3>`)
	// 摘要:compText 容器内首个 <p>
	reYahooSnippet = regexp.MustCompile(`(?is)<div[^>]*class="[^"]*\bcompText\b[^"]*"[^>]*>.*?<p[^>]*>(.*?)</p>`)
	// 相关搜索:s-requery 锚点文本
	reYahooRelated = regexp.MustCompile(`(?is)<a\b[^>]*class="[^"]*\bs-requery\b[^"]*"[^>]*>(.*?)</a>`)
	// 重定向真实 URL 段:RU=<encoded>/RK
	reYahooRU = regexp.MustCompile(`/RU=([^/]+)/RK=`)
)
