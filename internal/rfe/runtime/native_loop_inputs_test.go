//go:build !renvo

package runtime

import "testing"

func TestNativeLoopReadonlyAndTemporarilyChangedInputs(t *testing.T) {
	const words = 32
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(words, 31); err != nil {
		t.Fatal(err)
	}
	for _, oneShot := range []bool{false, true} {
		for _, transient := range []bool{false, true} {
			var b Builder
			original := b.Load(0)
			if transient {
				b.Store(0, b.Binary(Add, original, b.Constant(77)))
			}
			sum := b.Constant(0)
			for i := 0; i < 28; i++ {
				sum = b.Binary(Add, sum, b.Binary(Mul, b.Load(i), b.Constant(uint64(i+1))))
			}
			b.Store(28, b.Binary(Add, b.Load(28), sum))
			b.Store(29, b.Binary(Add, b.Load(29), b.Constant(1)))
			b.Checkpoint(words, 1)
			b.MemoryLoad(b.Load(30), 1)
			b.Store(0, original)
			condition := b.Binary(Less, b.Load(29), b.Constant(3))
			if oneShot {
				condition = b.Constant(0)
			}
			b.Store(31, b.Choose(condition, b.Constant(0), b.Constant(4)))
			b.Checkpoint(words, 2)
			b.LoopContinue(condition)
			entry, err := n.CompileLoop(b.FinishMemory(words), words, 2)
			if err != nil {
				t.Fatal(err)
			}
			for _, fault := range []bool{false, true} {
				var page [4096]byte
				epoch := uint64(1)
				m := new(MemoryContext)
				m.Fill(1, &page, 1, &epoch)
				m.ClaimLinks(n)
				m.PublishLink(0, entry, 2, [17]uint8{0, 0, 1})
				state := make([]uint64, words)
				wantSum := uint64(0)
				for i := 0; i < 28; i++ {
					state[i] = uint64(100 + i)
					wantSum += state[i] * uint64(i+1)
				}
				if transient {
					wantSum += 77
				}
				state[30] = 4096
				if fault {
					state[30] = 8192
				}
				if err := n.CallLinked(state, m, 64); err != nil {
					t.Fatal(err)
				}
				count, total, status, pc, full := uint64(3), uint64(6), uint64(0), uint64(4), uint64(3)
				if oneShot {
					count, total, full = 1, 2, 1
				}
				if fault {
					count, total, status, pc, full = 1, 1, 1, 0, 0
				}
				if state[28] != wantSum*count || state[29] != count || state[31] != pc || m.Total != total || m.Status != status || m.LoopIterations != full || m.MemoryTotal != full || m.CodeView != [4]uint64{} || m.PreparedTargets != 0 {
					t.Fatalf("oneshot=%v transient=%v fault=%v sum=%d count=%d state=%v progress=%d status=%d", oneShot, transient, fault, wantSum, count, state, m.Total, m.Status)
				}
				for i := 0; i < 28; i++ {
					want := uint64(100 + i)
					if i == 0 && transient && fault {
						want += 77
					}
					if state[i] != want {
						t.Fatalf("input restoration: slot=%d got=%d want=%d", i, state[i], want)
					}
				}
			}
		}
	}
}
