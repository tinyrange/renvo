package rtg

import (
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"testing"
)

// Check the generated profile against the actual compiler profile type. In
// particular, unrelated pointer widths must not collapse to the language int
// width, and endianness/alignment must survive preparation.
func TestPreparedProfilePreservesIndependentLayout(t *testing.T) {
	policy, err := os.ReadFile("../../backend/compiler_target_policy_impl.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, endian := range []string{"little", "big"} {
		t.Run(endian, func(t *testing.T) {
			descriptor := TargetDescriptor{Name: "unfamiliar/mixed-width", OS: "none",
				WordBits: 32, PointerBits: 16, CodePointerBits: 24,
				FunctionPointerBits: 64, MaxAlign: 256, Endian: endian}
			files := token.NewFileSet()
			core, err := parser.ParseFile(files, "policy.go", policy, 0)
			if err != nil {
				t.Fatal(err)
			}
			generated, err := parser.ParseFile(files, "facts.go",
				appendPreparedTargetFacts([]byte("package main\n"), descriptor, true), 0)
			if err != nil {
				t.Fatal(err)
			}
			unit := &ast.File{Name: ast.NewIdent("main")}
			for _, declaration := range core.Decls {
				decl, ok := declaration.(*ast.GenDecl)
				if !ok {
					continue
				}
				if decl.Tok == token.CONST {
					unit.Decls = append(unit.Decls, decl)
				}
				if decl.Tok == token.TYPE {
					for _, spec := range decl.Specs {
						if spec.(*ast.TypeSpec).Name.Name == "renvoTargetProfile" {
							unit.Decls = append(unit.Decls, decl)
						}
					}
				}
			}
			var profile *ast.FuncDecl
			for _, declaration := range generated.Decls {
				if fn, ok := declaration.(*ast.FuncDecl); ok && fn.Name.Name == "renvoRTGProfileForTarget" {
					profile = fn
					unit.Decls = append(unit.Decls, fn)
				}
				if decl, ok := declaration.(*ast.GenDecl); ok && decl.Tok == token.CONST {
					for _, spec := range decl.Specs {
						if spec.(*ast.ValueSpec).Names[0].Name == "renvoRTGPreparedMaxAlign" {
							unit.Decls = append(unit.Decls, decl)
						}
					}
				}
			}
			if profile == nil {
				t.Fatal("prepared profile missing")
			}
			info := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue)}
			pkg, err := new(types.Config).Check("profile", files, []*ast.File{unit}, info)
			if err != nil {
				t.Fatal(err)
			}
			values := make(map[string]int64)
			ast.Inspect(profile, func(node ast.Node) bool {
				assignment, ok := node.(*ast.AssignStmt)
				if !ok || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
					return true
				}
				field, ok := assignment.Lhs[0].(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if value := info.Types[assignment.Rhs[0]].Value; value != nil {
					if n, ok := constant.Int64Val(value); ok {
						values[field.Sel.Name] = n
					}
				}
				return true
			})
			wantEndian := int64(1)
			if endian == "big" {
				wantEndian = 2
			}
			for name, want := range map[string]int64{
				"charBits": 8, "intBits": 32, "pointerBits": 16, "codePointerBits": 24,
				"funcPointerBits": 64, "maxAlign": 256, "endian": wantEndian,
			} {
				if got, ok := values[name]; !ok || got != want {
					t.Errorf("profile.%s = %d (present %v), want %d", name, got, ok, want)
				}
			}
			alignment := pkg.Scope().Lookup("renvoRTGPreparedMaxAlign").(*types.Const).Val()
			if got, _ := constant.Int64Val(alignment); got != 256 {
				t.Errorf("prepared alignment = %d, want 256", got)
			}
		})
	}
}
