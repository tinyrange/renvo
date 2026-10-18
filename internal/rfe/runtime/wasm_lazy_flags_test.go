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

// Real graph execution vs independently checkpointed ordinary leaves. Two flag
// words may be pending together, change kind/width, be read by another block,
// be overwritten without a read, or need publication at faults/odd budgets.
func TestWasmGraphLazyFlagsReadsOverwritesAndFaults(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node required")
	}
	type fixture struct {
		Layout WasmMemoryLayout
		Graph  []byte
		Leaves [][]byte
	}
	fixtures := []fixture{}
	for _, width := range []int{32, 64} {
		for _, kind := range []int{0, 1, 2} {
			for _, overwrite := range []bool{false, true} {
				for _, service := range []bool{false, true} {
					l := wasmLoopTestLayout()
					if service {
						l.ServiceSP, l.ServiceFP, l.ServiceWrite = 2176, 2180, 2184
					}
					blocks := make([]GraphBlock, 3)
					for i := range blocks {
						var b Builder
						b.MemoryInstructionProgress = true
						b.Store(0, b.Constant(uint64(4*i)))
						b.Checkpoint(8, 0)
						switch i {
						case 0:
							a, rhs := b.Load(1), b.Load(2)
							flags := b.ArithmeticFlags(a, rhs, width, kind == 1)
							if kind == 2 {
								flags = b.LogicalFlags(a, width)
							}
							b.Store(4, flags)
							b.Checkpoint(8, 1)
							b.Store(1, b.Binary(Add, a, b.Constant(1)))
							b.Store(5, b.LogicalFlags(rhs, width))
							b.Checkpoint(8, 2)
							b.Store(0, b.Constant(4))
							blocks[i].Successors[0] = 4
							blocks[i].SuccessorCount = 1
						case 1:
							if overwrite {
								b.Store(4, b.Constant(0x9abcdef012345678))
							}
							b.Store(3, b.Load(4))
							b.Checkpoint(8, 1)
							value := b.MemoryLoad(b.Load(6), 8)
							b.Store(3, b.Binary(Xor, b.Load(3), value))
							b.Store(4, b.ArithmeticFlags(b.Load(1), b.Load(2), 96-width, kind != 1))
							b.Checkpoint(8, 2)
							b.Store(0, b.Constant(8))
							blocks[i].Successors[0] = 8
							blocks[i].SuccessorCount = 1
						case 2:
							b.Store(3, b.Binary(Xor, b.Load(3), b.Load(5)))
							b.Checkpoint(8, 1)
							b.MemoryStore(b.Load(6), b.Load(3), 8)
							b.Store(6, b.Binary(Add, b.Load(6), b.Constant(8)))
							b.Checkpoint(8, 2)
							b.Store(0, b.Choose(b.Binary(Less, b.Load(1), b.Load(7)), b.Constant(0), b.Constant(12)))
							blocks[i].Successors[0] = 0
							blocks[i].SuccessorCount = 1
						}
						b.Checkpoint(8, 2)
						blocks[i].PC, blocks[i].Instructions, blocks[i].Ops = uint64(4*i), 2, b.FinishMemory(8)
					}
					graph, err := emitWasmGraph(blocks, 8, 0, l)
					if err != nil {
						t.Fatal(err)
					}
					f := fixture{Layout: l, Graph: graph}
					for _, block := range blocks {
						code, err := EmitWasmMemoryBlock(block.Ops, 8, l)
						if err != nil {
							t.Fatal(err)
						}
						f.Leaves = append(f.Leaves, code)
					}
					fixtures = append(fixtures, f)
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
	cmd := exec.CommandContext(ctx, node, "-e", wasmLazyFlagsJS)
	cmd.Stdin = bytes.NewReader(input)
	out, err := cmd.CombinedOutput()
	if err != nil || string(out) != "PASS\n" {
		t.Fatalf("lazy flags: %v\n%s", err, out)
	}
}

const wasmLazyFlagsJS = `
const p=JSON.parse(require('node:fs').readFileSync(0,'utf8')),mod=b=>new WebAssembly.Module(Buffer.from(b,'base64'));
for(let fIndex=0;fIndex<p.Fixtures.length;fIndex++){
 const f=p.Fixtures[fIndex],l=f.Layout,graph=mod(f.Graph),leaves=f.Leaves.map(mod);
 function run(compiled,o){
  const memory=new WebAssembly.Memory({initial:8}),v=new DataView(memory.buffer),state=4096,ctx=32768,data=131072,clock=98304,epoch=98312;
  const set=(o,x)=>v.setBigUint64(ctx+o,BigInt(x),true),get=o=>v.getBigUint64(ctx+o,true),gs=i=>v.getBigUint64(state+8*i,true);
  const initial=[0n,o.start,o.rhs,99n,0xa0000000n,0xf0000000n,BigInt(o.address),o.start+4n];initial.forEach((x,i)=>v.setBigUint64(state+8*i,x,true));
  v.setBigUint64(clock,5n,true);v.setBigUint64(epoch,3n,true);v.setUint32(ctx+l.Clock,clock,true);for(let i=0;i<8192;i++)v.setUint8(data+i,(i*17+31)&255);
  const page=n=>{const a=ctx+l.Pages+n*l.PageSize;v.setBigUint64(a+l.PageNumber,BigInt(n),true);v.setUint32(a+l.PageData,data+(n-1)*4096,true);v.setBigUint64(a+l.PagePermissions,BigInt(o.permissions),true);v.setUint32(a+l.PageEpoch,epoch,true)};
  if(o.present)page(1);set(l.Linked,1);set(l.Remaining,o.budget);if(l.ServiceSP){v.setUint32(ctx+l.ServiceSP,1,true);v.setUint32(ctx+l.ServiceFP,2,true)}
  const observations=[];
  const service=(...args)=>{observations.push([l.Total,l.MemoryTotal,l.Remaining,l.Retired,l.MemoryRetired,l.Status,l.Address].map(x=>String(get(x))));const address=get(l.Address),a=ctx+l.Pages+l.PageSize;if(address>=4096n&&address<8192n&&!v.getUint32(a+l.PageData,true)){page(1);return [1,0,0]}return [0,0,0]};
  const funcs=compiled?[new WebAssembly.Instance(graph,{env:{memory,service}}).exports.run]:leaves.map(m=>new WebAssembly.Instance(m,{env:{memory,service}}).exports.run);
  if(compiled)funcs[0](state,ctx);
  else for(let step=0;step<100;step++){
   const index=Number(gs(0)/4n);if(index>2)throw Error('reference bad PC');funcs[index](state,ctx);if(get(l.Status)!==0n)break;
   if(gs(0)===12n||get(l.Remaining)<4n){set(l.Status,4);break}
   set(l.Total,get(l.Total)+get(l.Retired));set(l.MemoryTotal,get(l.MemoryTotal)+get(l.MemoryRetired));set(l.Remaining,get(l.Remaining)-get(l.Retired));
  }
  set(l.Total,get(l.Total)+get(l.Retired));set(l.MemoryTotal,get(l.MemoryTotal)+get(l.MemoryRetired));set(l.Remaining,get(l.Remaining)-get(l.Retired));
  return JSON.stringify({state:initial.map((_,i)=>String(gs(i))),context:[l.Retired,l.MemoryRetired,l.Status,l.Address,l.Total,l.MemoryTotal,l.Remaining].map(x=>String(get(x))),observations,data:Buffer.from(memory.buffer,data,8192).toString('base64')});
 }
 for(const [start,rhs] of [[0n,0n],[0x7fffffffn,0xffffffffn],[0x8000000000000000n,0x7fffffffffffffffn]])for(const budget of [6,7,8,9,10,11,12,13,64,65536])for(const permissions of [1,2,3,7])for(const address of [4096,8184])for(const present of [false,true]){
  const o={start,rhs,budget,permissions,address,present},got=run(true,o),want=run(false,o);if(got!==want)throw Error('lazy mismatch '+JSON.stringify({fIndex,o,got,want},(_,x)=>typeof x==='bigint'?String(x):x))
 }
}
console.log('PASS');
`

// Execute the actual pure materializer with an independent interpreter oracle,
// including high bits in narrow inputs, carry/overflow edges and random pairs.
func TestWasmDeferredFlagHelperAgainstInterpreter(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node required")
	}
	var m, exports, code wasmCode
	m.code = []byte{0, 97, 115, 109, 1, 0, 0, 0}
	m.section(1, []byte{1, 0x60, 3, 0x7e, 0x7e, 0x7e, 1, 0x7e})
	m.section(3, []byte{1, 0})
	exports.u32(1)
	exports.name("flags")
	exports.byte(0)
	exports.u32(0)
	m.section(7, exports.code)
	body := wasmFlagFunctions(0)[0]
	code.u32(1)
	code.u32(uint32(len(body)))
	code.code = append(code.code, body...)
	m.section(10, code.code)
	type trial struct{ A, B, Mode, Expected string }
	trials := []trial{}
	appendPair := func(a, b uint64) {
		for _, mode := range []uint64{32, 64, 160, 192, 288, 320} {
			kind := ArithmeticFlags
			if mode&256 != 0 {
				kind = LogicalFlags
			}
			want := flagsOperation(kind, a, b, mode&255)
			trials = append(trials, trial{strconv.FormatUint(a, 10), strconv.FormatUint(b, 10), strconv.FormatUint(mode, 10), strconv.FormatUint(want, 10)})
		}
	}
	seeds := []uint64{0, 1, 2, 31, 32, 63, 64, 255, 0xffffffff, 0x100000000, 0x7fffffff, 0x80000000, 0x7fffffffffffffff, 0x8000000000000000, 0xffff800080000000, ^uint64(0)}
	for _, a := range seeds {
		for _, b := range seeds {
			appendPair(a, b)
		}
	}
	random := uint64(0x123456789abcdef)
	for i := 0; i < 256; i++ {
		random ^= random << 13
		random ^= random >> 7
		random ^= random << 17
		a := random
		random ^= random << 13
		random ^= random >> 7
		random ^= random << 17
		appendPair(a, random)
	}
	input, err := json.Marshal(struct {
		Code   []byte
		Trials []trial
	}{m.code, trials})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "-e", `const p=JSON.parse(require('node:fs').readFileSync(0,'utf8')),flags=new WebAssembly.Instance(new WebAssembly.Module(Buffer.from(p.Code,'base64'))).exports.flags;for(const t of p.Trials){const got=flags(BigInt(t.A),BigInt(t.B),BigInt(t.Mode));if(got!==BigInt(t.Expected))throw Error('flag helper '+JSON.stringify({...t,got:String(got)}))}console.log('PASS');`)
	cmd.Stdin = bytes.NewReader(input)
	out, err := cmd.CombinedOutput()
	if err != nil || string(out) != "PASS\n" {
		t.Fatalf("flag helper oracle: %v\n%s", err, out)
	}
}
