package load

import (
	"strconv"
	"strings"

	"j5.nz/rtg/rtg/scan"
)

type sourceError string

func (e sourceError) Error() string {
	return string(e)
}

func newSourceError(path string, line int, column int, message string) sourceError {
	return sourceError(path + ":" + strconv.Itoa(line) + ":" + strconv.Itoa(column) + ": " + message)
}

type SourceInfo struct {
	PackageName string
	Imports     []ImportInfo
}

type ImportInfo struct {
	Path   string
	Alias  string
	Line   int
	Column int
}

func ParseSourceInfo(path string, src []byte) (SourceInfo, error) {
	scanner := sourceInfoScanner{path: path, src: src, line: 1, column: 1}
	first, err := scanner.next()
	if err != nil {
		return SourceInfo{}, err
	}
	second, err := scanner.next()
	if err != nil {
		return SourceInfo{}, err
	}
	if !scanner.tokenTextIs(first, "package") || second.Kind != scan.Ident {
		return SourceInfo{}, newSourceError(path, 1, 1, "missing package declaration")
	}
	info := SourceInfo{PackageName: scanner.tokenText(second)}
	for {
		tok, err := scanner.next()
		if err != nil {
			return SourceInfo{}, err
		}
		if !scanner.tokenTextIs(tok, "import") {
			break
		}
		tok, err = scanner.next()
		if err != nil {
			return SourceInfo{}, err
		}
		if scanner.tokenTextIs(tok, "(") {
			for {
				tok, err = scanner.next()
				if err != nil {
					return SourceInfo{}, err
				}
				if scanner.tokenTextIs(tok, ")") {
					break
				}
				if tok.Kind == scan.EOF {
					return SourceInfo{}, newSourceError(path, int(tok.Line), int(tok.Column), "unterminated import block")
				}
				imp, err := sourceInfoImport(path, &scanner, tok)
				if err != nil {
					return SourceInfo{}, err
				}
				info.Imports = append(info.Imports, imp)
			}
			continue
		}
		imp, err := sourceInfoImport(path, &scanner, tok)
		if err != nil {
			return SourceInfo{}, err
		}
		info.Imports = append(info.Imports, imp)
	}
	return info, nil
}

type sourceInfoScanner struct {
	path   string
	src    []byte
	pos    int
	line   int
	column int
}

func sourceInfoToken(kind int, start int, end int, line int, column int) scan.Token {
	return scan.Token{Kind: kind, Start: int32(start), End: int32(end), Line: int32(line), Column: int32(column)}
}

func (s *sourceInfoScanner) next() (scan.Token, error) {
	for s.pos < len(s.src) {
		c := s.src[s.pos]
		if c == ' ' || c == '\t' || c == '\r' {
			s.pos++
			s.column++
			continue
		}
		if c == '\n' {
			s.pos++
			s.line++
			s.column = 1
			continue
		}
		if c == '/' && s.pos+1 < len(s.src) && s.src[s.pos+1] == '/' {
			s.pos += 2
			s.column += 2
			for s.pos < len(s.src) && s.src[s.pos] != '\n' {
				s.pos++
				s.column++
			}
			continue
		}
		if c == '/' && s.pos+1 < len(s.src) && s.src[s.pos+1] == '*' {
			startLine := s.line
			startColumn := s.column
			s.pos += 2
			s.column += 2
			closed := false
			for s.pos+1 < len(s.src) {
				if s.src[s.pos] == '*' && s.src[s.pos+1] == '/' {
					s.pos += 2
					s.column += 2
					closed = true
					break
				}
				s.advanceByte()
			}
			if !closed {
				return scan.Token{}, newSourceError(s.path, startLine, startColumn, "unterminated block comment")
			}
			continue
		}
		break
	}
	if s.pos >= len(s.src) {
		return sourceInfoToken(scan.EOF, s.pos, s.pos, s.line, s.column), nil
	}
	c := s.src[s.pos]
	if isSourceInfoIdentStart(c) {
		start := s.pos
		startLine := s.line
		startColumn := s.column
		s.pos++
		s.column++
		for s.pos < len(s.src) && isSourceInfoIdent(s.src[s.pos]) {
			s.pos++
			s.column++
		}
		return sourceInfoToken(scan.Ident, start, s.pos, startLine, startColumn), nil
	}
	if c == '"' || c == '`' {
		return s.stringToken()
	}
	start := s.pos
	startLine := s.line
	startColumn := s.column
	s.pos++
	s.column++
	return sourceInfoToken(scan.Op, start, s.pos, startLine, startColumn), nil
}

func (s *sourceInfoScanner) stringToken() (scan.Token, error) {
	start := s.pos
	startLine := s.line
	startColumn := s.column
	quote := s.src[s.pos]
	s.pos++
	s.column++
	for s.pos < len(s.src) {
		if quote != '`' && s.src[s.pos] == '\\' {
			s.pos += 2
			s.column += 2
			continue
		}
		if s.src[s.pos] == quote {
			s.pos++
			s.column++
			return sourceInfoToken(scan.String, start, s.pos, startLine, startColumn), nil
		}
		s.advanceByte()
	}
	return scan.Token{}, newSourceError(s.path, startLine, startColumn, "unterminated literal")
}

func (s *sourceInfoScanner) tokenText(tok scan.Token) string {
	return string(s.src[int(tok.Start):int(tok.End)])
}

func (s *sourceInfoScanner) tokenTextIs(tok scan.Token, text string) bool {
	start := int(tok.Start)
	end := int(tok.End)
	if end-start != len(text) {
		return false
	}
	for i := 0; i < len(text); i++ {
		if s.src[start+i] != text[i] {
			return false
		}
	}
	return true
}

func (s *sourceInfoScanner) advanceByte() {
	if s.src[s.pos] == '\n' {
		s.pos++
		s.line++
		s.column = 1
		return
	}
	s.pos++
	s.column++
}

func sourceInfoImport(path string, scanner *sourceInfoScanner, tok scan.Token) (ImportInfo, error) {
	alias := ""
	if tok.Kind == scan.Ident || scanner.tokenTextIs(tok, ".") || scanner.tokenTextIs(tok, "_") {
		next, err := scanner.next()
		if err != nil {
			return ImportInfo{}, err
		}
		if next.Kind != scan.String || tok.Line != next.Line {
			return ImportInfo{}, newSourceError(path, int(tok.Line), int(tok.Column), "malformed import declaration")
		}
		alias = scanner.tokenText(tok)
		tok = next
	}
	if tok.Kind != scan.String {
		return ImportInfo{}, newSourceError(path, int(tok.Line), int(tok.Column), "malformed import declaration")
	}
	value, err := scan.UnquoteString(scanner.tokenText(tok))
	if err != nil {
		return ImportInfo{}, newSourceError(path, int(tok.Line), int(tok.Column), err.Error())
	}
	return ImportInfo{Path: value, Alias: alias, Line: int(tok.Line), Column: int(tok.Column)}, nil
}

func isSourceInfoIdentStart(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || c == '_'
}

func isSourceInfoIdent(c byte) bool {
	return isSourceInfoIdentStart(c) || (c >= '0' && c <= '9')
}

func appendPackageImports(path string, src []byte, pkg *Package, importSet *[]string) error {
	toks, err := scan.Tokens(src)
	if err != nil {
		return scannerError(path, err)
	}
	pos := 2
	for pos < len(toks) {
		tok := toks[pos]
		if tok.Text != "import" {
			break
		}
		pos++
		tok = toks[pos]
		if tok.Text == "(" {
			pos++
			for pos < len(toks) {
				tok = toks[pos]
				if tok.Text == ")" || tok.Kind == scan.EOF {
					break
				}
				alias := ""
				if tok.Kind == scan.Ident || tok.Text == "." || tok.Text == "_" {
					if pos+1 < len(toks) {
						nextTok := toks[pos+1]
						if nextTok.Kind == scan.String && tok.Line == nextTok.Line {
							alias = tok.Text
							pos++
							tok = toks[pos]
						}
					}
				}
				if tok.Kind != scan.String {
					return newSourceError(path, int(tok.Line), int(tok.Column), "malformed import declaration")
				}
				value, err := scan.UnquoteString(tok.Text)
				if err != nil {
					return newSourceError(path, int(tok.Line), int(tok.Column), err.Error())
				}
				appendPackageImport(path, pkg, importSet, value, alias, int(tok.Line), int(tok.Column))
				pos++
			}
			if pos >= len(toks) || toks[pos].Text != ")" {
				at := toks[pos-1]
				if pos < len(toks) {
					at = toks[pos]
				}
				return newSourceError(path, int(at.Line), int(at.Column), "unterminated import block")
			}
			pos++
			continue
		}
		alias := ""
		if tok.Kind == scan.Ident || tok.Text == "." || tok.Text == "_" {
			if pos+1 >= len(toks) {
				return newSourceError(path, int(tok.Line), int(tok.Column), "malformed import declaration")
			}
			nextTok := toks[pos+1]
			if nextTok.Kind != scan.String || tok.Line != nextTok.Line {
				return newSourceError(path, int(tok.Line), int(tok.Column), "malformed import declaration")
			}
			alias = tok.Text
			pos++
			tok = toks[pos]
		}
		if tok.Kind != scan.String {
			return newSourceError(path, int(tok.Line), int(tok.Column), "malformed import declaration")
		}
		value, err := scan.UnquoteString(tok.Text)
		if err != nil {
			return newSourceError(path, int(tok.Line), int(tok.Column), err.Error())
		}
		appendPackageImport(path, pkg, importSet, value, alias, int(tok.Line), int(tok.Column))
		pos++
	}
	return nil
}

func appendPackageImport(path string, pkg *Package, importSet *[]string, value string, alias string, line int, column int) {
	impPath := copyLoadString(value)
	values := *importSet
	if !containsString(values, impPath) {
		values = append(values, impPath)
		*importSet = values
		pkg.Imports = append(pkg.Imports, impPath)
	}
	if !hasImportPosition(pkg.ImportPositions, impPath) {
		pkg.ImportPositions = append(pkg.ImportPositions, ImportPosition{ImportPath: impPath, Path: path, Line: line, Column: column})
	}
}

func scannerError(path string, err error) sourceError {
	line, column, message, ok := splitPositionMessage(err.Error())
	if !ok {
		return newSourceError(path, 1, 1, err.Error())
	}
	return newSourceError(path, line, column, message)
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
