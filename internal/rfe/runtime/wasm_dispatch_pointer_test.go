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

// All four colliding ways retain full-width admission while descriptor pointers
// use Wasm32 address arithmetic. Exercise both inline and indirect block arrays.
func TestWasmDispatcherPointersAndFullWidthAdmissions(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node required to execute generated Wasm")
	}
	type fixture struct {
		Layout     WasmMemoryLayout
		Dispatcher []byte
	}
	var fixtures []fixture
	for i := 0; i < 5; i++ {
		l := wasmLoopTestLayout()
		l.LinkSource = 2176
		if i == 1 {
			l.BlockSets = 1024
			l.BlocksIndirect = true
		}
		if i == 2 {
			l.BlockSets = 4096
			l.BlocksIndirect = true
		}
		if i == 3 {
			l.Blocks = 0xfffff000
		}
		if i == 4 {
			l.BlockSets = 4096
		}
		code, err := EmitWasmDispatcher(2, 0, l)
		if err != nil {
			t.Fatal(err)
		}
		fixtures = append(fixtures, fixture{l, code})
	}
	pcs := [4]uint64{0xffff8000000003f0, 0xffff8001000003f0, 0xffff8002000003f0, 0xffff8003000003f0}
	var blocks [][]byte
	for i := 0; i < 4; i++ {
		var b Builder
		b.Store(1, b.Binary(Add, b.Load(1), b.Constant(1)))
		b.Store(0, b.Constant(pcs[(i+1)%4]))
		code, err := EmitWasmBlock(b.Finish(2), 2)
		if err != nil {
			t.Fatal(err)
		}
		blocks = append(blocks, code)
	}
	payload, err := json.Marshal(struct {
		Fixtures []fixture
		Blocks   [][]byte
		Proof    []byte
	}{fixtures, blocks, emitWasmProof()})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "-e", wasmDispatchPointerJS)
	cmd.Stdin = bytes.NewReader(payload)
	out, err := cmd.CombinedOutput()
	if err != nil || string(out) != "PASS\n" {
		t.Fatalf("dispatcher pointers: %v\n%s", err, out)
	}
}

const wasmDispatchPointerJS = `
const p=JSON.parse(require('node:fs').readFileSync(0,'utf8')),module=b=>new WebAssembly.Module(Buffer.from(b,'base64'));
const pcs=Array.from({length:4},(_,i)=>0xffff8000000003f0n+(BigInt(i)<<32n)),proof=module(p.Proof),blocks=p.Blocks.map(module);
for(const f of p.Fixtures){
 const l=f.Layout,dispatch=module(f.Dispatcher),sets=l.BlockSets||256;
 for(const mode of ['cycle','high-entry','high-count','count257','forged-count','wrong-tag','wrong-words','missing-proof']){
  for(const budget of mode==='cycle'?[0,1,2,3,4,7,65,65536]:[65536]){
   const memory=new WebAssembly.Memory({initial:64}),v=new DataView(memory.buffer),state=4096,context=32768;
   const table=new WebAssembly.Table({element:'anyfunc',initial:16});
   const root=l.BlocksIndirect?1048576:((context+l.Blocks)>>>0);
   if(l.BlocksIndirect)v.setUint32(context+l.Blocks,root,true);
   const desc=(pc,way)=>((root+(Number(((pc>>2n)^(pc>>12n))&BigInt(sets-1))+way*sets)*l.DescriptorSize)>>>0);
   const set=(o,x)=>v.setBigUint64(context+o,BigInt(x),true),get=o=>v.getBigUint64(context+o,true);
   v.setBigUint64(state,pcs[0],true);set(l.Remaining,budget);set(l.LinkSource,123);
   for(let i=0;i<4;i++){
    const entry=BigInt((i+1)*16),d=desc(pcs[i],i);
    table.set(2*(i+1),new WebAssembly.Instance(blocks[i],{env:{memory}}).exports.run);
    if(!(mode==='missing-proof'&&i===0))table.set(2*(i+1)+1,new WebAssembly.Instance(proof,{proof:{pc:pcs[i],entry,instructions:1n,words:mode==='wrong-words'&&i===0?1:2}}).exports.prove);
    v.setBigUint64(d+l.DescriptorPC,pcs[i],true);v.setBigUint64(d+l.DescriptorEntry,entry,true);v.setBigUint64(d+l.DescriptorInstructions,1n,true);
   }
   const d=desc(pcs[0],0);
   if(mode==='high-entry')v.setBigUint64(d+l.DescriptorEntry,0x100000010n,true);
   if(mode==='high-count')v.setBigUint64(d+l.DescriptorInstructions,0x100000001n,true);
   if(mode==='count257')v.setBigUint64(d+l.DescriptorInstructions,257n,true);
   if(mode==='forged-count')v.setBigUint64(d+l.DescriptorInstructions,2n,true);
   if(mode==='wrong-tag')v.setBigUint64(d+l.DescriptorPC,pcs[0]^(1n<<32n),true);
   new WebAssembly.Instance(dispatch,{env:{memory,table}}).exports.run(state,context);
   const n=mode==='cycle'?BigInt(budget):0n,status=['high-entry','count257','forged-count','wrong-words','missing-proof'].includes(mode)?3n:0n;
   if(v.getBigUint64(state,true)!==pcs[Number(n%4n)]||v.getBigUint64(state+8,true)!==n||get(l.Total)!==n||get(l.Remaining)!==BigInt(budget)-n||get(l.Status)!==status||get(l.LinkSource)!==(n?pcs[Number((n-1n)%4n)]:123n))throw Error('mismatch '+mode+' sets='+sets+' indirect='+l.BlocksIndirect+' budget='+budget);
  }
 }
}
console.log('PASS');
`
