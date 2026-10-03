package check

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"renvo.dev/internal/syntax"
)

func TestReturnCountUsesFunctionLiteralBody(t *testing.T) {
	for _, tc := range []struct {
		name, function string
		valid          bool
	}{
		{"struct_parameter", `func F(){f:=func(v struct{Value int})int{return v.Value};_=f}`, true},
		{"struct_result", `func F(){f:=func()struct{Value int}{return struct{Value int}{42}};_=f}`, true},
		{"interface_parameter", `func F(){f:=func(v interface{Value()int})int{return v.Value()};_=f}`, true},
		{"nested_result", `func F(){f:=func()func()struct{Value int}{return func()struct{Value int}{return struct{Value int}{42}}};_=f}`, true},
		{"parent_return", `func F()int{f:=func(v struct{Value int})int{return v.Value};_=f;return 42}`, true},
		{"parent_missing_value", `func F()int{f:=func(v struct{Value int})int{return v.Value};_=f;return}`, false},
		{"parent_extra_values", `func F()int{f:=func(v struct{Value int})int{return v.Value};_=f;return 1,2}`, false},
		{"function_type_before_return", `func F()int{var f func(struct{Value int})int;_=f;return}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte("package main;" + tc.function + "\n")
			set := token.NewFileSet()
			goFile, err := parser.ParseFile(set, "main.go", source, 0)
			if err != nil {
				t.Fatal(err)
			}
			config := types.Config{}
			_, err = config.Check("example.com/case", set, []*ast.File{goFile}, nil)
			if (err == nil) != tc.valid {
				t.Fatalf("Go validity differs from test: %v", err)
			}
			file := syntax.ParseFile(source)
			if !file.Ok || len(file.Funcs) != 1 {
				t.Fatal("parse")
			}
			signature := buildFuncSignature(&file, &file.Funcs[0])
			code, tok := invalidReturnCount(&file, &file.Funcs[0], &signature)
			if (code == CheckOK) != tc.valid {
				t.Fatalf("valid=%v error=%d token=%d", tc.valid, code, tok)
			}
		})
	}
}
