package check

import "renvo.dev/internal/syntax"

func literalIntegerOverflows(file syntax.File, start int, end int, typ string) bool {
	start, end = trimDeclSpan(file, start, end)
	start, end = stripOuterParens(&file, start, end)
	negative := false
	if start < end && (tokenTextIs(&file, start, "-") || tokenTextIs(&file, start, "+")) {
		negative = tokenTextIs(&file, start, "-")
		start++
	}
	bits := 0
	signed := true
	if typ == "byte" || typ == "uint8" {
		bits = 8
		signed = false
	} else if typ == "int8" {
		bits = 8
	} else if typ == "uint16" {
		bits = 16
		signed = false
	} else if typ == "int16" {
		bits = 16
	} else if typ == "uint32" {
		bits = 32
		signed = false
	} else if typ == "int32" || typ == "rune" {
		bits = 32
	} else if typ == "uint64" {
		bits = 64
		signed = false
	} else if typ == "int64" {
		bits = 64
	}
	if bits == 0 || end-start != 1 || file.Tokens[start].KindLine&255 != syntax.TokenNumber {
		return false
	}
	text := tokenString(&file, start)
	base := uint64(10)
	i := 0
	if len(text) > 1 && text[0] == '0' {
		base = 8
		i = 1
		if text[1] == 'x' || text[1] == 'X' {
			base = 16
			i = 2
		} else if text[1] == 'b' || text[1] == 'B' {
			base = 2
			i = 2
		} else if text[1] == 'o' || text[1] == 'O' {
			i = 2
		}
	}
	limit := ^uint64(0)
	if bits < 64 {
		limit = uint64(1)<<uint(bits) - 1
	}
	if signed {
		limit >>= 1
		if negative {
			limit++
		}
	}
	value := uint64(0)
	overflow := false
	for ; i < len(text); i++ {
		c := text[i]
		if c == '_' {
			continue
		}
		digit := uint64(99)
		if c >= '0' && c <= '9' {
			digit = uint64(c - '0')
		} else if c >= 'a' && c <= 'f' {
			digit = uint64(c-'a') + 10
		} else if c >= 'A' && c <= 'F' {
			digit = uint64(c-'A') + 10
		}
		if digit >= base {
			return false
		}
		if value > limit/base || value == limit/base && digit > limit%base {
			overflow = true
		} else if !overflow {
			value = value*base + digit
		}
	}
	return overflow || negative && !signed && value != 0
}
