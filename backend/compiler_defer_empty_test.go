package main

import (
	"os"
	"renvo.dev/std/vm"
	"testing"
)

func TestNoDeferHelperPanicOwnershipAndNoDispatchLoop(t *testing.T) {
	source, err := os.ReadFile("tests/no_defer_control_propagation.go")
	if err != nil {
		t.Fatal(err)
	}
	resetRuntime()
	image, ok := RenvoCompileSourceToBytesWithOptions(source, "vm/vm32", RenvoCompileOptions{ArenaSize: 8 * 1024 * 1024, StripSymbols: true})
	if !ok {
		t.Fatal("VM compile")
	}
	result := vm.RunConfig(image, vm.Config{Limits: vm.Limits{Steps: 10000000, Memory: 16 * 1024 * 1024}})
	if result.Trap != vm.TrapNone || result.ExitCode != 0 || string(result.Output) != "PASS\n" || len(result.Stderr) != 0 {
		t.Fatalf("trap=%d exit=%d output=%q stderr=%q", result.Trap, result.ExitCode, result.Output, result.Stderr)
	}
	// The actual parsed helper still participates in a panic-enabled program,
	// but has no owned registrations and therefore needs no dispatch backedge.
	resetRuntime()
	c := renvoNewCompileContext(renvoTargetWasiWasm32, false, false, false)
	prog := renvoParseProgramWithContext(source, c)
	var meta renvoMeta
	renvoBuildMetaInto(&prog, &meta)
	if !prog.ok || !meta.ok || !meta.panicEnabled {
		t.Fatal("panic-enabled metadata")
	}
	meta.arenaSize = renvoResolveArenaSize(renvoTargetWasiWasm32, 0)
	g := renvoBeginScalarProgram(&prog, &meta)
	if g == nil || !renvoEmitAllQueuedFunctionsScratch(g) {
		t.Fatal("emission")
	}
	compiled := renvoFinishScalarProgram(g)
	if !compiled.ok {
		t.Fatal("Wasm image")
	}
	a := &g.asm
	pcs := renvoWasm32InstructionPcs(a.code)
	symbols := renvoWasm32SymbolPcs(a)
	routines := renvoWasm32RoutinePcs(a, a.code, pcs)
	ends := renvoWasm32RoutineEnds(routines, symbols, a.code, pcs)
	found := false
	for i := range a.symbols {
		if !a.SymbolNameEquals(i, "plain") {
			continue
		}
		start := a.SymbolPC(i)
		index := renvoWasm32FindRoutineIndex(routines, start)
		for _, pc := range pcs {
			if pc < start || pc >= ends[index] {
				continue
			}
			op := int(a.code[pc])
			if op < renvoWasm32OpJmp || op > renvoWasm32OpJCond {
				continue
			}
			field := 1
			if op == renvoWasm32OpJCond {
				field = 2
			}
			target := renvoGet32At(a.code, pc+field)
			if target <= pc {
				t.Fatalf("no-defer helper retains dispatch backedge pc=%d target=%d", pc, target)
			}
		}
		found = true
	}
	if !found {
		t.Fatal("parsed helper missing")
	}
}
