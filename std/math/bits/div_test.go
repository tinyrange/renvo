package bits

import (
	host "math/bits"
	"testing"
)

func TestDiv64CrossLimbAndOverflow(t *testing.T) {
	values := []uint64{0, 1, 2, 3, 0xffffffff, 1 << 32, 1 << 63, (1 << 63) + 1, ^uint64(0)}
	seed := uint64(0x123456789abcdef)
	for j := 0; j < 512; j++ {
		seed ^= seed << 13
		seed ^= seed >> 7
		seed ^= seed << 17
		values = append(values, seed)
	}
	for i, y := range values {
		if y == 0 {
			continue
		}
		for _, hi := range []uint64{0, y / 2, y - 1} {
			lo := values[(i*37+7)%len(values)]
			q, r := Div64(hi, lo, y)
			wantQ, wantR := host.Div64(hi, lo, y)
			if q != wantQ || r != wantR {
				t.Fatalf("%x:%x / %x = %x,%x want %x,%x", hi, lo, y, q, r, wantQ, wantR)
			}
		}
	}
	for _, v := range [][3]uint64{{0, 0, 0}, {1, 0, 1}, {^uint64(0), 0, 2}} {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatal("missing panic", v)
				}
			}()
			Div64(v[0], v[1], v[2])
		}()
	}
}
