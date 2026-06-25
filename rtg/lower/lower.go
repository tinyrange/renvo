package lower

import (
	"fmt"
	"sort"

	"j5.nz/rtg/rtg/load"
	"j5.nz/rtg/rtg/parse"
	"j5.nz/rtg/rtg/scan"
	"j5.nz/rtg/rtg/unit"
)

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
			if decl.Name != "" && decl.Name != "_" {
				topNames[decl.Name] = SymbolName(pkg.ImportPath, decl.Name)
			}
		}
	}
	for name, unitName := range topNames {
		if isExported(name) {
			u.Exports = append(u.Exports, unit.Symbol{ImportPath: pkg.ImportPath, Name: name, UnitName: unitName})
		}
	}
	sort.Slice(u.Exports, func(i int, j int) bool {
		return u.Exports[i].Name < u.Exports[j].Name
	})
	importRefs := importReferenceMap(pkg, graph)
	seenRefs := map[string]bool{}
	for _, parsed := range parsedFiles {
		for _, decl := range parsed.Decls {
			var refs []unit.Symbol
			body := rewriteDecl(parsed, decl, topNames, importRefs, &refs)
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
	sort.Slice(u.References, func(i int, j int) bool {
		if u.References[i].ImportPath == u.References[j].ImportPath {
			return u.References[i].Name < u.References[j].Name
		}
		return u.References[i].ImportPath < u.References[j].ImportPath
	})
	return u, nil
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
	localNames := localNamesForDecl(file, decl, topNames)
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
			if symbols, ok := importRefs[tok.Text]; ok {
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
		if tok.Kind == scan.Ident && prevText != "." && !localNames[tok.Text] {
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

func localNamesForDecl(file parse.File, decl parse.Decl, topNames map[string]string) map[string]bool {
	names := map[string]bool{}
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
	collectFuncSignatureLocals(toks, start, body, topNames, names)
	for i := body + 1; i < len(toks) && toks[i].Start < decl.End; i++ {
		if toks[i].Text == ":=" {
			collectShortDeclLocals(toks, i, topNames, names)
			continue
		}
		if toks[i].Text == "var" {
			collectVarLocals(toks, i, decl.End, topNames, names)
		}
	}
	return names
}

func collectFuncSignatureLocals(toks []scan.Token, start int, end int, topNames map[string]string, names map[string]bool) {
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

func collectParameterListLocals(toks []scan.Token, start int, end int, topNames map[string]string, names map[string]bool) {
	for i := start; i < end; i++ {
		if toks[i].Kind != scan.Ident || topNames[toks[i].Text] == "" {
			continue
		}
		if i+1 < end && isTypeStart(toks[i+1]) {
			names[toks[i].Text] = true
			continue
		}
		if i+2 < end && toks[i+1].Text == "," && toks[i+2].Kind == scan.Ident && isTypeStartAfterName(toks, i+2, end) {
			names[toks[i].Text] = true
		}
	}
}

func collectShortDeclLocals(toks []scan.Token, assign int, topNames map[string]string, names map[string]bool) {
	for i := assign - 1; i >= 0; i-- {
		if isStatementBoundary(toks[i].Text) {
			return
		}
		if toks[i].Kind == scan.Ident && topNames[toks[i].Text] != "" && (i == 0 || toks[i-1].Text != ".") {
			names[toks[i].Text] = true
		}
	}
}

func collectVarLocals(toks []scan.Token, pos int, end int, topNames map[string]string, names map[string]bool) {
	if pos+1 < len(toks) && toks[pos+1].Text == "(" {
		for i := pos + 2; i < len(toks) && toks[i].Start < end; i++ {
			if toks[i].Text == ")" || toks[i].Text == "}" {
				return
			}
			if toks[i].Kind != scan.Ident || topNames[toks[i].Text] == "" {
				continue
			}
			if toks[i-1].Text == "(" || toks[i-1].Text == "," || toks[i-1].Line != toks[i].Line {
				names[toks[i].Text] = true
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
			names[toks[i].Text] = true
		}
	}
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

func importReferenceMap(pkg load.Package, graph *load.Graph) map[string]map[string]unit.Symbol {
	refs := map[string]map[string]unit.Symbol{}
	if graph == nil {
		return refs
	}
	packages := map[string]load.Package{}
	for _, dep := range graph.Packages {
		packages[dep.ImportPath] = dep
	}
	for importPath, localName := range pkg.ImportNames {
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
				if isExported(decl.Name) {
					symbols[decl.Name] = unit.Symbol{ImportPath: importPath, Name: decl.Name, UnitName: SymbolName(importPath, decl.Name)}
				}
			}
		}
		refs[localName] = symbols
	}
	return refs
}

func isExported(name string) bool {
	if name == "" {
		return false
	}
	c := name[0]
	return c >= 'A' && c <= 'Z'
}
