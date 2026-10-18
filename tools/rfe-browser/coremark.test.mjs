import {readFile} from 'node:fs/promises';
import vm from 'node:vm';
import {newcRecord,newcRecords,addCoremark,parseInvocation,instrumentBenchmark,invocationComplete,validateArmTrial,main,benchmarkPaths} from './coremark.mjs';
import {compilerChanges} from './compiler-candidate.mjs';
const read=name=>readFile(name);
const EXPECTED={PERF:['e9f5','e714','1fd7','8e3a','382f'],VALID:['18f2','e3c1','0747','8d84','d304']};
import test from 'node:test';
import assert from 'node:assert/strict';
function syntheticInvocation(phase=1){
 const begin='RENVO-BROWSER-COREMARK: BEGIN '+phase,end='RENVO-BROWSER-COREMARK: END '+phase+' 0',lines=[begin],events=[{phase,line:begin,worker_ms:1}];
 for(const [i,seed] of ['PERF','VALID'].entries()){
  lines.push('RENVO-COREMARK: START '+seed,'RENVO-COREMARK: STOP '+seed,'Total ticks      : 15000000','Iterations       : 20000');
  ['seedcrc','[0]crclist','[0]crcmatrix','[0]crcstate','[0]crcfinal'].forEach((name,j)=>lines.push(name+' : 0x'+EXPECTED[seed][j]));
  lines.push('Correct operation validated.');
  events.push({phase,line:'RENVO-COREMARK: START '+seed,worker_ms:2+i*20000},{phase,line:'RENVO-COREMARK: STOP '+seed,worker_ms:15002+i*20000});
 }
 lines.push(end);events.push({phase,line:end,worker_ms:40000});return {serial:lines.join('\r\n')+'\r\n',events};
}
test('host timing is independent of guest-reported time, both phases validated',()=>{
 const first=syntheticInvocation(1),second=syntheticInvocation(2),serial=first.serial+second.serial,events=[...first.events,...second.events];
 const a=parseInvocation(serial,events,1),b=parseInvocation(serial,events,2);
 assert.equal(a.performance_score,20000/15);assert.equal(b.seed_reports.length,2);assert.equal(b.label,'second invocation in the same guest');
 const changed=first.serial.replaceAll('15000000','99000000');assert.equal(parseInvocation(changed,first.events,1).performance_score,a.performance_score);
});
test('reject corrupt CRC, final CRC, count, missing/duplicate/out-of-order markers and failure exit',()=>{
 const original=syntheticInvocation();
 for(const bad of [original.serial.replace('0xe9f5','0xffff'),original.serial.replace('0x382f','0xffff'),original.serial.replace('20000','99999'),original.serial.replace('Correct operation validated.','BAD'),original.serial.replace('END 1 0','END 1 1'),original.serial.replace('Total ticks      : 15000000','Total ticks      : 1')])assert.throws(()=>parseInvocation(bad,original.events,1));
 for(const bad of [original.events.slice(0,-1),[...original.events,original.events.at(-1)],[original.events[0],original.events[2],original.events[1],...original.events.slice(3)]])assert.throws(()=>parseInvocation(original.serial,bad,1));
 const short=structuredClone(original.events);short[2].worker_ms=100;assert.throws(()=>parseInvocation(original.serial,short,1));
 const wrongPhase=structuredClone(original.events);wrongPhase[2].phase=2;assert.throws(()=>parseInvocation(original.serial,wrongPhase,1));
});
test('additive newc preserves every old entry byte and exact ELF contents',()=>{
 const base=Buffer.concat([newcRecord('init',Buffer.from('#!/bin/sh\necho checks\n'),0o100755,1),newcRecord('old-file',Buffer.from('old'),0o100640,2),newcRecord('TRAILER!!!',Buffer.alloc(0),0,0)]),elf=Buffer.from([127,69,76,70,0,1,255]);
 const result=addCoremark(base,elf),before=newcRecords(base),after=newcRecords(result);
 assert.deepEqual(result.subarray(0,before.at(-1).start),base.subarray(0,before.at(-1).start));
 assert.deepEqual(after.find(r=>r.name==='coremark').data,elf);assert.equal(after.find(r=>r.name==='coremark').fields[1],0o100755);assert.equal(after.length,before.length+1);
 assert.throws(()=>addCoremark(result,elf));assert.throws(()=>newcRecords(base.subarray(0,120)));assert.throws(()=>newcRecords(Buffer.concat([base,Buffer.from('bad')])));
});
test('real production host hook timestamps complete fragmented lines before flushing',async()=>{
 const original=(await read('internal/rfe/browser_host.html')).toString(),instrumented=instrumentBenchmark(original),worker=[...instrumented.matchAll(/<script(?:\s[^>]*)?>([\s\S]*?)<\/script>/g)][0][1],messages=[],clock={now:()=>clock.ms,timeOrigin:10,ms:1};
 class FakeChannel {constructor(){this.port1={};this.port2={postMessage(){}}}}
 const ctx=vm.createContext({TextDecoder,TextEncoder,WebAssembly,MessageChannel:FakeChannel,performance:clock,postMessage:e=>messages.push(e),onmessage:null});
 new vm.Script(worker).runInContext(ctx);
 new vm.Script('observeBenchmark("RENVO-BROWSER-COREMARK: BEGIN 1\\r\\n"); observeBenchmark("RENVO-COREMARK: STA");').runInContext(ctx);
 assert.equal(messages.length,1);clock.ms=200;
 new vm.Script('observeBenchmark("RT PERF\\r\\n");').runInContext(ctx);
 assert.equal(messages[1].event.line,'RENVO-COREMARK: START PERF');assert.equal(messages[1].event.worker_ms,200);assert.equal(messages[1].event.phase,1);
 assert.ok(instrumented.includes('args.push("-jit-cache","balanced64")'));assert.ok(instrumented.includes('type="checkbox" checked'));
 assert.throws(()=>instrumentBenchmark(original.replace('args=data.shell?','args=unknown?')));
});

test('wait for complete serial flush after an earlier exit-marker event',()=>{
 const data=syntheticInvocation(1),end='RENVO-BROWSER-COREMARK: END 1 0',state={events:data.events,result:{output:data.serial.slice(0,data.serial.indexOf(end))}};
 assert.equal(invocationComplete(state,1),false);
 state.result.output+=end;assert.equal(invocationComplete(state,1),false);
 state.result.output+='\r\n';assert.equal(invocationComplete(state,1),true);
 assert.equal(parseInvocation(state.result.output,state.events,1).eligible,true);
});

test('ARM candidate labels cannot select paths or other architectures',async()=>{
 for(const label of ['', '.', '..', '../outside', '/absolute', 'a/b', 'a\\b', '-flag', 'a'.repeat(101)])assert.throws(()=>validateArmTrial(label));
 assert.equal(validateArmTrial('baseline-arm-01'),'baseline-arm-01');
 await assert.rejects(()=>main('profile','x86_64',undefined,'baseline-arm-01'),/ARM64-only/);
});

test('compiler selection cannot redirect the original workflow or accept arbitrary paths',()=>{
 const original=benchmarkPaths('aarch64','retained'),candidate=benchmarkPaths('aarch64',{compiler_candidate:'trial-1'});
 assert.equal(original.dir,'sandbox/rfe-browser-coremark/v2/arm-optimization/retained/aarch64');
 assert.equal(original.compiler,'sandbox/rfe-resume-bin/renvo-bootstrap');
 assert.equal(benchmarkPaths('x86_64').compiler,original.compiler);
 assert.equal(candidate.dir,'sandbox/rfe-browser-coremark/v2/arm-compiler/trial-1/aarch64');
 assert.equal(candidate.compiler,'sandbox/rfe-browser-coremark/v2/arm-compiler/trial-1/compiler/renvo-bootstrap');
 for(const value of [{compiler_candidate:null},{compiler_candidate:'../outside'},{compiler_candidate:'x',compiler:'/tmp/compiler'},[],{}])assert.throws(()=>benchmarkPaths('aarch64',value));
 assert.throws(()=>benchmarkPaths('x86_64',{compiler_candidate:'x'}));
});
test('compiler experiments reject runtime/policy changes and unchanged-source relabels',()=>{
 const base={'backend/definitions/wasm32.rtg':'a','internal/rfe/runtime/wasm_emit.go':'b','internal/perfgate/policy.json':'c'};
 assert.deepEqual(compilerChanges(base,base,'control'),[]);
 const changed={...base,'backend/definitions/wasm32.rtg':'d','internal/backendcompiled/compiler_generated.go':'e'};
 assert.deepEqual(compilerChanges(base,changed,'trial-1'),['backend/definitions/wasm32.rtg','internal/backendcompiled/compiler_generated.go']);
 assert.throws(()=>compilerChanges(base,base,'renamed'),/compiler change/);
 assert.throws(()=>compilerChanges(base,changed,'control'),/Control source changed/);
 for(const p of ['internal/rfe/runtime/wasm_emit.go','internal/perfgate/policy.json','backend/main_test.go','cmd/renvoemu/browser.go','go.mod']){
  assert.throws(()=>compilerChanges(base,{...changed,[p]:'bad'},'trial-1'),/Noncompiler source or policy/);
 }
});
