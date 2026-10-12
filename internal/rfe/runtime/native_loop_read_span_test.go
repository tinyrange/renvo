//go:build !renvo

package runtime

import (
	"reflect"
	"testing"
)

// Exercise the contract across a proven span, a rejected next span, and each
// precise load checkpoint. The scalar oracle uses architectural operations,
// not the native span recognizer or its range proof. Three guest checkpoints
// let one compatibility quantum cross the sixteen-iteration fast boundary.
func TestNativeLoopReadSpanSignedStateAndBoundaries(t *testing.T) {
	const words = 11
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(words, 10); err != nil {
		t.Fatal(err)
	}
	var b Builder
	mask := b.Constant(0xffffffff)
	i, j := b.Load(0), b.Load(1)
	ix, jx := b.Binary(And, i, mask), b.Binary(And, j, mask)
	a := b.Binary(Add, b.Load(6), b.Shift(Shl, ix, 1))
	z := b.Binary(Add, b.Load(7), b.Shift(Shl, jx, 1))
	signed := func(address Value) Value {
		v := b.MemoryLoad(address, 2)
		return b.Binary(And, b.Binary(Sub, b.Binary(Xor, v, b.Constant(32768)), b.Constant(32768)), mask)
	}
	left := signed(a)
	next := b.Binary(And, b.Binary(Add, ix, b.Constant(1)), mask)
	b.Store(3, left)
	b.Store(0, next)
	b.Store(10, b.Constant(4))
	b.Checkpoint(words, 1)
	right := signed(z)
	b.Store(4, right)
	b.Store(1, b.Binary(And, b.Binary(Add, jx, b.Load(8)), mask))
	b.Store(10, b.Constant(8))
	b.Checkpoint(words, 2)
	b.Store(2, b.Binary(And, b.Binary(Add, b.Load(2), b.Binary(Mul, left, right)), mask))
	flags := b.ArithmeticFlags(b.Load(9), next, 32, true)
	b.Store(5, flags)
	condition, ok := b.FlagCondition(flags, 1)
	if !ok {
		t.Fatal("condition")
	}
	b.Store(10, b.Choose(condition, b.Constant(0), b.Constant(12)))
	b.Checkpoint(words, 3)
	b.LoopContinue(condition)
	entry, err := n.CompileLoop(b.FinishMemory(words), words, 3)
	if err != nil {
		t.Fatal(err)
	}
	pages := [2]*[4096]byte{new([4096]byte), new([4096]byte)}
	for p, page := range pages {
		for k := range page {
			page[k] = byte(k*37 + p*19 + 129)
		}
	}
	for _, trial := range []struct {
		start, offA, offB, stride uint64
		permissions               uint64
	}{
		{0, 4096, 4096, 1, 1},
		{0, 4096, 4096, 0, 1},
		{0, 4096, 4096, 7, 1},
		{0, 4096, 4096, 4096, 1},
		{0, 8160, 4096, 1, 0}, // first load faults after one proven span
		{0, 4096, 8160, 1, 0}, // second load faults after one proven span
		{0, 8158, 8158, 1, 0}, // next-span proof fails before the actual fault
		{0, 4096, 8190, 1, 0}, // no premature fault or permission-fact publication
		{0, 8190, 4096, 1, 0},
		{0, 4096, 8192, 1, 0}, // rejected permission probe must not poison fallback
		{0, 8192, 4096, 1, 0},
		{0xfffffff0, 4096, 4096, 1, 1}, // W-index wrap just after a safe span
		{0xdeadbeef00000000, 4096, 4096, 1, 1},
		{0, 1 << 48, 4096, 1, 1},
	} {
		for _, limit := range []uint64{1, 15, 16, 17, 19, 100} {
			for budget := uint64(1); budget <= 64; budget++ {
				m := new(MemoryContext)
				epoch := uint64(1)
				m.Fill(1, pages[0], 1, &epoch)
				m.Fill(2, pages[1], trial.permissions, &epoch)
				m.ClaimLinks(n)
				m.PublishLink(0, entry, 3, [17]uint8{})
				baseA := trial.offA - 2*uint64(uint32(trial.start))
				baseB := trial.offB - 2*uint64(uint32(trial.start))
				state := []uint64{trial.start, trial.start, 0xfffffff1, 0xaabbccdd, 0x99887766, 0xf0000000, baseA, baseB, trial.stride, uint64(uint32(trial.start + limit)), 0}
				want := append([]uint64(nil), state...)
				var retired, memories, iterations, exits, status, fault uint64
				for retired+3 <= budget {
					failed := false
					for stream := 0; stream < 2; stream++ {
						address := want[6+stream] + 2*uint64(uint32(want[stream]))
						page, off := address/4096, address%4096
						if page < 1 || page > 2 || off > 4094 || page == 2 && trial.permissions&1 == 0 {
							status, fault, failed = 1, address, true
							break
						}
						value := uint16(pages[page-1][off]) | uint16(pages[page-1][off+1])<<8
						want[3+stream] = uint64(uint32(int32(int16(value))))
						step := uint64(1)
						if stream == 1 {
							step = want[8]
						}
						want[stream] = uint64(uint32(want[stream] + step))
						want[10] = uint64(4 + stream*4)
						retired++
						memories++
					}
					if failed {
						break
					}
					want[2] = uint64(uint32(want[2] + want[3]*want[4]))
					x, y := uint32(want[9]), uint32(want[0])
					difference := x - y
					flags := uint64(difference>>31) << 31
					if difference == 0 {
						flags |= 1 << 30
					}
					if x >= y {
						flags |= 1 << 29
					}
					if (x^y)&(x^difference)&0x80000000 != 0 {
						flags |= 1 << 28
					}
					want[5], want[10] = flags, 0
					retired++
					iterations++
					if x == y {
						want[10], exits = 12, 1
						break
					}
				}
				if err = n.CallLinked(state, m, budget); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(state, want) || m.Total != retired || m.Remaining != budget-retired || m.MemoryTotal != memories || m.LoopIterations != iterations || m.LoopExits != exits || m.Status != status || status != 0 && m.Address != fault {
					t.Fatalf("trial=%+v limit=%d budget=%d state=%x want=%x progress=%d/%d memory=%d/%d loops=%d/%d exits=%d/%d status=%d/%d address=%x/%x", trial, limit, budget, state, want, m.Total, retired, m.MemoryTotal, memories, m.LoopIterations, iterations, m.LoopExits, exits, m.Status, status, m.Address, fault)
				}
			}
		}
	}
}
