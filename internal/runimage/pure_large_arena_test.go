//go:build !renvo

package runimage_test

import (
	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/runimage"
	"runtime"
	"testing"
)

// Exercise actual installation and calls above the former 64 MiB boundary:
// accepting the new capacity alone would not detect truncated entry metadata.
func TestLargeArenaHighOffsetExecutionAndAdmission(t *testing.T) {
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		t.Skip("native host required")
	}
	if arena, err := runimage.NewCodeArena((128 << 20) + 1); err == nil {
		arena.Close()
		t.Fatal("accepted cache beyond supported maximum")
	}
	arena, err := runimage.NewCodeArena(128 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer arena.Close()
	// Installed padding is never executed. It advances the next real entry
	// through the public installation API, without forging arena internals.
	padding := make([]byte, (64<<20)+16)
	if _, err := arena.InstallBlock(padding, 2); err != nil {
		t.Fatal(err)
	}
	records := []int{0, 0, 0, 7, 2, 0, 0, 0, 0, 0, 0, 4, 2, 2, 0, 1}
	code, body, ok := backendcompiled.RenvoEmitSharedBlock(records, 2, false, runtime.GOARCH == "arm64")
	if !ok {
		t.Fatal("shared block emission")
	}
	entry, err := arena.InstallSharedBlock(code, 2, false, body)
	if err != nil || entry <= 64<<20 {
		t.Fatal("high-offset installation", entry, err)
	}
	state := []uint64{99, 98}
	if err := arena.Call(entry, state); err != nil || state[0] != 7 || state[1] != 4 {
		t.Fatal("high-offset execution", state, err)
	}
	if arena.Call(entry+16, state) == nil {
		t.Fatal("interior offset became a callable entry")
	}
	if arena.Call(entry, state[:1]) == nil {
		t.Fatal("high-offset state shape was not checked")
	}
	if err := arena.Close(); err != nil {
		t.Fatal(err)
	}
	if arena.Call(entry, state) == nil {
		t.Fatal("closed high-offset entry remained callable")
	}
}
