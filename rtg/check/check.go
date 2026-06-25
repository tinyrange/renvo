package check

import (
	"fmt"
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
	return fmt.Sprintf("%s:%d:%d: %s", d.Path, d.Line, d.Column, d.Message)
}

type Diagnostics []Diagnostic

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
				diags = append(diags, Diagnostic{Path: file.Path, Line: 1, Column: 1, Message: err.Error()})
				continue
			}
			if parsed.PackageName != pkg.Name {
				diags = append(diags, Diagnostic{Path: file.Path, Line: 1, Column: 1, Message: "package name changed during parsing"})
				continue
			}
			diags = append(diags, File(parsed)...)
			diags = append(diags, importedSelectorDiagnostics(pkg, parsed, exported)...)
			for _, decl := range parsed.Decls {
				if decl.Name == "" || decl.Name == "_" {
					continue
				}
				current := declDiagnostic(parsed, decl, "duplicate package-level declaration: "+decl.Name)
				if previous, ok := names[decl.Name]; ok {
					diags = append(diags, previous)
					diags = append(diags, current)
					continue
				}
				names[decl.Name] = current
			}
		}
	}
	if len(diags) > 0 {
		return diags
	}
	return nil
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
				if isExported(decl.Name) {
					names[decl.Name] = true
				}
			}
		}
		out[pkg.ImportPath] = names
	}
	return out
}

func importedSelectorDiagnostics(pkg load.Package, file parse.File, exported map[string]map[string]bool) Diagnostics {
	localImports := map[string]string{}
	for importPath, localName := range pkg.ImportNames {
		if localName != "" {
			localImports[localName] = importPath
		}
	}
	if len(localImports) == 0 {
		return nil
	}
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
	topFuncs := file.TopLevelFuncAt
	for i, tok := range file.Tokens {
		if tok.Kind == scan.EOF {
			break
		}
		switch tok.Text {
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
		case "func":
			if !topFuncs[i] {
				diags = append(diags, diag(file, tok, "function values and function types are not supported"))
			}
		}
		if startsArrayType(file.Tokens, i) {
			diags = append(diags, diag(file, tok, "arrays are not supported"))
		}
		if startsGenericDecl(file.Tokens, i, topFuncs) {
			diags = append(diags, diag(file, file.Tokens[i+2], "generics are not supported"))
		}
		if startsTypeAssertion(file.Tokens, i) {
			diags = append(diags, diag(file, file.Tokens[i+1], "type assertions and type switches are not supported"))
		}
	}
	return diags
}

func importDiagnostics(file parse.File) Diagnostics {
	var diags Diagnostics
	for _, imp := range file.Imports {
		if imp.Alias == "." {
			diags = append(diags, diag(file, imp.Tok, "dot imports are not supported"))
		}
		if imp.Alias == "_" {
			diags = append(diags, diag(file, imp.Tok, "blank imports are not supported"))
		}
	}
	return diags
}

func startsGenericDecl(toks []scan.Token, i int, topFuncs map[int]bool) bool {
	if i+2 >= len(toks) {
		return false
	}
	if toks[i].Text == "type" && toks[i+1].Kind == scan.Ident && toks[i+2].Text == "[" {
		return true
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
		return toks[namePos].Kind == scan.Ident && toks[namePos+1].Text == "["
	}
	return false
}

func startsArrayType(toks []scan.Token, i int) bool {
	if i+1 >= len(toks) || toks[i].Text != "[" || toks[i+1].Text == "]" {
		return false
	}
	if i == 0 {
		return false
	}
	prev := toks[i-1]
	if prev.Text == ")" {
		return true
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

func startsTypeAssertion(toks []scan.Token, i int) bool {
	if i+3 >= len(toks) {
		return false
	}
	return toks[i].Text == "." && toks[i+1].Text == "(" && toks[i+2].Text == "type" && toks[i+3].Text == ")"
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

func isExported(name string) bool {
	if name == "" {
		return false
	}
	c := name[0]
	return c >= 'A' && c <= 'Z'
}
