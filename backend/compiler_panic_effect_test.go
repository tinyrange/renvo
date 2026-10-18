package main

import (
	"os"
	"testing"
)

func TestWasmEmittedPanicEffectsIncludeImplicitFaultsAndCycles(t *testing.T) {
	source, err := os.ReadFile("tests/panic_effect_calls.go")
	if err != nil {
		t.Fatal(err)
	}
	resetRuntime()
	c := renvoNewCompileContext(renvoTargetWasiWasm32, false, false, false)
	prog := renvoParseProgramWithContext(source, c)
	var meta renvoMeta
	renvoBuildMetaInto(&prog, &meta)
	if !prog.ok || !meta.ok || !meta.panicEnabled {
		t.Fatal("panic metadata")
	}
	meta.arenaSize = renvoResolveArenaSize(renvoTargetWasiWasm32, 0)
	g := renvoBeginScalarProgram(&prog, &meta)
	if g == nil {
		t.Fatal("begin")
	}
	if !renvoEmitAllQueuedFunctionsScratch(g) {
		t.Fatal("emission")
	}
	before := append([]renvoPanicControlRange(nil), g.panicControls...)
	compiled := renvoFinishScalarProgram(g)
	if !compiled.ok {
		t.Fatal("finalization")
	}
	found := map[string]bool{}
	for i := range g.panicEffects {
		fn := meta.funcs[i]
		name := string(prog.src[fn.nameStart:fn.nameEnd])
		switch name {
		case "pureCycleA", "pureCycleB", "pureForward":
			if g.panicEffects[i].raises || g.panicEffects[i].needsEntry {
				t.Fatalf("pure cycle retained panic effect: %s", name)
			}
			found[name] = true
		case "faultIndex", "faultPointer", "faultDivision", "faultAllocation", "faultAssertion", "faultForward":
			if !g.panicEffects[i].raises {
				t.Fatalf("implicit fault lost panic effect: %s", name)
			}
			found[name] = true
		}
	}
	if len(found) != 9 {
		t.Fatalf("missing functions: %v", found)
	}
	removedCalls, retainedCalls := 0, 0
	for _, span := range before {
		if span.callee < 0 {
			continue
		}
		allNop := true
		for _, op := range g.asm.code[span.start:span.end] {
			if int(op) != renvoWasm32OpNop {
				allNop = false
				break
			}
		}
		if !g.panicEffects[span.callee].raises {
			if !allNop {
				t.Fatal("proved-safe call still checked")
			}
			removedCalls++
		} else {
			if allNop {
				t.Fatal("faulting call check removed")
			}
			retainedCalls++
		}
	}
	if removedCalls == 0 || retainedCalls == 0 {
		t.Fatalf("removed=%d retained=%d", removedCalls, retainedCalls)
	}
}
