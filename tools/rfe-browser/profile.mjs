// Diagnostic browser profile of fixed cached fixtures. Does not modify served artifacts.
import fs from "node:fs/promises";
import {createHash} from "node:crypto";
import {createServer} from "node:http";
import {spawn} from "node:child_process";
import {gunzipSync} from "node:zlib";
import {tmpdir,homedir,cpus,platform,arch as hostArch,release} from "node:os";
import {pathToFileURL} from "node:url";
import path from "node:path";
import vm from "node:vm";

const root=process.cwd();
const hash=b=>createHash("sha256").update(b).digest("hex");
const read=name=>fs.readFile(path.join(root,name));
const parse=async name=>JSON.parse(await read(name));
const pause=ms=>new Promise(resolve=>setTimeout(resolve,ms));
const candidates=["/usr/bin/chromium","/usr/bin/chromium-browser","/usr/bin/google-chrome",... ["chrome-linux/chrome","chrome-linux64/chrome"].map(p=>path.join(homedir(),".cache/ms-playwright/chromium-1194",p))];
const jitCachePresets=Object.freeze(["default","balanced8","balanced32","kernel32","balanced64"]);
export function parseOptions(args){
 const [architecture,...options]=args;
 if(!["aarch64","x86_64"].includes(architecture))throw Error("Expected aarch64 or x86_64 [--artifact sandbox/...html] [--output sandbox/rfe-browser-profile/...]");
 const result={architecture,artifact:"sandbox/rfe-browser/"+architecture+".html",output:"sandbox/rfe-browser-profile/"+architecture},seen=new Set();
 for(let i=0;i<options.length;i++){
  const flag=options[i];
  if(seen.has(flag))throw Error("Duplicate profile option: "+flag);
  seen.add(flag);
  if(flag==="--translation-details"){result.translationDetails=true;continue}
  if(flag==="--timing-only"){result.timingOnly=true;continue}
  if(flag==="--initrd-uncompressed"){result.initrdUncompressed=true;continue}
  if(flag==="--progress"){result.progress=true;continue}
  if(flag==="--engine-profile"){if(architecture!=="aarch64")throw Error("Engine profiling currently requires aarch64");result.engineProfile=true;continue}
  const value=options[++i];
  if(flag==="--jit-cache"){if(!jitCachePresets.includes(value))throw Error("Invalid JIT cache preset: "+value);result.jitCache=value;continue}
  if(!["--artifact","--output","--compiler"].includes(flag)||!value||value.startsWith("--"))throw Error("Invalid profile option: "+flag);
  const relative=path.relative(root,path.resolve(root,value));
  const prefix=flag==="--output"?"sandbox/rfe-browser-profile/":"sandbox/";
  if(!relative.startsWith(prefix)||flag==="--artifact"&&!relative.endsWith(".html"))throw Error("Expected "+flag+" path under "+prefix);
  result[flag.slice(2)]=relative;
 }
 // A report prefix ending in .html would overwrite a colocated build stamp.
 if(result.output.endsWith(".html"))throw Error("Output must be a report prefix, without .html");
 return result;
}
export async function compilerIdentity(compiler){
 const frontend=await fs.realpath(path.resolve(root,compiler));
 const identity={frontend,frontend_sha256:hash(await fs.readFile(frontend)),backend:null,backend_sha256:null};
 const backend=path.join(path.dirname(frontend),"renvo-backend"+(frontend.endsWith(".exe")?".exe":""));
 try{identity.backend=await fs.realpath(backend);identity.backend_sha256=hash(await fs.readFile(identity.backend))}catch(error){if(error.code!=="ENOENT")throw error}
 return identity;
}
export function verifyCompilerIdentity(proof,identity){
 if(proof.compiler_sha256!==identity.frontend_sha256||proof.compiler_path&&proof.compiler_path!==identity.frontend)throw Error("Browser frontend identity mismatch");
 if(proof.compiler_backend_sha256||proof.compiler_backend_path){
  if(proof.compiler_backend_sha256!==identity.backend_sha256||proof.compiler_backend_path!==identity.backend)throw Error("Browser sibling backend identity mismatch");
  return true;
 }
 return false; // Historical stamps do not prove the invoked backend identity.
}
export function prepareInitrd(original,uncompressed=false){
 const start=performance.now(),data=uncompressed?gunzipSync(original,{maxOutputLength:268435456}):original;
 if(uncompressed&&!["070701","070702"].includes(data.subarray(0,6).toString()))throw Error("Derived initramfs is not a newc archive");
 return {data,metadata:{variant:uncompressed?"derived-uncompressed-newc":"pinned-gzip",original_sha256:hash(original),original_bytes:original.length,served_sha256:hash(data),served_bytes:data.length,preparation_ms:performance.now()-start,preparation_scope:"Host asset preparation before browser/kernel timing"}};
}
function signalBrowser(browser,signal){
 if(!browser?.pid)return;
 try{if(process.platform==="win32")browser.kill(signal);else process.kill(-browser.pid,signal)}catch(e){if(e.code!=="ESRCH")throw e}
}
function waitForBrowserExit(browser){
 if(browser.exitCode!==null||browser.signalCode!==null)return Promise.resolve();
 return new Promise(resolve=>{
  const finish=()=>{clearTimeout(timer);browser.off("exit",finish);resolve()};
  const timer=setTimeout(finish,2000);browser.once("exit",finish);
 });
}
export async function stopBrowser(browser){
 if(!browser?.pid)return;
 signalBrowser(browser,"SIGTERM");await waitForBrowserExit(browser);
 // Chrome descendants may outlive the browser parent; its detached process
 // group belongs solely to this run, so reap that group even after parent exit.
 signalBrowser(browser,"SIGKILL");await waitForBrowserExit(browser);
}
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
 await new Promise((resolve,reject)=>{
  const timer=setTimeout(()=>{ws.close();reject(Error("Debugger connection timed out"))},10000);
  ws.addEventListener("open",()=>{clearTimeout(timer);resolve()},{once:true});
  ws.addEventListener("error",e=>{clearTimeout(timer);reject(e)},{once:true});
 });
 ws.addEventListener("message",({data})=>{const m=JSON.parse(data);if(m.id&&pending.has(m.id)){const p=pending.get(m.id);pending.delete(m.id);clearTimeout(p.timer);m.error?p.reject(Error(JSON.stringify(m.error))):p.resolve(m.result)}});
 ws.addEventListener("close",()=>{for(const p of pending.values()){clearTimeout(p.timer);p.reject(Error("Debugger disconnected"))}pending.clear()});
 return {ws,call(method,params={},sessionId){return new Promise((resolve,reject)=>{
  const id=next++,timer=setTimeout(()=>{pending.delete(id);reject(Error("Debugger command timed out: "+method))},10000);
  pending.set(id,{resolve,reject,timer});
  try{ws.send(JSON.stringify({id,method,params,...(sessionId?{sessionId}:{})}))}catch(e){pending.delete(id);clearTimeout(timer);reject(e)}
 })}};
}
// Self-contained helpers are inserted only in the isolated diagnostic worker.
// The fingerprint selects a bucket; byte equality, never the fingerprint alone,
// establishes duplicate identity. Unretained modules are explicitly untracked.
export function createTranslationTracker({maxModules=32768,maxBytes=33554432,maxPCs=65536,maxAssociations=8}={}){
 const buckets=new Map(),modules=[],pcs=new Map();let retainedBytes=0,arenaSequence=0;
 const totals={modules:0,module_bytes:0,duplicate_modules:0,duplicate_bytes:0,untracked_modules:0,untracked_bytes:0,proof_installations:0,untracked_pc_installations:0};
 function fingerprint(code){let h=2166136261;for(let i=0;i<code.length;i++)h=Math.imul(h^code[i],16777619);return (h>>>0).toString(16).padStart(8,"0")+":"+code.length}
 return {
  newArena(){return ++arenaSequence},
  module(code){
   totals.modules++;totals.module_bytes+=code.length;
   const key=fingerprint(code),bucket=buckets.get(key)||[];
   for(const record of bucket){
    if(record.code.length===code.length&&record.code.every((byte,i)=>byte===code[i])){
     record.installations++;totals.duplicate_modules++;totals.duplicate_bytes+=code.length;
     return {record,bytes:code.length,duplicate:true};
    }
   }
   if(modules.length>=maxModules||retainedBytes+code.length>maxBytes){totals.untracked_modules++;totals.untracked_bytes+=code.length;return {record:null,bytes:code.length,duplicate:false}}
   const record={id:modules.length+1,symbol:"rfe_block_"+key.replace(":","_")+"_"+bucket.length,fingerprint:key,code:code.slice(),installations:1,pcs:new Set(),pc_overflow:0};
   retainedBytes+=code.length;modules.push(record);bucket.push(record);buckets.set(key,bucket);
   return {record,bytes:code.length,duplicate:false};
  },
  proof(arena,pc,observation){
   totals.proof_installations++;
   const address="0x"+pc.toString(16).padStart(16,"0");let item=pcs.get(address);
   if(!item){
    if(pcs.size>=maxPCs){totals.untracked_pc_installations++;return}
    item={pc:address,installations:0,module_bytes:0,duplicate_modules:0,arena_transitions:0,last_arena:null,module_ids:new Set(),module_id_overflow:0};pcs.set(address,item);
   }
   item.installations++;item.module_bytes+=observation?.bytes||0;if(observation?.duplicate)item.duplicate_modules++;
   if(item.last_arena!==arena){item.arena_transitions++;item.last_arena=arena}
   const module=observation?.record;
   if(module){
    if(!item.module_ids.has(module.id)){if(item.module_ids.size<maxAssociations)item.module_ids.add(module.id);else item.module_id_overflow++}
    if(!module.pcs.has(address)){if(module.pcs.size<maxAssociations)module.pcs.add(address);else module.pc_overflow++}
   }
  },
  snapshot(){
   const pcRecords=[...pcs.values()].map(({last_arena,module_ids,...item})=>({...item,module_ids:[...module_ids]}));
   return {scope:"Kernel execution; startup probes excluded",identity:"FNV-1a bucket plus exact byte equality",...totals,retained_modules:modules.length,retained_bytes:retainedBytes,tracked_pcs:pcs.size,limits:{modules:maxModules,bytes:maxBytes,pcs:maxPCs,associations:maxAssociations},duplicates_are_lower_bound:totals.untracked_modules>0,top_repeated_pcs:pcRecords.sort((a,b)=>b.installations-a.installations).slice(0,100),modules:modules.map(record=>({id:record.id,symbol:record.symbol,fingerprint:record.fingerprint,bytes:record.code.length,installations:record.installations,pcs:[...record.pcs],pc_overflow:record.pc_overflow}))};
  }
 };
}
export function nameGeneratedModule(code,symbol){
 const name=new TextEncoder().encode(symbol);
 const u32=value=>{const out=[];do{const byte=value&127;value>>>=7;out.push(byte|(value?128:0))}while(value);return out};
 // Generated leaves export run at function index 0. This custom section changes
 // only profiler symbols; code, data and the bytes charged to arenas stay intact.
 const names=[1,0,...u32(name.length),...name],payload=[4,110,97,109,101,1,...u32(names.length),...names];
 const section=new Uint8Array([0,...u32(payload.length),...payload]),named=new Uint8Array(code.length+section.length);
 named.set(code);named.set(section,code.length);return named;
}
// Record serial milestones at the worker's WASI write, before flush/message/poll
// latency. All counter snapshots include currently active calls and are relative
// to the first kernel step, excluding compiler probes and machine setup.
export function instrumentHost(html,{translationDetails=false,timingOnly=false,jitCache=null,engineProfile=false}={}){
 if(jitCache!==null&&!jitCachePresets.includes(jitCache))throw Error("Invalid JIT cache preset: "+jitCache);
 const replace=(a,b)=>{if(html.split(a).length!==2)throw Error("Missing or ambiguous hook: "+a);html=html.replace(a,()=>b)};
 const extraArgs=[...(jitCache===null?[]:["-jit-cache",jitCache]),...(engineProfile?["-jit-profile"]:[])];
 if(extraArgs.length)replace('args=data.shell?["rfe","-shell"]:["rfe"];','args=data.shell?["rfe","-shell"]:["rfe"];args.push(...'+JSON.stringify(extraArgs)+');');
 const command="printf 'RENVO-RFE-PROFILE:%s\\n' SHELL_READY\r";
 replace('let instance,memory,done=false,initializing=true,output="",input=[],inputBytes=0;',String.raw`
let instance,memory,done=false,initializing=true,output="",input=[],inputBytes=0;
const counterNames=["stepMs","steps","runMs","runs","installMs","installs","arenas","modules","moduleBytes","proofs","retainedModules"];
const timing={workerStart:performance.now(),timeOrigin:performance.timeOrigin,kernelStart:null,kernelReady:null,startMs:0,instantiateMs:0,...Object.fromEntries(counterNames.map(n=>[n,0]))};
let activeStep=null,activeRun=null,activeInstall=null,serialTail="",profileStopped=false,lastPublishedStep=0;
const milestones={};
function profileSnapshot(now=performance.now()){
 const t={...timing,now};
 if(activeStep!==null)t.stepMs+=now-activeStep;
 if(activeRun!==null)t.runMs+=now-activeRun;
 if(activeInstall!==null)t.installMs+=now-activeInstall.at-(t.runMs-activeInstall.runMs);
 return t;
}
function publishProfile(){lastPublishedStep=timing.steps;postMessage({type:"profile",timing:profileSnapshot(),milestones:{...milestones}})}
function markProfile(name,now=performance.now()){
 if(milestones[name])return;
 const t=profileSnapshot(now);
 milestones[name]={worker_ms:now,worker_since_start_ms:now-timing.workerStart,kernel_ms:timing.kernelStart===null?null:now-timing.kernelStart,timing:t};
 publishProfile();
}
function profileSerial(text){
 serialTail+=text;
 for(const marker of ["ready pc=","Booting Linux","Linux version","Run /init","RENVO-RFE-LINUX: INIT","RENVO-RFE-LINUX: EXEC PASS","RENVO-RFE-LINUX: FILE PASS","RENVO-RFE-LINUX: PIPE PASS","RENVO-RFE-LINUX: PASS"]){if(!milestones[marker]&&serialTail.includes(marker))markProfile(marker)}
 if(/(?:^|\n)RENVO-RFE-PROFILE:SHELL_READY\r?\n/.test(serialTail)){markProfile("shell_ready");profileStopped=true}
 serialTail=serialTail.slice(-2048);
}
`);
 replace('function flush(){if(output){postMessage({type:"output",text:output});','function flush(){if(!output&&timing.steps-lastPublishedStep>=100)publishProfile();if(output){publishProfile();postMessage({type:"output",text:output});');
 // A generic request can contain command 2 (dispatch). Subtract its nested run
 // interval so install and dispatch remain disjoint even on the slow path.
 if(!timingOnly)replace('function jitRequest(f,parts){',
  'function jitRequest(f,parts){const t=performance.now(),before=timing.runMs;activeInstall={at:t,runMs:before};try{return jitRequestImpl(f,parts)}finally{timing.installMs+=performance.now()-t-(timing.runMs-before);timing.installs++;activeInstall=null}}function jitRequestImpl(f,parts){');
 replace('f.jit={capacity:cap,bytes:0,next:16,blocks:new Map(),table:',
  'timing.arenas++;f.jit={capacity:cap,bytes:0,next:16,blocks:new Map(),table:');
 replace('f.jit=fresh;return fresh.owner;',
  'timing.arenas++;timing.retainedModules+=count;f.jit=fresh;return fresh.owner;');
 replace('result=arena.next;arena.next+=16;arena.bytes+=code.length;',
  'timing.modules++;timing.moduleBytes+=code.length;result=arena.next;arena.next+=16;arena.bytes+=code.length;');
 replace('arena.table.set(entry/8+1,proof.exports.prove);arena.bytes+=code.length;',
  'timing.proofs++;arena.table.set(entry/8+1,proof.exports.prove);arena.bytes+=code.length;');
 if(!timingOnly)replace('block.run(state,context)',
  'const t=performance.now();activeRun=t;try{block.run(state,context)}finally{timing.runMs+=performance.now()-t;timing.runs++;activeRun=null}');
 replace('output+=decode.decode(part,{stream:true});',
  'const text=decode.decode(part,{stream:true});output+=text;profileSerial(text);');
 replace('schedule.port1.onmessage=()=>{if(done)return;',
  'schedule.port1.onmessage=()=>{if(done||profileStopped)return;');
 replace('instance.exports.renvo_browser_step();flush();',
  'const t=performance.now();if(timing.kernelStart===null){timing.kernelStart=t;timing.baseline=Object.fromEntries(counterNames.map(n=>[n,timing[n]]));markProfile("kernel_start",t)}activeStep=t;try{instance.exports.renvo_browser_step()}finally{timing.stepMs+=performance.now()-t;timing.steps++;activeStep=null}flush();');
 replace('const built=await WebAssembly.instantiate(data.wasm,imports());',
  'const instantiateStart=performance.now();const built=await WebAssembly.instantiate(data.wasm,imports());timing.instantiateMs=performance.now()-instantiateStart;');
 replace('try{instance.exports._start()}catch(e){if(!e||!e.wasiExit||e.code!==0)throw e}',
  'const initial=performance.now();try{instance.exports._start()}catch(e){if(!e||!e.wasiExit||e.code!==0)throw e}timing.startMs=performance.now()-initial;timing.kernelReady=performance.now();markProfile("kernel_ready");');
 replace('if(data.type==="input"){const b=new Uint8Array(data.bytes);',
  'if(data.type==="input"){markProfile("shell_input_received");const b=new Uint8Array(data.bytes);');
 replace('if(data.type==="output"){window.renvoBrowserResult.output+=data.text;render()}',
  'if(data.type==="profile"){window.renvoBrowserResult.timing=data.timing;window.renvoBrowserResult.milestones=data.milestones;if(data.milestones["RENVO-RFE-LINUX: PASS"]&&!window.renvoBrowserResult.shellCommandSent){window.renvoBrowserResult.shellCommandSent=true;send('+JSON.stringify(command)+')}}else if(data.type==="output"){window.renvoBrowserResult.output+=data.text;render()}');
 // A stopped guest worker remains alive until CPU/details extraction finishes.
 // Its done flag already prevents further guest execution; normal cleanup owns
 // worker termination, including failed runs that would otherwise lose evidence.
 replace('if(data.state==="halted"||data.state==="failed")stop()',
  'if(data.state==="halted"||data.state==="failed")document.getElementById("stop").disabled=true');
 replace('if(new URLSearchParams(location.search).get("boot")==="1"){' ,
  'document.getElementById("shell").checked=true;if(new URLSearchParams(location.search).get("boot")==="1"){');
 if(translationDetails){
  replace('const milestones={};','const translationProfile=('+createTranslationTracker.toString()+')();const nameProfileModule='+nameGeneratedModule.toString()+';function publishTranslationProfile(){postMessage({type:"translation_profile",details:translationProfile.snapshot()})}const milestones={};');
  replace('timing.arenas++;timing.retainedModules+=count;f.jit=fresh;return fresh.owner;',
   'timing.arenas++;timing.retainedModules+=count;fresh.profileArena=translationProfile.newArena();f.jit=fresh;return fresh.owner;');
  replace('timing.arenas++;f.jit={capacity:cap,','timing.arenas++;f.jit={profileArena:translationProfile.newArena(),capacity:cap,');
  replace('const module=new WebAssembly.Module(code),imports=WebAssembly.Module.imports(module),exports=WebAssembly.Module.exports(module);',
   'const profileModule=timing.kernelStart===null?null:translationProfile.module(code);const module=new WebAssembly.Module(profileModule?nameProfileModule(code,profileModule.record?.symbol||"rfe_block_untracked"):code),imports=WebAssembly.Module.imports(module),exports=WebAssembly.Module.exports(module);');
  replace('arena.blocks.set(result,{run:block.exports.run,words,flags,contextSize});',
   'arena.blocks.set(result,{run:block.exports.run,words,flags,contextSize,profileModule});');
  replace('timing.proofs++;arena.table.set(entry/8+1,proof.exports.prove);',
   'timing.proofs++;if(timing.kernelStart!==null)translationProfile.proof(arena.profileArena,v.getBigUint64(8,true),block.profileModule);arena.table.set(entry/8+1,proof.exports.prove);');
  replace('function fail(error){done=true;flush();','function fail(error){done=true;flush();publishTranslationProfile();');
  replace('markProfile("shell_ready");profileStopped=true','markProfile("shell_ready");publishTranslationProfile();profileStopped=true');
  replace('if(data.type==="input"){markProfile',
   'if(data.type==="profile-details"){publishTranslationProfile();return}if(data.type==="input"){markProfile');
  replace('if(data.type==="profile"){window.renvoBrowserResult.timing=',
   'if(data.type==="translation_profile"){window.renvoBrowserResult.translation_profile=data.details}else if(data.type==="profile"){window.renvoBrowserResult.timing=');
 }
 return html;
}
export function summarizeCPU(profile,translationProfile){
 const nodes=new Map(profile.nodes.map(n=>[n.id,n])),exclusive=new Map(),inclusive=new Map();
 for(const n of profile.nodes)for(const child of n.children||[]){const c=nodes.get(child);if(c)c.parent=n.id}
 for(let i=0;i<(profile.samples||[]).length;i++){
  const us=profile.timeDeltas?.[i]||0;let n=nodes.get(profile.samples[i]);const seen=new Set();
  if(n)exclusive.set(n.id,(exclusive.get(n.id)||0)+us);
  while(n&&!seen.has(n.id)){inclusive.set(n.id,(inclusive.get(n.id)||0)+us);seen.add(n.id);n=nodes.get(n.parent)}
 }
 const top=map=>[...map].sort((a,b)=>b[1]-a[1]).slice(0,25).map(([id,us])=>({ms:us/1000,...nodes.get(id).callFrame}));
 const result={duration_ms:(profile.endTime-profile.startTime)/1000,samples:profile.samples?.length||0,exclusive:top(exclusive),inclusive:top(inclusive)};
 if(translationProfile){
  const modules=new Map(translationProfile.modules.map(module=>[module.symbol,module])),moduleTime=new Map(),pcTime=new Map();let ambiguous=0,untracked=0;
  for(const [id,us] of exclusive){
   const name=nodes.get(id).callFrame.functionName,module=modules.get(name);
   if(!module){if(name==="rfe_block_untracked")untracked+=us;continue}
   moduleTime.set(name,(moduleTime.get(name)||0)+us);
   if(module.pcs.length===1&&!module.pc_overflow)pcTime.set(module.pcs[0],(pcTime.get(module.pcs[0])||0)+us);else ambiguous+=us;
  }
  result.generated_wasm={ambiguous_pc_ms:ambiguous/1000,untracked_module_ms:untracked/1000,by_module:[...moduleTime].sort((a,b)=>b[1]-a[1]).slice(0,100).map(([symbol,us])=>({symbol,exclusive_ms:us/1000,module_id:modules.get(symbol).id,pcs:modules.get(symbol).pcs})),by_pc:[...pcTime].sort((a,b)=>b[1]-a[1]).slice(0,100).map(([pc,us])=>({pc,exclusive_ms:us/1000}))};
 }
 return result;
}
export function progressSnapshot(state,elapsed){
 const timing=state?.timing,baseline=timing?.baseline;
 const delta=name=>baseline?timing[name]-baseline[name]:null;
 let latest=null;
 for(const [name,marker] of Object.entries(state?.milestones||{})){if(!latest||marker.worker_ms>latest.worker_ms)latest={name,worker_ms:marker.worker_ms}}
 return {type:"progress",wall_ms:Math.round(elapsed),kernel_ms:timing?.kernelStart==null?null:Math.round(timing.now-timing.kernelStart),steps:delta("steps"),modules:delta("modules"),arenas:delta("arenas"),latest_milestone:latest?.name||null,state:state?.state||"loading"};
}
async function profileRun(arch,executable,{artifact,output:out,compiler="sandbox/agent-bin/renvo-bootstrap",translationDetails=false,timingOnly=false,initrdUncompressed=false,jitCache=null,progress=false,engineProfile=false}){
 const fixture="sandbox/rfe-linux/"+arch;
 const compilerInputs=await compilerIdentity(compiler);
 const original=await read(artifact),proof=await parse(artifact+".json"),manifest=await parse(fixture+"/manifest.json"),inputs=await parse("tools/rfe-linux/inputs.json");
 if(proof.format!==1||proof.machine!==(arch==="aarch64"?"linux-arm64-machine":"linux-amd64-machine")||proof.target!=="browser/wasm32"||proof.arena!==536870912||proof.html_sha256!==hash(original)||proof.sources_sha256!==await sourceHash()||proof.compiler_sha256!==compilerInputs.frontend_sha256)throw Error("Stale browser build");
 if(manifest.architecture!==arch||manifest.alpine!==inputs.version||!["iso_sha256","rootfs_sha256"].every(k=>manifest.inputs?.[k]===inputs.architectures[arch]?.[k]))throw Error("Fixture provenance mismatch");
 const kernel=await read(fixture+"/kernel"),originalInitrd=await read(fixture+"/initramfs.cpio.gz");
 if(hash(kernel)!==manifest.kernel_sha256||hash(originalInitrd)!==manifest.initramfs_sha256)throw Error("Fixture checksum mismatch");
 const backendVerified=verifyCompilerIdentity(proof,compilerInputs);
 const preparedInitrd=prepareInitrd(originalInitrd,initrdUncompressed),initrd=preparedInitrd.data;
 const html=Buffer.from(instrumentHost(original.toString(),{translationDetails,timingOnly,jitCache,engineProfile}));const hostModule=checkScripts(html.toString());
 const directWasmDispatch=hostModule.imports.some(i=>i.module==="renvo_rfe"&&i.name==="gates"&&i.kind==="table");
 const routes=new Map([["/",{data:html,type:"text/html"}],["/kernel",{data:kernel,type:"application/octet-stream"}],["/initramfs.cpio.gz",{data:initrd,type:"application/octet-stream"}]]);
 const server=createServer((req,res)=>{const r=routes.get(new URL(req.url,"http://localhost").pathname);if(req.method!=="GET"||!r){res.writeHead(404);res.end();return}res.writeHead(200,{"Content-Type":r.type,"Content-Security-Policy":"default-src 'none'; script-src 'unsafe-inline' 'wasm-unsafe-eval'; worker-src blob:; connect-src 'self'; style-src 'unsafe-inline'"});res.end(r.data)});
 await fs.mkdir(path.dirname(path.join(root,out)),{recursive:true});
 const realOutputDirectory=await fs.realpath(path.dirname(path.join(root,out))),allowedOutputDirectory=path.join(await fs.realpath(root),"sandbox/rfe-browser-profile");
 if(realOutputDirectory!==allowedOutputDirectory&&!realOutputDirectory.startsWith(allowedOutputDirectory+path.sep))throw Error("Output directory escapes profile sandbox");
 let browser,cdp,directory,workerSession,workerReady,profilerError,lastState,profilerStarted=false,stderr="";
 const result={architecture:arch,artifact,compiler,compiler_sha256:proof.compiler_sha256,compiler_inputs:compilerInputs,backend_identity_verified:backendVerified,sources_sha256:proof.sources_sha256,output_prefix:out,ok:false,source_html_sha256:hash(original),kernel_sha256:hash(kernel),initramfs_sha256:hash(originalInitrd),initramfs:preparedInitrd.metadata,markers:{},host:{platform:platform(),release:release(),architecture:hostArch(),cpu:cpus()[0]?.model,logical_cpus:cpus().length,node:process.version},measurement:{instrumented:true,jit_cache:jitCache||"default",jit_cache_override:jitCache!==null,engine_profile:engineProfile,cpu_sampling_interval_us:timingOnly?null:1000,per_call_timing:!timingOnly,direct_wasm_dispatch:directWasmDispatch,translation_details:translationDetails,scope:"First kernel step to completed serial shell command; includes unchanged init correctness checks",counters:"Kernel baseline subtracted; timings recorded in worker; request time excludes nested dispatch",cpu_scope:timingOnly?null:"Worker startup, kernel execution, init checks and shell round trip"}};
 let interrupted;
 const onInterrupt=signal=>{interrupted=signal;result.interrupted=signal;result.ok=false;try{signalBrowser(browser,"SIGTERM")}catch(e){result.cleanup_error=String(e)}if(cdp)cdp.ws.close()};
 const onSIGINT=()=>onInterrupt("SIGINT"),onSIGTERM=()=>onInterrupt("SIGTERM");
 process.on("SIGINT",onSIGINT);process.on("SIGTERM",onSIGTERM);
 const checkInterrupted=()=>{if(interrupted)throw Error("Profile interrupted by "+interrupted)};
 try{
  await new Promise((resolve,reject)=>{server.once("error",reject);server.listen(0,"127.0.0.1",resolve)});
  directory=await fs.mkdtemp(path.join(tmpdir(),"renvo-rfe-profile-"));
  checkInterrupted();
  browser=spawn(executable,["--headless","--no-sandbox","--disable-gpu","--disable-background-networking","--disable-component-update","--no-first-run","--no-default-browser-check","--remote-debugging-address=127.0.0.1","--remote-debugging-port=0","--user-data-dir="+directory,"about:blank"],{stdio:["ignore","ignore","pipe"],detached:process.platform!=="win32"});
  const debuggerURL=await new Promise((resolve,reject)=>{const timer=setTimeout(()=>reject(Error("Browser startup timed out")),15000);browser.once("error",e=>{clearTimeout(timer);reject(e)});browser.once("exit",code=>{clearTimeout(timer);reject(Error("Browser exited during startup: "+code))});browser.stderr.on("data",b=>{stderr=(stderr+b).slice(-32768);const m=stderr.match(/DevTools listening on (ws:\/\/127\.0\.0\.1:[^\s]+)/);if(m){clearTimeout(timer);resolve(m[1])}})});
  checkInterrupted();
  cdp=await openCDP(debuggerURL);checkInterrupted();
  const {targetId}=await cdp.call("Target.createTarget",{url:"about:blank"});
  const {sessionId}=await cdp.call("Target.attachToTarget",{targetId,flatten:true});
  result.browser=await cdp.call("Browser.getVersion");
  cdp.ws.addEventListener("message",({data})=>{
   const m=JSON.parse(data);if(m.method!=="Target.attachedToTarget"||m.sessionId!==sessionId||m.params.targetInfo.type!=="worker")return;
   const attached=m.params.sessionId;
   if(workerSession){profilerError="Unexpected second worker";cdp.call("Runtime.runIfWaitingForDebugger",{},attached).catch(()=>{});return}
   workerSession=attached;result.worker_attached_paused=m.params.waitingForDebugger;
   workerReady=(async()=>{
    try{
     if(!m.params.waitingForDebugger)throw Error("Worker was not paused before profiling");
     await cdp.call("Profiler.enable",{},attached);
     await cdp.call("Profiler.setSamplingInterval",{interval:1000},attached);
     await cdp.call("Profiler.start",{},attached);profilerStarted=true;
    }catch(e){profilerError=String(e)}
    finally{try{await cdp.call("Runtime.runIfWaitingForDebugger",{},attached)}catch(e){profilerError=String(e)}}
   })();
  });
  if(!timingOnly)await cdp.call("Target.setAutoAttach",{autoAttach:true,waitForDebuggerOnStart:true,flatten:true},sessionId);
  const start=performance.now();let lastProgress=start;
  await cdp.call("Page.navigate",{url:"http://127.0.0.1:"+server.address().port+"/?boot=1"},sessionId);
  while(performance.now()-start<300000){
   checkInterrupted();
   const e=await cdp.call("Runtime.evaluate",{expression:"window.renvoBrowserResult || null",returnByValue:true},sessionId),state=e.result.value;
   if(progress&&performance.now()-lastProgress>=10000){lastProgress=performance.now();console.error(JSON.stringify({architecture:arch,...progressSnapshot(state,lastProgress-start)}))}
   if(state){lastState=state;const text=state.output||"";
    if(state.milestones?.shell_ready&&/^RENVO-RFE-PROFILE:SHELL_READY\r?$/m.test(text)){result.shell_ready_page_ms=performance.now()-start;break}
    if(["halted","failed"].includes(state.state))break;
   }
   await pause(100);
  }
  result.duration_ms=performance.now()-start;
  if(translationDetails&&!lastState?.translation_profile){
   await cdp.call("Runtime.evaluate",{expression:'if(worker)worker.postMessage({type:"profile-details"})'},sessionId);
   for(let i=0;i<10;i++){
    const evaluated=await cdp.call("Runtime.evaluate",{expression:"window.renvoBrowserResult || null",returnByValue:true},sessionId);
    if(evaluated.result.value)lastState=evaluated.result.value;
    if(lastState?.translation_profile||["halted","failed"].includes(lastState?.state))break;
    await pause(100);
   }
  }
  Object.assign(result,lastState||{});
  result.ok=Boolean(result.shell_ready_page_ms);result.timed_out=!result.ok&&!["halted","failed"].includes(result.state);
  result.markers=result.milestones||{};
  const ready=result.markers.shell_ready;
  if(ready){
   result.kernel_to_shell_roundtrip_ms=ready.kernel_ms;
   result.shell_roundtrip_ms=ready.worker_ms-result.markers.shell_input_received?.worker_ms;
  }
  // Prefer the exact response instant, including the in-flight kernel step.
  const t=ready?.timing||result.timing;
  if(!timingOnly&&t?.baseline){
   const delta=name=>t[name]-t.baseline[name];
   result.host_breakdown={kernel_execution_ms:t.now-t.kernelStart,step_ms:delta("stepMs"),compiled_dispatch_ms:delta("runMs"),jit_request_ms:delta("installMs"),emulator_other_estimate_ms:delta("stepMs")-delta("runMs")-delta("installMs"),between_steps_ms:t.now-t.kernelStart-delta("stepMs"),worker_initialize_ms:t.instantiateMs+t.startMs,modules:delta("modules"),module_bytes:delta("moduleBytes"),proofs:delta("proofs"),retained_modules:delta("retainedModules"),arenas:delta("arenas"),runs:delta("runs"),steps:delta("steps")};
   if(directWasmDispatch){
    // Direct Wasm calls intentionally bypass the JS timing hooks. Do not
    // misreport those calls as zero-cost or classify them as emulator-only.
    result.host_breakdown.compiled_dispatch_ms=null;
    result.host_breakdown.emulator_other_estimate_ms=null;
    result.host_breakdown.runs=null;
    result.host_breakdown.unpartitioned_step_ms=delta("stepMs")-delta("installMs");
    result.host_breakdown.attribution_note="Direct Wasm dispatch and emulator work share the unpartitioned step interval; CPU samples cover both. JS dispatch counters are not exhaustive.";
   }
  }
  if(JSON.stringify(await compilerIdentity(compiler))!==JSON.stringify(compilerInputs))throw Error("Compiler inputs changed during profiling");
  if(hash(await read(artifact))!==result.source_html_sha256||hash(await read(fixture+"/kernel"))!==result.kernel_sha256||hash(await read(fixture+"/initramfs.cpio.gz"))!==result.initramfs_sha256)throw Error("Inputs changed during profiling");
 }catch(e){if(lastState)Object.assign(result,lastState);result.error=String(e.stack||e);result.ok=false}
 finally{
  try{
  if(workerReady)await workerReady;
  if(profilerStarted){
   try{const {profile}=await cdp.call("Profiler.stop",{},workerSession);await fs.writeFile(path.join(root,out+"-cpu.json"),JSON.stringify(profile));result.cpu=summarizeCPU(profile,result.translation_profile)}catch(e){profilerError=String(e)}
  }else if(!timingOnly&&!profilerError)profilerError="Worker CPU profiler did not start";
  result.profiler_error=profilerError||null;if(profilerError)result.ok=false;
  if(cdp)cdp.ws.close();
  try{await stopBrowser(browser)}catch(e){result.cleanup_error=String(e);result.ok=false}
  if(server.listening){server.closeAllConnections();await new Promise(r=>server.close(r))}
  if(directory)try{await fs.rm(directory,{recursive:true,force:true})}catch(e){result.cleanup_error=String(e);result.ok=false}
  await fs.writeFile(path.join(root,out+".log"),result.output||"");
  await fs.writeFile(path.join(root,out+"-stderr.log"),stderr);
  const output=result.output;delete result.output;await fs.writeFile(path.join(root,out+".json"),JSON.stringify(result,null,2)+"\n");
  console.log(JSON.stringify(result,null,2));if(output)console.log(output.slice(-1500));
  }finally{process.off("SIGINT",onSIGINT);process.off("SIGTERM",onSIGTERM)}
 }
 if(interrupted)process.exitCode=interrupted==="SIGINT"?130:143;else if(!result.ok)process.exitCode=1;
}
if(process.argv[1]&&import.meta.url===pathToFileURL(path.resolve(process.argv[1])).href){
 try{const options=parseOptions(process.argv.slice(2)),executable=await browserPath();if(!executable||typeof WebSocket!=="function")throw Error("Installed Node/Chromium required; nothing was installed");await profileRun(options.architecture,executable,options)}catch(e){console.error(String(e.stack||e));process.exitCode=1}
}
