package bits

import (
	host "math/bits"
	"testing"
)

func TestWideOperationsMatchHost(t *testing.T) {
	values := []uint64{0, 1, 2, 3, 0xffffffff, 1 << 32, 1 << 63, ^uint64(0), 0x0123456789abcdef, 0xfedcba9876543210}
	// Deterministic cross-limb/carry patterns, not just small scalar products.
	seed := uint64(0xabcdef1234567890)
	for i := 0; i < 512; i++ {
		seed ^= seed << 13
		seed ^= seed >> 7
		seed ^= seed << 17
		values = append(values, seed)
	}
	for i, x := range values {
		y := values[(i*37+11)%len(values)]
		hi, lo := Mul64(x, y)
		wantHi, wantLo := host.Mul64(x, y)
		if hi != wantHi || lo != wantLo {
			t.Fatalf("Mul64(%x,%x)=%x:%x want %x:%x", x, y, hi, lo, wantHi, wantLo)
		}
		for carry := uint64(0); carry <= 1; carry++ {
			sum, out := Add64(x, y, carry)
			wantSum, wantOut := host.Add64(x, y, carry)
			if sum != wantSum || out != wantOut {
				t.Fatalf("Add64(%x,%x,%d)=%x,%d want %x,%d", x, y, carry, sum, out, wantSum, wantOut)
			}
		}
		if Len64(x) != host.Len64(x) || LeadingZeros64(x) != host.LeadingZeros64(x) || TrailingZeros64(x) != host.TrailingZeros64(x) || OnesCount64(x) != host.OnesCount64(x) || Reverse64(x) != host.Reverse64(x) || ReverseBytes64(x) != host.ReverseBytes64(x) {
			t.Fatalf("64-bit transforms differ for %x", x)
		}
		w := uint32(x)
		if Len32(w) != host.Len32(w) || LeadingZeros32(w) != host.LeadingZeros32(w) || TrailingZeros32(w) != host.TrailingZeros32(w) || OnesCount32(w) != host.OnesCount32(w) || Reverse32(w) != host.Reverse32(w) || ReverseBytes32(w) != host.ReverseBytes32(w) {
			t.Fatalf("32-bit transforms differ for %x", w)
		}
	}
	for _, x := range values[:10] {
		for _, y := range values[:10] {
			hi, lo := Mul64(x, y)
			a, b := host.Mul64(x, y)
			if hi != a || lo != b {
				t.Fatalf("boundary product %x*%x", x, y)
			}
		}
	}
}
