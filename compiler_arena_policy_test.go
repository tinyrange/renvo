package main

import (
	"bytes"
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

var arenaPolicySmokeSource = []byte(`package main

func appMain() int {
	print("PASS\n")
	return 0
}
`)

func TestArenaDefaultsAreTargetAware(t *testing.T) {
	tests := []struct {
		target string
		want   int
	}{
		{target: "linux/amd64", want: rtgArenaSize64BitHosted},
		{target: "linux/386", want: rtgArenaSize32BitHosted},
		{target: "linux/aarch64", want: rtgArenaSize64BitHosted},
		{target: "linux/arm", want: rtgArenaSize32BitHosted},
		{target: "windows/amd64", want: rtgArenaSize64BitHosted},
		{target: "windows/386", want: rtgArenaSize32BitHosted},
		{target: "windows/arm64", want: rtgArenaSize64BitHosted},
		{target: "darwin/arm64", want: rtgArenaSize64BitHosted},
		{target: "wasi/wasm32", want: rtgArenaSizeWasi},
	}
	for _, test := range tests {
		got, ok := RtgDefaultArenaSize(test.target)
		if !ok || got != test.want {
			t.Errorf("RtgDefaultArenaSize(%q) = (%d, %v), want (%d, true)", test.target, got, ok, test.want)
		}
	}
	if _, ok := RtgDefaultArenaSize("linux/mips"); ok {
		t.Fatal("unsupported target reported an arena default")
	}
}

func TestArenaOptionsRejectImpossibleSizesBeforeCompilation(t *testing.T) {
	for _, size := range []int{-1, 1, 255, rtgArenaSizeMaximum + 1} {
		if output, ok := RtgCompileSourceToBytesWithOptions(arenaPolicySmokeSource, "linux/amd64", RtgCompileOptions{ArenaSize: size}); ok || len(output) != 0 {
			t.Fatalf("arena size %d produced an artifact", size)
		}
	}
}

func TestDefaultHostedImagesStayBelowVirtualMemoryLimit(t *testing.T) {
	const limit = 800 * 1024 * 1024
	for _, target := range []string{"linux/amd64", "linux/386", "linux/aarch64", "linux/arm"} {
		image, ok := RtgCompileSourceToBytesWithOptions(arenaPolicySmokeSource, target, RtgCompileOptions{StripSymbols: true})
		if !ok {
			t.Fatalf("compile %s", target)
		}
		file, err := elf.NewFile(bytes.NewReader(image))
		if err != nil {
			t.Fatalf("parse %s ELF: %v", target, err)
		}
		var maxMemory uint64
		for _, program := range file.Progs {
			if program.Type == elf.PT_LOAD && program.Memsz > maxMemory {
				maxMemory = program.Memsz
			}
		}
		file.Close()
		if maxMemory >= limit {
			t.Errorf("%s load reservation = %d, want less than %d", target, maxMemory, limit)
		}
	}
}

func TestLinuxHelloStartsUnderSub800MiBAddressLimit(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("address-space launch probe requires linux/amd64")
	}
	image, ok := RtgCompileSourceToBytesWithOptions(arenaPolicySmokeSource, "linux/amd64", RtgCompileOptions{StripSymbols: true})
	if !ok {
		t.Fatal("compile linux/amd64")
	}
	path := filepath.Join(t.TempDir(), "arena-smoke")
	if err := os.WriteFile(path, image, 0755); err != nil {
		t.Fatalf("write smoke image: %v", err)
	}
	command := exec.Command("sh", "-c", "ulimit -v 524288; exec \"$1\"", "sh", path)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("launch under 512 MiB address limit: %v\n%s", err, output)
	}
	if string(output) != "PASS\n" {
		t.Fatalf("limited launch output = %q, want PASS", output)
	}
}

func TestWindowsAndDarwinImagesReportBoundedArenaCommitment(t *testing.T) {
	const limit = 800 * 1024 * 1024
	for _, target := range []string{"windows/amd64", "windows/386", "windows/arm64"} {
		image, ok := RtgCompileSourceToBytesWithOptions(arenaPolicySmokeSource, target, RtgCompileOptions{StripSymbols: true})
		if !ok {
			t.Fatalf("compile %s", target)
		}
		file, err := pe.NewFile(bytes.NewReader(image))
		if err != nil {
			t.Fatalf("parse %s PE: %v", target, err)
		}
		var imageSize uint32
		switch header := file.OptionalHeader.(type) {
		case *pe.OptionalHeader32:
			imageSize = header.SizeOfImage
		case *pe.OptionalHeader64:
			imageSize = header.SizeOfImage
		default:
			t.Fatalf("%s optional header = %T", target, file.OptionalHeader)
		}
		file.Close()
		if uint64(imageSize) >= limit {
			t.Errorf("%s image reservation = %d, want less than %d", target, imageSize, limit)
		}
	}

	image, ok := RtgCompileSourceToBytesWithOptions(arenaPolicySmokeSource, "darwin/arm64", RtgCompileOptions{StripSymbols: true})
	if !ok {
		t.Fatal("compile darwin/arm64")
	}
	file, err := macho.NewFile(bytes.NewReader(image))
	if err != nil {
		t.Fatalf("parse Darwin image: %v", err)
	}
	dataMemory := uint64(0)
	for _, load := range file.Loads {
		segment, segmentOK := load.(*macho.Segment)
		if segmentOK && segment.Name == "__DATA" {
			dataMemory = segment.Memsz
		}
	}
	file.Close()
	if dataMemory == 0 || dataMemory >= limit {
		t.Fatalf("Darwin data reservation = %d, want 1..%d", dataMemory, limit-1)
	}
}

func TestExplicitArenaSizeControlsWasiLinearMemory(t *testing.T) {
	image, ok := RtgCompileSourceToBytesWithOptions(arenaPolicySmokeSource, "wasi/wasm32", RtgCompileOptions{ArenaSize: 65536})
	if !ok {
		t.Fatal("compile wasi/wasm32")
	}
	pages, ok := wasmInitialMemoryPages(image)
	if !ok {
		t.Fatal("WASI image has no decodable memory section")
	}
	if pages < 128 || pages > 256 {
		t.Fatalf("WASI initial memory = %d pages, want a bounded arena-backed image", pages)
	}
}

func wasmInitialMemoryPages(image []byte) (int, bool) {
	if len(image) < 8 {
		return 0, false
	}
	position := 8
	for position < len(image) {
		sectionID := int(image[position])
		position++
		sectionSize, next, ok := readWasmUnsigned(image, position)
		if !ok || next+sectionSize > len(image) {
			return 0, false
		}
		if sectionID == 5 {
			count, at, countOK := readWasmUnsigned(image, next)
			if !countOK || count != 1 || at >= next+sectionSize || image[at] != 0 {
				return 0, false
			}
			pages, _, pagesOK := readWasmUnsigned(image, at+1)
			return pages, pagesOK
		}
		position = next + sectionSize
	}
	return 0, false
}

func readWasmUnsigned(data []byte, position int) (int, int, bool) {
	value := 0
	shift := 0
	for position < len(data) && shift <= 28 {
		current := data[position]
		position++
		value |= int(current&0x7f) << shift
		if current < 0x80 {
			return value, position, true
		}
		shift += 7
	}
	return 0, position, false
}
