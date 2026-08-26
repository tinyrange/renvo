package backendjit

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/driver"
)

func TestMSDOSCOMExample(t *testing.T) {
	if hostTarget() == "" {
		t.Skipf("no in-process prepared backend for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	definition := filepath.Join(root, "backends", "msdos.rtg")
	backend := New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"),
		t.TempDir(), backendcompiled.Backend{})
	result := driver.CompileFromFS([]string{
		"-backend", definition,
		"-t", "msdos/8086",
		"-s",
		"-o", "hello.com",
		filepath.Join(root, "examples", "msdos"),
	}, root, filepath.Join(root, "std"), driver.OSFS{},
		backend)
	if !result.Ok {
		t.Fatalf("MS-DOS custom backend compile failed: %#v", result.Diagnostic)
	}
	image := result.Binary
	if len(image) == 0 || len(image) > 0xff00 {
		t.Fatalf("COM image size = %d, want 1..65280", len(image))
	}
	if len(image) >= 2 && image[0] == 'M' && image[1] == 'Z' {
		t.Fatal("COM image unexpectedly has an MZ header")
	}
	if !bytes.Contains(image, []byte("Hello from Renvo on MS-DOS!\r\n")) {
		t.Fatal("COM image does not contain the program data")
	}
	if bytes.Count(image, []byte{0xcd, 0x21}) < 2 {
		t.Fatal("COM image lacks DOS write and exit interrupts")
	}

	raw := backend.CompileSourceWithArena([]byte(`package main
func appMain() int { print("RAW PASS\n"); return 0 }
`), "msdos/8086", true, 4096)
	if !raw.Ok {
		t.Fatalf("raw backend source compile failed: %#v", raw.Diagnostic)
	}
	if !bytes.Contains(raw.Binary, []byte("RAW PASS\n")) {
		t.Fatal("raw backend source output does not contain program data")
	}

	cRoot := t.TempDir()
	cPath := filepath.Join(cRoot, "main.c")
	if err := os.WriteFile(cPath, []byte(`#include <stdio.h>
int main(void) {
	puts("C PASS");
	return 0;
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	cArgs := driver.NormalizeCCompilerCommand([]string{
		"renvo", "cc", "-backend", definition, "-t", "msdos/8086",
		"-s", "-o", "hello.com", cPath,
	})
	cResult := driver.CompileFromFS(cArgs[1:], root, filepath.Join(root, "std"), driver.OSFS{}, backend)
	if !cResult.Ok {
		t.Fatalf("MS-DOS C compile failed: %#v", cResult.Diagnostic)
	}
	if !bytes.Contains(cResult.Binary, []byte("C PASS")) {
		t.Fatal("C COM image does not contain program data")
	}
}

func TestMSDOSMZTargetAndDeviceLibrary(t *testing.T) {
	if hostTarget() == "" {
		t.Skipf("no in-process prepared backend for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	definition := filepath.Join(root, "backends", "msdos.rtg")
	backend := New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"),
		t.TempDir(), backendcompiled.Backend{})
	project := t.TempDir()
	module := "module example.com/dosapp\nrequire renvo.dev v0.0.0\nreplace renvo.dev => " + root + "\n"
	if err := os.WriteFile(filepath.Join(project, "go.mod"), []byte(module), 0o644); err != nil {
		t.Fatal(err)
	}
	source := `package main
import "renvo.dev/device/dos"
func main() {
	dos.SetVideoMode(dos.ModeText80x25)
	screen := dos.OpenVGA13()
	screen.Clear(1)
	screen.Blit(0, []byte{1})
	segment := dos.Segment(0xa000)
	segment.Store8(0, 1)
	_ = segment.Load8(0)
	_ = dos.Now()
	dos.SpeakerOff()
}
`
	if err := os.WriteFile(filepath.Join(project, "main.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	result := driver.CompileFromFS([]string{
		"-backend", definition, "-t", "msdos/8086-mz", "-arena-size", "4096", "-s", "-o", "app.exe", ".",
	}, project, filepath.Join(root, "std"), driver.OSFS{}, backend)
	if !result.Ok {
		t.Fatalf("MS-DOS MZ compile failed: %#v", result.Diagnostic)
	}
	image := result.Binary
	if len(image) < 32 || string(image[:2]) != "MZ" {
		t.Fatalf("MZ image header = %x", image[:min(len(image), 32)])
	}
	word := func(at int) int { return int(image[at]) | int(image[at+1])<<8 }
	if word(8) != 2 || word(20) != 0x100 || word(22) != 0 || word(24) != 0x1c {
		t.Fatalf("MZ execution fields: header=%d IP=%#x CS=%#x reloc=%#x", word(8), word(20), word(22), word(24))
	}
	if word(2) != len(image)&511 || word(4) != (len(image)+511)/512 || word(6) != 0 {
		t.Fatalf("MZ file accounting: size=%d last=%d pages=%d relocs=%d", len(image), word(2), word(4), word(6))
	}
	moduleSize := len(image) - 32
	if word(16) <= moduleSize || word(16) > moduleSize+word(10)*16 {
		t.Fatalf("MZ stack/minalloc: module=%d SP=%#x minalloc=%#x", moduleSize, word(16), word(10))
	}
	if !bytes.Contains(image, []byte{0xcd, 0x10}) || !bytes.Contains(image, []byte{0xcd, 0x21}) || !bytes.Contains(image, []byte{0xee}) {
		t.Fatal("MZ image omits device/dos RTGASM interrupt or port-I/O fragments")
	}
	segmentFill := []byte{0x06, 0x89, 0xc6, 0x8e, 0xc3, 0x89, 0xcf, 0x89, 0xd0, 0x89, 0xf1, 0xfc, 0xf3, 0xaa, 0x07, 0xc3}
	if !bytes.Contains(image, segmentFill) {
		t.Fatal("MZ image omits reverse-call-word segment fill fragment")
	}
	segmentWrite := []byte{0x06, 0x8e, 0xc7, 0x89, 0xdf, 0x89, 0xc6, 0x89, 0xd1, 0xfc, 0xf3, 0xa4, 0x07, 0xc3}
	if !bytes.Contains(image, segmentWrite) {
		t.Fatal("MZ image omits reverse-call-word segment write fragment")
	}
	segmentLoad := []byte{0x1e, 0x8e, 0xda, 0x89, 0xc3, 0x31, 0xc0, 0x8a, 0x07, 0x1f, 0xc3}
	segmentStore := []byte{0x06, 0x8e, 0xc1, 0x89, 0xd3, 0x26, 0x88, 0x07, 0x07, 0xc3}
	if !bytes.Contains(image, segmentLoad) || !bytes.Contains(image, segmentStore) {
		t.Fatal("MZ image omits reverse-call-word segment load/store fragments")
	}
	oldPortOut := []byte{0x89, 0xd3, 0x89, 0xc2, 0x89, 0xd8, 0xee, 0xc3}
	if bytes.Contains(image, oldPortOut) {
		t.Fatal("MZ image retains source-order port output fragment")
	}
}
