package check

// Integer constants have unbounded signed precision. Use one extra 15-bit
// limb for sign extension, operate on two's-complement limbs, then convert the
// result back to the evaluator's immutable sign/magnitude representation.
func wideBitwise(left, right wideConstant, operator string) wideConstant {
	if !left.ok || !right.ok {
		return wideConstant{}
	}
	if operator != "&" && operator != "|" && operator != "^" && operator != "&^" {
		return wideConstant{}
	}
	size := len(left.words)
	if len(right.words) > size {
		size = len(right.words)
	}
	out := wideConstant{words: make([]int, size+1), ok: true}
	leftCarry, rightCarry := 0, 0
	if left.negative {
		leftCarry = 1
	}
	if right.negative {
		rightCarry = 1
	}
	for i := 0; i <= size; i++ {
		var a, b int
		a, leftCarry = wideTwosLimb(left, i, leftCarry)
		b, rightCarry = wideTwosLimb(right, i, rightCarry)
		if operator == "&" {
			out.words[i] = a & b
		}
		if operator == "|" {
			out.words[i] = a | b
		}
		if operator == "^" {
			out.words[i] = a ^ b
		}
		if operator == "&^" {
			out.words[i] = a &^ b
		}
	}
	if out.words[size]&16384 != 0 {
		out.negative = true
		carry := 1
		for i := 0; i <= size; i++ {
			word := (out.words[i] ^ 32767) + carry
			out.words[i] = word & 32767
			carry = word >> 15
		}
	}
	return wideNormalize(out)
}

func wideTwosLimb(value wideConstant, index, carry int) (int, int) {
	word := 0
	if index < len(value.words) {
		word = value.words[index]
	}
	if value.negative {
		word = (word ^ 32767) + carry
		return word & 32767, word >> 15
	}
	return word, 0
}
