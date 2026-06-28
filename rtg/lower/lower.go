package lower

import (
	"fmt"
	"strconv"
	"strings"

	"j5.nz/rtg/rtg/load"
	"j5.nz/rtg/rtg/parse"
	"j5.nz/rtg/rtg/scan"
	"j5.nz/rtg/rtg/unit"
)

type localTypeInfo struct {
	qualifier string
	name      string
	pointer   bool
}

type methodInfo struct {
	name            string
	receiverType    string
	pointerReceiver bool
	unitName        string
	importPath      string
}

type symbolName struct {
	name     string
	unitName string
}

type methodEntry struct {
	lookup string
	info   methodInfo
}

type importSymbolGroup struct {
	localName string
	symbols   []unit.Symbol
}

type localNameRange struct {
	name  string
	start int
	end   int
}

type localTypeEntry struct {
	name string
	info localTypeInfo
}

type symbolNameTable []symbolName

type methodTable []methodEntry

type importSymbolTable []importSymbolGroup

type localNameTable []localNameRange

type localTypeTable []localTypeEntry

func symbolNameTableUnitName(table symbolNameTable, name string) string {
	for i := 0; i < len(table); i++ {
		entry := table[i]
		if entry.name == name {
			return entry.unitName
		}
	}
	return ""
}

func symbolNameTableSet(table symbolNameTable, name string, unitName string) symbolNameTable {
	for i := 0; i < len(table); i++ {
		if table[i].name == name {
			table[i].unitName = unitName
			return table
		}
	}
	return append(table, symbolName{name: name, unitName: unitName})
}

func methodTableLookup(table methodTable, name string) methodInfo {
	for i := 0; i < len(table); i++ {
		entry := table[i]
		if entry.lookup == name {
			info := entry.info
			return info
		}
	}
	return methodInfo{}
}

func methodTableSet(table methodTable, name string, info methodInfo) methodTable {
	for i := 0; i < len(table); i++ {
		if table[i].lookup == name {
			table[i].info = info
			return table
		}
	}
	return append(table, methodEntry{lookup: name, info: info})
}

func importSymbolTableSymbols(table importSymbolTable, localName string) ([]unit.Symbol, bool) {
	for i := 0; i < len(table); i++ {
		entry := table[i]
		if entry.localName == localName {
			return entry.symbols, true
		}
	}
	return nil, false
}

func symbolByName(symbols []unit.Symbol, name string) (unit.Symbol, bool) {
	for i := 0; i < len(symbols); i++ {
		sym := symbols[i]
		if sym.Name == name {
			return sym, true
		}
	}
	return unit.Symbol{}, false
}

func setSymbol(symbols []unit.Symbol, sym unit.Symbol) []unit.Symbol {
	for i := 0; i < len(symbols); i++ {
		if symbols[i].Name == sym.Name {
			symbols[i] = sym
			return symbols
		}
	}
	return append(symbols, sym)
}

func localTypeTableLookup(table localTypeTable, name string) localTypeInfo {
	for i := 0; i < len(table); i++ {
		entry := table[i]
		if entry.name == name {
			info := entry.info
			return info
		}
	}
	return localTypeInfo{}
}

func localTypeTableSet(table localTypeTable, name string, info localTypeInfo) localTypeTable {
	for i := 0; i < len(table); i++ {
		if table[i].name == name {
			table[i].info = info
			return table
		}
	}
	return append(table, localTypeEntry{name: name, info: info})
}

func Package(pkg load.Package) (unit.Unit, error) {
	return PackageWithGraph(pkg, nil)
}

func PackageWithGraph(pkg load.Package, graph *load.Graph) (unit.Unit, error) {
	u := unit.Unit{ImportPath: pkg.ImportPath, Package: pkg.Name, Entry: pkg.Entry}
	u.Imports = appendStrings(u.Imports, pkg.Imports)
	files := copyLoadFiles(pkg.Files)
	sortFilesByPath(files)
	parsedFiles := make([]parse.File, 0, len(files))
	var topNames symbolNameTable
	var topNameOrder []string
	var methods methodTable
	var methodOrder []string
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		file := files[fileIndex]
		parsed, err := parsedLoadFile(file)
		if err != nil {
			return unit.Unit{}, err
		}
		if parsed.PackageName != pkg.Name {
			return unit.Unit{}, fmt.Errorf("%s: package name %s does not match loaded package %s", file.Path, parsed.PackageName, pkg.Name)
		}
		parsedFiles = append(parsedFiles, parsed)
		decls := parsed.Decls
		for declIndex := 0; declIndex < len(decls); declIndex++ {
			decl := &decls[declIndex]
			names := declTopNames(&parsed, decl)
			for nameIndex := 0; nameIndex < len(names); nameIndex++ {
				name := names[nameIndex]
				if name != "" && name != "_" {
					if symbolNameTableUnitName(topNames, name) == "" {
						topNameOrder = append(topNameOrder, name)
					}
					topNames = symbolNameTableSet(topNames, name, SymbolName(pkg.ImportPath, name))
				}
			}
		}
	}
	for fileIndex := 0; fileIndex < len(parsedFiles); fileIndex++ {
		parsed := parsedFiles[fileIndex]
		decls := parsed.Decls
		for declIndex := 0; declIndex < len(decls); declIndex++ {
			decl := &decls[declIndex]
			if decl.Kind != "func" || !decl.Receiver {
				continue
			}
			info := methodDeclInfo(&parsed, decl)
			if info.name != "" {
				info.unitName = symbolNameTableUnitName(topNames, info.name)
				existingMethod := methodTableLookup(methods, info.name)
				if existingMethod.unitName == "" {
					methodOrder = append(methodOrder, info.name)
				}
				methods = methodTableSet(methods, info.name, info)
			}
		}
	}
	syntheticEntrypoint := false
	if pkg.Name == "main" && symbolNameTableUnitName(topNames, "appMain") == "" && symbolNameTableUnitName(topNames, "main") != "" && hasOrdinaryMain(parsedFiles) {
		topNameOrder = append(topNameOrder, "appMain")
		topNames = symbolNameTableSet(topNames, "appMain", SymbolName(pkg.ImportPath, "appMain"))
		syntheticEntrypoint = true
	}
	for i := 0; i < len(topNameOrder); i++ {
		name := topNameOrder[i]
		unitName := symbolNameTableUnitName(topNames, name)
		if isExported(name) {
			u.Exports = append(u.Exports, unit.Symbol{ImportPath: pkg.ImportPath, Name: name, UnitName: unitName})
		}
	}
	sortSymbolsByName(u.Exports)
	depPackages := dependencyPackages(graph)
	var seenRefs []string
	for fileIndex := 0; fileIndex < len(parsedFiles); fileIndex++ {
		parsed := &parsedFiles[fileIndex]
		importRefs, _ := importReferenceMap(parsed, depPackages)
		importMethods, importMethodOrder := importMethodMap(parsed, depPackages)
		allMethods := mergedMethodMap(methods, methodOrder, importMethods, importMethodOrder)
		decls := parsed.Decls
		for declIndex := 0; declIndex < len(decls); declIndex++ {
			decl := &decls[declIndex]
			var refs []unit.Symbol
			body := rewriteDecl(parsed, decl, topNames, importRefs, allMethods, &refs)
			if decl.Kind == "func" {
				body = normalizeFunctionExpressions(body, unitDeclSymbol(decl, parsed, topNames))
			}
			for refIndex := 0; refIndex < len(refs); refIndex++ {
				ref := refs[refIndex]
				key := ref.ImportPath + "\x00" + ref.Name
				if !containsString(seenRefs, key) {
					seenRefs = append(seenRefs, key)
					u.References = append(u.References, ref)
				}
			}
			var outDecl unit.Decl
			outDecl.Path = unitPathForDecl(files, parsed.Path)
			outDecl.Kind = decl.Kind
			outDecl.Name = unitDeclName(parsed, decl)
			outDecl.UnitName = unitDeclSymbol(decl, parsed, topNames)
			outDecl.Body = body
			u.Decls = append(u.Decls, outDecl)
		}
	}
	if syntheticEntrypoint {
		if containsString(pkg.Imports, "os") && !containsString(seenRefs, "os\x00Args") {
			u.References = append(u.References, unit.Symbol{ImportPath: "os", Name: "Args", UnitName: SymbolName("os", "Args")})
		}
		u.Decls = append(u.Decls, syntheticAppMainDecl(symbolNameTableUnitName(topNames, "appMain"), symbolNameTableUnitName(topNames, "main"), containsString(pkg.Imports, "os")))
	}
	sortSymbolsByImportPathName(u.References)
	return u, nil
}

func parsedLoadFile(file load.File) (parse.File, error) {
	if file.Parsed.Path != "" {
		return file.Parsed, nil
	}
	return parse.FileSource(file.Path, file.Source)
}

func sortFilesByPath(files []load.File) {
	for i := 1; i < len(files); i++ {
		value := files[i]
		j := i - 1
		for j >= 0 && stringGreater(files[j].Path, value.Path) {
			files[j+1] = files[j]
			j = j - 1
		}
		files[j+1] = value
	}
}

func sortSymbolsByName(symbols []unit.Symbol) {
	for i := 1; i < len(symbols); i++ {
		value := symbols[i]
		j := i - 1
		for j >= 0 && stringGreater(symbols[j].Name, value.Name) {
			symbols[j+1] = symbols[j]
			j = j - 1
		}
		symbols[j+1] = value
	}
}

func sortSymbolsByImportPathName(symbols []unit.Symbol) {
	for i := 1; i < len(symbols); i++ {
		value := symbols[i]
		j := i - 1
		for j >= 0 && symbolAfterByImportPathName(symbols[j], value) {
			symbols[j+1] = symbols[j]
			j = j - 1
		}
		symbols[j+1] = value
	}
}

func symbolAfterByImportPathName(a unit.Symbol, b unit.Symbol) bool {
	if a.ImportPath == b.ImportPath {
		return stringGreater(a.Name, b.Name)
	}
	return stringGreater(a.ImportPath, b.ImportPath)
}

func stringGreater(a string, b string) bool {
	i := 0
	for i < len(a) && i < len(b) {
		if a[i] > b[i] {
			return true
		}
		if a[i] < b[i] {
			return false
		}
		i = i + 1
	}
	return len(a) > len(b)
}

func unitDeclName(file *parse.File, decl *parse.Decl) string {
	if decl.Kind == "func" && decl.Receiver {
		return methodDeclName(file, decl)
	}
	names := declNames(decl)
	if len(names) == 1 {
		return names[0]
	}
	if decl.Kind == "func" {
		return decl.Name
	}
	return strings.Join(names, ", ")
}

func unitDeclSymbol(decl *parse.Decl, file *parse.File, topNames symbolNameTable) string {
	if decl.Kind == "func" && decl.Receiver {
		return symbolNameTableUnitName(topNames, methodDeclName(file, decl))
	}
	names := declNames(decl)
	if len(names) != 1 {
		return ""
	}
	return symbolNameTableUnitName(topNames, names[0])
}

func declTopNames(file *parse.File, decl *parse.Decl) []string {
	if decl.Kind == "func" && decl.Receiver {
		name := methodDeclName(file, decl)
		if name == "" {
			return nil
		}
		return []string{name}
	}
	return declNames(decl)
}

func methodDeclName(file *parse.File, decl *parse.Decl) string {
	info := methodDeclInfoFromTokens(file.Tokens, decl)
	if info.receiverType == "" || decl.Name == "" {
		return decl.Name
	}
	return info.name
}

func methodDeclInfo(file *parse.File, decl *parse.Decl) methodInfo {
	return methodDeclInfoFromTokens(file.Tokens, decl)
}

func methodDeclNameFromTokens(toks []scan.Token, decl *parse.Decl) string {
	info := methodDeclInfoFromTokens(toks, decl)
	if info.receiverType == "" || decl.Name == "" {
		return decl.Name
	}
	return info.name
}

func methodDeclInfoFromTokens(toks []scan.Token, decl *parse.Decl) methodInfo {
	receiver := methodReceiverTypeNameFromTokens(toks, decl)
	name := decl.Name
	if receiver != "" && decl.Name != "" {
		name = receiver + "_" + decl.Name
	}
	return methodInfo{
		name:            name,
		receiverType:    receiver,
		pointerReceiver: methodReceiverIsPointerFromTokens(toks, decl),
	}
}

func methodReceiverTypeName(file *parse.File, decl *parse.Decl) string {
	return methodReceiverTypeNameFromTokens(file.Tokens, decl)
}

func methodReceiverTypeNameFromTokens(toks []scan.Token, decl *parse.Decl) string {
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 || start+1 >= len(toks) || toks[start+1].Text != "(" {
		return ""
	}
	close := findClose(toks, start+1, "(", ")")
	if close < 0 {
		return ""
	}
	name := ""
	for i := start + 2; i < close; i++ {
		if toks[i].Kind == scan.Ident {
			name = toks[i].Text
		}
	}
	return name
}

func methodReceiverIsPointer(file *parse.File, decl *parse.Decl) bool {
	return methodReceiverIsPointerFromTokens(file.Tokens, decl)
}

func methodReceiverIsPointerFromTokens(toks []scan.Token, decl *parse.Decl) bool {
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 || start+1 >= len(toks) || toks[start+1].Text != "(" {
		return false
	}
	close := findClose(toks, start+1, "(", ")")
	if close < 0 {
		return false
	}
	return containsTokenText(toks, start+2, close, "*")
}

func hasOrdinaryMain(files []parse.File) bool {
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		file := files[fileIndex]
		for declIndex := 0; declIndex < len(file.Decls); declIndex++ {
			decl := file.Decls[declIndex]
			if isOrdinaryMainDecl(file, decl) {
				return true
			}
		}
	}
	return false
}

func isOrdinaryMainDecl(file parse.File, decl parse.Decl) bool {
	if decl.Kind != "func" || decl.Name != "main" || decl.Receiver {
		return false
	}
	name := tokenIndexAt(file.Tokens, decl.NameTok.Start)
	if name < 0 || name+1 >= len(file.Tokens) || file.Tokens[name+1].Text != "(" {
		return false
	}
	open := name + 1
	close := findClose(file.Tokens, open, "(", ")")
	if close != open+1 {
		return false
	}
	for i := close + 1; i < len(file.Tokens) && file.Tokens[i].Start < decl.End; i++ {
		if file.Tokens[i].Text == "{" {
			return true
		}
		return false
	}
	return false
}

func syntheticAppMainDecl(appMainUnitName string, mainUnitName string, setOSArgs bool) unit.Decl {
	body := "func " + appMainUnitName + "(args []string, env []string) int {\n"
	if setOSArgs {
		body = body + "\t" + SymbolName("os", "Args") + " = args\n"
	}
	body = body + "\t" + mainUnitName + "()\n\treturn 0\n}\n"
	return unit.Decl{
		Path:     "rtg-entrypoint",
		Kind:     "func",
		Name:     "appMain",
		UnitName: appMainUnitName,
		Body:     body,
	}
}

func declNames(decl *parse.Decl) []string {
	if len(decl.Names) > 1 {
		return decl.Names
	}
	if decl.Name == "" {
		if len(decl.Names) > 0 {
			return decl.Names
		}
		return nil
	}
	return []string{decl.Name}
}

func unitPathForDecl(files []load.File, path string) string {
	for i := 0; i < len(files); i++ {
		file := files[i]
		if file.Path == path {
			if file.UnitPath != "" {
				return file.UnitPath
			}
			return file.Path
		}
	}
	return path
}

func SymbolName(importPath string, name string) string {
	out := []byte("rtg_")
	for i := 0; i < len(importPath); i++ {
		c := importPath[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			out = append(out, c)
		} else {
			out = append(out, '_')
		}
	}
	out = append(out, '_')
	out = appendString(out, name)
	return string(out)
}

func appendString(out []byte, s string) []byte {
	for i := 0; i < len(s); i++ {
		out = append(out, s[i])
	}
	return out
}

func appendBytes(out []byte, values []byte) []byte {
	for i := 0; i < len(values); i++ {
		out = append(out, values[i])
	}
	return out
}

func appendStrings(out []string, values []string) []string {
	for i := 0; i < len(values); i++ {
		out = append(out, values[i])
	}
	return out
}

func copyLoadFiles(values []load.File) []load.File {
	out := make([]load.File, len(values))
	for i := 0; i < len(values); i++ {
		out[i] = values[i]
	}
	return out
}

func appendExpressionTemps(out []expressionTemp, values []expressionTemp) []expressionTemp {
	for i := 0; i < len(values); i++ {
		value := values[i]
		out = append(out, value)
	}
	return out
}

func appendExpressionReplacements(out []expressionReplacement, values []expressionReplacement) []expressionReplacement {
	for i := 0; i < len(values); i++ {
		value := values[i]
		out = append(out, value)
	}
	return out
}

func rewriteDecl(file *parse.File, decl *parse.Decl, topNames symbolNameTable, importRefs importSymbolTable, methods methodTable, refs *[]unit.Symbol) string {
	start := decl.Start
	end := decl.End
	if start < 0 {
		start = 0
	}
	if end > len(file.Source) {
		end = len(file.Source)
	}
	for end > start && (file.Source[end-1] == ' ' || file.Source[end-1] == '\t' || file.Source[end-1] == '\r' || file.Source[end-1] == '\n') {
		end--
	}
	var out []byte
	localNames := localNamesForDecl(file, decl, topNames)
	var importNames symbolNameTable
	for i := 0; i < len(importRefs); i++ {
		name := importRefs[i].localName
		importNames = symbolNameTableSet(importNames, name, name)
	}
	importLocalNames := localNamesForDecl(file, decl, importNames)
	var localTypes localTypeTable
	localTypes = localTypesForDecl(file, decl)
	cursor := start
	if decl.Kind == "func" && decl.Receiver {
		cursor = appendMethodDeclPrefix(file, decl, topNames, &out)
	}
	prevText := ""
	tokens := file.Tokens
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if tok.End <= cursor {
			prevText = tok.Text
			continue
		}
		if tok.Start >= end {
			break
		}
		if tok.Start > cursor {
			out = appendBytes(out, file.Source[cursor:tok.Start])
		}
		if tok.Kind == scan.Ident && i+2 < len(tokens) && tokens[i+1].Text == "." && tokens[i+2].Kind == scan.Ident {
			if i+3 < len(tokens) && tokens[i+3].Text == "(" {
				receiverType := localTypeTableLookup(localTypes, tok.Text)
				if receiverType.name != "" {
					memberTok := tokens[i+2]
					methodName := methodLookupName(receiverType, memberTok.Text)
					method := methodTableLookup(methods, methodName)
					if method.unitName != "" {
						open := tokens[i+3]
						close := findClose(tokens, i+3, "(", ")")
						if method.importPath != "" {
							appendUnitSymbolRef(refs, unit.Symbol{ImportPath: method.importPath, Name: method.name, UnitName: method.unitName})
						}
						receiverArg := tok.Text
						if method.pointerReceiver && !receiverType.pointer {
							receiverArg = "&" + receiverArg
						} else if !method.pointerReceiver && receiverType.pointer {
							receiverArg = "*" + receiverArg
						}
						replacement := method.unitName + "(" + receiverArg
						if close < 0 || open.End < tokens[close].Start {
							replacement = replacement + ", "
						}
						out = appendString(out, replacement)
						cursor = open.End
						prevText = "("
						i += 3
						continue
					}
				}
			}
			symbols, symbolsOK := importSymbolTableSymbols(importRefs, tok.Text)
			if symbolsOK && !isLocalNameAt(importLocalNames, tok.Text, tok.Start) {
				member := tokens[i+2]
				sym, symOK := symbolByName(symbols, member.Text)
				if symOK {
					if sym.ImportPath != "" {
						appendUnitSymbolRef(refs, sym)
					}
					out = appendString(out, sym.UnitName)
					cursor = member.End
					prevText = member.Text
					i += 2
					continue
				}
			}
		}
		if tok.Kind == scan.Ident && prevText != "." && !isCompositeKey(tokens, i) && !isStructFieldName(tokens, i) && !isLocalNameAt(localNames, tok.Text, tok.Start) {
			unitName := symbolNameTableUnitName(topNames, tok.Text)
			if unitName != "" {
				out = appendString(out, unitName)
				cursor = tok.End
				prevText = tok.Text
				continue
			}
		}
		out = appendBytes(out, file.Source[tok.Start:tok.End])
		cursor = tok.End
		prevText = tok.Text
	}
	if cursor < end {
		out = appendBytes(out, file.Source[cursor:end])
	}
	return string(out)
}

func appendUnitSymbolRef(refs *[]unit.Symbol, sym unit.Symbol) {
	values := *refs
	values = append(values, sym)
	*refs = values
}

func isCompositeKey(toks []scan.Token, pos int) bool {
	return pos+1 < len(toks) && toks[pos+1].Text == ":"
}

func isStructFieldName(toks []scan.Token, pos int) bool {
	if pos+1 >= len(toks) || !isTypeStartAfterName(toks, pos, len(toks)) {
		return false
	}
	if pos > 0 && toks[pos-1].Text == "type" {
		return false
	}
	if !startsStructFieldName(toks, pos) {
		return false
	}
	depth := 0
	for i := pos - 1; i >= 0; i-- {
		if toks[i].Text == "}" {
			depth++
			continue
		}
		if toks[i].Text == "{" {
			if depth == 0 {
				return i > 0 && toks[i-1].Text == "struct"
			}
			depth--
		}
	}
	return false
}

func startsStructFieldName(toks []scan.Token, pos int) bool {
	if pos <= 0 {
		return false
	}
	prev := toks[pos-1]
	if prev.Text == "{" || prev.Text == ";" || prev.Text == "," {
		return true
	}
	return prev.Line != toks[pos].Line
}

func methodLookupName(receiver localTypeInfo, methodName string) string {
	name := receiver.name + "_" + methodName
	if receiver.qualifier != "" {
		return receiver.qualifier + "." + name
	}
	return name
}

func appendMethodDeclPrefix(file *parse.File, decl *parse.Decl, topNames symbolNameTable, out *[]byte) int {
	toks := file.Tokens
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 || start+1 >= len(toks) || toks[start+1].Text != "(" {
		return decl.Start
	}
	receiverOpen := start + 1
	receiverClose := findClose(toks, receiverOpen, "(", ")")
	if receiverClose < 0 || receiverClose+2 >= len(toks) {
		return decl.Start
	}
	nameTok := receiverClose + 1
	paramsOpen := nameTok + 1
	if toks[nameTok].Kind != scan.Ident || toks[paramsOpen].Text != "(" {
		return decl.Start
	}
	paramsClose := findClose(toks, paramsOpen, "(", ")")
	if paramsClose < 0 {
		return decl.Start
	}
	unitName := symbolNameTableUnitName(topNames, methodDeclNameFromTokens(toks, decl))
	if unitName == "" {
		return decl.Start
	}
	appendStringRef(out, "func ")
	appendStringRef(out, unitName)
	appendByteRef(out, '(')
	appendStringRef(out, rewriteReceiverSegment(file, receiverOpen+1, receiverClose, topNames))
	if paramsOpen+1 < paramsClose {
		appendStringRef(out, ", ")
		return toks[paramsOpen].End
	}
	return toks[paramsClose].Start
}

func appendStringRef(out *[]byte, s string) {
	values := *out
	values = appendString(values, s)
	*out = values
}

func appendByteRef(out *[]byte, c byte) {
	values := *out
	values = append(values, c)
	*out = values
}

func rewriteReceiverSegment(file *parse.File, start int, end int, topNames symbolNameTable) string {
	toks := file.Tokens
	var out []byte
	cursor := toks[start].Start
	for i := start; i < end; i++ {
		tok := toks[i]
		if tok.Start > cursor {
			out = appendBytes(out, file.Source[cursor:tok.Start])
		}
		if tok.Kind == scan.Ident && (i > start || !receiverSegmentHasName(toks, start, end)) {
			unitName := symbolNameTableUnitName(topNames, tok.Text)
			if unitName != "" {
				out = appendString(out, unitName)
				cursor = tok.End
				continue
			}
		}
		out = appendBytes(out, file.Source[tok.Start:tok.End])
		cursor = tok.End
	}
	if end > start && cursor < toks[end-1].End {
		out = appendBytes(out, file.Source[cursor:toks[end-1].End])
	}
	return string(out)
}

func receiverSegmentHasName(toks []scan.Token, start int, end int) bool {
	idents := 0
	for i := start; i < end; i++ {
		if toks[i].Kind == scan.Ident {
			idents++
		}
	}
	return idents > 1
}

type expressionTemp struct {
	name string
	expr string
}

type expressionReplacement struct {
	start int
	end   int
	text  string
}

type expressionStatement struct {
	token     int
	exprStart int
	exprEnd   int
	kind      string
	openBrace int
}

type conditionalShortStatement struct {
	token     int
	initStart int
	semi      int
	condStart int
	condEnd   int
	openBrace int
	end       int
	kind      string
}

type classicForConditionStatement struct {
	token     int
	condStart int
	condEnd   int
	openBrace int
}

type classicForPostStatement struct {
	token     int
	initStart int
	initEnd   int
	condStart int
	condEnd   int
	postStart int
	postEnd   int
	openBrace int
	end       int
}

type shortCircuitIfStatement struct {
	token     int
	condStart int
	condEnd   int
	openBrace int
	end       int
	operands  []expressionRange
}

func normalizeFunctionExpressions(body string, unitName string) string {
	tempIndex := 0
	return normalizeFunctionExpressionsWithTemp(body, unitName, &tempIndex)
}

func normalizeFunctionExpressionsWithTemp(body string, unitName string, tempIndex *int) string {
	toks, err := scan.Tokens([]byte(body))
	if err != nil {
		return body
	}
	if !tokenSpansMatchSource(body, toks) {
		return body
	}
	var out []byte
	cursor := 0
	for i := 0; i < len(toks); i++ {
		short, ok := normalizationConditionalShortStatement(toks, i)
		if ok {
			initTemps, initReplacements := normalizeExpression(body, toks, short.initStart, short.semi, unitName, tempIndex)
			condTemps, condReplacements := normalizeExpression(body, toks, short.condStart, short.condEnd, unitName, tempIndex)
			insertStart := statementInsertStart(body, toks[short.token].Start)
			out = appendString(out, body[cursor:insertStart])
			indent := statementIndent(body, toks[short.token].Start)
			innerIndent := indent + "\t"
			init := strings.TrimSpace(applyExpressionReplacements(body, toks[short.initStart].Start, toks[short.semi-1].End, initReplacements))
			condition := strings.TrimSpace(applyExpressionReplacements(body, toks[short.condStart].Start, toks[short.condEnd-1].End, condReplacements))
			out = appendString(out, indent)
			out = appendString(out, "{\n")
			for j := 0; j < len(initTemps); j++ {
				temp := initTemps[j]
				out = appendString(out, innerIndent)
				out = appendString(out, temp.name)
				out = appendString(out, " := ")
				out = appendString(out, temp.expr)
				out = append(out, '\n')
			}
			out = appendString(out, innerIndent)
			out = appendString(out, init)
			out = append(out, '\n')
			for j := 0; j < len(condTemps); j++ {
				temp := condTemps[j]
				out = appendString(out, innerIndent)
				out = appendString(out, temp.name)
				out = appendString(out, " := ")
				out = appendString(out, temp.expr)
				out = append(out, '\n')
			}
			out = appendString(out, innerIndent)
			out = appendString(out, short.kind)
			out = append(out, ' ')
			out = appendString(out, condition)
			out = append(out, ' ')
			out = appendString(out, body[toks[short.openBrace].Start:toks[short.end].End])
			out = append(out, '\n')
			out = appendString(out, indent)
			out = append(out, '}')
			cursor = toks[short.end].End
			i = short.end
			continue
		}
		post, ok := normalizationClassicForPostStatement(toks, i)
		if ok {
			if expressionContainsCall(toks, post.postStart, post.postEnd) {
				initTemps, initReplacements := normalizeExpression(body, toks, post.initStart, post.initEnd, unitName, tempIndex)
				condTemps, condReplacements := normalizeExpression(body, toks, post.condStart, post.condEnd, unitName, tempIndex)
				postTemps, postReplacements := normalizeExpression(body, toks, post.postStart, post.postEnd, unitName, tempIndex)
				insertStart := statementInsertStart(body, toks[post.token].Start)
				out = appendString(out, body[cursor:insertStart])
				indent := statementIndent(body, toks[post.token].Start)
				innerIndent := indent + "\t"
				loopIndent := innerIndent + "\t"
				out = appendString(out, indent)
				out = appendString(out, "{\n")
				for j := 0; j < len(initTemps); j++ {
					temp := initTemps[j]
					out = appendString(out, innerIndent)
					out = appendString(out, temp.name)
					out = appendString(out, " := ")
					out = appendString(out, temp.expr)
					out = append(out, '\n')
				}
				if post.initStart < post.initEnd {
					init := strings.TrimSpace(applyExpressionReplacements(body, toks[post.initStart].Start, toks[post.initEnd-1].End, initReplacements))
					out = appendString(out, innerIndent)
					out = appendString(out, init)
					out = append(out, '\n')
				}
				out = appendString(out, innerIndent)
				out = appendString(out, "for {\n")
				if post.condStart < post.condEnd {
					condition := strings.TrimSpace(applyExpressionReplacements(body, toks[post.condStart].Start, toks[post.condEnd-1].End, condReplacements))
					for j := 0; j < len(condTemps); j++ {
						temp := condTemps[j]
						out = appendString(out, loopIndent)
						out = appendString(out, temp.name)
						out = appendString(out, " := ")
						out = appendString(out, temp.expr)
						out = append(out, '\n')
					}
					out = appendString(out, loopIndent)
					out = appendString(out, "if !(")
					out = appendString(out, condition)
					out = appendString(out, ") {\n")
					out = appendString(out, loopIndent)
					out = appendString(out, "\tbreak\n")
					out = appendString(out, loopIndent)
					out = appendString(out, "}\n")
				}
				out = appendString(out, body[toks[post.openBrace].End:toks[post.end].Start])
				if len(out) == 0 || out[len(out)-1] != '\n' {
					out = append(out, '\n')
				}
				for j := 0; j < len(postTemps); j++ {
					temp := postTemps[j]
					out = appendString(out, loopIndent)
					out = appendString(out, temp.name)
					out = appendString(out, " := ")
					out = appendString(out, temp.expr)
					out = append(out, '\n')
				}
				postExpr := strings.TrimSpace(applyExpressionReplacements(body, toks[post.postStart].Start, toks[post.postEnd-1].End, postReplacements))
				out = appendString(out, loopIndent)
				out = appendString(out, postExpr)
				out = append(out, '\n')
				out = appendString(out, innerIndent)
				out = appendString(out, "}\n")
				out = appendString(out, indent)
				out = append(out, '}')
				cursor = toks[post.end].End
				i = post.end
				continue
			}
		}
		classic, ok := normalizationClassicForConditionStatement(toks, i)
		if ok {
			temps, replacements := normalizeExpression(body, toks, classic.condStart, classic.condEnd, unitName, tempIndex)
			if len(temps) > 0 {
				condition := applyExpressionReplacements(body, toks[classic.condStart].Start, toks[classic.condEnd-1].End, replacements)
				out = appendString(out, body[cursor:toks[classic.condStart].Start])
				out = appendString(out, body[toks[classic.condEnd].Start:toks[classic.openBrace].End])
				indent := statementIndent(body, toks[classic.token].Start)
				innerIndent := indent + "\t"
				out = append(out, '\n')
				for j := 0; j < len(temps); j++ {
					temp := temps[j]
					out = appendString(out, innerIndent)
					out = appendString(out, temp.name)
					out = appendString(out, " := ")
					out = appendString(out, temp.expr)
					out = append(out, '\n')
				}
				out = appendString(out, innerIndent)
				out = appendString(out, "if !(")
				out = appendString(out, condition)
				out = appendString(out, ") {\n")
				out = appendString(out, innerIndent)
				out = appendString(out, "\tbreak\n")
				out = appendString(out, innerIndent)
				out = appendString(out, "}\n")
				cursor = toks[classic.openBrace].End
				i = classic.openBrace
				continue
			}
		}
		shortCircuit, ok := normalizationShortCircuitIfStatement(toks, i)
		if ok {
			insertStart := statementInsertStart(body, toks[shortCircuit.token].Start)
			out = appendString(out, body[cursor:insertStart])
			indent := statementIndent(body, toks[shortCircuit.token].Start)
			appendShortCircuitIf(&out, body, toks, shortCircuit, indent, unitName, tempIndex)
			cursor = toks[shortCircuit.end].End
			i = shortCircuit.end
			continue
		}
		stmt, ok := normalizationStatement(toks, i)
		if !ok {
			continue
		}
		temps, replacements := normalizeExpression(body, toks, stmt.exprStart, stmt.exprEnd, unitName, tempIndex)
		if len(temps) == 0 {
			continue
		}
		insertStart := statementInsertStart(body, toks[stmt.token].Start)
		out = appendString(out, body[cursor:insertStart])
		indent := statementIndent(body, toks[stmt.token].Start)
		if stmt.kind == "for-condition" {
			innerIndent := indent + "\t"
			condition := applyExpressionReplacements(body, toks[stmt.exprStart].Start, toks[stmt.exprEnd-1].End, replacements)
			out = appendString(out, body[insertStart:toks[stmt.token].Start])
			out = appendString(out, "for {\n")
			for j := 0; j < len(temps); j++ {
				temp := temps[j]
				out = appendString(out, innerIndent)
				out = appendString(out, temp.name)
				out = appendString(out, " := ")
				out = appendString(out, temp.expr)
				out = append(out, '\n')
			}
			out = appendString(out, innerIndent)
			out = appendString(out, "if !(")
			out = appendString(out, condition)
			out = appendString(out, ") {\n")
			out = appendString(out, innerIndent)
			out = appendString(out, "\tbreak\n")
			out = appendString(out, innerIndent)
			out = appendString(out, "}\n")
			cursor = toks[stmt.openBrace].End
			i = stmt.openBrace
			continue
		}
		if insertStart == toks[stmt.token].Start {
			out = append(out, '\n')
		}
		for j := 0; j < len(temps); j++ {
			temp := temps[j]
			out = appendString(out, indent)
			out = appendString(out, temp.name)
			out = appendString(out, " := ")
			out = appendString(out, temp.expr)
			out = append(out, '\n')
		}
		out = appendString(out, body[insertStart:toks[stmt.exprStart].Start])
		out = appendString(out, applyExpressionReplacements(body, toks[stmt.exprStart].Start, toks[stmt.exprEnd-1].End, replacements))
		cursor = toks[stmt.exprEnd-1].End
		i = stmt.exprEnd - 1
	}
	if len(out) == 0 {
		return body
	}
	out = appendString(out, body[cursor:])
	if strings.HasPrefix(strings.TrimSpace(body), "func ") && !trimmedBytesHavePrefix(out, "func ") {
		return body
	}
	return string(out)
}

func appendShortCircuitIf(out *[]byte, body string, toks []scan.Token, stmt shortCircuitIfStatement, indent string, unitName string, tempIndex *int) {
	currentIndent := indent
	for i := 0; i < len(stmt.operands); i++ {
		operand := stmt.operands[i]
		temps, replacements := normalizeExpression(body, toks, operand.start, operand.end, unitName, tempIndex)
		for j := 0; j < len(temps); j++ {
			temp := temps[j]
			appendStringRef(out, currentIndent)
			appendStringRef(out, temp.name)
			appendStringRef(out, " := ")
			appendStringRef(out, temp.expr)
			appendByteRef(out, '\n')
		}
		condition := strings.TrimSpace(applyExpressionReplacements(body, toks[operand.start].Start, toks[operand.end-1].End, replacements))
		appendStringRef(out, currentIndent)
		appendStringRef(out, "if ")
		appendStringRef(out, condition)
		appendStringRef(out, " {\n")
		currentIndent = currentIndent + "\t"
	}
	innerBody := normalizeFunctionExpressionsWithTemp(body[toks[stmt.openBrace].End:toks[stmt.end].Start], unitName, tempIndex)
	appendStringRef(out, innerBody)
	if len(*out) == 0 || (*out)[len(*out)-1] != '\n' {
		appendByteRef(out, '\n')
	}
	for i := len(stmt.operands) - 1; i >= 0; i-- {
		currentIndent = currentIndent[:len(currentIndent)-1]
		appendStringRef(out, currentIndent)
		appendByteRef(out, '}')
		if i > 0 {
			appendByteRef(out, '\n')
		}
	}
}

func trimmedBytesHavePrefix(body []byte, prefix string) bool {
	start := 0
	for start < len(body) && (body[start] == ' ' || body[start] == '\t' || body[start] == '\r' || body[start] == '\n') {
		start++
	}
	if start+len(prefix) > len(body) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if body[start+i] != prefix[i] {
			return false
		}
	}
	return true
}

func tokenSpansMatchSource(body string, toks []scan.Token) bool {
	for i := 0; i < len(toks); i++ {
		tok := toks[i]
		if tok.Start < 0 || tok.End < tok.Start || tok.End > len(body) {
			return false
		}
		if body[tok.Start:tok.End] != tok.Text {
			return false
		}
	}
	return true
}

func normalizationConditionalShortStatement(toks []scan.Token, pos int) (conditionalShortStatement, bool) {
	if toks[pos].Text != "if" && toks[pos].Text != "switch" {
		return conditionalShortStatement{}, false
	}
	exprStart := pos + 1
	exprEnd := conditionExpressionEnd(toks, pos)
	if exprEnd <= exprStart || exprEnd >= len(toks) || toks[exprEnd].Text != "{" {
		return conditionalShortStatement{}, false
	}
	semi := topLevelSemicolon(toks, exprStart, exprEnd)
	if semi < 0 || semi <= exprStart || semi+1 >= exprEnd {
		return conditionalShortStatement{}, false
	}
	end := conditionalStatementEnd(toks, pos, exprEnd)
	if end <= exprEnd {
		return conditionalShortStatement{}, false
	}
	posTok := toks[pos]
	var stmt conditionalShortStatement
	stmt.token = pos
	stmt.initStart = exprStart
	stmt.semi = semi
	stmt.condStart = semi + 1
	stmt.condEnd = exprEnd
	stmt.openBrace = exprEnd
	stmt.end = end
	stmt.kind = posTok.Text
	return stmt, true
}

func normalizationShortCircuitIfStatement(toks []scan.Token, pos int) (shortCircuitIfStatement, bool) {
	if toks[pos].Text != "if" {
		return shortCircuitIfStatement{}, false
	}
	exprStart := pos + 1
	exprEnd := conditionExpressionEnd(toks, pos)
	if exprEnd <= exprStart || exprEnd >= len(toks) || toks[exprEnd].Text != "{" {
		return shortCircuitIfStatement{}, false
	}
	if expressionContainsTopLevelSemicolon(toks, exprStart, exprEnd) {
		return shortCircuitIfStatement{}, false
	}
	closeBrace := findClose(toks, exprEnd, "{", "}")
	if closeBrace <= exprEnd {
		return shortCircuitIfStatement{}, false
	}
	if closeBrace+1 < len(toks) && toks[closeBrace+1].Text == "else" {
		return shortCircuitIfStatement{}, false
	}
	operands, ok := topLevelAndOperands(toks, exprStart, exprEnd)
	if !ok || len(operands) < 2 {
		return shortCircuitIfStatement{}, false
	}
	var stmt shortCircuitIfStatement
	stmt.token = pos
	stmt.condStart = exprStart
	stmt.condEnd = exprEnd
	stmt.openBrace = exprEnd
	stmt.end = closeBrace
	stmt.operands = operands
	return stmt, true
}

func normalizationClassicForPostStatement(toks []scan.Token, pos int) (classicForPostStatement, bool) {
	if toks[pos].Text != "for" {
		return classicForPostStatement{}, false
	}
	exprStart := pos + 1
	exprEnd := conditionExpressionEnd(toks, pos)
	if exprEnd <= exprStart || exprEnd >= len(toks) || toks[exprEnd].Text != "{" {
		return classicForPostStatement{}, false
	}
	firstSemi := topLevelSemicolon(toks, exprStart, exprEnd)
	if firstSemi < 0 {
		return classicForPostStatement{}, false
	}
	secondSemi := topLevelSemicolon(toks, firstSemi+1, exprEnd)
	if secondSemi < 0 || secondSemi+1 >= exprEnd {
		return classicForPostStatement{}, false
	}
	end := findClose(toks, exprEnd, "{", "}")
	if end <= exprEnd || containsTokenText(toks, exprEnd+1, end, "continue") {
		return classicForPostStatement{}, false
	}
	var stmt classicForPostStatement
	stmt.token = pos
	stmt.initStart = exprStart
	stmt.initEnd = firstSemi
	stmt.condStart = firstSemi + 1
	stmt.condEnd = secondSemi
	stmt.postStart = secondSemi + 1
	stmt.postEnd = exprEnd
	stmt.openBrace = exprEnd
	stmt.end = end
	return stmt, true
}

func normalizationClassicForConditionStatement(toks []scan.Token, pos int) (classicForConditionStatement, bool) {
	if toks[pos].Text != "for" {
		return classicForConditionStatement{}, false
	}
	exprStart := pos + 1
	exprEnd := conditionExpressionEnd(toks, pos)
	if exprEnd <= exprStart || exprEnd >= len(toks) || toks[exprEnd].Text != "{" {
		return classicForConditionStatement{}, false
	}
	firstSemi := topLevelSemicolon(toks, exprStart, exprEnd)
	if firstSemi < 0 {
		return classicForConditionStatement{}, false
	}
	secondSemi := topLevelSemicolon(toks, firstSemi+1, exprEnd)
	if secondSemi < 0 || secondSemi <= firstSemi+1 {
		return classicForConditionStatement{}, false
	}
	var stmt classicForConditionStatement
	stmt.token = pos
	stmt.condStart = firstSemi + 1
	stmt.condEnd = secondSemi
	stmt.openBrace = exprEnd
	return stmt, true
}

func normalizationStatement(toks []scan.Token, pos int) (expressionStatement, bool) {
	if toks[pos].Text == "return" {
		exprStart := pos + 1
		exprEnd := lineExpressionEnd(toks, pos)
		if exprEnd <= exprStart {
			return expressionStatement{}, false
		}
		return expressionStatement{token: pos, exprStart: exprStart, exprEnd: exprEnd}, true
	}
	if toks[pos].Text == "if" {
		exprStart := pos + 1
		exprEnd := conditionExpressionEnd(toks, pos)
		if exprEnd <= exprStart {
			return expressionStatement{}, false
		}
		if expressionContainsTopLevelSemicolon(toks, exprStart, exprEnd) {
			assign, semi := shortHeaderInitAssignment(toks, exprStart, exprEnd)
			if assign < 0 || semi <= assign+1 {
				return expressionStatement{}, false
			}
			return expressionStatement{token: pos, exprStart: assign + 1, exprEnd: semi}, true
		}
		return expressionStatement{token: pos, exprStart: exprStart, exprEnd: exprEnd}, true
	}
	if toks[pos].Text == "switch" {
		exprStart := pos + 1
		exprEnd := conditionExpressionEnd(toks, pos)
		if exprEnd <= exprStart {
			return expressionStatement{}, false
		}
		if expressionContainsTopLevelSemicolon(toks, exprStart, exprEnd) {
			assign, semi := shortHeaderInitAssignment(toks, exprStart, exprEnd)
			if assign < 0 || semi <= assign+1 {
				return expressionStatement{}, false
			}
			return expressionStatement{token: pos, exprStart: assign + 1, exprEnd: semi}, true
		}
		return expressionStatement{token: pos, exprStart: exprStart, exprEnd: exprEnd}, true
	}
	if toks[pos].Text == "for" {
		exprStart := pos + 1
		exprEnd := conditionExpressionEnd(toks, pos)
		if exprEnd <= exprStart {
			return expressionStatement{}, false
		}
		if expressionContainsTopLevelSemicolon(toks, exprStart, exprEnd) {
			assign, semi := classicForInitAssignment(toks, exprStart, exprEnd)
			if assign < 0 || semi <= assign+1 {
				return expressionStatement{}, false
			}
			return expressionStatement{token: pos, exprStart: assign + 1, exprEnd: semi}, true
		}
		return expressionStatement{token: pos, exprStart: exprStart, exprEnd: exprEnd, kind: "for-condition", openBrace: exprEnd}, true
	}
	if isInsideClassicForHeader(toks, pos) {
		return expressionStatement{}, false
	}
	if isInsideConditionalShortHeader(toks, pos) {
		return expressionStatement{}, false
	}
	if startsCallStatement(toks, pos) {
		exprEnd := lineExpressionEnd(toks, pos)
		if exprEnd <= pos {
			return expressionStatement{}, false
		}
		return expressionStatement{token: pos, exprStart: pos, exprEnd: exprEnd}, true
	}
	if toks[pos].Text == "var" {
		exprStart := varInitializerStart(toks, pos)
		if exprStart < 0 {
			return expressionStatement{}, false
		}
		exprEnd := lineExpressionEnd(toks, exprStart-1)
		if exprEnd <= exprStart {
			return expressionStatement{}, false
		}
		return expressionStatement{token: pos, exprStart: exprStart, exprEnd: exprEnd}, true
	}
	if !isAssignmentOperator(toks[pos].Text) {
		return expressionStatement{}, false
	}
	if isClassicForHeaderAssignment(toks, pos) {
		return expressionStatement{}, false
	}
	exprStart := pos + 1
	exprEnd := lineExpressionEnd(toks, pos)
	if exprEnd <= exprStart {
		return expressionStatement{}, false
	}
	stmtStart := statementStartToken(toks, pos)
	return expressionStatement{token: stmtStart, exprStart: exprStart, exprEnd: exprEnd}, true
}

func conditionExpressionEnd(toks []scan.Token, start int) int {
	paren := 0
	brack := 0
	brace := 0
	for i := start + 1; i < len(toks); i++ {
		tok := toks[i]
		if tok.Kind == scan.EOF {
			return i
		}
		if paren == 0 && brack == 0 && brace == 0 && tok.Text == "{" {
			return i
		}
		updateExpressionDepth(tok.Text, &paren, &brack, &brace)
	}
	return len(toks)
}

func lineExpressionEnd(toks []scan.Token, start int) int {
	paren := 0
	brack := 0
	brace := 0
	for i := start + 1; i < len(toks); i++ {
		tok := toks[i]
		if tok.Kind == scan.EOF {
			return i
		}
		if paren == 0 && brack == 0 && brace == 0 {
			if tok.Text == "}" || tok.Text == ";" {
				return i
			}
			if tok.Line != toks[start].Line {
				return i
			}
		}
		updateExpressionDepth(tok.Text, &paren, &brack, &brace)
	}
	return len(toks)
}

func statementStartToken(toks []scan.Token, pos int) int {
	line := toks[pos].Line
	for i := pos - 1; i >= 0; i-- {
		if toks[i].Line != line || toks[i].Text == ";" || toks[i].Text == "{" || toks[i].Text == "}" {
			return i + 1
		}
	}
	return 0
}

func isAssignmentOperator(text string) bool {
	return text == "=" || text == ":="
}

func isClassicForHeaderAssignment(toks []scan.Token, pos int) bool {
	start := statementStartToken(toks, pos)
	if start < len(toks) && toks[start].Text == "for" {
		exprEnd := conditionExpressionEnd(toks, start)
		return expressionContainsTopLevelSemicolon(toks, start+1, exprEnd)
	}
	return isForPostClauseAssignment(toks, pos)
}

func varInitializerStart(toks []scan.Token, pos int) int {
	if pos+1 < len(toks) && toks[pos+1].Text == "(" {
		return -1
	}
	paren := 0
	brack := 0
	brace := 0
	for i := pos + 1; i < len(toks) && toks[i].Line == toks[pos].Line; i++ {
		if paren == 0 && brack == 0 && brace == 0 && toks[i].Text == "=" {
			return i + 1
		}
		updateExpressionDepth(toks[i].Text, &paren, &brack, &brace)
	}
	return -1
}

func isForPostClauseAssignment(toks []scan.Token, pos int) bool {
	semi := -1
	for i := pos - 1; i >= 0 && toks[i].Line == toks[pos].Line; i-- {
		if toks[i].Text == "{" || toks[i].Text == "}" {
			return false
		}
		if toks[i].Text == ";" {
			semi = i
			break
		}
	}
	if semi < 0 {
		return false
	}
	for i := semi - 1; i >= 0 && toks[i].Line == toks[pos].Line; i-- {
		if toks[i].Text == "{" || toks[i].Text == "}" {
			return false
		}
		if toks[i].Text == "for" {
			return true
		}
	}
	return false
}

func isInsideClassicForHeader(toks []scan.Token, pos int) bool {
	for i := pos - 1; i >= 0 && toks[i].Line == toks[pos].Line; i-- {
		if toks[i].Text == "{" || toks[i].Text == "}" {
			return false
		}
		if toks[i].Text != "for" {
			continue
		}
		exprEnd := conditionExpressionEnd(toks, i)
		return pos < exprEnd && expressionContainsTopLevelSemicolon(toks, i+1, exprEnd)
	}
	return false
}

func classicForInitAssignment(toks []scan.Token, start int, end int) (int, int) {
	return shortHeaderInitAssignment(toks, start, end)
}

func shortHeaderInitAssignment(toks []scan.Token, start int, end int) (int, int) {
	paren := 0
	brack := 0
	brace := 0
	assign := -1
	for i := start; i < end; i++ {
		if paren == 0 && brack == 0 && brace == 0 {
			if toks[i].Text == ";" {
				return assign, i
			}
			if assign < 0 && isAssignmentOperator(toks[i].Text) {
				assign = i
			}
		}
		updateExpressionDepth(toks[i].Text, &paren, &brack, &brace)
	}
	return -1, -1
}

func isInsideConditionalShortHeader(toks []scan.Token, pos int) bool {
	for i := pos - 1; i >= 0; i-- {
		if toks[i].Text == "{" || toks[i].Text == "}" {
			return false
		}
		if toks[i].Text != "if" && toks[i].Text != "switch" {
			continue
		}
		exprEnd := conditionExpressionEnd(toks, i)
		return pos < exprEnd && expressionContainsTopLevelSemicolon(toks, i+1, exprEnd)
	}
	return false
}

func startsCallStatement(toks []scan.Token, pos int) bool {
	if pos+1 >= len(toks) || toks[pos].Kind != scan.Ident || toks[pos+1].Text != "(" {
		return false
	}
	return statementStartToken(toks, pos) == pos
}

func normalizeExpression(body string, toks []scan.Token, start int, end int, unitName string, tempIndex *int) ([]expressionTemp, []expressionReplacement) {
	var temps []expressionTemp
	var replacements []expressionReplacement
	paren := 0
	brack := 0
	brace := 0
	for i := start; i+1 < end; i++ {
		tok := toks[i]
		if paren == 0 && brack == 0 && brace == 0 && tok.Kind == scan.Ident && toks[i+1].Text == "(" {
			close := findClose(toks, i+1, "(", ")")
			if close > i+1 && close < end {
				callTemps, callReplacements := normalizeOneCallArguments(body, toks, i+2, close, unitName, tempIndex)
				temps = appendExpressionTemps(temps, callTemps)
				replacements = appendExpressionReplacements(replacements, callReplacements)
				i = close
				continue
			}
		}
		if paren == 0 && brack == 0 && brace == 0 && tok.Text == "[" {
			close := findClose(toks, i, "[", "]")
			if close > i && close < end {
				indexTemps, indexReplacements := normalizeIndexBounds(body, toks, i+1, close, unitName, tempIndex)
				temps = appendExpressionTemps(temps, indexTemps)
				replacements = appendExpressionReplacements(replacements, indexReplacements)
				i = close
				continue
			}
		}
		updateExpressionDepth(tok.Text, &paren, &brack, &brace)
	}
	return temps, replacements
}

func normalizeIndexBounds(body string, toks []scan.Token, start int, end int, unitName string, tempIndex *int) ([]expressionTemp, []expressionReplacement) {
	var temps []expressionTemp
	var replacements []expressionReplacement
	bounds := indexBoundRanges(toks, start, end)
	for i := 0; i < len(bounds); i++ {
		bound := bounds[i]
		if bound.start >= bound.end || !expressionContainsCall(toks, bound.start, bound.end) {
			continue
		}
		name := nextExpressionTempName(body, unitName, tempIndex)
		(*tempIndex)++
		exprStart := toks[bound.start].Start
		exprEnd := toks[bound.end-1].End
		temps = append(temps, expressionTemp{name: name, expr: body[exprStart:exprEnd]})
		replacements = append(replacements, expressionReplacement{start: exprStart, end: exprEnd, text: name})
	}
	return temps, replacements
}

type expressionRange struct {
	start int
	end   int
}

func topLevelAndOperands(toks []scan.Token, start int, end int) ([]expressionRange, bool) {
	var out []expressionRange
	operandStart := start
	paren := 0
	brack := 0
	brace := 0
	sawAnd := false
	for i := start; i < end; i++ {
		tok := toks[i]
		if paren == 0 && brack == 0 && brace == 0 {
			if tok.Text == "||" {
				return nil, false
			}
			if tok.Text == "&&" {
				if operandStart >= i {
					return nil, false
				}
				out = append(out, expressionRange{start: operandStart, end: i})
				operandStart = i + 1
				sawAnd = true
				continue
			}
		}
		updateExpressionDepth(tok.Text, &paren, &brack, &brace)
	}
	if !sawAnd || operandStart >= end {
		return nil, false
	}
	out = append(out, expressionRange{start: operandStart, end: end})
	return out, true
}

func indexBoundRanges(toks []scan.Token, start int, end int) []expressionRange {
	if start >= end {
		return nil
	}
	var out []expressionRange
	boundStart := start
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < end; i++ {
		if paren == 0 && brack == 0 && brace == 0 && toks[i].Text == ":" {
			out = append(out, expressionRange{start: boundStart, end: i})
			boundStart = i + 1
			continue
		}
		updateExpressionDepth(toks[i].Text, &paren, &brack, &brace)
	}
	out = append(out, expressionRange{start: boundStart, end: end})
	return out
}

func normalizeOneCallArguments(body string, toks []scan.Token, start int, end int, unitName string, tempIndex *int) ([]expressionTemp, []expressionReplacement) {
	var temps []expressionTemp
	var replacements []expressionReplacement
	argStart := start
	paren := 0
	brack := 0
	brace := 0
	for i := start; i <= end; i++ {
		if i == end || (paren == 0 && brack == 0 && brace == 0 && toks[i].Text == ",") {
			if argStart < i {
				if expressionStartsCompositeLiteral(toks, argStart, i) {
					argStart = i + 1
					continue
				}
				argTemps, argReplacements := normalizeExpression(body, toks, argStart, i, unitName, tempIndex)
				temps = appendExpressionTemps(temps, argTemps)
				exprStart := toks[argStart].Start
				exprEnd := toks[i-1].End
				expr := applyExpressionReplacements(body, exprStart, exprEnd, argReplacements)
				if !expressionContainsCall(toks, argStart, i) {
					replacements = appendExpressionReplacements(replacements, argReplacements)
					argStart = i + 1
					continue
				}
				name := nextExpressionTempName(body, unitName, tempIndex)
				(*tempIndex)++
				temps = append(temps, expressionTemp{name: name, expr: expr})
				replacements = append(replacements, expressionReplacement{start: exprStart, end: exprEnd, text: name})
			}
			argStart = i + 1
			continue
		}
		updateExpressionDepth(toks[i].Text, &paren, &brack, &brace)
	}
	return temps, replacements
}

func expressionStartsCompositeLiteral(toks []scan.Token, start int, end int) bool {
	if start+1 >= end || toks[start].Kind != scan.Ident {
		return false
	}
	if toks[start+1].Text == "{" {
		return true
	}
	if start+3 < end && toks[start+1].Text == "." && toks[start+2].Kind == scan.Ident && toks[start+3].Text == "{" {
		return true
	}
	return false
}

func nextExpressionTempName(body string, unitName string, tempIndex *int) string {
	for {
		name := unitName + "_tmp_" + strconv.Itoa(*tempIndex)
		if !strings.Contains(body, name) {
			return name
		}
		(*tempIndex)++
	}
}

func updateExpressionDepth(text string, paren *int, brack *int, brace *int) {
	switch text {
	case "(":
		(*paren)++
	case ")":
		if *paren > 0 {
			(*paren)--
		}
	case "[":
		(*brack)++
	case "]":
		if *brack > 0 {
			(*brack)--
		}
	case "{":
		(*brace)++
	case "}":
		if *brace > 0 {
			(*brace)--
		}
	}
}

func expressionContainsCall(toks []scan.Token, start int, end int) bool {
	for i := start; i+1 < end; i++ {
		if toks[i].Kind == scan.Ident && toks[i+1].Text == "(" {
			return true
		}
	}
	return false
}

func expressionContainsTopLevelSemicolon(toks []scan.Token, start int, end int) bool {
	return topLevelSemicolon(toks, start, end) >= 0
}

func topLevelSemicolon(toks []scan.Token, start int, end int) int {
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < end; i++ {
		if paren == 0 && brack == 0 && brace == 0 && toks[i].Text == ";" {
			return i
		}
		updateExpressionDepth(toks[i].Text, &paren, &brack, &brace)
	}
	return -1
}

func conditionalStatementEnd(toks []scan.Token, pos int, openBrace int) int {
	closeBrace := findClose(toks, openBrace, "{", "}")
	if closeBrace < 0 {
		return -1
	}
	if toks[pos].Text != "if" {
		return closeBrace
	}
	if closeBrace+1 >= len(toks) || toks[closeBrace+1].Text != "else" {
		return closeBrace
	}
	next := closeBrace + 2
	if next >= len(toks) {
		return closeBrace
	}
	if toks[next].Text == "if" {
		nextOpen := conditionExpressionEnd(toks, next)
		if nextOpen >= len(toks) || toks[nextOpen].Text != "{" {
			return closeBrace
		}
		end := conditionalStatementEnd(toks, next, nextOpen)
		if end >= 0 {
			return end
		}
		return closeBrace
	}
	if toks[next].Text == "{" {
		end := findClose(toks, next, "{", "}")
		if end >= 0 {
			return end
		}
	}
	return closeBrace
}

func applyExpressionReplacements(body string, start int, end int, replacements []expressionReplacement) string {
	var out []byte
	cursor := start
	for i := 0; i < len(replacements); i++ {
		repl := replacements[i]
		if repl.start < cursor || repl.end > end {
			continue
		}
		out = appendString(out, body[cursor:repl.start])
		out = appendString(out, repl.text)
		cursor = repl.end
	}
	out = appendString(out, body[cursor:end])
	return string(out)
}

func statementIndent(body string, pos int) string {
	lineStart := pos
	for lineStart > 0 && body[lineStart-1] != '\n' {
		lineStart--
	}
	for i := lineStart; i < pos; i++ {
		if body[i] != ' ' && body[i] != '\t' {
			return "\t"
		}
	}
	indent := body[lineStart:pos]
	if indent == "" {
		return "\t"
	}
	return indent
}

func statementInsertStart(body string, pos int) int {
	lineStart := pos
	for lineStart > 0 && body[lineStart-1] != '\n' {
		lineStart--
	}
	for i := lineStart; i < pos; i++ {
		if body[i] != ' ' && body[i] != '\t' {
			return pos
		}
	}
	return lineStart
}

func localNamesForDecl(file *parse.File, decl *parse.Decl, namesOfInterest symbolNameTable) localNameTable {
	var names localNameTable
	if decl.Kind != "func" {
		return names
	}
	toks := file.Tokens
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 {
		return names
	}
	body := findTokenText(toks, start, decl.End, "{")
	if body < 0 {
		return names
	}
	collectFuncSignatureLocals(toks, start, body, namesOfInterest, &names)
	for i := body + 1; i < len(toks) && toks[i].Start < decl.End; i++ {
		if toks[i].Text == ":=" {
			collectShortDeclLocals(toks, body, i, decl.End, namesOfInterest, &names)
			continue
		}
		if toks[i].Text == "var" {
			collectVarLocals(toks, body, i, decl.End, namesOfInterest, &names)
		}
	}
	return names
}

func localTypesForDecl(file *parse.File, decl *parse.Decl) localTypeTable {
	var types localTypeTable
	if decl.Kind != "func" {
		return types
	}
	toks := file.Tokens
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 {
		return types
	}
	body := findTokenText(toks, start, decl.End, "{")
	if body < 0 {
		return types
	}
	collectFuncSignatureLocalTypes(toks, start, body, &types)
	for i := body + 1; i < len(toks) && toks[i].Start < decl.End; i++ {
		if toks[i].Text == ":=" {
			collectShortDeclLocalTypes(toks, i, &types)
			continue
		}
		if toks[i].Text == "var" {
			collectVarLocalTypes(toks, i, decl.End, &types)
		}
	}
	return types
}

func collectFuncSignatureLocalTypes(toks []scan.Token, start int, end int, types *localTypeTable) {
	for i := start; i < end; i++ {
		if toks[i].Text != "(" {
			continue
		}
		close := findClose(toks, i, "(", ")")
		if close < 0 || close > end {
			continue
		}
		collectParameterListLocalTypes(toks, i+1, close, types)
		i = close
	}
}

func collectParameterListLocalTypes(toks []scan.Token, start int, end int, types *localTypeTable) {
	for i := start; i < end; i++ {
		if toks[i].Kind != scan.Ident {
			continue
		}
		if i+1 < end && isTypeStart(toks[i+1]) {
			typeStart := i + 1
			typeEnd := parameterTypeEnd(toks, typeStart, end)
			typ := typeInfoInRange(toks, typeStart, typeEnd)
			if typ.name != "" {
				typ.pointer = typeRangeIsPointer(toks, typeStart, typeEnd)
				*types = localTypeTableSet(*types, toks[i].Text, typ)
			}
			continue
		}
		if i+2 < end && toks[i+1].Text == "," && toks[i+2].Kind == scan.Ident && isTypeStartAfterName(toks, i+2, end) {
			typeStart := i + 3
			for typeStart < end && toks[typeStart].Text == "," {
				typeStart++
			}
			typeEnd := parameterTypeEnd(toks, typeStart, end)
			typ := typeInfoInRange(toks, typeStart, typeEnd)
			if typ.name != "" {
				info := typ
				info.pointer = typeRangeIsPointer(toks, typeStart, typeEnd)
				*types = localTypeTableSet(*types, toks[i].Text, info)
				*types = localTypeTableSet(*types, toks[i+2].Text, info)
			}
		}
	}
}

func collectShortDeclLocalTypes(toks []scan.Token, assign int, types *localTypeTable) {
	if assign-1 < 0 || assign+2 >= len(toks) {
		return
	}
	if toks[assign-1].Kind != scan.Ident {
		return
	}
	if toks[assign+1].Kind == scan.Ident && toks[assign+2].Text == "{" {
		*types = localTypeTableSet(*types, toks[assign-1].Text, localTypeInfo{name: toks[assign+1].Text})
		return
	}
	if assign+4 < len(toks) && toks[assign+1].Kind == scan.Ident && toks[assign+2].Text == "." && toks[assign+3].Kind == scan.Ident && toks[assign+4].Text == "{" {
		*types = localTypeTableSet(*types, toks[assign-1].Text, localTypeInfo{qualifier: toks[assign+1].Text, name: toks[assign+3].Text})
		return
	}
	if assign+3 < len(toks) && toks[assign+1].Text == "&" && toks[assign+2].Kind == scan.Ident && toks[assign+3].Text == "{" {
		*types = localTypeTableSet(*types, toks[assign-1].Text, localTypeInfo{name: toks[assign+2].Text, pointer: true})
		return
	}
	if assign+5 < len(toks) && toks[assign+1].Text == "&" && toks[assign+2].Kind == scan.Ident && toks[assign+3].Text == "." && toks[assign+4].Kind == scan.Ident && toks[assign+5].Text == "{" {
		*types = localTypeTableSet(*types, toks[assign-1].Text, localTypeInfo{qualifier: toks[assign+2].Text, name: toks[assign+4].Text, pointer: true})
		return
	}
	if assign+2 < len(toks) && toks[assign+1].Text == "&" && toks[assign+2].Kind == scan.Ident {
		if pointed := localTypeTableLookup(*types, toks[assign+2].Text); pointed.name != "" {
			pointed.pointer = true
			*types = localTypeTableSet(*types, toks[assign-1].Text, pointed)
		}
	}
}

func collectVarLocalTypes(toks []scan.Token, pos int, end int, types *localTypeTable) {
	if pos+2 >= len(toks) || toks[pos+1].Kind != scan.Ident {
		return
	}
	if toks[pos+2].Text == "=" {
		if pos+4 < len(toks) && toks[pos+3].Kind == scan.Ident && toks[pos+4].Text == "{" {
			*types = localTypeTableSet(*types, toks[pos+1].Text, localTypeInfo{name: toks[pos+3].Text})
		}
		if pos+6 < len(toks) && toks[pos+3].Kind == scan.Ident && toks[pos+4].Text == "." && toks[pos+5].Kind == scan.Ident && toks[pos+6].Text == "{" {
			*types = localTypeTableSet(*types, toks[pos+1].Text, localTypeInfo{qualifier: toks[pos+3].Text, name: toks[pos+5].Text})
		}
		if pos+5 < len(toks) && toks[pos+3].Text == "&" && toks[pos+4].Kind == scan.Ident && toks[pos+5].Text == "{" {
			*types = localTypeTableSet(*types, toks[pos+1].Text, localTypeInfo{name: toks[pos+4].Text, pointer: true})
		}
		if pos+7 < len(toks) && toks[pos+3].Text == "&" && toks[pos+4].Kind == scan.Ident && toks[pos+5].Text == "." && toks[pos+6].Kind == scan.Ident && toks[pos+7].Text == "{" {
			*types = localTypeTableSet(*types, toks[pos+1].Text, localTypeInfo{qualifier: toks[pos+4].Text, name: toks[pos+6].Text, pointer: true})
		}
		return
	}
	typeStart := pos + 2
	typeEnd := varTypeEnd(toks, typeStart, end)
	typ := typeInfoInRange(toks, typeStart, typeEnd)
	if typ.name != "" {
		typ.pointer = typeRangeIsPointer(toks, typeStart, typeEnd)
		*types = localTypeTableSet(*types, toks[pos+1].Text, typ)
	}
}

func parameterTypeEnd(toks []scan.Token, start int, end int) int {
	for i := start; i < end; i++ {
		if toks[i].Text == "," || toks[i].Text == ")" || toks[i].Text == "{" || toks[i].Text == "=" || toks[i].Text == ";" {
			return i
		}
	}
	return end
}

func varTypeEnd(toks []scan.Token, start int, end int) int {
	line := toks[start].Line
	for i := start; i < end; i++ {
		if toks[i].Line != line {
			return i
		}
		if toks[i].Text == "," || toks[i].Text == ")" || toks[i].Text == "{" || toks[i].Text == "=" || toks[i].Text == ";" {
			return i
		}
	}
	return end
}

func typeNameInRange(toks []scan.Token, start int, end int) string {
	info := typeInfoInRange(toks, start, end)
	return info.name
}

func typeInfoInRange(toks []scan.Token, start int, end int) localTypeInfo {
	name := ""
	qualifier := ""
	for i := start; i < end; i++ {
		if i+2 < end && toks[i].Kind == scan.Ident && toks[i+1].Text == "." && toks[i+2].Kind == scan.Ident {
			qualifier = toks[i].Text
			name = toks[i+2].Text
			i += 2
			continue
		}
		if toks[i].Kind == scan.Ident {
			qualifier = ""
			name = toks[i].Text
		}
	}
	return localTypeInfo{qualifier: qualifier, name: name}
}

func typeRangeIsPointer(toks []scan.Token, start int, end int) bool {
	return containsTokenText(toks, start, end, "*")
}

func isLocalNameAt(names localNameTable, name string, pos int) bool {
	for i := 0; i < len(names); i++ {
		scope := names[i]
		if scope.name == name && pos >= scope.start && pos < scope.end {
			return true
		}
	}
	return false
}

func collectFuncSignatureLocals(toks []scan.Token, start int, end int, topNames symbolNameTable, names *localNameTable) {
	for i := start; i < end; i++ {
		if toks[i].Text != "(" {
			continue
		}
		close := findClose(toks, i, "(", ")")
		if close < 0 || close > end {
			continue
		}
		collectParameterListLocals(toks, i+1, close, topNames, names)
		i = close
	}
}

func collectParameterListLocals(toks []scan.Token, start int, end int, topNames symbolNameTable, names *localNameTable) {
	for i := start; i < end; i++ {
		if toks[i].Kind != scan.Ident || symbolNameTableUnitName(topNames, toks[i].Text) == "" {
			continue
		}
		if i > start && toks[i-1].Text != "," {
			continue
		}
		if i+1 < end && isTypeStart(toks[i+1]) {
			addLocalName(names, toks[i].Text, 0, maxSourcePosition())
			continue
		}
		if i+2 < end && toks[i+1].Text == "," && toks[i+2].Kind == scan.Ident && isTypeStartAfterName(toks, i+2, end) {
			addLocalName(names, toks[i].Text, 0, maxSourcePosition())
		}
	}
}

func collectShortDeclLocals(toks []scan.Token, body int, assign int, declEnd int, topNames symbolNameTable, names *localNameTable) {
	line := toks[assign].Line
	scopeEnd := localScopeEnd(toks, body, assign, declEnd)
	for i := assign - 1; i >= 0; i-- {
		if toks[i].Line != line {
			return
		}
		if isStatementBoundary(toks[i].Text) {
			return
		}
		if toks[i].Kind == scan.Ident && symbolNameTableUnitName(topNames, toks[i].Text) != "" && (i == 0 || toks[i-1].Text != ".") {
			addLocalName(names, toks[i].Text, toks[i].Start, scopeEnd)
		}
	}
}

func collectVarLocals(toks []scan.Token, body int, pos int, end int, topNames symbolNameTable, names *localNameTable) {
	scopeEnd := localScopeEnd(toks, body, pos, end)
	if pos+1 < len(toks) && toks[pos+1].Text == "(" {
		for i := pos + 2; i < len(toks) && toks[i].Start < end; i++ {
			if toks[i].Text == ")" || toks[i].Text == "}" {
				return
			}
			if toks[i].Kind != scan.Ident || symbolNameTableUnitName(topNames, toks[i].Text) == "" {
				continue
			}
			if toks[i-1].Text == "(" || toks[i-1].Text == "," || toks[i-1].Line != toks[i].Line {
				addLocalName(names, toks[i].Text, toks[i].Start, scopeEnd)
			}
		}
		return
	}
	line := toks[pos].Line
	for i := pos + 1; i < len(toks) && toks[i].Start < end && toks[i].Line == line; i++ {
		if toks[i].Text == ")" || toks[i].Text == "}" || toks[i].Text == ":=" {
			return
		}
		if toks[i].Text == "=" {
			return
		}
		if toks[i].Kind != scan.Ident || symbolNameTableUnitName(topNames, toks[i].Text) == "" {
			continue
		}
		if i == pos+1 || toks[i-1].Text == "," {
			addLocalName(names, toks[i].Text, toks[i].Start, scopeEnd)
		}
	}
}

func localScopeEnd(toks []scan.Token, body int, pos int, fallback int) int {
	var opens []int
	for i := body; i <= pos && i < len(toks); i++ {
		if toks[i].Text == "{" {
			opens = append(opens, i)
		} else if toks[i].Text == "}" && len(opens) > 0 {
			opens = opens[:len(opens)-1]
		}
	}
	if len(opens) == 0 {
		return fallback
	}
	close := findClose(toks, opens[len(opens)-1], "{", "}")
	if close < 0 {
		return fallback
	}
	return toks[close].Start
}

func addLocalName(names *localNameTable, name string, start int, end int) {
	values := *names
	values = append(values, localNameRange{name: name, start: start, end: end})
	*names = values
}

func maxSourcePosition() int {
	return 2147483647
}

func tokenIndexAt(toks []scan.Token, start int) int {
	for i := 0; i < len(toks); i++ {
		tok := toks[i]
		if tok.Start == start {
			return i
		}
	}
	return -1
}

func findTokenText(toks []scan.Token, start int, end int, text string) int {
	for i := start; i < len(toks) && toks[i].Start < end; i++ {
		if toks[i].Text == text {
			return i
		}
	}
	return -1
}

func findClose(toks []scan.Token, pos int, open string, close string) int {
	depth := 0
	for pos < len(toks) {
		if toks[pos].Text == open {
			depth++
		} else if toks[pos].Text == close {
			depth--
			if depth == 0 {
				return pos
			}
		}
		pos++
	}
	return -1
}

func containsTokenText(toks []scan.Token, start int, end int, text string) bool {
	for i := start; i < end && i < len(toks); i++ {
		if toks[i].Text == text {
			return true
		}
	}
	return false
}

func isTypeStart(tok scan.Token) bool {
	return tok.Kind == scan.Ident || tok.Text == "*" || tok.Text == "[" || tok.Text == "..."
}

func isTypeStartAfterName(toks []scan.Token, pos int, end int) bool {
	if pos+1 >= end {
		return false
	}
	if toks[pos+1].Text == "," {
		return isTypeStartAfterName(toks, pos+2, end)
	}
	return isTypeStart(toks[pos+1])
}

func isStatementBoundary(text string) bool {
	return text == "{" || text == "}" || text == ";" || text == "if" || text == "for" || text == "switch"
}

func containsString(values []string, value string) bool {
	for i := 0; i < len(values); i++ {
		if values[i] == value {
			return true
		}
	}
	return false
}

func dependencyPackages(graph *load.Graph) []load.Package {
	var packages []load.Package
	if graph == nil {
		return packages
	}
	for i := 0; i < len(graph.Packages); i++ {
		dep := graph.Packages[i]
		packages = append(packages, dep)
	}
	return packages
}

func packageByImportPath(packages []load.Package, importPath string) (load.Package, bool) {
	for i := 0; i < len(packages); i++ {
		pkg := packages[i]
		if pkg.ImportPath == importPath {
			return pkg, true
		}
	}
	return load.Package{}, false
}

func mergedMethodMap(local methodTable, localOrder []string, imported methodTable, importedOrder []string) methodTable {
	if len(imported) == 0 {
		return local
	}
	var methods methodTable
	for i := 0; i < len(localOrder); i++ {
		name := localOrder[i]
		method := methodTableLookup(local, name)
		methods = methodTableSet(methods, name, method)
	}
	for i := 0; i < len(importedOrder); i++ {
		name := importedOrder[i]
		method := methodTableLookup(imported, name)
		existing := methodTableLookup(methods, name)
		if existing.unitName == "" {
			methods = methodTableSet(methods, name, method)
		}
	}
	return methods
}

func importReferenceMap(file *parse.File, packages []load.Package) (importSymbolTable, []string) {
	var refs importSymbolTable
	var refNames []string
	for impIndex := 0; impIndex < len(file.Imports); impIndex++ {
		imp := file.Imports[impIndex]
		localName := importLocalName(imp)
		importPath := imp.Path
		dep, ok := packageByImportPath(packages, importPath)
		if !ok || localName == "" {
			continue
		}
		var symbols []unit.Symbol
		for fileIndex := 0; fileIndex < len(dep.Files); fileIndex++ {
			depFile := dep.Files[fileIndex]
			names := dependencyExportedNames(depFile.Source)
			for nameIndex := 0; nameIndex < len(names); nameIndex++ {
				name := names[nameIndex]
				symbols = setSymbol(symbols, unit.Symbol{ImportPath: importPath, Name: name, UnitName: SymbolName(importPath, name)})
			}
		}
		intrinsicNames := intrinsicImportSymbolNames(importPath)
		for i := 0; i < len(intrinsicNames); i++ {
			name := intrinsicNames[i]
			intrinsic := intrinsicImportSymbol(importPath, name)
			symbols = setSymbol(symbols, unit.Symbol{Name: name, UnitName: intrinsic})
		}
		refNames = append(refNames, localName)
		refs = append(refs, importSymbolGroup{localName: localName, symbols: symbols})
	}
	return refs, refNames
}

func importMethodMap(file *parse.File, packages []load.Package) (methodTable, []string) {
	var methods methodTable
	var methodNames []string
	for impIndex := 0; impIndex < len(file.Imports); impIndex++ {
		imp := file.Imports[impIndex]
		localName := importLocalName(imp)
		importPath := imp.Path
		dep, ok := packageByImportPath(packages, importPath)
		if !ok || localName == "" {
			continue
		}
		for fileIndex := 0; fileIndex < len(dep.Files); fileIndex++ {
			depFile := dep.Files[fileIndex]
			methodsInFile := dependencyExportedMethods(importPath, depFile.Source)
			for methodIndex := 0; methodIndex < len(methodsInFile); methodIndex++ {
				info := methodsInFile[methodIndex]
				if info.name == "" || info.receiverType == "" {
					continue
				}
				methodName := localName + "." + info.name
				existing := methodTableLookup(methods, methodName)
				if existing.unitName == "" {
					methodNames = append(methodNames, methodName)
				}
				methods = methodTableSet(methods, methodName, info)
			}
		}
	}
	return methods, methodNames
}

func dependencyExportedNames(src []byte) []string {
	toks, err := scan.Tokens(src)
	if err != nil {
		return nil
	}
	var names []string
	pos := dependencyTopLevelStart(toks)
	for pos < len(toks) && toks[pos].Kind != scan.EOF {
		tok := toks[pos]
		if tok.Text == "func" {
			next := pos + 1
			if next < len(toks) && toks[next].Text == "(" {
				close := findClose(toks, next, "(", ")")
				if close > next && close+1 < len(toks) {
					nameTok := close + 1
					if toks[nameTok].Kind == scan.Ident && isExported(toks[nameTok].Text) {
						names = appendStringUnique(names, toks[nameTok].Text)
					}
				}
			} else if next < len(toks) && toks[next].Kind == scan.Ident && isExported(toks[next].Text) {
				names = appendStringUnique(names, toks[next].Text)
			}
			pos = dependencySkipFunc(toks, pos+1)
			continue
		}
		if tok.Text == "type" {
			next := pos + 1
			if next < len(toks) && toks[next].Text == "(" {
				close := findClose(toks, next, "(", ")")
				names = appendExportedGroupedNames(names, toks, next+1, close)
				if close > next {
					pos = close + 1
					continue
				}
			} else if next < len(toks) && toks[next].Kind == scan.Ident && isExported(toks[next].Text) {
				names = appendStringUnique(names, toks[next].Text)
			}
			pos = dependencySkipLine(toks, pos)
			continue
		}
		if tok.Text == "const" || tok.Text == "var" {
			next := pos + 1
			if next < len(toks) && toks[next].Text == "(" {
				close := findClose(toks, next, "(", ")")
				names = appendExportedGroupedNames(names, toks, next+1, close)
				if close > next {
					pos = close + 1
					continue
				}
			} else {
				lineEnd := dependencyLineEnd(toks, pos)
				names = appendExportedSingleValueNames(names, toks, next, lineEnd)
			}
			pos = dependencySkipLine(toks, pos)
			continue
		}
		pos++
	}
	return names
}

func dependencyExportedMethods(importPath string, src []byte) []methodInfo {
	toks, err := scan.Tokens(src)
	if err != nil {
		return nil
	}
	var methods []methodInfo
	pos := dependencyTopLevelStart(toks)
	for pos < len(toks) && toks[pos].Kind != scan.EOF {
		if toks[pos].Text != "func" || pos+1 >= len(toks) || toks[pos+1].Text != "(" {
			pos++
			continue
		}
		receiverOpen := pos + 1
		receiverClose := findClose(toks, receiverOpen, "(", ")")
		if receiverClose > receiverOpen && receiverClose+1 < len(toks) && toks[receiverClose+1].Kind == scan.Ident {
			nameTok := receiverClose + 1
			if isExported(toks[nameTok].Text) {
				var decl parse.Decl
				decl.Kind = "func"
				decl.Name = toks[nameTok].Text
				decl.Receiver = true
				decl.Start = toks[pos].Start
				info := methodDeclInfoFromTokens(toks, &decl)
				if info.name != "" {
					info.unitName = SymbolName(importPath, info.name)
					info.importPath = importPath
					methods = append(methods, info)
				}
			}
		}
		pos = dependencySkipFunc(toks, pos+1)
	}
	return methods
}

func dependencyTopLevelStart(toks []scan.Token) int {
	pos := 0
	if len(toks) >= 2 && toks[0].Text == "package" {
		pos = 2
	}
	for pos < len(toks) && toks[pos].Text == "import" {
		pos = dependencySkipImport(toks, pos)
	}
	return pos
}

func dependencySkipImport(toks []scan.Token, pos int) int {
	next := pos + 1
	if next < len(toks) && toks[next].Text == "(" {
		close := findClose(toks, next, "(", ")")
		if close > next {
			return close + 1
		}
	}
	return dependencySkipLine(toks, pos)
}

func dependencySkipFunc(toks []scan.Token, pos int) int {
	for pos < len(toks) && toks[pos].Kind != scan.EOF {
		if toks[pos].Text == "{" {
			close := findClose(toks, pos, "{", "}")
			if close > pos {
				return close + 1
			}
		}
		pos++
	}
	return pos
}

func dependencySkipLine(toks []scan.Token, pos int) int {
	line := toks[pos].Line
	for pos < len(toks) && toks[pos].Line == line && toks[pos].Kind != scan.EOF {
		pos++
	}
	return pos
}

func dependencyLineEnd(toks []scan.Token, pos int) int {
	line := toks[pos].Line
	for pos < len(toks) && toks[pos].Line == line && toks[pos].Kind != scan.EOF {
		pos++
	}
	return pos
}

func appendExportedGroupedNames(names []string, toks []scan.Token, start int, end int) []string {
	if end <= start {
		return names
	}
	lineStart := true
	lastLine := -1
	for i := start; i < end; i++ {
		tok := toks[i]
		if lastLine != tok.Line {
			lineStart = true
			lastLine = tok.Line
		}
		if lineStart && tok.Kind == scan.Ident {
			if isExported(tok.Text) {
				names = appendStringUnique(names, tok.Text)
			}
			lineStart = false
			continue
		}
		if tok.Text == ";" {
			lineStart = true
		}
	}
	return names
}

func appendExportedSingleValueNames(names []string, toks []scan.Token, start int, end int) []string {
	expectName := true
	for i := start; i < end; i++ {
		tok := toks[i]
		if tok.Text == "=" {
			return names
		}
		if tok.Kind == scan.Ident && expectName {
			if isExported(tok.Text) {
				names = appendStringUnique(names, tok.Text)
			}
			expectName = false
			continue
		}
		if tok.Text == "," {
			expectName = true
			continue
		}
		if len(names) > 0 {
			return names
		}
	}
	return names
}

func appendStringUnique(values []string, value string) []string {
	if containsString(values, value) {
		return values
	}
	return append(values, value)
}

func intrinsicImportSymbol(importPath string, name string) string {
	if importPath != "os" {
		return ""
	}
	switch name {
	case "Open":
		return "open"
	case "Close":
		return "close"
	case "Read":
		return "read"
	case "Write":
		return "write"
	case "Chmod":
		return "chmod"
	case "O_RDONLY":
		return "O_RDONLY"
	case "O_WRONLY":
		return "O_WRONLY"
	case "O_RDWR":
		return "O_RDWR"
	case "O_CREATE":
		return "O_CREATE"
	case "O_TRUNC":
		return "O_TRUNC"
	case "Stdin":
		return "0"
	case "Stdout":
		return "1"
	case "Stderr":
		return "2"
	}
	return ""
}

func intrinsicImportSymbolNames(importPath string) []string {
	if importPath != "os" {
		return nil
	}
	return []string{"Open", "Close", "Read", "Write", "Chmod", "O_RDONLY"}
}

func importLocalName(imp parse.Import) string {
	if imp.Alias != "" && imp.Alias != "." && imp.Alias != "_" {
		return imp.Alias
	}
	return load.PackageNameFromImportPath(imp.Path)
}

func isExported(name string) bool {
	if name == "" {
		return false
	}
	c := name[0]
	return c >= 'A' && c <= 'Z'
}
