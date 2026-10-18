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

// More than 127 nested labels exercises multi-byte branch depths. The value
// 15 is also a return opcode byte, so scanning raw bytes would corrupt this
// fixture. The independent oracle counts a ring one architectural step at a
// time rather than using the generated graph's scheduling logic.
func TestWasmGraphDeepExitAndImmediateBytes(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node required")
	}
	blocks := make([]GraphBlock, 128)
	for i := range blocks {
		var b Builder
		b.Checkpoint(3, 0)
		b.Store(1, b.Binary(Add, b.Load(1), b.Constant(15)))
		b.Store(0, b.Constant(uint64((i+1)%len(blocks)*4)))
		b.Checkpoint(3, 1)
		b.Guard(b.Load(2))
		blocks[i] = GraphBlock{PC: uint64(i * 4), Instructions: 1, Ops: b.FinishMemory(3), Successors: [4]uint64{uint64((i + 1) % len(blocks) * 4)}, SuccessorCount: 1}
	}
	l := wasmLoopTestLayout()
	l.Retired, l.MemoryRetired = 2256, 2264
	if !wasmGraphSessionSafe(l) {
		t.Fatal("deep fixture must exercise private prefixes")
	}
	code, err := emitWasmGraph(blocks, 3, 0, l)
	if err != nil {
		t.Fatal(err)
	}
	input, err := json.Marshal(struct {
		Code   []byte
		Layout WasmMemoryLayout
	}{code, l})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "-e", `
const p=JSON.parse(require('node:fs').readFileSync(0,'utf8')),l=p.Layout,mod=new WebAssembly.Module(Buffer.from(p.Code,'base64'));
for(const budget of [128,129,130,255,256,257,65536])for(const pass of [0,1]){
 const memory=new WebAssembly.Memory({initial:4}),v=new DataView(memory.buffer),state=4096,ctx=32768;
 const set=(o,x)=>v.setBigUint64(ctx+o,BigInt(x),true),get=o=>v.getBigUint64(ctx+o,true);
 v.setBigUint64(state+16,BigInt(pass),true);set(l.Remaining,budget);set(l.Linked,1);
 new WebAssembly.Instance(mod,{env:{memory}}).exports.run(state,ctx);
 let steps=0,remaining=budget;
 for(;;){steps++;if(!pass||remaining<2)break;remaining--;}
 const check=(got,want,label)=>{if(got!==BigInt(want))throw Error(JSON.stringify({budget,pass,label,got:String(got),want}))};
 check(v.getBigUint64(state,true),(steps%128)*4,'pc');check(v.getBigUint64(state+8,true),steps*15,'state');
 check(get(l.Total),steps-1,'total');check(get(l.Remaining),remaining,'remaining');check(get(l.Retired),1,'retired');check(get(l.Status),pass?4:2,'status');
}
console.log('PASS');`)
	cmd.Stdin = bytes.NewReader(input)
	out, err := cmd.CombinedOutput()
	if err != nil || string(out) != "PASS\n" {
		t.Fatalf("deep graph: %v\n%s", err, out)
	}
}

func TestWasmGraphExitRejectsMalformedCode(t *testing.T) {
	for _, code := range [][]byte{{0x42, 0x80}, {0x02, 0x40}, {0x0b}, {0x05}, {0x02, 0x7d, 0x0b}, {0x0e, 0xff, 0xff, 0xff, 0xff, 0x0f}, {0x11, 0}, {0xfc, 0}} {
		if _, err := wasmGraphExitBranches(code); err == nil {
			t.Fatalf("malformed code accepted: %x", code)
		}
	}
}
