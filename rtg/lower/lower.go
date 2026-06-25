package lower

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"j5.nz/rtg/rtg/load"
	"j5.nz/rtg/rtg/parse"
	"j5.nz/rtg/rtg/scan"
	"j5.nz/rtg/rtg/unit"
)

type localRange struct {
	start int
	end   int
}

func Package(pkg load.Package) (unit.Unit, error) {
	return PackageWithGraph(pkg, nil)
}

func PackageWithGraph(pkg load.Package, graph *load.Graph) (unit.Unit, error) {
	u := unit.Unit{ImportPath: pkg.ImportPath, Package: pkg.Name}
	u.Imports = append(u.Imports, pkg.Imports...)
	files := append([]load.File(nil), pkg.Files...)
	sort.Slice(files, func(i int, j int) bool {
		return files[i].Path < files[j].Path
	})
	parsedFiles := make([]parse.File, 0, len(files))
	topNames := map[string]string{}
	for _, file := range files {
		parsed, err := parse.FileSource(file.Path, file.Source)
		if err != nil {
			return unit.Unit{}, err
		}
		if parsed.PackageName != pkg.Name {
			return unit.Unit{}, fmt.Errorf("%s: package name %s does not match loaded package %s", file.Path, parsed.PackageName, pkg.Name)
		}
		parsedFiles = append(parsedFiles, parsed)
		for _, decl := range parsed.Decls {
			for _, name := range declNames(decl) {
				if name != "" && name != "_" {
					topNames[name] = SymbolName(pkg.ImportPath, name)
				}
			}
		}
	}
	syntheticEntrypoint := false
	if pkg.Name == "main" && topNames["appMain"] == "" && topNames["main"] != "" && hasOrdinaryMain(parsedFiles) {
		topNames["appMain"] = SymbolName(pkg.ImportPath, "appMain")
		syntheticEntrypoint = true
	}
	for name, unitName := range topNames {
		if isExported(name) {
			u.Exports = append(u.Exports, unit.Symbol{ImportPath: pkg.ImportPath, Name: name, UnitName: unitName})
		}
	}
	sort.Slice(u.Exports, func(i int, j int) bool {
		return u.Exports[i].Name < u.Exports[j].Name
	})
	depPackages := dependencyPackages(graph)
	seenRefs := map[string]bool{}
	for _, parsed := range parsedFiles {
		importRefs := importReferenceMap(parsed, depPackages)
		for _, decl := range parsed.Decls {
			var refs []unit.Symbol
			body := rewriteDecl(parsed, decl, topNames, importRefs, &refs)
			if decl.Kind == "func" {
				body = normalizeFunctionExpressions(body, topNames[decl.Name])
			}
			for _, ref := range refs {
				key := ref.ImportPath + "\x00" + ref.Name
				if !seenRefs[key] {
					seenRefs[key] = true
					u.References = append(u.References, ref)
				}
			}
			u.Decls = append(u.Decls, unit.Decl{
				Path:     unitPathForDecl(files, parsed.Path),
				Kind:     decl.Kind,
				Name:     decl.Name,
				UnitName: topNames[decl.Name],
				Body:     body,
			})
		}
	}
	if syntheticEntrypoint {
		u.Decls = append(u.Decls, syntheticAppMainDecl(topNames["appMain"], topNames["main"]))
	}
	sort.Slice(u.References, func(i int, j int) bool {
		if u.References[i].ImportPath == u.References[j].ImportPath {
			return u.References[i].Name < u.References[j].Name
		}
		return u.References[i].ImportPath < u.References[j].ImportPath
	})
	return u, nil
}

func hasOrdinaryMain(files []parse.File) bool {
	for _, file := range files {
		for _, decl := range file.Decls {
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

func syntheticAppMainDecl(appMainUnitName string, mainUnitName string) unit.Decl {
	return unit.Decl{
		Path:     "rtg-entrypoint",
		Kind:     "func",
		Name:     "appMain",
		UnitName: appMainUnitName,
		Body:     "func " + appMainUnitName + "() int {\n\t" + mainUnitName + "()\n\treturn 0\n}\n",
	}
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

func unitPathForDecl(files []load.File, path string) string {
	for _, file := range files {
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
	out = append(out, name...)
	return string(out)
}

func rewriteDecl(file parse.File, decl parse.Decl, topNames map[string]string, importRefs map[string]map[string]unit.Symbol, refs *[]unit.Symbol) string {
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
	localNames := localNamesForDecl(file, decl, localRewriteNames(topNames, importRefs))
	cursor := start
	prevText := ""
	for i := 0; i < len(file.Tokens); i++ {
		tok := file.Tokens[i]
		if tok.End <= start {
			prevText = tok.Text
			continue
		}
		if tok.Start >= end {
			break
		}
		if tok.Start > cursor {
			out = append(out, file.Source[cursor:tok.Start]...)
		}
		replacement := ""
		if tok.Kind == scan.Ident && i+2 < len(file.Tokens) && file.Tokens[i+1].Text == "." && file.Tokens[i+2].Kind == scan.Ident {
			if symbols, ok := importRefs[tok.Text]; ok && !isLocalNameAt(localNames, tok.Text, tok.Start) {
				member := file.Tokens[i+2]
				if sym, ok := symbols[member.Text]; ok {
					replacement = sym.UnitName
					*refs = append(*refs, sym)
					out = append(out, replacement...)
					cursor = member.End
					prevText = member.Text
					i += 2
					continue
				}
			}
		}
		if tok.Kind == scan.Ident && prevText != "." && !isLocalNameAt(localNames, tok.Text, tok.Start) {
			replacement = topNames[tok.Text]
		}
		if replacement != "" {
			out = append(out, replacement...)
		} else {
			out = append(out, file.Source[tok.Start:tok.End]...)
		}
		cursor = tok.End
		prevText = tok.Text
	}
	if cursor < end {
		out = append(out, file.Source[cursor:end]...)
	}
	return string(out)
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
}

func normalizeFunctionExpressions(body string, unitName string) string {
	toks, err := scan.Tokens([]byte(body))
	if err != nil {
		return body
	}
	var out []byte
	cursor := 0
	tempIndex := 0
	for i := 0; i < len(toks); i++ {
		stmt, ok := normalizationStatement(toks, i)
		if !ok {
			continue
		}
		temps, replacements := normalizeCallArgumentExpressions(body, toks, stmt.exprStart, stmt.exprEnd, unitName, &tempIndex)
		if len(temps) == 0 {
			continue
		}
		insertStart := statementInsertStart(body, toks[stmt.token].Start)
		out = append(out, body[cursor:insertStart]...)
		indent := statementIndent(body, toks[stmt.token].Start)
		if insertStart == toks[stmt.token].Start {
			out = append(out, '\n')
		}
		for _, temp := range temps {
			out = append(out, indent...)
			out = append(out, temp.name...)
			out = append(out, " := "...)
			out = append(out, temp.expr...)
			out = append(out, '\n')
		}
		out = append(out, body[insertStart:toks[stmt.exprStart].Start]...)
		out = append(out, applyExpressionReplacements(body, toks[stmt.exprStart].Start, toks[stmt.exprEnd-1].End, replacements)...)
		cursor = toks[stmt.exprEnd-1].End
		i = stmt.exprEnd - 1
	}
	if len(out) == 0 {
		return body
	}
	out = append(out, body[cursor:]...)
	return string(out)
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
		return expressionStatement{token: pos, exprStart: exprStart, exprEnd: exprEnd}, true
	}
	if startsCallStatement(toks, pos) {
		exprEnd := lineExpressionEnd(toks, pos)
		if exprEnd <= pos {
			return expressionStatement{}, false
		}
		return expressionStatement{token: pos, exprStart: pos, exprEnd: exprEnd}, true
	}
	if !isAssignmentOperator(toks[pos].Text) {
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

func startsCallStatement(toks []scan.Token, pos int) bool {
	if pos+1 >= len(toks) || toks[pos].Kind != scan.Ident || toks[pos+1].Text != "(" {
		return false
	}
	return statementStartToken(toks, pos) == pos
}

func normalizeCallArgumentExpressions(body string, toks []scan.Token, start int, end int, unitName string, tempIndex *int) ([]expressionTemp, []expressionReplacement) {
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
				temps = append(temps, callTemps...)
				replacements = append(replacements, callReplacements...)
				i = close
				continue
			}
		}
		updateExpressionDepth(tok.Text, &paren, &brack, &brace)
	}
	return temps, replacements
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
			if argStart < i && expressionContainsCall(toks, argStart, i) {
				name := nextExpressionTempName(body, unitName, tempIndex)
				(*tempIndex)++
				exprStart := toks[argStart].Start
				exprEnd := toks[i-1].End
				temps = append(temps, expressionTemp{name: name, expr: body[exprStart:exprEnd]})
				replacements = append(replacements, expressionReplacement{start: exprStart, end: exprEnd, text: name})
			}
			argStart = i + 1
			continue
		}
		updateExpressionDepth(toks[i].Text, &paren, &brack, &brace)
	}
	return temps, replacements
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

func applyExpressionReplacements(body string, start int, end int, replacements []expressionReplacement) string {
	var out []byte
	cursor := start
	for _, repl := range replacements {
		if repl.start < cursor || repl.end > end {
			continue
		}
		out = append(out, body[cursor:repl.start]...)
		out = append(out, repl.text...)
		cursor = repl.end
	}
	out = append(out, body[cursor:end]...)
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

func localRewriteNames(topNames map[string]string, importRefs map[string]map[string]unit.Symbol) map[string]string {
	names := map[string]string{}
	for name, unitName := range topNames {
		names[name] = unitName
	}
	for name := range importRefs {
		names[name] = name
	}
	return names
}

func localNamesForDecl(file parse.File, decl parse.Decl, namesOfInterest map[string]string) map[string][]localRange {
	names := map[string][]localRange{}
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
	collectFuncSignatureLocals(toks, start, body, namesOfInterest, names)
	for i := body + 1; i < len(toks) && toks[i].Start < decl.End; i++ {
		if toks[i].Text == ":=" {
			collectShortDeclLocals(toks, body, i, decl.End, namesOfInterest, names)
			continue
		}
		if toks[i].Text == "var" {
			collectVarLocals(toks, body, i, decl.End, namesOfInterest, names)
		}
	}
	return names
}

func isLocalNameAt(names map[string][]localRange, name string, pos int) bool {
	for _, scope := range names[name] {
		if pos >= scope.start && pos < scope.end {
			return true
		}
	}
	return false
}

func collectFuncSignatureLocals(toks []scan.Token, start int, end int, topNames map[string]string, names map[string][]localRange) {
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

func collectParameterListLocals(toks []scan.Token, start int, end int, topNames map[string]string, names map[string][]localRange) {
	for i := start; i < end; i++ {
		if toks[i].Kind != scan.Ident || topNames[toks[i].Text] == "" {
			continue
		}
		if i+1 < end && isTypeStart(toks[i+1]) {
			addLocalName(names, toks[i].Text, 0, math.MaxInt)
			continue
		}
		if i+2 < end && toks[i+1].Text == "," && toks[i+2].Kind == scan.Ident && isTypeStartAfterName(toks, i+2, end) {
			addLocalName(names, toks[i].Text, 0, math.MaxInt)
		}
	}
}

func collectShortDeclLocals(toks []scan.Token, body int, assign int, declEnd int, topNames map[string]string, names map[string][]localRange) {
	line := toks[assign].Line
	scopeEnd := localScopeEnd(toks, body, assign, declEnd)
	for i := assign - 1; i >= 0; i-- {
		if toks[i].Line != line {
			return
		}
		if isStatementBoundary(toks[i].Text) {
			return
		}
		if toks[i].Kind == scan.Ident && topNames[toks[i].Text] != "" && (i == 0 || toks[i-1].Text != ".") {
			addLocalName(names, toks[i].Text, toks[i].Start, scopeEnd)
		}
	}
}

func collectVarLocals(toks []scan.Token, body int, pos int, end int, topNames map[string]string, names map[string][]localRange) {
	scopeEnd := localScopeEnd(toks, body, pos, end)
	if pos+1 < len(toks) && toks[pos+1].Text == "(" {
		for i := pos + 2; i < len(toks) && toks[i].Start < end; i++ {
			if toks[i].Text == ")" || toks[i].Text == "}" {
				return
			}
			if toks[i].Kind != scan.Ident || topNames[toks[i].Text] == "" {
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
		if toks[i].Kind != scan.Ident || topNames[toks[i].Text] == "" {
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

func addLocalName(names map[string][]localRange, name string, start int, end int) {
	names[name] = append(names[name], localRange{start: start, end: end})
}

func tokenIndexAt(toks []scan.Token, start int) int {
	for i, tok := range toks {
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

func dependencyPackages(graph *load.Graph) map[string]load.Package {
	packages := map[string]load.Package{}
	if graph == nil {
		return packages
	}
	for _, dep := range graph.Packages {
		packages[dep.ImportPath] = dep
	}
	return packages
}

func importReferenceMap(file parse.File, packages map[string]load.Package) map[string]map[string]unit.Symbol {
	refs := map[string]map[string]unit.Symbol{}
	for _, imp := range file.Imports {
		localName := importLocalName(imp)
		importPath := imp.Path
		dep, ok := packages[importPath]
		if !ok || localName == "" {
			continue
		}
		symbols := map[string]unit.Symbol{}
		for _, file := range dep.Files {
			parsed, err := parse.FileSource(file.Path, file.Source)
			if err != nil {
				continue
			}
			for _, decl := range parsed.Decls {
				for _, name := range declNames(decl) {
					if isExported(name) {
						symbols[name] = unit.Symbol{ImportPath: importPath, Name: name, UnitName: SymbolName(importPath, name)}
					}
				}
			}
		}
		refs[localName] = symbols
	}
	return refs
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
