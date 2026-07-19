package check

import "renvo.dev/internal/syntax"

// invalidDefiniteStatement rejects statement errors that can be established
// without guessing the type of an unresolved expression. Calls and indexed
// expressions may be multi-valued, and function-valued expressions remain
// valid unless a preceding declaration gives the callee a definite literal
// value.
func invalidDefiniteStatement(file syntax.File, body syntax.Body) (int, int) {
	var literalLocals []int
	for i := 0; i < len(body.Stmts); i++ {
		stmt := body.Stmts[i]
		if stmt.Kind == syntax.StmtBreak && branchIsBare(file, stmt) && !branchHasEnclosing(body, stmt.StartTok, false) {
			return CheckErrBreak, stmt.StartTok
		}
		if stmt.Kind == syntax.StmtContinue && branchIsBare(file, stmt) && !branchHasEnclosing(body, stmt.StartTok, true) {
			return CheckErrContinue, stmt.StartTok
		}
		callStart := stmt.StartTok
		callEnd := stmt.EndTok
		if stmt.Kind == syntax.StmtBlock || stmt.Kind == syntax.StmtDefault || stmt.Kind == syntax.StmtLabel {
			callStart = -1
		} else if stmt.Kind == syntax.StmtIf || stmt.Kind == syntax.StmtFor || stmt.Kind == syntax.StmtSwitch || stmt.Kind == syntax.StmtCase {
			callStart = stmt.ExprStart
			callEnd = stmt.ExprEnd
		}
		for tok := callStart; tok >= 0 && tok+1 < callEnd; tok++ {
			if file.Tokens[tok].Kind == syntax.TokenIdent && file.Tokens[tok].Line == file.Tokens[tok+1].Line && tokCharIs(&file, tok+1, '(') && (tok == 0 || !tokCharIs(&file, tok-1, '.')) && definiteLiteralLocal(literalLocals, file, tok) {
				return CheckErrCall, tok
			}
		}
		if stmt.Kind != syntax.StmtAssign {
			continue
		}
		op := findTopLevelAssignOp(file, stmt.StartTok, stmt.EndTok)
		if op < 0 {
			continue
		}
		leftCount, leftFirst, invalidTarget := definiteExprListSummary(file, stmt.StartTok, op, true)
		if invalidTarget >= 0 {
			return CheckErrAssignTarget, invalidTarget
		}
		rightCount, rightFirst, _ := definiteExprListSummary(file, op+1, stmt.EndTok, false)
		if leftCount != rightCount && !(rightCount == 1 && expressionMayBeMultiValued(file, rightFirst)) {
			return CheckErrAssignCount, op
		}
		if leftCount == 1 && rightCount == 1 && leftFirst.EndTok-leftFirst.StartTok == 1 && file.Tokens[leftFirst.StartTok].Kind == syntax.TokenIdent {
			literal := expressionIsDefiniteLiteral(file, rightFirst)
			updated := false
			if !tokenTextIs(&file, op, ":=") {
				for j := len(literalLocals) - 3; j >= 0; j -= 3 {
					if stmt.StartTok < literalLocals[j+1] && statementTokensEqual(file, literalLocals[j], leftFirst.StartTok) {
						literalLocals[j+2] = 0
						if literal {
							literalLocals[j+2] = 1
						}
						updated = true
						break
					}
				}
			}
			if literal && !updated {
				scopeEnd := definiteStatementScopeEnd(body, stmt.StartTok)
				literalLocals = append(literalLocals, leftFirst.StartTok, scopeEnd, 1)
			}
		}
	}
	return CheckOK, -1
}

func definiteExprListSummary(file syntax.File, start int, end int, validateTargets bool) (int, ExprSpan, int) {
	start, end = trimExprSpan(file, start, end)
	var first ExprSpan
	count := 0
	invalid := -1
	for i := start; i >= 0 && i < end; {
		next := nextTopLevelComma(file, i, end)
		itemStart, itemEnd := trimExprSpan(file, i, next)
		if itemEnd > itemStart {
			span := ExprSpan{StartTok: itemStart, EndTok: itemEnd}
			if count == 0 {
				first = span
			}
			count++
			if validateTargets && invalid < 0 && definitelyInvalidAssignTarget(file, span) {
				invalid = itemStart
			}
		}
		i = next + 1
	}
	return count, first, invalid
}

func definiteLiteralLocal(locals []int, file syntax.File, tok int) bool {
	for i := len(locals) - 3; i >= 0; i -= 3 {
		if tok < locals[i+1] && statementTokensEqual(file, locals[i], tok) {
			return locals[i+2] != 0
		}
	}
	return false
}

func definiteStatementScopeEnd(body syntax.Body, tok int) int {
	end := 2147483647
	for i := 0; i < len(body.Stmts); i++ {
		stmt := body.Stmts[i]
		if stmt.Kind == syntax.StmtBlock && stmt.StartTok < tok && stmt.EndTok > tok && stmt.EndTok < end {
			end = stmt.EndTok
		}
	}
	if end == 2147483647 {
		return tok + 1
	}
	return end
}

func branchIsBare(file syntax.File, stmt syntax.Stmt) bool {
	for tok := stmt.StartTok + 1; tok < stmt.EndTok; tok++ {
		if tokCharIs(&file, tok, ';') {
			continue
		}
		return false
	}
	return true
}

func branchHasEnclosing(body syntax.Body, branchTok int, continueOnly bool) bool {
	for i := 0; i < len(body.Stmts); i++ {
		stmt := body.Stmts[i]
		if stmt.StartTok >= branchTok || stmt.EndTok <= branchTok {
			continue
		}
		if stmt.Kind == syntax.StmtFor || (!continueOnly && stmt.Kind == syntax.StmtSwitch) {
			return true
		}
	}
	return false
}

func definitelyInvalidAssignTarget(file syntax.File, span ExprSpan) bool {
	start, end := stripOuterParens(file, span.StartTok, span.EndTok)
	if end-start != 1 {
		return false
	}
	kind := file.Tokens[start].Kind
	if kind == syntax.TokenNumber || kind == syntax.TokenString || kind == syntax.TokenChar {
		return true
	}
	return tokenTextIs(&file, start, "true") || tokenTextIs(&file, start, "false") || tokenTextIs(&file, start, "nil")
}

func expressionMayBeMultiValued(file syntax.File, span ExprSpan) bool {
	start, end := stripOuterParens(file, span.StartTok, span.EndTok)
	if end <= start {
		return false
	}
	if tokCharIs(&file, end-1, ')') || tokCharIs(&file, end-1, ']') {
		return true
	}
	return tokenTextIs(&file, start, "<-")
}

func stripOuterParens(file syntax.File, start int, end int) (int, int) {
	for end-start >= 2 && tokCharIs(&file, start, '(') && findTypeMatching(file, start, '(', ')') == end {
		start++
		end--
	}
	return start, end
}

func statementTokensEqual(file syntax.File, left int, right int) bool {
	if left < 0 || left >= len(file.Tokens) || right < 0 || right >= len(file.Tokens) {
		return false
	}
	leftToken := file.Tokens[left]
	rightToken := file.Tokens[right]
	leftSize := leftToken.End - leftToken.Start
	if leftSize < 0 || rightToken.End-rightToken.Start != leftSize {
		return false
	}
	for i := 0; i < leftSize; i++ {
		if file.Src[leftToken.Start+i] != file.Src[rightToken.Start+i] {
			return false
		}
	}
	return true
}

func expressionIsDefiniteLiteral(file syntax.File, span ExprSpan) bool {
	start, end := stripOuterParens(file, span.StartTok, span.EndTok)
	if end-start != 1 {
		return false
	}
	kind := file.Tokens[start].Kind
	return kind == syntax.TokenNumber || kind == syntax.TokenString || kind == syntax.TokenChar || tokenTextIs(&file, start, "true") || tokenTextIs(&file, start, "false") || tokenTextIs(&file, start, "nil")
}
