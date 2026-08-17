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
		name := tokenString(&file, token)
		if signature, documentation, found := hoverImportedPackage(graph, program, pkgIndex, fileIndex, name); found {
			return HoverInfo{Signature: signature, Documentation: documentation, Start: file.Tokens[token].Start, End: file.Tokens[token].End, Ok: true}
		}
		if signature, documentation, found := hoverBuiltin(name); found {
			return HoverInfo{Signature: signature, Documentation: documentation, Start: file.Tokens[token].Start, End: file.Tokens[token].End, Ok: true}
		}
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
			typeText := ""
			if local.TypeStart >= 0 && local.TypeEnd > local.TypeStart {
				typeText = hoverSpanText(file, local.TypeStart, local.TypeEnd)
			}
			if local.Kind == SymbolConst {
				return hoverConstSignature(name, typeText, local.Const)
			}
			if typeText != "" {
				return "var " + name + " " + typeText
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
	body := syntax.ParseFuncBody(file, fn)
	if body.Ok {
		scope, scopeOK, _ := buildFuncScope(file, fn, body)
		if scopeOK {
			locals := buildFuncLocalDecls(file, target.fileIndex, info, program.Packages, body, scope)
			for i := len(locals) - 1; i >= 0; i-- {
				local := locals[i]
				if local.Name == name && local.Token >= 0 && local.Token < len(file.Tokens) && file.Tokens[local.Token].Start <= before && local.Kind == SymbolConst {
					return hoverConstSignature(name, hoverSpanText(file, local.TypeStart, local.TypeEnd), local.Const)
				}
			}
		}
	}
	if typ, found := completionNameType(graph, program, target.packageIndex, target.fileIndex, file, fn, name, before); found {
		return "var " + name + " " + hoverTypeName(program, target.packageIndex, typ)
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
		if symbol.Kind == SymbolConst {
			value := decl.Const
			if !value.Ok && decl.ValueStart >= 0 && decl.ValueEnd > decl.ValueStart {
				value = evalConstValue(file, splitExprList(file, decl.ValueStart, decl.ValueEnd), decl.ValueIndex)
			}
			text = hoverConstSignature(symbol.Name, hoverSpanText(file, decl.TypeStart, decl.TypeEnd), value)
		}
		return text, documentation
	}
	return symbol.Name, documentation
}

func hoverImportedPackage(graph load.Graph, program Program, pkgIndex, fileIndex int, name string) (string, string, bool) {
	if pkgIndex < 0 || pkgIndex >= len(program.Packages) {
		return "", "", false
	}
	info := program.Packages[pkgIndex]
	imported := completionImportPackage(info, fileIndex, name)
	if imported < 0 || imported >= len(program.Packages) || imported >= len(graph.Packages) {
		return "", "", false
	}
	importPath := ""
	for i := 0; i < len(info.Imports); i++ {
		imp := info.Imports[i]
		if imp.File == fileIndex && imp.Name == name && imp.Package == imported {
			importPath = imp.ImportPath
			break
		}
	}
	documentation := ""
	for i := 0; i < len(graph.Packages[imported].Files); i++ {
		file := graph.Packages[imported].Files[i].File
		if file.PackageName >= 0 && file.PackageName < len(file.Tokens) {
			documentation = sourceDocumentation(file.Src, file.Tokens[file.PackageName].Start)
			if documentation != "" {
				break
			}
		}
	}
	signature := "package " + program.Packages[imported].Name
	if importPath != "" {
		signature += " // \"" + importPath + "\""
	}
	return signature, documentation, true
}

func hoverConstSignature(name, typeText string, value ConstValue) string {
	text := "const " + name
	if typeText != "" {
		text += " " + typeText
	}
	if !value.Ok {
		return text
	}
	if value.Kind == ConstInt {
		return text + " = " + hoverDecimal(value.Int) + " // " + hoverHex(value.Int)
	}
	if value.Kind == ConstString {
		return text + " = \"" + value.String + "\""
	}
	if value.Kind == ConstBool {
		if value.Bool {
			return text + " = true"
		}
		return text + " = false"
	}
	return text
}

func hoverDecimal(value int) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	var magnitude uint
	if negative {
		magnitude = uint(-(value + 1)) + 1
	} else {
		magnitude = uint(value)
	}
	var reverse [32]byte
	count := 0
	for magnitude > 0 {
		reverse[count] = byte('0' + magnitude%10)
		count++
		magnitude /= 10
	}
	out := make([]byte, count)
	for i := 0; i < count; i++ {
		out[i] = reverse[count-i-1]
	}
	if negative {
		return "-" + string(out)
	}
	return string(out)
}

func hoverHex(value int) string {
	negative := value < 0
	var magnitude uint
	if negative {
		magnitude = uint(-(value + 1)) + 1
	} else {
		magnitude = uint(value)
	}
	if magnitude == 0 {
		return "0x0"
	}
	const digits = "0123456789abcdef"
	var reverse [32]byte
	count := 0
	for magnitude > 0 {
		reverse[count] = digits[magnitude%16]
		count++
		magnitude /= 16
	}
	out := make([]byte, count)
	for i := 0; i < count; i++ {
		out[i] = reverse[count-i-1]
	}
	if negative {
		return "-0x" + string(out)
	}
	return "0x" + string(out)
}

func hoverBuiltin(name string) (string, string, bool) {
	if hoverBuiltinType(name) {
		documentation := "Predeclared Go type."
		if name == "byte" {
			documentation = "Predeclared alias for uint8."
		} else if name == "rune" {
			documentation = "Predeclared alias for int32, conventionally used for Unicode code points."
		} else if name == "error" {
			documentation = "Predeclared interface for values that describe an error condition."
		} else if name == "any" {
			documentation = "Predeclared alias for interface{}; it accepts a value of any type."
		}
		return "type " + name, documentation, true
	}
	signature, documentation := "", ""
	switch name {
	case "append":
		signature, documentation = "func append(slice []T, values ...T) []T", "Appends values to a slice and returns the resulting slice."
	case "cap":
		signature, documentation = "func cap(value T) int", "Returns the capacity of an array, slice, or channel."
	case "clear":
		signature, documentation = "func clear(value T)", "Deletes all map entries or zeroes all slice elements."
	case "close":
		signature, documentation = "func close(channel chan<- T)", "Closes a channel."
	case "complex":
		signature, documentation = "func complex(real, imag T) ComplexT", "Constructs a complex value from real and imaginary components."
	case "copy":
		signature, documentation = "func copy(dst, src []T) int", "Copies elements and returns the number copied."
	case "delete":
		signature, documentation = "func delete(m map[K]V, key K)", "Deletes the map entry for key."
	case "imag":
		signature, documentation = "func imag(value ComplexT) T", "Returns the imaginary component of a complex value."
	case "len":
		signature, documentation = "func len(value T) int", "Returns the length of a string, array, slice, map, or channel."
	case "make":
		signature, documentation = "func make(T, size ...int) T", "Allocates and initializes a slice, map, or channel."
	case "max":
		signature, documentation = "func max(values ...T) T", "Returns the largest ordered value."
	case "min":
		signature, documentation = "func min(values ...T) T", "Returns the smallest ordered value."
	case "new":
		signature, documentation = "func new(T) *T", "Allocates a zero value and returns a pointer to it."
	case "panic":
		signature, documentation = "func panic(value any)", "Stops normal execution and begins panicking."
	case "print":
		signature, documentation = "func print(values ...T)", "Writes values using Renvo's implementation-defined debug format."
	case "println":
		signature, documentation = "func println(values ...T)", "Writes values using Renvo's implementation-defined debug format followed by a newline."
	case "real":
		signature, documentation = "func real(value ComplexT) T", "Returns the real component of a complex value."
	case "recover":
		signature, documentation = "func recover() any", "Stops a panicking sequence when called by a deferred function."
	default:
		return "", "", false
	}
	return signature, documentation, true
}

func hoverBuiltinType(name string) bool {
	types := []string{"any", "bool", "byte", "complex64", "complex128", "error", "float32", "float64", "int", "int8", "int16", "int32", "int64", "rune", "string", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr"}
	for i := 0; i < len(types); i++ {
		if name == types[i] {
			return true
		}
	}
	return false
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
	if hoverSimpleTypeName(name) && !completionPredeclaredType(name) && typ.Package >= 0 && typ.Package < len(program.Packages) && typ.Package != origin && program.Packages[typ.Package].Name != "" {
		name = program.Packages[typ.Package].Name + "." + name
	}
	if typ.Pointer {
		name = "*" + name
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
