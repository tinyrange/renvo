package parse

import (
	"fmt"

	"j5.nz/rtg/rtg/scan"
)

type File struct {
	Path           string
	Source         []byte
	PackageName    string
	Imports        []Import
	Decls          []Decl
	Tokens         []scan.Token
	TopLevelFuncAt map[int]bool
}

type Import struct {
	Path  string
	Alias string
	Tok   scan.Token
}

type Decl struct {
	Kind    string
	Name    string
	Tok     scan.Token
	NameTok scan.Token
	Start   int
	End     int
}

func FileSource(path string, src []byte) (File, error) {
	toks, err := scan.Tokens(src)
	if err != nil {
		return File{}, fmt.Errorf("%s: %w", path, err)
	}
	if len(toks) < 3 || toks[0].Text != "package" || toks[1].Kind != scan.Ident {
		return File{}, fmt.Errorf("%s: missing package declaration", path)
	}
	file := File{
		Path:           path,
		Source:         src,
		PackageName:    toks[1].Text,
		Tokens:         toks,
		TopLevelFuncAt: map[int]bool{},
	}
	pos := 2
	for pos < len(toks) && toks[pos].Text == "import" {
		next, err := parseImportDecl(&file, pos)
		if err != nil {
			return File{}, fmt.Errorf("%s: %w", path, err)
		}
		pos = next
	}
	for pos < len(toks) && toks[pos].Kind != scan.EOF {
		if isTopDecl(toks[pos].Text) {
			next := parseDecl(&file, pos)
			if next <= pos {
				pos++
			} else {
				pos = next
			}
			continue
		}
		return File{}, fmt.Errorf("%s:%d:%d: expected top-level declaration", path, toks[pos].Line, toks[pos].Column)
	}
	return file, nil
}

func parseImportDecl(file *File, pos int) (int, error) {
	toks := file.Tokens
	pos++
	if pos < len(toks) && toks[pos].Text == "(" {
		pos++
		for pos < len(toks) && toks[pos].Text != ")" && toks[pos].Kind != scan.EOF {
			alias := ""
			if toks[pos].Kind == scan.Ident || toks[pos].Text == "." || toks[pos].Text == "_" {
				if pos+1 < len(toks) && toks[pos+1].Kind == scan.String {
					alias = toks[pos].Text
					pos++
				}
			}
			if toks[pos].Kind == scan.String {
				path, err := scan.UnquoteString(toks[pos].Text)
				if err != nil {
					return pos, err
				}
				file.Imports = append(file.Imports, Import{Path: path, Alias: alias, Tok: toks[pos]})
			}
			pos++
		}
		if pos >= len(toks) || toks[pos].Text != ")" {
			return pos, fmt.Errorf("%d:%d: unterminated import block", toks[pos-1].Line, toks[pos-1].Column)
		}
		return pos + 1, nil
	}
	for pos < len(toks) && toks[pos].Kind != scan.EOF {
		alias := ""
		if toks[pos].Kind == scan.Ident || toks[pos].Text == "." || toks[pos].Text == "_" {
			if pos+1 < len(toks) && toks[pos+1].Kind == scan.String {
				alias = toks[pos].Text
				pos++
			}
		}
		if toks[pos].Kind == scan.String {
			path, err := scan.UnquoteString(toks[pos].Text)
			if err != nil {
				return pos, err
			}
			file.Imports = append(file.Imports, Import{Path: path, Alias: alias, Tok: toks[pos]})
			return pos + 1, nil
		}
		if isTopDecl(toks[pos].Text) || toks[pos].Text == "import" {
			break
		}
		pos++
	}
	return pos, fmt.Errorf("%d:%d: malformed import declaration", toks[pos].Line, toks[pos].Column)
}

func parseDecl(file *File, pos int) int {
	toks := file.Tokens
	kind := toks[pos].Text
	decl := Decl{Kind: kind, Tok: toks[pos], Start: toks[pos].Start}
	if kind == "func" {
		file.TopLevelFuncAt[pos] = true
		namePos := pos + 1
		if namePos < len(toks) && toks[namePos].Text == "(" {
			close := skipBalanced(toks, namePos, "(", ")")
			if close > namePos {
				namePos = close + 1
			}
		}
		if namePos < len(toks) && toks[namePos].Kind == scan.Ident {
			decl.Name = toks[namePos].Text
			decl.NameTok = toks[namePos]
		}
		next := findNextTopLevel(toks, pos+1)
		decl.End = declEnd(file, next)
		file.Decls = append(file.Decls, decl)
		return next
	}
	if kind == "type" || kind == "var" || kind == "const" {
		namePos := pos + 1
		if namePos < len(toks) && toks[namePos].Text == "(" {
			close := skipBalanced(toks, namePos, "(", ")")
			if close > namePos {
				next := close + 1
				decl.End = declEnd(file, next)
				file.Decls = append(file.Decls, decl)
				return next
			}
			next := namePos + 1
			decl.End = declEnd(file, next)
			file.Decls = append(file.Decls, decl)
			return next
		}
		if namePos < len(toks) && toks[namePos].Kind == scan.Ident {
			decl.Name = toks[namePos].Text
			decl.NameTok = toks[namePos]
		}
		next := findNextTopLevel(toks, pos+1)
		decl.End = declEnd(file, next)
		file.Decls = append(file.Decls, decl)
		return next
	}
	return pos + 1
}

func DeclText(file File, decl Decl) string {
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
	return string(file.Source[start:end])
}

func declEnd(file *File, next int) int {
	if next >= 0 && next < len(file.Tokens) {
		return file.Tokens[next].Start
	}
	return len(file.Source)
}

func findNextTopLevel(toks []scan.Token, pos int) int {
	paren := 0
	brack := 0
	brace := 0
	for pos < len(toks) {
		text := toks[pos].Text
		if toks[pos].Kind == scan.EOF {
			return pos
		}
		if paren == 0 && brack == 0 && brace == 0 && isTopDecl(text) {
			return pos
		}
		if text == "(" {
			paren++
		} else if text == ")" && paren > 0 {
			paren--
		} else if text == "[" {
			brack++
		} else if text == "]" && brack > 0 {
			brack--
		} else if text == "{" {
			brace++
		} else if text == "}" && brace > 0 {
			brace--
		}
		pos++
	}
	return pos
}

func skipBalanced(toks []scan.Token, pos int, open string, close string) int {
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

func isTopDecl(text string) bool {
	return text == "func" || text == "type" || text == "var" || text == "const"
}
