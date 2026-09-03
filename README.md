# bing-search-api

用 Go 编写的极简 SearXNG 风格搜索 API:**只搜索 Bing**,拿到结果后**不做任何重新排序、去重或聚合**,按 Bing 结果页的原始顺序整理成 JSON 返回。支持**全量语言筛选**,`language` 参数用法与 SearXNG 一致。

适合需要自建轻量搜索代理、给小工具/脚本/前端提供搜索能力的场景。

## 特性

- 单一引擎:Bing(网页结果)
- 原样返回:结果顺序与 Bing 完全一致,不重排、不去重、不打分
- SearXNG 风格的 JSON 响应结构,方便从 SearXNG 平滑迁移
- **全量语言支持:70 种语言 / 99 个市场**,通过 `language` 参数筛选,行为与 SearXNG 相同
- `GET /languages` 枚举全部可用语言/市场
- 未指定语言时自动使用请求的 `Accept-Language` 头
- 自动还原 Bing `/ck/a` 重定向为真实 URL
- 支持 GET / POST(表单与 JSON)、分页、每页条数
- 零第三方依赖,仅 Go 标准库,单二进制部署
- 提供全平台发行版(Linux / macOS / Windows × amd64 / arm64 / 386)与 Dockerfile
- 单元测试覆盖解析、语言解析、重定向解码

## 快速开始

### 从发行版安装(推荐)

从 [Releases](https://github.com/cshdotcom/bing-search-api/releases) 下载对应平台的压缩包:

| 平台 | 文件 |
| ---- | ---- |
| Linux amd64 / arm64 / 386 | `bing-search-api_<ver>_linux_<arch>.tar.gz` |
| macOS amd64 / arm64 | `bing-search-api_<ver>_darwin_<arch>.tar.gz` |
| Windows amd64 / 386 / arm64 | `bing-search-api_<ver>_windows_<arch>.zip` |

```bash
sha256sum -c SHA256SUMS                # 校验完整性
tar xzf bing-search-api_v1.0.0_linux_amd64.tar.gz
./bing-search-api_v1.0.0_linux_amd64/bing-search-api
./bing-search-api -version              # 查看版本号
```

### 从源码运行

```bash
go run .                                    # 默认监听 :8080
PORT=9000 go run .                          # 自定义端口
BING_BASE=https://cn.bing.com go run .      # 使用国内版入口(可选)
make build                                  # 或用 Makefile 编译
```

### Docker

```bash
docker build -t bing-search-api .
docker run -d -p 8080:8080 --name bing-search bing-search-api
```

## API

### GET /search

| 参数 | 必填 | 默认 | 说明 |
| ---- | ---- | ---- | ---- |
| q | 是 | - | 查询词 |
| count | 否 | 10 | 每页条数,1~50(传给 Bing 的提示值,Bing 实际返回条数以它为准) |
| page | 否 | 1 | 页码,从 1 开始(兼容 SearXNG 的 `pageno`) |
| language | 否 | 自动 | 语言/市场,如 `zh-CN`、`en`、`zh-Hans`、`all`,详见下方语言支持 |
| format | 否 | - | 仅为兼容 SearXNG 保留,传任何值都返回 JSON |

```bash
curl "http://localhost:8080/search?q=golang&count=5&page=1"
curl "http://localhost:8080/search?q=云计算&language=zh-CN&count=10"
```

### POST /search

支持表单或 JSON 两种方式:

```bash
curl -X POST http://localhost:8080/search \
     -H "Content-Type: application/json" \
     -d '{"q":"openai","count":5,"page":2,"language":"en-US"}'

curl -X POST http://localhost:8080/search -d "q=golang&page=2&language=de"
```

## 语言支持(与 SearXNG 相同)

`language` 参数的取值与行为和 SearXNG 保持一致,共支持 **70 种语言 / 99 个市场**:

| 写法 | 含义 | 示例 |
| ---- | ---- | ---- |
| `语言-地区` | 完整市场,映射到 Bing 的 `mkt` 参数 | `language=zh-CN`、`language=en-GB` |
| `语言` | 该语言,自动选择默认市场 | `language=zh` → zh-CN,`language=pt` → pt-BR |
| `别名` | 常见别名与旧代码自动归一化 | `zh-Hans` → zh-CN,`es-419` → es-MX,`in` → id-ID |
| `all` | 不限语言 | `language=all` |
| 不传 | 自动:按 `Accept-Language` 请求头,无头则由 Bing 判断 | - |

大小写与分隔符不敏感(`ZH_cn`、`zh_cn` 均可);语言可识别但地区未知时回落到该语言的默认市场(如 `de-XX` → de-DE);完全不认识的语言返回 400 并提示可用列表。

```bash
curl "http://localhost:8080/search?q=news&language=ja-JP"
curl "http://localhost:8080/search?q=nachrichten&language=de"
curl -H "Accept-Language: fr-FR" "http://localhost:8080/search?q=actualites"
curl "http://localhost:8080/search?q=news&language=all"
```

### GET /languages

枚举全部支持的语言/市场,供客户端动态渲染语言选择器:

```bash
curl http://localhost:8080/languages | python3 -m json.tool
```

```json
{
  "count": 99,
  "special": ["all", "auto"],
  "languages": [
    {"code": "zh-CN", "language": "zh", "name": "Chinese (Simplified, China)", "default": true},
    {"code": "zh-TW", "language": "zh", "name": "Chinese (Traditional, Taiwan)"},
    {"code": "ja-JP", "language": "ja", "name": "Japanese (Japan)", "default": true}
  ],
  "usage": "/search?q=example&language=zh-CN"
}
```

### 响应

```json
{
  "query": "golang",
  "language": "all",
  "number_of_results": 5,
  "results": [
    {
      "title": "The Go Programming Language",
      "url": "https://go.dev/",
      "content": "Get Started Playground Tour Stack Overflow Help ...",
      "engine": "bing",
      "engines": ["bing"],
      "score": 1.0,
      "position": 1
    }
  ],
  "answers": [],
  "corrections": [],
  "infoboxes": [],
  "suggestions": [],
  "unresponsive_engines": []
}
```

- `results` 顺序与 Bing 结果页完全一致,`position` 即原始位次
- `language` 为实际生效的语言(市场代码或 `all`)
- `content` 为 Bing 给出的摘要文本
- `suggestions` 为 Bing 底部"相关搜索"(如有)

### 错误

| 状态码 | 场景 | 示例 |
| ------ | ---- | ---- |
| 400 | 缺少 q 参数 | `{"error":"缺少查询参数 q"}` |
| 400 | 不支持的语言 | `{"error":"不支持的语言 \"xx\":共支持 99 个语言/市场(完整列表见 GET /languages)..."}` |
| 405 | 方法不支持 | `{"error":"仅支持 GET / POST"}` |
| 502 | Bing 抓取失败/被限流 | `{"error":"Bing 查询失败: ..."}` |

### 其他端点

- `GET /languages` 全部支持的语言/市场列表
- `GET /healthz` 健康检查
- `GET /` 服务与接口说明(含版本号与语言数量)

## 配置

| 环境变量 | 默认值 | 说明 |
| -------- | ------ | ---- |
| `PORT` | `8080` | 监听端口 |
| `BING_BASE` | `https://www.bing.com` | Bing 入口,可换成 `https://cn.bing.com` |

## 与 SearXNG 的关系

本项目借鉴了 SearXNG 的 API 风格,但刻意做得更小:

- 只保留 bing 一个引擎,没有聚合 / 重排 / 评分逻辑
- 响应字段(`query` / `results` / `answers` / ...)与 SearXNG 对齐,`format=json` 参数被接受但始终返回 JSON
- `language` 参数语义与 SearXNG 相同(语言-地区代码 / 语言代码 / `all`),映射到 Bing 的 `mkt` + `setlang`
- SearXNG 是完整的元搜索引擎,本项目只是一个「Bing → JSON」的透明网关

## 局限与声明

- 通过解析 Bing 结果页 HTML 实现,页面结构变化时需要更新 `bing.go` 中的正则
- `language` 映射到 Bing 的 `mkt`/`setlang`,是强提示但非强制:Bing 还会结合出口 IP 的地理定位与查询词本身判断市场,数据中心 IP 上个别查询可能被地理定位干扰(换部署位置或配 `BING_BASE` 可缓解)
- 高频调用可能触发 Bing 风控(验证码 / 空结果),请合理控制频率
- 仅供学习与个人使用,请遵守 Bing 的服务条款;生产环境建议使用官方 Bing Web Search API

## 开发

```bash
make test      # 单元测试(解析、语言解析、重定向解码)
make vet       # 静态检查
make fmt       # 格式化
make dist      # 交叉编译全平台发行包到 dist/
```

- 发布新版本:`make dist VERSION=vX.Y.Z` 后把 `dist/` 产物与 `SHA256SUMS` 上传到 Release

## 项目结构

```
├── main.go                 # HTTP 服务:路由、参数解析、中间件
├── bing.go                 # Bing 抓取与解析(结果、重定向解码、相关搜索)
├── languages.go            # 全量语言/市场表与语言参数解析(SearXNG 兼容)
├── types.go                # JSON 响应结构定义
├── bing_test.go            # 单元测试
├── build_release.sh        # 全平台交叉编译打包脚本
├── Makefile
├── go.mod
├── Dockerfile
├── LICENSE
└── README.md
```
