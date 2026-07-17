package unit

import (
	"reflect"
	"testing"
)

func TestProgramIsTheExplicitSharedLinkingContract(t *testing.T) {
	typeOf := reflect.TypeOf(Program{})
	want := []string{
		"Package", "ImportPath", "Text", "Tokens", "Imports", "Symbols",
		"Decls", "Funcs", "TypeRefs", "Calls", "Refs", "Selectors",
	}
	if typeOf.NumField() != len(want) {
		t.Fatalf("Program has %d fields, want %d", typeOf.NumField(), len(want))
	}
	for i := 0; i < len(want); i++ {
		if got := typeOf.Field(i).Name; got != want[i] {
			t.Fatalf("Program field %d = %s, want %s", i, got, want[i])
		}
	}
}

func TestCoreProgramIsTheExplicitSerializedContract(t *testing.T) {
	typeOf := reflect.TypeOf(CoreProgram{})
	want := []string{"Package", "ImportPath", "Text", "Tokens", "Decls", "Funcs"}
	if typeOf.NumField() != len(want) {
		t.Fatalf("CoreProgram has %d fields, want %d", typeOf.NumField(), len(want))
	}
	for i := 0; i < len(want); i++ {
		if got := typeOf.Field(i).Name; got != want[i] {
			t.Fatalf("CoreProgram field %d = %s, want %s", i, got, want[i])
		}
	}
}

func TestSharedProgramTypesHaveNoRichFrontendOwnershipFields(t *testing.T) {
	for _, value := range []any{Import{}, Symbol{}, Call{}, NameRef{}, Selector{}, TypeRef{}} {
		typeOf := reflect.TypeOf(value)
		if _, ok := typeOf.FieldByName("OwnerKind"); ok {
			t.Fatalf("%s leaks checker ownership into the backend contract", typeOf.Name())
		}
		if _, ok := typeOf.FieldByName("OwnerIndex"); ok {
			t.Fatalf("%s leaks checker ownership into the backend contract", typeOf.Name())
		}
	}
}
