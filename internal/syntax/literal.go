package syntax

func StringLiteralValue(src []byte, tok Token) (string, bool) {
	if tok.KindLine&255 != TokenString || tok.Start < 0 || tok.End-tok.Start < 2 || tok.End > len(src) {
		return "", false
	}
	quote := src[tok.Start]
	if quote == '`' {
		if src[tok.End-1] != '`' {
			return "", false
		}
		return string(src[tok.Start+1 : tok.End-1]), true
	}
	if quote != '"' || src[tok.End-1] != '"' {
		return "", false
	}
	out := make([]byte, 0, tok.End-tok.Start-2)
	i := tok.Start + 1
	for i < tok.End-1 {
		c := src[i]
		if c != '\\' {
			out = append(out, c)
			i++
			continue
		}
		i++
		if i >= tok.End-1 {
			return "", false
		}
		esc := src[i]
		if esc == 'n' {
			out = append(out, '\n')
		} else if esc == 'r' {
			out = append(out, '\r')
		} else if esc == 't' {
			out = append(out, '\t')
		} else if esc == '\\' || esc == '"' {
			out = append(out, esc)
		} else if esc == 'x' {
			if i+2 >= tok.End-1 {
				return "", false
			}
			hi, okHi := hexValue(src[i+1])
			lo, okLo := hexValue(src[i+2])
			if !okHi || !okLo {
				return "", false
			}
			out = append(out, byte(hi*16+lo))
			i += 2
		} else {
			return "", false
		}
		i++
	}
	return string(out), true
}

func hexValue(c byte) (int, bool) {
	if c >= '0' && c <= '9' {
		return int(c - '0'), true
	}
	if c >= 'a' && c <= 'f' {
		return int(c-'a') + 10, true
	}
	if c >= 'A' && c <= 'F' {
		return int(c-'A') + 10, true
	}
	return 0, false
}
