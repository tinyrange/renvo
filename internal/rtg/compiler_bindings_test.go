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

// Exercise whole generated packages, including the difficult cases where an
// entrypoint must remain callable even though another binding projects it.
func TestBundledCompilerBindingBodyProjection(t *testing.T) {
	cases := []struct {
		name     string
		rewrite  func(string, string) string
		retained bool
	}{
		{"private", func(source, hook string) string {
			return strings.Replace(source, "func "+hook+"(a *renvoAsm) {}",
				"func "+hook+"(a *renvoAsm) { if a.patchFailed { return }; a.patchFailed = true }", 1)
		}, false},
		{"terminal return with comment", func(source, hook string) string {
			return strings.Replace(source, "func "+hook+"(a *renvoAsm) {}",
				"func "+hook+"(a *renvoAsm) { a.patchFailed = true; return /* done */ }", 1)
		}, false},
		{"explicit return semicolon", func(source, hook string) string {
			return strings.Replace(source, "func "+hook+"(a *renvoAsm) {}",
				"func "+hook+"(a *renvoAsm) { a.patchFailed = true; return; }", 1)
		}, false},
		{"called by helper", func(source, hook string) string {
			return source + "\ngo compiler { func " + hook + "Caller(a *renvoAsm) { " + hook + "(a) } }\n"
		}, true},
		{"recursive", func(source, hook string) string {
			return strings.Replace(source, "func "+hook+"(a *renvoAsm) {}",
				"func "+hook+"(a *renvoAsm) { if a.patchFailed { a.patchFailed = false; "+hook+"(a) } }", 1)
		}, true},
		{"labels have function scope", func(source, hook string) string {
			return strings.Replace(source, "func "+hook+"(a *renvoAsm) {}",
				"func "+hook+"(a *renvoAsm) { again: if a.patchFailed { a.patchFailed = false; goto again } }", 1)
		}, true},
		{"renamed parameters", func(source, hook string) string {
			return strings.ReplaceAll(source, "a *renvoAsm", "receiver *renvoAsm")
		}, true},
		{"one hook with different binding parameter names", func(source, hook string) string {
			return strings.Replace(source, "store_primary_stack = "+hook+"StorePrimaryStack",
				"store_primary_stack = "+hook+"PushImm", 1)
		}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var definitions []ResolveResult
			for _, pair := range [][2]string{{"selectedOne", "firstHook"}, {"selectedTwo", "secondHook"}} {
				fixture := unfamiliarCompilerDefinition(t, pair[0], pair[1])
				document := Parse([]byte(tc.rewrite(string(fixture.Document.Source), pair[1])), "projection.rtg")
				if !document.Ok {
					t.Fatalf("parse: %#v", document.Diagnostics)
				}
				definitions = append(definitions, ResolveResult{Document: document, Ok: true})
			}
			prefix := []byte(`package bindings
 type context struct { renvoTargetArch int }
 type renvoAsm struct { c *context; patchFailed bool }
 func renvoNonNil(a *renvoAsm) {}
 const selectedOne = 41
 const selectedTwo = 73
`)
			for _, definition := range definitions {
				prefix = appendCompilerGoBlocks(prefix, definition.Document)
			}
			retained := strings.Contains(string(prefix), "func firstHook(")
			if retained != tc.retained {
				t.Fatalf("entrypoint retained = %v, want %v", retained, tc.retained)
			}
			if tc.name == "one hook with different binding parameter names" &&
				!strings.Contains(string(prefix), "func firstHookPushImm(") {
				t.Fatal("pruned a hook still used by a noncanonical binding")
			}
			generated := appendBundledCompilerBindings(prefix, definitions)
			if !generated.Ok {
				t.Fatalf("generate: %#v", generated.Diagnostics)
			}
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "bindings.go", generated.Source, 0)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := new(types.Config).Check("bindings", fset, []*ast.File{file}, nil); err != nil {
				t.Fatal(err)
			}
			ast.Inspect(file, func(node ast.Node) bool {
				block, ok := node.(*ast.BlockStmt)
				if !ok {
					return true
				}
				for i := 1; i < len(block.List); i++ {
					_, previousReturn := block.List[i-1].(*ast.ReturnStmt)
					_, nextReturn := block.List[i].(*ast.ReturnStmt)
					if previousReturn && nextReturn {
						t.Error("projected hook contains consecutive returns rejected by compact source lowering")
					}
				}
				return true
			})
		})
	}
}

// Identical implementations may share code even across unrelated selectors;
// a different body must remain separate and unknown selectors must still fail.
func TestBundledCompilerBindingBodyGroups(t *testing.T) {
	var definitions []ResolveResult
	for i, pair := range [][2]string{{"selectedOne", "firstHook"}, {"selectedTwo", "secondHook"}, {"selectedThree", "thirdHook"}} {
		fixture := unfamiliarCompilerDefinition(t, pair[0], pair[1])
		body := "a.patchFailed = false"
		if i == 1 {
			body = "a.patchFailed = true"
		}
		source := strings.Replace(string(fixture.Document.Source), "func "+pair[1]+"(a *renvoAsm) {}",
			"func "+pair[1]+"(a *renvoAsm) { "+body+" }", 1)
		document := Parse([]byte(source), "groups.rtg")
		if !document.Ok {
			t.Fatal(document.Diagnostics)
		}
		definitions = append(definitions, ResolveResult{Document: document, Ok: true})
	}
	generated := appendBundledCompilerBindings([]byte("package bindings\n"), definitions)
	if !generated.Ok {
		t.Fatal(generated.Diagnostics)
	}
	file, err := parser.ParseFile(token.NewFileSet(), "bindings.go", generated.Source, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, declaration := range file.Decls {
		fn := declaration.(*ast.FuncDecl)
		if fn.Name.Name != "renvoAsmCopyPrimaryToSecondary" {
			continue
		}
		if len(fn.Body.List) != 4 {
			t.Fatalf("want guard, two body groups, and unknown-selector failure; got %d statements", len(fn.Body.List))
		}
		first := fn.Body.List[1].(*ast.IfStmt)
		condition := first.Cond.(*ast.BinaryExpr)
		if condition.Op != token.LOR ||
			condition.X.(*ast.BinaryExpr).Y.(*ast.Ident).Name != "selectedOne" ||
			condition.Y.(*ast.BinaryExpr).Y.(*ast.Ident).Name != "selectedThree" {
			t.Fatal("nonadjacent equal bodies lost their explicit selectors")
		}
		second := fn.Body.List[2].(*ast.IfStmt)
		if second.Cond.(*ast.BinaryExpr).Y.(*ast.Ident).Name != "selectedTwo" {
			t.Fatal("different implementation was grouped")
		}
		for i, block := range []*ast.BlockStmt{first.Body, second.Body} {
			value := block.List[0].(*ast.AssignStmt).Rhs[0].(*ast.Ident).Name
			if value != []string{"false", "true"}[i] {
				t.Fatal("selector body changed")
			}
			if _, ok := block.List[1].(*ast.ReturnStmt); !ok {
				t.Fatal("selected body falls through to failure")
			}
		}
		failure := fn.Body.List[3].(*ast.AssignStmt)
		if failure.Lhs[0].(*ast.SelectorExpr).Sel.Name != "patchFailed" || failure.Rhs[0].(*ast.Ident).Name != "true" {
			t.Fatal("unknown selector no longer fails")
		}
		return
	}
	t.Fatal("expected operation missing")
}
