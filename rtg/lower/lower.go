package lower

import (
	"fmt"
	"math"
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

type localTypeInfo struct {
	name    string
	pointer bool
}

type methodInfo struct {
	name            string
	receiverType    string
	pointerReceiver bool
	unitName        string
}

func Package(pkg load.Package) (unit.Unit, error) {
	return PackageWithGraph(pkg, nil)
}

func PackageWithGraph(pkg load.Package, graph *load.Graph) (unit.Unit, error) {
	u := unit.Unit{ImportPath: pkg.ImportPath, Package: pkg.Name}
	u.Imports = append(u.Imports, pkg.Imports...)
	files := append([]load.File(nil), pkg.Files...)
	sortFilesByPath(files)
	parsedFiles := make([]parse.File, 0, len(files))
	topNames := map[string]string{}
	methods := map[string]methodInfo{}
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
			for _, name := range declTopNames(parsed, decl) {
				if name != "" && name != "_" {
					topNames[name] = SymbolName(pkg.ImportPath, name)
				}
			}
		}
	}
	for _, parsed := range parsedFiles {
		for _, decl := range parsed.Decls {
			if decl.Kind != "func" || !decl.Receiver {
				continue
			}
			info := methodDeclInfo(parsed, decl)
			if info.name != "" {
				info.unitName = topNames[info.name]
				methods[info.name] = info
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
	sortSymbolsByName(u.Exports)
	depPackages := dependencyPackages(graph)
	seenRefs := map[string]bool{}
	for _, parsed := range parsedFiles {
		importRefs := importReferenceMap(parsed, depPackages)
		for _, decl := range parsed.Decls {
			var refs []unit.Symbol
			body := rewriteDecl(parsed, decl, topNames, importRefs, methods, &refs)
			if decl.Kind == "func" {
				body = normalizeFunctionExpressions(body, unitDeclSymbol(decl, parsed, topNames))
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
				Name:     unitDeclName(parsed, decl),
				UnitName: unitDeclSymbol(decl, parsed, topNames),
				Body:     body,
			})
		}
	}
	if syntheticEntrypoint {
		u.Decls = append(u.Decls, syntheticAppMainDecl(topNames["appMain"], topNames["main"]))
	}
	sortSymbolsByImportPathName(u.References)
	return u, nil
}

func sortFilesByPath(files []load.File) {
	for i := 1; i < len(files); i++ {
		value := files[i]
		j := i - 1
		for j >= 0 && files[j].Path > value.Path {
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
		for j >= 0 && symbols[j].Name > value.Name {
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
		return a.Name > b.Name
	}
	return a.ImportPath > b.ImportPath
}

func unitDeclName(file parse.File, decl parse.Decl) string {
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

func unitDeclSymbol(decl parse.Decl, file parse.File, topNames map[string]string) string {
	if decl.Kind == "func" && decl.Receiver {
		return topNames[methodDeclName(file, decl)]
	}
	names := declNames(decl)
	if len(names) != 1 {
		return ""
	}
	return topNames[names[0]]
}

func declTopNames(file parse.File, decl parse.Decl) []string {
	if decl.Kind == "func" && decl.Receiver {
		name := methodDeclName(file, decl)
		if name == "" {
			return nil
		}
		return []string{name}
	}
	return declNames(decl)
}

func methodDeclName(file parse.File, decl parse.Decl) string {
	info := methodDeclInfo(file, decl)
	if info.receiverType == "" || decl.Name == "" {
		return decl.Name
	}
	return info.name
}

func methodDeclInfo(file parse.File, decl parse.Decl) methodInfo {
	receiver := methodReceiverTypeName(file, decl)
	name := decl.Name
	if receiver != "" && decl.Name != "" {
		name = receiver + "_" + decl.Name
	}
	return methodInfo{
		name:            name,
		receiverType:    receiver,
		pointerReceiver: methodReceiverIsPointer(file, decl),
	}
}

func methodReceiverTypeName(file parse.File, decl parse.Decl) string {
	toks := file.Tokens
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

func methodReceiverIsPointer(file parse.File, decl parse.Decl) bool {
	toks := file.Tokens
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

func rewriteDecl(file parse.File, decl parse.Decl, topNames map[string]string, importRefs map[string]map[string]unit.Symbol, methods map[string]methodInfo, refs *[]unit.Symbol) string {
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
	localTypes := localTypesForDecl(file, decl)
	cursor := start
	if decl.Kind == "func" && decl.Receiver {
		cursor = appendMethodDeclPrefix(file, decl, topNames, &out)
	}
	prevText := ""
	for i := 0; i < len(file.Tokens); i++ {
		tok := file.Tokens[i]
		if tok.End <= cursor {
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
			if i+3 < len(file.Tokens) && file.Tokens[i+3].Text == "(" {
				if receiverType := localTypes[tok.Text]; receiverType.name != "" {
					methodName := receiverType.name + "_" + file.Tokens[i+2].Text
					if method := methods[methodName]; method.unitName != "" {
						open := file.Tokens[i+3]
						close := findClose(file.Tokens, i+3, "(", ")")
						receiverArg := tok.Text
						if method.pointerReceiver && !receiverType.pointer {
							receiverArg = "&" + receiverArg
						}
						replacement = method.unitName + "(" + receiverArg
						if close < 0 || open.End < file.Tokens[close].Start {
							replacement = replacement + ", "
						}
						out = append(out, replacement...)
						cursor = open.End
						prevText = "("
						i += 3
						continue
					}
				}
			}
			if symbols, ok := importRefs[tok.Text]; ok && !isLocalNameAt(localNames, tok.Text, tok.Start) {
				member := file.Tokens[i+2]
				if sym, ok := symbols[member.Text]; ok {
					replacement = sym.UnitName
					if sym.ImportPath != "" {
						*refs = append(*refs, sym)
					}
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

func appendMethodDeclPrefix(file parse.File, decl parse.Decl, topNames map[string]string, out *[]byte) int {
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
	unitName := topNames[methodDeclName(file, decl)]
	if unitName == "" {
		return decl.Start
	}
	*out = append(*out, "func "...)
	*out = append(*out, unitName...)
	*out = append(*out, '(')
	*out = append(*out, rewriteReceiverSegment(file, receiverOpen+1, receiverClose, topNames)...)
	if paramsOpen+1 < paramsClose {
		*out = append(*out, ", "...)
		return toks[paramsOpen].End
	}
	return toks[paramsClose].Start
}

func rewriteReceiverSegment(file parse.File, start int, end int, topNames map[string]string) string {
	toks := file.Tokens
	var out []byte
	cursor := toks[start].Start
	for i := start; i < end; i++ {
		tok := toks[i]
		if tok.Start > cursor {
			out = append(out, file.Source[cursor:tok.Start]...)
		}
		replacement := ""
		if tok.Kind == scan.Ident && (i > start || !receiverSegmentHasName(toks, start, end)) {
			replacement = topNames[tok.Text]
		}
		if replacement != "" {
			out = append(out, replacement...)
		} else {
			out = append(out, file.Source[tok.Start:tok.End]...)
		}
		cursor = tok.End
	}
	if end > start && cursor < toks[end-1].End {
		out = append(out, file.Source[cursor:toks[end-1].End]...)
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

func normalizeFunctionExpressions(body string, unitName string) string {
	toks, err := scan.Tokens([]byte(body))
	if err != nil {
		return body
	}
	var out []byte
	cursor := 0
	tempIndex := 0
	for i := 0; i < len(toks); i++ {
		short, ok := normalizationConditionalShortStatement(toks, i)
		if ok {
			initTemps, initReplacements := normalizeExpression(body, toks, short.initStart, short.semi, unitName, &tempIndex)
			condTemps, condReplacements := normalizeExpression(body, toks, short.condStart, short.condEnd, unitName, &tempIndex)
			insertStart := statementInsertStart(body, toks[short.token].Start)
			out = append(out, body[cursor:insertStart]...)
			indent := statementIndent(body, toks[short.token].Start)
			innerIndent := indent + "\t"
			init := strings.TrimSpace(applyExpressionReplacements(body, toks[short.initStart].Start, toks[short.semi-1].End, initReplacements))
			condition := strings.TrimSpace(applyExpressionReplacements(body, toks[short.condStart].Start, toks[short.condEnd-1].End, condReplacements))
			out = append(out, indent...)
			out = append(out, "{\n"...)
			for _, temp := range initTemps {
				out = append(out, innerIndent...)
				out = append(out, temp.name...)
				out = append(out, " := "...)
				out = append(out, temp.expr...)
				out = append(out, '\n')
			}
			out = append(out, innerIndent...)
			out = append(out, init...)
			out = append(out, '\n')
			for _, temp := range condTemps {
				out = append(out, innerIndent...)
				out = append(out, temp.name...)
				out = append(out, " := "...)
				out = append(out, temp.expr...)
				out = append(out, '\n')
			}
			out = append(out, innerIndent...)
			out = append(out, short.kind...)
			out = append(out, ' ')
			out = append(out, condition...)
			out = append(out, ' ')
			out = append(out, body[toks[short.openBrace].Start:toks[short.end].End]...)
			out = append(out, '\n')
			out = append(out, indent...)
			out = append(out, '}')
			cursor = toks[short.end].End
			i = short.end
			continue
		}
		post, ok := normalizationClassicForPostStatement(toks, i)
		if ok {
			if expressionContainsCall(toks, post.postStart, post.postEnd) {
				initTemps, initReplacements := normalizeExpression(body, toks, post.initStart, post.initEnd, unitName, &tempIndex)
				condTemps, condReplacements := normalizeExpression(body, toks, post.condStart, post.condEnd, unitName, &tempIndex)
				postTemps, postReplacements := normalizeExpression(body, toks, post.postStart, post.postEnd, unitName, &tempIndex)
				insertStart := statementInsertStart(body, toks[post.token].Start)
				out = append(out, body[cursor:insertStart]...)
				indent := statementIndent(body, toks[post.token].Start)
				innerIndent := indent + "\t"
				loopIndent := innerIndent + "\t"
				out = append(out, indent...)
				out = append(out, "{\n"...)
				for _, temp := range initTemps {
					out = append(out, innerIndent...)
					out = append(out, temp.name...)
					out = append(out, " := "...)
					out = append(out, temp.expr...)
					out = append(out, '\n')
				}
				if post.initStart < post.initEnd {
					init := strings.TrimSpace(applyExpressionReplacements(body, toks[post.initStart].Start, toks[post.initEnd-1].End, initReplacements))
					out = append(out, innerIndent...)
					out = append(out, init...)
					out = append(out, '\n')
				}
				out = append(out, innerIndent...)
				out = append(out, "for {\n"...)
				if post.condStart < post.condEnd {
					condition := strings.TrimSpace(applyExpressionReplacements(body, toks[post.condStart].Start, toks[post.condEnd-1].End, condReplacements))
					for _, temp := range condTemps {
						out = append(out, loopIndent...)
						out = append(out, temp.name...)
						out = append(out, " := "...)
						out = append(out, temp.expr...)
						out = append(out, '\n')
					}
					out = append(out, loopIndent...)
					out = append(out, "if !("...)
					out = append(out, condition...)
					out = append(out, ") {\n"...)
					out = append(out, loopIndent...)
					out = append(out, "\tbreak\n"...)
					out = append(out, loopIndent...)
					out = append(out, "}\n"...)
				}
				out = append(out, body[toks[post.openBrace].End:toks[post.end].Start]...)
				if len(out) == 0 || out[len(out)-1] != '\n' {
					out = append(out, '\n')
				}
				for _, temp := range postTemps {
					out = append(out, loopIndent...)
					out = append(out, temp.name...)
					out = append(out, " := "...)
					out = append(out, temp.expr...)
					out = append(out, '\n')
				}
				postExpr := strings.TrimSpace(applyExpressionReplacements(body, toks[post.postStart].Start, toks[post.postEnd-1].End, postReplacements))
				out = append(out, loopIndent...)
				out = append(out, postExpr...)
				out = append(out, '\n')
				out = append(out, innerIndent...)
				out = append(out, "}\n"...)
				out = append(out, indent...)
				out = append(out, '}')
				cursor = toks[post.end].End
				i = post.end
				continue
			}
		}
		classic, ok := normalizationClassicForConditionStatement(toks, i)
		if ok {
			temps, replacements := normalizeExpression(body, toks, classic.condStart, classic.condEnd, unitName, &tempIndex)
			if len(temps) > 0 {
				condition := applyExpressionReplacements(body, toks[classic.condStart].Start, toks[classic.condEnd-1].End, replacements)
				out = append(out, body[cursor:toks[classic.condStart].Start]...)
				out = append(out, body[toks[classic.condEnd].Start:toks[classic.openBrace].End]...)
				indent := statementIndent(body, toks[classic.token].Start)
				innerIndent := indent + "\t"
				out = append(out, '\n')
				for _, temp := range temps {
					out = append(out, innerIndent...)
					out = append(out, temp.name...)
					out = append(out, " := "...)
					out = append(out, temp.expr...)
					out = append(out, '\n')
				}
				out = append(out, innerIndent...)
				out = append(out, "if !("...)
				out = append(out, condition...)
				out = append(out, ") {\n"...)
				out = append(out, innerIndent...)
				out = append(out, "\tbreak\n"...)
				out = append(out, innerIndent...)
				out = append(out, "}\n"...)
				cursor = toks[classic.openBrace].End
				i = classic.openBrace
				continue
			}
		}
		stmt, ok := normalizationStatement(toks, i)
		if !ok {
			continue
		}
		temps, replacements := normalizeExpression(body, toks, stmt.exprStart, stmt.exprEnd, unitName, &tempIndex)
		if len(temps) == 0 {
			continue
		}
		insertStart := statementInsertStart(body, toks[stmt.token].Start)
		out = append(out, body[cursor:insertStart]...)
		indent := statementIndent(body, toks[stmt.token].Start)
		if stmt.kind == "for-condition" {
			innerIndent := indent + "\t"
			condition := applyExpressionReplacements(body, toks[stmt.exprStart].Start, toks[stmt.exprEnd-1].End, replacements)
			out = append(out, body[insertStart:toks[stmt.token].Start]...)
			out = append(out, "for {\n"...)
			for _, temp := range temps {
				out = append(out, innerIndent...)
				out = append(out, temp.name...)
				out = append(out, " := "...)
				out = append(out, temp.expr...)
				out = append(out, '\n')
			}
			out = append(out, innerIndent...)
			out = append(out, "if !("...)
			out = append(out, condition...)
			out = append(out, ") {\n"...)
			out = append(out, innerIndent...)
			out = append(out, "\tbreak\n"...)
			out = append(out, innerIndent...)
			out = append(out, "}\n"...)
			cursor = toks[stmt.openBrace].End
			i = stmt.openBrace
			continue
		}
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
	return conditionalShortStatement{
		token:     pos,
		initStart: exprStart,
		semi:      semi,
		condStart: semi + 1,
		condEnd:   exprEnd,
		openBrace: exprEnd,
		end:       end,
		kind:      toks[pos].Text,
	}, true
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
	return classicForPostStatement{
		token:     pos,
		initStart: exprStart,
		initEnd:   firstSemi,
		condStart: firstSemi + 1,
		condEnd:   secondSemi,
		postStart: secondSemi + 1,
		postEnd:   exprEnd,
		openBrace: exprEnd,
		end:       end,
	}, true
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
	return classicForConditionStatement{
		token:     pos,
		condStart: firstSemi + 1,
		condEnd:   secondSemi,
		openBrace: exprEnd,
	}, true
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
				temps = append(temps, callTemps...)
				replacements = append(replacements, callReplacements...)
				i = close
				continue
			}
		}
		if paren == 0 && brack == 0 && brace == 0 && tok.Text == "[" {
			close := findClose(toks, i, "[", "]")
			if close > i && close < end {
				indexTemps, indexReplacements := normalizeIndexBounds(body, toks, i+1, close, unitName, tempIndex)
				temps = append(temps, indexTemps...)
				replacements = append(replacements, indexReplacements...)
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
	for _, bound := range bounds {
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
				argTemps, argReplacements := normalizeExpression(body, toks, argStart, i, unitName, tempIndex)
				temps = append(temps, argTemps...)
				exprStart := toks[argStart].Start
				exprEnd := toks[i-1].End
				expr := applyExpressionReplacements(body, exprStart, exprEnd, argReplacements)
				if !expressionContainsCall(toks, argStart, i) {
					replacements = append(replacements, argReplacements...)
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

func localTypesForDecl(file parse.File, decl parse.Decl) map[string]localTypeInfo {
	types := map[string]localTypeInfo{}
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
	collectFuncSignatureLocalTypes(toks, start, body, types)
	for i := body + 1; i < len(toks) && toks[i].Start < decl.End; i++ {
		if toks[i].Text == ":=" {
			collectShortDeclLocalTypes(toks, i, types)
			continue
		}
		if toks[i].Text == "var" {
			collectVarLocalTypes(toks, i, decl.End, types)
		}
	}
	return types
}

func collectFuncSignatureLocalTypes(toks []scan.Token, start int, end int, types map[string]localTypeInfo) {
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

func collectParameterListLocalTypes(toks []scan.Token, start int, end int, types map[string]localTypeInfo) {
	for i := start; i < end; i++ {
		if toks[i].Kind != scan.Ident {
			continue
		}
		if i+1 < end && isTypeStart(toks[i+1]) {
			typeStart := i + 1
			typeEnd := parameterTypeEnd(toks, typeStart, end)
			typ := typeNameInRange(toks, typeStart, typeEnd)
			if typ != "" {
				types[toks[i].Text] = localTypeInfo{name: typ, pointer: typeRangeIsPointer(toks, typeStart, typeEnd)}
			}
			continue
		}
		if i+2 < end && toks[i+1].Text == "," && toks[i+2].Kind == scan.Ident && isTypeStartAfterName(toks, i+2, end) {
			typeStart := i + 3
			for typeStart < end && toks[typeStart].Text == "," {
				typeStart++
			}
			typeEnd := parameterTypeEnd(toks, typeStart, end)
			typ := typeNameInRange(toks, typeStart, typeEnd)
			if typ != "" {
				info := localTypeInfo{name: typ, pointer: typeRangeIsPointer(toks, typeStart, typeEnd)}
				types[toks[i].Text] = info
				types[toks[i+2].Text] = info
			}
		}
	}
}

func collectShortDeclLocalTypes(toks []scan.Token, assign int, types map[string]localTypeInfo) {
	if assign-1 < 0 || assign+2 >= len(toks) {
		return
	}
	if toks[assign-1].Kind != scan.Ident {
		return
	}
	if toks[assign+1].Kind == scan.Ident && toks[assign+2].Text == "{" {
		types[toks[assign-1].Text] = localTypeInfo{name: toks[assign+1].Text}
		return
	}
	if assign+3 < len(toks) && toks[assign+1].Text == "&" && toks[assign+2].Kind == scan.Ident && toks[assign+3].Text == "{" {
		types[toks[assign-1].Text] = localTypeInfo{name: toks[assign+2].Text, pointer: true}
	}
}

func collectVarLocalTypes(toks []scan.Token, pos int, end int, types map[string]localTypeInfo) {
	if pos+2 >= len(toks) || toks[pos+1].Kind != scan.Ident {
		return
	}
	if toks[pos+2].Text == "=" {
		if pos+4 < len(toks) && toks[pos+3].Kind == scan.Ident && toks[pos+4].Text == "{" {
			types[toks[pos+1].Text] = localTypeInfo{name: toks[pos+3].Text}
		}
		if pos+5 < len(toks) && toks[pos+3].Text == "&" && toks[pos+4].Kind == scan.Ident && toks[pos+5].Text == "{" {
			types[toks[pos+1].Text] = localTypeInfo{name: toks[pos+4].Text, pointer: true}
		}
		return
	}
	typeStart := pos + 2
	typeEnd := varTypeEnd(toks, typeStart, end)
	typ := typeNameInRange(toks, typeStart, typeEnd)
	if typ != "" {
		types[toks[pos+1].Text] = localTypeInfo{name: typ, pointer: typeRangeIsPointer(toks, typeStart, typeEnd)}
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
	name := ""
	for i := start; i < end; i++ {
		if toks[i].Kind == scan.Ident {
			name = toks[i].Text
		}
	}
	return name
}

func typeRangeIsPointer(toks []scan.Token, start int, end int) bool {
	return containsTokenText(toks, start, end, "*")
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
		for name, intrinsic := range intrinsicImportSymbols(importPath) {
			symbols[name] = unit.Symbol{Name: name, UnitName: intrinsic}
		}
		refs[localName] = symbols
	}
	return refs
}

func intrinsicImportSymbols(importPath string) map[string]string {
	if importPath != "os" {
		return nil
	}
	return map[string]string{
		"Open":     "open",
		"Close":    "close",
		"Read":     "read",
		"Write":    "write",
		"Chmod":    "chmod",
		"O_RDONLY": "O_RDONLY",
		"O_WRONLY": "O_WRONLY",
		"O_RDWR":   "O_RDWR",
		"O_CREATE": "O_CREATE",
		"O_TRUNC":  "O_TRUNC",
		"Stdin":    "0",
		"Stdout":   "1",
		"Stderr":   "2",
	}
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
