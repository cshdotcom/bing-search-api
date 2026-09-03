package main

// Result 单条搜索结果。
// 字段命名与 SearXNG 的 JSON 输出保持一致,
// 额外增加 position 字段,标注该结果在 Bing 原始结果中的位次。
type Result struct {
	Title    string   `json:"title"`
	URL      string   `json:"url"`
	Content  string   `json:"content"`
	Engine   string   `json:"engine"`
	Engines  []string `json:"engines"`
	Score    float64  `json:"score"`
	Position int      `json:"position"`
}

// SearchResponse SearXNG 风格的搜索响应。
// 本服务只有 bing 一个引擎,answers / corrections / infoboxes 恒为空数组,
// 与 SearXNG 单引擎模式下的返回结构保持一致。
// language 为本服务扩展字段,标注实际生效的语言(市场代码或 all)。
type SearchResponse struct {
	Query               string   `json:"query"`
	Language            string   `json:"language"`
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
