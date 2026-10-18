import assert from "node:assert/strict";
import fs from "node:fs/promises";
import test from "node:test";
import vm from "node:vm";
import {spawn} from "node:child_process";
import {gzipSync} from "node:zlib";
import {verifyCompilerIdentity,instrumentHost,parseOptions,stopBrowser,createTranslationTracker,nameGeneratedModule,summarizeCPU,prepareInitrd,progressSnapshot} from "./profile.mjs";

const template=await fs.readFile(new URL("../../internal/rfe/browser_host.html",import.meta.url),"utf8");
const html=instrumentHost(template.replace("@@WASM@@","AGFzbQEAAAA="));
const scripts=[...html.matchAll(/<script(?:\s[^>]*)?>([\s\S]*?)<\/script>/g)].map(match=>match[1]);

// Run the real instrumented WASI host against a deterministic simulated guest.
// Only the WASM exports and worker transport are replaced; serial writes, JIT
// requests, scheduling, boot initialization and timing hooks are the real code.
function workerHarness(options={}){
 let now=100,wasi;
 const messages=[],callbacks={},memory=new WebAssembly.Memory({initial:1});
 const context=vm.createContext({
  onmessage:null,TextDecoder,TextEncoder,
  performance:{now:()=>now,timeOrigin:1000},
  postMessage:message=>messages.push(structuredClone(message)),
  MessageChannel:class{constructor(){this.port1={};this.port2={postMessage(){}}}},
  WebAssembly:{
   Module:WebAssembly.Module,Instance:WebAssembly.Instance,Global:WebAssembly.Global,
   Table:class{constructor(){now+=5}set(){}},
   async instantiate(_bytes,imports){
    now+=7;wasi=imports.wasi_snapshot_preview1;
    return {instance:{exports:{memory,_start:()=>callbacks.start?.(),renvo_browser_step:()=>callbacks.step?.()}}};
   }
  }
 });
 const workerSource=[...instrumentHost(template,options).matchAll(/<script(?:\s[^>]*)?>([\s\S]*?)<\/script>/g)][0][1];
 vm.runInContext(workerSource,context);
 const execute=code=>vm.runInContext(code,context);
 return {
  messages,callbacks,execute,advance:ms=>{now+=ms},
  async boot(){await context.onmessage({data:{type:"boot",wasm:new ArrayBuffer(0),kernel:new ArrayBuffer(0),initrd:new ArrayBuffer(0),shell:true}})},
  step(){execute("schedule.port1.onmessage()")},
  write(text){
   const bytes=new TextEncoder().encode(text),view=new DataView(memory.buffer);
   assert.ok(bytes.length<=2048,"test write fits simulated memory");
   new Uint8Array(memory.buffer,1024,bytes.length).set(bytes);
   view.setUint32(0,1024,true);view.setUint32(4,bytes.length,true);
   assert.equal(wasi.fd_write(1,0,1,8),0);
  },
  newArena(run){
   context.guestRun=run;
   execute(`{
    if(!globalThis.service){const replyBytes=new Uint8Array(4);globalThis.service={replyBytes,replyView:new DataView(replyBytes.buffer)}}
    const request=new Uint8Array(8),requestView=new DataView(request.buffer);
    requestView.setUint32(4,4096,true);
    jitRequest(service,[request]);
    service.jit.blocks.set(16,{run:guestRun,words:1,flags:0,contextSize:64});
   }`);
  },
  install(code){
   const request=new Uint8Array(code.length+16),view=new DataView(request.buffer);
   view.setUint32(0,1,true);view.setUint32(4,1,true);view.setUint32(12,64,true);request.set(code,16);
   context.installRequest=request;execute("jitRequest(service,[installRequest])");return execute("service.replyView.getInt32(0,true)");
  },
  dispatch(){
   // Use the generic command-2 path, which nests compiled execution inside a
   // timed JIT request and previously counted that execution twice.
   execute(`{
    const request=new Uint8Array(16),requestView=new DataView(request.buffer);
    requestView.setUint32(0,2,true);requestView.setUint32(4,16,true);
    requestView.setUint32(8,64,true);jitRequest(service,[request]);
   }`);
  },
  marker(name){return messages.filter(m=>m.type==="profile").at(-1)?.milestones[name]}
 };
}

test("both instrumented host scripts remain valid JavaScript",()=>{
 assert.equal(scripts.length,2);
 for(const [index,source] of scripts.entries())new vm.Script(source,{filename:"profile-host-"+index+".js"});
});

test("kernel baseline excludes probes and snapshots account for active nested execution once",async()=>{
 const h=workerHarness();
 h.callbacks.start=()=>{h.newArena(()=>h.advance(11));h.dispatch()};
 await h.boot();
 // Waiting between machine setup and the first kernel step is not boot time.
 h.advance(23);
 h.callbacks.step=()=>{
  h.advance(3);
  h.newArena(()=>{h.advance(7);h.write("RENVO-RFE-LINUX: INIT\n");h.advance(4)});
  h.dispatch();h.advance(2);
 };
 h.step();
 const start=h.marker("kernel_start"),marker=h.marker("RENVO-RFE-LINUX: INIT");
 assert.equal(start.worker_ms,151,"one-time root-table construction is before the kernel baseline");
 assert.equal(start.timing.baseline.runMs,11);
 assert.equal(start.timing.baseline.installMs,5);
 assert.equal(start.timing.baseline.runs,1);
 assert.equal(marker.kernel_ms,15);
 const delta=name=>marker.timing[name]-marker.timing.baseline[name];
 assert.equal(delta("stepMs"),15,"includes the unfinished step at the serial write");
 assert.equal(delta("runMs"),7,"excludes startup and includes the unfinished compiled call");
 assert.equal(delta("installMs"),5,"nested dispatch is not also installation time");
 assert.equal(delta("stepMs")-delta("runMs")-delta("installMs"),3,"only emulator overhead remains");
 const completed=h.messages.filter(m=>m.type==="profile").at(-1).timing;
 assert.equal(completed.stepMs-completed.baseline.stepMs,21);
 assert.equal(completed.runMs-completed.baseline.runMs,11);
 assert.equal(completed.installMs-completed.baseline.installMs,5);
});

test("shell readiness requires a complete response line, including across serial writes",async()=>{
 const h=workerHarness();await h.boot();h.step();
 h.write("/ # printf 'RENVO-RFE-PROFILE:%s\\n' SHELL_READY\r\n");
 h.write("prefix RENVO-RFE-PROFILE:SHELL_READY\r\n");
 h.write("RENVO-RFE-PROFILE:SHELL_READY suffix\r\n");
 assert.equal(h.marker("shell_ready"),undefined,"echoed or embedded markers cannot prove a shell round trip");
 h.advance(9);h.write("RENVO-RFE-PROFILE:SHELL_");h.advance(4);h.write("READY\r");
 assert.equal(h.marker("shell_ready"),undefined,"the response must end with a newline");
 h.advance(3);h.write("\n");
 const marker=h.marker("shell_ready");
 assert.equal(marker.kernel_ms,16,"records the write completing the line, without a polling delay");
 h.advance(100);h.execute("flush()");
 assert.equal(h.marker("shell_ready").worker_ms,marker.worker_ms,"later flushes preserve the exact milestone");
 let steps=0;h.callbacks.step=()=>steps++;h.step();
 assert.equal(steps,0,"profiling stops guest work after the proven response");
});

test("the page sends one correctly escaped command immediately after the worker PASS marker",async()=>{
 const sent=[],elements=new Map(),window={};let worker;
 for(const id of ["terminal","status","stop","boot","shell"]){elements.set(id,{disabled:false,checked:false,textContent:"",focus(){}})}
 elements.set("worker-code",{textContent:scripts[0]});
 const context=vm.createContext({
  window,document:{getElementById:id=>elements.get(id)},TextEncoder,Uint8Array,atob,
  URL:{createObjectURL:()=>"blob:test",revokeObjectURL(){}},URLSearchParams,Blob,
  location:{search:""},requestAnimationFrame(){},
  Worker:class{constructor(){worker=this}postMessage(message){sent.push(message)}}
 });
 vm.runInContext(scripts[1],context);
 await vm.runInContext("boot(new ArrayBuffer(0),new ArrayBuffer(0))",context);
 const milestone={worker_ms:123};
 worker.onmessage({data:{type:"profile",timing:{},milestones:{"RENVO-RFE-LINUX: PASS":milestone}}});
 worker.onmessage({data:{type:"profile",timing:{},milestones:{"RENVO-RFE-LINUX: PASS":milestone}}});
 const inputs=sent.filter(message=>message.type==="input");
 assert.equal(inputs.length,1);
 assert.equal(new TextDecoder().decode(inputs[0].bytes),"printf 'RENVO-RFE-PROFILE:%s\\n' SHELL_READY\r");
 assert.equal(window.renvoBrowserResult.shellCommandSent,true);
 assert.equal(sent[0].shell,true);
 worker.onmessage({data:{type:"state",state:"failed",error:"controlled guest failure"}});
 assert.equal(vm.runInContext("worker!==null",context),true,"stopped worker remains available for CPU collection");
 assert.equal(window.renvoBrowserResult.state,"failed");
 assert.equal(elements.get("stop").disabled,true);
});


test("quiet kernels publish bounded periodic timing without waiting for serial output",async()=>{
 const h=workerHarness();await h.boot();h.callbacks.step=()=>h.advance(2);h.step();
 const initial=h.messages.length;
 for(let i=1;i<100;i++)h.step();
 const periodic=h.messages.slice(initial).filter(message=>message.type==="profile");
 assert.equal(periodic.length,1);
 assert.equal(periodic[0].timing.steps,100);
 assert.equal(periodic[0].timing.stepMs,200);
 h.execute("flush();flush()");
 assert.equal(h.messages.length,initial+1,"repeated flushes do not republish the same step count");
});

test("candidate paths preserve defaults and cannot redirect reports into deployed artifacts",()=>{
 assert.deepEqual(parseOptions(["aarch64"]),{
  architecture:"aarch64",artifact:"sandbox/rfe-browser/aarch64.html",output:"sandbox/rfe-browser-profile/aarch64"
 });
 const candidate=parseOptions(["x86_64","--artifact","sandbox/rfe-browser-profile/candidate/x86_64.html","--output","sandbox/rfe-browser-profile/candidate/x86_64"]);
 assert.equal(candidate.artifact,"sandbox/rfe-browser-profile/candidate/x86_64.html");
 assert.equal(candidate.output,"sandbox/rfe-browser-profile/candidate/x86_64");
 assert.equal(parseOptions(["aarch64","--compiler","sandbox/range-bin/renvo-bootstrap"]).compiler,"sandbox/range-bin/renvo-bootstrap");
 for(const options of [
  ["--output","sandbox/rfe-browser/aarch64-latest"],
  ["--output","sandbox/rfe-browser-profile/../rfe-browser/aarch64"],
  ["--output","sandbox/rfe-browser-profile/candidate/aarch64.html"],
  ["--artifact","https://example.com/aarch64.html"],
  ["--artifact","sandbox/candidate.wasm"],
  ["--compiler","/usr/bin/other"],
  ["--compiler"],
  ["--artifact"],
  ["--output","sandbox/rfe-browser-profile/a","--output","sandbox/rfe-browser-profile/b"],
  ["--timeout","600000"]
 ])assert.throws(()=>parseOptions(["aarch64",...options]));
});

test("cleanup kills owned descendants after their browser parent exits",{skip:process.platform!=="linux",timeout:5000},async()=>{
 // A small Node process substitutes for Chrome. Its child ignores SIGTERM,
 // proving cleanup reaches the owned group after the parent exits normally.
 const code=`
  const {spawn}=require("node:child_process");
  const child=spawn(process.execPath,["-e",'process.on("SIGTERM",()=>{});console.log("ready");setInterval(()=>{},1000)'],{stdio:["ignore","pipe","ignore"]});
  child.stdout.once("data",()=>console.log(child.pid));
  process.on("SIGTERM",()=>process.exit(0));
  setInterval(()=>{},1000);
 `;
 const browser=spawn(process.execPath,["-e",code],{detached:true,stdio:["ignore","pipe","ignore"]});
 try{
  const descendant=await new Promise((resolve,reject)=>{
   const timer=setTimeout(()=>reject(Error("Process fixture startup timed out")),2000);
   browser.once("error",e=>{clearTimeout(timer);reject(e)});
   browser.stdout.once("data",bytes=>{clearTimeout(timer);resolve(Number(bytes.toString().trim()))});
  });
  assert.ok(descendant>0);
  await stopBrowser(browser);
  assert.notEqual(browser.exitCode,null,"browser parent exited");
  let state;
  for(let attempt=0;attempt<50;attempt++){
   try{state=(await fs.readFile("/proc/"+descendant+"/stat","utf8")).split(") ")[1][0]}catch(e){if(e.code!=="ENOENT")throw e;state="absent"}
   if(state==="absent"||state==="Z")break;
   await new Promise(resolve=>setTimeout(resolve,10));
  }
  assert.ok(state==="absent"||state==="Z","no live descendant remains: "+state);
 }finally{await stopBrowser(browser)}
});


const leafModule=new Uint8Array([
 0,97,115,109,1,0,0,0,
 1,6,1,96,2,127,127,0,
 2,15,1,3,101,110,118,6,109,101,109,111,114,121,2,0,1,
 3,2,1,0,
 7,7,1,3,114,117,110,0,0,
 10,4,1,2,0,11
]);

test("optional diagnostics retain valid host scripts and preserve timer-only boot measurements",async()=>{
 for(const options of [{translationDetails:true},{timingOnly:true},{translationDetails:true,timingOnly:true}]){
  for(const match of instrumentHost(template,options).matchAll(/<script(?:\s[^>]*)?>([\s\S]*?)<\/script>/g))new vm.Script(match[1]);
 }
 const h=workerHarness({timingOnly:true});
 h.callbacks.start=()=>{h.newArena(()=>h.advance(11));h.dispatch()};await h.boot();
 h.callbacks.step=()=>{h.dispatch();h.write("RENVO-RFE-LINUX: INIT\n")};h.step();
 const marker=h.marker("RENVO-RFE-LINUX: INIT");
 assert.equal(marker.kernel_ms,11);
 assert.equal(marker.timing.runMs,0,"per-call timing is disabled");
 assert.equal(marker.timing.installMs,0,"request timing is disabled");
 assert.equal(parseOptions(["aarch64","--timing-only","--translation-details","--initrd-uncompressed"]).initrdUncompressed,true);
});

test("translation identities survive arena recycling and retained bytes are bounded",()=>{
 const tracker=createTranslationTracker({maxModules:2,maxBytes:8,maxPCs:2}),first=new Uint8Array([1,2,3,4]);
 const a=tracker.module(first);tracker.proof(tracker.newArena(),0xffff000000001234n,a);
 first[0]=99; // Retained identity owns its bytes, even when the request is reused.
 const duplicate=tracker.module(new Uint8Array([1,2,3,4]));tracker.proof(tracker.newArena(),0xffff000000001234n,duplicate);
 assert.equal(duplicate.record.id,a.record.id);
 const b=tracker.module(first);tracker.proof(tracker.newArena(),0x1234n,b);
 const dropped=tracker.module(new Uint8Array([5,6,7,8]));tracker.proof(tracker.newArena(),0x5678n,dropped);
 assert.equal(dropped.record,null);
 const report=tracker.snapshot();
 assert.equal(report.retained_bytes,8);
 assert.equal(report.duplicate_modules,1);
 assert.equal(report.duplicate_bytes,4);
 assert.equal(report.untracked_modules,1);
 assert.equal(report.untracked_pc_installations,1);
 assert.equal(report.duplicates_are_lower_bound,true);
 assert.equal(report.top_repeated_pcs[0].pc,"0xffff000000001234");
 assert.equal(report.top_repeated_pcs[0].installations,2);
 assert.equal(report.top_repeated_pcs[0].arena_transitions,2);
 const other=createTranslationTracker();other.module(new Uint8Array([7]));
 assert.equal(other.module(new Uint8Array([1,2,3,4])).record.symbol,a.record.symbol,"symbols depend on bytes, not encounter order");
});

test("WASM names preserve module exports, execution and original arena byte accounting",async()=>{
 const tracker=createTranslationTracker(),observation=tracker.module(leafModule),named=nameGeneratedModule(leafModule,observation.record.symbol);
 const original=new WebAssembly.Module(leafModule),instrumented=new WebAssembly.Module(named);
 assert.deepEqual(WebAssembly.Module.imports(instrumented),WebAssembly.Module.imports(original));
 assert.deepEqual(WebAssembly.Module.exports(instrumented),WebAssembly.Module.exports(original));
 assert.deepEqual(named.subarray(0,leafModule.length),leafModule);
 assert.equal(WebAssembly.Module.customSections(instrumented,"name").length,1);
 const instance=new WebAssembly.Instance(instrumented,{env:{memory:new WebAssembly.Memory({initial:1})}});
 assert.equal(instance.exports.run(0,0),undefined);
 const h=workerHarness({translationDetails:true});await h.boot();h.step();h.newArena(()=>{});const entry=h.install(leafModule);
 assert.equal(entry,32,"installation skips an already-occupied retained slot");
 assert.equal(h.execute("service.jit.bytes"),leafModule.length,"arena charges original module size");
 assert.equal(h.execute("service.jit.blocks.get("+entry+").profileModule.record.symbol"),observation.record.symbol);
});

test("CPU attribution reports shared module PCs as ambiguous",()=>{
 const tracker=createTranslationTracker(),a=tracker.module(new Uint8Array([1])),b=tracker.module(new Uint8Array([2]));
 tracker.proof(1,0x1000n,a);tracker.proof(1,0x2000n,b);tracker.proof(1,0x3000n,b);
 const frame=functionName=>({functionName,scriptId:"1",url:"wasm://test",lineNumber:0,columnNumber:0});
 const summary=summarizeCPU({startTime:0,endTime:3000,nodes:[{id:1,callFrame:frame(a.record.symbol)},{id:2,callFrame:frame(b.record.symbol)}],samples:[1,2],timeDeltas:[1000,2000]},tracker.snapshot());
 assert.deepEqual(summary.generated_wasm.by_pc,[{pc:"0x0000000000001000",exclusive_ms:1}]);
 assert.equal(summary.generated_wasm.ambiguous_pc_ms,2);
});

test("uncompressed initrd is derived in memory and preserves provenance",()=>{
 const archive=Buffer.from("070701"+"0".repeat(104)),compressed=gzipSync(archive),copy=Buffer.from(compressed);
 const original=prepareInitrd(compressed),derived=prepareInitrd(compressed,true);
 assert.equal(original.data,compressed);
 assert.deepEqual(derived.data,archive);
 assert.deepEqual(compressed,copy);
 assert.equal(derived.metadata.variant,"derived-uncompressed-newc");
 assert.equal(derived.metadata.original_sha256,original.metadata.served_sha256);
 assert.equal(derived.metadata.original_bytes,compressed.length);
 assert.equal(derived.metadata.served_bytes,archive.length);
 assert.ok(derived.metadata.preparation_ms>=0);
 assert.throws(()=>prepareInitrd(gzipSync(Buffer.from("not an archive")),true),/newc/);
});


test("JIT cache overrides accept only existing presets and preserve default worker args",async()=>{
 const defaultWorker=workerHarness();await defaultWorker.boot();
 assert.deepEqual(JSON.parse(defaultWorker.execute("JSON.stringify(args)")),["rfe","-shell"]);
 for(const preset of ["default","balanced8","balanced32","kernel32","balanced64"]){
  const options=parseOptions(["aarch64","--jit-cache",preset,"--progress"]);
  assert.equal(options.jitCache,preset);assert.equal(options.progress,true);
  const h=workerHarness(options);await h.boot();
  assert.deepEqual(JSON.parse(h.execute("JSON.stringify(args)")),["rfe","-shell","-jit-cache",preset]);
 }
 for(const options of [["--jit-cache"],["--jit-cache",""],["--jit-cache","balanced128"],["--jit-cache","default","--jit-cache","balanced8"]])assert.throws(()=>parseOptions(["aarch64",...options]));
 assert.throws(()=>instrumentHost(template,{jitCache:"unknown"}));
});

test("progress snapshots report kernel counters and the latest exact milestone",()=>{
 assert.equal(progressSnapshot(null,10001).kernel_ms,null);
 const snapshot=progressSnapshot({state:"running",timing:{kernelStart:200,now:1520,steps:120,modules:30,arenas:4,baseline:{steps:0,modules:10,arenas:3}},milestones:{early:{worker_ms:210},late:{worker_ms:1500}}},10001);
 assert.deepEqual(snapshot,{type:"progress",wall_ms:10001,kernel_ms:1320,steps:120,modules:20,arenas:1,latest_milestone:"late",state:"running"});
});


test("guest failures publish partial translation evidence before the failed state",async()=>{
 const h=workerHarness({translationDetails:true});await h.boot();h.step();
 h.newArena(()=>{});const entry=h.install(leafModule);
 assert.equal(entry,32,"installation skips an already-occupied retained slot");
 h.callbacks.step=()=>{throw Error("controlled guest failure")};h.step();
 const detailIndex=h.messages.findIndex(message=>message.type==="translation_profile"),failureIndex=h.messages.findIndex(message=>message.type==="state"&&message.state==="failed");
 assert.ok(detailIndex>=0&&failureIndex>detailIndex);
 assert.equal(h.messages[detailIndex].details.modules[0].bytes,leafModule.length);
 assert.equal(h.messages[detailIndex].details.modules[0].installations,1);
 let steps=0;h.callbacks.step=()=>steps++;h.step();assert.equal(steps,0,"failed guest execution remains stopped");
});


test("engine statistics are opt-in and forwarded independently of cache choice",async()=>{
 const options=parseOptions(["aarch64","--engine-profile","--jit-cache","balanced64"]);
 assert.equal(options.engineProfile,true);
 const h=workerHarness(options);await h.boot();
 assert.deepEqual(JSON.parse(h.execute("JSON.stringify(args)")),["rfe","-shell","-jit-cache","balanced64","-jit-profile"]);
 const only=workerHarness({engineProfile:true});await only.boot();
 assert.deepEqual(JSON.parse(only.execute("JSON.stringify(args)")),["rfe","-shell","-jit-profile"]);
 assert.throws(()=>parseOptions(["x86_64","--engine-profile"]),/aarch64/);
 assert.throws(()=>parseOptions(["aarch64","--engine-profile","--engine-profile"]),/Duplicate/);
});


test("compiler proof distinguishes changed sibling backends with identical frontends",()=>{
 const identity={frontend:"/repo/renvo-bootstrap",frontend_sha256:"same-frontend",backend:"/repo/renvo-backend",backend_sha256:"fixed-backend"};
 const proof={compiler_path:identity.frontend,compiler_sha256:identity.frontend_sha256,compiler_backend_path:identity.backend,compiler_backend_sha256:identity.backend_sha256};
 assert.equal(verifyCompilerIdentity(proof,identity),true);
 assert.throws(()=>verifyCompilerIdentity({...proof,compiler_backend_sha256:"old-backend"},identity),/backend identity mismatch/);
 assert.throws(()=>verifyCompilerIdentity({...proof,compiler_backend_path:"/other/renvo-backend"},identity),/backend identity mismatch/);
 assert.equal(verifyCompilerIdentity({compiler_sha256:identity.frontend_sha256},identity),false);
});
