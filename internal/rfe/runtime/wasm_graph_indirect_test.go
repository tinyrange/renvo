//go:build !renvo

package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"strconv"
	"testing"
	"time"
)

// The target comes from architectural data, not a statically assumed branch.
// Compare the graph with ordinary leaf execution for four admitted targets,
// absent targets, high-PC aliases, repeated returns and odd bounded budgets.
func TestWasmGraphIndirectTargetsRemainFullWidthAndGuarded(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node required")
	}
	for _, base := range []uint64{0, 0xffff000000000000} {
		blocks := make([]GraphBlock, 5)
		for i := range blocks {
			var b Builder
			b.MemoryInstructionProgress = true
			b.Store(0, b.Constant(base+uint64(i*4)))
			b.Checkpoint(4, 0)
			if i == 0 {
				b.Store(0, b.Load(1))
				blocks[i].Successors = [4]uint64{base + 4, base + 8, base + 12, base + 16}
				blocks[i].SuccessorCount = 4
			} else {
				count := b.Binary(Add, b.Load(2), b.Constant(uint64(i)))
				b.Store(2, count)
				b.Store(0, b.Choose(b.Binary(Less, count, b.Load(3)), b.Constant(base), b.Constant(base+20)))
				blocks[i].Successors[0], blocks[i].SuccessorCount = base, 1
			}
			b.Checkpoint(4, 1)
			blocks[i].PC, blocks[i].Instructions, blocks[i].Ops = base+uint64(i*4), 1, b.FinishMemory(4)
		}
		l := wasmLoopTestLayout()
		l.LinkSource = 2176
		graph, err := emitWasmGraph(blocks, 4, 0, l)
		if err != nil {
			t.Fatal(err)
		}
		refs := make([][]byte, len(blocks))
		for i, block := range blocks {
			refs[i], err = EmitWasmMemoryBlock(block.Ops, 4, l)
			if err != nil {
				t.Fatal(err)
			}
		}
		payload, err := json.Marshal(struct {
			Layout     WasmMemoryLayout
			Graph      []byte
			References [][]byte
			Base       string
		}{l, graph, refs, strconv.FormatUint(base, 10)})
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		cmd := exec.CommandContext(ctx, node, "-e", wasmGraphIndirectJS)
		cmd.Stdin = bytes.NewReader(payload)
		out, err := cmd.CombinedOutput()
		cancel()
		if err != nil {
			t.Fatalf("base=%x: %v\n%s", base, err, out)
		}
	}
}

const wasmGraphIndirectJS = `
const p=JSON.parse(require('node:fs').readFileSync(0,'utf8')),l=p.Layout,base=BigInt(p.Base);
const gm=new WebAssembly.Module(Buffer.from(p.Graph,'base64')),rm=p.References.map(b=>new WebAssembly.Module(Buffer.from(b,'base64')));
function run(target,budget,limit,compiled){
 const memory=new WebAssembly.Memory({initial:8}),v=new DataView(memory.buffer),s=4096,c=8192;
 const get=o=>v.getBigUint64(c+o,true),set=(o,x)=>v.setBigUint64(c+o,BigInt(x),true),gs=i=>v.getBigUint64(s+i*8,true);
 [base,target,0n,BigInt(limit)].forEach((x,i)=>v.setBigUint64(s+i*8,x,true));set(l.Linked,1);set(l.Remaining,budget);set(l.LinkSource,777);
 const fs=compiled?[new WebAssembly.Instance(gm,{env:{memory}}).exports.run]:rm.map(m=>new WebAssembly.Instance(m,{env:{memory}}).exports.run);
 if(compiled)fs[0](s,c);
 else for(let tries=0;tries<100000;tries++){
  const pc=gs(0),n=(pc-base)/4n;if(n<0n||n>4n||(pc-base)%4n)throw Error('reference invalid entry');
  fs[Number(n)](s,c);if(get(l.Status)!==0n)break;
  const next=gs(0),k=(next-base)/4n;
  if(k<0n||k>4n||(next-base)%4n||get(l.Remaining)<2n){set(l.LinkSource,pc);set(l.Status,4);break}
  set(l.Total,get(l.Total)+get(l.Retired));set(l.MemoryTotal,get(l.MemoryTotal)+get(l.MemoryRetired));set(l.Remaining,get(l.Remaining)-get(l.Retired));
 }
 set(l.Total,get(l.Total)+get(l.Retired));set(l.MemoryTotal,get(l.MemoryTotal)+get(l.MemoryRetired));set(l.Remaining,get(l.Remaining)-get(l.Retired));
 if(get(l.Total)>BigInt(budget)||get(l.Remaining)+get(l.Total)!==BigInt(budget))throw Error('budget overrun');
 return JSON.stringify({state:[0,1,2,3].map(i=>String(gs(i))),total:String(get(l.Total)),remaining:String(get(l.Remaining)),memory:String(get(l.MemoryTotal)),retired:String(get(l.Retired)),source:String(get(l.LinkSource)),status:String(get(l.Status))});
}
for(const target of [base+4n,base+8n,base+12n,base+16n,base+20n,(base+4n)^0x100000000n,base+5n])for(const budget of [5,6,7,8,9,17,65,65536])for(const limit of [1,7,100000]){
 const got=run(target,budget,limit,true),want=run(target,budget,limit,false);
 if(got!==want)throw Error('indirect mismatch '+JSON.stringify({target:String(target),budget,limit,got,want}));
}
`
