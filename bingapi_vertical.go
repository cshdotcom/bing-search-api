package main

// bingapi_vertical.go — Bing 官方垂直 Search API v7 兼容端点:
//
//      GET|POST /v7/images/search   (官方 Image Search API 兼容;别名 /v7.0/、/bing/v7(.0)/)
//      GET|POST /v7/videos/search   (官方 Video Search API 兼容)
//      GET|POST /v7/news/search     (官方 News Search API 兼容)
//      GET|POST /v7/dict/search     (服务扩展:官方无此端点,词典查询,Bing 词典中英双向)
//
// 鉴权与 /v7/search 一致:设 BING_API_KEY 后需 Ocp-Apim-Subscription-Key
// 头(或 subscription-key 参数),否则 401。
//
// 与官方的兼容边界(README 同步注明):
//   - images:官方 responseFilter/aspect/color/imageType/size/freshness 等
//     参数接受但忽略;totalEstimatedMatches 以 offset+结果数兜底
//   - videos:SERP 单页约 50 条,offset 在单页范围内切片(官方深分页能力有限)
//   - news:RSS 固定一批(约 11~15 条),官方 News API 本无 count/offset,接受但忽略
//   - dict:非官方端点(官方从未提供词典 API),响应为本服务定义的 v7 风格结构

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// httpBody 读取请求体(限 1MB,供 JSON 参数解析)
func httpBody(r *http.Request) io.Reader {
	return io.LimitReader(r.Body, 1<<20)
}

// ── v7 参数解析扩展(按端点放宽 count 上限)──────────────────

// v7Bounds 单个端点的参数边界
type v7Bounds struct {
	MaxCount  int
	MaxOffset int
}

// v7ParamsFromRequestBounded 带边界的参数解析(官方各端点 count 上限不同:
// web 50 / images 150 / videos 100)。
func v7ParamsFromRequestBounded(r *http.Request, b v7Bounds) (v7Params, *v7ParamError) {
	p := v7Params{Count: 10, Offset: 0, SafeSearch: "Moderate"}

	// POST + JSON body(官方 SDK 大查询场景)
	body := map[string]any{}
	if r.Method == http.MethodPost && strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		var m map[string]any
		if err := json.NewDecoder(httpBody(r)).Decode(&m); err == nil {
			body = m
		}
	}

	get := func(name string) string {
		if v, ok := body[name]; ok {
			switch tv := v.(type) {
			case string:
				return tv
			case float64:
				return strconv.FormatFloat(tv, 'f', -1, 64)
			case bool:
				return strconv.FormatBool(tv)
			}
		}
		return r.FormValue(name)
	}

	// q:必填
	if p.Q = strings.TrimSpace(get("q")); p.Q == "" {
		return p, &v7ParamError{
			SubCode: "ParameterMissing", Param: "q",
			Message: "缺少必填参数 q(查询词)",
		}
	}

	// count:1~MaxCount
	if s := strings.TrimSpace(get("count")); s != "" {
		if n, err := strconv.Atoi(s); err != nil || n < 1 || n > b.MaxCount {
			return p, &v7ParamError{
				SubCode: "ParameterInvalid", Param: "count", Value: s,
				Message: "count 必须是 1~" + strconv.Itoa(b.MaxCount) + " 的整数",
			}
		} else {
			p.Count = n
		}
	}

	// offset:0~MaxOffset
	if s := strings.TrimSpace(get("offset")); s != "" {
		if n, err := strconv.Atoi(s); err != nil || n < 0 || n > b.MaxOffset {
			return p, &v7ParamError{
				SubCode: "ParameterInvalid", Param: "offset", Value: s,
				Message: "offset 必须是 0~" + strconv.Itoa(b.MaxOffset) + " 的整数",
			}
		} else {
			p.Offset = n
		}
	}

	p.Mkt = strings.TrimSpace(get("mkt"))
	p.SetLang = strings.TrimSpace(get("setLang"))

	if s := strings.TrimSpace(get("safeSearch")); s != "" {
		switch strings.ToLower(s) {
		case "off":
			p.SafeSearch = "Off"
		case "moderate":
			p.SafeSearch = "Moderate"
		case "strict":
			p.SafeSearch = "Strict"
		default:
			return p, &v7ParamError{
				SubCode: "ParameterInvalid", Param: "safeSearch", Value: s,
				Message: "safeSearch 必须是 Off / Moderate / Strict",
			}
		}
	}

	if s := strings.TrimSpace(get("responseFilter")); s != "" {
		for _, tok := range strings.Split(s, ",") {
			if tok = strings.TrimSpace(strings.ToLower(tok)); tok != "" {
				p.RespFilter = append(p.RespFilter, tok)
			}
		}
	}
	return p, nil
}

// ── 官方 Images 响应结构 ──────────────────────────────────────

type bingV7ImagesResponse struct {
	Type                  string             `json:"_type"` // "Images"
	QueryContext          bingV7QueryContext `json:"queryContext"`
	ReadLink              string             `json:"readLink"`
	CurrentOffset         int                `json:"currentOffset"`
	NextOffset            int                `json:"nextOffset"`
	TotalEstimatedMatches int64              `json:"totalEstimatedMatches"`
	Value                 []bingV7Image      `json:"value"`
}

type bingV7Image struct {
	WebSearchURL       string               `json:"webSearchUrl"`
	Name               string               `json:"name"`
	ThumbnailURL       string               `json:"thumbnailUrl"`
	ContentURL         string               `json:"contentUrl"`
	HostPageURL        string               `json:"hostPageUrl"`
	HostPageDisplayURL string               `json:"hostPageDisplayUrl"`
	EncodingFormat     string               `json:"encodingFormat,omitempty"`
	Thumbnail          bingV7ImageThumbnail `json:"thumbnail"`
	ImageInsightsToken string               `json:"imageInsightsToken,omitempty"`
}

type bingV7ImageThumbnail struct {
	Width      int    `json:"width,omitempty"`
	Height     int    `json:"height,omitempty"`
	ContentURL string `json:"contentUrl"`
}

// ── 官方 Videos 响应结构 ──────────────────────────────────────

type bingV7VideosResponse struct {
	Type                  string             `json:"_type"` // "Videos"
	QueryContext          bingV7QueryContext `json:"queryContext"`
	ReadLink              string             `json:"readLink"`
	CurrentOffset         int                `json:"currentOffset"`
	TotalEstimatedMatches int64              `json:"totalEstimatedMatches"`
	Value                 []bingV7Video      `json:"value"`
}

type bingV7Video struct {
	WebSearchURL       string               `json:"webSearchUrl"`
	Name               string               `json:"name"`
	ThumbnailURL       string               `json:"thumbnailUrl"`
	DatePublished      string               `json:"datePublished,omitempty"`
	Publisher          []bingV7Organization `json:"publisher,omitempty"`
	ContentURL         string               `json:"contentUrl,omitempty"`
	HostPageURL        string               `json:"hostPageUrl"`
	HostPageDisplayURL string               `json:"hostPageDisplayUrl,omitempty"`
	EncodingFormat     string               `json:"encodingFormat,omitempty"`
	Duration           string               `json:"duration,omitempty"` // ISO 8601,如 PT4M13S
	AllowHTTPSEmbed    bool                 `json:"allowHttpsEmbed,omitempty"`
	VideoID            string               `json:"videoId,omitempty"`
}

type bingV7Organization struct {
	Type string `json:"_type"` // "Organization"
	Name string `json:"name"`
}

// ── 官方 News 响应结构 ────────────────────────────────────────

type bingV7NewsResponse struct {
	Type         string             `json:"_type"` // "News"
	QueryContext bingV7QueryContext `json:"queryContext"`
	ReadLink     string             `json:"readLink"`
	Value        []bingV7News       `json:"value"`
}

type bingV7News struct {
	Type          string               `json:"_type"` // "NewsArticle"
	Name          string               `json:"name"`
	URL           string               `json:"url"`
	Description   string               `json:"description"`
	DatePublished string               `json:"datePublished"`
	Category      string               `json:"category,omitempty"`
	Headline      bool                 `json:"headline"`
	Provider      []bingV7Organization `json:"provider,omitempty"`
}

// ── 词典(服务扩展,非官方结构)──────────────────────────────

type bingV7DictResponse struct {
	Type          string             `json:"_type"` // "Dict"(服务扩展)
	QueryContext  bingV7QueryContext `json:"queryContext"`
	ReadLink      string             `json:"readLink"`
	WebSearchURL  string             `json:"webSearchUrl"`
	Word          string             `json:"word,omitempty"`
	Pronunciation *bingV7DictPron    `json:"pronunciation,omitempty"`
	Value         []bingV7DictSense  `json:"value"`
}

type bingV7DictPron struct {
	US     string `json:"us,omitempty"`
	UK     string `json:"uk,omitempty"`
	Pinyin string `json:"pinyin,omitempty"`
}

type bingV7DictSense struct {
	Pos      string   `json:"pos,omitempty"`
	Def      string   `json:"def"`
	Examples []string `json:"examples,omitempty"`
}

// ── 端点实现 ─────────────────────────────────────────────────

// v7VerticalDesc 单个垂直端点的元信息
type v7VerticalDesc struct {
	Name      string // images / videos / news / dict
	MaxCount  int
	MaxOffset int
	// 该端点对 count/offset 的语义说明(用于日志/文档,不影响解析)
	Note string
}

var (
	v7VertImages = v7VerticalDesc{Name: "images", MaxCount: 150, MaxOffset: 9000,
		Note: "offset 经 async first 参数翻页"}
	v7VertVideos = v7VerticalDesc{Name: "videos", MaxCount: 100, MaxOffset: 9000,
		Note: "SERP 单页约 50 条,offset 在单页内切片"}
	v7VertNews = v7VerticalDesc{Name: "news", MaxCount: 50, MaxOffset: 9000,
		Note: "RSS 固定批次,count/offset 接受但仅做切片"}
	v7VertDict = v7VerticalDesc{Name: "dict", MaxCount: 10, MaxOffset: 0,
		Note: "词条查询,无分页"}
)

// handleBingV7Images GET|POST /v7/images/search(及别名)
func (s *server) handleBingV7Images(w http.ResponseWriter, r *http.Request) {
	s.serveV7Vertical(w, r, v7VertImages)
}

// handleBingV7Videos GET|POST /v7/videos/search(及别名)
func (s *server) handleBingV7Videos(w http.ResponseWriter, r *http.Request) {
	s.serveV7Vertical(w, r, v7VertVideos)
}

// handleBingV7News GET|POST /v7/news/search(及别名)
func (s *server) handleBingV7News(w http.ResponseWriter, r *http.Request) {
	s.serveV7Vertical(w, r, v7VertNews)
}

// handleBingV7Dict GET|POST /v7/dict/search(及别名;服务扩展端点)
func (s *server) handleBingV7Dict(w http.ResponseWriter, r *http.Request) {
	s.serveV7Vertical(w, r, v7VertDict)
}

// serveV7Vertical 垂直端点统一流程:方法 → 鉴权 → 参数 → mkt → 抓取 → 组装
func (s *server) serveV7Vertical(w http.ResponseWriter, r *http.Request, v v7VerticalDesc) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeBingV7Error(w, http.StatusMethodNotAllowed, bingV7Error{
			Code:    "InvalidRequest",
			Message: "仅支持 GET / POST",
		})
		return
	}
	if !s.v7AuthOK(w, r) {
		return
	}
	p, perr := v7ParamsFromRequestBounded(r, v7Bounds{MaxCount: v.MaxCount, MaxOffset: v.MaxOffset})
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

	// mkt:非空时必须是可识别市场(与 /v7/search 一致)
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

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	switch v.Name {
	case "images":
		items, err := s.engine.SearchImages(ctx, p.Q, market, p.Offset+1, p.Count, p.SafeSearch == "Strict")
		if err != nil {
			s.v7VerticalFail(w, r, "images", p, err)
			return
		}
		// async 端点按自身页大小返回,这里按 count 精确切片(官方语义)
		writeJSON(w, http.StatusOK, buildBingV7ImagesResponse(p, market, sliceByOffset(items, 0, p.Count)))
	case "videos":
		items, err := s.engine.SearchVideos(ctx, p.Q, market, p.SafeSearch == "Strict")
		if err != nil {
			s.v7VerticalFail(w, r, "videos", p, err)
			return
		}
		writeJSON(w, http.StatusOK, buildBingV7VideosResponse(p, market, sliceByOffset(items, p.Offset, p.Count)))
	case "news":
		// mkt 会破坏 RSS 输出,语言仅通过 Accept-Language 表达
		items, err := s.engine.SearchNews(ctx, p.Q, market)
		if err != nil {
			s.v7VerticalFail(w, r, "news", p, err)
			return
		}
		writeJSON(w, http.StatusOK, buildBingV7NewsResponse(p, market, items))
	case "dict":
		entry, err := s.engine.SearchDict(ctx, p.Q)
		if err != nil {
			s.v7VerticalFail(w, r, "dict", p, err)
			return
		}
		writeJSON(w, http.StatusOK, buildBingV7DictResponse(p, entry))
	}
}

// v7VerticalFail 垂直抓取失败的统一 502 输出
func (s *server) v7VerticalFail(w http.ResponseWriter, r *http.Request, vert string, p v7Params, err error) {
	log.Printf("v7/%s 查询失败 q=%q mkt=%q: %v", vert, p.Q, p.Mkt, err)
	writeBingV7Error(w, http.StatusBadGateway, bingV7Error{
		Code:    "ServerError",
		SubCode: "UnexpectedError",
		Message: "上游 Bing 查询失败: " + err.Error(),
	})
}

// sliceByOffset 在单页结果上按 offset/count 切片(视频/新闻)
func sliceByOffset[T any](items []T, offset, count int) []T {
	if offset < 0 {
		offset = 0
	}
	if offset >= len(items) {
		return []T{}
	}
	end := offset + count
	if end > len(items) || end < offset {
		end = len(items)
	}
	return items[offset:end]
}

// ── 响应组装 ─────────────────────────────────────────────────

// buildBingV7ImagesResponse 组装官方 Images 结构
func buildBingV7ImagesResponse(p v7Params, market string, items []ImageItem) bingV7ImagesResponse {
	resp := bingV7ImagesResponse{
		Type:          "Images",
		QueryContext:  bingV7QueryContext{OriginalQuery: p.Q},
		ReadLink:      bingV7ReadLink("images/search", p, market),
		CurrentOffset: p.Offset,
		NextOffset:    p.Offset + len(items),
		Value:         make([]bingV7Image, 0, len(items)),
	}
	resp.TotalEstimatedMatches = int64(p.Offset + len(items)) // 无总数来源,兜底
	for _, it := range items {
		img := bingV7Image{
			WebSearchURL:       bingV7ImageWebSearchURL(it),
			Name:               it.Title,
			ThumbnailURL:       it.ThumbURL,
			ContentURL:         it.ImageURL,
			HostPageURL:        it.PageURL,
			HostPageDisplayURL: it.PageURL,
			EncodingFormat:     imageExtToFormat(it.ImageURL),
			Thumbnail: bingV7ImageThumbnail{
				Width:      it.Width,
				Height:     it.Height,
				ContentURL: it.ThumbURL,
			},
		}
		if it.MID != "" {
			img.ImageInsightsToken = "ccid_" + it.MID
		}
		resp.Value = append(resp.Value, img)
	}
	return resp
}

// buildBingV7VideosResponse 组装官方 Videos 结构
func buildBingV7VideosResponse(p v7Params, market string, items []VideoItem) bingV7VideosResponse {
	resp := bingV7VideosResponse{
		Type:          "Videos",
		QueryContext:  bingV7QueryContext{OriginalQuery: p.Q},
		ReadLink:      bingV7ReadLink("videos/search", p, market),
		CurrentOffset: p.Offset,
		Value:         make([]bingV7Video, 0, len(items)),
	}
	resp.TotalEstimatedMatches = int64(p.Offset + len(items))
	for _, it := range items {
		host := hostOf(it.PageURL)
		v := bingV7Video{
			WebSearchURL:       bingV7DetailURL("videos", it.MID, p.Q),
			Name:               it.Title,
			ThumbnailURL:       it.ThumbURL,
			Publisher:          []bingV7Organization{{Type: "Organization", Name: host}},
			ContentURL:         it.ContentURL,
			HostPageURL:        it.PageURL,
			HostPageDisplayURL: it.PageURL,
			Duration:           durationToISO8601(it.Duration),
			AllowHTTPSEmbed:    true,
			VideoID:            it.ThumbID,
		}
		resp.Value = append(resp.Value, v)
	}
	return resp
}

// buildBingV7NewsResponse 组装官方 News 结构
func buildBingV7NewsResponse(p v7Params, market string, items []NewsItem) bingV7NewsResponse {
	resp := bingV7NewsResponse{
		Type:         "News",
		QueryContext: bingV7QueryContext{OriginalQuery: p.Q},
		ReadLink:     bingV7ReadLink("news/search", p, market),
		Value:        make([]bingV7News, 0, len(items)),
	}
	for _, it := range items {
		n := bingV7News{
			Type:          "NewsArticle",
			Name:          it.Title,
			URL:           it.URL,
			Description:   it.Desc,
			DatePublished: rssDateToISO(it.Date),
			Headline:      false,
		}
		if it.Source != "" {
			n.Provider = []bingV7Organization{{Type: "Organization", Name: it.Source}}
		}
		resp.Value = append(resp.Value, n)
	}
	return resp
}

// buildBingV7DictResponse 组装词典(服务扩展)结构
func buildBingV7DictResponse(p v7Params, entry *DictEntry) bingV7DictResponse {
	resp := bingV7DictResponse{
		Type:         "Dict",
		QueryContext: bingV7QueryContext{OriginalQuery: p.Q},
		ReadLink:     "https://cn.bing.com/bilingualdictionary/search?q=" + url.QueryEscape(p.Q),
		WebSearchURL: "https://cn.bing.com/dict/search?q=" + url.QueryEscape(p.Q),
		Value:        make([]bingV7DictSense, 0, 8),
	}
	if entry == nil {
		return resp // 未收录:空 value
	}
	resp.Word = entry.Word
	if entry.PronUS != "" || entry.PronUK != "" || entry.Pinyin != "" {
		resp.Pronunciation = &bingV7DictPron{
			US:     entry.PronUS,
			UK:     entry.PronUK,
			Pinyin: entry.Pinyin,
		}
	}
	// 首要释义放第一条;例句挂在第一条上
	first := bingV7DictSense{Pos: entry.Def.Pos, Def: entry.Def.Text, Examples: entry.Examples}
	if first.Def != "" {
		resp.Value = append(resp.Value, first)
	}
	for i, s := range entry.Senses {
		sense := bingV7DictSense{Pos: s.Pos, Def: s.Text}
		if i == 0 && first.Def == "" {
			sense.Examples = entry.Examples
		}
		resp.Value = append(resp.Value, sense)
	}
	return resp
}

// ── 小工具 ───────────────────────────────────────────────────

// bingV7ReadLink 官方 readLink 形态(指向官方 API 主机,与退役前一致)
func bingV7ReadLink(path string, p v7Params, market string) string {
	u := bingV7IDPrefix + path + "?q=" + url.QueryEscape(p.Q)
	if p.Count != 10 {
		u += "&count=" + strconv.Itoa(p.Count)
	}
	if p.Offset != 0 {
		u += "&offset=" + strconv.Itoa(p.Offset)
	}
	if market != "" {
		u += "&mkt=" + url.QueryEscape(market)
	}
	return u
}

// bingV7ImageWebSearchURL 图片详情页(官方 view=detailv2 模式)
func bingV7ImageWebSearchURL(it ImageItem) string {
	id := it.MID
	if id == "" {
		return it.PageURL
	}
	return "https://www.bing.com/images/search?view=detailv2&mid=" + url.QueryEscape(id)
}

// bingV7DetailURL 视频/图片详情页(view=detail 模式)
func bingV7DetailURL(vert, mid, q string) string {
	if mid == "" {
		return "https://www.bing.com/" + vert + "/search?q=" + url.QueryEscape(q)
	}
	return "https://www.bing.com/" + vert + "/search?view=detail&mid=" + url.QueryEscape(mid)
}

// imageExtToFormat 从图片 URL 后缀推断官方 encodingFormat(jpeg/png/…)
func imageExtToFormat(u string) string {
	// 先去掉查询串与锚点,再取后缀(避免把 ?xxx 当后缀)
	if q := strings.IndexAny(u, "?#"); q >= 0 {
		u = u[:q]
	}
	i := strings.LastIndex(u, ".")
	if i < 0 {
		return ""
	}
	switch ext := strings.ToLower(u[i+1:]); ext {
	case "jpg", "jpeg":
		return "jpeg"
	case "png", "gif", "webp", "bmp", "svg", "tiff", "ico":
		return ext
	}
	return "" // 未知后缀(含主机名里的 .com 等)不输出
}

// hostOf 提取主机名(失败返回原串的截断)
func hostOf(u string) string {
	if i := strings.Index(u, "://"); i > 0 {
		rest := u[i+3:]
		if j := strings.IndexAny(rest, "/?#"); j > 0 {
			return rest[:j]
		}
		return rest
	}
	if len(u) > 40 {
		return u[:40]
	}
	return u
}
