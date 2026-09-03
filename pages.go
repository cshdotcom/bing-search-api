package main

// pages.go 内联页面:测试界面(访问 / 即打开)。
// 占位符 __VERSION__ / __LANG_COUNT__ / __PORT__ / __SEARXNG__ / __V7KEY__
// 由 renderPage 替换。
//
// 功能:双接口(SearXNG 兼容 / Bing 官方 v7)× 五类搜索(综合/图片/视频/新闻/词典);
// v7 模式提供 API Key 输入框(BING_API_KEY 鉴权场景的前端测试入口,
// 密钥仅存本浏览器 localStorage,随 Ocp-Apim-Subscription-Key 头发送)。

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

.keyrow{display:none;gap:10px;margin-top:12px;align-items:center;flex-wrap:wrap}
.keyrow input{flex:1;min-width:220px;background:var(--panel2);border:1px solid var(--line);color:var(--fg);
  border-radius:8px;padding:9px 12px;font:13px/1 var(--mono);outline:none}
.keyrow input:focus{border-color:var(--accent)}
.keyrow .hint{font-size:12px;color:var(--dim)}
.keyrow button{background:var(--panel2);border:1px solid var(--line);color:var(--dim);border-radius:8px;
  padding:8px 12px;font-size:12px;cursor:pointer;white-space:nowrap}
.keyrow button:hover{color:var(--fg);border-color:var(--accent)}

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
.rmeta{margin-left:33px;font-size:12.5px;color:var(--dim);margin-top:4px}
.rmeta b{color:#9ecbff;font-weight:600}

/* 图片网格 */
.igrid{display:grid;grid-template-columns:repeat(auto-fill,minmax(190px,1fr));gap:14px;margin-top:14px}
.tile{background:var(--panel2);border:1px solid var(--line);border-radius:10px;overflow:hidden;
  display:flex;flex-direction:column;transition:border .15s}
.tile:hover{border-color:var(--accent)}
.tile .timg{position:relative;height:140px;background:#0b0e13;display:flex;align-items:center;justify-content:center}
.tile img{max-width:100%;max-height:100%;object-fit:contain;display:block}
.tile .tt{padding:8px 10px;font-size:12.5px;line-height:1.45;color:var(--fg);
  display:-webkit-box;-webkit-line-clamp:2;-webkit-box-orient:vertical;overflow:hidden;min-height:38px}
.tile .th{padding:0 10px 8px;font:11px/1.4 var(--mono);color:var(--dim);word-break:break-all}

/* 视频卡片 */
.vgrid{display:grid;grid-template-columns:repeat(auto-fill,minmax(240px,1fr));gap:14px;margin-top:14px}
.vcard{background:var(--panel2);border:1px solid var(--line);border-radius:10px;overflow:hidden;transition:border .15s}
.vcard:hover{border-color:var(--accent)}
.vthumb{position:relative;height:135px;background:#0b0e13}
.vthumb img{width:100%;height:100%;object-fit:cover;display:block;opacity:.92}
.vdur{position:absolute;right:8px;bottom:8px;background:rgba(0,0,0,.75);color:#fff;
  font:11px/1 var(--mono);padding:4px 7px;border-radius:5px}
.vbody{padding:10px 12px 12px}
.vtitle{font-size:14px;font-weight:600;line-height:1.4;
  display:-webkit-box;-webkit-line-clamp:2;-webkit-box-orient:vertical;overflow:hidden}
.vhost{font:11.5px/1.4 var(--mono);color:var(--dim);margin-top:6px;word-break:break-all}

/* 词典卡片 */
.dictcard{margin-top:14px}
.dictcard .dword{font-size:26px;font-weight:700;line-height:1.3}
.dictcard .dpron{color:var(--dim);font-size:13.5px;margin-top:4px}
.dictcard .dpron span{margin-right:14px}
.dictcard ol{margin:14px 0 0;padding-left:22px}
.dictcard li{margin-bottom:8px;color:#c3ccd6;line-height:1.6}
.dictcard .dex{margin:14px 0 0;border-left:3px solid var(--chip);padding:4px 0 4px 14px}
.dictcard .dex h4{margin:0 0 6px;font-size:12.5px;color:var(--dim);font-weight:600}
.dictcard .dex p{margin:0 0 8px;font-size:13.5px;color:#aab6c2;line-height:1.55}
.dictempty{color:var(--dim);margin-top:14px;font-size:14px}
.answer{margin-top:14px;background:var(--panel2);border:1px solid var(--line);border-left:3px solid var(--accent);
  border-radius:8px;padding:12px 15px;font-size:14px;color:#c3ccd6;line-height:1.6}

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
@media(max-width:560px){.rdesc,.rurl,.rmeta{margin-left:0}.searchbar{flex-direction:column}
  .igrid{grid-template-columns:repeat(auto-fill,minmax(140px,1fr))}}
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
    <button id="go" onclick="doSearchReset()">搜 索</button>
  </div>
  <div class="opts">
    <label>接口
      <select id="mode">
        <option value="searxng">SearXNG 兼容 · /search</option>
        <option value="bing">Bing 官方 API v7 · /v7/*</option>
      </select>
    </label>
    <label>类别
      <select id="cat">
        <option value="">综合 · 网页</option>
        <option value="images">图片</option>
        <option value="videos">视频</option>
        <option value="news">新闻</option>
        <option value="dict">词典</option>
      </select>
    </label>
    <label><span id="langLabel">语言</span>
      <select id="lang"><option value="auto">auto · 浏览器语言</option><option value="all">all · 不限语言</option></select>
    </label>
    <label>每页
      <select id="count"><option>10</option><option>20</option><option>30</option><option>50</option></select>
    </label>
    <label><span id="pageLabel">页码</span> <input id="page" type="number" min="1" max="100" value="1"></label>
    <label id="safeRow" style="display:none">safeSearch
      <select id="safe"><option>Moderate</option><option>Off</option><option>Strict</option></select>
    </label>
  </div>
  <div class="keyrow" id="keyrow">
    <input id="apikey" type="password" placeholder="Ocp-Apim-Subscription-Key(设 BING_API_KEY 后必填)" autocomplete="off" spellcheck="false">
    <button id="keyToggle" onclick="toggleKey()" title="显示/隐藏">显示</button>
    <span class="hint" id="keyHint"></span>
  </div>
  <div id="notice" class="notice"></div>
  <div id="statusbar" class="statusbar"></div>
  <div id="requrl" class="requrl" style="display:none"></div>
</div>

<div class="card" id="resultCard" style="display:none">
  <div id="answer" class="answer" style="display:none"></div>
  <div id="results" class="results"></div>
  <div id="rel" class="rel" style="display:none"><h3>相关搜索</h3><div class="chips" id="chips"></div></div>
  <div class="pager">
    <button id="prev" onclick="pageStep(-1)">← 上一页</button>
    <span id="pageInfo" style="color:var(--dim);font-size:13px;min-width:120px;text-align:center"></span>
    <button id="next" onclick="pageStep(1)">下一页 →</button>
  </div>
  <details class="raw"><summary>原始 JSON 响应</summary><pre id="raw"></pre></details>
  <div class="curl">
    <code id="curl"></code>
    <button onclick="copyCurl(this)">复制 curl</button>
  </div>
</div>

<footer id="pageFooter">仅抓取 Bing · 网页 / 图片 / 视频 / 新闻 / 词典 · <span id="footerModes">SearXNG 兼容 + </span>Bing 官方 API v7 兼容 · 端口 __PORT__ · <a href="/help">API 文档</a></footer>
</div>

<script>
var $=function(s){return document.querySelector(s)};
var LANGS=null;
var MODE='searxng'; // searxng = /search;bing = /v7/*(官方 API 兼容)
var SEARXNG_ON=__SEARXNG__; // 服务端是否启用 SearXNG 兼容接口(默认 0)
var V7KEY_ON=__V7KEY__; // 服务端是否设置 BING_API_KEY(1=已启用鉴权)
var V7PATH={'':'', 'images':'images/', 'videos':'videos/', 'news':'news/', 'dict':'dict/'};
var CAT_LABEL={'':'网页', 'images':'图片', 'videos':'视频', 'news':'新闻', 'dict':'词典'};

// 接口模式切换:label 与参数随模式自适应
$('#mode').addEventListener('change',function(){
  MODE=this.value;
  syncUI();
});

// 类别切换:分页/语言/safeSearch 可用性随类别与模式变化
$('#cat').addEventListener('change',syncUI);

function syncUI(){
  var v7=MODE==='bing', cat=$('#cat').value;
  $('#langLabel').textContent=v7?'mkt':'语言';
  $('#pageLabel').textContent=v7?'offset':'页码';
  var pg=$('#page');
  pg.min=v7?'0':'1';
  pg.max=v7?'9000':'100';
  pg.value=v7?'0':'1';
  // safeSearch:v7 的 web/images/videos 有意义;news(RSS)/dict 忽略
  var safeUse=v7&&(cat===''||cat==='images'||cat==='videos');
  $('#safeRow').style.display=safeUse?'':'none';
  // 分页:图片/视频支持;新闻固定一批;词典为词条查询
  var pagable=cat===''||cat==='images'||cat==='videos';
  $('.pager').style.display=pagable?'flex':'none';
  // 语言:词典固定 zh-Hans 双语,隐藏 mkt 选择
  var langRow=$('#lang').parentNode;
  langRow.style.display=cat==='dict'?'none':'';
  // API Key 行:仅 v7 模式
  $('#keyrow').style.display=v7?'flex':'none';
}

// API Key 输入:仅存本浏览器 localStorage,随官方头发送
function toggleKey(){
  var inp=$('#apikey'),btn=$('#keyToggle');
  if(inp.type==='password'){inp.type='text';btn.textContent='隐藏'}
  else{inp.type='password';btn.textContent='显示'}
}
(function(){
  var k=localStorage.getItem('v7_api_key');
  if(k)$('#apikey').value=k;
  $('#apikey').addEventListener('input',function(){localStorage.setItem('v7_api_key',this.value)});
  $('#keyHint').textContent=V7KEY_ON?'服务端已启用密钥鉴权(BING_API_KEY 已设置)':'服务端未设 BING_API_KEY,留空即可';
})();

// 服务端禁用 /search(默认)时:移除该选项、锁定 v7 模式并给出提示
if(!SEARXNG_ON){
  var sel=$('#mode');
  if(sel.options[0]&&sel.options[0].value==='searxng')sel.remove(0);
  sel.value='bing';
  MODE='bing';
  sel.dispatchEvent(new Event('change'));
  var fm=$('#footerModes');if(fm)fm.textContent='';
  showNotice('info','SearXNG 兼容接口已禁用(默认):设 ENABLE_SEARXNG=1 并重启服务可开启,当前使用 Bing 官方 API v7 兼容接口');
}

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

$('#q').addEventListener('keydown',function(e){if(e.key==='Enter')doSearchReset()});

// 页码/offset 输入框:回车直接跳页(之前输入页码无任何反应,翻页“看似失效”)
$('#page').addEventListener('keydown',function(e){
  if(e.key!=='Enter')return;
  var v=this.value|0, min=MODE==='bing'?0:1;
  if(v<min){v=min;this.value=min}
  doSearch(v);
});

// 新搜索的起始点:searxng 第 1 页,bing v7 offset 0
function doSearchReset(){doSearch(MODE==='bing'?0:1)}

function showNotice(cls,msg){var n=$('#notice');n.className='notice '+cls;n.textContent=msg;n.style.display='block';if(cls==='err')window.scrollTo({top:0,behavior:'smooth'})}
function hideNotice(){$('#notice').style.display='none'}

var LAST=null;

function doSearch(pg){
  var q=$('#q').value.trim();
  if(!q){showNotice('warn','请输入查询词');return}
  $('#page').value=pg;
  hideNotice();
  var btn=$('#go');btn.disabled=true;btn.innerHTML='<span class="spin"></span>搜索中';
  var cat=$('#cat').value,lang=$('#lang').value,url,t0=performance.now(),headers={};
  if(MODE==='bing'){
    // Bing 官方 API v7 兼容:端点随类别切换,/v7/{images/,videos/,news/,dict/,}search
    var p={q:q,count:$('#count').value,offset:pg,safeSearch:$('#safe').value};
    if(lang!=='auto'&&lang!=='all'&&cat!=='dict')p.mkt=lang;
    url='/v7/'+V7PATH[cat]+'search?'+new URLSearchParams(p).toString();
    var key=$('#apikey').value.trim();
    if(key)headers['Ocp-Apim-Subscription-Key']=key;
  }else{
    // SearXNG 兼容:categories 参数选垂直
    var p={q:q,count:$('#count').value,page:pg};
    if(cat)p.categories=cat;
    if(lang!=='auto')p.language=lang;
    url='/search?'+new URLSearchParams(p).toString();
  }
  fetch(url,{headers:headers}).then(function(r){
    return r.json().then(function(d){return {ok:r.ok,status:r.status,d:d}})
  }).then(function(res){
    var ms=Math.round(performance.now()-t0);
    render(res,ms,url,q,lang,cat);
  }).catch(function(e){
    showNotice('err','请求失败: '+e);
  }).finally(function(){
    btn.disabled=false;btn.textContent='搜 索';
  });
}

// 错误消息兼容两种格式:/search 的 {error} 与 v7 的 {errors:[{message}]}
function apiErrMsg(d,status){
  if(d&&d.error)return d.error;
  if(d&&d.errors&&d.errors.length&&d.errors[0].message)
    return d.errors[0].message+(d.errors[0].parameter?(' (参数 '+d.errors[0].parameter+')'):'');
  return 'HTTP '+status;
}

// 统一条目:pos/title/url/desc/img/thumb/dur/date/source
function render(res,ms,url,q,lang,cat){
  var sb=$('#statusbar');
  $('#requrl').style.display='block';
  $('#requrl').textContent='GET '+url;
  if(!res.ok){
    sb.style.display='flex';
    sb.innerHTML='<span style="color:var(--err)">HTTP '+res.status+'</span>';
    showNotice('err',apiErrMsg(res.d,res.status));
    $('#resultCard').style.display='none';
    return;
  }
  var d=res.d;LAST={d:d,url:url,q:q,cat:cat};
  sb.style.display='flex';

  var items=[],suggestions=[],langTxt,total=null,answer=null,dict=null;
  if(MODE==='bing'){
    // v7 响应:按端点解析
    if(cat==='images'){
      items=(d.value||[]).map(function(it,i){
        return {pos:i+1,title:it.name,url:it.hostPageUrl||it.contentUrl,
          img:it.contentUrl,thumb:(it.thumbnail||{}).contentUrl||it.thumbnailUrl,desc:''};
      });
      total=(d.totalEstimatedMatches!==undefined)?d.totalEstimatedMatches:null;
      langTxt=$('#lang').value;
    }else if(cat==='videos'){
      items=(d.value||[]).map(function(it,i){
        return {pos:i+1,title:it.name,url:it.hostPageUrl||it.contentUrl,
          thumb:it.thumbnailUrl,dur:it.duration,content:it.contentUrl,
          pub:((it.publisher||[]).map(function(x){return x.name}).join(' / '))};
      });
      total=(d.totalEstimatedMatches!==undefined)?d.totalEstimatedMatches:null;
      langTxt=$('#lang').value;
    }else if(cat==='news'){
      items=(d.value||[]).map(function(it,i){
        return {pos:i+1,title:it.name,url:it.url,desc:it.description,
          date:it.datePublished,src:((it.provider||[]).map(function(x){return x.name}).join(' · '))};
      });
      langTxt=$('#lang').value;
    }else if(cat==='dict'){
      dict=d;langTxt='zh-Hans';
    }else{
      // v7 网页:webPages.value / relatedSearches.value
      var wp=d.webPages||{};
      items=(wp.value||[]).map(function(it,i){
        return {pos:i+1,title:it.name,url:it.url,desc:it.snippet};
      });
      suggestions=((d.relatedSearches||{}).value||[]).map(function(x){return x.displayText||x.text});
      langTxt=$('#lang').value; // v7 响应无 language 字段,展示所选 mkt
      total=(wp.totalEstimatedMatches!==undefined)?wp.totalEstimatedMatches:null;
    }
  }else{
    // SearXNG 风格响应:results 带垂直字段
    items=(d.results||[]).map(function(r,i){
      return {pos:r.position||i+1,title:r.title,url:r.url,desc:r.content,
        img:r.img_src,thumb:r.thumbnail_src,dur:r.length,date:r.publishedDate,src:r.source};
    });
    suggestions=d.suggestions||[];
    answer=(d.answers&&d.answers[0])||null;
    langTxt=d.language||'auto';
    if(cat==='dict'){
      dict={word:q,pronunciation:null,value:items.map(function(it){return {def:it.desc}})};
    }
  }

  var catTxt=CAT_LABEL[cat]||'网页';
  var sbHtml='<span><span class="dot"></span>完成</span>'+
    '<span class="kv">耗时 <b>'+ms+' ms</b></span>'+
    '<span class="kv">类别 <b>'+catTxt+'</b></span>'+
    '<span class="kv">结果 <b>'+(cat==='dict'?(dict?(dict.value||[]).length:0):items.length)+'</b> 条</span>';
  if(total!==null&&cat!=='dict')sbHtml+='<span class="kv">总数≈ <b>'+total+'</b></span>';
  sbHtml+='<span class="kv">'+(MODE==='bing'?'mkt':'语言')+' <b>'+langTxt+'</b></span>'+
    '<span class="kv">引擎 <b>bing</b></span>';
  sb.innerHTML=sbHtml;

  renderResults(cat,items,dict,answer);

  var rel=$('#rel');
  if(suggestions.length&&cat===''){
    rel.style.display='block';
    var ch=$('#chips');ch.innerHTML='';
    suggestions.forEach(function(s){
      var c=document.createElement('button');c.className='chip';c.textContent=s;
      c.onclick=function(){$('#q').value=s;doSearchReset();window.scrollTo({top:0,behavior:'smooth'})};
      ch.appendChild(c);
    });
  }else{rel.style.display='none'}

  $('#resultCard').style.display='block';
  $('#raw').textContent=JSON.stringify(d,null,2);
  var curlTxt='curl ';
  if(MODE==='bing'){var k=$('#apikey').value.trim();if(k)curlTxt+='-H "Ocp-Apim-Subscription-Key: '+k+'" '}
  curlTxt+='"'+location.origin+url+'"';
  $('#curl').textContent=curlTxt;
  var pagable=cat===''||cat==='images'||cat==='videos';
  updatePager(items,total,pagable);
  if(items.length===0&&!dict){
    showNotice('info',emptyHint(cat));
  }
}

// updatePager 翻页栏状态:上下文标签 + 按钮可用性。
// v7 模式显示 offset 区间(有总数时附总数);SearXNG 模式显示页码。
// next 禁用条件统一为"本页 0 条"(后端到末页返回空;totalEstimatedMatches
// 有 offset+len 兜底语义,不能作为末页判据)。
function updatePager(items,total,pagable){
  var info=$('#pageInfo');
  if(!pagable){info.textContent='';return}
  if(MODE==='bing'){
    var off=$('#page').value|0, n=items.length;
    var txt='offset '+off+(n?'–'+(off+n):'');
    if(total!==null&&total>0)txt+=' / 总数≈'+total;
    info.textContent=txt;
    $('#prev').disabled=off<=0;
    $('#next').disabled=n===0;
  }else{
    var pg=$('#page').value|0;
    info.textContent='第 '+pg+' 页'+($('#count').value?' · 每页 '+$('#count').value:'');
    $('#prev').disabled=pg<=1;
    $('#next').disabled=items.length===0;
  }
}

function emptyHint(cat){
  if(cat==='dict')return 'Bing 词典未收录该词(或该词无释义),试试其他拼写';
  if(cat==='news')return 'Bing 未返回新闻(该端点为 RSS 固定批次,可换关键词重试)';
  if(cat==='images')return 'Bing 未返回图片(可能 offset 超出范围/触发风控),可减小 offset 或换关键词重试';
  if(cat==='videos')return 'Bing 未返回视频(可能 offset 超出单页范围/触发风控),可换关键词重试';
  return MODE==='bing'
    ?'Bing 未返回结果(可能 offset 超出范围/触发风控),可减小 offset 或换关键词重试'
    :'Bing 未返回结果(可能触发风控/语言过滤),可换关键词或 language=all 重试';
}

// ── 分类别渲染 ──────────────────────────────────────────────
function renderResults(cat,items,dict,answer){
  var box=$('#results');box.innerHTML='';
  var ans=$('#answer');
  if(answer){ans.style.display='block';ans.textContent=answer}
  else if(dict&&MODE==='bing'&&dict.pronunciation){
    ans.style.display='none';
  }else{ans.style.display='none'}

  if(cat==='images'){renderImages(box,items);return}
  if(cat==='videos'){renderVideos(box,items);return}
  if(cat==='news'){renderNews(box,items);return}
  if(cat==='dict'){renderDict(box,dict,answer);return}
  renderWeb(box,items);
}

// 网页列表(原有样式)
function renderWeb(box,items){
  items.forEach(function(r){
    var div=document.createElement('div');div.className='result';
    var host='';try{host=new URL(r.url).hostname}catch(e){}
    var a=document.createElement('a');a.className='rtitle';a.href=r.url;a.target='_blank';a.rel='noopener';
    a.innerHTML='<span class="pos">'+r.pos+'</span>';
    a.appendChild(document.createTextNode(r.title||r.url));
    div.appendChild(a);
    var u=document.createElement('div');u.className='rurl';
    u.innerHTML='<a href="'+escAttr(r.url)+'" target="_blank" rel="noopener">'+esc(host||r.url)+'</a>';
    div.appendChild(u);
    if(r.desc){var p=document.createElement('div');p.className='rdesc';p.textContent=r.desc;div.appendChild(p)}
    box.appendChild(div);
  });
}

// 图片网格
function renderImages(box,items){
  var grid=document.createElement('div');grid.className='igrid';
  items.forEach(function(r){
    var tile=document.createElement('a');tile.className='tile';tile.href=r.url||r.img;tile.target='_blank';tile.rel='noopener';
    var t=document.createElement('div');t.className='timg';
    var img=document.createElement('img');
    img.src=r.thumb||r.img;img.loading='lazy';img.alt=r.title||'';
    img.onerror=function(){img.style.display='none'};
    t.appendChild(img);
    tile.appendChild(t);
    var cap=document.createElement('div');cap.className='tt';cap.textContent=r.title||'(无标题)';
    tile.appendChild(cap);
    var th=document.createElement('div');th.className='th';
    var host='';try{host=new URL(r.img||r.url).hostname}catch(e){}
    th.textContent=host;
    tile.appendChild(th);
    grid.appendChild(tile);
  });
  box.appendChild(grid);
}

// 视频卡片
function renderVideos(box,items){
  var grid=document.createElement('div');grid.className='vgrid';
  items.forEach(function(r){
    var card=document.createElement('a');card.className='vcard';card.href=r.url||r.content;card.target='_blank';card.rel='noopener';
    var th=document.createElement('div');th.className='vthumb';
    if(r.thumb){
      var img=document.createElement('img');img.src=r.thumb;img.loading='lazy';img.alt=r.title||'';
      img.onerror=function(){img.style.display='none'};
      th.appendChild(img);
    }
    if(r.dur){
      var dur=document.createElement('span');dur.className='vdur';
      dur.textContent=isoDur(r.dur)||r.dur;
      th.appendChild(dur);
    }
    card.appendChild(th);
    var body=document.createElement('div');body.className='vbody';
    var t=document.createElement('div');t.className='vtitle';t.textContent=r.title||'(无标题)';
    body.appendChild(t);
    var host='';try{host=new URL(r.url||r.content).hostname}catch(e){}
    var h=document.createElement('div');h.className='vhost';
    h.textContent=(r.pub?r.pub+' · ':'')+(host||'');
    body.appendChild(h);
    card.appendChild(body);
    grid.appendChild(card);
  });
  box.appendChild(grid);
}

// 新闻列表(带日期/来源元信息行)
function renderNews(box,items){
  items.forEach(function(r,i){
    var div=document.createElement('div');div.className='result';
    var host='';try{host=new URL(r.url).hostname}catch(e){}
    var a=document.createElement('a');a.className='rtitle';a.href=r.url;a.target='_blank';a.rel='noopener';
    a.innerHTML='<span class="pos">'+(r.pos||i+1)+'</span>';
    a.appendChild(document.createTextNode(r.title||r.url));
    div.appendChild(a);
    var u=document.createElement('div');u.className='rurl';
    u.innerHTML='<a href="'+escAttr(r.url)+'" target="_blank" rel="noopener">'+esc(r.src||host||r.url)+'</a>';
    div.appendChild(u);
    if(r.desc){var p=document.createElement('div');p.className='rdesc';p.textContent=r.desc;div.appendChild(p)}
    var meta=[];
    if(r.date){var dtxt=r.date;try{var dt=new Date(r.date);if(!isNaN(dt))dtxt=dt.toLocaleString()}catch(e){}
      meta.push('<span>发布于 <b>'+esc(dtxt)+'</b></span>')}
    if(r.src&&host&&r.src.toLowerCase()!==host.toLowerCase())meta.push('<span>来源 <b>'+esc(r.src)+'</b></span>');
    if(meta.length){var m=document.createElement('div');m.className='rmeta';m.innerHTML=meta.join(' · ');div.appendChild(m)}
    box.appendChild(div);
  });
}

// 词典卡片
function renderDict(box,dict,answer){
  if(!dict||((!dict.value||!dict.value.length)&&!answer&&!dict.word)){
    var e=document.createElement('div');e.className='dictempty';
    e.textContent='Bing 词典未收录该词,或该词无释义';
    box.appendChild(e);
    return;
  }
  var card=document.createElement('div');card.className='dictcard';
  var w=document.createElement('div');w.className='dword';
  w.textContent=dict.word||$('#q').value;
  card.appendChild(w);
  var pron=dict.pronunciation;
  var pr= [];
  if(pron){
    if(pron.us)pr.push('美 ['+pron.us+']');
    if(pron.uk)pr.push('英 ['+pron.uk+']');
    if(pron.pinyin)pr.push('拼音 ['+pron.pinyin+']');
  }
  if(pr.length){
    var pd=document.createElement('div');pd.className='dpron';
    pd.innerHTML=pr.map(function(x){return '<span>'+esc(x)+'</span>'}).join('');
    card.appendChild(pd);
  }
  var senses=(dict.value||[]).filter(function(s){return s&&(s.def||s.text)});
  if(senses.length){
    var ol=document.createElement('ol');
    senses.forEach(function(s){
      var li=document.createElement('li');
      var txt=s.def||s.text||'';
      if(s.pos)txt='('+s.pos+') '+txt;
      li.textContent=txt;
      if(s.examples&&s.examples.length){
        s.examples.forEach(function(x){
          var ex=document.createElement('div');ex.style.cssText='color:#8b97a6;font-size:12.5px;margin-top:3px';
          ex.textContent='例: '+x;
          li.appendChild(ex);
        });
      }
      ol.appendChild(li);
    });
    card.appendChild(ol);
  }else if(answer){
    // SearXNG 模式:义项列表为空时展示 answers 摘要
    var p=document.createElement('div');p.style.cssText='margin-top:12px;color:#c3ccd6';
    p.textContent=answer;
    card.appendChild(p);
  }
  box.appendChild(card);
}

// PT4M13S → 4:13(展示用)
function isoDur(iso){
  if(!iso||iso.indexOf('PT')!==0)return '';
  var m=/PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?/.exec(iso);
  if(!m)return '';
  var h=+(m[1]||0),mi=+(m[2]||0),s=+(m[3]||0);
  var ss=('0'+s).slice(-2);
  return h?h+':'+('0'+mi).slice(-2)+':'+ss:mi+':'+ss;
}

function pageStep(n){
  var cat=$('#cat').value;
  var pagable=cat===''||cat==='images'||cat==='videos';
  if(!pagable)return;
  var cur=$('#page').value|0;
  if(MODE==='bing'){
    // v7 模式:offset 按每页条数步进
    var step=(($('#count').value|0)||10);
    var next=cur+n*step;
    if(next<0)next=0;
    doSearch(next);
  }else{
    var nextp=cur+n;
    if(nextp<1)return;
    doSearch(nextp);
  }
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
