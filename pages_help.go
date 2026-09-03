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
<p class="lead">SearXNG 风格 + Bing 官方 API v7 调用兼容的极简搜索服务:只搜 Bing,结果保持原始顺序,JSON 返回。__LANG_COUNT__ 个语言/市场可选。</p>
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
<tr><td><code>/search</code></td><td>GET/POST</td><td>SearXNG 风格搜索,参数见下表</td></tr>
<tr><td><code>/v7/search</code></td><td>GET/POST</td><td><b>Bing 官方 Search API v7 调用兼容</b>(见下方专节)</td></tr>
<tr><td><code>/languages</code></td><td>GET</td><td>全部支持的语言/市场列表(JSON)</td></tr>
<tr><td><code>/help</code></td><td>GET</td><td>本帮助页</td></tr>
<tr><td><code>/healthz</code></td><td>GET</td><td>健康检查</td></tr>
</table>
<h2 style="margin-top:18px">/search 参数</h2>
<table>
<tr><th>参数</th><th>必填</th><th>默认</th><th>说明</th></tr>
<tr><td><code>q</code></td><td>是</td><td>-</td><td>查询词</td></tr>
<tr><td><code>count</code></td><td>否</td><td>10</td><td>每页条数 1~50(Bing 实际返回以其为准)</td></tr>
<tr><td><code>page</code></td><td>否</td><td>1</td><td>页码,兼容 SearXNG 的 <code>pageno</code></td></tr>
<tr><td><code>language</code></td><td>否</td><td>自动</td><td>语言/市场:<code>zh-CN</code>、<code>zh</code>、<code>zh-Hans</code>、<code>all</code>;未传时用 Accept-Language 头</td></tr>
</table>
<pre><code>curl "http://localhost:__PORT__/search?q=golang&count=10&language=zh-CN"

curl -X POST http://localhost:__PORT__/search \
     -H "Content-Type: application/json" \
     -d '{"q":"openai","count":5,"page":2,"language":"en-US"}'</code></pre>
<p style="color:var(--dim);font-size:13.5px">错误:400 缺参/语言不支持,405 方法错误,502 Bing 抓取失败,响应均为 JSON。</p>
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
<tr><td><code>offset</code></td><td>否</td><td>0</td><td>0 基偏移(官方语义,与 page 不同)</td></tr>
<tr><td><code>mkt</code></td><td>否</td><td>自动</td><td>市场,如 <code>en-US</code>、<code>zh-CN</code>(即 <code>/search</code> 的 language)</td></tr>
<tr><td><code>safeSearch</code></td><td>否</td><td>Moderate</td><td><code>Off</code> / <code>Moderate</code> / <code>Strict</code>(Strict 映射 Bing adlt=strict)</td></tr>
<tr><td><code>responseFilter</code></td><td>否</td><td>-</td><td>答案类型过滤,支持 <code>Webpages</code>、<code>RelatedSearches</code></td></tr>
<tr><td><code>setLang</code></td><td>否</td><td>-</td><td>接受但忽略(官方语义仅影响界面字符串)</td></tr>
</table>
<p>等价别名(覆盖官方两代路径):<code>/v7.0/search</code> · <code>/bing/v7/search</code> · <code>/bing/v7.0/search</code>。</p>
<p>响应结构与官方对齐:<code>_type</code>、<code>queryContext</code>、<code>webPages.webSearchUrl / totalEstimatedMatches / value[](id, name, url, displayUrl, snippet, …)</code>、<code>relatedSearches</code>;错误为官方 <code>ErrorResponse</code> 格式(<code>errors[].code / subCode / message / parameter</code>)。</p>
<p><b>可选鉴权</b>:设环境变量 <code>BING_API_KEY</code> 后,请求须携带一致的 <code>Ocp-Apim-Subscription-Key</code> 头(或 <code>subscription-key</code> 参数),否则返回官方 401 格式;未设则开放访问。</p>
<pre><code>curl "http://localhost:__PORT__/v7/search?q=golang&count=25&offset=50&mkt=en-US"

curl -X POST http://localhost:__PORT__/v7/search \
     -H "Content-Type: application/json" \
     -H "Ocp-Apim-Subscription-Key: $BING_API_KEY" \
     -d '{"q":"openai","count":30,"offset":0,"mkt":"en-US","safeSearch":"Strict"}'</code></pre>
<p style="color:var(--dim);font-size:13.5px">兼容边界:仅实现 webPages/relatedSearches 两类答案;freshness 等参数接受但忽略;totalEstimatedMatches 取自 SERP 计数条,解析不到时以已知结果数兜底。</p>
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
<h2>安全设计</h2>
<ul>
<li><b>安装/卸载仅限本机 CLI</b>:必须在服务器终端执行 <code>sudo bing-search-api install</code>,sudo 密码即鉴权;Web 端不提供任何安装入口(HTTP 端口无鉴权,网页按钮会变成远程改配置的后门)</li>
<li><b>服务进程非特权运行</b>:systemd 动态用户(DynamicUser) + 文件系统只读 + 禁设备/内核写入,仅保留低端口绑定能力;老版本 systemd 自动退回兼容单元</li>
<li><b>搜索 API 本身公开</b>(与 SearXNG 部署形态一致):如需限制访问,建议前面加反代(Nginx BasicAuth / IP 白名单)</li>
</ul>
</div>

<div class="card">
<h2>与 SearXNG 的关系</h2>
<ul>
<li>只保留 <b>bing</b> 一个引擎:无聚合、无重排、无打分,结果顺序 = Bing 原始顺序(<code>position</code> 标注位次)</li>
<li>响应字段与 SearXNG 对齐:<code>query / results / answers / corrections / infoboxes / suggestions / unresponsive_engines</code></li>
<li><code>language</code> 参数语义相同,映射到 Bing 的 <code>mkt + setlang</code>;自动还原 Bing <code>/ck/a</code> 重定向</li>
<li>零第三方依赖,单二进制,标准库实现</li>
</ul>
</div>

<footer>Bing Search API v__VERSION__ · <a href="/">测试界面</a> · <a href="https://github.com/cshdotcom/bing-search-api">GitHub</a></footer>
</div>
</body>
</html>`
