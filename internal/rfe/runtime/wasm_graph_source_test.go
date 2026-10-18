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

// Optional edge hints must not overwrite public progress, mapping authority or
// descriptor-array metadata in nonstandard caller layouts. Compare complete
// context bytes with separately emitted leaves, not a model of alias guards.
func TestWasmGraphSourceHintsNeverAlterAliasedContext(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node required")
	}
	type trial struct {
		Layout WasmMemoryLayout
		Graph  []byte
		Leaves [][]byte
		Hint   bool
	}
	var trials []trial
	for _, indirect := range []bool{false, true} {
		l := wasmLoopTestLayout()
		l.Retired, l.MemoryRetired = 2256, 2264
		l.BlocksIndirect = indirect
		offsets := []uint32{0, l.Retired, l.MemoryRetired, l.Status, l.Address, l.Total, l.Remaining, l.MemoryTotal, l.Linked, l.Clock, l.LoopExits, l.LoopIterations, l.Pages + l.PageSize + l.PageNumber, l.Pages + l.PageSize + l.PagePermissions, l.Blocks, l.Blocks + 4, ^uint32(0) - 3, 2176}
		for _, off := range offsets {
			l.LinkSource = off
			blocks := make([]GraphBlock, 2)
			for i := range blocks {
				var b Builder
				b.MemoryInstructionProgress = true
				b.Store(0, b.Constant(uint64(4*i)))
				b.Checkpoint(2, 0)
				b.Store(1, b.Binary(Add, b.Load(1), b.Constant(uint64(i+1))))
				b.Store(0, b.Constant(uint64(4*(i+1))))
				b.Checkpoint(2, 1)
				blocks[i] = GraphBlock{PC: uint64(4 * i), Instructions: 1, Ops: b.FinishMemory(2)}
			}
			blocks[0].Successors[0] = 4
			blocks[0].SuccessorCount = 1
			graph, err := emitWasmGraph(blocks, 2, 0, l)
			if err != nil {
				t.Fatal(err)
			}
			f := trial{Layout: l, Graph: graph, Hint: off == 2176}
			for _, block := range blocks {
				code, err := EmitWasmMemoryBlock(block.Ops, 2, l)
				if err != nil {
					t.Fatal(err)
				}
				f.Leaves = append(f.Leaves, code)
			}
			trials = append(trials, f)
		}
	}
	input, err := json.Marshal(trials)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "-e", wasmGraphSourceAliasesJS)
	cmd.Stdin = bytes.NewReader(input)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("source hints: %v\n%s", err, out)
	}
}

const wasmGraphSourceAliasesJS = `
const trials=JSON.parse(require('node:fs').readFileSync(0,'utf8')),mod=b=>new WebAssembly.Module(Buffer.from(b,'base64'));
for(const t of trials){
 const l=t.Layout,gm=mod(t.Graph),refs=t.Leaves.map(mod);
 function run(compiled){
  const memory=new WebAssembly.Memory({initial:8}),v=new DataView(memory.buffer),s=4096,c=32768;
  const set=(o,x)=>v.setBigUint64(c+o,BigInt(x),true),get=o=>v.getBigUint64(c+o,true);
  set(l.Linked,1);set(l.Remaining,65536);if(l.BlocksIndirect)v.setUint32(c+l.Blocks,131072,true);
  const d=c+l.Pages+l.PageSize;v.setBigUint64(d+l.PageNumber,33n,true);v.setBigUint64(d+l.PagePermissions,3n,true);
  if(compiled)new WebAssembly.Instance(gm,{env:{memory}}).exports.run(s,c);
  else{
   for(let i=0;i<2;i++){
    new WebAssembly.Instance(refs[i],{env:{memory}}).exports.run(s,c);
    if(i===0){set(l.Total,get(l.Total)+get(l.Retired));set(l.MemoryTotal,get(l.MemoryTotal)+get(l.MemoryRetired));set(l.Remaining,get(l.Remaining)-get(l.Retired));}
   }
   if(t.Hint)set(l.LinkSource,4);set(l.Status,4);
  }
  return JSON.stringify({state:[v.getBigUint64(s,true),v.getBigUint64(s+8,true)].map(String),context:Array.from(new Uint8Array(memory.buffer,c,40960))});
 }
 const got=run(true),want=run(false);if(got!==want)throw Error('alias hint '+l.LinkSource+' indirect='+l.BlocksIndirect);
}
`
