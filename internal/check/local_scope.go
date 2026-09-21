package check

import "renvo.dev/internal/syntax"

// Each switch/select clause introduces an implicit block, independent of its
// siblings. The innermost enclosing owner determines which following clauses
// can end this scope; clauses inside a nested owner cannot end it.
func localRuleScopeEnd(body syntax.Body, tok int) int {
	end := definiteStatementScopeEnd(body, tok)
	ownerIndex := -1
	ownerStart := -1
	for i := 0; i < len(body.Stmts); i++ {
		owner := body.Stmts[i]
		if (owner.Kind == syntax.StmtSwitch || owner.Kind == syntax.StmtSelect) && owner.BodyStart < tok && tok < owner.BodyEnd && owner.BodyStart > ownerStart {
			ownerIndex, ownerStart = i, owner.BodyStart
		}
	}
	if ownerIndex < 0 {
		return end
	}
	ownerEnd := body.Stmts[ownerIndex].BodyEnd
	nestedEnd := -1
	// Statements are in source order. Skip each nested switch/select interval
	// once instead of rescanning all preceding statements for every clause.
	for i := ownerIndex + 1; i < len(body.Stmts); i++ {
		stmt := body.Stmts[i]
		if stmt.StartTok >= end || stmt.StartTok >= ownerEnd {
			break
		}
		if (stmt.Kind == syntax.StmtSwitch || stmt.Kind == syntax.StmtSelect) && stmt.BodyEnd > nestedEnd {
			nestedEnd = stmt.BodyEnd
		}
		if (stmt.Kind == syntax.StmtCase || stmt.Kind == syntax.StmtDefault) && stmt.StartTok > tok && stmt.StartTok >= nestedEnd {
			return stmt.StartTok
		}
	}
	return end
}

// Precompute lexical endpoints for an ordered statement traversal. A binding
// collector asks for many endpoints in one body; rescanning that body for every
// declaration makes large functions unnecessarily quadratic.
func localRuleScopeEnds(body syntax.Body) []int {
	ends := make([]int, len(body.Stmts))
	clauses := make([]int, len(body.Stmts))
	clauseEnds := make([]int, len(body.Stmts))
	currentClause := make([]int, len(body.Stmts))
	var blocks, owners []int
	for i, stmt := range body.Stmts {
		for len(blocks) > 0 && body.Stmts[blocks[len(blocks)-1]].EndTok <= stmt.StartTok {
			blocks = blocks[:len(blocks)-1]
		}
		for len(owners) > 0 && body.Stmts[owners[len(owners)-1]].BodyEnd <= stmt.StartTok {
			owners = owners[:len(owners)-1]
		}
		ends[i] = stmt.StartTok + 1
		if len(blocks) > 0 {
			ends[i] = body.Stmts[blocks[len(blocks)-1]].EndTok
		}
		if len(owners) > 0 {
			owner := owners[len(owners)-1]
			if body.Stmts[owner].BodyStart < stmt.StartTok {
				if stmt.Kind == syntax.StmtCase || stmt.Kind == syntax.StmtDefault {
					previous := currentClause[owner]
					if previous > 0 {
						clauseEnds[previous-1] = stmt.StartTok
					}
					currentClause[owner] = i + 1
					clauseEnds[i] = body.Stmts[owner].BodyEnd
				}
				clauses[i] = currentClause[owner]
			}
		}
		if stmt.Kind == syntax.StmtBlock {
			blocks = append(blocks, i)
		}
		if stmt.Kind == syntax.StmtSwitch || stmt.Kind == syntax.StmtSelect {
			owners = append(owners, i)
		}
	}
	for i, clause := range clauses {
		if clause > 0 && clauseEnds[clause-1] < ends[i] {
			ends[i] = clauseEnds[clause-1]
		}
	}
	return ends
}
