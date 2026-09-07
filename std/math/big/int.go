// Package big implements arbitrary-precision arithmetic.
package big

// Int is a signed integer. Its zero value is zero. Magnitudes use little-endian
// base-2^32 limbs, independently of the target's native integer width.
type Int struct {
	negative bool
	words    []uint32
}

func NewInt(value int64) *Int { return new(Int).SetInt64(value) }
func normalize(words []uint32) []uint32 {
	for len(words) > 0 && words[len(words)-1] == 0 {
		words = words[:len(words)-1]
	}
	return words
}
func clone(words []uint32) []uint32 { return append([]uint32(nil), words...) }
func (z *Int) put(words []uint32, negative bool) *Int {
	z.words = normalize(words)
	z.negative = negative && len(z.words) != 0
	return z
}
func (z *Int) Set(x *Int) *Int {
	if z == x {
		return z
	}
	return z.put(clone(x.words), x.negative)
}
func (z *Int) SetUint64(value uint64) *Int {
	if value == 0 {
		return z.put(nil, false)
	}
	return z.put([]uint32{uint32(value), uint32(value >> 32)}, false)
}
func (z *Int) SetInt64(value int64) *Int {
	magnitude := uint64(value)
	if value < 0 {
		magnitude = 0 - magnitude
	}
	z.SetUint64(magnitude)
	z.negative = value < 0
	return z
}
func (x *Int) Sign() int {
	if len(x.words) == 0 {
		return 0
	}
	if x.negative {
		return -1
	}
	return 1
}
func (x *Int) BitLen() int {
	if len(x.words) == 0 {
		return 0
	}
	n := (len(x.words) - 1) * 32
	top := x.words[len(x.words)-1]
	for top != 0 {
		n++
		top >>= 1
	}
	return n
}
func (x *Int) Uint64() uint64 {
	var value uint64
	if len(x.words) > 0 {
		value = uint64(x.words[0])
	}
	if len(x.words) > 1 {
		value |= uint64(x.words[1]) << 32
	}
	return value
}
func (x *Int) Int64() int64 {
	value := x.Uint64()
	if x.negative {
		value = 0 - value
	}
	return int64(value)
}
func (x *Int) IsUint64() bool { return !x.negative && len(x.words) <= 2 }
func (x *Int) IsInt64() bool {
	bits := x.BitLen()
	return bits < 64 || bits == 64 && x.negative && x.Uint64() == uint64(1)<<63
}
func magnitudeCmp(x, y []uint32) int {
	if len(x) < len(y) {
		return -1
	}
	if len(x) > len(y) {
		return 1
	}
	for i := len(x) - 1; i >= 0; i-- {
		if x[i] < y[i] {
			return -1
		}
		if x[i] > y[i] {
			return 1
		}
	}
	return 0
}
func (x *Int) Cmp(y *Int) int {
	if x.negative != y.negative {
		if x.negative {
			return -1
		}
		return 1
	}
	cmp := magnitudeCmp(x.words, y.words)
	if x.negative {
		return -cmp
	}
	return cmp
}
func (z *Int) Abs(x *Int) *Int { return z.put(clone(x.words), false) }
func (z *Int) Neg(x *Int) *Int { return z.put(clone(x.words), !x.negative) }

func magnitudeAdd(x, y []uint32) []uint32 {
	n := len(x)
	if len(y) > n {
		n = len(y)
	}
	out := make([]uint32, n+1)
	var carry uint64
	for i := 0; i < n; i++ {
		sum := carry
		if i < len(x) {
			sum += uint64(x[i])
		}
		if i < len(y) {
			sum += uint64(y[i])
		}
		out[i] = uint32(sum)
		carry = sum >> 32
	}
	out[n] = uint32(carry)
	return normalize(out)
}

// magnitudeSub requires x >= y.
func magnitudeSub(x, y []uint32) []uint32 {
	out := make([]uint32, len(x))
	var borrow uint64
	for i := 0; i < len(x); i++ {
		part := borrow
		if i < len(y) {
			part += uint64(y[i])
		}
		current := uint64(x[i])
		out[i] = uint32(current - part)
		borrow = 0
		if current < part {
			borrow = 1
		}
	}
	return normalize(out)
}
func (z *Int) Add(x, y *Int) *Int {
	if x.negative == y.negative {
		return z.put(magnitudeAdd(x.words, y.words), x.negative)
	}
	if magnitudeCmp(x.words, y.words) >= 0 {
		return z.put(magnitudeSub(x.words, y.words), x.negative)
	}
	return z.put(magnitudeSub(y.words, x.words), y.negative)
}
func (z *Int) Sub(x, y *Int) *Int {
	if x.negative != y.negative {
		return z.put(magnitudeAdd(x.words, y.words), x.negative)
	}
	if magnitudeCmp(x.words, y.words) >= 0 {
		return z.put(magnitudeSub(x.words, y.words), x.negative)
	}
	return z.put(magnitudeSub(y.words, x.words), !y.negative)
}
func (z *Int) Mul(x, y *Int) *Int {
	if len(x.words) == 0 || len(y.words) == 0 {
		return z.put(nil, false)
	}
	out := make([]uint32, len(x.words)+len(y.words))
	for i, a := range x.words {
		var carry uint64
		for j, b := range y.words {
			product := uint64(a)*uint64(b) + uint64(out[i+j]) + carry
			out[i+j] = uint32(product)
			carry = product >> 32
		}
		out[i+len(y.words)] = uint32(carry)
	}
	return z.put(out, x.negative != y.negative)
}

func (z *Int) Lsh(x *Int, shift uint) *Int {
	if len(x.words) == 0 {
		return z.put(nil, false)
	}
	whole := shift / 32
	bits := shift % 32
	if whole > uint(^uint(0)>>1)-uint(len(x.words))-1 {
		panic("big: shift too large")
	}
	out := make([]uint32, len(x.words)+int(whole)+1)
	var carry uint64
	for i, word := range x.words {
		value := uint64(word)<<bits | carry
		out[i+int(whole)] = uint32(value)
		carry = value >> 32
	}
	out[len(x.words)+int(whole)] = uint32(carry)
	return z.put(out, x.negative)
}
func (z *Int) Rsh(x *Int, shift uint) *Int {
	whole := shift / 32
	bits := shift % 32
	negative := x.negative
	if whole >= uint(len(x.words)) {
		if negative {
			return z.SetInt64(-1)
		}
		return z.SetInt64(0)
	}
	out := make([]uint32, len(x.words)-int(whole))
	discarded := false
	for i := 0; i < int(whole); i++ {
		if x.words[i] != 0 {
			discarded = true
		}
	}
	if bits != 0 && x.words[int(whole)]&((uint32(1)<<bits)-1) != 0 {
		discarded = true
	}
	for i := 0; i < len(out); i++ {
		value := uint64(x.words[i+int(whole)]) >> bits
		if bits != 0 && i+int(whole)+1 < len(x.words) {
			value |= uint64(x.words[i+int(whole)+1]) << (32 - bits)
		}
		out[i] = uint32(value)
	}
	out = normalize(out)
	if negative && discarded {
		out = magnitudeAdd(out, []uint32{1})
	}
	return z.put(out, negative)
}

func magnitudeDiv(x, y []uint32) ([]uint32, []uint32) {
	if len(y) == 0 {
		panic("division by zero")
	}
	if magnitudeCmp(x, y) < 0 {
		return nil, clone(x)
	}
	quotient := make([]uint32, len(x))
	remainder := make([]uint32, 0, len(y)+1)
	for word := len(x) - 1; word >= 0; word-- {
		for bit := 31; bit >= 0; bit-- {
			carry := (x[word] >> uint(bit)) & 1
			for i := 0; i < len(remainder); i++ {
				next := remainder[i] >> 31
				remainder[i] = remainder[i]<<1 | carry
				carry = next
			}
			if carry != 0 {
				remainder = append(remainder, carry)
			}
			if magnitudeCmp(remainder, y) >= 0 {
				var borrow uint64
				for i := 0; i < len(remainder); i++ {
					part := borrow
					if i < len(y) {
						part += uint64(y[i])
					}
					current := uint64(remainder[i])
					remainder[i] = uint32(current - part)
					borrow = 0
					if current < part {
						borrow = 1
					}
				}
				remainder = normalize(remainder)
				quotient[word] |= uint32(1) << uint(bit)
			}
		}
	}
	return normalize(quotient), remainder
}
func (z *Int) QuoRem(x, y, r *Int) (*Int, *Int) {
	negative := x.negative != y.negative
	remainderNegative := x.negative
	q, rem := magnitudeDiv(x.words, y.words)
	z.put(q, negative)
	r.put(rem, remainderNegative)
	return z, r
}
func (z *Int) Quo(x, y *Int) *Int { var remainder Int; z.QuoRem(x, y, &remainder); return z }
func (z *Int) Rem(x, y *Int) *Int { var quotient Int; quotient.QuoRem(x, y, z); return z }

func twosWords(x *Int, n int) []uint32 {
	out := make([]uint32, n)
	copy(out, x.words)
	if x.negative {
		carry := uint64(1)
		for i := 0; i < n; i++ {
			value := uint64(^out[i]) + carry
			out[i] = uint32(value)
			carry = value >> 32
		}
	}
	return out
}
func (z *Int) bitwise(x, y *Int, op byte) *Int {
	n := len(x.words)
	if len(y.words) > n {
		n = len(y.words)
	}
	n++
	a, b := twosWords(x, n), twosWords(y, n)
	for i := 0; i < n; i++ {
		if op == '&' {
			a[i] &= b[i]
		} else if op == '|' {
			a[i] |= b[i]
		} else {
			a[i] ^= b[i]
		}
	}
	negative := a[n-1]>>31 != 0
	if negative {
		carry := uint64(1)
		for i := 0; i < n; i++ {
			value := uint64(^a[i]) + carry
			a[i] = uint32(value)
			carry = value >> 32
		}
	}
	return z.put(a, negative)
}
func (z *Int) And(x, y *Int) *Int { return z.bitwise(x, y, '&') }
func (z *Int) Or(x, y *Int) *Int  { return z.bitwise(x, y, '|') }
func (z *Int) Xor(x, y *Int) *Int { return z.bitwise(x, y, '^') }
func (z *Int) Not(x *Int) *Int    { var neg Int; neg.Neg(x); return z.Sub(&neg, NewInt(1)) }
