package json

import (
	"unicode/utf16"
	"unicode/utf8"
)

// SyntaxError describes malformed JSON. Offset counts bytes read, starting at 1.
type SyntaxError struct {
	Offset  int64
	message string
}

func (e *SyntaxError) Error() string { return e.message }

// jsonValue retains number spelling and object member order. In particular,
// duplicate object members are not discarded before typed decoding.
type jsonValue struct {
	kind  byte
	text  string
	keys  []string
	items []jsonValue
}

type jsonParser struct {
	source string
	at     int
	err    error
}

// Valid reports whether data is exactly one complete JSON value, surrounded
// only by JSON whitespace.
func Valid(data []byte) bool {
	p := jsonParser{source: string(data)}
	p.space()
	p.value(0)
	p.space()
	return p.err == nil && p.at == len(p.source)
}

// parseJSON consumes one value without consuming subsequent values. Streaming
// decoding can retain the returned cursor; whole-document callers check EOF.
func parseJSON(data []byte) (jsonValue, int, error) {
	p := jsonParser{source: string(data)}
	p.space()
	value := p.value(0)
	return value, p.at, p.err
}

func (p *jsonParser) space() {
	for p.at < len(p.source) {
		b := p.source[p.at]
		if b != ' ' && b != '\t' && b != '\r' && b != '\n' {
			return
		}
		p.at++
	}
}

func (p *jsonParser) fail(message string) {
	if p.err != nil {
		return
	}
	offset := p.at + 1
	if p.at >= len(p.source) {
		offset = len(p.source)
		message = "unexpected end of JSON input"
	}
	p.err = &SyntaxError{Offset: int64(offset), message: message}
}

func (p *jsonParser) take(want byte) bool {
	if p.at >= len(p.source) || p.source[p.at] != want {
		p.fail("invalid character in JSON input")
		return false
	}
	p.at++
	return true
}

func (p *jsonParser) value(depth int) jsonValue {
	var out jsonValue
	if p.err != nil {
		return out
	}
	if p.at >= len(p.source) {
		p.fail("")
		return out
	}
	b := p.source[p.at]
	if b == '{' || b == '[' {
		if depth >= 10000 {
			p.fail("invalid character: exceeded max depth")
			return out
		}
	}
	switch b {
	case 'n':
		out.kind = '0'
		p.literal("null")
	case 't':
		out.kind = 't'
		p.literal("true")
	case 'f':
		out.kind = 'f'
		p.literal("false")
	case '"':
		out.kind = 's'
		out.text = p.string()
	case '[':
		out.kind = 'a'
		p.at++
		p.space()
		if p.at < len(p.source) && p.source[p.at] == ']' {
			p.at++
			return out
		}
		for p.err == nil {
			out.items = append(out.items, p.value(depth+1))
			p.space()
			if p.at < len(p.source) && p.source[p.at] == ']' {
				p.at++
				break
			}
			if !p.take(',') {
				break
			}
			p.space()
		}
	case '{':
		out.kind = 'o'
		p.at++
		p.space()
		if p.at < len(p.source) && p.source[p.at] == '}' {
			p.at++
			return out
		}
		for p.err == nil {
			key := p.string()
			p.space()
			if !p.take(':') {
				break
			}
			p.space()
			out.keys = append(out.keys, key)
			out.items = append(out.items, p.value(depth+1))
			p.space()
			if p.at < len(p.source) && p.source[p.at] == '}' {
				p.at++
				break
			}
			if !p.take(',') {
				break
			}
			p.space()
		}
	default:
		if b == '-' || b >= '0' && b <= '9' {
			out.kind = 'n'
			out.text = p.number()
		} else {
			p.fail("invalid character looking for beginning of value")
		}
	}
	return out
}

func (p *jsonParser) literal(text string) {
	for i := 0; i < len(text); i++ {
		if !p.take(text[i]) {
			return
		}
	}
}

func (p *jsonParser) number() string {
	start := p.at
	if p.source[p.at] == '-' {
		p.at++
	}
	if p.at >= len(p.source) {
		p.fail("")
		return ""
	}
	if p.source[p.at] == '0' {
		p.at++
	} else {
		if p.source[p.at] < '1' || p.source[p.at] > '9' {
			p.fail("invalid character in number")
			return ""
		}
		p.digits()
	}
	if p.at < len(p.source) && p.source[p.at] == '.' {
		p.at++
		before := p.at
		p.digits()
		if before == p.at {
			p.fail("invalid character after decimal point")
			return ""
		}
	}
	if p.at < len(p.source) && (p.source[p.at] == 'e' || p.source[p.at] == 'E') {
		p.at++
		if p.at < len(p.source) && (p.source[p.at] == '+' || p.source[p.at] == '-') {
			p.at++
		}
		before := p.at
		p.digits()
		if before == p.at {
			p.fail("invalid character in exponent")
			return ""
		}
	}
	return p.source[start:p.at]
}

func (p *jsonParser) digits() {
	for p.at < len(p.source) && p.source[p.at] >= '0' && p.source[p.at] <= '9' {
		p.at++
	}
}

func (p *jsonParser) hexRune() rune {
	var value rune
	for i := 0; i < 4; i++ {
		if p.at >= len(p.source) {
			p.fail("")
			return 0
		}
		b := p.source[p.at]
		var digit byte
		if b >= '0' && b <= '9' {
			digit = b - '0'
		} else if b >= 'a' && b <= 'f' {
			digit = b - 'a' + 10
		} else if b >= 'A' && b <= 'F' {
			digit = b - 'A' + 10
		} else {
			p.fail("invalid character in Unicode escape")
			return 0
		}
		value = value*16 + rune(digit)
		p.at++
	}
	return value
}

func (p *jsonParser) string() string {
	if !p.take('"') {
		return ""
	}
	var out []byte
	for p.at < len(p.source) && p.err == nil {
		b := p.source[p.at]
		p.at++
		if b == '"' {
			return string(out)
		}
		if b < 32 {
			p.at--
			p.fail("invalid control character in string")
			return ""
		}
		if b == '\\' {
			if p.at >= len(p.source) {
				p.fail("")
				return ""
			}
			escape := p.source[p.at]
			p.at++
			switch escape {
			case '"', '\\', '/':
				out = append(out, escape)
			case 'b':
				out = append(out, '\b')
			case 'f':
				out = append(out, '\f')
			case 'n':
				out = append(out, '\n')
			case 'r':
				out = append(out, '\r')
			case 't':
				out = append(out, '\t')
			case 'u':
				r := p.hexRune()
				if p.err != nil {
					return ""
				}
				if r >= 0xd800 && r <= 0xdbff {
					if p.at+6 <= len(p.source) && p.source[p.at:p.at+2] == "\\u" {
						saved := p.at
						p.at += 2
						second := p.hexRune()
						if p.err != nil {
							return ""
						}
						if second >= 0xdc00 && second <= 0xdfff {
							r = utf16.DecodeRune(r, second)
						} else {
							p.at = saved
							r = utf8.RuneError
						}
					} else {
						r = utf8.RuneError
					}
				} else if r >= 0xdc00 && r <= 0xdfff {
					r = utf8.RuneError
				}
				out = append(out, string(r)...)
			default:
				p.at--
				p.fail("invalid character in string escape")
				return ""
			}
		} else if b < 128 {
			out = append(out, b)
		} else {
			p.at--
			r, size := utf8.DecodeRuneInString(p.source[p.at:])
			out = append(out, string(r)...)
			p.at += size
		}
	}
	if p.err == nil {
		p.fail("")
	}
	return ""
}
