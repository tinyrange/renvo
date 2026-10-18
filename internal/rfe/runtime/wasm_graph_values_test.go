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

func TestWasmGraphDelayedArithmeticCapturesStateSnapshots(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node required")
	}
	// Snapshot both inputs before overwriting an architectural source. Pure
	// arithmetic may be delayed, but the old state reads must not be.
	ops := []Op{{Kind: LoadState, Imm: 1}, {Kind: LoadState, Imm: 2}, {Kind: Const, Imm: 0xabcdef}, {Kind: StoreState, A: 2, Imm: 2}}
	value := Value(1)
	for i := 0; i < 250; i++ {
		constant := Value(len(ops))
		ops = append(ops, Op{Kind: Const, Imm: uint64(i)*0x9e3779b97f4a7c15 + 0x12345})
		next := Value(len(ops))
		ops = append(ops, Op{Kind: Add, A: value, B: constant})
		value = Value(len(ops))
		ops = append(ops, Op{Kind: Xor, A: next, B: 0})
	}
	ops = append(ops, Op{Kind: StoreState, A: value, Imm: 1}, Op{Kind: StoreState, A: 0, Imm: 2})
	pc := Value(len(ops))
	ops = append(ops, Op{Kind: Const, Imm: 4}, Op{Kind: StoreState, A: pc, Imm: 0})
	l := wasmLoopTestLayout()
	// The guard makes the intervening store observable on a possible exit;
	// its nonzero operand passes in this fixture without changing the oracle.
	graphOps := append(append([]Op(nil), ops[:4]...), Op{Kind: Guard, A: 2})
	for _, op := range ops[4:] {
		left, right := memoryOperands(op)
		if left && op.A >= 4 {
			op.A++
		}
		if right && op.B >= 4 {
			op.B++
		}
		graphOps = append(graphOps, op)
	}
	graphOps = append(graphOps, Op{Kind: Progress, Imm: 1})
	code, err := emitWasmGraph([]GraphBlock{
		{PC: 0, Instructions: 1, Ops: graphOps, Successors: [4]uint64{4}, SuccessorCount: 1},
		{PC: 4, Instructions: 1, Ops: []Op{{Kind: Const, Imm: 8}, {Kind: StoreState, A: 0, Imm: 0}, {Kind: Progress, Imm: 1}}},
	}, 3, 0, l)
	if err != nil {
		t.Fatal(err)
	}
	type trial struct{ Initial, Expected []string }
	trials := []trial{}
	seed := uint64(0x123456789abcdef)
	for i := 0; i < 128; i++ {
		seed ^= seed << 13
		seed ^= seed >> 7
		seed ^= seed << 17
		state := []uint64{0, seed, ^seed + uint64(i)}
		want := append([]uint64(nil), state...)
		if err := Interpret(ops, want); err != nil {
			t.Fatal(err)
		}
		want[0] = 8 // second independent block
		tr := trial{}
		for _, x := range state {
			tr.Initial = append(tr.Initial, strconv.FormatUint(x, 10))
		}
		for _, x := range want {
			tr.Expected = append(tr.Expected, strconv.FormatUint(x, 10))
		}
		trials = append(trials, tr)
	}
	input, err := json.Marshal(struct {
		Code   []byte
		Trials []trial
		Layout WasmMemoryLayout
	}{code, trials, l})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "-e", `const p=JSON.parse(require('node:fs').readFileSync(0,'utf8')),memory=new WebAssembly.Memory({initial:2}),v=new DataView(memory.buffer),run=new WebAssembly.Instance(new WebAssembly.Module(Buffer.from(p.Code,'base64')),{env:{memory}}).exports.run;for(const t of p.Trials){t.Initial.forEach((x,i)=>v.setBigUint64(i*8,BigInt(x),true));v.setBigUint64(32768+p.Layout.Linked,1n,true);v.setBigUint64(32768+p.Layout.Remaining,65536n,true);run(0,32768);t.Expected.forEach((x,i)=>{if(v.getBigUint64(i*8,true)!==BigInt(x))throw Error('graph expression snapshots corruption at '+i)})}console.log('PASS');`)
	cmd.Stdin = bytes.NewReader(input)
	out, err := cmd.CombinedOutput()
	if err != nil || string(out) != "PASS\n" {
		t.Fatalf("graph expression snapshots: %v\n%s", err, out)
	}
}
