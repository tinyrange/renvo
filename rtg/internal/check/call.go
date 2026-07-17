package check

import "j5.nz/rtg/rtg/internal/syntax"

const (
	CallUnknown = iota
	CallScope
	CallPackage
	CallImportSelector
	CallBuiltin
)

type CallRef struct {
	Kind        int
	Name        string
	BaseName    string
	CalleeToken int
	BaseToken   int
	DotToken    int
	ArgsStart   int
	ArgsEnd     int
	Args        []ExprSpan
	Ref         NameRef
	Selector    SelectorRef
	Package     int
	Symbol      int
}

func LookupCall(body FuncBody, base string, name string, kind int) int {
	for i := 0; i < len(body.Calls); i++ {
		call := body.Calls[i]
		if call.BaseName == base && call.Name == name && call.Kind == kind {
			return i
		}
	}
	return -1
}

func buildFuncCalls(file syntax.File, fileIndex int, info PackageInfo, checked []PackageInfo, body syntax.Body, scope FuncScope) []CallRef {
	var calls []CallRef
	for i := 0; i < len(body.Stmts); i++ {
		stmt := body.Stmts[i]
		if stmt.Kind == syntax.StmtAssign {
			calls = appendAssignCalls(calls, file, fileIndex, info, checked, scope, stmt)
		} else if stmt.Kind == syntax.StmtDecl {
			calls = appendDeclCalls(calls, file, fileIndex, info, checked, scope, stmt)
		} else if stmt.ExprStart >= 0 && stmt.ExprEnd > stmt.ExprStart {
			calls = appendExprCalls(calls, file, fileIndex, info, checked, scope, stmt.ExprStart, stmt.ExprEnd)
		}
	}
	return calls
}

func appendAssignCalls(calls []CallRef, file syntax.File, fileIndex int, info PackageInfo, checked []PackageInfo, scope FuncScope, stmt syntax.Stmt) []CallRef {
	assign := findTokenText(file, stmt.StartTok, stmt.EndTok, "=")
	shortAssign := findTokenText(file, stmt.StartTok, stmt.EndTok, ":=")
	if shortAssign >= 0 {
		return appendExprCalls(calls, file, fileIndex, info, checked, scope, shortAssign+1, stmt.EndTok)
	}
	if assign < 0 {
		return appendExprCalls(calls, file, fileIndex, info, checked, scope, stmt.StartTok, stmt.EndTok)
	}
	calls = appendExprCalls(calls, file, fileIndex, info, checked, scope, stmt.StartTok, assign)
	return appendExprCalls(calls, file, fileIndex, info, checked, scope, assign+1, stmt.EndTok)
}

func appendDeclCalls(calls []CallRef, file syntax.File, fileIndex int, info PackageInfo, checked []PackageInfo, scope FuncScope, stmt syntax.Stmt) []CallRef {
	start := stmt.StartTok + 1
	end := stmt.EndTok
	if start >= end {
		return calls
	}
	if tokCharIs(&file, start, '(') {
		i := start + 1
		for i < end {
			i = skipLocalSeparators(file, i, end)
			if i >= end || tokCharIs(&file, i, ')') {
				break
			}
			specEnd := statementSpecEnd(file, i, end)
			calls = appendSpecInitializerCalls(calls, file, fileIndex, info, checked, scope, i, specEnd)
			i = specEnd
		}
		return calls
	}
	return appendSpecInitializerCalls(calls, file, fileIndex, info, checked, scope, start, end)
}

func appendSpecInitializerCalls(calls []CallRef, file syntax.File, fileIndex int, info PackageInfo, checked []PackageInfo, scope FuncScope, start int, end int) []CallRef {
	assign := findTokenText(file, start, end, "=")
	if assign < 0 {
		return calls
	}
	return appendExprCalls(calls, file, fileIndex, info, checked, scope, assign+1, end)
}

func appendExprCalls(calls []CallRef, file syntax.File, fileIndex int, info PackageInfo, checked []PackageInfo, scope FuncScope, start int, end int) []CallRef {
	for i := start; i < end && i < len(file.Tokens); i++ {
		if !tokCharIs(&file, i, '(') {
			continue
		}
		closeTok := findTypeMatching(file, i, '(', ')')
		if closeTok <= i || closeTok > end+1 {
			continue
		}
		callee := i - 1
		if callee < start || file.Tokens[callee].Kind != syntax.TokenIdent {
			continue
		}
		if callee-1 >= start && tokenTextIs(&file, callee-1, ".") && callee-2 >= start && file.Tokens[callee-2].Kind == syntax.TokenIdent {
			calls = append(calls, resolveSelectorCall(file, fileIndex, info, checked, scope, tokenString(&file, callee-2), tokenString(&file, callee), callee-2, callee-1, callee, i, closeTok-1))
		} else {
			calls = append(calls, resolveDirectCall(file, fileIndex, info, scope, tokenString(&file, callee), callee, i, closeTok-1))
		}
	}
	return calls
}

func resolveDirectCall(file syntax.File, fileIndex int, info PackageInfo, scope FuncScope, name string, callee int, argsStart int, argsEnd int) CallRef {
	ref := resolveNameRef(fileIndex, info, scope, name, callee)
	call := CallRef{
		Kind:        CallUnknown,
		Name:        name,
		CalleeToken: callee,
		BaseToken:   -1,
		DotToken:    -1,
		ArgsStart:   argsStart + 1,
		ArgsEnd:     argsEnd,
		Args:        splitExprList(file, argsStart+1, argsEnd),
		Ref:         ref,
		Package:     -1,
		Symbol:      -1,
	}
	if ref.Kind == RefScope {
		call.Kind = CallScope
	} else if ref.Kind == RefPackage {
		call.Kind = CallPackage
		call.Package = ref.Package
		call.Symbol = ref.Index
	} else if ref.Kind == RefBuiltin {
		call.Kind = CallBuiltin
	}
	return call
}

func resolveSelectorCall(file syntax.File, fileIndex int, info PackageInfo, checked []PackageInfo, scope FuncScope, base string, name string, baseTok int, dotTok int, callee int, argsStart int, argsEnd int) CallRef {
	selector := resolveSelector(fileIndex, info, checked, scope, base, name, baseTok, dotTok, callee)
	call := CallRef{
		Kind:        CallUnknown,
		Name:        name,
		BaseName:    base,
		CalleeToken: callee,
		BaseToken:   baseTok,
		DotToken:    dotTok,
		ArgsStart:   argsStart + 1,
		ArgsEnd:     argsEnd,
		Args:        splitExprList(file, argsStart+1, argsEnd),
		Selector:    selector,
		Package:     -1,
		Symbol:      -1,
	}
	if selector.Kind == SelectorImport {
		call.Kind = CallImportSelector
		call.Package = selector.Package
		call.Symbol = selector.Symbol
	}
	return call
}
