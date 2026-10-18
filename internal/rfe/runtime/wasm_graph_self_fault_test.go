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

// Exercise both arms of a diamond, its join and backedge using actual Wasm.
// The reference executes independently emitted ordinary leaves, not the graph
// scheduler or its per-edge counters. Faults cover an unexecuted alternate arm,
// a committed prefix, denied/watched stores, and a later loop iteration.
func TestWasmSelfGraphBranchesBudgetsAndFaults(t *testing.T) {
	t.Run("ordinary-accesses", func(t *testing.T) { testWasmSelfGraphBranchesBudgetsAndFaults(t, false, false) })
	t.Run("distinct-span-scratch", func(t *testing.T) { testWasmSelfGraphBranchesBudgetsAndFaults(t, true, false) })
	t.Run("refill-observes-exact-counters", func(t *testing.T) { testWasmSelfGraphBranchesBudgetsAndFaults(t, true, true) })
}

func testWasmSelfGraphBranchesBudgetsAndFaults(t *testing.T, paired, refill bool) {
	testWasmSelfGraphFaultDimensions(t, paired, refill, 4)
}
func TestWasmLargeSelfGraphBranchesBudgetsAndFaults(t *testing.T) {
	t.Run("ordinary", func(t *testing.T) { testWasmSelfGraphFaultDimensions(t, false, false, 36) })
	t.Run("span-refill", func(t *testing.T) { testWasmSelfGraphFaultDimensions(t, true, true, 36) })
}
func testWasmSelfGraphFaultDimensions(t *testing.T, paired, refill bool, count int) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node required for generated Wasm")
	}
	blocks := make([]GraphBlock, count)
	for i := range blocks {
		var b Builder
		b.MemoryInstructionProgress = true
		b.Store(0, b.Constant(uint64(i*4)))
		b.Checkpoint(6, 0)
		switch i {
		case 0:
			count := b.Binary(Add, b.Load(1), b.Constant(1))
			b.Store(1, count)
			b.Checkpoint(6, 1)
			odd := b.Binary(And, count, b.Constant(1))
			b.Store(0, b.Choose(odd, b.Constant(4), b.Constant(8)))
			blocks[i].Successors = [4]uint64{4, 8}
			blocks[i].SuccessorCount = 2
		case 1:
			b.Store(3, b.Binary(Add, b.Load(3), b.Constant(7)))
			b.Checkpoint(6, 1)
			b.MemoryStore(b.Load(2), b.Load(3), 8)
			if paired {
				b.MemoryStore(b.Binary(Add, b.Load(2), b.Constant(8)), b.Binary(Add, b.Load(3), b.Constant(1)), 8)
			}
			b.Store(0, b.Choose(b.Binary(Less, b.Load(3), b.Constant(30)), b.Constant(uint64(i*4)), b.Constant(12)))
			blocks[i].Successors = [4]uint64{uint64(i * 4), 12}
			blocks[i].SuccessorCount = 2
		case 2:
			b.Store(3, b.Binary(Sub, b.Load(3), b.Constant(2)))
			b.Checkpoint(6, 1)
			if paired {
				address := b.Load(2)
				if refill {
					address = b.Binary(Add, address, b.Constant(4096))
				}
				a := b.MemoryLoad(address, 8)
				z := b.MemoryLoad(b.Binary(Add, address, b.Constant(8)), 8)
				b.Store(4, b.Binary(Xor, a, z))
			} else {
				b.Store(4, b.MemoryLoad(b.Binary(Add, b.Load(2), b.Constant(4096)), 8))
			}
			b.Store(0, b.Choose(b.Binary(Less, b.Load(3), b.Constant(30)), b.Constant(uint64(i*4)), b.Constant(12)))
			blocks[i].Successors = [4]uint64{uint64(i * 4), 12}
			blocks[i].SuccessorCount = 2
		case 3:
			b.Store(4, b.Elapsed(b.Constant(17), 3, 1))
			b.Store(2, b.Binary(Add, b.Load(2), b.Constant(8)))
			b.Checkpoint(6, 1)
			b.Store(0, b.Choose(b.Binary(Less, b.Load(1), b.Load(5)), b.Constant(0), b.Constant(16)))
			blocks[i].Successors[0] = 0
			blocks[i].SuccessorCount = 1
		}
		b.Checkpoint(6, 2)
		blocks[i].PC, blocks[i].Instructions, blocks[i].Ops = uint64(i*4), 2, b.FinishMemory(6)
	}
	layout := wasmLoopTestLayout()
	if count >= 2 {
		layout.Retired, layout.MemoryRetired = 2256, 2264
		if !wasmGraphSessionSafe(layout) {
			t.Fatal("large fixture must exercise private prefixes")
		}
	}
	if refill {
		layout.ServiceSP, layout.ServiceFP, layout.ServiceWrite = 2176, 2180, 2184
	}
	graph, err := emitWasmGraph(blocks, 6, 0, layout)
	if err != nil {
		t.Fatal(err)
	}
	references := make([][]byte, len(blocks))
	for i := range blocks {
		references[i], err = EmitWasmMemoryBlock(blocks[i].Ops, 6, layout)
		if err != nil {
			t.Fatal(err)
		}
	}
	payload, err := json.Marshal(struct {
		Layout     WasmMemoryLayout
		Graph      []byte
		References [][]byte
		Refill     bool
	}{layout, graph, references, refill})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "-e", wasmSelfGraphTestJS)
	cmd.Stdin = bytes.NewReader(payload)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated graph: %v\n%s", err, output)
	}
}

const wasmSelfGraphTestJS = `
const p=JSON.parse(require('node:fs').readFileSync(0,'utf8')),l=p.Layout;
const graph=new WebAssembly.Module(Buffer.from(p.Graph,'base64'));
const refs=p.References.map(b=>new WebAssembly.Module(Buffer.from(b,'base64')));
function run(o,compiled){
 const memory=new WebAssembly.Memory({initial:8}),v=new DataView(memory.buffer),state=4096,ctx=8192,data=131072,clock=98304,epoch=98312;
 const get=o=>v.getBigUint64(ctx+o,true),set=(o,x)=>v.setBigUint64(ctx+o,BigInt(x),true);
 const gs=i=>v.getBigUint64(state+i*8,true);
 const init=[0,o.start,4096,0,99,o.limit];init.forEach((x,i)=>v.setBigUint64(state+i*8,BigInt(x),true));
 v.setBigUint64(clock,5n,true);v.setBigUint64(epoch,3n,true);v.setUint32(ctx+l.Clock,clock,true);
 for(let i=0;i<8192;i++)v.setUint8(data+i,(i*37+129)&255);
 for(let n=1;n<=2;n++){if(n===2&&!o.second)continue;const a=ctx+l.Pages+n*l.PageSize;v.setBigUint64(a+l.PageNumber,BigInt(n),true);v.setUint32(a+l.PageData,data+(n-1)*4096,true);v.setBigUint64(a+l.PagePermissions,BigInt(o.permissions),true);v.setUint32(a+l.PageEpoch,epoch,true)}
 set(l.Remaining,o.budget);set(l.Linked,1);
 const observations=[];
 if(p.Refill){v.setUint32(ctx+l.ServiceSP,1,true);v.setUint32(ctx+l.ServiceFP,2,true)}
 const service=(...args)=>{
  observations.push([l.Total,l.MemoryTotal,l.Remaining,l.Retired,l.MemoryRetired,l.Status,l.Address].map(x=>String(get(x))));
  const address=get(l.Address),a=ctx+l.Pages+2*l.PageSize;
  if(address>=8192n&&address<12288n&&!v.getUint32(a+l.PageData,true)){
   v.setBigUint64(a+l.PageNumber,2n,true);v.setUint32(a+l.PageData,data+4096,true);v.setBigUint64(a+l.PagePermissions,BigInt(o.permissions),true);v.setUint32(a+l.PageEpoch,epoch,true);
   // Replace the first block's previously proved mapping during another block's refill.
   const first=ctx+l.Pages+l.PageSize;v.setUint32(first+l.PageData,data+16384,true);
   return [1,0,0];
  }
  return [0,0,0];
 };
 const funcs=compiled?[new WebAssembly.Instance(graph,{env:{memory,service}}).exports.run]:refs.map(m=>new WebAssembly.Instance(m,{env:{memory,service}}).exports.run);
 if(compiled)funcs[0](state,ctx);
 else {
  for(let attempts=0;attempts<100000;attempts++){
   const pc=gs(0),index=Number(pc/4n);if(index<0||index>=4)throw Error('reference bad pc');
   funcs[index](state,ctx);if(get(l.Status)!==0n)break;
   const next=gs(0)/4n;
   if(next>=4n||get(l.Remaining)<4n){set(l.Status,4);break}
   set(l.Total,get(l.Total)+get(l.Retired));set(l.MemoryTotal,get(l.MemoryTotal)+get(l.MemoryRetired));set(l.Remaining,get(l.Remaining)-get(l.Retired));
  }
 }
 // The ordinary caller consumes only the last leaf's precise prefix.
 set(l.Total,get(l.Total)+get(l.Retired));set(l.MemoryTotal,get(l.MemoryTotal)+get(l.MemoryRetired));set(l.Remaining,get(l.Remaining)-get(l.Retired));
 return {observations,state:init.map((_,i)=>String(gs(i))),total:String(get(l.Total)),memory:String(get(l.MemoryTotal)),remaining:String(get(l.Remaining)),status:String(get(l.Status)),address:String(get(l.Address)),retired:String(get(l.Retired)),memoryRetired:String(get(l.MemoryRetired)),data:Buffer.from(memory.buffer,data,8192).toString('base64'),replacement:Buffer.from(memory.buffer,data+16384,4096).toString('base64')};
}
for(const start of [0,1])for(const budget of [8,9,10,11,16,17,64,65536])for(const permissions of [1,2,3,7])for(const second of [false,true])for(const limit of [2,7,10000]){
 if(budget<refs.length*2)continue;
 const o={start,budget,permissions,second,limit},got=run(o,true),want=run(o,false);
 if(JSON.stringify(got)!==JSON.stringify(want))throw Error('graph/reference mismatch '+JSON.stringify({o,got:{...got,data:got.data.slice(0,32)},want:{...want,data:want.data.slice(0,32)}}));
 if(BigInt(got.total)>BigInt(budget)||BigInt(got.remaining)+BigInt(got.total)!==BigInt(budget))throw Error('graph budget overrun');
}
`
