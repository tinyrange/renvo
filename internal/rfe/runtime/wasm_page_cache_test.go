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

// A later block's successful refill replaces/revokes an earlier cached mapping.
// The refilled page is read-only, so it cannot itself replace the cached RW
// entry. Removing callback invalidation must expose stale data or a bad store.
func TestWasmPrivatePageCacheRefillRevokesMappings(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node required")
	}
	l := wasmLoopTestLayout()
	l.ServiceSP, l.ServiceFP, l.ServiceWrite = 2176, 2180, 2184
	type fixture struct {
		Graph  []byte
		Leaves [][]byte
	}
	fixtures := []fixture{}
	for _, highAlias := range []bool{false, true} {
		blocks := make([]GraphBlock, 3)
		for i := range blocks {
			var b Builder
			b.MemoryInstructionProgress = true
			b.Store(0, b.Constant(uint64(i*4)))
			b.Checkpoint(4, 0)
			address := uint64(4096)
			if i == 1 {
				address = 8192
			}
			if i == 2 && highAlias {
				address |= 1 << 32
			}
			value := b.MemoryLoad(b.Constant(address), 8)
			b.Store(1+i, value)
			b.Checkpoint(4, 1)
			count := 1
			if i != 1 {
				b.MemoryStore(b.Constant(address), b.Binary(Add, value, b.Constant(1)), 8)
				count++
				b.Checkpoint(4, count)
			}
			b.Store(0, b.Constant(uint64((i+1)*4)))
			b.Checkpoint(4, count)
			blocks[i] = GraphBlock{PC: uint64(i * 4), Instructions: count, Ops: b.FinishMemory(4)}
			if i < 2 {
				blocks[i].Successors[0] = uint64((i + 1) * 4)
				blocks[i].SuccessorCount = 1
			}
		}
		graph, err := emitWasmGraph(blocks, 4, 0, l)
		if err != nil {
			t.Fatal(err)
		}
		f := fixture{Graph: graph}
		for _, block := range blocks {
			leaf, err := EmitWasmMemoryBlock(block.Ops, 4, l)
			if err != nil {
				t.Fatal(err)
			}
			f.Leaves = append(f.Leaves, leaf)
		}
		fixtures = append(fixtures, f)
	}
	input, err := json.Marshal(struct {
		Layout   WasmMemoryLayout
		Fixtures []fixture
	}{l, fixtures})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "-e", wasmPrivatePageCacheJS)
	cmd.Stdin = bytes.NewReader(input)
	out, err := cmd.CombinedOutput()
	if err != nil || string(out) != "PASS\n" {
		t.Fatalf("private mapping revocation: %v\n%s", err, out)
	}
}

const wasmPrivatePageCacheJS = `
const p=JSON.parse(require('node:fs').readFileSync(0,'utf8')),l=p.Layout,mod=b=>new WebAssembly.Module(Buffer.from(b,'base64'));
for(const f of p.Fixtures){
 const graph=mod(f.Graph),leaves=f.Leaves.map(mod);
 function run(compiled,mode,budget){
  const memory=new WebAssembly.Memory({initial:8}),v=new DataView(memory.buffer),state=4096,ctx=32768,data=131072,replacement=147456,clock=98304,epoch=98312;
  const set=(o,x)=>v.setBigUint64(ctx+o,BigInt(x),true),get=o=>v.getBigUint64(ctx+o,true),gs=i=>v.getBigUint64(state+8*i,true);
  [0n,99n,100n,101n].forEach((x,i)=>v.setBigUint64(state+8*i,x,true));
  v.setBigUint64(data,1001n,true);v.setBigUint64(data+4096,2002n,true);v.setBigUint64(replacement,3003n,true);
  v.setBigUint64(clock,5n,true);v.setBigUint64(epoch,3n,true);v.setUint32(ctx+l.Clock,clock,true);
  const page=(n,ptr,permissions)=>{const a=ctx+l.Pages+n*l.PageSize;v.setBigUint64(a+l.PageNumber,BigInt(n),true);v.setUint32(a+l.PageData,ptr,true);v.setBigUint64(a+l.PagePermissions,BigInt(permissions),true);v.setUint32(a+l.PageEpoch,epoch,true)};
  page(1,data,3);set(l.Linked,1);set(l.Remaining,budget);v.setUint32(ctx+l.ServiceSP,1,true);v.setUint32(ctx+l.ServiceFP,2,true);
  const observations=[];
  const service=(...args)=>{
   observations.push([l.Total,l.MemoryTotal,l.Remaining,l.Retired,l.MemoryRetired,l.Status,l.Address].map(o=>String(get(o))));
   if(get(l.Address)!==8192n||v.getUint32(ctx+l.Pages+2*l.PageSize+l.PageData,true)!==0)return [0,0,0];
   page(2,data+4096,1);page(1,mode==='missing'?0:replacement,mode==='readonly'?1:mode==='watched'?7:mode==='denied'?0:3);
   if(mode==='tag')v.setBigUint64(ctx+l.Pages+l.PageSize+l.PageNumber,0x100001n,true);
   return [1,0,0];
  };
  const funcs=compiled?[new WebAssembly.Instance(graph,{env:{memory,service}}).exports.run]:leaves.map(m=>new WebAssembly.Instance(m,{env:{memory,service}}).exports.run);
  if(compiled)funcs[0](state,ctx);
  else for(let step=0;step<3;step++){
   funcs[Number(gs(0)/4n)](state,ctx);if(get(l.Status)!==0n)break;
   const next=gs(0);if(next===12n||get(l.Remaining)<get(l.Retired)+(next===4n?1n:2n)){set(l.Status,4);break}
   set(l.Total,get(l.Total)+get(l.Retired));set(l.MemoryTotal,get(l.MemoryTotal)+get(l.MemoryRetired));set(l.Remaining,get(l.Remaining)-get(l.Retired));
  }
  set(l.Total,get(l.Total)+get(l.Retired));set(l.MemoryTotal,get(l.MemoryTotal)+get(l.MemoryRetired));set(l.Remaining,get(l.Remaining)-get(l.Retired));
  return JSON.stringify({state:[0,1,2,3].map(i=>String(gs(i))),context:[l.Retired,l.MemoryRetired,l.Status,l.Address,l.Total,l.MemoryTotal,l.Remaining].map(o=>String(get(o))),old:String(v.getBigUint64(data,true)),replacement:String(v.getBigUint64(replacement,true)),observations});
 }
 for(const mode of ['rw','readonly','watched','denied','missing','tag'])for(const budget of [5,6,7,65536]){const got=run(true,mode,budget),want=run(false,mode,budget);if(got!==want)throw Error('mapping '+JSON.stringify({mode,budget,got,want}))}
}
console.log('PASS');
`
