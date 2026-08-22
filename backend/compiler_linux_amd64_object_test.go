package main

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"testing"
)

func TestLinuxAmd64ReverseObjectCallUsesSysVArgumentOrder(t *testing.T) {
	emitter := renvoAsm{staticImports: []renvoStaticImport{{dll: "libc", name: "foreign"}}}
	renvoAmd64EmitObjectStaticCallReverse(&emitter, 0, 6)
	wantPrefix := []byte("\x41\x59\x41\x58\x59\x5a\x5e\x5f")
	if !bytes.HasPrefix(emitter.code, wantPrefix) {
		t.Fatalf("reverse object call register pops = %x, want prefix %x", emitter.code, wantPrefix)
	}
	if len(emitter.absRelocs) != 3 || len(emitter.kernelImportOffsets) != 2 ||
		string(emitter.kernelImportNames) != "foreign" {
		t.Fatalf("reverse object call relocation/import metadata is incomplete: relocs=%v offsets=%v names=%q",
			emitter.absRelocs, emitter.kernelImportOffsets, emitter.kernelImportNames)
	}
}

func TestLinuxAmd64ObjectUserAccessInstructionsRequireSMAPEnabled(t *testing.T) {
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	source := []byte(`package main
func renvo_runtime_CEnableUserAccess() {}
func renvo_runtime_CDisableUserAccess() {}
// renvo:object function access .text 0 0 0 - 0 - 0
//export access
func access() {
	renvo_runtime_CEnableUserAccess()
	renvo_runtime_CDisableUserAccess()
}
`)
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("guarded user-access source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("guarded user-access metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("guarded user-access object compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse guarded user-access object: %v", err)
	}
	var textData []byte
	for _, section := range file.Sections {
		if section.Flags&elf.SHF_EXECINSTR == 0 {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		textData = append(textData, data...)
	}
	guard := []byte("\x0f\x20\xe0\x0f\xba\xe0\x15\x73\x03")
	for _, instruction := range [][]byte{{0x0f, 0x01, 0xcb}, {0x0f, 0x01, 0xca}} {
		if !bytes.Contains(textData, append(append([]byte{}, guard...), instruction...)) {
			t.Fatalf("user-access instruction %x is not guarded by CR4.SMAP: code=%x", instruction, textData)
		}
	}
}

func TestLinuxAmd64ObjectDoesNotFoldLoopVaryingSingleCallArgument(t *testing.T) {
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	source := []byte(`package main
// renvo:c11
var values [3]uint64 = [3]uint64{11, 17, 23}
func lookup(index int32) uint64 { return values[index] }
// renvo:object function sum .text 0 1 0 - 8 0 - 0
//export sum
func sum() uint64 {
	var result uint64
	for index := int32(0); index < 3; index++ {
		result += lookup(index)
	}
	return result
}
`)
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("loop-varying single-call source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("loop-varying single-call metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("loop-varying single-call object compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse loop-varying single-call object: %v", err)
	}
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	for _, symbol := range symbols {
		if symbol.Name != "lookup" || symbol.Section == elf.SHN_UNDEF || symbol.Size == 0 {
			continue
		}
		section := file.Sections[symbol.Section]
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		start := int(symbol.Value - section.Addr)
		end := start + int(symbol.Size)
		if start < 0 || end > len(data) {
			t.Fatalf("lookup symbol range [%d,%d) exceeds section size %d", start, end, len(data))
		}
		// The first and only parameter is saved at -8(%rbp). Its use must
		// survive compilation because the single syntactic call is inside a
		// loop and therefore observes a different value on each execution.
		loadsIndex := false
		for at := start; at+4 <= end; at++ {
			// MOV r64,-8(%rbp); the destination register is an allocator choice.
			if data[at] == 0x48 && data[at+1] == 0x8b && data[at+2]&0xc7 == 0x45 && data[at+3] == 0xf8 {
				loadsIndex = true
				break
			}
		}
		if !loadsIndex {
			t.Fatalf("lookup folded its loop-varying index: code=%x", data[start:end])
		}
		return
	}
	t.Fatal("lookup function symbol not found")
}

func TestLinuxAmd64InvalidateProcessContextUsesDescriptorMemoryOperand(t *testing.T) {
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	source := []byte(`package main
func renvo_runtime_CInvalidateProcessContext(address uintptr, kind uintptr) {}
// renvo:object function invalidate - 0 2 0 - 8 8 0 - 0
//export invalidate
func invalidate(address uintptr, kind uintptr) {
	renvo_runtime_CInvalidateProcessContext(address, kind)
}
`)
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("INVPCID operand source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("INVPCID operand metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("INVPCID operand object compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse INVPCID operand object: %v", err)
	}
	want := []byte("\x58\x5a\x66\x0f\x38\x82\x02")
	wrong := []byte("\x5a\x58\x66\x0f\x38\x82\x02")
	for _, section := range file.Sections {
		if section.Flags&elf.SHF_EXECINSTR == 0 {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(data, wrong) {
			t.Fatal("INVPCID uses its kind as the descriptor address")
		}
		if bytes.Contains(data, want) {
			return
		}
	}
	t.Fatal("INVPCID descriptor/kind register sequence not found")
}

func TestLinuxAmd64C11ObjectOmitsGoNilDereferenceCheck(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)

	compileSymbols := func(source string) []elf.Symbol {
		t.Helper()
		context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
		context.objectFile = true
		program := renvoParseProgramWithContext([]byte(source), context)
		if !program.ok {
			t.Fatal("pointer-dereference source did not parse")
		}
		var meta renvoMeta
		renvoBuildMetaInto(&program, &meta)
		if !meta.ok {
			t.Fatal("pointer-dereference metadata failed")
		}
		meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
		result := renvoTryCompileScalarProgramAmd64(&program, &meta)
		if !result.ok {
			t.Fatal("pointer-dereference object compilation failed")
		}
		file, err := elf.NewFile(bytes.NewReader(result.data))
		if err != nil {
			t.Fatalf("parse pointer-dereference object: %v", err)
		}
		symbols, err := file.Symbols()
		if err != nil {
			t.Fatal(err)
		}
		return symbols
	}

	source := `package main
// renvo:object function read - 0 1 0 - 8 0 - 0
//export read
func read(value *int) int { return *value }
`
	goSymbols := compileSymbols(source)
	cSymbols := compileSymbols("package main\n// renvo:c11\n" + source[len("package main\n"):])
	hasNonNilCheck := func(symbols []elf.Symbol) bool {
		for _, symbol := range symbols {
			if symbol.Name == "__renvo_check_non_nil" && symbol.Section != elf.SHN_UNDEF {
				return true
			}
		}
		return false
	}
	if !hasNonNilCheck(goSymbols) {
		t.Fatal("ordinary Go object lost its nil-dereference trap")
	}
	if hasNonNilCheck(cSymbols) {
		t.Fatal("C11 object retained a Go-only nil-dereference trap")
	}
}

func TestLinuxAmd64C11ObjectDoesNotEnableGoPanicStateForCIdentifier(t *testing.T) {
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	panicEnabled := func(marker string) bool {
		source := []byte("package main\n" + marker + "var panic int\nfunc read() int { return panic }\n")
		program := renvoParseProgramWithContext(source, context)
		if !program.ok {
			t.Fatal("panic-identifier source did not parse")
		}
		var meta renvoMeta
		renvoBuildMetaInto(&program, &meta)
		if !meta.ok {
			t.Fatal("panic-identifier metadata failed")
		}
		return meta.panicEnabled
	}
	if !panicEnabled("") {
		t.Fatal("ordinary Go source lost conservative panic state")
	}
	if panicEnabled("// renvo:c11\n") {
		t.Fatal("C11 identifier named panic enabled Go panic state")
	}
	cleanupSource := []byte("package main\n// renvo:c11\nfunc cleanup(*int) {}\nfunc read() int { value := 1; defer cleanup(&value); return value }\n")
	program := renvoParseProgramWithContext(cleanupSource, context)
	if !program.ok {
		t.Fatal("C11 cleanup source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("C11 cleanup metadata failed")
	}
	if !meta.panicEnabled {
		t.Fatal("C11 cleanup defer lost required unwind state")
	}
	cleanupStart := bytes.Index(cleanupSource, []byte("cleanup"))
	readStart := bytes.Index(cleanupSource, []byte("read"))
	cleanupIndex := renvoFindMetaFunction(&meta, cleanupStart, cleanupStart+len("cleanup"))
	readIndex := renvoFindMetaFunction(&meta, readStart, readStart+len("read"))
	if cleanupIndex < 0 || readIndex < 0 {
		t.Fatalf("C11 cleanup functions not found: cleanup=%d read=%d", cleanupIndex, readIndex)
	}
	cleanupHasDefer := renvoTokenRangeHasIdent(&program, meta.funcs[cleanupIndex].bodyStart, meta.funcs[cleanupIndex].bodyEnd, "defer")
	readHasDefer := renvoTokenRangeHasIdent(&program, meta.funcs[readIndex].bodyStart, meta.funcs[readIndex].bodyEnd, "defer")
	if cleanupHasDefer || !readHasDefer {
		t.Fatalf("C11 per-function defer scan: cleanup=%t read=%t", cleanupHasDefer, readHasDefer)
	}
}

func TestLinuxAmd64ObjectArenaHelperHasFunctionSymbol(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
// renvo:object function initValue - 0 0 0 - 8 0 - 0
func initValue(value int) int { return sharedValue(value) }
// renvo:object function sharedValue - 0 0 0 - 8 0 - 0
func sharedValue(value int) int { return value }
// renvo:object function callbackValue .init.text 0 0 0 - 0 0 - 0
func callbackValue() {}
type callbackTable struct { callback func() }
// renvo:object variable callbackTableValue .data 8 0 0 - 8 0 - 0
var callbackTableValue callbackTable = callbackTable{callback: callbackValue}
// renvo:object function initCallbackValue .init.text 0 0 0 - 0 0 - 0
func initCallbackValue() {}
// renvo:object variable initCallbackTableValue .init.data 8 0 0 - 8 0 - 0
var initCallbackTableValue callbackTable = callbackTable{callback: initCallbackValue}
// renvo:object function earlyCallbackValue .init.text 0 0 0 - 0 0 - 0
func earlyCallbackValue() {}
// renvo:object variable earlyCallbackTableValue __earlycon_table 8 0 0 - 8 0 - 0
var earlyCallbackTableValue callbackTable = callbackTable{callback: earlyCallbackValue}
// renvo:object function consoleInitCallbackValue .init.text 0 0 0 - 0 0 - 0
func consoleInitCallbackValue() {}
// renvo:object variable consoleInitCallbackTableValue .con_initcall.init 8 0 0 - 8 0 - 0
var consoleInitCallbackTableValue callbackTable = callbackTable{callback: consoleInitCallbackValue}
// renvo:object function refValue .ref.text 0 1 0 - 8 0 - 0
//export refValue
func refValue(value int) int { return sharedValue(value) }
// renvo:object function allocate_length .init.text 0 1 0 - 8 0 - 0
//export allocate_length
func allocate_length(length int) int { values := make([]byte, length); callbackValue(); return len(values) + initValue(length) }
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("arena-helper source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("arena-helper metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("arena-helper object compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse arena-helper object: %v", err)
	}
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	foundHelper := false
	foundInferred := false
	foundRefShared := false
	foundEscapedText := false
	foundEscapedWrapperText := false
	foundInitWrapper := false
	foundEarlyWrapper := false
	foundConsoleInitWrapper := false
	for _, symbol := range symbols {
		if symbol.Name == "__renvo_arena_alloc" && elf.ST_TYPE(symbol.Info) == elf.STT_FUNC &&
			elf.ST_BIND(symbol.Info) == elf.STB_LOCAL && symbol.Section != elf.SHN_UNDEF &&
			int(symbol.Section) < len(file.Sections) && file.Sections[symbol.Section].Name == ".text" {
			foundHelper = true
		}
		if symbol.Name == "initValue" && symbol.Section != elf.SHN_UNDEF &&
			int(symbol.Section) < len(file.Sections) && file.Sections[symbol.Section].Name == ".init.text" {
			foundInferred = true
		}
		if symbol.Name == "sharedValue" && symbol.Section != elf.SHN_UNDEF &&
			int(symbol.Section) < len(file.Sections) && file.Sections[symbol.Section].Name == ".ref.text" {
			foundRefShared = true
		}
		if symbol.Name == "callbackValue" && symbol.Section != elf.SHN_UNDEF &&
			int(symbol.Section) < len(file.Sections) && file.Sections[symbol.Section].Name == ".init.text" {
			foundEscapedText = true
		}
		if symbol.Name == "__renvo_cabi_callbackValue" && symbol.Section != elf.SHN_UNDEF &&
			int(symbol.Section) < len(file.Sections) && file.Sections[symbol.Section].Name == ".text" {
			foundEscapedWrapperText = true
		}
		if symbol.Name == "__renvo_cabi_initCallbackValue" && symbol.Section != elf.SHN_UNDEF &&
			int(symbol.Section) < len(file.Sections) && file.Sections[symbol.Section].Name == ".init.text" {
			foundInitWrapper = true
		}
		if symbol.Name == "__renvo_cabi_earlyCallbackValue" && symbol.Section != elf.SHN_UNDEF &&
			int(symbol.Section) < len(file.Sections) && file.Sections[symbol.Section].Name == ".init.text" {
			foundEarlyWrapper = true
		}
		if symbol.Name == "__renvo_cabi_consoleInitCallbackValue" && symbol.Section != elf.SHN_UNDEF &&
			int(symbol.Section) < len(file.Sections) && file.Sections[symbol.Section].Name == ".init.text" {
			foundConsoleInitWrapper = true
		}
	}
	if !foundHelper {
		t.Fatal("arena allocation helper lacks a defined local .text function symbol")
	}
	if !foundInferred {
		t.Fatal("init-only local function did not inherit .init.text")
	}
	if !foundRefShared {
		t.Fatal("mixed ref/init local function did not inherit .ref.text")
	}
	if !foundEscapedText {
		t.Fatal("address-escaped implementation did not retain its declared .init.text section")
	}
	if !foundEscapedWrapperText {
		t.Fatal("address-escaped C-ABI wrapper did not remain in .text")
	}
	if !foundInitWrapper {
		t.Fatal("init-data C-ABI wrapper did not retain .init.text")
	}
	if !foundEarlyWrapper {
		t.Fatal("early-console C-ABI wrapper did not retain .init.text")
	}
	if !foundConsoleInitWrapper {
		t.Fatal("console-initcall C-ABI wrapper did not retain .init.text")
	}
}

func TestLinuxAmd64ObjectSignedDivisionHelperHasFunctionSymbol(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
// renvo:object function divide - 0 2 0 - 8 8 0 - 0
//export divide
func divide(left int, right int) int { return left / right }
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("signed-division-helper source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("signed-division-helper metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("signed-division-helper object compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse signed-division-helper object: %v", err)
	}
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	for _, symbol := range symbols {
		if symbol.Name == "__renvo_signed_divide" && elf.ST_TYPE(symbol.Info) == elf.STT_FUNC &&
			elf.ST_BIND(symbol.Info) == elf.STB_LOCAL && symbol.Section != elf.SHN_UNDEF {
			return
		}
	}
	t.Fatal("signed division helper lacks a defined local function symbol")
}

func TestLinuxAmd64ObjectStringEqualHelperHasFunctionSymbol(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
func equal(left string, right string) bool { return left == right }
// renvo:object function invoke - 0 1 0 - 8 0 - 0
//export invoke
func invoke() bool { return equal("left", "right") }
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("string-equal-helper source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("string-equal-helper metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("string-equal-helper object compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse string-equal-helper object: %v", err)
	}
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	for _, symbol := range symbols {
		if symbol.Name == "__renvo_string_equal" && elf.ST_TYPE(symbol.Info) == elf.STT_FUNC &&
			elf.ST_BIND(symbol.Info) == elf.STB_LOCAL && symbol.Section != elf.SHN_UNDEF {
			return
		}
	}
	t.Fatal("string equality helper lacks a defined local function symbol")
}

func TestLinuxAmd64ObjectPanicStateDoesNotRelocateIntoDeclaredSection(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
// renvo:object variable anchor .discard.addressable 8 0 0 - 8 0 - 0
var anchor uintptr
func cleanup() {}
// renvo:object function invoke - 0 1 0 - 8 0 - 0
//export invoke
func invoke() { defer cleanup() }
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
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
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("object panic-state compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse object panic-state object: %v", err)
	}
	text, err := file.Section(".text").Data()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(text, []byte{0x49, 0x8b, 0x87}) || bytes.Contains(text, []byte{0x49, 0x89, 0x87}) {
		t.Fatal("object panic/defer state uses the R15 wrapper-helper register")
	}
	for _, section := range file.Sections {
		if section.Type != elf.SHT_RELA || section.Info == 0 || int(section.Info) >= len(file.Sections) || file.Sections[section.Info].Flags&elf.SHF_EXECINSTR == 0 {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		for at := 0; at+24 <= len(data); at += 24 {
			symbolIndex := binary.LittleEndian.Uint64(data[at+8:at+16]) >> 32
			symbols, err := file.Symbols()
			if err != nil {
				t.Fatal(err)
			}
			if symbolIndex == 0 || int(symbolIndex) > len(symbols) {
				continue
			}
			symbol := symbols[symbolIndex-1]
			if symbol.Section != elf.SHN_UNDEF && int(symbol.Section) < len(file.Sections) && file.Sections[symbol.Section].Name == ".discard.addressable" {
				t.Fatal("panic/defer runtime relocation targets .discard.addressable")
			}
		}
	}
}

func TestLinuxAmd64ObjectDirectDeferDoesNotReachSameSignatureFunction(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
// renvo:linkstatic libc,missing_defer_peer
func missing()
func missing() {}
func cleanup() {}
func unrelated() { missing() }
// renvo:object function invoke - 0 1 0 - 8 0 - 0
//export invoke
func invoke() { defer cleanup() }
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("direct-defer object source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok || !meta.panicEnabled {
		t.Fatal("direct-defer object metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("direct-defer object compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse direct-defer object: %v", err)
	}
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	for _, symbol := range symbols {
		if symbol.Name == "missing_defer_peer" || symbol.Name == "unrelated" {
			t.Fatalf("direct defer retained unreachable peer %q", symbol.Name)
		}
	}
}

func TestLinuxAmd64ObjectConstantSwitchDoesNotImportDeadCase(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
// renvo:linkstatic libc,missing_dead_case
func missing()
// renvo:linkstatic libc,missing_unused_declaration
func unusedDeclaration()
// renvo:object function choose - 0 1 0 - 8 0 - 0
//export choose
func choose() int {
	switch 8 {
	case 1:
		missing()
		return 1
	case 8:
		return 8
	default:
		missing()
		return 0
	}
}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("constant-switch object source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("constant-switch object metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("constant-switch object compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse constant-switch object: %v", err)
	}
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	for _, symbol := range symbols {
		if symbol.Name == "missing_dead_case" || symbol.Name == "missing_unused_declaration" {
			t.Fatalf("constant switch object retained dead import %q", symbol.Name)
		}
	}
}

func TestLinuxAmd64ObjectReturningBlockDoesNotImportFollowingCode(t *testing.T) {
	oldTarget := renvoTarget
	oldArch := renvoTargetArch
	oldOS := renvoTargetOS
	oldObjectFile := renvoCompilerObjectFile
	oldFixedTarget := renvoFixedTarget
	t.Cleanup(func() {
		renvoTarget = oldTarget
		renvoTargetArch = oldArch
		renvoTargetOS = oldOS
		renvoCompilerObjectFile = oldObjectFile
		renvoFixedTarget = oldFixedTarget
	})
	renvoCompilerObjectFile = true
	renvoFixedTarget = renvoTargetLinuxAmd64
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
// renvo:linkstatic libc,missing_dead_if
func missing()
func missing() {}
// renvo:linkstatic libc,required_conditional
func required()
func required() {}
func falseBranch() { if false { missing() } }
func __c_bool_int(value bool) int { if value { return 1 }; return 0 }
func alwaysTrue() uint8 { return uint8(__c_bool_int(true)) }
func constantCallFalse() { if !alwaysTrue() { missing() } }
func labelAfterConstantReturn() { goto later; if alwaysTrue() { return }; later: }
func convertedFalse() { value := uint8(__c_bool_int(false)); _ = &value; if value != 0 { missing() } }
func assignedFalse() { var value uint8; _ = &value; value = uint8(__c_bool_int(false)); if value != 0 { missing() } }
// renvo:object variable flowCondition .data 1 1 0 - 1 0 - 0
var flowCondition byte
func assignedFalseAfterNonReturningBranch() { var value uint8; _ = &value; value = uint8(__c_bool_int(false)); if flowCondition != 0 { for true {} }; if value != 0 { missing() } }
func assignedFalseAcrossLaterLabel() { var value uint8; _ = &value; value = uint8(__c_bool_int(false)); later: if value != 0 { missing() }; if flowCondition != 0 { goto later } }
func conditionalAssignmentMustRemain(condition byte) { value := byte(1); if condition != 0 { value = 0 }; if value != 0 { required() } }
func maskedFalse(value uint8) { value &= 0; if value != 0 { missing(); goto done }; done: }
func algebraicFalse(value uint8) { value = uint8(__c_bool_int(value != 0)) & uint8(__c_bool_int(false)); if value != 0 { missing() } }
func singleCallFalse(value uint8) { if value != 0 { missing() } }
func callSingle() { value := uint8(__c_bool_int(false)); _ = &value; singleCallFalse(value) }
func afterBlock() { { return }; missing() }
func nestedFalse() uint8 { { return uint8(__c_bool_int(false)) }; missing(); return 1 }
func callNestedFalse() { value := nestedFalse(); if value != 0 { missing() } }
// renvo:object variable-extern missing_dead_data - 1 1 0 - 1 0 - 0
var missing_dead_data byte
func emptyFeatureStub(value *byte, enabled uint8) {}
func callEmptyFeatureStub() { emptyFeatureStub(&missing_dead_data, uint8(__c_bool_int(true))) }
// renvo:object function invoke - 0 1 0 - 8 0 - 0
//export invoke
func invoke() { falseBranch(); constantCallFalse(); labelAfterConstantReturn(); convertedFalse(); assignedFalse(); assignedFalseAfterNonReturningBranch(); assignedFalseAcrossLaterLabel(); conditionalAssignmentMustRemain(flowCondition); maskedFalse(1); algebraicFalse(1); callSingle(); afterBlock(); callNestedFalse(); callEmptyFeatureStub() }
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("returning-block object source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("returning-block object metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("returning-block object compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse returning-block object: %v", err)
	}
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	foundRequired := false
	for _, symbol := range symbols {
		if symbol.Name == "missing_dead_if" {
			t.Fatalf("returning block retained dead import %q", symbol.Name)
		}
		if symbol.Name == "required_conditional" && symbol.Section == elf.SHN_UNDEF {
			foundRequired = true
		}
	}
	if !foundRequired {
		t.Fatal("conditional assignment incorrectly removed a reachable import")
	}
	for _, section := range file.Sections {
		if section.Type != elf.SHT_RELA {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		for at := 0; at+24 <= len(data); at += 24 {
			symbolIndex := binary.LittleEndian.Uint64(data[at+8:at+16]) >> 32
			if symbolIndex > 0 && int(symbolIndex) <= len(symbols) && symbols[symbolIndex-1].Name == "missing_dead_data" {
				t.Fatal("empty feature stub retained relocation for a discardable address argument")
			}
		}
	}
}

func TestLinuxAmd64ObjectInt3SelftestDefinesInlineAsmLabel(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
func renvo_runtime_CInt3Selftest(value *uint32) {}
// renvo:linkstatic libc,int3_selftest_ip
func int3_selftest_ip() {}
// renvo:object function invoke .init.text 0 1 0 - 8 0 - 0
//export invoke
func invoke() uintptr { var value uint32; renvo_runtime_CInt3Selftest(&value); return uintptr(int3_selftest_ip) }
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("int3 selftest object source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("int3 selftest object metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("int3 selftest object compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse int3 selftest object: %v", err)
	}
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	for _, symbol := range symbols {
		if symbol.Name == "int3_selftest_ip" && elf.ST_BIND(symbol.Info) == elf.STB_LOCAL && symbol.Section != elf.SHN_UNDEF {
			return
		}
	}
	t.Fatal("int3 selftest inline-assembly label is not a defined local symbol")
}

func TestLinuxAmd64ObjectEmitsStaticCallTemplateAsFunction(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
// renvo:linkstatic libc,__SCT__target
func __SCT__target(){}
// renvo:object static-call __SCT__target .static_call.text 4 1 0 - 8 0 - 0
var __c_static_call_1 [8]uint8=[8]uint8{195,204,144,144,144,15,185,204}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("static-call object source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("static-call object metadata failed")
	}
	if len(meta.globals) != 1 || meta.globals[0].objectDecl <= 0 ||
		meta.objectDecls[meta.globals[0].objectDecl].kind != renvoObjectDeclStaticCall {
		t.Fatalf("static-call object declaration was not retained: globals=%d declarations=%d", len(meta.globals), len(meta.objectDecls))
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("static-call object compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse static-call object: %v", err)
	}
	section := file.Section(".static_call.text")
	if section == nil || section.Flags&elf.SHF_EXECINSTR == 0 {
		var sections []string
		for _, candidate := range file.Sections {
			sections = append(sections, candidate.Name)
		}
		t.Fatalf("static-call template is not in an executable section: %v", sections)
	}
	data, err := section.Data()
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{195, 204, 144, 144, 144, 15, 185, 204}
	if !bytes.Equal(data, want) {
		t.Fatalf("static-call template = %v, want %v", data, want)
	}
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	for _, symbol := range symbols {
		if symbol.Name == "__SCT__target" && elf.ST_TYPE(symbol.Info) == elf.STT_FUNC &&
			elf.ST_BIND(symbol.Info) == elf.STB_GLOBAL && symbol.Section != elf.SHN_UNDEF && symbol.Size == 8 {
			return
		}
	}
	t.Fatal("static-call template lacks its defined global function symbol")
}

func TestLinuxAmd64ObjectExportFunctionPointerResult(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
type work struct{value uint64}
// renvo:object function retain_callback - 0 1 0 - 8 0 - 0
//export retain_callback
func retain_callback(callback func(*work)) func(*work){return callback}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("function-pointer-result source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("function-pointer-result metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("function-pointer-result object compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse function-pointer-result object: %v", err)
	}
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, symbol := range symbols {
		found = found || symbol.Name == "retain_callback" && elf.ST_BIND(symbol.Info) == elf.STB_GLOBAL && symbol.Section != elf.SHN_UNDEF
	}
	if !found {
		t.Fatal("function-pointer-result object does not export retain_callback")
	}
}

func TestLinuxAmd64ObjectNullFunctionPointerArgument(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
// renvo:linkstatic libc,install
func install(name string,callback func(*byte)){}
// renvo:object function invoke - 0 1 0 - 8 0 - 0
//export invoke
func invoke(){install("callback\x00",nil)}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("null function-pointer argument source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("null function-pointer argument metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("null function-pointer argument object compilation failed")
	}
}

func TestLinuxAmd64ObjectAddressTakenLocalFunctionUsesCABIWrapper(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
// renvo:object function compare - 0 2 0 - 8 8 0 - 0
func compare(left *byte,right *byte) int32{return 0}
// renvo:linkstatic libc,install
func install(callback func(*byte,*byte) int32){}
// renvo:object function invoke - 0 1 0 - 8 0 - 0
//export invoke
func invoke(){install(compare)}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("address-taken local function source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("address-taken local function metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("address-taken local function object compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse address-taken local function object: %v", err)
	}
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	foundWrapper, foundRelocation, wrapperInsideFunction := false, false, false
	var wrapperValue uint64
	for _, symbol := range symbols {
		if symbol.Name == "__renvo_cabi_compare" && symbol.Section != elf.SHN_UNDEF {
			foundWrapper = true
			wrapperValue = symbol.Value
		}
	}
	for _, symbol := range symbols {
		if symbol.Name != "__renvo_cabi_compare" && elf.ST_TYPE(symbol.Info) == elf.STT_FUNC &&
			symbol.Size > 0 && wrapperValue >= symbol.Value && wrapperValue < symbol.Value+symbol.Size {
			wrapperInsideFunction = true
		}
	}
	for _, section := range file.Sections {
		if section.Type != elf.SHT_RELA {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		for at := 0; at+24 <= len(data); at += 24 {
			info := binary.LittleEndian.Uint64(data[at+8 : at+16])
			symbolIndex := int(info >> 32)
			foundRelocation = foundRelocation || symbolIndex > 0 && symbolIndex-1 < len(symbols) &&
				symbols[symbolIndex-1].Name == "__renvo_cabi_compare"
		}
	}
	if !foundWrapper || !foundRelocation || wrapperInsideFunction {
		t.Fatalf("address-taken local function wrapper=%v relocation=%v nested=%v",
			foundWrapper, foundRelocation, wrapperInsideFunction)
	}
}

func TestLinuxAmd64ObjectFunctionPointerInIntegerUnionCarrier(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
type callbackUnion struct{__c_align uint64}
type task struct{callback callbackUnion}
func target(value *task){}
// renvo:object variable work .data 8 0 0 - 8 0 - 0
var work task=task{callback:callbackUnion{__c_align:uint64(target)}}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("integer function-pointer carrier source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("integer function-pointer carrier metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("integer function-pointer carrier object compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse integer function-pointer carrier object: %v", err)
	}
	relocations := 0
	for _, section := range file.Sections {
		if section.Type != elf.SHT_RELA {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		for at := 0; at+24 <= len(data); at += 24 {
			if elf.R_X86_64(binary.LittleEndian.Uint64(data[at+8:at+16])&0xffffffff) == elf.R_X86_64_64 {
				relocations++
			}
		}
	}
	if relocations != 1 {
		t.Fatalf("integer function-pointer carrier has %d absolute relocations, want 1", relocations)
	}
}

func TestLinuxAmd64ObjectDirectiveDoesNotCrossDeclaration(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
// renvo:object variable first .data 1 0 0 - 1 0 - 0
var first uint8=1
func helper(){}
var name=[3]int8{1,2,3}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("object directive adjacency source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("object directive adjacency metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("object directive crossed an intervening declaration")
	}
}

func TestLinuxAmd64ObjectSectionUsesDeclaredSize(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
// renvo:object variable callback .con_initcall.init 4 0 0 - 4 2 target 0
var callback [4]uint8=[4]uint8{0,0,0,0}
// renvo:object function target .init.text 0 0 0 - 8 0 - 0
func target() int32{return 0}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("declared-size object source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("declared-size object metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("declared-size object compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse declared-size object: %v", err)
	}
	section := file.Section(".con_initcall.init")
	if section == nil {
		t.Fatal("declared-size object is missing .con_initcall.init")
	}
	if section.Size != 4 {
		t.Fatalf(".con_initcall.init size = %d, want 4", section.Size)
	}
	bss := file.Section(".bss")
	if bss == nil || bss.Size != 4 {
		if bss == nil {
			t.Fatal("internal carrier tail is missing from .bss")
		}
		t.Fatalf("internal carrier tail size = %d, want 4", bss.Size)
	}
}

func TestLinuxAmd64ObjectPrunesLiteralFalseForeignCallBesideStructKey(t *testing.T) {
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	source := []byte(`package main
type descriptor struct{callback func() int32}
// renvo:linkstatic libc,missing
func missing() int32{return 0}
// renvo:object function choose .text 0 1 0 - 8 0 - 0
//export choose
func choose() int32 {
	if false {
		value := descriptor{callback:nil}
		_ = &value
		return missing()
	}
	return 7
}
`)
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("literal-false foreign-call source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("literal-false foreign-call metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("literal-false foreign-call object compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse literal-false foreign-call object: %v", err)
	}
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	for _, symbol := range symbols {
		if symbol.Name == "missing" {
			t.Fatal("literal-false branch retained an undefined foreign call")
		}
	}
}

func TestLinuxAmd64ObjectKernelLinkAddressRelocation(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
import __c_unsafe "unsafe"
func renvo_runtime_CKernelLinkAddress(value uint64) uint64{return value}
func __c_pointer_step_dec_1(pointer *int8,count uintptr) *int8{return (*int8)(__c_unsafe.Pointer(uintptr(__c_unsafe.Pointer(pointer))-count*uintptr(1)))}
// renvo:object variable-extern _text - 1 1 0 - 0 0 - 0
var _text [0]int8
// renvo:object function linked_address .head.text 1 1 0 - 8 0 - 0
//export linked_address
func linked_address() uint64 {
	return renvo_runtime_CKernelLinkAddress(uint64(__c_pointer_step_dec_1((*int8)(__c_unsafe.Pointer(&(_text))),uintptr(^uint64(2147483647)))))
}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("kernel link-address source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("kernel link-address metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	g := renvoBeginScalarProgramAmd64(&program, &meta)
	if g == nil {
		t.Fatal("kernel link-address object setup failed")
	}
	if !renvoEmitAllQueuedFunctionsScratch(g) {
		t.Fatal("kernel link-address function emission failed")
	}
	result := renvoFinishScalarProgramAmd64(g)
	if !result.ok {
		t.Fatal("kernel link-address object compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse kernel link-address object: %v", err)
	}
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, section := range file.Sections {
		if section.Type != elf.SHT_RELA {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		for at := 0; at+24 <= len(data); at += 24 {
			info := binary.LittleEndian.Uint64(data[at+8 : at+16])
			symbolIndex := int(info >> 32)
			addend := int64(binary.LittleEndian.Uint64(data[at+16 : at+24]))
			found = found || elf.R_X86_64(info&0xffffffff) == elf.R_X86_64_64 && symbolIndex > 0 &&
				symbolIndex-1 < len(symbols) && symbols[symbolIndex-1].Name == "_text" && addend == 2147483648
		}
	}
	if !found {
		t.Fatal("kernel link-address object is missing the absolute _text+0x80000000 relocation")
	}
}

func TestLinuxAmd64ObjectIndexedStaticArrayAddress(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
type table struct{extra *byte}
func __c_pointer_index_1(pointer *int32,index uintptr)*int32{return pointer}
// renvo:object variable values .data 4 0 0 - 64 0 - 0
var values [16]int32
// renvo:object variable entry .data 8 0 0 - 8 0 - 0
var entry table=table{extra:(*byte)(__c_pointer_index_1(&(values)[0],uintptr(2)))}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("indexed static-array address source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("indexed static-array address metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("indexed static-array address object compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse indexed static-array address object: %v", err)
	}
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, section := range file.Sections {
		if section.Type != elf.SHT_RELA {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		for at := 0; at+24 <= len(data); at += 24 {
			symbolIndex := int(binary.LittleEndian.Uint64(data[at+8:at+16]) >> 32)
			addend := int64(binary.LittleEndian.Uint64(data[at+16 : at+24]))
			found = found || symbolIndex > 0 && symbolIndex-1 < len(symbols) &&
				symbols[symbolIndex-1].Name == "values" && addend == 8
		}
	}
	if !found {
		t.Fatal("indexed static-array address is missing its values+8 relocation")
	}
}

func TestLinuxAmd64ObjectPassesIndexedFunctionPointer(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
type work struct{value uint64}
func first(work *work){}
func install(work *work,callback func(*work)){callback(work)}
// renvo:object function invoke - 0 1 0 - 8 0 - 0
//export invoke
func invoke(work *work){callbacks:=[1]func(*work){first};install(work,callbacks[0])}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("indexed function-pointer source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("indexed function-pointer metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("kernel object could not pass indexed function pointer")
	}
}

func TestLinuxAmd64ObjectFunctionPointerStackArgument(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
func target(a uint64,b uint64,c uint64,d uint64,e uint64,f uint64,g uint64) uint64{return a+b+c+d+e+f+g}
// renvo:object variable callback .data 8 1 0 - 8 1 target 0
var callback func(uint64,uint64,uint64,uint64,uint64,uint64,uint64) uint64=target
// renvo:object function invoke - 0 1 0 - 8 0 - 0
//export invoke
func invoke(a uint64,b uint64,c uint64,d uint64,e uint64,f uint64,g uint64) uint64{return callback(a,b,c,d,e,f,g)}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("function-pointer stack-argument source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("function-pointer stack-argument metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("function-pointer stack-argument object compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse function-pointer stack-argument object: %v", err)
	}
	var textData []byte
	for _, section := range file.Sections {
		if section.Flags&elf.SHF_EXECINSTR == 0 {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		textData = append(textData, data...)
	}
	for _, want := range [][]byte{
		[]byte("\x48\x89\x04\x24"),
		[]byte("\x4c\x89\x54\x24"),
		[]byte("\x41\xff\xd3"),
	} {
		if !bytes.Contains(textData, want) {
			t.Fatalf("function-pointer stack-argument object is missing instruction sequence %x", want)
		}
	}
}

func TestLinuxAmd64ObjectFunctionPointerIntegerAggregateArgument(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
type pair struct{pointer *byte;length uint64}
// renvo:object variable callback .data 8 0 0 - 8 0 - 0
var callback func(*byte,*byte,pair) int32
// renvo:object function invoke - 0 1 0 - 8 0 - 0
//export invoke
func invoke(first *byte,second *byte) int32{return callback(first,second,pair{pointer:first,length:9})}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("function-pointer aggregate-argument source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("function-pointer aggregate-argument metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("function-pointer integer-aggregate argument did not compile")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse function-pointer aggregate object: %v", err)
	}
	var textData []byte
	for _, section := range file.Sections {
		if section.Flags&elf.SHF_EXECINSTR == 0 {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		textData = append(textData, data...)
	}
	for _, want := range [][]byte{
		[]byte("\x48\x89\xc7"),
		[]byte("\x48\x89\xc6"),
		[]byte("\x48\x89\xc2"),
		[]byte("\x48\x89\xc1"),
		[]byte("\xff\xd0"),
	} {
		if !bytes.Contains(textData, want) {
			t.Fatalf("function-pointer aggregate object is missing instruction sequence %x", want)
		}
	}
}

func TestLinuxAmd64ObjectAddressedFunctionAggregateRelocation(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
type descriptor struct{padding uint64;callback func() int32}
// renvo:object function callback - 0 0 0 - 8 0 - 0
func callback() int32{return 7}
// renvo:object variable value .data 8 0 0 - 16 0 - 0
var value descriptor=descriptor{padding:1,callback:&(callback)}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("addressed function aggregate source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("addressed function aggregate metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("addressed function aggregate object compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse addressed function aggregate object: %v", err)
	}
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatalf("read addressed function aggregate symbols: %v", err)
	}
	found := false
	for _, section := range file.Sections {
		if section.Type != elf.SHT_RELA {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		for at := 0; at+24 <= len(data); at += 24 {
			offset := binary.LittleEndian.Uint64(data[at : at+8])
			symbolIndex := int(binary.LittleEndian.Uint64(data[at+8:at+16]) >> 32)
			if offset == 8 && symbolIndex > 0 && symbolIndex-1 < len(symbols) && symbols[symbolIndex-1].Name == "__renvo_cabi_callback" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("addressed function aggregate is missing its callback relocation at byte 8")
	}
}

func TestLinuxAmd64ObjectMapsInteriorAnonymousBSS(t *testing.T) {
	var asm renvoAsm
	asm.objectData = append(asm.objectData, renvoObjectDataSymbol{
		offset: 16, size: 8, storageSize: 8, alignment: 8, initialized: 1, value: 42,
	})
	asm.bssSize = 32

	image := renvoAsmImageRelocatableObjectAmd64(&asm)
	if len(image) == 0 {
		t.Fatal("object writer rejected an interior anonymous BSS range")
	}
	file, err := elf.NewFile(bytes.NewReader(image))
	if err != nil {
		t.Fatalf("parse relocatable object: %v", err)
	}
	section := file.Section(".bss")
	if section == nil || section.Size != 24 {
		if section == nil {
			t.Fatal("object has no .bss section")
		}
		t.Fatalf("anonymous .bss size = %d, want 24", section.Size)
	}
}

func TestLinuxAmd64ObjectConstantPointerStepRelocations(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
// renvo:object variable base .rodata 1 0 0 - 16 0 - 0
var base [16]uint8=[16]uint8{}
func __c_pointer_step_inc_2(pointer *uint8,count uintptr) *uint8{return pointer}
// renvo:object variable pointers .data 8 1 0 - 32 0 - 0
var pointers [4]*uint8=[4]*uint8{nil,&(base)[0],__c_pointer_step_inc_2(&(base)[0],uintptr(1)),__c_pointer_step_inc_2(__c_pointer_step_inc_2(&(base)[0],uintptr(1)),uintptr(2))}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("pointer-step object source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("pointer-step object metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("pointer-step object compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse pointer-step object: %v", err)
	}
	var addends []int64
	for _, section := range file.Sections {
		if section.Type != elf.SHT_RELA {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		for at := 0; at+24 <= len(data); at += 24 {
			addends = append(addends, int64(binary.LittleEndian.Uint64(data[at+16:at+24])))
		}
	}
	for _, want := range []int64{0, 1, 3} {
		found := false
		for _, addend := range addends {
			found = found || addend == want
		}
		if !found {
			t.Fatalf("pointer-step object relocation addends = %v, missing %d", addends, want)
		}
	}
}

func TestLinuxAmd64ObjectIRQStackCallSequence(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
// renvo:linkstatic libc,irq_enter_rcu
func irq_enter_rcu(){}
// renvo:linkstatic libc,irq_exit_rcu
func irq_exit_rcu(){}
func target(first uintptr,second uintptr){}
func renvo_runtime_CIRQStackCall2_target(tos *byte,first uintptr,second uintptr){}
// renvo:object function invoke - 0 1 0 - 8 0 - 0
//export invoke
func invoke(tos *byte,first uintptr,second uintptr){renvo_runtime_CIRQStackCall2_target(tos,first,second)}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("IRQ stack-call object source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("IRQ stack-call object metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("IRQ stack-call object compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse IRQ stack-call object: %v", err)
	}
	foundSequence := false
	sequence := []byte("\x41\x54\x41\x55\x48\x8b\x45")
	for _, section := range file.Sections {
		if section.Flags&elf.SHF_EXECINSTR == 0 {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		foundSequence = foundSequence || bytes.Contains(data, sequence)
	}
	if !foundSequence {
		t.Fatal("IRQ stack-call object is missing the callee-save and operand-load sequence")
	}
	foundInternalArgumentOrder := false
	for _, section := range file.Sections {
		if section.Flags&elf.SHF_EXECINSTR == 0 {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		// mov %r12,%rsi; mov %r13,%rdi passes the first and second source
		// arguments according to Renvo's reversed internal parameter binding.
		foundInternalArgumentOrder = foundInternalArgumentOrder || bytes.Contains(data, []byte("\x4c\x89\xe6\x4c\x89\xef"))
	}
	if !foundInternalArgumentOrder {
		t.Fatal("IRQ stack-call object does not bind two target arguments in Renvo internal order")
	}
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatalf("read IRQ stack-call symbols: %v", err)
	}
	wanted := map[string]bool{"irq_enter_rcu": false, "irq_exit_rcu": false}
	for _, section := range file.Sections {
		if section.Type != elf.SHT_RELA {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		for at := 0; at+24 <= len(data); at += 24 {
			symbolIndex := int(binary.LittleEndian.Uint64(data[at+8:at+16]) >> 32)
			if symbolIndex > 0 && symbolIndex-1 < len(symbols) {
				name := symbols[symbolIndex-1].Name
				if _, ok := wanted[name]; ok {
					wanted[name] = true
				}
			}
		}
	}
	for name, found := range wanted {
		if !found {
			t.Fatalf("IRQ stack-call object has no relocation for %s", name)
		}
	}
}

func TestLinuxAmd64ObjectPackedAggregatePointerRelocation(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
type packed struct{__c_align uint8;__c_tail [9]byte}
// renvo:object variable table .bss 1 0 0 - 1 0 - 0
var table [1]uint8
func __c_object_aggregate_0_2_x1(size uint16,address uint64) packed{return packed{}}
// renvo:object variable descriptor .data 1 0 0 - 10 0 - 0
var descriptor packed=__c_object_aggregate_0_2_x1(4095,uint64(table))
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("packed aggregate object source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("packed aggregate object metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("packed aggregate object compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse packed aggregate object: %v", err)
	}
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatalf("read packed aggregate symbols: %v", err)
	}
	found := false
	for _, section := range file.Sections {
		if section.Type != elf.SHT_RELA {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		for at := 0; at+24 <= len(data); at += 24 {
			offset := binary.LittleEndian.Uint64(data[at : at+8])
			symbolIndex := int(binary.LittleEndian.Uint64(data[at+8:at+16]) >> 32)
			if offset == 2 && symbolIndex > 0 && symbolIndex-1 < len(symbols) && symbols[symbolIndex-1].Name == "table" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("packed aggregate object is missing its unaligned table relocation at byte 2")
	}
}

func TestLinuxAmd64ObjectCStringLiteralPointerArgument(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
// renvo:linkstatic libc,compare
func compare(value *byte) int32{return 0}
// renvo:object function invoke - 0 1 0 - 8 0 - 0
//export invoke
func invoke() int32{return compare("PCMP\x00")}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("C string pointer object source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("C string pointer object metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("C string literal did not decay to a foreign pointer argument")
	}
}

func TestLinuxAmd64ObjectForeignIntegerAggregateArgument(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
type pair struct{pointer *byte;length uint64}
// renvo:linkstatic libc,consume_pair
func consume_pair(value pair,target *byte,mode int32){}
// renvo:object function invoke - 0 1 0 - 8 0 - 0
//export invoke
func invoke(value pair,target *byte){consume_pair(value,target,1)}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("foreign aggregate argument source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("foreign aggregate argument metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("foreign 16-byte integer aggregate call did not compile")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse foreign aggregate object: %v", err)
	}
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatalf("read foreign aggregate symbols: %v", err)
	}
	found := false
	for _, symbol := range symbols {
		found = found || symbol.Name == "consume_pair" && symbol.Section == elf.SHN_UNDEF
	}
	if !found {
		t.Fatal("foreign aggregate object is missing consume_pair undefined symbol")
	}
}

func TestLinuxAmd64ObjectForeignSmallIntegerAggregateReturn(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
// renvo:c11
type pair struct{first uint64}
// renvo:linkstatic libc,read_pair
func read_pair(value pair,context *byte) pair{return pair{}}
// renvo:object function inspect - 0 1 0 - 8 0 - 0
//export inspect
func inspect(index int32) uint64{value:=read_pair(pair{first:uint64(index)},nil);return value.first}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("foreign aggregate return source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("foreign aggregate return metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("foreign small integer aggregate return did not compile")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse foreign aggregate return object: %v", err)
	}
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatalf("read foreign aggregate return symbols: %v", err)
	}
	found := false
	for _, symbol := range symbols {
		found = found || symbol.Name == "read_pair" && symbol.Section == elf.SHN_UNDEF
	}
	if !found {
		t.Fatal("foreign aggregate return object is missing read_pair undefined symbol")
	}
}

func TestLinuxAmd64ObjectSmallIntegerAggregateReturn(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
type one struct{value uint64}
type two struct{first uint64;second uint64}
// renvo:object function make_one - 0 1 0 - 8 0 - 0
//export make_one
func make_one(value uint64) one{return one{value:value}}
func make_two(value uint64) two{return two{first:value,second:value+1}}
// renvo:object variable callback .data 8 1 0 - 8 1 make_two 0
var callback func(uint64) two=make_two
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("small aggregate return source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("small aggregate return metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("small aggregate return wrappers did not compile")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse small aggregate return object: %v", err)
	}
	var textData []byte
	for _, section := range file.Sections {
		if section.Flags&elf.SHF_EXECINSTR == 0 {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		textData = append(textData, data...)
	}
	if !bytes.Contains(textData, []byte("\x48\x8b\x04\x24\x48\x83\xc4\x10")) ||
		!bytes.Contains(textData, []byte("\x48\x8b\x04\x24\x48\x8b\x54\x24\x08\x48\x83\xc4\x10")) {
		t.Fatal("small aggregate export/function-pointer wrappers do not load their SysV return words")
	}
	if !bytes.Contains(textData, []byte("\x48\x83\xec\x10\x57\x48\x8d\x44\x24\x08\x50")) {
		t.Fatal("small aggregate export wrapper does not preserve its private result slot across argument marshaling")
	}
}

func TestLinuxAmd64ObjectExportScalarStackArguments(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
// renvo:object function seven - 0 1 0 - 8 0 - 0
//export seven
func seven(a uintptr,b uintptr,c uintptr,d uintptr,e uintptr,f uintptr,g uintptr) uintptr{return a+b+c+d+e+f+g}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("stack-argument export source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("stack-argument export metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("stack-argument export did not compile")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse stack-argument object: %v", err)
	}
	found := false
	for _, section := range file.Sections {
		if section.Flags&elf.SHF_EXECINSTR == 0 {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		found = found || bytes.Contains(data, []byte("\x4c\x8d\x5c\x24\x30\x57\x56\x52\x51\x41\x50\x41\x51\x41\xff\x73\x00"))
	}
	if !found {
		t.Fatal("stack-argument export wrapper does not preserve and push the first incoming stack word")
	}
}

func TestLinuxAmd64ObjectExportNormalizesNarrowSysVArguments(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
// renvo:object function narrow - 0 1 0 - 8 0 - 0
//export narrow
func narrow(a uintptr,b uintptr,c int32,d uint32,e uintptr,f uintptr,g int32) int32{return c+int32(d)+g}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("narrow SysV argument source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("narrow SysV argument metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("narrow SysV argument wrapper did not compile")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse narrow SysV argument object: %v", err)
	}
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	for _, symbol := range symbols {
		if symbol.Name != "narrow" || symbol.Section == elf.SHN_UNDEF || symbol.Size == 0 {
			continue
		}
		section := file.Sections[symbol.Section]
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		start := int(symbol.Value - section.Addr)
		end := start + int(symbol.Size)
		if start < 0 || end > len(data) {
			t.Fatalf("narrow wrapper range [%d,%d) exceeds section size %d", start, end, len(data))
		}
		code := data[start:end]
		// SysV leaves the high bits of narrow integer argument registers and
		// stack slots unspecified. Renvo's internal ABI uses full words, so the
		// wrapper must sign/zero extend before pushing those words.
		for _, want := range [][]byte{
			[]byte("\x48\x89\xd0\x48\x63\xc0\x50"),
			[]byte("\x48\x89\xc8\x89\xc0\x50"),
			[]byte("\x49\x8b\x43\x00\x48\x63\xc0\x50"),
		} {
			if !bytes.Contains(code, want) {
				t.Fatalf("narrow wrapper is missing normalization %x: code=%x", want, code)
			}
		}
		return
	}
	t.Fatal("narrow export wrapper symbol not found")
}

func TestLinuxAmd64ObjectSafeMSROperationsEmitExceptionTable(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
func renvo_runtime_CReadMSRSafe(register uint32,err *int32,low *uint32,high *uint32) {}
func renvo_runtime_CWriteMSRSafe(register uint32,low uint32,high uint32) int32 { return 0 }
// renvo:object function safe_msr - 0 1 0 - 8 0 - 0
//export safe_msr
func safe_msr(register uint32) int32 {
	var err int32
	var low uint32
	var high uint32
	renvo_runtime_CReadMSRSafe(register,&err,&low,&high)
	return err+renvo_runtime_CWriteMSRSafe(register,low,high)
}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("safe MSR source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("safe MSR metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("safe MSR object did not compile")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse safe MSR object: %v", err)
	}
	exTable := file.Section("__ex_table")
	if exTable == nil {
		t.Fatal("safe MSR object has no __ex_table section")
	}
	data, err := exTable.Data()
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 24 || binary.LittleEndian.Uint32(data[8:12]) != 11|(8<<8) ||
		binary.LittleEndian.Uint32(data[20:24]) != 10 {
		t.Fatalf("safe MSR exception records = %x", data)
	}
	relocations := 0
	for _, section := range file.Sections {
		if section.Type != elf.SHT_RELA || int(section.Info) >= len(file.Sections) || file.Sections[section.Info].Name != "__ex_table" {
			continue
		}
		relocationData, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		relocations += len(relocationData) / 24
	}
	if relocations != 4 {
		t.Fatalf("safe MSR exception relocations = %d, want 4", relocations)
	}
	var textData []byte
	for _, section := range file.Sections {
		if section.Flags&elf.SHF_EXECINSTR == 0 {
			continue
		}
		sectionData, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		textData = append(textData, sectionData...)
	}
	if !bytes.Contains(textData, []byte("\x0f\x32\x45\x31\xc0")) ||
		!bytes.Contains(textData, []byte("\x0f\x30\x31\xc0")) {
		t.Fatalf("safe MSR instruction/fixup sequences are missing: text=%x", textData)
	}
}

func TestLinuxAmd64ObjectNestedUnionAggregateConstant(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
type member struct{word uint32}
type carrier struct{storage uint32}
func __c_object_union_init_0_x1(value member) carrier{return carrier{storage:value.word}}
type outer struct{value carrier;tail uint32}
// renvo:object variable global .data 8 0 0 - 8 0 - 0
var global outer=outer{value:__c_object_union_init_0_x1(member{word:7}),tail:9}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("nested union constant source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("nested union constant metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("nested union aggregate constant did not compile")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse nested union constant object: %v", err)
	}
	data, err := file.Section(".data").Data()
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 8 || binary.LittleEndian.Uint32(data[:4]) != 7 || binary.LittleEndian.Uint32(data[4:8]) != 9 {
		t.Fatalf("nested union constant data = %v, want words 7 and 9", data)
	}
}

func TestLinuxAmd64ObjectVariadicSixFixedArguments(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
// renvo:object function variadic - 0 1 0 - 8 0 - 1
//export variadic
func variadic(a uintptr,b uintptr,c uintptr,d uintptr,e uintptr,f uintptr,__c_va *[3]uintptr) uintptr{return a+b+c+d+e+f}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("six-fixed-argument variadic source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("six-fixed-argument variadic metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("six-fixed-argument variadic export did not compile")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse variadic object: %v", err)
	}
	found := false
	for _, section := range file.Sections {
		if section.Flags&elf.SHF_EXECINSTR == 0 {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		found = found || bytes.Contains(data, []byte("\x49\x89\xc2\x57\x56\x52\x51\x41\x50\x41\x51\x41\x52"))
	}
	if !found {
		t.Fatal("variadic wrapper did not save all six fixed SysV register arguments before its va_list")
	}
}

func TestLinuxAmd64ObjectForeignMemoryAggregateArgument(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
type block struct{first uint64;second uint64;third uint64}
// renvo:linkstatic libc,consume_block
func consume_block(pointer *byte,value block){}
// renvo:object function invoke - 0 1 0 - 8 0 - 0
//export invoke
func invoke(pointer *byte){var value block;value.first=1;value.second=2;value.third=3;consume_block(pointer,value)}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("foreign memory aggregate source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("foreign memory aggregate metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("foreign memory aggregate call did not compile")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse foreign memory aggregate object: %v", err)
	}
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, symbol := range symbols {
		found = found || symbol.Name == "consume_block" && symbol.Section == elf.SHN_UNDEF
	}
	if !found {
		t.Fatal("foreign memory aggregate object is missing consume_block")
	}
}

func TestLinuxAmd64ObjectForeignIntegerAggregateStackArguments(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
type carrier struct{first uint64;second uint64}
// renvo:linkstatic libc,foreign
func foreign(a uintptr,b uintptr,c uintptr,d uintptr,e uintptr,value carrier,tail uintptr) uintptr{return 0}
// renvo:object function invoke - 0 1 0 - 8 0 - 0
//export invoke
func invoke() uintptr{var value carrier;value.first=6;value.second=7;return foreign(1,2,3,4,5,value,8)}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("foreign integer aggregate stack source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("foreign integer aggregate stack metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("foreign call with an integer aggregate and stack arguments did not compile")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse foreign integer aggregate stack object: %v", err)
	}
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, symbol := range symbols {
		found = found || symbol.Name == "foreign" && symbol.Section == elf.SHN_UNDEF
	}
	if !found {
		t.Fatal("foreign integer aggregate stack object is missing its undefined function symbol")
	}
}

func TestLinuxAmd64ObjectForeignTwentyOneIntegerArguments(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
// renvo:linkstatic libc,foreign_many
func foreign_many(p0 string,p1 uint64,p2 uint64,p3 uint64,p4 uint64,p5 uint64,p6 uint64,p7 uint64,p8 uint64,p9 uint64,p10 uint64,p11 uint64,p12 uint64,p13 uint64,p14 uint64,p15 uint64,p16 uint64,p17 uint64,p18 uint64,p19 uint64,p20 uint64) int32{return 0}
// renvo:object function invoke - 0 1 0 - 8 0 - 0
//export invoke
func invoke() int32{return foreign_many("values",1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20)}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("21-argument foreign-call source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("21-argument foreign-call metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("21-argument foreign call did not compile")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse 21-argument foreign-call object: %v", err)
	}
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, symbol := range symbols {
		found = found || symbol.Name == "foreign_many" && symbol.Section == elf.SHN_UNDEF
	}
	if !found {
		t.Fatal("21-argument foreign-call object is missing foreign_many")
	}
}

func TestLinuxAmd64ObjectExportMemoryAggregateArgument(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
type block struct{first uint64;second uint64;third uint64}
// renvo:object function consume - 0 1 0 - 8 0 - 0
//export consume
func consume(value block) uint64{return value.first+value.second+value.third}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("memory-aggregate export source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("memory-aggregate export metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("memory-aggregate export did not compile")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse memory-aggregate export object: %v", err)
	}
	found := false
	for _, section := range file.Sections {
		if section.Flags&elf.SHF_EXECINSTR == 0 {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		found = found || bytes.Contains(data, []byte("\x4c\x8d\x5c\x24\x30\x41\xff\x73\x00\x41\xff\x73\x08\x41\xff\x73\x10"))
	}
	if !found {
		t.Fatal("memory-aggregate export wrapper did not push its incoming stack words")
	}
}

func TestLinuxAmd64ObjectRuntimeShiftSection(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
func renvo_runtime_CRuntimeShift_runtime_shift_hash(value uint64) uint64{return value}
// renvo:object function shift - 0 1 0 - 8 0 - 0
//export shift
func shift(value uint64) uint64{return renvo_runtime_CRuntimeShift_runtime_shift_hash(value)}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("runtime-shift source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("runtime-shift metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("runtime-shift object did not compile")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse runtime-shift object: %v", err)
	}
	section := file.Section("runtime_shift_hash")
	if section == nil || section.Size != 4 || section.Flags != elf.SHF_ALLOC {
		t.Fatalf("runtime-shift section = %#v, want four alloc-only bytes", section)
	}
	relocation := file.Section(".relaruntime_shift_hash")
	if relocation == nil || relocation.Size != 24 {
		t.Fatalf("runtime-shift relocation section = %#v, want one RELA entry", relocation)
	}
}

func TestLinuxAmd64ObjectPutUserRelocation(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
func renvo_runtime_CPutUser4(address uintptr,value uint64) int32{return 0}
// renvo:object function invoke - 0 1 0 - 8 0 - 0
//export invoke
func invoke(address uintptr,value uint64) int32{return renvo_runtime_CPutUser4(address,value)}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("put-user object source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("put-user object metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("put-user object compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse put-user object: %v", err)
	}
	foundSequence := false
	for _, section := range file.Sections {
		if section.Flags&elf.SHF_EXECINSTR == 0 {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		foundSequence = foundSequence || bytes.Contains(data, []byte("\x48\x8b\x4c\x24\x08\x48\x8b\x04\x24\xe8"))
	}
	if !foundSequence {
		t.Fatal("put-user object is missing the RCX/RAX operand-load sequence")
	}
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatalf("read put-user symbols: %v", err)
	}
	foundRelocation := false
	for _, section := range file.Sections {
		if section.Type != elf.SHT_RELA {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		for at := 0; at+24 <= len(data); at += 24 {
			symbolIndex := int(binary.LittleEndian.Uint64(data[at+8:at+16]) >> 32)
			if symbolIndex > 0 && symbolIndex-1 < len(symbols) && symbols[symbolIndex-1].Name == "__put_user_4" {
				foundRelocation = true
			}
		}
	}
	if !foundRelocation {
		t.Fatal("put-user object has no relocation for __put_user_4")
	}
}

func TestLinuxAmd64ObjectMSABIFunctionPointerCall(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
func renvo_runtime_CMSABICall_1(function func(uintptr,uint64,uint64,uint64,uint64) uintptr,p0 uintptr,p1 uint64,p2 uint64,p3 uint64,p4 uint64) uintptr{return 0}
// renvo:object function invoke - 0 1 0 - 8 0 - 0
//export invoke
func invoke(function func(uintptr,uint64,uint64,uint64,uint64) uintptr,p0 uintptr,p1 uint64,p2 uint64,p3 uint64,p4 uint64) uintptr{return renvo_runtime_CMSABICall_1(function,p0,p1,p2,p3,p4)}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("Microsoft-ABI object source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("Microsoft-ABI object metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("Microsoft-ABI object compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse Microsoft-ABI object: %v", err)
	}
	wanted := [][]byte{
		[]byte("\x48\x83\xec\x30"),
		[]byte("\x48\x89\x44\x24\x20"),
		[]byte("\x48\x89\xc1"),
		[]byte("\x48\x89\xc2"),
		[]byte("\x49\x89\xc0"),
		[]byte("\x49\x89\xc1"),
		[]byte("\x41\xff\xd3\x48\x83\xc4\x30"),
	}
	for _, section := range file.Sections {
		if section.Flags&elf.SHF_EXECINSTR == 0 {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < len(wanted); i++ {
			if wanted[i] != nil && bytes.Contains(data, wanted[i]) {
				wanted[i] = nil
			}
		}
	}
	for _, sequence := range wanted {
		if sequence != nil {
			t.Fatalf("Microsoft-ABI object is missing instruction sequence %x", sequence)
		}
	}
}

func TestLinuxAmd64ObjectZeroStorageAggregateInitializer(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
type empty struct{}
type unionCarrier struct{__c_align uint8}
type spinlock struct{field unionCarrier}
func __c_union_init(v empty) unionCarrier{return unionCarrier{}}
// renvo:object variable rtc_lock - 1 1 0 - 0 0 - 0
var rtc_lock spinlock=spinlock{field:__c_union_init(empty{})}
// renvo:object function anchor - 0 1 0 - 8 0 - 0
//export anchor
func anchor(){}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("zero-storage aggregate source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("zero-storage aggregate metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("zero-storage aggregate initializer was encoded through its synthetic carrier")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse zero-storage aggregate object: %v", err)
	}
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatalf("read zero-storage aggregate symbols: %v", err)
	}
	found := false
	for _, symbol := range symbols {
		found = found || symbol.Name == "rtc_lock" && symbol.Size == 0 && symbol.Section != elf.SHN_UNDEF
	}
	if !found {
		t.Fatal("zero-storage aggregate has no defined zero-sized rtc_lock symbol")
	}
}

func TestLinuxAmd64ObjectExportIntegerAggregateParameter(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
type timespec64 struct{sec int64;nsec int64}
// renvo:object function update_persistent_clock64 - 0 1 0 - 8 0 - 0
//export update_persistent_clock64
func update_persistent_clock64(now timespec64) int32{return int32(now.sec+now.nsec)}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("integer-aggregate export source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("integer-aggregate export metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("two-eightbyte integer aggregate was rejected by the C ABI wrapper")
	}
}

func TestLinuxAmd64ObjectExportIntegerAggregateRegisterExhaustion(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
type pair struct{first uint64;second uint64}
// renvo:object function consume - 0 1 0 - 8 0 - 0
//export consume
func consume(a uintptr,b uintptr,c uintptr,d uintptr,e uintptr,value pair,tail uintptr) uintptr{return a+b+c+d+e+value.first+value.second+tail}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("register-exhausted integer aggregate source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("register-exhausted integer aggregate metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("register-exhausted integer aggregate export did not compile")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse register-exhausted integer aggregate object: %v", err)
	}
	found := false
	for _, section := range file.Sections {
		if section.Flags&elf.SHF_EXECINSTR == 0 {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		found = found || bytes.Contains(data, []byte("\x4c\x8d\x5c\x24\x30\x57\x56\x52\x51\x41\x50\x41\xff\x73\x00\x41\xff\x73\x08\x41\x51"))
	}
	if !found {
		t.Fatal("register-exhausted aggregate was split instead of passed wholly on the stack")
	}
}

func TestLinuxAmd64ObjectUserStoreExceptionTable(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
func renvo_runtime_CUserStore64_Efault(address uintptr,value uint64){}
// renvo:object function invoke - 0 1 0 - 8 0 - 0
//export invoke
func invoke(address uintptr,value uint64) int32 {
	renvo_runtime_CUserStore64_Efault(address,value)
	if false { goto Efault }
	return 0
Efault:
	return -14
}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("user-store exception-table source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("user-store exception-table metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("user-store exception-table object compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse user-store exception-table object: %v", err)
	}
	foundStore := false
	for _, section := range file.Sections {
		if section.Flags&elf.SHF_EXECINSTR == 0 {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		foundStore = foundStore || bytes.Contains(data, []byte("\x48\x89\x02"))
	}
	if !foundStore {
		t.Fatal("user-store object is missing its faulting 64-bit store")
	}
	exTable := file.Section("__ex_table")
	if exTable == nil {
		t.Fatal("user-store object has no __ex_table section")
	}
	exData, err := exTable.Data()
	if err != nil {
		t.Fatal(err)
	}
	if len(exData) != 12 || binary.LittleEndian.Uint32(exData[8:12]) != 3 {
		t.Fatalf("user-store exception-table data = %x, want two offsets and type 3", exData)
	}
	relocationCount := 0
	for _, section := range file.Sections {
		if section.Type != elf.SHT_RELA || int(section.Info) >= len(file.Sections) || file.Sections[section.Info].Name != "__ex_table" {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		for at := 0; at+24 <= len(data); at += 24 {
			if elf.R_X86_64(binary.LittleEndian.Uint64(data[at+8:at+16])&0xffffffff) == elf.R_X86_64_PC32 {
				relocationCount++
			}
		}
	}
	if relocationCount != 2 {
		t.Fatalf("user-store exception table has %d PC-relative relocations, want 2", relocationCount)
	}
}

func TestLinuxAmd64ObjectUserLoadExceptionTable(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
func renvo_runtime_CUserLoad64_Efault(address uintptr) uint64{return 0}
// renvo:object function invoke - 0 1 0 - 8 0 - 0
//export invoke
func invoke(address uintptr) uint64 {
	value := renvo_runtime_CUserLoad64_Efault(address)
	if false { goto Efault }
	return value
Efault:
	return ^uint64(13)
}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("user-load exception-table source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("user-load exception-table metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("user-load exception-table object compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse user-load exception-table object: %v", err)
	}
	foundLoad := false
	for _, section := range file.Sections {
		if section.Flags&elf.SHF_EXECINSTR == 0 {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		foundLoad = foundLoad || bytes.Contains(data, []byte("\x48\x8b\x00"))
	}
	if !foundLoad {
		t.Fatal("user-load object is missing its faulting 64-bit load")
	}
	exTable := file.Section("__ex_table")
	if exTable == nil {
		t.Fatal("user-load object has no __ex_table section")
	}
	exData, err := exTable.Data()
	if err != nil {
		t.Fatal(err)
	}
	if len(exData) != 12 || binary.LittleEndian.Uint32(exData[8:12]) != 3 {
		t.Fatalf("user-load exception-table data = %x, want two offsets and type 3", exData)
	}
	relocationCount := 0
	for _, section := range file.Sections {
		if section.Type != elf.SHT_RELA || int(section.Info) >= len(file.Sections) || file.Sections[section.Info].Name != "__ex_table" {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		for at := 0; at+24 <= len(data); at += 24 {
			if elf.R_X86_64(binary.LittleEndian.Uint64(data[at+8:at+16])&0xffffffff) == elf.R_X86_64_PC32 {
				relocationCount++
			}
		}
	}
	if relocationCount != 2 {
		t.Fatalf("user-load exception table has %d PC-relative relocations, want 2", relocationCount)
	}
}

func TestLinuxAmd64ObjectExceptionLoadTable(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
func renvo_runtime_CExceptionLoad64(address uintptr) uint64{return 0}
// renvo:object function invoke - 0 1 0 - 8 0 - 0
//export invoke
func invoke(address uintptr) uint64 {
	return renvo_runtime_CExceptionLoad64(address)
}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("exception-load source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("exception-load metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("exception-load object compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse exception-load object: %v", err)
	}
	foundLoad := false
	for _, section := range file.Sections {
		if section.Flags&elf.SHF_EXECINSTR == 0 {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		foundLoad = foundLoad || bytes.Contains(data, []byte("\x48\x8b\x00"))
	}
	if !foundLoad {
		t.Fatal("exception-load object is missing its faulting 64-bit load")
	}
	exTable := file.Section("__ex_table")
	if exTable == nil {
		t.Fatal("exception-load object has no __ex_table section")
	}
	exData, err := exTable.Data()
	if err != nil {
		t.Fatal(err)
	}
	if len(exData) != 12 || binary.LittleEndian.Uint32(exData[8:12]) != 20 {
		t.Fatalf("exception-load table data = %x, want two offsets and type 20", exData)
	}
	relocationCount := 0
	for _, section := range file.Sections {
		if section.Type != elf.SHT_RELA || int(section.Info) >= len(file.Sections) || file.Sections[section.Info].Name != "__ex_table" {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		for at := 0; at+24 <= len(data); at += 24 {
			if elf.R_X86_64(binary.LittleEndian.Uint64(data[at+8:at+16])&0xffffffff) == elf.R_X86_64_PC32 {
				relocationCount++
			}
		}
	}
	if relocationCount != 2 {
		t.Fatalf("exception-load table has %d PC-relative relocations, want 2", relocationCount)
	}
}

func TestLinuxAmd64ObjectSafeSegmentExceptionTable(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
func renvo_runtime_CWriteSegmentSafees(value uint16){}
// renvo:object function invoke - 0 1 0 - 8 0 - 0
//export invoke
func invoke(value uint16){renvo_runtime_CWriteSegmentSafees(value)}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("safe-segment exception-table source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("safe-segment exception-table metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("safe-segment exception-table object compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse safe-segment exception-table object: %v", err)
	}
	foundLoad := false
	for _, section := range file.Sections {
		if section.Flags&elf.SHF_EXECINSTR == 0 {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		foundLoad = foundLoad || bytes.Contains(data, []byte("\x8e\xc0"))
	}
	if !foundLoad {
		t.Fatal("safe-segment object is missing the ES load instruction")
	}
	exTable := file.Section("__ex_table")
	if exTable == nil {
		t.Fatal("safe-segment object has no __ex_table section")
	}
	exData, err := exTable.Data()
	if err != nil {
		t.Fatal(err)
	}
	if len(exData) != 12 || binary.LittleEndian.Uint32(exData[8:12]) != 17 {
		t.Fatalf("safe-segment exception-table data = %x, want two offsets and type 17", exData)
	}
	relocationCount := 0
	for _, section := range file.Sections {
		if section.Type != elf.SHT_RELA || int(section.Info) >= len(file.Sections) || file.Sections[section.Info].Name != "__ex_table" {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		for at := 0; at+24 <= len(data); at += 24 {
			if elf.R_X86_64(binary.LittleEndian.Uint64(data[at+8:at+16])&0xffffffff) == elf.R_X86_64_PC32 {
				relocationCount++
			}
		}
	}
	if relocationCount != 2 {
		t.Fatalf("safe-segment exception table has %d PC-relative relocations, want 2", relocationCount)
	}
}

func TestLinuxAmd64ObjectRelativeRelocationToForeignFunction(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
// renvo:linkstatic libc,__SCT__target
func __SCT__target(){}
// renvo:object variable entry .static_call_tramp_key 4 0 0 - 4 2 __SCT__target 0
var entry [4]uint8=[4]uint8{}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("foreign-function relocation source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("foreign-function relocation metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("foreign-function PC-relative relocation compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse foreign-function relocation object: %v", err)
	}
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatalf("read foreign-function relocation symbols: %v", err)
	}
	found := false
	for _, section := range file.Sections {
		if section.Type != elf.SHT_RELA {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		for at := 0; at+24 <= len(data); at += 24 {
			info := binary.LittleEndian.Uint64(data[at+8 : at+16])
			symbolIndex := int(info >> 32)
			found = found || elf.R_X86_64(info&0xffffffff) == elf.R_X86_64_PC32 &&
				symbolIndex > 0 && symbolIndex-1 < len(symbols) && symbols[symbolIndex-1].Name == "__SCT__target"
		}
	}
	if !found {
		t.Fatal("object has no PC-relative relocation to the foreign static-call target")
	}
}

func TestLinuxAmd64C11ObjectDoesNotFoldForeignStubResult(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
// renvo:c11
// renvo:linkstatic libc,query
func query() uintptr{return 0}
// renvo:linkstatic libc,observe
func observe(){}
// renvo:object function invoke - 0 1 0 - 8 0 - 0
//export invoke
func invoke() uintptr{value:=query();if value==0{return 1};observe();return 2}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("C11 foreign-result source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("C11 foreign-result metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("C11 foreign-result object did not compile")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse C11 foreign-result object: %v", err)
	}
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	queryFound := false
	observeFound := false
	for _, symbol := range symbols {
		queryFound = queryFound || symbol.Name == "query" && symbol.Section == elf.SHN_UNDEF
		observeFound = observeFound || symbol.Name == "observe" && symbol.Section == elf.SHN_UNDEF
	}
	if !queryFound || !observeFound {
		t.Fatalf("C11 foreign-result imports: query=%t observe=%t", queryFound, observeFound)
	}
}

func TestLinuxAmd64C11ObjectInlinesBooleanIntegerNormalization(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
// renvo:c11
func __c_bool_int(value bool) int32{if value{return 1};return 0}
// renvo:object function normalize - 0 1 0 - 4 0 - 0
//export normalize
func normalize(value uintptr) int32{return __c_bool_int(value!=0)}
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("C11 boolean normalization source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("C11 boolean normalization metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("C11 boolean normalization object did not compile")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse C11 boolean normalization object: %v", err)
	}
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	for _, symbol := range symbols {
		if symbol.Name == "__c_bool_int" {
			t.Fatal("C11 boolean normalization retained an out-of-line helper")
		}
	}
}

func TestLinuxAmd64ObjectNestedSelfPointerInitializer(t *testing.T) {
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
	renvoSetTarget(renvoTargetLinuxAmd64)
	source := []byte(`package main
type list struct{next *list;prev *list}
type work struct{data uint64;entry list;dead *byte}
type indirect struct{__c_align uint64}
func __c_pointer_step_inc_1(pointer *byte,count uintptr)*byte{return pointer}
func(p *indirect)__c_ptr_0_nested()*indirect{return p}
func(p *indirect)__c_nested_ptr_99_0_entry()*list{return nil}
func __c_object_aggregate_0_x1(entry list) indirect{return indirect{}}
// renvo:object variable self .data 8 0 0 - 32 0 - 0
var self work=work{data:1,entry:list{next:&self.entry,prev:&self.entry},dead:(*byte)(__c_pointer_step_inc_1((*byte)(0x300),uintptr(^uint64(2401263026318606335))))}
// renvo:object variable indirect_self .data 8 0 0 - 16 0 - 0
var indirect_self indirect=__c_object_aggregate_0_x1(list{next:&((*indirect_self.__c_ptr_0_nested().__c_nested_ptr_99_0_entry())),prev:&((*indirect_self.__c_ptr_0_nested().__c_nested_ptr_99_0_entry()))})
`)
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext(source, context)
	if !program.ok {
		t.Fatal("nested self-pointer initializer source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("nested self-pointer initializer metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	result := renvoTryCompileScalarProgramAmd64(&program, &meta)
	if !result.ok {
		t.Fatal("nested self-pointer initializer object compilation failed")
	}
	file, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse nested self-pointer initializer object: %v", err)
	}
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	relocations := 0
	indirectRelocations := 0
	for _, section := range file.Sections {
		if section.Type != elf.SHT_RELA {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		for at := 0; at+24 <= len(data); at += 24 {
			symbolIndex := int(binary.LittleEndian.Uint64(data[at+8:at+16]) >> 32)
			addend := int64(binary.LittleEndian.Uint64(data[at+16 : at+24]))
			if symbolIndex > 0 && symbolIndex-1 < len(symbols) && symbols[symbolIndex-1].Name == "self" && addend == 8 {
				relocations++
			}
			if symbolIndex > 0 && symbolIndex-1 < len(symbols) && symbols[symbolIndex-1].Name == "indirect_self" && addend == 0 {
				indirectRelocations++
			}
		}
	}
	if relocations != 2 {
		t.Fatalf("nested self-pointer initializer has %d self+8 relocations, want 2", relocations)
	}
	if indirectRelocations != 2 {
		t.Fatalf("indirect self-pointer initializer has %d self relocations, want 2", indirectRelocations)
	}
	var selfSymbol *elf.Symbol
	for i := range symbols {
		if symbols[i].Name == "self" {
			selfSymbol = &symbols[i]
			break
		}
	}
	if selfSymbol == nil || int(selfSymbol.Section) >= len(file.Sections) {
		t.Fatal("nested self-pointer initializer has no defined self symbol")
	}
	sectionData, err := file.Sections[selfSymbol.Section].Data()
	if err != nil {
		t.Fatal(err)
	}
	at := int(selfSymbol.Value) + 24
	if at+8 > len(sectionData) {
		t.Fatal("nested self-pointer initializer data is truncated")
	}
	base, complement := uint64(0x300), ^uint64(2401263026318606335)
	if got, want := binary.LittleEndian.Uint64(sectionData[at:at+8]), base+complement; got != want {
		t.Fatalf("nested numeric pointer = %#x, want %#x", got, want)
	}
}
