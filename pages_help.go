package main

// pages_help.go 内联页面:帮助文档(/help)与安装向导(/install)。

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
.ok{color:#3fb96b}.no{color:#ef6a6a}
footer{margin-top:26px;text-align:center;font-size:12.5px;color:var(--dim)}
.tag{display:inline-block;font:11px var(--mono);border:1px solid var(--line);color:var(--dim);
  border-radius:999px;padding:3px 9px;margin-left:8px;vertical-align:3px}
</style>
</head>
<body>
<div class="wrap">
<header>
  <div class="logo"><span class="mark">b</span> Bing Search API <span class="tag">v__VERSION__</span></div>
  <nav><a href="/">测试界面</a><a href="/languages">语言列表</a><a href="/install">安装服务</a></nav>
</header>

<div class="card">
<h1>帮助文档</h1>
<p class="lead">SearXNG 风格的极简搜索服务:只搜 Bing,结果保持原始顺序,JSON 返回。__LANG_COUNT__ 个语言/市场可选,行为与 SearXNG 一致。</p>
</div>

<div class="card">
<h2>快速开始</h2>
<table>
<tr><th>方式</th><th>命令</th></tr>
<tr><td>直接运行</td><td><pre><code>./bing-search-api                 <span style="color:#8b97a6"># 默认端口 8080</span>
./bing-search-api -port 9000      <span style="color:#8b97a6"># 指定端口</span>
PORT=9000 ./bing-search-api       <span style="color:#8b97a6"># 环境变量方式</span></code></pre></td></tr>
<tr><td>安装为系统服务<br><span style="color:var(--dim);font-size:12.5px">systemd + 开机自启动</span></td><td><pre><code>sudo ./bing-search-api install -port 8080   <span style="color:#8b97a6"># 安装并启动</span>
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
<tr><td><code>install</code></td><td>安装为 systemd 服务(自动复制二进制到 <code>/usr/local/bin</code>,注册服务、设开机自启、立即启动;非 root 自动请求 sudo)</td></tr>
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
<tr><td><code>/search</code></td><td>GET/POST</td><td>搜索,参数见下表</td></tr>
<tr><td><code>/languages</code></td><td>GET</td><td>全部支持的语言/市场列表(JSON)</td></tr>
<tr><td><code>/help</code></td><td>GET</td><td>本帮助页</td></tr>
<tr><td><code>/install</code></td><td>GET/POST</td><td>安装向导页 / 在权限内执行安装</td></tr>
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

const installPageHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>安装服务 · Bing Search API</title>
<style>
:root{--bg:#0f1216;--panel:#161b23;--panel2:#1c232e;--line:#2a3240;--fg:#e7edf3;
  --dim:#8b97a6;--accent:#3b8bff;--accent2:#67a2ff;--ok:#3fb96b;--err:#ef6a6a;--warn:#e0a13a;
  --radius:12px;--mono:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace}
*{box-sizing:border-box}
body{margin:0;background:var(--bg);color:var(--fg);
  font:15px/1.75 -apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Hiragino Sans GB","Microsoft YaHei",sans-serif}
a{color:var(--accent2);text-decoration:none}a:hover{text-decoration:underline}
.wrap{max-width:820px;margin:0 auto;padding:0 20px 70px}
header{display:flex;align-items:center;gap:14px;padding:22px 0 16px;flex-wrap:wrap}
.logo{display:flex;align-items:center;gap:10px;font-weight:700;font-size:19px}
.logo .mark{width:30px;height:30px;border-radius:8px;display:inline-flex;align-items:center;justify-content:center;
  background:linear-gradient(135deg,#0078d4,#41a5ee);color:#fff;font-size:15px;font-weight:800}
nav{margin-left:auto;display:flex;gap:6px}
nav a{color:var(--dim);padding:7px 12px;border-radius:8px;font-size:13.5px}
nav a:hover{color:var(--fg);background:var(--panel2);text-decoration:none}
.card{background:var(--panel);border:1px solid var(--line);border-radius:var(--radius);padding:22px 24px;margin-top:14px}
h1{font-size:22px;margin:2px 0 6px}h2{font-size:16.5px;margin:2px 0 12px}
p.lead{color:var(--dim);margin:0;font-size:14.5px}
table{border-collapse:collapse;width:100%;font-size:14px}
td{border-bottom:1px solid var(--line);padding:10px 6px}
td.k{color:var(--dim);width:200px}
.yes{color:var(--ok);font-weight:600}.no{color:var(--err);font-weight:600}.na{color:var(--warn)}
code,pre{font-family:var(--mono)}
pre{background:#0b0e13;border:1px solid var(--line);border-radius:10px;padding:14px 16px;
  font-size:12.8px;line-height:1.7;overflow:auto;margin:12px 0 4px;user-select:all}
.big{display:inline-block;background:var(--accent);color:#fff;border:0;border-radius:10px;
  padding:13px 30px;font-size:15px;font-weight:600;cursor:pointer;margin-top:14px}
.big:hover{background:var(--accent2)}.big:disabled{opacity:.55;cursor:not-allowed}
.port{background:var(--panel2);border:1px solid var(--line);color:var(--fg);border-radius:8px;
  padding:9px 10px;font-size:14px;width:90px;outline:none}
label{font-size:13.5px;color:var(--dim)}
#result{display:none;margin-top:14px;border-radius:10px;padding:13px 16px;font-size:14px;white-space:pre-wrap}
#result.ok{background:#12271a;border:1px solid #245c3a;color:#8fd9a8}
#result.err{background:#2d1a1a;border:1px solid #5c2f2f;color:#ff9c9c}
.spin{display:inline-block;width:14px;height:14px;border:2px solid rgba(255,255,255,.25);
  border-top-color:#fff;border-radius:50%;animation:sp .7s linear infinite;vertical-align:-2px;margin-right:6px}
@keyframes sp{to{transform:rotate(360deg)}}
footer{margin-top:26px;text-align:center;font-size:12.5px;color:var(--dim)}
</style>
</head>
<body>
<div class="wrap">
<header>
  <div class="logo"><span class="mark">b</span> 安装为系统服务</div>
  <nav><a href="/">测试界面</a><a href="/help">帮助文档</a></nav>
</header>

<div class="card">
<h1>一键安装到 systemd</h1>
<p class="lead">安装会把二进制复制到 <code>/usr/local/bin/bing-search-api</code>,注册 systemd 服务 <code>bing-search-api</code>,设置<b>开机自启动</b>并立即启动;崩溃 3 秒后自动重启。</p>
<table id="probe">
<tr><td class="k">环境检测中 …</td><td><span class="spin"></span></td></tr>
</table>
<div>
  <label>服务端口&nbsp;&nbsp;<input id="port" class="port" type="number" min="1" max="65535" value="__PORT__"></label>
</div>
<button class="big" id="go" onclick="installNow()">在当前进程权限内安装</button>
<div id="result"></div>
</div>

<div class="card">
<h2>推荐:在终端用 CLI 安装</h2>
<p class="lead">服务进程通常不是 root,推荐在服务器终端执行(会自动请求 sudo 提权):</p>
<pre>sudo /path/to/bing-search-api install -port 8080
<span style="color:#8b97a6"># 若已在 PATH:sudo bing-search-api install -port 8080</span>
<span style="color:#8b97a6"># 卸载:sudo bing-search-api uninstall</span></pre>
</div>

<div class="card">
<h2>安装后管理</h2>
<pre>systemctl status  bing-search-api
systemctl restart bing-search-api
journalctl -u bing-search-api -f</pre>
<p class="lead">访问 <code>http://服务器IP:端口/</code> 即测试界面,更改端口可重新执行 install。</p>
</div>

<footer>Bing Search API v__VERSION__</footer>
</div>

<script>
function probe(){
  var t=document.getElementById('probe');
  fetch('/install?probe=1').then(function(r){return r.json()}).then(function(d){
    t.innerHTML='';
    row(t,'当前进程是 root',d.root,'yes:no','root 才能直接安装');
    row(t,'systemd 可用',d.systemd,'yes:no','容器/WSL1 可能没有 systemd');
    row(t,'服务已安装',d.installed,'yes:na','unit 文件已存在');
    if(d.installed)row(t,'服务运行中',d.active,'yes:no','');
    row(t,'监听端口',d.port||'__PORT__','k:','');
    var go=document.getElementById('go');
    go.disabled=!d.systemd;
    go.textContent=d.root?'在当前进程权限内安装':'尝试安装(非 root 将给出指引)';
  }).catch(function(e){
    t.innerHTML='<tr><td class="k">检测失败</td><td class="no">'+e+'</td></tr>';
  });
}
function row(t,k,v,cls,tip){
  var tr=document.createElement('tr'),td1=document.createElement('td'),td2=document.createElement('td');
  td1.className='k';td1.textContent=k;
  if(cls==='k:'){td2.innerHTML='<code>'+v+'</code>'}
  else{var map={'yes':'<span class="yes">是 ✓</span>','no':'<span class="no">否 ✗</span>','na':'<span class="na">否</span>'};
    td2.innerHTML=map[cls.split(':')[0]];
    if(tip&&cls.indexOf('no')>=0)td2.innerHTML+=' <span style="color:var(--dim);font-size:12.5px">'+tip+'</span>';}
  tr.appendChild(td1);tr.appendChild(td2);t.appendChild(tr);
}
function installNow(){
  var go=document.getElementById('go'),res=document.getElementById('result');
  go.disabled=true;go.innerHTML='<span class="spin"></span>安装中 …';
  res.className='';res.style.display='none';
  var fd=new FormData();fd.append('port',document.getElementById('port').value||'8080');
  fetch('/install',{method:'POST',body:fd}).then(function(r){return r.json().then(function(d){return {ok:r.ok,d:d}})})
  .then(function(x){
    go.disabled=false;go.textContent='在当前进程权限内安装';
    res.style.display='block';
    if(x.ok){
      res.className='ok';
      res.textContent='安装成功 ✓ 服务 '+x.service+' 已启动(端口 '+x.port+')\n浏览器打开 http://'+location.hostname+':'+x.port+'/ 即测试界面\n管理: '+x.manage;
      probe();
    }else{
      res.className='err';
      res.textContent='✗ '+(x.d.error||'安装失败')+'\n\n'+(x.d.hint||'');
    }
  }).catch(function(e){
    go.disabled=false;go.textContent='在当前进程权限内安装';
    res.style.display='block';res.className='err';res.textContent='✗ 请求失败: '+e;
  });
}
probe();
</script>
</body>
</html>`
