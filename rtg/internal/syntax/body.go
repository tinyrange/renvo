package syntax

const (
	BodyOK = iota
	BodyErrFunc
	BodyErrBlock
	BodyErrStmt
)

const (
	StmtOther = iota
	StmtReturn
	StmtIf
	StmtFor
	StmtSwitch
	StmtCase
	StmtDefault
	StmtDecl
	StmtAssign
	StmtExpr
	StmtBlock
	StmtBreak
	StmtContinue
	StmtGoto
	StmtDefer
	StmtGo
	StmtFallthrough
	StmtLabel
)

const (
	ExprOther = iota
	ExprIdent
	ExprLiteral
	ExprCall
	ExprSelector
	ExprIndex
	ExprComposite
	ExprUnary
	ExprBinary
)

type Body struct {
	Stmts    []Stmt
	Exprs    []Expr
	Ok       bool
	Error    int
	ErrorTok int
}

type Stmt struct {
	Kind      int
	StartTok  int
	EndTok    int
	ExprStart int
	ExprEnd   int
	BodyStart int
	BodyEnd   int
	ElseStart int
	ElseEnd   int
}

type Expr struct {
	Kind     int
	StartTok int
	EndTok   int
}

func ParseFuncBody(file File, fn FuncDecl) Body {
	body := Body{Ok: true, Error: BodyOK, ErrorTok: -1}
	closeTok := fn.BodyEnd - 1
	if fn.BodyStart < 0 || closeTok <= fn.BodyStart || !tokCharIs(file.Src, file.Tokens, fn.BodyStart, '{') || !tokCharIs(file.Src, file.Tokens, closeTok, '}') {
		return bodyFail(body, BodyErrFunc, fn.BodyStart)
	}
	return parseBlockInto(body, file, fn.BodyStart, closeTok)
}

func parseBlockInto(body Body, file File, openTok int, closeTok int) Body {
	if !tokCharIs(file.Src, file.Tokens, openTok, '{') || !tokCharIs(file.Src, file.Tokens, closeTok, '}') {
		return bodyFail(body, BodyErrBlock, openTok)
	}
	body.Stmts = append(body.Stmts, Stmt{
		Kind:      StmtBlock,
		StartTok:  openTok,
		EndTok:    closeTok + 1,
		ExprStart: -1,
		ExprEnd:   -1,
		BodyStart: openTok,
		BodyEnd:   closeTok + 1,
		ElseStart: -1,
		ElseEnd:   -1,
	})
	i := openTok + 1
	for i < closeTok && body.Ok {
		i = skipStmtSeparators(file, i)
		if i >= closeTok {
			break
		}
		next, nextBody := parseStmt(body, file, i, closeTok)
		body = nextBody
		if !body.Ok {
			return body
		}
		if next <= i {
			return bodyFail(body, BodyErrStmt, i)
		}
		i = next
	}
	return body
}

func parseStmt(body Body, file File, start int, limit int) (int, Body) {
	kind := file.Tokens[start].Kind
	if kind == TokenReturn {
		end := findStmtEnd(file, start+1, limit)
		stmt := newStmt(StmtReturn, start, end)
		stmt.ExprStart, stmt.ExprEnd = trimSpan(file, start+1, end)
		body = appendStmtExpr(body, file, stmt)
		return end, body
	}
	if kind == TokenIf {
		return parseBlockStmt(body, file, start, limit, StmtIf)
	}
	if kind == TokenFor {
		return parseBlockStmt(body, file, start, limit, StmtFor)
	}
	if kind == TokenSwitch {
		return parseBlockStmt(body, file, start, limit, StmtSwitch)
	}
	if kind == TokenCase {
		end := findCaseHeaderEnd(file, start+1, limit)
		stmt := newStmt(StmtCase, start, end)
		colon := findTopLevelChar(file, start+1, end, ':')
		if colon >= 0 {
			stmt.ExprStart, stmt.ExprEnd = trimSpan(file, start+1, colon)
		}
		body = appendStmtExpr(body, file, stmt)
		return end, body
	}
	if kind == TokenDefault {
		end := findCaseHeaderEnd(file, start+1, limit)
		stmt := newStmt(StmtDefault, start, end)
		body.Stmts = append(body.Stmts, stmt)
		return end, body
	}
	if kind == TokenConst || kind == TokenVar || kind == TokenType {
		end := findStmtEnd(file, start+1, limit)
		body.Stmts = append(body.Stmts, newStmt(StmtDecl, start, end))
		return end, body
	}
	if kind == TokenBreak {
		end := findStmtEnd(file, start+1, limit)
		body.Stmts = append(body.Stmts, newStmt(StmtBreak, start, end))
		return end, body
	}
	if kind == TokenContinue {
		end := findStmtEnd(file, start+1, limit)
		body.Stmts = append(body.Stmts, newStmt(StmtContinue, start, end))
		return end, body
	}
	if kind == TokenFallthrough {
		end := findStmtEnd(file, start+1, limit)
		body.Stmts = append(body.Stmts, newStmt(StmtFallthrough, start, end))
		return end, body
	}
	if kind == TokenGoto {
		end := findStmtEnd(file, start+1, limit)
		stmt := newStmt(StmtGoto, start, end)
		stmt.ExprStart, stmt.ExprEnd = trimSpan(file, start+1, end)
		body = appendStmtExpr(body, file, stmt)
		return end, body
	}
	if kind == TokenDefer {
		end := findStmtEnd(file, start+1, limit)
		stmt := newStmt(StmtDefer, start, end)
		stmt.ExprStart, stmt.ExprEnd = trimSpan(file, start+1, end)
		body = appendStmtExpr(body, file, stmt)
		return end, body
	}
	if kind == TokenGo {
		end := findStmtEnd(file, start+1, limit)
		stmt := newStmt(StmtGo, start, end)
		stmt.ExprStart, stmt.ExprEnd = trimSpan(file, start+1, end)
		body = appendStmtExpr(body, file, stmt)
		return end, body
	}
	if tokCharIs(file.Src, file.Tokens, start, '{') {
		closeTok := skipBalanced(file, start, '{', '}') - 1
		if closeTok < start || closeTok >= limit {
			return start, bodyFail(body, BodyErrBlock, start)
		}
		body = parseBlockInto(body, file, start, closeTok)
		return closeTok + 1, body
	}
	if file.Tokens[start].Kind == TokenIdent && tokCharIs(file.Src, file.Tokens, start+1, ':') {
		stmt := newStmt(StmtLabel, start, start+2)
		body.Stmts = append(body.Stmts, stmt)
		return start + 2, body
	}
	end := findStmtEnd(file, start+1, limit)
	stmtKind := StmtExpr
	if spanHasAssign(file, start, end) {
		stmtKind = StmtAssign
	}
	stmt := newStmt(stmtKind, start, end)
	if stmtKind == StmtAssign {
		assign := findAssign(file, start, end)
		stmt.ExprStart, stmt.ExprEnd = trimSpan(file, assign+1, end)
	} else {
		stmt.ExprStart, stmt.ExprEnd = trimSpan(file, start, end)
	}
	body = appendStmtExpr(body, file, stmt)
	return end, body
}

func parseBlockStmt(body Body, file File, start int, limit int, stmtKind int) (int, Body) {
	bodyStart := findStmtBlockStart(file, start+1, limit)
	if bodyStart < 0 {
		return start, bodyFail(body, BodyErrBlock, start)
	}
	bodyEnd := skipBalanced(file, bodyStart, '{', '}')
	if bodyEnd <= bodyStart || bodyEnd > limit+1 {
		return start, bodyFail(body, BodyErrBlock, bodyStart)
	}
	stmt := newStmt(stmtKind, start, bodyEnd)
	stmt.ExprStart, stmt.ExprEnd = trimSpan(file, start+1, bodyStart)
	stmt.BodyStart = bodyStart
	stmt.BodyEnd = bodyEnd
	elseIfStart := -1
	elseBlockStart := -1
	elseBlockEnd := -1
	if stmtKind == StmtIf {
		next := skipStmtSeparators(file, bodyEnd)
		if next < limit && file.Tokens[next].Kind == TokenElse {
			stmt.ElseStart = next
			if next+1 < limit && file.Tokens[next+1].Kind == TokenIf {
				elseEnd := findIfEnd(file, next+1, limit)
				if elseEnd <= next+1 {
					return start, bodyFail(body, BodyErrStmt, next)
				}
				elseIfStart = next + 1
				stmt.ElseEnd = elseEnd
				stmt.EndTok = elseEnd
			} else if tokCharIs(file.Src, file.Tokens, next+1, '{') {
				closeTok := skipBalanced(file, next+1, '{', '}') - 1
				if closeTok <= next || closeTok >= limit {
					return start, bodyFail(body, BodyErrBlock, next+1)
				}
				elseBlockStart = next + 1
				elseBlockEnd = closeTok
				stmt.ElseEnd = closeTok + 1
				stmt.EndTok = closeTok + 1
			} else {
				return start, bodyFail(body, BodyErrStmt, next)
			}
		}
	}
	body = appendStmtExpr(body, file, stmt)
	closeTok := bodyEnd - 1
	body = parseBlockInto(body, file, bodyStart, closeTok)
	if !body.Ok {
		return start, body
	}
	if elseIfStart >= 0 {
		_, body = parseStmt(body, file, elseIfStart, limit)
	} else if elseBlockStart >= 0 {
		body = parseBlockInto(body, file, elseBlockStart, elseBlockEnd)
	}
	return stmt.EndTok, body
}

func findIfEnd(file File, start int, limit int) int {
	bodyStart := findStmtBlockStart(file, start+1, limit)
	if bodyStart < 0 {
		return -1
	}
	bodyEnd := skipBalanced(file, bodyStart, '{', '}')
	if bodyEnd <= bodyStart || bodyEnd > limit+1 {
		return -1
	}
	next := skipStmtSeparators(file, bodyEnd)
	if next < limit && file.Tokens[next].Kind == TokenElse {
		if next+1 < limit && file.Tokens[next+1].Kind == TokenIf {
			return findIfEnd(file, next+1, limit)
		}
		if tokCharIs(file.Src, file.Tokens, next+1, '{') {
			closeTok := skipBalanced(file, next+1, '{', '}') - 1
			if closeTok <= next || closeTok >= limit {
				return -1
			}
			return closeTok + 1
		}
		return -1
	}
	return bodyEnd
}

func appendStmtExpr(body Body, file File, stmt Stmt) Body {
	body.Stmts = append(body.Stmts, stmt)
	if stmt.ExprStart >= 0 && stmt.ExprEnd > stmt.ExprStart {
		body.Exprs = append(body.Exprs, Expr{
			Kind:     classifyExpr(file, stmt.ExprStart, stmt.ExprEnd),
			StartTok: stmt.ExprStart,
			EndTok:   stmt.ExprEnd,
		})
	}
	return body
}

func newStmt(kind int, start int, end int) Stmt {
	return Stmt{
		Kind:      kind,
		StartTok:  start,
		EndTok:    end,
		ExprStart: -1,
		ExprEnd:   -1,
		BodyStart: -1,
		BodyEnd:   -1,
		ElseStart: -1,
		ElseEnd:   -1,
	}
}

func findStmtBlockStart(file File, start int, limit int) int {
	i := start
	parenDepth := 0
	bracketDepth := 0
	for i < limit {
		if tokCharIs(file.Src, file.Tokens, i, '(') {
			parenDepth++
		} else if tokCharIs(file.Src, file.Tokens, i, ')') {
			if parenDepth > 0 {
				parenDepth--
			}
		} else if tokCharIs(file.Src, file.Tokens, i, '[') {
			bracketDepth++
		} else if tokCharIs(file.Src, file.Tokens, i, ']') {
			if bracketDepth > 0 {
				bracketDepth--
			}
		} else if tokCharIs(file.Src, file.Tokens, i, '{') && parenDepth == 0 && bracketDepth == 0 {
			return i
		}
		i++
	}
	return -1
}

func findStmtEnd(file File, start int, limit int) int {
	if start < limit && start > 0 && file.Tokens[start].Line != file.Tokens[start-1].Line {
		return start
	}
	i := start
	parenDepth := 0
	bracketDepth := 0
	braceDepth := 0
	prev := start - 1
	for i < limit {
		if parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 {
			if tokCharIs(file.Src, file.Tokens, i, ';') {
				return i + 1
			}
			if i > start && file.Tokens[i].Line != file.Tokens[prev].Line && !lineContinues(file, prev, i) {
				return i
			}
		}
		if tokCharIs(file.Src, file.Tokens, i, '(') {
			parenDepth++
		} else if tokCharIs(file.Src, file.Tokens, i, ')') {
			if parenDepth > 0 {
				parenDepth--
			}
		} else if tokCharIs(file.Src, file.Tokens, i, '[') {
			bracketDepth++
		} else if tokCharIs(file.Src, file.Tokens, i, ']') {
			if bracketDepth > 0 {
				bracketDepth--
			}
		} else if tokCharIs(file.Src, file.Tokens, i, '{') {
			braceDepth++
		} else if tokCharIs(file.Src, file.Tokens, i, '}') {
			if braceDepth == 0 {
				return i
			}
			braceDepth--
		}
		prev = i
		i++
	}
	return limit
}

func findCaseHeaderEnd(file File, start int, limit int) int {
	i := start
	for i < limit {
		if tokCharIs(file.Src, file.Tokens, i, ':') {
			return i + 1
		}
		i++
	}
	return limit
}

func findTopLevelChar(file File, start int, limit int, c byte) int {
	i := start
	parenDepth := 0
	bracketDepth := 0
	braceDepth := 0
	for i < limit {
		if parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 && tokCharIs(file.Src, file.Tokens, i, c) {
			return i
		}
		if tokCharIs(file.Src, file.Tokens, i, '(') {
			parenDepth++
		} else if tokCharIs(file.Src, file.Tokens, i, ')') {
			if parenDepth > 0 {
				parenDepth--
			}
		} else if tokCharIs(file.Src, file.Tokens, i, '[') {
			bracketDepth++
		} else if tokCharIs(file.Src, file.Tokens, i, ']') {
			if bracketDepth > 0 {
				bracketDepth--
			}
		} else if tokCharIs(file.Src, file.Tokens, i, '{') {
			braceDepth++
		} else if tokCharIs(file.Src, file.Tokens, i, '}') {
			if braceDepth > 0 {
				braceDepth--
			}
		}
		i++
	}
	return -1
}

func spanHasAssign(file File, start int, end int) bool {
	return findAssign(file, start, end) >= 0
}

func findAssign(file File, start int, end int) int {
	for i := start; i < end; i++ {
		if tokenTextIs(file.Src, file.Tokens[i], "=") || tokenTextIs(file.Src, file.Tokens[i], ":=") ||
			tokenTextIs(file.Src, file.Tokens[i], "+=") || tokenTextIs(file.Src, file.Tokens[i], "-=") ||
			tokenTextIs(file.Src, file.Tokens[i], "*=") || tokenTextIs(file.Src, file.Tokens[i], "/=") ||
			tokenTextIs(file.Src, file.Tokens[i], "%=") || tokenTextIs(file.Src, file.Tokens[i], "&=") ||
			tokenTextIs(file.Src, file.Tokens[i], "|=") || tokenTextIs(file.Src, file.Tokens[i], "^=") {
			return i
		}
	}
	return -1
}

func classifyExpr(file File, start int, end int) int {
	start, end = trimSpan(file, start, end)
	if start >= end {
		return ExprOther
	}
	if end-start == 1 {
		kind := file.Tokens[start].Kind
		if kind == TokenIdent {
			return ExprIdent
		}
		if kind == TokenNumber || kind == TokenString || kind == TokenChar {
			return ExprLiteral
		}
	}
	if isUnaryExpr(file, start) {
		return ExprUnary
	}
	if hasTopLevelBinary(file, start, end) {
		return ExprBinary
	}
	if spanHasChar(file, start, end, '{') {
		return ExprComposite
	}
	if spanHasCall(file, start, end) {
		return ExprCall
	}
	if spanHasChar(file, start, end, '[') {
		return ExprIndex
	}
	if spanHasChar(file, start, end, '.') {
		return ExprSelector
	}
	return ExprOther
}

func hasTopLevelBinary(file File, start int, end int) bool {
	parenDepth := 0
	bracketDepth := 0
	braceDepth := 0
	for i := start; i < end; i++ {
		if tokCharIs(file.Src, file.Tokens, i, '(') {
			parenDepth++
		} else if tokCharIs(file.Src, file.Tokens, i, ')') {
			if parenDepth > 0 {
				parenDepth--
			}
		} else if tokCharIs(file.Src, file.Tokens, i, '[') {
			bracketDepth++
		} else if tokCharIs(file.Src, file.Tokens, i, ']') {
			if bracketDepth > 0 {
				bracketDepth--
			}
		} else if tokCharIs(file.Src, file.Tokens, i, '{') {
			braceDepth++
		} else if tokCharIs(file.Src, file.Tokens, i, '}') {
			if braceDepth > 0 {
				braceDepth--
			}
		} else if parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 && isBinaryOp(file, i) {
			return true
		}
	}
	return false
}

func isBinaryOp(file File, i int) bool {
	return tokenTextIs(file.Src, file.Tokens[i], "||") || tokenTextIs(file.Src, file.Tokens[i], "&&") ||
		tokenTextIs(file.Src, file.Tokens[i], "==") || tokenTextIs(file.Src, file.Tokens[i], "!=") ||
		tokenTextIs(file.Src, file.Tokens[i], "<") || tokenTextIs(file.Src, file.Tokens[i], "<=") ||
		tokenTextIs(file.Src, file.Tokens[i], ">") || tokenTextIs(file.Src, file.Tokens[i], ">=") ||
		tokenTextIs(file.Src, file.Tokens[i], "+") || tokenTextIs(file.Src, file.Tokens[i], "-") ||
		tokenTextIs(file.Src, file.Tokens[i], "*") || tokenTextIs(file.Src, file.Tokens[i], "/") ||
		tokenTextIs(file.Src, file.Tokens[i], "%") || tokenTextIs(file.Src, file.Tokens[i], "&") ||
		tokenTextIs(file.Src, file.Tokens[i], "|") || tokenTextIs(file.Src, file.Tokens[i], "^") ||
		tokenTextIs(file.Src, file.Tokens[i], "<<") || tokenTextIs(file.Src, file.Tokens[i], ">>") ||
		tokenTextIs(file.Src, file.Tokens[i], "&^")
}

func isUnaryExpr(file File, start int) bool {
	return tokenTextIs(file.Src, file.Tokens[start], "+") || tokenTextIs(file.Src, file.Tokens[start], "-") ||
		tokenTextIs(file.Src, file.Tokens[start], "!") || tokenTextIs(file.Src, file.Tokens[start], "^") ||
		tokenTextIs(file.Src, file.Tokens[start], "*") || tokenTextIs(file.Src, file.Tokens[start], "&") ||
		tokenTextIs(file.Src, file.Tokens[start], "<-")
}

func spanHasCall(file File, start int, end int) bool {
	for i := start + 1; i < end; i++ {
		if tokCharIs(file.Src, file.Tokens, i, '(') {
			return true
		}
	}
	return false
}

func spanHasChar(file File, start int, end int, c byte) bool {
	for i := start; i < end; i++ {
		if tokCharIs(file.Src, file.Tokens, i, c) {
			return true
		}
	}
	return false
}

func trimSpan(file File, start int, end int) (int, int) {
	for start < end && tokCharIs(file.Src, file.Tokens, start, ';') {
		start++
	}
	for end > start && tokCharIs(file.Src, file.Tokens, end-1, ';') {
		end--
	}
	return start, end
}

func skipStmtSeparators(file File, start int) int {
	for start < len(file.Tokens) && tokCharIs(file.Src, file.Tokens, start, ';') {
		start++
	}
	return start
}

func lineContinues(file File, prev int, next int) bool {
	if prev < 0 || next < 0 || prev >= len(file.Tokens) || next >= len(file.Tokens) {
		return false
	}
	if isBinaryOp(file, prev) || tokenTextIs(file.Src, file.Tokens[prev], ",") || tokenTextIs(file.Src, file.Tokens[prev], ".") {
		return true
	}
	if tokCharIs(file.Src, file.Tokens, next, '.') || tokCharIs(file.Src, file.Tokens, next, ',') {
		return true
	}
	return false
}

func tokenTextIs(src []byte, tok Token, text string) bool {
	if tok.End-tok.Start != len(text) || tok.Start < 0 || tok.End > len(src) {
		return false
	}
	for i := 0; i < len(text); i++ {
		if src[tok.Start+i] != text[i] {
			return false
		}
	}
	return true
}

func bodyFail(body Body, err int, tok int) Body {
	body.Ok = false
	body.Error = err
	body.ErrorTok = tok
	return body
}
