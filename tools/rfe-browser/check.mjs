// Offline, fixed-fixture browser gate. No downloads, shell or caller commands.
import fs from "node:fs/promises";
import {createHash} from "node:crypto";
import {createServer} from "node:http";
import {spawn} from "node:child_process";
import {tmpdir,homedir} from "node:os";
import path from "node:path";
import vm from "node:vm";
import {compilerIdentity,verifyCompilerIdentity} from "./profile.mjs";

const root=process.cwd();
const hash=b=>createHash("sha256").update(b).digest("hex");
const read=name=>fs.readFile(path.join(root,name));
const parse=async name=>JSON.parse(await read(name));
const pause=ms=>new Promise(resolve=>setTimeout(resolve,ms));
const candidates=["/usr/bin/chromium","/usr/bin/chromium-browser","/usr/bin/google-chrome",... ["chrome-linux/chrome","chrome-linux64/chrome"].map(p=>path.join(homedir(),".cache/ms-playwright/chromium-1194",p))];
async function browserPath(){for(const name of candidates){try{await fs.access(name,fs.constants.X_OK);return name}catch{}}return null}
async function sourceHash(){
 const names=["go.mod"];
 for(const e of await fs.readdir(root,{withFileTypes:true}))if(e.isFile()&&e.name.endsWith(".go")&&!e.name.endsWith("_test.go"))names.push(e.name);
 async function walk(dir){for(const e of await fs.readdir(path.join(root,dir),{withFileTypes:true})){
  const name=dir+"/"+e.name;
  if(e.isDirectory()){if(e.name!==".git"&&name!=="backend/tests")await walk(name)}
  else if(!e.isFile())throw Error("Non-regular source: "+name);
  else if(!name.endsWith("_test.go")&&[".go",".rtg",".rbe",".rfe",".html"].includes(path.extname(name)))names.push(name)
 }}
 for(const dir of ["backend","cmd","internal","std","emulators"])await walk(dir);
 names.sort();const h=createHash("sha256");for(const name of names){h.update(name);h.update("\0");h.update(await read(name));h.update("\0")}return h.digest("hex")
}
function checkScripts(html){
 const scripts=[...html.matchAll(/<script(?:\s[^>]*)?>([\s\S]*?)<\/script>/g)];
 if(scripts.length!==2)throw Error("Unexpected browser host script count");
 for(const [i,s] of scripts.entries())new vm.Script(s[1],{filename:"rfe-host-"+i+".js"});
 const wasm=html.match(/const wasm64="([A-Za-z0-9+/=]+)"/);
 if(!wasm)throw Error("Missing embedded WASM");
 const module=new WebAssembly.Module(Buffer.from(wasm[1],"base64"));
 const exports=WebAssembly.Module.exports(module).map(e=>e.name);
 if(!["memory","_start","renvo_browser_step"].every(name=>exports.includes(name)))throw Error("Missing WASM exports");
 return {imports:WebAssembly.Module.imports(module),exports}
}
async function openCDP(url){
 const u=new URL(url);if(u.hostname!=="127.0.0.1"||u.protocol!=="ws:")throw Error("Non-loopback debugger URL");
 const ws=new WebSocket(url),pending=new Map();let next=1;
 await new Promise((resolve,reject)=>{ws.addEventListener("open",resolve,{once:true});ws.addEventListener("error",reject,{once:true})});
 ws.addEventListener("message",({data})=>{const m=JSON.parse(data);if(m.id&&pending.has(m.id)){const p=pending.get(m.id);pending.delete(m.id);clearTimeout(p.timer);m.error?p.reject(Error(JSON.stringify(m.error))):p.resolve(m.result)}});
 ws.addEventListener("close",()=>{for(const p of pending.values()){clearTimeout(p.timer);p.reject(Error("Debugger disconnected"))}pending.clear()});
 return {ws,call(method,params={},sessionId){return new Promise((resolve,reject)=>{const id=next++,timer=setTimeout(()=>{pending.delete(id);reject(Error("Debugger command timed out: "+method))},10000);pending.set(id,{resolve,reject,timer});ws.send(JSON.stringify({id,method,params,...(sessionId?{sessionId}:{})}))})}}
}
async function run(arch,executable){
 const machine=arch==="aarch64"?"linux-arm64-machine":"linux-amd64-machine";
 const artifact="sandbox/rfe-browser/"+arch+".html",fixture="sandbox/rfe-linux/"+arch;
 const proof=await parse(artifact+".json"),html=await read(artifact),manifest=await parse(fixture+"/manifest.json"),inputs=await parse("tools/rfe-linux/inputs.json");
 if(proof.format!==1||proof.machine!==machine||proof.target!=="browser/wasm32"||proof.arena!==536870912||proof.html_sha256!==hash(html)||proof.sources_sha256!==await sourceHash()||proof.compiler_sha256!==hash(await read("sandbox/agent-bin/renvo-bootstrap")))throw Error("Stale or incompatible browser build; rebuild first");
 if(!verifyCompilerIdentity(proof,await compilerIdentity("sandbox/agent-bin/renvo-bootstrap")))throw Error("Browser build stamp does not prove the sibling backend; rebuild first");
 if(manifest.architecture!==arch||manifest.alpine!==inputs.version||!["iso_sha256","rootfs_sha256"].every(key=>manifest.inputs?.[key]===inputs.architectures[arch]?.[key]))throw Error("Cached fixture provenance mismatch");
 const kernel=await read(fixture+"/kernel"),initrd=await read(fixture+"/initramfs.cpio.gz");
 if(hash(kernel)!==manifest.kernel_sha256||hash(initrd)!==manifest.initramfs_sha256)throw Error("Cached boot asset checksum mismatch");
 const module=checkScripts(html.toString());
 const routes=new Map([["/",{data:html,type:"text/html"}],["/kernel",{data:kernel,type:"application/octet-stream"}],["/initramfs.cpio.gz",{data:initrd,type:"application/octet-stream"}]]);
 const server=createServer((req,res)=>{const p=new URL(req.url,"http://localhost").pathname,r=routes.get(p);if(req.method!=="GET"||!r){res.writeHead(404);res.end();return}res.writeHead(200,{"Content-Type":r.type,"Cache-Control":"no-store","Content-Security-Policy":"default-src 'none'; script-src 'unsafe-inline' 'wasm-unsafe-eval'; worker-src blob:; connect-src 'self'; style-src 'unsafe-inline'"});res.end(r.data)});
 let browser,cdp,profile,lastState,stderr="",result={architecture:arch,ok:false,kernel_sha256:hash(kernel),initramfs_sha256:hash(initrd),module};
 const out="sandbox/rfe-browser/"+arch+"-latest";
 try{
  await new Promise((resolve,reject)=>{server.once("error",reject);server.listen(0,"127.0.0.1",resolve)});
  profile=await fs.mkdtemp(path.join(tmpdir(),"renvo-rfe-browser-"));
  browser=spawn(executable,["--headless","--no-sandbox","--disable-gpu","--disable-background-networking","--disable-component-update","--no-first-run","--no-default-browser-check","--remote-debugging-address=127.0.0.1","--remote-debugging-port=0","--user-data-dir="+profile,"about:blank"],{stdio:["ignore","ignore","pipe"]});
  const debuggerURL=await new Promise((resolve,reject)=>{
   const timer=setTimeout(()=>reject(Error("Headless browser startup timed out")),15000);
   browser.once("error",e=>{clearTimeout(timer);reject(e)});browser.once("exit",code=>{clearTimeout(timer);reject(Error("Browser exited during startup: "+code))});
   browser.stderr.on("data",b=>{stderr=(stderr+b).slice(-32768);const match=stderr.match(/DevTools listening on (ws:\/\/127\.0\.0\.1:[^\s]+)/);if(match){clearTimeout(timer);resolve(match[1])}})
  });
  cdp=await openCDP(debuggerURL);
  const {targetId}=await cdp.call("Target.createTarget",{url:"about:blank"});
  const {sessionId}=await cdp.call("Target.attachToTarget",{targetId,flatten:true});
  const version=await cdp.call("Browser.getVersion");result.browser=version;
  const start=performance.now();
  await cdp.call("Page.navigate",{url:"http://127.0.0.1:"+server.address().port+"/?boot=1"},sessionId);
  while(performance.now()-start<300000){
   const evaluated=await cdp.call("Runtime.evaluate",{expression:"window.renvoBrowserResult || null",returnByValue:true},sessionId);
   const state=evaluated.result.value;if(state)lastState=state;
   if(state&&["halted","failed"].includes(state.state)){result={...result,...state};break}
   await pause(250)
  }
  result.duration_ms=performance.now()-start;
  if(!result.state){const e=await cdp.call("Runtime.evaluate",{expression:"window.renvoBrowserResult || null",returnByValue:true},sessionId);Object.assign(result,e.result.value||{});result.timed_out=true}
  const markers=["INIT","EXEC PASS","FILE PASS","PIPE PASS","PASS"].map(s=>"RENVO-RFE-LINUX: "+s);
  let at=0;result.markers_in_order=markers.every(marker=>{const i=(result.output||"").indexOf(marker,at);if(i<0)return false;at=i+marker.length;return true});
  result.ok=result.state==="halted"&&result.exitCode===0&&result.markers_in_order&&!(result.output||"").includes("RENVO-RFE-LINUX: FAIL")&&!result.timed_out;
  const shot=await cdp.call("Page.captureScreenshot",{format:"png"},sessionId);await fs.writeFile(path.join(root,out+".png"),Buffer.from(shot.data,"base64"));
  if(hash(await read(fixture+"/kernel"))!==result.kernel_sha256||hash(await read(fixture+"/initramfs.cpio.gz"))!==result.initramfs_sha256)throw Error("Boot fixtures changed during validation");
 }catch(e){if(lastState)Object.assign(result,lastState);result.error=String(e.stack||e);result.ok=false}
 finally{
  if(cdp)cdp.ws.close();if(browser){browser.kill("SIGTERM");await Promise.race([new Promise(resolve=>browser.once("exit",resolve)),pause(2000)]);if(browser.exitCode===null){browser.kill("SIGKILL");await new Promise(resolve=>browser.once("exit",resolve))}}
  await new Promise(resolve=>server.close(resolve));if(profile)await fs.rm(profile,{recursive:true,force:true});
  await fs.writeFile(path.join(root,out+".log"),result.output||"");await fs.writeFile(path.join(root,out+"-stderr.log"),stderr);await fs.writeFile(path.join(root,out+".json"),JSON.stringify(result,null,2)+"\n")
 }
 console.log(JSON.stringify({...result,output:undefined},null,2));if(result.output)console.log(result.output.slice(-12000));if(!result.ok)process.exitCode=1;
}
try{
 const mode=process.argv[2];if(process.argv.length!==3||!["--inspect","aarch64","x86_64"].includes(mode))throw Error("Expected --inspect, aarch64 or x86_64");
 const executable=await browserPath();
 if(mode==="--inspect"){
  const host=await read("internal/rfe/browser_host.html");for(const s of host.toString().matchAll(/<script(?:\s[^>]*)?>([\s\S]*?)<\/script>/g))new vm.Script(s[1]);
  console.log(JSON.stringify({node:process.version,websocket:typeof WebSocket==="function",chromium:executable,host_syntax:true},null,2))
 }
 if(!executable||typeof WebSocket!=="function")throw Error("Need installed Chromium and Node with WebSocket; no dependencies were installed");
 if(mode!=="--inspect")await run(mode,executable)
}catch(e){console.error(String(e.stack||e));process.exitCode=1}
