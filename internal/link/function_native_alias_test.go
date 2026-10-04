package link

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"renvo.dev/internal/unit"
	"testing"
)

func TestNativeCallbackAliasIdentityMatchesGo(t *testing.T) {
	for _, tc := range []struct{ name, decls, actual, want string }{
		{"scalar_alias", "type A=int;type B=int", "(v A)A", "func(B)B"},
		{"alias_chain", "type A=int;type B=A", "(v A)A", "func(B)B"},
		{"distinct_defined", "type A int;type B int", "(v A)A", "func(B)B"},
		{"same_defined", "type A int;type B=A", "(v A)A", "func(B)B"},
		{"aggregate_alias", "type A=struct{Value int};type B=struct{Value int}", "(v A)A", "func(B)B"},
		{"aggregate_distinct", "type A=struct{Value int};type B=struct{Other int}", "(v A)A", "func(B)B"},
		{"grouped_parameters", "type A=int;type B=int", "(a,b A)A", "func(B,B)B"},
		{"grouped_results", "type A=int;type B=int", "()(a,b A)", "func()(B,B)"},
		{"variadic", "type A=int;type B=int", "(v ...A)A", "func(...B)B"},
		{"variadic_not_slice", "type A=int;type B=int", "(v ...A)A", "func([]B)B"},
		{"pointer_alias", "type A=int;type B=int", "(v *A)*A", "func(*B)*B"},
		{"nested_callback", "type A=int;type B=int", "(v func(A)A)func(A)A", "func(func(B)B)func(B)B"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte("package main;" + tc.decls + ";type Want " + tc.want + ";func Actual" + tc.actual + "{panic(0)}\n")
			set := token.NewFileSet()
			file, err := parser.ParseFile(set, "case.go", source, 0)
			if err != nil {
				t.Fatal(err)
			}
			config := types.Config{}
			pkg, err := config.Check("example.com/native", set, []*ast.File{file}, nil)
			if err != nil {
				t.Fatal(err)
			}
			want := types.Identical(pkg.Scope().Lookup("Want").Type().Underlying(), pkg.Scope().Lookup("Actual").Type())
			program := unit.Program{Package: "main", ImportPath: "example.com/native"}
			if !reparseFunctionValueProgram(&program, source, nil, len(source), -1) || !lowerFunctionSignatureAliases(&program, false) {
				t.Fatal("parse/normalize")
			}
			actualToken, wantToken := -1, -1
			for _, fn := range program.Funcs {
				if functionValueTokenEquals(&program, fn.NameTok, "Actual") {
					actualToken = fn.NameTok
				}
			}
			for _, decl := range program.Decls {
				if ordinarySpanEquals(program.Text, decl.NameStart, decl.NameEnd, "Want") {
					wantToken = functionValueTokenAtSpan(&program, decl.NameStart, decl.NameEnd) + 1
				}
			}
			actual, _, ok := parseFunctionValueCallableSignature(&program, actualToken, "")
			if !ok {
				t.Fatal("native signature")
			}
			callback, _, ok := parseFunctionValueSignature(&program, wantToken, "")
			if !ok {
				t.Fatal("callback signature")
			}
			if got := functionValueSameShape(actual, callback); got != want {
				t.Fatalf("identity=%v Go=%v: %s", got, want, program.Text)
			}
		})
	}
}
