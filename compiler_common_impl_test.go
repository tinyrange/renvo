package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"j5.nz/rtg/rtgunit"
)

func TestTargetProfilesSeparateMachineWidthsFromBackendSlots(t *testing.T) {
	tests := []struct {
		target      int
		arch        int
		os          int
		intBits     int
		pointerBits int
	}{
		{rtgTargetLinuxAmd64, rtgArchAmd64, rtgOSLinux, 64, 64},
		{rtgTargetLinux386, rtgArch386, rtgOSLinux, 32, 32},
		{rtgTargetLinuxAarch64, rtgArchAarch64, rtgOSLinux, 64, 64},
		{rtgTargetLinuxArm, rtgArchArm, rtgOSLinux, 32, 32},
		{rtgTargetWindowsAmd64, rtgArchAmd64, rtgOSWindows, 64, 64},
		{rtgTargetWindows386, rtgArch386, rtgOSWindows, 32, 32},
		{rtgTargetWasiWasm32, rtgArchWasm32, rtgOSWasi, 32, 32},
		{rtgTargetDarwinArm64, rtgArchAarch64, rtgOSDarwin, 64, 64},
	}
	for _, test := range tests {
		p, ok := rtgProfileForTarget(test.target)
		if !ok || !rtgProfileIsValid(p) {
			t.Fatalf("target %d profile invalid: %#v", test.target, p)
		}
		if p.arch != test.arch || p.os != test.os || p.intBits != test.intBits || p.pointerBits != test.pointerBits {
			t.Fatalf("target %d profile = %#v", test.target, p)
		}
		if p.backendSlotSize != rtgBackendValueSlotSize {
			t.Fatalf("target %d backend slot = %d, want %d", test.target, p.backendSlotSize, rtgBackendValueSlotSize)
		}
		if p.codePointerBits != p.pointerBits || p.funcPointerBits != p.pointerBits || p.addressModel != rtgAddressModelFlat {
			t.Fatalf("target %d flat pointer model = %#v", test.target, p)
		}
		if p.floatModel != rtgFloatScaledInteger {
			t.Fatalf("target %d float model = %d, want explicitly documented scaled-integer compatibility mode", test.target, p.floatModel)
		}
		if !rtgProfileHasRuntime(p, rtgRuntimePrint|rtgRuntimeRead|rtgRuntimeWrite) {
			t.Fatalf("target %d missing required runtime capabilities: %#v", test.target, p)
		}
	}
	if _, ok := rtgProfileForTarget(999); ok {
		t.Fatal("unknown target unexpectedly has a profile")
	}
	p, _ := rtgProfileForTarget(rtgTargetLinuxAmd64)
	p.charBits = 7
	if rtgProfileIsValid(p) {
		t.Fatal("profile accepted CHAR_BIT < 8")
	}
	p, _ = rtgProfileForTarget(rtgTargetLinuxAmd64)
	p.pointerBits = 8
	if rtgProfileIsValid(p) {
		t.Fatal("profile accepted unsupported pointer width")
	}
}

func TestFreestandingProfileRequiresExplicitRuntimeContracts(t *testing.T) {
	p, _ := rtgProfileForTarget(rtgTargetLinux386)
	p.runtimeCaps = rtgRuntimePrint | rtgRuntimeHeap | rtgRuntimeVolatileMemory | rtgRuntimeInterrupts
	p.heapModel = rtgHeapBump
	p.oomModel = rtgOOMResult
	p.volatileWidths = rtgVolatileWidth8 | rtgVolatileWidth16 | rtgVolatileWidth32
	p.interruptModel = rtgInterruptVector
	p.addressModel = rtgAddressModelHarvard
	p.pointerBits = 16
	p.codePointerBits = 24
	p.funcPointerBits = 24
	p.maxAlign = 2
	p.floatModel = rtgFloatIEEESoft
	if !rtgProfileIsValid(p) {
		t.Fatalf("explicit freestanding profile rejected: %#v", p)
	}

	p.heapModel = rtgHeapNone
	if rtgProfileIsValid(p) {
		t.Fatal("heap capability accepted without an allocation model")
	}
	p.heapModel = rtgHeapBump
	p.volatileWidths = 0
	if rtgProfileIsValid(p) {
		t.Fatal("volatile-memory capability accepted without supported access widths")
	}
	p.volatileWidths = rtgVolatileWidth8
	p.interruptModel = rtgInterruptNone
	if rtgProfileIsValid(p) {
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
		got, ok := rtgParsePositiveDecimal(test.value)
		if got != test.want || ok != test.ok {
			t.Fatalf("rtgParsePositiveDecimal(%q) = (%d, %v), want (%d, %v)", test.value, got, ok, test.want, test.ok)
		}
	}

	oldSize := rtgCompilerArenaSize
	oldArch := rtgTargetArch
	t.Cleanup(func() {
		rtgCompilerArenaSize = oldSize
		rtgTargetArch = oldArch
	})
	rtgCompilerArenaSize = 2048
	rtgTargetArch = rtgArchAmd64
	if got := rtgStringArenaSize(); got != 2048 {
		t.Fatalf("configured arena size = %d, want 2048", got)
	}
}

func TestPointerTypesRetainAddressSpace(t *testing.T) {
	var m rtgMeta
	dataPointer := rtgAddPointerType(&m, 0, rtgPointerSpaceData)
	codePointer := rtgAddPointerType(&m, 0, rtgPointerSpaceCode)
	functionPointer := rtgAddPointerType(&m, 0, rtgPointerSpaceFunction)
	if rtgPointerAddressSpace(&m, dataPointer) != rtgPointerSpaceData {
		t.Fatal("data pointer lost its address space")
	}
	if rtgPointerAddressSpace(&m, codePointer) != rtgPointerSpaceCode {
		t.Fatal("code pointer lost its address space")
	}
	if rtgPointerAddressSpace(&m, functionPointer) != rtgPointerSpaceFunction {
		t.Fatal("function pointer lost its address space")
	}
	if m.types[dataPointer].size != rtgBackendValueSlotSize {
		t.Fatalf("pointer backend value size = %d, want %d", m.types[dataPointer].size, rtgBackendValueSlotSize)
	}
}

func TestExpressionParserCapacityTracksTokenRange(t *testing.T) {
	oldFixedTarget := rtgCompilerFixedTarget
	rtgCompilerFixedTarget = 0
	t.Cleanup(func() { rtgCompilerFixedTarget = oldFixedTarget })

	program := rtgParseProgram([]byte("package main\nvar value = 1 + 2\n"))
	if !program.ok {
		t.Fatal("test program did not parse")
	}
	var expression rtgExprParse
	rtgParseExpressionInto(&expression, &program, 5, 8)
	if !expression.ok {
		t.Fatal("test expression did not parse")
	}
	if cap(expression.exprs) > 8 || cap(expression.args) > 8 || cap(expression.fields) > 4 {
		t.Fatalf("small expression retained oversized scratch storage: exprs=%d args=%d fields=%d",
			cap(expression.exprs), cap(expression.args), cap(expression.fields))
	}
}

func TestBuildMetaHandlesMoreThanInitialFuncCapacity(t *testing.T) {
	src := []byte("package main\n")
	for i := 0; i < 1301; i++ {
		name := strconv.Itoa(i)
		src = append(src, []byte("func f"+name+"() int { return "+name+" }\n")...)
	}
	src = append(src, []byte("func appMain(args []string, env []string) int { return f1300() }\n")...)

	prog := rtgParseProgram(src)
	if !prog.ok {
		t.Fatalf("failed to parse generated source")
	}
	meta := rtgBuildMeta(&prog)
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
	program.Decls = []rtgunit.Decl{{
		Kind:      rtgTokConst,
		NameStart: unitTokenStart(program, bodyOK),
		NameEnd:   unitTokenEnd(program, bodyOK),
		StartTok:  bodyOK,
		EndTok:    iotaTok + 1,
	}}
	data, err := rtgunit.Marshal(program)
	if err != nil {
		t.Fatalf("unit marshal failed: %v", err)
	}
	prog, isUnit, ok := rtgDecodeUnitProgram(data)
	if !isUnit || !ok {
		t.Fatalf("unit decode failed: isUnit=%v ok=%v", isUnit, ok)
	}
	meta := rtgBuildMeta(&prog)
	if !meta.ok {
		t.Fatalf("metadata failed for grouped const spec-start unit row")
	}
	bodyErr := -1
	for i := 0; i < len(meta.globals); i++ {
		global := meta.globals[i]
		if rtgBytesEqualText(prog.src, global.nameStart, global.nameEnd, "BodyErrFunc") {
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
	data, ok := RtgCompileSourceToBytes(src, "linux/amd64")
	if !ok {
		t.Fatalf("RtgCompileSourceToBytes failed")
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
	data, ok := RtgCompileSourceToBytesStrip(src, "darwin/arm64", true)
	if !ok {
		t.Fatal("RtgCompileSourceToBytesStrip failed")
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
	if _, ok := RtgCompileSourceToBytes(src, "darwin/arm64"); ok {
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
	if _, ok := RtgCompileSourceToBytes(src, "linux/amd64"); ok {
		t.Fatal("method from a different receiver type compiled successfully")
	}
}

func unitProgramFromSource(t *testing.T, src []byte) rtgunit.Program {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	if err := os.WriteFile(path, src, 0644); err != nil {
		t.Fatalf("failed to write unit source: %v", err)
	}
	program, err := rtgunit.ConvertFiles([]string{path})
	if err != nil {
		t.Fatalf("unit conversion failed: %v", err)
	}
	return program
}

func unitTokenByText(t *testing.T, program rtgunit.Program, text string) int {
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

func unitTokenStart(program rtgunit.Program, tok int) int {
	pos := tok * 8
	return int(program.Tokens[pos+1]) | int(program.Tokens[pos+2])<<8 | int(program.Tokens[pos+3])<<16
}

func unitTokenEnd(program rtgunit.Program, tok int) int {
	pos := tok * 8
	size := int(program.Tokens[pos+4])
	if int(program.Tokens[pos]) != rtgTokOp {
		size = size | int(program.Tokens[pos+5])<<8
	}
	return unitTokenStart(program, tok) + size
}

func TestLinkStaticAddsWindowsImport(t *testing.T) {
	src := []byte(`package main

// rtg:linkstatic user32.dll,MessageBeep
func messageBeep(kind int) int { return 0 }

func appMain(args []string, env []string) int {
	return messageBeep(0)
}
`)
	for _, target := range []string{"windows/amd64", "windows/386"} {
		target := target
		t.Run(target, func(t *testing.T) {
			data, ok := RtgCompileSourceToBytes(src, target)
			if !ok {
				t.Fatalf("RtgCompileSourceToBytes failed")
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
	var asm rtgAsm
	rtgAsmInit(&asm)
	rtgWinAmd64CallStaticImport(&asm, 0, 2)
	// pop rcx; pop rdx; sub rsp,40
	want := []byte{0x59, 0x5a, 0x48, 0x83, 0xec, 40}
	match := len(asm.code) >= len(want)
	for i := 0; match && i < len(want); i++ {
		match = asm.code[i] == want[i]
	}
	if !match {
		t.Fatalf("linkstatic call prefix = % x, want % x", asm.code, want)
	}
}
