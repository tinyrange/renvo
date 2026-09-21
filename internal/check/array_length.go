package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

func invalidLocalArrayLengths(pkg *load.Package, info *PackageInfo, fileIndex int, fn syntax.FuncDecl, body syntax.Body) int {
	file := pkg.Files[fileIndex].File
	bindings := collectScopedTypeBindings(file, fn, body)
	context := constantIndexContext{pkg: pkg, info: info, fileIndex: fileIndex, bindings: bindings, strict: true}
	for _, binding := range bindings {
		if numericBuiltinInNestedFunction(file, fn, binding.name) {
			continue
		}
		start, end := binding.typeStart, binding.typeEnd
		context.before = binding.name
		if !binding.writable && !binding.constant {
			// Local type names enter scope at their identifier, unlike values.
			// Their body follows the name and an optional alias equals sign.
			start, end = binding.name+1, binding.visible
			if tokenTextIs(&file, start, "=") {
				start++
			}
			context.before = binding.visible
		}
		if tok := invalidArrayLengthTypeSpan(context, start, end); tok >= 0 {
			return tok
		}
	}
	return -1
}

func invalidArrayLengthTypeSpan(context constantIndexContext, start int, end int) int {
	file := context.pkg.Files[context.fileIndex].File
	for tok := start; tok >= 0 && tok+1 < end; tok++ {
		if !tokCharIs(&file, tok, '[') || tokCharIs(&file, tok+1, ']') || tokenTextIs(&file, tok+1, "...") {
			continue
		}
		if tok > start && file.Tokens[tok-1].KindLine&255 == syntax.TokenMap {
			continue
		}
		close := findTypeMatching(file, tok, '[', ']')
		if close <= tok || close > end {
			continue
		}
		if arrayLengthVariableName(context, tok+1, close-1) {
			return tok + 1
		}
		operand := numericBuiltinExprValue(*context.pkg, *context.info, context.fileIndex, context.scope, context.bindings, tok+1, close-1, context.before, 0)
		if operand.kind == "bool" || operand.kind == "string" || operand.kind == "other" {
			return tok + 1
		}
		if unsafeAddFractionalDecimal(file, tok+1, close-1) {
			return tok + 1
		}
		value := wideConstantExpr(context, tok+1, close-1, 0)
		if value.ok {
			// Every supported target has at most 64-bit ints. Target-specific
			// smaller bounds remain the target layout checker's responsibility.
			if value.negative || len(value.words) > 5 || len(value.words) == 5 && value.words[4] >= 8 {
				return tok + 1
			}
		}
		tok = close - 1
	}
	return -1
}

func arrayLengthVariableName(context constantIndexContext, start, end int) bool {
	file := context.pkg.Files[context.fileIndex].File
	start, end = stripOuterParens(file, start, end)
	if end-start != 1 || file.Tokens[start].KindLine&255 != syntax.TokenIdent {
		return false
	}
	chosen := -1
	for i, binding := range context.bindings {
		if binding.visible <= context.before && context.before < binding.end && coreTokensEqual(&file, binding.name, start) && (chosen < 0 || binding.visible > context.bindings[chosen].visible) {
			chosen = i
		}
	}
	if chosen >= 0 {
		return context.bindings[chosen].writable
	}
	if lookupScopeTokenNameCore(context.scope, &file, start) >= 0 {
		return false
	}
	index := LookupDecl(*context.info, tokenString(&file, start))
	return index >= 0 && context.info.Decls[index].Kind == SymbolVar
}
