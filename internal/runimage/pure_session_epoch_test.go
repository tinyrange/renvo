//go:build !renvo

package runimage_test

import (
	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/runimage"
	"runtime"
	"testing"
)

// Force generation exhaustion after a real cached execution. Reusing token 1
// must not revive a proof whose public descriptor has since been invalidated.
func TestSessionAdmissionGenerationWrap(t *testing.T) {
	if !runimage.ForeignSessionsAvailable() {
		t.Skip("foreign session unavailable")
	}
	a, err := runimage.NewCodeArena(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	dispatch, ok := backendcompiled.RenvoEmitLinkedDispatcher(1, 2, runtime.GOARCH == "arm64")
	if !ok {
		t.Fatal("dispatcher")
	}
	d, err := a.InstallLinkedDispatcher(dispatch, 2)
	if err != nil {
		t.Fatal(err)
	}
	records := []int{1, 0, 0, 0, 0, 0, 0, 1, 3, 0, 1, 0, 2, 2, 0, 0, 0, 0, 0, 0, 2, 4, 0, 1}
	code, ok := backendcompiled.RenvoEmitPureBlock(records, 2, runtime.GOARCH == "arm64")
	if !ok {
		t.Fatal("leaf")
	}
	entry, err := a.InstallBlock(code, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err = a.AdmitLinkedBlock(entry, 2, 1); err != nil {
		t.Fatal(err)
	}
	call, err := a.PrepareLinkedCall(d, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err = call.PrepareTarget(0, entry, 1); err != nil {
		t.Fatal(err)
	}
	m := new(runimage.LinkedContextABI)
	m.Blocks[0] = runimage.LinkedDescriptorABI{PC: 0, Entry: uint64(entry), Instructions: 1}
	state := []uint64{7, 0}
	if err = call.CallSession(state, m, 17); err != nil || state[0] != 24 {
		t.Fatal("warm", err, state)
	}
	runimage.ForceSessionEpochForTesting(a, ^uint64(0))
	m.Blocks[0].Entry++
	state = []uint64{7, 0}
	if err = call.CallSession(state, m, 17); err != nil {
		t.Fatal(err)
	}
	if state[0] != 7 || m.Total != 0 || m.Remaining != 17 || m.AdmissionEpoch != 0 {
		t.Fatal("generation replay", state, m.Total, m.Remaining)
	}
}
