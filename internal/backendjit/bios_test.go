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

func TestBIOSBootImageAndDeviceLibrary(t *testing.T) {
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
		"-backend", definition, "-t", "bios/8086", "-s", "-o", "renvo-bios.img",
		filepath.Join(root, "examples", "bios-smoke"),
	}, root, filepath.Join(root, "std"), driver.OSFS{}, backend)
	if !result.Ok {
		t.Fatalf("BIOS custom backend compile failed: %#v", result.Diagnostic)
	}
	image := result.Binary
	if len(image) < 1024 || len(image)%512 != 0 {
		t.Fatalf("BIOS disk image size = %d", len(image))
	}
	if image[510] != 0x55 || image[511] != 0xaa {
		t.Fatalf("BIOS boot signature = %x", image[510:512])
	}
	if !bytes.Equal(image[:15], []byte{
		0xfa, 0x31, 0xc0, 0x8e, 0xd8, 0x8e, 0xd0, 0xbc, 0x00, 0x7c, 0xfb,
		0x88, 0x16, 0x90, 0x7d,
	}) {
		t.Fatalf("BIOS loader prefix = %x", image[:15])
	}
	sectors := int(image[0x182]) | int(image[0x183])<<8
	if sectors != len(image)/512-1 || sectors <= 0 || sectors > 127 {
		t.Fatalf("BIOS DAP sectors = %d for %d-byte image", sectors, len(image))
	}
	if image[512+0x100] != 0xfa {
		t.Fatalf("BIOS second-stage entry byte = %#x", image[512+0x100])
	}
	for _, instruction := range [][]byte{
		{0xcd, 0x10}, // teletype runtime/device path
		{0xcd, 0x13}, // disk service
		{0xcd, 0x14}, // serial service
		{0xcd, 0x1a}, // timer service
		{0xee},       // QEMU exit/device port output
	} {
		if !bytes.Contains(image[512:], instruction) {
			t.Fatalf("BIOS stage omits firmware instruction %x", instruction)
		}
	}

	oversized := t.TempDir()
	source := filepath.Join(oversized, "main.go")
	if err := os.WriteFile(source, []byte(`package main
var reserve [60000]byte
func main() { reserve[0] = 1 }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	tooLarge := driver.CompileFromFS([]string{
		"-backend", definition, "-t", "bios/8086", "-arena-size", "256", "-s", "-o", "large.img", source,
	}, oversized, filepath.Join(root, "std"), driver.OSFS{}, backend)
	if tooLarge.Ok {
		t.Fatal("oversized BIOS memory image compiled successfully")
	}
}
