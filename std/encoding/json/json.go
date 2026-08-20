package json

import (
	"errors"
	"io"
	"reflect"
	"sort"
	"strconv"
)

type RawMessage []byte

type Marshaler interface{ MarshalJSON() ([]byte, error) }
type Unmarshaler interface{ UnmarshalJSON([]byte) error }

type SyntaxError struct{ Offset int64 }

func (e *SyntaxError) Error() string { return "invalid character in JSON" }

func (m RawMessage) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	if !Valid(m) {
		return nil, errors.New("json: invalid RawMessage")
	}
	out := make([]byte, len(m))
	copy(out, m)
	return out, nil
}
func (m *RawMessage) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.New("json.RawMessage: UnmarshalJSON on nil pointer")
	}
	*m = append((*m)[:0], data...)
	return nil
}
func Marshal(v any) ([]byte, error) { return appendJSON(nil, v) }

func appendJSON(out []byte, v any) ([]byte, error) {
	if v == nil {
		return append(out, "null"...), nil
	}
	if m, ok := v.(Marshaler); ok {
		b, err := m.MarshalJSON()
		if err != nil {
			return nil, err
		}
		if !Valid(b) {
			return nil, errors.New("json: invalid Marshaler output")
		}
		return append(out, b...), nil
	}
	switch x := v.(type) {
	case bool:
		if x {
			return append(out, "true"...), nil
		}
		return append(out, "false"...), nil
	case string:
		return append(out, quote(x)...), nil
	case int:
		return append(out, strconv.Itoa(x)...), nil
	case int8:
		return append(out, strconv.FormatInt(int64(x), 10)...), nil
	case int16:
		return append(out, strconv.FormatInt(int64(x), 10)...), nil
	case int32:
		return append(out, strconv.FormatInt(int64(x), 10)...), nil
	case int64:
		return append(out, strconv.FormatInt(x, 10)...), nil
	case uint:
		return append(out, strconv.FormatUint(uint64(x), 10)...), nil
	case uint8:
		return append(out, strconv.FormatUint(uint64(x), 10)...), nil
	case uint16:
		return append(out, strconv.FormatUint(uint64(x), 10)...), nil
	case uint32:
		return append(out, strconv.FormatUint(uint64(x), 10)...), nil
	case uint64:
		return append(out, strconv.FormatUint(x, 10)...), nil
	case []any:
		out = append(out, '[')
		for i := 0; i < len(x); i++ {
			if i > 0 {
				out = append(out, ',')
			}
			var err error
			out, err = appendJSON(out, x[i])
			if err != nil {
				return nil, err
			}
		}
		return append(out, ']'), nil
	case map[string]any:
		keys := make([]string, 0, len(x))
		for key := range x {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		out = append(out, '{')
		for i := 0; i < len(keys); i++ {
			if i > 0 {
				out = append(out, ',')
			}
			out = append(out, quote(keys[i])...)
			out = append(out, ':')
			var err error
			out, err = appendJSON(out, x[keys[i]])
			if err != nil {
				return nil, err
			}
		}
		return append(out, '}'), nil
	case map[string]string:
		keys := make([]string, 0, len(x))
		for key := range x {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		out = append(out, '{')
		for i := 0; i < len(keys); i++ {
			if i > 0 {
				out = append(out, ',')
			}
			out = append(out, quote(keys[i])...)
			out = append(out, ':')
			out = append(out, quote(x[keys[i]])...)
		}
		return append(out, '}'), nil
	case map[string]RawMessage:
		keys := make([]string, 0, len(x))
		for key := range x {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		out = append(out, '{')
		for i := 0; i < len(keys); i++ {
			if i > 0 {
				out = append(out, ',')
			}
			out = append(out, quote(keys[i])...)
			out = append(out, ':')
			var err error
			out, err = appendJSON(out, x[keys[i]])
			if err != nil {
				return nil, err
			}
		}
		return append(out, '}'), nil
	}
	value := reflect.ValueOf(v)
	if value.IsValid() {
		switch value.Kind() {
		case reflect.Pointer:
			if value.IsNil() {
				return append(out, "null"...), nil
			}
			return appendJSON(out, value.Elem().Interface())
		case reflect.Struct:
			return appendStructJSON(out, value)
		case reflect.Slice, reflect.Array:
			if value.Kind() == reflect.Slice && value.IsNil() {
				return append(out, "null"...), nil
			}
			out = append(out, '[')
			for i := 0; i < value.Len(); i++ {
				if i > 0 {
					out = append(out, ',')
				}
				var err error
				out, err = appendJSON(out, value.Index(i).Interface())
				if err != nil {
					return nil, err
				}
			}
			return append(out, ']'), nil
		}
	}
	return nil, errors.New("json: unsupported value type")
}

func appendStructJSON(out []byte, value reflect.Value) ([]byte, error) {
	out = append(out, '{')
	var err error
	out, _, err = appendStructFields(out, value, false)
	if err != nil {
		return nil, err
	}
	return append(out, '}'), nil
}

func appendStructFields(out []byte, value reflect.Value, wrote bool) ([]byte, bool, error) {
	typ := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" && !field.Anonymous {
			continue
		}
		name, options, skip := jsonFieldTag(field)
		if skip {
			continue
		}
		fieldValue := value.Field(i)
		if field.Anonymous && name == "" {
			embedded := fieldValue
			if embedded.Kind() == reflect.Pointer {
				if embedded.IsNil() {
					continue
				}
				embedded = embedded.Elem()
			}
			if embedded.Kind() == reflect.Struct {
				var err error
				out, wrote, err = appendStructFields(out, embedded, wrote)
				if err != nil {
					return nil, wrote, err
				}
				continue
			}
		}
		if name == "" {
			name = field.Name
		}
		if hasJSONOption(options, "omitempty") && emptyJSONValue(fieldValue) {
			continue
		}
		if wrote {
			out = append(out, ',')
		}
		out = append(out, quote(name)...)
		out = append(out, ':')
		var err error
		if hasJSONOption(options, "string") {
			var encoded []byte
			encoded, err = appendJSON(nil, fieldValue.Interface())
			if err == nil {
				out = append(out, quote(string(encoded))...)
			}
		} else {
			out, err = appendJSON(out, fieldValue.Interface())
		}
		if err != nil {
			return nil, wrote, err
		}
		wrote = true
	}
	return out, wrote, nil
}

func jsonFieldTag(field reflect.StructField) (string, string, bool) {
	tag, ok := field.Tag.Lookup("json")
	if !ok {
		return "", "", false
	}
	name := tag
	options := ""
	for i := 0; i < len(tag); i++ {
		if tag[i] == ',' {
			name = tag[:i]
			options = tag[i+1:]
			break
		}
	}
	return name, options, name == "-"
}

func hasJSONOption(options string, wanted string) bool {
	for len(options) > 0 {
		part := options
		rest := ""
		for i := 0; i < len(options); i++ {
			if options[i] == ',' {
				part = options[:i]
				rest = options[i+1:]
				break
			}
		}
		if part == wanted {
			return true
		}
		options = rest
	}
	return false
}

func emptyJSONValue(value reflect.Value) bool {
	if !value.IsValid() {
		return true
	}
	switch value.Kind() {
	case reflect.Pointer, reflect.Slice:
		return value.IsNil() || value.Len() == 0
	case reflect.String:
		return value.Len() == 0
	}
	switch x := value.Interface().(type) {
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
	}
	return false
}
func Unmarshal(data []byte, v any) error {
	if !Valid(data) {
		return &SyntaxError{Offset: int64(len(data))}
	}
	if u, ok := v.(Unmarshaler); ok {
		return u.UnmarshalJSON(data)
	}
	p := decodeParser{data: data}
	value, err := p.value()
	if err != nil {
		return err
	}
	result := assignDecoded(v, value)
	return result
}

type decodeParser struct {
	data []byte
	pos  int
}

func (p *decodeParser) space() {
	for p.pos < len(p.data) {
		c := p.data[p.pos]
		if c != ' ' && c != '\n' && c != '\r' && c != '\t' {
			return
		}
		p.pos++
	}
}

func (p *decodeParser) value() (any, error) {
	p.space()
	if p.pos >= len(p.data) {
		return nil, errors.New("json: unexpected end of input")
	}
	switch p.data[p.pos] {
	case 'n':
		p.pos += 4
		return nil, nil
	case 't':
		p.pos += 4
		return true, nil
	case 'f':
		p.pos += 5
		return false, nil
	case '"':
		return p.string()
	case '[':
		return p.array()
	case '{':
		return p.object()
	}
	return p.number()
}

func (p *decodeParser) string() (any, error) {
	value, err := p.stringValue()
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (p *decodeParser) stringValue() (string, error) {
	p.pos++
	out := make([]byte, 0, 16)
	for p.pos < len(p.data) {
		c := p.data[p.pos]
		p.pos++
		if c == '"' {
			return string(out), nil
		}
		if c != '\\' {
			out = append(out, c)
			continue
		}
		e := p.data[p.pos]
		p.pos++
		switch e {
		case '"', '\\', '/':
			out = append(out, e)
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
			if r >= 0xd800 && r <= 0xdbff && p.pos+6 <= len(p.data) && p.data[p.pos] == '\\' && p.data[p.pos+1] == 'u' {
				p.pos += 2
				low := p.hexRune()
				if low >= 0xdc00 && low <= 0xdfff {
					r = 0x10000 + (r-0xd800)*0x400 + low - 0xdc00
				}
			}
			out = appendUTF8(out, r)
		}
	}
	return "", errors.New("json: unterminated string")
}

func (p *decodeParser) hexRune() int {
	r := 0
	for i := 0; i < 4; i++ {
		c := p.data[p.pos]
		p.pos++
		r *= 16
		if c >= '0' && c <= '9' {
			r += int(c - '0')
		} else if c >= 'a' && c <= 'f' {
			r += int(c-'a') + 10
		} else {
			r += int(c-'A') + 10
		}
	}
	return r
}

func appendUTF8(out []byte, r int) []byte {
	if r <= 0x7f {
		return append(out, byte(r))
	}
	if r <= 0x7ff {
		return append(out, byte(0xc0|r>>6), byte(0x80|r&0x3f))
	}
	if r <= 0xffff {
		return append(out, byte(0xe0|r>>12), byte(0x80|r>>6&0x3f), byte(0x80|r&0x3f))
	}
	return append(out, byte(0xf0|r>>18), byte(0x80|r>>12&0x3f), byte(0x80|r>>6&0x3f), byte(0x80|r&0x3f))
}

func (p *decodeParser) array() (any, error) {
	p.pos++
	values := make([]any, 0)
	p.space()
	if p.data[p.pos] == ']' {
		p.pos++
		return values, nil
	}
	for {
		value, err := p.value()
		if err != nil {
			return nil, err
		}
		values = append(values, value)
		p.space()
		c := p.data[p.pos]
		p.pos++
		if c == ']' {
			return values, nil
		}
	}
}

func (p *decodeParser) object() (any, error) {
	p.pos++
	values := make(map[string]any)
	p.space()
	if p.data[p.pos] == '}' {
		p.pos++
		return values, nil
	}
	for {
		key, err := p.stringValue()
		if err != nil {
			return nil, err
		}
		p.space()
		p.pos++
		value, err := p.value()
		if err != nil {
			return nil, err
		}
		values[key] = value
		p.space()
		c := p.data[p.pos]
		p.pos++
		if c == '}' {
			return values, nil
		}
		p.space()
	}
}

func (p *decodeParser) number() (any, error) {
	start := p.pos
	for p.pos < len(p.data) {
		c := p.data[p.pos]
		if (c < '0' || c > '9') && c != '-' && c != '+' && c != '.' && c != 'e' && c != 'E' {
			break
		}
		if c == '.' || c == 'e' || c == 'E' {
			return nil, errors.New("json: floating-point numbers are not supported")
		}
		p.pos++
	}
	value, err := strconv.Atoi(string(p.data[start:p.pos]))
	if err != nil {
		return nil, errors.New("json: integer out of range")
	}
	return value, nil
}

func assignDecoded(destination any, decoded any) error {
	if destination == nil {
		return errors.New("json: Unmarshal(nil)")
	}
	if unmarshaler, ok := destination.(Unmarshaler); ok {
		data, err := Marshal(decoded)
		if err != nil {
			return err
		}
		return unmarshaler.UnmarshalJSON(data)
	}
	if out, ok := destination.(*any); ok {
		*out = decoded
		return nil
	}
	if out, ok := destination.(*string); ok {
		value, valueOK := decoded.(string)
		if !valueOK {
			return errors.New("json: cannot unmarshal value into string")
		}
		*out = value
		return nil
	}
	if out, ok := destination.(*bool); ok {
		value, valueOK := decoded.(bool)
		if !valueOK {
			return errors.New("json: cannot unmarshal value into bool")
		}
		*out = value
		return nil
	}
	if out, ok := destination.(*int); ok {
		value, err := decodedInteger(decoded)
		if err != nil {
			return err
		}
		*out = value
		return nil
	}
	if out, ok := destination.(*[]int); ok {
		values, valuesOK := decoded.([]any)
		if !valuesOK {
			return errors.New("json: cannot unmarshal value into int slice")
		}
		result := make([]int, len(values))
		for i := 0; i < len(values); i++ {
			value, err := decodedInteger(values[i])
			if err != nil {
				return err
			}
			result[i] = value
		}
		*out = result
		return nil
	}
	if out, ok := destination.(*map[string]string); ok {
		values, valuesOK := decoded.(map[string]any)
		if !valuesOK {
			return errors.New("json: cannot unmarshal value into string map")
		}
		result := make(map[string]string, len(values))
		for key, item := range values {
			value, err := decodedString(item)
			if err != nil {
				return err
			}
			result[key] = value
		}
		*out = result
		return nil
	}
	if out, ok := destination.(*map[string]RawMessage); ok {
		values, valuesOK := decoded.(map[string]any)
		if !valuesOK {
			return errors.New("json: cannot unmarshal value into RawMessage map")
		}
		result := make(map[string]RawMessage, len(values))
		for key, item := range values {
			data, err := Marshal(item)
			if err != nil {
				return err
			}
			var raw RawMessage = data
			result[key] = raw
		}
		*out = result
		return nil
	}
	switch out := destination.(type) {
	case *string:
		value, ok := decoded.(string)
		if !ok {
			return errors.New("json: cannot unmarshal value into string")
		}
		*out = value
		return nil
	case *bool:
		value, ok := decoded.(bool)
		if !ok {
			return errors.New("json: cannot unmarshal value into bool")
		}
		*out = value
		return nil
	case *int:
		return assignInt(decoded, func(value int) { *out = value })
	case *int8:
		return assignInt(decoded, func(value int) { *out = int8(value) })
	case *int16:
		return assignInt(decoded, func(value int) { *out = int16(value) })
	case *int32:
		return assignInt(decoded, func(value int) { *out = int32(value) })
	case *int64:
		return assignInt(decoded, func(value int) { *out = int64(value) })
	case *uint:
		return assignUint(decoded, func(value int) { *out = uint(value) })
	case *uint8:
		return assignUint(decoded, func(value int) { *out = uint8(value) })
	case *uint16:
		return assignUint(decoded, func(value int) { *out = uint16(value) })
	case *uint32:
		return assignUint(decoded, func(value int) { *out = uint32(value) })
	case *uint64:
		return assignUint(decoded, func(value int) { *out = uint64(value) })
	case *[]any:
		value, ok := decoded.([]any)
		if !ok {
			return errors.New("json: cannot unmarshal value into slice")
		}
		*out = value
		return nil
	case *[]string:
		values, ok := decoded.([]any)
		if !ok {
			return errors.New("json: cannot unmarshal value into string slice")
		}
		result := make([]string, len(values))
		for i := 0; i < len(values); i++ {
			if err := assignDecoded(&result[i], values[i]); err != nil {
				return err
			}
		}
		*out = result
		return nil
	case *[]int:
		values, ok := decoded.([]any)
		if !ok {
			return errors.New("json: cannot unmarshal value into int slice")
		}
		result := make([]int, len(values))
		for i := 0; i < len(values); i++ {
			if err := assignDecoded(&result[i], values[i]); err != nil {
				return err
			}
		}
		*out = result
		return nil
	case *map[string]any:
		value, ok := decoded.(map[string]any)
		if !ok {
			return errors.New("json: cannot unmarshal value into map")
		}
		*out = value
		return nil
	}
	value := reflect.ValueOf(destination)
	if !value.IsValid() || value.Kind() != reflect.Pointer || value.IsNil() {
		return errors.New("json: destination must be a non-nil pointer")
	}
	elem := value.Elem()
	if elem.Kind() != reflect.Struct {
		return errors.New("json: unsupported destination type")
	}
	object, ok := decoded.(map[string]any)
	if !ok {
		return errors.New("json: cannot unmarshal value into struct")
	}
	return assignStruct(elem, object)
}

func decodedString(decoded any) (string, error) {
	if value, ok := decoded.(string); ok {
		return value, nil
	}
	return "", errors.New("json: cannot unmarshal value into string")
}

func decodedInteger(decoded any) (int, error) {
	if value, ok := decoded.(int); ok {
		return value, nil
	}
	if text, ok := decoded.(string); ok {
		value, err := strconv.Atoi(text)
		if err == nil {
			return value, nil
		}
	}
	return 0, errors.New("json: cannot unmarshal value into integer")
}

func assignInt(decoded any, set func(int)) error {
	value, ok := decoded.(int)
	if !ok {
		if text, stringOK := decoded.(string); stringOK {
			parsed, err := strconv.Atoi(text)
			if err == nil {
				set(parsed)
				return nil
			}
		}
		return errors.New("json: cannot unmarshal value into integer")
	}
	set(value)
	return nil
}

func assignUint(decoded any, set func(int)) error {
	value, ok := decoded.(int)
	if !ok || value < 0 {
		return errors.New("json: cannot unmarshal value into unsigned integer")
	}
	set(value)
	return nil
}

func assignStruct(value reflect.Value, object map[string]any) error {
	typ := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" && !field.Anonymous {
			continue
		}
		name, _, skip := jsonFieldTag(field)
		if skip {
			continue
		}
		fieldValue := value.Field(i)
		if field.Anonymous && name == "" && fieldValue.Kind() == reflect.Struct {
			if err := assignStruct(fieldValue, object); err != nil {
				return err
			}
			continue
		}
		if name == "" {
			name = field.Name
		}
		decoded, present := object[name]
		if !present {
			continue
		}
		address := fieldValue.Addr()
		if !address.IsValid() {
			return errors.New("json: field is not addressable: " + field.Name)
		}
		if err := assignDecoded(address.Interface(), decoded); err != nil {
			return err
		}
	}
	return nil
}
func Valid(data []byte) bool {
	p := syntaxParser{data: data}
	if !p.value() {
		return false
	}
	p.space()
	return p.pos == len(data)
}

type syntaxParser struct {
	data []byte
	pos  int
}

func (p *syntaxParser) space() {
	for p.pos < len(p.data) {
		c := p.data[p.pos]
		if c != ' ' && c != '\n' && c != '\r' && c != '\t' {
			break
		}
		p.pos++
	}
}
func (p *syntaxParser) literal(s string) bool {
	if p.pos+len(s) > len(p.data) {
		return false
	}
	for i := 0; i < len(s); i++ {
		if p.data[p.pos+i] != s[i] {
			return false
		}
	}
	p.pos += len(s)
	return true
}
func (p *syntaxParser) value() bool {
	p.space()
	if p.pos >= len(p.data) {
		return false
	}
	switch p.data[p.pos] {
	case 'n':
		return p.literal("null")
	case 't':
		return p.literal("true")
	case 'f':
		return p.literal("false")
	case '"':
		return p.string()
	case '[':
		return p.array()
	case '{':
		return p.object()
	}
	return p.number()
}
func (p *syntaxParser) string() bool {
	p.pos++
	for p.pos < len(p.data) {
		c := p.data[p.pos]
		p.pos++
		if c == '"' {
			return true
		}
		if c < 0x20 {
			return false
		}
		if c == '\\' {
			if p.pos >= len(p.data) {
				return false
			}
			e := p.data[p.pos]
			p.pos++
			if e == 'u' {
				if p.pos+4 > len(p.data) {
					return false
				}
				for i := 0; i < 4; i++ {
					c := p.data[p.pos+i]
					if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
						return false
					}
				}
				p.pos += 4
			} else if e != '"' && e != '\\' && e != '/' && e != 'b' && e != 'f' && e != 'n' && e != 'r' && e != 't' {
				return false
			}
		}
	}
	return false
}
func (p *syntaxParser) number() bool {
	start := p.pos
	if p.pos < len(p.data) && p.data[p.pos] == '-' {
		p.pos++
	}
	if p.pos >= len(p.data) {
		return false
	}
	if p.data[p.pos] == '0' {
		p.pos++
	} else {
		if p.data[p.pos] < '1' || p.data[p.pos] > '9' {
			return false
		}
		for p.pos < len(p.data) && p.data[p.pos] >= '0' && p.data[p.pos] <= '9' {
			p.pos++
		}
	}
	if p.pos < len(p.data) && p.data[p.pos] == '.' {
		p.pos++
		n := p.pos
		for p.pos < len(p.data) && p.data[p.pos] >= '0' && p.data[p.pos] <= '9' {
			p.pos++
		}
		if n == p.pos {
			return false
		}
	}
	if p.pos < len(p.data) && (p.data[p.pos] == 'e' || p.data[p.pos] == 'E') {
		p.pos++
		if p.pos < len(p.data) && (p.data[p.pos] == '+' || p.data[p.pos] == '-') {
			p.pos++
		}
		n := p.pos
		for p.pos < len(p.data) && p.data[p.pos] >= '0' && p.data[p.pos] <= '9' {
			p.pos++
		}
		if n == p.pos {
			return false
		}
	}
	return p.pos > start
}
func (p *syntaxParser) array() bool {
	p.pos++
	p.space()
	if p.pos < len(p.data) && p.data[p.pos] == ']' {
		p.pos++
		return true
	}
	for {
		if !p.value() {
			return false
		}
		p.space()
		if p.pos >= len(p.data) {
			return false
		}
		c := p.data[p.pos]
		p.pos++
		if c == ']' {
			return true
		}
		if c != ',' {
			return false
		}
	}
}
func (p *syntaxParser) object() bool {
	p.pos++
	p.space()
	if p.pos < len(p.data) && p.data[p.pos] == '}' {
		p.pos++
		return true
	}
	for {
		p.space()
		if p.pos >= len(p.data) || p.data[p.pos] != '"' || !p.string() {
			return false
		}
		p.space()
		if p.pos >= len(p.data) || p.data[p.pos] != ':' {
			return false
		}
		p.pos++
		if !p.value() {
			return false
		}
		p.space()
		if p.pos >= len(p.data) {
			return false
		}
		c := p.data[p.pos]
		p.pos++
		if c == '}' {
			return true
		}
		if c != ',' {
			return false
		}
	}
}
func quote(s string) []byte {
	out := []byte{'"'}
	hex := "0123456789abcdef"
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '"' || c == '\\' {
			out = append(out, '\\', c)
		} else if c == '\n' {
			out = append(out, '\\', 'n')
		} else if c == '\r' {
			out = append(out, '\\', 'r')
		} else if c == '\t' {
			out = append(out, '\\', 't')
		} else if c < 0x20 {
			out = append(out, '\\', 'u', '0', '0', hex[c>>4], hex[c&15])
		} else {
			out = append(out, c)
		}
	}
	return append(out, '"')
}

func MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	data, err := Marshal(v)
	if err != nil {
		return nil, err
	}
	out := make([]byte, 0, len(data)+16)
	depth := 0
	inString := false
	escaped := false
	pad := func() {
		out = append(out, prefix...)
		for i := 0; i < depth; i++ {
			out = append(out, indent...)
		}
	}
	for i := 0; i < len(data); i++ {
		c := data[i]
		if inString {
			out = append(out, c)
			if escaped {
				escaped = false
			} else if c == '\\' {
				escaped = true
			} else if c == '"' {
				inString = false
			}
			continue
		}
		if c == '"' {
			inString = true
			out = append(out, c)
			continue
		}
		switch c {
		case '{', '[':
			out = append(out, c)
			depth++
			if i+1 < len(data) && data[i+1] != '}' && data[i+1] != ']' {
				out = append(out, '\n')
				pad()
			}
		case '}', ']':
			depth--
			if i > 0 && data[i-1] != '{' && data[i-1] != '[' {
				out = append(out, '\n')
				pad()
			}
			out = append(out, c)
		case ',':
			out = append(out, c, '\n')
			pad()
		case ':':
			out = append(out, ':', ' ')
		default:
			out = append(out, c)
		}
	}
	return out, nil
}

type Encoder struct {
	w      io.Writer
	prefix string
	indent string
}

func NewEncoder(w io.Writer) *Encoder { return &Encoder{w: w} }

func (e *Encoder) SetIndent(prefix, indent string) {
	e.prefix = prefix
	e.indent = indent
}

func (e *Encoder) SetEscapeHTML(on bool) { _ = on }

func (e *Encoder) Encode(v any) error {
	var data []byte
	var err error
	if e.indent != "" {
		data, err = MarshalIndent(v, e.prefix, e.indent)
	} else {
		data, err = Marshal(v)
	}
	if err != nil {
		return err
	}
	data = append(data, '\n')
	n, err := e.w.Write(data)
	if err == nil && n != len(data) {
		return io.ErrShortWrite
	}
	return err
}

type Decoder struct {
	r      io.Reader
	data   []byte
	pos    int
	loaded bool
}

func NewDecoder(r io.Reader) *Decoder { return &Decoder{r: r} }

func (d *Decoder) Decode(v any) error {
	if !d.loaded {
		data, err := io.ReadAll(d.r)
		if err != nil {
			return err
		}
		d.data = data
		d.loaded = true
	}
	p := syntaxParser{data: d.data, pos: d.pos}
	p.space()
	if p.pos == len(p.data) {
		return io.EOF
	}
	start := p.pos
	if !p.value() {
		return &SyntaxError{Offset: int64(p.pos)}
	}
	d.pos = p.pos
	return Unmarshal(d.data[start:p.pos], v)
}
