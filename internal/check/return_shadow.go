package check

import "renvo.dev/internal/syntax"

type returnShadowBinding struct {
	token int
	end   int
}

func invalidBareReturnShadow(file syntax.File, fn syntax.FuncDecl, body syntax.Body, signature FuncSignature) int {
	if !resultsAreNamed(signature.Results) {
		return -1
	}
	var returns []int
	for _, stmt := range body.Stmts {
		if stmt.Kind == syntax.StmtReturn {
			_, _, count := returnValueList(file, stmt.StartTok, stmt.EndTok)
			if count == 0 {
				returns = append(returns, stmt.StartTok)
			}
		}
	}
	if len(returns) == 0 {
		return -1
	}
	var shadows []returnShadowBinding
	for _, stmt := range body.Stmts {
		var names CoreScope
		end := localRuleScopeEnd(body, stmt.StartTok)
		if stmt.Kind == syntax.StmtDecl {
			collectCoreDeclScope(file, stmt.StartTok, stmt.EndTok, &names)
		} else {
			start, finish := stmt.StartTok, stmt.EndTok
			if stmt.Kind == syntax.StmtIf || stmt.Kind == syntax.StmtFor || stmt.Kind == syntax.StmtSwitch {
				start++
				finish = stmt.BodyStart
				end = stmt.EndTok // includes the else arm and for/switch body
			} else if stmt.Kind == syntax.StmtCase {
				start++
			} else if stmt.Kind != syntax.StmtAssign {
				continue
			}
			op := findTopLevelAssignOp(file, start, finish)
			if op < 0 || !tokenTextIs(&file, op, ":=") {
				continue
			}
			// Short declarations in the function's own block reuse named
			// results; only declarations in an inner scope can shadow them.
			if end == fn.BodyEnd {
				continue
			}
			collectCoreLeadingIdentList(file, start, op, &names, false)
		}
		for _, name := range names.Names {
			for _, result := range signature.Results {
				if coreTokensEqual(&file, name.Token, result.NameTok) {
					shadows = append(shadows, returnShadowBinding{name.Token, end})
				}
			}
		}
	}
	for _, tok := range returns {
		for _, shadow := range shadows {
			if shadow.token < tok && tok < shadow.end {
				return tok
			}
		}
	}
	return -1
}
