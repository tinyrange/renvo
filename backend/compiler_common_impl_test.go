package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"renvo.dev/backend/unit"
)

func TestTargetProfilesSeparateMachineWidthsFromBackendSlots(t *testing.T) {
	tests := []struct {
		target      int
		arch        int
		os          int
		intBits     int
		pointerBits int
	}{
		{renvoTargetLinuxAmd64, renvoArchAmd64, renvoOSLinux, 64, 64},
		{renvoTargetLinux386, renvoArch386, renvoOSLinux, 32, 32},
		{renvoTargetLinuxAarch64, renvoArchAarch64, renvoOSLinux, 64, 64},
		{renvoTargetLinuxArm, renvoArchArm, renvoOSLinux, 32, 32},
		{renvoTargetWindowsAmd64, renvoArchAmd64, renvoOSWindows, 64, 64},
		{renvoTargetWindows386, renvoArch386, renvoOSWindows, 32, 32},
		{renvoTargetWindowsArm64, renvoArchAarch64, renvoOSWindows, 64, 64},
		{renvoTargetWasiWasm32, renvoArchWasm32, renvoOSWasi, 32, 32},
		{renvoTargetDarwinArm64, renvoArchAarch64, renvoOSDarwin, 64, 64},
	}
	for _, test := range tests {
		p, ok := renvoProfileForTarget(test.target)
		if !ok || !renvoProfileIsValid(p) {
			t.Fatalf("target %d profile invalid: %#v", test.target, p)
		}
		if p.arch != test.arch || p.os != test.os || p.intBits != test.intBits || p.pointerBits != test.pointerBits {
			t.Fatalf("target %d profile = %#v", test.target, p)
		}
		if p.backendSlotSize != renvoBackendValueSlotSize {
			t.Fatalf("target %d backend slot = %d, want %d", test.target, p.backendSlotSize, renvoBackendValueSlotSize)
		}
		if p.codePointerBits != p.pointerBits || p.funcPointerBits != p.pointerBits || p.addressModel != renvoAddressModelFlat {
			t.Fatalf("target %d flat pointer model = %#v", test.target, p)
		}
		if p.floatModel != renvoFloatScaledInteger {
			t.Fatalf("target %d float model = %d, want explicitly documented scaled-integer compatibility mode", test.target, p.floatModel)
		}
		if !renvoProfileHasRuntime(p, renvoRuntimePrint|renvoRuntimeRead|renvoRuntimeWrite) {
			t.Fatalf("target %d missing required runtime capabilities: %#v", test.target, p)
		}
	}
	if _, ok := renvoProfileForTarget(999); ok {
		t.Fatal("unknown target unexpectedly has a profile")
	}
	p, _ := renvoProfileForTarget(renvoTargetLinuxAmd64)
	p.charBits = 7
	if renvoProfileIsValid(p) {
		t.Fatal("profile accepted CHAR_BIT < 8")
	}
	p, _ = renvoProfileForTarget(renvoTargetLinuxAmd64)
	p.pointerBits = 8
	if renvoProfileIsValid(p) {
		t.Fatal("profile accepted unsupported pointer width")
	}
}

func TestFreestandingProfileRequiresExplicitRuntimeContracts(t *testing.T) {
	p, _ := renvoProfileForTarget(renvoTargetLinux386)
	p.runtimeCaps = renvoRuntimePrint | renvoRuntimeHeap | renvoRuntimeVolatileMemory | renvoRuntimeInterrupts
	p.heapModel = renvoHeapBump
	p.oomModel = renvoOOMResult
	p.volatileWidths = renvoVolatileWidth8 | renvoVolatileWidth16 | renvoVolatileWidth32
	p.interruptModel = renvoInterruptVector
	p.addressModel = renvoAddressModelHarvard
	p.pointerBits = 16
	p.codePointerBits = 24
	p.funcPointerBits = 24
	p.maxAlign = 2
	p.floatModel = renvoFloatIEEESoft
	if !renvoProfileIsValid(p) {
		t.Fatalf("explicit freestanding profile rejected: %#v", p)
	}

	p.heapModel = renvoHeapNone
	if renvoProfileIsValid(p) {
		t.Fatal("heap capability accepted without an allocation model")
	}
	p.heapModel = renvoHeapBump
	p.volatileWidths = 0
	if renvoProfileIsValid(p) {
		t.Fatal("volatile-memory capability accepted without supported access widths")
	}
	p.volatileWidths = renvoVolatileWidth8
	p.interruptModel = renvoInterruptNone
	if renvoProfileIsValid(p) {
		t.Fatal("interrupt capability accepted without an interrupt ABI")
	}
}

func TestArenaSizeConfigurationIsBounded(t *testing.T) {
	for _, test := range []struct {
		value string
		want  int
		ok    bool
	}{
		{"256", 256, true},
		{"2048", 2048, true},
		{"1073741824", 1073741824, true},
		{"", 0, false},
		{"255", 0, false},
		{"1073741825", 0, false},
		{"2k", 0, false},
	} {
		got, ok := renvoParsePositiveDecimal(test.value)
		if got != test.want || ok != test.ok {
			t.Fatalf("renvoParsePositiveDecimal(%q) = (%d, %v), want (%d, %v)", test.value, got, ok, test.want, test.ok)
		}
	}

	oldArch := targetArch
	oldOS := targetOS
	oldTarget := currentTarget
	t.Cleanup(func() {
		targetArch = oldArch
		targetOS = oldOS
		currentTarget = oldTarget
	})
	g := renvoLinearGen{arenaSize: 2048}
	if got := renvoStringArenaSize(&g); got != 2048 {
		t.Fatalf("configured arena size = %d, want 2048", got)
	}
	g.arenaSize = 0
	renvoSetTarget(renvoTargetWindows386)
	if got := renvoStringArenaSize(&g); got != renvoArenaSize32BitHosted {
		t.Fatalf("Windows/386 arena size = %d, want %d", got, renvoArenaSize32BitHosted)
	}
}

func TestLargeStaticSliceZeroingHasBoundedCodeSize(t *testing.T) {
	oldArch := targetArch
	oldOS := targetOS
	oldFixedTarget := compilerFixedTarget
	t.Cleanup(func() {
		targetArch = oldArch
		targetOS = oldOS
		compilerFixedTarget = oldFixedTarget
	})
	renvoSetTarget(renvoTargetLinuxAmd64)
	compilerFixedTarget = renvoTargetLinuxAmd64

	var g renvoLinearGen
	var meta renvoMeta
	g.meta = &meta
	renvoAsmInit(&g.asm)
	renvoEmitMakeStaticRingPrimary(&g, 65536*8, 65536*8)

	if got := len(g.asm.code); got > 512 {
		t.Fatalf("large static make emitted %d bytes of code, want at most 512", got)
	}
	if !g.makeZeroEmitted {
		t.Fatal("large static make did not use the bounded zeroing helper")
	}
}

func TestPointerTypesRetainAddressSpace(t *testing.T) {
	var m renvoMeta
	dataPointer := renvoAddPointerType(&m, 0, renvoPointerSpaceData)
	codePointer := renvoAddPointerType(&m, 0, renvoPointerSpaceCode)
	functionPointer := renvoAddPointerType(&m, 0, renvoPointerSpaceFunction)
	if renvoPointerAddressSpace(&m, dataPointer) != renvoPointerSpaceData {
		t.Fatal("data pointer lost its address space")
	}
	if renvoPointerAddressSpace(&m, codePointer) != renvoPointerSpaceCode {
		t.Fatal("code pointer lost its address space")
	}
	if renvoPointerAddressSpace(&m, functionPointer) != renvoPointerSpaceFunction {
		t.Fatal("function pointer lost its address space")
	}
	if m.types[dataPointer].size != renvoBackendValueSlotSize {
		t.Fatalf("pointer backend value size = %d, want %d", m.types[dataPointer].size, renvoBackendValueSlotSize)
	}
}

func TestStructLayoutsFollowTargetFieldAlignment(t *testing.T) {
	oldFixedTarget := compilerFixedTarget
	oldCurrentTarget := currentTarget
	oldOS := targetOS
	oldArch := targetArch
	oldIntSize := renvoNativeIntSize
	t.Cleanup(func() {
		compilerFixedTarget = oldFixedTarget
		currentTarget = oldCurrentTarget
		targetOS = oldOS
		targetArch = oldArch
		renvoNativeIntSize = oldIntSize
	})
	compilerFixedTarget = 0
	renvoSetTarget(renvoTargetWindowsAmd64)

	program := renvoParseProgram([]byte(`package main

type pair struct {
	A uint32
	B uint32
}

type mixed struct {
	A byte
	B uint32
	C uint16
	D byte
}

type outer struct {
	Flag byte
	Pair pair
	Tail uint16
}

type arrayed struct {
	Flag byte
	Values [2]uint32
	Tail byte
}

// renvo:linkstatic test.dll,consumeLayouts
func consumeLayouts(pairValue *pair, mixedValue *mixed, outerValue *outer, arrayedValue *arrayed) {}
`))
	if !program.ok {
		t.Fatal("struct layout test program did not parse")
	}
	meta := renvoBuildMeta(&program)
	if !meta.ok {
		t.Fatal("struct layout test metadata failed")
	}

	tests := []struct {
		name    string
		offsets []int
		size    int
	}{
		{name: "pair", offsets: []int{0, 4}, size: 8},
		{name: "mixed", offsets: []int{0, 4, 8, 10}, size: 12},
		{name: "outer", offsets: []int{0, 4, 12}, size: 16},
		{name: "arrayed", offsets: []int{0, 4, 12}, size: 16},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			typeIndex := -1
			for i := 0; i < len(meta.types); i++ {
				if renvoBytesEqualText(program.src, meta.types[i].nameStart, meta.types[i].nameEnd, test.name) {
					typeIndex = i
					break
				}
			}
			if typeIndex < 0 {
				t.Fatalf("type %s not found", test.name)
			}
			resolved := renvoResolveType(&meta, typeIndex)
			if resolved.size != test.size || resolved.count != len(test.offsets) {
				t.Fatalf("%s layout = size %d fields %d, want size %d fields %d", test.name, resolved.size, resolved.count, test.size, len(test.offsets))
			}
			for i := 0; i < len(test.offsets); i++ {
				got := meta.fields[resolved.first+i].offset
				if got != test.offsets[i] {
					t.Fatalf("%s field %d offset = %d, want %d", test.name, i, got, test.offsets[i])
				}
			}
		})
	}
}

func TestStructLayoutHonorsTargetMaximumAlignment(t *testing.T) {
	oldFixedTarget := compilerFixedTarget
	oldCurrentTarget := currentTarget
	oldOS := targetOS
	oldArch := targetArch
	oldIntSize := renvoNativeIntSize
	t.Cleanup(func() {
		compilerFixedTarget = oldFixedTarget
		currentTarget = oldCurrentTarget
		targetOS = oldOS
		targetArch = oldArch
		renvoNativeIntSize = oldIntSize
	})
	compilerFixedTarget = 0

	tests := []struct {
		name           string
		target         int
		offsets        []int
		size           int
		pointerOffsets []int
		pointerSize    int
	}{
		{name: "windows-amd64", target: renvoTargetWindowsAmd64, offsets: []int{0, 8, 16}, size: 24, pointerOffsets: []int{0, 8, 16}, pointerSize: 24},
		{name: "windows-386", target: renvoTargetWindows386, offsets: []int{0, 4, 12}, size: 16, pointerOffsets: []int{0, 4, 8}, pointerSize: 12},
		{name: "windows-arm64", target: renvoTargetWindowsArm64, offsets: []int{0, 8, 16}, size: 24, pointerOffsets: []int{0, 8, 16}, pointerSize: 24},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			renvoSetTarget(test.target)
			program := renvoParseProgram([]byte(`package main

type wide struct {
	Head byte
	Value uint64
	Tail byte
}

type pointerLayout struct {
	Head byte
	Value *byte
	Tail byte
}

// renvo:linkstatic test.dll,consumeWide
func consumeWide(value *wide, pointerValue *pointerLayout) {}
`))
			meta := renvoBuildMeta(&program)
			if !program.ok || !meta.ok {
				t.Fatal("wide struct metadata failed")
			}
			typeIndex := -1
			for i := 0; i < len(meta.types); i++ {
				if renvoBytesEqualText(program.src, meta.types[i].nameStart, meta.types[i].nameEnd, "wide") {
					typeIndex = i
					break
				}
			}
			if typeIndex < 0 {
				t.Fatal("wide type not found")
			}
			resolved := renvoResolveType(&meta, typeIndex)
			if resolved.size != test.size {
				t.Fatalf("wide size = %d, want %d", resolved.size, test.size)
			}
			for i := 0; i < len(test.offsets); i++ {
				got := meta.fields[resolved.first+i].offset
				if got != test.offsets[i] {
					t.Fatalf("wide field %d offset = %d, want %d", i, got, test.offsets[i])
				}
			}

			pointerIndex := -1
			for i := 0; i < len(meta.types); i++ {
				if renvoBytesEqualText(program.src, meta.types[i].nameStart, meta.types[i].nameEnd, "pointerLayout") {
					pointerIndex = i
					break
				}
			}
			if pointerIndex < 0 {
				t.Fatal("pointerLayout type not found")
			}
			pointerType := renvoResolveType(&meta, pointerIndex)
			if pointerType.size != test.pointerSize {
				t.Fatalf("pointerLayout size = %d, want %d", pointerType.size, test.pointerSize)
			}
			for i := 0; i < len(test.pointerOffsets); i++ {
				got := meta.fields[pointerType.first+i].offset
				if got != test.pointerOffsets[i] {
					t.Fatalf("pointerLayout field %d offset = %d, want %d", i, got, test.pointerOffsets[i])
				}
			}
		})
	}
}

func TestExpressionParserCapacityTracksTokenRange(t *testing.T) {
	oldFixedTarget := compilerFixedTarget
	compilerFixedTarget = 0
	t.Cleanup(func() { compilerFixedTarget = oldFixedTarget })

	program := renvoParseProgram([]byte("package main\nvar value = 1 + 2\n"))
	if !program.ok {
		t.Fatal("test program did not parse")
	}
	var expression renvoExprParse
	renvoParseExpressionInto(&expression, &program, 5, 8)
	if !expression.ok {
		t.Fatal("test expression did not parse")
	}
	if cap(expression.exprs) > 8 || cap(expression.args) > 8 || cap(expression.fields) > 4 {
		t.Fatalf("small expression retained oversized scratch storage: exprs=%d args=%d fields=%d",
			cap(expression.exprs), cap(expression.args), cap(expression.fields))
	}
}

func TestAppendAssignmentRecognizesSameSource(t *testing.T) {
	program := renvoParseProgram([]byte(`package main

func appMain() int {
	var out []byte
	out = append(out, []byte("x")...)
	return len(out)
}
`))
	if !program.ok {
		t.Fatal("test program did not parse")
	}

	lhsTok := -1
	for i := 0; i+2 < renvoTokCount(&program); i++ {
		if renvoBytesEqualText(program.src, renvoTokStart(&program, i), renvoTokEnd(&program, i), "out") &&
			renvoTokCharIs(&program, i+1, '=') && renvoTokIdentIs(&program, i+2, "append") {
			lhsTok = i
			break
		}
	}
	if lhsTok < 0 {
		t.Fatal("append assignment tokens not found")
	}
	assignTok := lhsTok + 1
	appendTok := lhsTok + 2
	openTok := appendTok + 1
	if !renvoTokCharIs(&program, openTok, '(') {
		t.Fatal("append call opening parenthesis not found")
	}
	closeTok := renvoFindMatchingExprClose(&program, openTok+1, renvoTokCount(&program), '(', ')')
	if closeTok <= openTok {
		t.Fatal("append call closing parenthesis not found")
	}

	ep := renvoNewExprParse()
	rootIndex := renvoParseExpressionRoot(ep, &program, appendTok, closeTok+1)
	if rootIndex < 0 {
		t.Fatal("append call expression did not parse")
	}
	stmt := renvoStmt{startTok: lhsTok, endTok: closeTok + 1}
	if !renvoAppendAssignLhsMatchesSource(&program, &stmt, ep, &ep.exprs[rootIndex], assignTok) {
		t.Fatal("out = append(out, ...) was not recognized as an in-place append")
	}
}

func TestBuildMetaHandlesMoreThanInitialFuncCapacity(t *testing.T) {
	src := []byte("package main\n")
	for i := 0; i < 1301; i++ {
		name := strconv.Itoa(i)
		src = append(src, []byte("func f"+name+"() int { return "+name+" }\n")...)
	}
	src = append(src, []byte("func appMain(args []string, env []string) int { return f1300() }\n")...)

	prog := renvoParseProgram(src)
	if !prog.ok {
		t.Fatalf("failed to parse generated source")
	}
	meta := renvoBuildMeta(&prog)
	if !meta.ok {
		t.Fatalf("failed to build metadata")
	}
	if len(meta.funcs) != 1302 {
		t.Fatalf("metadata function count = %d, want 1302", len(meta.funcs))
	}
}

func TestBuildMetaHandlesUnitGroupedConstSpecRows(t *testing.T) {
	program := unitProgramFromSource(t, []byte(`package main

const (
	BodyOK = iota
	BodyErrFunc
)

func appMain(args []string, env []string) int { return BodyErrFunc }
`))
	bodyOK := unitTokenByText(t, program, "BodyOK")
	iotaTok := unitTokenByText(t, program, "iota")
	program.Decls = []unit.Decl{{
		Kind:      renvoTokConst,
		NameStart: unitTokenStart(program, bodyOK),
		NameEnd:   unitTokenEnd(program, bodyOK),
		StartTok:  bodyOK,
		EndTok:    iotaTok + 1,
	}}
	data, err := unit.Marshal(program)
	if err != nil {
		t.Fatalf("unit marshal failed: %v", err)
	}
	prog, isUnit, ok := renvoDecodeUnitProgram(data)
	if !isUnit || !ok {
		t.Fatalf("unit decode failed: isUnit=%v ok=%v", isUnit, ok)
	}
	meta := renvoBuildMeta(&prog)
	if !meta.ok {
		t.Fatalf("metadata failed for grouped const spec-start unit row")
	}
	bodyErr := -1
	for i := 0; i < len(meta.globals); i++ {
		global := meta.globals[i]
		if renvoBytesEqualText(prog.src, global.nameStart, global.nameEnd, "BodyErrFunc") {
			bodyErr = i
			break
		}
	}
	if bodyErr < 0 || meta.globals[bodyErr].constValueOK == 0 || meta.globals[bodyErr].constValue != 1 {
		t.Fatalf("BodyErrFunc const = index %d globals %#v", bodyErr, meta.globals)
	}
}

func TestArbitrarySyscallLinuxAmd64Write(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("linux/amd64 syscall execution test requires linux/amd64 host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	src := []byte(`package main

func syscall(num int, fd int, msg string, n int) int { return 0 }

func appMain(args []string, env []string) int {
	syscall(1, 1, "PASS\n", 5)
	return 0
}

`)
	data, ok := RenvoCompileSourceToBytes(src, "linux/amd64")
	if !ok {
		t.Fatalf("RenvoCompileSourceToBytes failed")
	}
	out := filepath.Join(t.TempDir(), "syscall-write")
	if err := os.WriteFile(out, data, 0755); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	cmd := exec.Command(out)
	got, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled syscall test failed: %v\n%s", err, string(got))
	}
	if string(got) != "PASS\n" {
		t.Fatalf("compiled syscall output = %q, want PASS", string(got))
	}
}

func TestDarwinArm64LibSystemRuntime(t *testing.T) {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Skipf("darwin/arm64 execution test requires darwin/arm64 host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	src := []byte(`package main

func syscall(num int, fd int, buf []byte, size int) int { return 0 }

func appMain(args []string, env []string) int {
	fd := open("darwin-runtime.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 { return 1 }
	if write(fd, []byte("PASS\n"), -1) != 5 { return 2 }
	if chmod(fd, 420) != 0 { return 3 }
	if close(fd) != 0 { return 4 }
	fd = open("darwin-runtime.tmp", O_RDONLY)
	if fd < 0 { return 5 }
	buf := make([]byte, 5)
	if read(fd, buf, -1) != 5 { return 6 }
	if close(fd) != 0 { return 7 }
	fd = open(".", O_RDONLY)
	if fd < 0 { return 8 }
	dirbuf := make([]byte, 4096)
	n := syscall(217, fd, dirbuf, len(dirbuf))
	if close(fd) != 0 { return 9 }
	if n < 12 { return 10 }
	reclen := int(dirbuf[4]) | int(dirbuf[5])<<8
	if reclen < 12 || reclen > n { return 11 }
	if dirbuf[6] != 4 || dirbuf[8] != '.' { return 12 }
print(string(buf))
	return 0
}
`)
	data, ok := RenvoCompileSourceToBytesStrip(src, "darwin/arm64", true)
	if !ok {
		t.Fatal("RenvoCompileSourceToBytesStrip failed")
	}
	dir := t.TempDir()
	out := filepath.Join(dir, "darwin-runtime")
	if err := os.WriteFile(out, data, 0755); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	cmd := exec.Command(out)
	cmd.Dir = dir
	got, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled Darwin test failed: %v\n%s", err, string(got))
	}
	if string(got) != "PASS\n" {
		t.Fatalf("compiled Darwin output = %q, want PASS", string(got))
	}
}

func TestDarwinArm64RejectsUnsupportedArbitrarySyscall(t *testing.T) {
	src := []byte(`package main

func syscall(num int, fd int, buf []byte, size int) int { return 0 }

func appMain(args []string, env []string) int {
	buf := make([]byte, 8)
	return syscall(1, 1, buf, len(buf))
}
`)
	if _, ok := RenvoCompileSourceToBytes(src, "darwin/arm64"); ok {
		t.Fatal("unsupported Darwin syscall compiled successfully")
	}
}

func TestMethodLookupRejectsDifferentReceiverWithSameMethodName(t *testing.T) {
	src := []byte(`package main

type firstReceiver struct { value int }
type secondReceiver struct { value int }

func (receiver firstReceiver) read() int { return receiver.value }

func appMain(args []string, env []string) int {
	var receiver secondReceiver
	return receiver.read()
}
`)
	if _, ok := RenvoCompileSourceToBytes(src, "linux/amd64"); ok {
		t.Fatal("method from a different receiver type compiled successfully")
	}
}

func unitProgramFromSource(t *testing.T, src []byte) unit.Program {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	if err := os.WriteFile(path, src, 0644); err != nil {
		t.Fatalf("failed to write unit source: %v", err)
	}
	program, err := unit.ConvertFiles([]string{path})
	if err != nil {
		t.Fatalf("unit conversion failed: %v", err)
	}
	return program
}

func unitTokenByText(t *testing.T, program unit.Program, text string) int {
	t.Helper()
	count := len(program.Tokens) / 8
	for i := 0; i < count; i++ {
		if string(program.Text[unitTokenStart(program, i):unitTokenEnd(program, i)]) == text {
			return i
		}
	}
	t.Fatalf("token %q not found", text)
	return -1
}

func unitTokenStart(program unit.Program, tok int) int {
	pos := tok * 8
	return int(program.Tokens[pos+1]) | int(program.Tokens[pos+2])<<8 | int(program.Tokens[pos+3])<<16
}

func unitTokenEnd(program unit.Program, tok int) int {
	pos := tok * 8
	size := int(program.Tokens[pos+4])
	if int(program.Tokens[pos]) != renvoTokOp {
		size = size | int(program.Tokens[pos+5])<<8
	}
	return unitTokenStart(program, tok) + size
}

func TestLinkStaticAddsWindowsImport(t *testing.T) {
	src := []byte(`package main

// renvo:linkstatic user32.dll,MessageBeep
func messageBeep(kind int) int { return 0 }

func appMain(args []string, env []string) int {
	return messageBeep(0)
}
`)
	for _, target := range []string{"windows/amd64", "windows/386", "windows/arm64"} {
		target := target
		t.Run(target, func(t *testing.T) {
			data, ok := RenvoCompileSourceToBytes(src, target)
			if !ok {
				t.Fatalf("RenvoCompileSourceToBytes failed")
			}
			text := string(data)
			for _, want := range []string{"user32.dll", "MessageBeep"} {
				if !strings.Contains(text, want) {
					t.Fatalf("windows import table missing %q", want)
				}
			}
		})
	}
}

func TestWindowsAmd64LinkStaticCallReservesAlignedShadowSpace(t *testing.T) {
	renvoSetTarget(renvoTargetWindowsAmd64)
	var asm renvoAsm
	renvoAsmInit(&asm)
	renvoWinAmd64CallStaticImport(&asm, 0, 2)
	// pop rcx; pop rdx; save rsp; align rsp; reserve shadow/save space.
	want := []byte{0x59, 0x5a, 0x49, 0x89, 0xe2, 0x48, 0x83, 0xe4, 0xf0, 0x48, 0x83, 0xec, 48}
	match := len(asm.code) >= len(want)
	for i := 0; match && i < len(want); i++ {
		match = asm.code[i] == want[i]
	}
	if !match {
		t.Fatalf("linkstatic call prefix = % x, want % x", asm.code, want)
	}
}

func TestWindowsAmd64LinkStaticCallAlignsStackArguments(t *testing.T) {
	renvoSetTarget(renvoTargetWindowsAmd64)
	tests := []struct {
		name      string
		wordCount int
		want      []byte
	}{
		{
			name:      "odd stack word count",
			wordCount: 5,
			want: []byte{
				0x59, 0x5a, 0x41, 0x58, 0x41, 0x59,
				0x49, 0x89, 0xe2,
				0x48, 0x83, 0xe4, 0xf0,
				0x48, 0x83, 0xec, 48,
				0x49, 0x8b, 0x42, 0,
				0x48, 0x89, 0x44, 0x24, 32,
			},
		},
		{
			name:      "even stack word count",
			wordCount: 6,
			want: []byte{
				0x59, 0x5a, 0x41, 0x58, 0x41, 0x59,
				0x49, 0x89, 0xe2,
				0x48, 0x83, 0xe4, 0xf0,
				0x48, 0x83, 0xec, 64,
				0x49, 0x8b, 0x42, 0,
				0x48, 0x89, 0x44, 0x24, 32,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var asm renvoAsm
			renvoAsmInit(&asm)
			renvoWinAmd64CallStaticImport(&asm, 0, test.wordCount)
			if len(asm.code) < len(test.want) {
				t.Fatalf("linkstatic call length = %d, want at least %d", len(asm.code), len(test.want))
			}
			for i := 0; i < len(test.want); i++ {
				if asm.code[i] != test.want[i] {
					t.Fatalf("linkstatic call = % x, want % x", asm.code, test.want)
				}
			}
		})
	}
}

func TestWindowsAmd64LinkStaticCallAlignsTwelveWordImport(t *testing.T) {
	renvoSetTarget(renvoTargetWindowsAmd64)
	var asm renvoAsm
	renvoAsmInit(&asm)
	renvoWinAmd64CallStaticImport(&asm, 0, 12)
	wantPrefix := []byte{
		0x59, 0x5a, 0x41, 0x58, 0x41, 0x59,
		0x49, 0x89, 0xe2,
		0x48, 0x83, 0xe4, 0xf0,
		0x48, 0x83, 0xec, 112,
		0x49, 0x8b, 0x42, 0,
		0x48, 0x89, 0x44, 0x24, 32,
	}
	if len(asm.code) < len(wantPrefix) {
		t.Fatalf("12-word linkstatic call length = %d, want at least %d", len(asm.code), len(wantPrefix))
	}
	for i := 0; i < len(wantPrefix); i++ {
		if asm.code[i] != wantPrefix[i] {
			t.Fatalf("12-word call prefix = % x, want % x", asm.code[:len(wantPrefix)], wantPrefix)
		}
	}
	wantCleanup := []byte{
		0x49, 0x89, 0xc2,
		0x48, 0x8b, 0x44, 0x24, 96,
		0x48, 0x89, 0xc4,
		0x48, 0x83, 0xc4, 64,
		0x4c, 0x89, 0xd0,
	}
	foundCleanup := false
	for i := 0; i+len(wantCleanup) <= len(asm.code); i++ {
		matched := true
		for j := 0; j < len(wantCleanup); j++ {
			if asm.code[i+j] != wantCleanup[j] {
				matched = false
			}
		}
		if matched {
			foundCleanup = true
		}
	}
	if !foundCleanup {
		t.Fatalf("12-word linkstatic call missing dynamic cleanup: % x", asm.code)
	}
}
