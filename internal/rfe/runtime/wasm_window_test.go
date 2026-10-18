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

// Execute real generated graphs against independent ordinary leaves. Accesses
// revisit admitted pages after extending in both directions, alternate two
// distant ranges, encounter holes and discontiguous host mappings, and cross
// full-width address boundaries. Every fault's prefix and all counters matter.
func TestWasmGraphContiguousMappingWindows(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node required")
	}
	l := wasmLoopTestLayout()
	type fixture struct {
		Name      string
		Graph     []byte
		Leaves    [][]byte
		Addresses []string
		Pages     []string
	}
	var fixtures []fixture
	sequences := [][]uint64{
		{1, 2, 3, 1, 2, 3, 4, 1}, {4, 3, 2, 1, 4, 3, 2, 1},
		{1, 17, 2, 18, 3, 19, 1, 17, 2, 18}, {1, 3, 1, 2, 3, 2},
		{1, 2, 1 + (1<<32)/4096, 1},
		{(1 << 52) - 1, 0, (1 << 52) - 1, 0},
	}
	for si, seq := range sequences {
		for _, edge := range []uint64{0, 4080, 4088, 4095} {
			var blocks []GraphBlock
			f := fixture{Name: strconv.Itoa(si) + "/" + strconv.FormatUint(edge, 10)}
			seen := map[uint64]bool{}
			for i, page := range seq {
				address := page*4096 + edge
				f.Addresses = append(f.Addresses, strconv.FormatUint(address, 10))
				// The high alias is deliberately absent; it must not reuse the low page.
				if page != 1+(1<<32)/4096 && !seen[page] {
					seen[page] = true
					f.Pages = append(f.Pages, strconv.FormatUint(page, 10))
				}
				var b Builder
				b.MemoryInstructionProgress = true
				b.Store(0, b.Constant(uint64(i*4)))
				b.Checkpoint(3, 0)
				a := b.Constant(address)
				v := b.MemoryLoad(a, 8)
				b.Store(1, b.Binary(Xor, b.Load(1), v))
				b.Checkpoint(3, 1)
				b.MemoryStore(a, b.Binary(Add, v, b.Constant(1)), 8)
				count := b.Binary(Add, b.Load(2), b.Constant(1))
				b.Store(2, count)
				next := uint64((i + 1) % len(seq) * 4)
				b.Store(0, b.Choose(b.Binary(Less, count, b.Constant(uint64(2*len(seq)+1))), b.Constant(next), b.Constant(uint64(4*len(seq)))))
				b.Checkpoint(3, 2)
				block := GraphBlock{PC: uint64(i * 4), Instructions: 2, Ops: b.FinishMemory(3), SuccessorCount: 1}
				block.Successors[0] = next
				blocks = append(blocks, block)
				leaf, err := EmitWasmMemoryBlock(block.Ops, 3, l)
				if err != nil {
					t.Fatal(err)
				}
				f.Leaves = append(f.Leaves, leaf)
			}
			f.Graph, err = emitWasmGraph(blocks, 3, 0, l)
			if err != nil {
				t.Fatal(err)
			}
			fixtures = append(fixtures, f)
		}
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
	cmd := exec.CommandContext(ctx, node, "-e", wasmMappingWindowJS)
	cmd.Stdin = bytes.NewReader(input)
	out, err := cmd.CombinedOutput()
	if err != nil || string(out) != "PASS\n" {
		t.Fatalf("mapping windows: %v\n%s", err, out)
	}
}

const wasmMappingWindowJS = `
const p=JSON.parse(require('node:fs').readFileSync(0,'utf8')),l=p.Layout,mod=b=>new WebAssembly.Module(Buffer.from(b,'base64'));
for(const f of p.Fixtures){
 const graph=mod(f.Graph),leaves=f.Leaves.map(mod),n=leaves.length;
 function run(compiled,mode,budget){
  const memory=new WebAssembly.Memory({initial:8}),v=new DataView(memory.buffer),state=4096,ctx=32768,data=131072,clock=98304,epoch=98312;
  const set=(o,x)=>v.setBigUint64(ctx+o,BigInt(x),true),get=o=>v.getBigUint64(ctx+o,true),gs=i=>v.getBigUint64(state+8*i,true);
  v.setBigUint64(state+8,99n,true);v.setBigUint64(clock,5n,true);v.setBigUint64(epoch,3n,true);v.setUint32(ctx+l.Clock,clock,true);
  // Fill all potential physical data, including deliberately unmapped holes.
  for(let i=0;i<32;i++){v.setBigUint64(data+4096*i,BigInt(1001+101*i),true);v.setBigUint64(data+4096*i+4080,BigInt(2002+101*i),true);v.setBigUint64(data+4096*i+4088,BigInt(3003+101*i),true)}
  const pages=f.Pages.map(BigInt);
  pages.forEach((page,i)=>{
   if(mode==='hole'&&i===1)return;
   const slot=Number(page&63n),a=ctx+l.Pages+slot*l.PageSize;
   const physical=mode==='discontiguous'?((i*7)%32):page<32n?Number(page):30;
   v.setBigUint64(a+l.PageNumber,page,true);v.setUint32(a+l.PageData,data+4096*physical,true);
   const perm=i===1?(mode==='readonly'?1:mode==='denied'?0:mode==='watched'?7:3):3;
   v.setBigUint64(a+l.PagePermissions,BigInt(perm),true);v.setUint32(a+l.PageEpoch,epoch,true);
  });
  set(l.Linked,1);set(l.Remaining,budget);
  const funcs=compiled?[new WebAssembly.Instance(graph,{env:{memory}}).exports.run]:leaves.map(m=>new WebAssembly.Instance(m,{env:{memory}}).exports.run);
  if(compiled)funcs[0](state,ctx);
  else for(let step=0;step<2*n+2;step++){
   funcs[Number(gs(0)/4n)](state,ctx);if(get(l.Status)!==0n)break;
   if(gs(0)===BigInt(4*n)||get(l.Remaining)<get(l.Retired)+2n){set(l.Status,4);break}
   set(l.Total,get(l.Total)+get(l.Retired));set(l.MemoryTotal,get(l.MemoryTotal)+get(l.MemoryRetired));set(l.Remaining,get(l.Remaining)-get(l.Retired));
  }
  set(l.Total,get(l.Total)+get(l.Retired));set(l.MemoryTotal,get(l.MemoryTotal)+get(l.MemoryRetired));set(l.Remaining,get(l.Remaining)-get(l.Retired));
  return JSON.stringify({state:[0,1,2].map(i=>String(gs(i))),context:[l.Retired,l.MemoryRetired,l.Status,l.Address,l.Total,l.MemoryTotal,l.Remaining].map(o=>String(get(o))),data:Buffer.from(memory.buffer,data,32*4096).toString('base64')});
 }
 for(const mode of ['rw','discontiguous','hole','readonly','denied','watched'])for(const budget of [2*n,2*n+1,2*n+3,65536]){const got=run(true,mode,budget),want=run(false,mode,budget);if(got!==want)throw Error('window '+JSON.stringify({name:f.Name,mode,budget,got:JSON.parse(got).context,want:JSON.parse(want).context}))}
}
console.log('PASS');
`
