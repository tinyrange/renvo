package link

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	wireunit "renvo.dev/backend/unit"
	"strings"
	"testing"

	"renvo.dev/internal/load"
	"renvo.dev/internal/unit"
)

func TestPackageAliasesPreserveAuthoredBindings(t *testing.T) {
	for _, tc := range []struct{ name, dependency, root string }{
		{"type", `type Base struct{Value int}`, `type Base=callback.Base;type Renvop0_Base int;type Renvop0_Base_ string;func Id[T any](v T)T{return v};func appMain()int{b:=Id(callback.Base{42});x:=Id(Renvop0_Base(23));y:=Renvop0_Base_("suffix");if y!="suffix"{return 1};return b.Value+int(x)}`},
		{"local", `func Value()int{return 17}`, `func Value()int{return 19};func appMain()int{Renvop0_Value:=31;Renvop0_Value_:=37;return callback.Value()+Value()+Renvop0_Value+Renvop0_Value_}`},
		{"parameter", `func Value()int{return 17}`, `func Value()int{return 19};func Sum(Renvop0_Value int)int{return callback.Value()+Renvop0_Value};func appMain()int{return Sum(31)}`},
		{"init", `var Initialized int;func init(){Initialized=5}`, `func renvoi0_0(v int)int{return v};func main(){_=callback.Initialized+renvoi0_0(29)}`},
		{"root_init", `func Value()int{return 17}`, `var Initialized int;func init(){Initialized=5};func renvoi1_0(v int)int{return v};func main(){_=callback.Value()+Initialized+renvoi1_0(29)}`},
		{"process_init", `var Initialized int;func init(){Initialized=5};func renvo_runtime_SetProcess(args,env []string){}`, `func renvoi0_0(v int)int{return v};func main(){_=callback.Initialized+renvoi0_0(29)}`},
		{"private", `type base int;func Value()int{return int(base(17))}`, `type base int;type renvop0_base string;func appMain()int{v:=renvop0_base("private");if v!="private"{return 1};return callback.Value()+int(base(23))}`},
		{"unused_parameter", `func Unique()int{return 17}`, `func Sum(Unique int)int{return callback.Unique()};func appMain()int{return Sum(31)}`},
		{"grouped_parameters", `func Unique()int{return 17}`, `func Sum(Unique,Other int)int{return callback.Unique()+Other};func appMain()int{return Sum(31,37)}`},
		{"named_result", `func Unique()int{return 17}`, `func Sum()(Unique int){return callback.Unique()};func appMain()int{return Sum()}`},
		{"constant", `func Unique()int{return 17}`, `func appMain()int{const Unique=31;return callback.Unique()+Unique}`},
		{"type_binding", `type Value struct{N int}`, `func appMain()int{type Value int;x:=callback.Value{N:42};y:=Value(31);return x.N+int(y)}`},
		{"variable_binding", `var Value=17`, `func appMain()int{Value:=31;return callback.Value+Value}`},
		{"method_init", `type Value struct{N int};func(v Value)init()int{return v.N};func Read()int{return Value{42}.init()}`, `func main(){_=callback.Read()}`},
		{"root_method_init", `func Read()int{return 17}`, `type Value struct{N int};func(v Value)init()int{return v.N};func main(){_=callback.Read()+Value{42}.init()}`},
		{"generic_method_init", `type Box[T any]struct{Value T};func(b Box[T])init()T{return b.Value};func Read[T any](v T)T{return Box[T]{v}.init()}`, `func main(){_=callback.Read(42)}`},
		{"typed_initializer_shadow", `func Unique()int{return 17}`, `func Id[T any](v T)T{return v};func Unique()int{return 19};func main(){var Unique func()int=((Unique));_=func()int{return 31};print(Id(Unique)()+callback.Unique())}`},
		{"grouped_inferred_global", `func Unique()int{return 17};func Other()int{return 19}`, `var f,g=((callback.Unique)),(callback.Other);func Id[T any](v T)T{return v};func main(){_=func()int{return 31};print(Id(f)()+Id(g)())}`},
		{"grouped_local_specs", `func Unique()int{return 17};func Other()int{return 19}`, `func Id[T any](v T)T{return v};func main(){var(f=((callback.Unique));g=(callback.Other));_=func()int{return 31};print(Id(f)()+Id(g)())}`},
		{"grouped_local_typed_shadow", `func Unique()int{return 17}`, `func Id[T any](v T)T{return v};func Unique()int{return 19};func main(){var(Unique func()int=((Unique)));_=func()int{return 31};print(Id(Unique)()+callback.Unique())}`},
		{"grouped_local_inferred_shadow", `func Unique()int{return 17}`, `func Id[T any](v T)T{return v};func Unique()int{return 19};func main(){var(Unique=((Unique)));_=func()int{return 31};print(Id(Unique)()+callback.Unique())}`},
		{"grouped_local_multiple_names", `func Unique()int{return 17};func Other()int{return 19}`, `func Id[T any](v T)T{return v};func main(){var(f,g=((callback.Unique)),(callback.Other));_=func()int{return 31};print(Id(f)()+Id(g)())}`},
		{"grouped_global_specs", `func Unique()int{return 17};func Other()int{return 19}`, `var(f=((callback.Unique));g=(callback.Other));func Id[T any](v T)T{return v};func main(){_=func()int{return 31};print(Id(f)()+Id(g)())}`},
		{"grouped_global_multiple_names", `func Unique()int{return 17};func Other()int{return 19}`, `var(f,g=((callback.Unique)),(callback.Other));func Id[T any](v T)T{return v};func main(){_=func()int{return 31};print(Id(f)()+Id(g)())}`},
		{"grouped_tuple", `func Pair()(int,string){return 17,"yes"}`, `func Id[T any](v T)T{return v};func main(){var(a,b=callback.Pair());print(Id(a));print(Id(b))}`},
		{"parenthesized_local", `func Unique()int{return 17}`, `func Id[T any](v T)T{return v};func main(){f:=((callback.Unique));_=func()int{return 31};print(Id(f)())}`},
		{"parenthesized_global", `func Unique()int{return 17}`, `var f=((callback.Unique));func Id[T any](v T)T{return v};func main(){_=func()int{return 31};print(Id(f)())}`},
		{"parenthesized_group", `func Unique()int{return 17};func Other()int{return 19}`, `func Id[T any](v T)T{return v};func main(){f,g:=(callback.Unique),((callback.Other));_=func()int{return 31};print(Id(f)()+Id(g)())}`},
		{"defined_function_conversion", `func Unique()int{return 17}`, `func Id[T any](v T)T{return v};func main(){type Fn func()int;f:=Fn(((callback.Unique)));_=func()int{return 31};print(Id(f)())}`},
		{"initializer_shadow", `func Unique()int{return 17}`, `func Id[T any](v T)T{return v};func Unique()int{return 19};func main(){Unique:=((Unique));_=func()int{return 31};print(Id(Unique)()+callback.Unique())}`},
		{"function_result_call", `func Unique()int{return 17}`, `func Id[T any](v T)T{return v};func Factory()func()int{return ((callback.Unique))};func main(){f:=(Factory)();_=func()int{return 31};print(Id(f)())}`},
		{"inferred_callback", `func Unique()int{return 17}`, `func Id[T any](v T)T{return v};func main(){Unique:=func()int{return 31};f:=callback.Unique;print(Id(f)()+Unique())}`},
		{"inferred_assignment", `func Unique()int{return 17};func Other()int{return 19}`, `func Id[T any](v T)T{return v};func main(){Unique:=func()int{return 31};f:=callback.Unique;f=callback.Other;print(Id(f)()+Unique())}`},
		{"inferred_capture", `func Unique()int{return 17};func Id[T any](v T)T{return v}`, `func main(){f:=callback.Unique;read:=func()int{return f()};print(callback.Id(read)())}`},
		{"inferred_var", `func Unique()int{return 17}`, `func Id[T any](v T)T{return v};func main(){var f=callback.Unique;_=func()int{return 31};print(Id(f)())}`},
		{"inferred_grouped_var", `func Unique()int{return 17};func Other()int{return 19}`, `func Id[T any](v T)T{return v};func main(){var f,g=callback.Unique,callback.Other;_=func()int{return 31};print(Id(f)()+Id(g)())}`},
		{"inferred_grouped_short", `func Unique()int{return 17};func Other()int{return 19}`, `func Id[T any](v T)T{return v};func main(){f,g:=callback.Unique,callback.Other;_=func()int{return 31};print(Id(f)()+Id(g)())}`},
		{"embedded_key", `type Base struct{Value int};type Arg=struct{Base}`, `func main(){v:=callback.Arg{Base:callback.Base{42}};print(v.Base.Value)}`},
		{"dot_import_collision", `type Value struct{N int};var Counter=19;func Unique()int{return 17};func Id[T any](v T)T{return v}`, `type Alias=Value;var Initialized=Counter;func Param(Unique int)int{return Unique};func main(){type Value int;Counter:=31;print(Id(Unique())+Initialized+Counter+int(Value(3))+Param(5));var v Alias;_=v;var fields struct{Counter int;Unique int};fields.Counter,fields.Unique=1,2;print(fields.Counter+fields.Unique)}`},
		{"dot_import", `type Value struct{N int};func Unique()int{return 17}`, `func main(){print(Unique()+Value{42}.N)}`},
		{"closure_parameter", `func Unique()int{return 17}`, `func main(){f:=func(Unique int)int{return callback.Unique()};print(f(31))}`},
		{"grouped_const", `func Unique()int{return 17}`, `func appMain()int{const(Unique=31;Other=37);return callback.Unique()+Unique+Other}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dependency := "package callback;" + tc.dependency
			root := "package main;import \"example.com/case/callback\";" + tc.root
			if tc.name == "dot_import" || tc.name == "dot_import_collision" {
				root = "package main;import . \"example.com/case/callback\";" + tc.root
			}
			set := token.NewFileSet()
			dep, err := parser.ParseFile(set, "callback.go", dependency, 0)
			if err != nil {
				t.Fatal(err)
			}
			config := types.Config{}
			pkg, err := config.Check("example.com/case/callback", set, []*ast.File{dep}, nil)
			if err != nil {
				t.Fatal(err)
			}
			main, err := parser.ParseFile(set, "main.go", root, 0)
			if err != nil {
				t.Fatal(err)
			}
			config.Importer = aggregateIdentityImporter{pkg: pkg}
			if _, err = config.Check("example.com/case/cmd/app", set, []*ast.File{main}, nil); err != nil {
				t.Fatal(err)
			}
			files := []load.SourceFile{
				{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\ngo 1.25\n")},
				{Path: "/repo/case/callback/callback.go", Src: []byte(dependency)},
				{Path: "/repo/case/cmd/app/main.go", Src: []byte(root)},
			}
			for _, mode := range []string{"normal", "incremental", "transient", "incremental_transient", "cache"} {
				t.Run(mode, func(t *testing.T) {
					input := buildFromFiles(t, files)
					if mode == "cache" {
						for i := range input.Units {
							data, ok := unit.MarshalFrontendCache(input.Units[i].Program)
							if !ok {
								t.Fatal("cache encoding")
							}
							program, ok := unit.UnmarshalFrontendCache(data)
							if !ok {
								t.Fatal("cache decoding")
							}
							input.Units[i].Program = program
						}
					}
					var linked Result
					switch mode {
					case "incremental":
						linked = LinkBuildCoreIncremental(input)
					case "transient":
						linked = LinkBuildCoreTransient(input)
					case "incremental_transient":
						session := BeginPackageSession(input, true)
						for !session.Step() {
						}
						linked = session.Result()
					default:
						linked = LinkBuildCore(input)
					}
					if !linked.Ok {
						t.Fatalf("link: %d", linked.Error)
					}
					decoded, err := wireunit.Unmarshal(linked.Data)
					if err != nil {
						t.Fatal(err)
					}
					source := packageAliasGoSource(decoded)
					set := token.NewFileSet()
					file, err := parser.ParseFile(set, "linked.go", source, 0)
					if err != nil {
						t.Fatalf("parse: %v\n%s", err, source)
					}
					config := types.Config{}
					if _, err = config.Check("example.com/linked", set, []*ast.File{file}, nil); err != nil {
						t.Fatalf("check: %v\n%s", err, source)
					}
				})
			}
		})
	}
}

// Sanitizing mixed-language names can give two generated aliases the same
// spelling even when neither appears in authored Go identifiers.
func TestPackageAliasesAvoidSanitizationCollisions(t *testing.T) {
	programs := []unit.Program{{Symbols: []unit.Symbol{{Name: "C.Value"}, {Name: "C_Value"}, {Name: "Renvop0_C_Value_"}}}, {Symbols: []unit.Symbol{{Name: "C_Value"}}}}
	aliases := corePackageSymbolAliases(programs, 1, corePackageSymbolOffsets(programs))
	if aliases[0] != "Renvop0_C_Value" || aliases[1] != "Renvop0_C_Value__" {
		t.Fatalf("aliases = %q", aliases)
	}
}

// Compact units retain package clauses for source locations. Render their
// authoritative token stream as one Go package for the standard type checker.
func packageAliasGoSource(decoded wireunit.Program) string {
	program := unit.Program{Text: decoded.Text}
	for p := 0; p+8 <= len(decoded.Tokens); p += 8 {
		data := decoded.Tokens[p : p+8]
		start := int(data[1]) | int(data[2])<<8 | int(data[3])<<16
		size := int(data[4]) | int(data[5])<<8
		if int(data[0]) == unit.TokenOp {
			size = int(data[4])
		}
		line := int(data[6]) | int(data[7])<<8
		program.Tokens = append(program.Tokens, unit.MakeToken(int(data[0]), start, size, line))
	}
	var out strings.Builder
	out.WriteString("package main\n")
	line := -1
	braces := 0
	for i := 0; i < len(program.Tokens); i++ {
		token := program.Tokens[i]
		kind := token.KindLine & 255
		if kind == unit.TokenPackage {
			i++
			if i+1 < len(program.Tokens) && tokenAt(program, i+1) == ";" {
				i++
			}
			out.WriteByte('\n')
			continue
		}
		if kind == unit.TokenEOF {
			continue
		}
		text := tokenAt(program, i)
		if text == "." && i+2 < len(program.Tokens) && tokenAt(program, i+1) == "." && tokenAt(program, i+2) == "." && program.Tokens[i+1].Start == token.Start+1 && program.Tokens[i+2].Start == token.Start+2 {
			text = "..."
			i += 2
		}
		// The compact grammar permits empty top-level separators, including
		// ones left by removed generic declarations. Go uses newlines here.
		if braces == 0 && text == ";" {
			out.WriteByte('\n')
			continue
		}
		if text == "{" {
			braces++
		} else if text == "}" {
			braces--
		}
		if token.KindLine>>8 != line {
			out.WriteByte('\n')
			line = token.KindLine >> 8
		} else {
			out.WriteByte(' ')
		}
		out.WriteString(text)
	}
	out.WriteByte('\n')
	return out.String()
}
