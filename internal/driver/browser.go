package driver

func backendTarget(target string) string {
	if target == "browser/wasm32" {
		return "wasi/wasm32"
	}
	return target
}

// PackageBrowserHTML embeds a WASI module and its complete browser host in one
// file. The host displays fd 1/2 as a terminal and fd 3 as RGBA frames.
func PackageBrowserHTML(wasm []byte) []byte {
	out := make([]byte, 0, len(browserHTMLPrefix)+len(browserHTMLSuffix)+(len(wasm)+2)/3*4)
	out = append(out, browserHTMLPrefix...)
	out = appendBase64(out, wasm)
	out = append(out, browserHTMLSuffix...)
	return out
}

func appendBase64(out []byte, data []byte) []byte {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	for i := 0; i < len(data); i += 3 {
		value := int(data[i]) << 16
		remaining := len(data) - i
		if remaining > 1 {
			value |= int(data[i+1]) << 8
		}
		if remaining > 2 {
			value |= int(data[i+2])
		}
		out = append(out, alphabet[(value>>18)&63], alphabet[(value>>12)&63])
		if remaining > 1 {
			out = append(out, alphabet[(value>>6)&63])
		} else {
			out = append(out, '=')
		}
		if remaining > 2 {
			out = append(out, alphabet[value&63])
		} else {
			out = append(out, '=')
		}
	}
	return out
}

const browserHTMLPrefix = `<!doctype html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Renvo Desktop</title><style>
html,body,#desktop{width:100%;height:100%;margin:0;overflow:hidden}body{background:#1d2128;color:#e8e8e8;font:14px ui-monospace,SFMono-Regular,Consolas,monospace}
#desktop{position:relative}.renvo-window{position:absolute;display:flex;flex-direction:column;background:#f7f8fa;box-shadow:0 18px 70px #0009;overflow:hidden}.renvo-window.root{inset:0;box-shadow:none}.renvo-window.floating{left:8%;top:8%;width:640px;height:420px;min-width:240px;min-height:160px;border:1px solid #596171;border-radius:7px;resize:both}
.renvo-title{height:30px;flex:0 0 30px;display:flex;align-items:center;padding-left:11px;background:#303744;color:#fff;cursor:move;user-select:none}.renvo-title span{flex:1;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.renvo-close{width:42px;height:30px;border:0;background:transparent;color:#fff;font:18px sans-serif}.renvo-close:hover{background:#c42b3a}.renvo-canvas{display:block;flex:1 1 auto;width:100%;height:100%;min-height:0;outline:none}.renvo-terminal{box-sizing:border-box;flex:1;margin:0;padding:14px;overflow:auto;background:#12151a;color:#e8e8e8;white-space:pre-wrap}
</style></head><body><div id="desktop"></div><script>
"use strict";const wasm64="`

const browserHTMLSuffix = `";
const desktop=document.getElementById("desktop"),decoder=new TextDecoder(),encoder=new TextEncoder(),EXIT={};let topZ=10,rootContext=null;
const files=new Map(),directories=new Set([".","workspace","std"]),storagePrefix="renvo-vfs:";
function bytes64(data){let out="",step=32768;for(let i=0;i<data.length;i+=step)out+=String.fromCharCode.apply(null,data.subarray(i,Math.min(i+step,data.length)));return btoa(out)}
function loadFiles(){try{for(let i=0;i<localStorage.length;i++){const k=localStorage.key(i);if(k&&k.indexOf(storagePrefix)===0){const raw=atob(localStorage.getItem(k)),data=Uint8Array.from(raw,c=>c.charCodeAt(0));files.set(k.slice(storagePrefix.length),data)}}}catch(e){}}
function persist(path,data){if(data.length>262144)return;try{localStorage.setItem(storagePrefix+path,bytes64(data))}catch(e){}}
function clean(path,base="."){const input=(path&&path[0]==="/"?path:base+"/"+path).split("/"),out=[];for(const part of input){if(!part||part===".")continue;if(part==="..")out.pop();else out.push(part)}return out.join("/")||"."}
function parent(path){const at=path.lastIndexOf("/");return at<0?".":path.slice(0,at)||"."}
function isDirectory(path){if(directories.has(path))return true;const prefix=path+"/";for(const name of files.keys())if(name.indexOf(prefix)===0)return true;return false}
function directoryEntries(path){const prefix=path==="."?"":path+"/",found=new Map();for(const name of files.keys()){if(name.indexOf(prefix)!==0)continue;const rest=name.slice(prefix.length),at=rest.indexOf("/"),entry=at<0?rest:rest.slice(0,at);if(entry)found.set(entry,at>=0)}return Array.from(found,([name,dir])=>({name,dir})).sort((a,b)=>a.name.localeCompare(b.name))}
function put32(a,o,v){a[o]=v&255;a[o+1]=(v>>>8)&255;a[o+2]=(v>>>16)&255;a[o+3]=(v>>>24)&255}
function contextView(ctx){return new DataView(ctx.memory.buffer)}
function iovecs(ctx,at,count){const view=contextView(ctx),out=[];for(let i=0;i<count;i++){const ptr=view.getUint32(at+i*8,true),len=view.getUint32(at+i*8+4,true);out.push(new Uint8Array(ctx.memory.buffer,ptr,len))}return out}
function eventRecord(type,x=0,y=0,wx=0,wy=0,key=0,button=0,mods=0,repeat=0,text=""){const t=encoder.encode(text),a=new Uint8Array(40+t.length);put32(a,0,type);put32(a,4,x);put32(a,8,y);put32(a,12,wx);put32(a,16,wy);put32(a,20,key);put32(a,24,button);put32(a,28,mods);put32(a,32,repeat);put32(a,36,t.length);a.set(t,40);return a}
function enqueue(ctx,...args){ctx.input.push(eventRecord(...args));stepContext(ctx)}
function stepContext(ctx){if(ctx.instance&&ctx.instance.exports.renvo_browser_step){try{ctx.instance.exports.renvo_browser_step()}catch(e){if(e!==EXIT)showError(ctx,e)}}}
function writeFile(fd,parts,offset){let size=0;for(const part of parts)size+=part.length;const start=offset===undefined?fd.pos:offset,end=start+size,old=files.get(fd.path)||new Uint8Array(0),next=new Uint8Array(Math.max(old.length,end));next.set(old);let at=start;for(const part of parts){next.set(part,at);at+=part.length}files.set(fd.path,next);fd.pos=end;persist(fd.path,next);return size}
function readFile(ctx,fd,parts,offset){const data=files.get(fd.path)||new Uint8Array(0);let at=offset===undefined?fd.pos:offset,used=0;for(const part of parts){const count=Math.min(part.length,data.length-at);if(count<=0)break;part.set(data.subarray(at,at+count));at+=count;used+=count}fd.pos=at;return used}
function fdWrite(ctx,fd,at,count,written){const parts=iovecs(ctx,at,count),view=contextView(ctx);let size=0;for(const part of parts)size+=part.length;view.setUint32(written,size,true);if(fd===3){for(const part of parts)graphicsData(ctx,part);return 0}if(fd===4){for(const part of parts)launchFile(ctx,decoder.decode(part));return 0}if(fd===1||fd===2){const terminal=ensureTerminal(ctx);for(const part of parts)terminal.textContent+=decoder.decode(part,{stream:true});terminal.scrollTop=terminal.scrollHeight;return 0}const file=ctx.fds.get(fd);if(!file||file.kind!=="file")return 8;writeFile(file,parts);return 0}
function fdRead(ctx,fd,at,count,read){const parts=iovecs(ctx,at,count),view=contextView(ctx);if(fd===0){if(ctx.input.length===0){view.setUint32(read,0,true);return 0}const source=ctx.input.shift();let used=0;for(const part of parts){const size=Math.min(part.length,source.length-used);part.set(source.subarray(used,used+size));used+=size;if(used===source.length)break}view.setUint32(read,used,true);return 0}const file=ctx.fds.get(fd);if(!file||file.kind!=="file")return 8;view.setUint32(read,readFile(ctx,file,parts),true);return 0}
function fdPread(ctx,fd,at,count,offset,read){const file=ctx.fds.get(fd);if(!file||file.kind!=="file")return 8;contextView(ctx).setUint32(read,readFile(ctx,file,iovecs(ctx,at,count),Number(offset)),true);return 0}
function fdPwrite(ctx,fd,at,count,offset,written){const file=ctx.fds.get(fd);if(!file||file.kind!=="file")return 8;contextView(ctx).setUint32(written,writeFile(file,iovecs(ctx,at,count),Number(offset)),true);return 0}
function pathOpen(ctx,dirfd,dirflags,pathAt,pathLen,oflags,rightsBase,rightsInherit,fdflags,result){const path=clean(decoder.decode(new Uint8Array(ctx.memory.buffer,pathAt,pathLen))),create=(oflags&1)!==0,trunc=(oflags&8)!==0,dir=isDirectory(path);if(!dir&&!files.has(path)&&!create)return 44;if(create&&!files.has(path))files.set(path,new Uint8Array(0));if(trunc&&!dir)files.set(path,new Uint8Array(0));const fd=ctx.nextFd++;ctx.fds.set(fd,{kind:dir?"dir":"file",path,pos:0});contextView(ctx).setUint32(result,fd,true);return 0}
function fdClose(ctx,fd){const file=ctx.fds.get(fd);if(file&&file.kind==="file")persist(file.path,files.get(file.path)||new Uint8Array(0));ctx.fds.delete(fd);return 0}
function fdStat(ctx,fd,at){const file=ctx.fds.get(fd);if(fd===3||file){new Uint8Array(ctx.memory.buffer,at,24).fill(0);new Uint8Array(ctx.memory.buffer)[at]=file&&file.kind==="dir"?3:4;return 0}return 8}
function fdReaddir(ctx,fd,at,len,cookie,usedAt){const file=ctx.fds.get(fd),view=contextView(ctx);if(!file||file.kind!=="dir")return 54;const entries=directoryEntries(file.path),dest=new Uint8Array(ctx.memory.buffer,at,len);let used=0,start=Number(cookie);for(let i=start;i<entries.length;i++){const name=encoder.encode(entries[i].name),size=24+name.length;if(used+size>len)break;const d=new DataView(dest.buffer,dest.byteOffset+used,size);d.setBigUint64(0,BigInt(i+1),true);d.setBigUint64(8,BigInt(i+1),true);d.setUint32(16,name.length,true);d.setUint8(20,entries[i].dir?3:4);dest.set(name,used+24);used+=size}view.setUint32(usedAt,used,true);return 0}
function stringSizes(values){let size=0;for(const value of values)size+=encoder.encode(value).length+1;return size}
function writeStrings(ctx,values,pointers,dataAt){const view=contextView(ctx),memoryBytes=new Uint8Array(ctx.memory.buffer);let at=dataAt;for(let i=0;i<values.length;i++){view.setUint32(pointers+i*4,at,true);const data=encoder.encode(values[i]);memoryBytes.set(data,at);at+=data.length;memoryBytes[at++]=0}return 0}
function importsFor(ctx){const args=ctx.root?["renvoide","/workspace"]:["app"],env=["PWD="+ctx.cwd];return{wasi_snapshot_preview1:{fd_write:(...a)=>fdWrite(ctx,...a),fd_read:(...a)=>fdRead(ctx,...a),fd_pread:(...a)=>fdPread(ctx,...a),fd_pwrite:(...a)=>fdPwrite(ctx,...a),path_open:(...a)=>pathOpen(ctx,...a),fd_close:fd=>fdClose(ctx,fd),fd_fdstat_get:(fd,at)=>fdStat(ctx,fd,at),fd_readdir:(...a)=>fdReaddir(ctx,...a),args_sizes_get:(c,s)=>{const v=contextView(ctx);v.setUint32(c,args.length,true);v.setUint32(s,stringSizes(args),true);return 0},args_get:(p,d)=>writeStrings(ctx,args,p,d),environ_sizes_get:(c,s)=>{const v=contextView(ctx);v.setUint32(c,env.length,true);v.setUint32(s,stringSizes(env),true);return 0},environ_get:(p,d)=>writeStrings(ctx,env,p,d),proc_exit:()=>{throw EXIT}}}}
function graphicsData(ctx,part){if(ctx.pending){const pending=ctx.pending;ctx.pending=null;if(part.length!==pending.length)return;if(pending.kind==="window")createWindow(ctx,pending.id,pending.width,pending.height,decoder.decode(part));else if(pending.kind==="frame")presentWindow(ctx,pending.id,pending.width,pending.height,part);else if(pending.kind==="title")setWindowTitle(ctx,pending.id,decoder.decode(part));return}if(part.length<8)return;const magic=String.fromCharCode(part[0],part[1],part[2],part[3]),d=new DataView(part.buffer,part.byteOffset,part.byteLength),id=d.getUint32(4,true);if(magic==="RNVX"){closeWindow(ctx,id);return}if(magic==="RNVW"&&part.length===20){const pending={kind:"window",id,width:d.getUint32(8,true),height:d.getUint32(12,true),length:d.getUint32(16,true)};if(pending.length===0)createWindow(ctx,id,pending.width,pending.height,"");else ctx.pending=pending}else if(magic==="RNVF"&&part.length===20)ctx.pending={kind:"frame",id,width:d.getUint32(8,true),height:d.getUint32(12,true),length:d.getUint32(16,true)};else if(magic==="RNVT"&&part.length===12)ctx.pending={kind:"title",id,length:d.getUint32(8,true)};else if(magic==="RNVM"&&part.length===16)setWindowTimer(ctx,id,d.getUint32(8,true),d.getUint32(12,true));else if(magic==="RNVC"&&part.length===12)cancelWindowTimer(ctx,id,d.getUint32(8,true))}
function makeShell(ctx,title,width,height,terminal=false){const shell=document.createElement("div"),root=ctx.root&&ctx.windows.size===0;shell.className="renvo-window "+(root?"root":"floating");if(!root){shell.style.width=width+"px";shell.style.height=(height+30)+"px";const bar=document.createElement("div");bar.className="renvo-title";const caption=document.createElement("span");caption.textContent=title||"Renvo Application";const close=document.createElement("button");close.className="renvo-close";close.textContent="×";bar.append(caption,close);shell.append(bar);shell.caption=caption;close.onclick=()=>{if(shell.view)enqueue(ctx,1);shell.remove()};dragWindow(shell,bar)}desktop.append(shell);focusWindow(shell);if(terminal){const pre=document.createElement("pre");pre.className="renvo-terminal";shell.append(pre);return{shell,terminal:pre,root}}const canvas=document.createElement("canvas");canvas.className="renvo-canvas";canvas.tabIndex=0;shell.append(canvas);return{shell,canvas,root}}
function focusWindow(shell){if(!shell.classList.contains("root"))shell.style.zIndex=String(++topZ)}
function dragWindow(shell,bar){bar.addEventListener("pointerdown",e=>{if(e.target.classList.contains("renvo-close"))return;focusWindow(shell);bar.setPointerCapture(e.pointerId);const sx=e.clientX,sy=e.clientY,left=shell.offsetLeft,top=shell.offsetTop;const move=m=>{shell.style.left=left+m.clientX-sx+"px";shell.style.top=top+m.clientY-sy+"px"};const up=()=>{bar.removeEventListener("pointermove",move);bar.removeEventListener("pointerup",up)};bar.addEventListener("pointermove",move);bar.addEventListener("pointerup",up)})}
function createWindow(ctx,id,width,height,title){const made=makeShell(ctx,title,width,height),view={id,ctx,...made,width,height,gl:null,texture:null};made.shell.view=view;ctx.windows.set(id,view);wireCanvas(view);if(view.root)requestAnimationFrame(()=>resizeView(view));else new ResizeObserver(()=>resizeView(view)).observe(view.canvas)}
function closeWindow(ctx,id){const view=ctx.windows.get(id);if(view){view.shell.remove();ctx.windows.delete(id)}for(const [key,timer] of ctx.timers)if(key.indexOf(id+":")===0){clearTimeout(timer);ctx.timers.delete(key)}}
function timerKey(id,timerID){return id+":"+timerID}
function cancelWindowTimer(ctx,id,timerID){const key=timerKey(id,timerID),timer=ctx.timers.get(key);if(timer!==undefined){clearTimeout(timer);ctx.timers.delete(key)}}
function setWindowTimer(ctx,id,timerID,milliseconds){cancelWindowTimer(ctx,id,timerID);const key=timerKey(id,timerID),timer=setTimeout(()=>{ctx.timers.delete(key);enqueue(ctx,14,0,0,0,0,timerID)},milliseconds);ctx.timers.set(key,timer)}
function setWindowTitle(ctx,id,title){const view=ctx.windows.get(id);if(view&&view.shell.caption)view.shell.caption.textContent=title}
function initGL(view){const gl=view.canvas.getContext("webgl2",{alpha:false,antialias:false,preserveDrawingBuffer:true});if(!gl)throw Error("WebGL2 is required");const shader=(type,source)=>{const item=gl.createShader(type);gl.shaderSource(item,source);gl.compileShader(item);if(!gl.getShaderParameter(item,gl.COMPILE_STATUS))throw Error(gl.getShaderInfoLog(item));return item},nl=String.fromCharCode(10),vs="#version 300 es"+nl+"in vec2 p;out vec2 uv;void main(){uv=vec2((p.x+1.)*.5,(1.-p.y)*.5);gl_Position=vec4(p,0,1);}",fs="#version 300 es"+nl+"precision mediump float;in vec2 uv;uniform sampler2D t;out vec4 c;void main(){c=texture(t,uv);}",program=gl.createProgram();gl.attachShader(program,shader(gl.VERTEX_SHADER,vs));gl.attachShader(program,shader(gl.FRAGMENT_SHADER,fs));gl.linkProgram(program);gl.useProgram(program);const buffer=gl.createBuffer();gl.bindBuffer(gl.ARRAY_BUFFER,buffer);gl.bufferData(gl.ARRAY_BUFFER,new Float32Array([-1,-1,1,-1,-1,1,-1,1,1,-1,1,1]),gl.STATIC_DRAW);const p=gl.getAttribLocation(program,"p");gl.enableVertexAttribArray(p);gl.vertexAttribPointer(p,2,gl.FLOAT,false,0,0);view.texture=gl.createTexture();gl.bindTexture(gl.TEXTURE_2D,view.texture);gl.texParameteri(gl.TEXTURE_2D,gl.TEXTURE_MIN_FILTER,gl.NEAREST);gl.texParameteri(gl.TEXTURE_2D,gl.TEXTURE_MAG_FILTER,gl.NEAREST);view.gl=gl}
function presentWindow(ctx,id,width,height,pixels){const view=ctx.windows.get(id);if(!view)return;if(!view.gl)initGL(view);const gl=view.gl;view.canvas.width=width;view.canvas.height=height;view.width=width;view.height=height;gl.viewport(0,0,width,height);gl.bindTexture(gl.TEXTURE_2D,view.texture);gl.texImage2D(gl.TEXTURE_2D,0,gl.RGBA,width,height,0,gl.RGBA,gl.UNSIGNED_BYTE,pixels);gl.drawArrays(gl.TRIANGLES,0,6);view.canvas.focus()}
function resizeView(view){const width=Math.max(1,Math.round(view.canvas.clientWidth)),height=Math.max(1,Math.round(view.canvas.clientHeight));if(width!==view.width||height!==view.height)enqueue(view.ctx,2,width,height)}
function modifiers(e){return(e.shiftKey?1:0)|(e.ctrlKey?2:0)|(e.altKey?4:0)|(e.metaKey?8:0)}
const keys={Backspace:1,Delete:2,Enter:3,Tab:4,Escape:5," ":6,ArrowLeft:7,ArrowRight:8,ArrowUp:9,ArrowDown:10,Home:11,End:12,PageUp:13,PageDown:14,a:15,b:16,c:17,i:18,n:19,o:20,q:21,s:22,v:23,x:24,y:25,z:26};
function wireCanvas(view){const canvas=view.canvas,ctx=view.ctx,point=e=>{const rect=canvas.getBoundingClientRect();return[Math.round((e.clientX-rect.left)*view.width/rect.width),Math.round((e.clientY-rect.top)*view.height/rect.height)]};canvas.addEventListener("pointerdown",e=>{focusWindow(view.shell);canvas.setPointerCapture(e.pointerId);const p=point(e);enqueue(ctx,7,p[0],p[1],0,0,0,e.button+1,modifiers(e))});canvas.addEventListener("pointerup",e=>{const p=point(e);enqueue(ctx,8,p[0],p[1],0,0,0,e.button+1,modifiers(e))});canvas.addEventListener("pointermove",e=>{const p=point(e);enqueue(ctx,6,p[0],p[1],0,0,0,e.buttons,modifiers(e))});canvas.addEventListener("wheel",e=>{e.preventDefault();const p=point(e);enqueue(ctx,9,p[0],p[1],Math.round(e.deltaX),Math.round(e.deltaY),0,0,modifiers(e))},{passive:false});canvas.addEventListener("keydown",e=>{const key=keys[e.key]||keys[e.key.toLowerCase()]||0;enqueue(ctx,11,0,0,0,0,key,0,modifiers(e),e.repeat?1:0);if(e.key.length===1&&!e.ctrlKey&&!e.altKey&&!e.metaKey)enqueue(ctx,13,0,0,0,0,0,0,0,0,e.key);if(key||e.key.length===1)e.preventDefault()});canvas.addEventListener("keyup",e=>enqueue(ctx,12,0,0,0,0,keys[e.key]||keys[e.key.toLowerCase()]||0,0,modifiers(e)))}
function ensureTerminal(ctx){if(ctx.terminal)return ctx.terminal;const made=makeShell(ctx,"Console",720,420,true);ctx.terminal=made.terminal;return ctx.terminal}
function showError(ctx,error){const terminal=ensureTerminal(ctx),nl=String.fromCharCode(10);terminal.textContent+=nl+String(error)+nl}
function embeddedWasm(data){const html=decoder.decode(data),marker="const wasm64=",at=html.indexOf(marker);if(at<0)return null;const quote=html.indexOf(String.fromCharCode(34),at+marker.length),end=html.indexOf(String.fromCharCode(34),quote+1);if(quote<0||end<0)return null;const raw=atob(html.slice(quote+1,end));return Uint8Array.from(raw,c=>c.charCodeAt(0))}
function launchFile(ctx,path){const name=clean(path,ctx.cwd),data=files.get(name),wasm=data&&embeddedWasm(data);if(!wasm){showError(ctx,"Run failed: "+name+" is not a browser application");return}setTimeout(()=>startContext(wasm,false,parent(name)),0)}
async function startContext(binary,root,cwd="."){const ctx={root,cwd,input:[],pending:null,windows:new Map(),timers:new Map(),fds:new Map(),nextFd:8,memory:null,instance:null,terminal:null};if(root)rootContext=ctx;try{const result=await WebAssembly.instantiate(binary,importsFor(ctx));ctx.instance=result.instance;ctx.memory=result.instance.exports.memory;try{ctx.instance.exports._start()}catch(e){if(e!==EXIT)throw e}return ctx}catch(e){showError(ctx,e);return ctx}}
loadFiles();window.addEventListener("resize",()=>{if(rootContext)for(const view of rootContext.windows.values())if(view.root)resizeView(view)});
(async()=>{const raw=atob(wasm64),bytes=Uint8Array.from(raw,c=>c.charCodeAt(0));await startContext(bytes,true,"workspace")})();
</script></body></html>
`
