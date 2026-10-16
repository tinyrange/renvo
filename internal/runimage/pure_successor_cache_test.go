//go:build !renvo

package runimage

import (
	"runtime"
	"testing"
)

// Re-preparation preserves learned edges, not admission. Changing any body
// identity (including its original tail) or evicting a colliding PC loses them.
func TestPreparedSuccessorLifetime(t *testing.T) {
	a, err := NewCodeArena(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	// This test exercises preparation only, never execution. A return is a
	// valid installed image; runtime tests exercise compiler-emitted bodies.
	code := []byte{0xc3}
	if runtime.GOARCH == "arm64" {
		code = []byte{0xc0, 0x03, 0x5f, 0xd6}
	}
	d, err := a.InstallLinkedDispatcher(code, 2)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := a.InstallBlock(code, 2)
	if err != nil {
		t.Fatal(err)
	}
	other, err := a.InstallBlock(code, 2)
	if err != nil {
		t.Fatal(err)
	}
	region, err := a.InstallLoopBlock(code, 2, 3)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range [][2]int{{leaf, 1}, {other, 1}, {region, 3}} {
		if err = a.AdmitLinkedBlock(item[0], 2, item[1]); err != nil {
			t.Fatal(err)
		}
	}
	c, err := a.PrepareLinkedCall(d, 2)
	if err != nil {
		t.Fatal(err)
	}
	find := func(pc uint64) *linkedTarget {
		for i := range a.targets {
			if a.targets[i].PC == pc && a.targets[i].Instructions != 0 {
				return &a.targets[i]
			}
		}
		t.Fatal("missing proof", pc)
		return nil
	}
	const hints = uint64(65) | uint64(32769)<<32
	if err = c.PrepareTarget(0, leaf, 1); err != nil {
		t.Fatal(err)
	}
	p := find(0)
	p.SuccessorOffset, p.CheckedSession, p.DescriptorOffset = hints, 71, 192
	if err = c.PrepareTarget(0, leaf, 1); err != nil {
		t.Fatal(err)
	}
	if p.SuccessorOffset != hints || p.CheckedSession != 0 || p.DescriptorOffset != 0 {
		t.Fatal("re-prepare lost hints or retained authority", *p)
	}
	if err = c.PrepareTarget(0, other, 1); err != nil {
		t.Fatal(err)
	}
	if p.SuccessorOffset != 0 {
		t.Fatal("replacement retained edges")
	}
	if err = c.PrepareTargetWithTail(0, region, 3, leaf, 1); err != nil {
		t.Fatal(err)
	}
	p.SuccessorOffset = hints
	if err = c.PrepareTargetWithTail(0, region, 3, leaf, 1); err != nil || p.SuccessorOffset != hints {
		t.Fatal("same tail", err, *p)
	}
	if err = c.PrepareTargetWithTail(0, region, 3, other, 1); err != nil || p.SuccessorOffset != 0 {
		t.Fatal("changed tail", err, *p)
	}
	p.SuccessorOffset = hints
	for i := uint64(1); i <= 4; i++ {
		if err = c.PrepareTarget(i<<22, leaf, 1); err != nil {
			t.Fatal(err)
		}
	}
	if p.PC == 0 || p.SuccessorOffset != 0 || p.CheckedSession != 0 {
		t.Fatal("collision reused private identity", *p)
	}
}
