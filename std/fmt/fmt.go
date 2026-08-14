//go:build !renvo

// Package fmt provides compact, allocation-conscious formatting for Renvo's
// supported scalar, string, byte-slice, Boolean, and error values.
package fmt

// Writer is the minimal byte sink accepted by the Fprint functions.
type Writer interface {
	Write(p []byte) (n int, err error)
}

// Print formats its operands, writes them to standard output, and returns the
// number of bytes written.
func Print(a ...interface{}) (int, error) {
	text := Sprint(a...)
	print(text)
	return len(text), nil
}

// Println formats its operands, writes them to standard output followed by a
// newline, and returns the number of bytes written.
func Println(a ...interface{}) (int, error) {
	text := Sprint(a...) + "\n"
	print(text)
	return len(text), nil
}

// Printf formats according to format, writes the result to standard output,
// and returns the number of bytes written.
func Printf(format string, a ...interface{}) (int, error) {
	text := Sprintf(format, a...)
	print(text)
	return len(text), nil
}

// Sprint formats its operands and returns the resulting string.
func Sprint(a ...interface{}) string {
	out := ""
	prevString := false
	for i := 0; i < len(a); i++ {
		text, isString := formatValue(a[i], "v")
		if i > 0 && !prevString && !isString {
			out += " "
		}
		out += text
		prevString = isString
	}
	return out
}

// Sprintf formats according to format and returns the resulting string. Renvo
// supports %v, %s, %d, %x, %q, %t, and %%.
func Sprintf(format string, a ...interface{}) string {
	out := ""
	arg := 0
	for i := 0; i < len(format); i++ {
		c := format[i]
		if c != '%' {
			out += string(c)
			continue
		}
		if i+1 >= len(format) {
			out += "%"
			continue
		}
		i++
		verb := string(format[i])
		if verb == "%" {
			out += "%"
			continue
		}
		if arg >= len(a) {
			out += "%!" + verb + "(MISSING)"
			continue
		}
		text, _ := formatValue(a[arg], verb)
		out += text
		arg++
	}
	return out
}

// Fprint formats its operands and writes them to w.
func Fprint(w Writer, a ...interface{}) (int, error) {
	return writeString(w, Sprint(a...))
}

// Fprintf formats according to format and writes the result to w.
func Fprintf(w Writer, format string, a ...interface{}) (int, error) {
	return writeString(w, Sprintf(format, a...))
}

// Fprintln formats its operands and writes them to w followed by a newline.
func Fprintln(w Writer, a ...interface{}) (int, error) {
	return writeString(w, Sprint(a...)+"\n")
}

func formatValue(v interface{}, verb string) (string, bool) {
	if verb == "s" {
		if s, ok := v.(string); ok {
			return s, true
		}
	}
	if verb == "q" {
		if s, ok := v.(string); ok {
			return quote(s), true
		}
	}
	if verb == "t" {
		if b, ok := v.(bool); ok {
			if b {
				return "true", false
			}
			return "false", false
		}
	}
	if verb == "d" || verb == "v" {
		switch x := v.(type) {
		case string:
			return x, true
		case bool:
			if x {
				return "true", false
			}
			return "false", false
		case int:
			return formatInt(int64(x), 10), false
		case int8:
			return formatInt(int64(x), 10), false
		case int16:
			return formatInt(int64(x), 10), false
		case int32:
			return formatInt(int64(x), 10), false
		case int64:
			return formatInt(x, 10), false
		case uint:
			return formatUint(uint64(x), 10), false
		case uint8:
			return formatUint(uint64(x), 10), false
		case uint16:
			return formatUint(uint64(x), 10), false
		case uint32:
			return formatUint(uint64(x), 10), false
		case uint64:
			return formatUint(x, 10), false
		case []byte:
			return string(x), true
		case error:
			return x.Error(), true
		}
	}
	if verb == "x" {
		switch x := v.(type) {
		case string:
			return hexBytes([]byte(x)), true
		case []byte:
			return hexBytes(x), true
		case int:
			return formatInt(int64(x), 16), false
		case int8:
			return formatInt(int64(x), 16), false
		case int16:
			return formatInt(int64(x), 16), false
		case int32:
			return formatInt(int64(x), 16), false
		case int64:
			return formatInt(x, 16), false
		case uint:
			return formatUint(uint64(x), 16), false
		case uint8:
			return formatUint(uint64(x), 16), false
		case uint16:
			return formatUint(uint64(x), 16), false
		case uint32:
			return formatUint(uint64(x), 16), false
		case uint64:
			return formatUint(x, 16), false
		}
	}
	return "%!" + verb, false
}

func writeString(w Writer, s string) (int, error) {
	return w.Write([]byte(s))
}

func formatInt(v int64, base int) string {
	if v < 0 {
		return "-" + formatUint(uint64(-v), base)
	}
	return formatUint(uint64(v), base)
}

func formatUint(v uint64, base int) string {
	if v == 0 {
		return "0"
	}
	var buf [65]byte
	pos := len(buf)
	b := uint64(base)
	for v > 0 {
		d := v % b
		pos--
		if d < 10 {
			buf[pos] = byte('0' + d)
		} else {
			buf[pos] = byte('a' + d - 10)
		}
		v = v / b
	}
	return string(buf[pos:])
}

func hexBytes(b []byte) string {
	out := ""
	for i := 0; i < len(b); i++ {
		out += string("0123456789abcdef"[b[i]>>4])
		out += string("0123456789abcdef"[b[i]&15])
	}
	return out
}

func quote(s string) string {
	out := "\""
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' || c == '"' {
			out += "\\" + string(c)
		} else if c == '\n' {
			out += "\\n"
		} else {
			out += string(c)
		}
	}
	return out + "\""
}
