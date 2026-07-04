package check

import "j5.nz/rtg/rtg/internal/syntax"

const (
	SelectorUnknown = iota
	SelectorImport
)

type SelectorRef struct {
	Kind      int
	BaseName  string
	Name      string
	BaseToken int
	DotToken  int
	NameToken int
	BaseRef   NameRef
	Package   int
	Symbol    int
}

func LookupSelector(body FuncBody, base string, name string, kind int) int {
	for i := 0; i < len(body.Selectors); i++ {
		selector := body.Selectors[i]
		if selector.BaseName == base && selector.Name == name && selector.Kind == kind {
			return i
		}
	}
	return -1
}

func buildFuncSelectors(file syntax.File, fileIndex int, info PackageInfo, checked []PackageInfo, body syntax.Body, scope FuncScope) []SelectorRef {
	var selectors []SelectorRef
	for i := 0; i < len(body.Stmts); i++ {
		stmt := body.Stmts[i]
		if stmt.Kind == syntax.StmtAssign {
			selectors = appendAssignSelectors(selectors, file, fileIndex, info, checked, scope, stmt)
		} else if stmt.Kind == syntax.StmtDecl {
			selectors = appendDeclSelectors(selectors, file, fileIndex, info, checked, scope, stmt)
		} else if stmt.ExprStart >= 0 && stmt.ExprEnd > stmt.ExprStart {
			selectors = appendExprSelectors(selectors, file, fileIndex, info, checked, scope, stmt.ExprStart, stmt.ExprEnd)
		}
	}
	return selectors
}

func appendAssignSelectors(selectors []SelectorRef, file syntax.File, fileIndex int, info PackageInfo, checked []PackageInfo, scope FuncScope, stmt syntax.Stmt) []SelectorRef {
	assign := findTokenText(file, stmt.StartTok, stmt.EndTok, "=")
	shortAssign := findTokenText(file, stmt.StartTok, stmt.EndTok, ":=")
	if shortAssign >= 0 {
		return appendExprSelectors(selectors, file, fileIndex, info, checked, scope, shortAssign+1, stmt.EndTok)
	}
	if assign < 0 {
		return appendExprSelectors(selectors, file, fileIndex, info, checked, scope, stmt.StartTok, stmt.EndTok)
	}
	selectors = appendExprSelectors(selectors, file, fileIndex, info, checked, scope, stmt.StartTok, assign)
	return appendExprSelectors(selectors, file, fileIndex, info, checked, scope, assign+1, stmt.EndTok)
}

func appendDeclSelectors(selectors []SelectorRef, file syntax.File, fileIndex int, info PackageInfo, checked []PackageInfo, scope FuncScope, stmt syntax.Stmt) []SelectorRef {
	start := stmt.StartTok + 1
	end := stmt.EndTok
	if start >= end {
		return selectors
	}
	if tokCharIs(file, start, '(') {
		i := start + 1
		for i < end {
			i = skipLocalSeparators(file, i, end)
			if i >= end || tokCharIs(file, i, ')') {
				break
			}
			specEnd := statementSpecEnd(file, i, end)
			selectors = appendSpecInitializerSelectors(selectors, file, fileIndex, info, checked, scope, i, specEnd)
			i = specEnd
		}
		return selectors
	}
	return appendSpecInitializerSelectors(selectors, file, fileIndex, info, checked, scope, start, end)
}

func appendSpecInitializerSelectors(selectors []SelectorRef, file syntax.File, fileIndex int, info PackageInfo, checked []PackageInfo, scope FuncScope, start int, end int) []SelectorRef {
	assign := findTokenText(file, start, end, "=")
	if assign < 0 {
		return selectors
	}
	return appendExprSelectors(selectors, file, fileIndex, info, checked, scope, assign+1, end)
}

func appendExprSelectors(selectors []SelectorRef, file syntax.File, fileIndex int, info PackageInfo, checked []PackageInfo, scope FuncScope, start int, end int) []SelectorRef {
	for i := start + 1; i+1 < end && i+1 < len(file.Tokens); i++ {
		if !tokenTextIs(file, i, ".") {
			continue
		}
		if file.Tokens[i-1].Kind != syntax.TokenIdent || file.Tokens[i+1].Kind != syntax.TokenIdent {
			continue
		}
		baseName := tokenString(file, i-1)
		name := tokenString(file, i+1)
		if baseName == "_" || name == "_" {
			continue
		}
		selectors = append(selectors, resolveSelector(fileIndex, info, checked, scope, baseName, name, i-1, i, i+1))
	}
	return selectors
}

func resolveSelector(fileIndex int, info PackageInfo, checked []PackageInfo, scope FuncScope, baseName string, name string, baseTok int, dotTok int, nameTok int) SelectorRef {
	baseRef := resolveNameRef(fileIndex, info, scope, baseName, baseTok)
	selector := SelectorRef{
		Kind:      SelectorUnknown,
		BaseName:  baseName,
		Name:      name,
		BaseToken: baseTok,
		DotToken:  dotTok,
		NameToken: nameTok,
		BaseRef:   baseRef,
		Package:   -1,
		Symbol:    -1,
	}
	if baseRef.Kind == RefImport && baseRef.Package >= 0 {
		selector.Kind = SelectorImport
		selector.Package = baseRef.Package
		if baseRef.Package < len(checked) {
			selector.Symbol = LookupPackageSymbol(checked[baseRef.Package], name)
		}
	}
	return selector
}
