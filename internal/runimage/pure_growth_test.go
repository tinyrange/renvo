//go:build !renvo

package runimage_test

import (
	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/runimage"
	"runtime"
	"testing"
)

func TestGrowableArenaKeepsLiveEntriesAndRevokesReleasedOnes(t *testing.T) {
	a, err := runimage.NewCodeArena(4096)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.AllowGrowth()
	// Add one to state[0], through the same compiler used for all native code.
	code, ok := backendcompiled.RenvoEmitPureBlock([]int{1, 0, 0, 0, 0, 0, 0, 1, 3, 0, 1, 0, 2, 2, 0, 0}, 1, runtime.GOARCH == "arm64")
	if !ok {
		t.Fatal("fixture emission")
	}
	padded := make([]byte, 8192)
	copy(padded, code)
	first, err := a.InstallBlock(padded, 1)
	if err != nil {
		t.Fatal(err)
	}
	retired, err := a.InstallBlock(padded, 1)
	if err != nil {
		t.Fatal(err)
	}
	if a.Release(retired) != len(padded) {
		t.Fatal("retirement did not release fragment")
	}
	for i := 0; i < 12; i++ {
		if _, err = a.InstallBlock(padded, 1); err != nil {
			t.Fatal(err)
		}
	}
	state := []uint64{9}
	if err = a.Call(first, state); err != nil || state[0] != 10 {
		t.Fatal("live code lost after growth", state, err)
	}
	if err = a.Call(retired, state); err == nil {
		t.Fatal("retired offset resurrected")
	}
	if err = a.AdmitLinkedBlock(retired, 1, 1); err == nil {
		t.Fatal("retired code reacquired admission")
	}
}
