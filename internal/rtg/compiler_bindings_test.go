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

func compilerFixtureReturn(operation compilerEmitterOperation) string {
	if operation.Result != "" {
		return " return " + operation.Failure + " "
	}
	return ""
}

// This is deliberately not one of the compiler's known architectures. A
// bundled operation must be selectable without adding generator-side policy.
func unfamiliarCompilerDefinition(t *testing.T, selector string, hook string) ResolveResult {
	t.Helper()
	source := "definition 1\nunit unfamiliar\nimplements direct_emitter_v1\narch unfamiliar {}\nextend arch unfamiliar {\ncompiler_selector = " + selector + "\ncompiler_bindings {\n"
	for _, operation := range compilerEmitterOperations {
		name := hook
		if len(operation.Parameters) != 0 || operation.Receiver.Name != "" || operation.Result != "" {
			name += operation.Suffix
		}
		source += operation.Name + " = " + name + "\n"
	}
	source += "}\n}\ngo compiler {\nfunc " + hook + "(a *renvoAsm) {}\n"
	for _, operation := range compilerEmitterOperations {
		if len(operation.Parameters) != 0 || operation.Receiver.Name != "" || operation.Result != "" {
			source += "func " + hook + operation.Suffix + operation.signature() + " {" + compilerFixtureReturn(operation) + "}\n"
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

type renvoExprParse struct {}
type renvoExpr struct {}
type renvoFuncInfo struct {}
type renvoAsmReserves struct {}
type renvoCompileResult struct {}
type renvoLinearGen struct { c *context; asm renvoAsm }
type context struct { renvoTargetArch int }
type renvoCompileContext = context
var renvoFixedTarget int
type renvoAsm struct { c *context; patchFailed bool }
func renvoNonNil(values ...interface{}) {}
const selectedOne = 41
const selectedTwo = 73
const renvoBackendValueSlotSize = 8
const renvoStaticCallUnavailable = 0
func firstHook(a *renvoAsm) {}
func secondHook(a *renvoAsm) {}
`)
	for _, operation := range compilerEmitterOperations {
		if len(operation.Parameters) != 0 || operation.Receiver.Name != "" || operation.Result != "" {
			for _, hook := range []string{"firstHook", "secondHook"} {
				prefix = append(prefix, "func "+hook+operation.Suffix+operation.signature()+" {"+compilerFixtureReturn(operation)+"}\n"...)
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
		{"lowering wrong result", "func firstHookGlobalInitFrameStart(g *renvoLinearGen) int", "func firstHookGlobalInitFrameStart(g *renvoLinearGen) bool", "RTG-COMPILER-004"},
		{"context wrong receiver", "func firstHookObjectCallABI(c *renvoCompileContext) int", "func firstHookObjectCallABI(c *renvoAsm) int", "RTG-COMPILER-004"},
		{"context wrong result", "func firstHookObjectCallABI(c *renvoCompileContext) int", "func firstHookObjectCallABI(c *renvoCompileContext) bool", "RTG-COMPILER-004"},
		{"lowering wrong receiver", "g *renvoLinearGen", "g *renvoAsm", "RTG-COMPILER-004"},
		{"lowering bool result", "op byte, size int) bool", "op byte, size int) int", "RTG-COMPILER-004"},
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
			if fn.Name.Name == operation.functionName() {
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
		{"typed result with renamed receiver", func(source, hook string) string {
			return strings.ReplaceAll(source, "g *renvoLinearGen", "receiver *renvoLinearGen")
		}, false},
		{"typed result with labels", func(source, hook string) string {
			return strings.Replace(source, "func "+hook+"GlobalInitFrameStart(g *renvoLinearGen) int { return -1 }",
				"func "+hook+"GlobalInitFrameStart(g *renvoLinearGen) int { again: if g.asm.patchFailed { g.asm.patchFailed = false; goto again }; return 17 }", 1)
		}, false},
		{"expression and switch colons are not labels", func(source, hook string) string {
			return strings.Replace(source, "func "+hook+"(a *renvoAsm) {}",
				"func "+hook+"(a *renvoAsm) { values := []int{0: 1}; values = values[0:1]; switch values[0] { case 1: a.patchFailed = true; default: a.patchFailed = false } }", 1)
		}, false},
		{"nested label has function scope", func(source, hook string) string {
			return strings.Replace(source, "func "+hook+"(a *renvoAsm) {}",
				"func "+hook+"(a *renvoAsm) { if a.patchFailed { again: a.patchFailed = false; if a.patchFailed { goto again } } }", 1)
		}, true},
		{"selector local does not capture a definition name", func(source, hook string) string {
			name := "renvoCompilerSelector"
			if hook == "secondHook" {
				name += "_"
			}
			return strings.Replace(source, "func "+hook+"(a *renvoAsm) {}",
				"var "+name+" = true\nfunc "+hook+"(a *renvoAsm) { a.patchFailed = "+name+" }", 1)
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
 type renvoExprParse struct {}
type renvoExpr struct {}
type renvoFuncInfo struct {}
type renvoAsmReserves struct {}
type renvoCompileResult struct {}
type renvoLinearGen struct { c *context; asm renvoAsm }
type context struct { renvoTargetArch int }
type renvoCompileContext = context
var renvoFixedTarget int
 type renvoAsm struct { c *context; patchFailed bool }
 func renvoNonNil(values ...interface{}) {}
 const selectedOne = 41
 const selectedTwo = 73
 const renvoBackendValueSlotSize = 8
const renvoStaticCallUnavailable = 0
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
			if strings.HasPrefix(tc.name, "typed result") {
				if !strings.Contains(string(generated.Source), "return firstHookGlobalInitFrameStart(g)") {
					t.Fatal("nonprojectable typed hook lost its result")
				}
				if !strings.Contains(string(generated.Source), "g.asm.patchFailed = true\nreturn -1") {
					t.Fatal("unknown lowering selector must fail with the frame sentinel")
				}
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
		fn, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue // Generated capability constants are not operation bodies.
		}
		if fn.Name.Name != "renvoAsmCopyPrimaryToSecondary" {
			continue
		}
		if len(fn.Body.List) != 6 {
			t.Fatalf("want receiver guard, context snapshot, context invariant, two body groups, and unknown-selector failure; got %d statements", len(fn.Body.List))
		}
		first := fn.Body.List[3].(*ast.IfStmt)
		second := fn.Body.List[4].(*ast.IfStmt)
		for _, fixed := range []int{0, 1, -1} {
			for _, selector := range []int{0, 41, 73, 99, 101} {
				for group, branch := range []*ast.IfStmt{first, second} {
					want := selector == 41 || selector == 99
					if group == 1 {
						want = selector == 73
					}
					got := evalCompilerBindingCondition(t, branch.Cond, fixed, selector) != 0
					if got != want {
						t.Fatalf("fixed=%d selector=%d group=%d selected=%v, want %v", fixed, selector, group, got, want)
					}
				}
			}
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
		failure := fn.Body.List[5].(*ast.AssignStmt)
		if failure.Lhs[0].(*ast.SelectorExpr).Sel.Name != "patchFailed" || failure.Rhs[0].(*ast.Ident).Name != "true" {
			t.Fatal("unknown selector no longer fails")
		}
		return
	}
	t.Fatal("expected operation missing")
}

// Evaluate generated selection independently of branch spelling. Equal-body
// grouping and fixed-target specialization must both preserve the truth table,
// including selectors not present in any bundled definition.
func evalCompilerBindingCondition(t *testing.T, expression ast.Expr, fixed, selector int) int {
	t.Helper()
	switch e := expression.(type) {
	case *ast.ParenExpr:
		return evalCompilerBindingCondition(t, e.X, fixed, selector)
	case *ast.Ident:
		switch e.Name {
		case "renvoFixedTarget":
			return fixed
		case "renvoCompilerSelector":
			return selector
		case "selectedOne":
			return 41
		case "selectedTwo":
			return 73
		case "selectedThree":
			return 99
		}
	case *ast.SelectorExpr:
		if e.Sel.Name == "renvoTargetArch" {
			return selector
		}
	case *ast.BasicLit:
		if e.Kind == token.INT && e.Value == "0" {
			return 0
		}
	case *ast.BinaryExpr:
		left := evalCompilerBindingCondition(t, e.X, fixed, selector)
		right := evalCompilerBindingCondition(t, e.Y, fixed, selector)
		value := false
		switch e.Op {
		case token.EQL:
			value = left == right
		case token.NEQ:
			value = left != right
		case token.LAND:
			value = left != 0 && right != 0
		case token.LOR:
			value = left != 0 || right != 0
		default:
			t.Fatalf("unexpected selection operator %v", e.Op)
		}
		if value {
			return 1
		}
		return 0
	}
	t.Fatalf("unexpected selection expression %#v", expression)
	return 0
}

// Prefixes can return early, bind scoped variables, or mutate the selector.
// Check the resulting control-flow tree, not just textual suffix matching.
func TestCompilerBindingSharedTailSelection(t *testing.T) {
	tail := "a.patchFailed = false\nreturn\n"
	prefixes := []string{
		"if local := renvoFixedTarget; local != 0 { return }\n",
		"if changeSelector(a) { return }\n",
	}
	bodies := []string{prefixes[0] + tail, prefixes[1] + tail, tail}
	conditions := []string{
		"a.c.renvoTargetArch == selectedOne",
		"a.c.renvoTargetArch == selectedTwo",
		"a.c.renvoTargetArch == selectedThree",
	}
	source := []byte(`package bindings
const selectedOne = 41
const selectedTwo = 73
const renvoBackendValueSlotSize = 8
const renvoStaticCallUnavailable = 0
const selectedThree = 99
var renvoFixedTarget int
type context struct { renvoTargetArch int }
type renvoCompileContext = context
type asm struct { c *context; patchFailed bool }
func changeSelector(a *asm) bool { a.c.renvoTargetArch = selectedOne; return false }
func projected(a *asm) {
`)
	source = appendCompilerBodyGroups(source, bodies, conditions)
	source = append(source, "a.patchFailed = true\n}\n"...)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "shared.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := new(types.Config).Check("bindings", fset, []*ast.File{file}, nil); err != nil {
		t.Fatal(err)
	}
	fn := file.Decls[len(file.Decls)-1].(*ast.FuncDecl)
	if len(fn.Body.List) != 2 {
		t.Fatal("tail was not shared, or failure path moved")
	}
	outer := fn.Body.List[0].(*ast.IfStmt)
	if len(outer.Body.List) != 3 {
		t.Fatal("want prefix selection, shared assignment and return")
	}
	first := outer.Body.List[0].(*ast.IfStmt)
	second, ok := first.Else.(*ast.IfStmt)
	if !ok || second.Else != nil {
		t.Fatal("prefix selection must be one exclusive chain")
	}
	for _, selector := range []int{0, 41, 73, 99, 101} {
		selected := evalCompilerBindingCondition(t, outer.Cond, 0, selector) != 0
		if selected != (selector == 41 || selector == 73 || selector == 99) {
			t.Fatalf("selector %d entered wrong shared body", selector)
		}
		for i, branch := range []*ast.IfStmt{first, second} {
			if (evalCompilerBindingCondition(t, branch.Cond, 0, selector) != 0) != (selector == []int{41, 73}[i]) {
				t.Fatalf("selector %d entered wrong prefix %d", selector, i)
			}
			if len(branch.Body.List) != 1 {
				t.Fatal("prefix scope changed")
			}
			if _, ok := branch.Body.List[0].(*ast.IfStmt); !ok {
				t.Fatal("prefix statement changed")
			}
		}
	}
	if _, ok := outer.Body.List[2].(*ast.ReturnStmt); !ok {
		t.Fatal("selected tail falls through")
	}
	failure := fn.Body.List[1].(*ast.AssignStmt)
	if failure.Rhs[0].(*ast.Ident).Name != "true" {
		t.Fatal("unknown selector did not fail")
	}
}

func TestCompilerBindingSharedTailConservative(t *testing.T) {
	for _, body := range []string{
		"value := 1\nif value != 0 { return }\nreturn\n",
		"if true { goto done }\ndone: return\n",
		"if true { return }",
		"if { broken",
	} {
		options := compilerBodyTails(body)
		if len(options) != 1 {
			t.Fatalf("must retain original scope/control flow: %q", body)
		}
	}
	body := "if true { return }\na.patchFailed = false\nreturn\n"
	output := appendCompilerBodyGroups(nil, []string{body, "a.patchFailed = true\nreturn\n"}, []string{"first", "second"})
	if !strings.Contains(string(output), body) {
		t.Fatal("changed a prefix without a matching complete tail")
	}
}

// Both selected bodies have effectful prefixes; neither complete body is the
// shared tail. A selector mutation must not run another target's prefix, and
// an if-initializer's local must not shadow the tail's independent declaration.
func TestCompilerBindingSharedTailAfterCalls(t *testing.T) {
	tail := "value := 7\na.patchFailed = value != 7\nreturn\n"
	bodies := []string{
		"changeSelector(a)\nif value := 1; value != 1 { return }\n" + tail,
		"observe(a)\n" + tail,
	}
	conditions := []string{"a.c.renvoTargetArch == selectedOne", "a.c.renvoTargetArch == selectedTwo"}
	source := []byte(`package bindings
const selectedOne = 41
const selectedTwo = 73
const renvoBackendValueSlotSize = 8
const renvoStaticCallUnavailable = 0
type context struct { renvoTargetArch int }
type renvoCompileContext = context
type asm struct { c *context; patchFailed bool }
func changeSelector(a *asm) { a.c.renvoTargetArch = selectedTwo }
func observe(a *asm) { a.patchFailed = true }
func projected(a *asm) {
`)
	source = appendCompilerBodyGroups(source, bodies, conditions)
	source = append(source, "a.patchFailed = true\n}\n"...)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "calls.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := new(types.Config).Check("bindings", fset, []*ast.File{file}, nil); err != nil {
		t.Fatal(err)
	}
	fn := file.Decls[len(file.Decls)-1].(*ast.FuncDecl)
	if len(fn.Body.List) != 2 {
		t.Fatal("shared body or unknown-selector failure missing")
	}
	outer := fn.Body.List[0].(*ast.IfStmt)
	if len(outer.Body.List) != 4 {
		t.Fatal("tail declaration escaped its shared scope")
	}
	first := outer.Body.List[0].(*ast.IfStmt)
	second, ok := first.Else.(*ast.IfStmt)
	if !ok || second.Else != nil {
		t.Fatal("prefix effects must execute in one exclusive chain")
	}
	if len(first.Body.List) != 2 || len(second.Body.List) != 1 {
		t.Fatal("prefix calls/conditionals changed")
	}
	if first.Body.List[0].(*ast.ExprStmt).X.(*ast.CallExpr).Fun.(*ast.Ident).Name != "changeSelector" ||
		second.Body.List[0].(*ast.ExprStmt).X.(*ast.CallExpr).Fun.(*ast.Ident).Name != "observe" {
		t.Fatal("prefix effects changed order or target")
	}
	if first.Body.List[1].(*ast.IfStmt).Init == nil {
		t.Fatal("scoped declaration was lost")
	}
	for _, selector := range []int{0, 41, 73, 99} {
		selected := evalCompilerBindingCondition(t, outer.Cond, 0, selector) != 0
		if selected != (selector == 41 || selector == 73) {
			t.Fatal("unknown selector entered shared code")
		}
		branch := first
		if evalCompilerBindingCondition(t, branch.Cond, 0, selector) == 0 {
			branch = second
		}
		if selected && (evalCompilerBindingCondition(t, branch.Cond, 0, selector) == 0) {
			t.Fatal("selected prefix missing")
		}
	}
	if strings.Count(string(source), "value := 7") != 1 {
		t.Fatal("tail was duplicated")
	}
}

func TestCompilerBindingTailScopeBoundaries(t *testing.T) {
	for _, body := range []string{
		"local := 1\nconsume(local)\nreturn\n",
		"var local int\nconsume(local)\nreturn\n",
		"local = 1\nconsume(local)\nreturn\n",
		"defer cleanup()\nconsume(1)\nreturn\n",
		"for local := 0; local < 1; local++ { consume(local) }\nconsume(2)\nreturn\n",
		"if true { goto done }\nconsume(1)\ndone: return\n",
		"consume(1)\nreturn\n",
		"consume(1)\nreturn true\n",
	} {
		if len(compilerBodyTails(body)) != 1 {
			t.Fatalf("unsafe or unhelpful split: %q", body)
		}
	}
}

func TestCompilerBindingTailFilterKeepsNestedReturnSuffix(t *testing.T) {
	tail := "value := 7\na.patchFailed = value != 7\nreturn\n"
	bodies := []string{
		"if first {\nreturn\n}\n" + tail,
		"if second {\nreturn\n}\n" + tail,
	}
	output := appendCompilerBodyGroups(nil, bodies, []string{"selectedOne", "selectedTwo"})
	if strings.Count(string(output), "value := 7") != 1 {
		t.Fatal("matching nested returns hid the shared tail")
	}
}

// Query defaults may be shared, but exceptional definitions and side effects
// must retain selection, and emission defaults must still reject unknown ISAs.
func TestCompilerBindingQueryDefaultSelection(t *testing.T) {
	var definitions []ResolveResult
	for i, pair := range [][2]string{{"selectedOne", "firstHook"}, {"selectedTwo", "secondHook"}, {"selectedThree", "thirdHook"}} {
		fixture := unfamiliarCompilerDefinition(t, pair[0], pair[1])
		source := string(fixture.Document.Source)
		if i != 0 {
			body := "return true"
			if i == 2 {
				body = "observe(c); return false"
			}
			source = strings.Replace(source,
				"func "+pair[1]+"ArenaDiscardSupported(c *renvoCompileContext) bool { return false }",
				"func "+pair[1]+"ArenaDiscardSupported(c *renvoCompileContext) bool { "+body+" }", 1)
		}
		document := Parse([]byte(source), "query-default.rtg")
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
	found := false
	for _, declaration := range file.Decls {
		fn, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue // Generated capability constants are not operation bodies.
		}
		if fn.Name.Name != "renvoArenaDiscardSupported" {
			continue
		}
		found = true
		for _, selector := range []int{0, 41, 73, 99, 101} {
			result, observed := false, false
			for _, statement := range fn.Body.List {
				branch, ok := statement.(*ast.IfStmt)
				if !ok || evalCompilerBindingCondition(t, branch.Cond, 0, selector) == 0 {
					continue
				}
				for _, selected := range branch.Body.List {
					switch s := selected.(type) {
					case *ast.ExprStmt:
						call := s.X.(*ast.CallExpr)
						if call.Fun.(*ast.Ident).Name != "observe" {
							t.Fatal("unexpected query effect")
						}
						observed = true
					case *ast.ReturnStmt:
						result = s.Results[0].(*ast.Ident).Name == "true"
					default:
						t.Fatalf("unexpected query statement %T", s)
					}
				}
			}
			if result != (selector == 73) || observed != (selector == 99) {
				t.Fatalf("selector=%d result=%v observed=%v", selector, result, observed)
			}
		}
		last := fn.Body.List[len(fn.Body.List)-1].(*ast.ReturnStmt)
		if last.Results[0].(*ast.Ident).Name != "false" {
			t.Fatal("unknown query selector lost unavailable result")
		}
	}
	if !found {
		t.Fatal("query operation missing")
	}
}

// A disabled bundled capability must not hide a new definition's implementation.
// Nonliteral bodies remain reachable, including observable side effects.
func TestCompilerCapabilityReachability(t *testing.T) {
	disabled := unfamiliarCompilerDefinition(t, "selectedOne", "firstHook")
	for _, body := range []string{"return true", "renvoNonNil(c); return false"} {
		original := "func secondHookLabelNotifications(c *renvoCompileContext) bool { return false }"
		source := string(unfamiliarCompilerDefinition(t, "selectedTwo", "secondHook").Document.Source)
		if !strings.Contains(source, original) {
			t.Fatal("fixture hook missing")
		}
		source = strings.Replace(source, original, "func secondHookLabelNotifications(c *renvoCompileContext) bool { "+body+" }", 1)
		document := Parse([]byte(source), "capability.rtg")
		if !document.Ok {
			t.Fatal(document.Diagnostics)
		}
		enabled := ResolveResult{Document: document, Ok: true}
		for _, tc := range []struct {
			definitions []ResolveResult
			want        string
		}{
			{[]ResolveResult{disabled}, "false"},
			{[]ResolveResult{disabled, enabled}, "true"},
			{[]ResolveResult{enabled, disabled}, "true"},
		} {
			generated := appendBundledCompilerBindings(nil, tc.definitions)
			if !generated.Ok {
				t.Fatal(generated.Diagnostics)
			}
			if !strings.Contains(string(generated.Source), "const renvoMayNotifyLabels = "+tc.want) {
				t.Fatalf("body %q: missing conservative guard %s", body, tc.want)
			}
		}
	}
	prepared := string(appendPreparedCompilerBindings(nil))
	for _, name := range []string{"renvoMayNotifyLabels", "renvoMayUseStructuredFunctions"} {
		if !strings.Contains(prepared, "const "+name+" = true") {
			t.Fatalf("prepared capability %s hidden", name)
		}
	}
}
