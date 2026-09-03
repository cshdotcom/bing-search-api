package main

// pages_help.go 内联页面:帮助文档(/help)。
// 安装/卸载只在本机 CLI 完成,帮助页仅展示命令,不提供任何 Web 安装入口。

const helpPageHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>帮助 · Bing Search API</title>
<style>
:root{--bg:#0f1216;--panel:#161b23;--panel2:#1c232e;--line:#2a3240;--fg:#e7edf3;
  --dim:#8b97a6;--accent:#3b8bff;--accent2:#67a2ff;--radius:12px;
  --mono:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace}
*{box-sizing:border-box}
body{margin:0;background:var(--bg);color:var(--fg);
  font:15px/1.75 -apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Hiragino Sans GB","Microsoft YaHei",sans-serif}
a{color:var(--accent2);text-decoration:none}a:hover{text-decoration:underline}
.wrap{max-width:920px;margin:0 auto;padding:0 20px 70px}
header{display:flex;align-items:center;gap:14px;padding:22px 0 16px;flex-wrap:wrap}
.logo{display:flex;align-items:center;gap:10px;font-weight:700;font-size:19px;color:var(--fg)}
.logo .mark{width:30px;height:30px;border-radius:8px;display:inline-flex;align-items:center;justify-content:center;
  background:linear-gradient(135deg,#0078d4,#41a5ee);color:#fff;font-size:15px;font-weight:800}
nav{margin-left:auto;display:flex;gap:6px}
nav a{color:var(--dim);padding:7px 12px;border-radius:8px;font-size:13.5px}
nav a:hover{color:var(--fg);background:var(--panel2);text-decoration:none}
.card{background:var(--panel);border:1px solid var(--line);border-radius:var(--radius);padding:22px 24px;margin-top:14px}
h1{font-size:22px;margin:2px 0 6px}h2{font-size:17px;margin:2px 0 12px;color:var(--fg)}
p.lead{color:var(--dim);margin:0 0 4px;font-size:14.5px}
table{border-collapse:collapse;width:100%;font-size:13.8px;margin:10px 0 4px}
th,td{border:1px solid var(--line);padding:8px 12px;text-align:left;vertical-align:top}
th{background:var(--panel2);color:var(--dim);font-weight:600;white-space:nowrap}
code{font:12.5px/1.6 var(--mono);background:var(--panel2);border:1px solid var(--line);
  border-radius:6px;padding:2px 7px;user-select:all}
pre{background:#0b0e13;border:1px solid var(--line);border-radius:10px;padding:14px 16px;
  font:12.5px/1.7 var(--mono);overflow:auto;margin:10px 0 4px;user-select:all}
pre code{background:none;border:0;padding:0}
kbd{font:12px var(--mono);background:var(--panel2);border:1px solid var(--line);border-bottom-width:2px;
  border-radius:5px;padding:2px 7px}
ul{padding-left:22px;margin:8px 0}li{margin:4px 0}
.ok{color:#3fb96b}
footer{margin-top:26px;text-align:center;font-size:12.5px;color:var(--dim)}
.tag{display:inline-block;font:11px var(--mono);border:1px solid var(--line);color:var(--dim);
  border-radius:999px;padding:3px 9px;margin-left:8px;vertical-align:3px}
</style>
</head>
<body>
<div class="wrap">
<header>
  <div class="logo"><span class="mark">b</span> Bing Search API <span class="tag">v__VERSION__</span></div>
  <nav><a href="/">测试界面</a><a href="/languages">语言列表</a></nav>
</header>

<div class="card">
<h1>帮助文档</h1>
<p class="lead">SearXNG 风格 + Bing 官方 API v7 调用兼容的极简搜索服务:只搜 Bing,支持<b>网页 / 图片 / 视频 / 新闻 / 词典</b>五类搜索,结果保持原始顺序,JSON 返回。__LANG_COUNT__ 个语言/市场可选。<b>SearXNG 兼容接口默认禁用</b>(设 ENABLE_SEARXNG=1 开启),默认仅开放 v7 兼容接口。</p>
</div>

<div class="card">
<h2>快速开始</h2>
<table>
<tr><th>方式</th><th>命令</th></tr>
<tr><td>直接运行</td><td><pre><code>./bing-search-api                 <span style="color:#8b97a6"># 默认端口 8080</span>
./bing-search-api -port 9000      <span style="color:#8b97a6"># 指定端口</span>
PORT=9000 ./bing-search-api       <span style="color:#8b97a6"># 环境变量方式</span></code></pre></td></tr>
<tr><td>安装为系统服务<br><span style="color:var(--dim);font-size:12.5px">systemd + 开机自启动<br>(仅限本机终端执行)</span></td><td><pre><code>sudo ./bing-search-api install -port 8080   <span style="color:#8b97a6"># 安装并启动</span>
sudo ./bing-search-api uninstall             <span style="color:#8b97a6"># 卸载</span></code></pre></td></tr>
<tr><td>查看帮助/版本</td><td><code>./bing-search-api help</code> · <code>./bing-search-api version</code></td></tr>
<tr><td>Docker</td><td><pre><code>docker build -t bing-search-api .
docker run -d -p 8080:8080 --name bing-search bing-search-api</code></pre></td></tr>
</table>
<p style="color:var(--dim);font-size:13.5px">安装完成后浏览器打开 <code>http://服务器IP:端口/</code> 即为测试界面。</p>
</div>

<div class="card">
<h2>CLI 子命令</h2>
<table>
<tr><th>命令</th><th>说明</th></tr>
<tr><td><code>(无参数)</code></td><td>前台启动 HTTP 服务</td></tr>
<tr><td><code>install</code></td><td>安装为 systemd 服务(复制二进制到 <code>/usr/local/bin</code>,注册服务、设开机自启、立即启动;非 root 自动请求 sudo;<b>只能在本机终端执行</b>)</td></tr>
<tr><td><code>uninstall</code></td><td>停止并卸载 systemd 服务与二进制</td></tr>
<tr><td><code>help</code></td><td>终端打印帮助(即本页内容的命令行版)</td></tr>
<tr><td><code>version</code></td><td>打印版本号</td></tr>
</table>
<p style="color:var(--dim);font-size:13.5px">通用参数:<code>-port N</code> 端口(默认 8080 或环境变量 PORT)、<code>-host IP</code> 监听地址(默认 0.0.0.0)、<code>-bing URL</code> Bing 入口。参数可写在子命令前或后。</p>
</div>

<div class="card">
<h2>API 端点</h2>
<table>
<tr><th>端点</th><th>方法</th><th>说明</th></tr>
<tr><td><code>/</code></td><td>GET</td><td>浏览器打开 → 测试界面;curl → 服务信息 JSON</td></tr>
<tr><td><code>/search</code></td><td>GET/POST</td><td>SearXNG 风格搜索(参数见下表;支持类别:网页/图片/视频/新闻/词典)<br><span style="color:#e8b44f;font-size:12.5px">⚠ 默认禁用:设 <code>ENABLE_SEARXNG=1</code> 后重启开启</span></td></tr>
<tr><td><code>/v7/search</code></td><td>GET/POST</td><td><b>Bing 官方 Web Search API v7 调用兼容</b>(见下方专节)</td></tr>
<tr><td><code>/v7/images/search</code><br><code>/v7/videos/search</code><br><code>/v7/news/search</code><br><code>/v7/dict/search</code></td><td>GET/POST</td><td><b>垂直搜索</b>:官方 Image/Video/News API 兼容 + 词典(服务扩展),详见下方专节;均支持 <code>/v7.0/</code>、<code>/bing/v7(.0)/</code> 别名前缀,同受 BING_API_KEY 鉴权</td></tr>
<tr><td><code>/languages</code></td><td>GET</td><td>全部支持的语言/市场列表(JSON)</td></tr>
<tr><td><code>/help</code></td><td>GET</td><td>本帮助页</td></tr>
<tr><td><code>/healthz</code></td><td>GET</td><td>健康检查</td></tr>
</table>
<h2 style="margin-top:18px">/search 参数<span class="tag">默认禁用</span></h2>
<p style="color:#e8b44f;font-size:13.5px;margin:2px 0 4px">⚠ 开放式搜索代理会被陌生流量滥用(白嫖出口 IP 抓取、触发 Bing 风控),故本接口<b>默认关闭</b>;仅在确需 SearXNG 兼容时,设环境变量 <code>ENABLE_SEARXNG=1</code> 并重启服务开启(配置方法见下方环境变量表)。</p>
<table>
<tr><th>参数</th><th>必填</th><th>默认</th><th>说明</th></tr>
<tr><td><code>q</code></td><td>是</td><td>-</td><td>查询词</td></tr>
<tr><td><code>category</code> / <code>categories</code></td><td>否</td><td>综合</td><td>搜索类别:<code>images</code> / <code>videos</code> / <code>news</code> / <code>dict</code>(不传 = 网页综合;SearXNG 协议用 <code>categories=images</code>,多值取首个)</td></tr>
<tr><td><code>count</code></td><td>否</td><td>10</td><td>每页条数 1~50(Bing 实际返回以其为准;图片/视频按 count 精确切片)</td></tr>
<tr><td><code>page</code></td><td>否</td><td>1</td><td>页码,兼容 SearXNG 的 <code>pageno</code>(新闻/词典不分页);换算为 0 基 offset 多页聚合,count&gt;10 时自动跨 SERP 页补齐不跳空</td></tr>
<tr><td><code>language</code></td><td>否</td><td>自动</td><td>语言/市场:<code>zh-CN</code>、<code>zh</code>、<code>zh-Hans</code>、<code>all</code>;未传时用 Accept-Language 头;词典固定中英双语</td></tr>
</table>
<pre><code>curl "http://localhost:__PORT__/search?q=golang&count=10&language=zh-CN"
curl "http://localhost:__PORT__/search?q=cat&categories=images&count=20"   <span style="color:#8b97a6"># 图片</span>
curl "http://localhost:__PORT__/search?q=hello&category=dict"             <span style="color:#8b97a6"># 词典</span>

curl -X POST http://localhost:__PORT__/search \
     -H "Content-Type: application/json" \
     -d '{"q":"openai","count":5,"page":2,"language":"en-US"}'</code></pre>
<p style="color:var(--dim);font-size:13.5px">错误:400 缺参/语言不支持/类别不支持,405 方法错误,502 Bing 抓取失败,响应均为 JSON。图片/视频结果附带 SearXNG 垂直字段(<code>template</code>、<code>img_src</code>、<code>thumbnail_src</code>、<code>length</code>、<code>publishedDate</code> 等);词典以 <code>answers</code> 返回词条摘要。</p>
</div>

<div class="card">
<h2>Bing 官方 API v7 兼容(/v7/search)</h2>
<p>微软已于 2025-08-31 退役官方 Bing Search API(v7)。本端点把官方调用协议原样接住——请求参数、响应 JSON 结构、错误格式、鉴权头均与官方一致,<b>存量代码只需把 base URL 换成本服务地址即可继续工作</b>:</p>
<pre><code># 官方退役前的调用
GET https://api.bing.microsoft.com/v7/search?q=golang&mkt=en-US \
    -H "Ocp-Apim-Subscription-Key: &lt;your-key&gt;"

# 改成自己的服务(零改造迁移)
GET http://localhost:__PORT__/v7/search?q=golang&mkt=en-US</code></pre>
<table>
<tr><th>参数</th><th>必填</th><th>默认</th><th>说明</th></tr>
<tr><td><code>q</code></td><td>是</td><td>-</td><td>查询词</td></tr>
<tr><td><code>count</code></td><td>否</td><td>10</td><td>返回条数 1~50,不足时自动多页聚合</td></tr>
<tr><td><code>offset</code></td><td>否</td><td>0</td><td>0 基偏移(官方语义,与 page 不同);对齐 Bing 页边界(0 基、10 的倍数)直取,不足时跟随 Bing 翻页链接补齐;风控拦截翻页时返回明确错误而非重复第 1 页</td></tr>
<tr><td><code>mkt</code></td><td>否</td><td>自动</td><td>市场,如 <code>en-US</code>、<code>zh-CN</code>(即 <code>/search</code> 的 language)</td></tr>
<tr><td><code>safeSearch</code></td><td>否</td><td>Moderate</td><td><code>Off</code> / <code>Moderate</code> / <code>Strict</code>(Strict 映射 Bing adlt=strict)</td></tr>
<tr><td><code>responseFilter</code></td><td>否</td><td>-</td><td>答案类型过滤,支持 <code>Webpages</code>、<code>RelatedSearches</code></td></tr>
<tr><td><code>setLang</code></td><td>否</td><td>-</td><td>接受但忽略(官方语义仅影响界面字符串)</td></tr>
</table>
<p>等价别名(覆盖官方两代路径):<code>/v7.0/search</code> · <code>/bing/v7/search</code> · <code>/bing/v7.0/search</code>。</p>
<p>响应结构与官方对齐:<code>_type</code>、<code>queryContext</code>、<code>webPages.webSearchUrl / totalEstimatedMatches / value[](id, name, url, displayUrl, snippet, …)</code>、<code>relatedSearches</code>;错误为官方 <code>ErrorResponse</code> 格式(<code>errors[].code / subCode / message / parameter</code>)。</p>
<p><b>可选鉴权</b>:设环境变量 <code>BING_API_KEY</code> 后,请求须携带一致的 <code>Ocp-Apim-Subscription-Key</code> 头(或 <code>subscription-key</code> 参数),否则返回官方 401 格式;未设则开放访问。密钥在服务启动时从环境变量读取,不写入配置文件,按部署方式配置:</p>
<pre><code># 直接运行
BING_API_KEY=你的密钥 ./bing-search-api

# systemd 服务:drop-in 追加,不改动主单元(升级不丢配置)
sudo systemctl edit bing-search-api     <span style="color:#8b97a6"># 在编辑器中加入下面两行</span>
#   [Service]
#   Environment=BING_API_KEY=你的密钥
sudo systemctl restart bing-search-api
journalctl -u bing-search-api -n 5       <span style="color:#8b97a6"># 出现"已启用密钥鉴权"即生效</span>

# Docker
docker run -d -p 8080:8080 -e BING_API_KEY=你的密钥 bing-search-api</code></pre>
<p style="color:var(--dim);font-size:13.5px">建议用 <code>openssl rand -hex 32</code> 生成密钥;密钥经 HTTP 头明文传输,公网部署请前置反代套 HTTPS;<code>/search</code>、<code>/languages</code> 等其他端点不受鉴权影响,仍开放访问。</p>
<pre><code>curl "http://localhost:__PORT__/v7/search?q=golang&count=25&offset=50&mkt=en-US"

curl -X POST http://localhost:__PORT__/v7/search \
     -H "Content-Type: application/json" \
     -H "Ocp-Apim-Subscription-Key: $BING_API_KEY" \
     -d '{"q":"openai","count":30,"offset":0,"mkt":"en-US","safeSearch":"Strict"}'</code></pre>
<p style="color:var(--dim);font-size:13.5px">兼容边界:仅实现 webPages/relatedSearches 两类答案;freshness 等参数接受但忽略;totalEstimatedMatches 取自 SERP 计数条,解析不到时以已知结果数兜底。</p>
</div>

<div class="card">
<h2>垂直搜索(图片 / 视频 / 新闻 / 词典)</h2>
<p>除网页综合搜索外,服务直接抓取 Bing 各垂直端点的<b>服务端可返回数据</b>,同时提供两套接口:SearXNG 风格(<code>/search?categories=…</code>)与官方 v7 风格(<code>/v7/{images,videos,news,dict}/search</code>)。两者参数一致(q/count/offset/mkt/safeSearch,POST JSON body 优先),鉴权策略一致(受 BING_API_KEY 保护,与 /v7/search 相同)。</p>
<table>
<tr><th>端点</th><th>响应结构</th><th>分页与边界</th></tr>
<tr><td><code>/v7/images/search</code></td><td>官方 Images 结构:<code>_type=Images</code>,<code>value[]{webSearchUrl,name,thumbnailUrl,contentUrl,hostPageUrl,encodingFormat,thumbnail,imageInsightsToken}</code>,<code>nextOffset / totalEstimatedMatches</code></td><td>count≤150;offset 经 async <code>first</code> 翻页;aspect/color 等参数接受但忽略</td></tr>
<tr><td><code>/v7/videos/search</code></td><td>官方 Videos 结构:<code>_type=Videos</code>,<code>value[]{name,thumbnailUrl,contentUrl,hostPageUrl,duration(ISO 8601),publisher[],videoId}</code></td><td>count≤100;SERP 单页约 50 条,offset 在单页内切片(深分页能力有限)</td></tr>
<tr><td><code>/v7/news/search</code></td><td>官方 News 结构:<code>_type=News</code>,<code>value[]{name,url,description,datePublished,provider[],headline}</code></td><td>RSS 固定一批(约 11~15 条);count/offset 接受但仅切片;语言由查询词与 Accept-Language 决定</td></tr>
<tr><td><code>/v7/dict/search</code><span class="tag">服务扩展</span></td><td>词典结构:<code>_type=Dict</code>,<code>word</code>,<code>pronunciation{us,uk,pinyin}</code>,<code>value[]{pos,def,examples[]}</code></td><td>中英双向(英文词→中文释义,中文词→英文释义);无分页;数据源 cn.bing.com 词典</td></tr>
</table>
<pre><code>curl "http://localhost:__PORT__/v7/images/search?q=cat&count=20&mkt=en-US"
curl "http://localhost:__PORT__/v7/videos/search?q=golang&count=10"
curl "http://localhost:__PORT__/v7/news/search?q=artificial+intelligence"
curl "http://localhost:__PORT__/v7/dict/search?q=hello"

# 设 BING_API_KEY 后:
curl -H "Ocp-Apim-Subscription-Key: $BING_API_KEY" \
     "http://localhost:__PORT__/v7/images/search?q=cat&count=20"</code></pre>
<p style="color:#e8b44f;font-size:13.5px">⚠ <b>不支持学术/购物/地图搜索</b>:Bing 学术(cn.bing.com/academic)、购物(/shop)、地图均为纯客户端(JS)渲染页面,服务端抓取拿不到结果数据;与其提供假接口,不如明确拒绝——传 <code>categories=academic|shopping|maps</code> 会返回 400 与未支持原因说明。SearXNG 的 science 类别在官方生态下由 arXiv/crossref 等引擎提供,与 Bing 无关,故不映射。</p>
<p style="color:var(--dim);font-size:13.5px">测试界面(/)已内置五类切换:选择「类别」后,两种接口模式分别发往对应端点;v7 模式提供 <b>API Key 输入框</b>(密钥仅存本浏览器 localStorage,以 Ocp-Apim-Subscription-Key 头发送,便于 BING_API_KEY 鉴权场景实测)。</p>
</div>

<div class="card">
<h2>语言支持(与 SearXNG 相同)</h2>
<p>共 <b>__LANG_COUNT__</b> 个市场,取值规则与 SearXNG 保持一致:</p>
<table>
<tr><th>写法</th><th>含义</th><th>示例</th></tr>
<tr><td><code>语言-地区</code></td><td>完整市场 → Bing <code>mkt</code></td><td><code>zh-CN</code>、<code>en-GB</code>、<code>ja-JP</code></td></tr>
<tr><td><code>语言</code></td><td>自动选默认市场</td><td><code>zh</code>→zh-CN,<code>pt</code>→pt-BR</td></tr>
<tr><td><code>别名</code></td><td>自动归一化</td><td><code>zh-Hans</code>→zh-CN,<code>es-419</code>→es-MX</td></tr>
<tr><td><code>all</code></td><td>不限语言</td><td><code>language=all</code></td></tr>
<tr><td>不传</td><td>按 Accept-Language 头</td><td><code>curl -H "Accept-Language: fr-FR" ...</code></td></tr>
</table>
<p style="color:var(--dim);font-size:13.5px">大小写与分隔符不敏感(<kbd>ZH_cn</kbd> 可用);未识别的语言返回 400 并提示。完整列表 <a href="/languages">GET /languages</a>。</p>
</div>

<div class="card">
<h2>systemd 服务管理(安装后)</h2>
<pre><code>systemctl status bing-search-api      <span style="color:#8b97a6"># 查看状态</span>
systemctl restart bing-search-api     <span style="color:#8b97a6"># 重启</span>
systemctl stop bing-search-api        <span style="color:#8b97a6"># 停止</span>
journalctl -u bing-search-api -f      <span style="color:#8b97a6"># 跟踪日志</span>
systemctl disable bing-search-api     <span style="color:#8b97a6"># 取消开机自启</span></code></pre>
<p style="color:var(--dim);font-size:13.5px">服务单元位于 <code>/etc/systemd/system/bing-search-api.service</code>,二进制位于 <code>/usr/local/bin/bing-search-api</code>,崩溃自动重启(3 秒)。</p>
</div>

<div class="card">
<h2>环境变量</h2>
<table>
<tr><th>变量</th><th>默认</th><th>说明</th></tr>
<tr><td><code>PORT</code></td><td>8080</td><td>监听端口(命令行 <code>-port</code> 优先)</td></tr>
<tr><td><code>HOST</code></td><td>0.0.0.0</td><td>监听地址(命令行 <code>-host</code> 优先)</td></tr>
<tr><td><code>BING_BASE</code></td><td>www.bing.com</td><td>Bing 入口,可换 <code>cn.bing.com</code></td></tr>
<tr><td><code>ENABLE_SEARXNG</code></td><td>关</td><td><b>SearXNG 兼容接口 /search 开关</b>:设 <code>1</code>(或 true/yes/on)开启;不设则该接口返回 403</td></tr>
<tr><td><code>BING_API_KEY</code></td><td>空</td><td>设置后 /v7/*(含垂直端点)需 <code>Ocp-Apim-Subscription-Key</code> 头(或 subscription-key 参数)匹配,详见上节</td></tr>
<tr><td><code>BING_DICT_BASE</code></td><td>cn.bing.com</td><td>词典数据源入口(www 域不提供词典服务,固定走 cn;自建反代时可覆盖)</td></tr>
</table>
<pre><code># 直接运行时开启 SearXNG 接口
ENABLE_SEARXNG=1 ./bing-search-api

# systemd 服务:drop-in 方式(升级不丢配置)
sudo systemctl edit bing-search-api     <span style="color:#8b97a6"># 在编辑器中加入下面两行</span>
#   [Service]
#   Environment=ENABLE_SEARXNG=1
sudo systemctl restart bing-search-api</code></pre>
</div>

<div class="card">
<h2>安全设计</h2>
<ul>
<li><b>安装/卸载仅限本机 CLI</b>:必须在服务器终端执行 <code>sudo bing-search-api install</code>,sudo 密码即鉴权;Web 端不提供任何安装入口(HTTP 端口无鉴权,网页按钮会变成远程改配置的后门)</li>
<li><b>SearXNG 兼容接口默认禁用</b>:无鉴权的开放搜索代理会被陌生流量滥用(匿名白嫖你的出口 IP 抓 Bing、触发风控连坐),故 <code>/search</code> 默认返回 403,须设 <code>ENABLE_SEARXNG=1</code> 显式开启;v7 兼容接口建议配 <code>BING_API_KEY</code> 使用</li>
<li><b>服务进程非特权运行</b>:systemd 动态用户(DynamicUser) + 文件系统只读 + 禁设备/内核写入,仅保留低端口绑定能力;老版本 systemd 自动退回兼容单元</li>
<li><b>默认最小暴露面</b>:开箱即用时仅 /v7 全部端点(建议配密钥)+ /languages + /help + /healthz 可访问</li>
</ul>
</div>

<div class="card">
<h2>与 SearXNG 的关系</h2>
<ul>
<li>只保留 <b>bing</b> 一个引擎:无聚合、无重排、无打分,结果顺序 = Bing 原始顺序(<code>position</code> 标注位次)</li>
<li>响应字段与 SearXNG 对齐:<code>query / results / answers / corrections / infoboxes / suggestions / unresponsive_engines</code>;垂直结果附带 <code>template / img_src / thumbnail_src / length / publishedDate</code> 等 SearXNG 垂直字段</li>
<li><code>language</code> 参数语义相同,映射到 Bing 的 <code>mkt + setlang</code>;自动还原 Bing <code>/ck/a</code> 重定向</li>
<li><code>categories</code> 参数与 SearXNG 一致(取第一个类别),词典为本服务扩展类别</li>
<li>零第三方依赖,单二进制,标准库实现</li>
</ul>
</div>

<footer>Bing Search API v__VERSION__ · <a href="/">测试界面</a> · <a href="https://github.com/cshdotcom/bing-search-api">GitHub</a></footer>
</div>
</body>
</html>`
