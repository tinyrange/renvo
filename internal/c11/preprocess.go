package c11

const (
	PreprocessOK = iota
	PreprocessErrToken
	PreprocessErrDirective
	PreprocessErrInclude
	PreprocessErrExpression
	PreprocessErrMacro
	PreprocessErrDepth
)

// Macro is one command-line or target predefined macro. Value is tokenized
// once when preprocessing starts; ordinary source definitions use the same
// compact replacement-list representation.
type Macro struct {
	Name  string
	Value string
}

// PreprocessConfig describes one translation unit. Included declarations are
// emitted for -E, but object translation can suppress them while retaining
// their definitions and expanding the root source.
type PreprocessConfig struct {
	Path           string
	Source         []byte
	Reader         IncludeReader
	Predefined     []Macro
	Undefined      []string
	ForcedIncludes []string
	EmitIncludes   bool
	// EmitQuotedIncludes keeps project-header declarations in object-mode
	// translation while system-header declarations remain demand-selected.
	EmitQuotedIncludes bool
	// SuppressForcedIncludes retains forced-header macro state without adding
	// its declaration tokens to the result.
	SuppressForcedIncludes bool
	LineMarkers            bool
}

// PreprocessResult owns only the final rendered token stream plus dependency
// paths. Expansion itself uses byte spans into a source table rather than a
// pointer-rich token graph.
type PreprocessResult struct {
	Source       []byte
	Dependencies []string
	Ok           bool
	Error        int
	ErrorPath    string
	Line         int
	Detail       string
}

const (
	ppEOF = iota
	ppIdent
	ppNumber
	ppString
	ppChar
	ppPunct
)

const (
	ppKindMask    = 15
	ppSpaceFlag   = 16
	ppSourceShift = 5
	ppSourceBits  = 18
	ppSourceMask  = (1 << ppSourceBits) - 1
	ppHideShift   = ppSourceShift + ppSourceBits
)

// ppToken is three machine words, matching the ordinary C scanner token. Its
// spelling is a span into sources[meta>>ppSourceShift].
type ppToken struct {
	start int
	end   int
	meta  int
}

func ppMakeToken(kind int, source int, start int, end int, space bool) ppToken {
	meta := kind | source<<ppSourceShift
	if space {
		meta |= ppSpaceFlag
	}
	return ppToken{start: start, end: end, meta: meta}
}

func ppTokenKind(tok ppToken) int   { return tok.meta & ppKindMask }
func ppTokenSource(tok ppToken) int { return tok.meta >> ppSourceShift & ppSourceMask }
func ppTokenHide(tok ppToken) int   { return tok.meta >> ppHideShift }
func ppTokenSpace(tok ppToken) bool { return tok.meta&ppSpaceFlag != 0 }
func ppWithSpace(tok ppToken, space bool) ppToken {
	if space {
		tok.meta |= ppSpaceFlag
	} else {
		tok.meta &^= ppSpaceFlag
	}
	return tok
}

func ppWithHide(tok ppToken, hide int) ppToken {
	tok.meta = tok.meta&(1<<ppHideShift-1) | hide<<ppHideShift
	return tok
}

type ppHide struct {
	macro int
	next  int
}

type ppMacro struct {
	name     string
	params   []string
	replace  []ppToken
	function bool
	variadic bool
	deleted  bool
}

type ppConditional struct {
	parentActive bool
	active       bool
	matched      bool
	seenElse     bool
}

type preprocessor struct {
	config         PreprocessConfig
	sources        [][]byte
	macros         []ppMacro
	buckets        []int
	hides          []ppHide
	conditionals   []ppConditional
	dependencies   []string
	once           []string
	out            []byte
	ok             bool
	err            int
	errorPath      string
	errorLine      int
	errorDetail    string
	currentPath    string
	currentDisplay string
	currentLine    int
	counter        int
}

// Preprocess runs translation phases 2-4 for one C translation unit.
func Preprocess(config PreprocessConfig) PreprocessResult {
	p := preprocessor{config: config, ok: true}
	p.sources = append(p.sources, nil) // source zero is generated spelling.
	p.hides = append(p.hides, ppHide{})
	p.installPredefined(config.Predefined)
	for i := 0; i < len(config.Undefined); i++ {
		if index := p.lookupName([]byte(config.Undefined[i])); index >= 0 {
			p.macros[index].deleted = true
		}
	}
	for i := 0; i < len(config.ForcedIncludes) && p.ok; i++ {
		if config.Reader == nil {
			p.fail(PreprocessErrInclude, config.ForcedIncludes[i], 1)
			break
		}
		src, path, ok := config.Reader.ReadInclude(config.Path, config.ForcedIncludes[i], false)
		if !ok {
			p.fail(PreprocessErrInclude, config.ForcedIncludes[i], 1)
			break
		}
		p.addDependency(path)
		// A compiler-forced header supplies the translation environment. Object
		// mode retains its macros and dependencies without injecting its often
		// target-specific declaration corpus ahead of the source-level frontend.
		p.processFile(path, src, config.EmitIncludes && !config.SuppressForcedIncludes, 0)
	}
	if p.ok {
		p.processFile(config.Path, config.Source, true, 0)
	}
	if p.ok && len(p.conditionals) != 0 {
		p.fail(PreprocessErrDirective, p.currentPath, p.currentLine)
	}
	if !p.ok {
		return PreprocessResult{Ok: false, Error: p.err, ErrorPath: p.errorPath, Line: p.errorLine, Detail: p.errorDetail}
	}
	return PreprocessResult{Source: p.out, Dependencies: p.dependencies, Ok: true, Error: PreprocessOK}
}

// PreprocessProbe preserves the compiler-identity API while using the same
// engine as real translation units.
func PreprocessProbe(src []byte, predefined []Macro) PreprocessResult {
	return Preprocess(PreprocessConfig{Path: "<stdin>", Source: src, Predefined: predefined, EmitIncludes: true})
}

func (p *preprocessor) installPredefined(predefined []Macro) {
	p.installMacroText(ppDefaultMacroText)
	for i := 0; i < len(predefined); i++ {
		p.defineText(predefined[i].Name, nil, false, false, predefined[i].Value)
	}
	for start := 0; start < len(ppFeatureMacros); {
		end := start
		for end < len(ppFeatureMacros) && ppFeatureMacros[end] != ' ' {
			end++
		}
		p.defineText(ppFeatureMacros[start:end], []string{"x"}, true, false, "0")
		start = end + 1
	}
}

const ppFeatureMacros = "__has_attribute __has_builtin __has_feature __has_extension __has_c_attribute"

const ppDefaultMacroText = `__STDC__=1
__STDC_VERSION__=201112L
__STDC_HOSTED__=0
__RENVO__=1
__GNUC__=5
__GNUC_MINOR__=1
__GNUC_PATCHLEVEL__=0
__VERSION__="Renvo 5.1.0 compatible"
__x86_64__=1
__x86_64=1
__amd64__=1
__linux__=1
__linux=1
linux=1
__ELF__=1
__LP64__=1
_LP64=1
__CHAR_BIT__=8
__SIZEOF_SHORT__=2
__SIZEOF_INT__=4
__SIZEOF_LONG__=8
__SIZEOF_LONG_LONG__=8
__SIZEOF_POINTER__=8
__SIZEOF_SIZE_T__=8
__SIZEOF_PTRDIFF_T__=8
__SIZEOF_INT128__=16
__SIZE_TYPE__=long unsigned int
__PTRDIFF_TYPE__=long int
__INTPTR_TYPE__=long int
__UINTPTR_TYPE__=long unsigned int
__ORDER_LITTLE_ENDIAN__=1234
__ORDER_BIG_ENDIAN__=4321
__BYTE_ORDER__=__ORDER_LITTLE_ENDIAN__
__GCC_ATOMIC_BOOL_LOCK_FREE=2
__GCC_ATOMIC_CHAR_LOCK_FREE=2
__GCC_ATOMIC_SHORT_LOCK_FREE=2
__GCC_ATOMIC_INT_LOCK_FREE=2
__GCC_ATOMIC_LONG_LOCK_FREE=2
__GCC_ATOMIC_LLONG_LOCK_FREE=2
__GCC_ATOMIC_POINTER_LOCK_FREE=2`

func (p *preprocessor) installMacroText(text string) {
	for start := 0; start < len(text); {
		separator := start
		for separator < len(text) && text[separator] != '=' {
			separator++
		}
		end := separator + 1
		for end < len(text) && text[end] != '\n' {
			end++
		}
		p.defineText(text[start:separator], nil, false, false, text[separator+1:end])
		start = end + 1
	}
}

func (p *preprocessor) processFile(path string, src []byte, emit bool, depth int) {
	if !p.ok {
		return
	}
	if depth >= 128 {
		p.fail(PreprocessErrDepth, path, 1)
		return
	}
	if findPPText(p.once, path) >= 0 {
		return
	}
	normalized, lines, lineNumbers, ok := ppNormalize(src)
	if !ok {
		p.fail(PreprocessErrToken, path, 1)
		return
	}
	sourceID := len(p.sources)
	p.sources = append(p.sources, normalized)
	previousPath, previousDisplay, previousLine := p.currentPath, p.currentDisplay, p.currentLine
	p.currentPath = path
	p.currentDisplay = path
	lineBias := 0
	displayPath := path
	baseConditional := len(p.conditionals)
	var pending []ppToken
	pendingLine := 0
	parenDepth := 0
	if emit && p.config.LineMarkers {
		p.appendLineMarker(1, displayPath)
	}
	for i := 0; i+1 < len(lines) && p.ok; i++ {
		physicalLine := lineNumbers[i]
		p.currentLine = physicalLine + lineBias
		start, end := lines[i], lines[i+1]
		if end > start && normalized[end-1] == '\n' {
			end--
		}
		tokens, valid := p.lex(sourceID, start, end)
		if !valid {
			p.errorDetail = string(normalized[start:end])
			p.fail(PreprocessErrToken, displayPath, p.currentLine)
			break
		}
		if len(tokens) > 0 && p.tokenIs(tokens[0], "#") {
			if len(pending) > 0 {
				p.flushPending(pending, pendingLine)
				pending = nil
				parenDepth = 0
			}
			newBias, newPath := p.directive(tokens[1:], path, displayPath, p.currentLine, emit, depth)
			if newPath != "" {
				displayPath = newPath
				p.currentDisplay = newPath
			}
			if newBias != 0 {
				lineBias = newBias - physicalLine - 1
				if emit && p.config.LineMarkers {
					p.appendLineMarker(newBias, displayPath)
				}
			}
			continue
		}
		if p.active() && emit {
			if len(tokens) == 0 {
				continue
			}
			if len(pending) == 0 {
				pendingLine = p.currentLine
			}
			pending = append(pending, tokens...)
			for j := 0; j < len(tokens); j++ {
				if p.tokenIs(tokens[j], "(") {
					parenDepth++
				} else if p.tokenIs(tokens[j], ")") {
					parenDepth--
				}
			}
			if parenDepth <= 0 {
				p.flushPending(pending, pendingLine)
				pending = nil
				parenDepth = 0
			}
		}
	}
	if p.ok && len(pending) > 0 {
		p.flushPending(pending, pendingLine)
	}
	if p.ok && len(p.conditionals) != baseConditional {
		p.fail(PreprocessErrDirective, displayPath, p.currentLine)
	}
	p.conditionals = p.conditionals[:baseConditional]
	p.currentPath, p.currentDisplay, p.currentLine = previousPath, previousDisplay, previousLine
}

func (p *preprocessor) flushPending(tokens []ppToken, line int) {
	currentLine := p.currentLine
	p.currentLine = line
	expanded := p.expand(tokens, 0)
	if p.ok {
		p.renderLine(expanded)
	}
	p.currentLine = currentLine
}

func (p *preprocessor) active() bool {
	return len(p.conditionals) == 0 || p.conditionals[len(p.conditionals)-1].active
}

func (p *preprocessor) fail(err int, path string, line int) {
	if !p.ok {
		return
	}
	p.ok = false
	p.err = err
	p.errorPath = path
	p.errorLine = line
}

func (p *preprocessor) addDependency(path string) {
	if findPPText(p.dependencies, path) < 0 {
		p.dependencies = append(p.dependencies, path)
	}
}

func findPPText(values []string, value string) int {
	for i := 0; i < len(values); i++ {
		if values[i] == value {
			return i
		}
	}
	return -1
}

// ppNormalize performs line splicing before replacing comments with spaces.
// Newlines inside block comments are retained so diagnostics remain stable.
func ppNormalize(src []byte) ([]byte, []int, []int, bool) {
	out := make([]byte, 0, len(src)+1)
	lines := []int{0}
	lineNumbers := []int{1}
	physicalLine := 1
	inString := byte(0)
	for i := 0; i < len(src); {
		if src[i] == '\\' && i+1 < len(src) && src[i+1] == '\n' {
			physicalLine++
			i += 2
			continue
		}
		if inString != 0 {
			ch := src[i]
			out = append(out, ch)
			i++
			if ch == '\\' && i < len(src) {
				out = append(out, src[i])
				i++
			} else if ch == inString {
				inString = 0
			} else if ch == '\n' {
				return nil, nil, nil, false
			}
			continue
		}
		if src[i] == '"' || src[i] == '\'' {
			inString = src[i]
			out = append(out, src[i])
			i++
			continue
		}
		if src[i] == '/' && i+1 < len(src) && src[i+1] == '/' {
			out = append(out, ' ')
			i += 2
			for i < len(src) && src[i] != '\n' {
				if src[i] == '\\' && i+1 < len(src) && src[i+1] == '\n' {
					physicalLine++
					i += 2
					continue
				}
				i++
			}
			continue
		}
		if src[i] == '/' && i+1 < len(src) && src[i+1] == '*' {
			out = append(out, ' ')
			i += 2
			closed := false
			for i < len(src) {
				if src[i] == '\\' && i+1 < len(src) && src[i+1] == '\n' {
					physicalLine++
					i += 2
					continue
				}
				if i+1 < len(src) && src[i] == '*' && src[i+1] == '/' {
					i += 2
					closed = true
					break
				}
				if src[i] == '\n' {
					out = append(out, '\n')
					physicalLine++
					lines = append(lines, len(out))
					lineNumbers = append(lineNumbers, physicalLine)
				}
				i++
			}
			if !closed {
				return nil, nil, nil, false
			}
			continue
		}
		out = append(out, src[i])
		if src[i] == '\n' {
			physicalLine++
			lines = append(lines, len(out))
			lineNumbers = append(lineNumbers, physicalLine)
		}
		i++
	}
	if inString != 0 {
		return nil, nil, nil, false
	}
	if len(out) == 0 || out[len(out)-1] != '\n' {
		out = append(out, '\n')
		lines = append(lines, len(out))
		lineNumbers = append(lineNumbers, physicalLine+1)
	}
	return out, lines, lineNumbers, true
}

func (p *preprocessor) lex(source int, start int, end int) ([]ppToken, bool) {
	src := p.sources[source]
	var out []ppToken
	space := false
	for i := start; i < end; {
		ch := src[i]
		if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\v' || ch == '\f' {
			space = true
			i++
			continue
		}
		tokenStart := i
		kind := ppPunct
		if isIdentStart(ch) {
			kind = ppIdent
			i++
			for i < end && isIdentPart(src[i]) {
				i++
			}
			if i < end && (src[i] == '"' || src[i] == '\'') && literalPrefix(src[tokenStart:i]) {
				delim := src[i]
				i++
				for i < end && src[i] != delim {
					if src[i] == '\\' {
						i++
					}
					i++
				}
				if i >= end {
					return nil, false
				}
				i++
				kind = ppString
				if delim == '\'' {
					kind = ppChar
				}
			}
		} else if isDigit(ch) || ch == '.' && i+1 < end && isDigit(src[i+1]) {
			kind = ppNumber
			i = scanNumber(src[:end], i)
		} else if ch == '"' || ch == '\'' {
			delim := ch
			i++
			for i < end && src[i] != delim {
				if src[i] == '\\' {
					i++
				}
				i++
			}
			if i >= end {
				return nil, false
			}
			i++
			kind = ppString
			if delim == '\'' {
				kind = ppChar
			}
		} else {
			i = punctEnd(src[:end], i)
			if i == tokenStart {
				// GNU headers carry assembler macro spellings such as \@ and $
				// through preprocessing. Preserve printable extension bytes as
				// single tokens; the later language parser remains strict.
				if ch < 32 || ch == 127 {
					return nil, false
				}
				i++
			}
		}
		out = append(out, ppMakeToken(kind, source, tokenStart, i, space))
		space = false
	}
	return out, true
}

func (p *preprocessor) tokenText(tok ppToken) []byte {
	source := ppTokenSource(tok)
	if source < 0 || source >= len(p.sources) || tok.start < 0 || tok.end < tok.start || tok.end > len(p.sources[source]) {
		return nil
	}
	return p.sources[source][tok.start:tok.end]
}

func (p *preprocessor) tokenIs(tok ppToken, text string) bool {
	return ppBytesStringEqual(p.tokenText(tok), text)
}

func (p *preprocessor) generatedToken(kind int, text []byte, space bool) ppToken {
	start := len(p.sources[0])
	p.sources[0] = append(p.sources[0], text...)
	return ppMakeToken(kind, 0, start, len(p.sources[0]), space)
}

func (p *preprocessor) renderLine(tokens []ppToken) {
	for i := 0; i < len(tokens); i++ {
		if i > 0 {
			p.out = append(p.out, ' ')
		}
		p.out = append(p.out, p.tokenText(tokens[i])...)
	}
	p.out = append(p.out, '\n')
}

func (p *preprocessor) appendLineMarker(line int, path string) {
	p.out = append(p.out, '#', ' ')
	p.out = ppAppendDecimal(p.out, line)
	p.out = append(p.out, ' ', '"')
	for i := 0; i < len(path); i++ {
		if path[i] == '\\' || path[i] == '"' {
			p.out = append(p.out, '\\')
		}
		p.out = append(p.out, path[i])
	}
	p.out = append(p.out, '"', '\n')
}

func ppAppendDecimal(out []byte, value int) []byte {
	if value == 0 {
		return append(out, '0')
	}
	var digits [24]byte
	i := len(digits)
	for value > 0 {
		i--
		digits[i] = byte('0' + value%10)
		value /= 10
	}
	return append(out, digits[i:]...)
}
