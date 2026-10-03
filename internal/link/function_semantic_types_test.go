package link

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"renvo.dev/internal/unit"
	"strings"
	"testing"
)

func TestAggregateAliasDeclarationsMatchGoIdentity(t *testing.T) {
	for _, tc := range []struct{ name, source string }{
		{"scalar_alias", `type I=int;type A=struct{Value I};type B=struct{Value int}`},
		{"alias_chain", `type I=int;type J=I;type A=struct{Value J};type B=struct{Value int}`},
		{"universe_byte", `type A=struct{Value byte};type B=struct{Value uint8}`},
		{"universe_rune", `type A=struct{Value rune};type B=struct{Value int32}`},
		{"shadowed_byte_target", `type uint8 string;type A=struct{Value byte};type B=struct{Value uint8}`},
		{"defined_scalar", `type I int;type A=struct{Value I};type B=struct{Value int}`},
		{"distinct_definitions", `type I int;type J int;type A=struct{Value I};type B=struct{Value J}`},
		{"same_defined_target", `type I int;type J=I;type A=struct{Value I};type B=struct{Value J}`},
		{"container_aliases", `type I=int;type A=struct{Value map[I]chan<-*I};type B=struct{Value map[int]chan<-*int}`},
		{"function_aliases", `type I=int;type A=struct{Value func(I)I};type B=struct{Value func(int)int}`},
		{"function_group_names", `type I=int;type A=struct{Value func(I,b,c int)(x,y,z int)};type B=struct{Value func(int,int,int)(int,int,int)}`},
		{"variadic", `type I=int;type A=struct{Value func(...I)};type B=struct{Value func(...int)}`},
		{"variadic_slice", `type I=int;type A=struct{Value func(...I)};type B=struct{Value func([]int)}`},
		{"literal_array_lengths", `type A=struct{Value [1+1]int};type B=struct{Value [2]int}`},
		{"literal_wide_array_lengths", `type A=struct{Value [(1<<50)-(1<<50)+2]int};type B=struct{Value [2]int}`},
		{"literal_array_bitwise", `type A=struct{Value [(9&^1)/2]int};type B=struct{Value [4]int}`},
		{"nested_private_alias", `type I=struct{value int};type A=struct{Value I};type B=struct{Value struct{value int}}`},
		{"embedded_interface", `type I interface{First()int};type A=interface{I;Second()int};type B=interface{Second()int;First()int}`},
		{"embedded_alias_interface", `type I interface{First()int};type J=I;type A=interface{J};type B=interface{First()int}`},
		{"duplicate_embedded_methods", `type I interface{First()int};type J interface{First()int};type A=interface{I;J};type B=interface{First()int}`},
		{"embedded_empty_interface", `type A=interface{any};type B=interface{}`},
		{"any_field", `type A=struct{Value any};type B=struct{Value interface{}}`},
		{"embedded_error", `type A=interface{error};type B=interface{Error()string}`},
		{"embedded_error_alias_result", `type S=string;type A=interface{error};type B=interface{Error()S}`},
		{"embedded_error_shadow_result", `type string int;type A=interface{error};type B=interface{Error()string}`},
		{"named_interface_identity", `type I interface{First()int};type A=struct{Value I};type B=struct{Value interface{First()int}}`},
		{"local_alias", `func left(){type I=int;type A=struct{Value I};var a A;_=a};type B=struct{Value int}`},
		{"local_distinct_definitions", `func left(){type N int;type A=struct{Value N};var a A;_=a};func right(){type N int;type B=struct{Value N};var b B;_=b}`},
		{"local_same_definition", `func main(){type N int;type I=N;type A=struct{Value I};type B=struct{Value N};var a A;var b B;_,_=a,b}`},
		{"local_shadow", `type N int;type A=struct{Value N};func main(){type N int;type B=struct{Value N};var b B;_=b}`},
		{"alias_carries_global_target", `type N int;type I=N;type A=struct{Value N};func main(){type N string;type B=struct{Value I};var b B;var n N;_,_=b,n}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte("package main;" + tc.source + "\n")
			set := token.NewFileSet()
			file, err := parser.ParseFile(set, "case.go", source, 0)
			if err != nil {
				t.Fatal(err)
			}
			info := types.Info{Defs: make(map[*ast.Ident]types.Object)}
			config := types.Config{}
			if _, err = config.Check("example.com/case", set, []*ast.File{file}, &info); err != nil {
				t.Fatal(err)
			}
			var a, b types.Type
			for id, obj := range info.Defs {
				if obj == nil {
					continue
				}
				if id.Name == "A" {
					a = obj.Type()
				}
				if id.Name == "B" {
					b = obj.Type()
				}
			}
			if a == nil || b == nil {
				t.Fatal("missing oracle types")
			}
			program := unit.Program{Package: "main", ImportPath: "example.com/case"}
			if !reparseFunctionValueProgram(&program, source, nil, len(source), -1) {
				t.Fatal("parse")
			}
			left, right := -1, -1
			for i := 1; i+2 < len(program.Tokens); i++ {
				if !functionValueTokenEquals(&program, i-1, "type") {
					continue
				}
				if functionValueTokenEquals(&program, i, "A") {
					left = i + 2
				}
				if functionValueTokenEquals(&program, i, "B") {
					right = i + 2
				}
			}
			if left < 0 || right < 0 {
				t.Fatal("missing parsed types")
			}
			got := functionValueSameSemanticType(&program, left, functionValueTypeEnd(&program, left), right, functionValueTypeEnd(&program, right), 0)
			if want := types.Identical(a, b); got != want {
				t.Fatalf("identity=%v, Go=%v: %s", got, want, source)
			}
		})
	}
}

type aggregateIdentityImporter struct{ pkg *types.Package }

func (i aggregateIdentityImporter) Import(path string) (*types.Package, error) { return i.pkg, nil }

func TestAggregateAliasesRetainEmbeddedMemberOwners(t *testing.T) {
	for _, tc := range []struct{ name, left, right string }{
		{"private_interface_imported_owner", `type Base interface{secret()int};type A=interface{Base}`, `type B=interface{left.Base}`},
		{"private_interface_new_owner", `type Base interface{secret()int};type A=interface{Base}`, `type B=interface{secret()int}`},
		{"private_field_imported_owner", `type Inner=struct{value int};type A=struct{Value Inner}`, `type B=struct{Value left.Inner}`},
		{"private_field_new_owner", `type Inner=struct{value int};type A=struct{Value Inner}`, `type B=struct{Value struct{value int}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			set := token.NewFileSet()
			lf, err := parser.ParseFile(set, "left.go", "package left;"+tc.left, 0)
			if err != nil {
				t.Fatal(err)
			}
			config := types.Config{}
			lp, err := config.Check("example.com/left", set, []*ast.File{lf}, nil)
			if err != nil {
				t.Fatal(err)
			}
			rf, err := parser.ParseFile(set, "right.go", "package right;import \"example.com/left\";var _ left.A;"+tc.right, 0)
			if err != nil {
				t.Fatal(err)
			}
			config.Importer = aggregateIdentityImporter{pkg: lp}
			rp, err := config.Check("example.com/right", set, []*ast.File{rf}, nil)
			if err != nil {
				t.Fatal(err)
			}
			want := types.Identical(lp.Scope().Lookup("A").Type(), rp.Scope().Lookup("B").Type())
			left := "type BaseUnused int;" + tc.left + ";\n"
			right := strings.ReplaceAll(tc.right, "left.", "") + ";\n"
			prefix := "package main;\n"
			source := []byte(prefix + left + right + "type F func(B)B;func main(){}\n")
			program := unit.Program{Package: "main", ImportPath: "example.com/right"}
			if !reparseFunctionValueProgram(&program, source, nil, len(source), -1) {
				t.Fatal("parse")
			}
			program.Packages = []unit.PackageInfo{
				{ImportPath: "example.com/left", TextStart: len(prefix), TextEnd: len(prefix) + len(left)},
				{ImportPath: "example.com/right", TextStart: len(prefix) + len(left), TextEnd: len(prefix) + len(left) + len(right)},
			}
			if !lowerFunctionSignatureAliases(&program, false) {
				t.Fatal("normalize")
			}
			got := strings.Contains(functionValueCompactTypeText(string(program.Text)), "typeFfunc(A)A")
			if got != want {
				t.Fatalf("identity=%v Go=%v: %s", got, want, program.Text)
			}
		})
	}
}
