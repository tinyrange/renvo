package target

import (
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestC89SymbolsAreShortUniqueAndOrderIndependent(t *testing.T) {
	left, err := MangleC89Symbols([]string{"z/pkg.Value", "a/pkg.Value", "main.RunAll"}, C89MinimumExternalName)
	if err != nil {
		t.Fatal(err)
	}
	right, err := MangleC89Symbols([]string{"main.RunAll", "z/pkg.Value", "a/pkg.Value"}, C89MinimumExternalName)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(left, right) {
		t.Fatalf("input order changed C names: %#v != %#v", left, right)
	}
	seen := make(map[string]bool)
	for _, symbol := range left {
		if len(symbol.C) != C89MinimumExternalName || !strings.HasPrefix(symbol.C, "rg") {
			t.Fatalf("invalid short C name %q", symbol.C)
		}
		if seen[symbol.C] {
			t.Fatalf("duplicate short C name %q", symbol.C)
		}
		seen[symbol.C] = true
	}
}

func TestC89SymbolMappingIsDeterministicAtScale(t *testing.T) {
	names := make([]string, 1000)
	for i := range names {
		names[i] = "linked/unit/symbol/" + strconv.Itoa(i)
	}
	first, err := MangleC89Symbols(names, 6)
	if err != nil {
		t.Fatal(err)
	}
	second, err := MangleC89Symbols(names, 6)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("fixed linked unit produced different C symbol mappings")
	}
	if first[0].C != "rg0000" || first[len(first)-1].C == first[0].C {
		t.Fatalf("unexpected ordinal mapping: first=%q last=%q", first[0].C, first[len(first)-1].C)
	}
}

func TestC89SymbolMappingRejectsAmbiguousInput(t *testing.T) {
	if _, err := MangleC89Symbols([]string{"same", "same"}, 6); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate error = %v", err)
	}
	if _, err := MangleC89Symbols([]string{"ok"}, 5); err == nil || !strings.Contains(err.Error(), "between 6 and 31") {
		t.Fatalf("width error = %v", err)
	}
	if _, err := MangleC89Symbols([]string{""}, 6); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("empty-name error = %v", err)
	}
}
