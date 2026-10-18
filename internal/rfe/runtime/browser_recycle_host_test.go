//go:build !renvo

package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestBrowserHostCodeTransferRevocationBudgetsAndNoRecompile(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node required for actual host/Wasm transfer")
	}
	host, err := os.ReadFile("../browser_host.html")
	if err != nil {
		t.Fatal(err)
	}
	ops := []Op{{Kind: LoadState, Imm: 0}, {Kind: Const, Imm: 1}, {Kind: Add, A: 0, B: 1}, {Kind: StoreState, A: 2, Imm: 0}}
	block, err := EmitWasmBlock(ops, 1)
	if err != nil {
		t.Fatal(err)
	}
	dispatcher, err := EmitWasmDispatcher(1, 0, wasmLoopTestLayout())
	if err != nil {
		t.Fatal(err)
	}
	input, err := json.Marshal(struct {
		Host                           string
		Gate, Block, Dispatcher, Proof []byte
	}{string(host), emitWasmGate(), block, dispatcher, emitWasmProof()})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "-e", `
const fs=require('node:fs'),vm=require('node:vm'),assert=require('node:assert/strict'),x=JSON.parse(fs.readFileSync(0,'utf8'));
const source=[...x.Host.matchAll(/<script(?:\s[^>]*)?>([\s\S]*?)<\/script>/g)][0][1];
let proofInstances=0;const trackedWasm=Object.create(WebAssembly);trackedWasm.Instance=new Proxy(WebAssembly.Instance,{construct(target,args){if(WebAssembly.Module.imports(args[0]).some(i=>i.module==='proof'))proofInstances++;return Reflect.construct(target,args)}});
const c=vm.createContext({onmessage:null,TextDecoder,TextEncoder,WebAssembly:trackedWasm,postMessage(){},MessageChannel:class{constructor(){this.port1={};this.port2={postMessage(){}}}}});
vm.runInContext(source,c);c.mainMemory=new WebAssembly.Memory({initial:1});vm.runInContext('memory=mainMemory;instance={exports:{}}',c);
const service=()=>{const replyBytes=new Uint8Array(4);return {replyBytes,replyView:new DataView(replyBytes.buffer)}};
function request(f,words,code=[]){const p=new Uint8Array(4*words.length+code.length),v=new DataView(p.buffer);words.forEach((w,i)=>v.setUint32(4*i,w,true));p.set(code,4*words.length);c.f=f;c.packet=p;vm.runInContext('jitRequest(f,[packet])',c);return f.replyView.getInt32(0,true)}
const gate=Buffer.from(x.Gate,'base64'),block=Buffer.from(x.Block,'base64'),dispatcher=Buffer.from(x.Dispatcher,'base64'),proof=Buffer.from(x.Proof,'base64');
const f=service(),owner=request(f,[0,1<<20],gate);assert.ok(owner>0);
const a=request(f,[1,1,0,64],block),d=request(f,[1,1,1,64],dispatcher),old=f.jit,oldGate=vm.runInContext('gates.get(f.jit.owner&32767)',c);
request(f,[3,a,0,0,1,1],proof);request(f,[3,d,4,0,1,1],proof);
assert.equal(old.table.get(a/8+1),old.table.get(d/8+1),'one arena prover, not per-entry instances');
const admissionBytes=old.bytes;for(const changed of [[3,a,1,0,1,1],[3,a,0,1,1,1],[3,a,0,0,1,2],[3,a,0,0,2,1]])assert.throws(()=>request(f,changed,proof));assert.equal(old.bytes,admissionBytes,'rejected proof tuple not charged');
const before=old.bytes;assert.equal(request(f,[4,owner]),0);assert.equal(oldGate(owner,a,4,1,0),-1);
assert.equal(request(f,[5,owner,1,d]),-1,'table-capturing dispatcher cannot transfer');assert.equal(f.jit,old);assert.equal(old.bytes,before);
assert.equal(request(f,[5,owner,2,a,a]),-1,'duplicates cannot transfer');assert.equal(f.jit,old);
// The proof slot is intentionally populated; successful compaction must erase it.
const oldProof=old.table.get(a/8+1),fn=old.blocks.get(a).run;
const replacement=request(f,[5,owner,1,a]);assert.notEqual(replacement,owner);assert.equal(f.jit.capacity,1<<20);assert.equal(f.jit.bytes,gate.length+block.length+proof.length);
assert.equal(f.jit.blocks.get(16).run,fn,'identical Wasm function transferred without compilation/instantiation');const newProof=f.jit.table.get(a/8+1);assert.notEqual(newProof,oldProof,'new owner gets a fresh checked prover');assert.equal(f.jit.table.get(d/8+1),null,'dropped-entry proof revoked');assert.equal(oldProof(0n,BigInt(a),1n,1),0,'stale exported prover revoked');assert.equal(newProof(0n,BigInt(a),1n,1),1);assert.equal(newProof(4n,BigInt(a),1n,1),0);
assert.equal(oldGate(owner,a,4,1,0),-1,'old exported gate remains revoked after private storage reuse');
assert.equal(request(f,[6,owner]),0,'stale copy cannot close new device lease');assert.equal(request(f,[6,replacement]),1);
c.f=f;const nextGate=vm.runInContext('gates.get(f.jit.owner&32767)',c);assert.equal(nextGate(replacement,16,4,1,0),0);assert.equal(new DataView(c.mainMemory.buffer).getBigUint64(4,true),1n);
assert.equal(request(f,[4,owner]),-1,'stale suspension cannot revoke replacement');assert.equal(request(f,[5,owner,0]),-1,'stale transfer cannot replace new owner');
// Genuine module bytes, including a large custom section, still consume the
// same original 4096-byte arena budget after transfer. No external code cache.
function leb(n){const a=[];do{let b=n&127;n>>>=7;a.push(b|(n?128:0))}while(n);return a}
const large=Buffer.concat([block,Buffer.from([0,...leb(2001),0]),Buffer.alloc(2000)]),small=service(),smallOwner=request(small,[0,4096],gate);
const largeEntry=request(small,[1,1,0,64],large);assert.equal(request(small,[4,smallOwner]),0);const charged=small.jit.bytes;
assert.equal(request(small,[5,smallOwner,1,largeEntry]),-1,'retained code must fit half the original cap');assert.equal(small.jit.bytes,charged);
assert.equal(proofInstances,0,'no per-entry proof Wasm instances');
console.log('PASS');`)
	cmd.Stdin = bytes.NewReader(input)
	out, err := cmd.CombinedOutput()
	if err != nil || string(out) != "PASS\n" {
		t.Fatalf("actual host transfer: %v\n%s", err, out)
	}
}
