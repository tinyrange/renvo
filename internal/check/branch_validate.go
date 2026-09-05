package check

import "renvo.dev/internal/syntax"

func invalidBranchTarget(file syntax.File, body syntax.Body) (int, int) {
	for i := 0; i < len(body.Stmts); i++ {
		stmt := body.Stmts[i]
		if stmt.Kind == syntax.StmtFallthrough {
			owner := -1
			clause := -1
			for j := 0; j < i; j++ {
				outer := body.Stmts[j]
				if outer.StartTok < stmt.StartTok && outer.EndTok > stmt.StartTok && (outer.Kind == syntax.StmtSwitch || outer.Kind == syntax.StmtSelect || outer.Kind == syntax.StmtFor || outer.Kind == syntax.StmtIf) {
					owner = j
				}
				if outer.Kind == syntax.StmtCase || outer.Kind == syntax.StmtDefault {
					clause = j
				}
			}
			if owner < 0 || body.Stmts[owner].Kind != syntax.StmtSwitch || clause < owner {
				return CheckErrBody, stmt.StartTok
			}
			switchStmt := body.Stmts[owner]
			for tok := switchStmt.ExprStart; tok < switchStmt.ExprEnd; tok++ {
				if file.Tokens[tok].KindLine&255 == syntax.TokenType {
					return CheckErrBody, stmt.StartTok
				}
			}
			for tok := body.Stmts[clause].EndTok; tok < stmt.StartTok; tok++ {
				if tokCharIs(&file, tok, '{') {
					end := findTypeMatching(file, tok, '{', '}')
					if end > stmt.StartTok {
						return CheckErrBody, stmt.StartTok
					}
					tok = end - 1
				}
			}
			next := stmt.EndTok
			for next < switchStmt.BodyEnd && tokCharIs(&file, next, ';') {
				next++
			}
			if next >= switchStmt.BodyEnd || file.Tokens[next].KindLine&255 != syntax.TokenCase && file.Tokens[next].KindLine&255 != syntax.TokenDefault {
				return CheckErrBody, stmt.StartTok
			}
		}
		if stmt.Kind != syntax.StmtGoto && stmt.Kind != syntax.StmtContinue && stmt.Kind != syntax.StmtBreak {
			continue
		}
		if branchIsBare(file, stmt) {
			continue
		}
		label := -1
		for j := 0; j < len(body.Stmts); j++ {
			candidate := body.Stmts[j]
			if candidate.Kind == syntax.StmtLabel && statementTokensEqual(&file, candidate.StartTok, stmt.StartTok+1) {
				label = j
				break
			}
		}
		if label < 0 {
			return CheckErrScope, stmt.StartTok + 1
		}
		target := body.Stmts[label]
		if stmt.Kind == syntax.StmtContinue || stmt.Kind == syntax.StmtBreak {
			valid := false
			for j := label + 1; j < len(body.Stmts); j++ {
				loop := body.Stmts[j]
				if loop.StartTok != target.EndTok {
					break
				}
				valid = loop.StartTok < stmt.StartTok && loop.EndTok > stmt.StartTok && (loop.Kind == syntax.StmtFor || stmt.Kind == syntax.StmtBreak && (loop.Kind == syntax.StmtSwitch || loop.Kind == syntax.StmtSelect))
				break
			}
			if !valid {
				if stmt.Kind == syntax.StmtContinue {
					return CheckErrContinue, stmt.StartTok
				}
				return CheckErrBreak, stmt.StartTok
			}
			continue
		}
		for j := 0; j < len(body.Stmts); j++ {
			block := body.Stmts[j]
			if block.Kind == syntax.StmtBlock && block.StartTok < target.StartTok && block.EndTok > target.StartTok && !(block.StartTok < stmt.StartTok && block.EndTok > stmt.StartTok) {
				return CheckErrScope, stmt.StartTok
			}
			if block.StartTok <= stmt.StartTok || block.StartTok >= target.StartTok {
				continue
			}
			decl := file.Tokens[block.StartTok].KindLine&255 == syntax.TokenVar
			if block.Kind == syntax.StmtAssign {
				op := findTopLevelAssignOp(file, block.StartTok, block.EndTok)
				decl = op >= 0 && tokenTextIs(&file, op, ":=")
			}
			if decl && definiteStatementScopeEnd(body, block.StartTok) > target.StartTok {
				return CheckErrScope, stmt.StartTok
			}
		}
	}
	return CheckOK, -1
}
