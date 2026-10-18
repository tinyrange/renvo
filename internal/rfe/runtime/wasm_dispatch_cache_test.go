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

// Mutate public descriptor words AFTER the target has been admitted and cached.
// Also move an indirect descriptor array and cross a new dispatcher invocation.
// The second leaf is real Wasm, not a JS replacement for native guest execution.
func TestWasmDispatcherCachedAdmissionRevocation(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node required")
	}
	const a, b = uint64(0xffff8000000003f0), uint64(0xffff8000000003f4)
	type trial struct {
		Layout                     WasmMemoryLayout
		Mode                       string
		Dispatcher, First, Mutator []byte
	}
	var trials []trial
	for _, indirect := range []bool{false, true} {
		l := wasmLoopTestLayout()
		l.LinkSource = 2176
		l.BlocksIndirect = indirect
		dispatch, err := EmitWasmDispatcher(2, 0, l)
		if err != nil {
			t.Fatal(err)
		}
		var builder Builder
		builder.Store(1, builder.Binary(Add, builder.Load(1), builder.Constant(1)))
		builder.Store(0, builder.Constant(b))
		first, err := EmitWasmBlock(builder.Finish(2), 2)
		if err != nil {
			t.Fatal(err)
		}
		for _, mode := range []string{"entry", "count", "zero-count", "high-count", "tag", "array", "new-call"} {
			if mode == "array" && !indirect {
				continue
			}
			var w wasmCode
			w.u32(0)
			w.get(0)
			w.get(0)
			w.mem(0x29, 3, 8)
			w.i64(1)
			w.byte(0x7c)
			w.mem(0x37, 3, 8)
			if mode == "array" {
				w.get(1)
				w.i32(524288)
				w.mem(0x36, 2, l.Blocks)
			} else if mode != "new-call" {
				w.get(1)
				if indirect {
					w.mem(0x28, 2, l.Blocks)
				} else {
					w.i32(l.Blocks)
					w.byte(0x6a)
				}
				w.i32(uint32(((a>>2)^(a>>12))&255) * l.DescriptorSize)
				w.byte(0x6a)
				offset, value := l.DescriptorEntry, uint64(32)
				if mode == "count" {
					offset, value = l.DescriptorInstructions, 2
				}
				if mode == "zero-count" {
					offset, value = l.DescriptorInstructions, 0
				}
				if mode == "high-count" {
					offset, value = l.DescriptorInstructions, 1<<32|1
				}
				if mode == "tag" {
					offset, value = l.DescriptorPC, a^(1<<32)
				}
				w.i64(value)
				w.mem(0x37, 3, offset)
			}
			w.get(0)
			w.i64(a)
			w.mem(0x37, 3, 0)
			w.byte(0x0b)
			trials = append(trials, trial{l, mode, dispatch, first, wasmFunctionModule(w.code, false, false)})
		}
	}
	payload, err := json.Marshal(struct {
		Trials []trial
		Proof  []byte
	}{trials, emitWasmProof()})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "-e", wasmCachedAdmissionJS)
	cmd.Stdin = bytes.NewReader(payload)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cached admission: %v\n%s", err, out)
	}
}

const wasmCachedAdmissionJS = `
const p=JSON.parse(require('node:fs').readFileSync(0,'utf8')),mod=b=>new WebAssembly.Module(Buffer.from(b,'base64')),proof=mod(p.Proof),pcs=[0xffff8000000003f0n,0xffff8000000003f4n];
for(const t of p.Trials){
 const l=t.Layout,memory=new WebAssembly.Memory({initial:16}),v=new DataView(memory.buffer),state=4096,context=32768,table=new WebAssembly.Table({element:'anyfunc',initial:8});
 const root=l.BlocksIndirect?262144:context+l.Blocks;
 if(l.BlocksIndirect)v.setUint32(context+l.Blocks,root,true);
 const desc=pc=>root+Number(((pc>>2n)^(pc>>12n))&255n)*l.DescriptorSize;
 const get=o=>v.getBigUint64(context+o,true),set=(o,x)=>v.setBigUint64(context+o,BigInt(x),true);
 v.setBigUint64(state,pcs[0],true);
 for(let i=0;i<2;i++){
  const entry=BigInt((i+1)*16),d=desc(pcs[i]);
  table.set(2*(i+1),new WebAssembly.Instance(mod(i?t.Mutator:t.First),{env:{memory}}).exports.run);
  table.set(2*(i+1)+1,new WebAssembly.Instance(proof,{proof:{pc:pcs[i],entry,instructions:1n,words:2}}).exports.prove);
  v.setBigUint64(d+l.DescriptorPC,pcs[i],true);v.setBigUint64(d+l.DescriptorEntry,entry,true);v.setBigUint64(d+l.DescriptorInstructions,1n,true);
 }
 const run=new WebAssembly.Instance(mod(t.Dispatcher),{env:{memory,table}}).exports.run;
 set(l.Remaining,t.Mode==='new-call'?2:65536);set(l.LinkSource,123);run(state,context);
 let total=2n,status=['entry','count'].includes(t.Mode)?3n:0n,remaining=65534n,source=pcs[1];
 if(t.Mode==='new-call'){
  v.setBigUint64(desc(pcs[0])+l.DescriptorEntry,32n,true);
  set(l.Total,0);set(l.MemoryTotal,0);set(l.Status,0);set(l.Remaining,65536);set(l.LinkSource,123);run(state,context);
  total=0n;status=3n;remaining=65536n;source=123n;
 }
 if(v.getBigUint64(state,true)!==pcs[0]||v.getBigUint64(state+8,true)!==2n||get(l.Total)!==total||get(l.Status)!==status||get(l.Remaining)!==remaining||get(l.LinkSource)!==source)throw Error('cached descriptor authority '+JSON.stringify({mode:t.Mode,indirect:l.BlocksIndirect,total:String(get(l.Total)),remaining:String(get(l.Remaining)),status:String(get(l.Status))}));
}
`
