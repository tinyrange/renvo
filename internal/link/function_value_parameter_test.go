package link

import (
	"renvo.dev/internal/unit"
	"testing"
)

func TestFunctionValueTypeIdentityIgnoresParameterAndResultNames(t *testing.T) {
	signatures := []functionValueSignature{{name: "callback", paramTypes: []string{"*T", "int"}, resultTypes: []string{"bool", "int"}}}
	for _, typ := range []string{
		"func(*T, int) (bool, int)",
		"func(value *T, count int) (ok bool, total int)",
	} {
		if functionValueSignatureByTypeText(signatures, typ) != 0 {
			t.Fatalf("did not match %s", typ)
		}
	}
	if functionValueSignatureByTypeText(signatures, "func(value *T, count string) (bool, int)") != -1 {
		t.Fatal("matched different parameter type")
	}
}

func TestFunctionValueParameterSignatureIgnoresNames(t *testing.T) {
	source := []byte(`package main
type T struct {}
func run(name string, callback func(value *T)) {}
func grouped(first, second func(value *T)) {}
func main() {}
`)
	program := unit.Program{Package: "main"}
	if !reparseFunctionValueProgram(&program, source, nil, len(source), -1) {
		t.Fatal("reparse")
	}
	signatures := []functionValueSignature{{name: "callback", paramTypes: []string{"*T"}}}
	for _, fn := range program.Funcs {
		name := functionValueTokenText(&program, fn.NameTok)
		types := functionValueFunctionParamTypes(&program, fn)
		if name == "run" {
			if functionValueParameterSignature(&program, fn, 0, types[0], signatures) != -1 {
				t.Fatal("string parameter matched callback")
			}
			if functionValueParameterSignature(&program, fn, 1, types[1], signatures) != 0 {
				t.Fatal("named callback parameter did not match")
			}
		}
		if name == "grouped" {
			for i := range types {
				if functionValueParameterSignature(&program, fn, i, types[i], signatures) != 0 {
					t.Fatal("grouped callback parameter did not match")
				}
			}
		}
	}
}
