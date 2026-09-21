package check

import "renvo.dev/internal/syntax"

// Each switch/select clause introduces an implicit block, independent of its
// siblings. Use its boundary as well as the explicit brace block boundary.
func localRuleScopeEnd(body syntax.Body, tok int) int {
	end := definiteStatementScopeEnd(body, tok)
	for i := 0; i < len(body.Stmts); i++ {
		owner := body.Stmts[i]
		if (owner.Kind != syntax.StmtSwitch && owner.Kind != syntax.StmtSelect) || tok <= owner.BodyStart || tok >= owner.BodyEnd {
			continue
		}
		for j := i + 1; j < len(body.Stmts); j++ {
			clause := body.Stmts[j]
			if (clause.Kind != syntax.StmtCase && clause.Kind != syntax.StmtDefault) || clause.StartTok <= tok || clause.StartTok >= end || clause.StartTok >= owner.BodyEnd {
				continue
			}
			nested := false
			for k := i + 1; k < j; k++ {
				inner := body.Stmts[k]
				if (inner.Kind == syntax.StmtSwitch || inner.Kind == syntax.StmtSelect) && inner.BodyStart < clause.StartTok && clause.StartTok < inner.BodyEnd {
					nested = true
				}
			}
			if !nested {
				end = clause.StartTok
			}
		}
	}
	return end
}

