//go:build !renvo

package difftest

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
)

// Variant is a semantics-preserving source transformation used to expose
// compiler behavior that incorrectly depends on surface syntax or context.
type Variant struct {
	Name   string
	Source []byte
}

// Variants returns deterministic transformations of one valid Go program.
func Variants(source []byte, seed uint64) ([]Variant, error) {
	builders := []struct {
		name      string
		transform func(*ast.File)
	}{
		{name: "rename-locals", transform: renameLocalObjects},
		{name: "reorder-declarations", transform: func(file *ast.File) { reorderDeclarations(file, seed) }},
		{name: "identity-calls", transform: wrapCaseCalls},
		{name: "temporary-results", transform: materializeCaseResults},
	}
	variants := make([]Variant, 0, len(builders))
	for _, builder := range builders {
		fileSet := token.NewFileSet()
		file, err := parser.ParseFile(fileSet, "main.go", source, parser.AllErrors)
		if err != nil {
			return nil, fmt.Errorf("parse %s variant: %w", builder.name, err)
		}
		builder.transform(file)
		var output bytes.Buffer
		if err := format.Node(&output, fileSet, file); err != nil {
			return nil, fmt.Errorf("format %s variant: %w", builder.name, err)
		}
		variants = append(variants, Variant{Name: builder.name, Source: output.Bytes()})
	}
	return variants, nil
}

func renameLocalObjects(file *ast.File) {
	fields := make(map[*ast.Object]bool)
	ast.Inspect(file, func(node ast.Node) bool {
		structure, ok := node.(*ast.StructType)
		if !ok || structure.Fields == nil {
			return true
		}
		for _, field := range structure.Fields.List {
			for _, name := range field.Names {
				if name.Obj != nil {
					fields[name.Obj] = true
				}
			}
		}
		return true
	})
	names := make(map[*ast.Object]string)
	next := 0
	ast.Inspect(file, func(node ast.Node) bool {
		identifier, ok := node.(*ast.Ident)
		if !ok || identifier.Obj == nil || fields[identifier.Obj] || identifier.Name == "_" || identifier.Obj.Kind != ast.Var && identifier.Obj.Kind != ast.Con {
			return true
		}
		name, exists := names[identifier.Obj]
		if !exists {
			name = fmt.Sprintf("metamorphic%d", next)
			next++
			names[identifier.Obj] = name
		}
		identifier.Name = name
		return true
	})
}

func reorderDeclarations(file *ast.File, seed uint64) {
	var positions []int
	var declarations []ast.Decl
	for index, declaration := range file.Decls {
		movable := false
		switch value := declaration.(type) {
		case *ast.FuncDecl:
			movable = value.Recv == nil && value.Name.Name != "init"
		case *ast.GenDecl:
			movable = value.Tok == token.TYPE
		}
		if movable {
			positions = append(positions, index)
			declarations = append(declarations, declaration)
		}
	}
	if len(declarations) < 2 {
		return
	}
	rotation := int(seed % uint64(len(declarations)))
	for index, position := range positions {
		from := len(declarations) - 1 - ((index + rotation) % len(declarations))
		file.Decls[position] = declarations[from]
	}
}

func wrapCaseCalls(file *ast.File) {
	mainFunction := findFunction(file, "main")
	if mainFunction == nil || mainFunction.Body == nil {
		return
	}
	for _, statement := range mainFunction.Body.List {
		assignment, ok := statement.(*ast.AssignStmt)
		if !ok {
			continue
		}
		for index, expression := range assignment.Rhs {
			assignment.Rhs[index] = rewriteCaseCalls(expression)
		}
	}
	file.Decls = append(file.Decls, &ast.FuncDecl{
		Name: ast.NewIdent("metamorphicIdentityInt64"),
		Type: &ast.FuncType{
			Params:  &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{ast.NewIdent("value")}, Type: ast.NewIdent("int64")}}},
			Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("int64")}}},
		},
		Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent("value")}}}},
	})
}

func rewriteCaseCalls(expression ast.Expr) ast.Expr {
	switch value := expression.(type) {
	case *ast.CallExpr:
		value.Fun = rewriteCaseCalls(value.Fun)
		for index, argument := range value.Args {
			value.Args[index] = rewriteCaseCalls(argument)
		}
		if identifier, ok := value.Fun.(*ast.Ident); ok && generatedCaseName(identifier.Name) {
			return &ast.CallExpr{Fun: ast.NewIdent("metamorphicIdentityInt64"), Args: []ast.Expr{value}}
		}
	case *ast.BinaryExpr:
		value.X = rewriteCaseCalls(value.X)
		value.Y = rewriteCaseCalls(value.Y)
	case *ast.UnaryExpr:
		value.X = rewriteCaseCalls(value.X)
	case *ast.ParenExpr:
		value.X = rewriteCaseCalls(value.X)
	case *ast.IndexExpr:
		value.X = rewriteCaseCalls(value.X)
		value.Index = rewriteCaseCalls(value.Index)
	case *ast.SelectorExpr:
		value.X = rewriteCaseCalls(value.X)
	}
	return expression
}

func materializeCaseResults(file *ast.File) {
	mainFunction := findFunction(file, "main")
	if mainFunction == nil || mainFunction.Body == nil {
		return
	}
	statements := make([]ast.Stmt, 0, len(mainFunction.Body.List)*2)
	next := 0
	for _, statement := range mainFunction.Body.List {
		assignment, ok := statement.(*ast.AssignStmt)
		if !ok {
			statements = append(statements, statement)
			continue
		}
		for index, expression := range assignment.Rhs {
			name := fmt.Sprintf("metamorphicResult%d", next)
			rewritten, call, found := extractFirstCaseCall(expression, name)
			assignment.Rhs[index] = rewritten
			if found {
				statements = append(statements, &ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent(name)}, Tok: token.DEFINE, Rhs: []ast.Expr{call}})
				next++
				break
			}
		}
		statements = append(statements, statement)
	}
	mainFunction.Body.List = statements
}

func extractFirstCaseCall(expression ast.Expr, name string) (ast.Expr, *ast.CallExpr, bool) {
	switch value := expression.(type) {
	case *ast.CallExpr:
		if identifier, ok := value.Fun.(*ast.Ident); ok && generatedCaseName(identifier.Name) {
			return ast.NewIdent(name), value, true
		}
		for index, argument := range value.Args {
			rewritten, call, found := extractFirstCaseCall(argument, name)
			value.Args[index] = rewritten
			if found {
				return value, call, true
			}
		}
	case *ast.BinaryExpr:
		rewritten, call, found := extractFirstCaseCall(value.X, name)
		value.X = rewritten
		if found {
			return value, call, true
		}
		rewritten, call, found = extractFirstCaseCall(value.Y, name)
		value.Y = rewritten
		return value, call, found
	case *ast.UnaryExpr:
		rewritten, call, found := extractFirstCaseCall(value.X, name)
		value.X = rewritten
		return value, call, found
	case *ast.ParenExpr:
		rewritten, call, found := extractFirstCaseCall(value.X, name)
		value.X = rewritten
		return value, call, found
	}
	return expression, nil, false
}

func generatedCaseName(name string) bool {
	if len(name) <= 4 || name[:4] != "case" {
		return false
	}
	for index := 4; index < len(name); index++ {
		if name[index] < '0' || name[index] > '9' {
			return false
		}
	}
	return true
}

func findFunction(file *ast.File, name string) *ast.FuncDecl {
	for _, declaration := range file.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok && function.Name.Name == name {
			return function
		}
	}
	return nil
}
