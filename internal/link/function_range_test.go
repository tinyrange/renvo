package link

import (
	"bytes"
	"testing"

	"renvo.dev/internal/load"
	"renvo.dev/internal/unit"
)

func TestFunctionRangesLowerBeforeBackendInBothLinkPaths(t *testing.T) {
	files := []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\ngo 1.25\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
type Seq[V any] func(func(V)bool)
func Values[V any](v V)Seq[V]{return func(yield func(V)bool){yield(v)}}
func Find()(n int){for n:=range Values(42){return n};return -1}
func Catch()(ok bool){var cb func()=func(){ok=recover()!=nil};defer cb();panic("expected")}
func Owned()(n int){defer func(){n++}();for value:=range Values(42){defer func(v int){n+=v}(value)};return 1}
type Invoker interface{Invoke()}
func OwnedNil(v Invoker){for range Values(42){defer v.Invoke()}}
func Discard[V any](v V){recover();_=v}
func main(){Outer:for range Values(1){for range Values(2){continue Outer}};for _=range Values(3){};for range func()Seq[int]{return Values(1)}(){};_=Find();_=Catch();_=Owned();defer Discard(42);OwnedNil(nil)}
`)},
	}
	persistent := LinkBuildCore(buildFromFiles(t, files))
	transient := LinkBuildCoreTransient(buildFromFiles(t, files))
	if !persistent.Ok || !transient.Ok {
		t.Fatalf("link: persistent=%v transient=%v", persistent.Ok, transient.Ok)
	}
	if !bytes.Equal(persistent.Data, transient.Data) {
		t.Fatal("transient and retained compact units differ")
	}
	for _, transient := range []bool{false, true} {
		session := BeginPackageSession(buildFromFiles(t, files), transient)
		for !session.Step() {
		}
		got := session.Result()
		if !got.Ok || !bytes.Equal(got.Data, persistent.Data) {
			t.Fatalf("incremental link differs (transient=%v ok=%v)", transient, got.Ok)
		}
	}
	program := &persistent.Program
	for tok := 0; tok < len(program.Tokens); tok++ {
		if functionValueTokenEquals(program, tok, "range") {
			fn, ok := functionValueLexicalFunction(program, tok)
			if !ok || !functionValueHasPrefix(functionValueTokenText(program, fn.NameTok), "__renvo_range_defer_flush") {
				t.Fatal("function range reached compact backend")
			}
		}
		if functionValueTokenEquals(program, tok, "defer") && functionValueHasPrefix(functionValueTokenText(program, tok+1), "__renvo_call_") {
			t.Fatal("deferred dispatcher changed direct recover ownership")
		}
	}
	if !bytes.Contains(program.Text, []byte("range function continued iteration")) {
		t.Fatal("yield protocol checks missing")
	}
	if !bytes.Contains(program.Text, []byte("//renvo:defer-forward owner")) || !bytes.Contains(program.Text, []byte("//renvo:defer-forward call")) {
		t.Fatal("defer ownership ABI missing")
	}
}

func TestFunctionRangeDeferNeedsEnclosingFunctionOwnership(t *testing.T) {
	program := unit.Program{Package: "main"}
	source := []byte(`package main;func seq(yield func(int)bool){yield(1)};func main(){for n:=range seq{defer print(n)}}`)
	if !reparseFunctionValueProgram(&program, source, nil, len(source), -1) {
		t.Fatal("parse")
	}
	// The range expansion alone must still reject an unlowered registration.
	// Ownership lowering must precede moving this body into a callback.
	if lowerFunctionRangesCore(&program, false) {
		t.Fatal("defer was assigned to the iteration callback")
	}
	if !lowerFunctionRangeDefers(&program, false) || !lowerFunctionRangesCore(&program, false) {
		t.Fatal("owned defer lowering failed")
	}
}

func TestFunctionRangeDeferSnapshotsNamedParameters(t *testing.T) {
	program := unit.Program{Package: "main"}
	source := []byte(`package main
func add(p *string,s string){*p+=s}
func seq(y func(int)bool){y(42)}
func run(p *string){defer add(p,"o");for range seq{defer add(p,"d")};defer add(p,"a")}
func main(){s:="";run(&s)}
`)
	if !reparseFunctionValueProgram(&program, source, nil, len(source), -1) {
		t.Fatal("parse")
	}
	if !lowerFunctionRangeDefers(&program, false) {
		t.Fatal("owner")
	}
	if !lowerFunctionRangesCore(&program, false) {
		t.Fatal("range")
	}
	if !lowerFunctionValuesCore(&program, false) {
		t.Fatalf("values: %s", program.Text)
	}
	files := []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\ngo 1.25\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: source},
	}
	if got := LinkBuildCore(buildFromFiles(t, files)); !got.Ok {
		t.Fatal("whole link")
	}
}

func TestFunctionValueVariadicAndNamedResultBindingTypes(t *testing.T) {
	program := unit.Program{Package: "main"}
	source := []byte(`package main;func f(values ...int)(first,second string){p:=&first;flag:=false;truth:=bool(false);_ = p;_ = values;return "",""}`)
	if !reparseFunctionValueProgram(&program, source, nil, len(source), -1) {
		t.Fatal("parse")
	}
	fn := program.Funcs[0]
	for name, want := range map[string]string{"values": "[]int", "first": "string", "second": "string", "p": "*string", "flag": "bool", "truth": "bool"} {
		if got := functionValueEnclosingLocalType(&program, fn.BodyEnd-2, name); got != want {
			t.Fatalf("%s: got %q want %q", name, got, want)
		}
	}
}

func TestFunctionRangeInterfaceAndParenthesizedYieldTypes(t *testing.T) {
	for name, source := range map[string]string{
		"named":          `type Iterator interface{Each(func(int)bool)};func f(it Iterator){for range it.Each{}}`,
		"embedded_alias": `type Base interface{Each(func(int)bool)};type Alias=Base;type Iterator interface{Alias};func f(it Iterator){for range it.Each{}}`,
		"anonymous":      `func f(it interface{Each(func(int)bool)}){for range it.Each{}}`,
		"parenthesized":  `type Yield func(int)bool;type Seq func(((Yield)));func f(it Seq){for range it{}}`,
	} {
		t.Run(name, func(t *testing.T) {
			program := unit.Program{Package: "main"}
			text := []byte("package main;" + source)
			if !reparseFunctionValueProgram(&program, text, nil, len(text), -1) || !lowerFunctionRangesCore(&program, false) {
				t.Fatal("lowering failed")
			}
			for tok := range program.Tokens {
				if functionValueTokenEquals(&program, tok, "range") {
					t.Fatal("iterator range reached backend")
				}
			}
		})
	}
}
