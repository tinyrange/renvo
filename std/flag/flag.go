// Package flag parses command-line flags.
package flag

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
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
	Name     string
	Usage    string
	Value    Value
	DefValue string
}
type ErrorHandling int

const (
	ContinueOnError ErrorHandling = iota
	ExitOnError
	PanicOnError
)

var ErrHelp = errors.New("flag: help requested")

type FlagSet struct {
	Usage    func()
	name     string
	handling ErrorHandling
	output   io.Writer
	formal   []*Flag
	actual   map[string]bool
	args     []string
	parsed   bool
}

func NewFlagSet(name string, handling ErrorHandling) *FlagSet {
	f := new(FlagSet)
	f.Init(name, handling)
	return f
}
func (f *FlagSet) Init(name string, handling ErrorHandling) { f.name = name; f.handling = handling }
func (f *FlagSet) Name() string                             { return f.name }
func (f *FlagSet) ErrorHandling() ErrorHandling             { return f.handling }
func (f *FlagSet) SetOutput(w io.Writer)                    { f.output = w }
func (f *FlagSet) Output() io.Writer {
	if f.output == nil {
		return os.Stderr
	}
	return f.output
}
func (f *FlagSet) Args() []string { return f.args }
func (f *FlagSet) NArg() int      { return len(f.args) }
func (f *FlagSet) Arg(i int) string {
	if i < 0 || i >= len(f.args) {
		return ""
	}
	return f.args[i]
}
func (f *FlagSet) Parsed() bool { return f.parsed }
func (f *FlagSet) NFlag() int   { return len(f.actual) }
func (f *FlagSet) Lookup(name string) *Flag {
	for _, item := range f.formal {
		if item.Name == name {
			return item
		}
	}
	return nil
}
func (f *FlagSet) Var(value Value, name, usage string) {
	if strings.HasPrefix(name, "-") || strings.Contains(name, "=") {
		panic("flag: invalid flag name " + name)
	}
	if f.Lookup(name) != nil {
		panic("flag redefined: " + name)
	}
	f.formal = append(f.formal, &Flag{Name: name, Usage: usage, Value: value, DefValue: value.String()})
}
func (f *FlagSet) Set(name, value string) error {
	item := f.Lookup(name)
	if item == nil {
		return errors.New("no such flag -" + name)
	}
	if err := item.Value.Set(value); err != nil {
		return err
	}
	if f.actual == nil {
		f.actual = make(map[string]bool)
	}
	f.actual[name] = true
	return nil
}
func (f *FlagSet) ordered() []*Flag {
	items := append([]*Flag{}, f.formal...)
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j].Name < items[j-1].Name; j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
	return items
}
func (f *FlagSet) VisitAll(fn func(*Flag)) {
	for _, item := range f.ordered() {
		fn(item)
	}
}
func (f *FlagSet) Visit(fn func(*Flag)) {
	for _, item := range f.ordered() {
		if f.actual[item.Name] {
			fn(item)
		}
	}
}
func (f *FlagSet) usage() {
	if f.Usage != nil {
		f.Usage()
		return
	}
	fmt.Fprintf(f.Output(), "Usage of %s:\n", f.name)
	f.PrintDefaults()
}
func (f *FlagSet) fail(err error) error {
	if err != ErrHelp {
		fmt.Fprintln(f.Output(), err)
	}
	f.usage()
	if f.handling == PanicOnError {
		panic(err)
	}
	if f.handling == ExitOnError {
		if err == ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(2)
		}
	}
	return err
}
func (f *FlagSet) Parse(arguments []string) error {
	f.parsed = true
	f.args = arguments
	for len(f.args) > 0 {
		argument := f.args[0]
		if len(argument) < 2 || argument[0] != '-' {
			return nil
		}
		f.args = f.args[1:]
		if argument == "--" {
			return nil
		}
		name := argument[1:]
		if name[0] == '-' {
			name = name[1:]
		}
		if name == "" || name[0] == '-' || name[0] == '=' {
			return f.fail(errors.New("bad flag syntax: " + argument))
		}
		value := ""
		hasValue := false
		if at := strings.IndexByte(name, '='); at >= 0 {
			value = name[at+1:]
			name = name[:at]
			hasValue = true
		}
		item := f.Lookup(name)
		if item == nil {
			if name == "h" || name == "help" {
				return f.fail(ErrHelp)
			}
			return f.fail(errors.New("flag provided but not defined: -" + name))
		}
		boolean, ok := item.Value.(boolFlag)
		if !hasValue {
			if ok && boolean.IsBoolFlag() {
				value = "true"
			} else {
				if len(f.args) == 0 {
					return f.fail(errors.New("flag needs an argument: -" + name))
				}
				value = f.args[0]
				f.args = f.args[1:]
			}
		}
		if err := f.Set(name, value); err != nil {
			return f.fail(errors.New("invalid value " + strconv.Quote(value) + " for flag -" + name + ": " + err.Error()))
		}
	}
	return nil
}

func UnquoteUsage(item *Flag) (name, usage string) {
	usage = item.Usage
	start := strings.IndexByte(usage, '`')
	if start >= 0 {
		end := strings.IndexByte(usage[start+1:], '`')
		if end >= 0 {
			end += start + 1
			name = usage[start+1 : end]
			usage = usage[:start] + name + usage[end+1:]
			return name, usage
		}
	}
	name = "value"
	switch item.Value.(type) {
	case *stringValue:
		name = "string"
	case *boolValue:
		name = ""
	case *intValue, *int64Value:
		name = "int"
	case *uintValue, *uint64Value:
		name = "uint"
	case *float64Value:
		name = "float"
	}
	return name, usage
}
func (f *FlagSet) PrintDefaults() {
	for _, item := range f.ordered() {
		name, usage := UnquoteUsage(item)
		line := "  -" + item.Name
		if name != "" {
			line += " " + name
		}
		if len(line) <= 4 {
			line += "\t"
		} else {
			line += "\n    \t"
		}
		line += strings.ReplaceAll(usage, "\n", "\n    \t")
		showDefault := item.DefValue != "" && item.DefValue != "0" && item.DefValue != "false" && item.DefValue != "-0"
		if _, ok := item.Value.(*stringValue); ok {
			showDefault = item.DefValue != ""
		}
		if showDefault {
			value := item.DefValue
			if _, ok := item.Value.(*stringValue); ok {
				value = strconv.Quote(value)
			}
			line += " (default " + value + ")"
		}
		fmt.Fprintln(f.Output(), line)
	}
}

func commandName() string {
	if len(os.Args) > 0 {
		return os.Args[0]
	}
	return ""
}

var CommandLine = NewFlagSet(commandName(), ExitOnError)
var Usage = func() {
	fmt.Fprintf(CommandLine.Output(), "Usage of %s:\n", CommandLine.Name())
	CommandLine.PrintDefaults()
}

func Parse()                              { CommandLine.Usage = Usage; CommandLine.Parse(os.Args[1:]) }
func Parsed() bool                        { return CommandLine.Parsed() }
func Args() []string                      { return CommandLine.Args() }
func NArg() int                           { return CommandLine.NArg() }
func Arg(i int) string                    { return CommandLine.Arg(i) }
func NFlag() int                          { return CommandLine.NFlag() }
func Lookup(name string) *Flag            { return CommandLine.Lookup(name) }
func Set(name, value string) error        { return CommandLine.Set(name, value) }
func Var(value Value, name, usage string) { CommandLine.Var(value, name, usage) }
func Visit(fn func(*Flag))                { CommandLine.Visit(fn) }
func VisitAll(fn func(*Flag))             { CommandLine.VisitAll(fn) }
func PrintDefaults()                      { CommandLine.PrintDefaults() }
