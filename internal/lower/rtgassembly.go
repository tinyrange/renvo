package lower

type rtgAssemblyEntry struct {
	name   string
	offset int
}

type rtgAssemblyDocument struct {
	entries     []rtgAssemblyEntry
	errorOffset int
	ok          bool
}

// parseRTGAssemblyBindings only discovers top-level entry signatures. The
// CompilerJIT parser remains authoritative for the preserved source and body.
func parseRTGAssemblyBindings(source []byte) rtgAssemblyDocument {
	var out rtgAssemblyDocument
	depth := 0
	for at := 0; at < len(source); at++ {
		if at+1 < len(source) && source[at] == '/' && source[at+1] == '/' {
			for at < len(source) && source[at] != '\n' {
				at++
			}
			continue
		}
		if at+1 < len(source) && source[at] == '/' && source[at+1] == '*' {
			at += 2
			for at+1 < len(source) && (source[at] != '*' || source[at+1] != '/') {
				at++
			}
			at++
			continue
		}
		if source[at] == '"' || source[at] == '\'' || source[at] == '`' {
			quote := source[at]
			for at++; at < len(source) && source[at] != quote; at++ {
				if quote != '`' && source[at] == '\\' && at+1 < len(source) {
					at++
				}
			}
			continue
		}
		if source[at] == '{' {
			depth++
			continue
		}
		if source[at] == '}' {
			depth--
			continue
		}
		if depth != 1 || !rtgAssemblyIdent(source[at]) || (at > 0 && rtgAssemblyIdent(source[at-1])) {
			continue
		}
		end := at
		for end < len(source) && rtgAssemblyIdent(source[end]) {
			end++
		}
		signatureEnd, matched := rtgAssemblySignatureEnd(source, end)
		if !matched {
			continue
		}
		name := string(source[at:end])
		for i := 0; i < len(out.entries); i++ {
			if out.entries[i].name == name {
				return rtgAssemblyDocument{entries: out.entries, errorOffset: at}
			}
		}
		out.entries = append(out.entries, rtgAssemblyEntry{name: name, offset: at})
		at = signatureEnd - 1
	}
	out.errorOffset = len(source)
	out.ok = depth == 0 && len(out.entries) != 0
	return out
}

func rtgAssemblyIdent(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9' || value == '_'
}

// Discover the wrapper with the same trivia freedom as the authoritative RTG
// parser. This stays portable: body semantics belong to the selected backend.
func rtgAssemblySignatureEnd(source []byte, at int) (int, bool) {
	parts := []string{"(", "out", ":", "emitter", ")"}
	for i := 0; i < len(parts); i++ {
		at = rtgAssemblySkipTrivia(source, at)
		part := parts[i]
		if at+len(part) > len(source) || string(source[at:at+len(part)]) != part {
			return at, false
		}
		at += len(part)
		if (part == "out" || part == "emitter") && at < len(source) && rtgAssemblyIdent(source[at]) {
			return at, false
		}
	}
	body := rtgAssemblySkipTrivia(source, at)
	return at, body < len(source) && source[body] == '{'
}
func rtgAssemblySkipTrivia(source []byte, at int) int {
	for at < len(source) {
		c := source[at]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			at++
			continue
		}
		if at+1 < len(source) && c == '/' && source[at+1] == '/' {
			for at < len(source) && source[at] != '\n' {
				at++
			}
			continue
		}
		if at+1 < len(source) && c == '/' && source[at+1] == '*' {
			at += 2
			for at+1 < len(source) && (source[at] != '*' || source[at+1] != '/') {
				at++
			}
			if at+1 >= len(source) {
				return len(source)
			}
			at += 2
			continue
		}
		break
	}
	return at
}
