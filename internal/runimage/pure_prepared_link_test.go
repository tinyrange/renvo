//go:build !renvo

package runimage_test

import (
	"renvo.dev/internal/backendcompiled"
	emu "renvo.dev/internal/rfe/runtime"
	"renvo.dev/internal/runimage"
	"runtime"
	"sync"
	"testing"
	"unsafe"
)

func TestPreparedLinkedCallLifecycleGrowthAndSerialization(t *testing.T) {
	a, err := runimage.NewCodeArena(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	dispatch, ok := backendcompiled.RenvoEmitLinkedDispatcher(1, 2, runtime.GOARCH == "arm64")
	if !ok {
		t.Fatal("dispatcher emission")
	}
	d, err := a.InstallLinkedDispatcher(dispatch, 2)
	if err != nil {
		t.Fatal(err)
	}
	records := []int{1, 0, 0, 0, 0, 0, 0, 1, 3, 0, 1, 0, 2, 2, 0, 0, 0, 0, 0, 0, 2, 4, 0, 1}
	code, ok := backendcompiled.RenvoEmitPureBlock(records, 2, runtime.GOARCH == "arm64")
	if !ok {
		t.Fatal("leaf emission")
	}
	leaf, err := a.InstallBlock(code, 2)
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range [][2]int{{-1, 2}, {d + 1, 2}, {leaf, 2}, {d, 1}, {d, 0}, {d, 257}} {
		if _, err := a.PrepareLinkedCall(bad[0], bad[1]); err == nil {
			t.Fatal("invalid prepared entry", bad)
		}
	}
	call, err := a.PrepareLinkedCall(d, 2)
	if err != nil {
		t.Fatal(err)
	}
	m := new(emu.MemoryContext)
	view := [4]uint64{1, 2, 3, 4}
	for _, state := range [][]uint64{nil, {1}, {1, 2, 3}} {
		if call.Call(state, unsafe.Pointer(m), &view) == nil || view != [4]uint64{} {
			t.Fatal("invalid shape retained view")
		}
	}
	if call.Call([]uint64{0, 0}, nil, &view) == nil || view != [4]uint64{} || call.Call([]uint64{0, 0}, unsafe.Pointer(m), nil) == nil {
		t.Fatal("invalid borrow")
	}
	var zero runimage.LinkedCall
	if zero.Call([]uint64{0, 0}, unsafe.Pointer(m), &view) == nil || view != [4]uint64{} {
		t.Fatal("zero handle")
	}
	// Grow both code and the metadata slice after preparing the dispatcher.
	for i := 0; i < 128; i++ {
		leaf, err = a.InstallBlock(code, 2)
		if err != nil {
			t.Fatal(err)
		}
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			state := []uint64{0, 0}
			context := new(emu.MemoryContext)
			context.PublishLink(0, leaf, 1, [17]uint8{})
			for j := 0; j < 8; j++ {
				context.Remaining, context.Total, context.MemoryTotal = 64, 0, 0
				if err := call.Call(state, unsafe.Pointer(context), &context.CodeView); err != nil || context.CodeView != [4]uint64{} || context.Total != 64 || context.Remaining != 0 {
					t.Errorf("serialized prepared call: %v, %d/%d", err, context.Total, context.Remaining)
					return
				}
			}
			if state[0] != 512 {
				t.Errorf("private stack/state lost: %v", state)
			}
		}()
	}
	wg.Wait()
	if err = a.Close(); err != nil {
		t.Fatal(err)
	}
	if call.Call([]uint64{0, 0}, unsafe.Pointer(m), &view) == nil || view != [4]uint64{} {
		t.Fatal("closed prepared handle")
	}
	if _, err = a.PrepareLinkedCall(d, 2); err == nil {
		t.Fatal("closed preparation")
	}
}
