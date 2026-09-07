package big

const alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func (z *Int) SetString(text string, base int) (*Int, bool) {
	if base != 0 && (base < 2 || base > 62) {
		panic("invalid number base")
	}
	if len(text) == 0 {
		return nil, false
	}
	negative := false
	if text[0] == '-' || text[0] == '+' {
		negative = text[0] == '-'
		text = text[1:]
	}
	if len(text) == 0 {
		return nil, false
	}
	automatic := base == 0
	prefix := false
	if automatic {
		base = 10
		if len(text) > 1 && text[0] == '0' {
			base = 8
			if text[1] == 'x' || text[1] == 'X' {
				base = 16
				text = text[2:]
				prefix = true
			} else if text[1] == 'b' || text[1] == 'B' {
				base = 2
				text = text[2:]
				prefix = true
			} else if text[1] == 'o' || text[1] == 'O' {
				text = text[2:]
				prefix = true
			}
		}
	}
	var words []uint32
	previous := false
	for i := 0; i < len(text); i++ {
		c := text[i]
		if c == '_' && automatic {
			if (!previous && !(i == 0 && prefix)) || i+1 == len(text) {
				return nil, false
			}
			previous = false
			continue
		}
		digit := -1
		if c >= '0' && c <= '9' {
			digit = int(c - '0')
		} else if c >= 'a' && c <= 'z' {
			digit = int(c-'a') + 10
		} else if c >= 'A' && c <= 'Z' {
			digit = int(c-'A') + 10
			if base > 36 {
				digit += 26
			}
		}
		if digit < 0 || digit >= base {
			return nil, false
		}
		carry := uint64(digit)
		for j := 0; j < len(words); j++ {
			value := uint64(words[j])*uint64(base) + carry
			words[j] = uint32(value)
			carry = value >> 32
		}
		if carry != 0 {
			words = append(words, uint32(carry))
		}
		previous = true
	}
	if !previous {
		return nil, false
	}
	return z.put(words, negative), true
}
func (x *Int) Text(base int) string {
	if base < 2 || base > 62 {
		panic("invalid number base")
	}
	if x == nil {
		return "<nil>"
	}
	if len(x.words) == 0 {
		return "0"
	}
	words := clone(x.words)
	var digits []byte
	for len(words) > 0 {
		var remainder uint64
		for i := len(words) - 1; i >= 0; i-- {
			value := remainder<<32 | uint64(words[i])
			words[i] = uint32(value / uint64(base))
			remainder = value % uint64(base)
		}
		digits = append(digits, alphabet[int(remainder)])
		words = normalize(words)
	}
	if x.negative {
		digits = append(digits, '-')
	}
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	return string(digits)
}
func (x *Int) String() string { return x.Text(10) }
