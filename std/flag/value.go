package flag

import "strconv"

type stringValue struct{ target *string }

func (p *stringValue) String() string {
	if p == nil || p.target == nil {
		return ""
	}
	v := *p.target
	return string(v)
}
func (p *stringValue) Get() any              { return *p.target }
func (p *stringValue) Set(text string) error { *p.target = text; return nil }
func (f *FlagSet) StringVar(p *string, name string, value string, usage string) {
	*p = value
	f.Var(&stringValue{target: p}, name, usage)
}
func (f *FlagSet) String(name string, value string, usage string) *string {
	p := new(string)
	f.StringVar(p, name, value, usage)
	return p
}
func StringVar(p *string, name string, value string, usage string) {
	CommandLine.StringVar(p, name, value, usage)
}
func String(name string, value string, usage string) *string {
	return CommandLine.String(name, value, usage)
}

type boolValue struct{ target *bool }

func (p *boolValue) String() string {
	if p == nil || p.target == nil {
		return ""
	}
	v := *p.target
	return strconv.FormatBool(v)
}
func (p *boolValue) Get() any { return *p.target }
func (p *boolValue) Set(text string) error {
	value, err := strconv.ParseBool(text)
	*p.target = bool(value)
	return err
}
func (p *boolValue) IsBoolFlag() bool { return true }
func (f *FlagSet) BoolVar(p *bool, name string, value bool, usage string) {
	*p = value
	f.Var(&boolValue{target: p}, name, usage)
}
func (f *FlagSet) Bool(name string, value bool, usage string) *bool {
	p := new(bool)
	f.BoolVar(p, name, value, usage)
	return p
}
func BoolVar(p *bool, name string, value bool, usage string) {
	CommandLine.BoolVar(p, name, value, usage)
}
func Bool(name string, value bool, usage string) *bool { return CommandLine.Bool(name, value, usage) }

type intValue struct{ target *int }

func (p *intValue) String() string {
	if p == nil || p.target == nil {
		return ""
	}
	v := *p.target
	return strconv.FormatInt(int64(v), 10)
}
func (p *intValue) Get() any { return *p.target }
func (p *intValue) Set(text string) error {
	value, err := strconv.ParseInt(text, 0, strconv.IntSize)
	*p.target = int(value)
	return err
}
func (f *FlagSet) IntVar(p *int, name string, value int, usage string) {
	*p = value
	f.Var(&intValue{target: p}, name, usage)
}
func (f *FlagSet) Int(name string, value int, usage string) *int {
	p := new(int)
	f.IntVar(p, name, value, usage)
	return p
}
func IntVar(p *int, name string, value int, usage string) { CommandLine.IntVar(p, name, value, usage) }
func Int(name string, value int, usage string) *int       { return CommandLine.Int(name, value, usage) }

type int64Value struct{ target *int64 }

func (p *int64Value) String() string {
	if p == nil || p.target == nil {
		return ""
	}
	v := *p.target
	return strconv.FormatInt(v, 10)
}
func (p *int64Value) Get() any { return *p.target }
func (p *int64Value) Set(text string) error {
	value, err := strconv.ParseInt(text, 0, 64)
	*p.target = int64(value)
	return err
}
func (f *FlagSet) Int64Var(p *int64, name string, value int64, usage string) {
	*p = value
	f.Var(&int64Value{target: p}, name, usage)
}
func (f *FlagSet) Int64(name string, value int64, usage string) *int64 {
	p := new(int64)
	f.Int64Var(p, name, value, usage)
	return p
}
func Int64Var(p *int64, name string, value int64, usage string) {
	CommandLine.Int64Var(p, name, value, usage)
}
func Int64(name string, value int64, usage string) *int64 {
	return CommandLine.Int64(name, value, usage)
}

type uintValue struct{ target *uint }

func (p *uintValue) String() string {
	if p == nil || p.target == nil {
		return ""
	}
	v := *p.target
	return strconv.FormatUint(uint64(v), 10)
}
func (p *uintValue) Get() any { return *p.target }
func (p *uintValue) Set(text string) error {
	value, err := strconv.ParseUint(text, 0, strconv.IntSize)
	*p.target = uint(value)
	return err
}
func (f *FlagSet) UintVar(p *uint, name string, value uint, usage string) {
	*p = value
	f.Var(&uintValue{target: p}, name, usage)
}
func (f *FlagSet) Uint(name string, value uint, usage string) *uint {
	p := new(uint)
	f.UintVar(p, name, value, usage)
	return p
}
func UintVar(p *uint, name string, value uint, usage string) {
	CommandLine.UintVar(p, name, value, usage)
}
func Uint(name string, value uint, usage string) *uint { return CommandLine.Uint(name, value, usage) }

type uint64Value struct{ target *uint64 }

func (p *uint64Value) String() string {
	if p == nil || p.target == nil {
		return ""
	}
	v := *p.target
	return strconv.FormatUint(v, 10)
}
func (p *uint64Value) Get() any { return *p.target }
func (p *uint64Value) Set(text string) error {
	value, err := strconv.ParseUint(text, 0, 64)
	*p.target = uint64(value)
	return err
}
func (f *FlagSet) Uint64Var(p *uint64, name string, value uint64, usage string) {
	*p = value
	f.Var(&uint64Value{target: p}, name, usage)
}
func (f *FlagSet) Uint64(name string, value uint64, usage string) *uint64 {
	p := new(uint64)
	f.Uint64Var(p, name, value, usage)
	return p
}
func Uint64Var(p *uint64, name string, value uint64, usage string) {
	CommandLine.Uint64Var(p, name, value, usage)
}
func Uint64(name string, value uint64, usage string) *uint64 {
	return CommandLine.Uint64(name, value, usage)
}

type float64Value struct{ target *float64 }

func (p *float64Value) String() string {
	if p == nil || p.target == nil {
		return ""
	}
	v := *p.target
	return strconv.FormatFloat(v, 'g', -1, 64)
}
func (p *float64Value) Get() any { return *p.target }
func (p *float64Value) Set(text string) error {
	value, err := strconv.ParseFloat(text, 64)
	*p.target = float64(value)
	return err
}
func (f *FlagSet) Float64Var(p *float64, name string, value float64, usage string) {
	*p = value
	f.Var(&float64Value{target: p}, name, usage)
}
func (f *FlagSet) Float64(name string, value float64, usage string) *float64 {
	p := new(float64)
	f.Float64Var(p, name, value, usage)
	return p
}
func Float64Var(p *float64, name string, value float64, usage string) {
	CommandLine.Float64Var(p, name, value, usage)
}
func Float64(name string, value float64, usage string) *float64 {
	return CommandLine.Float64(name, value, usage)
}
