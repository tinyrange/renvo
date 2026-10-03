package rtg

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"strings"
	"testing"
)

// This is deliberately not one of the compiler's known architectures. A
// bundled operation must be selectable without adding generator-side policy.
func unfamiliarCompilerDefinition(t *testing.T, selector string, hook string) ResolveResult {
	t.Helper()
	source := "definition 1\nunit unfamiliar\nimplements direct_emitter_v1\narch unfamiliar {}\nextend arch unfamiliar {\ncompiler_selector = " + selector + "\ncompiler_bindings {\n"
	for _, operation := range compilerEmitterOperations {
		name := hook
		if len(operation.Parameters) != 0 {
			name += operation.Suffix
		}
		source += operation.Name + " = " + name + "\n"
	}
	source += "}\n}\ngo compiler {\nfunc " + hook + "(a *renvoAsm) {}\n"
	for _, operation := range compilerEmitterOperations {
		if len(operation.Parameters) != 0 {
			source += "func " + hook + operation.Suffix + operation.signature() + " {}\n"
		}
	}
	source += "}\n"
	document := Parse([]byte(source), "unfamiliar.rtg")
	if !document.Ok {
		t.Fatalf("parse unfamiliar definition: %#v", document.Diagnostics)
	}
	return ResolveResult{Document: document, Ok: true}
}

func TestBundledCompilerBindingsAreDefinitionSelected(t *testing.T) {
	first := unfamiliarCompilerDefinition(t, "selectedOne", "firstHook")
	second := unfamiliarCompilerDefinition(t, "selectedTwo", "secondHook")
	prefix := []byte(`package bindings

type context struct { renvoTargetArch int }
type renvoAsm struct { c *context; patchFailed bool }
func renvoNonNil(a *renvoAsm) {}
const selectedOne = 41
const selectedTwo = 73
func firstHook(a *renvoAsm) {}
func secondHook(a *renvoAsm) {}
`)
	for _, operation := range compilerEmitterOperations {
		if len(operation.Parameters) != 0 {
			for _, hook := range []string{"firstHook", "secondHook"} {
				prefix = append(prefix, "func "+hook+operation.Suffix+operation.signature()+" {}\n"...)
			}
		}
	}
	both := appendBundledCompilerBindings(prefix, []ResolveResult{first, second})
	if !both.Ok {
		t.Fatalf("generate: %#v", both.Diagnostics)
	}
	files := token.NewFileSet()
	file, err := parser.ParseFile(files, "bindings.go", both.Source, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := new(types.Config).Check("bindings", files, []*ast.File{file}, nil); err != nil {
		t.Fatal(err)
	}
	one := appendBundledCompilerBindings(nil, []ResolveResult{second})
	if !one.Ok || strings.Contains(string(one.Source), "selectedOne") || strings.Contains(string(one.Source), "firstHook") {
		t.Fatal("removing a bundled definition retained its selector or implementation")
	}
	if !strings.Contains(string(one.Source), "a.patchFailed = true") {
		t.Fatal("unknown selectors must fail instead of silently emitting another ISA")
	}
	duplicate := appendBundledCompilerBindings(nil, []ResolveResult{first, first})
	if duplicate.Ok || len(duplicate.Diagnostics) == 0 || duplicate.Diagnostics[0].Code != "RTG-COMPILER-006" {
		t.Fatalf("duplicate selector accepted: %#v", duplicate.Diagnostics)
	}
}

func TestCompilerBindingsRejectIncompleteAndInvalidContracts(t *testing.T) {
	definition := unfamiliarCompilerDefinition(t, "selectedOne", "firstHook")
	source := string(definition.Document.Source)
	cases := []struct{ name, old, replacement, code string }{
		{"missing", "copy_primary_to_secondary = firstHook\n", "", "RTG-COMPILER-005"},
		{"unknown", "copy_primary_to_secondary = firstHook", "unknown_operation = firstHook", "RTG-COMPILER-003"},
		{"duplicate", "copy_primary_to_secondary = firstHook", "copy_primary_to_secondary = firstHook\ncopy_primary_to_secondary = firstHook", "RTG-COMPILER-003"},
		{"missing hook", "copy_primary_to_secondary = firstHook", "copy_primary_to_secondary = missingHook", "RTG-COMPILER-004"},
		{"wrong parameter", "imm int", "imm bool", "RTG-COMPILER-004"},
		{"missing parameter", "(a *renvoAsm, imm int)", "(a *renvoAsm)", "RTG-COMPILER-004"},
		{"wrong argument", "a *renvoAsm", "a int", "RTG-COMPILER-004"},
		{"wrong result", "(a *renvoAsm) {}", "(a *renvoAsm) int { return 0 }", "RTG-COMPILER-004"},
		{"expression selector", "compiler_selector = selectedOne", "compiler_selector = selectedOne + 1", "RTG-COMPILER-001"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			document := Parse([]byte(strings.Replace(source, tc.old, tc.replacement, 1)), "invalid.rtg")
			if !document.Ok {
				t.Fatalf("expected structurally valid fixture: %#v", document.Diagnostics)
			}
			arch, ok := document.Declaration(DeclArch, "unfamiliar")
			if !ok {
				t.Fatal("fixture architecture missing")
			}
			diagnostics := validateCompilerBindings(document, arch)
			for _, diagnostic := range diagnostics {
				if diagnostic.Code == tc.code {
					return
				}
			}
			t.Fatalf("wanted %s, got %#v", tc.code, diagnostics)
		})
	}
}

func TestCompilerBindingExtensionsCannotOverrideMachineFacts(t *testing.T) {
	definition := unfamiliarCompilerDefinition(t, "selectedOne", "firstHook")
	source := strings.Replace(string(definition.Document.Source), "compiler_selector = selectedOne", "compiler_selector = selectedOne\nword_bits = 128", 1)
	document := Parse([]byte(source), "invalid-extension.rtg")
	if document.Ok {
		t.Fatal("compiler extension overrode ISA facts")
	}
	for _, diagnostic := range document.Diagnostics {
		if diagnostic.Code == "RTG-EXTEND-002" {
			return
		}
	}
	t.Fatalf("unexpected diagnostics: %#v", document.Diagnostics)
}

func TestMigratedCompilerOperationsAreNotHandwrittenInCore(t *testing.T) {
	data, err := os.ReadFile("../../backend/compiler_common_impl.go")
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.ParseFile(token.NewFileSet(), "core.go", data, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, declaration := range file.Decls {
		fn, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		for _, operation := range compilerEmitterOperations {
			if fn.Name.Name == "renvoAsm"+operation.Suffix {
				t.Errorf("%s must be supplied by generated backend bindings", fn.Name.Name)
			}
		}
	}
}
