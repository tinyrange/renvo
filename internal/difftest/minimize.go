//go:build !renvo

package difftest

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"sort"
)

type Predicate func([]byte) (bool, error)

// Minimize repeatedly removes declarations and statements, then simplifies
// expressions. A candidate is retained only when predicate reports that the
// original discrepancy remains interesting.
func Minimize(source []byte, predicate Predicate) ([]byte, error) {
	current, err := format.Source(source)
	if err != nil {
		return nil, err
	}
	type predicateResult struct {
		interesting bool
	}
	results := make(map[string]predicateResult)
	evaluate := func(candidate []byte) (bool, error) {
		key := string(candidate)
		if result, ok := results[key]; ok {
			return result.interesting, nil
		}
		interesting, predicateErr := predicate(candidate)
		if predicateErr == nil {
			results[key] = predicateResult{interesting: interesting}
		}
		return interesting, predicateErr
	}
	interesting, err := evaluate(current)
	if err != nil {
		return nil, err
	}
	if !interesting {
		return current, nil
	}

	for {
		changed := false
		for _, phase := range []func([]byte) []sourceEdit{deletionEdits, structuralEdits, expressionEdits} {
			for _, edit := range phase(current) {
				candidate := applyEdit(current, edit)
				formatted, formatErr := format.Source(candidate)
				if formatErr != nil || !sourceIsSimpler(formatted, current) {
					continue
				}
				keep, predicateErr := evaluate(formatted)
				if predicateErr != nil {
					continue
				}
				if keep {
					current = formatted
					changed = true
					break
				}
			}
			if changed {
				break
			}
		}
		if !changed {
			return current, nil
		}
	}
}

// structuralEdits preserve Go syntax while discarding control-flow and value
// construction structure. The predicate remains responsible for rejecting
// candidates that no longer type check or reproduce the discrepancy.
func structuralEdits(source []byte) []sourceEdit {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "main.go", source, parser.SkipObjectResolution)
	if err != nil {
		return nil
	}
	var edits []sourceEdit
	ast.Inspect(file, func(node ast.Node) bool {
		switch value := node.(type) {
		case *ast.IfStmt:
			edits = append(edits, nodeSourceEdit(fileSet, source, value.Body, value, 2))
			if value.Else != nil {
				edits = append(edits, nodeSourceEdit(fileSet, source, value.Else, value, 2))
			}
		case *ast.CompositeLit:
			if value.Type != nil && len(value.Elts) != 0 {
				start, end := offset(fileSet, value.Type.Pos()), offset(fileSet, value.Type.End())
				edits = append(edits, nodeEdit(fileSet, value, string(source[start:end])+"{}", 3))
			}
		}
		return true
	})
	sortEdits(edits)
	return edits
}

func sourceIsSimpler(candidate, current []byte) bool {
	if len(candidate) != len(current) {
		return len(candidate) < len(current)
	}
	return bytes.Compare(candidate, current) < 0
}

type sourceEdit struct {
	start       int
	end         int
	replacement string
	priority    int
}

func deletionEdits(source []byte) []sourceEdit {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "main.go", source, parser.SkipObjectResolution)
	if err != nil {
		return nil
	}
	var edits []sourceEdit
	for _, declaration := range file.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok && function.Name.Name == "main" {
			continue
		}
		edits = append(edits, nodeEdit(fileSet, declaration, "", 0))
	}
	ast.Inspect(file, func(node ast.Node) bool {
		switch value := node.(type) {
		case *ast.BlockStmt:
			for size := len(value.List); size >= 1; size /= 2 {
				for start := 0; start+size <= len(value.List); start++ {
					edits = append(edits, spanEdit(fileSet, value.List[start], value.List[start+size-1], "", 1))
				}
				if size == 1 {
					break
				}
			}
		case *ast.GenDecl:
			if len(value.Specs) > 1 {
				for _, spec := range value.Specs {
					edits = append(edits, nodeEdit(fileSet, spec, "", 2))
				}
			}
		}
		return true
	})
	sortEdits(edits)
	return edits
}

func expressionEdits(source []byte) []sourceEdit {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "main.go", source, parser.SkipObjectResolution)
	if err != nil {
		return nil
	}
	var edits []sourceEdit
	ast.Inspect(file, func(node ast.Node) bool {
		expression, ok := node.(ast.Expr)
		if !ok {
			return true
		}
		if _, identifier := expression.(*ast.Ident); identifier {
			return true
		}
		switch value := expression.(type) {
		case *ast.BasicLit:
			var replacements []string
			switch value.Kind {
			case token.INT:
				replacements = []string{"0", "1", "-1"}
			case token.FLOAT:
				replacements = []string{"0.0", "1.0", "-1.0"}
			case token.IMAG:
				replacements = []string{"0i", "1i"}
			case token.CHAR:
				replacements = []string{`'a'`, `'\x00'`}
			case token.STRING:
				replacements = []string{`""`}
			}
			for _, replacement := range replacements {
				edits = append(edits, nodeEdit(fileSet, expression, replacement, 4))
			}
		case *ast.BinaryExpr:
			edits = append(edits, nodeSourceEdit(fileSet, source, value.X, expression, 3))
			edits = append(edits, nodeSourceEdit(fileSet, source, value.Y, expression, 3))
		case *ast.ParenExpr:
			edits = append(edits, nodeSourceEdit(fileSet, source, value.X, expression, 3))
		case *ast.UnaryExpr:
			edits = append(edits, nodeSourceEdit(fileSet, source, value.X, expression, 3))
		case *ast.CallExpr:
			for _, argument := range value.Args {
				edits = append(edits, nodeSourceEdit(fileSet, source, argument, expression, 3))
			}
		case *ast.IndexExpr:
			edits = append(edits, nodeSourceEdit(fileSet, source, value.X, expression, 3))
			edits = append(edits, nodeSourceEdit(fileSet, source, value.Index, expression, 3))
		case *ast.SelectorExpr:
			edits = append(edits, nodeSourceEdit(fileSet, source, value.X, expression, 3))
		}
		return true
	})
	sortEdits(edits)
	return edits
}

func nodeEdit(fileSet *token.FileSet, node ast.Node, replacement string, priority int) sourceEdit {
	return sourceEdit{start: offset(fileSet, node.Pos()), end: offset(fileSet, node.End()), replacement: replacement, priority: priority}
}

func spanEdit(fileSet *token.FileSet, first, last ast.Node, replacement string, priority int) sourceEdit {
	return sourceEdit{start: offset(fileSet, first.Pos()), end: offset(fileSet, last.End()), replacement: replacement, priority: priority}
}

func nodeSourceEdit(fileSet *token.FileSet, source []byte, replacement ast.Node, target ast.Node, priority int) sourceEdit {
	start, end := offset(fileSet, replacement.Pos()), offset(fileSet, replacement.End())
	return sourceEdit{start: offset(fileSet, target.Pos()), end: offset(fileSet, target.End()), replacement: string(source[start:end]), priority: priority}
}

func offset(fileSet *token.FileSet, position token.Pos) int {
	return fileSet.PositionFor(position, false).Offset
}

func applyEdit(source []byte, edit sourceEdit) []byte {
	if edit.start < 0 || edit.end < edit.start || edit.end > len(source) {
		return source
	}
	result := make([]byte, 0, len(source)-(edit.end-edit.start)+len(edit.replacement))
	result = append(result, source[:edit.start]...)
	result = append(result, edit.replacement...)
	result = append(result, source[edit.end:]...)
	return result
}

func sortEdits(edits []sourceEdit) {
	sort.SliceStable(edits, func(i, j int) bool {
		if edits[i].priority != edits[j].priority {
			return edits[i].priority < edits[j].priority
		}
		leftSize := edits[i].end - edits[i].start
		rightSize := edits[j].end - edits[j].start
		return leftSize > rightSize
	})
}
