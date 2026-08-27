package backendjit

import (
	"encoding/binary"
	"path/filepath"
	"runtime"
	"testing"

	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/driver"
)

func TestUEFIHelloImage(t *testing.T) {
	if hostTarget() == "" {
		t.Skipf("no in-process prepared backend for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	definition := filepath.Join(root, "backends", "uefi_amd64.rtg")
	backend := New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"),
		t.TempDir(), backendcompiled.Backend{})
	result := driver.CompileFromFS([]string{
		"-backend", definition, "-t", "uefi/amd64", "-s", "-o", "BOOTX64.EFI",
		filepath.Join(root, "examples", "uefi-hello"),
	}, root, filepath.Join(root, "std"), driver.OSFS{}, backend)
	if !result.Ok {
		t.Fatalf("UEFI custom backend compile failed: %#v", result.Diagnostic)
	}
	image := result.Binary
	if len(image) < 0x200 || string(image[:2]) != "MZ" {
		t.Fatalf("UEFI image header = %x", image[:min(len(image), 32)])
	}
	pe := int(binary.LittleEndian.Uint32(image[0x3c:]))
	if string(image[pe:pe+4]) != "PE\x00\x00" || binary.LittleEndian.Uint16(image[pe+4:]) != 0x8664 {
		t.Fatalf("UEFI PE signature or machine is invalid")
	}
	optional := pe + 24
	if binary.LittleEndian.Uint16(image[optional:]) != 0x20b ||
		binary.LittleEndian.Uint16(image[optional+68:]) != 10 {
		t.Fatalf("UEFI optional header is not PE32+ EFI_APPLICATION")
	}
}
