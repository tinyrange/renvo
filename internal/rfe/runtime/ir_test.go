package runtime

import (
	"math/rand"
	"runtime"
	"testing"
)

func TestNativeIR(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		if (runtime.GOOS == "linux" || runtime.GOOS == "windows") && (runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64") || runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
			t.Fatal(err)
		}
		t.Skip(err)
	}
	defer n.Close()
	var choice Builder
	choice.Store(1, choice.Choose(choice.Load(0), choice.Constant(0x1234), choice.Constant(0xabcd)))
	ops := choice.Finish(2)
	entry, err := n.Compile(ops, 2)
	if err != nil {
		t.Fatal(err)
	}
	for _, condition := range []uint64{0, 1, 2, 1 << 63, ^uint64(0)} {
		want := uint64(0x1234)
		if condition == 0 {
			want = 0xabcd
		}
		for _, native := range []bool{false, true} {
			state := []uint64{condition, 0}
			if native {
				err = n.Call(entry, state)
			} else {
				err = Interpret(ops, state)
			}
			if err != nil || state[1] != want {
				t.Fatalf("choose(%x), native=%v: %x, %v", condition, native, state[1], err)
			}
		}
	}
	r := rand.New(rand.NewSource(473))
	for kind := Add; kind <= Less; kind++ {
		for _, shift := range []uint64{0, 1, 15, 31, 63, 64, 255} {
			var b Builder
			a, c := b.Load(0), b.Load(1)
			var v Value
			if kind == Shl || kind == Shr {
				v = b.Shift(kind, a, shift)
			} else {
				v = b.Binary(kind, a, c)
			}
			b.Store(2, v)
			ops := b.Finish(3)
			entry, err := n.Compile(ops, 3)
			if err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 20; i++ {
				x, y := r.Uint64(), r.Uint64()
				want := []uint64{x, y, 0}
				got := append([]uint64{}, want...)
				if err = Interpret(ops, want); err != nil {
					t.Fatal(err)
				}
				if err = n.Call(entry, got); err != nil {
					t.Fatal(err)
				}
				for j := range want {
					if got[j] != want[j] {
						t.Fatalf("op %d shift %d state[%d]: got %x want %x (x=%x y=%x)", kind, shift, j, got[j], want[j], x, y)
					}
				}
			}
		}
	}
}
func TestIRValidation(t *testing.T) {
	for _, ops := range [][]Op{nil, {{Kind: LoadState, Imm: 9}}, {{Kind: Add, A: 0, B: 0}}, {{Kind: 99}}, {{Kind: Const}, {Kind: StoreState, A: 0}, {Kind: Add, A: 1, B: 0}}} {
		if Validate(ops, 2) == nil {
			t.Fatalf("accepted invalid block %v", ops)
		}
	}
}
func TestQueueOrdering(t *testing.T) {
	var q Queue
	first, _ := q.Schedule(0, 10, 1, 1)
	second, _ := q.Schedule(0, 10, 1, 2)
	third, _ := q.Schedule(0, 9, 1, 3)
	if !q.Cancel(second) {
		t.Fatal("cancel")
	}
	for _, id := range []uint64{third, first} {
		e, ok := q.Pop(10)
		if !ok || e.ID != id {
			t.Fatal("event ordering", e)
		}
	}
	if _, ok := q.Pop(10); ok {
		t.Fatal("cancelled event delivered")
	}
	if _, err := q.Schedule(10, 9, 0, 0); err == nil {
		t.Fatal("scheduled past event")
	}
}
