//go:build !renvo

package backendjit

import (
	"bytes"
	"path/filepath"
	"testing"

	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/unit"
)

func TestCompilerJITTypedUnitDoesNotTrustMaterializedBytes(t *testing.T) {
	if hostTarget() == "" {
		t.Skip("no in-process backend for this host")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	backend := New(filepath.Join(root, "backends", "msdos.rtg"), filepath.Join(root, "backend"), filepath.Join(root, "std"), backendJITTestCacheDir, backendcompiled.Backend{})
	prepared := backend.prepare("msdos/8086")
	if !prepared.Ok {
		t.Fatalf("prepare: %+v", prepared.Diagnostic)
	}
	core := unit.CoreProgram{Package: "main", Funcs: []unit.Func{{}},
		RTGAssembly:      []unit.RTGAssemblySource{{Path: "answer.rtgasm", Source: []byte("rtgasm 2 assembly { answer(out:emitter) { move_immediate(register(ax), int(42)); return() } }")}},
		RTGAssemblyFuncs: []unit.RTGAssemblyBinding{{Func: 0, Source: 0, Entry: 0, Code: []byte{0xcc}}},
	}
	data, ok := unit.MarshalCore(core)
	if !ok {
		t.Fatal("marshal")
	}
	evaluated, result := backend.evaluateRTGAssembly(data, prepared)
	if !result.Ok {
		t.Fatalf("evaluate: %+v", result.Diagnostic)
	}
	_, bindings, ok := readRTGAssembly(evaluated)
	if !ok || len(bindings) != 1 || !bytes.Equal(bindings[0].Code, []byte{0xb8, 42, 0, 0xc3}) {
		t.Fatal("typed unit accepted supplied machine bytes")
	}
	core.RTGAssembly[0].Source = []byte("rtgasm 2 assembly { answer(out:emitter) { emitReturn(out) } }")
	data, ok = unit.MarshalCore(core)
	if !ok {
		t.Fatal("marshal invalid body")
	}
	if _, result := backend.evaluateRTGAssembly(data, prepared); result.Ok || result.Diagnostic.Code != "RENVO-RTGASM-003" {
		t.Fatalf("materialized code bypassed typed source validation: %+v", result.Diagnostic)
	}
}
