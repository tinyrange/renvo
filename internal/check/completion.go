package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

const (
	CompletionVariable = iota + 1
	CompletionField
	CompletionMethod
	CompletionFunction
	CompletionType
	CompletionPackage
	CompletionKeyword
)

type CompletionItem struct {
	Name          string
	Detail        string
	Kind          int
	Signature     string
	Documentation string
	Parameters    []CompletionParameter
}

type CompletionParameter struct {
	Name string
	Type string
}

type SignatureHelp struct {
	Ok              bool
	Label           string
	Parameters      []CompletionParameter
	ActiveParameter int
}

type completionType struct {
	Package int
	Name    string
	Pointer bool
}

// CompleteKeywords returns Go keywords matching the identifier at offset.
// Language-service hosts use this small syntax-independent fallback while an
// editor buffer is temporarily too incomplete to build a package graph.
func CompleteKeywords(source []byte, offset int) []CompletionItem {
	if offset < 0 {
		offset = 0
	}
	if offset > len(source) {
		offset = len(source)
	}
	start := completionIdentifierStart(source, offset)
	return completionKeywordItems(nil, string(source[start:offset]))
}

// CompleteGraph returns semantic names visible at a source offset. It is
// deliberately a query over the frontend graph: editor widgets remain unaware
// of Go packages and the same answer is available in host and self-hosted IDEs.
func CompleteGraph(graph load.Graph, path string, offset int) []CompletionItem {
	return CompleteProgram(graph, CheckGraphBestEffort(graph), path, offset)
}

// CheckGraphBestEffort builds every package independently and retains the
// semantic information produced before a diagnostic. Editor features use it
// while a buffer is temporarily incomplete.
func CheckGraphBestEffort(graph load.Graph) Program {
	return completionProgram(graph)
}

// CompleteProgram queries an existing checked snapshot. Interactive callers
// use this path so completion, diagnostics, and signature help share one check.
func CompleteProgram(graph load.Graph, prog Program, path string, offset int) []CompletionItem {
	pkgIndex, fileIndex := completionFile(graph, path)
	if pkgIndex < 0 || fileIndex < 0 {
		return nil
	}
	file := graph.Packages[pkgIndex].Files[fileIndex].File
	if offset < 0 {
		offset = 0
	}
	if offset > len(file.Src) {
		offset = len(file.Src)
	}
	prefixStart := completionIdentifierStart(file.Src, offset)
	prefix := string(file.Src[prefixStart:offset])
	if pkgIndex >= len(prog.Packages) {
		return nil
	}
	var items []CompletionItem
	if prefixStart > 0 && file.Src[prefixStart-1] == '.' {
		components := completionSelectorComponents(file.Src, prefixStart-1)
		items = completionSelectorItems(items, graph, prog, pkgIndex, fileIndex, file, offset, components, prefix)
	} else {
		items = completionScopeItems(items, graph, prog, pkgIndex, fileIndex, file, offset, prefix)
	}
	completionSort(items, prefix)
	return items
}

func completionProgram(graph load.Graph) Program {
	prog := CheckGraphHeadersCore(graph)
	if len(prog.Packages) == 0 {
		return prog
	}
	for i := 0; i < len(graph.Packages) && i < len(prog.Packages); i++ {
		attempt := prog
		attempt.Ok = true
		attempt.Error = CheckOK
		attempt.ErrorPackage = -1
		attempt.ErrorFile = -1
		attempt.ErrorToken = -1
		checked := CheckGraphPackageCore(graph, attempt, i)
		if i < len(checked.Packages) {
			prog.Packages[i] = checked.Packages[i]
		}
	}
	return prog
}

func completionFile(graph load.Graph, path string) (int, int) {
	path = load.CleanPath(path)
	for p := 0; p < len(graph.Packages); p++ {
		for f := 0; f < len(graph.Packages[p].Files); f++ {
			if load.CleanPath(graph.Packages[p].Files[f].Path) == path {
				return p, f
			}
		}
	}
	return -1, -1
}

func completionScopeItems(items []CompletionItem, graph load.Graph, prog Program, pkgIndex, fileIndex int, file syntax.File, offset int, prefix string) []CompletionItem {
	fn, hasFunc := completionFunctionAt(file, offset)
	if hasFunc {
		scope, _, _ := buildFuncScopeCore(file, fn)
		for i := 0; i < len(scope.Names); i++ {
			name := tokenString(&file, scope.Names[i].Token)
			tok := file.Tokens[scope.Names[i].Token]
			if scope.Names[i].Kind != NameLabel && tok.Start < offset {
				items = completionAdd(items, name, completionScopeDetail(scope.Names[i].Kind), CompletionVariable, prefix)
			}
		}
	}
	info := prog.Packages[pkgIndex]
	for i := 0; i < len(info.Imports); i++ {
		imp := info.Imports[i]
		if imp.File == fileIndex && !imp.Blank && !imp.Dot {
			items = completionAdd(items, imp.Name, imp.ImportPath, CompletionPackage, prefix)
		}
	}
	for i := 0; i < len(info.Symbols); i++ {
		symbol := info.Symbols[i]
		if symbol.Kind == SymbolMethod || completionHasByte(symbol.Name, '.') {
			continue
		}
		kind := CompletionVariable
		detail := "package variable"
		if symbol.Kind == SymbolFunc {
			kind, detail = CompletionFunction, "function"
		} else if symbol.Kind == SymbolType {
			kind, detail = CompletionType, "type"
		} else if symbol.Kind == SymbolConst {
			detail = "constant"
		}
		if symbol.Kind == SymbolFunc {
			items = completionAddSymbol(items, graph, pkgIndex, symbol, symbol.Name, prefix)
		} else {
			items = completionAdd(items, symbol.Name, detail, kind, prefix)
		}
	}
	builtins := []string{"append", "cap", "clear", "close", "complex", "copy", "delete", "imag", "len", "make", "max", "min", "new", "panic", "print", "println", "real", "recover"}
	for i := 0; i < len(builtins); i++ {
		items = completionAdd(items, builtins[i], "builtin", CompletionFunction, prefix)
	}
	return completionKeywordItems(items, prefix)
}

func completionKeywordItems(items []CompletionItem, prefix string) []CompletionItem {
	keywords := []string{"break", "case", "const", "continue", "defer", "else", "fallthrough", "for", "func", "go", "goto", "if", "import", "interface", "map", "package", "range", "return", "struct", "switch", "type", "var"}
	for i := 0; i < len(keywords); i++ {
		items = completionAdd(items, keywords[i], "keyword", CompletionKeyword, prefix)
	}
	return items
}

func completionSelectorItems(items []CompletionItem, graph load.Graph, prog Program, pkgIndex, fileIndex int, file syntax.File, offset int, components []string, prefix string) []CompletionItem {
	if len(components) == 0 {
		return items
	}
	info := prog.Packages[pkgIndex]
	if imported := completionImportPackage(info, fileIndex, components[0]); imported >= 0 {
		if len(components) == 1 {
			return completionPackageItems(items, prog, imported, prefix)
		}
		typ, ok := completionPackageNameType(graph, prog, imported, components[1])
		if !ok {
			return items
		}
		for i := 2; i < len(components); i++ {
			typ, ok = completionFieldType(graph, prog, typ, components[i])
			if !ok {
				return items
			}
		}
		return completionTypeItems(items, graph, prog, typ, pkgIndex, prefix, 0)
	}
	fn, ok := completionFunctionAt(file, offset)
	if !ok {
		return items
	}
	typ, ok := completionNameType(graph, prog, pkgIndex, fileIndex, file, fn, components[0], offset)
	if !ok {
		return items
	}
	for i := 1; i < len(components); i++ {
		typ, ok = completionFieldType(graph, prog, typ, components[i])
		if !ok {
			return items
		}
	}
	return completionTypeItems(items, graph, prog, typ, pkgIndex, prefix, 0)
}

func completionPackageNameType(graph load.Graph, prog Program, pkg int, name string) (completionType, bool) {
	if pkg < 0 || pkg >= len(prog.Packages) || pkg >= len(graph.Packages) {
		return completionType{}, false
	}
	info := prog.Packages[pkg]
	for i := 0; i < len(info.Decls); i++ {
		decl := info.Decls[i]
		if decl.Name != name || decl.File < 0 || decl.File >= len(graph.Packages[pkg].Files) {
			continue
		}
		if decl.TypeStart >= 0 && decl.TypeEnd > decl.TypeStart {
			return completionSpanType(graph, prog, pkg, decl.File, decl.TypeStart, decl.TypeEnd)
		}
		if decl.ValueIndex >= 0 && decl.ValueIndex < len(decl.Values) {
			span := decl.Values[decl.ValueIndex]
			file := graph.Packages[pkg].Files[decl.File].File
			return completionExpressionType(graph, prog, pkg, decl.File, file, span.StartTok, span.EndTok, 0)
		}
		if decl.ValueStart >= 0 && decl.ValueEnd > decl.ValueStart {
			file := graph.Packages[pkg].Files[decl.File].File
			return completionExpressionType(graph, prog, pkg, decl.File, file, decl.ValueStart, decl.ValueEnd, decl.ValueIndex)
		}
	}
	return completionType{}, false
}

func completionPackageItems(items []CompletionItem, prog Program, pkg int, prefix string) []CompletionItem {
	if pkg < 0 || pkg >= len(prog.Packages) {
		return items
	}
	for i := 0; i < len(prog.Packages[pkg].Symbols); i++ {
		symbol := prog.Packages[pkg].Symbols[i]
		if symbol.Kind == SymbolMethod || !completionExported(symbol.Name) {
			continue
		}
		kind, detail := CompletionVariable, "package variable"
		if symbol.Kind == SymbolFunc {
			kind, detail = CompletionFunction, "function"
		} else if symbol.Kind == SymbolType {
			kind, detail = CompletionType, "type"
		} else if symbol.Kind == SymbolConst {
			detail = "constant"
		}
		if symbol.Kind == SymbolFunc {
			items = completionAddSymbol(items, prog.Graph, pkg, symbol, symbol.Name, prefix)
		} else {
			items = completionAdd(items, symbol.Name, detail, kind, prefix)
		}
	}
	return items
}

func completionTypeItems(items []CompletionItem, graph load.Graph, prog Program, typ completionType, origin int, prefix string, depth int) []CompletionItem {
	if depth > 5 || typ.Package < 0 || typ.Package >= len(prog.Packages) || typ.Package >= len(graph.Packages) {
		return items
	}
	info := prog.Packages[typ.Package]
	typeIndex := LookupType(info, typ.Name)
	if typeIndex < 0 || typeIndex >= len(info.Types) {
		return items
	}
	typeInfo := info.Types[typeIndex]
	for i := 0; i < len(typeInfo.Fields); i++ {
		field := typeInfo.Fields[i]
		if field.Name == "" {
			fieldType, ok := completionSpanType(graph, prog, typ.Package, typeInfo.File, field.TypeStart, field.TypeEnd)
			if ok {
				items = completionTypeItems(items, graph, prog, fieldType, origin, prefix, depth+1)
			}
			continue
		}
		if typ.Package == origin || completionExported(field.Name) {
			items = completionAdd(items, field.Name, "field", CompletionField, prefix)
		}
	}
	prefixName := typ.Name + "."
	for i := 0; i < len(info.Symbols); i++ {
		symbol := info.Symbols[i]
		name := symbol.Name
		if symbol.Kind == SymbolMethod && completionStartsWith(name, prefixName) {
			method := name[len(prefixName):]
			if typ.Package == origin || completionExported(method) {
				items = completionAddSymbol(items, graph, typ.Package, symbol, method, prefix)
			}
		}
	}
	if typeInfo.Kind == TypeNamed || typeInfo.Kind == TypePointer {
		base, ok := completionSpanType(graph, prog, typ.Package, typeInfo.File, typeInfo.TypeStart, typeInfo.TypeEnd)
		if ok && (base.Package != typ.Package || base.Name != typ.Name) {
			items = completionTypeItems(items, graph, prog, base, origin, prefix, depth+1)
		}
	}
	return items
}

func completionNameType(graph load.Graph, prog Program, pkgIndex, fileIndex int, file syntax.File, fn syntax.FuncDecl, name string, offset int) (completionType, bool) {
	signature := buildFuncSignature(file, fn)
	groups := [][]Field{signature.Receiver, signature.Params, signature.Results}
	for i := 0; i < len(groups); i++ {
		for j := 0; j < len(groups[i]); j++ {
			field := groups[i][j]
			if field.Name == name {
				return completionSpanType(graph, prog, pkgIndex, fileIndex, field.TypeStart, field.TypeEnd)
			}
		}
	}
	for i := fn.BodyStart + 1; i < fn.BodyEnd && i < len(file.Tokens); i++ {
		tok := file.Tokens[i]
		if tok.Start >= offset || tok.KindLine&255 != syntax.TokenIdent || tokenString(&file, i) != name {
			continue
		}
		if i > 0 && file.Tokens[i-1].KindLine&255 == syntax.TokenVar {
			end := completionStatementEnd(file, i+1, fn.BodyEnd)
			value := findDeclAssign(file, i+1, end)
			typeEnd := end
			if value >= 0 {
				typeEnd = value
			}
			if typ, ok := completionSpanType(graph, prog, pkgIndex, fileIndex, i+1, typeEnd); ok {
				return typ, true
			}
			if value >= 0 {
				return completionExpressionType(graph, prog, pkgIndex, fileIndex, file, value+1, end, completionShortAssignValueIndex(file, i, value))
			}
		}
		assign := completionFindShortAssign(file, i, fn.BodyEnd)
		if assign >= 0 {
			valueIndex := completionShortAssignValueIndex(file, i, assign)
			return completionExpressionType(graph, prog, pkgIndex, fileIndex, file, assign+1, completionStatementEnd(file, assign+1, fn.BodyEnd), valueIndex)
		}
	}
	info := prog.Packages[pkgIndex]
	for i := 0; i < len(info.Decls); i++ {
		decl := info.Decls[i]
		if decl.Name == name {
			return completionPackageNameType(graph, prog, pkgIndex, name)
		}
	}
	return completionType{}, false
}

func completionExpressionType(graph load.Graph, prog Program, pkgIndex, fileIndex int, file syntax.File, start, end, resultIndex int) (completionType, bool) {
	if operator, kind := completionTopLevelBinary(file, start, end); operator >= 0 {
		if kind == exprBinaryCompare || kind == exprBinaryLogical {
			return completionType{Package: pkgIndex, Name: "bool"}, true
		}
		return completionExpressionType(graph, prog, pkgIndex, fileIndex, file, start, operator, 0)
	}
	address := false
	for start < end && start < len(file.Tokens) && tokenTextIs(&file, start, "&") {
		address = true
		start++
	}
	if start >= end || start >= len(file.Tokens) {
		return completionType{}, false
	}
	kind := file.Tokens[start].KindLine & 255
	if kind == syntax.TokenString {
		return completionType{Package: pkgIndex, Name: "string"}, true
	}
	if kind == syntax.TokenChar {
		return completionType{Package: pkgIndex, Name: "rune"}, true
	}
	if kind == syntax.TokenNumber {
		return completionType{Package: pkgIndex, Name: "int"}, true
	}
	if tokenTextIs(&file, start, "true") || tokenTextIs(&file, start, "false") {
		return completionType{Package: pkgIndex, Name: "bool"}, true
	}
	if kind != syntax.TokenIdent {
		return completionType{}, false
	}
	owner := pkgIndex
	nameTok := start
	if start+2 < end && tokenTextIs(&file, start+1, ".") && file.Tokens[start+2].KindLine&255 == syntax.TokenIdent {
		imported := completionImportPackage(prog.Packages[pkgIndex], fileIndex, tokenString(&file, start))
		if imported >= 0 {
			owner = imported
			nameTok = start + 2
		}
	}
	if owner < 0 {
		return completionType{}, false
	}
	name := tokenString(&file, nameTok)
	next := nameTok + 1
	if next < end && tokenTextIs(&file, next, "{") {
		return completionType{Package: owner, Name: name, Pointer: address}, true
	}
	if next < end && tokenTextIs(&file, next, "(") {
		if completionPredeclaredType(name) {
			return completionType{Package: pkgIndex, Name: name}, true
		}
		if completionBuiltinIntResult(name) {
			return completionType{Package: pkgIndex, Name: "int"}, true
		}
		return completionFunctionResultType(graph, prog, owner, name, resultIndex)
	}
	if owner == pkgIndex && next+1 < end && tokenTextIs(&file, next, ".") && file.Tokens[next+1].KindLine&255 == syntax.TokenIdent {
		fn, ok := completionFunctionAt(file, file.Tokens[start].Start)
		if !ok {
			return completionType{}, false
		}
		receiver, ok := completionNameType(graph, prog, pkgIndex, fileIndex, file, fn, name, file.Tokens[start].Start)
		if !ok {
			return completionType{}, false
		}
		member := tokenString(&file, next+1)
		if next+2 >= end || !tokenTextIs(&file, next+2, "(") {
			typ, found := completionFieldType(graph, prog, receiver, member)
			if address {
				typ.Pointer = true
			}
			return typ, found
		}
		target, ok := navigationFindMember(graph, prog, receiver, member, 0)
		if !ok || target.field {
			return completionType{}, false
		}
		return completionSymbolResultType(graph, prog, target.packageIndex, target.symbolIndex, resultIndex)
	}
	if owner == pkgIndex {
		if fn, ok := completionFunctionAt(file, file.Tokens[start].Start); ok {
			if typ, found := completionNameType(graph, prog, pkgIndex, fileIndex, file, fn, name, file.Tokens[start].Start); found {
				if address {
					typ.Pointer = true
				}
				return typ, true
			}
		}
	}
	return completionType{Package: owner, Name: name, Pointer: address}, true
}

func completionBuiltinIntResult(name string) bool {
	return name == "cap" || name == "copy" || name == "len"
}

func completionTopLevelBinary(file syntax.File, start, end int) (int, int) {
	depth := 0
	fallback := -1
	fallbackKind := exprBinaryNone
	for i := start; i < end && i < len(file.Tokens); i++ {
		if tokenTextIs(&file, i, "(") || tokenTextIs(&file, i, "[") || tokenTextIs(&file, i, "{") {
			depth++
			continue
		}
		if tokenTextIs(&file, i, ")") || tokenTextIs(&file, i, "]") || tokenTextIs(&file, i, "}") {
			if depth > 0 {
				depth--
			}
			continue
		}
		if depth == 0 && i > start {
			if kind := exprBinaryOperatorKind(file, i); kind != exprBinaryNone {
				if kind == exprBinaryCompare || kind == exprBinaryLogical {
					return i, kind
				}
				if fallback < 0 {
					fallback, fallbackKind = i, kind
				}
			}
		}
	}
	return fallback, fallbackKind
}

func completionPredeclaredType(name string) bool {
	types := []string{"bool", "byte", "complex64", "complex128", "error", "float32", "float64", "int", "int8", "int16", "int32", "int64", "rune", "string", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr"}
	for i := 0; i < len(types); i++ {
		if name == types[i] {
			return true
		}
	}
	return false
}

func completionFunctionResultType(graph load.Graph, prog Program, pkg int, name string, resultIndex int) (completionType, bool) {
	if pkg < 0 || pkg >= len(prog.Packages) || pkg >= len(graph.Packages) {
		return completionType{}, false
	}
	info := prog.Packages[pkg]
	for i := 0; i < len(info.Symbols); i++ {
		symbol := info.Symbols[i]
		if symbol.Kind != SymbolFunc || symbol.Name != name || symbol.File < 0 || symbol.File >= len(graph.Packages[pkg].Files) {
			continue
		}
		return completionSymbolResultType(graph, prog, pkg, i, resultIndex)
	}
	return completionType{}, false
}

func completionSymbolResultType(graph load.Graph, prog Program, pkg, symbolIndex, resultIndex int) (completionType, bool) {
	if pkg < 0 || pkg >= len(prog.Packages) || pkg >= len(graph.Packages) || symbolIndex < 0 || symbolIndex >= len(prog.Packages[pkg].Symbols) {
		return completionType{}, false
	}
	symbol := prog.Packages[pkg].Symbols[symbolIndex]
	if symbol.File < 0 || symbol.File >= len(graph.Packages[pkg].Files) {
		return completionType{}, false
	}
	file := graph.Packages[pkg].Files[symbol.File].File
	for i := 0; i < len(file.Funcs); i++ {
		if file.Funcs[i].NameTok != symbol.Token {
			continue
		}
		results := buildFuncSignature(file, file.Funcs[i]).Results
		if resultIndex >= 0 && resultIndex < len(results) {
			return completionSpanType(graph, prog, pkg, symbol.File, results[resultIndex].TypeStart, results[resultIndex].TypeEnd)
		}
	}
	return completionType{}, false
}

func completionShortAssignValueIndex(file syntax.File, name, assign int) int {
	index := 0
	for i := name - 1; i >= 0 && i < assign; i-- {
		if syntax.TokenLine(file.Tokens[i]) != syntax.TokenLine(file.Tokens[name]) || tokenTextIs(&file, i, ";") {
			break
		}
		if tokenTextIs(&file, i, ",") {
			index++
		}
	}
	return index
}

func completionFieldType(graph load.Graph, prog Program, typ completionType, fieldName string) (completionType, bool) {
	if typ.Package < 0 || typ.Package >= len(prog.Packages) || typ.Package >= len(graph.Packages) {
		return completionType{}, false
	}
	info := prog.Packages[typ.Package]
	index := LookupType(info, typ.Name)
	if index < 0 {
		return completionType{}, false
	}
	typeInfo := info.Types[index]
	for i := 0; i < len(typeInfo.Fields); i++ {
		if typeInfo.Fields[i].Name == fieldName {
			return completionSpanType(graph, prog, typ.Package, typeInfo.File, typeInfo.Fields[i].TypeStart, typeInfo.Fields[i].TypeEnd)
		}
	}
	return completionType{}, false
}

func completionSpanType(graph load.Graph, prog Program, pkg, fileIndex, start, end int) (completionType, bool) {
	if pkg < 0 || pkg >= len(graph.Packages) || fileIndex < 0 || fileIndex >= len(graph.Packages[pkg].Files) {
		return completionType{}, false
	}
	file := graph.Packages[pkg].Files[fileIndex].File
	pointer := false
	for start < end && start < len(file.Tokens) && file.Tokens[start].KindLine&255 != syntax.TokenIdent {
		if tokenTextIs(&file, start, "*") {
			pointer = true
		}
		start++
	}
	if start >= end || start >= len(file.Tokens) {
		return completionType{}, false
	}
	name := tokenString(&file, start)
	owner := pkg
	if start+2 < end && tokenTextIs(&file, start+1, ".") && file.Tokens[start+2].KindLine&255 == syntax.TokenIdent {
		owner = completionImportPackage(prog.Packages[pkg], fileIndex, name)
		name = tokenString(&file, start+2)
	}
	if owner < 0 || name == "" {
		return completionType{}, false
	}
	return completionType{Package: owner, Name: name, Pointer: pointer}, true
}

func completionFunctionAt(file syntax.File, offset int) (syntax.FuncDecl, bool) {
	for i := 0; i < len(file.Funcs); i++ {
		fn := file.Funcs[i]
		if fn.BodyStart >= 0 && fn.BodyEnd > fn.BodyStart && file.Tokens[fn.BodyStart].Start <= offset && offset <= file.Tokens[fn.BodyEnd-1].End {
			return fn, true
		}
	}
	return syntax.FuncDecl{}, false
}

func completionImportPackage(info PackageInfo, file int, name string) int {
	for i := 0; i < len(info.Imports); i++ {
		if info.Imports[i].File == file && info.Imports[i].Name == name && !info.Imports[i].Blank && !info.Imports[i].Dot {
			return info.Imports[i].Package
		}
	}
	return -1
}

func completionIdentifierStart(src []byte, offset int) int {
	for offset > 0 && completionIdentifierByte(src[offset-1]) {
		offset--
	}
	return offset
}

func completionSelectorComponents(src []byte, dot int) []string {
	var reversed []string
	end := dot
	for end > 0 {
		start := completionIdentifierStart(src, end)
		if start == end {
			break
		}
		reversed = append(reversed, string(src[start:end]))
		if start == 0 || src[start-1] != '.' {
			break
		}
		end = start - 1
	}
	components := make([]string, len(reversed))
	for i := 0; i < len(reversed); i++ {
		components[i] = reversed[len(reversed)-i-1]
	}
	return components
}

func completionFindShortAssign(file syntax.File, name, end int) int {
	line := syntax.TokenLine(file.Tokens[name])
	for i := name + 1; i < end && i < len(file.Tokens) && syntax.TokenLine(file.Tokens[i]) == line; i++ {
		if tokenTextIs(&file, i, ":=") {
			return i
		}
		if tokenTextIs(&file, i, ";") {
			break
		}
	}
	return -1
}

func completionStatementEnd(file syntax.File, start, limit int) int {
	if start >= len(file.Tokens) {
		return start
	}
	line := syntax.TokenLine(file.Tokens[start])
	for i := start; i < limit && i < len(file.Tokens); i++ {
		if tokenTextIs(&file, i, ";") || i > start && syntax.TokenLine(file.Tokens[i]) != line {
			return i
		}
	}
	return limit
}

func completionScopeDetail(kind int) string {
	if kind == NameReceiver {
		return "receiver"
	}
	if kind == NameParam {
		return "parameter"
	}
	if kind == NameResult {
		return "result"
	}
	return "local variable"
}

func completionAdd(items []CompletionItem, name, detail string, kind int, prefix string) []CompletionItem {
	if name == "" || name == prefix || !completionFuzzyMatch(name, prefix) {
		return items
	}
	for i := 0; i < len(items); i++ {
		if items[i].Name == name {
			return items
		}
	}
	return append(items, CompletionItem{Name: name, Detail: detail, Kind: kind})
}

func completionAddSymbol(items []CompletionItem, graph load.Graph, pkg int, symbol Symbol, displayName, prefix string) []CompletionItem {
	if displayName == "" || displayName == prefix || !completionFuzzyMatch(displayName, prefix) {
		return items
	}
	for i := 0; i < len(items); i++ {
		if items[i].Name == displayName {
			return items
		}
	}
	file, fn, ok := completionSymbolFunction(graph, pkg, symbol)
	if !ok {
		kind := CompletionFunction
		if symbol.Kind == SymbolMethod {
			kind = CompletionMethod
		}
		return append(items, CompletionItem{Name: displayName, Detail: "function", Kind: kind})
	}
	parameters := completionParameters(file, buildFuncSignature(file, fn).Params)
	label, detail := completionFunctionLabels(file, fn, displayName)
	kind := CompletionFunction
	if symbol.Kind == SymbolMethod {
		kind = CompletionMethod
	}
	documentation := sourceDocumentation(file.Src, file.Tokens[fn.StartTok].Start)
	return append(items, CompletionItem{Name: displayName, Detail: detail, Kind: kind, Signature: label, Documentation: documentation, Parameters: parameters})
}

func completionSymbolFunction(graph load.Graph, pkg int, symbol Symbol) (syntax.File, syntax.FuncDecl, bool) {
	if pkg < 0 || pkg >= len(graph.Packages) || symbol.File < 0 || symbol.File >= len(graph.Packages[pkg].Files) {
		return syntax.File{}, syntax.FuncDecl{}, false
	}
	file := graph.Packages[pkg].Files[symbol.File].File
	for i := 0; i < len(file.Funcs); i++ {
		if file.Funcs[i].NameTok == symbol.Token {
			return file, file.Funcs[i], true
		}
	}
	return syntax.File{}, syntax.FuncDecl{}, false
}

func completionFunctionLabels(file syntax.File, fn syntax.FuncDecl, name string) (string, string) {
	start := fn.ParamsStart
	end := fn.ResultEnd
	if start < 0 || start >= len(file.Tokens) {
		return name, "function"
	}
	if end <= start || end > len(file.Tokens) {
		end = fn.ParamsEnd
	}
	if end <= start || end > len(file.Tokens) {
		return name, "function"
	}
	startOffset := file.Tokens[start].Start
	endOffset := file.Tokens[end-1].End
	if startOffset < 0 || endOffset < startOffset || endOffset > len(file.Src) {
		return name, "function"
	}
	detail := string(file.Src[startOffset:endOffset])
	return name + detail, detail
}

func completionParameters(file syntax.File, fields []Field) []CompletionParameter {
	parameters := make([]CompletionParameter, 0, len(fields))
	for i := 0; i < len(fields); i++ {
		field := fields[i]
		parameters = append(parameters, CompletionParameter{Name: field.Name, Type: completionFieldTypeText(file, field)})
	}
	return parameters
}

func completionFieldTypeText(file syntax.File, field Field) string {
	if field.TypeStart < 0 || field.TypeEnd <= field.TypeStart || field.TypeEnd > len(file.Tokens) {
		return ""
	}
	start := file.Tokens[field.TypeStart].Start
	end := file.Tokens[field.TypeEnd-1].End
	if start < 0 || end < start || end > len(file.Src) {
		return ""
	}
	return string(file.Src[start:end])
}

func completionSort(items []CompletionItem, pattern string) {
	for i := 1; i < len(items); i++ {
		item := items[i]
		j := i - 1
		for j >= 0 && completionAfterMatch(items[j].Name, item.Name, pattern) {
			items[j+1] = items[j]
			j--
		}
		items[j+1] = item
	}
}

func completionAfterMatch(left, right, pattern string) bool {
	leftScore, _ := completionFuzzyScore(left, pattern)
	rightScore, _ := completionFuzzyScore(right, pattern)
	if leftScore != rightScore {
		return leftScore < rightScore
	}
	return completionAfter(left, right)
}

func completionAfter(left, right string) bool {
	limit := len(left)
	if len(right) < limit {
		limit = len(right)
	}
	for i := 0; i < limit; i++ {
		l, r := completionLower(left[i]), completionLower(right[i])
		if l != r {
			return l > r
		}
	}
	return len(left) > len(right)
}

func completionPrefixFold(name, prefix string) bool {
	if len(prefix) > len(name) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if completionLower(name[i]) != completionLower(prefix[i]) {
			return false
		}
	}
	return true
}

func completionFuzzyMatch(name, pattern string) bool {
	_, ok := completionFuzzyScore(name, pattern)
	return ok
}

// completionFuzzyScore treats the typed text as a case-insensitive
// subsequence. Prefixes remain strongest, followed by adjacent characters and
// word/camel-case boundaries, so fuzzy results never displace an exact prefix.
func completionFuzzyScore(name, pattern string) (int, bool) {
	if pattern == "" {
		return -len(name), true
	}
	searchAt := 0
	previous := -2
	score := 0
	for p := 0; p < len(pattern); p++ {
		matched := -1
		want := completionLower(pattern[p])
		for n := searchAt; n < len(name); n++ {
			if completionLower(name[n]) == want {
				matched = n
				break
			}
		}
		if matched < 0 {
			return 0, false
		}
		score += 10
		if matched == previous+1 {
			score += 12
		}
		if completionWordStart(name, matched) {
			score += 8
		}
		if name[matched] == pattern[p] {
			score++
		}
		previous = matched
		searchAt = matched + 1
	}
	if completionPrefixFold(name, pattern) {
		score += 1000
	}
	return score - len(name), true
}

func completionWordStart(name string, at int) bool {
	if at == 0 {
		return true
	}
	previous := name[at-1]
	current := name[at]
	return previous == '_' || previous == '-' || previous >= 'a' && previous <= 'z' && current >= 'A' && current <= 'Z'
}

func completionStartsWith(value, prefix string) bool {
	if len(prefix) > len(value) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if value[i] != prefix[i] {
			return false
		}
	}
	return true
}

func completionHasByte(value string, want byte) bool {
	for i := 0; i < len(value); i++ {
		if value[i] == want {
			return true
		}
	}
	return false
}

func completionIdentifierByte(value byte) bool {
	return value == '_' || value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9'
}

func completionExported(name string) bool {
	return name != "" && name[0] >= 'A' && name[0] <= 'Z'
}

func completionLower(value byte) byte {
	if value >= 'A' && value <= 'Z' {
		return value + ('a' - 'A')
	}
	return value
}
