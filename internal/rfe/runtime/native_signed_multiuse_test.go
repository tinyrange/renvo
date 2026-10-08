//go:build !renvo

package runtime

import "testing"

func TestNativeSignedLoadFullAndLowWordUsers(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(4, 3); err != nil {
		t.Fatal(err)
	}
	for _, size := range []int{1, 2, 4} {
		var b Builder
		raw := b.MemoryLoad(b.Load(0), size)
		sign := b.Constant(uint64(1) << (size*8 - 1))
		full := b.Binary(Sub, b.Binary(Xor, raw, sign), sign)
		b.Store(1, full)
		b.Checkpoint(4, 1)
		b.Store(2, b.Binary(And, full, b.Constant(0xffffffff)))
		b.Store(3, b.Constant(4))
		b.Checkpoint(4, 2)
		b.LoopContinue(b.Constant(0))
		entry, err := n.CompileLoop(b.FinishMemory(4), 4, 2)
		if err != nil {
			t.Fatal(err)
		}
		var page [4096]byte
		epoch := uint64(1)
		m := new(MemoryContext)
		m.Fill(1, &page, 1, &epoch)
		m.ClaimLinks(n)
		m.PublishLink(0, entry, 2, [17]uint8{0, 1, 1})
		for _, raw := range []uint64{0, 1, 127, 128, 255, 32767, 32768, 65535, 0x7fffffff, 0x80000000, 0xffffffff} {
			for i := 0; i < size; i++ {
				page[i] = byte(raw >> (i * 8))
			}
			var want uint64
			switch size {
			case 1:
				want = uint64(int64(int8(raw)))
			case 2:
				want = uint64(int64(int16(raw)))
			case 4:
				want = uint64(int64(int32(raw)))
			}
			state := []uint64{4096, 0, 0, 0}
			if err := n.CallLinked(state, m, 64); err != nil {
				t.Fatal(err)
			}
			if state[1] != want || state[2] != uint64(uint32(want)) || state[3] != 4 || m.Total != 2 || m.MemoryTotal != 1 || m.Status != 0 {
				t.Fatal("full/low signed users", size, raw, want, state, m.Total, m.Status)
			}
		}
	}
}
