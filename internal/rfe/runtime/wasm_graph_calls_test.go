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

// Real Wasm function-call graph: a polymorphic call and two direct calls reuse
// one callee. Return targets remain architectural data, never stack guesses.
// Compare every checkpoint, memory fault and odd budget with ordinary leaves.
func TestWasmGraphSharedCalleeReturnsStayGuarded(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node required")
	}
	pcs := []uint64{0, 4, 16, 20, 24, 32, 48}
	blocks := make([]GraphBlock, len(pcs))
	for i, pc := range pcs {
		var b Builder
		b.MemoryInstructionProgress = true
		b.Store(0, b.Constant(pc))
		b.Checkpoint(8, 0)
		count := 1
		var targets []uint64
		switch pc {
		case 0:
			b.Store(1, b.Constant(4))
			b.Store(0, b.Load(3))
			targets = []uint64{16, 32}
		case 4:
			next := b.Binary(Add, b.Load(4), b.Constant(1))
			b.Store(4, next)
			b.Store(0, b.Choose(b.Binary(Less, next, b.Load(7)), b.Constant(0), b.Constant(64)))
			targets = []uint64{0}
		case 16:
			b.Store(2, b.Load(1))
			b.Store(1, b.Constant(20))
			b.Store(0, b.Constant(48))
			targets = []uint64{48}
		case 20:
			b.Store(1, b.Constant(24))
			b.Store(0, b.Constant(48))
			targets = []uint64{48}
		case 24:
			b.Store(1, b.Load(2))
			b.Store(0, b.Load(2))
			targets = []uint64{4, 20, 24}
		case 32:
			b.Store(0, b.Load(1))
			targets = []uint64{4, 20, 24}
		case 48:
			b.Store(6, b.MemoryLoad(b.Load(5), 8))
			b.Checkpoint(8, 1)
			b.Store(4, b.Binary(Add, b.Load(4), b.Constant(1)))
			b.Store(0, b.Load(1))
			targets = []uint64{4, 20, 24}
			count = 2
		}
		b.Checkpoint(8, count)
		blocks[i] = GraphBlock{PC: pc, Instructions: count, Ops: b.FinishMemory(8), SuccessorCount: len(targets)}
		copy(blocks[i].Successors[:], targets)
	}
	l := wasmLoopTestLayout()
	l.Retired, l.MemoryRetired = 2256, 2264
	graph, err := emitWasmGraph(blocks, 8, 0, l)
	if err != nil {
		t.Fatal(err)
	}
	var refs [][]byte
	var counts []int
	for _, block := range blocks {
		code, err := EmitWasmMemoryBlock(block.Ops, 8, l)
		if err != nil {
			t.Fatal(err)
		}
		refs = append(refs, code)
		counts = append(counts, block.Instructions)
	}
	payload, err := json.Marshal(struct {
		Layout     WasmMemoryLayout
		Graph      []byte
		References [][]byte
		PCs        []uint64
		Counts     []int
	}{l, graph, refs, pcs, counts})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "-e", wasmGraphSharedCallsJS)
	cmd.Stdin = bytes.NewReader(payload)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("shared calls: %v\n%s", err, out)
	}
}

const wasmGraphSharedCallsJS = `
const p=JSON.parse(require('node:fs').readFileSync(0,'utf8')),l=p.Layout,gm=new WebAssembly.Module(Buffer.from(p.Graph,'base64')),rm=p.References.map(x=>new WebAssembly.Module(Buffer.from(x,'base64'))),index=pc=>p.PCs.findIndex(x=>BigInt(x)===pc);
function run(compiled,target,budget,limit,permissions,present){
 const memory=new WebAssembly.Memory({initial:8}),v=new DataView(memory.buffer),s=4096,c=32768,data=131072;
 const get=o=>v.getBigUint64(c+o,true),set=(o,x)=>v.setBigUint64(c+o,BigInt(x),true),gs=i=>v.getBigUint64(s+8*i,true);
 [0n,999n,888n,target,0n,4096n,777n,BigInt(limit)].forEach((x,i)=>v.setBigUint64(s+8*i,x,true));set(l.Linked,1);set(l.Remaining,budget);
 v.setBigUint64(data,0x123456789abcdefn,true);
 const d=c+l.Pages+l.PageSize;v.setBigUint64(d+l.PageNumber,1n,true);v.setBigUint64(d+l.PagePermissions,BigInt(permissions),true);if(present)v.setUint32(d+l.PageData,data,true);
 const fs=compiled?[new WebAssembly.Instance(gm,{env:{memory}}).exports.run]:rm.map(m=>new WebAssembly.Instance(m,{env:{memory}}).exports.run);
 if(compiled)fs[0](s,c);
 else for(let steps=0;steps<100000;steps++){
  const i=index(gs(0));if(i<0)throw Error('invalid reference entry');fs[i](s,c);if(get(l.Status)!==0n)break;
  const next=index(gs(0));if(next<0||get(l.Remaining)<get(l.Retired)+BigInt(p.Counts[next])){set(l.Status,4);break}
  set(l.Total,get(l.Total)+get(l.Retired));set(l.MemoryTotal,get(l.MemoryTotal)+get(l.MemoryRetired));set(l.Remaining,get(l.Remaining)-get(l.Retired));
 }
 set(l.Total,get(l.Total)+get(l.Retired));set(l.MemoryTotal,get(l.MemoryTotal)+get(l.MemoryRetired));set(l.Remaining,get(l.Remaining)-get(l.Retired));
 if(get(l.Total)>BigInt(budget)||get(l.Total)+get(l.Remaining)!==BigInt(budget))throw Error('budget overrun');
 return JSON.stringify({state:Array.from({length:8},(_,i)=>String(gs(i))),context:[l.Total,l.MemoryTotal,l.Remaining,l.Retired,l.MemoryRetired,l.Status,l.Address].map(o=>String(get(o)))});
}
for(const target of [16n,32n,0x100000010n,64n,17n])for(const budget of [8,9,17,65,65536])for(const limit of [1,7,1000000])for(const permissions of [0,1,2,3,7])for(const present of [false,true]){
 const got=run(true,target,budget,limit,permissions,present),want=run(false,target,budget,limit,permissions,present);
 if(got!==want)throw Error('call return mismatch '+JSON.stringify({target:String(target),budget,limit,permissions,present,got,want}));
}
`
