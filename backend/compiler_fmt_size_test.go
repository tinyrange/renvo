package main

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"os"
	"testing"

	"renvo.dev/internal/driver"
	"renvo.dev/internal/load"
)

const fmtHelloMaxLinuxAmd64StrippedSize = 2 * 1024

func TestFmtPrintlnHelloBinarySize(t *testing.T) {
	fmtSource, err := os.ReadFile("../std/fmt/fmt_renvo.go")
	if err != nil {
		t.Fatal(err)
	}
	files := []load.SourceFile{
		{Path: "/repo/hello/go.mod", Src: []byte("module example.com/hello\n")},
		{Path: "/repo/hello/main.go", Src: []byte("package main\nimport \"fmt\"\nfunc main() { fmt.Println(\"hello\") }\n")},
		{Path: "/std/fmt/fmt_renvo.go", Src: fmtSource},
	}
	targets := []string{
		"linux/amd64",
		"linux/386",
		"linux/aarch64",
		"linux/arm",
		"windows/amd64",
		"windows/386",
		"windows/arm64",
		"darwin/arm64",
		"wasi/wasm32",
	}
	for _, target := range targets {
		image := compileFmtHelloUnit(t, target, true, files)
		t.Logf("%s stripped total: %d bytes", target, len(image))
		if target == "linux/amd64" && len(image) > fmtHelloMaxLinuxAmd64StrippedSize {
			t.Fatalf("linux/amd64 stripped fmt.Println hello binary = %d bytes, want <= %d", len(image), fmtHelloMaxLinuxAmd64StrippedSize)
		}
	}

	image := compileFmtHelloUnit(t, "linux/amd64", false, files)
	file, err := elf.NewFile(bytes.NewReader(image))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	text := file.Section(".text")
	rodata := file.Section(".rodata")
	if text == nil || rodata == nil {
		t.Fatalf("unstripped hello image lacks text/rodata sections")
	}
	headerSize := int(binary.LittleEndian.Uint16(image[52:54])) + int(binary.LittleEndian.Uint16(image[54:56]))*int(binary.LittleEndian.Uint16(image[56:58]))
	stripped := compileFmtHelloUnit(t, "linux/amd64", true, files)
	t.Logf("linux/amd64 components: headers=%d text=%d rodata=%d stripped-total=%d", headerSize, text.Size, rodata.Size, len(stripped))
	if headerSize+int(text.Size)+int(rodata.Size) != len(stripped) {
		t.Fatalf("stripped size components = %d + %d + %d, total = %d", headerSize, text.Size, rodata.Size, len(stripped))
	}
}

func compileFmtHelloUnit(t *testing.T, target string, strip bool, files []load.SourceFile) []byte {
	t.Helper()
	built := driver.BuildUnit([]string{"-t", target, "-o", "hello", "."}, "/repo/hello", "/std", files)
	if !built.Ok {
		t.Fatalf("build %s fmt hello unit: error=%d diagnostic=%#v", target, built.Error, built.Diagnostic)
	}
	targetID := renvoParseTargetArg(target)
	if targetID == 0 {
		t.Fatalf("unknown target %q", target)
	}
	oldStrip := renvoCompilerStripSymbols
	renvoSetStripSymbols(strip)
	defer renvoSetStripSymbols(oldStrip)
	renvoSetTarget(targetID)
	program, isUnit, ok := renvoDecodeUnitProgram(built.Unit)
	if !isUnit || !ok {
		t.Fatalf("decode %s fmt hello unit", target)
	}
	result := renvoCompileParsedProgram(&program, targetID)
	if !result.ok {
		t.Fatalf("compile %s fmt hello unit", target)
	}
	return result.data
}
