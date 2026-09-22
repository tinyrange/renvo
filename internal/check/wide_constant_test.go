package check

import "testing"

func TestWideConstantArithmetic(t *testing.T) {
	for _, bits := range []int{15, 30, 31, 32, 63, 64, 100, 255, 1024} {
		large := wideShift(wideSmall(1), bits, true)
		if _, ok := wideInt(large); ok && bits >= 64 {
			t.Fatalf("2^%d fit in int", bits)
		}
		one := wideAdd(wideAdd(large, wideSmall(1)), wideNegate(large))
		if v, ok := wideInt(one); !ok || v != 1 {
			t.Fatalf("large cancellation at bit %d: %d %v", bits, v, ok)
		}
		product := wideMultiply(wideAdd(large, wideSmall(1)), wideAdd(large, wideSmall(-1)))
		want := wideAdd(wideShift(wideSmall(1), bits*2, true), wideSmall(-1))
		if wideMagnitudeCompare(product, want) != 0 || product.negative {
			t.Fatalf("large multiplication at bit %d", bits)
		}
		if v, ok := wideInt(wideShift(large, bits, false)); !ok || v != 1 {
			t.Fatalf("large shift at bit %d", bits)
		}
		negative := wideNegate(wideAdd(large, wideSmall(1)))
		if v, ok := wideInt(wideShift(negative, bits, false)); !ok || v != -2 {
			t.Fatalf("negative rounding at bit %d: %d %v", bits, v, ok)
		}
	}
	for _, left := range []int{-32769, -32768, -1, 0, 1, 32767, 32768} {
		for _, right := range []int{-32769, -1, 0, 1, 32768} {
			if v, ok := wideInt(wideAdd(wideSmall(left), wideSmall(right))); !ok || v != left+right {
				t.Fatalf("add %d %d", left, right)
			}
			if v, ok := wideInt(wideMultiply(wideSmall(left), wideSmall(right))); !ok || v != left*right {
				t.Fatalf("multiply %d %d", left, right)
			}
		}
	}
	maximum := int(^uint(0) >> 1)
	for _, v := range []int{maximum, -maximum - 1} {
		if got, ok := wideInt(wideSmall(v)); !ok || got != v {
			t.Fatalf("int boundary %d: %d %v", v, got, ok)
		}
	}
}
