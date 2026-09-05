package check

import "renvo.dev/internal/syntax"

type switchCaseValue struct {
	owner int
	span  ExprSpan
	value ConstValue
}

// Only compare proven literal constants here. In particular, equal runtime
// expressions are legal cases, and typed constants in an interface switch
// may have equal numeric values but distinct dynamic types.
func invalidDuplicateSwitchCase(file syntax.File, body syntax.Body) int {
	var seen []switchCaseValue
	for i := 0; i < len(body.Stmts); i++ {
		clause := body.Stmts[i]
		if clause.Kind != syntax.StmtCase && clause.Kind != syntax.StmtDefault {
			continue
		}
		owner := -1
		for j := 0; j < i; j++ {
			stmt := body.Stmts[j]
			if (stmt.Kind == syntax.StmtSwitch || stmt.Kind == syntax.StmtSelect) && stmt.BodyStart < clause.StartTok && clause.StartTok < stmt.BodyEnd {
				owner = j
			}
		}
		if owner < 0 || body.Stmts[owner].Kind != syntax.StmtSwitch {
			continue
		}
		if clause.Kind == syntax.StmtDefault {
			for _, prev := range seen {
				if prev.owner == owner && prev.span.StartTok < 0 {
					return clause.StartTok
				}
			}
			seen = append(seen, switchCaseValue{owner: owner, span: ExprSpan{StartTok: -1}})
			continue
		}
		typeSwitch := false
		stmt := body.Stmts[owner]
		for tok := stmt.ExprStart; tok >= 0 && tok < stmt.ExprEnd; tok++ {
			if file.Tokens[tok].KindLine&255 == syntax.TokenType {
				typeSwitch = true
			}
		}
		for _, span := range splitExprList(file, clause.ExprStart, clause.ExprEnd) {
			span.StartTok, span.EndTok = stripOuterParens(file, span.StartTok, span.EndTok)
			value := duplicateCaseLiteral(file, span)
			if !typeSwitch && !value.Ok {
				continue
			}
			for _, prev := range seen {
				if prev.owner != owner || prev.span.StartTok < 0 {
					continue
				}
				if typeSwitch {
					if duplicateCaseTypeTokens(file, prev.span, span) {
						return span.StartTok
					}
				} else if value.Kind == prev.value.Kind && ((value.Kind == ConstInt && value.Int == prev.value.Int) || (value.Kind == ConstString && value.String == prev.value.String)) {
					return span.StartTok
				}
			}
			seen = append(seen, switchCaseValue{owner: owner, span: span, value: value})
		}
	}
	return -1
}

func duplicateCaseTypeTokens(file syntax.File, a ExprSpan, b ExprSpan) bool {
	if a.EndTok-a.StartTok != b.EndTok-b.StartTok {
		return false
	}
	for i := 0; i < a.EndTok-a.StartTok; i++ {
		if !statementTokensEqual(&file, a.StartTok+i, b.StartTok+i) {
			return false
		}
	}
	return true
}

func duplicateCaseLiteral(file syntax.File, span ExprSpan) ConstValue {
	if span.StartTok < span.EndTok && tokenTextIs(&file, span.StartTok, "+") {
		span.StartTok++
	}
	value := evalConstSpan(file, span)
	// Duplicate boolean cases are permitted; predeclared names can also be
	// shadowed. This check intentionally handles only integer/string literals.
	if value.Kind != ConstInt && value.Kind != ConstString {
		return ConstValue{}
	}
	return value
}
