//go:build !renvo

package runimage_test

import (
	"fmt"
	"renvo.dev/internal/backendcompiled"
	emu "renvo.dev/internal/rfe/runtime"
	"renvo.dev/internal/runimage"
	"runtime"
	"testing"
	"time"
	"unsafe"
)

func TestPreparedBatchBorrowCleanupAndCloseSerialization(t *testing.T) {
	a, err := runimage.NewCodeArena(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	code, ok := backendcompiled.RenvoEmitLinkedDispatcher(1, 2, runtime.GOARCH == "arm64")
	if !ok {
		t.Fatal("dispatcher emission")
	}
	entry, err := a.InstallLinkedDispatcher(code, 2)
	if err != nil {
		t.Fatal(err)
	}
	call, err := a.PrepareLinkedCall(entry, 2)
	if err != nil {
		t.Fatal(err)
	}
	m := new(emu.MemoryContext)
	view := [4]uint64{1, 2, 3, 4}
	for _, limit := range []int{0, 17} {
		if _, err := call.CallBatch([]uint64{0, 0}, unsafe.Pointer(m), &view, limit, func(int) (bool, error) { t.Fatal("invalid limit called selector"); return false, nil }); err == nil || view != [4]uint64{} {
			t.Fatal("invalid batch retained view")
		}
	}
	if _, err := call.CallBatch([]uint64{0}, unsafe.Pointer(m), &view, 1, func(int) (bool, error) { t.Fatal("invalid shape called selector"); return false, nil }); err == nil || view != [4]uint64{} {
		t.Fatal("shape bypass")
	}
	stop := fmt.Errorf("stop")
	if done, err := call.CallBatch([]uint64{0, 0}, unsafe.Pointer(m), &view, 1, func(int) (bool, error) {
		if view[0] == 0 {
			t.Fatal("missing serialized view")
		}
		return false, stop
	}); done != 0 || err != stop || view != [4]uint64{} {
		t.Fatal("selector error borrow", done, err, view)
	}
	entered, release, closeStarted := make(chan struct{}), make(chan struct{}), make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	finished, closed := make(chan error, 1), make(chan error, 1)
	go func() {
		_, err := call.CallBatch([]uint64{0, 0}, unsafe.Pointer(m), &view, 1, func(int) (bool, error) {
			close(entered)
			<-release
			return false, nil
		})
		finished <- err
	}()
	<-entered
	go func() { close(closeStarted); closed <- a.Close() }()
	<-closeStarted
	select {
	case err := <-closed:
		t.Fatal("Close escaped serialized prepared selector", err)
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	if err := <-finished; err != nil || view != [4]uint64{} {
		t.Fatal("borrow cleanup before unlock", err, view)
	}
	if err := <-closed; err != nil {
		t.Fatal(err)
	}
	if _, err := call.CallBatch([]uint64{0, 0}, unsafe.Pointer(m), &view, 1, func(int) (bool, error) { t.Fatal("closed handle called selector"); return false, nil }); err == nil || view != [4]uint64{} {
		t.Fatal("closed handle borrow")
	}
}
