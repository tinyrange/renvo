package check

import "renvo.dev/internal/syntax"

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
	assigns := make([]AssignInfo, 0, countBodyStatements(body, syntax.StmtAssign))
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
	returns := make([]ReturnInfo, 0, countBodyStatements(body, syntax.StmtReturn))
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
	spans := splitExprList(file, start, end)
	targets := make([]AssignTarget, 0, len(spans))
	for i := 0; i < len(spans); i++ {
		span := spans[i]
		if span.EndTok-span.StartTok != 1 || span.StartTok < 0 || span.StartTok >= len(file.Tokens) {
			continue
		}
		if file.Tokens[span.StartTok].KindLine&255 != syntax.TokenIdent {
			continue
		}
		name := tokenString(&file, span.StartTok)
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
	spans = make([]ExprSpan, 0, countExprListItems(file, start, end))
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

func countExprListItems(file syntax.File, start int, end int) int {
	count := 0
	i := start
	for i < end {
		next := nextTopLevelComma(file, i, end)
		itemStart, itemEnd := trimExprSpan(file, i, next)
		if itemEnd > itemStart {
			count++
		}
		i = next + 1
	}
	return count
}

func countBodyStatements(body syntax.Body, kind int) int {
	count := 0
	for i := 0; i < len(body.Stmts); i++ {
		if body.Stmts[i].Kind == kind {
			count++
		}
	}
	return count
}

func findTopLevelAssignOp(file syntax.File, start int, end int) int {
	parenDepth := 0
	bracketDepth := 0
	braceDepth := 0
	for i := start; i < end; i++ {
		if parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 && isAssignOp(file, i) {
			return i
		}
		tok := file.Tokens[i]
		c := byte(0)
		if tok.KindLine&255 == syntax.TokenOperator && tok.End == tok.Start+1 {
			c = file.Src[tok.Start]
		}
		if c == '(' {
			parenDepth++
		} else if c == ')' {
			if parenDepth > 0 {
				parenDepth--
			}
		} else if c == '[' {
			bracketDepth++
		} else if c == ']' {
			if bracketDepth > 0 {
				bracketDepth--
			}
		} else if c == '{' {
			braceDepth++
		} else if c == '}' {
			if braceDepth > 0 {
				braceDepth--
			}
		}
	}
	return -1
}

func isAssignOp(file syntax.File, tok int) bool {
	if tok < 0 || tok >= len(file.Tokens) {
		return false
	}
	token := file.Tokens[tok]
	if token.KindLine&255 != syntax.TokenOperator || token.Start < 0 || token.End > len(file.Src) {
		return false
	}
	size := token.End - token.Start
	if size == 1 {
		return file.Src[token.Start] == '='
	}
	if size == 2 && file.Src[token.Start+1] == '=' {
		first := file.Src[token.Start]
		return first == ':' || first == '+' || first == '-' || first == '*' || first == '/' || first == '%' || first == '&' || first == '|' || first == '^'
	}
	if size == 3 && file.Src[token.Start+2] == '=' {
		first := file.Src[token.Start]
		second := file.Src[token.Start+1]
		return first == '<' && second == '<' || first == '>' && second == '>' || first == '&' && second == '^'
	}
	return false
}

func assignKind(file syntax.File, tok int) int {
	if tokenTextIs(&file, tok, "=") {
		return AssignSet
	}
	if tokenTextIs(&file, tok, ":=") {
		return AssignDefine
	}
	if tokenTextIs(&file, tok, "+=") {
		return AssignAdd
	}
	if tokenTextIs(&file, tok, "-=") {
		return AssignSub
	}
	if tokenTextIs(&file, tok, "*=") {
		return AssignMul
	}
	if tokenTextIs(&file, tok, "/=") {
		return AssignDiv
	}
	if tokenTextIs(&file, tok, "%=") {
		return AssignMod
	}
	if tokenTextIs(&file, tok, "&=") {
		return AssignAnd
	}
	if tokenTextIs(&file, tok, "|=") {
		return AssignOr
	}
	if tokenTextIs(&file, tok, "^=") {
		return AssignXor
	}
	return AssignUnknown
}

func trimExprSpan(file syntax.File, start int, end int) (int, int) {
	for start < end && (tokCharIs(&file, start, ';') || tokCharIs(&file, start, ',')) {
		start++
	}
	for end > start && (tokCharIs(&file, end-1, ';') || tokCharIs(&file, end-1, ',')) {
		end--
	}
	return start, end
}
