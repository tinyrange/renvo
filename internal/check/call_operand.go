package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

func invalidCallOperandCount(graph load.Graph, pkgIndex int, info *PackageInfo, checked []PackageInfo, fileIndex int, fn syntax.FuncDecl, refs []CoreNameRef, selectors []CoreSelectorRef) int {
	file := &graph.Packages[pkgIndex].Files[fileIndex].File
	for _, ref := range refs {
		if ref.Index < 0 || ref.Index >= len(info.Symbols) {
			continue
		}
		if invalidResolvedCallOperand(graph.Packages[pkgIndex], info.Symbols[ref.Index], file, fn, ref.Token, ref.Token) {
			return ref.Token
		}
	}
	for _, selector := range selectors {
		pkg := selector.BasePackage
		if pkg < 0 || pkg >= len(checked) || pkg >= len(graph.Packages) || selector.Symbol < 0 || selector.Symbol >= len(checked[pkg].Symbols) {
			continue
		}
		if invalidResolvedCallOperand(graph.Packages[pkg], checked[pkg].Symbols[selector.Symbol], file, fn, selector.BaseTok, selector.NameTok) {
			return selector.NameTok
		}
	}
	return -1
}

func invalidResolvedCallOperand(pkg load.Package, symbol Symbol, file *syntax.File, fn syntax.FuncDecl, start int, callee int) bool {
	if symbol.Kind != SymbolFunc || callee+1 >= fn.BodyEnd || !tokCharIs(file, callee+1, '(') {
		return false
	}
	end := findTypeMatching(*file, callee+1, '(', ')')
	if end <= callee+1 || end > fn.BodyEnd {
		return false
	}
	for start > fn.BodyStart+1 && tokCharIs(file, start-1, '(') && findTypeMatching(*file, start-1, '(', ')') == end+1 {
		previous := file.Tokens[start-2].KindLine & 255
		if previous == syntax.TokenIdent || previous == syntax.TokenNumber || previous == syntax.TokenString || previous == syntax.TokenChar || tokCharIs(file, start-2, ')') || tokCharIs(file, start-2, ']') || tokCharIs(file, start-2, '}') {
			break // argument parentheses belong to a containing call
		}
		start--
		end++
	}
	// A newline after ')' terminates a statement. A following '*' may start
	// an unrelated pointer assignment rather than multiply this call's result.
	operand := end < fn.BodyEnd && syntax.TokenLine(file.Tokens[end]) == syntax.TokenLine(file.Tokens[end-1]) && (isExprBinaryOp(*file, end) || tokCharIs(file, end, '[') || tokCharIs(file, end, '.') || tokCharIs(file, end, '('))
	if start > fn.BodyStart+1 {
		operand = operand || isExprBinaryOp(*file, start-1) || tokenTextIs(file, start-1, "!") || tokenTextIs(file, start-1, "<-")
	}
	if !operand || symbol.File < 0 || symbol.File >= len(pkg.Files) {
		return false
	}
	target := pkg.Files[symbol.File].File
	decl, ok := findDefinitePackageFuncDecl(target, symbol.Token)
	if !ok {
		return false
	}
	resultStart, resultEnd := trimTypeSpan(target, decl.ResultStart, decl.ResultEnd)
	if resultStart < 0 || resultEnd <= resultStart {
		return true // a void call cannot supply an operand either
	}
	if !tokCharIs(&target, resultStart, '(') {
		return false
	}
	if resultEnd-resultStart == 2 {
		return true
	}
	// A top-level comma separates results or grouped result names. Commas
	// inside a function/aggregate result type do not make it multi-valued.
	comma := nextTopLevelComma(target, resultStart+1, resultEnd-1)
	remainingStart, remainingEnd := trimFieldSpan(target, comma+1, resultEnd-1)
	return comma < resultEnd-1 && remainingStart < remainingEnd
}
