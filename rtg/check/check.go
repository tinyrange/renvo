package check

import (
	"strconv"
	"strings"

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

type sourceRange struct {
	start int
	end   int
}

func (d Diagnostics) Error() string {
	if len(d) == 0 {
		return ""
	}
	var parts []string
	for _, diag := range d {
		parts = append(parts, diag.Error())
	}
	return strings.Join(parts, "\n")
}

func Graph(g *load.Graph) error {
	var diags Diagnostics
	exported := exportedDecls(g)
	for _, pkg := range g.Packages {
		names := map[string]Diagnostic{}
		for _, file := range pkg.Files {
			parsed, err := parse.FileSource(file.Path, file.Source)
			if err != nil {
				diags = append(diags, parseDiagnostic(file.Path, err))
				continue
			}
			if parsed.PackageName != pkg.Name {
				diags = append(diags, Diagnostic{Path: file.Path, Line: 1, Column: 1, Message: "package name changed during parsing"})
				continue
			}
			diags = append(diags, File(parsed)...)
			diags = append(diags, importedSelectorDiagnostics(parsed, exported)...)
			for _, decl := range parsed.Decls {
				for i, name := range declNames(decl) {
					if name == "" || name == "_" {
						continue
					}
					current := declNameDiagnostic(parsed, decl, i, "duplicate package-level declaration: "+name)
					if previous, ok := names[name]; ok {
						diags = append(diags, previous)
						diags = append(diags, current)
						continue
					}
					names[name] = current
				}
			}
		}
	}
	if len(diags) > 0 {
		return diags
	}
	return nil
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
	for _, decl := range file.Decls {
		if decl.Kind == "func" && decl.Receiver {
			diags = append(diags, declDiagnostic(file, decl, "methods are not supported"))
		}
		if decl.Kind == "func" {
			if decl.Name == "init" {
				diags = append(diags, declDiagnostic(file, decl, "init functions are not supported"))
			}
			if tok, ok := namedResultToken(file, decl); ok {
				diags = append(diags, diag(file, tok, "named result parameters are not supported"))
			}
		}
		if decl.Kind == "const" {
			if tok, ok := declToken(file, decl, "iota"); ok {
				diags = append(diags, diag(file, tok, "iota is not supported"))
			}
		}
		if file.PackageName == "main" && decl.Kind == "func" && decl.Name == "main" && !hasOrdinaryMainSignature(file, decl) {
			diags = append(diags, declDiagnostic(file, decl, "main function must have no parameters or results"))
		}
	}
	return diags
}

func exportedDecls(g *load.Graph) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	for _, pkg := range g.Packages {
		names := map[string]bool{}
		for _, file := range pkg.Files {
			parsed, err := parse.FileSource(file.Path, file.Source)
			if err != nil {
				continue
			}
			for _, decl := range parsed.Decls {
				for _, name := range declNames(decl) {
					if isExported(name) {
						names[name] = true
					}
				}
			}
		}
		out[pkg.ImportPath] = names
	}
	return out
}

func declNames(decl parse.Decl) []string {
	if len(decl.Names) > 0 {
		return decl.Names
	}
	if decl.Name == "" {
		return nil
	}
	return []string{decl.Name}
}

func importedSelectorDiagnostics(file parse.File, exported map[string]map[string]bool) Diagnostics {
	localImports := map[string]string{}
	importNames := map[string]bool{}
	for _, imp := range file.Imports {
		localName := importLocalName(imp)
		if localName != "" {
			localImports[localName] = imp.Path
			importNames[localName] = true
		}
	}
	if len(localImports) == 0 {
		return nil
	}
	shadows := localImportShadows(file, importNames)
	var diags Diagnostics
	for i := 0; i+2 < len(file.Tokens); i++ {
		local := file.Tokens[i]
		dot := file.Tokens[i+1]
		member := file.Tokens[i+2]
		if local.Kind != scan.Ident || dot.Text != "." || member.Kind != scan.Ident {
			continue
		}
		importPath, ok := localImports[local.Text]
		if !ok {
			continue
		}
		if isLocalShadowAt(shadows, local.Text, local.Start) {
			continue
		}
		if exported[importPath][member.Text] {
			continue
		}
		diags = append(diags, diag(file, member, "unresolved imported selector: "+importPath+"."+member.Text))
	}
	return diags
}

func File(file parse.File) Diagnostics {
	var diags Diagnostics
	diags = append(diags, importDiagnostics(file)...)
	diags = append(diags, declDiagnostics(file)...)
	topFuncs := file.TopLevelFuncAt
	for i, tok := range file.Tokens {
		if tok.Kind == scan.EOF {
			break
		}
		if tok.Kind == scan.String && strings.HasPrefix(tok.Text, "`") {
			diags = append(diags, diag(file, tok, "raw string literals are not supported"))
		}
		if tok.Kind == scan.Number && isImaginaryLiteral(tok.Text) {
			diags = append(diags, diag(file, tok, "imaginary literals are not supported"))
		}
		if tok.Kind == scan.Number && isOctalLiteral(tok.Text) {
			diags = append(diags, diag(file, tok, "octal literals are not supported"))
		}
		switch tok.Text {
		case "...":
			diags = append(diags, diag(file, tok, "variadic syntax is not supported"))
		case "go":
			diags = append(diags, diag(file, tok, "goroutines are not supported"))
		case "chan", "<-":
			diags = append(diags, diag(file, tok, "channels are not supported"))
		case "select":
			diags = append(diags, diag(file, tok, "select statements are not supported"))
		case "interface":
			diags = append(diags, diag(file, tok, "interfaces are not supported"))
		case "map":
			diags = append(diags, diag(file, tok, "maps are not supported"))
		case "defer":
			diags = append(diags, diag(file, tok, "defer is not supported"))
		case "range":
			diags = append(diags, diag(file, tok, "range is not supported"))
		case "fallthrough":
			diags = append(diags, diag(file, tok, "fallthrough is not supported"))
		case "func":
			if !topFuncs[i] {
				diags = append(diags, diag(file, tok, "function values and function types are not supported"))
			}
		}
		if startsArrayType(file.Tokens, i) {
			diags = append(diags, diag(file, tok, "arrays are not supported"))
		}
		if startsAnyInterfaceType(file.Tokens, i) {
			diags = append(diags, diag(file, tok, "interfaces are not supported"))
		}
		if startsGenericDecl(file.Tokens, i, topFuncs) {
			diags = append(diags, diag(file, file.Tokens[i+2], "generics are not supported"))
		}
		if startsGenericInstantiation(file.Tokens, i) {
			diags = append(diags, diag(file, file.Tokens[i+1], "generics are not supported"))
		}
		if startsTypeAssertion(file.Tokens, i) {
			diags = append(diags, diag(file, file.Tokens[i+1], "type assertions and type switches are not supported"))
		}
		if colon := fullSliceSecondColon(file.Tokens, i); colon >= 0 {
			diags = append(diags, diag(file, file.Tokens[colon], "full slice expressions are not supported"))
		}
		if startsUnsupportedBuiltinCall(file.Tokens, i) {
			diags = append(diags, diag(file, tok, "unsupported builtin: "+tok.Text))
		}
	}
	return diags
}

func importDiagnostics(file parse.File) Diagnostics {
	var diags Diagnostics
	names := map[string]scan.Token{}
	importNames := map[string]bool{}
	for _, imp := range file.Imports {
		localName := importLocalName(imp)
		if localName != "" && localName != "." && localName != "_" {
			importNames[localName] = true
		}
	}
	used := usedImportNames(file, importNames)
	for _, imp := range file.Imports {
		if imp.Alias == "." {
			diags = append(diags, diag(file, imp.Tok, "dot imports are not supported"))
		}
		if imp.Alias == "_" {
			diags = append(diags, diag(file, imp.Tok, "blank imports are not supported"))
		}
		localName := importLocalName(imp)
		if localName == "" || localName == "." || localName == "_" {
			continue
		}
		if _, ok := names[localName]; ok {
			diags = append(diags, diag(file, imp.Tok, "duplicate import name: "+localName))
			continue
		}
		names[localName] = imp.Tok
		if !used[localName] {
			diags = append(diags, diag(file, imp.Tok, "unused import: "+localName))
		}
	}
	return diags
}

func usedImportNames(file parse.File, importNames map[string]bool) map[string]bool {
	used := map[string]bool{}
	shadows := localImportShadows(file, importNames)
	for i := 0; i+1 < len(file.Tokens); i++ {
		if file.Tokens[i].Kind == scan.Ident && file.Tokens[i+1].Text == "." {
			if isLocalShadowAt(shadows, file.Tokens[i].Text, file.Tokens[i].Start) {
				continue
			}
			used[file.Tokens[i].Text] = true
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

func startsGenericDecl(toks []scan.Token, i int, topFuncs map[int]bool) bool {
	if i+2 >= len(toks) {
		return false
	}
	if toks[i].Text == "type" && toks[i+1].Kind == scan.Ident && toks[i+2].Text == "[" {
		close := findClose(toks, i+2, "[", "]")
		return close > i+4
	}
	if toks[i].Text == "func" && topFuncs[i] {
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
	if i+3 >= len(toks) || toks[i].Kind != scan.Ident || toks[i+1].Text != "[" {
		return false
	}
	close := findClose(toks, i+1, "[", "]")
	if close < 0 || close+1 >= len(toks) {
		return false
	}
	return toks[close+1].Text == "{" || toks[close+1].Text == "("
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
	if i+1 >= len(toks) || toks[i].Text != "[" || toks[i+1].Text == "]" {
		return false
	}
	if i == 0 {
		return false
	}
	prev := toks[i-1]
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
	if toks[i].Text != "any" || i == 0 {
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
	if prev.Kind != scan.Ident || i < 2 {
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
	if i+2 >= len(toks) || toks[i].Text != "." || toks[i+1].Text != "(" {
		return false
	}
	close := findClose(toks, i+1, "(", ")")
	return close > i+2
}

func fullSliceSecondColon(toks []scan.Token, i int) int {
	if i >= len(toks) || toks[i].Text != "[" {
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
	if i+1 >= len(toks) || toks[i].Kind != scan.Ident || toks[i+1].Text != "(" {
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
	case "var", "type", "{", "*", "]", ")":
		return true
	}
	if prev.Kind == scan.Ident && pos >= 2 {
		if isKeyword(prev.Text) {
			return false
		}
		beforeName := toks[pos-2].Text
		if beforeName == "var" || beforeName == "type" || beforeName == "{" {
			return true
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

func localImportShadows(file parse.File, importNames map[string]bool) map[string][]sourceRange {
	shadows := map[string][]sourceRange{}
	if len(importNames) == 0 {
		return shadows
	}
	for _, decl := range file.Decls {
		if decl.Kind != "func" {
			continue
		}
		start := tokenIndexAt(file.Tokens, decl.Start)
		if start < 0 {
			continue
		}
		body := findTokenText(file.Tokens, start, decl.End, "{")
		if body < 0 {
			continue
		}
		collectFuncSignatureImportShadows(file.Tokens, start, body, importNames, shadows)
		for i := body + 1; i < len(file.Tokens) && file.Tokens[i].Start < decl.End; i++ {
			if file.Tokens[i].Text == ":=" {
				collectShortDeclImportShadows(file.Tokens, body, i, decl.End, importNames, shadows)
			}
			if file.Tokens[i].Text == "var" {
				collectVarImportShadows(file.Tokens, body, i, decl.End, importNames, shadows)
			}
		}
	}
	return shadows
}

func collectFuncSignatureImportShadows(toks []scan.Token, start int, end int, names map[string]bool, shadows map[string][]sourceRange) {
	for i := start; i < end; i++ {
		if toks[i].Text != "(" {
			continue
		}
		close := findClose(toks, i, "(", ")")
		if close < 0 || close > end {
			continue
		}
		collectParameterImportShadows(toks, i+1, close, names, shadows)
		i = close
	}
}

func collectParameterImportShadows(toks []scan.Token, start int, end int, names map[string]bool, shadows map[string][]sourceRange) {
	for i := start; i < end; i++ {
		if toks[i].Kind != scan.Ident || !names[toks[i].Text] {
			continue
		}
		if i+1 < end && isTypeStart(toks[i+1]) {
			addLocalShadow(shadows, toks[i].Text, 0, maxSourcePosition())
			continue
		}
		if i+2 < end && toks[i+1].Text == "," && toks[i+2].Kind == scan.Ident && isTypeStartAfterName(toks, i+2, end) {
			addLocalShadow(shadows, toks[i].Text, 0, maxSourcePosition())
		}
	}
}

func collectShortDeclImportShadows(toks []scan.Token, body int, assign int, declEnd int, names map[string]bool, shadows map[string][]sourceRange) {
	line := toks[assign].Line
	scopeEnd := localScopeEnd(toks, body, assign, declEnd)
	for i := assign - 1; i >= 0; i-- {
		if toks[i].Line != line || isStatementBoundary(toks[i].Text) {
			return
		}
		if toks[i].Kind == scan.Ident && names[toks[i].Text] && (i == 0 || toks[i-1].Text != ".") {
			addLocalShadow(shadows, toks[i].Text, toks[i].Start, scopeEnd)
		}
	}
}

func collectVarImportShadows(toks []scan.Token, body int, pos int, end int, names map[string]bool, shadows map[string][]sourceRange) {
	scopeEnd := localScopeEnd(toks, body, pos, end)
	if pos+1 < len(toks) && toks[pos+1].Text == "(" {
		for i := pos + 2; i < len(toks) && toks[i].Start < end; i++ {
			if toks[i].Text == ")" || toks[i].Text == "}" {
				return
			}
			if toks[i].Kind == scan.Ident && names[toks[i].Text] && (toks[i-1].Text == "(" || toks[i-1].Text == "," || toks[i-1].Line != toks[i].Line) {
				addLocalShadow(shadows, toks[i].Text, toks[i].Start, scopeEnd)
			}
		}
		return
	}
	line := toks[pos].Line
	for i := pos + 1; i < len(toks) && toks[i].Start < end && toks[i].Line == line; i++ {
		if toks[i].Text == ")" || toks[i].Text == "}" || toks[i].Text == ":=" || toks[i].Text == "=" {
			return
		}
		if toks[i].Kind == scan.Ident && names[toks[i].Text] && (i == pos+1 || toks[i-1].Text == ",") {
			addLocalShadow(shadows, toks[i].Text, toks[i].Start, scopeEnd)
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

func addLocalShadow(shadows map[string][]sourceRange, name string, start int, end int) {
	shadows[name] = append(shadows[name], sourceRange{start: start, end: end})
}

func isLocalShadowAt(shadows map[string][]sourceRange, name string, pos int) bool {
	for _, r := range shadows[name] {
		if pos >= r.start && pos < r.end {
			return true
		}
	}
	return false
}

func findTokenText(toks []scan.Token, start int, end int, text string) int {
	for i := start; i < len(toks) && toks[i].Start < end; i++ {
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
	return int(^uint(0) >> 1)
}

func namedResultToken(file parse.File, decl parse.Decl) (scan.Token, bool) {
	name := tokenIndexAt(file.Tokens, decl.NameTok.Start)
	if name < 0 || name+1 >= len(file.Tokens) || file.Tokens[name+1].Text != "(" {
		return scan.Token{}, false
	}
	paramsClose := findClose(file.Tokens, name+1, "(", ")")
	if paramsClose < 0 || paramsClose+1 >= len(file.Tokens) || file.Tokens[paramsClose+1].Text != "(" {
		return scan.Token{}, false
	}
	resultsOpen := paramsClose + 1
	resultsClose := findClose(file.Tokens, resultsOpen, "(", ")")
	if resultsClose < 0 || file.Tokens[resultsClose].Start >= decl.End {
		return scan.Token{}, false
	}
	for i := resultsOpen + 1; i < resultsClose; i++ {
		if file.Tokens[i].Kind == scan.Ident && isTypeStartAfterName(file.Tokens, i, resultsClose) {
			return file.Tokens[i], true
		}
	}
	return scan.Token{}, false
}

func hasOrdinaryMainSignature(file parse.File, decl parse.Decl) bool {
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
		return file.Tokens[i].Text == "{"
	}
	return false
}

func tokenIndexAt(toks []scan.Token, start int) int {
	for i, tok := range toks {
		if tok.Start == start {
			return i
		}
	}
	return -1
}

func declToken(file parse.File, decl parse.Decl, text string) (scan.Token, bool) {
	for _, tok := range file.Tokens {
		if tok.Start < decl.Start {
			continue
		}
		if tok.Start >= decl.End {
			break
		}
		if tok.Text == text {
			return tok, true
		}
	}
	return scan.Token{}, false
}

func diag(file parse.File, tok scan.Token, message string) Diagnostic {
	return Diagnostic{Path: file.Path, Line: tok.Line, Column: tok.Column, Message: message}
}

func declDiagnostic(file parse.File, decl parse.Decl, message string) Diagnostic {
	tok := decl.NameTok
	if tok.Text == "" {
		tok = decl.Tok
	}
	return diag(file, tok, message)
}

func declNameDiagnostic(file parse.File, decl parse.Decl, index int, message string) Diagnostic {
	if index >= 0 && index < len(decl.NameToks) {
		return diag(file, decl.NameToks[index], message)
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
