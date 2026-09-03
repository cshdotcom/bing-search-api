package main

// languages.go 全量语言/市场表与语言参数解析。
//
// Bing 通过 mkt(市场)与 setlang(界面语言)控制结果语言。
// 本表覆盖 Bing 支持的全部主要语言与市场(99 个市场/70 种语言),
// 用法与 SearXNG 的 language 参数保持一致:
//
//	language=zh-CN   语言-地区代码(完整市场)
//	language=zh      语言代码(自动选择该语言的默认市场)
//	language=zh-Hans 常见别名,自动归一化
//	language=all     不限语言
//	(未指定)         自动使用请求的 Accept-Language 头

import (
	"fmt"
	"strings"
)

// Lang 描述一个可用的语言/市场
type Lang struct {
	Code     string `json:"code"`              // 规范市场代码,如 zh-CN
	Language string `json:"language"`          // 语言子标签,如 zh
	Name     string `json:"name"`              // 英文名称
	Default  bool   `json:"default,omitempty"` // 是否为该语言的默认市场
}

// languages 全部支持的语言/市场(按语言字母序)
var languages = []Lang{
	{"af-ZA", "af", "Afrikaans (South Africa)", false},
	{"am-ET", "am", "Amharic (Ethiopia)", false},
	{"ar-AE", "ar", "Arabic (UAE)", false},
	{"ar-EG", "ar", "Arabic (Egypt)", false},
	{"ar-SA", "ar", "Arabic (Saudi Arabia)", true},
	{"az-AZ", "az", "Azerbaijani (Azerbaijan)", true},
	{"be-BY", "be", "Belarusian (Belarus)", true},
	{"bg-BG", "bg", "Bulgarian (Bulgaria)", true},
	{"bn-BD", "bn", "Bengali (Bangladesh)", false},
	{"bn-IN", "bn", "Bengali (India)", true},
	{"bs-BA", "bs", "Bosnian (Bosnia and Herzegovina)", true},
	{"ca-ES", "ca", "Catalan (Spain)", true},
	{"cs-CZ", "cs", "Czech (Czech Republic)", true},
	{"da-DK", "da", "Danish (Denmark)", true},
	{"de-AT", "de", "German (Austria)", false},
	{"de-CH", "de", "German (Switzerland)", false},
	{"de-DE", "de", "German (Germany)", true},
	{"el-GR", "el", "Greek (Greece)", true},
	{"en-AU", "en", "English (Australia)", false},
	{"en-CA", "en", "English (Canada)", false},
	{"en-GB", "en", "English (United Kingdom)", false},
	{"en-HK", "en", "English (Hong Kong SAR)", false},
	{"en-ID", "en", "English (Indonesia)", false},
	{"en-IE", "en", "English (Ireland)", false},
	{"en-IN", "en", "English (India)", false},
	{"en-MY", "en", "English (Malaysia)", false},
	{"en-NZ", "en", "English (New Zealand)", false},
	{"en-PH", "en", "English (Philippines)", false},
	{"en-SG", "en", "English (Singapore)", false},
	{"en-US", "en", "English (United States)", true},
	{"en-ZA", "en", "English (South Africa)", false},
	{"es-AR", "es", "Spanish (Argentina)", false},
	{"es-CL", "es", "Spanish (Chile)", false},
	{"es-ES", "es", "Spanish (Spain)", true},
	{"es-MX", "es", "Spanish (Mexico)", false},
	{"es-US", "es", "Spanish (United States)", false},
	{"et-EE", "et", "Estonian (Estonia)", true},
	{"eu-ES", "eu", "Basque (Spain)", false},
	{"fa-IR", "fa", "Persian (Iran)", true},
	{"fi-FI", "fi", "Finnish (Finland)", true},
	{"fr-BE", "fr", "French (Belgium)", false},
	{"fr-CA", "fr", "French (Canada)", false},
	{"fr-CH", "fr", "French (Switzerland)", false},
	{"fr-FR", "fr", "French (France)", true},
	{"gl-ES", "gl", "Galician (Spain)", false},
	{"gu-IN", "gu", "Gujarati (India)", true},
	{"he-IL", "he", "Hebrew (Israel)", true},
	{"hi-IN", "hi", "Hindi (India)", true},
	{"hr-HR", "hr", "Croatian (Croatia)", true},
	{"hu-HU", "hu", "Hungarian (Hungary)", true},
	{"hy-AM", "hy", "Armenian (Armenia)", true},
	{"id-ID", "id", "Indonesian (Indonesia)", true},
	{"is-IS", "is", "Icelandic (Iceland)", true},
	{"it-CH", "it", "Italian (Switzerland)", false},
	{"it-IT", "it", "Italian (Italy)", true},
	{"ja-JP", "ja", "Japanese (Japan)", true},
	{"ka-GE", "ka", "Georgian (Georgia)", true},
	{"kk-KZ", "kk", "Kazakh (Kazakhstan)", true},
	{"km-KH", "km", "Khmer (Cambodia)", true},
	{"kn-IN", "kn", "Kannada (India)", true},
	{"ko-KR", "ko", "Korean (South Korea)", true},
	{"ky-KG", "ky", "Kyrgyz (Kyrgyzstan)", true},
	{"lo-LA", "lo", "Lao (Laos)", true},
	{"lt-LT", "lt", "Lithuanian (Lithuania)", true},
	{"lv-LV", "lv", "Latvian (Latvia)", true},
	{"mk-MK", "mk", "Macedonian (North Macedonia)", true},
	{"ml-IN", "ml", "Malayalam (India)", true},
	{"mn-MN", "mn", "Mongolian (Mongolia)", true},
	{"mr-IN", "mr", "Marathi (India)", true},
	{"ms-MY", "ms", "Malay (Malaysia)", true},
	{"my-MM", "my", "Burmese (Myanmar)", true},
	{"nb-NO", "nb", "Norwegian Bokmål (Norway)", true},
	{"ne-NP", "ne", "Nepali (Nepal)", true},
	{"nl-BE", "nl", "Dutch (Belgium)", false},
	{"nl-NL", "nl", "Dutch (Netherlands)", true},
	{"pl-PL", "pl", "Polish (Poland)", true},
	{"pt-BR", "pt", "Portuguese (Brazil)", true},
	{"pt-PT", "pt", "Portuguese (Portugal)", false},
	{"ro-RO", "ro", "Romanian (Romania)", true},
	{"ru-RU", "ru", "Russian (Russia)", true},
	{"si-LK", "si", "Sinhala (Sri Lanka)", true},
	{"sk-SK", "sk", "Slovak (Slovakia)", true},
	{"sl-SI", "sl", "Slovenian (Slovenia)", true},
	{"sq-AL", "sq", "Albanian (Albania)", true},
	{"sr-Cyrl-RS", "sr", "Serbian (Cyrillic, Serbia)", false},
	{"sr-Latn-RS", "sr", "Serbian (Latin, Serbia)", true},
	{"sv-SE", "sv", "Swedish (Sweden)", true},
	{"ta-IN", "ta", "Tamil (India)", true},
	{"te-IN", "te", "Telugu (India)", true},
	{"th-TH", "th", "Thai (Thailand)", true},
	{"tl-PH", "tl", "Tagalog (Philippines)", true},
	{"tr-TR", "tr", "Turkish (Turkey)", true},
	{"uk-UA", "uk", "Ukrainian (Ukraine)", true},
	{"ur-PK", "ur", "Urdu (Pakistan)", true},
	{"uz-UZ", "uz", "Uzbek (Uzbekistan)", true},
	{"vi-VN", "vi", "Vietnamese (Vietnam)", true},
	{"zh-CN", "zh", "Chinese (Simplified, China)", true},
	{"zh-HK", "zh", "Chinese (Traditional, Hong Kong SAR)", false},
	{"zh-TW", "zh", "Chinese (Traditional, Taiwan)", false},
}

// aliases 常见写法/旧代码 -> 规范市场代码(键为小写)
var aliases = map[string]string{
	"zh-hans":    "zh-CN",
	"zh-hans-cn": "zh-CN",
	"zh-hant":    "zh-TW",
	"zh-hant-tw": "zh-TW",
	"zh-hant-hk": "zh-HK",
	"es-419":     "es-MX", // 拉美西语
	"es-xl":      "es-MX",
	"no":         "nb-NO",
	"nb":         "nb-NO",
	"fil":        "tl-PH",
	"fil-ph":     "tl-PH",
	"in":         "id-ID", // 印尼语旧代码
	"iw":         "he-IL", // 希伯来语旧代码
	"mo":         "ro-RO", // 摩尔多瓦语旧代码
}

var (
	// langByCode 小写市场代码 -> 规范市场代码
	langByCode = map[string]string{}
	// defaultByLang 语言代码 -> 该语言的默认市场
	defaultByLang = map[string]string{}
)

func init() {
	for _, l := range languages {
		langByCode[strings.ToLower(l.Code)] = l.Code
		if l.Default {
			defaultByLang[l.Language] = l.Code
		}
	}
}

// findLanguage 把任意用户输入解析为规范 Bing 市场代码。
// 支持:完整市场代码(zh-CN)、语言代码(zh)、别名(zh-Hans)、
// 语言+未知地区(de-XX → de-DE)。大小写与分隔符不敏感。
// 找不到时 ok 为 false。
func findLanguage(input string) (market, lang string, ok bool) {
	s := strings.ToLower(strings.TrimSpace(input))
	s = strings.ReplaceAll(s, "_", "-")
	if i := strings.IndexByte(s, '.'); i >= 0 {
		s = s[:i] // 去掉 zh-cn.UTF-8 之类的编码后缀
	}
	if a, found := aliases[s]; found {
		s = strings.ToLower(a)
	}
	if code, found := langByCode[s]; found {
		return code, langOf(code), true
	}
	if code, found := defaultByLang[s]; found {
		return code, s, true
	}
	if i := strings.IndexByte(s, '-'); i > 0 {
		base := s[:i]
		if code, found := defaultByLang[base]; found {
			return code, base, true
		}
	}
	return "", "", false
}

// langOf 从市场代码取语言子标签,如 sr-Latn-RS -> sr
func langOf(code string) string {
	if i := strings.IndexByte(code, '-'); i > 0 {
		return code[:i]
	}
	return code
}

// resolveLanguage 解析搜索请求的 language 参数,返回 Bing 市场代码。
// 返回空字符串表示"不限语言"(mkt/setlang 均不发送)。
//   - all/any → 不限语言
//   - 空或 auto → 依次尝试 Accept-Language 头,失败则交给 Bing 自动判断
//   - 其他 → 查表,未识别返回错误(与 SearXNG 一样按语言-地区代码传参)
func resolveLanguage(param, acceptLanguage string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(param)) {
	case "", "auto":
		return preferredFromAcceptLanguage(acceptLanguage), nil
	case "all", "any":
		return "", nil
	}
	market, _, ok := findLanguage(param)
	if !ok {
		return "", fmt.Errorf(
			"不支持的语言 %q:共支持 %d 个语言/市场(完整列表见 GET /languages),"+
				"示例:zh-CN、en-US、ja-JP、de、all",
			param, len(languages))
	}
	return market, nil
}

// preferredFromAcceptLanguage 解析 Accept-Language 头,
// 返回第一个可识别的市场代码;无法识别时返回空字符串。
func preferredFromAcceptLanguage(header string) string {
	if strings.TrimSpace(header) == "" {
		return ""
	}
	for _, part := range strings.Split(header, ",") {
		tag := strings.TrimSpace(strings.SplitN(part, ";", 2)[0])
		if tag == "" || tag == "*" {
			continue
		}
		if code, _, ok := findLanguage(tag); ok {
			return code
		}
	}
	return ""
}
