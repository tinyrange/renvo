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

func TestWasmResidentLazyFlagsIterationsReadsAndFaults(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node required")
	}
	type fixture struct {
		Layout     WasmMemoryLayout
		Loop, Leaf []byte
	}
	fixtures := []fixture{}
	for _, width := range []int{32, 64} {
		for _, kind := range []int{0, 1, 2} {
			for _, readPrior := range []bool{false, true} {
				for _, service := range []bool{false, true} {
					l := wasmLoopTestLayout()
					if service {
						l.ServiceSP, l.ServiceFP, l.ServiceWrite = 2176, 2180, 2184
					}
					var b Builder
					b.MemoryInstructionProgress = true
					b.Checkpoint(8, 0)
					if readPrior {
						b.Store(3, b.Load(4))
					}
					count := b.Binary(Add, b.Load(1), b.Constant(1))
					b.Store(1, count)
					b.Store(0, b.Constant(0))
					flags := b.ArithmeticFlags(count, b.Load(2), width, kind == 1)
					if kind == 2 {
						flags = b.LogicalFlags(count, width)
					}
					b.Store(4, flags)
					b.Checkpoint(8, 1)
					value := b.MemoryLoad(b.Load(5), 8)
					b.Store(3, b.Binary(Xor, b.Load(3), value))
					b.Checkpoint(8, 2)
					b.MemoryStore(b.Load(5), b.Load(3), 8)
					b.Store(5, b.Binary(Add, b.Load(5), b.Constant(8)))
					b.Checkpoint(8, 3)
					condition := b.Binary(Less, count, b.Load(6))
					b.Store(0, b.Choose(condition, b.Constant(0), b.Constant(4)))
					b.Checkpoint(8, 3)
					b.LoopContinue(condition)
					ops := b.FinishMemory(8)
					loop, err := emitWasmLoopMode(ops, 8, 3, &l, true)
					if err != nil {
						t.Fatal(err)
					}
					leaf, err := EmitWasmMemoryBlock(ops, 8, l)
					if err != nil {
						t.Fatal(err)
					}
					fixtures = append(fixtures, fixture{l, loop, leaf})
				}
			}
		}
	}
	input, err := json.Marshal(struct{ Fixtures []fixture }{fixtures})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "-e", wasmLazyLoopFlagsJS)
	cmd.Stdin = bytes.NewReader(input)
	out, err := cmd.CombinedOutput()
	if err != nil || string(out) != "PASS\n" {
		t.Fatalf("resident lazy flags: %v\n%s", err, out)
	}
}

const wasmLazyLoopFlagsJS = `
const p=JSON.parse(require('node:fs').readFileSync(0,'utf8')),mod=b=>new WebAssembly.Module(Buffer.from(b,'base64'));
for(let fIndex=0;fIndex<p.Fixtures.length;fIndex++){
 const f=p.Fixtures[fIndex],l=f.Layout,loop=mod(f.Loop),leaf=mod(f.Leaf);
 function run(compiled,o){
  const memory=new WebAssembly.Memory({initial:8}),v=new DataView(memory.buffer),state=4096,ctx=32768,data=131072,clock=98304,epoch=98312;
  const set=(o,x)=>v.setBigUint64(ctx+o,BigInt(x),true),get=o=>v.getBigUint64(ctx+o,true),gs=i=>v.getBigUint64(state+8*i,true);
  const initial=[0n,o.start,o.rhs,99n,0xf0000000n,BigInt(o.address),o.start+4n,0n];initial.forEach((x,i)=>v.setBigUint64(state+8*i,x,true));
  v.setBigUint64(clock,5n,true);v.setBigUint64(epoch,3n,true);v.setUint32(ctx+l.Clock,clock,true);for(let i=0;i<4096;i++)v.setUint8(data+i,(i*19+11)&255);
  const page=()=>{const a=ctx+l.Pages+l.PageSize;v.setBigUint64(a+l.PageNumber,1n,true);v.setUint32(a+l.PageData,data,true);v.setBigUint64(a+l.PagePermissions,BigInt(o.permissions),true);v.setUint32(a+l.PageEpoch,epoch,true)};
  if(o.present)page();set(l.Linked,1);set(l.Remaining,o.budget);if(l.ServiceSP){v.setUint32(ctx+l.ServiceSP,1,true);v.setUint32(ctx+l.ServiceFP,2,true)}
  const observations=[];
  const service=(...args)=>{observations.push([l.Total,l.MemoryTotal,l.Remaining,l.Retired,l.MemoryRetired,l.Status,l.Address].map(x=>String(get(x))));const address=get(l.Address),a=ctx+l.Pages+l.PageSize;if(address>=4096n&&address<8192n&&!v.getUint32(a+l.PageData,true)){page();return [1,0,0]}return [0,0,0]};
  const execute=new WebAssembly.Instance(compiled?loop:leaf,{env:{memory,service}}).exports.run;
  if(compiled)execute(state,ctx);
  else for(let i=0;i<100;i++){
   execute(state,ctx);if(get(l.Status)!==0n||get(l.Remaining)<6n)break;
   set(l.Total,get(l.Total)+get(l.Retired));set(l.MemoryTotal,get(l.MemoryTotal)+get(l.MemoryRetired));set(l.Remaining,get(l.Remaining)-get(l.Retired));
  }
  set(l.Total,get(l.Total)+get(l.Retired));set(l.MemoryTotal,get(l.MemoryTotal)+get(l.MemoryRetired));set(l.Remaining,get(l.Remaining)-get(l.Retired));
  return JSON.stringify({state:initial.map((_,i)=>String(gs(i))),context:[l.Retired,l.MemoryRetired,l.Status,l.Address,l.Total,l.MemoryTotal,l.Remaining].map(x=>String(get(x))),observations,data:Buffer.from(memory.buffer,data,4096).toString('base64')});
 }
 for(const [start,rhs] of [[0n,0n],[0x7fffffffn,0xffffffffn],[0x8000000000000000n,0x7fffffffffffffffn]])for(const budget of [3,4,5,6,7,8,9,10,17,65536])for(const permissions of [1,2,3,7])for(const address of [4096,8184])for(const present of [false,true]){
  const o={start,rhs,budget,permissions,address,present},got=run(true,o),want=run(false,o);if(got!==want)throw Error('resident mismatch '+JSON.stringify({fIndex,o,got,want},(_,x)=>typeof x==='bigint'?String(x):x))
 }
}
console.log('PASS');
`
