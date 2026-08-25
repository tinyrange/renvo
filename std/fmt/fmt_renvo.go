//go:build renvo && !i8086

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
		value := a[i]
		text, isString := formatValue(value, 'v')
		if i > 0 && !prevString && !isString {
			out = out + " "
		}
		out = out + text
		prevString = isString
	}
	return out
}

// Sprintf formats according to format and returns the resulting string. Renvo
// supports %v, %s, %d, %x, %q, %t, and %%.
func Sprintf(format string, a ...interface{}) string {
	var out []byte
	arg := 0
	for i := 0; i < len(format); i++ {
		c := format[i]
		if c != '%' {
			out = append(out, c)
			continue
		}
		if i+1 >= len(format) {
			out = append(out, '%')
			continue
		}
		i++
		verb := format[i]
		if verb == '%' {
			out = append(out, '%')
			continue
		}
		if arg >= len(a) {
			out = appendString(out, "%!")
			out = append(out, verb)
			out = appendString(out, "(MISSING)")
			continue
		}
		value := a[arg]
		text, _ := formatValue(value, verb)
		out = appendString(out, text)
		arg++
	}
	return string(out)
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

func formatValue(v interface{}, verb byte) (string, bool) {
	switch v.(type) {
	case string:
		value := v.(string)
		if verb == 'q' {
			return quote(value), true
		}
		if verb == 'x' {
			return hexBytes([]byte(value)), true
		}
		if verb == 's' || verb == 'v' {
			return value, true
		}
	case bool:
		value := v.(bool)
		if verb == 't' || verb == 'v' {
			if value {
				return "true", false
			}
			return "false", false
		}
	case int:
		value := v.(int)
		if verb == 'x' {
			return formatInt(int64(value), 16), false
		}
		if verb == 'd' || verb == 'v' {
			return formatInt(int64(value), 10), false
		}
	case int8:
		value := v.(int8)
		if verb == 'x' {
			return formatInt(int64(value), 16), false
		}
		if verb == 'd' || verb == 'v' {
			return formatInt(int64(value), 10), false
		}
	case int16:
		value := v.(int16)
		if verb == 'x' {
			return formatInt(int64(value), 16), false
		}
		if verb == 'd' || verb == 'v' {
			return formatInt(int64(value), 10), false
		}
	case int32:
		value := v.(int32)
		if verb == 'x' {
			return formatInt(int64(value), 16), false
		}
		if verb == 'd' || verb == 'v' {
			return formatInt(int64(value), 10), false
		}
	case int64:
		value := v.(int64)
		if verb == 'x' {
			return formatInt(value, 16), false
		}
		if verb == 'd' || verb == 'v' {
			return formatInt(value, 10), false
		}
	case uint:
		value := v.(uint)
		if verb == 'x' {
			return formatUint(uint64(value), 16), false
		}
		if verb == 'd' || verb == 'v' {
			return formatUint(uint64(value), 10), false
		}
	case uint8:
		value := v.(uint8)
		if verb == 'x' {
			return formatUint(uint64(value), 16), false
		}
		if verb == 'd' || verb == 'v' {
			return formatUint(uint64(value), 10), false
		}
	case uint16:
		value := v.(uint16)
		if verb == 'x' {
			return formatUint(uint64(value), 16), false
		}
		if verb == 'd' || verb == 'v' {
			return formatUint(uint64(value), 10), false
		}
	case uint32:
		value := v.(uint32)
		if verb == 'x' {
			return formatUint(uint64(value), 16), false
		}
		if verb == 'd' || verb == 'v' {
			return formatUint(uint64(value), 10), false
		}
	case uint64:
		value := v.(uint64)
		if verb == 'x' {
			return formatUint(value, 16), false
		}
		if verb == 'd' || verb == 'v' {
			return formatUint(value, 10), false
		}
	case []byte:
		value := v.([]byte)
		if verb == 'x' {
			return hexBytes(value), true
		}
		if verb == 's' || verb == 'v' {
			return string(value), true
		}
	case error:
		value := v.(error)
		if verb == 's' || verb == 'v' {
			return value.Error(), true
		}
	}
	var out []byte
	out = appendString(out, "%!")
	out = append(out, verb)
	return string(out), false
}

func writeString(w Writer, s string) (int, error) {
	count, err := w.Write([]byte(s))
	return count, err
}

func hexBytes(b []byte) string {
	var out []byte
	for i := 0; i < len(b); i++ {
		out = append(out, "0123456789abcdef"[b[i]>>4])
		out = append(out, "0123456789abcdef"[b[i]&15])
	}
	return string(out)
}

func quote(s string) string {
	var out []byte
	out = append(out, '"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' || c == '"' {
			out = append(out, '\\')
			out = append(out, c)
		} else if c == '\n' {
			out = appendString(out, "\\n")
		} else {
			out = append(out, c)
		}
	}
	out = append(out, '"')
	return string(out)
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
	var reversed []byte
	b := uint64(base)
	for v > 0 {
		d := v % b
		if d < 10 {
			reversed = append(reversed, byte('0'+d))
		} else {
			reversed = append(reversed, byte('a'+d-10))
		}
		v = v / b
	}
	var out []byte
	for i := len(reversed) - 1; i >= 0; i-- {
		out = append(out, reversed[i])
	}
	return string(out)
}

func appendString(out []byte, s string) []byte {
	for i := 0; i < len(s); i++ {
		out = append(out, s[i])
	}
	return out
}
