//go:build !renvo && (amd64 || arm64)

package runimage

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestGrowthRepublishesOnlyLiveSymbols(t *testing.T) {
	a, err := NewCodeArena(4096)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.AllowGrowth()
	var out bytes.Buffer
	a.SetSymbolWriter(&out)
	first, err := a.InstallBlock([]byte{0xc3}, 1)
	if err != nil {
		t.Fatal(err)
	}
	dead, err := a.InstallBlock([]byte{0xc3}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err = a.NameEntry(first, "live"); err != nil {
		t.Fatal(err)
	}
	if err = a.NameEntry(dead, "retired"); err != nil {
		t.Fatal(err)
	}
	a.Release(dead)
	out.Reset()
	old := a.base
	if _, err = a.InstallBlock(make([]byte, a.size), 1); err != nil {
		t.Fatal(err)
	}
	if a.base == old {
		t.Fatal("test did not relocate")
	}
	var address, size uintptr
	var name string
	if _, err = fmt.Fscanf(&out, "%x %x %s", &address, &size, &name); err != nil || address != a.base+uintptr(first) || name != "live" || size != 1 || strings.Contains(out.String(), "retired") {
		t.Fatal("relocation symbols stale", address, name, err)
	}
}
func TestReleaseSharedPagesKeepsLiveNeighborThenDiscardsAllDead(t *testing.T) {
	page := os.Getpagesize()
	a, err := NewCodeArena(page * 2)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	first, err := a.InstallBlock(bytes.Repeat([]byte{0x55}, page/2), 1)
	if err != nil {
		t.Fatal(err)
	}
	second, err := a.InstallBlock(bytes.Repeat([]byte{0x66}, page/2), 1)
	if err != nil {
		t.Fatal(err)
	}
	a.Release(first)
	if a.pageLive[0] != 1 {
		t.Fatal("live shared page released")
	}
	if a.Release(second) != page/2 || a.pageLive[0] != 0 {
		t.Fatal("dead fragments did not release shared page")
	}
	// Appending within a discarded tail page must publish normally again.
	if _, err = a.InstallBlock([]byte{0xc3}, 1); err != nil {
		t.Fatal(err)
	}
}
