// Package managedheap contains the allocation and reclamation core for Renvo's
// opt-in desktop memory strategy. Generated programs use a compiler-owned
// backend-subset adaptation; see README.md for its integration contract.
package managedheap

// ParseLimit parses a positive byte budget, optionally suffixed with KiB, MiB,
// GiB or TiB. Parsing is allocation-free and never wraps on overflow. Whitespace,
// signs, fractional values, zero, and ambiguous units are deliberately rejected.
func ParseLimit(text string) (uint64, bool) {
	end := len(text)
	multiplier := uint64(1)
	if end >= 3 {
		suffix := text[end-3:]
		if suffix == "KiB" {
			multiplier = 1 << 10
		} else if suffix == "MiB" {
			multiplier = 1 << 20
		} else if suffix == "GiB" {
			multiplier = 1 << 30
		} else if suffix == "TiB" {
			multiplier = 1 << 40
		}
		if multiplier != 1 {
			end -= 3
		}
	}
	if end == 0 {
		return 0, false
	}
	maximum := ^uint64(0) / multiplier
	value := uint64(0)
	for i := 0; i < end; i++ {
		c := text[i]
		if c < '0' || c > '9' {
			return 0, false
		}
		digit := uint64(c - '0')
		if value > maximum/10 || value == maximum/10 && digit > maximum%10 {
			return 0, false
		}
		value = value*10 + digit
	}
	return value * multiplier, value != 0
}

// ResolveLimit applies startup precedence. Zero target/compiled budgets mean
// unspecified, not a zero-byte heap. A present but empty environment value is an
// error. This function does not read the environment: only a full-manager target
// adapter should do that, once, before initialization.
func ResolveLimit(target, compiled uint64, environment string, present bool) (uint64, bool) {
	if present {
		return ParseLimit(environment)
	}
	if compiled != 0 {
		return compiled, true
	}
	return target, true
}
