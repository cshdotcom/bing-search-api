# bing-search-api

用 Go 编写的极简 SearXNG 风格搜索 API:**只搜索 Bing**,拿到结果后**不做任何重新排序、去重或聚合**,按 Bing 结果页的原始顺序整理成 JSON 返回。支持**全量语言筛选**,`language` 参数用法与 SearXNG 一致。

适合需要自建轻量搜索代理、给小工具/脚本/前端提供搜索能力的场景。

![测试界面](docs/test-ui.png)

## 特性

- 单一引擎:Bing(网页结果)
- 原样返回:结果顺序与 Bing 完全一致,不重排、不去重、不打分
- SearXNG 风格的 JSON 响应结构,方便从 SearXNG 平滑迁移
- **全量语言支持:70 种语言 / 99 个市场**,通过 `language` 参数筛选,行为与 SearXNG 相同
- **Web 测试界面**:浏览器打开 `/` 即可搜索,语言/分页可选,实时查看 JSON 与 curl 命令
- **一键安装为 systemd 服务**:`sudo bing-search-api install` 自动注册服务 + 开机自启 + 崩溃自动重启
- **帮助文档**:`bing-search-api help`(终端)与 `/help`(网页)
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
tar xzf bing-search-api_v1.1.0_linux_amd64.tar.gz
cd bing-search-api_v1.1.0_linux_amd64
./bing-search-api -port 9000           # 直接运行,指定端口
sudo ./bing-search-api install -port 9000   # 或一键安装为 systemd 服务(开机自启)
```

### 安装为 systemd 服务(开机自启动)

```bash
sudo ./bing-search-api install            # 默认端口 8080
sudo ./bing-search-api install -port 9000 # 指定端口
sudo ./bing-search-api uninstall          # 卸载
```

`install` 会自动:

1. 复制二进制到 `/usr/local/bin/bing-search-api`
2. 写入 systemd 单元 `/etc/systemd/system/bing-search-api.service`
3. `daemon-reload` → `enable`(开机自启)→ `restart`(立即启动)

安装完成后:

```bash
systemctl status bing-search-api          # 服务状态
journalctl -u bing-search-api -f          # 跟踪日志
```

浏览器打开 `http://服务器IP:端口/` 即为测试界面。

> 无 systemd 的环境(容器/WSL1)会给出明确提示,改用直接运行或 Docker 即可。

### 从源码运行

```bash
go run .                                    # 默认监听 :8080
go run . -port 9000                         # 命令行指定端口
PORT=9000 go run .                          # 环境变量指定端口
BING_BASE=https://cn.bing.com go run .      # 使用国内版入口(可选)
make build                                  # 或用 Makefile 编译
```

### Docker

```bash
docker build -t bing-search-api .
docker run -d -p 8080:8080 --name bing-search bing-search-api
```

## CLI 用法

| 命令 | 说明 |
| ---- | ---- |
| `bing-search-api` | 前台启动 HTTP 服务(默认) |
| `bing-search-api -port 9000` | 指定端口启动(参数写在子命令前后均可) |
| `bing-search-api install` | 安装为 systemd 服务并设开机自启(非 root 自动 sudo 提权;`-no-start` 只注册不启动) |
| `bing-search-api uninstall` | 卸载 systemd 服务与二进制 |
| `bing-search-api help` | 终端打印帮助 |
| `bing-search-api version` | 打印版本号 |

通用参数:`-port N`(默认 8080,或环境变量 PORT)、`-host IP`(默认 0.0.0.0)、`-bing URL`(Bing 入口)。

## Web 界面

| 路径 | 说明 |
| ---- | ---- |
| `/` | 测试界面:搜索框 + 语言下拉(全部 99 个市场,自动跟随浏览器语言)+ 分页,实时展示结果、相关搜索、原始 JSON 与可复制的 curl 命令(curl 访问 `/` 仍返回 JSON 服务信息,两种视图互不干扰) |
| `/help` | 帮助文档页:快速开始、API 参数、语言规则、systemd 管理命令 |
| `/install` | 安装向导页:环境检测(root/systemd/已安装/运行中)+ 一键安装 + 终端命令指引 |

![帮助文档](docs/help-page.png)

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
- `GET /help` 帮助文档页(浏览器打开)
- `GET /install` 安装向导页(`?probe=1` 返回状态 JSON;`POST /install` 在进程权限内执行安装)
- `GET /healthz` 健康检查
- `GET /` 测试界面(浏览器)/ 服务信息 JSON(curl)

## 配置

| 环境变量 | 默认值 | 说明 |
| -------- | ------ | ---- |
| `PORT` | `8080` | 监听端口(命令行 `-port` 优先) |
| `HOST` | `0.0.0.0` | 监听地址(命令行 `-host` 优先) |
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
├── main.go                 # CLI 入口(子命令/双顺序 flag 解析)与 HTTP 服务:路由、参数、中间件
├── bing.go                 # Bing 抓取与解析(结果、重定向解码、相关搜索)
├── languages.go            # 全量语言/市场表与语言参数解析(SearXNG 兼容)
├── install.go              # systemd 安装/卸载(CLI 与 Web 双入口,自动 sudo 提权)
├── web.go                  # Web 路由:/ /help /install(HTML/JSON 内容协商)
├── pages.go                # 测试界面(内联 HTML/CSS/JS,无外部资源)
├── pages_help.go           # 帮助页与安装向导页(内联 HTML)
├── types.go                # JSON 响应结构定义
├── bing_test.go            # 单元测试
├── docs/                   # 界面截图
├── build_release.sh        # 全平台交叉编译打包脚本
├── Makefile
├── go.mod
├── Dockerfile
├── LICENSE
└── README.md
```
