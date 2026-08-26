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

// parseRTGAssemblyBindings reads only the source wrapper needed to bind entry
// names to Go declarations. CompilerJIT owns interpretation of each preserved
// body against the selected backend definition.
func parseRTGAssemblyBindings(source []byte) rtgAssemblyDocument {
	p := rtgAssemblyBindingParser{source: source}
	if !p.takeWord("rtgasm") || !p.takeWord("1") || !p.takeWord("assembly") || !p.takeByte('{') {
		return p.result()
	}
	for {
		p.skipSpace()
		if p.takeByte('}') {
			break
		}
		start := p.at
		name := p.word()
		if name == "" || p.duplicate(name) || !p.takeByte('(') || !p.takeWord("out") ||
			!p.takeByte(':') || !p.takeWord("emitter") || !p.takeByte(')') || !p.takeByte('{') {
			return p.result()
		}
		p.entries = append(p.entries, rtgAssemblyEntry{name: name, offset: start})
		if !p.skipBody() {
			return p.result()
		}
	}
	p.skipSpace()
	p.ok = p.at == len(source) && len(p.entries) != 0
	return p.result()
}

type rtgAssemblyBindingParser struct {
	source  []byte
	at      int
	entries []rtgAssemblyEntry
	ok      bool
}

func (p *rtgAssemblyBindingParser) result() rtgAssemblyDocument {
	return rtgAssemblyDocument{entries: p.entries, errorOffset: p.at, ok: p.ok}
}

func (p *rtgAssemblyBindingParser) duplicate(name string) bool {
	for i := 0; i < len(p.entries); i++ {
		if p.entries[i].name == name {
			return true
		}
	}
	return false
}

func (p *rtgAssemblyBindingParser) takeWord(want string) bool {
	return p.word() == want
}

func (p *rtgAssemblyBindingParser) word() string {
	p.skipSpace()
	start := p.at
	for p.at < len(p.source) && (p.source[p.at] >= 'a' && p.source[p.at] <= 'z' ||
		p.source[p.at] >= 'A' && p.source[p.at] <= 'Z' || p.source[p.at] >= '0' && p.source[p.at] <= '9' ||
		p.source[p.at] == '_') {
		p.at++
	}
	return string(p.source[start:p.at])
}

func (p *rtgAssemblyBindingParser) takeByte(want byte) bool {
	p.skipSpace()
	if p.at >= len(p.source) || p.source[p.at] != want {
		return false
	}
	p.at++
	return true
}

func (p *rtgAssemblyBindingParser) skipSpace() {
	for p.at < len(p.source) {
		if p.source[p.at] == ' ' || p.source[p.at] == '\t' || p.source[p.at] == '\r' || p.source[p.at] == '\n' {
			p.at++
			continue
		}
		if p.at+1 < len(p.source) && p.source[p.at] == '/' && p.source[p.at+1] == '/' {
			p.at += 2
			for p.at < len(p.source) && p.source[p.at] != '\n' {
				p.at++
			}
			continue
		}
		if p.at+1 < len(p.source) && p.source[p.at] == '/' && p.source[p.at+1] == '*' {
			p.at += 2
			for p.at+1 < len(p.source) && (p.source[p.at] != '*' || p.source[p.at+1] != '/') {
				p.at++
			}
			if p.at+1 < len(p.source) {
				p.at += 2
			}
			continue
		}
		break
	}
}

func (p *rtgAssemblyBindingParser) skipBody() bool {
	depth := 1
	for p.at < len(p.source) {
		if p.at+1 < len(p.source) && p.source[p.at] == '/' && (p.source[p.at+1] == '/' || p.source[p.at+1] == '*') {
			p.skipSpace()
			continue
		}
		if p.source[p.at] == '"' || p.source[p.at] == '\'' || p.source[p.at] == '`' {
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
			continue
		}
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
	return false
}
