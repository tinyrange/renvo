//go:build !renvo

package runimage

import (
	"bytes"
	"fmt"
	"runtime"
	"testing"
)

// Profiling may be enabled after installation. Exact extents must survive
// alignment gaps, and neither gaps nor aligned interior addresses are entries.
func TestSymbolExtentsAfterLateEnable(t *testing.T) {
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		t.Skip("native host unavailable")
	}
	a, err := NewCodeArena(16384)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	lengths := []int{4097, 19, 512}
	entries := make([]int, len(lengths))
	for i, length := range lengths {
		entries[i], err = a.InstallBlock(make([]byte, length), 1)
		if err != nil {
			t.Fatal(err)
		}
	}
	var symbols bytes.Buffer
	a.SetSymbolWriter(&symbols)
	for i, entry := range entries {
		if err = a.NameEntry(entry, fmt.Sprintf("block%d", i)); err != nil {
			t.Fatal(err)
		}
	}
	for _, invalid := range []int{16, 4097, entries[2] + 16, (a.used + 15) &^ 15} {
		if a.NameEntry(invalid, "not-entry") == nil {
			t.Fatalf("admitted interior/gap %d", invalid)
		}
	}
	for i, entry := range entries {
		var address, length uintptr
		var name string
		if _, err = fmt.Fscanf(&symbols, "%x %x %s\n", &address, &length, &name); err != nil {
			t.Fatal(err)
		}
		if address != a.base+uintptr(entry) || length != uintptr(lengths[i]) || name != fmt.Sprintf("block%d", i) {
			t.Fatalf("incorrect symbol: %x %x %s", address, length, name)
		}
	}
}
