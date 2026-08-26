//go:build !renvo

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

// parseRTGAssemblyBindings reads the wrapper and preserves entry bodies for
// CompilerJIT, which interprets them against the selected backend definition.
func parseRTGAssemblyBindings(source []byte) rtgAssemblyDocument {
	p := rtgAssemblyBindingParser{source: source}
	var entries []rtgAssemblyEntry
	if p.word() != "rtgasm" || p.word() != "1" || p.word() != "assembly" || !p.take('{') {
		return rtgAssemblyDocument{errorOffset: p.at}
	}
	for {
		p.space()
		if p.take('}') {
			break
		}
		start := p.at
		name := p.word()
		duplicate := false
		for i := 0; i < len(entries); i++ {
			duplicate = duplicate || entries[i].name == name
		}
		if name == "" || duplicate || !p.take('(') || p.word() != "out" ||
			!p.take(':') || p.word() != "emitter" || !p.take(')') || !p.take('{') {
			return rtgAssemblyDocument{entries: entries, errorOffset: p.at}
		}
		entries = append(entries, rtgAssemblyEntry{name: name, offset: start})
		if !p.body() {
			return rtgAssemblyDocument{entries: entries, errorOffset: p.at}
		}
	}
	p.space()
	return rtgAssemblyDocument{entries: entries, errorOffset: p.at, ok: p.at == len(source) && len(entries) != 0}
}

type rtgAssemblyBindingParser struct {
	source []byte
	at     int
}

func (p *rtgAssemblyBindingParser) word() string {
	p.space()
	start := p.at
	for p.at < len(p.source) && (p.source[p.at] >= 'a' && p.source[p.at] <= 'z' ||
		p.source[p.at] >= 'A' && p.source[p.at] <= 'Z' || p.source[p.at] >= '0' && p.source[p.at] <= '9' || p.source[p.at] == '_') {
		p.at++
	}
	return string(p.source[start:p.at])
}

func (p *rtgAssemblyBindingParser) take(want byte) bool {
	p.space()
	if p.at >= len(p.source) || p.source[p.at] != want {
		return false
	}
	p.at++
	return true
}

func (p *rtgAssemblyBindingParser) space() {
	for p.at < len(p.source) {
		if p.source[p.at] == ' ' || p.source[p.at] == '\t' || p.source[p.at] == '\r' || p.source[p.at] == '\n' {
			p.at++
		} else if p.at+1 < len(p.source) && p.source[p.at] == '/' && p.source[p.at+1] == '/' {
			p.at += 2
			for p.at < len(p.source) && p.source[p.at] != '\n' {
				p.at++
			}
		} else if p.at+1 < len(p.source) && p.source[p.at] == '/' && p.source[p.at+1] == '*' {
			p.at += 2
			for p.at+1 < len(p.source) && (p.source[p.at] != '*' || p.source[p.at+1] != '/') {
				p.at++
			}
			if p.at+1 < len(p.source) {
				p.at += 2
			}
		} else {
			return
		}
	}
}

func (p *rtgAssemblyBindingParser) body() bool {
	depth := 1
	for p.at < len(p.source) {
		if p.at+1 < len(p.source) && p.source[p.at] == '/' && (p.source[p.at+1] == '/' || p.source[p.at+1] == '*') {
			p.space()
		} else if p.source[p.at] == '"' || p.source[p.at] == '\'' || p.source[p.at] == '`' {
			quote := p.source[p.at]
			p.at++
			for p.at < len(p.source) && p.source[p.at] != quote {
				if quote != '`' && p.source[p.at] == '\\' && p.at+1 < len(p.source) {
					p.at++
				}
				p.at++
			}
			if p.at < len(p.source) {
				p.at++
			}
		} else {
			if p.source[p.at] == '{' {
				depth++
			} else if p.source[p.at] == '}' {
				depth--
				if depth == 0 {
					p.at++
					return true
				}
			}
			p.at++
		}
	}
	return false
}
