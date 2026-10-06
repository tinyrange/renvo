package asmunit

import (
	"os"
	"path/filepath"
	"renvo.dev/internal/rtg"
	"renvo.dev/internal/unit"
	"strings"
	"testing"
)

type fileLoader struct{}

func (fileLoader) LoadImport(from, path string) rtg.ImportSource {
	path = filepath.Join(filepath.Dir(from), path)
	data, err := os.ReadFile(path)
	return rtg.ImportSource{Source: data, Filename: path, Ok: err == nil}
}

type recordingCompiler struct {
	calls    int
	bytecode []byte
	ok       bool
}

func (c *recordingCompiler) CompileAssemblyEvaluator(source []byte) ([]byte, string, bool) {
	c.calls++
	if len(source) == 0 {
		panic("empty evaluator")
	}
	return c.bytecode, "offline", c.ok
}

func TestMaterializerValidatesBeforeCompilerAndNeverTrustsCode(t *testing.T) {
	path := filepath.Join("..", "..", "backend", "definitions", "linux_amd64.rtg")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	resolved := rtg.Resolve(rtg.ParseImports(source, path, fileLoader{}))
	if !resolved.Ok {
		t.Fatal(resolved.Diagnostics)
	}
	build := func(body string, entry int) []byte {
		data, ok := unit.MarshalCore(unit.CoreProgram{
			Funcs:            []unit.Func{{}},
			RTGAssembly:      []unit.RTGAssemblySource{{Path: "answer.rtgasm", Source: []byte(body)}},
			RTGAssemblyFuncs: []unit.RTGAssemblyBinding{{Func: 0, Source: 0, Entry: entry, Code: []byte{0xcc}}},
		})
		if !ok {
			t.Fatal("marshal")
		}
		return data
	}
	good := build("rtgasm 2 assembly {answer(out:emitter){move_immediate(register(rax),int64(42));return()}}", 0)
	compiler := &recordingCompiler{}
	result := MaterializeBounded(good, resolved, "linux/amd64", compiler, 1024*1024)
	if result.Ok || compiler.calls != 1 || len(result.Diagnostics) != 1 || result.Diagnostics[0].Filename != "answer.rtgasm" || !strings.Contains(result.Diagnostics[0].Message, "offline") {
		t.Fatalf("compiler failure/code bypass: %+v calls=%d", result, compiler.calls)
	}
	for _, bad := range []struct {
		body  string
		entry int
	}{
		{"rtgasm 2 assembly {answer(out:emitter){emitReturn(out)}}", 0},
		{"rtgasm 3 assembly {answer(out:emitter){inputs=7;yield()}}", 0},
		{"rtgasm 2 assembly {answer(out:emitter){return()}}", 1},
	} {
		compiler.calls = 0
		result = Materialize(build(bad.body, bad.entry), resolved, "linux/amd64", compiler)
		if result.Ok || compiler.calls != 0 || len(result.Diagnostics) == 0 {
			t.Fatalf("invalid source reached compiler: %+v", result)
		}
	}
	compiler.calls = 0
	if MaterializeBounded(good, resolved, "linux/amd64", compiler, 96*1024*1024+1).Ok || compiler.calls != 0 {
		t.Fatal("memory ceiling bypass")
	}
	if MaterializeBounded(good, resolved, "linux/amd64", nil, 1024).Ok {
		t.Fatal("accepted missing compiler")
	}
	compiler.ok = true
	compiler.bytecode = []byte("invalid VM bytecode")
	if MaterializeBounded(good, resolved, "linux/amd64", compiler, 1024).Ok {
		t.Fatal("VM trap became successful emission")
	}
	plain, ok := unit.MarshalCore(unit.CoreProgram{})
	if !ok {
		t.Fatal("plain marshal")
	}
	result = Materialize(plain, resolved, "linux/amd64", nil)
	if !result.Ok || string(result.Unit) != string(plain) {
		t.Fatal("plain unit not preserved")
	}
	if Materialize(append(plain, 33), resolved, "linux/amd64", nil).Ok {
		t.Fatal("malformed stream accepted without assembly")
	}
}
