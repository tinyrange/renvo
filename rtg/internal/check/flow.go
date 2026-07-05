package check

import "j5.nz/rtg/rtg/internal/syntax"

const (
	AssignUnknown = iota
	AssignSet
	AssignDefine
	AssignAdd
	AssignSub
	AssignMul
	AssignDiv
	AssignMod
	AssignAnd
	AssignOr
	AssignXor
)

type ExprSpan struct {
	StartTok int
	EndTok   int
}

type AssignInfo struct {
	Kind       int
	StartTok   int
	EndTok     int
	OpTok      int
	LeftStart  int
	LeftEnd    int
	RightStart int
	RightEnd   int
	Targets    []AssignTarget
	Values     []ExprSpan
}

type AssignTarget struct {
	Name  string
	Token int
	Span  ExprSpan
	Ref   NameRef
}

type ReturnInfo struct {
	StartTok int
	EndTok   int
	Values   []ExprSpan
}

func LookupAssignTarget(assign AssignInfo, name string) int {
	for i := 0; i < len(assign.Targets); i++ {
		if assign.Targets[i].Name == name {
			return i
		}
	}
	return -1
}

func buildFuncAssignments(file syntax.File, fileIndex int, info PackageInfo, body syntax.Body, scope FuncScope) []AssignInfo {
	var assigns []AssignInfo
	for i := 0; i < len(body.Stmts); i++ {
		stmt := body.Stmts[i]
		if stmt.Kind != syntax.StmtAssign {
			continue
		}
		opTok := findTopLevelAssignOp(file, stmt.StartTok, stmt.EndTok)
		if opTok < 0 {
			continue
		}
		leftStart, leftEnd := trimExprSpan(file, stmt.StartTok, opTok)
		rightStart, rightEnd := trimExprSpan(file, opTok+1, stmt.EndTok)
		assign := AssignInfo{
			Kind:       assignKind(file, opTok),
			StartTok:   stmt.StartTok,
			EndTok:     stmt.EndTok,
			OpTok:      opTok,
			LeftStart:  leftStart,
			LeftEnd:    leftEnd,
			RightStart: rightStart,
			RightEnd:   rightEnd,
			Targets:    buildAssignTargets(file, fileIndex, info, scope, leftStart, leftEnd),
			Values:     splitExprList(file, rightStart, rightEnd),
		}
		assigns = append(assigns, assign)
	}
	return assigns
}

func buildFuncReturns(file syntax.File, body syntax.Body) []ReturnInfo {
	var returns []ReturnInfo
	for i := 0; i < len(body.Stmts); i++ {
		stmt := body.Stmts[i]
		if stmt.Kind != syntax.StmtReturn {
			continue
		}
		returns = append(returns, ReturnInfo{
			StartTok: stmt.StartTok,
			EndTok:   stmt.EndTok,
			Values:   splitExprList(file, stmt.ExprStart, stmt.ExprEnd),
		})
	}
	return returns
}

func buildAssignTargets(file syntax.File, fileIndex int, info PackageInfo, scope FuncScope, start int, end int) []AssignTarget {
	var targets []AssignTarget
	spans := splitExprList(file, start, end)
	for i := 0; i < len(spans); i++ {
		span := spans[i]
		if span.EndTok-span.StartTok != 1 || span.StartTok < 0 || span.StartTok >= len(file.Tokens) {
			continue
		}
		if file.Tokens[span.StartTok].Kind != syntax.TokenIdent {
			continue
		}
		name := tokenString(file, span.StartTok)
		if name == "_" {
			continue
		}
		targets = append(targets, AssignTarget{
			Name:  name,
			Token: span.StartTok,
			Span:  span,
			Ref:   resolveNameRef(fileIndex, info, scope, name, span.StartTok),
		})
	}
	return targets
}

func splitExprList(file syntax.File, start int, end int) []ExprSpan {
	start, end = trimExprSpan(file, start, end)
	var spans []ExprSpan
	if start < 0 || end <= start {
		return spans
	}
	i := start
	for i < end {
		next := nextTopLevelComma(file, i, end)
		itemStart, itemEnd := trimExprSpan(file, i, next)
		if itemEnd > itemStart {
			spans = append(spans, ExprSpan{StartTok: itemStart, EndTok: itemEnd})
		}
		i = next + 1
	}
	return spans
}

func findTopLevelAssignOp(file syntax.File, start int, end int) int {
	parenDepth := 0
	bracketDepth := 0
	braceDepth := 0
	for i := start; i < end; i++ {
		if parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 && isAssignOp(file, i) {
			return i
		}
		if tokCharIs(file, i, '(') {
			parenDepth++
		} else if tokCharIs(file, i, ')') {
			if parenDepth > 0 {
				parenDepth--
			}
		} else if tokCharIs(file, i, '[') {
			bracketDepth++
		} else if tokCharIs(file, i, ']') {
			if bracketDepth > 0 {
				bracketDepth--
			}
		} else if tokCharIs(file, i, '{') {
			braceDepth++
		} else if tokCharIs(file, i, '}') {
			if braceDepth > 0 {
				braceDepth--
			}
		}
	}
	return -1
}

func isAssignOp(file syntax.File, tok int) bool {
	if tokenTextIs(file, tok, "=") || tokenTextIs(file, tok, ":=") {
		return true
	}
	if tokenTextIs(file, tok, "+=") || tokenTextIs(file, tok, "-=") {
		return true
	}
	if tokenTextIs(file, tok, "*=") || tokenTextIs(file, tok, "/=") {
		return true
	}
	if tokenTextIs(file, tok, "%=") || tokenTextIs(file, tok, "&=") {
		return true
	}
	return tokenTextIs(file, tok, "|=") || tokenTextIs(file, tok, "^=")
}

func assignKind(file syntax.File, tok int) int {
	if tokenTextIs(file, tok, "=") {
		return AssignSet
	}
	if tokenTextIs(file, tok, ":=") {
		return AssignDefine
	}
	if tokenTextIs(file, tok, "+=") {
		return AssignAdd
	}
	if tokenTextIs(file, tok, "-=") {
		return AssignSub
	}
	if tokenTextIs(file, tok, "*=") {
		return AssignMul
	}
	if tokenTextIs(file, tok, "/=") {
		return AssignDiv
	}
	if tokenTextIs(file, tok, "%=") {
		return AssignMod
	}
	if tokenTextIs(file, tok, "&=") {
		return AssignAnd
	}
	if tokenTextIs(file, tok, "|=") {
		return AssignOr
	}
	if tokenTextIs(file, tok, "^=") {
		return AssignXor
	}
	return AssignUnknown
}

func trimExprSpan(file syntax.File, start int, end int) (int, int) {
	for start < end && (tokCharIs(file, start, ';') || tokCharIs(file, start, ',')) {
		start++
	}
	for end > start && (tokCharIs(file, end-1, ';') || tokCharIs(file, end-1, ',')) {
		end--
	}
	return start, end
}
