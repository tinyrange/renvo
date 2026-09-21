package check

import "renvo.dev/internal/load"

// Check known-invalid constant operations even when their declarations are
// unused. Unknown or out-of-word-range values are not truncated into evidence
// of an error; full-width constant evaluation remains a separate concern.
func invalidPackageConstantOperations(pkg *load.Package, info *PackageInfo) (int, int) {
	for i := 0; i < len(info.Decls); i++ {
		decl := info.Decls[i]
		if decl.Kind != SymbolConst || decl.ValueStart < 0 {
			continue
		}
		context := constantIndexContext{pkg: pkg, info: info, fileIndex: decl.File, strict: true}
		context.fn.BodyStart = -1
		file := pkg.Files[decl.File].File
		values := splitExprList(file, decl.ValueStart, decl.ValueEnd)
		if decl.ValueIndex >= 0 && decl.ValueIndex < len(values) {
			span := values[decl.ValueIndex]
			if tok := invalidConstantSpanOperation(context, span.StartTok, span.EndTok, 0); tok >= 0 {
				return decl.File, tok
			}
		}
	}
	return -1, -1
}

func invalidConstantSpanOperation(context constantIndexContext, start int, end int, depth int) int {
	if depth > 64 {
		return -1
	}
	file := context.pkg.Files[context.fileIndex].File
	start, end = stripOuterParens(file, start, end)
	if start < 0 || start >= end {
		return -1
	}
	for precedence := 1; precedence <= 2; precedence++ {
		op := constantIndexOperator(file, start, end, precedence)
		if op < 0 {
			continue
		}
		if tok := invalidConstantSpanOperation(context, start, op, depth+1); tok >= 0 {
			return tok
		}
		if tok := invalidConstantSpanOperation(context, op+1, end, depth+1); tok >= 0 {
			return tok
		}
		operator := tokenString(&file, op)
		if operator == "/" || operator == "%" || operator == "<<" || operator == ">>" {
			value, known := constantIndexInt(context, op+1, end, 0, 0)
			if known && ((operator == "/" || operator == "%") && value == 0 || (operator == "<<" || operator == ">>") && value < 0) {
				return op
			}
		}
		return -1
	}
	// Nested calls and unary expressions can contain constant arithmetic too.
	for i := start; i < end; i++ {
		if tokCharIs(&file, i, '(') {
			close := findTypeMatching(file, i, '(', ')')
			if close > i && close <= end {
				for _, arg := range splitExprList(file, i+1, close-1) {
					if tok := invalidConstantSpanOperation(context, arg.StartTok, arg.EndTok, depth+1); tok >= 0 {
						return tok
					}
				}
				i = close - 1
			}
		}
	}
	return -1
}
