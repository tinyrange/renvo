package syntax

// ExportDirective returns the external identifier from a //export directive
// immediately preceding a package-level function. It intentionally implements
// the cgo spelling and adjacency rule without making comments part of the
// ordinary token stream.
func ExportDirective(file File, fn FuncDecl) string {
	if fn.ReceiverStart >= 0 || fn.StartTok < 0 || fn.StartTok >= len(file.Tokens) {
		return ""
	}
	start := TokenStart(file.Tokens[fn.StartTok])
	if start <= 0 || start > len(file.Src) {
		return ""
	}
	i := start
	for i > 0 && (file.Src[i-1] == ' ' || file.Src[i-1] == '\t' || file.Src[i-1] == '\r') {
		i--
	}
	if i == 0 || file.Src[i-1] != '\n' {
		return ""
	}
	end := i - 1
	if end > 0 && file.Src[end-1] == '\r' {
		end--
	}
	line := end
	for line > 0 && file.Src[line-1] != '\n' {
		line--
	}
	for line < end && (file.Src[line] == ' ' || file.Src[line] == '\t') {
		line++
	}
	const prefix = "//export "
	if end-line <= len(prefix) || !bytesEqualString(file.Src[line:line+len(prefix)], prefix) {
		return ""
	}
	name := file.Src[line+len(prefix) : end]
	if !exportIdentifier(name) {
		return ""
	}
	return string(name)
}

func exportIdentifier(name []byte) bool {
	if len(name) == 0 || !exportIdentifierStart(name[0]) {
		return false
	}
	for i := 1; i < len(name); i++ {
		if !exportIdentifierStart(name[i]) && (name[i] < '0' || name[i] > '9') {
			return false
		}
	}
	return true
}

func exportIdentifierStart(ch byte) bool {
	return ch == '_' || ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z'
}

func bytesEqualString(value []byte, text string) bool {
	if len(value) != len(text) {
		return false
	}
	for i := 0; i < len(value); i++ {
		if value[i] != text[i] {
			return false
		}
	}
	return true
}

// CgoPreamble returns the comment immediately preceding a standalone import
// "C", with its comment delimiters removed.
func CgoPreamble(file File, decl ImportDecl) ([]byte, int, bool) {
	if decl.PathTok < 0 || decl.PathTok >= len(file.Tokens) {
		return nil, -1, false
	}
	path, ok := StringLiteralValue(file.Src, file.Tokens[decl.PathTok])
	if !ok || path != "C" {
		return nil, -1, false
	}
	importTok := decl.StartTok - 1
	if importTok < 0 || importTok >= len(file.Tokens) || file.Tokens[importTok].KindLine&255 != TokenImport {
		return nil, -1, false
	}
	i := TokenStart(file.Tokens[importTok])
	newlines := 0
	for i > 0 && cgoWhitespace(file.Src[i-1]) {
		if file.Src[i-1] == '\n' {
			newlines++
		}
		i--
	}
	if newlines > 1 {
		return nil, -1, true
	}
	if i >= 2 && file.Src[i-2] == '*' && file.Src[i-1] == '/' {
		end := i - 2
		for begin := end - 2; begin >= 0; begin-- {
			if file.Src[begin] == '/' && file.Src[begin+1] == '*' {
				return file.Src[begin+2 : end], begin + 2, true
			}
		}
		return nil, -1, false
	}
	lineEnd := i
	lineStart := cgoLineStart(file.Src, lineEnd)
	if !cgoLineComment(file.Src, lineStart, lineEnd) {
		return nil, -1, true
	}
	first := lineStart
	for first > 0 {
		previousEnd := first - 1
		if previousEnd > 0 && file.Src[previousEnd-1] == '\r' {
			previousEnd--
		}
		previousStart := cgoLineStart(file.Src, previousEnd)
		if !cgoLineComment(file.Src, previousStart, previousEnd) {
			break
		}
		first = previousStart
	}
	out := make([]byte, 0, lineEnd-first)
	contentStart := -1
	for pos := first; pos < lineEnd; {
		end := pos
		for end < lineEnd && file.Src[end] != '\n' {
			end++
		}
		trimmed := pos
		for trimmed < end && (file.Src[trimmed] == ' ' || file.Src[trimmed] == '\t') {
			trimmed++
		}
		content := trimmed + 2
		if content < end && file.Src[content] == ' ' {
			content++
		}
		if contentStart < 0 {
			contentStart = content
		}
		out = append(out, file.Src[content:end]...)
		out = append(out, '\n')
		pos = end + 1
	}
	return out, contentStart, true
}

func cgoWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n'
}

func cgoLineStart(src []byte, end int) int {
	for end > 0 && src[end-1] != '\n' {
		end--
	}
	return end
}

func cgoLineComment(src []byte, start int, end int) bool {
	for start < end && (src[start] == ' ' || src[start] == '\t') {
		start++
	}
	return start+2 <= end && src[start] == '/' && src[start+1] == '/'
}
