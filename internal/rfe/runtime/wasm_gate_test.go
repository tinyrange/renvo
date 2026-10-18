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

func TestWasmArenaGatePrivateAuthorityBoundsAndRevocation(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node required to execute generated Wasm")
	}
	ops := []Op{{Kind: LoadState, Imm: 0}, {Kind: Const, Imm: 1}, {Kind: Add, A: 0, B: 1}, {Kind: StoreState, A: 2, Imm: 0}}
	block, err := EmitWasmBlock(ops, 1)
	if err != nil {
		t.Fatal(err)
	}
	input, err := json.Marshal(struct{ Gate, Block []byte }{emitWasmGate(), block})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "-e", `
const fs=require('node:fs'),x=JSON.parse(fs.readFileSync(0,'utf8'));
const code=k=>Buffer.from(x[k],'base64'),memory=new WebAssembly.Memory({initial:1}),v=new DataView(memory.buffer);
const table=new WebAssembly.Table({element:'anyfunc',initial:32768,maximum:32768});
const block=new WebAssembly.Instance(new WebAssembly.Module(code('Block')),{env:{memory}});table.set(2,block.exports.run);
const metadata=new WebAssembly.Memory({initial:4,maximum:4}),m=new DataView(metadata.buffer),module=new WebAssembly.Module(code('Gate'));
function gate(owner){return new WebAssembly.Instance(module,{env:{metadata,table},gate:{owner:new WebAssembly.Global({value:'i32',mutable:false},owner),bytes:new WebAssembly.Global({value:'i64',mutable:false},65536n)}}).exports}
const original=gate(7),invoke=original.invoke;m.setUint32(0,7,true);m.setUint32(16,1025,true);m.setUint32(20,64,true);
function reject(args){const before=Buffer.from(memory.buffer);if(invoke(...args)!==-1||!before.equals(Buffer.from(memory.buffer)))throw Error('rejection wrote state '+args)}
for(const a of [[0,16,4,1,0],[8,16,4,1,0],[7,0,4,1,0],[7,17,4,1,0],[7,32,4,1,0],[7,262144,4,1,0],[7,-16,4,1,0],[7,16,4,0,0],[7,16,4,2,0],[7,16,65529,1,0],[7,16,-1,1,0],[7,16,4,1,4]])reject(a);
if(invoke(7,16,4,1,0)!==0||v.getBigUint64(4,true)!==1n)throw Error('valid pure call');
if(invoke(7,16,65528,1,0)!==0||v.getBigUint64(65528,true)!==1n)throw Error('exact state boundary');
m.setUint32(16,1537,true);reject([7,16,4,1,0]);reject([7,16,4,1,65473]);reject([7,16,4,1,-1]);
if(invoke(7,16,4,1,65472)!==0||v.getBigUint64(4,true)!==2n)throw Error('exact context boundary');
// Emulator memory cannot forge the private metadata's owner/revocation word.
m.setUint32(0,0,true);v.setUint32(0,7,true);reject([7,16,4,1,128]);
// A replacement at the same root slot has a new immutable owner token.
m.setUint32(0,32775,true);const replacement=gate(32775).invoke;reject([7,16,4,1,128]);
if(replacement(7,16,4,1,128)!==-1||replacement(32775,16,4,1,128)!==0)throw Error('recycled owner authority');
// The shared proof checks the complete tuple against worker-private metadata.
// Aliased emulator bytes, truncation, forged dimensions and absent admissions
// must not acquire authority or write emulator memory.
const fresh=gate(32775),prove=fresh.prove,pc=0xffff800080001234n;
m.setUint32(16,1|512|3072|(256<<12),true);m.setBigUint64(24,pc,true);
const snapshot=Buffer.from(memory.buffer);
for(const a of [[pc,0n,256n,1],[pc,17n,256n,1],[pc,262144n,256n,1],[pc,-16n,256n,1],[pc,0x100000010n,256n,1],[pc,16n,0n,1],[pc,16n,257n,1],[pc,16n,0x100000100n,1],[pc,16n,256n,0],[pc,16n,256n,257],[pc^0x100000000n,16n,256n,1],[pc^1n,16n,256n,1],[0n,32n,1n,1]])if(prove(...a)!==0)throw Error('forged shared proof '+a);
if(prove(pc,16n,256n,1)!==1||!snapshot.equals(Buffer.from(memory.buffer)))throw Error('valid proof or purity');
m.setUint32(16,1537,true);if(prove(pc,16n,256n,1)!==0)throw Error('unadmitted entry');
m.setUint32(16,1|512|3072|(256<<12),true);m.setUint32(0,0,true);
if(prove(pc,16n,256n,1)!==0)throw Error('closed proof');
m.setUint32(0,65543,true);if(prove(pc,16n,256n,1)!==0||gate(65543).prove(pc,16n,256n,1)!==1)throw Error('old owner proof revived');
console.log('PASS');`)
	cmd.Stdin = bytes.NewReader(input)
	out, err := cmd.CombinedOutput()
	if err != nil || string(out) != "PASS\n" {
		t.Fatalf("actual Wasm gate: %v\n%s", err, out)
	}
}
