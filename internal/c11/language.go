//go:build !renvo || renvo_wasi_language_service

package c11

// The language-service index deliberately describes the original C source.
// Translation into the shared Go-shaped frontend is an implementation detail
// and can rename declarations or move them by inserting generated helpers.

const (
	LanguageVariable = iota + 1
	LanguageField
	LanguageMethod
	LanguageFunction
	LanguageType
	LanguagePackage
	LanguageKeyword
)

const (
	languageSymbolObject = iota + 1
	languageSymbolFunction
	languageSymbolType
	languageSymbolField
	languageSymbolMacro
)

// LanguageFile is one original C source or header included in an analysis.
type LanguageFile struct {
	Path                string
	Source              []byte
	SuppressDiagnostics bool
}

// LanguageLocation is a byte range in an original source file.
type LanguageLocation struct {
	Path  string
	Start int
	End   int
}

// LanguageInclude describes a header name under a source cursor.
type LanguageInclude struct {
	Name   string
	Start  int
	End    int
	Angled bool
	Ok     bool
}

// LanguageParameter describes one C function parameter.
type LanguageParameter struct {
	Name string
	Type string
}

// LanguageCompletion is one visible C name or keyword.
type LanguageCompletion struct {
	Name          string
	Detail        string
	Kind          int
	Signature     string
	Documentation string
	Parameters    []LanguageParameter
}

// LanguageHover describes the declaration and documentation under a cursor.
type LanguageHover struct {
	Signature     string
	Documentation string
	Start         int
	End           int
	Ok            bool
}

// LanguageSignature describes the active C call and its parameters.
type LanguageSignature struct {
	Label           string
	Parameters      []LanguageParameter
	ActiveParameter int
	Ok              bool
}

// LanguageNavigation contains a definition and every bound declaration/use.
type LanguageNavigation struct {
	Name       string
	Definition LanguageLocation
	References []LanguageLocation
	// DefinitionIsDeclaration is true when C only supplied an extern
	// declaration. Mixed C/Go hosts can then prefer a matching Go body.
	DefinitionIsDeclaration bool
	Ok                      bool
}

// LanguageDiagnostic is a source-facing C scan, parse, or semantic error.
type LanguageDiagnostic struct {
	Path    string
	Start   int
	End     int
	Line    int
	Column  int
	Code    string
	Message string
}

type languageSymbol struct {
	name          string
	typeName      string
	signature     string
	documentation string
	path          string
	owner         string
	start         int
	end           int
	scopeStart    int
	scopeEnd      int
	binding       int
	kind          int
	pointer       int
	defined       bool
	external      bool
	internal      bool
	parameters    []LanguageParameter
}

type languageReference struct {
	path    string
	start   int
	end     int
	binding int
}

type languageToken struct {
	path  string
	index int
	tok   token
}

type LanguageAnalysis struct {
	files       []LanguageFile
	symbols     []languageSymbol
	references  []languageReference
	diagnostics []LanguageDiagnostic
	nextBinding int
}

type languageParser struct {
	analysis *LanguageAnalysis
	file     LanguageFile
	tokens   []token
	skip     []bool
	declared []bool
	types    []string
	depth    []int
	braces   []int
}

// AnalyzeLanguage builds the reusable, source-facing semantic snapshot used
// by completion, hover, signature help, navigation, references, and rename.
// Files may contain source and project headers; declarations with external
// linkage are unified across translation units after each file is indexed.
func AnalyzeLanguage(files []LanguageFile) LanguageAnalysis {
	analysis := LanguageAnalysis{files: files, nextBinding: 1}
	for i := 0; i < len(files); i++ {
		analysis.parseFile(files[i])
	}
	analysis.unifyGlobals()
	analysis.buildReferences()
	return analysis
}

// Diagnostics returns every problem found while building the snapshot.
func (a *LanguageAnalysis) Diagnostics() []LanguageDiagnostic {
	return a.diagnostics
}

// IncludeAt returns the #include header containing offset in path. Resolution
// remains the host's responsibility because include search paths are supplied
// by the compiler driver rather than the C language index.
func (a *LanguageAnalysis) IncludeAt(path string, offset int) LanguageInclude {
	src := a.source(path)
	if offset < 0 || offset > len(src) {
		return LanguageInclude{}
	}
	lineStart := offset
	for lineStart > 0 && src[lineStart-1] != '\n' {
		lineStart--
	}
	lineEnd := offset
	for lineEnd < len(src) && src[lineEnd] != '\n' {
		lineEnd++
	}
	at := languageSkipSpace(src, lineStart, lineEnd)
	if at >= lineEnd || src[at] != '#' {
		return LanguageInclude{}
	}
	at = languageSkipSpace(src, at+1, lineEnd)
	wordStart := at
	at = languageSkipIdent(src, at, lineEnd)
	if string(src[wordStart:at]) != "include" {
		return LanguageInclude{}
	}
	at = languageSkipSpace(src, at, lineEnd)
	if at >= lineEnd || src[at] != '<' && src[at] != '"' {
		return LanguageInclude{}
	}
	angled := src[at] == '<'
	close := byte('"')
	if angled {
		close = '>'
	}
	start := at + 1
	end := start
	for end < lineEnd && src[end] != close {
		end++
	}
	if end >= lineEnd || offset < start || offset > end {
		return LanguageInclude{}
	}
	return LanguageInclude{Name: string(src[start:end]), Start: start, End: end, Angled: angled, Ok: true}
}

func (a *LanguageAnalysis) parseFile(file LanguageFile) {
	scanned := scan(file.Source)
	if !scanned.ok {
		line, column := languageLineColumn(file.Source, scanned.errorAt)
		message := "invalid byte in C source"
		if scanned.err == ScanErrComment {
			message = "unterminated block comment"
		} else if scanned.err == ScanErrLiteral {
			message = "unterminated string or character literal"
		}
		a.diagnostics = append(a.diagnostics, LanguageDiagnostic{Path: file.Path, Start: scanned.errorAt,
			End: scanned.errorAt + 1, Line: line, Column: column, Code: "RENVO-C-SCAN-001", Message: message})
	}
	p := languageParser{analysis: a, file: file, tokens: scanned.tokens}
	p.skip = make([]bool, len(p.tokens))
	p.declared = make([]bool, len(p.tokens))
	p.types = make([]string, len(p.tokens))
	p.depth = make([]int, len(p.tokens))
	p.braces = make([]int, len(p.tokens))
	p.markDirectives()
	p.computeDepths()
	p.parseMacros()
	p.parseDeclarations()
}

func (p *languageParser) text(index int) string {
	if index < 0 || index >= len(p.tokens) {
		return ""
	}
	return string(tokenText(p.file.Source, p.tokens[index]))
}

func (p *languageParser) markDirectives() {
	src := p.file.Source
	for lineStart := 0; lineStart < len(src); {
		lineEnd := lineStart
		for lineEnd < len(src) && src[lineEnd] != '\n' {
			lineEnd++
		}
		at := lineStart
		for at < lineEnd && (src[at] == ' ' || src[at] == '\t' || src[at] == '\r') {
			at++
		}
		if at < lineEnd && src[at] == '#' {
			for i := 0; i < len(p.tokens); i++ {
				if p.tokens[i].start >= lineStart && p.tokens[i].start < lineEnd {
					p.skip[i] = true
				}
			}
		}
		lineStart = lineEnd + 1
	}
}

func (p *languageParser) computeDepths() {
	depth := 0
	var stack []int
	for i := 0; i < len(p.tokens); i++ {
		p.depth[i] = depth
		if p.skip[i] {
			continue
		}
		if p.text(i) == "{" {
			stack = append(stack, i)
			depth++
		} else if p.text(i) == "}" {
			if depth > 0 {
				depth--
			}
			p.depth[i] = depth
			if len(stack) > 0 {
				open := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				p.braces[open] = i + 1
				p.braces[i] = open + 1
			}
		}
	}
	for i := 0; i < len(stack); i++ {
		at := p.tokens[stack[i]].start
		line, column := languageLineColumn(p.file.Source, at)
		p.analysis.diagnostics = append(p.analysis.diagnostics, LanguageDiagnostic{Path: p.file.Path,
			Start: at, End: at + 1, Line: line, Column: column, Code: "RENVO-C-PARSE-001", Message: "unclosed '{'"})
	}
}

func (p *languageParser) parseMacros() {
	src := p.file.Source
	for lineStart := 0; lineStart < len(src); {
		lineEnd := lineStart
		for lineEnd < len(src) && src[lineEnd] != '\n' {
			lineEnd++
		}
		at := lineStart
		for at < lineEnd && (src[at] == ' ' || src[at] == '\t' || src[at] == '\r') {
			at++
		}
		if at < lineEnd && src[at] == '#' {
			at++
			at = languageSkipSpace(src, at, lineEnd)
			wordStart := at
			at = languageSkipIdent(src, at, lineEnd)
			if string(src[wordStart:at]) == "define" {
				at = languageSkipSpace(src, at, lineEnd)
				nameStart := at
				at = languageSkipIdent(src, at, lineEnd)
				if at > nameStart {
					name := string(src[nameStart:at])
					var parameters []LanguageParameter
					parameterText := ""
					if at < lineEnd && src[at] == '(' {
						parameterStart := at
						at++
						for at < lineEnd && src[at] != ')' {
							at = languageSkipSpace(src, at, lineEnd)
							start := at
							at = languageSkipIdent(src, at, lineEnd)
							if at > start {
								parameters = append(parameters, LanguageParameter{Name: string(src[start:at])})
							}
							at = languageSkipSpace(src, at, lineEnd)
							if at < lineEnd && src[at] == ',' {
								at++
							}
						}
						if at < lineEnd && src[at] == ')' {
							at++
							parameterText = string(src[parameterStart:at])
						}
					}
					replacement := string(languageTrimSpace(src[at:lineEnd]))
					signature := "#define " + name
					if parameterText != "" {
						signature += parameterText
					}
					if replacement != "" {
						signature += " " + replacement
					}
					p.analysis.addSymbol(languageSymbol{name: name, typeName: "macro", signature: signature,
						documentation: languageDocumentation(src, lineStart), path: p.file.Path, start: nameStart,
						end: nameStart + len(name), scopeEnd: len(src), kind: languageSymbolMacro, defined: true, parameters: parameters})
				}
			}
		}
		lineStart = lineEnd + 1
	}
}

func (p *languageParser) parseDeclarations() {
	for i := 0; i < len(p.tokens)-1; {
		if p.skip[i] || !p.declarationStart(i) {
			i++
			continue
		}
		next := p.parseDeclaration(i, "", false)
		if next <= i {
			i++
		} else {
			i = next
		}
	}
}

func (p *languageParser) declarationStart(index int) bool {
	text := p.text(index)
	if !p.typeStart(index) && !languageStorage(text) && !languageQualifier(text) {
		return false
	}
	previous := p.previous(index)
	if previous < 0 {
		return true
	}
	before := p.text(previous)
	if before == ";" || before == "{" || before == "}" || before == ":" {
		return true
	}
	// A declaration can be the initializer clause in a for statement.
	return before == "(" && previous > 0 && p.text(p.previous(previous)) == "for"
}

func (p *languageParser) typeStart(index int) bool {
	text := p.text(index)
	if languageBuiltinType(text) || text == "struct" || text == "union" || text == "enum" || text == "typeof" || text == "__typeof__" || text == "__auto_type" {
		return true
	}
	for i := len(p.analysis.symbols) - 1; i >= 0; i-- {
		if p.analysis.symbols[i].kind == languageSymbolType && p.analysis.symbols[i].name == text {
			return true
		}
	}
	return false
}

func (p *languageParser) parseDeclaration(start int, owner string, field bool) int {
	i := start
	typedef := false
	external := false
	internal := false
	for i < len(p.tokens) && !p.skip[i] {
		text := p.text(i)
		if text == "typedef" {
			typedef = true
			i++
		} else if text == "extern" {
			external = true
			i++
		} else if text == "static" {
			internal = true
			i++
		} else if languageStorage(text) || languageQualifier(text) || languageFunctionSpecifier(text) {
			i++
		} else {
			break
		}
	}
	base, next, aggregateOwner := p.parseType(i)
	if next <= i {
		return start
	}
	i = next
	if aggregateOwner != "" && owner == "" {
		owner = aggregateOwner
	}
	if i < len(p.tokens) && p.text(i) == ";" {
		return i + 1
	}
	for i < len(p.tokens)-1 {
		segmentEnd, terminator := p.declaratorEnd(i)
		nameIndex := p.declaratorName(i, segmentEnd)
		if nameIndex < 0 {
			return start
		}
		p.declared[nameIndex] = true
		name := p.text(nameIndex)
		pointer := p.declaratorPointers(i, nameIndex)
		typeName := base
		for j := 0; j < pointer; j++ {
			typeName += " *"
		}
		arrayEnd := p.arrayDeclaratorEnd(nameIndex+1, segmentEnd)
		if arrayEnd > nameIndex+1 {
			typeName += p.compactText(nameIndex+1, arrayEnd)
		}
		if typeName == "__auto_type" {
			if inferred := p.inferInitializerType(nameIndex, segmentEnd); inferred != "" {
				typeName = inferred
			}
		}
		functionOpen := p.functionOpen(nameIndex, segmentEnd)
		kind := languageSymbolObject
		defined := !external
		var parameters []LanguageParameter
		signature := typeName + " " + name
		bodyOpen := -1
		if functionOpen >= 0 && !p.functionPointerName(i, nameIndex) {
			kind = languageSymbolFunction
			parameters = p.parseParameters(functionOpen)
			signature = typeName + " " + name + "(" + languageParameterText(parameters) + ")"
			bodyOpen = p.afterFunctionBody(functionOpen, segmentEnd)
			defined = bodyOpen >= 0
		}
		if typedef {
			kind = languageSymbolType
			typeName = base
			signature = "typedef " + typeName + " " + name
		}
		if field {
			kind = languageSymbolField
			defined = true
		}
		scopeStart, scopeEnd := p.scopeAt(nameIndex)
		if p.depth[nameIndex] == 0 || field {
			scopeStart, scopeEnd = 0, len(p.file.Source)
		}
		symbol := languageSymbol{name: name, typeName: typeName, signature: signature,
			documentation: languageDocumentation(p.file.Source, p.tokens[start].start), path: p.file.Path,
			start: p.tokens[nameIndex].start, end: p.tokens[nameIndex].end, scopeStart: scopeStart,
			scopeEnd: scopeEnd, kind: kind, pointer: pointer, owner: owner, defined: defined, external: external,
			internal: internal, parameters: parameters}
		p.analysis.addSymbol(symbol)
		if kind == languageSymbolFunction {
			parameterClose := functionOpen
			if p.tokens[functionOpen].match > 0 {
				parameterClose = functionOpen + p.tokens[functionOpen].match
			}
			searchEnd, parameterScopeStart, parameterScopeEnd := parameterClose, p.tokens[functionOpen].start, p.tokens[parameterClose].end
			if bodyOpen >= 0 {
				searchEnd, parameterScopeStart, parameterScopeEnd = bodyOpen, p.tokens[bodyOpen].start, len(p.file.Source)
				if p.braces[bodyOpen] > 0 {
					parameterScopeEnd = p.tokens[p.braces[bodyOpen]-1].end
				}
			}
			p.addParameterSymbols(parameters, functionOpen, searchEnd, parameterScopeStart, parameterScopeEnd)
		}
		if terminator == "," {
			i = segmentEnd + 1
			continue
		}
		if terminator == "{" && bodyOpen >= 0 {
			// Keep walking the body so declarations in an incomplete or complete
			// function remain available to interactive queries.
			return bodyOpen + 1
		}
		return segmentEnd + 1
	}
	return start
}

func (p *languageParser) parseType(index int) (string, int, string) {
	if index >= len(p.tokens) {
		return "", index, ""
	}
	text := p.text(index)
	if text == "struct" || text == "union" || text == "enum" {
		kind := text
		i := index + 1
		name := ""
		if i < len(p.tokens) && tokenKind(p.tokens[i]) == tokenIdent && !p.skip[i] {
			name = p.text(i)
			i++
		}
		base := kind
		if name != "" {
			base += " " + name
		}
		if i < len(p.tokens) && p.text(i) == "{" {
			if name != "" {
				p.addTagSymbol(index+1, base, true)
			}
			close := p.braces[i] - 1
			if close < i {
				close = len(p.tokens) - 1
			}
			owner := base
			if name == "" {
				owner += "@" + languageDecimal(p.tokens[i].start)
			}
			if kind == "enum" {
				p.parseEnumConstants(i+1, close)
				return base, close + 1, owner
			}
			for at := i + 1; at < close; {
				if p.skip[at] || !p.typeStart(at) && !languageQualifier(p.text(at)) {
					at++
					continue
				}
				next := p.parseDeclaration(at, owner, true)
				if next <= at {
					at++
				} else {
					at = next
				}
			}
			return base, close + 1, owner
		}
		if name != "" && !p.hasTag(name) {
			p.addTagSymbol(index+1, base, false)
		}
		return base, i, base
	}
	if text == "typeof" || text == "__typeof__" {
		if index+1 < len(p.tokens) && p.text(index+1) == "(" && p.tokens[index+1].match > 0 {
			end := index + 1 + p.tokens[index+1].match + 1
			return p.compactText(index, end), end, ""
		}
		return text, index + 1, ""
	}
	if !languageBuiltinType(text) && !p.typeStart(index) {
		return "", index, ""
	}
	i := index
	base := ""
	for i < len(p.tokens) {
		word := p.text(i)
		if !languageBuiltinType(word) && !languageQualifier(word) {
			break
		}
		if base != "" {
			base += " "
		}
		base += word
		i++
	}
	if base == "" {
		base = text
		i = index + 1
	}
	return base, i, ""
}

func (p *languageParser) parseEnumConstants(start int, end int) {
	expectName := true
	paren, bracket, brace := 0, 0, 0
	for i := start; i < end; i++ {
		text := p.text(i)
		if text == "(" {
			paren++
		} else if text == ")" && paren > 0 {
			paren--
		} else if text == "[" {
			bracket++
		} else if text == "]" && bracket > 0 {
			bracket--
		} else if text == "{" {
			brace++
		} else if text == "}" && brace > 0 {
			brace--
		} else if text == "," && paren == 0 && bracket == 0 && brace == 0 {
			expectName = true
		} else if expectName && tokenKind(p.tokens[i]) == tokenIdent {
			p.declared[i] = true
			p.analysis.addSymbol(languageSymbol{name: text, typeName: "int", signature: "enum constant " + text,
				path: p.file.Path, start: p.tokens[i].start, end: p.tokens[i].end, scopeEnd: len(p.file.Source),
				kind: languageSymbolObject, defined: true})
			expectName = false
		}
	}
}

func (p *languageParser) addTagSymbol(index int, typeName string, defined bool) {
	if index < 0 || index >= len(p.tokens) || p.declared[index] {
		return
	}
	p.declared[index] = true
	p.analysis.addSymbol(languageSymbol{name: p.text(index), typeName: typeName, signature: typeName,
		documentation: languageDocumentation(p.file.Source, p.tokens[index].start), path: p.file.Path,
		start: p.tokens[index].start, end: p.tokens[index].end, scopeEnd: len(p.file.Source), kind: languageSymbolType, defined: defined, external: !defined})
}

func (p *languageParser) hasTag(name string) bool {
	for i := 0; i < len(p.analysis.symbols); i++ {
		if p.analysis.symbols[i].kind == languageSymbolType && p.analysis.symbols[i].name == name &&
			(languageHasPrefix(p.analysis.symbols[i].typeName, "struct ") || languageHasPrefix(p.analysis.symbols[i].typeName, "union ") || languageHasPrefix(p.analysis.symbols[i].typeName, "enum ")) {
			return true
		}
	}
	return false
}

func (p *languageParser) inferInitializerType(name int, end int) string {
	at := name + 1
	for at < end && p.text(at) != "=" {
		at++
	}
	if at+1 >= end {
		return ""
	}
	at++
	pointer := false
	if p.text(at) == "&" {
		pointer = true
		at++
	}
	if at >= end {
		return ""
	}
	kind := tokenKind(p.tokens[at])
	if kind == tokenString {
		return "char *"
	}
	if kind == tokenChar {
		return "int"
	}
	if kind == tokenNumber {
		value := p.text(at)
		for i := 0; i < len(value); i++ {
			if value[i] == '.' || value[i] == 'e' || value[i] == 'E' {
				return "double"
			}
		}
		return "int"
	}
	if kind != tokenIdent {
		return ""
	}
	nameText := p.text(at)
	for i := len(p.analysis.symbols) - 1; i >= 0; i-- {
		symbol := p.analysis.symbols[i]
		if symbol.name == nameText && symbol.kind != languageSymbolType && symbol.kind != languageSymbolField {
			typeName := symbol.typeName
			if pointer {
				typeName += " *"
			}
			return typeName
		}
	}
	return ""
}

func (p *languageParser) declaratorEnd(start int) (int, string) {
	paren, bracket, brace := 0, 0, 0
	for i := start; i < len(p.tokens); i++ {
		if p.skip[i] {
			continue
		}
		text := p.text(i)
		switch text {
		case "(":
			paren++
		case ")":
			if paren > 0 {
				paren--
			}
		case "[":
			bracket++
		case "]":
			if bracket > 0 {
				bracket--
			}
		case "{":
			if paren == 0 && bracket == 0 && brace == 0 {
				return i, "{"
			}
			brace++
		case "}":
			if brace > 0 {
				brace--
			} else if paren == 0 && bracket == 0 {
				return i, "}"
			}
		case ",", ";":
			if paren == 0 && bracket == 0 && brace == 0 {
				return i, text
			}
		}
	}
	return len(p.tokens) - 1, ""
}

func (p *languageParser) declaratorName(start int, end int) int {
	for i := start; i < end; i++ {
		if p.skip[i] || tokenKind(p.tokens[i]) != tokenIdent {
			continue
		}
		text := p.text(i)
		if languageKeyword(text) || languageAttributeWord(text) || p.typeStart(i) {
			continue
		}
		previous := p.previous(i)
		if previous >= start && (p.text(previous) == "." || p.text(previous) == "->") {
			continue
		}
		return i
	}
	return -1
}

func (p *languageParser) declaratorPointers(start int, name int) int {
	count := 0
	for i := start; i < name; i++ {
		if p.text(i) == "*" {
			count++
		}
	}
	return count
}

func (p *languageParser) arrayDeclaratorEnd(start int, end int) int {
	i := start
	for i < end && p.text(i) == "[" && p.tokens[i].match > 0 {
		i += p.tokens[i].match + 1
	}
	return i
}

func (p *languageParser) functionOpen(name int, end int) int {
	for i := name + 1; i < end; i++ {
		if p.text(i) == "(" {
			return i
		}
		if p.text(i) == "=" || p.text(i) == "[" {
			return -1
		}
	}
	return -1
}

func (p *languageParser) functionPointerName(start int, name int) bool {
	for i := start; i < name; i++ {
		if p.text(i) == "(" && i+1 < name && p.text(i+1) == "*" {
			return true
		}
	}
	return false
}

func (p *languageParser) afterFunctionBody(open int, end int) int {
	close := open
	if p.tokens[open].match > 0 {
		close = open + p.tokens[open].match
	}
	for i := close + 1; i <= end && i < len(p.tokens); i++ {
		if p.text(i) == "{" {
			return i
		}
		if p.text(i) == ";" || p.text(i) == "," {
			break
		}
	}
	return -1
}

func (p *languageParser) parseParameters(open int) []LanguageParameter {
	close := open
	if p.tokens[open].match > 0 {
		close = open + p.tokens[open].match
	}
	var parameters []LanguageParameter
	start := open + 1
	paren, bracket := 0, 0
	for i := start; i <= close; i++ {
		text := p.text(i)
		if i == close || text == "," && paren == 0 && bracket == 0 {
			param := p.parameter(start, i)
			if param.Type != "" && !(param.Type == "void" && param.Name == "") {
				parameters = append(parameters, param)
			}
			start = i + 1
			continue
		}
		if text == "(" {
			paren++
		} else if text == ")" && paren > 0 {
			paren--
		} else if text == "[" {
			bracket++
		} else if text == "]" && bracket > 0 {
			bracket--
		}
	}
	return parameters
}

func (p *languageParser) parameter(start int, end int) LanguageParameter {
	if start >= end {
		return LanguageParameter{}
	}
	if p.text(start) == "..." {
		return LanguageParameter{Name: "...", Type: ""}
	}
	base, at, _ := p.parseType(start)
	if at <= start {
		return LanguageParameter{Type: p.compactText(start, end)}
	}
	name := ""
	nameIndex := p.declaratorName(at, end)
	pointer := 0
	if nameIndex >= 0 {
		name = p.text(nameIndex)
		pointer = p.declaratorPointers(at, nameIndex)
		p.declared[nameIndex] = true
	}
	for i := 0; i < pointer; i++ {
		base += " *"
	}
	return LanguageParameter{Name: name, Type: base}
}

func (p *languageParser) addParameterSymbols(parameters []LanguageParameter, open int, searchEnd int, scopeStart int, scopeEnd int) {
	search := open + 1
	for i := 0; i < len(parameters); i++ {
		if parameters[i].Name == "" || parameters[i].Name == "..." {
			continue
		}
		for search < searchEnd {
			if p.text(search) == parameters[i].Name {
				p.declared[search] = true
				p.analysis.addSymbol(languageSymbol{name: parameters[i].Name, typeName: parameters[i].Type,
					signature: parameters[i].Type + " " + parameters[i].Name, path: p.file.Path,
					start: p.tokens[search].start, end: p.tokens[search].end, scopeStart: scopeStart,
					scopeEnd: scopeEnd, kind: languageSymbolObject, defined: true})
				search++
				break
			}
			search++
		}
	}
}

func (p *languageParser) scopeAt(index int) (int, int) {
	bestOpen := -1
	bestEnd := len(p.file.Source)
	for i := index - 1; i >= 0; i-- {
		if p.text(i) != "{" || p.braces[i] <= 0 || p.braces[i]-1 < index {
			continue
		}
		bestOpen = i
		bestEnd = p.tokens[p.braces[i]-1].end
		break
	}
	if bestOpen < 0 {
		return 0, len(p.file.Source)
	}
	return p.tokens[bestOpen].start, bestEnd
}

func (p *languageParser) previous(index int) int {
	for i := index - 1; i >= 0; i-- {
		if !p.skip[i] {
			return i
		}
	}
	return -1
}

func (p *languageParser) compactText(start int, end int) string {
	out := ""
	previousWord := false
	for i := start; i < end && i < len(p.tokens); i++ {
		if p.skip[i] {
			continue
		}
		text := p.text(i)
		word := tokenKind(p.tokens[i]) == tokenIdent || tokenKind(p.tokens[i]) == tokenNumber
		if out != "" && word && previousWord {
			out += " "
		}
		out += text
		previousWord = word
	}
	return out
}

func (a *LanguageAnalysis) addSymbol(symbol languageSymbol) {
	symbol.binding = a.nextBinding
	a.nextBinding++
	if symbol.scopeEnd <= 0 {
		symbol.scopeEnd = 1 << 30
	}
	a.symbols = append(a.symbols, symbol)
}

func (a *LanguageAnalysis) unifyGlobals() {
	for i := 0; i < len(a.symbols); i++ {
		left := &a.symbols[i]
		if left.scopeStart != 0 || left.kind == languageSymbolField {
			continue
		}
		for j := 0; j < i; j++ {
			right := a.symbols[j]
			if right.scopeStart == 0 && right.kind == left.kind && right.name == left.name && right.owner == left.owner &&
				(!left.internal && !right.internal || left.path == right.path) {
				left.binding = right.binding
				break
			}
		}
	}
}

func (a *LanguageAnalysis) buildReferences() {
	for f := 0; f < len(a.files); f++ {
		file := a.files[f]
		scanned := scan(file.Source)
		for i := 0; i < len(scanned.tokens); i++ {
			tok := scanned.tokens[i]
			if tokenKind(tok) != tokenIdent || a.declarationAt(file.Path, tok.start) >= 0 || languageOffsetInDirective(file.Source, tok.start) {
				continue
			}
			name := string(tokenText(file.Source, tok))
			if languageKeyword(name) {
				continue
			}
			binding := a.resolveBinding(file.Path, file.Source, scanned.tokens, i)
			if binding > 0 {
				a.references = append(a.references, languageReference{path: file.Path, start: tok.start, end: tok.end, binding: binding})
			} else if !file.SuppressDiagnostics && !languageUnresolvedIgnored(file.Source, scanned.tokens, i, name) {
				line, column := languageLineColumn(file.Source, tok.start)
				a.diagnostics = append(a.diagnostics, LanguageDiagnostic{Path: file.Path, Start: tok.start, End: tok.end,
					Line: line, Column: column, Code: "RENVO-C-CHECK-002", Message: "undefined identifier: " + name})
			}
		}
	}
}

func (a *LanguageAnalysis) resolveBinding(path string, src []byte, tokens []token, index int) int {
	name := string(tokenText(src, tokens[index]))
	if index > 0 && (tokenIs(src, tokens[index-1], ".") || tokenIs(src, tokens[index-1], "->")) {
		owner := a.memberOwner(path, src, tokens, index-1)
		for i := 0; i < len(a.symbols); i++ {
			if a.symbols[i].kind == languageSymbolField && a.symbols[i].name == name && (owner == "" || a.symbols[i].owner == owner) {
				return a.symbols[i].binding
			}
		}
	}
	offset := tokens[index].start
	best := -1
	bestScope := -1
	for i := 0; i < len(a.symbols); i++ {
		symbol := a.symbols[i]
		if symbol.name != name || symbol.kind == languageSymbolField || symbol.path == path && symbol.start == offset {
			continue
		}
		if symbol.scopeStart > 0 {
			if symbol.path != path || offset < symbol.start || offset > symbol.scopeEnd {
				continue
			}
			if symbol.scopeStart >= bestScope {
				best, bestScope = i, symbol.scopeStart
			}
		} else if symbol.internal && symbol.path != path {
			continue
		} else if best < 0 || symbol.internal && symbol.path == path {
			best = i
		}
	}
	if best >= 0 {
		return a.symbols[best].binding
	}
	return 0
}

func (a *LanguageAnalysis) memberOwner(path string, src []byte, tokens []token, operator int) string {
	if operator <= 0 {
		return ""
	}
	receiver := operator - 1
	if receiver >= 0 && tokenIs(src, tokens[receiver], ")") && tokens[receiver].match < 0 {
		open := receiver + tokens[receiver].match
		receiver = open - 1
	}
	if receiver < 0 || tokenKind(tokens[receiver]) != tokenIdent {
		return ""
	}
	name := string(tokenText(src, tokens[receiver]))
	binding := 0
	if receiver > 0 && (tokenIs(src, tokens[receiver-1], ".") || tokenIs(src, tokens[receiver-1], "->")) {
		binding = a.resolveBinding(path, src, tokens, receiver)
	} else {
		binding = a.bindingForNameAt(path, name, tokens[receiver].start)
	}
	if binding <= 0 {
		return ""
	}
	for i := 0; i < len(a.symbols); i++ {
		if a.symbols[i].binding == binding {
			return languageBaseAggregate(a.symbols[i].typeName)
		}
	}
	return ""
}

func (a *LanguageAnalysis) bindingForNameAt(path string, name string, offset int) int {
	best, bestScope := -1, -1
	for i := 0; i < len(a.symbols); i++ {
		symbol := a.symbols[i]
		if symbol.name != name || symbol.kind == languageSymbolField {
			continue
		}
		if symbol.scopeStart > 0 {
			if symbol.path == path && offset >= symbol.start && offset <= symbol.scopeEnd && symbol.scopeStart >= bestScope {
				best, bestScope = i, symbol.scopeStart
			}
		} else if symbol.internal && symbol.path != path {
			continue
		} else if best < 0 || symbol.internal && symbol.path == path {
			best = i
		}
	}
	if best >= 0 {
		return a.symbols[best].binding
	}
	return 0
}

func (a *LanguageAnalysis) declarationAt(path string, offset int) int {
	for i := 0; i < len(a.symbols); i++ {
		if a.symbols[i].path == path && a.symbols[i].start <= offset && offset < a.symbols[i].end {
			return i
		}
	}
	return -1
}

func (a *LanguageAnalysis) referenceAt(path string, offset int) int {
	for i := 0; i < len(a.references); i++ {
		if a.references[i].path == path && a.references[i].start <= offset && offset < a.references[i].end ||
			a.references[i].path == path && a.references[i].end == offset {
			return i
		}
	}
	return -1
}

func (a *LanguageAnalysis) symbolAt(path string, offset int) int {
	if symbol := a.declarationAt(path, offset); symbol >= 0 {
		return symbol
	}
	if reference := a.referenceAt(path, offset); reference >= 0 {
		for i := 0; i < len(a.symbols); i++ {
			if a.symbols[i].binding == a.references[reference].binding {
				return i
			}
		}
	}
	return -1
}

// Complete returns names visible at path and offset, including aggregate
// members after '.' and '->'.
func (a *LanguageAnalysis) Complete(path string, offset int) []LanguageCompletion {
	src := a.source(path)
	if offset < 0 {
		offset = 0
	}
	if offset > len(src) {
		offset = len(src)
	}
	prefixStart := offset
	for prefixStart > 0 && isIdentPart(src[prefixStart-1]) {
		prefixStart--
	}
	prefix := string(src[prefixStart:offset])
	member := false
	owner := ""
	operator := prefixStart - 1
	for operator >= 0 && (src[operator] == ' ' || src[operator] == '\t' || src[operator] == '\r' || src[operator] == '\n') {
		operator--
	}
	if operator >= 0 && src[operator] == '.' {
		member = true
	} else if operator >= 1 && src[operator-1] == '-' && src[operator] == '>' {
		member = true
		operator--
	}
	if member {
		scanned := scan(src)
		for i := 0; i < len(scanned.tokens); i++ {
			if scanned.tokens[i].start == operator {
				owner = a.memberOwner(path, src, scanned.tokens, i)
				break
			}
		}
	}
	var items []LanguageCompletion
	for i := 0; i < len(a.symbols); i++ {
		symbol := a.symbols[i]
		if member {
			if symbol.kind != languageSymbolField || owner != "" && symbol.owner != owner {
				continue
			}
		} else {
			if symbol.kind == languageSymbolField || symbol.internal && symbol.path != path || symbol.scopeStart > 0 && (symbol.path != path || offset < symbol.start || offset > symbol.scopeEnd) {
				continue
			}
		}
		if !languageHasPrefix(symbol.name, prefix) || languageCompletionExists(items, symbol.name) {
			continue
		}
		items = append(items, languageCompletionForSymbol(symbol))
	}
	if !member {
		for i := 0; i < len(languageKeywords); i++ {
			if languageHasPrefix(languageKeywords[i], prefix) {
				items = append(items, LanguageCompletion{Name: languageKeywords[i], Detail: "keyword", Kind: LanguageKeyword})
			}
		}
	}
	languageSortCompletions(items, prefix)
	return items
}

// CompleteGlobals exposes C declarations to a Go file in a mixed project.
// Local C scopes and C keywords are intentionally excluded.
func (a *LanguageAnalysis) CompleteGlobals(prefix string) []LanguageCompletion {
	var items []LanguageCompletion
	for i := 0; i < len(a.symbols); i++ {
		symbol := a.symbols[i]
		if symbol.scopeStart != 0 || symbol.internal || symbol.kind == languageSymbolField || !languageHasPrefix(symbol.name, prefix) || languageCompletionExists(items, symbol.name) {
			continue
		}
		items = append(items, languageCompletionForSymbol(symbol))
	}
	languageSortCompletions(items, prefix)
	return items
}

func languageCompletionForSymbol(symbol languageSymbol) LanguageCompletion {
	kind := LanguageVariable
	detail := symbol.typeName
	if symbol.kind == languageSymbolFunction {
		kind, detail = LanguageFunction, "function"
	} else if symbol.kind == languageSymbolType {
		kind, detail = LanguageType, "type"
	} else if symbol.kind == languageSymbolField {
		kind, detail = LanguageField, symbol.typeName
	} else if symbol.kind == languageSymbolMacro {
		detail = "macro"
	}
	return LanguageCompletion{Name: symbol.name, Detail: detail, Kind: kind, Signature: symbol.signature,
		Documentation: symbol.documentation, Parameters: symbol.parameters}
}

// Hover returns the inferred C declaration/type at path and offset.
func (a *LanguageAnalysis) Hover(path string, offset int) LanguageHover {
	symbolIndex := a.symbolAt(path, offset)
	if symbolIndex < 0 {
		return LanguageHover{}
	}
	symbol := a.symbols[symbolIndex]
	start, end := symbol.start, symbol.end
	if reference := a.referenceAt(path, offset); reference >= 0 {
		start, end = a.references[reference].start, a.references[reference].end
	}
	return LanguageHover{Signature: symbol.signature, Documentation: symbol.documentation, Start: start, End: end, Ok: true}
}

// HoverName supports the mixed-language boundary where a Go identifier is
// satisfied by an extern C declaration. start and end are the querying Go
// token's span, while the signature and documentation still come from C.
func (a *LanguageAnalysis) HoverName(name string, start int, end int) LanguageHover {
	for i := 0; i < len(a.symbols); i++ {
		if a.symbols[i].name == name && a.symbols[i].kind != languageSymbolField {
			return LanguageHover{Signature: a.symbols[i].signature, Documentation: a.symbols[i].documentation,
				Start: start, End: end, Ok: true}
		}
	}
	return LanguageHover{}
}

// Signature returns help for the innermost call containing offset.
func (a *LanguageAnalysis) Signature(path string, offset int) LanguageSignature {
	src := a.source(path)
	if offset > len(src) {
		offset = len(src)
	}
	scanned := scan(src)
	open := -1
	paren, bracket, brace := 0, 0, 0
	for i := len(scanned.tokens) - 1; i >= 0; i-- {
		tok := scanned.tokens[i]
		if tok.start >= offset || tokenKind(tok) == tokenEOF {
			continue
		}
		text := string(tokenText(src, tok))
		switch text {
		case ")":
			paren++
		case "]":
			bracket++
		case "}":
			brace++
		case "(":
			if paren > 0 {
				paren--
			} else if bracket == 0 && brace == 0 {
				open = i
				i = -1
			}
		case "[":
			if bracket > 0 {
				bracket--
			}
		case "{":
			if brace > 0 {
				brace--
			}
		}
	}
	if open <= 0 || tokenKind(scanned.tokens[open-1]) != tokenIdent {
		return LanguageSignature{}
	}
	name := string(tokenText(src, scanned.tokens[open-1]))
	binding := a.bindingForNameAt(path, name, scanned.tokens[open-1].start)
	for i := 0; i < len(a.symbols); i++ {
		symbol := a.symbols[i]
		if symbol.binding != binding || symbol.kind != languageSymbolFunction && symbol.kind != languageSymbolMacro {
			continue
		}
		active := 0
		paren, bracket, brace = 0, 0, 0
		for j := open + 1; j < len(scanned.tokens) && scanned.tokens[j].start < offset; j++ {
			text := string(tokenText(src, scanned.tokens[j]))
			if text == "(" {
				paren++
			} else if text == ")" && paren > 0 {
				paren--
			} else if text == "[" {
				bracket++
			} else if text == "]" && bracket > 0 {
				bracket--
			} else if text == "{" {
				brace++
			} else if text == "}" && brace > 0 {
				brace--
			} else if text == "," && paren == 0 && bracket == 0 && brace == 0 {
				active++
			}
		}
		if len(symbol.parameters) > 0 && active >= len(symbol.parameters) {
			active = len(symbol.parameters) - 1
		}
		return LanguageSignature{Label: symbol.signature, Parameters: symbol.parameters, ActiveParameter: active, Ok: true}
	}
	return LanguageSignature{}
}

// SignatureName returns a global C function signature for mixed-language use.
func (a *LanguageAnalysis) SignatureName(name string) LanguageSignature {
	for i := 0; i < len(a.symbols); i++ {
		if a.symbols[i].name == name && (a.symbols[i].kind == languageSymbolFunction || a.symbols[i].kind == languageSymbolMacro) {
			return LanguageSignature{Label: a.symbols[i].signature, Parameters: a.symbols[i].parameters, Ok: true}
		}
	}
	return LanguageSignature{}
}

// Navigate resolves the C name at path and offset.
func (a *LanguageAnalysis) Navigate(path string, offset int) LanguageNavigation {
	symbolIndex := a.symbolAt(path, offset)
	if symbolIndex < 0 {
		return LanguageNavigation{}
	}
	symbol := a.symbols[symbolIndex]
	binding := symbol.binding
	definition := symbol
	for i := 0; i < len(a.symbols); i++ {
		candidate := a.symbols[i]
		if candidate.binding == binding && candidate.defined && (!definition.defined || definition.external || candidate.path == path) {
			definition = candidate
		}
	}
	result := LanguageNavigation{Name: symbol.name,
		Definition:              LanguageLocation{Path: definition.path, Start: definition.start, End: definition.end},
		DefinitionIsDeclaration: !definition.defined, Ok: true}
	for i := 0; i < len(a.symbols); i++ {
		if a.symbols[i].binding == binding {
			result.References = languageAppendLocation(result.References, LanguageLocation{Path: a.symbols[i].path, Start: a.symbols[i].start, End: a.symbols[i].end})
		}
	}
	for i := 0; i < len(a.references); i++ {
		if a.references[i].binding == binding {
			result.References = languageAppendLocation(result.References, LanguageLocation{Path: a.references[i].path, Start: a.references[i].start, End: a.references[i].end})
		}
	}
	return result
}

// NavigateName resolves a global C name from a mixed-language query.
func (a *LanguageAnalysis) NavigateName(name string) LanguageNavigation {
	for i := 0; i < len(a.symbols); i++ {
		if a.symbols[i].name == name && a.symbols[i].kind != languageSymbolField {
			return a.navigateSymbol(i)
		}
	}
	return LanguageNavigation{}
}

func (a *LanguageAnalysis) navigateSymbol(symbolIndex int) LanguageNavigation {
	if symbolIndex < 0 || symbolIndex >= len(a.symbols) {
		return LanguageNavigation{}
	}
	symbol := a.symbols[symbolIndex]
	binding := symbol.binding
	definition := symbol
	for i := 0; i < len(a.symbols); i++ {
		candidate := a.symbols[i]
		if candidate.binding == binding && candidate.defined && (!definition.defined || definition.external) {
			definition = candidate
		}
	}
	result := LanguageNavigation{Name: symbol.name,
		Definition:              LanguageLocation{Path: definition.path, Start: definition.start, End: definition.end},
		DefinitionIsDeclaration: !definition.defined, Ok: true}
	for i := 0; i < len(a.symbols); i++ {
		if a.symbols[i].binding == binding {
			result.References = languageAppendLocation(result.References, LanguageLocation{Path: a.symbols[i].path, Start: a.symbols[i].start, End: a.symbols[i].end})
		}
	}
	for i := 0; i < len(a.references); i++ {
		if a.references[i].binding == binding {
			result.References = languageAppendLocation(result.References, LanguageLocation{Path: a.references[i].path, Start: a.references[i].start, End: a.references[i].end})
		}
	}
	return result
}

func (a *LanguageAnalysis) source(path string) []byte {
	for i := 0; i < len(a.files); i++ {
		if a.files[i].Path == path {
			return a.files[i].Source
		}
	}
	return nil
}

func languageAppendLocation(locations []LanguageLocation, location LanguageLocation) []LanguageLocation {
	for i := 0; i < len(locations); i++ {
		if locations[i] == location {
			return locations
		}
	}
	return append(locations, location)
}

var languageKeywords = []string{"auto", "break", "case", "char", "const", "continue", "default", "do", "double", "else", "enum", "extern", "float", "for", "goto", "if", "inline", "int", "long", "register", "restrict", "return", "short", "signed", "sizeof", "static", "struct", "switch", "typedef", "union", "unsigned", "void", "volatile", "while", "_Alignas", "_Alignof", "_Atomic", "_Bool", "_Generic", "_Noreturn", "_Static_assert", "_Thread_local"}

func languageKeyword(value string) bool {
	for i := 0; i < len(languageKeywords); i++ {
		if languageKeywords[i] == value {
			return true
		}
	}
	return languageStorage(value) || languageQualifier(value) || languageFunctionSpecifier(value) || languageAttributeWord(value)
}

func languageBuiltinType(value string) bool {
	return value == "void" || value == "_Bool" || value == "char" || value == "short" || value == "int" || value == "long" || value == "signed" || value == "unsigned" || value == "float" || value == "double" || value == "__int128" || value == "__auto_type"
}

func languageStorage(value string) bool {
	return value == "auto" || value == "register" || value == "static" || value == "extern" || value == "typedef" || value == "_Thread_local" || value == "__thread"
}

func languageQualifier(value string) bool {
	return value == "const" || value == "volatile" || value == "restrict" || value == "_Atomic" || value == "__restrict" || value == "__restrict__" || value == "__const" || value == "__const__" || value == "__volatile" || value == "__volatile__"
}

func languageFunctionSpecifier(value string) bool {
	return value == "inline" || value == "_Noreturn" || value == "__inline" || value == "__inline__"
}

func languageAttributeWord(value string) bool {
	return value == "__attribute__" || value == "__declspec" || value == "asm" || value == "__asm" || value == "__asm__"
}

func languageParameterText(parameters []LanguageParameter) string {
	out := ""
	for i := 0; i < len(parameters); i++ {
		if i > 0 {
			out += ", "
		}
		if parameters[i].Name == "..." {
			out += "..."
			continue
		}
		out += parameters[i].Type
		if parameters[i].Name != "" {
			out += " " + parameters[i].Name
		}
	}
	return out
}

func languageDocumentation(src []byte, at int) string {
	i := at
	for i > 0 && (src[i-1] == ' ' || src[i-1] == '\t' || src[i-1] == '\r' || src[i-1] == '\n') {
		i--
	}
	if i >= 2 && src[i-2] == '*' && src[i-1] == '/' {
		start := i - 2
		for start >= 2 && !(src[start-2] == '/' && src[start-1] == '*') {
			start--
		}
		if start >= 2 {
			return languageCleanComment(string(src[start:i]))
		}
	}
	end := i
	start := end
	for start > 0 {
		lineStart := start
		for lineStart > 0 && src[lineStart-1] != '\n' {
			lineStart--
		}
		atLine := lineStart
		for atLine < start && (src[atLine] == ' ' || src[atLine] == '\t') {
			atLine++
		}
		if atLine+1 >= start || src[atLine] != '/' || src[atLine+1] != '/' {
			break
		}
		start = lineStart
		if start > 0 {
			start--
		}
	}
	if start < end {
		return languageCleanComment(string(src[start:end]))
	}
	return ""
}

func languageCleanComment(value string) string {
	var out []byte
	for i := 0; i < len(value); {
		if i+1 < len(value) && value[i] == '/' && (value[i+1] == '/' || value[i+1] == '*') {
			i += 2
			continue
		}
		if i+1 < len(value) && value[i] == '*' && value[i+1] == '/' {
			i += 2
			continue
		}
		if value[i] == '*' && (i == 0 || value[i-1] == '\n') {
			i++
			continue
		}
		out = append(out, value[i])
		i++
	}
	trimmed := languageTrimSpace(out)
	return string(trimmed)
}

func languageBaseAggregate(value string) string {
	end := len(value)
	for end > 0 && (value[end-1] == ' ' || value[end-1] == '*') {
		end--
	}
	return value[:end]
}

func languageLineColumn(src []byte, offset int) (int, int) {
	if offset < 0 {
		offset = 0
	}
	if offset > len(src) {
		offset = len(src)
	}
	line, column := 1, 1
	for i := 0; i < offset; i++ {
		if src[i] == '\n' {
			line, column = line+1, 1
		} else {
			column++
		}
	}
	return line, column
}

func languageSkipSpace(src []byte, at int, end int) int {
	for at < end && (src[at] == ' ' || src[at] == '\t' || src[at] == '\r') {
		at++
	}
	return at
}
func languageSkipIdent(src []byte, at int, end int) int {
	for at < end && isIdentPart(src[at]) {
		at++
	}
	return at
}
func languageTrimSpace(src []byte) []byte {
	for len(src) > 0 && (src[0] == ' ' || src[0] == '\t' || src[0] == '\r' || src[0] == '\n') {
		src = src[1:]
	}
	for len(src) > 0 && (src[len(src)-1] == ' ' || src[len(src)-1] == '\t' || src[len(src)-1] == '\r' || src[len(src)-1] == '\n') {
		src = src[:len(src)-1]
	}
	return src
}
func languageHasPrefix(value string, prefix string) bool {
	return len(value) >= len(prefix) && value[:len(prefix)] == prefix
}
func languageDecimal(value int) string {
	if value == 0 {
		return "0"
	}
	var reverse [24]byte
	count := 0
	for value > 0 {
		reverse[count] = byte('0' + value%10)
		count++
		value /= 10
	}
	out := make([]byte, count)
	for i := 0; i < count; i++ {
		out[i] = reverse[count-i-1]
	}
	return string(out)
}

func languageOffsetInDirective(src []byte, offset int) bool {
	start := offset
	for start > 0 && src[start-1] != '\n' {
		start--
	}
	for start < len(src) && (src[start] == ' ' || src[start] == '\t') {
		start++
	}
	return start < len(src) && src[start] == '#'
}

func languageUnresolvedIgnored(src []byte, tokens []token, index int, name string) bool {
	if languageHasPrefix(name, "__builtin_") || name == "__func__" || name == "__FUNCTION__" || name == "__PRETTY_FUNCTION__" ||
		name == "__extension__" || name == "__label__" {
		return true
	}
	previous, next := "", ""
	if index > 0 {
		previous = string(tokenText(src, tokens[index-1]))
	}
	if index+1 < len(tokens) {
		next = string(tokenText(src, tokens[index+1]))
	}
	if previous == "goto" || next == ":" || previous == "asm" || previous == "__asm" || previous == "__asm__" ||
		previous == "__attribute__" || previous == "__declspec" {
		return true
	}
	return false
}

func languageCompletionExists(items []LanguageCompletion, name string) bool {
	for i := 0; i < len(items); i++ {
		if items[i].Name == name {
			return true
		}
	}
	return false
}

func languageSortCompletions(items []LanguageCompletion, prefix string) {
	for i := 1; i < len(items); i++ {
		value := items[i]
		j := i
		for j > 0 && languageCompletionLess(value, items[j-1], prefix) {
			items[j] = items[j-1]
			j--
		}
		items[j] = value
	}
}

func languageCompletionLess(left LanguageCompletion, right LanguageCompletion, prefix string) bool {
	leftExact, rightExact := left.Name == prefix, right.Name == prefix
	if leftExact != rightExact {
		return leftExact
	}
	return left.Name < right.Name
}
