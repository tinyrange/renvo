// Package starlark implements a compact, embeddable Starlark interpreter.
//
// The package is designed to run unchanged with both Go and Renvo. Programs
// can exchange values with their host through predeclared names and built-in
// functions, and execution can be bounded or cancelled through Thread.
package starlark

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type Value interface {
	String() string
	Type() string
}

type StringDict map[string]Value

// Keys returns the dictionary keys in deterministic lexical order.
func (d StringDict) Keys() []string {
	keys := make([]string, 0, len(d))
	for key := range d {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

type noneType struct{}

func (noneType) String() string { return "None" }
func (noneType) Type() string   { return "NoneType" }

var None Value = noneType{}

type Bool bool

func (b Bool) String() string {
	if b {
		return "True"
	}
	return "False"
}
func (Bool) Type() string { return "bool" }

const (
	False Bool = false
	True  Bool = true
)

type String string

func (s String) String() string { return strconv.Quote(string(s)) }
func (String) Type() string     { return "string" }
func (s String) Iterate() Iterator {
	values := make([]Value, 0, len([]rune(s)))
	for _, r := range []rune(s) {
		piece := stringFromRune(r)
		values = append(values, piece)
	}
	return newSliceIterator(values)
}

func stringFromRune(r rune) String { return String(string([]rune{r})) }

func stringFromRunes(r []rune) String { return String(string(r)) }

func AsString(v Value) (string, bool) { s, ok := v.(String); return string(s), ok }

type Int int64

func (i Int) String() string { return strconv.FormatInt(int64(i), 10) }
func (Int) Type() string     { return "int" }
func MakeInt(v int) Int      { return Int(v) }
func MakeInt64(v int64) Int  { return Int(v) }

type Float float64

func (f Float) String() string { return formatFloat(float64(f)) }
func (Float) Type() string     { return "float" }

type Tuple []Value

func (t Tuple) Type() string { return "tuple" }
func (t Tuple) String() string {
	parts := renderValues(t)
	if len(t) == 1 {
		return "(" + parts[0] + ",)"
	}
	return "(" + strings.Join(parts, ", ") + ")"
}
func (t Tuple) Iterate() Iterator {
	var values []Value
	values = append(values, t...)
	return newSliceIterator(values)
}

type List struct{ elems []Value }

func NewList(values []Value) *List {
	list := &List{}
	var elems []Value
	elems = append(elems, values...)
	list.elems = elems
	return list
}
func (l *List) Type() string      { return "list" }
func (l *List) String() string    { return "[" + strings.Join(renderValues(l.elems), ", ") + "]" }
func (l *List) Len() int          { return len(l.elems) }
func (l *List) Index(i int) Value { return l.elems[i] }
func (l *List) Iterate() Iterator { return newSliceIterator(l.elems) }

type dictEntry struct{ key, value Value }
type Dict struct{ entries []dictEntry }

func NewDict() *Dict         { return &Dict{} }
func (d *Dict) Type() string { return "dict" }
func (d *Dict) String() string {
	parts := make([]string, len(d.entries))
	for i, e := range d.entries {
		parts[i] = e.key.String() + ": " + e.value.String()
	}
	return "{" + strings.Join(parts, ", ") + "}"
}
func (d *Dict) Len() int { return len(d.entries) }
func (d *Dict) SetKey(k, v Value) error {
	if !hashable(k) {
		return errorf("unhashable type: %s", k.Type())
	}
	for i := range d.entries {
		if equal(d.entries[i].key, k) {
			d.entries[i].value = v
			return nil
		}
	}
	d.entries = append(d.entries, dictEntry{k, v})
	return nil
}
func (d *Dict) Get(k Value) (Value, bool, error) {
	for _, e := range d.entries {
		if equal(e.key, k) {
			return e.value, true, nil
		}
	}
	return nil, false, nil
}
func (d *Dict) Iterate() Iterator {
	keys := make([]Value, len(d.entries))
	for i, e := range d.entries {
		keys[i] = e.key
	}
	return newSliceIterator(keys)
}

type HasAttrs interface {
	Value
	Attr(string) (Value, error)
	AttrNames() []string
}
type Struct struct {
	name   string
	fields StringDict
}

func NewStruct(name Value, fields StringDict) *Struct {
	n := "struct"
	if s, ok := AsString(name); ok {
		n = s
	}
	copy := StringDict{}
	for k, v := range fields {
		copy[k] = v
	}
	result := &Struct{}
	result.name = n
	result.fields = copy
	return result
}
func (s *Struct) Type() string { return s.name }
func (s *Struct) String() string {
	names := s.AttrNames()
	parts := make([]string, len(names))
	for i, n := range names {
		parts[i] = n + " = " + s.fields[n].String()
	}
	return "struct(" + strings.Join(parts, ", ") + ")"
}
func (s *Struct) Attr(name string) (Value, error) {
	v, ok := s.fields[name]
	if !ok {
		return nil, nil
	}
	return v, nil
}
func (s *Struct) AttrNames() []string {
	names := make([]string, 0, len(s.fields))
	for n := range s.fields {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

type Iterator interface {
	Next(*Value) bool
	Done()
}
type Iterable interface {
	Value
	Iterate() Iterator
}
type sliceIterator struct {
	values []Value
	index  int
}

func (it *sliceIterator) Next(v *Value) bool {
	if it.index >= len(it.values) {
		return false
	}
	*v = it.values[it.index]
	it.index++
	return true
}
func (*sliceIterator) Done() {}

func newSliceIterator(values []Value) Iterator {
	iterator := &sliceIterator{}
	iterator.values = values
	var result Iterator
	result = iterator
	return result
}

type Thread struct {
	Name     string
	Print    func(*Thread, string)
	mu       threadLock
	locals   map[string]any
	maxSteps uint64
	steps    uint64
	cancel   string
}

func (t *Thread) SetLocal(k string, v any) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.locals == nil {
		t.locals = map[string]any{}
	}
	t.locals[k] = v
}
func (t *Thread) Local(k string) any            { t.mu.RLock(); defer t.mu.RUnlock(); return t.locals[k] }
func (t *Thread) SetMaxExecutionSteps(n uint64) { t.maxSteps = n }
func (t *Thread) Cancel(reason string)          { t.mu.Lock(); t.cancel = reason; t.mu.Unlock() }
func (t *Thread) step() error {
	t.mu.RLock()
	reason := t.cancel
	t.mu.RUnlock()
	if reason != "" {
		return errorf("Starlark computation cancelled: %s", reason)
	}
	t.steps++
	if t.maxSteps > 0 && t.steps > t.maxSteps {
		return errorf("Starlark computation cancelled: too many steps")
	}
	return nil
}

// BuiltinFunc implements a host-provided Starlark function.
type BuiltinFunc func(*Thread, *Builtin, Tuple, []Tuple) (Value, error)
type Builtin struct {
	name string
	fn   BuiltinFunc
}

func NewBuiltin(name string, fn BuiltinFunc) *Builtin {
	builtin := &Builtin{name: name}
	builtin.fn = fn
	return builtin
}
func (b *Builtin) Name() string   { return b.name }
func (b *Builtin) Type() string   { return "builtin_function_or_method" }
func (b *Builtin) String() string { return "<built-in function " + b.name + ">" }

type parameter struct {
	name       string
	def        Value
	hasDefault bool
}
type Function struct {
	name, doc string
	params    []parameter
	body      []stmt
	closure   *env
}

func (f *Function) Name() string              { return f.name }
func (f *Function) Type() string              { return "function" }
func (f *Function) String() string            { return "<function " + f.name + ">" }
func (f *Function) Doc() string               { return f.doc }
func (f *Function) NumParams() int            { return len(f.params) }
func (*Function) NumKwonlyParams() int        { return 0 }
func (f *Function) Param(i int) (string, any) { return f.params[i].name, nil }
func (f *Function) ParamDefault(i int) Value {
	if f.params[i].hasDefault {
		return f.params[i].def
	}
	return nil
}
func (*Function) HasVarargs() bool { return false }
func (*Function) HasKwargs() bool  { return false }

func renderValues(v []Value) []string {
	out := make([]string, len(v))
	for i, x := range v {
		out[i] = x.String()
	}
	return out
}
func truth(v Value) bool {
	switch x := v.(type) {
	case noneType:
		return false
	case Bool:
		return bool(x)
	case String:
		return len(x) > 0
	case Int:
		return x != 0
	case Float:
		return x != 0
	case *List:
		return x.Len() > 0
	case Tuple:
		return len(x) > 0
	case *Dict:
		return x.Len() > 0
	}
	return true
}
func equal(a, b Value) bool {
	switch x := a.(type) {
	case noneType:
		_, ok := b.(noneType)
		return ok
	case Bool:
		y, ok := b.(Bool)
		return ok && x == y
	case String:
		y, ok := b.(String)
		return ok && x == y
	case Int:
		y, ok := b.(Int)
		return ok && x == y
	case Float:
		y, ok := b.(Float)
		return ok && x == y
	case Tuple:
		y, ok := b.(Tuple)
		if !ok || len(x) != len(y) {
			return false
		}
		for i := range x {
			if !equal(x[i], y[i]) {
				return false
			}
		}
		return true
	case *List:
		y, ok := b.(*List)
		if !ok || x.Len() != y.Len() {
			return false
		}
		for i := range x.elems {
			if !equal(x.elems[i], y.elems[i]) {
				return false
			}
		}
		return true
	case *Dict:
		y, ok := b.(*Dict)
		if !ok || x.Len() != y.Len() {
			return false
		}
		for _, entry := range x.entries {
			value, found, err := y.Get(entry.key)
			if err != nil || !found || !equal(entry.value, value) {
				return false
			}
		}
		return true
	}
	return a == b
}

func hashable(v Value) bool {
	switch x := v.(type) {
	case noneType, Bool, String, Int, Float, *Builtin, *Function:
		return true
	case Tuple:
		for _, item := range x {
			if !hashable(item) {
				return false
			}
		}
		return true
	}
	return false
}

type evaluationError struct{ message string }

func (e *evaluationError) Error() string { return e.message }

func errorf(format string, args ...any) error {
	result := &evaluationError{}
	result.message = fmt.Sprintf(format, args...)
	return result
}

type stringBuilder struct{ bytes []byte }

func (b *stringBuilder) WriteByte(c byte) { b.bytes = append(b.bytes, c) }
func (b *stringBuilder) WriteString(s string) {
	b.bytes = append(b.bytes, []byte(s)...)
}
func (b *stringBuilder) String() string { return string(b.bytes) }

func formatFloat(value float64) string {
	if value != value {
		return "nan"
	}
	if value == 0 {
		return "0.0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	whole := int64(value)
	fraction := value - float64(whole)
	out := strconv.FormatInt(whole, 10)
	if fraction == 0 {
		out += ".0"
	} else {
		var digits []byte
		for i := 0; i < 12 && fraction != 0; i++ {
			fraction *= 10
			digit := int(fraction)
			digits = append(digits, byte('0'+digit))
			fraction -= float64(digit)
		}
		for len(digits) > 1 && digits[len(digits)-1] == '0' {
			digits = digits[:len(digits)-1]
		}
		out += "." + string(digits)
	}
	if negative {
		return "-" + out
	}
	return out
}

func parseFloat(text string) (float64, error) {
	text = strings.ReplaceAll(text, "_", "")
	if text == "" {
		return 0, errorf("invalid float literal")
	}
	value := float64(0)
	fractionScale := float64(0)
	for i := 0; i < len(text); i++ {
		c := text[i]
		if c == '.' {
			if fractionScale != 0 {
				return 0, errorf("invalid float literal %q", text)
			}
			fractionScale = 1
			continue
		}
		if c < '0' || c > '9' {
			return 0, errorf("invalid float literal %q", text)
		}
		if fractionScale == 0 {
			value = value*10 + float64(c-'0')
		} else {
			fractionScale *= 10
			value += float64(c-'0') / fractionScale
		}
	}
	return value, nil
}

func lowerString(s string) string {
	bytes := []byte(s)
	for i := range bytes {
		if bytes[i] >= 'A' && bytes[i] <= 'Z' {
			bytes[i] += 'a' - 'A'
		}
	}
	return string(bytes)
}

func upperString(s string) string {
	bytes := []byte(s)
	for i := range bytes {
		if bytes[i] >= 'a' && bytes[i] <= 'z' {
			bytes[i] -= 'a' - 'A'
		}
	}
	return string(bytes)
}

func isSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f' || r == '\v' || r == '\u0085' || r == '\u00a0'
}

func UnpackArgs(fn string, args Tuple, kwargs []Tuple, pairs ...any) error {
	if len(pairs)%2 != 0 {
		return errorf("%s: invalid host signature", fn)
	}
	kw := map[string]Value{}
	for _, pair := range kwargs {
		if len(pair) != 2 {
			return errorf("%s: invalid keyword", fn)
		}
		name, ok := AsString(pair[0])
		if !ok {
			return errorf("%s: invalid keyword", fn)
		}
		if _, exists := kw[name]; exists {
			return errorf("%s: duplicate argument %s", fn, name)
		}
		kw[name] = pair[1]
	}
	if len(args) > len(pairs)/2 {
		return errorf("%s: got %d positional arguments, want at most %d", fn, len(args), len(pairs)/2)
	}
	for i := 0; i < len(pairs); i += 2 {
		spec, ok := pairs[i].(string)
		if !ok || spec == "" {
			return errorf("%s: invalid host signature", fn)
		}
		dst := pairs[i+1]
		optional := strings.HasSuffix(spec, "?")
		name := strings.TrimSuffix(spec, "?")
		var v Value
		found := false
		if i/2 < len(args) {
			v = args[i/2]
			found = true
			if _, dup := kw[name]; dup {
				return errorf("%s: got multiple values for %s", fn, name)
			}
		} else if x, ok := kw[name]; ok {
			v = x
			found = true
			delete(kw, name)
		}
		if !found {
			if optional {
				continue
			}
			return errorf("%s: missing argument %s", fn, name)
		}
		if err := assignGo(dst, v); err != nil {
			return errorf("%s: for parameter %s: %s", fn, name, err.Error())
		}
	}
	if len(kw) > 0 {
		for n := range kw {
			return errorf("%s: unexpected keyword argument %s", fn, n)
		}
	}
	return nil
}
func assignGo(dst any, v Value) error {
	switch p := dst.(type) {
	case *string:
		x, ok := AsString(v)
		if !ok {
			return errorf("got %s, want string", v.Type())
		}
		*p = x
	case *int:
		x, ok := v.(Int)
		if !ok {
			return errorf("got %s, want int", v.Type())
		}
		*p = int(x)
	case *bool:
		x, ok := v.(Bool)
		if !ok {
			return errorf("got %s, want bool", v.Type())
		}
		*p = bool(x)
	case *Value:
		*p = v
	case **List:
		x, ok := v.(*List)
		if !ok {
			return errorf("got %s, want list", v.Type())
		}
		*p = x
	default:
		return errorf("unsupported host destination")
	}
	return nil
}
