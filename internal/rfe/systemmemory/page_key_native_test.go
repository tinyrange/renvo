//go:build !renvo

package systemmemory

import (
	"encoding/binary"
	"fmt"
	emu "renvo.dev/internal/rfe/runtime"
	"runtime"
	"testing"
)

func TestNativeFoldedPagesAvoidThrashingAndKeepPreciseRevocation(t *testing.T) {
	for _, compatibility := range []bool{false, true} {
		t.Run(map[bool]string{false: "host-session", true: "compatibility-session"}[compatibility], func(t *testing.T) {
			testNativeFoldedPages(t, compatibility)
		})
	}
}

func testNativeFoldedPages(t *testing.T, compatibility bool) {
	const left, right = uint64(0x9000), uint64(0x49000)
	writable, invalid := true, false
	data := make([]byte, 3*4096)
	m := New(data, func() bool { return true }, func(a uint64, x bool) (uint64, bool, error) {
		switch a &^ 4095 {
		case left:
			return a - left, writable, nil
		case right:
			if !invalid {
				return 4096 + a - right, writable, nil
			}
		}
		return 0, false, fmt.Errorf("unmapped")
	})
	m.NativeAddress = UnfoldPageCacheAddress
	n, err := emu.NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err := n.PrepareLinks(5, 0); err != nil {
		t.Fatal(err)
	}
	ctx := m.NativeContext()
	var b emu.Builder
	b.MemoryInstructionProgress = true
	b.MemoryAddress = PageCacheAddress
	b.Store(3, b.MemoryLoad(b.Load(1), 8))
	b.Store(0, b.Constant(4))
	b.Checkpoint(5, 1)
	b.Store(4, b.MemoryLoad(b.Load(2), 8))
	b.Store(0, b.Constant(8))
	b.Checkpoint(5, 2)
	b.MemoryStore(b.Load(2), b.Binary(emu.Add, b.Load(4), b.Constant(1)), 8)
	b.Store(0, b.Constant(0))
	b.Checkpoint(5, 3)
	b.LoopContinue(b.Constant(1))
	entry, err := n.CompileCodeVersionedLoop(b.FinishMemory(5), 5, 3, ctx)
	if err != nil {
		t.Fatal(err)
	}
	ctx.ClaimLinks(n)
	ctx.PublishLink(0, entry, 3, [17]uint8{})
	refill := func() { m.NativeRefill(UnfoldPageCacheAddress(left)); m.NativeRefill(UnfoldPageCacheAddress(right)) }
	binary.LittleEndian.PutUint64(data, 10)
	binary.LittleEndian.PutUint64(data[4096:], 20)
	refill()
	runtime.GC()
	state := []uint64{0, left, right, 999, 999}
	if compatibility {
		// An architectural-state alias forces the same conservative boundary
		// even on hosts with a foreign bridge. No workload limit is changed.
		ctx.Clock = &state[2]
	}
	// Complete the same 100 iterations even on hosts retaining the
	// conservative 64-instruction call limit. A refill/link miss remains a
	// failure: only an independently bounded scheduling return is accepted.
	completed := uint64(0)
	for completed < 300 {
		budget := uint64(300) - completed
		if err := n.RunLinkedBackingSession(state, ctx, budget, data); err != nil {
			t.Fatal(err)
		}
		want := budget
		if (compatibility || !emu.NativeSessionsAvailable()) && want > 64 {
			want = 64
		}
		want -= want % 3
		if ctx.Status != 0 || ctx.Total != want || want == 0 || ctx.Remaining != budget-want {
			t.Fatal("unexpected native exit", ctx.Status, ctx.Total, ctx.Remaining, want)
		}
		completed += ctx.Total
	}
	if ctx.Status != 0 || completed != 300 || state[3] != 10 || state[4] != 119 || binary.LittleEndian.Uint64(data[4096:]) != 120 {
		t.Fatal("distant pages still thrash", state, ctx.Status, ctx.Total)
	}
	// New permissions revoke both tags; a native read succeeds but its store must
	// stop at the exact original checkpoint, without granting authority by hash.
	writable = false
	m.InvalidateMappings()
	refill()
	state = []uint64{0, left, right, 999, 999}
	if err := n.RunLinkedBackingSession(state, ctx, 300, data); err != nil {
		t.Fatal(err)
	}
	if ctx.Status != 1 || ctx.Total != 2 || state[0] != 8 || ctx.Address != UnfoldPageCacheAddress(right) || binary.LittleEndian.Uint64(data[4096:]) != 120 {
		t.Fatal("readonly hash authority", state, ctx.Status, ctx.Total)
	}
	writable = true
	if _, err := m.CodeVersion(right); err != nil {
		t.Fatal(err)
	}
	refill()
	state = []uint64{0, left, right, 999, 999}
	if err := n.RunLinkedBackingSession(state, ctx, 300, data); err != nil {
		t.Fatal(err)
	}
	if ctx.Status != 1 || ctx.Total != 2 || state[0] != 8 || binary.LittleEndian.Uint64(data[4096:]) != 120 {
		t.Fatal("code barrier lost", state, ctx.Status, ctx.Total)
	}
	invalid = true
	m.InvalidateMappings()
	refill()
	state = []uint64{0, left, right, 999, 999}
	if err := n.RunLinkedBackingSession(state, ctx, 300, data); err != nil {
		t.Fatal(err)
	}
	if ctx.Status != 1 || ctx.Total != 1 || state[0] != 4 || state[4] != 999 {
		t.Fatal("stale mapping survived inverse key", state, ctx.Status, ctx.Total)
	}
	invalid = false
	refill()
	state = []uint64{0, left, right + 4092, 999, 999}
	if err := n.RunLinkedBackingSession(state, ctx, 300, data); err != nil {
		t.Fatal(err)
	}
	if ctx.Status != 1 || ctx.Total != 1 || state[0] != 4 || state[4] != 999 || ctx.Address != UnfoldPageCacheAddress(right+4092) {
		t.Fatal("cross-page fault lost offset", state, ctx.Status, ctx.Total)
	}
}

func TestPageCacheKeyRetainsEveryAddressBit(t *testing.T) {
	x := uint64(0x8877665544332211)
	for i := 0; i < 8192; i++ {
		x = x*6364136223846793005 + 1442695040888963407
		key := UnfoldPageCacheAddress(x)
		if UnfoldPageCacheAddress(key) != x || key&4095 != x&4095 || key>>18 != x>>18 {
			t.Fatal("key lost address bits")
		}
	}
}
