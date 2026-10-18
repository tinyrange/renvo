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

// Real Wasm targets and proof calls: the optimization hint must yield only at
// the next block's boundary after a positive precisely accounted prefix. A
// forged public tuple fails its proof before any optimization request is used.
func TestWasmHotTargetUpgradeIsOnlyAPreciseHint(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node required")
	}
	type fixture struct {
		Layout                    WasmMemoryLayout
		Dispatcher, Leaf, Mutator []byte
		Base                      string
		Mode                      string
	}
	var fixtures []fixture
	for _, base := range []uint64{0, 0xffff800000000000} {
		for _, mode := range []string{"demand", "retry-delay", "zero-prefix", "acknowledged", "disabled", "forged-count", "forged-entry", "forged-pc"} {
			l := wasmLoopTestLayout()
			l.LinkSource = 2176
			l.DescriptorHot = 24
			if mode == "disabled" {
				l.DescriptorHot = 0
			}
			dispatcher, err := EmitWasmDispatcher(2, 0, l)
			if err != nil {
				t.Fatal(err)
			}
			var leaves [][]byte
			for i := 0; i < 2; i++ {
				var w wasmCode
				w.u32(0)
				w.get(0)
				w.get(0)
				w.mem(0x29, 3, 8)
				w.i64(1)
				w.byte(0x7c)
				w.mem(0x37, 3, 8)
				w.get(0)
				w.i64(base + uint64((1-i)*4))
				w.mem(0x37, 3, 0)
				if i == 0 && (mode == "forged-count" || mode == "forged-entry" || mode == "forged-pc") {
					w.get(0)
					w.mem(0x29, 3, 8)
					w.i64(509)
					w.byte(0x51)
					w.byte(0x04)
					w.byte(0x40)
					off, val := l.DescriptorInstructions, uint64(2)
					if mode == "forged-entry" {
						off, val = l.DescriptorEntry, 48
					}
					if mode == "forged-pc" {
						off, val = l.DescriptorPC, (base+4)^(1<<32)
					}
					w.get(1)
					w.i64(val)
					w.mem(0x37, 3, l.Blocks+uint32((((base+4)>>2)^((base+4)>>12))&255)*l.DescriptorSize+off)
					w.byte(0x0b)
				}
				w.byte(0x0b)
				leaves = append(leaves, wasmFunctionModule(w.code, false, false))
			}
			fixtures = append(fixtures, fixture{l, dispatcher, leaves[0], leaves[1], strconv.FormatUint(base, 10), mode})
		}
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
	cmd := exec.CommandContext(ctx, node, "-e", wasmDemandUpgradeJS)
	cmd.Stdin = bytes.NewReader(input)
	out, err := cmd.CombinedOutput()
	if err != nil || string(out) != "PASS\n" {
		t.Fatalf("demand hint: %v\n%s", err, out)
	}
}

const wasmDemandUpgradeJS = `
const p=JSON.parse(require('node:fs').readFileSync(0,'utf8')),mod=b=>new WebAssembly.Module(Buffer.from(b,'base64')),proof=mod(p.Proof);
for(const f of p.Fixtures){
 const l=f.Layout,base=BigInt(f.Base),memory=new WebAssembly.Memory({initial:8}),v=new DataView(memory.buffer),s=4096,c=32768,table=new WebAssembly.Table({element:'anyfunc',initial:8});
 const get=o=>v.getBigUint64(c+o,true),set=(o,x)=>v.setBigUint64(c+o,BigInt(x),true),gs=i=>v.getBigUint64(s+8*i,true);
 v.setBigUint64(s,base,true);set(l.Linked,1);set(l.Remaining,1500);
 const descriptors=[];
 for(let i=0;i<2;i++){
  const pc=base+4n*BigInt(i),entry=16n*(BigInt(i)+1n),d=c+l.Blocks+Number(((pc>>2n)^(pc>>12n))&255n)*l.DescriptorSize;descriptors.push(d);
  v.setBigUint64(d+l.DescriptorPC,pc,true);v.setBigUint64(d+l.DescriptorEntry,entry,true);v.setBigUint64(d+l.DescriptorInstructions,1n,true);v.setBigUint64(d+24,f.Mode==='acknowledged'?0xffffffffffffffffn:f.Mode==='retry-delay'?512n:f.Mode==='zero-prefix'?1n:0n,true);
  table.set(2+2*i,new WebAssembly.Instance(mod(i===0?f.Leaf:f.Mutator),{env:{memory}}).exports.run);
  table.set(3+2*i,new WebAssembly.Instance(proof,{proof:{pc,entry,instructions:1n,words:2}}).exports.prove);
 }
 const run=new WebAssembly.Instance(mod(f.Dispatcher),{env:{memory,table}}).exports.run;run(s,c);
 const forged=f.Mode.startsWith('forged'),demand=f.Mode==='demand'||f.Mode==='retry-delay'||f.Mode==='zero-prefix',stopped=forged||demand,count=f.Mode==='retry-delay'?1022n:f.Mode==='zero-prefix'?1n:demand?510n:forged?509n:1500n,status=demand?5n:f.Mode==='forged-pc'?0n:forged?3n:0n;
 if(gs(1)!==count||get(l.Total)!==count||get(l.Remaining)!==1500n-count||get(l.MemoryTotal)!==0n||get(l.Status)!==status||gs(0)!==base+(count%2n)*4n)throw Error('hint prefix '+JSON.stringify({mode:f.Mode,count:String(gs(1)),total:String(get(l.Total)),remaining:String(get(l.Remaining)),status:String(get(l.Status)),pc:String(gs(0))}));
 if(demand){
  descriptors.forEach(d=>v.setBigUint64(d+24,0xffffffffffffffffn,true));set(l.Status,0);run(s,c);
  if(gs(1)!==1500n||get(l.Total)!==1500n||get(l.Remaining)!==0n||get(l.Status)!==0n)throw Error('hint resume');
 }
}
console.log('PASS');
`
