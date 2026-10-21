//go:build !renvo && (amd64 || arm64)

package runimage

import (
	"renvo.dev/internal/rfeabi"
	"runtime"
	"testing"
)

func TestPhysicalPinLeaseSurvivesRootViewsAndNewWatchMasks(t *testing.T) {
	a, err := NewCodeArena(4096)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	data := make([]byte, 8*4096)
	clock := uint64(1)
	w := rfeabi.NewWriteWatches(len(data))
	one, two := rfeabi.NewOwnedMemory(data, &clock, w), rfeabi.NewOwnedMemory(data, &clock, w)
	w.Mark(17, 1, 1)
	a.pinBackingLocked(one)
	lease := a.backingPins
	for p := 1; p < 8; p++ {
		w.Mark(uint64(p*4096+7), 1, 2)
		a.pinBackingLocked(two)
		runtime.GC()
		a.pinBackingLocked(one)
		if a.backingPins != lease || lease.count != p+1 {
			t.Fatal("root switch rebuilt physical pins or missed new mask", p)
		}
	}
	if err = a.Close(); err != nil || a.backingPins != nil {
		t.Fatal("closed arena retained RAM pins", err)
	}
}
