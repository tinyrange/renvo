package syntax

// CgoPreamble returns the comment immediately preceding a standalone
// import "C" declaration. The returned bytes are C source with comment
// delimiters removed, matching the header role assigned by cmd/cgo.
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
	start := TokenStart(file.Tokens[importTok])
	i := start
	newlines := 0
	for i > 0 && cgoHorizontalOrNewline(file.Src[i-1]) {
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
	if !cgoLineCommentAt(file.Src, lineStart, lineEnd) {
		return nil, -1, true
	}
	first := lineStart
	for first > 0 {
		previousEnd := first - 1
		if previousEnd > 0 && file.Src[previousEnd-1] == '\r' {
			previousEnd--
		}
		previousStart := cgoLineStart(file.Src, previousEnd)
		if !cgoLineCommentAt(file.Src, previousStart, previousEnd) {
			break
		}
		first = previousStart
	}
	out := make([]byte, 0, lineEnd-first)
	pos := first
	contentStart := -1
	for pos < lineEnd {
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

func cgoHorizontalOrNewline(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n'
}

func cgoLineStart(src []byte, end int) int {
	for end > 0 && src[end-1] != '\n' {
		end--
	}
	return end
}

func cgoLineCommentAt(src []byte, start int, end int) bool {
	for start < end && (src[start] == ' ' || src[start] == '\t') {
		start++
	}
	return start+2 <= end && src[start] == '/' && src[start+1] == '/'
}
