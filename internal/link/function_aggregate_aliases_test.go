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

func TestAggregateCallbackAliasOwnerIdentity(t *testing.T) {
	for _, tc := range []struct {
		name, left, right string
		samePackage       bool
	}{
		{"public_struct", `struct{Value int}`, `struct{Value int}`, false},
		{"grouped_fields", `struct{A,B int}`, `struct{A int;B int}`, false},
		{"reordered_fields", `struct{A,B int}`, `struct{B int;A int}`, false},
		{"nested_grouped_fields", `struct{Value []struct{A,B int}}`, `struct{Value []struct{A int;B int}}`, false},
		{"equivalent_tag_spelling", "struct{Value int `json:\"value\"`}", `struct{Value int "json:\"value\""}`, false},
		{"empty_tag", "struct{Value int ``}", `struct{Value int}`, false},
		{"different_field", `struct{Value int}`, `struct{Other int}`, false},
		{"different_tag", "struct{Value int `left`}", "struct{Value int `right`}", false},
		{"private_same_owner", `struct{value int}`, `struct{value int}`, true},
		{"private_different_owner", `struct{value int}`, `struct{value int}`, false},
		{"grouped_private_same_owner", `struct{A,b int}`, `struct{A,b int}`, true},
		{"grouped_private_different_owner", `struct{A,b int}`, `struct{A,b int}`, false},
		{"unicode_public", `struct{É int}`, `struct{É int}`, false},
		{"unicode_private", `struct{é int}`, `struct{é int}`, false},
		{"nested_public", `struct{Value []struct{Value int}}`, `struct{Value []struct{Value int}}`, false},
		{"nested_private", `struct{Value []struct{value int}}`, `struct{Value []struct{value int}}`, false},
		{"public_method", `interface{Value()int}`, `interface{Value()int}`, false},
		{"reordered_methods", `interface{First()int;Second()int}`, `interface{Second()int;First()int}`, false},
		{"different_method_set", `interface{First()int;Second()int}`, `interface{First()int;Third()int}`, false},
		{"grouped_private_fields", `struct{A,b int}`, `struct{A int;b int}`, true},
		{"grouped_private_fields_owner", `struct{A,b int}`, `struct{A int;b int}`, false},
		{"method_result_tuple", `interface{Value()int}`, `interface{Value()(int)}`, false},
		{"method_parameter_names", `interface{Value(value int)int}`, `interface{Value(int)(result int)}`, false},
		{"private_method_same_owner", `interface{value()int}`, `interface{value()int}`, true},
		{"private_method_different_owner", `interface{value()int}`, `interface{value()int}`, false},
		{"nested_method_private", `interface{Value()struct{value int}}`, `interface{Value()struct{value int}}`, false},
		{"callback_private_parameter", `struct{Value func(struct{value int})int}`, `struct{Value func(struct{value int})int}`, false},
		{"callback_parameter_names", `struct{Value func(value int)int}`, `struct{Value func(int)int}`, false},
		{"callback_grouped_parameter_fields", `struct{Value func(struct{A,B int})int}`, `struct{Value func(struct{A int;B int})int}`, false},
		{"method_reordered_result_methods", `interface{Value()interface{First()int;Second()int}}`, `interface{Value()interface{Second()int;First()int}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			left := "type A = " + tc.left + ";\n"
			right := "type B = " + tc.right + ";\n"
			leftType := aggregateCallbackGoAlias(t, "example.com/left", left, "A")
			rightType := aggregateCallbackGoAlias(t, "example.com/right", right, "B")
			if tc.samePackage {
				set := token.NewFileSet()
				file, err := parser.ParseFile(set, "main.go", "package main;"+left+right, 0)
				if err != nil {
					t.Fatal(err)
				}
				config := types.Config{}
				pkg, err := config.Check("example.com/left", set, []*ast.File{file}, nil)
				if err != nil {
					t.Fatal(err)
				}
				leftType, rightType = pkg.Scope().Lookup("A").Type(), pkg.Scope().Lookup("B").Type()
			}
			want := types.Identical(leftType, rightType)
			prefix := "package main;\n"
			source := []byte(prefix + left + right + "type F func(B)B;func main(){}\n")
			program := unit.Program{Package: "main", ImportPath: "example.com/left"}
			if !reparseFunctionValueProgram(&program, source, nil, len(source), -1) {
				t.Fatal("parse")
			}
			rightPath := "example.com/right"
			if tc.samePackage {
				rightPath = "example.com/left"
			}
			program.Packages = []unit.PackageInfo{
				{ImportPath: "example.com/left", TextStart: len(prefix), TextEnd: len(prefix) + len(left)},
				{ImportPath: rightPath, TextStart: len(prefix) + len(left), TextEnd: len(prefix) + len(left) + len(right)},
			}
			if !lowerFunctionSignatureAliases(&program, false) {
				t.Fatal("normalize")
			}
			got := strings.Contains(functionValueCompactTypeText(string(program.Text)), "typeFfunc(A)A")
			if got != want {
				t.Fatalf("matching alias=%v Go identity=%v: %s", got, want, program.Text)
			}
		})
	}
}

func aggregateCallbackGoAlias(t *testing.T, path string, declaration string, name string) types.Type {
	t.Helper()
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, "main.go", "package main;"+declaration, 0)
	if err != nil {
		t.Fatal(err)
	}
	config := types.Config{}
	pkg, err := config.Check(path, set, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return pkg.Scope().Lookup(name).Type()
}
