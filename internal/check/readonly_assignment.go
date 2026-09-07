package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

func invalidReadOnlyAssignment(pkg load.Package, info PackageInfo, fileIndex int, fn syntax.FuncDecl, body syntax.Body) int {
	file := pkg.Files[fileIndex].File
	var bindings []scopedTypeBinding
	ready := false
	for _, stmt := range body.Stmts {
		if (stmt.Kind != syntax.StmtAssign && stmt.Kind != syntax.StmtExpr) || numericBuiltinInNestedFunction(file, fn, stmt.StartTok) {
			continue
		}
		op := findTopLevelAssignOp(file, stmt.StartTok, stmt.EndTok)
		if op < 0 && (tokenTextIs(&file, stmt.EndTok-1, "++") || tokenTextIs(&file, stmt.EndTok-1, "--")) {
			op = stmt.EndTok - 1
		}
		if op < 0 || tokenTextIs(&file, op, ":=") {
			continue
		}
		if !ready {
			bindings = collectScopedTypeBindings(file, fn, body)
			ready = true
		}
		for _, span := range splitExprList(file, stmt.StartTok, op) {
			if definitelyInvalidAssignTarget(file, span) {
				return span.StartTok
			}
			start, end := stripOuterParens(file, span.StartTok, span.EndTok)
			if end-start != 1 || file.Tokens[start].KindLine&255 != syntax.TokenIdent || tokenTextIs(&file, start, "_") {
				continue
			}
			chosen := -1
			for i, binding := range bindings {
				if binding.visible <= op && op < binding.end && coreTokensEqual(&file, binding.name, start) && (chosen < 0 || binding.visible > bindings[chosen].visible) {
					chosen = i
				}
			}
			if chosen >= 0 {
				if !bindings[chosen].writable {
					return start
				}
				continue
			}
			name := tokenString(&file, start)
			for _, decl := range info.Decls {
				if decl.Name == name && decl.Kind == SymbolConst {
					return start
				}
			}
		}
	}
	return -1
}
