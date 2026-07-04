package check

import "j5.nz/rtg/rtg/internal/syntax"

func LookupDecl(info PackageInfo, name string) int {
	for i := 0; i < len(info.Decls); i++ {
		if info.Decls[i].Name == name {
			return i
		}
	}
	return -1
}

func LookupLocalDecl(body FuncBody, name string) int {
	for i := 0; i < len(body.Locals); i++ {
		if body.Locals[i].Name == name {
			return i
		}
	}
	return -1
}

func buildDeclInfo(file syntax.File, fileIndex int, info PackageInfo, decl syntax.TopDecl) DeclInfo {
	name := tokenString(file, decl.NameTok)
	out := DeclInfo{
		Name:       name,
		Kind:       declSymbolKind(decl.Kind),
		File:       fileIndex,
		Token:      decl.NameTok,
		Symbol:     LookupPackageSymbol(info, name),
		TypeStart:  -1,
		TypeEnd:    -1,
		ValueStart: -1,
		ValueEnd:   -1,
	}
	if decl.Kind == syntax.TokenType {
		typeStart := decl.NameTok + 1
		if tokenTextIs(file, typeStart, "=") {
			out.Alias = true
			typeStart++
		}
		out.TypeStart, out.TypeEnd = trimDeclSpan(file, typeStart, decl.EndTok)
		return out
	}
	typeStart := declNameListEnd(file, decl)
	valueStart := findDeclAssign(file, typeStart, decl.EndTok)
	if valueStart >= 0 {
		out.TypeStart, out.TypeEnd = trimDeclSpan(file, typeStart, valueStart)
		out.ValueStart, out.ValueEnd = trimDeclSpan(file, valueStart+1, decl.EndTok)
		out.Values = splitExprList(file, out.ValueStart, out.ValueEnd)
	} else {
		out.TypeStart, out.TypeEnd = trimDeclSpan(file, typeStart, decl.EndTok)
	}
	return out
}

func buildFuncLocalDecls(file syntax.File, fileIndex int, body syntax.Body, scope FuncScope) []LocalDeclInfo {
	var decls []LocalDeclInfo
	for i := 0; i < len(body.Stmts); i++ {
		stmt := body.Stmts[i]
		if stmt.Kind != syntax.StmtDecl || stmt.StartTok < 0 || stmt.StartTok >= len(file.Tokens) {
			continue
		}
		kind := declSymbolKind(file.Tokens[stmt.StartTok].Kind)
		start := stmt.StartTok + 1
		end := stmt.EndTok
		if start < end && tokCharIs(file, start, '(') {
			j := start + 1
			for j < end {
				j = skipLocalSeparators(file, j, end)
				if j >= end || tokCharIs(file, j, ')') {
					break
				}
				specEnd := statementSpecEnd(file, j, end)
				decls = appendLocalDeclSpec(decls, file, fileIndex, scope, kind, j, specEnd)
				if specEnd <= j {
					j++
				} else {
					j = specEnd
				}
			}
			continue
		}
		decls = appendLocalDeclSpec(decls, file, fileIndex, scope, kind, start, end)
	}
	return decls
}

func appendLocalDeclSpec(decls []LocalDeclInfo, file syntax.File, fileIndex int, scope FuncScope, kind int, start int, end int) []LocalDeclInfo {
	start, end = trimDeclSpan(file, start, end)
	if start < 0 || end <= start || start >= len(file.Tokens) || file.Tokens[start].Kind != syntax.TokenIdent {
		return decls
	}
	if kind == SymbolType {
		return appendLocalTypeDecl(decls, file, fileIndex, scope, start, end)
	}
	names, namesEnd := localDeclNameTokens(file, start, end)
	if len(names) == 0 {
		return decls
	}
	valueStart := findDeclAssign(file, namesEnd, end)
	typeStart := -1
	typeEnd := -1
	valueSpanStart := -1
	valueSpanEnd := -1
	if valueStart >= 0 {
		typeStart, typeEnd = trimDeclSpan(file, namesEnd, valueStart)
		valueSpanStart, valueSpanEnd = trimDeclSpan(file, valueStart+1, end)
	} else {
		typeStart, typeEnd = trimDeclSpan(file, namesEnd, end)
	}
	values := splitExprList(file, valueSpanStart, valueSpanEnd)
	for i := 0; i < len(names); i++ {
		name := tokenString(file, names[i])
		if name == "_" {
			continue
		}
		decls = append(decls, LocalDeclInfo{
			Name:       name,
			Kind:       kind,
			File:       fileIndex,
			Token:      names[i],
			Scope:      LookupScopeName(scope, name),
			TypeStart:  typeStart,
			TypeEnd:    typeEnd,
			ValueStart: valueSpanStart,
			ValueEnd:   valueSpanEnd,
			Values:     values,
		})
	}
	return decls
}

func appendLocalTypeDecl(decls []LocalDeclInfo, file syntax.File, fileIndex int, scope FuncScope, start int, end int) []LocalDeclInfo {
	name := tokenString(file, start)
	if name == "_" {
		return decls
	}
	typeStart := start + 1
	alias := false
	if tokenTextIs(file, typeStart, "=") {
		alias = true
		typeStart++
	}
	typeStart, typeEnd := trimDeclSpan(file, typeStart, end)
	return append(decls, LocalDeclInfo{
		Name:      name,
		Kind:      SymbolType,
		File:      fileIndex,
		Token:     start,
		Scope:     LookupScopeName(scope, name),
		TypeStart: typeStart,
		TypeEnd:   typeEnd,
		Alias:     alias,
	})
}

func localDeclNameTokens(file syntax.File, start int, end int) ([]int, int) {
	var names []int
	i := start
	for i < end {
		if file.Tokens[i].Kind != syntax.TokenIdent {
			break
		}
		names = append(names, i)
		i++
		if i < end && tokCharIs(file, i, ',') {
			i++
			continue
		}
		break
	}
	return names, i
}

func declNameListEnd(file syntax.File, decl syntax.TopDecl) int {
	i := decl.StartTok + 1
	for i < decl.EndTok {
		if !tokCharIs(file, i, ',') {
			return i
		}
		i++
		if i >= decl.EndTok || file.Tokens[i].Kind != syntax.TokenIdent {
			return i
		}
		i++
	}
	return i
}

func findDeclAssign(file syntax.File, start int, end int) int {
	parenDepth := 0
	bracketDepth := 0
	braceDepth := 0
	for i := start; i < end; i++ {
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
		} else if parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 && tokenTextIs(file, i, "=") {
			return i
		}
	}
	return -1
}

func trimDeclSpan(file syntax.File, start int, end int) (int, int) {
	for start < end && isDeclSpanSeparator(file, start) {
		start++
	}
	for end > start && isDeclSpanSeparator(file, end-1) {
		end--
	}
	if start >= end {
		return -1, -1
	}
	return start, end
}

func isDeclSpanSeparator(file syntax.File, tok int) bool {
	return tokCharIs(file, tok, ';')
}

func sortDecls(decls []DeclInfo) {
	for i := 1; i < len(decls); i++ {
		item := decls[i]
		j := i - 1
		for j >= 0 && declAfter(decls[j], item) {
			decls[j+1] = decls[j]
			j--
		}
		decls[j+1] = item
	}
}

func declAfter(left DeclInfo, right DeclInfo) bool {
	if left.Name != right.Name {
		return left.Name > right.Name
	}
	if left.File != right.File {
		return left.File > right.File
	}
	return left.Token > right.Token
}
