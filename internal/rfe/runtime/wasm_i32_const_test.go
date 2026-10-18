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

// Decode and execute every signed LEB boundary, including values with bit31
// set, using actual Wasm. The oracle compares independent uint32 bit patterns.
func TestWasmI32ConstantsPreserveEveryBit(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node required")
	}
	values := []uint32{0, 1, 63, 64, 127, 128, 8191, 8192, 0xffff, 0x7fffffff, 0x80000000, 0xffffffc0, 0xffffffbf, 0xffffffff}
	for shift := uint(1); shift < 32; shift++ {
		values = append(values, (uint32(1)<<shift)-1, uint32(1)<<shift, ^(uint32(1) << shift))
	}
	random := uint32(0x12345678)
	for i := 0; i < 256; i++ {
		random ^= random << 13
		random ^= random >> 17
		random ^= random << 5
		values = append(values, random)
	}
	var w wasmCode
	w.u32(0)
	for i, v := range values {
		w.get(0)
		w.i32(v)
		w.byte(0xad)
		w.mem(0x37, 3, uint32(i*8))
	}
	w.byte(0x0b)
	payload, err := json.Marshal(struct {
		Code   []byte
		Values []uint32
	}{wasmFunctionModule(w.code, false, false), values})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "-e", `
const p=JSON.parse(require('node:fs').readFileSync(0,'utf8')),memory=new WebAssembly.Memory({initial:1}),view=new DataView(memory.buffer);
new WebAssembly.Instance(new WebAssembly.Module(Buffer.from(p.Code,'base64')),{env:{memory}}).exports.run(0,32768);
p.Values.forEach((value,i)=>{const got=view.getBigUint64(i*8,true);if(got!==BigInt(value))throw Error('constant '+value+' got '+got)});
console.log('PASS');`)
	cmd.Stdin = bytes.NewReader(payload)
	out, err := cmd.CombinedOutput()
	if err != nil || string(out) != "PASS\n" {
		t.Fatalf("i32 constants: %v\n%s", err, out)
	}
}
