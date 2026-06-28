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
	toks, err := scan.Tokens(src)
	if err != nil {
		return SourceInfo{}, scannerError(path, err)
	}
	if len(toks) < 2 || toks[0].Text != "package" || toks[1].Kind != scan.Ident {
		return SourceInfo{}, newSourceError(path, 1, 1, "missing package declaration")
	}
	info := SourceInfo{PackageName: toks[1].Text}
	pos := 2
	for pos < len(toks) {
		if toks[pos].Text != "import" {
			break
		}
		pos++
		if pos < len(toks) && toks[pos].Text == "(" {
			pos++
			for pos < len(toks) && toks[pos].Text != ")" && toks[pos].Kind != scan.EOF {
				alias := ""
				if toks[pos].Kind == scan.Ident || toks[pos].Text == "." || toks[pos].Text == "_" {
					if pos+1 < len(toks) && toks[pos+1].Kind == scan.String && toks[pos].Line == toks[pos+1].Line {
						alias = toks[pos].Text
						pos++
					}
				}
				if toks[pos].Kind != scan.String {
					return SourceInfo{}, newSourceError(path, toks[pos].Line, toks[pos].Column, "malformed import declaration")
				}
				value, err := scan.UnquoteString(toks[pos].Text)
				if err != nil {
					return SourceInfo{}, newSourceError(path, toks[pos].Line, toks[pos].Column, err.Error())
				}
				info.Imports = append(info.Imports, ImportInfo{Path: value, Alias: alias, Line: toks[pos].Line, Column: toks[pos].Column})
				pos++
			}
			if pos >= len(toks) || toks[pos].Text != ")" {
				at := toks[pos-1]
				if pos < len(toks) {
					at = toks[pos]
				}
				return SourceInfo{}, newSourceError(path, at.Line, at.Column, "unterminated import block")
			}
			pos++
			continue
		}
		alias := ""
		if pos < len(toks) && (toks[pos].Kind == scan.Ident || toks[pos].Text == "." || toks[pos].Text == "_") {
			if pos+1 >= len(toks) || toks[pos+1].Kind != scan.String || toks[pos].Line != toks[pos+1].Line {
				return SourceInfo{}, newSourceError(path, toks[pos].Line, toks[pos].Column, "malformed import declaration")
			}
			alias = toks[pos].Text
			pos++
		}
		if pos >= len(toks) || toks[pos].Kind != scan.String {
			tok := toks[pos]
			return SourceInfo{}, newSourceError(path, tok.Line, tok.Column, "malformed import declaration")
		}
		value, err := scan.UnquoteString(toks[pos].Text)
		if err != nil {
			return SourceInfo{}, newSourceError(path, toks[pos].Line, toks[pos].Column, err.Error())
		}
		info.Imports = append(info.Imports, ImportInfo{Path: value, Alias: alias, Line: toks[pos].Line, Column: toks[pos].Column})
		pos++
	}
	return info, nil
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
					return newSourceError(path, tok.Line, tok.Column, "malformed import declaration")
				}
				value, err := scan.UnquoteString(tok.Text)
				if err != nil {
					return newSourceError(path, tok.Line, tok.Column, err.Error())
				}
				appendPackageImport(path, pkg, importSet, value, alias, tok.Line, tok.Column)
				pos++
			}
			if pos >= len(toks) || toks[pos].Text != ")" {
				at := toks[pos-1]
				if pos < len(toks) {
					at = toks[pos]
				}
				return newSourceError(path, at.Line, at.Column, "unterminated import block")
			}
			pos++
			continue
		}
		alias := ""
		if tok.Kind == scan.Ident || tok.Text == "." || tok.Text == "_" {
			if pos+1 >= len(toks) {
				return newSourceError(path, tok.Line, tok.Column, "malformed import declaration")
			}
			nextTok := toks[pos+1]
			if nextTok.Kind != scan.String || tok.Line != nextTok.Line {
				return newSourceError(path, tok.Line, tok.Column, "malformed import declaration")
			}
			alias = tok.Text
			pos++
			tok = toks[pos]
		}
		if tok.Kind != scan.String {
			return newSourceError(path, tok.Line, tok.Column, "malformed import declaration")
		}
		value, err := scan.UnquoteString(tok.Text)
		if err != nil {
			return newSourceError(path, tok.Line, tok.Column, err.Error())
		}
		appendPackageImport(path, pkg, importSet, value, alias, tok.Line, tok.Column)
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
