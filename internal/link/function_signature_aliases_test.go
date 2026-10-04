package link

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"renvo.dev/internal/unit"
)

func TestFunctionSignatureAliasNormalization(t *testing.T) {
	for _, tc := range []struct{ name, source, want string }{
		{"alias", `type I=int;type F func(I)I;func main(){}`, `type F func(int)int`},
		{"universe_aliases", `type F func(byte)rune;func main(){}`, `type F func(uint8)int32`},
		{"shadowed_universe_target", `type uint8 string;type F func(byte)byte;func main(){var f F=func(v byte)byte{return v};_=f(42)}`, `type F func(byte)byte`},
		{"local_universe_target", `func main(){type int32 string;f:=func(v rune)rune{return v};_=f(42)}`, `func(v rune)rune`},
		{"nested_aliases", `type I=int;type F func([]I,map[I]*I)chan<-I;func main(){}`, `type F func([]int,map[int]*int)chan<-int`},
		{"defined", `type I int;type F func(I)I;func main(){}`, `type F func(I)I`},
		{"shadowed_universe", `type byte string;type F func(byte)byte;func main(){}`, `type F func(byte)byte`},
		{"local_alias", `func main(){type I=int;f:=func(v I)I{return v};_=f(42)}`, `func(v int)int`},
		{"local_universe_shadow", `func main(){type byte string;f:=func(v byte)byte{return v};_=f(byte("x"))}`, `func(v byte)byte`},
		{"captured_type_scope", `func main(){type byte string;make:=func()func(byte)byte{return func(v byte)byte{return v}};_=make()(byte("x"))}`, `func(v byte)byte`},
		{"closed_shadow_scope", `func main(){{type byte string;var b byte;_=b};f:=func(v byte)byte{return v};_=f(42)}`, `func(v uint8)uint8`},
		{"named_results", `type I=int;func main(){f:=func(v I)(result I){return v};_=f(42)}`, `func(v int)(result int)`},
		{"grouped_names", `type I=int;func main(){f:=func(a,b I)(x,y I){return a,b};_,_=f(1,2)}`, `func(a,b int)(x,y int)`},
		{"grouped_name_alias", `type a=int;func main(){f:=func(a,b,c int)int{return a+b+c};_=f(1,2,39)}`, `func(a,b,c int)int`},
		{"grouped_result_alias", `type a=int;func main(){f:=func()(a,b,c int){return 1,2,39};_,_,_=f()}`, `func()(a,b,c int)`},
		{"nested_function_type", `type I=int;type F func(func(I)I)func(I)I;func main(){}`, `type F func(func(int)int)func(int)int`},
		{"alias_chain_through_value_shadow", `type I=int;type J=I;func main(){int:=42;f:=func()J{return int};_=f()}`, `func()I`},
		{"alias_through_value_shadow", `type I=int;func main(){int:=42;f:=func()I{return int};_=f()}`, `func()I`},
		{"alias_through_type_shadow", `type I=int;func main(){type int string;v:=I(42);f:=func()I{return v};_=f()}`, `func()I`},
		{"local_alias_through_type_shadow", `type I=int;func main(){type int string;type T=I;v:=I(42);f:=func()T{return v};_=f()}`, `func()I`},
		{"aggregate_alias_through_type_shadow", `type A=struct{Value int};type B=struct{Value int};func main(){type A string;f:=func(v B)B{return v};_=f(B{42})}`, `func(v B)B`},
		{"aggregate_alias_through_value_shadow", `type A=struct{Value int};type B=struct{Value int};func main(){A:=42;f:=func(v B)B{return v};_=f(B{A})}`, `func(v B)B`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte("package main;" + tc.source + "\n")
			program := unit.Program{Package: "main"}
			if !reparseFunctionValueProgram(&program, source, nil, len(source), -1) {
				t.Fatal("parse")
			}
			if !lowerFunctionSignatureAliases(&program, false) {
				t.Fatal("normalize")
			}
			if !strings.Contains(functionValueCompactTypeText(string(program.Text)), functionValueCompactTypeText(tc.want)) {
				t.Fatalf("expected %s in %s", tc.want, program.Text)
			}
			set := token.NewFileSet()
			file, err := parser.ParseFile(set, "normalized.go", program.Text, 0)
			if err != nil {
				t.Fatal(err)
			}
			config := types.Config{}
			if _, err = config.Check("example.com/case", set, []*ast.File{file}, nil); err != nil {
				t.Fatal(err)
			}
		})
	}
}
