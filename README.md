# bing-search-api

用 Go 编写的极简 SearXNG 风格搜索 API:**只搜索 Bing**,拿到结果后**不做任何重新排序、去重或聚合**,按 Bing 结果页的原始顺序整理成 JSON 返回。

适合需要自建轻量搜索代理、给小工具/脚本/前端提供搜索能力的场景。

## 特性

- 单一引擎:Bing(网页结果)
- 原样返回:结果顺序与 Bing 完全一致,不重排、不去重、不打分
- SearXNG 风格的 JSON 响应结构,方便从 SearXNG 平滑迁移
- 支持 GET / POST(表单与 JSON)、分页、每页条数、语言/市场(mkt)
- 零第三方依赖,仅 Go 标准库
- 自动还原 Bing `/ck/a` 重定向为真实 URL
- CORS 开放,可直接被浏览器前端调用
- 单二进制部署,附带 Dockerfile

## 快速开始

### 直接运行

```bash
go run .                                    # 默认监听 :8080
PORT=9000 go run .                          # 自定义端口
BING_BASE=https://cn.bing.com go run .      # 使用国内版入口(可选)
```

### 编译

```bash
go build -o bing-search-api .
./bing-search-api
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
| language | 否 | 自动 | 语言/市场,如 `zh-CN`、`en-US`,映射到 Bing 的 `mkt` 参数 |
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

curl -X POST http://localhost:8080/search -d "q=golang&page=2"
```

### 响应

```json
{
  "query": "golang",
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
- `content` 为 Bing 给出的摘要文本
- `suggestions` 为 Bing 底部"相关搜索"(如有)

### 错误

| 状态码 | 场景 | 示例 |
| ------ | ---- | ---- |
| 400 | 缺少 q 参数 | `{"error":"缺少查询参数 q"}` |
| 405 | 方法不支持 | `{"error":"仅支持 GET / POST"}` |
| 502 | Bing 抓取失败/被限流 | `{"error":"Bing 查询失败: ..."}` |

### 其他端点

- `GET /healthz` 健康检查
- `GET /` 服务与接口说明

## 配置

| 环境变量 | 默认值 | 说明 |
| -------- | ------ | ---- |
| `PORT` | `8080` | 监听端口 |
| `BING_BASE` | `https://www.bing.com` | Bing 入口,可换成 `https://cn.bing.com` |

## 与 SearXNG 的关系

本项目借鉴了 SearXNG 的 API 风格,但刻意做得更小:

- 只保留 bing 一个引擎,没有聚合 / 重排 / 评分逻辑
- 响应字段(`query` / `results` / `answers` / ...)与 SearXNG 对齐,`format=json` 参数被接受但始终返回 JSON
- SearXNG 是完整的元搜索引擎,本项目只是一个「Bing → JSON」的透明网关

## 局限与声明

- 通过解析 Bing 结果页 HTML 实现,页面结构变化时需要更新 `bing.go` 中的正则
- 高频调用可能触发 Bing 风控(验证码 / 空结果),请合理控制频率
- 仅供学习与个人使用,请遵守 Bing 的服务条款;生产环境建议使用官方 Bing Web Search API

## 项目结构

```
├── main.go      # HTTP 服务:路由、参数解析、中间件
├── bing.go      # Bing 抓取与解析(结果、重定向解码、相关搜索)
├── types.go     # JSON 响应结构定义
├── go.mod
├── Dockerfile
├── LICENSE
└── README.md
```
