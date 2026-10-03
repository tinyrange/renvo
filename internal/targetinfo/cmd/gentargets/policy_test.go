package main

import (
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A compiler profile must project the descriptor, not infer pointer widths,
// alignment, endianness, or arena policy from an ISA or OS identity.
func TestPolicyProjectionKeepsIndependentDescriptorFacts(t *testing.T) {
	descriptors := []sourceDescriptor{
		{Name: "linux/amd64", Constant: "known", BackendID: 1, OSID: 1, ISAID: 1,
			WordBits: 64, PointerBits: 64, CodePointerBits: 64, FunctionPointerBits: 64,
			MaxAlign: 8, Endian: "little", DefaultArena: 134217728,
			Runtime: []string{"read"}, Capabilities: []string{"hosted"},
			RuntimeNumbers: map[string]int{"read": 0, "write": 1, "read_at": 17,
				"write_at": 18, "open": 2, "close": 3, "chmod": 91, "exit": 60}},
		// Same identity numbers, deliberately different independent layout.
		{Name: "unfamiliar/segmented", Constant: "unfamiliar", BackendID: 2, OSID: 1, ISAID: 1,
			WordBits: 32, PointerBits: 16, CodePointerBits: 24, FunctionPointerBits: 32,
			MaxAlign: 256, Endian: "big", DefaultArena: 8192, Runtime: []string{"print"}},
	}
	path := filepath.Join(t.TempDir(), "policy.go")
	source := "package policy\n// BEGIN GENERATED TARGET REGISTRY\n// END GENERATED TARGET REGISTRY\n"
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	if err := updatePolicyProjection(path, descriptors); err != nil {
		t.Fatal(err)
	}
	generated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	files := token.NewFileSet()
	file, err := parser.ParseFile(files, path, generated, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := new(types.Config).Check("policy", files, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]int{
		"IntBits": 32, "PointerBits": 16, "CodePointerBits": 24,
		"FunctionPointerBits": 32, "Endian": 2, "RuntimeCaps": 1,
	} {
		value := pkg.Scope().Lookup("renvoTarget" + name + "Table").(*types.Const).Val()
		table := constant.StringVal(value)
		if len(table) != 3 || int(table[2]) != want {
			t.Errorf("%s = %q, want descriptor value %d at selector 2", name, table, want)
		}
	}
	// These fields are not bytes: alignment must not wrap at 256 and an arena
	// must not be recomputed from the word size or the familiar ISA identity.
	for _, want := range []string{
		"if target == unfamiliar {\n\t\treturn 256\n\t}",
		"if target == unfamiliar {\n\t\treturn 8192\n\t}",
	} {
		if !strings.Contains(string(generated), want) {
			t.Errorf("missing descriptor projection %q", want)
		}
	}
}
