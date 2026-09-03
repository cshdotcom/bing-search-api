package main

// bingapi.go — Bing 官方 Search API v7 调用兼容层。
//
// 背景:微软已于 2025-08-31 退役官方 Bing Search API(v7)。大量存量代码
// (Azure SDK、教程、脚本)仍按官方调用协议发请求。本兼容层把协议原样接住:
// 请求参数、响应结构、错误格式、鉴权头均与官方一致,客户端只需把 base URL
// 从 https://api.bing.microsoft.com 改成本服务地址即可继续工作。
//
// 端点(以下别名完全等价,覆盖官方两代路径):
//
//      GET|POST /v7/search          ← 官方新路径 api.bing.microsoft.com/v7/search
//      GET|POST /bing/v7.0/search   ← 官方旧路径 api.cognitive.microsoft.com/bing/v7.0/search
//      GET|POST /v7.0/search
//      GET|POST /bing/v7/search
//
// 鉴权(可选):设置环境变量 BING_API_KEY 后,请求必须携带与之一致的
// Ocp-Apim-Subscription-Key 请求头(官方方式)或 subscription-key 查询参数
// (Azure APIM 方式),否则 401;未设置该变量时完全不校验(开放服务)。
//
// 兼容边界(诚实声明,README 同步注明):
//   - 只实现 Web 答案(webPages + relatedSearches);responseFilter 指定其他
//     答案类型时按官方"过滤后为空"的语义返回不含该答案的响应
//   - freshness / textDecorations / textFormat / answerCount / promote 等
//     参数接受但忽略(官方客户端常发,直接报错反而破坏兼容)
//   - totalEstimatedMatches 取自 SERP 计数条;解析不到时以已知结果数兜底

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ── 官方 v7 响应结构(字段名与 JSON 键逐一对齐官方文档)──────────────────

// bingV7SearchResponse 官方 SearchResponse 顶层结构
type bingV7SearchResponse struct {
	Type            string               `json:"_type"`                     // "SearchResponse"
	QueryContext    bingV7QueryContext   `json:"queryContext"`              // 查询上下文
	WebPages        *bingV7WebPages      `json:"webPages,omitempty"`        // 网页答案
	RelatedSearches *bingV7RelatedSearch `json:"relatedSearches,omitempty"` // 相关搜索答案
}

// bingV7QueryContext 官方 queryContext
type bingV7QueryContext struct {
	OriginalQuery string `json:"originalQuery"`
	AdultIntent   bool   `json:"adultIntent"`
}

// bingV7WebPages 官方 webPages 答案
type bingV7WebPages struct {
	WebSearchURL          string          `json:"webSearchUrl"`
	TotalEstimatedMatches int64           `json:"totalEstimatedMatches"`
	Value                 []bingV7WebPage `json:"value"`
}

// bingV7WebPage 官方单条网页结果
type bingV7WebPage struct {
	ID               string `json:"id"`                        // 形如 …/api/v7/#WebPages.0
	Name             string `json:"name"`                      // 标题
	URL              string `json:"url"`                       // 真实链接
	DisplayURL       string `json:"displayUrl"`                // 展示链接
	Snippet          string `json:"snippet"`                   // 摘要
	DateLastCrawled  string `json:"dateLastCrawled,omitempty"` // SERP 无可靠日期,解析到才给
	Language         string `json:"language,omitempty"`
	IsFamilyFriendly bool   `json:"isFamilyFriendly"`
	IsNavigational   bool   `json:"isNavigational"`
}

// bingV7RelatedSearch 官方 relatedSearches 答案
type bingV7RelatedSearch struct {
	ID    string              `json:"id"`
	Value []bingV7RelatedItem `json:"value"`
}

// bingV7RelatedItem 官方单条相关搜索
type bingV7RelatedItem struct {
	Text         string `json:"text"`
	DisplayText  string `json:"displayText"`
	WebSearchURL string `json:"webSearchUrl"`
}

// bingV7ErrorResponse 官方 ErrorResponse 结构
type bingV7ErrorResponse struct {
	Type   string        `json:"_type"` // "ErrorResponse"
	Errors []bingV7Error `json:"errors"`
}

// bingV7Error 官方单条错误
type bingV7Error struct {
	Code      string `json:"code"`              // InvalidRequest / UnauthorizedAccess / ServerError
	SubCode   string `json:"subCode,omitempty"` // ParameterMissing / ParameterInvalid …
	Message   string `json:"message"`
	Parameter string `json:"parameter,omitempty"`
	Value     string `json:"value,omitempty"`
}

// ── 参数解析与校验 ──────────────────────────────────────────────

// v7Params v7 兼容端点解析后的参数(均已校验,除 mkt 需查表外)
type v7Params struct {
	Q          string
	Count      int      // 1~50,默认 10
	Offset     int      // 0~9000,默认 0
	Mkt        string   // 市场,如 en-US,可为空
	SetLang    string   // 接受但忽略
	SafeSearch string   // Off / Moderate / Strict
	RespFilter []string // responseFilter 小写 token
}

// v7ParamError 参数错误(携带官方错误响应需要的定位信息)
type v7ParamError struct {
	SubCode string // ParameterMissing / ParameterInvalid
	Param   string
	Value   string
	Message string
}

// v7ParamsFromRequest 合并三处来源:JSON body(POST)优先,其次表单,再次查询串。
// 返回已校验的参数;校验失败时返回 *v7ParamError(官方格式)。
// 实际解析由 v7ParamsFromRequestBounded 承载(web 端点边界:count 1~50、offset 0~9000)。
func v7ParamsFromRequest(r *http.Request) (v7Params, *v7ParamError) {
	return v7ParamsFromRequestBounded(r, v7Bounds{MaxCount: 50, MaxOffset: 9000})
}

// ── 处理器 ─────────────────────────────────────────────────────

// handleBingV7Search GET|POST /v7/search(及别名)入口
func (s *server) handleBingV7Search(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeBingV7Error(w, http.StatusMethodNotAllowed, bingV7Error{
			Code:    "InvalidRequest",
			Message: "仅支持 GET / POST",
		})
		return
	}

	// 鉴权:BING_API_KEY 设置后,校验官方订阅密钥头(或 APIM 查询参数)
	if !s.v7AuthOK(w, r) {
		return
	}

	p, perr := v7ParamsFromRequest(r)
	if perr != nil {
		writeBingV7Error(w, http.StatusBadRequest, bingV7Error{
			Code:      "InvalidRequest",
			SubCode:   perr.SubCode,
			Message:   perr.Message,
			Parameter: perr.Param,
			Value:     perr.Value,
		})
		return
	}

	// mkt:非空时必须是可识别的市场(官方对非法 mkt 返回 400)
	market := ""
	if p.Mkt != "" {
		m, _, ok := findLanguage(p.Mkt)
		if !ok {
			writeBingV7Error(w, http.StatusBadRequest, bingV7Error{
				Code:      "InvalidRequest",
				SubCode:   "ParameterInvalid",
				Message:   "不支持的 mkt 市场,完整列表见 GET /languages",
				Parameter: "mkt",
				Value:     p.Mkt,
			})
			return
		}
		market = m
	}

	// responseFilter 语义:显式过滤且不含 webpages → 不返回网页答案;
	// 不含 relatedsearches → 不返回相关搜索(与官方"过滤后为空"一致)
	wantWeb := len(p.RespFilter) == 0 || containsFold(p.RespFilter, "webpages")
	wantRelated := len(p.RespFilter) == 0 || containsFold(p.RespFilter, "relatedsearches")

	var (
		results     []Result
		suggestions []string
		total       int64
	)
	if wantWeb || wantRelated {
		ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
		defer cancel()
		var err error
		results, suggestions, total, err = s.engine.SearchPaged(ctx, PagedQuery{
			Term:       p.Q,
			Language:   market,
			Offset:     p.Offset,
			Count:      p.Count,
			SafeStrict: p.SafeSearch == "Strict",
		})
		if err != nil {
			log.Printf("v7 搜索失败 q=%q mkt=%q offset=%d: %v", p.Q, market, p.Offset, err)
			writeBingV7Error(w, http.StatusBadGateway, bingV7Error{
				Code:    "ServerError",
				SubCode: "UnexpectedError",
				Message: "上游 Bing 查询失败: " + err.Error(),
			})
			return
		}
	}

	resp := buildBingV7Response(p, market, results, suggestions, total, wantWeb, wantRelated)
	writeJSON(w, http.StatusOK, resp)
}

// buildBingV7Response 把聚合结果组装成官方 v7 响应结构
func buildBingV7Response(p v7Params, market string, results []Result, suggestions []string, total int64, wantWeb, wantRelated bool) bingV7SearchResponse {
	resp := bingV7SearchResponse{
		Type:         "SearchResponse",
		QueryContext: bingV7QueryContext{OriginalQuery: p.Q},
	}

	if wantWeb {
		wp := &bingV7WebPages{
			WebSearchURL: bingV7WebSearchURL(p.Q, market),
			Value:        make([]bingV7WebPage, 0, len(results)),
		}
		// 官方字段为估计值;SERP 计数条解析不到时,以"至少这么多"兜底
		wp.TotalEstimatedMatches = total
		if floor := int64(p.Offset) + int64(len(results)); wp.TotalEstimatedMatches < floor {
			wp.TotalEstimatedMatches = floor
		}
		for i, res := range results {
			wp.Value = append(wp.Value, bingV7WebPage{
				ID:               bingV7IDPrefix + "#WebPages." + strconv.Itoa(i),
				Name:             res.Title,
				URL:              res.URL,
				DisplayURL:       res.URL,
				Snippet:          res.Content,
				Language:         market,
				IsFamilyFriendly: true,
				IsNavigational:   false,
			})
		}
		resp.WebPages = wp
	}

	if wantRelated && len(suggestions) > 0 {
		rel := &bingV7RelatedSearch{
			ID:    bingV7IDPrefix + "#RelatedSearches",
			Value: make([]bingV7RelatedItem, 0, len(suggestions)),
		}
		for _, sg := range suggestions {
			rel.Value = append(rel.Value, bingV7RelatedItem{
				Text:         sg,
				DisplayText:  sg,
				WebSearchURL: bingV7WebSearchURL(sg, market),
			})
		}
		resp.RelatedSearches = rel
	}
	return resp
}

// bingV7IDPrefix 官方结果 id 的规范前缀(与退役前官方响应一致)
const bingV7IDPrefix = "https://api.bing.microsoft.com/api/v7/"

// bingV7WebSearchURL 生成官方 webSearchUrl(始终指向 bing.com,与官方响应一致)
func bingV7WebSearchURL(q, market string) string {
	u := "https://www.bing.com/search?q=" + url.QueryEscape(q)
	if market != "" {
		u += "&mkt=" + url.QueryEscape(market)
	}
	return u
}

// v7AuthOK v7 端点统一鉴权(设 BING_API_KEY 后校验):
// 校验 Ocp-Apim-Subscription-Key 请求头(官方方式)或 subscription-key
// 查询参数(Azure APIM 方式);失败时已写出 401 响应,返回 false。
// bingapi.go 与 bingapi_vertical.go 的全部 v7 端点共用。
func (s *server) v7AuthOK(w http.ResponseWriter, r *http.Request) bool {
	if want := envOr("BING_API_KEY", ""); want != "" {
		provided := r.Header.Get("Ocp-Apim-Subscription-Key")
		if provided == "" {
			provided = r.URL.Query().Get("subscription-key")
		}
		if provided != want {
			writeBingV7Error(w, http.StatusUnauthorized, bingV7Error{
				Code:    "UnauthorizedAccess",
				Message: "Access denied due to invalid subscription key. 请通过 Ocp-Apim-Subscription-Key 请求头携带正确密钥",
			})
			return false
		}
	}
	return true
}

// writeBingV7Error 按官方 ErrorResponse 结构输出错误
func writeBingV7Error(w http.ResponseWriter, status int, errs ...bingV7Error) {
	writeJSON(w, status, bingV7ErrorResponse{
		Type:   "ErrorResponse",
		Errors: errs,
	})
}

// containsFold 判断 token 列表中是否有某个小写值(忽略两边空白与大小写)
func containsFold(tokens []string, want string) bool {
	for _, t := range tokens {
		if strings.TrimSpace(strings.ToLower(t)) == want {
			return true
		}
	}
	return false
}
