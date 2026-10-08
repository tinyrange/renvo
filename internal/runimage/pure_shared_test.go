//go:build !renvo

package runimage_test

import (
	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/runimage"
	"runtime"
	"testing"
)

func TestSharedBlockBodyBoundsAndOpaqueOffsets(t *testing.T) {
	arena, err := runimage.NewCodeArena(16384)
	if err != nil {
		t.Fatal(err)
	}
	defer arena.Close()
	records := []int{0, 0, 0, 7, 2, 0, 0, 0, 0, 0, 0, 4, 2, 2, 0, 1}
	code, body, ok := backendcompiled.RenvoEmitSharedBlock(records, 2, false, runtime.GOARCH == "arm64")
	if !ok {
		t.Fatal("shared emission")
	}
	for _, offset := range []int{-1, 0, len(code), len(code) + 1, 1 << 30} {
		if _, err = arena.InstallSharedBlock(code, 2, false, offset); err == nil {
			t.Fatal("invalid shared body installed", offset)
		}
	}
	for _, words := range []int{0, 257} {
		if _, err = arena.InstallSharedBlock(code, words, false, body); err == nil {
			t.Fatal("invalid state shape installed", words)
		}
	}
	entry, err := arena.InstallSharedBlock(code, 2, false, body)
	if err != nil {
		t.Fatal(err)
	}
	state := []uint64{99, 98}
	// A private body offset is not a publicly callable entry, even when aligned.
	if arena.Call(entry+body, state) == nil || state[0] != 99 || state[1] != 98 {
		t.Fatal("body escaped checked standalone API", state)
	}
	if err = arena.Call(entry, state); err != nil || state[0] != 7 || state[1] != 4 {
		t.Fatal("compatibility entry", err, state)
	}
	if err = arena.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = arena.InstallSharedBlock(code, 2, false, body); err == nil {
		t.Fatal("closed shared installation")
	}
}
