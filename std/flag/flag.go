package flag

import (
	"errors"
	"io"
	"os"
	"strconv"
	"time"
)

type ErrorHandling int

const (
	ContinueOnError ErrorHandling = iota
	ExitOnError
	PanicOnError
)

type Value interface {
	String() string
	Set(string) error
}
type Getter interface {
	Value
	Get() any
}
type boolFlag interface{ IsBoolFlag() bool }
type Flag struct {
	Name, Usage string
	Value       Value
	DefValue    string
}

type FlagSet struct {
	name          string
	errorHandling ErrorHandling
	output        io.Writer
	Usage         func()
	formal        []*Flag
	actual        []*Flag
	args          []string
	parsed        bool
}

func NewFlagSet(name string, handling ErrorHandling) *FlagSet {
	f := &FlagSet{name: name, errorHandling: handling, output: os.Stderr}
	return f
}
func (f *FlagSet) SetOutput(output io.Writer)   { f.output = output }
func (f *FlagSet) Output() io.Writer            { return f.output }
func (f *FlagSet) ErrorHandling() ErrorHandling { return f.errorHandling }
func (f *FlagSet) Name() string                 { return f.name }
func (f *FlagSet) Parsed() bool                 { return f.parsed }
func (f *FlagSet) Args() []string               { return f.args }
func (f *FlagSet) Arg(i int) string {
	if i < 0 || i >= len(f.args) {
		return ""
	}
	return f.args[i]
}
func (f *FlagSet) NArg() int  { return len(f.args) }
func (f *FlagSet) NFlag() int { return len(f.actual) }
func (f *FlagSet) Lookup(name string) *Flag {
	for _, flag := range f.formal {
		if flag.Name == name {
			return flag
		}
	}
	return nil
}
func (f *FlagSet) add(flag *Flag) {
	if f.Lookup(flag.Name) != nil {
		panic("flag redefined: " + flag.Name)
	}
	f.formal = append(f.formal, flag)
}
func (f *FlagSet) Var(value Value, name, usage string) {
	f.add(&Flag{Name: name, Usage: usage, Value: value, DefValue: value.String()})
}
func (f *FlagSet) Set(name, value string) error {
	flag := f.Lookup(name)
	if flag == nil {
		return errors.New("no such flag -" + name)
	}
	if err := flag.Value.Set(value); err != nil {
		return err
	}
	for _, set := range f.actual {
		if set == flag {
			return nil
		}
	}
	f.actual = append(f.actual, flag)
	return nil
}
func (f *FlagSet) Parse(arguments []string) error {
	f.parsed = true
	f.args = nil
	for i := 0; i < len(arguments); i++ {
		arg := arguments[i]
		if arg == "--" {
			f.args = append(f.args, arguments[i+1:]...)
			return nil
		}
		if len(arg) < 2 || arg[0] != '-' || arg == "-" {
			f.args = append(f.args, arguments[i:]...)
			return nil
		}
		nameValue := arg[1:]
		if len(nameValue) > 0 && nameValue[0] == '-' {
			nameValue = nameValue[1:]
		}
		name, value, hasValue := splitValue(nameValue)
		flag := f.Lookup(name)
		if flag == nil {
			return f.fail(errors.New("flag provided but not defined: -" + name))
		}
		if !hasValue {
			if bf, ok := flag.Value.(boolFlag); ok && bf.IsBoolFlag() {
				value = "true"
			} else {
				i++
				if i >= len(arguments) {
					return f.fail(errors.New("flag needs an argument: -" + name))
				}
				value = arguments[i]
			}
		}
		if err := f.Set(name, value); err != nil {
			return f.fail(errors.New("invalid value for flag -" + name + ": " + value))
		}
	}
	return nil
}
func (f *FlagSet) fail(err error) error {
	if f.errorHandling == PanicOnError {
		panic(err)
	}
	if f.errorHandling == ExitOnError {
		os.Exit(2)
	}
	return err
}
func (f *FlagSet) Visit(fn func(*Flag)) {
	flags := sorted(f.actual)
	for _, flag := range flags {
		fn(flag)
	}
}
func (f *FlagSet) VisitAll(fn func(*Flag)) {
	flags := sorted(f.formal)
	for _, flag := range flags {
		fn(flag)
	}
}
func sorted(input []*Flag) []*Flag {
	out := append([]*Flag(nil), input...)
	for i := 1; i < len(out); i++ {
		item := out[i]
		j := i - 1
		for j >= 0 && out[j].Name > item.Name {
			out[j+1] = out[j]
			j--
		}
		out[j+1] = item
	}
	return out
}
func splitValue(s string) (string, string, bool) {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return s[:i], s[i+1:], true
		}
	}
	return s, "", false
}

func (f *FlagSet) PrintDefaults() {
	f.VisitAll(func(flag *Flag) {
		line := "  -" + flag.Name
		name, usage := UnquoteUsage(flag)
		if name != "" {
			line += " " + name
		}
		if usage != "" {
			line += "\n    " + usage
		}
		if flag.DefValue != "" && flag.DefValue != "false" && flag.DefValue != "0" {
			line += " (default " + flag.DefValue + ")"
		}
		line += "\n"
		if f.output != nil {
			f.output.Write([]byte(line))
		}
	})
}
func UnquoteUsage(flag *Flag) (name string, usage string) {
	usage = flag.Usage
	start := -1
	for i := 0; i < len(usage); i++ {
		if usage[i] == '`' {
			start = i
			break
		}
	}
	if start < 0 {
		return "value", usage
	}
	end := start + 1
	for end < len(usage) && usage[end] != '`' {
		end++
	}
	if end == len(usage) {
		return "value", usage
	}
	name = usage[start+1 : end]
	usage = usage[:start] + name + usage[end+1:]
	return name, usage
}

type stringValue struct{ p *string }

func (v *stringValue) String() string {
	if v == nil || v.p == nil {
		return ""
	}
	return *v.p
}
func (v *stringValue) Set(s string) error { *v.p = s; return nil }
func (v *stringValue) Get() any           { return *v.p }
func (f *FlagSet) String(name, value, usage string) *string {
	p := new(string)
	f.StringVar(p, name, value, usage)
	return p
}
func (f *FlagSet) StringVar(p *string, name, value, usage string) {
	*p = value
	f.Var(&stringValue{p}, name, usage)
}

type boolValue struct{ p *bool }

func (v *boolValue) String() string {
	if v == nil || v.p == nil {
		return "false"
	}
	return strconv.FormatBool(*v.p)
}
func (v *boolValue) Set(s string) error {
	value, err := strconv.ParseBool(s)
	if err == nil {
		*v.p = value
	}
	return err
}
func (v *boolValue) Get() any         { return *v.p }
func (v *boolValue) IsBoolFlag() bool { return true }
func (f *FlagSet) Bool(name string, value bool, usage string) *bool {
	p := new(bool)
	f.BoolVar(p, name, value, usage)
	return p
}
func (f *FlagSet) BoolVar(p *bool, name string, value bool, usage string) {
	*p = value
	f.Var(&boolValue{p}, name, usage)
}

type intValue struct{ p *int }

func (v *intValue) String() string {
	if v == nil || v.p == nil {
		return "0"
	}
	return strconv.Itoa(*v.p)
}
func (v *intValue) Set(s string) error {
	value, err := strconv.Atoi(s)
	if err == nil {
		*v.p = value
	}
	return err
}
func (v *intValue) Get() any { return *v.p }
func (f *FlagSet) Int(name string, value int, usage string) *int {
	p := new(int)
	f.IntVar(p, name, value, usage)
	return p
}
func (f *FlagSet) IntVar(p *int, name string, value int, usage string) {
	*p = value
	f.Var(&intValue{p}, name, usage)
}

type int64Value struct{ p *int64 }

func (v *int64Value) String() string {
	if v == nil || v.p == nil {
		return "0"
	}
	return strconv.FormatInt(*v.p, 10)
}
func (v *int64Value) Set(s string) error {
	value, err := strconv.ParseInt(s, 0, 64)
	if err == nil {
		*v.p = value
	}
	return err
}
func (v *int64Value) Get() any { return *v.p }
func (f *FlagSet) Int64(name string, value int64, usage string) *int64 {
	p := new(int64)
	f.Int64Var(p, name, value, usage)
	return p
}
func (f *FlagSet) Int64Var(p *int64, name string, value int64, usage string) {
	*p = value
	f.Var(&int64Value{p}, name, usage)
}

type uintValue struct{ p *uint }

func (v *uintValue) String() string {
	if v == nil || v.p == nil {
		return "0"
	}
	return strconv.FormatUint(uint64(*v.p), 10)
}
func (v *uintValue) Set(s string) error {
	value, err := strconv.ParseUint(s, 0, 64)
	if err == nil {
		*v.p = uint(value)
	}
	return err
}
func (v *uintValue) Get() any { return *v.p }
func (f *FlagSet) Uint(name string, value uint, usage string) *uint {
	p := new(uint)
	f.UintVar(p, name, value, usage)
	return p
}
func (f *FlagSet) UintVar(p *uint, name string, value uint, usage string) {
	*p = value
	f.Var(&uintValue{p}, name, usage)
}

var CommandLine = NewFlagSet("command", ExitOnError)

func Parse()                                           { CommandLine.Parse(os.Args[1:]) }
func Parsed() bool                                     { return CommandLine.Parsed() }
func Args() []string                                   { return CommandLine.Args() }
func Arg(i int) string                                 { return CommandLine.Arg(i) }
func NArg() int                                        { return CommandLine.NArg() }
func NFlag() int                                       { return CommandLine.NFlag() }
func Lookup(name string) *Flag                         { return CommandLine.Lookup(name) }
func Set(name, value string) error                     { return CommandLine.Set(name, value) }
func Var(value Value, name, usage string)              { CommandLine.Var(value, name, usage) }
func String(name, value, usage string) *string         { return CommandLine.String(name, value, usage) }
func StringVar(p *string, name, value, usage string)   { CommandLine.StringVar(p, name, value, usage) }
func Bool(name string, value bool, usage string) *bool { return CommandLine.Bool(name, value, usage) }
func BoolVar(p *bool, name string, value bool, usage string) {
	CommandLine.BoolVar(p, name, value, usage)
}
func Int(name string, value int, usage string) *int       { return CommandLine.Int(name, value, usage) }
func IntVar(p *int, name string, value int, usage string) { CommandLine.IntVar(p, name, value, usage) }
func Visit(fn func(*Flag))                                { CommandLine.Visit(fn) }
func VisitAll(fn func(*Flag))                             { CommandLine.VisitAll(fn) }

type float64Value struct{ p *float64 }

func (v *float64Value) String() string {
	if v == nil || v.p == nil {
		return "0"
	}
	return formatFloat(*v.p)
}
func (v *float64Value) Set(s string) error {
	value, err := parseFloat(s)
	if err == nil {
		*v.p = value
	}
	return err
}
func (v *float64Value) Get() any { return *v.p }
func (f *FlagSet) Float64(name string, value float64, usage string) *float64 {
	p := new(float64)
	f.Float64Var(p, name, value, usage)
	return p
}
func (f *FlagSet) Float64Var(p *float64, name string, value float64, usage string) {
	*p = value
	f.Var(&float64Value{p}, name, usage)
}
func parseFloat(s string) (float64, error) {
	if s == "" {
		return 0, errors.New("invalid syntax")
	}
	neg := false
	if s[0] == '-' || s[0] == '+' {
		neg = s[0] == '-'
		s = s[1:]
	}
	value := float64(0)
	digits := 0
	for len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
		value = value*10 + float64(s[0]-'0')
		s = s[1:]
		digits++
	}
	if len(s) > 0 && s[0] == '.' {
		s = s[1:]
		scale := float64(1)
		for len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
			scale *= 10
			value += float64(s[0]-'0') / scale
			s = s[1:]
			digits++
		}
	}
	if digits == 0 {
		return 0, errors.New("invalid syntax")
	}
	exp := 0
	expNeg := false
	if len(s) > 0 && (s[0] == 'e' || s[0] == 'E') {
		s = s[1:]
		if len(s) > 0 && (s[0] == '-' || s[0] == '+') {
			expNeg = s[0] == '-'
			s = s[1:]
		}
		if len(s) == 0 {
			return 0, errors.New("invalid syntax")
		}
		for len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
			exp = exp*10 + int(s[0]-'0')
			s = s[1:]
		}
	}
	if s != "" {
		return 0, errors.New("invalid syntax")
	}
	for exp > 0 {
		if expNeg {
			value /= 10
		} else {
			value *= 10
		}
		exp--
	}
	if neg {
		value = -value
	}
	return value, nil
}

type durationValue struct{ p *time.Duration }

func (v *durationValue) String() string {
	if v == nil || v.p == nil {
		return "0s"
	}
	return v.p.String()
}
func (v *durationValue) Set(s string) error {
	value, err := time.ParseDuration(s)
	if err == nil {
		*v.p = value
	}
	return err
}
func (v *durationValue) Get() any { return *v.p }
func (f *FlagSet) Duration(name string, value time.Duration, usage string) *time.Duration {
	p := new(time.Duration)
	f.DurationVar(p, name, value, usage)
	return p
}
func (f *FlagSet) DurationVar(p *time.Duration, name string, value time.Duration, usage string) {
	*p = value
	f.Var(&durationValue{p}, name, usage)
}
func Float64(name string, value float64, usage string) *float64 {
	return CommandLine.Float64(name, value, usage)
}
func Float64Var(p *float64, name string, value float64, usage string) {
	CommandLine.Float64Var(p, name, value, usage)
}
func Duration(name string, value time.Duration, usage string) *time.Duration {
	return CommandLine.Duration(name, value, usage)
}
func DurationVar(p *time.Duration, name string, value time.Duration, usage string) {
	CommandLine.DurationVar(p, name, value, usage)
}
func PrintDefaults() { CommandLine.PrintDefaults() }

func formatFloat(v float64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	whole := int64(v)
	fraction := v - float64(whole)
	out := strconv.FormatInt(whole, 10)
	if fraction > 0 {
		out += "."
		for i := 0; i < 9 && fraction > 0; i++ {
			fraction *= 10
			digit := int(fraction)
			out += string([]byte{byte('0' + digit)})
			fraction -= float64(digit)
		}
		for len(out) > 0 && out[len(out)-1] == '0' {
			out = out[:len(out)-1]
		}
	}
	if neg {
		return "-" + out
	}
	return out
}
