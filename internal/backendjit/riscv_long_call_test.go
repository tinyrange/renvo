package backendjit

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"renvo.dev/internal/backendcompiled"
	"testing"
)

func TestRV32CallsBeyondJALRange(t *testing.T) {
	if hostTarget() == "" {
		t.Skip("no native prepared backend")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(filepath.Join(root, "backend/tests/riscv_long_control_flow.go"))
	if err != nil {
		t.Fatal(err)
	}
	definition := filepath.Join(root, "backends/esp32p4.rtg")
	backend := New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"), backendJITTestCacheDir, backendcompiled.Backend{})
	result := backend.CompileSourceWithArena(source, "esp32p4/riscv32", true, 0)
	if !result.Ok {
		t.Fatalf("long-call fixture: %#v", result.Diagnostic)
	}
	image := result.Binary
	if len(image) < 52+32 {
		t.Fatal("truncated ELF")
	}
	// First PT_LOAD maps code/data at 0x40000000; the final file includes
	// descriptor/section metadata which must not count as executable targets.
	end := int(binary.LittleEndian.Uint32(image[52+16 : 52+20]))
	if end <= 1<<20 || end > len(image) {
		t.Fatalf("fixture does not span JAL range: %d", end)
	}
	longCalls := 0
	for at := 0x100; at+8 <= end; at += 4 {
		hi := binary.LittleEndian.Uint32(image[at : at+4])
		lo := binary.LittleEndian.Uint32(image[at+4 : at+8])
		if hi&0xfff != 0x97 || lo&0xfffff != 0x80e7 {
			continue
		} // AUIPC ra; JALR ra,ra
		delta := int(int32(hi&0xfffff000)) + int(int32(lo)>>20)
		target := at + delta
		if target < 0x100 || target >= end || target&3 != 0 {
			t.Fatalf("call at %#x targets %#x outside code", at, target)
		}
		if delta < -(1<<20) || delta >= 1<<20 {
			longCalls++
		}
	}
	if longCalls == 0 {
		t.Fatal("no full-range call: JAL would silently wrap")
	}
}
