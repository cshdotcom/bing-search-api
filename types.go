package main

// Result 单条搜索结果。
// 字段命名与 SearXNG 的 JSON 输出保持一致,
// 额外增加 position 字段,标注该结果在 Bing 原始结果中的位次。
// 垂直搜索(图片/视频/新闻)时附带 SearXNG 垂直结果字段(均 omitempty,
// 网页搜索时不出现,保持既有响应形状不变)。
type Result struct {
	Title    string   `json:"title"`
	URL      string   `json:"url"`
	Content  string   `json:"content"`
	Engine   string   `json:"engine"`
	Engines  []string `json:"engines"`
	Score    float64  `json:"score"`
	Position int      `json:"position"`

	// ── 垂直搜索扩展字段(omitempty)──
	Template      string `json:"template,omitempty"`      // images.html / videos.html(SearXNG 垂直模板)
	ImgSrc        string `json:"img_src,omitempty"`       // 图片:原图直链
	ThumbnailSrc  string `json:"thumbnail_src,omitempty"` // 图片/视频:缩略图
	Length        string `json:"length,omitempty"`        // 视频:时长 mm:ss
	PublishedDate string `json:"publishedDate,omitempty"` // 新闻:发布时间(RFC 1123)
	Source        string `json:"source,omitempty"`        // 新闻:来源媒体
}

// SearchResponse SearXNG 风格的搜索响应。
// 本服务只有 bing 一个引擎,answers / corrections / infoboxes 恒为空数组,
// 与 SearXNG 单引擎模式下的返回结构保持一致。
// language 为本服务扩展字段,标注实际生效的语言(市场代码或 all);
// category 为扩展字段,标注垂直搜索类别(images/videos/news/dict,
// 网页搜索时为空、不输出)。
type SearchResponse struct {
	Query               string   `json:"query"`
	Language            string   `json:"language"`
	Category            string   `json:"category,omitempty"`
	NumberOfResults     int      `json:"number_of_results"`
	Results             []Result `json:"results"`
	Answers             []string `json:"answers"`
	Corrections         []string `json:"corrections"`
	Infoboxes           []string `json:"infoboxes"`
	Suggestions         []string `json:"suggestions"`
	UnresponsiveEngines []string `json:"unresponsive_engines"`
}

// ErrorResponse 统一错误响应
type ErrorResponse struct {
	Error string `json:"error"`
	Query string `json:"query,omitempty"`
}
