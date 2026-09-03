package main

// pages.go 内联页面:测试界面(访问 / 即打开)。
// 占位符 __VERSION__ / __LANG_COUNT__ / __PORT__ 由 renderPage 替换。

const testPageHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Bing Search API · 测试界面</title>
<style>
:root{
  --bg:#0f1216; --panel:#161b23; --panel2:#1c232e; --line:#2a3240;
  --fg:#e7edf3; --dim:#8b97a6; --accent:#3b8bff; --accent2:#67a2ff;
  --ok:#3fb96b; --warn:#e0a13a; --err:#ef6a6a; --chip:#223046;
  --radius:12px; --mono:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace;
}
*{box-sizing:border-box}
body{margin:0;background:var(--bg);color:var(--fg);
  font:15px/1.65 -apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Hiragino Sans GB","Microsoft YaHei",sans-serif;}
a{color:var(--accent2);text-decoration:none} a:hover{text-decoration:underline}
.wrap{max-width:880px;margin:0 auto;padding:0 20px 60px}

header{display:flex;align-items:center;gap:14px;padding:22px 0 14px;flex-wrap:wrap}
.logo{display:flex;align-items:center;gap:10px;font-weight:700;font-size:19px}
.logo .mark{width:30px;height:30px;border-radius:8px;display:inline-flex;align-items:center;justify-content:center;
  background:linear-gradient(135deg,#0078d4,#41a5ee);color:#fff;font-size:15px;font-weight:800}
.badge{font:11px/1 var(--mono);color:var(--dim);border:1px solid var(--line);border-radius:999px;padding:5px 10px}
nav{margin-left:auto;display:flex;gap:6px;flex-wrap:wrap}
nav a{color:var(--dim);padding:7px 12px;border-radius:8px;font-size:13.5px}
nav a:hover{color:var(--fg);background:var(--panel2);text-decoration:none}

.card{background:var(--panel);border:1px solid var(--line);border-radius:var(--radius);padding:20px;margin-top:14px}
.searchbar{display:flex;gap:10px}
.searchbar input[type=text]{flex:1;background:var(--panel2);border:1px solid var(--line);color:var(--fg);
  border-radius:10px;padding:12px 15px;font-size:16px;outline:none;transition:border .15s}
.searchbar input[type=text]:focus{border-color:var(--accent)}
.searchbar button{background:var(--accent);color:#fff;border:0;border-radius:10px;padding:12px 26px;
  font-size:15px;font-weight:600;cursor:pointer;transition:background .15s}
.searchbar button:hover{background:var(--accent2)} .searchbar button:disabled{opacity:.6;cursor:wait}

.opts{display:flex;gap:14px;margin-top:14px;flex-wrap:wrap;align-items:center}
.opts label{font-size:13px;color:var(--dim);display:flex;align-items:center;gap:7px}
select,input[type=number]{background:var(--panel2);border:1px solid var(--line);color:var(--fg);
  border-radius:8px;padding:8px 10px;font-size:13.5px;outline:none}
select:focus,input[type=number]:focus{border-color:var(--accent)}
input[type=number]{width:76px}

.statusbar{display:none;align-items:center;gap:16px;margin-top:16px;font-size:13px;color:var(--dim);flex-wrap:wrap}
.statusbar .kv b{color:var(--fg);font-weight:600}
.dot{width:7px;height:7px;border-radius:50%;background:var(--ok);display:inline-block;margin-right:5px}
.requrl{font:12px/1.5 var(--mono);color:var(--dim);background:var(--panel2);border-radius:8px;
  padding:8px 10px;margin-top:10px;word-break:break-all;user-select:all}

.results{margin-top:6px}
.result{padding:15px 6px 13px;border-bottom:1px solid var(--line)}
.result:last-child{border-bottom:0}
.pos{display:inline-flex;min-width:24px;height:24px;align-items:center;justify-content:center;
  font:12px/1 var(--mono);color:var(--accent2);background:var(--chip);border-radius:6px;margin-right:9px;vertical-align:2px}
.rtitle{font-size:16.5px;font-weight:600;line-height:1.45}
.rurl{font:12px/1.4 var(--mono);color:var(--dim);margin:3px 0 4px 33px;word-break:break-all;user-select:all}
.rurl a{color:var(--dim)} .rurl a:hover{color:var(--accent2)}
.rdesc{color:#c3ccd6;margin-left:33px;font-size:14px}
.rdesc:empty{display:none}

.rel{margin-top:18px}
.rel h3{font-size:13px;color:var(--dim);font-weight:600;margin:0 0 10px}
.rel .chips{display:flex;flex-wrap:wrap;gap:8px}
.chip{background:var(--chip);border:1px solid var(--line);color:var(--fg);border-radius:999px;
  padding:7px 14px;font-size:13px;cursor:pointer;transition:background .15s}
.chip:hover{background:#2b3d5c;text-decoration:none}

.pager{display:flex;justify-content:center;gap:10px;margin-top:18px}
.pager button{background:var(--panel2);border:1px solid var(--line);color:var(--fg);border-radius:8px;
  padding:9px 18px;cursor:pointer;font-size:13.5px}
.pager button:hover:not(:disabled){border-color:var(--accent)} .pager button:disabled{opacity:.4;cursor:default}

details.raw{margin-top:16px}
details.raw summary{cursor:pointer;font-size:13px;color:var(--dim)}
details.raw pre{background:#0b0e13;border:1px solid var(--line);border-radius:10px;padding:14px;
  font:12px/1.55 var(--mono);overflow:auto;max-height:420px;margin-top:10px}

.notice{display:none;border-radius:10px;padding:12px 15px;font-size:14px;margin-top:14px}
.notice.err{background:#2d1a1a;border:1px solid #5c2f2f;color:#ff9c9c}
.notice.warn{background:#2a2213;border:1px solid #5a4a26;color:var(--warn)}
.notice.info{background:#14212e;border:1px solid #2a4258;color:#9ecbff}

footer{margin-top:26px;font-size:12.5px;color:var(--dim);text-align:center}
.curl{display:flex;gap:8px;align-items:center;margin-top:12px}
.curl code{flex:1;font:12px/1.5 var(--mono);background:var(--panel2);border:1px solid var(--line);
  border-radius:8px;padding:9px 11px;word-break:break-all;user-select:all;display:block}
.curl button{background:var(--panel2);border:1px solid var(--line);color:var(--dim);border-radius:8px;
  padding:8px 12px;font-size:12px;cursor:pointer;white-space:nowrap}
.curl button:hover{color:var(--fg);border-color:var(--accent)}
.spin{display:inline-block;width:14px;height:14px;border:2px solid rgba(255,255,255,.25);
  border-top-color:#fff;border-radius:50%;animation:sp .7s linear infinite;vertical-align:-2px;margin-right:6px}
@keyframes sp{to{transform:rotate(360deg)}}
@media(max-width:560px){.rdesc,.rurl{margin-left:0}.searchbar{flex-direction:column}}
</style>
</head>
<body>
<div class="wrap">
<header>
  <div class="logo"><span class="mark">b</span> Bing Search API</div>
  <span class="badge">v__VERSION__ · __LANG_COUNT__ 语言</span>
  <nav>
    <a href="/help">帮助文档</a>
    <a href="/languages">语言列表</a>
    <a href="https://github.com/cshdotcom/bing-search-api" target="_blank">GitHub</a>
  </nav>
</header>

<div class="card">
  <div class="searchbar">
    <input id="q" type="text" placeholder="输入关键词,回车或点击搜索(仅 Bing,结果保持原始顺序)" autocomplete="off" autofocus>
    <button id="go" onclick="doSearch(1)">搜 索</button>
  </div>
  <div class="opts">
    <label>语言
      <select id="lang"><option value="auto">auto · 浏览器语言</option><option value="all">all · 不限语言</option></select>
    </label>
    <label>每页
      <select id="count"><option>10</option><option>20</option><option>30</option><option>50</option></select>
    </label>
    <label>页码 <input id="page" type="number" min="1" max="100" value="1"></label>
  </div>
  <div id="notice" class="notice"></div>
  <div id="statusbar" class="statusbar"></div>
  <div id="requrl" class="requrl" style="display:none"></div>
</div>

<div class="card" id="resultCard" style="display:none">
  <div id="results" class="results"></div>
  <div id="rel" class="rel" style="display:none"><h3>相关搜索</h3><div class="chips" id="chips"></div></div>
  <div class="pager">
    <button id="prev" onclick="pageStep(-1)">← 上一页</button>
    <button id="next" onclick="pageStep(1)">下一页 →</button>
  </div>
  <details class="raw"><summary>原始 JSON 响应</summary><pre id="raw"></pre></details>
  <div class="curl">
    <code id="curl"></code>
    <button onclick="copyCurl(this)">复制 curl</button>
  </div>
</div>

<footer>仅抓取 Bing 网页结果 · 响应结构与 SearXNG 兼容 · 端口 __PORT__ · <a href="/help">API 文档</a></footer>
</div>

<script>
var $=function(s){return document.querySelector(s)};
var LANGS=null;

// 语言下拉:从 /languages 动态加载
fetch('/languages').then(function(r){return r.json()}).then(function(d){
  LANGS=d.languages||[];
  var sel=$('#lang'),cur=sel.value;
  var groups={};
  LANGS.forEach(function(l){
    var g=l.name.replace(/\s*\(.*\)/,'');
    (groups[g]=groups[g]||[]).push(l);
  });
  Object.keys(groups).sort().forEach(function(g){
    var og=document.createElement('optgroup');og.label=g;
    groups[g].forEach(function(l){
      var o=document.createElement('option');o.value=l.code;
      o.textContent=l.code+' · '+l.name;og.appendChild(o);
    });
    sel.appendChild(og);
  });
  sel.value=cur;
  var n=navigator.language||'';
  if(!n)return;
  var best=LANGS.filter(function(l){return l.code.toLowerCase()===n.toLowerCase()})[0];
  if(!best)best=LANGS.filter(function(l){return l.code.toLowerCase().indexOf((n.split('-')[0]||'').toLowerCase()+'-')===0})[0];
  if(best)sel.value=best.code;
}).catch(function(){});

$('#q').addEventListener('keydown',function(e){if(e.key==='Enter')doSearch(1)});

function showNotice(cls,msg){var n=$('#notice');n.className='notice '+cls;n.textContent=msg;n.style.display='block';if(cls==='err')window.scrollTo({top:0,behavior:'smooth'})}
function hideNotice(){$('#notice').style.display='none'}

var LAST=null;

function doSearch(page){
  var q=$('#q').value.trim();
  if(!q){showNotice('warn','请输入查询词');return}
  $('#page').value=page;
  hideNotice();
  var btn=$('#go');btn.disabled=true;btn.innerHTML='<span class="spin"></span>搜索中';
  var p={q:q,count:$('#count').value,page:page};
  var lang=$('#lang').value;
  if(lang!=='auto')p.language=lang;
  var qs=new URLSearchParams(p).toString();
  var url='/search?'+qs;
  var t0=performance.now();
  fetch(url).then(function(r){
    return r.json().then(function(d){return {ok:r.ok,status:r.status,d:d}})
  }).then(function(res){
    var ms=Math.round(performance.now()-t0);
    render(res,ms,url,q,lang);
  }).catch(function(e){
    showNotice('err','请求失败: '+e);
  }).finally(function(){
    btn.disabled=false;btn.textContent='搜 索';
  });
}

function render(res,ms,url,q,lang){
  var sb=$('#statusbar');
  $('#requrl').style.display='block';
  $('#requrl').textContent='GET '+url;
  if(!res.ok){
    sb.style.display='flex';
    sb.innerHTML='<span style="color:var(--err)">HTTP '+res.status+'</span>';
    showNotice('err',(res.d&&res.d.error)||('HTTP '+res.status));
    $('#resultCard').style.display='none';
    return;
  }
  var d=res.d;LAST={d:d,url:url,q:q};
  sb.style.display='flex';
  var langTxt=d.language||'auto';
  sb.innerHTML='<span><span class="dot"></span>完成</span>'+
    '<span class="kv">耗时 <b>'+ms+' ms</b></span>'+
    '<span class="kv">结果 <b>'+d.number_of_results+'</b> 条</span>'+
    '<span class="kv">语言 <b>'+langTxt+'</b></span>'+
    '<span class="kv">引擎 <b>bing</b></span>';

  var box=$('#results');box.innerHTML='';
  (d.results||[]).forEach(function(r){
    var div=document.createElement('div');div.className='result';
    var host='';try{host=new URL(r.url).hostname}catch(e){}
    var a=document.createElement('a');a.className='rtitle';a.href=r.url;a.target='_blank';a.rel='noopener';
    a.innerHTML='<span class="pos">'+r.position+'</span>';
    a.appendChild(document.createTextNode(r.title||r.url));
    div.appendChild(a);
    var u=document.createElement('div');u.className='rurl';
    u.innerHTML='<a href="'+escAttr(r.url)+'" target="_blank" rel="noopener">'+esc(host||r.url)+'</a>';
    div.appendChild(u);
    if(r.content){var p=document.createElement('div');p.className='rdesc';p.textContent=r.content;div.appendChild(p)}
    box.appendChild(div);
  });

  var rel=$('#rel');
  if(d.suggestions&&d.suggestions.length){
    rel.style.display='block';
    var ch=$('#chips');ch.innerHTML='';
    d.suggestions.forEach(function(s){
      var c=document.createElement('button');c.className='chip';c.textContent=s;
      c.onclick=function(){$('#q').value=s;doSearch(1);window.scrollTo({top:0,behavior:'smooth'})};
      ch.appendChild(c);
    });
  }else{rel.style.display='none'}

  $('#resultCard').style.display='block';
  $('#raw').textContent=JSON.stringify(d,null,2);
  $('#curl').textContent='curl "'+location.origin+url+'"';
  $('#prev').disabled=($('#page').value|0)<=1;
  $('#next').disabled=(d.results||[]).length===0;
  if((d.results||[]).length===0){
    showNotice('info','Bing 未返回结果(可能触发风控/语言过滤),可换关键词或 language=all 重试');
  }
}

function pageStep(n){
  var cur=$('#page').value|0;var next=cur+n;
  if(next<1)return;doSearch(next);
}

function copyCurl(btn){
  var t=$('#curl').textContent;
  function done(ok){btn.textContent=ok?'已复制 ✓':'复制失败';setTimeout(function(){btn.textContent='复制 curl'},1500)}
  if(navigator.clipboard&&navigator.clipboard.writeText){navigator.clipboard.writeText(t).then(function(){done(true)},function(){done(false)})}
  else{var ta=document.createElement('textarea');ta.value=t;document.body.appendChild(ta);ta.select();
    done(document.execCommand('copy'));document.body.removeChild(ta)}
}

function esc(s){return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;')}
function escAttr(s){return esc(s).replace(/"/g,'&quot;')}
</script>
</body>
</html>`
