package main

import (
	"bytes"
	"debug/elf"
	"testing"
)

func testKernelBTF() []byte {
	var types []byte
	types = renvoAppend32(types, 1)
	types = renvoAppend32(types, 4<<24|3)
	types = renvoAppend32(types, 128)
	for _, member := range []struct {
		name int
		off  int
	}{{8, 16}, {13, 32}, {18, 64}} {
		types = renvoAppend32(types, member.name)
		types = renvoAppend32(types, 0)
		types = renvoAppend32(types, member.off*8)
	}
	strings := []byte("\x00module\x00name\x00init\x00exit\x00")
	var out []byte
	out = append(out, 0x9f, 0xeb, 1, 0)
	out = renvoAppend32(out, 24)
	out = renvoAppend32(out, 0)
	out = renvoAppend32(out, len(types))
	out = renvoAppend32(out, len(types))
	out = renvoAppend32(out, len(strings))
	out = append(out, types...)
	return append(out, strings...)
}

func TestKernelBTFModuleLayout(t *testing.T) {
	btf := testKernelBTF()
	size, name, init, exit, ok := renvoKernelBTFModuleLayout(btf)
	if !ok || size != 128 || name != 16 || init != 32 || exit != 64 {
		t.Fatalf("layout = size:%d name:%d init:%d exit:%d ok:%v", size, name, init, exit, ok)
	}
	if _, _, _, _, ok := renvoKernelBTFModuleLayout(testKernelBTF()[:20]); ok {
		t.Fatal("truncated BTF was accepted")
	}
	first := append([]byte(nil), btf[36:48]...)
	copy(btf[36:48], btf[60:72])
	copy(btf[60:72], first)
	size, name, init, exit, ok = renvoKernelBTFModuleLayout(btf)
	if !ok || size != 128 || name != 16 || init != 32 || exit != 64 {
		t.Fatalf("reordered layout = size:%d name:%d init:%d exit:%d ok:%v", size, name, init, exit, ok)
	}
}

func TestKernelModuleNameFromOutput(t *testing.T) {
	tests := map[string]string{
		"hello.ko":            "hello",
		"/tmp/hello-world.ko": "hello_world",
		"/tmp/UPPER_123.ko":   "UPPER_123",
		"-":                   "_",
		"/tmp/abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789.ko": "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ012",
	}
	for input, want := range tests {
		if got := renvoKernelNameFromOutput(input); got != want {
			t.Errorf("name(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestKernelModuleELF(t *testing.T) {
	savedFixed := renvoFixedTarget
	savedRelease := renvoKernelRelease
	savedName := renvoKernelModuleName
	savedLicense := renvoKernelLicense
	savedBTF := renvoKernelBTF
	savedSymvers := renvoKernelSymvers
	savedVersion := renvoKernelVersion
	savedSize := renvoKernelModuleSize
	savedNameOff := renvoKernelModuleNameOff
	savedInitOff := renvoKernelModuleInitOff
	savedExitOff := renvoKernelModuleExitOff
	defer func() {
		renvoFixedTarget = savedFixed
		renvoKernelRelease = savedRelease
		renvoKernelModuleName = savedName
		renvoKernelLicense = savedLicense
		renvoKernelBTF = savedBTF
		renvoKernelSymvers = savedSymvers
		renvoKernelVersion = savedVersion
		renvoKernelModuleSize = savedSize
		renvoKernelModuleNameOff = savedNameOff
		renvoKernelModuleInitOff = savedInitOff
		renvoKernelModuleExitOff = savedExitOff
	}()

	renvoFixedTarget = renvoTargetLinuxKernelAmd64
	renvoSetTarget(renvoTargetLinuxKernelAmd64)
	renvoKernelRelease = "6.18.0-test"
	renvoKernelModuleName = "hello"
	renvoKernelLicense = "GPL"
	renvoKernelBTF = testKernelBTF()
	renvoKernelSymvers = []byte("0xb1976aeb\tmodule_layout\tvmlinux\tEXPORT_SYMBOL\n0x92997ed8\t_printk\tvmlinux\tEXPORT_SYMBOL\n0x11223344\tktime_get_ns\tvmlinux\tEXPORT_SYMBOL_GPL\n0x55667788\tfor_each_kernel_tracepoint\tvmlinux\tEXPORT_SYMBOL_GPL\n")
	renvoKernelVersion = "Linux version 6.18.0-test SMP PREEMPT"
	renvoKernelModuleSize = 128
	renvoKernelModuleNameOff = 16
	renvoKernelModuleInitOff = 32
	renvoKernelModuleExitOff = 64

	prog := renvoParseProgram([]byte("package main\nvar count int\nvar stamp uint64\n// renvo:linkstatic kernel,ktime_get_ns\nfunc kernelKtimeGetNS() uint64 { return 0 }\n// renvo:linkstatic kernel,for_each_kernel_tracepoint\nfunc kernelForEach(callback func(uintptr, uintptr), data uintptr) {}\nfunc callback(tp uintptr, data uintptr) {}\nfunc bump() { count++ }\nfunc appMain() { kernelForEach(callback, 0); stamp = kernelKtimeGetNS(); for i := 0; i < 3; i++ { bump() }; if count == 3 && stamp >= 0 { print(\"100% PASS\\n\") } }\nfunc moduleExit() { kernelForEach(callback, 0); print(\"EXIT\\n\") }\n"))
	renvoCaptureKernelCompileContext(&prog.c)
	if !prog.ok {
		t.Fatal("test program did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&prog, &meta)
	meta.arenaSize = 4096
	result := renvoTryCompileScalarProgramAmd64(&prog, &meta)
	if !result.ok {
		t.Fatal("kernel module compilation failed")
	}
	f, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatalf("parse module ELF: %v", err)
	}
	if f.Type != elf.ET_REL || f.Machine != elf.EM_X86_64 {
		t.Fatalf("ELF type/machine = %v/%v", f.Type, f.Machine)
	}
	for _, name := range []string{".text", ".rela.text", ".modinfo", "__versions", ".gnu.linkonce.this_module", ".rela.gnu.linkonce.this_module"} {
		if f.Section(name) == nil {
			t.Fatalf("missing section %s", name)
		}
	}
	relocations, err := f.Section(".rela.text").Data()
	if err != nil || len(relocations) < 4*24 {
		t.Fatalf("text relocations = %d bytes, err %v", len(relocations), err)
	}
	symbols, err := f.Symbols()
	if err != nil {
		t.Fatalf("read symbols: %v", err)
	}
	want := map[string]bool{"init_module": false, "cleanup_module": false, "__this_module": false, "_printk": false, "ktime_get_ns": false, "for_each_kernel_tracepoint": false}
	for _, symbol := range symbols {
		if _, ok := want[symbol.Name]; ok {
			want[symbol.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Fatalf("missing symbol %s", name)
		}
	}
	modinfo, err := f.Section(".modinfo").Data()
	if err != nil || !bytes.Contains(modinfo, []byte("vermagic=6.18.0-test SMP preempt mod_unload modversions ")) {
		t.Fatalf("modinfo = %q, err %v", modinfo, err)
	}
	if !bytes.Contains(modinfo, []byte("license=GPL")) {
		t.Fatalf("modinfo does not contain GPL license: %q", modinfo)
	}
	text, err := f.Section(".text").Data()
	if err != nil {
		t.Fatal(err)
	}
	target := -1
	leas := 0
	cleanup := -1
	for _, symbol := range symbols {
		if symbol.Name == "cleanup_module" {
			cleanup = int(symbol.Value)
		}
	}
	checkRuntimeRegs := func(name string, at int) {
		if at < 0 || at+80 > len(text) {
			t.Fatalf("%s entry offset = %d", name, at)
		}
		for _, opcode := range [][]byte{{0x4c, 0x8d, 0x25}, {0x4c, 0x8d, 0x2d}, {0x4c, 0x8d, 0x35}, {0x4c, 0x8d, 0x3d}, {0x48, 0x8d, 0x1d}} {
			if !bytes.Contains(text[at:at+80], opcode) {
				t.Fatalf("%s does not initialize runtime check register opcode %x", name, opcode)
			}
		}
	}
	checkRuntimeRegs("cleanup_module", cleanup)
	for i := 0; i+7 <= len(text); i++ {
		if text[i] == 0x48 && text[i+1] == 0x8d && text[i+2] == 0x05 {
			got := i + 7 + int(int32(uint32(text[i+3])|uint32(text[i+4])<<8|uint32(text[i+5])<<16|uint32(text[i+6])<<24))
			if got < 0 || got+4 > len(text) || !bytes.Equal(text[got:got+4], []byte{0xf3, 0x0f, 0x1e, 0xfa}) {
				continue
			}
			if target < 0 {
				target = got
			} else if got != target {
				t.Fatalf("callback LEA targets differ: %d and %d", target, got)
			}
			leas++
		}
	}
	if leas != 2 {
		t.Fatalf("callback LEA count = %d, want 2", leas)
	}
	checkRuntimeRegs("kernel callback", target)
}

func TestKernelModuleLicensePolicy(t *testing.T) {
	savedSymvers := renvoKernelSymvers
	savedLicense := renvoKernelLicense
	defer func() {
		renvoKernelSymvers = savedSymvers
		renvoKernelLicense = savedLicense
	}()
	renvoKernelSymvers = []byte("0x11223344\tktime_get_ns\tvmlinux\tEXPORT_SYMBOL_GPL\n0x55667788\tktime_get\tvmlinux\tEXPORT_SYMBOL\n")
	if !renvoKernelSymbolGPLOnly("ktime_get_ns") {
		t.Fatal("GPL-only symbol was not detected")
	}
	if renvoKernelSymbolGPLOnly("ktime_get") {
		t.Fatal("ordinary export was classified as GPL-only")
	}
	for _, license := range []string{"GPL", "GPL v2", "GPL and additional rights", "Dual BSD/GPL", "Dual MIT/GPL", "Dual MPL/GPL"} {
		renvoKernelLicense = license
		if !renvoKernelLicenseGPLCompatible() {
			t.Errorf("license %q should be GPL-compatible", license)
		}
	}
	for _, license := range []string{"", "Proprietary", "MIT", "BSD"} {
		renvoKernelLicense = license
		if renvoKernelLicenseGPLCompatible() {
			t.Errorf("license %q should not be GPL-compatible", license)
		}
	}
}

func TestKernelModuleWithoutExitOmitsCleanup(t *testing.T) {
	savedFixed := renvoFixedTarget
	savedRelease := renvoKernelRelease
	savedName := renvoKernelModuleName
	savedSymvers := renvoKernelSymvers
	savedVersion := renvoKernelVersion
	savedSize := renvoKernelModuleSize
	savedNameOff := renvoKernelModuleNameOff
	savedInitOff := renvoKernelModuleInitOff
	savedExitOff := renvoKernelModuleExitOff
	defer func() {
		renvoFixedTarget = savedFixed
		renvoKernelRelease = savedRelease
		renvoKernelModuleName = savedName
		renvoKernelSymvers = savedSymvers
		renvoKernelVersion = savedVersion
		renvoKernelModuleSize = savedSize
		renvoKernelModuleNameOff = savedNameOff
		renvoKernelModuleInitOff = savedInitOff
		renvoKernelModuleExitOff = savedExitOff
	}()
	renvoFixedTarget = renvoTargetLinuxKernelAmd64
	renvoSetTarget(renvoTargetLinuxKernelAmd64)
	renvoKernelRelease = "6.18.0-test"
	renvoKernelModuleName = "init_only"
	renvoKernelSymvers = []byte("0xb1976aeb\tmodule_layout\tvmlinux\tEXPORT_SYMBOL\n0x92997ed8\t_printk\tvmlinux\tEXPORT_SYMBOL\n")
	renvoKernelVersion = "Linux version 6.18.0-test SMP PREEMPT"
	renvoKernelModuleSize = 128
	renvoKernelModuleNameOff = 16
	renvoKernelModuleInitOff = 32
	renvoKernelModuleExitOff = 64

	prog := renvoParseProgram([]byte("package main\nfunc appMain() { print(\"INIT\\n\") }\n"))
	renvoCaptureKernelCompileContext(&prog.c)
	var meta renvoMeta
	renvoBuildMetaInto(&prog, &meta)
	meta.arenaSize = 4096
	result := renvoTryCompileScalarProgramAmd64(&prog, &meta)
	if !result.ok {
		t.Fatal("init-only module compilation failed")
	}
	f, err := elf.NewFile(bytes.NewReader(result.data))
	if err != nil {
		t.Fatal(err)
	}
	symbols, err := f.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	for _, symbol := range symbols {
		if symbol.Name == "cleanup_module" {
			t.Fatal("init-only module exported cleanup_module")
		}
	}
	relocations, err := f.Section(".rela.gnu.linkonce.this_module").Data()
	if err != nil || len(relocations) != 24 {
		t.Fatalf("init-only module relocations = %d bytes, err %v", len(relocations), err)
	}
}
