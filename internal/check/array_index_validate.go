package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

type constantIndexContext struct {
	pkg       *load.Package
	info      *PackageInfo
	fileIndex int
	fn        syntax.FuncDecl
}

func invalidConstantArrayIndex(pkg *load.Package, info *PackageInfo, fileIndex int, fn syntax.FuncDecl, body *syntax.Body) int {
	if fileIndex < 0 || fileIndex >= len(pkg.Files) {
		return -1
	}
	file := pkg.Files[fileIndex].File
	indexes := buildFuncIndexExprs(&file, body)
	if len(indexes) == 0 {
		return -1
	}
	context := constantIndexContext{pkg: pkg, info: info, fileIndex: fileIndex, fn: fn}
	signature := buildFuncSignature(file, fn)
	locals := collectDefiniteLocalTypes(file, fn)
	for i := 0; i < len(indexes); i++ {
		index := &indexes[i]
		length, array := constantIndexArrayLength(context, signature, locals, index.BaseStart, index.BaseEnd, index.OpenTok, 0)
		if !array {
			continue
		}
		value, constant := constantIndexInt(context, index.IndexStart, index.IndexEnd, index.OpenTok, 0)
		if constant && (value < 0 || value >= length) {
			return index.IndexStart
		}
	}
	return -1
}

func constantIndexInt(context constantIndexContext, start int, end int, before int, depth int) (int, bool) {
	if depth > 32 || context.fileIndex < 0 || context.fileIndex >= len(context.pkg.Files) {
		return 0, false
	}
	file := context.pkg.Files[context.fileIndex].File
	start, end = trimExprSpan(file, start, end)
	start, end = stripOuterParens(file, start, end)
	if start < 0 || end <= start {
		return 0, false
	}
	if file.Tokens[start].KindLine&255 == syntax.TokenIdent && start+1 < end && tokCharIs(&file, start+1, '(') && findTypeMatching(file, start+1, '(', ')') == end && constantIndexType(context, start, 0) {
		return constantIndexInt(context, start+2, end-1, before, depth+1)
	}
	for precedence := 1; precedence <= 2; precedence++ {
		operator := constantIndexOperator(file, start, end, precedence)
		if operator < 0 {
			continue
		}
		left, leftOK := constantIndexInt(context, start, operator, before, depth+1)
		right, rightOK := constantIndexInt(context, operator+1, end, before, depth+1)
		if !leftOK || !rightOK {
			return 0, false
		}
		return applyConstantIndexOperator(tokenString(&file, operator), left, right)
	}
	if tokenTextIs(&file, start, "+") || tokenTextIs(&file, start, "-") || tokenTextIs(&file, start, "^") {
		value, ok := constantIndexInt(context, start+1, end, before, depth+1)
		if !ok {
			return 0, false
		}
		if tokenTextIs(&file, start, "-") {
			return -value, true
		}
		if tokenTextIs(&file, start, "^") {
			return value ^ -1, true
		}
		return value, true
	}
	if end != start+1 {
		return 0, false
	}
	if file.Tokens[start].KindLine&255 == syntax.TokenNumber {
		return parseConstInt(file, start)
	}
	if file.Tokens[start].KindLine&255 != syntax.TokenIdent {
		return 0, false
	}
	for i := context.fn.BodyStart + 1; i+3 < before; i++ {
		if file.Tokens[i].KindLine&255 == syntax.TokenConst && statementTokensEqual(&file, i+1, start) && tokenTextIs(&file, i+2, "=") {
			return constantIndexInt(context, i+3, statementSpecEnd(file, i+1, before), before, depth+1)
		}
	}
	name := tokenString(&file, start)
	for i := 0; i < len(context.info.Decls); i++ {
		if context.info.Decls[i].Kind == SymbolConst && context.info.Decls[i].Name == name && context.info.Decls[i].File >= 0 && context.info.Decls[i].File < len(context.pkg.Files) {
			values := splitExprList(context.pkg.Files[context.info.Decls[i].File].File, context.info.Decls[i].ValueStart, context.info.Decls[i].ValueEnd)
			if context.info.Decls[i].ValueIndex >= 0 && context.info.Decls[i].ValueIndex < len(values) {
				context.fileIndex = context.info.Decls[i].File
				return constantIndexInt(context, values[context.info.Decls[i].ValueIndex].StartTok, values[context.info.Decls[i].ValueIndex].EndTok, context.info.Decls[i].Token, depth+1)
			}
		}
	}
	return 0, false
}

func constantIndexOperator(file syntax.File, start int, end int, precedence int) int {
	parenDepth, bracketDepth, braceDepth := 0, 0, 0
	for i := end - 1; i >= start; i-- {
		if tokCharIs(&file, i, ')') {
			parenDepth++
		} else if tokCharIs(&file, i, '(') {
			parenDepth--
		} else if tokCharIs(&file, i, ']') {
			bracketDepth++
		} else if tokCharIs(&file, i, '[') {
			bracketDepth--
		} else if tokCharIs(&file, i, '}') {
			braceDepth++
		} else if tokCharIs(&file, i, '{') {
			braceDepth--
		}
		if parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 && constantIndexPrecedence(tokenString(&file, i)) == precedence && i > start && constantIndexPrecedence(tokenString(&file, i-1)) == 0 {
			return i
		}
	}
	return -1
}

func constantIndexPrecedence(operator string) int {
	if operator == "+" || operator == "-" || operator == "|" || operator == "^" {
		return 1
	}
	if operator == "*" || operator == "/" || operator == "%" || operator == "<<" || operator == ">>" || operator == "&" || operator == "&^" {
		return 2
	}
	return 0
}

func applyConstantIndexOperator(operator string, left int, right int) (int, bool) {
	switch operator {
	case "+":
		return left + right, true
	case "-":
		return left - right, true
	case "*":
		return left * right, true
	case "/", "%":
		if right == 0 {
			return 0, false
		}
		if operator == "/" {
			return left / right, true
		}
		return left % right, true
	case "<<", ">>":
		if right < 0 || right >= 63 {
			return 0, false
		}
		if operator == "<<" {
			return left << uint(right), true
		}
		return left >> uint(right), true
	case "&":
		return left & right, true
	case "&^":
		return left &^ right, true
	case "|":
		return left | right, true
	case "^":
		return left ^ right, true
	}
	return 0, false
}

func constantIndexType(context constantIndexContext, tok int, depth int) bool {
	if depth > 16 || context.fileIndex < 0 || context.fileIndex >= len(context.pkg.Files) {
		return false
	}
	file := context.pkg.Files[context.fileIndex].File
	name := tokenString(&file, tok)
	if name == "int" || name == "int8" || name == "int16" || name == "int32" || name == "int64" || name == "uint" || name == "uint8" || name == "uint16" || name == "uint32" || name == "uint64" || name == "uintptr" || name == "byte" || name == "rune" {
		return true
	}
	typeIndex := LookupType(*context.info, name)
	if typeIndex < 0 {
		return false
	}
	if context.info.Types[typeIndex].Kind != TypeNamed || context.info.Types[typeIndex].TypeEnd != context.info.Types[typeIndex].TypeStart+1 {
		return false
	}
	context.fileIndex = context.info.Types[typeIndex].File
	return constantIndexType(context, context.info.Types[typeIndex].TypeStart, depth+1)
}

func constantIndexArrayLength(context constantIndexContext, signature FuncSignature, locals []definiteLocalTypeSpan, start int, end int, before int, depth int) (int, bool) {
	if depth > 16 {
		return 0, false
	}
	file := context.pkg.Files[context.fileIndex].File
	start, end = stripOuterParens(file, start, end)
	for start < end && (tokCharIs(&file, start, '&') || tokCharIs(&file, start, '*')) {
		start++
		start, end = stripOuterParens(file, start, end)
	}
	if start < end && tokCharIs(&file, end-1, '}') {
		return constantIndexTypeLength(context, start, findTypeTopLevelChar(file, start, end, '{'), before, depth+1)
	}
	if end != start+1 || file.Tokens[start].KindLine&255 != syntax.TokenIdent {
		return 0, false
	}
	name := tokenString(&file, start)
	for group := 0; group < 3; group++ {
		var fields []Field
		if group == 0 {
			fields = signature.Receiver
		} else if group == 1 {
			fields = signature.Params
		} else {
			fields = signature.Results
		}
		for i := 0; i < len(fields); i++ {
			if fields[i].Name == name {
				return constantIndexTypeLength(context, fields[i].TypeStart, fields[i].TypeEnd, before, depth+1)
			}
		}
	}
	if typeStart, typeEnd, ok := findDefiniteLocalType(&file, locals, start, before); ok && typeStart >= 0 && typeEnd > typeStart {
		return constantIndexTypeLength(context, typeStart, typeEnd, before, depth+1)
	}
	for i := before - 1; i > context.fn.BodyStart; i-- {
		if file.Tokens[i].KindLine&255 == syntax.TokenIdent && statementTokensEqual(&file, i, start) && i+2 < before && tokenTextIs(&file, i+1, ":=") {
			valueStart, valueEnd := trimExprSpan(file, i+2, statementSpecEnd(file, i+2, before))
			return constantIndexArrayLength(context, signature, locals, valueStart, valueEnd, i, depth+1)
		}
	}
	for i := 0; i < len(context.info.Decls); i++ {
		if context.info.Decls[i].Kind == SymbolVar && context.info.Decls[i].Name == name && context.info.Decls[i].TypeStart >= 0 && context.info.Decls[i].TypeEnd > context.info.Decls[i].TypeStart {
			context.fileIndex = context.info.Decls[i].File
			return constantIndexTypeLength(context, context.info.Decls[i].TypeStart, context.info.Decls[i].TypeEnd, before, depth+1)
		}
	}
	return 0, false
}

func constantIndexTypeLength(context constantIndexContext, start int, end int, before int, depth int) (int, bool) {
	if depth > 16 || context.fileIndex < 0 || context.fileIndex >= len(context.pkg.Files) {
		return 0, false
	}
	file := context.pkg.Files[context.fileIndex].File
	start, end = trimTypeSpan(file, start, end)
	for start < end && tokCharIs(&file, start, '*') {
		start++
		start, end = trimTypeSpan(file, start, end)
	}
	if classifyType(file, start, end) == TypeArray {
		lengthStart, lengthEnd, _, _ := parseArrayTypeShape(file, start, end)
		length, ok := constantIndexInt(context, lengthStart, lengthEnd, before, depth+1)
		return length, ok && length >= 0
	}
	if end != start+1 || file.Tokens[start].KindLine&255 != syntax.TokenIdent {
		return 0, false
	}
	typeIndex := LookupType(*context.info, tokenString(&file, start))
	if typeIndex < 0 {
		return 0, false
	}
	context.fileIndex = context.info.Types[typeIndex].File
	return constantIndexTypeLength(context, context.info.Types[typeIndex].TypeStart, context.info.Types[typeIndex].TypeEnd, before, depth+1)
}
