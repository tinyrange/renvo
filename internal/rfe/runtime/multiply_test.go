package runtime

import (
	"math/rand"
	"runtime"
	"testing"
)

func TestMultiplyIRNativeAndStrengthReduction(t *testing.T) {
	n, err := NewNative(1 << 20)
	supported := (runtime.GOOS == "linux" || runtime.GOOS == "windows") && (runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64") || runtime.GOOS == "darwin" && runtime.GOARCH == "arm64"
	if err != nil && supported {
		t.Fatal(err)
	}
	if n != nil {
		defer n.Close()
	}
	rng := rand.New(rand.NewSource(12))
	for _, factor := range []uint64{0, 1, 2, 3, 255, 1 << 31, 1 << 63, ^uint64(0)} {
		var b Builder
		b.Store(2, b.Binary(Mul, b.Load(0), b.Load(1)))
		b.Store(3, b.Binary(Mul, b.Load(0), b.Constant(factor)))
		ops := b.Finish(4)
		entry := 0
		if n != nil {
			entry, err = n.Compile(ops, 4)
			if err != nil {
				t.Fatal(err)
			}
		}
		for i := 0; i < 200; i++ {
			x, y := rng.Uint64(), rng.Uint64()
			if i == 0 {
				x, y = ^uint64(0), 2
			}
			state := []uint64{x, y, 0, 0}
			compiled := append([]uint64(nil), state...)
			if err = Interpret(ops, state); err != nil {
				t.Fatal(err)
			}
			if state[2] != x*y || state[3] != x*factor {
				t.Fatalf("multiplication factor=%x: %x", factor, state)
			}
			if n != nil {
				if err = n.Call(entry, compiled); err != nil {
					t.Fatal(err)
				}
				for j := range state {
					if state[j] != compiled[j] {
						t.Fatal("native multiplication")
					}
				}
			}
		}
	}
}
