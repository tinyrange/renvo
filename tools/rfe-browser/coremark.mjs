// Separate, offline browser CoreMark diagnostic. No boot-gate or live-demo edits.
// Editable repository implementation; all entry points have fixed inputs/paths.
import fs from 'node:fs/promises';
import {createHash} from 'node:crypto';
import {createServer} from 'node:http';
import {spawn} from 'node:child_process';
import {gunzipSync} from 'node:zlib';
import {tmpdir, homedir, cpus, platform, arch as hostArch} from 'node:os';
import path from 'node:path';
import vm from 'node:vm';

const ROOT=process.cwd(), BASE='sandbox/rfe-browser-coremark/v2';
const COMPILER='sandbox/rfe-resume-bin/renvo-bootstrap';
const ITERATIONS=20000, DEADLINE_MS=900000, BOOT_MS=300000, MAX_SERIAL=1048576;
const ORIGINAL_LIMIT='const browserInstructionLimit uint64 = 2000000000';
const PERFORMANCE_LIMIT='const browserInstructionLimit uint64 = 1000000000000';
const EXPECTED={PERF:['e9f5','e714','1fd7','8e3a','382f'],VALID:['18f2','e3c1','0747','8d84','d304']};
const hash=b=>createHash('sha256').update(b).digest('hex');
const read=name=>fs.readFile(path.join(ROOT,name));
const parse=async name=>JSON.parse(await read(name));
const pause=ms=>new Promise(resolve=>setTimeout(resolve,ms));
export function validateArmTrial(trial){
 if(typeof trial!=='string'||trial.length>100||! /^[A-Za-z0-9][A-Za-z0-9_-]*$/.test(trial))throw Error('Invalid ARM candidate label');
 return trial;
}
function architecture(arch,trial=null){
 if(!['aarch64','x86_64'].includes(arch))throw Error('Expected aarch64 or x86_64');
 const compilerCandidate=trial!==null&&typeof trial==='object';
 if(compilerCandidate&&(Array.isArray(trial)||Object.keys(trial).length!==1||!Object.hasOwn(trial,'compiler_candidate')))throw Error('Invalid compiler candidate selector');
 const label=compilerCandidate?trial.compiler_candidate:trial;
 if(compilerCandidate)validateArmTrial(label);
 if(label!==null){validateArmTrial(label);if(arch!=='aarch64')throw Error('Candidates are ARM64-only')}
 const candidate=BASE+'/'+(compilerCandidate?'arm-compiler':'arm-optimization')+'/'+label;
 return {arch,machine:arch==='aarch64'?'linux-arm64-machine':'linux-amd64-machine',cpu:arch==='aarch64'?'aarch64':'amd64',
  dir:label===null?BASE+'/'+arch:candidate+'/'+arch,compiler:compilerCandidate?candidate+'/compiler/renvo-bootstrap':COMPILER,compilerCandidate,
  fixture:'sandbox/rfe-linux/'+arch,benchmark:'sandbox/rfe-linux/coremark-opt-v2/'+arch};
}
async function confinedDirectory(relative){
 const root=await fs.realpath(ROOT),actual=await fs.realpath(path.join(ROOT,relative));
 if(!actual.startsWith(root+path.sep)||!actual.startsWith(path.join(root,BASE)+path.sep))throw Error('Benchmark directory escapes its sandbox');
 return actual;
}
async function regular(name){const s=await fs.lstat(path.join(ROOT,name));if(!s.isFile()||s.isSymbolicLink())throw Error('Non-regular input '+name);return read(name)}
async function exclusive(name,data){await fs.writeFile(path.join(ROOT,name),data,{flag:'wx'})}
function uniqueReplace(text,from,to){if(text.split(from).length!==2)throw Error('Missing/ambiguous hook '+from);return text.replace(from,()=>to)}
function alignment(n){return (n+3)&~3}
export function newcRecords(data){
 const records=[];let at=0;
 while(at+110<=data.length){
  const start=at,header=data.subarray(at,at+110).toString('ascii');
  if(!/^070701[0-9a-fA-F]{104}$/.test(header))throw Error('Invalid newc header');
  const fields=Array.from({length:13},(_,i)=>parseInt(header.slice(6+8*i,14+8*i),16));
  const size=fields[6],namesize=fields[11];at+=110;
  if(namesize<1||namesize>4096||at+namesize>data.length||data[at+namesize-1]!==0)throw Error('Invalid newc name');
  const name=data.subarray(at,at+namesize-1).toString();at=alignment(at+namesize);
  if(at+size>data.length)throw Error('Truncated newc data');
  const contents=data.subarray(at,at+size);at=alignment(at+size);
  records.push({name,start,end:at,fields,data:contents});
  if(name==='TRAILER!!!'){if(data.subarray(at).some(b=>b!==0))throw Error('Unexpected trailing newc archive');return records}
 }
 throw Error('Missing newc trailer');
}
export function newcRecord(name,data,mode,inode){
 const n=Buffer.from(name+'\0'),f=[inode,mode,0,0,1,0,data.length,0,0,0,0,n.length,0];
 const h=Buffer.from('070701'+f.map(v=>v.toString(16).padStart(8,'0')).join(''));
 return Buffer.concat([h,n,Buffer.alloc((-h.length-n.length)&3),data,Buffer.alloc((-data.length)&3)]);
}
export function addCoremark(base,elf){
 const records=newcRecords(base);
 if(records.some(r=>r.name.replace(/^\.\//,'')==='coremark'))throw Error('CoreMark path already exists');
 const initial=records.find(r=>r.name.replace(/^\.\//,'')==='init');
 if(!initial||!(initial.fields[1]&0o100000))throw Error('Missing base init');
 const inode=Math.max(...records.map(r=>r.fields[0]))+1;
 const combined=Buffer.concat([base.subarray(0,records.at(-1).start),newcRecord('coremark',elf,0o100755,inode),newcRecord('TRAILER!!!',Buffer.alloc(0),0,0)]);
 return Buffer.concat([combined,Buffer.alloc((-combined.length)&511)]);
}
async function sourceNames(metadata=false){
 const names=['go.mod'];
 if(metadata)try{await fs.lstat(path.join(ROOT,'go.sum'));names.push('go.sum')}catch(e){if(e.code!=='ENOENT')throw e}
 for(const e of await fs.readdir(ROOT,{withFileTypes:true}))if(e.isFile()&&e.name.endsWith('.go')&&!e.name.endsWith('_test.go'))names.push(e.name);
 async function walk(dir){for(const e of await fs.readdir(path.join(ROOT,dir),{withFileTypes:true})){
  const name=dir+'/'+e.name;
  if(e.isDirectory()){if(e.name!=='.git'&&name!=='backend/tests')await walk(name)}
  else if(!e.isFile())throw Error('Non-regular source '+name);
  else if(!name.endsWith('_test.go')&&(['.go','.rtg','.rbe','.rfe','.html'].includes(path.extname(name))||metadata&&path.extname(name)==='.json'))names.push(name);
 }}
 for(const dir of ['backend','cmd','internal','std','emulators'])await walk(dir);
 return names.sort();
}
export async function sourceManifest(){
 const files={};for(const name of await sourceNames(true))files[name]=hash(await regular(name));return files;
}
async function sourceHash(){
 const h=createHash('sha256');for(const name of await sourceNames()){h.update(name);h.update('\0');h.update(await read(name));h.update('\0')}return h.digest('hex');
}
async function fixedInputs(arch,trial=null){
 const a=architecture(arch,trial),build=await parse(a.benchmark+'/build.json'),manifest=await parse(a.fixture+'/manifest.json'),pins=await parse('tools/rfe-linux/inputs.json');
 if(build.architecture!==arch||build.iterations_per_seed!==ITERATIONS||build.upstream!=='d5fad6bd094899101a4e5fd53af7298160ced6ab')throw Error('Unexpected cached benchmark');
 if(manifest.architecture!==arch||manifest.alpine!==pins.version||!['iso_sha256','rootfs_sha256'].every(k=>manifest.inputs?.[k]===pins.architectures[arch]?.[k]))throw Error('Alpine provenance mismatch');
 const kernel=await regular(a.fixture+'/kernel'),compressed=await regular(a.fixture+'/initramfs.cpio.gz'),elf=await regular(a.benchmark+'/coremark.elf');
 const reference=await parse(a.benchmark+'/reference.json');
 if(!reference.success||reference.exit_code!==0||reference.timed_out||reference.truncated||!reference.inputs_unchanged||reference.build?.benchmark_sha256!==build.benchmark_sha256||reference.build?.kernel_sha256!==build.kernel_sha256)throw Error('Benchmark differs from archived QEMU reference');
 for(const seed of ['PERF','VALID']){const r=reference.seed_reports.find(r=>r.seed===seed);if(!r?.crc_valid||r.iterations!==ITERATIONS||JSON.stringify([...r.crcs,...r.crcfinal])!==JSON.stringify(EXPECTED[seed]))throw Error('Invalid archived same-ELF CRC reference')}
 if(hash(kernel)!==manifest.kernel_sha256||hash(kernel)!==build.kernel_sha256||hash(compressed)!==manifest.initramfs_sha256||hash(compressed)!==build.base_initramfs_sha256||hash(elf)!==build.benchmark_sha256)throw Error('Cached fixture checksum mismatch');
 for(const [name,digest] of Object.entries(build.source_hashes)){
  if(name.startsWith('/')||name.split('/').includes('..')||hash(await regular(a.benchmark+'/source/'+name))!==digest)throw Error('Benchmark source checksum mismatch '+name);
 }
 const base=gunzipSync(compressed,{maxOutputLength:134217728}),initrd=addCoremark(base,elf);
 const original=await regular('emulators/'+a.machine+'.rfe'),archive=Buffer.from(uniqueReplace(original.toString(),ORIGINAL_LIMIT,PERFORMANCE_LIMIT));
 return {...a,kernel,elf,base,initrd,original,archive,build,manifest};
}
async function prepare(arch,trial=null){
 const a=await fixedInputs(arch,trial);
 if(trial!==null){
  const parent=BASE+'/'+(a.compilerCandidate?'arm-compiler':'arm-optimization');await fs.mkdir(path.join(ROOT,parent),{recursive:true});await confinedDirectory(parent);
 }
 await fs.mkdir(path.join(ROOT,a.dir),{recursive:true});await confinedDirectory(a.dir);
 const manifest={format:1,architecture:arch,original_archive_sha256:hash(a.original),performance_archive_sha256:hash(a.archive),root_sources_sha256:await sourceHash(),
  original_instruction_limit:2000000000,performance_instruction_limit:1000000000000,guest_memory_mib:256,host_arena_bytes:536870912,jit_cache:'balanced64',wall_deadline_seconds:900,
  kernel_sha256:hash(a.kernel),base_initramfs_sha256:a.manifest.initramfs_sha256,base_newc_sha256:hash(a.base),benchmark_sha256:hash(a.elf),initramfs_sha256:hash(a.initrd),
  iterations_per_seed:ITERATIONS,upstream:a.build.upstream,source_hashes:a.build.source_hashes,initramfs_change:'Add only /coremark; preserve original /init and every existing newc record'};
 await exclusive(a.dir+'/'+a.machine+'.rfe',a.archive);await exclusive(a.dir+'/initramfs.cpio',a.initrd);await exclusive(a.dir+'/inputs.json',JSON.stringify(manifest,null,2)+'\n');
 console.log(JSON.stringify({ok:true,action:'prepare',architecture:arch,manifest},null,2));
}
async function checkPrepared(arch,trial=null){
 const a=await fixedInputs(arch,trial);await confinedDirectory(a.dir);const m=await parse(a.dir+'/inputs.json');
 if(m.format!==1||m.architecture!==arch||m.root_sources_sha256!==await sourceHash()||m.original_archive_sha256!==hash(a.original)||m.performance_archive_sha256!==hash(a.archive)||m.initramfs_sha256!==hash(a.initrd)||m.benchmark_sha256!==hash(a.elf)||m.performance_instruction_limit!==1000000000000)throw Error('Stale benchmark preparation');
 if(hash(await regular(a.dir+'/'+a.machine+'.rfe'))!==hash(a.archive)||hash(await regular(a.dir+'/initramfs.cpio'))!==hash(a.initrd))throw Error('Prepared artifact changed');
 return {...a,prepared:m};
}
async function checkBuild(arch,trial=null){
 const a=await checkPrepared(arch,trial),artifact=a.dir+'/'+arch+'.html',html=await regular(artifact),proof=await parse(artifact+'.json');
 const frontend=await fs.realpath(path.join(ROOT,a.compiler)),backend=await fs.realpath(path.join(ROOT,path.dirname(a.compiler),'renvo-backend'));
 if(proof.format!==1||proof.machine!==a.machine||proof.target!=='browser/wasm32'||proof.arena!==536870912||proof.sources_sha256!==await sourceHash()||proof.html_sha256!==hash(html)||proof.compiler_path!==frontend||proof.compiler_backend_path!==backend||proof.compiler_sha256!==hash(await fs.readFile(frontend))||proof.compiler_backend_sha256!==hash(await fs.readFile(backend)))throw Error('Stale or incompatible browser benchmark build');
 const wasm=html.toString().match(/const wasm64="([A-Za-z0-9+/=]+)"/);if(!wasm)throw Error('Missing embedded Wasm');
 const module=new WebAssembly.Module(Buffer.from(wasm[1],'base64')),exports=WebAssembly.Module.exports(module).map(e=>e.name);
 if(!['memory','_start','renvo_browser_step'].every(n=>exports.includes(n)))throw Error('Missing cooperative exports');
 return {...a,artifact,html,proof,compilerReceipt:a.compilerCandidate?await verifyCompilerCandidate(a):undefined};
}
// Compiler experiments share the ordinary benchmark; only their compiler pair
// and output namespace differ. This receipt is provenance, not guest authority.
export async function verifyCompilerCandidate(a){
 if(!a.compilerCandidate)throw Error('Expected a compiler candidate');
 const dir=path.posix.dirname(a.dir),r=await parse(dir+'/compiler.json');
 if(await fs.realpath(path.join(ROOT,dir))!==path.join(await fs.realpath(ROOT),dir))throw Error('Aliased compiler candidate');
 if(r.format!==1||r.source_sha256!==await sourceHash()||r.source_manifest_sha256!==hash(await regular(dir+'/source.json')))throw Error('Stale compiler source receipt');
 const recorded=await parse(dir+'/source.json'),current=await sourceManifest();
 if(Object.keys(recorded.files||{}).length!==Object.keys(current).length||Object.entries(current).some(([name,digest])=>recorded.files[name]!==digest))throw Error('Compiler runtime or policy changed');
 for(const [name,digest] of Object.entries(r.compiler||{})){
  if(!['renvo-bootstrap','renvo-backend'].includes(name)||hash(await regular(dir+'/compiler/'+name))!==digest)throw Error('Changed candidate compiler '+name);
 }
 if(Object.keys(r.compiler||{}).sort().join(',')!=='renvo-backend,renvo-bootstrap')throw Error('Incomplete compiler pair');
 for(const name of ['renvo-backend','renvo-bootstrap'])if(!r.original_binaries?.['sandbox/rfe-resume-bin/'+name])throw Error('Missing original compiler provenance');
 for(const [name,digest] of Object.entries(r.original_binaries||{})){
  if(!/^sandbox\/(agent-bin|range-bin|rfe-range-bin|rfe-resume-bin)\/[A-Za-z0-9_.-]+$/.test(name)||hash(await regular(name))!==digest)throw Error('Changed original compiler '+name);
 }
 for(const name of ['coremark.mjs','compiler-candidate.mjs'])if(hash(await regular('tools/rfe-browser/'+name))!==r.workflow?.[name])throw Error('Compiler workflow changed');
 return r;
}
export {architecture as benchmarkPaths,sourceHash};

export function invocationComplete(state,phase){
 const event=state.events.find(e=>e.phase===phase&&e.line.startsWith('RENVO-BROWSER-COREMARK: END '+phase+' '));
 // Marker events precede the ordinary serial flush. Wait for BOTH transports;
 // keep the original worker timestamp, never the later CDP polling time.
 return !!event&&('\n'+(state.result?.output||'').replace(/\r/g,'')).includes('\n'+event.line+'\n');
}
export function parseInvocation(serial,events,phase){
 const begin='RENVO-BROWSER-COREMARK: BEGIN '+phase,end='RENVO-BROWSER-COREMARK: END '+phase+' 0';
 const list=events.filter(e=>e.phase===phase),expected=[begin,'RENVO-COREMARK: START PERF','RENVO-COREMARK: STOP PERF','RENVO-COREMARK: START VALID','RENVO-COREMARK: STOP VALID',end];
 if(JSON.stringify(list.map(e=>e.line))!==JSON.stringify(expected)||list.some((e,i)=>!Number.isFinite(e.worker_ms)||(i>0&&e.worker_ms<list[i-1].worker_ms)))throw Error('Missing, duplicate, unordered or invalid timing markers');
 const lines=serial.replace(/\r/g,'').split('\n'),lo=lines.indexOf(begin),hi=lines.indexOf(end);
 if(lo<0||hi<=lo||lines.indexOf(begin,lo+1)>=0||lines.indexOf(end,hi+1)>=0)throw Error('Invalid invocation boundaries');
 const text=lines.slice(lo+1,hi).join('\n'),reports=[];
 if(/ERROR|Errors detected/.test(text))throw Error('Guest rejected CoreMark run');
 for(const [i,seed] of ['PERF','VALID'].entries()){
  const stop='RENVO-COREMARK: STOP '+seed,start=text.indexOf(stop),next=i===0?text.indexOf('RENVO-COREMARK: START VALID'):text.length;
  if(start<0||next<=start)throw Error('Invalid seed report order');
  const report=text.slice(start+stop.length,next),crcs=[...report.matchAll(/(?:seedcrc|\[0\]crc(?:list|matrix|state|final))\s*:\s*0x([0-9a-fA-F]{4})/g)].map(m=>m[1].toLowerCase());
  const count=[...report.matchAll(/^Iterations\s*:\s*(\d+)\s*$/gm)],ticks=[...report.matchAll(/^Total ticks\s*:\s*(\d+)\s*$/gm)];
  if(JSON.stringify(crcs)!==JSON.stringify(EXPECTED[seed])||count.length!==1||Number(count[0][1])!==ITERATIONS||ticks.length!==1||(report.match(/Correct operation validated\./g)||[]).length!==1)throw Error('Invalid '+seed+' CRC/count/report');
  const hostSeconds=(list[2+i*2].worker_ms-list[1+i*2].worker_ms)/1000,guestSeconds=Number(ticks[0][1])/1000000;
  if(hostSeconds<10||guestSeconds<10||!Number.isFinite(guestSeconds))throw Error('Run shorter than ten seconds');
  reports.push({seed,iterations:ITERATIONS,host_seconds:hostSeconds,guest_seconds:guestSeconds,iterations_per_host_second:ITERATIONS/hostSeconds,crcs,crc_valid:true});
 }
 return {invocation:phase,label:phase===1?'first invocation after boot':'second invocation in the same guest',performance_score:reports[0].iterations_per_host_second,seed_reports:reports,eligible:true};
}

export function instrumentBenchmark(html){
 html=uniqueReplace(html,'id="shell" type="checkbox"','id="shell" type="checkbox" checked');
 html=uniqueReplace(html,'args=data.shell?["rfe","-shell"]:["rfe"];','args=data.shell?["rfe","-shell"]:["rfe"];args.push("-jit-cache","balanced64");');
 html=uniqueReplace(html,'let instance,memory,done=false,initializing=true,output="",input=[],inputBytes=0;',`let instance,memory,done=false,initializing=true,output="",input=[],inputBytes=0;
 let benchmarkPhase=0,benchmarkLine='',benchmarkBytes=0;
 function observeBenchmark(text){
  benchmarkBytes+=text.length;if(benchmarkBytes>1048576)throw Error('Benchmark serial limit exceeded');
  benchmarkLine+=text;if(benchmarkLine.length>1048576)throw Error('Benchmark serial line limit exceeded');
  let end;while((end=benchmarkLine.indexOf('\\n'))>=0){
   const line=benchmarkLine.slice(0,end).replace(/\\r/g,'');benchmarkLine=benchmarkLine.slice(end+1);
   if(/^RENVO-BROWSER-COREMARK: BEGIN [12]$/.test(line))benchmarkPhase=Number(line.slice(-1));
   if(/^RENVO-COREMARK: (START|STOP) (PERF|VALID)$/.test(line)||/^RENVO-BROWSER-COREMARK: (READY|BEGIN [12]|END [12] [0-9]+)$/.test(line)){
    postMessage({type:'coremark-event',event:{line,phase:benchmarkPhase,worker_ms:performance.now(),time_origin_ms:performance.timeOrigin}});
   }
  }
 }`);
 html=uniqueReplace(html,'output+=decode.decode(part,{stream:true});','const text=decode.decode(part,{stream:true});observeBenchmark(text);output+=text;');
 html=uniqueReplace(html,'window.renvoBrowserResult={state:"idle",','window.renvoCoremarkEvents=[];window.renvoBrowserResult={state:"idle",');
 html=uniqueReplace(html,'worker.onmessage=({data})=>{if(data.type==="output"){','worker.onmessage=({data})=>{if(data.type==="coremark-event"){window.renvoCoremarkEvents.push(data.event);return}else if(data.type==="output"){');
 const scripts=[...html.matchAll(/<script(?:\s[^>]*)?>([\s\S]*?)<\/script>/g)];
 if(scripts.length!==2)throw Error('Unexpected script count');
 scripts.forEach((s,i)=>new vm.Script(s[1],{filename:'coremark-host-'+i+'.js'}));
 return html;
}
async function browserPath(){
 const candidates=['/usr/bin/chromium','/usr/bin/chromium-browser','/usr/bin/google-chrome',...['chrome-linux/chrome','chrome-linux64/chrome'].map(p=>path.join(homedir(),'.cache/ms-playwright/chromium-1194',p))];
 for(const name of candidates)try{await fs.access(name,fs.constants.X_OK);return name}catch{}
 throw Error('Installed Chromium required; no download or installation attempted');
}
async function openCDP(url){
 const u=new URL(url);if(u.hostname!=='127.0.0.1'||u.protocol!=='ws:')throw Error('Debugger must be loopback');
 const ws=new WebSocket(url),pending=new Map();let next=0;
 await new Promise((resolve,reject)=>{const timer=setTimeout(()=>reject(Error('Debugger startup timed out')),10000);ws.addEventListener('open',()=>{clearTimeout(timer);resolve()},{once:true});ws.addEventListener('error',e=>{clearTimeout(timer);reject(e)},{once:true})});
 ws.addEventListener('message',({data})=>{const m=JSON.parse(data),p=pending.get(m.id);if(p){pending.delete(m.id);clearTimeout(p.timer);m.error?p.reject(Error(JSON.stringify(m.error))):p.resolve(m.result)}});
 ws.addEventListener('close',()=>{for(const p of pending.values()){clearTimeout(p.timer);p.reject(Error('Debugger disconnected'))}pending.clear()});
 return {ws,call(method,params={},sessionId){return new Promise((resolve,reject)=>{const id=++next,timer=setTimeout(()=>{pending.delete(id);reject(Error('Debugger command timed out: '+method))},10000);pending.set(id,{resolve,reject,timer});try{ws.send(JSON.stringify({id,method,params,...(sessionId?{sessionId}:{})}))}catch(e){clearTimeout(timer);pending.delete(id);reject(e)}})}};
}
function signalBrowser(browser,signal){if(!browser?.pid)return;try{if(process.platform==='win32')browser.kill(signal);else process.kill(-browser.pid,signal)}catch(e){if(e.code!=='ESRCH')throw e}}
async function stopBrowser(browser){
 if(!browser?.pid)return;
 signalBrowser(browser,'SIGTERM');
 if(browser.exitCode===null&&browser.signalCode===null)await new Promise(resolve=>{const timer=setTimeout(resolve,2000);browser.once('exit',()=>{clearTimeout(timer);resolve()})});
 signalBrowser(browser,'SIGKILL');
}
async function runBenchmark(arch,harnessSource,trial=null,profile=false){
 const a=await checkBuild(arch,trial),executable=await browserPath();if(typeof WebSocket!=='function')throw Error('Node WebSocket support required');
 const html=Buffer.from(instrumentBenchmark(a.html.toString()));
 const routes=new Map([['/',{data:html,type:'text/html'}],['/kernel',{data:a.kernel,type:'application/octet-stream'}],['/initramfs.cpio.gz',{data:a.initrd,type:'application/octet-stream'}]]);
 const server=createServer((req,res)=>{const route=routes.get(new URL(req.url,'http://localhost').pathname);if(req.method!=='GET'||!route){res.writeHead(404);res.end();return}res.writeHead(200,{'Content-Type':route.type,'Cache-Control':'no-store','Content-Security-Policy':"default-src 'none'; script-src 'unsafe-inline' 'wasm-unsafe-eval'; worker-src blob:; connect-src 'self'; style-src 'unsafe-inline'"});res.end(route.data)});
 await confinedDirectory(a.dir);
 const lock=BASE+'/RUNNING';await exclusive(lock,JSON.stringify({architecture:arch,pid:process.pid}));
 let outputDirectory,temporary,browser,cdp,session,latest,workerSession,workerReady,profilerStarted=false,stderr='',fatalError=null,interrupted=null,lastProgress=0;
 const result={...(a.compilerCandidate?{compiler_candidate:a.compilerReceipt}:{}),format:1,architecture:arch,candidate:trial,profile_mode:profile?'warm-invocation':null,score_eligible:!profile,ok:false,date_utc:new Date().toISOString(),harness_sha256:hash(harnessSource),browser_artifact:a.artifact,
  original_html_sha256:hash(a.html),instrumented_html_sha256:hash(html),inputs:a.prepared,build:a.proof,
  host:{platform:platform(),architecture:hostArch(),cpu:cpus()[0]?.model,logical_cpus:cpus().length,node:process.version},
  methodology:{backend:'browser/Wasm full-system RFE',timing:profile?'DIAGNOSTIC: worker CPU sampling during second invocation; NOT a throughput score':'Host monotonic performance.now() at complete serial line inside worker WASI write; no CPU or per-call profiling',
   scope:'CoreMark start/stop only; excludes boot, shell, executable loading and result printing; includes guest kernel and timer work',
   invocations:'Exactly two in the same booted guest; no unreported warm-up',iterations_per_seed:ITERATIONS,minimum_host_and_guest_seconds:10,jit_cache:'balanced64',official_eembc_submission:false},
  runs:[],serial_events:[],timed_out:false,stderr_truncated:false};
 const onSignal=signal=>{interrupted=signal;try{signalBrowser(browser,'SIGTERM')}catch{}};
 const onINT=()=>onSignal('SIGINT'),onTERM=()=>onSignal('SIGTERM');process.on('SIGINT',onINT);process.on('SIGTERM',onTERM);
 const start=performance.now(),deadline=start+DEADLINE_MS;
 try{
  for(let i=1;i<=4;i++){const name=a.dir+'/run-'+i;try{await fs.mkdir(path.join(ROOT,name));outputDirectory=name;break}catch(e){if(e.code!=='EEXIST')throw e}}
  if(!outputDirectory)throw Error('Four preserved attempts exist; refusing more runs');
  await confinedDirectory(outputDirectory);
  await exclusive(outputDirectory+'/instrumented.html',html);
  await new Promise((resolve,reject)=>{server.once('error',reject);server.listen(0,'127.0.0.1',resolve)});
  temporary=await fs.mkdtemp(path.join(tmpdir(),'renvo-browser-coremark-'));
  browser=spawn(executable,['--headless','--no-sandbox','--disable-gpu','--disable-background-networking','--disable-component-update','--no-first-run','--no-default-browser-check','--remote-debugging-address=127.0.0.1','--remote-debugging-port=0','--user-data-dir='+temporary,'about:blank'],{stdio:['ignore','ignore','pipe'],detached:process.platform!=='win32'});
  const debuggerURL=await new Promise((resolve,reject)=>{
   const timer=setTimeout(()=>reject(Error('Browser startup timed out')),15000);
   browser.once('error',e=>{clearTimeout(timer);reject(e)});browser.once('exit',code=>{clearTimeout(timer);reject(Error('Browser exited during startup '+code))});
   browser.stderr.on('data',b=>{if(stderr.length+b.length>MAX_SERIAL){result.stderr_truncated=true;fatalError=Error('Chromium stderr exceeded capture limit')}else stderr+=b.toString();const m=stderr.match(/DevTools listening on (ws:\/\/127\.0\.0\.1:[^\s]+)/);if(m){clearTimeout(timer);resolve(m[1])}});
  });
  cdp=await openCDP(debuggerURL);result.browser=await cdp.call('Browser.getVersion');
  const {targetId}=await cdp.call('Target.createTarget',{url:'about:blank'});({sessionId:session}=await cdp.call('Target.attachToTarget',{targetId,flatten:true}));
  const evaluate=async expression=>{const e=await cdp.call('Runtime.evaluate',{expression,returnByValue:true},session);if(e.exceptionDetails)throw Error(JSON.stringify(e.exceptionDetails));return e.result.value};
  const send=async command=>evaluate('send('+JSON.stringify(command)+')');
  if(profile){
   cdp.ws.addEventListener('message',({data})=>{
    const m=JSON.parse(data);if(m.method!=='Target.attachedToTarget'||m.sessionId!==session||m.params.targetInfo.type!=='worker')return;
    const attached=m.params.sessionId;
    if(workerSession){fatalError=Error('Unexpected second worker');cdp.call('Runtime.runIfWaitingForDebugger',{},attached).catch(()=>{});return}
    workerSession=attached;
    workerReady=(async()=>{try{
     if(!m.params.waitingForDebugger)throw Error('Worker not paused for diagnostic attachment');
     await cdp.call('Profiler.enable',{},attached);await cdp.call('Profiler.setSamplingInterval',{interval:1000},attached);
    }catch(e){fatalError=e}finally{await cdp.call('Runtime.runIfWaitingForDebugger',{},attached)}})();
   });
   await cdp.call('Target.setAutoAttach',{autoAttach:true,waitForDebuggerOnStart:true,flatten:true},session);
  }
  await cdp.call('Page.navigate',{url:'http://127.0.0.1:'+server.address().port+'/?boot=1'},session);
  async function waitUntil(predicate,until){
   for(;;){
    if(interrupted)throw Error('Interrupted '+interrupted);if(fatalError)throw fatalError;
    if(performance.now()>until){result.timed_out=true;throw Error('Fixed browser benchmark deadline exceeded')}
    latest=await evaluate('({result:window.renvoBrowserResult||null,events:window.renvoCoremarkEvents||[]})');
    const state=latest.result;if(state?.output?.length>MAX_SERIAL)throw Error('Serial capture limit exceeded');
    if(state?.state==='failed'||state?.state==='halted')throw Error('Guest stopped: '+JSON.stringify({state:state.state,error:state.error,exitCode:state.exitCode}));
    if(predicate(latest))return;
    if(performance.now()-lastProgress>30000){lastProgress=performance.now();console.error(JSON.stringify({architecture:arch,elapsed_seconds:Math.round((performance.now()-start)/1000),state:state?.state||'loading',last_marker:latest.events.at(-1)?.line||null}))}
    await pause(100);
   }
  }
  await waitUntil(s=>['RENVO-RFE-LINUX: EXEC PASS','RENVO-RFE-LINUX: FILE PASS','RENVO-RFE-LINUX: PIPE PASS','RENVO-RFE-LINUX: PASS'].every(m=>s.result?.output?.includes(m)),Math.min(deadline,start+BOOT_MS));
  await send("printf '\\nRENVO-BROWSER-COREMARK: READY\\n'\r");
  await waitUntil(s=>s.events.some(e=>e.line==='RENVO-BROWSER-COREMARK: READY'),Math.min(deadline,start+BOOT_MS));
  result.boot_to_verified_shell_seconds=(performance.now()-start)/1000;
  for(const phase of [1,2]){
   if(profile&&phase===2){
    if(!workerReady||!workerSession)throw Error('Diagnostic worker did not attach');await workerReady;if(fatalError)throw fatalError;
    await cdp.call('Profiler.start',{},workerSession);profilerStarted=true;
   }
   const command="printf '\\nRENVO-BROWSER-COREMARK: BEGIN "+phase+"\\n'; /coremark; result=$?; printf '\\nRENVO-BROWSER-COREMARK: END "+phase+" %s\\n' \"$result\"\r";
   await send(command);
   await waitUntil(s=>invocationComplete(s,phase),deadline);
   const report=parseInvocation(latest.result.output,latest.events,phase);if(profile){report.eligible=false;report.diagnostic=true}result.runs.push(report);
   console.error(JSON.stringify({architecture:arch,invocation:phase,performance_score:report.performance_score,crc_valid:true}));
  }
  if(profile){
   const stopped=await cdp.call('Profiler.stop',{},workerSession);profilerStarted=false;
   const cpu=stopped.profile,raw=JSON.stringify(cpu);
   if(!cpu?.samples?.length||cpu.samples.length!==cpu.timeDeltas?.length||raw.length>16777216)throw Error('Invalid/oversized diagnostic CPU profile');
   await exclusive(outputDirectory+'/warm-cpu.json',raw+'\n');
   result.cpu_profile={path:outputDirectory+'/warm-cpu.json',samples:cpu.samples.length,duration_ms:(cpu.endTime-cpu.startTime)/1000,scope:'Second invocation only; includes shell/loading/reporting; diagnostic, not score'};
  }
  if(await sourceHash()!==a.prepared.root_sources_sha256)throw Error('Implementation changed during benchmark');
  if(a.compilerCandidate)await verifyCompilerCandidate(a);
  result.ok=true;
 }catch(e){result.error=String(e.stack||e);result.ok=false}
 finally{
  result.host_total_seconds=(performance.now()-start)/1000;result.interrupted=interrupted;result.serial_events=latest?.events||[];result.score_eligible=!profile&&result.ok;
  if(profilerStarted&&cdp&&workerSession)try{await cdp.call('Profiler.stop',{},workerSession)}catch{}
  if(cdp){try{cdp.ws.close()}catch{}}
  try{await stopBrowser(browser)}catch(e){result.cleanup_error=String(e);result.ok=false}
  if(server.listening){server.closeAllConnections();await new Promise(resolve=>server.close(resolve))}
  if(temporary)try{await fs.rm(temporary,{recursive:true,force:true})}catch(e){result.cleanup_error=String(e);result.ok=false}
  process.off('SIGINT',onINT);process.off('SIGTERM',onTERM);
  if(outputDirectory){await exclusive(outputDirectory+'/serial.log',latest?.result?.output||'');await exclusive(outputDirectory+'/stderr.log',stderr);await exclusive(outputDirectory+'/result.json',JSON.stringify(result,null,2)+'\n')}
  await fs.unlink(path.join(ROOT,lock));
 }
 console.log(JSON.stringify({ok:result.ok,architecture:arch,report:outputDirectory+'/result.json',runs:result.runs,error:result.error||null},null,2));
 if(!result.ok)throw Error(result.error||'Browser benchmark failed');
}
export async function main(action,arch,harnessSource,trial=null){
 if(harnessSource!==undefined&&(await read('tools/rfe-browser/coremark.mjs')).toString()!==harnessSource)throw Error('Benchmark harness changed during launch');
 architecture(arch,trial);
 if(action==='profile'&&(arch!=='aarch64'||trial===null))throw Error('Profiling requires an isolated ARM candidate');
 if(action==='prepare')return prepare(arch,trial);
 if(action==='before-build'){
  const a=await checkPrepared(arch,trial);
  for(const name of [a.dir+'/'+arch+'.html',a.dir+'/'+arch+'.html.json'])try{await fs.lstat(path.join(ROOT,name));throw Error('Build artifact already exists')}catch(e){if(e.code!=='ENOENT')throw e}
  console.log('PASS: prepared provenance and empty build destination');return;
 }
 if(action==='check'){const a=await checkBuild(arch,trial);console.log(JSON.stringify({ok:true,architecture:arch,artifact:a.artifact,prepared:a.prepared},null,2));return}
 if(action==='run'||action==='profile')return runBenchmark(arch,harnessSource,trial,action==='profile');
 throw Error('Unknown fixed benchmark action');
}
