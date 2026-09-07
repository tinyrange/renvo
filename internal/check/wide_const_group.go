package check

import "renvo.dev/internal/syntax"

// Omitted const expressions repeat the previous specification, but iota uses
// the receiving specification's ordinal. Count specifications, not names, and
// include blank declarations without borrowing values from another group.
func wideDeclaredConstant(context constantIndexContext, target DeclInfo, depth int) wideConstant {
	// Package initializers cannot see bindings at a local use site.
	context.bindings = nil
	context.scope = CoreScope{}
	context.fileIndex = target.File
	file := context.pkg.Files[target.File].File
	if target.ValueStart >= 0 && target.TypeEnd <= target.TypeStart {
		hasIota := false
		for tok := target.ValueStart; tok < target.ValueEnd; tok++ {
			if tokenTextIs(&file, tok, "iota") {
				hasIota = true
			}
		}
		if !hasIota {
			values := splitExprList(file, target.ValueStart, target.ValueEnd)
			if target.ValueIndex >= 0 && target.ValueIndex < len(values) {
				context.iotaKnown = false
				return wideConstantExpr(context, values[target.ValueIndex].StartTok, values[target.ValueIndex].EndTok, depth)
			}
		}
	}
	previousStart, valueStart, valueEnd, typeStart, typeEnd := -1, -1, -1, -1, -1
	previousEnd := 0
	ordinal := 0
	for _, decl := range file.Decls {
		if decl.Kind != syntax.TokenConst {
			continue
		}
		if decl.StartTok != previousStart {
			start := decl.StartTok
			newGroup := previousStart < 0
			for tok := previousEnd; tok < start; tok++ {
				if file.Tokens[tok].KindLine&255 == syntax.TokenConst {
					newGroup = true
				}
			}
			if newGroup {
				ordinal = 0
				valueStart, valueEnd, typeStart, typeEnd = -1, -1, -1, -1
			} else {
				ordinal++
			}
			previousStart = start
			previousEnd = decl.EndTok
			namesEnd := declNameListEnd(file, decl)
			assign := findDeclAssign(file, namesEnd, decl.EndTok)
			if assign >= 0 {
				typeStart, typeEnd = trimDeclSpan(file, namesEnd, assign)
				valueStart, valueEnd = trimDeclSpan(file, assign+1, decl.EndTok)
			}
		}
		if decl.NameTok != target.Token {
			continue
		}
		if valueStart < 0 || typeEnd > typeStart {
			return wideConstant{}
		}
		values := splitExprList(file, valueStart, valueEnd)
		index := declNameIndex(file, decl)
		if index < 0 || index >= len(values) {
			return wideConstant{}
		}
		context.iotaKnown, context.iotaValue = true, ordinal
		return wideConstantExpr(context, values[index].StartTok, values[index].EndTok, depth)
	}
	return wideConstant{}
}
