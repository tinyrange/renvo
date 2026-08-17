package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

type SourceLocation struct {
	Path  string
	Start int
	End   int
}

type NavigationResult struct {
	Definition SourceLocation
	References []SourceLocation
	Ok         bool
}

type navigationTarget struct {
	packageIndex int
	symbolIndex  int
	fileIndex    int
	bodyIndex    int
	scopeIndex   int
	local        bool
	member       bool
	memberType   string
	memberName   string
	field        bool
}

// NavigateProgram resolves the identifier at a byte offset and returns its
// declaration and semantically equivalent uses. Package selectors are followed
// across imports; locals are compared through the same scope used by checking.
func NavigateProgram(graph load.Graph, program Program, path string, offset int) NavigationResult {
	pkgIndex, fileIndex := completionFile(graph, path)
	if pkgIndex < 0 || fileIndex < 0 || pkgIndex >= len(program.Packages) {
		return NavigationResult{}
	}
	file := graph.Packages[pkgIndex].Files[fileIndex].File
	token := navigationToken(file, offset)
	if token < 0 {
		return NavigationResult{}
	}
	target, ok := navigationResolve(graph, program, pkgIndex, fileIndex, token)
	if !ok {
		if imported := navigationImportedPackage(graph, program, pkgIndex, fileIndex, token); imported.Ok {
			return imported
		}
		return NavigationResult{}
	}
	if target.local {
		return navigationLocal(graph, program, target)
	}
	if target.member {
		return navigationMember(graph, program, target)
	}
	return navigationPackage(graph, program, target.packageIndex, target.symbolIndex)
}

func navigationImportedPackage(graph load.Graph, program Program, pkgIndex, fileIndex, token int) NavigationResult {
	if pkgIndex < 0 || pkgIndex >= len(program.Packages) || pkgIndex >= len(graph.Packages) ||
		fileIndex < 0 || fileIndex >= len(graph.Packages[pkgIndex].Files) {
		return NavigationResult{}
	}
	file := graph.Packages[pkgIndex].Files[fileIndex].File
	if fn, ok := completionFunctionAt(file, file.Tokens[token].Start); ok {
		if scope, scopeOK, _ := buildFuncScopeCore(file, fn); scopeOK && lookupScopeTokenNameCore(scope, &file, token) >= 0 {
			return NavigationResult{}
		}
	}
	name := tokenString(&file, token)
	imported := completionImportPackage(program.Packages[pkgIndex], fileIndex, name)
	if imported < 0 || imported >= len(graph.Packages) {
		return NavigationResult{}
	}
	var definition SourceLocation
	definitionOK := false
	for i := 0; i < len(graph.Packages[imported].Files); i++ {
		packageFile := graph.Packages[imported].Files[i].File
		if packageFile.PackageName >= 0 {
			definition, definitionOK = navigationLocation(graph, imported, i, packageFile.PackageName)
			if definitionOK {
				break
			}
		}
	}
	if !definitionOK {
		return NavigationResult{}
	}
	result := NavigationResult{Definition: definition, Ok: true}
	navigationAppend(&result.References, definition)
	for i := 0; i < len(file.Tokens); i++ {
		if file.Tokens[i].KindLine&255 != syntax.TokenIdent || tokenString(&file, i) != name {
			continue
		}
		isAlias := false
		for j := 0; j < len(file.Imports); j++ {
			if file.Imports[j].NameTok == i {
				isAlias = true
				break
			}
		}
		if !isAlias && (i+1 >= len(file.Tokens) || !tokenTextIs(&file, i+1, ".")) {
			continue
		}
		if location, valid := navigationLocation(graph, pkgIndex, fileIndex, i); valid {
			navigationAppend(&result.References, location)
		}
	}
	return result
}

func navigationToken(file syntax.File, offset int) int {
	previous := -1
	for i := 0; i < len(file.Tokens); i++ {
		tok := file.Tokens[i]
		if tok.KindLine&255 != syntax.TokenIdent {
			continue
		}
		if tok.Start <= offset && offset < tok.End {
			return i
		}
		if tok.End == offset {
			previous = i
		}
		if tok.Start > offset {
			break
		}
	}
	return previous
}

func navigationResolve(graph load.Graph, program Program, pkgIndex int, fileIndex int, token int) (navigationTarget, bool) {
	info := program.Packages[pkgIndex]
	for i := 0; i < len(info.Symbols); i++ {
		if info.Symbols[i].File == fileIndex && info.Symbols[i].Token == token {
			return navigationTarget{packageIndex: pkgIndex, symbolIndex: i}, true
		}
	}
	for i := 0; i < len(info.Decls); i++ {
		decl := info.Decls[i]
		if decl.File != fileIndex {
			continue
		}
		if target, ok := navigationRefs(decl.CoreRefs, decl.CoreSelectors, pkgIndex, token); ok {
			return target, true
		}
	}
	if target, ok := navigationTypeRefs(info.CoreTypeRefs, fileIndex, token); ok {
		return target, true
	}
	for i := 0; i < len(info.CoreBodies); i++ {
		body := info.CoreBodies[i]
		if body.File != fileIndex {
			continue
		}
		if target, ok := navigationRefs(body.CoreRefs, body.CoreSelectors, pkgIndex, token); ok {
			return target, true
		}
		if target, ok := navigationTypeRefs(body.CoreTypeRefs, fileIndex, token); ok {
			return target, true
		}
		if target, ok := navigationMemberAt(graph, program, pkgIndex, fileIndex, token); ok {
			return target, true
		}
		if body.Func < 0 || body.Func >= len(graph.Packages[pkgIndex].Files[fileIndex].File.Funcs) {
			continue
		}
		file := graph.Packages[pkgIndex].Files[fileIndex].File
		fn := file.Funcs[body.Func]
		if token < fn.StartTok || token >= fn.EndTok {
			continue
		}
		scope, scopeOK, _ := buildFuncScopeCore(file, fn)
		if !scopeOK {
			continue
		}
		index := lookupScopeTokenNameCore(scope, &file, token)
		if index >= 0 {
			return navigationTarget{packageIndex: pkgIndex, fileIndex: fileIndex, bodyIndex: i, scopeIndex: index, local: true}, true
		}
	}
	return navigationTarget{}, false
}

func navigationMemberAt(graph load.Graph, program Program, pkgIndex int, fileIndex int, token int) (navigationTarget, bool) {
	file := graph.Packages[pkgIndex].Files[fileIndex].File
	if token < 2 || !tokenTextIs(&file, token-1, ".") {
		return navigationTarget{}, false
	}
	components := completionSelectorComponents(file.Src, file.Tokens[token-1].Start)
	if len(components) == 0 {
		return navigationTarget{}, false
	}
	var typ completionType
	var ok bool
	start := 1
	if imported := completionImportPackage(program.Packages[pkgIndex], fileIndex, components[0]); imported >= 0 {
		if len(components) < 2 {
			return navigationTarget{}, false
		}
		typ, ok = completionPackageNameType(graph, program, imported, components[1])
		start = 2
	} else {
		fn, found := completionFunctionAt(file, file.Tokens[token].Start)
		if !found {
			return navigationTarget{}, false
		}
		typ, ok = completionNameType(graph, program, pkgIndex, fileIndex, file, fn, components[0], file.Tokens[token].Start)
		if !ok {
			typ, ok = navigationShortAssignType(graph, program, pkgIndex, fileIndex, file, fn, components[0], file.Tokens[token].Start)
		}
	}
	if !ok {
		return navigationTarget{}, false
	}
	for i := start; i < len(components); i++ {
		typ, ok = completionFieldType(graph, program, typ, components[i])
		if !ok {
			return navigationTarget{}, false
		}
	}
	if typ.Package < 0 || typ.Package >= len(program.Packages) {
		return navigationTarget{}, false
	}
	name := tokenString(&file, token)
	return navigationFindMember(graph, program, typ, name, 0)
}

func navigationShortAssignType(graph load.Graph, program Program, pkgIndex int, fileIndex int, file syntax.File, fn syntax.FuncDecl, name string, offset int) (completionType, bool) {
	for i := fn.BodyStart + 1; i < fn.BodyEnd && i < len(file.Tokens); i++ {
		if file.Tokens[i].Start >= offset || file.Tokens[i].KindLine&255 != syntax.TokenIdent || tokenString(&file, i) != name {
			continue
		}
		assign := completionFindShortAssign(file, i, fn.BodyEnd)
		start := assign + 1
		if assign < 0 || start+3 >= fn.BodyEnd || file.Tokens[start].KindLine&255 != syntax.TokenIdent ||
			!tokenTextIs(&file, start+1, ".") || file.Tokens[start+2].KindLine&255 != syntax.TokenIdent || !tokenTextIs(&file, start+3, "(") {
			continue
		}
		owner := completionImportPackage(program.Packages[pkgIndex], fileIndex, tokenString(&file, start))
		if owner < 0 || owner >= len(program.Packages) || owner >= len(graph.Packages) {
			continue
		}
		functionName := tokenString(&file, start+2)
		info := program.Packages[owner]
		for symbolIndex := 0; symbolIndex < len(info.Symbols); symbolIndex++ {
			symbol := info.Symbols[symbolIndex]
			if symbol.Kind != SymbolFunc || symbol.Name != functionName || symbol.File < 0 || symbol.File >= len(graph.Packages[owner].Files) {
				continue
			}
			functionFile := graph.Packages[owner].Files[symbol.File].File
			for functionIndex := 0; functionIndex < len(functionFile.Funcs); functionIndex++ {
				function := functionFile.Funcs[functionIndex]
				if function.NameTok != symbol.Token {
					continue
				}
				signature := buildFuncSignature(functionFile, function)
				if len(signature.Results) > 0 {
					return completionSpanType(graph, program, owner, symbol.File, signature.Results[0].TypeStart, signature.Results[0].TypeEnd)
				}
			}
		}
	}
	return completionType{}, false
}

func navigationFindMember(graph load.Graph, program Program, typ completionType, name string, depth int) (navigationTarget, bool) {
	if depth > 5 || typ.Package < 0 || typ.Package >= len(program.Packages) {
		return navigationTarget{}, false
	}
	info := program.Packages[typ.Package]
	if symbol := LookupPackageSymbol(info, typ.Name+"."+name); symbol >= 0 {
		return navigationTarget{packageIndex: typ.Package, symbolIndex: symbol, member: true, memberType: typ.Name, memberName: name}, true
	}
	typeIndex := LookupType(info, typ.Name)
	if typeIndex < 0 || typeIndex >= len(info.Types) {
		return navigationTarget{}, false
	}
	typeInfo := info.Types[typeIndex]
	if field := LookupField(typeInfo.Fields, name); field >= 0 {
		return navigationTarget{packageIndex: typ.Package, symbolIndex: typeIndex, member: true, memberType: typ.Name, memberName: name, field: true}, true
	}
	for i := 0; i < len(typeInfo.Fields); i++ {
		if typeInfo.Fields[i].Name != "" {
			continue
		}
		embedded, ok := completionSpanType(graph, program, typ.Package, typeInfo.File, typeInfo.Fields[i].TypeStart, typeInfo.Fields[i].TypeEnd)
		if ok {
			if target, found := navigationFindMember(graph, program, embedded, name, depth+1); found {
				return target, true
			}
		}
	}
	if typeInfo.Kind == TypeNamed || typeInfo.Kind == TypePointer {
		base, ok := completionSpanType(graph, program, typ.Package, typeInfo.File, typeInfo.TypeStart, typeInfo.TypeEnd)
		if ok && (base.Package != typ.Package || base.Name != typ.Name) {
			return navigationFindMember(graph, program, base, name, depth+1)
		}
	}
	return navigationTarget{}, false
}

func navigationMember(graph load.Graph, program Program, target navigationTarget) NavigationResult {
	if target.packageIndex < 0 || target.packageIndex >= len(program.Packages) {
		return NavigationResult{}
	}
	info := program.Packages[target.packageIndex]
	var definition SourceLocation
	var ok bool
	if target.field {
		if target.symbolIndex < 0 || target.symbolIndex >= len(info.Types) {
			return NavigationResult{}
		}
		typ := info.Types[target.symbolIndex]
		field := LookupField(typ.Fields, target.memberName)
		if field >= 0 {
			definition, ok = navigationLocation(graph, target.packageIndex, typ.File, typ.Fields[field].NameTok)
		}
	} else if target.symbolIndex >= 0 && target.symbolIndex < len(info.Symbols) {
		symbol := info.Symbols[target.symbolIndex]
		definition, ok = navigationLocation(graph, target.packageIndex, symbol.File, symbol.Token)
	}
	if !ok {
		return NavigationResult{}
	}
	result := NavigationResult{Definition: definition, Ok: true}
	navigationAppend(&result.References, definition)
	for pkg := 0; pkg < len(graph.Packages) && pkg < len(program.Packages); pkg++ {
		for fileIndex := 0; fileIndex < len(graph.Packages[pkg].Files); fileIndex++ {
			file := graph.Packages[pkg].Files[fileIndex].File
			for token := 2; token < len(file.Tokens); token++ {
				if file.Tokens[token].KindLine&255 != syntax.TokenIdent || !tokenTextIs(&file, token-1, ".") {
					continue
				}
				candidate, resolved := navigationMemberAt(graph, program, pkg, fileIndex, token)
				if !resolved || candidate.packageIndex != target.packageIndex || candidate.memberType != target.memberType || candidate.memberName != target.memberName || candidate.field != target.field {
					continue
				}
				if location, valid := navigationLocation(graph, pkg, fileIndex, token); valid {
					navigationAppend(&result.References, location)
				}
			}
		}
	}
	return result
}

func navigationRefs(refs []CoreNameRef, selectors []CoreSelectorRef, ownPackage int, token int) (navigationTarget, bool) {
	for i := 0; i < len(refs); i++ {
		if refs[i].Token == token {
			return navigationTarget{packageIndex: ownPackage, symbolIndex: refs[i].Index}, true
		}
	}
	for i := 0; i < len(selectors); i++ {
		if selectors[i].NameTok == token && selectors[i].BasePackage >= 0 && selectors[i].Symbol >= 0 {
			return navigationTarget{packageIndex: selectors[i].BasePackage, symbolIndex: selectors[i].Symbol}, true
		}
	}
	return navigationTarget{}, false
}

func navigationTypeRefs(refs []CoreTypeRef, file int, token int) (navigationTarget, bool) {
	for i := 0; i < len(refs); i++ {
		if refs[i].File == file && refs[i].Token == token && refs[i].Package >= 0 && refs[i].Symbol >= 0 {
			return navigationTarget{packageIndex: refs[i].Package, symbolIndex: refs[i].Symbol}, true
		}
	}
	return navigationTarget{}, false
}

func navigationPackage(graph load.Graph, program Program, packageIndex int, symbolIndex int) NavigationResult {
	if packageIndex < 0 || packageIndex >= len(program.Packages) || packageIndex >= len(graph.Packages) {
		return NavigationResult{}
	}
	info := program.Packages[packageIndex]
	if symbolIndex < 0 || symbolIndex >= len(info.Symbols) {
		return NavigationResult{}
	}
	symbol := info.Symbols[symbolIndex]
	definition, ok := navigationLocation(graph, packageIndex, symbol.File, symbol.Token)
	if !ok {
		return NavigationResult{}
	}
	result := NavigationResult{Definition: definition, Ok: true}
	navigationAppend(&result.References, definition)
	for pkg := 0; pkg < len(program.Packages) && pkg < len(graph.Packages); pkg++ {
		candidate := program.Packages[pkg]
		for i := 0; i < len(candidate.Decls); i++ {
			decl := candidate.Decls[i]
			navigationAppendResolved(graph, &result.References, pkg, decl.File, decl.CoreRefs, decl.CoreSelectors, packageIndex, symbolIndex)
		}
		navigationAppendTypeRefs(graph, &result.References, pkg, candidate.CoreTypeRefs, packageIndex, symbolIndex)
		for i := 0; i < len(candidate.CoreBodies); i++ {
			body := candidate.CoreBodies[i]
			navigationAppendResolved(graph, &result.References, pkg, body.File, body.CoreRefs, body.CoreSelectors, packageIndex, symbolIndex)
			navigationAppendTypeRefs(graph, &result.References, pkg, body.CoreTypeRefs, packageIndex, symbolIndex)
		}
	}
	return result
}

func navigationAppendResolved(graph load.Graph, locations *[]SourceLocation, ownPackage int, file int, refs []CoreNameRef, selectors []CoreSelectorRef, targetPackage int, targetSymbol int) {
	for i := 0; i < len(refs); i++ {
		if ownPackage == targetPackage && refs[i].Index == targetSymbol {
			if location, ok := navigationLocation(graph, ownPackage, file, refs[i].Token); ok {
				navigationAppend(locations, location)
			}
		}
	}
	for i := 0; i < len(selectors); i++ {
		if selectors[i].BasePackage == targetPackage && selectors[i].Symbol == targetSymbol {
			if location, ok := navigationLocation(graph, ownPackage, file, selectors[i].NameTok); ok {
				navigationAppend(locations, location)
			}
		}
	}
}

func navigationAppendTypeRefs(graph load.Graph, locations *[]SourceLocation, ownPackage int, refs []CoreTypeRef, targetPackage int, targetSymbol int) {
	for i := 0; i < len(refs); i++ {
		if refs[i].Package == targetPackage && refs[i].Symbol == targetSymbol {
			if location, ok := navigationLocation(graph, ownPackage, refs[i].File, refs[i].Token); ok {
				navigationAppend(locations, location)
			}
		}
	}
}

func navigationLocal(graph load.Graph, program Program, target navigationTarget) NavigationResult {
	info := program.Packages[target.packageIndex]
	if target.bodyIndex < 0 || target.bodyIndex >= len(info.CoreBodies) {
		return NavigationResult{}
	}
	body := info.CoreBodies[target.bodyIndex]
	file := graph.Packages[target.packageIndex].Files[target.fileIndex].File
	if body.Func < 0 || body.Func >= len(file.Funcs) {
		return NavigationResult{}
	}
	fn := file.Funcs[body.Func]
	scope, ok, _ := buildFuncScopeCore(file, fn)
	if !ok || target.scopeIndex < 0 || target.scopeIndex >= len(scope.Names) {
		return NavigationResult{}
	}
	declaration := scope.Names[target.scopeIndex].Token
	definition, ok := navigationLocation(graph, target.packageIndex, target.fileIndex, declaration)
	if !ok {
		return NavigationResult{}
	}
	result := NavigationResult{Definition: definition, Ok: true}
	for token := fn.StartTok; token < fn.EndTok && token < len(file.Tokens); token++ {
		if file.Tokens[token].KindLine&255 != syntax.TokenIdent {
			continue
		}
		if lookupScopeTokenNameCore(scope, &file, token) == target.scopeIndex {
			if location, valid := navigationLocation(graph, target.packageIndex, target.fileIndex, token); valid {
				navigationAppend(&result.References, location)
			}
		}
	}
	return result
}

func navigationLocation(graph load.Graph, pkg int, file int, token int) (SourceLocation, bool) {
	if pkg < 0 || pkg >= len(graph.Packages) || file < 0 || file >= len(graph.Packages[pkg].Files) {
		return SourceLocation{}, false
	}
	source := graph.Packages[pkg].Files[file]
	if token < 0 || token >= len(source.File.Tokens) {
		return SourceLocation{}, false
	}
	tok := source.File.Tokens[token]
	return SourceLocation{Path: load.CleanPath(source.Path), Start: tok.Start, End: tok.End}, true
}

func navigationAppend(locations *[]SourceLocation, location SourceLocation) {
	for i := 0; i < len(*locations); i++ {
		if (*locations)[i] == location {
			return
		}
	}
	*locations = append(*locations, location)
}
