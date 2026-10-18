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

func wasmLoopTestLayout() WasmMemoryLayout {
	return WasmMemoryLayout{
		Retired: 0, Status: 8, Address: 16, Clock: 24, Pages: 64,
		PageSize: 32, PageNumber: 0, PageData: 8, PagePermissions: 16, PageEpoch: 24,
		Total: 2112, Remaining: 2120, MemoryTotal: 2128, LoopExits: 2136, LoopIterations: 2144,
		MemoryRetired: 2152, Linked: 2160, Blocks: 2192,
		DescriptorSize: 64, DescriptorPC: 0, DescriptorEntry: 8, DescriptorInstructions: 16,
	}
}

// Node executes the emitted bytes and the real proof-checked WASM dispatcher.
// The ordinary one-iteration emitter remains the differential reference.
func TestWasmResidentLoopBudgetsFaultsAndElapsed(t *testing.T) {
	t.Run("native-index", func(t *testing.T) { testWasmResidentLoopBudgetsFaultsAndElapsed(t, 64, false) })
	t.Run("browser-working-set", func(t *testing.T) { testWasmResidentLoopBudgetsFaultsAndElapsed(t, 4096, false) })
	t.Run("browser-index", func(t *testing.T) { testWasmResidentLoopBudgetsFaultsAndElapsed(t, 64, true) })
}
func testWasmResidentLoopBudgetsFaultsAndElapsed(t *testing.T, slots uint32, hashed bool) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node is required to execute generated WASM")
	}
	layout := wasmLoopTestLayout()
	layout.HashPages = hashed
	layout.PageSlots = slots
	if slots > 64 {
		layout.Pages = 262144
	}
	dispatcher, err := EmitWasmDispatcher(6, 0, layout)
	if err != nil {
		t.Fatal(err)
	}
	type trial struct {
		Name                string
		Resident, Reference []byte
		State               []string
		Budget, Permissions uint64
		Clock               string
		ExpectedTotal       uint64
		ExpectedMemory      uint64
		ExpectedStatus      uint64
		ExpectedIterations  uint64
		Direct              bool
		RejectDirect        bool
		Unlinked            bool
		Owned               bool
		AliasState          bool
		Instructions        uint64
		SecondPage          bool
		PageNumber          string
	}
	var trials []trial
	for _, kind := range []string{"pure", "memory", "pair", "guard", "exit"} {
		kindStart := len(trials)
		var b Builder
		b.MemoryInstructionProgress = true
		b.Checkpoint(6, 0)
		count := b.Binary(Add, b.Load(1), b.Constant(1))
		b.Store(1, count)
		b.Store(0, b.Constant(0))
		b.Checkpoint(6, 1)
		if kind == "memory" || kind == "pair" {
			address := b.Load(2)
			width := uint64(8)
			if kind == "pair" {
				b.MemoryPairStore(address, count, b.Constant(77))
				width = 16
			} else {
				b.MemoryStore(address, count, 8)
				// Two accesses within one guest instruction count only once.
				b.Store(5, b.MemoryLoad(address, 8))
			}
			b.Store(2, b.Binary(Add, address, b.Constant(width)))
		}
		if kind == "guard" {
			b.RegionGuard(b.Binary(Less, count, b.Constant(3)))
		}
		b.Store(3, b.Elapsed(b.Load(4), 2, 2))
		condition := b.Constant(1)
		if kind == "exit" {
			condition = b.Binary(Less, count, b.Constant(3))
			b.Store(0, b.Choose(condition, b.Constant(0), b.Constant(4)))
		}
		b.Checkpoint(6, 2)
		b.LoopContinue(condition)
		ops := b.FinishMemory(6)
		resident, err := emitWasmLoop(ops, 6, 2, &layout)
		if err != nil {
			t.Fatal(err)
		}
		reference, err := EmitWasmMemoryBlock(ops, 6, layout)
		if err != nil {
			t.Fatal(err)
		}
		for _, budget := range []uint64{1, 2, 3, 4, 5, 32, 65536} {
			if kind != "pure" && budget > 32 {
				continue
			}
			iterations, total, status := budget/2, budget/2*2, uint64(0)
			if kind == "guard" && budget >= 6 {
				iterations, total, status = 2, 5, RegionExit
			}
			if kind == "exit" && budget >= 6 {
				iterations, total = 3, 6
			}
			memory := uint64(0)
			if kind == "memory" || kind == "pair" {
				memory = iterations
			}
			trials = append(trials, trial{Name: kind + "/" + strconv.FormatUint(budget, 10), Resident: resident, Reference: reference, State: []string{"0", "0", "4096", "0", "100", "0"}, Budget: budget, Permissions: 3, Clock: "5", ExpectedTotal: total, ExpectedMemory: memory, ExpectedStatus: status, ExpectedIterations: iterations})
		}
		if kind == "memory" || kind == "pair" {
			address := "8184"
			if kind == "pair" {
				address = "8176"
			}
			trials = append(trials, trial{Name: kind + "/fault-after-iteration", Resident: resident, Reference: reference, State: []string{"0", "0", address, "0", "100", "0"}, Budget: 64, Permissions: 3, Clock: "5", ExpectedTotal: 3, ExpectedMemory: 1, ExpectedStatus: 1, ExpectedIterations: 1})
			trials = append(trials, trial{Name: kind + "/readonly", Resident: resident, Reference: reference, State: []string{"0", "0", "4096", "0", "100", "0"}, Budget: 64, Permissions: 1, Clock: "5", ExpectedTotal: 1, ExpectedStatus: 1})
			trials = append(trials, trial{Name: kind + "/clock-exhausted", Resident: resident, Reference: reference, State: []string{"0", "0", "4096", "0", "100", "0"}, Budget: 64, Permissions: 3, Clock: "18446744073709551615", ExpectedTotal: 1, ExpectedStatus: 1})
		}
		if kind == "pure" {
			trials = append(trials, trial{Name: "resident-leaf-internal-backedge", Resident: resident, Reference: reference, State: []string{"0", "0", "4096", "0", "100", "0"}, Budget: 32, Permissions: 3, Clock: "5", ExpectedTotal: 30, ExpectedIterations: 16, Direct: true})
			for _, budget := range []uint64{0, 1, 65537} {
				trials = append(trials, trial{Name: "reject-direct-budget", Resident: resident, State: []string{"0", "0", "4096", "0", "100", "0"}, Budget: budget, Permissions: 3, Clock: "5", ExpectedStatus: 3, Direct: true, RejectDirect: true})
			}
			trials = append(trials, trial{Name: "reject-direct-unlinked", Resident: resident, State: []string{"0", "0", "4096", "0", "100", "0"}, Budget: 32, Permissions: 3, Clock: "5", ExpectedStatus: 3, Direct: true, RejectDirect: true, Unlinked: true})
		}
		owned, err := emitWasmLoopMode(ops, 6, 2, &layout, true)
		if err != nil {
			t.Fatal(err)
		}
		kindEnd := len(trials)
		for i := kindStart; i < kindEnd; i++ {
			trial := trials[i]
			if trial.Direct || trial.Clock == "18446744073709551615" {
				continue
			}
			trial.Name, trial.Resident, trial.Owned = "owned/"+trial.Name, owned, true
			trials = append(trials, trial)
		}
		if kind == "memory" {
			// Generic guest data is allowed to alias architectural state. The
			// store changes the elapsed-counter input before its LoadState.
			trials = append(trials, trial{Name: "generic/state-alias", Resident: resident, Reference: reference, State: []string{"0", "0", "4128", "0", "100", "0"}, Budget: 4, Permissions: 3, Clock: "5", ExpectedTotal: 4, ExpectedMemory: 2, ExpectedIterations: 2, AliasState: true})
		}
	}
	for _, kind := range []string{"span", "mixed", "guarded-span", "negative-span", "negative-offset", "overlap"} {
		var b Builder
		b.MemoryInstructionProgress = true
		b.Checkpoint(6, 0)
		address, count := b.Load(2), b.Load(1)
		for i := 0; i < 4; i++ {
			at := b.Binary(Add, address, b.Constant(uint64(i*16)))
			if kind == "negative-offset" {
				at = b.Binary(Sub, b.Binary(Add, b.Constant(uint64(i*16)), address), b.Constant(16))
			}
			if kind == "overlap" {
				at = b.Binary(Add, address, b.Constant(uint64(i*8)))
				value := b.MemoryLoad(at, 8)
				b.MemoryStore(b.Binary(Add, at, b.Constant(8)), value, 8)
				b.Store(5, value)
			} else if kind == "mixed" && i == 0 {
				b.Store(5, b.MemoryLoad(at, 8))
			} else {
				b.MemoryPairStore(at, count, b.Constant(uint64(i+77)))
			}
			b.Checkpoint(6, i+1)
			if kind == "guarded-span" && i == 1 {
				b.RegionGuard(b.Binary(Less, count, b.Constant(1)))
			}
		}
		stride := uint64(64)
		if kind == "negative-span" {
			stride = ^uint64(63)
		}
		b.Store(2, b.Binary(Add, address, b.Constant(stride)))
		count = b.Binary(Add, count, b.Constant(1))
		b.Store(1, count)
		b.Store(3, b.Elapsed(b.Load(4), 5, 2))
		condition := b.Binary(Less, count, b.Constant(4))
		b.Store(0, b.Choose(condition, b.Constant(0), b.Constant(4)))
		b.Checkpoint(6, 5)
		b.LoopContinue(condition)
		ops := b.FinishMemory(6)
		resident, err := emitWasmLoopMode(ops, 6, 5, &layout, true)
		if err != nil {
			t.Fatal(err)
		}
		reference, err := EmitWasmMemoryBlock(ops, 6, layout)
		if err != nil {
			t.Fatal(err)
		}
		base := "4096"
		if kind == "negative-span" {
			base = "4352"
		}
		if kind == "negative-offset" {
			base = "4112"
		}
		for _, budget := range []uint64{1, 4, 5, 6, 9, 10, 11, 20, 21, 65536} {
			iterations := min(budget/5, 4)
			total, memory, status := iterations*5, iterations*4, uint64(0)
			if kind == "guarded-span" && budget >= 10 {
				iterations, total, memory, status = 1, 7, 6, RegionExit
			}
			trials = append(trials, trial{Name: kind + "/" + strconv.FormatUint(budget, 10), Resident: resident, Reference: reference, State: []string{"0", "0", base, "0", "100", "99"}, Budget: budget, Permissions: 3, Clock: "5", Owned: true, Instructions: 5, ExpectedTotal: total, ExpectedMemory: memory, ExpectedStatus: status, ExpectedIterations: iterations})
		}
		if kind == "span" {
			trials = append(trials, trial{Name: "span/missing-later-page", Resident: resident, Reference: reference, State: []string{"0", "0", "8160", "0", "100", "99"}, Budget: 64, Permissions: 3, Clock: "5", Owned: true, Instructions: 5, ExpectedTotal: 2, ExpectedMemory: 2, ExpectedStatus: 1})
			trials = append(trials, trial{Name: "span/cross-page-fallback-then-fast", Resident: resident, Reference: reference, State: []string{"0", "0", "8160", "0", "100", "99"}, Budget: 64, Permissions: 3, Clock: "5", Owned: true, Instructions: 5, SecondPage: true, ExpectedTotal: 20, ExpectedMemory: 16, ExpectedIterations: 4})
			trials = append(trials, trial{Name: "span/uint64-wrap", Resident: resident, Reference: reference, State: []string{"0", "0", "18446744073709551552", "0", "100", "99"}, PageNumber: "4503599627370495", Budget: 64, Permissions: 3, Clock: "5", Owned: true, Instructions: 5, ExpectedTotal: 5, ExpectedMemory: 4, ExpectedStatus: 1, ExpectedIterations: 1})
			for _, permissions := range []uint64{1, 2, 7} {
				total, memory, iterations, status := uint64(0), uint64(0), uint64(0), uint64(1)
				if permissions == 2 {
					total, memory, iterations, status = 20, 16, 4, 0
				}
				trials = append(trials, trial{Name: "span/permissions/" + strconv.FormatUint(permissions, 10), Resident: resident, Reference: reference, State: []string{"0", "0", "4096", "0", "100", "99"}, Budget: 64, Permissions: permissions, Clock: "5", Owned: true, Instructions: 5, ExpectedTotal: total, ExpectedMemory: memory, ExpectedStatus: status, ExpectedIterations: iterations})
			}
		}
		if kind == "mixed" {
			trials = append(trials, trial{Name: "span/readonly-after-read", Resident: resident, Reference: reference, State: []string{"0", "0", "4096", "0", "100", "99"}, Budget: 64, Permissions: 1, Clock: "5", Owned: true, Instructions: 5, ExpectedTotal: 1, ExpectedMemory: 1, ExpectedStatus: 1})
		}
	}
	payload, err := json.Marshal(struct {
		Layout     WasmMemoryLayout
		Dispatcher []byte
		Proof      []byte
		Trials     []trial
	}{layout, dispatcher, emitWasmProof(), trials})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "-e", wasmLoopTestJS)
	cmd.Stdin = bytes.NewReader(payload)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated WASM: %v\n%s", err, output)
	}
}

const wasmLoopTestJS = `
const fs = require('node:fs');
const p = JSON.parse(fs.readFileSync(0, 'utf8')), l=p.Layout;
const module = code => new WebAssembly.Module(Buffer.from(code,'base64'));
const dispatcher=module(p.Dispatcher), proof=module(p.Proof);
function run(t, resident) {
 const memory=new WebAssembly.Memory({initial:8}), v=new DataView(memory.buffer);
 const state=4096, context=8192, data=131072, clock=98304, epoch=98312;
 for(let i=0;i<8192;i++)v.setUint8(data+i,(i*37+129)&255);
 const get=o=>v.getBigUint64(context+o,true), set=(o,x)=>v.setBigUint64(context+o,BigInt(x),true);
 const leaf=new WebAssembly.Instance(module(resident?t.Resident:t.Reference),{env:{memory}});
 const table=new WebAssembly.Table({initial:4,element:'anyfunc'});
 const admission=new WebAssembly.Instance(proof,{proof:{pc:0n,entry:16n,instructions:BigInt(t.Instructions||2),words:6}});
 table.set(2,leaf.exports.run);table.set(3,admission.exports.prove);
 const d=new WebAssembly.Instance(dispatcher,{env:{memory,table}});
 for(let i=0;i<t.State.length;i++)v.setBigUint64(state+i*8,BigInt(t.State[i]),true);
 v.setBigUint64(clock,BigInt(t.Clock),true);v.setBigUint64(epoch,3n,true);v.setBigUint64(epoch+8,3n,true);
 v.setUint32(context+l.Clock,clock,true);
 const slot=n=>Number((l.HashPages?n^(n>>6n)^(n>>12n):n)&BigInt((l.PageSlots||64)-1));
 const pageNumber=BigInt(t.PageNumber||1),page=context+l.Pages+slot(pageNumber)*l.PageSize;
 v.setBigUint64(page+l.PageNumber,pageNumber,true);v.setUint32(page+l.PageData,t.AliasState?state:data,true);
 v.setBigUint64(page+l.PagePermissions,BigInt(t.Permissions),true);v.setUint32(page+l.PageEpoch,epoch,true);
 if(t.SecondPage){const second=context+l.Pages+slot(pageNumber+1n)*l.PageSize;v.setBigUint64(second+l.PageNumber,pageNumber+1n,true);v.setUint32(second+l.PageData,data+4096,true);v.setBigUint64(second+l.PagePermissions,BigInt(t.Permissions),true);v.setUint32(second+l.PageEpoch,epoch+8,true);}
 const descriptor=context+l.Blocks;
 v.setBigUint64(descriptor+l.DescriptorPC,0n,true);v.setBigUint64(descriptor+l.DescriptorEntry,16n,true);v.setBigUint64(descriptor+l.DescriptorInstructions,BigInt(t.Instructions||2),true);
 set(l.Remaining,t.Budget);set(l.Linked,t.Unlinked?0:1);
 if(t.Direct)leaf.exports.run(state,context);else d.exports.run(state,context);
 const result={state:t.State.map((_,i)=>String(v.getBigUint64(state+i*8,true))),total:String(get(l.Total)),remaining:String(get(l.Remaining)),memory:String(get(l.MemoryTotal)),status:String(get(l.Status)),address:String(get(l.Address)),retired:String(get(l.Retired)),memoryRetired:String(get(l.MemoryRetired)),clock:String(v.getBigUint64(clock,true)),epoch:String(v.getBigUint64(epoch,true)),epoch2:String(v.getBigUint64(epoch+8,true)),data:Buffer.from(memory.buffer,data,8192).toString('base64')};
 return {result,iterations:Number(get(l.LoopIterations))};
}
for(const t of p.Trials){
 const got=run(t,true);
 if(!t.Direct){const reference=run(t,false);if(t.Owned){if(got.result.clock!==t.Clock||got.result.epoch!=='3'||got.result.epoch2!=='3')throw Error(t.Name+' mutated owned code generation');reference.result.clock=got.result.clock;reference.result.epoch=got.result.epoch;reference.result.epoch2=got.result.epoch2;}if(JSON.stringify(got.result)!==JSON.stringify(reference.result))throw Error(t.Name+' differential mismatch '+JSON.stringify({got,reference}));}
 if(got.result.total!==String(t.ExpectedTotal)||got.result.memory!==String(t.ExpectedMemory)||got.result.status!==String(t.ExpectedStatus)||got.iterations!==t.ExpectedIterations)throw Error(t.Name+' accounting mismatch '+JSON.stringify(got));
 if(t.Direct&&!t.RejectDirect&&(got.result.retired!=='2'||got.result.remaining!=='2'||got.result.state[1]!=='16'||got.result.state[3]!=='164'))throw Error('leaf did not repeat internally '+JSON.stringify(got));
 if(t.RejectDirect&&(got.result.state[1]!=='0'||got.result.clock!=='5'))throw Error('invalid direct entry had effects '+JSON.stringify(got));
}
`

func TestWasmLoopRejectsInvalidProgress(t *testing.T) {
	layout := wasmLoopTestLayout()
	for _, ops := range [][]Op{
		nil,
		{{Kind: Const, Imm: 1}, {Kind: Progress, Imm: 1}, {Kind: LoopContinue, A: 0}},
		{{Kind: Const, Imm: 1}, {Kind: Progress, Imm: 2}, {Kind: LoopContinue, A: 0}, {Kind: LoopContinue, A: 0}},
		{{Kind: Const, Imm: 1}, {Kind: Progress, Imm: 2}, {Kind: Progress, Imm: 1}, {Kind: LoopContinue, A: 0}},
		{{Kind: Const, Imm: 1}, {Kind: Progress, Imm: 1}, {Kind: Progress, B: 1, Imm: 2}, {Kind: LoopContinue, A: 0}},
	} {
		if _, err := emitWasmLoop(ops, 1, 2, &layout); err == nil {
			t.Fatal("invalid loop accepted", ops)
		}
	}
}
