package c11

func (p *preprocessor) directive(tokens []ppToken, includeFrom string, displayPath string, line int, emit bool, depth int) (int, string) {
	if len(tokens) == 0 {
		return 0, ""
	}
	if ppTokenKind(tokens[0]) == ppNumber {
		newLine, newPath := p.lineDirective(tokens, displayPath)
		if newLine == 0 {
			p.fail(PreprocessErrDirective, displayPath, line)
		}
		return newLine, newPath
	}
	if ppTokenKind(tokens[0]) != ppIdent {
		p.fail(PreprocessErrDirective, displayPath, line)
		return 0, ""
	}
	name := tokens[0]
	directiveName := string(p.tokenText(name))
	rest := tokens[1:]
	conditional := directiveName == "if" || directiveName == "ifdef" || directiveName == "ifndef"
	alternative := directiveName == "elif" || directiveName == "elifdef" || directiveName == "elifndef"
	if conditional || alternative {
		if alternative && (len(p.conditionals) == 0 || p.conditionals[len(p.conditionals)-1].seenElse) {
			p.fail(PreprocessErrDirective, displayPath, line)
			return 0, ""
		}
		condition := false
		valid := true
		evaluate := conditional && p.active()
		if alternative {
			level := p.conditionals[len(p.conditionals)-1]
			evaluate = level.parentActive && !level.matched
		}
		if evaluate {
			if directiveName == "if" || directiveName == "elif" {
				condition, valid = p.evaluate(rest)
			} else {
				valid = len(rest) == 1 && ppTokenKind(rest[0]) == ppIdent
				if valid {
					condition = p.lookupToken(rest[0]) >= 0
					if directiveName == "ifndef" || directiveName == "elifndef" {
						condition = !condition
					}
				}
			}
		}
		if !valid {
			p.fail(PreprocessErrExpression, displayPath, line)
			return 0, ""
		}
		if conditional {
			parent := p.active()
			p.conditionals = append(p.conditionals, ppConditional{parentActive: parent, active: parent && condition, matched: condition})
		} else {
			level := &p.conditionals[len(p.conditionals)-1]
			level.active = level.parentActive && !level.matched && condition
			level.matched = level.matched || condition
		}
		return 0, ""
	}
	if directiveName == "else" {
		if len(rest) != 0 || len(p.conditionals) == 0 || p.conditionals[len(p.conditionals)-1].seenElse {
			p.fail(PreprocessErrDirective, displayPath, line)
			return 0, ""
		}
		level := &p.conditionals[len(p.conditionals)-1]
		level.seenElse = true
		level.active = level.parentActive && !level.matched
		level.matched = true
		return 0, ""
	}
	if directiveName == "endif" {
		if len(rest) != 0 || len(p.conditionals) == 0 {
			p.fail(PreprocessErrDirective, displayPath, line)
			return 0, ""
		}
		p.conditionals = p.conditionals[:len(p.conditionals)-1]
		return 0, ""
	}
	if !p.active() {
		return 0, ""
	}
	if directiveName == "define" {
		if !p.defineTokens(rest) {
			if len(rest) > 0 {
				p.errorDetail = string(p.tokenText(rest[0]))
			}
			p.fail(PreprocessErrMacro, displayPath, line)
		}
		return 0, ""
	}
	if directiveName == "undef" {
		if len(rest) != 1 || ppTokenKind(rest[0]) != ppIdent {
			p.fail(PreprocessErrMacro, displayPath, line)
		} else {
			if index := p.lookupToken(rest[0]); index >= 0 {
				p.macros[index].deleted = true
			}
		}
		return 0, ""
	}
	if directiveName == "include" || directiveName == "include_next" {
		p.include(rest, includeFrom, displayPath, line, emit, depth, directiveName == "include_next")
		return 0, ""
	}
	if directiveName == "pragma" {
		if len(rest) == 1 && p.tokenIs(rest[0], "once") && findPPText(p.once, includeFrom) < 0 {
			p.once = append(p.once, includeFrom)
		}
		return 0, ""
	}
	if directiveName == "line" {
		newLine, newPath := p.lineDirective(rest, displayPath)
		if newLine == 0 {
			p.fail(PreprocessErrDirective, displayPath, line)
		}
		return newLine, newPath
	}
	if directiveName == "error" {
		p.fail(PreprocessErrDirective, displayPath, line)
		return 0, ""
	}
	if directiveName == "warning" {
		return 0, ""
	}
	// GCC accepts these dependency-control directives in generated headers.
	if directiveName == "ident" || directiveName == "sccs" {
		return 0, ""
	}
	p.fail(PreprocessErrDirective, displayPath, line)
	return 0, ""
}

func (p *preprocessor) lineDirective(tokens []ppToken, oldPath string) (int, string) {
	expanded := p.expand(tokens, 0)
	if !p.ok || len(expanded) == 0 || ppTokenKind(expanded[0]) != ppNumber {
		return 0, ""
	}
	line, ok := ppParsePositive(p.tokenText(expanded[0]))
	if !ok {
		return 0, ""
	}
	path := oldPath
	if len(expanded) > 1 && ppTokenKind(expanded[1]) == ppString {
		text := p.tokenText(expanded[1])
		if len(text) >= 2 {
			path = string(text[1 : len(text)-1])
		}
	}
	return line, path
}

func (p *preprocessor) include(tokens []ppToken, from string, displayPath string, line int, emit bool, depth int, next bool) {
	if p.config.Reader == nil {
		p.fail(PreprocessErrInclude, displayPath, line)
		return
	}
	expanded := tokens
	literalHeader := len(tokens) == 1 && ppTokenKind(tokens[0]) == ppString ||
		len(tokens) >= 3 && p.tokenIs(tokens[0], "<") && p.tokenIs(tokens[len(tokens)-1], ">")
	if !literalHeader {
		expanded = p.expand(tokens, 0)
	}
	if !p.ok {
		return
	}
	name := ""
	angled := false
	if len(expanded) == 1 && ppTokenKind(expanded[0]) == ppString {
		text := p.tokenText(expanded[0])
		if len(text) >= 2 {
			name = string(text[1 : len(text)-1])
		}
	} else if len(expanded) >= 3 && p.tokenIs(expanded[0], "<") && p.tokenIs(expanded[len(expanded)-1], ">") {
		angled = true
		var bytes []byte
		for i := 1; i+1 < len(expanded); i++ {
			bytes = append(bytes, p.tokenText(expanded[i])...)
		}
		name = string(bytes)
	}
	if name == "" {
		p.fail(PreprocessErrInclude, displayPath, line)
		return
	}
	var src []byte
	path := ""
	ok := false
	if next {
		src, path, ok = p.config.Reader.ReadIncludeNext(from, name, angled)
	} else {
		src, path, ok = p.config.Reader.ReadInclude(from, name, angled)
	}
	if !ok {
		p.fail(PreprocessErrInclude, name, line)
		return
	}
	p.addDependency(path)
	includeEmit := emit && p.config.EmitIncludes
	p.processFile(path, src, includeEmit, depth+1)
	if includeEmit && p.config.LineMarkers && p.ok {
		p.appendLineMarker(line+1, displayPath)
	}
}

func (p *preprocessor) defineTokens(tokens []ppToken) bool {
	if len(tokens) == 0 || ppTokenKind(tokens[0]) != ppIdent {
		return false
	}
	name := string(p.tokenText(tokens[0]))
	function := false
	variadic := false
	var params []string
	at := 1
	if at < len(tokens) && p.tokenIs(tokens[at], "(") && !ppTokenSpace(tokens[at]) {
		function = true
		at++
		if at < len(tokens) && p.tokenIs(tokens[at], ")") {
			at++
		} else {
			for {
				if at >= len(tokens) {
					return false
				}
				if p.tokenIs(tokens[at], "...") {
					variadic = true
					params = append(params, "__VA_ARGS__")
					at++
					if at >= len(tokens) || !p.tokenIs(tokens[at], ")") {
						return false
					}
					at++
					break
				}
				if ppTokenKind(tokens[at]) != ppIdent {
					return false
				}
				param := string(p.tokenText(tokens[at]))
				at++
				if at < len(tokens) && p.tokenIs(tokens[at], "...") {
					variadic = true
					params = append(params, param)
					at++
					if at >= len(tokens) || !p.tokenIs(tokens[at], ")") {
						return false
					}
					at++
					break
				}
				if findPPText(params, param) >= 0 {
					return false
				}
				params = append(params, param)
				if at < len(tokens) && p.tokenIs(tokens[at], ",") {
					at++
					continue
				}
				if at < len(tokens) && p.tokenIs(tokens[at], ")") {
					at++
					break
				}
				return false
			}
		}
	}
	replacement := ppCloneTokens(tokens[at:])
	p.define(ppMacro{name: name, params: params, replace: replacement, function: function, variadic: variadic})
	return true
}

func (p *preprocessor) defineText(name string, params []string, function bool, variadic bool, value string) {
	source := len(p.sources)
	p.sources = append(p.sources, []byte(value))
	tokens, ok := p.lex(source, 0, len(value))
	if !ok {
		p.fail(PreprocessErrMacro, "<predefined>", 1)
		return
	}
	p.define(ppMacro{name: name, params: params, replace: tokens, function: function, variadic: variadic})
}

func (p *preprocessor) define(macro ppMacro) {
	index := p.lookupAnyName([]byte(macro.name))
	if index >= 0 {
		p.macros[index] = macro
		return
	}
	p.macros = append(p.macros, macro)
	if len(p.buckets) == 0 || len(p.macros)*2 > len(p.buckets) {
		p.rebuildBuckets()
	} else {
		p.insertBucket(len(p.macros) - 1)
	}
}

func (p *preprocessor) rebuildBuckets() {
	size := 64
	for size < len(p.macros)*2 {
		size *= 2
	}
	p.buckets = make([]int, size)
	for i := 0; i < len(p.macros); i++ {
		p.insertBucket(i)
	}
}

func (p *preprocessor) insertBucket(index int) {
	hash := ppHash([]byte(p.macros[index].name))
	at := int(hash & uint64(len(p.buckets)-1))
	for p.buckets[at] != 0 {
		at = (at + 1) & (len(p.buckets) - 1)
	}
	p.buckets[at] = index + 1
}

func (p *preprocessor) lookupToken(tok ppToken) int { return p.lookupName(p.tokenText(tok)) }

func (p *preprocessor) lookupName(name []byte) int {
	index := p.lookupAnyName(name)
	if index < 0 || p.macros[index].deleted {
		return -1
	}
	return index
}

func (p *preprocessor) lookupAnyName(name []byte) int {
	if len(p.buckets) == 0 {
		return -1
	}
	at := int(ppHash(name) & uint64(len(p.buckets)-1))
	for probes := 0; probes < len(p.buckets); probes++ {
		entry := p.buckets[at]
		if entry == 0 {
			return -1
		}
		index := entry - 1
		if ppBytesStringEqual(name, p.macros[index].name) {
			return index
		}
		at = (at + 1) & (len(p.buckets) - 1)
	}
	return -1
}

func ppHash(text []byte) uint64 {
	hash := uint64(1469598103934665603)
	for i := 0; i < len(text); i++ {
		hash ^= uint64(text[i])
		hash *= 1099511628211
	}
	return hash
}

func ppBytesStringEqual(value []byte, text string) bool {
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

func ppParsePositive(text []byte) (int, bool) {
	value := 0
	if len(text) == 0 {
		return 0, false
	}
	for i := 0; i < len(text); i++ {
		if text[i] < '0' || text[i] > '9' {
			return 0, false
		}
		value = value*10 + int(text[i]-'0')
	}
	return value, value > 0
}
