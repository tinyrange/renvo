package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

// HoverInfo is the source-facing type and documentation for one identifier.
type HoverInfo struct {
	Signature     string
	Documentation string
	Start         int
	End           int
	Ok            bool
}

// HoverProgram returns the inferred type or declaration under offset together
// with its Go documentation. It uses the checked snapshot shared by completion
// and navigation, so unsaved editor contents receive the same answer.
func HoverProgram(graph load.Graph, program Program, path string, offset int) HoverInfo {
	pkgIndex, fileIndex := completionFile(graph, path)
	if pkgIndex < 0 || fileIndex < 0 || pkgIndex >= len(program.Packages) {
		return HoverInfo{}
	}
	file := graph.Packages[pkgIndex].Files[fileIndex].File
	token := navigationToken(file, offset)
	if token < 0 {
		return HoverInfo{}
	}
	target, ok := navigationResolve(graph, program, pkgIndex, fileIndex, token)
	if !ok {
		return HoverInfo{}
	}
	hover := HoverInfo{Start: file.Tokens[token].Start, End: file.Tokens[token].End}
	if target.local {
		hover.Signature = hoverLocalSignature(graph, program, target, tokenString(&file, token), file.Tokens[token].End)
	} else if target.member && target.field {
		hover.Signature, hover.Documentation = hoverField(graph, program, target)
	} else {
		hover.Signature, hover.Documentation = hoverSymbol(graph, program, target.packageIndex, target.symbolIndex)
	}
	hover.Ok = hover.Signature != "" || hover.Documentation != ""
	return hover
}

func hoverLocalSignature(graph load.Graph, program Program, target navigationTarget, name string, before int) string {
	if target.packageIndex < 0 || target.packageIndex >= len(program.Packages) || target.fileIndex < 0 ||
		target.fileIndex >= len(graph.Packages[target.packageIndex].Files) {
		return ""
	}
	file := graph.Packages[target.packageIndex].Files[target.fileIndex].File
	fn, ok := completionFunctionAt(file, before)
	if !ok {
		return ""
	}
	signature := buildFuncSignature(file, fn)
	groups := [][]Field{signature.Receiver, signature.Params, signature.Results}
	for i := 0; i < len(groups); i++ {
		for j := 0; j < len(groups[i]); j++ {
			if groups[i][j].Name == name {
				return "var " + name + " " + completionFieldTypeText(file, groups[i][j])
			}
		}
	}
	if typ, found := completionNameType(graph, program, target.packageIndex, target.fileIndex, file, fn, name, before); found {
		return "var " + name + " " + hoverTypeName(program, target.packageIndex, typ)
	}
	info := program.Packages[target.packageIndex]
	for i := 0; i < len(info.Bodies); i++ {
		body := info.Bodies[i]
		if body.File != target.fileIndex || body.Func < 0 || body.Func >= len(file.Funcs) ||
			before < file.Tokens[file.Funcs[body.Func].StartTok].Start || before > file.Tokens[file.Funcs[body.Func].EndTok-1].End {
			continue
		}
		for j := len(body.Locals) - 1; j >= 0; j-- {
			local := body.Locals[j]
			if local.Name != name || local.Token < 0 || local.Token >= len(file.Tokens) || file.Tokens[local.Token].Start > before {
				continue
			}
			if local.TypeStart >= 0 && local.TypeEnd > local.TypeStart {
				return "var " + name + " " + hoverSpanText(file, local.TypeStart, local.TypeEnd)
			}
			resultIndex := local.ValueIndex
			start, end := local.ValueStart, local.ValueEnd
			if len(local.Values) > local.ValueIndex {
				start, end = local.Values[local.ValueIndex].StartTok, local.Values[local.ValueIndex].EndTok
				resultIndex = 0
			}
			if typ, found := completionExpressionType(graph, program, target.packageIndex, target.fileIndex, file, start, end, resultIndex); found {
				return "var " + name + " " + hoverTypeName(program, target.packageIndex, typ)
			}
			if inferred := hoverLiteralType(file, start, end); inferred != "" {
				return "var " + name + " " + inferred
			}
		}
	}
	return "var " + name
}

func hoverSymbol(graph load.Graph, program Program, pkgIndex, symbolIndex int) (string, string) {
	if pkgIndex < 0 || pkgIndex >= len(program.Packages) || pkgIndex >= len(graph.Packages) ||
		symbolIndex < 0 || symbolIndex >= len(program.Packages[pkgIndex].Symbols) {
		return "", ""
	}
	symbol := program.Packages[pkgIndex].Symbols[symbolIndex]
	if symbol.File < 0 || symbol.File >= len(graph.Packages[pkgIndex].Files) {
		return "", ""
	}
	file := graph.Packages[pkgIndex].Files[symbol.File].File
	documentation := sourceDocumentation(file.Src, file.Tokens[symbol.Token].Start)
	if symbol.Kind == SymbolFunc || symbol.Kind == SymbolMethod {
		for i := 0; i < len(file.Funcs); i++ {
			if file.Funcs[i].NameTok == symbol.Token {
				label, _ := completionFunctionLabels(file, file.Funcs[i], tokenString(&file, symbol.Token))
				return "func " + label, documentation
			}
		}
	}
	for i := 0; i < len(program.Packages[pkgIndex].Decls); i++ {
		decl := program.Packages[pkgIndex].Decls[i]
		if decl.Symbol != symbolIndex {
			continue
		}
		kind := "var"
		if symbol.Kind == SymbolConst {
			kind = "const"
		} else if symbol.Kind == SymbolType {
			kind = "type"
		}
		text := kind + " " + symbol.Name
		if decl.TypeStart >= 0 && decl.TypeEnd > decl.TypeStart {
			text += " " + hoverSpanText(file, decl.TypeStart, decl.TypeEnd)
		}
		return text, documentation
	}
	return symbol.Name, documentation
}

func hoverField(graph load.Graph, program Program, target navigationTarget) (string, string) {
	if target.packageIndex < 0 || target.packageIndex >= len(program.Packages) || target.packageIndex >= len(graph.Packages) ||
		target.symbolIndex < 0 || target.symbolIndex >= len(program.Packages[target.packageIndex].Types) {
		return "", ""
	}
	typ := program.Packages[target.packageIndex].Types[target.symbolIndex]
	fieldIndex := LookupField(typ.Fields, target.memberName)
	if fieldIndex < 0 || typ.File < 0 || typ.File >= len(graph.Packages[target.packageIndex].Files) {
		return "", ""
	}
	file := graph.Packages[target.packageIndex].Files[typ.File].File
	field := typ.Fields[fieldIndex]
	documentation := sourceDocumentation(file.Src, file.Tokens[field.NameTok].Start)
	return "field " + field.Name + " " + hoverSpanText(file, field.TypeStart, field.TypeEnd), documentation
}

func hoverSpanText(file syntax.File, start, end int) string {
	if start < 0 || end <= start || end > len(file.Tokens) {
		return ""
	}
	first, last := file.Tokens[start].Start, file.Tokens[end-1].End
	if first < 0 || last < first || last > len(file.Src) {
		return ""
	}
	return string(file.Src[first:last])
}

func hoverTypeName(program Program, origin int, typ completionType) string {
	name := typ.Name
	if hoverSimpleTypeName(name) && typ.Package >= 0 && typ.Package < len(program.Packages) && typ.Package != origin && program.Packages[typ.Package].Name != "" {
		name = program.Packages[typ.Package].Name + "." + name
	}
	return name
}

func hoverSimpleTypeName(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		ch := name[i]
		if !(ch == '_' || ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || i > 0 && ch >= '0' && ch <= '9') {
			return false
		}
	}
	return true
}

func hoverLiteralType(file syntax.File, start, end int) string {
	if start < 0 || start >= end || start >= len(file.Tokens) {
		return ""
	}
	token := file.Tokens[start]
	if token.KindLine&255 == syntax.TokenString {
		return "string"
	}
	if token.KindLine&255 == syntax.TokenChar {
		return "rune"
	}
	if token.KindLine&255 == syntax.TokenNumber {
		return "int"
	}
	if tokenTextIs(&file, start, "true") || tokenTextIs(&file, start, "false") {
		return "bool"
	}
	return ""
}
