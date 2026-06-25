package load

import (
	"fmt"

	"j5.nz/rtg/rtg/scan"
)

type SourceInfo struct {
	PackageName string
	Imports     []ImportInfo
}

type ImportInfo struct {
	Path  string
	Alias string
}

func ParseSourceInfo(path string, src []byte) (SourceInfo, error) {
	toks, err := scan.Tokens(src)
	if err != nil {
		return SourceInfo{}, fmt.Errorf("%s: %w", path, err)
	}
	if len(toks) < 2 || toks[0].Text != "package" || toks[1].Kind != scan.Ident {
		return SourceInfo{}, fmt.Errorf("%s: missing package declaration", path)
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
			for pos < len(toks) && toks[pos].Text != ")" {
				if toks[pos].Kind == scan.String {
					value, err := scan.UnquoteString(toks[pos].Text)
					if err != nil {
						return SourceInfo{}, fmt.Errorf("%s: %w", path, err)
					}
					info.Imports = append(info.Imports, ImportInfo{Path: value, Alias: aliasBefore(toks, pos)})
				}
				pos++
			}
			if pos >= len(toks) {
				return SourceInfo{}, fmt.Errorf("%s: unterminated import block", path)
			}
			pos++
			continue
		}
		found := false
		for pos < len(toks) {
			if toks[pos].Kind == scan.String {
				value, err := scan.UnquoteString(toks[pos].Text)
				if err != nil {
					return SourceInfo{}, fmt.Errorf("%s: %w", path, err)
				}
				alias := importAliasBefore(toks, pos)
				info.Imports = append(info.Imports, ImportInfo{Path: value, Alias: alias})
				pos++
				found = true
				break
			}
			if toks[pos].Text == "import" || toks[pos].Text == "func" || toks[pos].Text == "var" || toks[pos].Text == "const" || toks[pos].Text == "type" {
				break
			}
			pos++
		}
		if !found {
			return SourceInfo{}, fmt.Errorf("%s: malformed import declaration", path)
		}
	}
	return info, nil
}

func aliasBefore(toks []scan.Token, pos int) string {
	if pos > 0 && (toks[pos-1].Kind == scan.Ident || toks[pos-1].Text == "." || toks[pos-1].Text == "_") {
		if toks[pos-1].Text == "import" {
			return ""
		}
		return toks[pos-1].Text
	}
	return ""
}

func importAliasBefore(toks []scan.Token, pos int) string {
	return aliasBefore(toks, pos)
}
