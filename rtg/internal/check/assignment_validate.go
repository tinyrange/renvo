package check

import (
	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtg/internal/syntax"
)

func invalidDefiniteAssignmentType(file syntax.File, fn syntax.FuncDecl) int {
	for i := fn.BodyStart + 2; i+1 < fn.BodyEnd; i++ {
		if !tokenTextIs(&file, i, "=") || file.Tokens[i-1].Kind != syntax.TokenIdent {
			continue
		}
		valueKind := definiteLiteralKind(file, i+1)
		if valueKind == "" {
			continue
		}
		name := tokenString(&file, i-1)
		for j := i - 2; j >= fn.BodyStart+1; j-- {
			if file.Tokens[j].Kind != syntax.TokenVar || j+2 >= i || file.Tokens[j+1].Kind != syntax.TokenIdent || tokenString(&file, j+1) != name {
				continue
			}
			declared := tokenString(&file, j+2)
			if definiteBuiltinType(declared) && declared != valueKind {
				return i + 1
			}
			break
		}
	}
	return -1
}

func invalidDefiniteInterfaceAssignment(pkg load.Package, info PackageInfo, fileIndex int, fn syntax.FuncDecl) int {
	file := pkg.Files[fileIndex].File
	for i := fn.BodyStart + 1; i < fn.BodyEnd; i++ {
		if file.Tokens[i].Kind != syntax.TokenVar || i+1 >= fn.BodyEnd || tokCharIs(&file, i+1, '(') {
			continue
		}
		end := statementSpecEnd(file, i+1, fn.BodyEnd)
		names, namesEnd := localDeclNameTokens(file, i+1, end)
		assign := findDeclAssign(file, namesEnd, end)
		if len(names) != 1 || assign < 0 {
			continue
		}
		typeStart, typeEnd := trimDeclSpan(file, namesEnd, assign)
		if typeEnd-typeStart != 1 || file.Tokens[typeStart].Kind != syntax.TokenIdent {
			continue
		}
		interfaceType := LookupType(info, tokenString(&file, typeStart))
		if interfaceType < 0 || info.Types[interfaceType].Kind != TypeInterface {
			continue
		}
		valueStart, valueEnd := trimExprSpan(file, assign+1, end)
		concreteName, pointer, ok := definiteCompositeType(file, valueStart, valueEnd)
		if !ok {
			continue
		}
		concreteType := LookupType(info, concreteName)
		if concreteType >= 0 && !definiteTypeImplementsInterface(info, concreteType, pointer, interfaceType) {
			return valueStart
		}
	}
	return -1
}

func definiteCompositeType(file syntax.File, start int, end int) (string, bool, bool) {
	pointer := false
	if start < end && tokCharIs(&file, start, '&') {
		pointer = true
		start++
	}
	if start+1 >= end || file.Tokens[start].Kind != syntax.TokenIdent || !tokCharIs(&file, start+1, '{') {
		return "", false, false
	}
	return tokenString(&file, start), pointer, true
}

func definiteTypeImplementsInterface(info PackageInfo, concreteType int, pointer bool, interfaceType int) bool {
	concrete := info.Types[concreteType]
	wanted := info.Types[interfaceType]
	for i := 0; i < len(wanted.InterfaceMethods); i++ {
		interfaceMethod := wanted.InterfaceMethods[i]
		found := false
		for j := 0; j < len(concrete.Methods); j++ {
			method := info.Methods[concrete.Methods[j]]
			if method.Name == interfaceMethod.Name && (pointer || !method.Pointer) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func excludedFileFeature(file syntax.File) (int, int) {
	for i := 0; i < len(file.Tokens); i++ {
		if file.Tokens[i].Kind == syntax.TokenSelect {
			return CheckErrSelect, i
		}
	}
	for i := 0; i < len(file.Tokens); i++ {
		kind := file.Tokens[i].Kind
		if kind == syntax.TokenGo {
			return CheckErrGoroutine, i
		}
		if kind == syntax.TokenChan || tokenTextIs(&file, i, "<-") {
			return CheckErrChannel, i
		}
	}
	return CheckOK, -1
}

func definiteLiteralKind(file syntax.File, tok int) string {
	if tok < 0 || tok >= len(file.Tokens) {
		return ""
	}
	if file.Tokens[tok].Kind == syntax.TokenString {
		return "string"
	}
	if file.Tokens[tok].Kind == syntax.TokenNumber {
		return "int"
	}
	if tokenTextIs(&file, tok, "true") || tokenTextIs(&file, tok, "false") {
		return "bool"
	}
	return ""
}

func definiteBuiltinType(name string) bool {
	return name == "int" || name == "string" || name == "bool"
}
