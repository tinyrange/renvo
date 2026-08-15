package c11

// IncludeReader resolves one source-level include without exposing the C
// frontend to a particular filesystem implementation.
type IncludeReader interface {
	ReadInclude(from string, name string, angled bool) ([]byte, string, bool)
}

type HeaderResult struct {
	Prelude   []byte
	Ok        bool
	ErrorPath string
	ErrorAt   int
}

type includeSpec struct {
	name   string
	angled bool
	at     int
}

// BuildObjectPrelude reads the translation unit's real headers and retains
// only externally linked function declarations referenced by the source. This
// is the deliberately narrow hosted-header slice: it bounds memory by avoiding
// a materialized copy of every unrelated declaration in a large libc header.
func BuildObjectPrelude(path string, src []byte, reader IncludeReader) HeaderResult {
	result := HeaderResult{Ok: true, ErrorAt: -1}
	includes := sourceIncludes(src)
	if len(includes) == 0 {
		return result
	}
	wanted := sourceCallNames(src)
	var emitted []string
	for i := 0; i < len(includes); i++ {
		header, headerPath, ok := reader.ReadInclude(path, includes[i].name, includes[i].angled)
		if !ok {
			return HeaderResult{Ok: false, ErrorPath: includes[i].name, ErrorAt: includes[i].at}
		}
		declarations, names, ok := headerDeclarations(header, wanted, emitted)
		if !ok {
			return HeaderResult{Ok: false, ErrorPath: headerPath, ErrorAt: includes[i].at}
		}
		result.Prelude = append(result.Prelude, declarations...)
		emitted = append(emitted, names...)
	}
	return result
}

func sourceIncludes(src []byte) []includeSpec {
	var out []includeSpec
	for lineStart := 0; lineStart < len(src); {
		lineEnd := lineStart
		for lineEnd < len(src) && src[lineEnd] != '\n' {
			lineEnd++
		}
		at := lineStart
		for at < lineEnd && (src[at] == ' ' || src[at] == '\t') {
			at++
		}
		directiveAt := at
		if at < lineEnd && src[at] == '#' {
			at++
			for at < lineEnd && (src[at] == ' ' || src[at] == '\t') {
				at++
			}
			wordStart := at
			for at < lineEnd && isIdentPart(src[at]) {
				at++
			}
			if textEquals(src[wordStart:at], "include") {
				for at < lineEnd && (src[at] == ' ' || src[at] == '\t') {
					at++
				}
				if at < lineEnd && (src[at] == '<' || src[at] == '"') {
					open := src[at]
					close := byte('>')
					if open == '"' {
						close = '"'
					}
					at++
					nameStart := at
					for at < lineEnd && src[at] != close {
						at++
					}
					if at < lineEnd && at > nameStart {
						out = append(out, includeSpec{name: string(src[nameStart:at]), angled: open == '<', at: directiveAt})
					}
				}
			}
		}
		lineStart = lineEnd + 1
	}
	return out
}

func sourceCallNames(src []byte) []string {
	scanned := scan(src)
	if !scanned.ok {
		return nil
	}
	var out []string
	for i := 0; i+1 < len(scanned.tokens); i++ {
		if tokenKind(scanned.tokens[i]) != tokenIdent || !tokenIs(src, scanned.tokens[i+1], "(") ||
			tokenIs(src, scanned.tokens[i], "if") || tokenIs(src, scanned.tokens[i], "for") ||
			tokenIs(src, scanned.tokens[i], "while") || tokenIs(src, scanned.tokens[i], "switch") ||
			tokenIs(src, scanned.tokens[i], "sizeof") || tokenIs(src, scanned.tokens[i], "_Alignof") {
			continue
		}
		name := string(tokenText(src, scanned.tokens[i]))
		if findText(out, name) < 0 {
			out = append(out, name)
		}
	}
	return out
}

func headerDeclarations(src []byte, wanted []string, emitted []string) ([]byte, []string, bool) {
	scanned := scan(src)
	if !scanned.ok {
		return nil, nil, false
	}
	var out []byte
	var names []string
	for i := 0; i+1 < len(scanned.tokens); i++ {
		if tokenKind(scanned.tokens[i]) != tokenIdent || !tokenIs(src, scanned.tokens[i+1], "(") {
			continue
		}
		name := string(tokenText(src, scanned.tokens[i]))
		if findText(wanted, name) < 0 || findText(emitted, name) >= 0 || findText(names, name) >= 0 ||
			tokenOnDirectiveLine(src, scanned.tokens, i) {
			continue
		}
		start := i
		for start > 0 && !tokenIs(src, scanned.tokens[start-1], ";") &&
			!tokenIs(src, scanned.tokens[start-1], "{") && !tokenIs(src, scanned.tokens[start-1], "}") {
			start--
		}
		for start < i && tokenOnDirectiveLine(src, scanned.tokens, start) {
			line := tokenLine(scanned.tokens[start])
			for start < i && tokenLine(scanned.tokens[start]) == line {
				start++
			}
		}
		for start < i && headerDeclarationDecoration(src, scanned.tokens[start]) {
			start++
		}
		hasStatic := false
		hasTypedef := false
		for j := start; j < i; j++ {
			if tokenIs(src, scanned.tokens[j], "static") {
				hasStatic = true
			}
			if tokenIs(src, scanned.tokens[j], "typedef") {
				hasTypedef = true
			}
		}
		// A file-scope function declaration has external linkage unless it is
		// explicitly static. libc commonly spells prototypes with extern while
		// other system libraries, including OpenSSL, rely on the C default.
		if hasStatic || hasTypedef {
			continue
		}
		depth := 0
		close := -1
		for j := i + 1; j < len(scanned.tokens); j++ {
			if tokenIs(src, scanned.tokens[j], "(") {
				depth++
			} else if tokenIs(src, scanned.tokens[j], ")") {
				depth--
				if depth == 0 {
					close = j
					break
				}
			}
		}
		if close < 0 || close+1 >= len(scanned.tokens) ||
			!tokenIs(src, scanned.tokens[close+1], ";") {
			continue
		}
		for j := start; j <= close; j++ {
			if j > start {
				out = append(out, ' ')
			}
			out = append(out, tokenText(src, scanned.tokens[j])...)
		}
		out = append(out, ';', '\n')
		names = append(names, name)
	}
	return out, names, true
}

func headerDeclarationDecoration(src []byte, tok token) bool {
	text := tokenText(src, tok)
	return textHasPrefix(text, "__") || textHasSuffix(text, "_EXPORT") ||
		textContains(text, "DEPRECATED")
}

func tokenOnDirectiveLine(src []byte, tokens []token, index int) bool {
	line := tokenLine(tokens[index])
	for i := index; i >= 0 && tokenLine(tokens[i]) == line; i-- {
		if tokenIs(src, tokens[i], "#") {
			return true
		}
	}
	return false
}

func findText(values []string, value string) int {
	for i := 0; i < len(values); i++ {
		if values[i] == value {
			return i
		}
	}
	return -1
}

func textEquals(value []byte, text string) bool {
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

func textHasPrefix(value []byte, prefix string) bool {
	if len(value) < len(prefix) {
		return false
	}
	return textEquals(value[:len(prefix)], prefix)
}

func textHasSuffix(value []byte, suffix string) bool {
	if len(value) < len(suffix) {
		return false
	}
	return textEquals(value[len(value)-len(suffix):], suffix)
}

func textContains(value []byte, part string) bool {
	if len(part) == 0 {
		return true
	}
	for i := 0; i+len(part) <= len(value); i++ {
		if textEquals(value[i:i+len(part)], part) {
			return true
		}
	}
	return false
}
