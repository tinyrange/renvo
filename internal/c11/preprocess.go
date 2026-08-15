package c11

// Macro is one object-like preprocessor definition. Function-like macros are
// deliberately left to the full token preprocessor; the compiler-identification
// handshake only needs object replacement and conditional selection.
type Macro struct {
	Name  string
	Value string
}

// PreprocessResult is the small, allocation-bounded preprocessing result used
// by compiler-driver probes. Unsupported or malformed directives fail instead
// of being copied through, which keeps feature probes truthful.
type PreprocessResult struct {
	Source []byte
	Ok     bool
	Line   int
}

type conditionalLevel struct {
	parentActive bool
	active       bool
	matched      bool
	seenElse     bool
}

// PreprocessProbe handles the conditional and object-macro subset needed by
// compiler identification probes. It is intentionally a real general subset,
// not a spelling check for Linux's probe source.
func PreprocessProbe(src []byte, predefined []Macro) PreprocessResult {
	macros := make([]Macro, len(predefined))
	copy(macros, predefined)
	out := make([]byte, 0, len(src))
	var stack []conditionalLevel
	active := true
	line := 1
	for start := 0; start <= len(src); line++ {
		end := start
		for end < len(src) && src[end] != '\n' {
			end++
		}
		text := src[start:end]
		at := skipHorizontal(text, 0)
		if at < len(text) && text[at] == '#' {
			nameStart := skipHorizontal(text, at+1)
			nameEnd := nameStart
			for nameEnd < len(text) && identByte(text[nameEnd]) {
				nameEnd++
			}
			name := string(text[nameStart:nameEnd])
			rest := text[skipHorizontal(text, nameEnd):]
			switch name {
			case "if", "ifdef", "ifndef":
				condition := false
				ok := true
				if name == "ifdef" || name == "ifndef" {
					identifier, valid := singleIdentifier(rest)
					ok = valid
					condition = macroIndex(macros, identifier) >= 0
					if name == "ifndef" {
						condition = !condition
					}
				} else {
					condition, ok = evaluateProbeExpression(rest, macros)
				}
				if !ok {
					return PreprocessResult{Ok: false, Line: line}
				}
				level := conditionalLevel{parentActive: active, active: active && condition, matched: condition}
				stack = append(stack, level)
				active = level.active
			case "elif":
				if len(stack) == 0 || stack[len(stack)-1].seenElse {
					return PreprocessResult{Ok: false, Line: line}
				}
				condition, ok := evaluateProbeExpression(rest, macros)
				if !ok {
					return PreprocessResult{Ok: false, Line: line}
				}
				level := &stack[len(stack)-1]
				level.active = level.parentActive && !level.matched && condition
				level.matched = level.matched || condition
				active = level.active
			case "else":
				if len(stack) == 0 || stack[len(stack)-1].seenElse {
					return PreprocessResult{Ok: false, Line: line}
				}
				level := &stack[len(stack)-1]
				level.seenElse = true
				level.active = level.parentActive && !level.matched
				level.matched = true
				active = level.active
			case "endif":
				if len(stack) == 0 {
					return PreprocessResult{Ok: false, Line: line}
				}
				stack = stack[:len(stack)-1]
				active = true
				if len(stack) > 0 {
					active = stack[len(stack)-1].active
				}
			case "define":
				if active {
					identifier, value, ok := macroDefinition(rest)
					if !ok {
						return PreprocessResult{Ok: false, Line: line}
					}
					macros = defineMacro(macros, identifier, value)
				}
			case "undef":
				if active {
					identifier, ok := singleIdentifier(rest)
					if !ok {
						return PreprocessResult{Ok: false, Line: line}
					}
					macros = undefineMacro(macros, identifier)
				}
			case "":
			default:
				if active {
					return PreprocessResult{Ok: false, Line: line}
				}
			}
		} else if active {
			out = appendExpandedIdentifiers(out, text, macros)
			if end < len(src) {
				out = append(out, '\n')
			}
		}
		if end == len(src) {
			break
		}
		start = end + 1
	}
	if len(stack) != 0 {
		return PreprocessResult{Ok: false, Line: line - 1}
	}
	return PreprocessResult{Source: out, Ok: true}
}

func appendExpandedIdentifiers(out []byte, line []byte, macros []Macro) []byte {
	for at := 0; at < len(line); {
		if !identStartByte(line[at]) {
			out = append(out, line[at])
			at++
			continue
		}
		end := at + 1
		for end < len(line) && identByte(line[end]) {
			end++
		}
		index := macroIndex(macros, string(line[at:end]))
		if index >= 0 {
			out = append(out, macros[index].Value...)
		} else {
			out = append(out, line[at:end]...)
		}
		at = end
	}
	return out
}

func macroDefinition(text []byte) (string, string, bool) {
	at := skipHorizontal(text, 0)
	if at >= len(text) || !identStartByte(text[at]) {
		return "", "", false
	}
	end := at + 1
	for end < len(text) && identByte(text[end]) {
		end++
	}
	if end < len(text) && text[end] == '(' {
		return "", "", false
	}
	valueAt := skipHorizontal(text, end)
	value := "1"
	if valueAt < len(text) {
		value = string(text[valueAt:])
	}
	return string(text[at:end]), value, true
}

func defineMacro(macros []Macro, name string, value string) []Macro {
	if index := macroIndex(macros, name); index >= 0 {
		macros[index].Value = value
		return macros
	}
	return append(macros, Macro{Name: name, Value: value})
}

func undefineMacro(macros []Macro, name string) []Macro {
	index := macroIndex(macros, name)
	if index < 0 {
		return macros
	}
	copy(macros[index:], macros[index+1:])
	return macros[:len(macros)-1]
}

func macroIndex(macros []Macro, name string) int {
	for i := len(macros) - 1; i >= 0; i-- {
		if macros[i].Name == name {
			return i
		}
	}
	return -1
}

func singleIdentifier(text []byte) (string, bool) {
	at := skipHorizontal(text, 0)
	if at >= len(text) || !identStartByte(text[at]) {
		return "", false
	}
	end := at + 1
	for end < len(text) && identByte(text[end]) {
		end++
	}
	return string(text[at:end]), skipHorizontal(text, end) == len(text)
}

type probeExpressionParser struct {
	text   []byte
	at     int
	macros []Macro
	ok     bool
}

func evaluateProbeExpression(text []byte, macros []Macro) (bool, bool) {
	p := probeExpressionParser{text: text, macros: macros, ok: true}
	value := p.parseOr()
	p.at = skipHorizontal(p.text, p.at)
	return value, p.ok && p.at == len(p.text)
}

func (p *probeExpressionParser) parseOr() bool {
	value := p.parseAnd()
	for p.take("||") {
		value = p.parseAnd() || value
	}
	return value
}

func (p *probeExpressionParser) parseAnd() bool {
	value := p.parseUnary()
	for p.take("&&") {
		value = p.parseUnary() && value
	}
	return value
}

func (p *probeExpressionParser) parseUnary() bool {
	p.at = skipHorizontal(p.text, p.at)
	if p.take("!") {
		return !p.parseUnary()
	}
	if p.take("(") {
		value := p.parseOr()
		if !p.take(")") {
			p.ok = false
		}
		return value
	}
	start := p.at
	for p.at < len(p.text) && identByte(p.text[p.at]) {
		p.at++
	}
	word := string(p.text[start:p.at])
	if word == "defined" {
		parenthesized := p.take("(")
		name, ok := p.identifier()
		if !ok || parenthesized && !p.take(")") {
			p.ok = false
			return false
		}
		return macroIndex(p.macros, name) >= 0
	}
	if word != "" {
		index := macroIndex(p.macros, word)
		if index < 0 {
			return false
		}
		return decimalNonzero(p.macros[index].Value)
	}
	if p.at < len(p.text) && p.text[p.at] >= '0' && p.text[p.at] <= '9' {
		start = p.at
		for p.at < len(p.text) && p.text[p.at] >= '0' && p.text[p.at] <= '9' {
			p.at++
		}
		return decimalNonzero(string(p.text[start:p.at]))
	}
	p.ok = false
	return false
}

func (p *probeExpressionParser) identifier() (string, bool) {
	p.at = skipHorizontal(p.text, p.at)
	if p.at >= len(p.text) || !identStartByte(p.text[p.at]) {
		return "", false
	}
	start := p.at
	p.at++
	for p.at < len(p.text) && identByte(p.text[p.at]) {
		p.at++
	}
	return string(p.text[start:p.at]), true
}

func (p *probeExpressionParser) take(value string) bool {
	p.at = skipHorizontal(p.text, p.at)
	if p.at+len(value) > len(p.text) {
		return false
	}
	for i := 0; i < len(value); i++ {
		if p.text[p.at+i] != value[i] {
			return false
		}
	}
	p.at += len(value)
	return true
}

func decimalNonzero(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] >= '1' && value[i] <= '9' {
			return true
		}
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return false
}

func skipHorizontal(text []byte, at int) int {
	for at < len(text) && (text[at] == ' ' || text[at] == '\t' || text[at] == '\r') {
		at++
	}
	return at
}

func identStartByte(ch byte) bool {
	return ch == '_' || ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z'
}

func identByte(ch byte) bool {
	return identStartByte(ch) || ch >= '0' && ch <= '9'
}
