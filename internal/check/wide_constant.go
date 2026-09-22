package check

// Magnitudes use 15-bit limbs so multiplication and carry arithmetic are exact
// even when the checking compiler itself has 32-bit ints. Values are immutable:
// operations allocate their result instead of altering shared constant limbs.
type wideConstant struct {
	words    []int
	negative bool
	ok       bool
}

func wideNormalize(value wideConstant) wideConstant {
	for len(value.words) > 0 && value.words[len(value.words)-1] == 0 {
		value.words = value.words[:len(value.words)-1]
	}
	if len(value.words) == 0 {
		value.negative = false
	}
	return value
}

func wideSmall(value int) wideConstant {
	out := wideConstant{ok: true}
	if value < 0 {
		out.negative = true
	}
	magnitude := uint(value)
	if value < 0 {
		magnitude = uint(-(value + 1)) + 1
	}
	for magnitude != 0 {
		out.words = append(out.words, int(magnitude&32767))
		magnitude >>= 15
	}
	return out
}

func wideMagnitudeCompare(left wideConstant, right wideConstant) int {
	if len(left.words) < len(right.words) {
		return -1
	}
	if len(left.words) > len(right.words) {
		return 1
	}
	for i := len(left.words) - 1; i >= 0; i-- {
		if left.words[i] < right.words[i] {
			return -1
		}
		if left.words[i] > right.words[i] {
			return 1
		}
	}
	return 0
}

func wideNegate(value wideConstant) wideConstant {
	if len(value.words) > 0 {
		value.negative = !value.negative
	}
	return value
}

func wideAdd(left wideConstant, right wideConstant) wideConstant {
	if !left.ok || !right.ok {
		return wideConstant{}
	}
	if left.negative != right.negative {
		if wideMagnitudeCompare(left, right) < 0 {
			left, right = right, left
		}
		out := wideConstant{words: make([]int, len(left.words)), negative: left.negative, ok: true}
		borrow := 0
		for i := 0; i < len(left.words); i++ {
			value := left.words[i] - borrow
			if i < len(right.words) {
				value -= right.words[i]
			}
			borrow = 0
			if value < 0 {
				value += 32768
				borrow = 1
			}
			out.words[i] = value
		}
		return wideNormalize(out)
	}
	size := len(left.words)
	if len(right.words) > size {
		size = len(right.words)
	}
	out := wideConstant{words: make([]int, size+1), negative: left.negative, ok: true}
	carry := 0
	for i := 0; i < size; i++ {
		value := carry
		if i < len(left.words) {
			value += left.words[i]
		}
		if i < len(right.words) {
			value += right.words[i]
		}
		out.words[i] = value & 32767
		carry = value >> 15
	}
	out.words[size] = carry
	return wideNormalize(out)
}

func wideMultiply(left wideConstant, right wideConstant) wideConstant {
	if !left.ok || !right.ok {
		return wideConstant{}
	}
	out := wideConstant{words: make([]int, len(left.words)+len(right.words)), negative: left.negative != right.negative, ok: true}
	for i := 0; i < len(left.words); i++ {
		carry := 0
		for j := 0; j < len(right.words); j++ {
			value := out.words[i+j] + left.words[i]*right.words[j] + carry
			out.words[i+j] = value & 32767
			carry = value >> 15
		}
		if len(right.words) > 0 {
			out.words[i+len(right.words)] = carry
		}
	}
	return wideNormalize(out)
}

func wideShift(value wideConstant, count int, left bool) wideConstant {
	if !value.ok || count < 0 {
		return wideConstant{}
	}
	if len(value.words) == 0 {
		return value
	}
	whole, part := count/15, count%15
	if left {
		out := wideConstant{words: make([]int, len(value.words)+whole+1), negative: value.negative, ok: true}
		carry := 0
		for i := 0; i < len(value.words); i++ {
			word := value.words[i]<<uint(part) | carry
			out.words[i+whole] = word & 32767
			carry = word >> 15
		}
		out.words[len(value.words)+whole] = carry
		return wideNormalize(out)
	}
	if whole >= len(value.words) {
		if value.negative {
			return wideSmall(-1)
		}
		return wideSmall(0)
	}
	out := wideConstant{words: make([]int, len(value.words)-whole), negative: value.negative, ok: true}
	lost := false
	for i := 0; i < whole; i++ {
		if value.words[i] != 0 {
			lost = true
		}
	}
	if value.words[whole]&((1<<uint(part))-1) != 0 {
		lost = true
	}
	for i := 0; i < len(out.words); i++ {
		out.words[i] = value.words[i+whole] >> uint(part)
		if part != 0 && i+whole+1 < len(value.words) {
			out.words[i] |= (value.words[i+whole+1] << uint(15-part)) & 32767
		}
	}
	out = wideNormalize(out)
	if value.negative && lost {
		return wideAdd(out, wideSmall(-1))
	}
	return out
}

func wideInt(value wideConstant) (int, bool) {
	if !value.ok {
		return 0, false
	}
	limit := uint(^uint(0) >> 1)
	if value.negative {
		limit++
	}
	magnitude := uint(0)
	for i := len(value.words) - 1; i >= 0; i-- {
		word := uint(value.words[i])
		if magnitude > (limit-word)/32768 {
			return 0, false
		}
		magnitude = magnitude*32768 + word
	}
	if value.negative {
		return -int(magnitude), true
	}
	return int(magnitude), true
}
