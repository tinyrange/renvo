//go:build renvo

package reflect

type Kind uint

const (
	Invalid Kind = iota
	Bool
	Int
	Int8
	Int16
	Int32
	Int64
	Uint
	Uint8
	Uint16
	Uint32
	Uint64
	Uintptr
	Float32
	Float64
	Complex64
	Complex128
	Array
	Chan
	Func
	Interface
	Map
	Pointer
	Slice
	String
	Struct
	UnsafePointer
)

type StructTag string

func (tag StructTag) Get(key string) string {
	value, _ := tag.Lookup(key)
	return value
}

func (tag StructTag) Lookup(key string) (value string, ok bool) {
	s := string(tag)
	for len(s) > 0 {
		for len(s) > 0 && s[0] == ' ' {
			s = s[1:]
		}
		if len(s) == 0 {
			break
		}
		colon := -1
		for i := 0; i < len(s); i++ {
			if s[i] == ':' {
				colon = i
				break
			}
			if s[i] == ' ' || s[i] == '"' {
				return "", false
			}
		}
		if colon <= 0 || colon+1 >= len(s) || s[colon+1] != '"' {
			return "", false
		}
		end := colon + 2
		for end < len(s) {
			if s[end] == '\\' {
				end += 2
				continue
			}
			if s[end] == '"' {
				break
			}
			end++
		}
		if end >= len(s) {
			return "", false
		}
		if s[:colon] == key {
			return unquoteTag(s[colon+2 : end]), true
		}
		s = s[end+1:]
	}
	return "", false
}

func unquoteTag(value string) string {
	out := ""
	for i := 0; i < len(value); i++ {
		if value[i] == '\\' && i+1 < len(value) {
			i++
			switch value[i] {
			case 'n':
				out += "\n"
			case 'r':
				out += "\r"
			case 't':
				out += "\t"
			default:
				out += value[i : i+1]
			}
		} else {
			out += value[i : i+1]
		}
	}
	return out
}

type StructField struct {
	Name      string
	PkgPath   string
	Type      Type
	Tag       StructTag
	Offset    uintptr
	Index     []int
	Anonymous bool
}

type Type interface {
	Kind() Kind
	Name() string
	NumField() int
	Field(i int) StructField
}

type renvoType struct{ value any }

func (t renvoType) Kind() Kind    { return Kind(renvo_runtime_ReflectKind(t.value)) }
func (t renvoType) Name() string  { return renvo_runtime_ReflectTypeName(t.value) }
func (t renvoType) NumField() int { return renvo_runtime_ReflectNumField(t.value) }
func (t renvoType) Field(i int) StructField {
	value := renvo_runtime_ReflectField(t.value, i)
	name := renvo_runtime_ReflectFieldName(t.value, i)
	pkgPath := ""
	if !exported(name) {
		pkgPath = "renvo"
	}
	return StructField{
		Name: name, PkgPath: pkgPath, Type: TypeOf(value),
		Tag:    StructTag(renvo_runtime_ReflectFieldTag(t.value, i)),
		Offset: uintptr(renvo_runtime_ReflectFieldOffset(t.value, i)),
		Index:  []int{i}, Anonymous: renvo_runtime_ReflectFieldAnonymous(t.value, i),
	}
}

type Value struct {
	value   any
	address any
	valid   bool
}

func TypeOf(value any) Type {
	if value == nil {
		return nil
	}
	return renvoType{value}
}

func ValueOf(value any) Value { return Value{value: value, valid: value != nil} }
func Indirect(value Value) Value {
	if !value.valid || value.Kind() != Pointer {
		return value
	}
	return value.Elem()
}
func (v Value) IsValid() bool { return v.valid }
func (v Value) Kind() Kind {
	if !v.valid {
		return Invalid
	}
	return Kind(renvo_runtime_ReflectKind(v.value))
}
func (v Value) Type() Type {
	if !v.valid {
		return nil
	}
	return renvoType{v.value}
}
func (v Value) Interface() any { return v.value }
func (v Value) Elem() Value {
	value := renvo_runtime_ReflectElem(v.value)
	return Value{value: value, address: v.value, valid: value != nil}
}
func (v Value) Len() int      { return renvo_runtime_ReflectLen(v.value) }
func (v Value) NumField() int { return renvo_runtime_ReflectNumField(v.value) }
func (v Value) Field(i int) Value {
	value := renvo_runtime_ReflectField(v.value, i)
	address := any(nil)
	if v.address != nil {
		address = renvo_runtime_ReflectFieldAddress(v.address, i)
	}
	return Value{value: value, address: address, valid: true}
}
func (v Value) Index(i int) Value {
	value := renvo_runtime_ReflectIndex(v.value, i)
	return Value{value: value, valid: true}
}
func (v Value) IsNil() bool { return renvo_runtime_ReflectIsNil(v.value) }
func (v Value) Addr() Value {
	return Value{value: v.address, valid: v.address != nil}
}

func exported(name string) bool { return len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z' }

func renvo_runtime_ReflectKind(value any) int                       { return 0 }
func renvo_runtime_ReflectTypeName(value any) string                { return "" }
func renvo_runtime_ReflectNumField(value any) int                   { return 0 }
func renvo_runtime_ReflectFieldName(value any, index int) string    { return "" }
func renvo_runtime_ReflectFieldTag(value any, index int) string     { return "" }
func renvo_runtime_ReflectFieldOffset(value any, index int) int     { return 0 }
func renvo_runtime_ReflectFieldAnonymous(value any, index int) bool { return false }
func renvo_runtime_ReflectField(value any, index int) any           { return nil }
func renvo_runtime_ReflectElem(value any) any                       { return nil }
func renvo_runtime_ReflectLen(value any) int                        { return 0 }
func renvo_runtime_ReflectIndex(value any, index int) any           { return nil }
func renvo_runtime_ReflectIsNil(value any) bool                     { return value == nil }
func renvo_runtime_ReflectFieldAddress(value any, index int) any    { return nil }
