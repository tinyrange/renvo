package main

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"os"
	"testing"
)

func Test386ObjectPanicStateNeedsNoExecutableStartup(t *testing.T) {
	oldTarget, oldArch, oldOS, oldObject := renvoTarget, renvoTargetArch, renvoTargetOS, renvoCompilerObjectFile
	t.Cleanup(func() {
		renvoTarget, renvoTargetArch, renvoTargetOS, renvoCompilerObjectFile = oldTarget, oldArch, oldOS, oldObject
	})
	renvoCompilerObjectFile = true
	renvoSetTarget(renvoTargetLinux386)
	source, err := os.ReadFile("tests/object_panic_state.go")
	if err != nil {
		t.Fatal(err)
	}
	context := renvoNewCompileContext(renvoTargetLinux386, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("object panic-state source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok || !meta.panicEnabled {
		t.Fatal("object panic-state metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinux386)
	result := renvoTryCompileScalarProgram386(&program, &meta)
	if !result.ok {
		t.Fatal("object panic-state compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	code, err := file.Section(".text").Data()
	if err != nil {
		t.Fatal(err)
	}
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	var prologue []byte
	for _, symbol := range symbols {
		if symbol.Name == "ObjectPanicState" {
			wrapper := code[symbol.Value : symbol.Value+symbol.Size]
			call := bytes.IndexByte(wrapper, 0xe8)
			if call < 0 || call+5 > len(wrapper) {
				t.Fatal("object wrapper does not call its Go body")
			}
			body := int(symbol.Value) + call + 5 + int(int32(binary.LittleEndian.Uint32(wrapper[call+1:])))
			prologue = code[body : body+64]
		}
	}
	if prologue == nil {
		t.Fatal("missing object export")
	}
	// A foreign caller starts with zeroed BSS and never initializes a thread
	// pointer. These indirect accesses previously faulted at NULL+0x10 on the
	// first Windows driver callback. The unwind slots must be direct BSS data.
	for _, instruction := range [][]byte{{0x8b, 0x42, 0x10}, {0x8b, 0x42, 0x28}, {0x89, 0x42, 0x28}} {
		if bytes.Contains(prologue, instruction) {
			t.Fatalf("foreign callback dereferences an uninitialized ThreadState pointer: %x", instruction)
		}
	}
}
