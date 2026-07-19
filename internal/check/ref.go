package check

import "renvo.dev/internal/syntax"

const (
	RefUnknown = iota
	RefScope
	RefPackage
	RefImport
	RefBuiltin
	RefLabel
)

type NameRef struct {
	Name    string
	Kind    int
	Token   int
	Index   int
	Package int
}

func LookupBodyRef(body FuncBody, name string, kind int) int {
	for i := 0; i < len(body.Refs); i++ {
		if body.Refs[i].Name == name && body.Refs[i].Kind == kind {
			return i
		}
	}
	return -1
}

func buildFuncRefs(file syntax.File, fileIndex int, info PackageInfo, body syntax.Body, scope FuncScope) []NameRef {
	var refs []NameRef
	for i := 0; i < len(body.Stmts); i++ {
		stmt := body.Stmts[i]
		if stmt.Kind == syntax.StmtAssign {
			refs = appendAssignRefs(refs, file, fileIndex, info, scope, stmt)
		} else if stmt.Kind == syntax.StmtDecl {
			refs = appendDeclRefs(refs, file, fileIndex, info, scope, stmt)
		} else if stmt.Kind == syntax.StmtGoto || stmt.Kind == syntax.StmtBreak || stmt.Kind == syntax.StmtContinue {
			refs = appendBranchLabelRef(refs, file, scope, stmt)
		} else if stmt.ExprStart >= 0 && stmt.ExprEnd > stmt.ExprStart {
			refs = appendExprRefs(refs, file, fileIndex, info, scope, stmt.ExprStart, stmt.ExprEnd)
		}
	}
	return refs
}

func appendAssignRefs(refs []NameRef, file syntax.File, fileIndex int, info PackageInfo, scope FuncScope, stmt syntax.Stmt) []NameRef {
	assign := findTokenText(file, stmt.StartTok, stmt.EndTok, "=")
	shortAssign := findTokenText(file, stmt.StartTok, stmt.EndTok, ":=")
	if shortAssign >= 0 {
		return appendExprRefs(refs, file, fileIndex, info, scope, shortAssign+1, stmt.EndTok)
	}
	if assign < 0 {
		return appendExprRefs(refs, file, fileIndex, info, scope, stmt.StartTok, stmt.EndTok)
	}
	refs = appendExprRefs(refs, file, fileIndex, info, scope, stmt.StartTok, assign)
	return appendExprRefs(refs, file, fileIndex, info, scope, assign+1, stmt.EndTok)
}

func appendDeclRefs(refs []NameRef, file syntax.File, fileIndex int, info PackageInfo, scope FuncScope, stmt syntax.Stmt) []NameRef {
	start := stmt.StartTok + 1
	end := stmt.EndTok
	if start >= end {
		return refs
	}
	if tokCharIs(&file, start, '(') {
		i := start + 1
		for i < end {
			i = skipLocalSeparators(file, i, end)
			if i >= end || tokCharIs(&file, i, ')') {
				break
			}
			specEnd := statementSpecEnd(file, i, end)
			refs = appendSpecInitializerRefs(refs, file, fileIndex, info, scope, i, specEnd)
			i = specEnd
		}
		return refs
	}
	return appendSpecInitializerRefs(refs, file, fileIndex, info, scope, start, end)
}

func appendSpecInitializerRefs(refs []NameRef, file syntax.File, fileIndex int, info PackageInfo, scope FuncScope, start int, end int) []NameRef {
	assign := findTokenText(file, start, end, "=")
	if assign < 0 {
		return refs
	}
	return appendExprRefs(refs, file, fileIndex, info, scope, assign+1, end)
}

func appendBranchLabelRef(refs []NameRef, file syntax.File, scope FuncScope, stmt syntax.Stmt) []NameRef {
	tok := stmt.StartTok + 1
	if tok >= stmt.EndTok || tok >= len(file.Tokens) || file.Tokens[tok].KindLine&255 != syntax.TokenIdent {
		return refs
	}
	name := tokenString(&file, tok)
	index := lookupScopeNameKind(scope, name, NameLabel)
	kind := RefUnknown
	if index >= 0 {
		kind = RefLabel
	}
	return append(refs, NameRef{Name: name, Kind: kind, Token: tok, Index: index, Package: -1})
}

func appendExprRefs(refs []NameRef, file syntax.File, fileIndex int, info PackageInfo, scope FuncScope, start int, end int) []NameRef {
	for i := start; i < end && i < len(file.Tokens); i++ {
		if file.Tokens[i].KindLine&255 != syntax.TokenIdent || shouldSkipIdentRef(&file, i, end) {
			continue
		}
		name := tokenString(&file, i)
		if name == "_" {
			continue
		}
		refs = append(refs, resolveNameRef(fileIndex, info, scope, name, i))
	}
	return refs
}

func shouldSkipIdentRef(file *syntax.File, tok int, end int) bool {
	if tok > 0 {
		prev := file.Tokens[tok-1]
		if prev.KindLine&255 == syntax.TokenOperator && prev.End == prev.Start+1 && file.Src[prev.Start] == '.' {
			return true
		}
	}
	if tok+1 < end {
		next := file.Tokens[tok+1]
		if next.KindLine&255 == syntax.TokenOperator && next.End == next.Start+1 && file.Src[next.Start] == ':' {
			return true
		}
	}
	return false
}

func resolveNameRef(fileIndex int, info PackageInfo, scope FuncScope, name string, tok int) NameRef {
	ref := NameRef{Name: name, Kind: RefUnknown, Token: tok, Index: -1, Package: -1}
	scopeIndex := LookupScopeName(scope, name)
	if scopeIndex >= 0 && scope.Names[scopeIndex].Kind != NameLabel {
		ref.Kind = RefScope
		ref.Index = scopeIndex
		return ref
	}
	importIndex := LookupImport(info, fileIndex, name)
	if importIndex >= 0 && !info.Imports[importIndex].Blank && !info.Imports[importIndex].Dot {
		info.Imports[importIndex].Used = true
		ref.Kind = RefImport
		ref.Index = importIndex
		ref.Package = info.Imports[importIndex].Package
		return ref
	}
	symbolIndex := LookupPackageSymbol(info, name)
	if symbolIndex >= 0 {
		ref.Kind = RefPackage
		ref.Index = symbolIndex
		ref.Package = info.Symbols[symbolIndex].Package
		return ref
	}
	if isBuiltinName(name) {
		ref.Kind = RefBuiltin
		return ref
	}
	return ref
}

func lookupScopeNameKind(scope FuncScope, name string, kind int) int {
	for i := 0; i < len(scope.Names); i++ {
		if scope.Names[i].Name == name && scope.Names[i].Kind == kind {
			return i
		}
	}
	return -1
}

func isBuiltinName(name string) bool {
	if name == "append" || name == "bool" || name == "byte" || name == "cap" || name == "clear" {
		return true
	}
	if name == "close" || name == "complex" || name == "complex64" || name == "complex128" {
		return true
	}
	if name == "copy" || name == "delete" || name == "error" || name == "false" {
		return true
	}
	if name == "float32" || name == "float64" || name == "imag" || name == "int" {
		return true
	}
	if name == "int8" || name == "int16" || name == "int32" || name == "int64" {
		return true
	}
	if name == "iota" || name == "len" || name == "make" || name == "max" || name == "min" || name == "new" {
		return true
	}
	if name == "nil" || name == "panic" || name == "print" || name == "println" {
		return true
	}
	if name == "real" || name == "recover" || name == "rune" || name == "string" {
		return true
	}
	if name == "true" || name == "uint" || name == "uint8" || name == "uint16" {
		return true
	}
	return name == "uint32" || name == "uint64" || name == "uintptr"
}
