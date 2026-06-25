package load

import (
	"strconv"
	"strings"

	"j5.nz/rtg/rtg/scan"
)

type sourceError struct {
	path    string
	line    int
	column  int
	message string
}

func (e sourceError) Error() string {
	return e.path + ":" + strconv.Itoa(e.line) + ":" + strconv.Itoa(e.column) + ": " + e.message
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
		return SourceInfo{}, sourceError{path: path, line: 1, column: 1, message: "missing package declaration"}
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
					return SourceInfo{}, sourceError{path: path, line: toks[pos].Line, column: toks[pos].Column, message: "malformed import declaration"}
				}
				value, err := scan.UnquoteString(toks[pos].Text)
				if err != nil {
					return SourceInfo{}, sourceError{path: path, line: toks[pos].Line, column: toks[pos].Column, message: err.Error()}
				}
				info.Imports = append(info.Imports, ImportInfo{Path: value, Alias: alias, Line: toks[pos].Line, Column: toks[pos].Column})
				pos++
			}
			if pos >= len(toks) || toks[pos].Text != ")" {
				at := toks[pos-1]
				if pos < len(toks) {
					at = toks[pos]
				}
				return SourceInfo{}, sourceError{path: path, line: at.Line, column: at.Column, message: "unterminated import block"}
			}
			pos++
			continue
		}
		alias := ""
		if pos < len(toks) && (toks[pos].Kind == scan.Ident || toks[pos].Text == "." || toks[pos].Text == "_") {
			if pos+1 >= len(toks) || toks[pos+1].Kind != scan.String || toks[pos].Line != toks[pos+1].Line {
				return SourceInfo{}, sourceError{path: path, line: toks[pos].Line, column: toks[pos].Column, message: "malformed import declaration"}
			}
			alias = toks[pos].Text
			pos++
		}
		if pos >= len(toks) || toks[pos].Kind != scan.String {
			tok := toks[pos]
			return SourceInfo{}, sourceError{path: path, line: tok.Line, column: tok.Column, message: "malformed import declaration"}
		}
		value, err := scan.UnquoteString(toks[pos].Text)
		if err != nil {
			return SourceInfo{}, sourceError{path: path, line: toks[pos].Line, column: toks[pos].Column, message: err.Error()}
		}
		info.Imports = append(info.Imports, ImportInfo{Path: value, Alias: alias, Line: toks[pos].Line, Column: toks[pos].Column})
		pos++
	}
	return info, nil
}

func scannerError(path string, err error) sourceError {
	line, column, message, ok := splitPositionMessage(err.Error())
	if !ok {
		return sourceError{path: path, line: 1, column: 1, message: err.Error()}
	}
	return sourceError{path: path, line: line, column: column, message: message}
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
