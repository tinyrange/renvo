package driver

import "renvo.dev/internal/load"

func collectSourceImports(module load.Module, stdRoot string, src []byte) ([]load.PackageRef, bool) {
	pos := renvoImportSkipSpace(src, 0)
	wordStart, wordEnd, next, ok := renvoImportIdent(src, pos)
	if !ok || !renvoImportTextIs(src, wordStart, wordEnd, "package") {
		return nil, false
	}
	pos = renvoImportSkipSpace(src, next)
	_, _, pos, ok = renvoImportIdent(src, pos)
	if !ok {
		return nil, false
	}
	var out []load.PackageRef
	for {
		pos = renvoImportSkipSeparators(src, pos)
		wordStart, wordEnd, next, ok = renvoImportIdent(src, pos)
		if !ok {
			return out, true
		}
		if !renvoImportTextIs(src, wordStart, wordEnd, "import") {
			return out, true
		}
		pos = renvoImportSkipSpace(src, next)
		if pos < len(src) && src[pos] == '(' {
			pos++
			for {
				pos = renvoImportSkipSeparators(src, pos)
				if pos >= len(src) {
					return nil, false
				}
				if src[pos] == ')' {
					pos++
					break
				}
				path, specEnd, ok := renvoImportSpec(src, pos)
				if !ok {
					return nil, false
				}
				out = append(out, load.ResolveImport(module, stdRoot, path))
				pos = specEnd
			}
			continue
		}
		path, specEnd, ok := renvoImportSpec(src, pos)
		if !ok {
			return nil, false
		}
		out = append(out, load.ResolveImport(module, stdRoot, path))
		pos = specEnd
	}
}

func renvoImportSpec(src []byte, pos int) (string, int, bool) {
	pos = renvoImportSkipSpace(src, pos)
	if pos >= len(src) {
		return "", pos, false
	}
	if src[pos] != '"' && src[pos] != '`' {
		if src[pos] == '.' {
			pos++
		} else {
			_, _, next, ok := renvoImportIdent(src, pos)
			if !ok {
				return "", pos, false
			}
			pos = next
		}
		pos = renvoImportSkipSpace(src, pos)
	}
	path, next, ok := renvoImportString(src, pos)
	if !ok {
		return "", pos, false
	}
	return path, renvoImportSkipLine(src, next), true
}

func renvoImportSkipSpace(src []byte, pos int) int {
	for pos < len(src) {
		c := src[pos]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			pos++
			continue
		}
		if c == '/' && pos+1 < len(src) && src[pos+1] == '/' {
			pos += 2
			for pos < len(src) && src[pos] != '\n' {
				pos++
			}
			continue
		}
		if c == '/' && pos+1 < len(src) && src[pos+1] == '*' {
			pos += 2
			for pos+1 < len(src) && !(src[pos] == '*' && src[pos+1] == '/') {
				pos++
			}
			if pos+1 < len(src) {
				pos += 2
			}
			continue
		}
		break
	}
	return pos
}

func renvoImportSkipSeparators(src []byte, pos int) int {
	for {
		pos = renvoImportSkipSpace(src, pos)
		if pos < len(src) && src[pos] == ';' {
			pos++
			continue
		}
		return pos
	}
}

func renvoImportSkipLine(src []byte, pos int) int {
	for pos < len(src) {
		if src[pos] == ';' {
			return pos + 1
		}
		if src[pos] == '\n' {
			return pos + 1
		}
		if src[pos] == ')' {
			return pos
		}
		pos++
	}
	return pos
}

func renvoImportIdent(src []byte, pos int) (int, int, int, bool) {
	if pos >= len(src) || !renvoImportIdentStart(src[pos]) {
		return pos, pos, pos, false
	}
	start := pos
	pos++
	for pos < len(src) && renvoImportIdentPart(src[pos]) {
		pos++
	}
	return start, pos, pos, true
}

func renvoImportString(src []byte, pos int) (string, int, bool) {
	if pos >= len(src) {
		return "", pos, false
	}
	quote := src[pos]
	if quote == '`' {
		start := pos + 1
		pos++
		for pos < len(src) && src[pos] != '`' {
			pos++
		}
		if pos >= len(src) {
			return "", pos, false
		}
		return string(src[start:pos]), pos + 1, true
	}
	if quote != '"' {
		return "", pos, false
	}
	pos++
	out := make([]byte, 0, 32)
	for pos < len(src) {
		c := src[pos]
		if c == '"' {
			return string(out), pos + 1, true
		}
		if c == '\n' || c == '\r' {
			return "", pos, false
		}
		if c != '\\' {
			out = append(out, c)
			pos++
			continue
		}
		pos++
		if pos >= len(src) {
			return "", pos, false
		}
		c = src[pos]
		if c == '"' || c == '\\' {
			out = append(out, c)
		} else {
			return "", pos, false
		}
		pos++
	}
	return "", pos, false
}

func renvoImportTextIs(src []byte, start int, end int, text string) bool {
	if end-start != len(text) {
		return false
	}
	for i := 0; i < len(text); i++ {
		if src[start+i] != text[i] {
			return false
		}
	}
	return true
}

func renvoImportIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func renvoImportIdentPart(c byte) bool {
	return renvoImportIdentStart(c) || (c >= '0' && c <= '9')
}
