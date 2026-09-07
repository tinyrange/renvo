package check

// Binary long division keeps a remainder smaller than the divisor after each
// step. Only the newly allocated quotient/remainder buffers are mutated;
// operands remain immutable, and limb arithmetic fits in a 32-bit int.
func wideDivide(left, right wideConstant) (wideConstant, wideConstant) {
	if !left.ok || !right.ok || len(right.words) == 0 {
		return wideConstant{}, wideConstant{}
	}
	if wideMagnitudeCompare(left, right) < 0 {
		return wideSmall(0), left
	}
	quotient := wideConstant{words: make([]int, len(left.words)), negative: left.negative != right.negative, ok: true}
	remainder := wideConstant{words: make([]int, 0, len(right.words)+1), ok: true}
	for limb := len(left.words) - 1; limb >= 0; limb-- {
		for bit := 14; bit >= 0; bit-- {
			carry := left.words[limb] >> uint(bit) & 1
			for i := 0; i < len(remainder.words); i++ {
				word := remainder.words[i]*2 + carry
				remainder.words[i] = word & 32767
				carry = word >> 15
			}
			if carry != 0 {
				remainder.words = append(remainder.words, carry)
			}
			if wideMagnitudeCompare(remainder, right) >= 0 {
				borrow := 0
				for i := 0; i < len(remainder.words); i++ {
					word := remainder.words[i] - borrow
					if i < len(right.words) {
						word -= right.words[i]
					}
					borrow = 0
					if word < 0 {
						word += 32768
						borrow = 1
					}
					remainder.words[i] = word
				}
				remainder = wideNormalize(remainder)
				quotient.words[limb] |= 1 << uint(bit)
			}
		}
	}
	remainder.negative = left.negative
	return wideNormalize(quotient), wideNormalize(remainder)
}
