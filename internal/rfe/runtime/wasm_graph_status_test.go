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

// Refill failure changes Status to a non-default value. Compare complete
// context and state bytes against ordinary leaves, including a non-root cycle
// whose header index shares the private Status local between blocks.
func TestWasmGraphStatusReloadAfterFailedService(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node required")
	}
	l := wasmLoopTestLayout()
	l.Retired, l.MemoryRetired = 2256, 2264
	l.ServiceSP, l.ServiceFP, l.ServiceWrite = 2176, 2180, 2184
	if !wasmGraphStatusSafe(l) {
		t.Fatal("fixture must carry Status")
	}
	blocks := make([]GraphBlock, 4)
	refs := make([][]byte, len(blocks))
	for i := range blocks {
		var b Builder
		b.Store(0, b.Constant(uint64(i*4)))
		b.Checkpoint(3, 0)
		b.Store(1, b.Binary(Add, b.Load(1), b.Constant(17)))
		b.Store(2, b.MemoryLoad(b.Constant(uint64((i+1)*4096)), 8))
		next := uint64((i + 1) * 4)
		if i == 3 {
			next = 4
		}
		b.Store(0, b.Constant(next))
		b.Checkpoint(3, 1)
		blocks[i] = GraphBlock{PC: uint64(i * 4), Instructions: 1, Ops: b.FinishMemory(3), Successors: [4]uint64{next}, SuccessorCount: 1}
		refs[i], err = EmitWasmMemoryBlock(blocks[i].Ops, 3, l)
		if err != nil {
			t.Fatal(err)
		}
	}
	graph, err := emitWasmGraph(blocks, 3, 0, l)
	if err != nil {
		t.Fatal(err)
	}
	input, err := json.Marshal(struct {
		Layout WasmMemoryLayout
		Graph  []byte
		Refs   [][]byte
	}{l, graph, refs})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "-e", `
const p=JSON.parse(require('node:fs').readFileSync(0,'utf8')),l=p.Layout;
const graph=new WebAssembly.Module(Buffer.from(p.Graph,'base64')),refs=p.Refs.map(x=>new WebAssembly.Module(Buffer.from(x,'base64')));
function run(compiled,failAt){
 const memory=new WebAssembly.Memory({initial:8}),v=new DataView(memory.buffer),state=4096,ctx=32768,data=131072;
 const get=o=>v.getBigUint64(ctx+o,true),set=(o,x)=>v.setBigUint64(ctx+o,BigInt(x),true);
 set(l.Linked,1);set(l.Remaining,65536);v.setUint32(ctx+l.ServiceSP,1,true);v.setUint32(ctx+l.ServiceFP,2,true);
 let calls=0;const observations=[];
 const service=()=>{calls++;observations.push([l.Total,l.MemoryTotal,l.Remaining,l.Retired,l.MemoryRetired,l.Status,l.Address].map(o=>String(get(o))));
  if(calls===failAt){set(l.Status,9);return [0,0,0]}
  const n=Number(get(l.Address)>>12n),a=ctx+l.Pages+n*l.PageSize;
  v.setBigUint64(a+l.PageNumber,BigInt(n),true);v.setUint32(a+l.PageData,data+(n-1)*4096,true);v.setBigUint64(a+l.PagePermissions,1n,true);return [1,0,0];
 };
 const env={memory,service};
 if(compiled)new WebAssembly.Instance(graph,{env}).exports.run(state,ctx);
 else {const runs=refs.map(m=>new WebAssembly.Instance(m,{env}).exports.run);for(let steps=0;steps<65536;steps++){
  const pc=Number(v.getBigUint64(state,true));runs[pc/4](state,ctx);
  if(get(l.Status)!==0n)break;
  set(l.Total,get(l.Total)+get(l.Retired));set(l.MemoryTotal,get(l.MemoryTotal)+get(l.MemoryRetired));set(l.Remaining,get(l.Remaining)-get(l.Retired));
 }}
 if(get(l.Status)!==9n||calls!==failAt)throw Error('callback Status lost '+get(l.Status));
 return {state:Array.from(new Uint8Array(memory.buffer,state,24)),context:Array.from(new Uint8Array(memory.buffer,ctx,2400)),observations};
}
for(const failAt of [1,2,3,4]){const a=run(true,failAt),b=run(false,failAt);if(JSON.stringify(a)!==JSON.stringify(b))throw Error('status/publication mismatch '+failAt+' '+JSON.stringify([a.observations,b.observations]));}
console.log('PASS');`)
	cmd.Stdin = bytes.NewReader(input)
	out, err := cmd.CombinedOutput()
	if err != nil || string(out) != "PASS\n" {
		t.Fatalf("graph Status: %v\n%s", err, out)
	}
}
