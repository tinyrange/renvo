package parse

import (
	"fmt"
	"strconv"
	"strings"

	"j5.nz/rtg/rtg/scan"
)

type Error struct {
	Path    string
	Line    int
	Column  int
	Message string
}

func (e Error) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s", e.Path, e.Line, e.Column, e.Message)
}

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
	Kind     string
	Name     string
	Names    []string
	Tok      scan.Token
	NameTok  scan.Token
	NameToks []scan.Token
	Receiver bool
	Start    int
	End      int
}

func FileSource(path string, src []byte) (File, error) {
	toks, err := scan.Tokens(src)
	if err != nil {
		return File{}, scannerError(path, err)
	}
	if len(toks) < 3 || toks[0].Text != "package" || toks[1].Kind != scan.Ident {
		return File{}, Error{Path: path, Line: 1, Column: 1, Message: "missing package declaration"}
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
			return File{}, err
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
		return File{}, Error{Path: path, Line: toks[pos].Line, Column: toks[pos].Column, Message: "expected top-level declaration"}
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
					return pos, Error{Path: file.Path, Line: toks[pos].Line, Column: toks[pos].Column, Message: err.Error()}
				}
				file.Imports = append(file.Imports, Import{Path: path, Alias: alias, Tok: toks[pos]})
				pos++
				continue
			}
			return pos, Error{Path: file.Path, Line: toks[pos].Line, Column: toks[pos].Column, Message: "malformed import declaration"}
		}
		if pos >= len(toks) || toks[pos].Text != ")" {
			return pos, Error{Path: file.Path, Line: toks[pos-1].Line, Column: toks[pos-1].Column, Message: "unterminated import block"}
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
				return pos, Error{Path: file.Path, Line: toks[pos].Line, Column: toks[pos].Column, Message: err.Error()}
			}
			file.Imports = append(file.Imports, Import{Path: path, Alias: alias, Tok: toks[pos]})
			return pos + 1, nil
		}
		if isTopDecl(toks[pos].Text) || toks[pos].Text == "import" {
			break
		}
		pos++
	}
	return pos, Error{Path: file.Path, Line: toks[pos].Line, Column: toks[pos].Column, Message: "malformed import declaration"}
}

func scannerError(path string, err error) Error {
	line, column, message, ok := splitPositionMessage(err.Error())
	if !ok {
		return Error{Path: path, Line: 1, Column: 1, Message: err.Error()}
	}
	return Error{Path: path, Line: line, Column: column, Message: message}
}

func splitPositionMessage(message string) (int, int, string, bool) {
	first := strings.IndexByte(message, ':')
	if first < 0 {
		return 0, 0, "", false
	}
	second := strings.IndexByte(message[first+1:], ':')
	if second < 0 {
		return 0, 0, "", false
	}
	second += first + 1
	line, err := strconv.Atoi(message[:first])
	if err != nil {
		return 0, 0, "", false
	}
	column, err := strconv.Atoi(message[first+1 : second])
	if err != nil {
		return 0, 0, "", false
	}
	return line, column, strings.TrimSpace(message[second+1:]), true
}

func parseDecl(file *File, pos int) int {
	toks := file.Tokens
	kind := toks[pos].Text
	decl := Decl{Kind: kind, Tok: toks[pos], Start: toks[pos].Start}
	if kind == "func" {
		file.TopLevelFuncAt[pos] = true
		namePos := pos + 1
		if namePos < len(toks) && toks[namePos].Text == "(" {
			decl.Receiver = true
			close := skipBalanced(toks, namePos, "(", ")")
			if close > namePos {
				namePos = close + 1
			}
		}
		if namePos < len(toks) && toks[namePos].Kind == scan.Ident {
			decl.Name = toks[namePos].Text
			decl.Names = []string{decl.Name}
			decl.NameTok = toks[namePos]
			decl.NameToks = []scan.Token{decl.NameTok}
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
				decl.Names, decl.NameToks = groupedDeclNames(toks, namePos, close)
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
			decl.Names = []string{decl.Name}
			decl.NameTok = toks[namePos]
			decl.NameToks = []scan.Token{decl.NameTok}
		}
		next := findNextTopLevel(toks, pos+1)
		decl.End = declEnd(file, next)
		file.Decls = append(file.Decls, decl)
		return next
	}
	return pos + 1
}

func groupedDeclNames(toks []scan.Token, open int, close int) ([]string, []scan.Token) {
	var names []string
	var nameToks []scan.Token
	line := -1
	expectName := true
	paren := 0
	brack := 0
	brace := 0
	for i := open + 1; i < close; i++ {
		tok := toks[i]
		if tok.Text == "(" {
			paren++
			continue
		}
		if tok.Text == ")" && paren > 0 {
			paren--
			continue
		}
		if tok.Text == "[" {
			brack++
			continue
		}
		if tok.Text == "]" && brack > 0 {
			brack--
			continue
		}
		if tok.Text == "{" {
			brace++
			continue
		}
		if tok.Text == "}" && brace > 0 {
			brace--
			continue
		}
		if paren > 0 || brack > 0 || brace > 0 {
			continue
		}
		if tok.Line != line {
			line = tok.Line
			expectName = true
		}
		if tok.Text == "," {
			expectName = true
			continue
		}
		if tok.Text == "=" {
			expectName = false
			continue
		}
		if tok.Kind == scan.Ident && expectName {
			names = append(names, tok.Text)
			nameToks = append(nameToks, tok)
			expectName = false
			continue
		}
		expectName = false
	}
	return names, nameToks
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
