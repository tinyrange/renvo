package syntax

// AppendQuotedStringLiteral canonicalizes an interpreted string directly into
// its destination, without retaining a decoded byte buffer or string copy.
func AppendQuotedStringLiteral(out []byte, src []byte, tok Token) ([]byte, bool) {
	start, end := int(tok.Start), int(tok.End)
	if tok.KindLine&255 != TokenString || start < 0 || end-start < 2 || end > len(src) || src[start] != '"' || src[end-1] != '"' {
		return out, false
	}
	out = append(out, '"')
	for i := start + 1; i < end-1; {
		if src[i] != '\\' {
			out = appendQuotedByte(out, src[i])
			i++
			continue
		}
		next, value, unicode, ok := stringEscapeValue(src, i, end-1)
		if !ok {
			return out, false
		}
		if !unicode || value < 0x80 {
			out = appendQuotedByte(out, byte(value))
		} else if value < 0x800 {
			out = appendQuotedByte(out, byte(0xc0|value>>6))
			out = appendQuotedByte(out, byte(0x80|value&0x3f))
		} else if value < 0x10000 {
			out = appendQuotedByte(out, byte(0xe0|value>>12))
			out = appendQuotedByte(out, byte(0x80|value>>6&0x3f))
			out = appendQuotedByte(out, byte(0x80|value&0x3f))
		} else {
			out = appendQuotedByte(out, byte(0xf0|value>>18))
			out = appendQuotedByte(out, byte(0x80|value>>12&0x3f))
			out = appendQuotedByte(out, byte(0x80|value>>6&0x3f))
			out = appendQuotedByte(out, byte(0x80|value&0x3f))
		}
		i = next
	}
	return append(out, '"'), true
}

func appendQuotedByte(out []byte, ch byte) []byte {
	const hex = "0123456789abcdef"
	if ch == '"' || ch == '\\' {
		return append(out, '\\', ch)
	}
	if ch < 32 || ch >= 127 {
		return append(out, '\\', 'x', hex[ch/16], hex[ch%16])
	}
	return append(out, ch)
}
