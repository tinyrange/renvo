//go:build !renvo

package reflect

import stdreflect "reflect"

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
	return stdreflect.StructTag(tag).Get(key)
}

func (tag StructTag) Lookup(key string) (value string, ok bool) {
	return stdreflect.StructTag(tag).Lookup(key)
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

type hostType struct{ typ stdreflect.Type }

func (t hostType) Kind() Kind    { return Kind(t.typ.Kind()) }
func (t hostType) Name() string  { return t.typ.Name() }
func (t hostType) NumField() int { return t.typ.NumField() }
func (t hostType) Field(i int) StructField {
	f := t.typ.Field(i)
	return StructField{Name: f.Name, PkgPath: f.PkgPath, Type: hostType{f.Type}, Tag: StructTag(f.Tag), Offset: f.Offset, Index: f.Index, Anonymous: f.Anonymous}
}

type Value struct{ value stdreflect.Value }

func TypeOf(value any) Type {
	typ := stdreflect.TypeOf(value)
	if typ == nil {
		return nil
	}
	return hostType{typ}
}

func ValueOf(value any) Value    { return Value{stdreflect.ValueOf(value)} }
func Indirect(value Value) Value { return Value{stdreflect.Indirect(value.value)} }
func (v Value) IsValid() bool    { return v.value.IsValid() }
func (v Value) Kind() Kind       { return Kind(v.value.Kind()) }
func (v Value) Type() Type {
	if !v.value.IsValid() {
		return nil
	}
	return hostType{v.value.Type()}
}
func (v Value) Interface() any    { return v.value.Interface() }
func (v Value) Elem() Value       { return Value{v.value.Elem()} }
func (v Value) Len() int          { return v.value.Len() }
func (v Value) NumField() int     { return v.value.NumField() }
func (v Value) Field(i int) Value { return Value{v.value.Field(i)} }
func (v Value) Index(i int) Value { return Value{v.value.Index(i)} }
func (v Value) IsNil() bool       { return v.value.IsNil() }
func (v Value) Addr() Value       { return Value{v.value.Addr()} }
