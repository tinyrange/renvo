package check

import "j5.nz/rtg/rtg/internal/syntax"

type IndexExpr struct {
	StartTok   int
	EndTok     int
	BaseStart  int
	BaseEnd    int
	OpenTok    int
	CloseTok   int
	IndexStart int
	IndexEnd   int
	Index      ExprSpan
}

type CompositeExpr struct {
	StartTok  int
	EndTok    int
	TypeStart int
	TypeEnd   int
	OpenTok   int
	CloseTok  int
	Elems     []ExprSpan
}

func buildFuncIndexExprs(file syntax.File, body syntax.Body) []IndexExpr {
	var indexes []IndexExpr
	for i := 0; i < len(body.Stmts); i++ {
		stmt := body.Stmts[i]
		if stmt.Kind == syntax.StmtDecl {
			indexes = appendDeclIndexExprs(indexes, file, stmt)
		} else if stmt.Kind == syntax.StmtAssign {
			indexes = appendExprIndexes(indexes, file, stmt.StartTok, stmt.EndTok)
		} else if stmt.ExprStart >= 0 && stmt.ExprEnd > stmt.ExprStart {
			indexes = appendExprIndexes(indexes, file, stmt.ExprStart, stmt.ExprEnd)
		}
	}
	return indexes
}

func buildFuncCompositeExprs(file syntax.File, body syntax.Body) []CompositeExpr {
	var composites []CompositeExpr
	for i := 0; i < len(body.Stmts); i++ {
		stmt := body.Stmts[i]
		if stmt.Kind == syntax.StmtDecl {
			composites = appendDeclCompositeExprs(composites, file, stmt)
		} else if stmt.Kind == syntax.StmtAssign {
			composites = appendExprComposites(composites, file, stmt.StartTok, stmt.EndTok)
		} else if stmt.ExprStart >= 0 && stmt.ExprEnd > stmt.ExprStart {
			composites = appendExprComposites(composites, file, stmt.ExprStart, stmt.ExprEnd)
		}
	}
	return composites
}

func appendDeclIndexExprs(indexes []IndexExpr, file syntax.File, stmt syntax.Stmt) []IndexExpr {
	start := stmt.StartTok + 1
	end := stmt.EndTok
	if start >= end {
		return indexes
	}
	if tokCharIs(file, start, '(') {
		i := start + 1
		for i < end {
			i = skipLocalSeparators(file, i, end)
			if i >= end || tokCharIs(file, i, ')') {
				break
			}
			specEnd := statementSpecEnd(file, i, end)
			indexes = appendSpecInitializerIndexes(indexes, file, i, specEnd)
			i = specEnd
		}
		return indexes
	}
	return appendSpecInitializerIndexes(indexes, file, start, end)
}

func appendDeclCompositeExprs(composites []CompositeExpr, file syntax.File, stmt syntax.Stmt) []CompositeExpr {
	start := stmt.StartTok + 1
	end := stmt.EndTok
	if start >= end {
		return composites
	}
	if tokCharIs(file, start, '(') {
		i := start + 1
		for i < end {
			i = skipLocalSeparators(file, i, end)
			if i >= end || tokCharIs(file, i, ')') {
				break
			}
			specEnd := statementSpecEnd(file, i, end)
			composites = appendSpecInitializerComposites(composites, file, i, specEnd)
			i = specEnd
		}
		return composites
	}
	return appendSpecInitializerComposites(composites, file, start, end)
}

func appendSpecInitializerIndexes(indexes []IndexExpr, file syntax.File, start int, end int) []IndexExpr {
	assign := findTokenText(file, start, end, "=")
	if assign < 0 {
		return indexes
	}
	return appendExprIndexes(indexes, file, assign+1, end)
}

func appendSpecInitializerComposites(composites []CompositeExpr, file syntax.File, start int, end int) []CompositeExpr {
	assign := findTokenText(file, start, end, "=")
	if assign < 0 {
		return composites
	}
	return appendExprComposites(composites, file, assign+1, end)
}

func appendExprIndexes(indexes []IndexExpr, file syntax.File, start int, end int) []IndexExpr {
	for i := start; i < end && i < len(file.Tokens); i++ {
		if !tokCharIs(file, i, '[') {
			continue
		}
		close := findTypeMatching(file, i, '[', ']')
		if close <= i || close > end {
			continue
		}
		baseStart := exprOperandStartBefore(file, start, i)
		if baseStart >= i || isIndexTypePrefix(file, baseStart) {
			continue
		}
		indexStart, indexEnd := trimExprSpan(file, i+1, close-1)
		indexes = append(indexes, IndexExpr{
			StartTok:   baseStart,
			EndTok:     close,
			BaseStart:  baseStart,
			BaseEnd:    i,
			OpenTok:    i,
			CloseTok:   close - 1,
			IndexStart: indexStart,
			IndexEnd:   indexEnd,
			Index:      ExprSpan{StartTok: indexStart, EndTok: indexEnd},
		})
	}
	return indexes
}

func appendExprComposites(composites []CompositeExpr, file syntax.File, start int, end int) []CompositeExpr {
	for i := start; i < end && i < len(file.Tokens); i++ {
		if !tokCharIs(file, i, '{') {
			continue
		}
		if isCompositeTypeBodyOpen(file, i) {
			continue
		}
		close := findTypeMatching(file, i, '{', '}')
		if close <= i || close > end {
			continue
		}
		typeStart := exprOperandStartBefore(file, start, i)
		if typeStart >= i {
			continue
		}
		typeStart, typeEnd := trimExprSpan(file, typeStart, i)
		if typeStart < 0 || typeEnd <= typeStart {
			continue
		}
		composites = append(composites, CompositeExpr{
			StartTok:  typeStart,
			EndTok:    close,
			TypeStart: typeStart,
			TypeEnd:   typeEnd,
			OpenTok:   i,
			CloseTok:  close - 1,
			Elems:     splitExprList(file, i+1, close-1),
		})
	}
	return composites
}

func exprOperandStartBefore(file syntax.File, start int, before int) int {
	depth := 0
	for i := before - 1; i >= start; i-- {
		if tokCharIs(file, i, ']') || tokCharIs(file, i, ')') || tokCharIs(file, i, '}') {
			depth++
			continue
		}
		if tokCharIs(file, i, '[') || tokCharIs(file, i, '(') || tokCharIs(file, i, '{') {
			if depth > 0 {
				depth--
				continue
			}
			return i + 1
		}
		if depth == 0 && isExprLeftBoundary(file, i) {
			return i + 1
		}
	}
	return start
}

func isExprLeftBoundary(file syntax.File, tok int) bool {
	if tokCharIs(file, tok, ',') || tokCharIs(file, tok, ';') || tokCharIs(file, tok, ':') {
		return true
	}
	if isAssignOp(file, tok) {
		return true
	}
	return isExprBinaryOp(file, tok)
}

func isExprBinaryOp(file syntax.File, tok int) bool {
	if tokenTextIs(file, tok, "||") || tokenTextIs(file, tok, "&&") {
		return true
	}
	if tokenTextIs(file, tok, "==") || tokenTextIs(file, tok, "!=") {
		return true
	}
	if tokenTextIs(file, tok, "<") || tokenTextIs(file, tok, "<=") {
		return true
	}
	if tokenTextIs(file, tok, ">") || tokenTextIs(file, tok, ">=") {
		return true
	}
	if tokenTextIs(file, tok, "+") || tokenTextIs(file, tok, "-") {
		return true
	}
	if tokenTextIs(file, tok, "*") || tokenTextIs(file, tok, "/") {
		return true
	}
	if tokenTextIs(file, tok, "%") || tokenTextIs(file, tok, "&") {
		return true
	}
	if tokenTextIs(file, tok, "|") || tokenTextIs(file, tok, "^") {
		return true
	}
	if tokenTextIs(file, tok, "<<") || tokenTextIs(file, tok, ">>") {
		return true
	}
	return tokenTextIs(file, tok, "&^")
}

func isIndexTypePrefix(file syntax.File, start int) bool {
	if start < 0 || start >= len(file.Tokens) {
		return false
	}
	return file.Tokens[start].Kind == syntax.TokenMap
}

func isCompositeTypeBodyOpen(file syntax.File, open int) bool {
	prev := open - 1
	if prev < 0 || prev >= len(file.Tokens) {
		return false
	}
	return file.Tokens[prev].Kind == syntax.TokenStruct || file.Tokens[prev].Kind == syntax.TokenInterface
}
