package main

import (
	"bytes"
	"debug/elf"
	"testing"
)

func Test386Code16RewriteRemapsObjectMetadata(t *testing.T) {
	emitter := renvoAsm{
		code:            []byte{0x55, 0x89, 0xe5, 0x8b, 0x45, 0x08, 0xe8, 0, 0, 0, 0, 0xc3},
		labelPos:        []int32{0, 11},
		relocs:          []int32{7, 1},
		absRelocs:       []int32{7, 0, renvoImportReloc},
		objectFunctions: []renvoObjectFunctionRange{{label: 0, end: 12}},
	}
	if !renvo386ApplyCode16(&emitter) {
		t.Fatal("code16 rewrite failed")
	}
	want := []byte{
		0x66, 0x55,
		0x66, 0x89, 0xe5,
		0x67, 0x66, 0x8b, 0x45, 0x08,
		0x66, 0xe8, 0, 0, 0, 0,
		0x66, 0xc3,
	}
	if !bytes.Equal(emitter.code, want) {
		t.Fatalf("code16 bytes = %x, want %x", emitter.code, want)
	}
	if emitter.labelPos[0] != 0 || emitter.labelPos[1] != 16 ||
		emitter.relocs[0] != 12 || emitter.absRelocs[0] != 12 ||
		emitter.objectFunctions[0].end != len(want) {
		t.Fatalf("remapped metadata = labels %v relocs %v absolute %v functions %#v",
			emitter.labelPos, emitter.relocs, emitter.absRelocs, emitter.objectFunctions)
	}
}

func Test386Code16RewritePreservesExplicit16BitOperand(t *testing.T) {
	got := renvo386RewriteCode16([]byte{0x66, 0x89, 0xc8})
	if !bytes.Equal(got, []byte{0x89, 0xc8}) {
		t.Fatalf("explicit 16-bit move became %x", got)
	}
}

func Test386Code16RelaxedJumpKeepsPC16Relocation(t *testing.T) {
	emitter := renvoAsm{
		code:     []byte{0xe9, 0, 0, 0, 0, 0x89, 0xc0, 0xc3},
		labelPos: []int32{7},
		relocs:   []int32{1, 0},
	}
	if !renvo386ApplyCode16(&emitter) {
		t.Fatal("code16 rewrite failed")
	}
	want := []byte{0xe9, 0, 0, 0x66, 0x89, 0xc0, 0x66, 0xc3}
	if !bytes.Equal(emitter.code, want) {
		t.Fatalf("relaxed code16 jump = %x, want %x", emitter.code, want)
	}
	if len(emitter.relocs) != 2 || emitter.relocs[0] >= 0 || int(emitter.relocs[0])&2147483647 != 1 || emitter.relocs[1] != 0 {
		t.Fatalf("relaxed PC16 relocation = %v", emitter.relocs)
	}
}

func Test386Code16CallKeeps32BitReturnAddress(t *testing.T) {
	emitter := renvoAsm{
		code:     []byte{0xe8, 0, 0, 0, 0, 0xc3},
		labelPos: []int32{5},
		relocs:   []int32{1, 0},
	}
	if !renvo386ApplyCode16(&emitter) {
		t.Fatal("code16 rewrite failed")
	}
	want := []byte{0x66, 0xe8, 0, 0, 0, 0, 0x66, 0xc3}
	if !bytes.Equal(emitter.code, want) {
		t.Fatalf("code16 call = %x, want %x", emitter.code, want)
	}
	if len(emitter.relocs) != 2 || emitter.relocs[0] != 2 || emitter.relocs[1] != 0 {
		t.Fatalf("code16 call relocation = %v", emitter.relocs)
	}
}

func Test386Code16ObjectPreservesWideIntegerInitializers(t *testing.T) {
	oldTarget := renvoTarget
	oldArch := renvoTargetArch
	oldOS := renvoTargetOS
	oldObjectFile := renvoCompilerObjectFile
	t.Cleanup(func() {
		renvoTarget = oldTarget
		renvoTargetArch = oldArch
		renvoTargetOS = oldOS
		renvoCompilerObjectFile = oldObjectFile
	})
	renvoCompilerObjectFile = true
	renvoSetTarget(renvoTargetLinux386)
	source := []byte(`package main
// renvo:object variable table .data 8 0 0 - 40 0 - 0
var table [5]uint64 = [5]uint64{0, 0, 58435744481476607, 58426948388454399, 150633361440871}
`)
	context := renvoNewCompileContext(renvoTargetLinux386, false, false, false)
	context.objectFile = true
	context.code16 = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("wide initializer object source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("wide initializer object metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinux386)
	result := renvoTryCompileScalarProgram386(&program, &meta)
	if !result.ok {
		t.Fatal("wide initializer object compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse wide initializer object: %v", err)
	}
	section := file.Section(".data")
	if section == nil {
		t.Fatal("wide initializer object has no .data section")
	}
	data, err := section.Data()
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0xff, 0xff, 0, 0, 0, 0x9b, 0xcf, 0,
		0xff, 0xff, 0, 0, 0, 0x93, 0xcf, 0,
		0x67, 0, 0, 0x10, 0, 0x89, 0, 0,
	}
	if !bytes.Equal(data, want) {
		t.Fatalf("wide initializer data = %x, want %x", data, want)
	}
}

func Test386Code16ObjectPointerStoreUsesNativeWidth(t *testing.T) {
	oldTarget := renvoTarget
	oldArch := renvoTargetArch
	oldOS := renvoTargetOS
	oldObjectFile := renvoCompilerObjectFile
	t.Cleanup(func() {
		renvoTarget = oldTarget
		renvoTargetArch = oldArch
		renvoTargetOS = oldOS
		renvoCompilerObjectFile = oldObjectFile
	})
	renvoCompilerObjectFile = true
	renvoSetTarget(renvoTargetLinux386)
	source := []byte(`package main
// renvo:c11
// renvo:object function store_pointer .text 0 2 0 - 4 4 0 - 0
//export store_pointer
func store_pointer(out **int32, value *int32) { *out = value }
`)
	context := renvoNewCompileContext(renvoTargetLinux386, false, false, false)
	context.objectFile = true
	context.code16 = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("pointer-store object source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("pointer-store object metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinux386)
	result := renvoTryCompileScalarProgram386(&program, &meta)
	if !result.ok {
		t.Fatal("pointer-store object compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse pointer-store object: %v", err)
	}
	section := file.Section(".text")
	if section == nil {
		t.Fatal("pointer-store object has no .text section")
	}
	data, err := section.Data()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte{0x67, 0x66, 0x89, 0x42, 0x04}) {
		t.Fatalf("pointer assignment wrote a second word past its 32-bit C object: %x", data)
	}
	if !bytes.Contains(data, []byte{0x67, 0x66, 0x89, 0x02}) {
		t.Fatalf("pointer assignment lacks its native-width store: %x", data)
	}
}
