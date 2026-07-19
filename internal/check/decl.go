package check

import "renvo.dev/internal/syntax"

func LookupDecl(info PackageInfo, name string) int {
	for i := 0; i < len(info.Decls); i++ {
		if info.Decls[i].Name == name {
			return i
		}
	}
	return -1
}

func LookupDeclRef(decl DeclInfo, name string, kind int) int {
	for i := 0; i < len(decl.Refs); i++ {
		if decl.Refs[i].Name == name && decl.Refs[i].Kind == kind {
			return i
		}
	}
	return -1
}

func LookupDeclSelector(decl DeclInfo, base string, name string, kind int) int {
	for i := 0; i < len(decl.Selectors); i++ {
		selector := decl.Selectors[i]
		if selector.BaseName == base && selector.Name == name && selector.Kind == kind {
			return i
		}
	}
	return -1
}

func LookupDeclCall(decl DeclInfo, base string, name string, kind int) int {
	for i := 0; i < len(decl.Calls); i++ {
		call := decl.Calls[i]
		if call.BaseName == base && call.Name == name && call.Kind == kind {
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

func LookupLocalDeclRef(decl LocalDeclInfo, name string, kind int) int {
	for i := 0; i < len(decl.Refs); i++ {
		if decl.Refs[i].Name == name && decl.Refs[i].Kind == kind {
			return i
		}
	}
	return -1
}

func LookupLocalDeclSelector(decl LocalDeclInfo, base string, name string, kind int) int {
	for i := 0; i < len(decl.Selectors); i++ {
		selector := decl.Selectors[i]
		if selector.BaseName == base && selector.Name == name && selector.Kind == kind {
			return i
		}
	}
	return -1
}

func LookupLocalDeclCall(decl LocalDeclInfo, base string, name string, kind int) int {
	for i := 0; i < len(decl.Calls); i++ {
		call := decl.Calls[i]
		if call.BaseName == base && call.Name == name && call.Kind == kind {
			return i
		}
	}
	return -1
}

func buildDeclInfo(file syntax.File, fileIndex int, info PackageInfo, checked []PackageInfo, decl syntax.TopDecl) DeclInfo {
	name := tokenString(&file, decl.NameTok)
	out := DeclInfo{
		Name:       name,
		Kind:       declSymbolKind(decl.Kind),
		File:       fileIndex,
		Token:      decl.NameTok,
		Symbol:     LookupPackageSymbol(info, name),
		ValueIndex: declNameIndex(file, decl),
		TypeStart:  -1,
		TypeEnd:    -1,
		ValueStart: -1,
		ValueEnd:   -1,
	}
	if decl.Kind == syntax.TokenType {
		typeStart := decl.NameTok + 1
		if tokenTextIs(&file, typeStart, "=") {
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
		out.Refs = appendExprRefs(out.Refs, file, fileIndex, info, FuncScope{}, out.ValueStart, out.ValueEnd)
		out.Selectors = appendExprSelectors(out.Selectors, file, fileIndex, info, checked, FuncScope{}, out.ValueStart, out.ValueEnd)
		out.Calls = appendExprCalls(out.Calls, file, fileIndex, info, checked, FuncScope{}, out.ValueStart, out.ValueEnd)
		out.Indexes = appendExprIndexes(out.Indexes, &file, out.ValueStart, out.ValueEnd)
		out.Composites = appendExprComposites(out.Composites, file, out.ValueStart, out.ValueEnd)
		if out.Kind == SymbolConst {
			out.Const = evalConstValue(file, out.Values, out.ValueIndex)
		}
	} else {
		out.TypeStart, out.TypeEnd = trimDeclSpan(file, typeStart, decl.EndTok)
	}
	return out
}

func buildFuncLocalDecls(file syntax.File, fileIndex int, info PackageInfo, checked []PackageInfo, body syntax.Body, scope FuncScope) []LocalDeclInfo {
	decls := make([]LocalDeclInfo, 0, countBodyStatements(body, syntax.StmtDecl))
	for i := 0; i < len(body.Stmts); i++ {
		stmt := body.Stmts[i]
		if stmt.Kind != syntax.StmtDecl || stmt.StartTok < 0 || stmt.StartTok >= len(file.Tokens) {
			continue
		}
		kind := declSymbolKind(file.Tokens[stmt.StartTok].KindLine & 255)
		start := stmt.StartTok + 1
		end := stmt.EndTok
		if start < end && tokCharIs(&file, start, '(') {
			j := start + 1
			for j < end {
				j = skipLocalSeparators(file, j, end)
				if j >= end || tokCharIs(&file, j, ')') {
					break
				}
				specEnd := statementSpecEnd(file, j, end)
				decls = appendLocalDeclSpec(decls, file, fileIndex, info, checked, scope, kind, j, specEnd)
				if specEnd <= j {
					j++
				} else {
					j = specEnd
				}
			}
			continue
		}
		decls = appendLocalDeclSpec(decls, file, fileIndex, info, checked, scope, kind, start, end)
	}
	return decls
}

func appendLocalDeclSpec(decls []LocalDeclInfo, file syntax.File, fileIndex int, info PackageInfo, checked []PackageInfo, scope FuncScope, kind int, start int, end int) []LocalDeclInfo {
	start, end = trimDeclSpan(file, start, end)
	if start < 0 || end <= start || start >= len(file.Tokens) || file.Tokens[start].KindLine&255 != syntax.TokenIdent {
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
	refs := appendExprRefs(nil, file, fileIndex, info, scope, valueSpanStart, valueSpanEnd)
	selectors := appendExprSelectors(nil, file, fileIndex, info, checked, scope, valueSpanStart, valueSpanEnd)
	calls := appendExprCalls(nil, file, fileIndex, info, checked, scope, valueSpanStart, valueSpanEnd)
	indexes := appendExprIndexes(nil, &file, valueSpanStart, valueSpanEnd)
	composites := appendExprComposites(nil, file, valueSpanStart, valueSpanEnd)
	for i := 0; i < len(names); i++ {
		name := tokenString(&file, names[i])
		if name == "_" {
			continue
		}
		decls = append(decls, LocalDeclInfo{
			Name:       name,
			Kind:       kind,
			File:       fileIndex,
			Token:      names[i],
			Scope:      LookupScopeName(scope, name),
			ValueIndex: i,
			TypeStart:  typeStart,
			TypeEnd:    typeEnd,
			ValueStart: valueSpanStart,
			ValueEnd:   valueSpanEnd,
			Values:     values,
			Refs:       refs,
			Selectors:  selectors,
			Calls:      calls,
			Indexes:    indexes,
			Composites: composites,
		})
		if kind == SymbolConst {
			decls[len(decls)-1].Const = evalConstValue(file, values, i)
		}
	}
	return decls
}

func appendLocalTypeDecl(decls []LocalDeclInfo, file syntax.File, fileIndex int, scope FuncScope, start int, end int) []LocalDeclInfo {
	name := tokenString(&file, start)
	if name == "_" {
		return decls
	}
	typeStart := start + 1
	alias := false
	if tokenTextIs(&file, typeStart, "=") {
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
	capacity := end - start
	if capacity < 0 {
		capacity = 0
	}
	names := make([]int, 0, capacity)
	i := start
	for i < end {
		if file.Tokens[i].KindLine&255 != syntax.TokenIdent {
			break
		}
		names = append(names, i)
		i++
		if i < end && tokCharIs(&file, i, ',') {
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
		if !tokCharIs(&file, i, ',') {
			return i
		}
		i++
		if i >= decl.EndTok || file.Tokens[i].KindLine&255 != syntax.TokenIdent {
			return i
		}
		i++
	}
	return i
}

func declNameIndex(file syntax.File, decl syntax.TopDecl) int {
	index := 0
	i := decl.StartTok
	for i < decl.EndTok {
		if i == decl.NameTok {
			return index
		}
		if file.Tokens[i].KindLine&255 == syntax.TokenIdent {
			index++
		}
		i++
	}
	return -1
}

func findDeclAssign(file syntax.File, start int, end int) int {
	parenDepth := 0
	bracketDepth := 0
	braceDepth := 0
	for i := start; i < end; i++ {
		if tokCharIs(&file, i, '(') {
			parenDepth++
		} else if tokCharIs(&file, i, ')') {
			if parenDepth > 0 {
				parenDepth--
			}
		} else if tokCharIs(&file, i, '[') {
			bracketDepth++
		} else if tokCharIs(&file, i, ']') {
			if bracketDepth > 0 {
				bracketDepth--
			}
		} else if tokCharIs(&file, i, '{') {
			braceDepth++
		} else if tokCharIs(&file, i, '}') {
			if braceDepth > 0 {
				braceDepth--
			}
		} else if parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 && tokenTextIs(&file, i, "=") {
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
	return tokCharIs(&file, tok, ';')
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

func buildDeclOrder(decls []DeclInfo) []int {
	order := make([]int, len(decls))
	for i := 0; i < len(decls); i++ {
		order[i] = i
	}
	for i := 1; i < len(order); i++ {
		item := order[i]
		j := i - 1
		for j >= 0 && declIndexAfter(decls, order[j], item) {
			order[j+1] = order[j]
			j--
		}
		order[j+1] = item
	}
	return order
}

func buildDeclDeps(info *PackageInfo) {
	for i := 0; i < len(info.Decls); i++ {
		decl := &info.Decls[i]
		for j := 0; j < len(decl.Refs); j++ {
			ref := decl.Refs[j]
			if ref.Kind != RefPackage {
				continue
			}
			dep := lookupDeclBySymbol(info.Decls, ref.Index)
			if dep < 0 || dep == i || intSliceHas(decl.Deps, dep) {
				continue
			}
			decl.Deps = append(decl.Deps, dep)
		}
	}
}

func buildInitOrder(decls []DeclInfo, sourceOrder []int) []int {
	var order []int
	state := make([]int, len(decls))
	for i := 0; i < len(sourceOrder); i++ {
		index := sourceOrder[i]
		if index >= 0 && index < len(decls) && decls[index].Kind == SymbolVar {
			order = appendInitVisit(order, decls, state, index)
		}
	}
	return order
}

func appendInitVisit(order []int, decls []DeclInfo, state []int, index int) []int {
	if index < 0 || index >= len(decls) || decls[index].Kind != SymbolVar {
		return order
	}
	if state[index] == 2 {
		return order
	}
	if state[index] == 1 {
		return order
	}
	state[index] = 1
	deps := decls[index].Deps
	for i := 0; i < len(deps); i++ {
		dep := deps[i]
		if dep >= 0 && dep < len(decls) && decls[dep].Kind == SymbolVar {
			order = appendInitVisit(order, decls, state, dep)
		}
	}
	state[index] = 2
	return append(order, index)
}

func declAfter(left DeclInfo, right DeclInfo) bool {
	if left.Name != right.Name {
		return checkStringAfter(left.Name, right.Name)
	}
	if left.File != right.File {
		return left.File > right.File
	}
	return left.Token > right.Token
}

func declIndexAfter(decls []DeclInfo, left int, right int) bool {
	leftDecl := decls[left]
	rightDecl := decls[right]
	if leftDecl.File != rightDecl.File {
		return leftDecl.File > rightDecl.File
	}
	if leftDecl.Token != rightDecl.Token {
		return leftDecl.Token > rightDecl.Token
	}
	if leftDecl.Kind != rightDecl.Kind {
		return leftDecl.Kind > rightDecl.Kind
	}
	return checkStringAfter(leftDecl.Name, rightDecl.Name)
}

func lookupDeclBySymbol(decls []DeclInfo, symbol int) int {
	for i := 0; i < len(decls); i++ {
		if decls[i].Symbol == symbol {
			return i
		}
	}
	return -1
}

func intSliceHas(values []int, value int) bool {
	for i := 0; i < len(values); i++ {
		if values[i] == value {
			return true
		}
	}
	return false
}
