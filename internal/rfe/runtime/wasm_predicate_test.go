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

// Execute real generated Wasm with dynamic operands, checking every state word
// against the interpreter, including packed flags retained alongside predicates.
func TestWasmDirectConditionsDifferential(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node required for generated Wasm")
	}
	for mode, operands := range [][2]Value{{0, 1}, {0, 0}, {1, 0}} {
		t.Run(strconv.Itoa(mode), func(t *testing.T) {
			ops := []Op{{Kind: LoadState, Imm: 0}, {Kind: LoadState, Imm: 1}}
			slot := 2
			add := func(kind int, a, b Value, imm uint64) {
				index := Value(len(ops))
				ops = append(ops, Op{Kind: kind, A: a, B: b, Imm: imm}, Op{Kind: StoreState, A: index, Imm: uint64(slot)})
				slot++
			}
			for _, width := range []uint64{32, 64} {
				for _, sub := range []uint64{0, 128} {
					add(ArithmeticFlags, operands[0], operands[1], width|sub)
					for c := uint64(0); c < 16; c++ {
						add(ArithmeticCondition, operands[0], operands[1], width|sub|c<<8)
					}
				}
				add(LogicalFlags, operands[0], operands[1], width)
				for c := uint64(0); c < 16; c++ {
					add(LogicalCondition, operands[0], operands[1], width|c<<8)
				}
			}
			code, err := EmitWasmBlock(ops, slot)
			if err != nil {
				t.Fatal(err)
			}
			type trial struct{ Initial, Expected []string }
			var trials []trial
			appendTrial := func(a, b uint64) {
				state := make([]uint64, slot)
				for i := range state {
					state[i] = 0xcafef00d00000000 | uint64(i)
				}
				state[0], state[1] = a, b
				want := append([]uint64(nil), state...)
				if err := Interpret(ops, want); err != nil {
					t.Fatal(err)
				}
				tr := trial{Initial: make([]string, slot), Expected: make([]string, slot)}
				for i := range state {
					tr.Initial[i] = strconv.FormatUint(state[i], 10)
					tr.Expected[i] = strconv.FormatUint(want[i], 10)
				}
				trials = append(trials, tr)
			}
			seeds := []uint64{0, 1, 2, 31, 32, 63, 64, 255, 0xffffffff, 0x100000000, 0x7fffffff, 0x80000000, 0x7fffffffffffffff, 0x8000000000000000, 0xffff800080000000, 0xffffffffff5fe000, ^uint64(0)}
			for _, a := range seeds {
				for _, b := range seeds {
					appendTrial(a, b)
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
				appendTrial(a, random)
			}
			input, err := json.Marshal(struct {
				Code   []byte
				Trials []trial
			}{code, trials})
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, node, "-e", `
const fs=require('node:fs'),p=JSON.parse(fs.readFileSync(0,'utf8'));
const memory=new WebAssembly.Memory({initial:1}),v=new DataView(memory.buffer);
const run=new WebAssembly.Instance(new WebAssembly.Module(Buffer.from(p.Code,'base64')),{env:{memory}}).exports.run;
for(let n=0;n<p.Trials.length;n++) {
 const t=p.Trials[n];
 t.Initial.forEach((x,i)=>v.setBigUint64(8*i,BigInt(x),true));
 run(0,32768);
 t.Expected.forEach((x,i)=>{const got=v.getBigUint64(8*i,true);if(got!==BigInt(x))throw Error('trial='+n+' slot='+i+' a='+t.Initial[0]+' b='+t.Initial[1]+' got='+got+' want='+x)});
}
console.log('PASS');`)
			cmd.Stdin = bytes.NewReader(input)
			output, err := cmd.CombinedOutput()
			if err != nil || string(output) != "PASS\n" {
				t.Fatalf("generated predicates: %v\n%s", err, output)
			}
		})
	}
}
