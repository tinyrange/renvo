package backendjit

import (
	"os"
	"path/filepath"
	"renvo.dev/internal/backendcompiled"
	"testing"
)

// Native corpus execution checks semantics; this also exercises the prepared
// RV32 path, where interface float comparison formerly reached an x87 helper.
func TestRV32SDMMCRegressions(t *testing.T) {
	if hostTarget() == "" {
		t.Skip("no native prepared backend")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	backend := New(filepath.Join(root, "backends/esp32p4.rtg"), filepath.Join(root, "backend"), filepath.Join(root, "std"), backendJITTestCacheDir, backendcompiled.Backend{})
	for _, name := range []string{"returned_slice_dma_alias", "prepared_fixed_interface_equal", "indexed_uint16_sentinel"} {
		t.Run(name, func(t *testing.T) {
			source, err := os.ReadFile(filepath.Join(root, "backend/tests", name+".go"))
			if err != nil {
				t.Fatal(err)
			}
			result := backend.CompileSourceWithArena(source, "esp32p4/riscv32", true, 0)
			if !result.Ok {
				t.Fatalf("prepared RV32: %#v", result.Diagnostic)
			}
		})
	}
}
