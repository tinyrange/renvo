package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestBackendValueContractConstants(t *testing.T) {
	if rtgBackendValueSlotSize != 8 {
		t.Fatalf("backend value slot size = %d, want 8", rtgBackendValueSlotSize)
	}
	if rtgBackendStringWordCount != 2 || rtgBackendStringValueSize != 16 {
		t.Fatalf("string contract = %d words/%d bytes", rtgBackendStringWordCount, rtgBackendStringValueSize)
	}
	if rtgBackendSliceWordCount != 3 || rtgBackendSliceValueSize != 24 {
		t.Fatalf("slice contract = %d words/%d bytes", rtgBackendSliceWordCount, rtgBackendSliceValueSize)
	}
	if rtgBackendHiddenResultWordCount != 1 {
		t.Fatalf("hidden result contract = %d words, want 1", rtgBackendHiddenResultWordCount)
	}
	if rtgBackendRegisterCallWordCount != 6 {
		t.Fatalf("fast call word count = %d, want 6", rtgBackendRegisterCallWordCount)
	}
}

func TestSharedBackendAPIHasNoPhysicalRegisterNames(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "compiler_common_impl.go", nil, 0)
	if err != nil {
		t.Fatalf("parse shared backend: %v", err)
	}
	forbidden := []string{"Rax", "Rdx", "Rcx", "Rdi", "Rsi", "R8", "R9", "R10"}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		name := fn.Name.Name
		if strings.HasPrefix(name, "rtgAmd64") || strings.HasPrefix(name, "rtgWinAmd64") {
			continue
		}
		for _, physical := range forbidden {
			if strings.Contains(name, physical) {
				t.Errorf("shared function %s exposes physical register %s", name, physical)
			}
		}
	}
}
