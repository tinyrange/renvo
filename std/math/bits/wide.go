package bits

// Add64 returns the sum and carry-out of x, y and a carry-in of 0 or 1.
func Add64(x, y, carry uint64) (sum, carryOut uint64) {
	sum = x + y
	if sum < x {
		carryOut = 1
	}
	next := sum + carry
	if next < sum {
		carryOut = 1
	}
	return next, carryOut
}

// Mul64 returns the high and low halves of the full 128-bit unsigned product.
// Four 32-bit limb products suffice without requiring a host 128-bit integer.
func Mul64(x, y uint64) (hi, lo uint64) {
	const mask uint64 = 0xffffffff
	x0, x1 := x&mask, x>>32
	y0, y1 := y&mask, y>>32
	w0 := x0 * y0
	t := x1*y0 + (w0 >> 32)
	w1, w2 := t&mask, t>>32
	w1 += x0 * y1
	hi = x1*y1 + w2 + (w1 >> 32)
	lo = x * y
	return hi, lo
}

func Len32(x uint32) int {
	n := 0
	if x >= 1<<16 {
		x >>= 16
		n += 16
	}
	if x >= 1<<8 {
		x >>= 8
		n += 8
	}
	if x >= 1<<4 {
		x >>= 4
		n += 4
	}
	if x >= 1<<2 {
		x >>= 2
		n += 2
	}
	if x >= 1<<1 {
		x >>= 1
		n++
	}
	if x != 0 {
		n++
	}
	return n
}
func Len64(x uint64) int {
	if x>>32 != 0 {
		return 32 + Len32(uint32(x>>32))
	}
	return Len32(uint32(x))
}
func LeadingZeros32(x uint32) int  { return 32 - Len32(x) }
func LeadingZeros64(x uint64) int  { return 64 - Len64(x) }
func TrailingZeros32(x uint32) int { return Len32((x & -x) - 1) }
func TrailingZeros64(x uint64) int { return Len64((x & -x) - 1) }

func Reverse32(x uint32) uint32 {
	x = (x>>1)&0x55555555 | (x&0x55555555)<<1
	x = (x>>2)&0x33333333 | (x&0x33333333)<<2
	x = (x>>4)&0x0f0f0f0f | (x&0x0f0f0f0f)<<4
	return ReverseBytes32(x)
}
func Reverse64(x uint64) uint64 {
	return uint64(Reverse32(uint32(x)))<<32 | uint64(Reverse32(uint32(x>>32)))
}
func ReverseBytes32(x uint32) uint32 {
	x = (x>>8)&0x00ff00ff | (x&0x00ff00ff)<<8
	return x>>16 | x<<16
}
func ReverseBytes64(x uint64) uint64 {
	return uint64(ReverseBytes32(uint32(x)))<<32 | uint64(ReverseBytes32(uint32(x>>32)))
}
func OnesCount32(x uint32) int {
	x = x - (x>>1)&0x55555555
	x = (x & 0x33333333) + (x>>2)&0x33333333
	x = (x + (x >> 4)) & 0x0f0f0f0f
	x += x >> 8
	x += x >> 16
	return int(x & 0x3f)
}
func OnesCount64(x uint64) int { return OnesCount32(uint32(x)) + OnesCount32(uint32(x>>32)) }
