package bits

// Div64 returns the quotient and remainder of (hi<<64 | lo) / y.
// The quotient must fit in 64 bits. Like math/bits, zero divisors and
// overflowing quotients panic. Shift/subtract retains the 65th remainder bit
// explicitly, so divisors above 1<<63 do not lose their carry.
func Div64(hi, lo, y uint64) (quo, rem uint64) {
	if y == 0 {
		panic("integer divide by zero")
	}
	if hi >= y {
		panic("integer overflow")
	}
	if hi == 0 {
		return lo / y, lo % y
	}
	rem = hi
	for bit := 64; bit > 0; bit-- {
		carry := rem >> 63
		rem = rem<<1 | (lo>>uint(bit-1))&1
		if carry != 0 || rem >= y {
			rem -= y
			quo |= uint64(1) << uint(bit-1)
		}
	}
	return quo, rem
}
