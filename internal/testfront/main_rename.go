//go:build !renvo

package testfront

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"sort"
	"strconv"
)

// The runner owns main, but tests may call the application's main themselves.
// Rename its binding and references, not same-spelled fields, methods, imports,
// or shadowing locals. Preserve source layout and all application bodies.
func renamePackageMain(files []GeneratedFile) error {
	fset := token.NewFileSet()
	parsed := make([]*ast.File, len(files))
	names := map[string]bool{}
	hasMain := false
	for i, file := range files {
		f, err := parser.ParseFile(fset, file.Name, file.Data, 0)
		if err != nil {
			return err
		}
		parsed[i] = f
		for _, decl := range f.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil && fn.Name.Name == "main" {
				hasMain = true
			}
		}
		ast.Inspect(f, func(node ast.Node) bool {
			if id, ok := node.(*ast.Ident); ok {
				names[id.Name] = true
			}
			return true
		})
	}
	if !hasMain {
		return nil
	}
	replacement := "__renvo_application_main"
	for n := 0; names[replacement]; n++ {
		replacement = "__renvo_application_main_" + strconv.Itoa(n)
	}
	info := &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}}
	// Renvo-only imports/APIs need not type-check with host Go. The checker
	// still resolves local package bindings; compilation reports actual errors.
	config := types.Config{Importer: importer.Default(), Error: func(error) {}}
	pkg, _ := config.Check("renvo.test/package", fset, parsed, info)
	main := pkg.Scope().Lookup("main")
	for i, f := range parsed {
		var offsets []int
		ast.Inspect(f, func(node ast.Node) bool {
			id, ok := node.(*ast.Ident)
			if ok && main != nil && (info.Defs[id] == main || info.Uses[id] == main) {
				offsets = append(offsets, fset.Position(id.Pos()).Offset)
			}
			return true
		})
		sort.Ints(offsets)
		for j := len(offsets) - 1; j >= 0; j-- {
			at := offsets[j]
			data := files[i].Data
			updated := make([]byte, 0, len(data)+len(replacement)-4)
			updated = append(updated, data[:at]...)
			updated = append(updated, replacement...)
			updated = append(updated, data[at+4:]...)
			files[i].Data = updated
		}
	}
	return nil
}
