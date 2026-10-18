//go:build !renvo

package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"testing"
	"time"
)

// Execute real generated indirect block-to-block calls, not a Go dispatcher
// model. Full-PC proofs, source attribution, budget and partial-prefix rejection
// must remain exact across calls and returns, including addresses above 32 bits.
func TestWasmIndirectInterlinking(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node is required to execute generated WASM")
	}
	l := wasmLoopTestLayout()
	l.LinkSource = 2176
	dispatcher, err := EmitWasmDispatcher(5, 0, l)
	if err != nil {
		t.Fatal(err)
	}
	var blocks [][]byte
	for i := 0; i < 3; i++ {
		var b Builder
		b.Store(1, b.Binary(Add, b.Load(1), b.Constant(1)))
		b.Store(0, b.Load(2+i)) // runtime-loaded call/return successor
		code, err := EmitWasmBlock(b.Finish(5), 5)
		if err != nil {
			t.Fatal(err)
		}
		blocks = append(blocks, code)
	}
	payload, err := json.Marshal(struct {
		Layout            WasmMemoryLayout
		Dispatcher, Proof []byte
		Blocks            [][]byte
	}{l, dispatcher, emitWasmProof(), blocks})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "-e", wasmInterlinkTestJS)
	cmd.Stdin = bytes.NewReader(payload)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated interlinks: %v\n%s", err, out)
	}
}

const wasmInterlinkTestJS = `
const fs=require('node:fs'),p=JSON.parse(fs.readFileSync(0,'utf8')),l=p.Layout;
const module=c=>new WebAssembly.Module(Buffer.from(c,'base64'));
const dispatcher=module(p.Dispatcher),proof=module(p.Proof),blocks=p.Blocks.map(module);
const base=0xffff000000000000n,pcs=[base,base+4n,base+8n];
function run(budget,mode){
 const memory=new WebAssembly.Memory({initial:8}),v=new DataView(memory.buffer),state=4096,context=8192;
 const table=new WebAssembly.Table({initial:8,element:'anyfunc'});
 const set=(o,x)=>v.setBigUint64(context+o,BigInt(x),true),get=o=>v.getBigUint64(context+o,true);
 const reg=(i,x)=>v.setBigUint64(state+8*i,x,true);
 reg(0,base);reg(1,0n);reg(2,pcs[1]);reg(3,pcs[2]);reg(4,mode==='cycle'?base:base+12n);
 const desc=pc=>context+l.Blocks+Number(((pc>>2n)^(pc>>12n))&255n)*l.DescriptorSize;
 for(let i=0;i<3;i++){
  const entry=BigInt(16*(i+1)),leaf=new WebAssembly.Instance(blocks[i],{env:{memory}});
  const admission=new WebAssembly.Instance(proof,{proof:{pc:pcs[i],entry,instructions:1n,words:5}});
  table.set(2*(i+1),leaf.exports.run);table.set(2*(i+1)+1,admission.exports.prove);
  if(mode==='missing'&&i===1)continue;
  const d=desc(pcs[i]);v.setBigUint64(d+l.DescriptorPC,pcs[i],true);v.setBigUint64(d+l.DescriptorEntry,entry,true);v.setBigUint64(d+l.DescriptorInstructions,1n,true);
 }
 if(mode==='forged-pc')v.setBigUint64(desc(pcs[1])+l.DescriptorPC,pcs[1]-(1n<<32n),true);
 if(mode==='forged-entry')v.setBigUint64(desc(pcs[1])+l.DescriptorEntry,48n,true);
 if(mode==='forged-count')v.setBigUint64(desc(pcs[1])+l.DescriptorInstructions,2n,true);
 set(l.Remaining,budget);set(l.LinkSource,123);
 new WebAssembly.Instance(dispatcher,{env:{memory,table}}).exports.run(state,context);
 return {pc:v.getBigUint64(state,true),count:v.getBigUint64(state+8,true),total:get(l.Total),remaining:get(l.Remaining),status:get(l.Status),source:get(l.LinkSource)};
}
for(const budget of [0,1,2,3,7,65536]){
 const r=run(budget,'cycle'),n=BigInt(budget),source=budget?pcs[(budget-1)%3]:123n;
 if(r.pc!==pcs[budget%3]||r.count!==n||r.total!==n||r.remaining!==0n||r.status!==0n||r.source!==source)throw Error('cycle budget '+budget+' '+JSON.stringify(r,(_,x)=>typeof x==='bigint'?String(x):x));
}
for(const mode of ['exit','missing','forged-pc','forged-entry','forged-count']){
 const r=run(32,mode),n=mode==='exit'?3n:1n,status=['forged-entry','forged-count'].includes(mode)?3n:0n;
 if(r.count!==n||r.total!==n||r.remaining!==32n-n||r.status!==status||r.source!==(n===3n?pcs[2]:base))throw Error('unsafe edge '+mode+' '+JSON.stringify(r,(_,x)=>typeof x==='bigint'?String(x):x));
}
`
