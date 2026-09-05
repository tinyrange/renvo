package check

import "renvo.dev/internal/syntax"

// Go requires the final statement of a result-bearing function to terminate;
// unreachable statements after a return do not satisfy that requirement.
func returnBlockTerminates(file syntax.File, body syntax.Body, start int, end int, panicBuiltin bool) bool {
	last := -1
	for i := 0; i < len(body.Stmts); i++ {
		stmt := body.Stmts[i]
		if stmt.StartTok < start || stmt.EndTok > end {
			continue
		}
		if last >= 0 && stmt.StartTok < body.Stmts[last].EndTok {
			continue
		}
		last = i
	}
	if last < 0 {
		return false
	}
	stmt := body.Stmts[last]
	if stmt.Kind == syntax.StmtReturn || stmt.Kind == syntax.StmtGoto {
		return true
	}
	if stmt.Kind == syntax.StmtBlock {
		return returnBlockTerminates(file, body, stmt.BodyStart+1, stmt.BodyEnd-1, panicBuiltin)
	}
	if stmt.Kind == syntax.StmtIf {
		if stmt.ElseStart < 0 || !returnBlockTerminates(file, body, stmt.BodyStart+1, stmt.BodyEnd-1, panicBuiltin) {
			return false
		}
		if tokCharIs(&file, stmt.ElseStart+1, '{') {
			return returnBlockTerminates(file, body, stmt.ElseStart+2, stmt.ElseEnd-1, panicBuiltin)
		}
		return returnBlockTerminates(file, body, stmt.ElseStart+1, stmt.ElseEnd, panicBuiltin)
	}
	if stmt.Kind == syntax.StmtFor {
		if returnStatementHasBreak(file, body, last) {
			return false
		}
		start, end := trimExprSpan(file, stmt.ExprStart, stmt.ExprEnd)
		if start == end {
			return true
		}
		first := findTypeTopLevelChar(file, start, end, ';')
		return first >= 0 && tokCharIs(&file, first+1, ';')
	}
	if stmt.Kind == syntax.StmtSwitch || stmt.Kind == syntax.StmtSelect {
		if returnStatementHasBreak(file, body, last) {
			return false
		}
		hasDefault := stmt.Kind == syntax.StmtSelect
		clause := -1
		for i := last + 1; i < len(body.Stmts); i++ {
			next := body.Stmts[i]
			if next.StartTok >= stmt.BodyEnd-1 {
				break
			}
			if (next.Kind != syntax.StmtCase && next.Kind != syntax.StmtDefault) || !sameEnclosingStatement(body, stmt.Kind, next.StartTok, stmt.BodyStart+1) {
				continue
			}
			if clause >= 0 && !returnClauseTerminates(file, body, clause, next.StartTok, true, panicBuiltin) {
				return false
			}
			clause = i
			if next.Kind == syntax.StmtDefault {
				hasDefault = true
			}
		}
		return hasDefault && (clause < 0 && stmt.Kind == syntax.StmtSelect || clause >= 0 && returnClauseTerminates(file, body, clause, stmt.BodyEnd-1, false, panicBuiltin))
	}
	if stmt.Kind == syntax.StmtExpr && panicBuiltin {
		start, end := trimExprSpan(file, stmt.ExprStart, stmt.ExprEnd)
		if start+2 < end && tokenTextIs(&file, start, "panic") && tokCharIs(&file, start+1, '(') {
			for i := 0; i < len(file.Funcs); i++ {
				fn := file.Funcs[i]
				if fn.BodyStart < start && fn.BodyEnd > end {
					scope, _, _ := buildFuncScopeCore(file, fn)
					return lookupScopeTokenNameCore(scope, &file, start) < 0
				}
			}
		}
	}
	return false
}

func returnClauseTerminates(file syntax.File, body syntax.Body, clause int, end int, canFallthrough bool, panicBuiltin bool) bool {
	start := body.Stmts[clause].EndTok
	if canFallthrough {
		last := end - 1
		for last >= start && tokCharIs(&file, last, ';') {
			last--
		}
		if last >= start && file.Tokens[last].KindLine&255 == syntax.TokenFallthrough {
			return true
		}
	}
	return returnBlockTerminates(file, body, start, end, panicBuiltin)
}

func returnStatementHasBreak(file syntax.File, body syntax.Body, owner int) bool {
	stmt := body.Stmts[owner]
	for i := owner + 1; i < len(body.Stmts); i++ {
		branch := body.Stmts[i]
		if branch.StartTok >= stmt.EndTok {
			break
		}
		if branch.Kind != syntax.StmtBreak {
			continue
		}
		if !branchIsBare(file, branch) {
			for j := owner - 1; j >= 0; j-- {
				label := body.Stmts[j]
				if label.Kind == syntax.StmtLabel && label.EndTok == stmt.StartTok && statementTokensEqual(&file, label.StartTok, branch.StartTok+1) {
					return true
				}
			}
			continue
		}
		nearest := owner
		for j := owner + 1; j < i; j++ {
			inner := body.Stmts[j]
			if inner.StartTok < branch.StartTok && inner.EndTok > branch.StartTok && (inner.Kind == syntax.StmtFor || inner.Kind == syntax.StmtSwitch || inner.Kind == syntax.StmtSelect) {
				nearest = j
			}
		}
		if nearest == owner {
			return true
		}
	}
	return false
}
