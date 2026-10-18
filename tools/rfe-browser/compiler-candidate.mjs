// Isolated compiler experiments. Benchmark execution stays in coremark.mjs.
import fs from 'node:fs/promises';
import path from 'node:path';
import {pathToFileURL} from 'node:url';
import {createHash} from 'node:crypto';
import {spawn,execFile} from 'node:child_process';
import {main as benchmark,benchmarkPaths,sourceHash,sourceManifest,validateArmTrial,verifyCompilerCandidate} from './coremark.mjs';

const ROOT=process.cwd(),BASE='sandbox/rfe-browser-coremark/v2/arm-compiler';
const RETAINED='sandbox/rfe-browser-coremark/v2/arm-optimization/rooted-tree-coordinated-host-and-cold-paths/aarch64';
const SOURCE='5343b8b9991839fca396cb85773c27bc4508912a36a107a5497d478c4e7e99c3';
const ORIGINAL_DIRS=['sandbox/agent-bin','sandbox/range-bin','sandbox/rfe-range-bin','sandbox/rfe-resume-bin'];
const OFFLINE={...process.env,GOTOOLCHAIN:'local',GOPROXY:'off',GOSUMDB:'off',GOWORK:'off'};
const hash=b=>createHash('sha256').update(b).digest('hex');
async function regular(p){const s=await fs.lstat(path.join(ROOT,p));if(!s.isFile()||s.isSymbolicLink())throw Error('Nonregular input '+p);return fs.readFile(path.join(ROOT,p))}
const parse=async p=>JSON.parse(await regular(p));
async function absent(p){try{await fs.lstat(path.join(ROOT,p));throw Error('Existing output '+p)}catch(e){if(e.code!=='ENOENT')throw e}}
async function directory(p){if(await fs.realpath(path.join(ROOT,p))!==path.join(await fs.realpath(ROOT),p))throw Error('Aliased output directory '+p)}
async function write(p,value){await fs.writeFile(path.join(ROOT,p),JSON.stringify(value,null,2)+'\n',{flag:'wx'})}
async function workflow(){const files={};for(const name of ['coremark.mjs','compiler-candidate.mjs'])files[name]=hash(await regular('tools/rfe-browser/'+name));return files}
function same(a,b){return Object.keys(a).length===Object.keys(b).length&&Object.entries(a).every(([p,digest])=>b[p]===digest)}
export function compilerSourceAllowed(p){
 return /^backend\/compiler_[^/]+_impl\.go$/.test(p)||p==='backend/compiler_main.go'||/^backend\/definitions\/[^/]+\.(rtg|rbe)$/.test(p)||
  ['backend/rtg_vm32_contract_generated.go','internal/backendcompiled/compiler_generated.go','internal/backendcompiled/sources_generated.go',
   'internal/targetinfo/registry_generated.go','internal/targetinfo/targets.json'].includes(p);
}
export function compilerChanges(baseline,current,label){
 validateArmTrial(label);const changed=[];
 for(const p of [...new Set([...Object.keys(baseline),...Object.keys(current)])].sort())if(baseline[p]!==current[p]){
  if(!compilerSourceAllowed(p))throw Error('Noncompiler source or policy changed: '+p);changed.push(p);
 }
 if(label==='control'&&changed.length)throw Error('Control source changed');
 if(label!=='control'&&!changed.length)throw Error('A candidate needs a compiler change, not a new label');
 return changed;
}
async function originals(){
 const binaries={};for(const dir of ORIGINAL_DIRS)try{for(const e of await fs.readdir(path.join(ROOT,dir),{withFileTypes:true})){
  if(!e.isFile())throw Error('Nonregular original artifact '+dir+'/'+e.name);
  const p=dir+'/'+e.name;binaries[p]=hash(await regular(p));
 }}catch(e){if(e.code!=='ENOENT')throw e}
 const proof=await parse(RETAINED+'/aarch64.html.json');
 if(proof.sources_sha256!==SOURCE||binaries['sandbox/rfe-resume-bin/renvo-bootstrap']!==proof.compiler_sha256||
  binaries['sandbox/rfe-resume-bin/renvo-backend']!==proof.compiler_backend_sha256)throw Error('Original compiler provenance changed');
 return binaries;
}
async function verifySource(dir){
 await directory(dir);await directory(dir+'/compiler');const receipt=await parse(dir+'/source.json');
 if(receipt.format!==1||receipt.source_sha256!==await sourceHash()||!same(receipt.files,await sourceManifest())||
  !same(receipt.original_binaries,await originals())||!same(receipt.workflow,await workflow()))throw Error('Compiler experiment inputs changed');
 return receipt;
}
let installedGo;
async function goToolchain(){
 if(installedGo)return installedGo;
 const capture=(file,args)=>new Promise((resolve,reject)=>execFile(file,args,{cwd:ROOT,env:OFFLINE,timeout:10000,maxBuffer:65536},(error,out)=>error?reject(error):resolve(out.trim())));
 // Query the system driver without auto-selection, then execute only the
 // already cached required toolchain. Never ask Go to download a toolchain.
 const cache=await capture('go',['env','GOMODCACHE']);
 const executable=path.join(cache,'golang.org/toolchain@v0.0.1-go1.25.5.linux-amd64/bin/go');
 const version=await capture(executable,['version']);
 if(version!=='go version go1.25.5 linux/amd64')throw Error('Required cached Go1.25.5 is unavailable');
 installedGo={path:executable,version,sha256:hash(await fs.readFile(executable))};return installedGo;
}
async function go(args,timeout){
 const toolchain=await goToolchain();
 await new Promise((resolve,reject)=>{const child=spawn(toolchain.path,args,{cwd:ROOT,env:OFFLINE,stdio:'inherit',timeout,killSignal:'SIGKILL'});
  child.once('error',reject);child.once('exit',(code,signal)=>code===0?resolve():reject(Error('Fixed Go build failed: '+(signal||code))));
 });
}
async function prepare(label,dir,selector,harness){
 const files=await sourceManifest(),source_sha256=await sourceHash(),original_binaries=await originals();
 let changed=[];
 if(label==='control'){if(source_sha256!==SOURCE)throw Error('Control must start from the reviewed retained sources')}
 else {const baseline=await parse(BASE+'/control/source.json');changed=compilerChanges(baseline.files,files,label);
  if(!same(baseline.original_binaries,original_binaries)||!same(baseline.workflow,await workflow()))throw Error('Baseline provenance changed');
 }
 await fs.mkdir(path.join(ROOT,BASE),{recursive:true});await directory(BASE);await absent(dir);
 for(const e of await fs.readdir(path.join(ROOT,BASE),{withFileTypes:true})){
  if(!e.isDirectory())throw Error('Unexpected compiler namespace entry');validateArmTrial(e.name);await directory(BASE+'/'+e.name);
  try{if((await parse(BASE+'/'+e.name+'/source.json')).source_sha256===source_sha256)throw Error('Compiler source already preserved as '+e.name)}catch(error){if(error.code!=='ENOENT')throw error}
 }
 await fs.mkdir(path.join(ROOT,dir));await fs.mkdir(path.join(ROOT,dir+'/compiler'));await fs.mkdir(path.join(ROOT,dir+'/sources'));
 for(const p of Object.keys(files).filter(compilerSourceAllowed)){
  const dest=dir+'/sources/'+p;await fs.mkdir(path.dirname(path.join(ROOT,dest)),{recursive:true});await directory(path.posix.dirname(dest));
  const bytes=await regular(p);if(hash(bytes)!==files[p])throw Error('Source changed during snapshot');await fs.writeFile(path.join(ROOT,dest),bytes,{flag:'wx'});
 }
 await write(dir+'/source.json',{format:1,label,source_sha256,files,changed,original_binaries,workflow:await workflow()});
 await benchmark('prepare','aarch64',harness,selector);
}
async function buildCompiler(dir){
 const source=await verifySource(dir);for(const name of ['renvo-backend','renvo-bootstrap'])await absent(dir+'/compiler/'+name);
 await go(['build','-o',path.join(ROOT,dir+'/compiler/renvo-backend'),'./backend'],180000);
 await go(['build','-o',path.join(ROOT,dir+'/compiler/renvo-bootstrap'),'-tags=renvo_bundle','./cmd/renvobootstrap'],180000);
 await verifySource(dir);const compiler={};for(const name of ['renvo-backend','renvo-bootstrap'])compiler[name]=hash(await regular(dir+'/compiler/'+name));
 await write(dir+'/compiler.json',{format:1,source_sha256:source.source_sha256,source_manifest_sha256:hash(await regular(dir+'/source.json')),
  compiler,go_toolchain:await goToolchain(),original_binaries:source.original_binaries,workflow:source.workflow});
 console.log(JSON.stringify({ok:true,compiler}));
}
export async function compilerMain(action,label){
 if(process.platform!=='linux'||process.arch!=='x64')throw Error('Compiler workflow requires Linux/amd64');
 if(!['prepare','build-compiler','build','check','run'].includes(action))throw Error('Invalid fixed compiler action');
 validateArmTrial(label);const selector={compiler_candidate:label},paths=benchmarkPaths('aarch64',selector),dir=path.posix.dirname(paths.dir);
 const harness=(await regular('tools/rfe-browser/coremark.mjs')).toString();
 if(action==='prepare')return prepare(label,dir,selector,harness);
 await verifySource(dir);
 if(action==='build-compiler')return buildCompiler(dir);
 await verifyCompilerCandidate(paths);
 if(action==='build'){
  await benchmark('before-build','aarch64',harness,selector);
  await go(['run','./cmd/renvoemu','build','-host','renvo','-target','browser/wasm32','-compiler',path.join(ROOT,paths.compiler),
   '-arena','536870912','-I','emulators','-o',path.join(ROOT,paths.dir+'/aarch64.html'),paths.dir+'/linux-arm64-machine.rfe'],300000);
  await verifySource(dir);await benchmark('check','aarch64',harness,selector);return;
 }
 await benchmark(action,'aarch64',harness,selector);
}
if(process.argv[1]&&import.meta.url===pathToFileURL(path.resolve(process.argv[1])).href){
 const [action,label,...extra]=process.argv.slice(2);if(extra.length)throw Error('Unexpected compiler arguments');await compilerMain(action,label);
}
