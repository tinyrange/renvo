package check

import (
	"strconv"
	"strings"

	"j5.nz/rtg/rtg/arena"
	"j5.nz/rtg/rtg/load"
	"j5.nz/rtg/rtg/parse"
	"j5.nz/rtg/rtg/scan"
)

type Diagnostic struct {
	Path    string
	Line    int
	Column  int
	Message string
}

func (d Diagnostic) Error() string {
	return d.Path + ":" + strconv.Itoa(d.Line) + ":" + strconv.Itoa(d.Column) + ": " + d.Message
}

type Diagnostics []Diagnostic

type diagnosticError string

func (err diagnosticError) Error() string {
	return string(err)
}

type exportedPackage struct {
	importPath string
	names      []string
}

type importEntry struct {
	name string
	path string
}

type importNameToken struct {
	name string
	tok  scan.Token
}

type localShadow struct {
	name  string
	start int
	end   int
}

func (d Diagnostics) Error() string {
	if len(d) == 0 {
		return ""
	}
	var parts []string
	for i := 0; i < len(d); i++ {
		diag := d[i]
		parts = append(parts, diagnosticString(diag))
	}
	return strings.Join(parts, "\n")
}

func diagnosticString(diag Diagnostic) string {
	return diag.Error()
}

func Graph(g *load.Graph) error {
	var diags Diagnostics
	var parseDiags Diagnostics
	exported, parseDiags := exportedDecls(g)
	diags = appendDiagnostics(diags, parseDiags)
	if len(diags) > 0 {
		return diagnosticError(diags.Error())
	}
	var packages []load.Package
	packages = g.Packages
	for pkgIndex := 0; pkgIndex < len(packages); pkgIndex++ {
		var pkg load.Package
		pkg = packages[pkgIndex]
		var names []string
		var nameDiags []Diagnostic
		var files []load.File
		files = pkg.Files
		for fileIndex := 0; fileIndex < len(files); fileIndex++ {
			var file load.File
			file = files[fileIndex]
			mark := arena.Mark()
			parsed, err := parsedLoadFile(file)
			if err != nil {
				diags = appendParseDiagnostic(diags, file.Path, err)
				continue
			}
			parsedPackageName := parsed.PackageName
			loadedPackageName := pkg.Name
			if !sameString(parsedPackageName, loadedPackageName) {
				diags = appendDiagnostic(diags, Diagnostic{Path: file.Path, Line: 1, Column: 1, Message: "package name changed during parsing"})
				continue
			}
			diags = appendDiagnostics(diags, File(parsed))
			diags = appendDiagnostics(diags, importedSelectorDiagnostics(parsed, exported))
			var decls []parse.Decl
			decls = parsed.Decls
			for declIndex := 0; declIndex < len(decls); declIndex++ {
				var decl parse.Decl
				decl = decls[declIndex]
				namesForDecl := packageLevelDeclNames(decl)
				for i := 0; i < len(namesForDecl); i++ {
					name := namesForDecl[i]
					if name == "" || name == "_" {
						continue
					}
					current := declNameDiagnostic(parsed, decl, i, "duplicate package-level declaration: "+name)
					previousIndex := stringIndex(names, name)
					if previousIndex >= 0 {
						previous := nameDiags[previousIndex]
						diags = append(diags, previous)
						diags = append(diags, current)
						continue
					}
					names = append(names, arena.PersistString(name))
					nameDiags = append(nameDiags, current)
				}
			}
			if len(diags) == 0 {
				arena.Reset(mark)
			}
		}
	}
	if len(diags) > 0 {
		return diagnosticError(diags.Error())
	}
	return nil
}

func parseGraphFiles(g *load.Graph) Diagnostics {
	var diags Diagnostics
	for pkgIndex := 0; pkgIndex < len(g.Packages); pkgIndex++ {
		files := g.Packages[pkgIndex].Files
		for fileIndex := 0; fileIndex < len(files); fileIndex++ {
			if files[fileIndex].Parsed.Path != "" {
				continue
			}
			parsed, err := parse.FileSource(files[fileIndex].Path, files[fileIndex].Source)
			if err != nil {
				diags = appendParseDiagnostic(diags, files[fileIndex].Path, err)
				continue
			}
			files[fileIndex].Parsed = parsed
		}
		g.Packages[pkgIndex].Files = files
	}
	return diags
}

func parseDiagnostic(path string, err error) Diagnostic {
	line, column, message, ok := splitPathPositionMessage(path, err.Error())
	if ok {
		return Diagnostic{
			Path:    path,
			Line:    line,
			Column:  column,
			Message: message,
		}
	}
	return Diagnostic{Path: path, Line: 1, Column: 1, Message: err.Error()}
}

func splitPathPositionMessage(path string, message string) (int, int, string, bool) {
	prefix := path + ":"
	if !strings.HasPrefix(message, prefix) {
		return 0, 0, "", false
	}
	rest := message[len(prefix):]
	first := strings.IndexByte(rest, ':')
	if first < 0 {
		return 0, 0, "", false
	}
	second := strings.IndexByte(rest[first+1:], ':')
	if second < 0 {
		return 0, 0, "", false
	}
	second = second + first + 1
	line, err := strconv.Atoi(rest[:first])
	if err != nil {
		return 0, 0, "", false
	}
	column, err := strconv.Atoi(rest[first+1 : second])
	if err != nil {
		return 0, 0, "", false
	}
	return line, column, strings.TrimSpace(rest[second+1:]), true
}

func declDiagnostics(file parse.File) Diagnostics {
	var diags Diagnostics
	var decls []parse.Decl
	decls = file.Decls
	for i := 0; i < len(decls); i++ {
		var decl parse.Decl
		decl = decls[i]
		if decl.Kind == "func" {
			if decl.Name == "init" {
				diags = appendDeclDiagnostic(diags, file, decl, "init functions are not supported")
			}
			if tok, ok := namedResultToken(file, decl); ok {
				diags = appendDiag(diags, file, tok, "named result parameters are not supported")
			}
		}
		if decl.Kind == "const" {
			if tok, ok := declToken(file, decl, "iota"); ok {
				diags = appendDiag(diags, file, tok, "iota is not supported")
			}
		}
		if file.PackageName == "main" && decl.Kind == "func" && decl.Name == "main" && !hasOrdinaryMainSignature(file, decl) {
			diags = appendDeclDiagnostic(diags, file, decl, "main function must have no parameters or results")
		}
	}
	return diags
}

func exportedDecls(g *load.Graph) ([]exportedPackage, Diagnostics) {
	var out []exportedPackage
	var diags Diagnostics
	var packages []load.Package
	packages = g.Packages
	for pkgIndex := 0; pkgIndex < len(packages); pkgIndex++ {
		var pkg load.Package
		pkg = packages[pkgIndex]
		var names []string
		var files []load.File
		files = pkg.Files
		for fileIndex := 0; fileIndex < len(files); fileIndex++ {
			var file load.File
			file = files[fileIndex]
			mark := arena.Mark()
			parsed, err := parsedLoadFile(file)
			if err != nil {
				diags = appendParseDiagnostic(diags, file.Path, err)
				continue
			}
			var decls []parse.Decl
			decls = parsed.Decls
			for declIndex := 0; declIndex < len(decls); declIndex++ {
				var decl parse.Decl
				decl = decls[declIndex]
				namesForDecl := packageLevelDeclNames(decl)
				for nameIndex := 0; nameIndex < len(namesForDecl); nameIndex++ {
					name := namesForDecl[nameIndex]
					if isExported(name) && !containsString(names, name) {
						names = append(names, arena.PersistString(name))
					}
				}
			}
			arena.Reset(mark)
		}
		var exported exportedPackage
		exported.importPath = pkg.ImportPath
		for i := 0; i < len(names); i++ {
			exported.names = append(exported.names, names[i])
		}
		out = append(out, exported)
	}
	return out, diags
}

func packageLevelDeclNames(decl parse.Decl) []string {
	if decl.Receiver {
		return nil
	}
	return declNames(decl)
}

func declNames(decl parse.Decl) []string {
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

func importedSelectorDiagnostics(file parse.File, exported []exportedPackage) Diagnostics {
	var localImports []importEntry
	var importNames []string
	var imports []parse.Import
	imports = file.Imports
	for i := 0; i < len(imports); i++ {
		var imp parse.Import
		imp = imports[i]
		localName := importLocalName(imp)
		if localName != "" {
			localImports = append(localImports, importEntry{name: localName, path: imp.Path})
			if !containsString(importNames, localName) {
				importNames = append(importNames, localName)
			}
		}
	}
	if len(localImports) == 0 {
		return nil
	}
	shadows := localImportShadows(file, importNames)
	var diags Diagnostics
	var tokens []scan.Token
	tokens = file.Tokens
	for i := 0; i+2 < len(tokens); i++ {
		var local scan.Token
		local = tokens[i]
		var dot scan.Token
		dot = tokens[i+1]
		var member scan.Token
		member = tokens[i+2]
		if local.Kind != scan.Ident || dot.Text != "." || member.Kind != scan.Ident {
			continue
		}
		importPath, ok := importEntryPath(localImports, local.Text)
		if !ok {
			continue
		}
		if isLocalShadowAt(shadows, local.Text, int(local.Start)) {
			continue
		}
		if exportedNameExists(exported, importPath, member.Text) {
			continue
		}
		diags = appendDiag(diags, file, member, "unresolved imported selector: "+importPath+"."+member.Text)
	}
	return diags
}

func File(file parse.File) Diagnostics {
	var diags Diagnostics
	diags = appendDiagnostics(diags, importDiagnostics(file))
	diags = appendDiagnostics(diags, declDiagnostics(file))
	var tokens []scan.Token
	tokens = file.Tokens
	for i := 0; i < len(tokens); i++ {
		var tok scan.Token
		tok = tokens[i]
		if tok.Kind == scan.EOF {
			break
		}
		if tok.Kind == scan.String && strings.HasPrefix(tok.Text, "`") {
			diags = appendDiag(diags, file, tok, "raw string literals are not supported")
		}
		if tok.Kind == scan.Number && isImaginaryLiteral(tok.Text) {
			diags = appendDiag(diags, file, tok, "imaginary literals are not supported")
		}
		if tok.Kind == scan.Number && isOctalLiteral(tok.Text) {
			diags = appendDiag(diags, file, tok, "octal literals are not supported")
		}
		switch tok.Text {
		case "...":
			diags = appendDiag(diags, file, tok, "variadic syntax is not supported")
		case "go":
			diags = appendDiag(diags, file, tok, "goroutines are not supported")
		case "chan", "<-":
			diags = appendDiag(diags, file, tok, "channels are not supported")
		case "select":
			diags = appendDiag(diags, file, tok, "select statements are not supported")
		case "interface":
			diags = appendDiag(diags, file, tok, "interfaces are not supported")
		case "map":
			diags = appendDiag(diags, file, tok, "maps are not supported")
		case "defer":
			diags = appendDiag(diags, file, tok, "defer is not supported")
		case "range":
			diags = appendDiag(diags, file, tok, "range is not supported")
		case "fallthrough":
			diags = appendDiag(diags, file, tok, "fallthrough is not supported")
		case "func":
			if !file.IsTopLevelFuncAt(i) {
				diags = appendDiag(diags, file, tok, "function values and function types are not supported")
			}
		}
		if startsArrayType(tokens, i) {
			diags = appendDiag(diags, file, tok, "arrays are not supported")
		}
		if startsAnyInterfaceType(tokens, i) {
			diags = appendDiag(diags, file, tok, "interfaces are not supported")
		}
		if startsGenericDecl(file, i) {
			var genericTok scan.Token
			genericTok = tokens[i+2]
			diags = appendDiag(diags, file, genericTok, "generics are not supported")
		}
		if startsGenericInstantiation(tokens, i) {
			var genericTok scan.Token
			genericTok = tokens[i+1]
			diags = appendDiag(diags, file, genericTok, "generics are not supported")
		}
		if startsTypeAssertion(tokens, i) {
			var typeTok scan.Token
			typeTok = tokens[i+1]
			diags = appendDiag(diags, file, typeTok, "type assertions and type switches are not supported")
		}
		if colon := fullSliceSecondColon(tokens, i); colon >= 0 {
			var colonTok scan.Token
			colonTok = tokens[colon]
			diags = appendDiag(diags, file, colonTok, "full slice expressions are not supported")
		}
		if startsUnsupportedBuiltinCall(tokens, i) && !isRuntimeOSIntrinsicCall(file, tokens, i) {
			diags = appendDiag(diags, file, tok, "unsupported builtin: "+tok.Text)
		}
	}
	return diags
}

func appendDiagnostics(out Diagnostics, values Diagnostics) Diagnostics {
	for i := 0; i < len(values); i++ {
		diag := values[i]
		out = append(out, diag)
	}
	return out
}

func parsedLoadFile(file load.File) (parse.File, error) {
	if file.Parsed.Path != "" {
		return file.Parsed, nil
	}
	return parse.FileSource(file.Path, file.Source)
}

func appendParseDiagnostic(diags Diagnostics, path string, err error) Diagnostics {
	d := parseDiagnostic(path, err)
	return appendDiagnostic(diags, d)
}

func appendDeclDiagnostic(diags Diagnostics, file parse.File, decl parse.Decl, message string) Diagnostics {
	d := declDiagnostic(file, decl, message)
	return appendDiagnostic(diags, d)
}

func appendDiag(diags Diagnostics, file parse.File, tok scan.Token, message string) Diagnostics {
	d := diag(file, tok, message)
	return appendDiagnostic(diags, d)
}

func appendDiagnostic(diags Diagnostics, d Diagnostic) Diagnostics {
	return append(diags, d)
}

func containsString(values []string, value string) bool {
	return stringIndex(values, value) >= 0
}

func sameString(a string, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func stringIndex(values []string, value string) int {
	for i := 0; i < len(values); i++ {
		if sameString(values[i], value) {
			return i
		}
	}
	return -1
}

func importEntryPath(values []importEntry, name string) (string, bool) {
	for i := 0; i < len(values); i++ {
		if values[i].name == name {
			return values[i].path, true
		}
	}
	return "", false
}

func exportedNameExists(values []exportedPackage, importPath string, name string) bool {
	for i := 0; i < len(values); i++ {
		value := values[i]
		if value.importPath == importPath {
			if len(value.names) == 0 {
				return true
			}
			return containsString(value.names, name)
		}
	}
	return true
}

func hasImportNameToken(values []importNameToken, name string) bool {
	for i := 0; i < len(values); i++ {
		if values[i].name == name {
			return true
		}
	}
	return false
}

func importDiagnostics(file parse.File) Diagnostics {
	var diags Diagnostics
	var names []importNameToken
	var importNames []string
	var imports []parse.Import
	imports = file.Imports
	for i := 0; i < len(imports); i++ {
		var imp parse.Import
		imp = imports[i]
		localName := importLocalName(imp)
		if localName != "" && localName != "." && localName != "_" {
			if !containsString(importNames, localName) {
				importNames = append(importNames, localName)
			}
		}
	}
	used := usedImportNames(file, importNames)
	for i := 0; i < len(imports); i++ {
		var imp parse.Import
		imp = imports[i]
		if imp.Alias == "." {
			diags = appendDiag(diags, file, imp.Tok, "dot imports are not supported")
		}
		if imp.Alias == "_" {
			diags = appendDiag(diags, file, imp.Tok, "blank imports are not supported")
		}
		localName := importLocalName(imp)
		if localName == "" || localName == "." || localName == "_" {
			continue
		}
		if hasImportNameToken(names, localName) {
			diags = appendDiag(diags, file, imp.Tok, "duplicate import name: "+localName)
			continue
		}
		names = append(names, importNameToken{name: localName, tok: imp.Tok})
		if !containsString(used, localName) {
			diags = appendDiag(diags, file, imp.Tok, "unused import: "+localName)
		}
	}
	return diags
}

func usedImportNames(file parse.File, importNames []string) []string {
	var used []string
	shadows := localImportShadows(file, importNames)
	var tokens []scan.Token
	tokens = file.Tokens
	for i := 0; i+1 < len(tokens); i++ {
		var tok scan.Token
		tok = tokens[i]
		var next scan.Token
		next = tokens[i+1]
		if tok.Kind == scan.Ident && next.Text == "." {
			if isLocalShadowAt(shadows, tok.Text, int(tok.Start)) {
				continue
			}
			name := tok.Text
			if !containsString(used, name) {
				used = append(used, name)
			}
		}
	}
	return used
}

func importLocalName(imp parse.Import) string {
	if imp.Alias != "" {
		return imp.Alias
	}
	return load.PackageNameFromImportPath(imp.Path)
}

func startsGenericDecl(file parse.File, i int) bool {
	toks := file.Tokens
	if i+2 >= len(toks) {
		return false
	}
	if toks[i].Text == "type" && toks[i+1].Kind == scan.Ident && toks[i+2].Text == "[" {
		close := findClose(toks, i+2, "[", "]")
		return close > i+4
	}
	if toks[i].Text == "func" && file.IsTopLevelFuncAt(i) {
		namePos := i + 1
		if toks[namePos].Text == "(" {
			close := findClose(toks, namePos, "(", ")")
			if close < 0 || close+2 >= len(toks) {
				return false
			}
			namePos = close + 1
		}
		if toks[namePos].Kind != scan.Ident || toks[namePos+1].Text != "[" {
			return false
		}
		close := findClose(toks, namePos+1, "[", "]")
		return close > namePos+3
	}
	return false
}

func startsGenericInstantiation(toks []scan.Token, i int) bool {
	if i+3 >= len(toks) {
		return false
	}
	if toks[i].Kind != scan.Ident {
		return false
	}
	if toks[i+1].Text != "[" {
		return false
	}
	close := findClose(toks, i+1, "[", "]")
	if close < 0 {
		return false
	}
	if close+1 >= len(toks) {
		return false
	}
	if toks[close+1].Text == "{" && isControlBlockOpen(toks, close+1) {
		return false
	}
	return toks[close+1].Text == "{" || toks[close+1].Text == "("
}

func isControlBlockOpen(toks []scan.Token, open int) bool {
	parenDepth := 0
	brackDepth := 0
	for i := open - 1; i >= 0; i-- {
		text := toks[i].Text
		if text == ")" {
			parenDepth++
			continue
		}
		if text == "(" && parenDepth > 0 {
			parenDepth--
			continue
		}
		if text == "]" {
			brackDepth++
			continue
		}
		if text == "[" && brackDepth > 0 {
			brackDepth--
			continue
		}
		if parenDepth != 0 || brackDepth != 0 {
			continue
		}
		if text == "if" || text == "for" || text == "switch" || text == "select" {
			return true
		}
		if text == "{" || text == "}" || text == "func" {
			return false
		}
	}
	return false
}

func isImaginaryLiteral(text string) bool {
	return strings.HasSuffix(text, "i")
}

func isOctalLiteral(text string) bool {
	if len(text) < 2 || text[0] != '0' {
		return false
	}
	next := text[1]
	if next == 'x' || next == 'X' || next == 'b' || next == 'B' || next == '.' {
		return false
	}
	if next == 'o' || next == 'O' {
		return true
	}
	return next >= '0' && next <= '9'
}

func startsArrayType(toks []scan.Token, i int) bool {
	if i+1 >= len(toks) {
		return false
	}
	if toks[i].Text != "[" {
		return false
	}
	if toks[i+1].Text == "]" {
		return false
	}
	if i == 0 {
		return false
	}
	prev := toks[i-1]
	if prev.Text == "map" {
		return false
	}
	if prev.Text == "*" {
		return precededByTypeContext(toks, i-1)
	}
	if prev.Text == "]" {
		open := findOpen(toks, i-1, "[", "]")
		return open >= 0 && open+1 == i-1 && precededByTypeContext(toks, open)
	}
	if prev.Text == ")" {
		return closesFunctionSignature(toks, i-1)
	}
	if prev.Kind != scan.Ident {
		return false
	}
	return precededByTypeContext(toks, i-1)
}

func closesFunctionSignature(toks []scan.Token, close int) bool {
	open := findOpen(toks, close, "(", ")")
	if open < 0 {
		return false
	}
	if open > 0 && toks[open-1].Text == "func" {
		return true
	}
	if open > 1 && toks[open-2].Text == "func" && toks[open-1].Kind == scan.Ident {
		return true
	}
	return false
}

func startsAnyInterfaceType(toks []scan.Token, i int) bool {
	if toks[i].Text != "any" {
		return false
	}
	if i == 0 {
		return false
	}
	prev := toks[i-1]
	if prev.Text == "*" {
		return true
	}
	if prev.Text == "]" && i >= 2 && toks[i-2].Text == "[" {
		return true
	}
	if prev.Text == ")" {
		return isFunctionSignatureResult(toks, i)
	}
	if prev.Kind != scan.Ident {
		return false
	}
	if i < 2 {
		return false
	}
	beforeName := toks[i-2].Text
	return beforeName == "var" || beforeName == "type" || beforeName == "(" || beforeName == "{" || beforeName == ","
}

func isFunctionSignatureResult(toks []scan.Token, pos int) bool {
	for i := pos - 2; i >= 0 && toks[i].Line == toks[pos].Line; i-- {
		if toks[i].Text == "func" {
			return true
		}
		if toks[i].Text == "{" || toks[i].Text == ";" {
			return false
		}
	}
	return false
}

func startsTypeAssertion(toks []scan.Token, i int) bool {
	if i+2 >= len(toks) {
		return false
	}
	if toks[i].Text != "." {
		return false
	}
	if toks[i+1].Text != "(" {
		return false
	}
	close := findClose(toks, i+1, "(", ")")
	return close > i+2
}

func fullSliceSecondColon(toks []scan.Token, i int) int {
	if i >= len(toks) {
		return -1
	}
	if toks[i].Text != "[" {
		return -1
	}
	close := findClose(toks, i, "[", "]")
	if close < 0 {
		return -1
	}
	colons := 0
	paren := 0
	brack := 0
	brace := 0
	for j := i + 1; j < close; j++ {
		if paren == 0 && brack == 0 && brace == 0 && toks[j].Text == ":" {
			colons++
			if colons == 2 {
				return j
			}
			continue
		}
		updateDepth(toks[j].Text, &paren, &brack, &brace)
	}
	return -1
}

func startsUnsupportedBuiltinCall(toks []scan.Token, i int) bool {
	if i+1 >= len(toks) {
		return false
	}
	if toks[i].Kind != scan.Ident {
		return false
	}
	if toks[i+1].Text != "(" {
		return false
	}
	if i > 0 && toks[i-1].Text == "." {
		return false
	}
	switch toks[i].Text {
	case "cap", "close", "complex", "delete", "imag", "new", "panic", "println", "real", "recover":
		return true
	}
	return false
}

func isRuntimeOSIntrinsicCall(file parse.File, toks []scan.Token, i int) bool {
	if file.PackageName != "os" {
		return false
	}
	if file.Path != "rtg/std/os/os_rtg.go" && file.Path != "rtg\\std\\os\\os_rtg.go" && !strings.HasSuffix(file.Path, "/rtg/std/os/os_rtg.go") && !strings.HasSuffix(file.Path, "\\rtg\\std\\os\\os_rtg.go") {
		return false
	}
	if i+1 >= len(toks) {
		return false
	}
	if toks[i].Kind != scan.Ident {
		return false
	}
	if toks[i+1].Text != "(" {
		return false
	}
	switch toks[i].Text {
	case "open", "close", "read", "write", "chmod":
		return true
	}
	return false
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

func findOpen(toks []scan.Token, pos int, open string, close string) int {
	depth := 0
	for pos >= 0 {
		if toks[pos].Text == close {
			depth++
		} else if toks[pos].Text == open {
			depth--
			if depth == 0 {
				return pos
			}
		}
		pos--
	}
	return -1
}

func updateDepth(text string, paren *int, brack *int, brace *int) {
	switch text {
	case "(":
		*paren = *paren + 1
	case ")":
		*paren = *paren - 1
	case "[":
		*brack = *brack + 1
	case "]":
		*brack = *brack - 1
	case "{":
		*brace = *brace + 1
	case "}":
		*brace = *brace - 1
	}
}

func precededByTypeContext(toks []scan.Token, pos int) bool {
	if pos <= 0 {
		return false
	}
	prev := toks[pos-1]
	switch prev.Text {
	case "var", "type", "*":
		return true
	case "]":
		return toks[pos].Text == "*"
	}
	if prev.Kind == scan.Ident && pos >= 2 {
		if isKeyword(prev.Text) {
			return false
		}
		beforeName := toks[pos-2].Text
		if beforeName == "var" || beforeName == "type" {
			return true
		}
		if beforeName == "{" {
			return nameInStructFieldList(toks, pos-1)
		}
		if beforeName == "(" || beforeName == "," {
			namePos := pos - 1
			return nameInFunctionSignature(toks, namePos) || nameInStructFieldList(toks, namePos)
		}
	}
	return false
}

func nameInFunctionSignature(toks []scan.Token, namePos int) bool {
	open := containingOpen(toks, namePos, "(", ")")
	if open < 0 {
		return false
	}
	if open > 0 && toks[open-1].Text == "func" {
		return true
	}
	return open > 1 && toks[open-2].Text == "func" && toks[open-1].Kind == scan.Ident
}

func nameInStructFieldList(toks []scan.Token, namePos int) bool {
	open := containingOpen(toks, namePos, "{", "}")
	return open > 0 && toks[open-1].Text == "struct"
}

func containingOpen(toks []scan.Token, pos int, openText string, closeText string) int {
	depth := 0
	for i := pos - 1; i >= 0; i-- {
		if toks[i].Text == closeText {
			depth++
			continue
		}
		if toks[i].Text == openText {
			if depth == 0 {
				return i
			}
			depth--
		}
	}
	return -1
}

func isKeyword(text string) bool {
	switch text {
	case "break", "case", "chan", "const", "continue", "default", "defer", "else", "fallthrough", "for", "func", "go", "goto", "if", "import", "interface", "map", "package", "range", "return", "select", "struct", "switch", "type", "var":
		return true
	}
	return false
}

func localImportShadows(file parse.File, importNames []string) []localShadow {
	var shadows []localShadow
	if len(importNames) == 0 {
		return shadows
	}
	var decls []parse.Decl
	decls = file.Decls
	var tokens []scan.Token
	tokens = file.Tokens
	for i := 0; i < len(decls); i++ {
		var decl parse.Decl
		decl = decls[i]
		if decl.Kind != "func" {
			continue
		}
		start := tokenIndexAt(tokens, decl.Start)
		if start < 0 {
			continue
		}
		body := findTokenText(tokens, start, decl.End, "{")
		if body < 0 {
			continue
		}
		shadows = collectFuncSignatureImportShadows(tokens, start, body, importNames, shadows)
		for i := body + 1; i < len(tokens); i++ {
			var tok scan.Token
			tok = tokens[i]
			if int(tok.Start) >= decl.End {
				break
			}
			if tok.Text == ":=" {
				shadows = collectShortDeclImportShadows(tokens, body, i, decl.End, importNames, shadows)
			}
			if tok.Text == "var" {
				shadows = collectVarImportShadows(tokens, body, i, decl.End, importNames, shadows)
			}
		}
	}
	return shadows
}

func collectFuncSignatureImportShadows(toks []scan.Token, start int, end int, names []string, shadows []localShadow) []localShadow {
	for i := start; i < end; i++ {
		if toks[i].Text != "(" {
			continue
		}
		close := findClose(toks, i, "(", ")")
		if close < 0 || close > end {
			continue
		}
		shadows = collectParameterImportShadows(toks, i+1, close, names, shadows)
		i = close
	}
	return shadows
}

func collectParameterImportShadows(toks []scan.Token, start int, end int, names []string, shadows []localShadow) []localShadow {
	for i := start; i < end; i++ {
		if toks[i].Kind != scan.Ident || !containsString(names, toks[i].Text) {
			continue
		}
		if i > start && toks[i-1].Text != "," {
			continue
		}
		if i+1 < end && isTypeStart(toks[i+1]) {
			shadows = addLocalShadow(shadows, toks[i].Text, 0, maxSourcePosition())
			continue
		}
		if i+2 < end && toks[i+1].Text == "," && toks[i+2].Kind == scan.Ident && isTypeStartAfterName(toks, i+2, end) {
			shadows = addLocalShadow(shadows, toks[i].Text, 0, maxSourcePosition())
		}
	}
	return shadows
}

func collectShortDeclImportShadows(toks []scan.Token, body int, assign int, declEnd int, names []string, shadows []localShadow) []localShadow {
	line := toks[assign].Line
	scopeEnd := localScopeEnd(toks, body, assign, declEnd)
	for i := assign - 1; i >= 0; i-- {
		if toks[i].Line != line || isStatementBoundary(toks[i].Text) {
			return shadows
		}
		if toks[i].Kind == scan.Ident && containsString(names, toks[i].Text) && (i == 0 || toks[i-1].Text != ".") {
			shadows = addLocalShadow(shadows, toks[i].Text, int(toks[i].Start), scopeEnd)
		}
	}
	return shadows
}

func collectVarImportShadows(toks []scan.Token, body int, pos int, end int, names []string, shadows []localShadow) []localShadow {
	scopeEnd := localScopeEnd(toks, body, pos, end)
	if pos+1 < len(toks) && toks[pos+1].Text == "(" {
		for i := pos + 2; i < len(toks) && int(toks[i].Start) < end; i++ {
			if toks[i].Text == ")" || toks[i].Text == "}" {
				return shadows
			}
			if toks[i].Kind == scan.Ident && containsString(names, toks[i].Text) && (toks[i-1].Text == "(" || toks[i-1].Text == "," || toks[i-1].Line != toks[i].Line) {
				shadows = addLocalShadow(shadows, toks[i].Text, int(toks[i].Start), scopeEnd)
			}
		}
		return shadows
	}
	line := toks[pos].Line
	for i := pos + 1; i < len(toks) && int(toks[i].Start) < end && toks[i].Line == line; i++ {
		if toks[i].Text == ")" || toks[i].Text == "}" || toks[i].Text == ":=" || toks[i].Text == "=" {
			return shadows
		}
		if toks[i].Kind == scan.Ident && containsString(names, toks[i].Text) && (i == pos+1 || toks[i-1].Text == ",") {
			shadows = addLocalShadow(shadows, toks[i].Text, int(toks[i].Start), scopeEnd)
		}
	}
	return shadows
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
	return int(toks[close].Start)
}

func addLocalShadow(shadows []localShadow, name string, start int, end int) []localShadow {
	return append(shadows, localShadow{name: name, start: start, end: end})
}

func isLocalShadowAt(shadows []localShadow, name string, pos int) bool {
	for i := 0; i < len(shadows); i++ {
		shadow := shadows[i]
		if shadow.name == name && pos >= shadow.start && pos < shadow.end {
			return true
		}
	}
	return false
}

func findTokenText(toks []scan.Token, start int, end int, text string) int {
	for i := start; i < len(toks) && int(toks[i].Start) < end; i++ {
		if toks[i].Text == text {
			return i
		}
	}
	return -1
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

func maxSourcePosition() int {
	return 2147483647
}

func namedResultToken(file parse.File, decl parse.Decl) (scan.Token, bool) {
	var tokens []scan.Token
	tokens = file.Tokens
	name := tokenIndexAt(tokens, int(decl.NameTok.Start))
	if name < 0 || name+1 >= len(tokens) {
		return scan.Token{}, false
	}
	var openTok scan.Token
	openTok = tokens[name+1]
	if openTok.Text != "(" {
		return scan.Token{}, false
	}
	paramsClose := findClose(tokens, name+1, "(", ")")
	if paramsClose < 0 || paramsClose+1 >= len(tokens) {
		return scan.Token{}, false
	}
	var resultOpenTok scan.Token
	resultOpenTok = tokens[paramsClose+1]
	if resultOpenTok.Text != "(" {
		return scan.Token{}, false
	}
	resultsOpen := paramsClose + 1
	resultsClose := findClose(tokens, resultsOpen, "(", ")")
	if resultsClose < 0 {
		return scan.Token{}, false
	}
	var closeTok scan.Token
	closeTok = tokens[resultsClose]
	if int(closeTok.Start) >= decl.End {
		return scan.Token{}, false
	}
	for i := resultsOpen + 1; i < resultsClose; i++ {
		var tok scan.Token
		tok = tokens[i]
		if tok.Kind == scan.Ident && isTypeStartAfterName(tokens, i, resultsClose) {
			return tok, true
		}
	}
	return scan.Token{}, false
}

func hasOrdinaryMainSignature(file parse.File, decl parse.Decl) bool {
	var tokens []scan.Token
	tokens = file.Tokens
	name := tokenIndexAt(tokens, int(decl.NameTok.Start))
	if name < 0 || name+1 >= len(tokens) {
		return false
	}
	var openTok scan.Token
	openTok = tokens[name+1]
	if openTok.Text != "(" {
		return false
	}
	open := name + 1
	close := findClose(tokens, open, "(", ")")
	if close != open+1 {
		return false
	}
	for i := close + 1; i < len(tokens); i++ {
		var tok scan.Token
		tok = tokens[i]
		if int(tok.Start) >= decl.End {
			break
		}
		return tok.Text == "{"
	}
	return false
}

func tokenIndexAt(toks []scan.Token, start int) int {
	for i := 0; i < len(toks); i++ {
		tok := toks[i]
		if int(tok.Start) == start {
			return i
		}
	}
	return -1
}

func declToken(file parse.File, decl parse.Decl, text string) (scan.Token, bool) {
	var tokens []scan.Token
	tokens = file.Tokens
	for i := 0; i < len(tokens); i++ {
		var tok scan.Token
		tok = tokens[i]
		if int(tok.Start) < decl.Start {
			continue
		}
		if int(tok.Start) >= decl.End {
			break
		}
		if tok.Text == text {
			return tok, true
		}
	}
	return scan.Token{}, false
}

func diag(file parse.File, tok scan.Token, message string) Diagnostic {
	return Diagnostic{Path: file.Path, Line: int(tok.Line), Column: int(tok.Column), Message: message}
}

func declDiagnostic(file parse.File, decl parse.Decl, message string) Diagnostic {
	tok := decl.NameTok
	if tok.Text == "" {
		tok = decl.Tok
	}
	return diag(file, tok, message)
}

func declNameDiagnostic(file parse.File, decl parse.Decl, index int, message string) Diagnostic {
	var nameToks []scan.Token
	nameToks = decl.NameToks
	if index >= 0 && index < len(nameToks) {
		var tok scan.Token
		tok = nameToks[index]
		return diag(file, tok, message)
	}
	return declDiagnostic(file, decl, message)
}

func isExported(name string) bool {
	if name == "" {
		return false
	}
	c := name[0]
	return c >= 'A' && c <= 'Z'
}
