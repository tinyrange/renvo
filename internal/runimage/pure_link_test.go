//go:build !renvo

package runimage_test

import (
	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/runimage"
	"runtime"
	"testing"
	"unsafe"
)

func TestLinkedDispatcherWrapperRejectsAndClearsView(t *testing.T) {
	arena, err := runimage.NewCodeArena(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer arena.Close()
	code, ok := backendcompiled.RenvoEmitLinkedDispatcher(1, 2, runtime.GOARCH == "arm64")
	if !ok {
		t.Fatal("dispatcher emission")
	}
	dispatcher, err := arena.InstallLinkedDispatcher(code, 2)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := arena.InstallBlock(code, 2)
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := arena.Install(code)
	if err != nil {
		t.Fatal(err)
	}
	// The empty link table exits before touching a leaf or guest memory.
	context := make([]uint64, 8464)
	pointer := unsafe.Pointer(&context[0])
	state := []uint64{7, 0}
	view := [4]uint64{1, 2, 3, 4}
	for _, entry := range []int{-1, dispatcher + 1, dispatcher + 16, leaf, legacy, 1 << 20} {
		if arena.CallLinked(entry, state, pointer, &view) == nil || view != [4]uint64{} {
			t.Fatal("bad dispatcher admitted/view retained", entry, view)
		}
		view = [4]uint64{1, 2, 3, 4}
	}
	if arena.CallLinked(dispatcher, state, nil, &view) == nil || view != [4]uint64{} {
		t.Fatal("nil context admitted")
	}
	if arena.CallLinked(dispatcher, state, pointer, nil) == nil {
		t.Fatal("nil view admitted")
	}
	view = [4]uint64{1, 2, 3, 4}
	if arena.CallLinked(dispatcher, state[:1], pointer, &view) == nil || view != [4]uint64{} {
		t.Fatal("wrong shape admitted")
	}
	// A valid call after all rejection paths proves cleanup also released the lock.
	if err = arena.CallLinked(dispatcher, state, pointer, &view); err != nil || view != [4]uint64{} || state[0] != 7 {
		t.Fatal("reentry/cleanup", err, view, state)
	}
	if arena.AdmitLinkedBlock(legacy, 2, 1) == nil || arena.AdmitLinkedBlock(dispatcher, 2, 1) == nil {
		t.Fatal("nonleaf admission accepted")
	}
	if err = arena.Close(); err != nil {
		t.Fatal(err)
	}
	view = [4]uint64{1, 2, 3, 4}
	if arena.CallLinked(dispatcher, state, pointer, &view) == nil || view != [4]uint64{} {
		t.Fatal("closed dispatcher/view")
	}
}
