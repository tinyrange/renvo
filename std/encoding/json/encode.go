// Package json encodes JSON using Renvo's explicitly opted-in reflection API.
package json

import (
	"encoding/base64"
	"errors"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

type fieldInfo struct{ Name, Tag string }
type collectionInfo struct {
	Kind         string
	Nil          bool
	Keys, Values []any
	Element      any
	Key          any
}

// Marshal encodes v. Named structs must explicitly opt into Renvo reflection.
func Marshal(v any) ([]byte, error) {
	e := encoder{}
	e.value(v, 0)
	if e.err != nil {
		return nil, e.err
	}
	return e.data, nil
}

type encoder struct {
	data []byte
	err  error
}

func (e *encoder) value(v any, depth int) {
	if e.err != nil {
		return
	}
	if depth > 1000 {
		e.err = errors.New("json: nesting limit exceeded (possibly a cycle)")
		return
	}
	if v == nil {
		e.data = append(e.data, "null"...)
		return
	}
	if scalar, ok := scalarValue(v); ok {
		v = scalar
	}
	switch x := v.(type) {
	case string:
		e.string(x)
		return
	case bool:
		e.data = append(e.data, strconv.FormatBool(x)...)
		return
	case int:
		e.integer(int64(x))
		return
	case int8:
		e.integer(int64(x))
		return
	case int16:
		e.integer(int64(x))
		return
	case int32:
		e.integer(int64(x))
		return
	case int64:
		e.integer(x)
		return
	case uint:
		e.unsigned(uint64(x))
		return
	case uint8:
		e.unsigned(uint64(x))
		return
	case uint16:
		e.unsigned(uint64(x))
		return
	case uint32:
		e.unsigned(uint64(x))
		return
	case uint64:
		e.unsigned(x)
		return
	case uintptr:
		e.unsigned(uint64(x))
		return
	case float32:
		e.floating(float64(x), 32)
		return
	case float64:
		e.floating(x, 64)
		return
	case []byte:
		if x == nil {
			e.data = append(e.data, "null"...)
		} else {
			e.string(base64.StdEncoding.EncodeToString(x))
		}
		return
	}
	if c, ok := inspectCollection(v); ok {
		if c.Nil {
			e.data = append(e.data, "null"...)
			return
		}
		if c.Kind == "pointer" {
			e.value(c.Values[0], depth+1)
			return
		}
		if c.Kind == "map" {
			e.mapping(c, depth+1)
			return
		}
		if c.Kind == "slice" {
			elem := c.Element
			if normalized, ok := scalarValue(elem); ok {
				elem = normalized
			}
			if _, ok := elem.(uint8); ok {
				bytes := make([]byte, len(c.Values))
				for i, item := range c.Values {
					if normalized, ok := scalarValue(item); ok {
						item = normalized
					}
					b, ok := item.(uint8)
					if !ok {
						e.err = errors.New("json: invalid byte slice reflection")
						return
					}
					bytes[i] = b
				}
				e.string(base64.StdEncoding.EncodeToString(bytes))
				return
			}
		}
		e.data = append(e.data, '[')
		for i, item := range c.Values {
			if i > 0 {
				e.data = append(e.data, ',')
			}
			e.value(item, depth+1)
		}
		e.data = append(e.data, ']')
		return
	}
	if fields, ok := describeFields(v); ok {
		e.data = append(e.data, '{')
		count := 0
		for i, field := range fields {
			name, options, skip := jsonFieldTag(field.Tag)
			if skip {
				continue
			}
			if name == "" {
				name = field.Name
			}
			item, ok := readField(v, i)
			if !ok {
				e.err = errors.New("json: unavailable reflected field")
				return
			}
			if tagOption(options, "omitempty") && emptyValue(item) {
				continue
			}
			if count > 0 {
				e.data = append(e.data, ',')
			}
			count++
			e.string(name)
			e.data = append(e.data, ':')
			if tagOption(options, "string") && quoteable(item) {
				nested := encoder{}
				nested.value(item, depth+1)
				if nested.err != nil {
					e.err = nested.err
					return
				}
				e.string(string(nested.data))
			} else {
				e.value(item, depth+1)
			}
		}
		e.data = append(e.data, '}')
		return
	}
	e.err = errors.New("json: unsupported type or struct missing //renvo:reflect")
}

func (e *encoder) integer(n int64)   { e.data = append(e.data, strconv.FormatInt(n, 10)...) }
func (e *encoder) unsigned(n uint64) { e.data = append(e.data, strconv.FormatUint(n, 10)...) }
func (e *encoder) floating(n float64, bits int) {
	if math.IsNaN(n) || math.IsInf(n, 0) {
		e.err = errors.New("json: unsupported non-finite number")
		return
	}
	format := byte('f')
	abs := math.Abs(n)
	exponential := abs != 0 && (abs < 1e-6 || abs >= 1e21)
	if bits == 32 {
		exponential = abs != 0 && (float32(abs) < float32(1e-6) || float32(abs) >= float32(1e21))
	}
	if exponential {
		format = 'e'
	}
	text := strconv.FormatFloat(n, format, -1, bits)
	if at := strings.Index(text, "e-0"); at >= 0 {
		text = text[:at+2] + text[at+3:]
	}
	if at := strings.Index(text, "e+0"); at >= 0 {
		text = text[:at+2] + text[at+3:]
	}
	e.data = append(e.data, text...)
}

func (e *encoder) string(s string) {
	const hex = "0123456789abcdef"
	e.data = append(e.data, '"')
	for i := 0; i < len(s); {
		b := s[i]
		if b < 128 {
			i++
			switch b {
			case '"', '\\':
				e.data = append(e.data, '\\', b)
			case '\b':
				e.data = append(e.data, '\\', 'b')
			case '\f':
				e.data = append(e.data, '\\', 'f')
			case '\n':
				e.data = append(e.data, '\\', 'n')
			case '\r':
				e.data = append(e.data, '\\', 'r')
			case '\t':
				e.data = append(e.data, '\\', 't')
			default:
				if b < 32 || b == '<' || b == '>' || b == '&' {
					e.data = append(e.data, '\\', 'u', '0', '0', hex[b>>4], hex[b&15])
				} else {
					e.data = append(e.data, b)
				}
			}
			continue
		}
		r, width := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && width == 1 {
			e.data = append(e.data, "\\ufffd"...)
			i++
			continue
		}
		if r == 0x2028 || r == 0x2029 {
			e.data = append(e.data, "\\u202"...)
			e.data = append(e.data, hex[r&15])
		} else {
			e.data = append(e.data, s[i:i+width]...)
		}
		i += width
	}
	e.data = append(e.data, '"')
}

func (e *encoder) mapping(c collectionInfo, depth int) {
	names := make([]string, len(c.Keys))
	order := make([]int, len(c.Keys))
	for i, key := range c.Keys {
		if normalized, ok := scalarValue(key); ok {
			key = normalized
		}
		switch k := key.(type) {
		case string:
			names[i] = k
		case int:
			names[i] = strconv.FormatInt(int64(k), 10)
		case int8:
			names[i] = strconv.FormatInt(int64(k), 10)
		case int16:
			names[i] = strconv.FormatInt(int64(k), 10)
		case int32:
			names[i] = strconv.FormatInt(int64(k), 10)
		case int64:
			names[i] = strconv.FormatInt(k, 10)
		case uint:
			names[i] = strconv.FormatUint(uint64(k), 10)
		case uint8:
			names[i] = strconv.FormatUint(uint64(k), 10)
		case uint16:
			names[i] = strconv.FormatUint(uint64(k), 10)
		case uint32:
			names[i] = strconv.FormatUint(uint64(k), 10)
		case uint64:
			names[i] = strconv.FormatUint(k, 10)
		case uintptr:
			names[i] = strconv.FormatUint(uint64(k), 10)
		default:
			e.err = errors.New("json: unsupported map key type")
			return
		}
		order[i] = i
	}
	for i := 1; i < len(order); i++ {
		for j := i; j > 0 && names[order[j]] < names[order[j-1]]; j-- {
			order[j], order[j-1] = order[j-1], order[j]
		}
	}
	e.data = append(e.data, '{')
	for i, index := range order {
		if i > 0 {
			e.data = append(e.data, ',')
		}
		e.string(names[index])
		e.data = append(e.data, ':')
		e.value(c.Values[index], depth+1)
	}
	e.data = append(e.data, '}')
}

func quoteable(v any) bool {
	for depth := 0; depth < 1000; depth++ {
		if normalized, ok := scalarValue(v); ok {
			v = normalized
		}
		switch v.(type) {
		case string, bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, uintptr, float32, float64:
			return true
		}
		if c, ok := inspectCollection(v); ok && c.Kind == "pointer" && !c.Nil {
			v = c.Values[0]
			continue
		}
		return false
	}
	return false
}

func emptyValue(v any) bool {
	if v == nil {
		return true
	}
	if normalized, ok := scalarValue(v); ok {
		v = normalized
	}
	switch x := v.(type) {
	case string:
		return x == ""
	case bool:
		return !x
	case int:
		return x == 0
	case int8:
		return x == 0
	case int16:
		return x == 0
	case int32:
		return x == 0
	case int64:
		return x == 0
	case uint:
		return x == 0
	case uint8:
		return x == 0
	case uint16:
		return x == 0
	case uint32:
		return x == 0
	case uint64:
		return x == 0
	case uintptr:
		return x == 0
	case float32:
		return x == 0
	case float64:
		return x == 0
	case []byte:
		return len(x) == 0
	}
	if c, ok := inspectCollection(v); ok {
		if c.Kind == "pointer" {
			return c.Nil
		}
		return len(c.Values) == 0
	}
	return false
}

func jsonFieldTag(tag string) (string, string, bool) {
	for len(tag) > 0 {
		tag = strings.TrimLeft(tag, " ")
		at := strings.Index(tag, ":\"")
		if at < 1 {
			break
		}
		key := tag[:at]
		end := at + 2
		for end < len(tag) {
			if tag[end] == '\\' {
				end += 2
				continue
			}
			if tag[end] == '"' {
				break
			}
			end++
		}
		if end >= len(tag) {
			break
		}
		if key == "json" {
			value, err := strconv.Unquote(tag[at+1 : end+1])
			if err != nil {
				return "", "", false
			}
			if comma := strings.Index(value, ","); comma >= 0 {
				return value[:comma], value[comma+1:], false
			}
			return value, "", value == "-"
		}
		tag = tag[end+1:]
	}
	return "", "", false
}

func tagOption(options, wanted string) bool {
	for len(options) > 0 {
		at := strings.Index(options, ",")
		if at < 0 {
			return options == wanted
		}
		if options[:at] == wanted {
			return true
		}
		options = options[at+1:]
	}
	return false
}
