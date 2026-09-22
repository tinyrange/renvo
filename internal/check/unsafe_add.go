package check

import "renvo.dev/internal/load"
import "renvo.dev/internal/syntax"

// Unsafe intrinsics have operand rules beyond their compact runtime signatures.
// Reject definitely invalid operands before ordinary call lowering can turn a
// floating-point offset or length into an integer.
func invalidUnsafeIntrinsicCalls(pkg *load.Package, info *PackageInfo, fileIndex int, fn syntax.FuncDecl, signature *FuncSignature, selectors []CoreSelectorRef) (int, int) {
	file := &pkg.Files[fileIndex].File
	var locals []definiteLocalTypeSpan
	ready := false
	for _, selector := range selectors {
		if selector.BaseIndex < 0 || selector.BaseIndex >= len(info.Imports) || info.Imports[selector.BaseIndex].ImportPath != "unsafe" {
			continue
		}
		callee := selector.NameTok
		isString := tokenTextIs(file, callee, "String")
		isStringData := tokenTextIs(file, callee, "StringData")
		if !tokenTextIs(file, callee, "Add") && !isString && !isStringData {
			continue
		}
		if !tokCharIs(file, callee+1, '(') {
			return CheckErrBuiltinOperand, callee
		}
		close := findTypeMatching(file, callee+1, '(', ')')
		if close <= callee+1 {
			continue
		}
		args := splitExprList(*file, callee+2, close-1)
		arity := 2
		if isStringData {
			arity = 1
		}
		if len(args) != arity {
			return CheckErrBuiltinArity, callee
		}
		if !ready {
			locals = collectDefiniteLocalTypes(*file, fn)
			ready = true
		}
		if isStringData {
			typ := definiteBuiltinExprTypeName(pkg, info, fileIndex, signature, locals, args[0], callee, 0)
			start, end := stripOuterParens(file, args[0].StartTok, args[0].EndTok)
			if typ != "" && typ != "string" || end-start == 1 && (file.Tokens[start].KindLine&255 == syntax.TokenNumber || file.Tokens[start].KindLine&255 == syntax.TokenChar || tokenTextIs(file, start, "nil")) {
				return CheckErrBuiltinOperand, start
			}
			continue
		}
		pointer := args[0]
		pointerType := definiteBuiltinExprTypeName(pkg, info, fileIndex, signature, locals, pointer, callee, 0)
		pointerStart, pointerEnd := stripOuterParens(file, pointer.StartTok, pointer.EndTok)
		byteAddress := isString && tokenTextIs(file, pointerStart, "&") && (pointerType == "byte" || pointerType == "uint8")
		if pointerType != "" && pointerType != "unsafe.Pointer" && !byteAddress {
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
		start, end := stripOuterParens(file, offset.StartTok, offset.EndTok)
		if unsafeAddFractionalDecimal(*file, start, end) {
			return CheckErrBuiltinOperand, start
		}
		if end-start == 1 && (file.Tokens[start].KindLine&255 == syntax.TokenString || tokenTextIs(file, start, "nil") || tokenTextIs(file, start, "true") || tokenTextIs(file, start, "false")) {
			return CheckErrBuiltinOperand, start
		}
		if literalIntegerOverflows(*file, start, end, "int64") {
			return CheckErrBuiltinOperand, start
		}
		if isString && literalIntegerOverflows(*file, start, end, "uint64") {
			return CheckErrBuiltinOperand, start
		}
	}
	return CheckOK, -1
}
