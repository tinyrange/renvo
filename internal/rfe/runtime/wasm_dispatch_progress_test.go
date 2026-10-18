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

// Public state may alias progress words. Counter stores and RegionExit clearing
// must precede the same-PC observation, exactly as in the original dispatcher.
// Overlapping custom layouts must keep memory-observed status rather than cache
// it across a counter store. These are real generated calls, not a scheduler model.
func TestWasmDispatcherProgressAliasingAndOverlappingLayouts(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node required")
	}
	type fixture struct {
		Mode             string
		Layout           WasmMemoryLayout
		Dispatcher, Leaf []byte
		Root             uint64
	}
	fixtures := []fixture{}
	for _, mode := range []string{"separate", "state-status", "state-total", "state-remaining", "overlap", "partial-overlap"} {
		l := wasmLoopTestLayout()
		l.LinkSource = 2176
		if mode == "overlap" {
			l.Total = l.Status
		}
		if mode == "partial-overlap" {
			l.Total = l.Status + 1
		}
		root, next := uint64(0), uint64(0)
		if mode == "state-status" {
			next = 4
		}
		if mode == "state-remaining" {
			root, next = 4, 4
		}
		var b Builder
		b.Store(1, b.Binary(Add, b.Load(1), b.Constant(1)))
		b.Store(0, b.Constant(next))
		leaf, err := EmitWasmBlock(b.Finish(2), 2)
		if err != nil {
			t.Fatal(err)
		}
		dispatcher, err := EmitWasmDispatcher(2, 0, l)
		if err != nil {
			t.Fatal(err)
		}
		fixtures = append(fixtures, fixture{mode, l, dispatcher, leaf, root})
	}
	input, err := json.Marshal(struct {
		Fixtures []fixture
		Proof    []byte
	}{fixtures, emitWasmProof()})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "-e", wasmDispatcherProgressAliasJS)
	cmd.Stdin = bytes.NewReader(input)
	out, err := cmd.CombinedOutput()
	if err != nil || string(out) != "PASS\n" {
		t.Fatalf("progress aliasing: %v\n%s", err, out)
	}
}

const wasmDispatcherProgressAliasJS = `
const p=JSON.parse(require('node:fs').readFileSync(0,'utf8')),mod=b=>new WebAssembly.Module(Buffer.from(b,'base64')),proof=mod(p.Proof);
for(const f of p.Fixtures){
 const l=f.Layout,memory=new WebAssembly.Memory({initial:8}),v=new DataView(memory.buffer),ctx=32768;
 let state=4096;if(f.Mode==='state-status')state=ctx+l.Status;if(f.Mode==='state-total')state=ctx+l.Total;if(f.Mode==='state-remaining')state=ctx+l.Remaining;
 const set=(o,x)=>v.setBigUint64(ctx+o,BigInt(x),true),get=o=>v.getBigUint64(ctx+o,true),root=BigInt(f.Root);
 v.setBigUint64(state,root,true);v.setBigUint64(state+8,0n,true);set(l.Remaining,4);set(l.Linked,1);
 const d=ctx+l.Blocks+Number(((root>>2n)^(root>>12n))&255n)*l.DescriptorSize;
 v.setBigUint64(d+l.DescriptorPC,root,true);v.setBigUint64(d+l.DescriptorEntry,16n,true);v.setBigUint64(d+l.DescriptorInstructions,1n,true);
 const table=new WebAssembly.Table({element:'anyfunc',initial:4});
 table.set(2,new WebAssembly.Instance(mod(f.Leaf),{env:{memory}}).exports.run);
 table.set(3,new WebAssembly.Instance(proof,{proof:{pc:root,entry:16n,instructions:1n,words:2}}).exports.prove);
 new WebAssembly.Instance(mod(f.Dispatcher),{env:{memory,table}}).exports.run(state,ctx);
 const got={pc:String(v.getBigUint64(state,true)),counter:String(v.getBigUint64(state+8,true)),total:String(get(l.Total)),remaining:String(get(l.Remaining)),memory:String(get(l.MemoryTotal)),status:String(get(l.Status)),exits:String(get(l.LoopExits)),source:String(get(l.LinkSource))};
 let want={pc:'0',counter:'4',total:'4',remaining:'0',memory:'0',status:'0',exits:'0',source:'0'};
 if(f.Mode==='state-status')want.exits='4';
 if(f.Mode==='state-total')want={pc:'1',counter:'4',total:'1',remaining:'4',memory:'0',status:'0',exits:'0',source:'0'};
 if(f.Mode==='state-remaining')want={pc:'3',counter:'1',total:'1',remaining:'3',memory:'1',status:'0',exits:'0',source:'4'};
 if(f.Mode==='overlap'||f.Mode==='partial-overlap')want={pc:'0',counter:'1',total:'1',remaining:'3',memory:'0',status:f.Mode==='overlap'?'1':'256',exits:'0',source:'0'};
 if(JSON.stringify(got)!==JSON.stringify(want))throw Error('progress ordering '+JSON.stringify({mode:f.Mode,got,want}));
}
console.log('PASS');
`
