package parse

import (
	"strconv"
	"strings"

	"j5.nz/rtg/rtg/scan"
)

type Error string

func (e Error) Error() string {
	return string(e)
}

func newError(path string, line int, column int, message string) Error {
	return Error(path + ":" + strconv.Itoa(line) + ":" + strconv.Itoa(column) + ": " + message)
}

type File struct {
	Path          string
	Source        []byte
	PackageName   string
	Imports       []Import
	Decls         []Decl
	Tokens        []scan.Token
	TopLevelFuncs []int
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
		return File{}, newError(path, 1, 1, "missing package declaration")
	}
	file := File{
		Path:        path,
		Source:      src,
		PackageName: toks[1].Text,
		Tokens:      toks,
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
		return File{}, newError(path, toks[pos].Line, toks[pos].Column, "expected top-level declaration")
	}
	return file, nil
}

func (file File) IsTopLevelFuncAt(pos int) bool {
	tokens := file.Tokens
	if pos >= 0 && pos < len(tokens) {
		if tokens[pos].Text == "func" && tokens[pos].Column == 1 {
			return true
		}
		start := tokens[pos].Start
		decls := file.Decls
		for i := 0; i < len(decls); i++ {
			decl := decls[i]
			if decl.Kind == "func" && decl.Start == start {
				return true
			}
		}
	}
	topLevelFuncs := file.TopLevelFuncs
	for i := 0; i < len(topLevelFuncs); i++ {
		if topLevelFuncs[i] == pos {
			return true
		}
	}
	return false
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
					return pos, newError(file.Path, toks[pos].Line, toks[pos].Column, err.Error())
				}
				file.Imports = append(file.Imports, Import{Path: path, Alias: alias, Tok: toks[pos]})
				pos++
				continue
			}
			return pos, newError(file.Path, toks[pos].Line, toks[pos].Column, "malformed import declaration")
		}
		if pos >= len(toks) || toks[pos].Text != ")" {
			return pos, newError(file.Path, toks[pos-1].Line, toks[pos-1].Column, "unterminated import block")
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
				return pos, newError(file.Path, toks[pos].Line, toks[pos].Column, err.Error())
			}
			file.Imports = append(file.Imports, Import{Path: path, Alias: alias, Tok: toks[pos]})
			return pos + 1, nil
		}
		if isTopDecl(toks[pos].Text) || toks[pos].Text == "import" {
			break
		}
		pos++
	}
	return pos, newError(file.Path, toks[pos].Line, toks[pos].Column, "malformed import declaration")
}

func scannerError(path string, err error) Error {
	line, column, message, ok := splitPositionMessage(err.Error())
	if !ok {
		return newError(path, 1, 1, err.Error())
	}
	return newError(path, line, column, message)
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
		file.TopLevelFuncs = append(file.TopLevelFuncs, pos)
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
		next := findNextTopLevel(toks, pos+1)
		if kind == "var" || kind == "const" {
			decl.Names, decl.NameToks = singleValueDeclNames(toks, namePos, next)
			names := decl.Names
			nameToks := decl.NameToks
			if len(names) > 0 {
				firstName := names[0]
				firstTok := nameToks[0]
				decl.Name = firstName
				decl.NameTok = firstTok
			}
		} else if namePos < len(toks) && toks[namePos].Kind == scan.Ident {
			decl.Name = toks[namePos].Text
			decl.Names = []string{decl.Name}
			decl.NameTok = toks[namePos]
			decl.NameToks = []scan.Token{decl.NameTok}
		}
		decl.End = declEnd(file, next)
		file.Decls = append(file.Decls, decl)
		return next
	}
	return pos + 1
}

func singleValueDeclNames(toks []scan.Token, start int, end int) ([]string, []scan.Token) {
	var names []string
	var nameToks []scan.Token
	expectName := true
	for i := start; i < end && i < len(toks); i++ {
		tok := toks[i]
		if tok.Kind == scan.Ident && expectName {
			names = append(names, tok.Text)
			nameToks = append(nameToks, tok)
			expectName = false
			continue
		}
		if tok.Text == "," && len(names) > 0 {
			expectName = true
			continue
		}
		break
	}
	return names, nameToks
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
	source := file.Source
	if start < 0 {
		start = 0
	}
	if end > len(source) {
		end = len(source)
	}
	for end > start && (source[end-1] == ' ' || source[end-1] == '\t' || source[end-1] == '\r' || source[end-1] == '\n') {
		end--
	}
	return string(source[start:end])
}

func declEnd(file *File, next int) int {
	tokens := file.Tokens
	if next >= 0 && next < len(tokens) {
		tok := tokens[next]
		return tok.Start
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
