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
	for _, paired := range []bool{false, true} {
		testSessionAdmissionGenerationWrap(t, paired)
	}
}

func testSessionAdmissionGenerationWrap(t *testing.T, paired bool) {
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
	if paired {
		records[19] = 4
	}
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
	if paired {
		records[19] = 0
		secondCode, emitted := backendcompiled.RenvoEmitPureBlock(records, 2, runtime.GOARCH == "arm64")
		if !emitted {
			t.Fatal("second leaf")
		}
		second, e := a.InstallBlock(secondCode, 2)
		if e != nil {
			t.Fatal(e)
		}
		if e = a.AdmitLinkedBlock(second, 2, 1); e != nil {
			t.Fatal(e)
		}
		if e = call.PrepareTarget(4, second, 1); e != nil {
			t.Fatal(e)
		}
		m.Blocks[1] = runimage.LinkedDescriptorABI{PC: 4, Entry: uint64(second), Instructions: 1}
	}
	state := []uint64{7, 0}
	if err = call.CallSession(state, m, 17); err != nil || state[0] != 24 {
		t.Fatal("warm", err, state)
	}
	runimage.ForceSessionEpochForTesting(a, ^uint64(0))
	want, retired := uint64(7), uint64(0)
	if paired {
		// Root remains publicly valid. Its warmed outgoing edge must not
		// replay the old epoch-one admission of the now-revoked second body.
		m.Blocks[1].Entry++
		want, retired = 8, 1
	} else {
		m.Blocks[0].Entry++
	}
	state = []uint64{7, 0}
	if err = call.CallSession(state, m, 17); err != nil {
		t.Fatal(err)
	}
	if state[0] != want || m.Total != retired || m.Remaining != 17-retired || m.AdmissionEpoch != 0 {
		t.Fatal("generation replay", state, m.Total, m.Remaining)
	}
}
