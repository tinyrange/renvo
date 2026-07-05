package check

import "j5.nz/rtg/rtg/internal/syntax"

const (
	NameReceiver = iota + 1
	NameParam
	NameResult
	NameLocal
	NameLabel
)

type FuncScope struct {
	Names []ScopeName
}

type ScopeName struct {
	Name  string
	Kind  int
	Token int
}

func LookupScopeName(scope FuncScope, name string) int {
	for i := 0; i < len(scope.Names); i++ {
		if scope.Names[i].Name == name {
			return i
		}
	}
	return -1
}

func buildFuncScope(file syntax.File, fn syntax.FuncDecl, body syntax.Body) (FuncScope, bool, int) {
	var scope FuncScope
	if fn.ReceiverStart >= 0 {
		tok := receiverNameToken(file, fn)
		if tok >= 0 {
			if !addScopeName(&scope, tokenString(file, tok), NameReceiver, tok, true, false) {
				return scope, false, tok
			}
		}
	}
	if fn.ParamsStart >= 0 && fn.ParamsEnd > fn.ParamsStart {
		ok, tok := collectFieldNames(file, fn.ParamsStart+1, fn.ParamsEnd-1, NameParam, &scope)
		if !ok {
			return scope, false, tok
		}
	}
	if fn.ResultStart >= 0 && fn.ResultEnd > fn.ResultStart && tokCharIs(file, fn.ResultStart, '(') {
		end := fn.ResultEnd - 1
		if tokCharIs(file, end, ')') {
			ok, tok := collectFieldNames(file, fn.ResultStart+1, end, NameResult, &scope)
			if !ok {
				return scope, false, tok
			}
		}
	}
	for i := 0; i < len(body.Stmts); i++ {
		stmt := body.Stmts[i]
		if stmt.Kind == syntax.StmtDecl {
			collectDeclNames(file, stmt, &scope)
		} else if stmt.Kind == syntax.StmtAssign {
			collectShortDeclNames(file, stmt, &scope)
		} else if stmt.Kind == syntax.StmtLabel {
			if stmt.StartTok >= 0 && stmt.StartTok < len(file.Tokens) {
				name := tokenString(file, stmt.StartTok)
				if !addScopeName(&scope, name, NameLabel, stmt.StartTok, true, true) {
					return scope, false, stmt.StartTok
				}
			}
		}
	}
	return scope, true, -1
}

func receiverNameToken(file syntax.File, fn syntax.FuncDecl) int {
	start := fn.ReceiverStart
	end := fn.ReceiverEnd
	if start < 0 || end <= start || end > len(file.Tokens) {
		return -1
	}
	if file.Tokens[start].Kind != syntax.TokenIdent {
		return -1
	}
	if end-start <= 1 {
		return -1
	}
	return start
}

func collectFieldNames(file syntax.File, start int, end int, kind int, scope *FuncScope) (bool, int) {
	pending := make([]int, 0, 2)
	i := start
	for i < end {
		segStart := i
		segEnd := nextTopLevelComma(file, i, end)
		first := firstNonSeparator(file, segStart, segEnd)
		if first < segEnd && file.Tokens[first].Kind == syntax.TokenIdent {
			next := first + 1
			if next >= segEnd {
				pending = append(pending, first)
			} else if tokCharIs(file, next, '.') {
				pending = pending[:0]
			} else {
				if !addPendingNames(file, pending, kind, scope) {
					return false, pending[0]
				}
				pending = pending[:0]
				if !addScopeName(scope, tokenString(file, first), kind, first, true, false) {
					return false, first
				}
			}
		} else {
			pending = pending[:0]
		}
		i = segEnd + 1
	}
	return true, -1
}

func addPendingNames(file syntax.File, pending []int, kind int, scope *FuncScope) bool {
	for i := 0; i < len(pending); i++ {
		if !addScopeName(scope, tokenString(file, pending[i]), kind, pending[i], true, false) {
			return false
		}
	}
	return true
}

func collectDeclNames(file syntax.File, stmt syntax.Stmt, scope *FuncScope) {
	start := stmt.StartTok + 1
	end := stmt.EndTok
	if start >= end {
		return
	}
	if tokCharIs(file, start, '(') {
		i := start + 1
		for i < end {
			i = skipLocalSeparators(file, i, end)
			if i >= end || tokCharIs(file, i, ')') {
				break
			}
			collectLeadingIdentList(file, i, statementSpecEnd(file, i, end), scope)
			i = statementSpecEnd(file, i, end)
		}
		return
	}
	collectLeadingIdentList(file, start, end, scope)
}

func collectShortDeclNames(file syntax.File, stmt syntax.Stmt, scope *FuncScope) {
	assign := findTokenText(file, stmt.StartTok, stmt.EndTok, ":=")
	if assign < 0 {
		return
	}
	i := stmt.StartTok
	for i < assign {
		if file.Tokens[i].Kind == syntax.TokenIdent && tokenString(file, i) != "_" {
			if LookupScopeName(*scope, tokenString(file, i)) < 0 {
				addScopeName(scope, tokenString(file, i), NameLocal, i, false, false)
			}
		}
		i++
	}
}

func collectLeadingIdentList(file syntax.File, start int, end int, scope *FuncScope) {
	i := start
	for i < end {
		if file.Tokens[i].Kind != syntax.TokenIdent {
			return
		}
		name := tokenString(file, i)
		if name != "_" && LookupScopeName(*scope, name) < 0 {
			addScopeName(scope, name, NameLocal, i, false, false)
		}
		i++
		if i < end && tokCharIs(file, i, ',') {
			i++
			continue
		}
		return
	}
}

func statementSpecEnd(file syntax.File, start int, end int) int {
	line := file.Tokens[start].Line
	i := start
	for i < end {
		if tokCharIs(file, i, ';') {
			return i + 1
		}
		if i > start && file.Tokens[i].Line != line {
			return i
		}
		i++
	}
	return end
}

func firstNonSeparator(file syntax.File, start int, end int) int {
	for start < end && tokCharIs(file, start, ',') {
		start++
	}
	return start
}

func nextTopLevelComma(file syntax.File, start int, end int) int {
	parenDepth := 0
	bracketDepth := 0
	braceDepth := 0
	i := start
	for i < end {
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
		} else if parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 && tokCharIs(file, i, ',') {
			return i
		}
		i++
	}
	return end
}

func skipLocalSeparators(file syntax.File, start int, end int) int {
	for start < end && tokCharIs(file, start, ';') {
		start++
	}
	return start
}

func findTokenText(file syntax.File, start int, end int, text string) int {
	for i := start; i < end; i++ {
		if tokenTextIs(file, i, text) {
			return i
		}
	}
	return -1
}

func addScopeName(scope *FuncScope, name string, kind int, tok int, rejectDup bool, labelsOnly bool) bool {
	if name == "" || name == "_" {
		return true
	}
	if rejectDup {
		for i := 0; i < len(scope.Names); i++ {
			if scope.Names[i].Name != name {
				continue
			}
			if labelsOnly {
				if scope.Names[i].Kind == NameLabel {
					return false
				}
				continue
			}
			if scope.Names[i].Kind != NameLabel {
				return false
			}
		}
	}
	scope.Names = append(scope.Names, ScopeName{Name: name, Kind: kind, Token: tok})
	return true
}

func tokCharIs(file syntax.File, tok int, c byte) bool {
	if tok < 0 || tok >= len(file.Tokens) {
		return false
	}
	if file.Tokens[tok].Kind != syntax.TokenOperator {
		return false
	}
	if file.Tokens[tok].End-file.Tokens[tok].Start != 1 {
		return false
	}
	return file.Src[file.Tokens[tok].Start] == c
}

func tokenTextIs(file syntax.File, tok int, text string) bool {
	if tok < 0 || tok >= len(file.Tokens) {
		return false
	}
	token := file.Tokens[tok]
	if token.End-token.Start != len(text) || token.Start < 0 || token.End > len(file.Src) {
		return false
	}
	for i := 0; i < len(text); i++ {
		if file.Src[token.Start+i] != text[i] {
			return false
		}
	}
	return true
}
