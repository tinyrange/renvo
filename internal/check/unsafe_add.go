package check

import "renvo.dev/internal/load"
import "renvo.dev/internal/syntax"

// unsafe.Add accepts all integer types, not just the int parameter used by its
// compact runtime implementation. Reject definitely invalid operands before
// ordinary call lowering can turn a floating-point offset into an integer.
func invalidUnsafeAddCalls(pkg *load.Package, info *PackageInfo, fileIndex int, fn syntax.FuncDecl, signature *FuncSignature, selectors []CoreSelectorRef) (int, int) {
	file := &pkg.Files[fileIndex].File
	var locals []definiteLocalTypeSpan
	ready := false
	for _, selector := range selectors {
		if selector.BaseIndex < 0 || selector.BaseIndex >= len(info.Imports) || info.Imports[selector.BaseIndex].ImportPath != "unsafe" || !tokenTextIs(file, selector.NameTok, "Add") {
			continue
		}
		callee := selector.NameTok
		if !tokCharIs(file, callee+1, '(') {
			return CheckErrBuiltinOperand, callee
		}
		close := findTypeMatching(*file, callee+1, '(', ')')
		if close <= callee+1 {
			continue
		}
		args := splitExprList(*file, callee+2, close-1)
		if len(args) != 2 {
			return CheckErrBuiltinArity, callee
		}
		if !ready {
			locals = collectDefiniteLocalTypes(*file, fn)
			ready = true
		}
		pointer := args[0]
		pointerType := definiteBuiltinExprTypeName(pkg, info, fileIndex, signature, locals, pointer, callee, 0)
		pointerStart, pointerEnd := stripOuterParens(*file, pointer.StartTok, pointer.EndTok)
		if pointerType != "" && pointerType != "unsafe.Pointer" {
			return CheckErrBuiltinOperand, pointer.StartTok
		}
		if pointerEnd-pointerStart == 1 && (file.Tokens[pointerStart].KindLine&255 == syntax.TokenNumber || file.Tokens[pointerStart].KindLine&255 == syntax.TokenString || file.Tokens[pointerStart].KindLine&255 == syntax.TokenChar) {
			return CheckErrBuiltinOperand, pointer.StartTok
		}
		offset := args[1]
		typ := definiteBuiltinExprTypeName(pkg, info, fileIndex, signature, locals, offset, callee, 0)
		if typ == "bool" || typ == "string" || typ == "float32" || typ == "float64" || typ == "complex64" || typ == "complex128" || typ == "!" {
			return CheckErrBuiltinOperand, offset.StartTok
		}
		start, end := stripOuterParens(*file, offset.StartTok, offset.EndTok)
		if unsafeAddFractionalDecimal(*file, start, end) {
			return CheckErrBuiltinOperand, start
		}
		if end-start == 1 && (file.Tokens[start].KindLine&255 == syntax.TokenString || tokenTextIs(file, start, "nil") || tokenTextIs(file, start, "true") || tokenTextIs(file, start, "false")) {
			return CheckErrBuiltinOperand, start
		}
		if literalIntegerOverflows(*file, start, end, "int64") {
			return CheckErrBuiltinOperand, start
		}
	}
	return CheckOK, -1
}

// Recognize fractional decimal literals exactly, without floating-point
// rounding. Other constant expressions remain the general checker's job.
func unsafeAddFractionalDecimal(file syntax.File, start int, end int) bool {
	if start < end && (tokenTextIs(&file, start, "+") || tokenTextIs(&file, start, "-")) {
		start++
	}
	if end-start != 1 || file.Tokens[start].KindLine&255 != syntax.TokenNumber {
		return false
	}
	text := tokenString(&file, start)
	if len(text) > 1 && text[0] == '0' && (text[1] == 'x' || text[1] == 'X') {
		return false
	}
	digits, point, lastNonzero := 0, -1, 0
	i := 0
	for ; i < len(text) && text[i] != 'e' && text[i] != 'E'; i++ {
		c := text[i]
		if c == '_' {
			continue
		}
		if c == '.' {
			point = digits
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
		digits++
		if c != '0' {
			lastNonzero = digits
		}
	}
	if point < 0 {
		point = digits
	}
	exponent, sign := 0, 1
	if i < len(text) {
		i++
		if i < len(text) && (text[i] == '+' || text[i] == '-') {
			if text[i] == '-' {
				sign = -1
			}
			i++
		}
		for ; i < len(text); i++ {
			if text[i] == '_' {
				continue
			}
			if text[i] < '0' || text[i] > '9' {
				return false
			}
			if exponent < len(text) {
				exponent = exponent*10 + int(text[i]-'0')
			}
		}
	}
	return lastNonzero > 0 && lastNonzero > point+sign*exponent
}
