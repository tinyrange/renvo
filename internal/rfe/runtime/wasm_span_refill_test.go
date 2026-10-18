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

// A page admitted earlier in a loop is not authority after a host callback.
// Exercise actual generated Wasm against the ordinary checked memory emitter.
func TestWasmOwnedSpanRefillRevokesPriorMapping(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node required to execute generated Wasm")
	}
	layout := wasmLoopTestLayout()
	layout.ServiceSP, layout.ServiceFP, layout.ServiceWrite = 2176, 2180, 2184
	type trial struct {
		Write, MiddleRefill bool
		Resident, Reference []byte
	}
	var trials []trial
	for _, shape := range []struct{ write, middle bool }{{false, false}, {true, false}, {false, true}, {true, true}} {
		write, middle := shape.write, shape.middle
		var b Builder
		b.MemoryInstructionProgress = true
		b.Checkpoint(6, 0)
		address := b.Load(2)
		first := b.MemoryLoad(address, 8)
		b.Store(4, first)
		b.Checkpoint(6, 1)
		if middle {
			b.MemoryLoad(b.Load(3), 8)
			b.Checkpoint(6, 2)
		}
		next := b.Binary(Add, address, b.Constant(8))
		if write {
			b.MemoryStore(next, b.Binary(Add, first, b.Constant(1)), 8)
		} else {
			b.Store(5, b.MemoryLoad(next, 8))
		}
		if !middle {
			b.Checkpoint(6, 2)
			b.MemoryLoad(b.Load(3), 8)
		}
		count := b.Binary(Add, b.Load(1), b.Constant(1))
		b.Store(1, count)
		more := b.Binary(Less, count, b.Constant(3))
		b.Store(0, b.Choose(more, b.Constant(0), b.Constant(4)))
		b.Checkpoint(6, 3)
		b.LoopContinue(more)
		ops := b.FinishMemory(6)
		resident, err := emitWasmCode(ops, 6, &layout, nil, 3, true)
		if err != nil {
			t.Fatal(err)
		}
		reference, err := emitWasmCode(ops, 6, &layout, nil, 3, false)
		if err != nil {
			t.Fatal(err)
		}
		trials = append(trials, trial{write, middle, resident, reference})
	}
	dispatcher, err := EmitWasmDispatcher(6, 0, layout)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(struct {
		Layout            WasmMemoryLayout
		Trials            []trial
		Dispatcher, Proof []byte
	}{layout, trials, dispatcher, emitWasmProof()})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "-e", `
const x=JSON.parse(require('node:fs').readFileSync(0,'utf8')),l=x.Layout;
const code=s=>Buffer.from(s,'base64');
function run(t,mode,resident) {
 const memory=new WebAssembly.Memory({initial:1}),v=new DataView(memory.buffer),context=1024,state=8192,first=16384,second=20480,replaced=24576;
 const put=(o,n)=>v.setBigUint64(context+o,BigInt(n),true),get=o=>v.getBigUint64(context+o,true);
 const page=(n,data,permissions,tag=BigInt(n))=>{const at=context+l.Pages+n*l.PageSize;v.setBigUint64(at+l.PageNumber,tag,true);v.setUint32(at+l.PageData,data,true);v.setBigUint64(at+l.PagePermissions,BigInt(permissions),true);v.setUint32(at+l.PageEpoch,6016+n*8,true)};
 page(1,first,3);v.setUint32(context+l.Clock,6000,true);v.setBigUint64(6000,9n,true);
 v.setBigUint64(first,10n,true);v.setBigUint64(first+8,20n,true);v.setBigUint64(second,30n,true);v.setBigUint64(replaced,70n,true);v.setBigUint64(replaced+8,80n,true);
 [0n,0n,4096n,8192n,99n,98n].forEach((n,i)=>v.setBigUint64(state+i*8,n,true));
 put(l.Linked,1);put(l.Remaining,20);v.setUint32(context+l.ServiceSP,1,true);v.setUint32(context+l.ServiceFP,2,true);
 const observations=[];
 const service=(...args)=>{
  observations.push([get(l.Total),get(l.MemoryTotal),get(l.Retired),get(l.MemoryRetired),get(l.Address)].map(String));
  if(get(l.Address)!==8192n)return [0,0,0];
  if(mode==='remap')page(1,replaced,3);
  if(mode==='missing-data')page(1,0,3);
  if(mode==='high-tag')page(1,first,3,0x100000001n);
  if(mode==='no-permissions')page(1,first,0);
  if(mode==='readonly')page(1,first,1);
  page(2,second,3);return [1,0,0];
 };
 const leaf=new WebAssembly.Instance(new WebAssembly.Module(code(resident?t.Resident:t.Reference)),{env:{memory,service}});
 const table=new WebAssembly.Table({element:'anyfunc',initial:8});table.set(2,leaf.exports.run);
 const immutable=(value,type='i64')=>new WebAssembly.Global({value:type,mutable:false},value);
 const proof=new WebAssembly.Instance(new WebAssembly.Module(code(x.Proof)),{proof:{pc:immutable(0n),entry:immutable(16n),instructions:immutable(3n),words:immutable(6,'i32')}});
 table.set(3,proof.exports.prove);
 const descriptor=context+l.Blocks;v.setBigUint64(descriptor+l.DescriptorPC,0n,true);v.setBigUint64(descriptor+l.DescriptorEntry,16n,true);v.setBigUint64(descriptor+l.DescriptorInstructions,3n,true);
 const dispatcher=new WebAssembly.Instance(new WebAssembly.Module(code(x.Dispatcher)),{env:{memory,table}});
 dispatcher.exports.run(state,context);
 return {state:Array.from({length:6},(_,i)=>String(v.getBigUint64(state+i*8,true))),total:String(get(l.Total)),remaining:String(get(l.Remaining)),memory:String(get(l.MemoryTotal)),status:String(get(l.Status)),address:String(get(l.Address)),retired:String(get(l.Retired)),memoryRetired:String(get(l.MemoryRetired)),iterations:String(get(l.LoopIterations)),observations,data:Buffer.from(memory.buffer,first,12288).toString('base64')};
}
for(const t of x.Trials)for(const mode of ['remap','missing-data','high-tag','no-permissions','readonly']) {
 const actual=run(t,mode,true),reference=run(t,mode,false);
 if(JSON.stringify(actual)!==JSON.stringify(reference))throw Error(JSON.stringify({write:t.Write,mode,actual:{...actual,data:undefined},reference:{...reference,data:undefined},dataEqual:actual.data===reference.data}));
 if(mode==='readonly'&&!t.Write||mode==='remap') { if(actual.total!=='9'||actual.status!=='0'||actual.state[1]!=='3')throw Error('successful remapped loop accounting'); }
 else { const address=t.MiddleRefill||mode==='readonly'?'4104':'4096',total=t.MiddleRefill?'2':mode==='readonly'?'4':'3'; if(actual.status!=='1'||actual.address!==address||actual.total!==total)throw Error('revocation precise prefix '+JSON.stringify({t,mode,actual})); }
}
console.log('PASS');`)
	cmd.Stdin = bytes.NewReader(data)
	output, err := cmd.CombinedOutput()
	if err != nil || string(output) != "PASS\n" {
		t.Fatalf("actual Wasm refill mapping revocation: %v\n%s", err, output)
	}
}
