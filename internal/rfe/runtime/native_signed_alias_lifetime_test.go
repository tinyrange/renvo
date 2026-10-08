//go:build !renvo

package runtime

import "testing"

func TestNativeSignedLoadDelayedNarrowConsumer(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(28, 27); err != nil {
		t.Fatal(err)
	}
	var b Builder
	raw := b.MemoryLoad(b.Load(0), 2)
	sign := b.Constant(32768)
	signed := b.Binary(And, b.Binary(Sub, b.Binary(Xor, raw, sign), sign), b.Constant(0xffffffff))
	sum := b.Binary(Add, signed, b.Load(1))
	// Independent live arithmetic sits between the fused source and consumer.
	for i := 2; i < 26; i++ {
		b.Store(i, b.Binary(Mul, b.Load(i), b.Constant(uint64(i+1))))
	}
	// Emit raw valid IR so Builder algebra does not move/recreate the add.
	b.Store(26, b.emit(Op{Kind: And, A: sum, B: b.Constant(0xffffffff)}))
	b.Store(27, b.Constant(4))
	b.Checkpoint(28, 2)
	b.LoopContinue(b.Constant(0))
	entry, err := n.CompileLoop(b.FinishMemory(28), 28, 2)
	if err != nil {
		t.Fatal(err)
	}
	var page [4096]byte
	epoch := uint64(1)
	m := new(MemoryContext)
	m.Fill(1, &page, 1, &epoch)
	m.ClaimLinks(n)
	m.PublishLink(0, entry, 2, [17]uint8{0, 1, 1})
	for _, rawValue := range []uint16{0, 1, 32767, 32768, 65535} {
		for _, addend := range []uint64{0, 1, 0xffffffff, 0xffffffffffffffff, 0xabcdef1234567890} {
			page[0], page[1] = byte(rawValue), byte(rawValue>>8)
			state := make([]uint64, 28)
			state[0], state[1] = 4096, addend
			for i := 2; i < 26; i++ {
				state[i] = uint64(i + 101)
			}
			want := uint64(uint32(uint64(int64(int16(rawValue))) + addend))
			if err := n.CallLinked(state, m, 64); err != nil {
				t.Fatal(err)
			}
			if state[26] != want || state[27] != 4 || m.Total != 2 || m.Status != 0 {
				t.Fatal("signed alias lifetime", rawValue, addend, want, state, m.Total, m.Status)
			}
			for i := 2; i < 26; i++ {
				if state[i] != uint64((i+101)*(i+1)) {
					t.Fatal("independent SSA value corrupted", i, state[i])
				}
			}
		}
	}
}
