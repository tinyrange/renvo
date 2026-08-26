//go:build !renvo

package rtg

// AssemblyDocument is the source-preserving parse surface for project
// .rtgasm files. Entry bodies deliberately remain byte spans: the selected
// CompilerJIT definition evaluates them later with the ordinary RTG sequence
// language implementation.
type AssemblyDocument struct {
	Filename    string
	Source      []byte
	Version     int
	Entries     []AssemblyEntry
	Diagnostics []Diagnostic
	Ok          bool
}

type AssemblyEntry struct {
	Name       string
	NameSpan   Span
	Parameters []AssemblyParameter
	Steps      []Statement
	BodyStart  int
	BodyEnd    int
	Span       Span
}

type AssemblyParameter struct {
	Name string
	Kind string
}

// ParseAssembly recognizes only the small file wrapper around existing RTG
// sequence bodies:
//
//	rtgasm 1
//	assembly { name(out:emitter) { ... } }
//
// Body syntax is intentionally not interpreted here. Keeping its exact bytes
// lets CompilerJIT evaluate the fragment against the selected definition.
func ParseAssembly(source []byte, filename string) AssemblyDocument {
	document := AssemblyDocument{Filename: filename, Source: source}
	if len(source) > maxDefinitionBytes {
		return assemblyFail(document, sourceSpan(source, 0, 0), "RTGASM-LIMIT-001", "assembly file exceeds the 16 MiB source limit")
	}
	if invalid := invalidUTF8Offset(source); invalid >= 0 {
		return assemblyFail(document, sourceSpan(source, invalid, invalid+1), "RTGASM-SCAN-001", "assembly file is not valid UTF-8")
	}
	tokens, diagnostics := scan(source, filename)
	if len(diagnostics) != 0 {
		document.Diagnostics = diagnostics
		return document
	}
	p := assemblyParser{source: source, tokens: tokens, document: &document}
	p.parse()
	document.Ok = len(document.Diagnostics) == 0
	return document
}

type assemblyParser struct {
	source   []byte
	tokens   []Token
	at       int
	document *AssemblyDocument
}

func (p *assemblyParser) parse() {
	if !p.takeIdent("rtgasm") {
		p.fail("RTGASM-PARSE-001", "expected rtgasm version declaration")
		return
	}
	if p.kind() != TokenNumber || p.text() != "1" {
		p.fail("RTGASM-PARSE-002", "expected supported rtgasm version 1")
		return
	}
	p.document.Version = 1
	p.at++
	if !p.takeIdent("assembly") {
		p.fail("RTGASM-PARSE-003", "expected assembly declaration")
		return
	}
	if !p.takeOperator("{") {
		p.fail("RTGASM-PARSE-004", "expected { after assembly")
		return
	}
	for p.kind() != TokenEOF && !p.operator("}") {
		p.parseEntry()
		if len(p.document.Diagnostics) != 0 {
			return
		}
	}
	if !p.takeOperator("}") {
		p.fail("RTGASM-PARSE-005", "expected } after assembly entries")
		return
	}
	if p.kind() != TokenEOF {
		p.fail("RTGASM-PARSE-006", "unexpected source after assembly declaration")
		return
	}
	if len(p.document.Entries) == 0 {
		p.fail("RTGASM-PARSE-007", "assembly declaration contains no entries")
	}
}

func (p *assemblyParser) parseEntry() {
	start := p.at
	if p.kind() != TokenIdent {
		p.fail("RTGASM-PARSE-008", "expected assembly entry name")
		return
	}
	nameToken := p.tokens[p.at]
	name := p.text()
	for i := 0; i < len(p.document.Entries); i++ {
		if p.document.Entries[i].Name == name {
			p.fail("RTGASM-PARSE-009", "duplicate assembly entry "+name)
			return
		}
	}
	p.at++
	parameters, ok := p.parseParameters()
	if !ok {
		p.fail("RTGASM-PARSE-010", "expected assembly entry parameter list")
		return
	}
	if !p.operator("{") {
		p.fail("RTGASM-PARSE-011", "expected assembly entry body")
		return
	}
	bodyOpen := p.tokens[p.at]
	bodyClose, ok := p.balancedEnd("{", "}")
	if !ok {
		p.fail("RTGASM-PARSE-012", "unterminated assembly entry body")
		return
	}
	parserDocument := Document{Filename: p.document.Filename, Source: p.source, Tokens: p.tokens}
	statementParser := documentParser{document: &parserDocument}
	steps := statementParser.parseStatements(p.at+1, bodyClose)
	if len(parserDocument.Diagnostics) != 0 {
		p.document.Diagnostics = append(p.document.Diagnostics, parserDocument.Diagnostics...)
		return
	}
	p.at = bodyClose + 1
	end := p.tokens[bodyClose]
	p.document.Entries = append(p.document.Entries, AssemblyEntry{
		Name:       name,
		NameSpan:   sourceSpan(p.source, nameToken.Start, nameToken.End),
		Parameters: parameters,
		Steps:      steps,
		BodyStart:  bodyOpen.End,
		BodyEnd:    end.Start,
		Span:       sourceSpan(p.source, p.tokens[start].Start, end.End),
	})
}

func (p *assemblyParser) parseParameters() ([]AssemblyParameter, bool) {
	if !p.takeOperator("(") {
		return nil, false
	}
	var parameters []AssemblyParameter
	for !p.operator(")") {
		if p.kind() != TokenIdent {
			return nil, false
		}
		name := p.text()
		p.at++
		if !p.takeOperator(":") || p.kind() != TokenIdent {
			return nil, false
		}
		kind := p.text()
		if _, ok := sequenceGoType(kind); !ok {
			return nil, false
		}
		parameters = append(parameters, AssemblyParameter{Name: name, Kind: kind})
		p.at++
		if p.operator(",") {
			p.at++
			continue
		}
		if !p.operator(")") {
			return nil, false
		}
	}
	p.at++
	return parameters, len(parameters) == 1 && parameters[0].Name == "out" && parameters[0].Kind == "emitter"
}

func (p *assemblyParser) skipBalanced(open string, close string) bool {
	end, ok := p.balancedEnd(open, close)
	if ok {
		p.at = end + 1
	}
	return ok
}

func (p *assemblyParser) balancedEnd(open string, close string) (int, bool) {
	if !p.operator(open) {
		return p.at, false
	}
	depth := 0
	for i := p.at; i < len(p.tokens); i++ {
		if p.tokens[i].Kind != TokenOperator {
			continue
		}
		text := tokenText(p.source, p.tokens[i])
		if text == open {
			depth++
		} else if text == close {
			depth--
			if depth == 0 {
				return i, true
			}
		}
	}
	return p.at, false
}

func (p *assemblyParser) takeIdent(value string) bool {
	if p.kind() != TokenIdent || p.text() != value {
		return false
	}
	p.at++
	return true
}

func (p *assemblyParser) takeOperator(value string) bool {
	if !p.operator(value) {
		return false
	}
	p.at++
	return true
}

func (p *assemblyParser) operator(value string) bool {
	return p.kind() == TokenOperator && p.text() == value
}

func (p *assemblyParser) kind() int {
	if p.at < 0 || p.at >= len(p.tokens) {
		return TokenEOF
	}
	return p.tokens[p.at].Kind
}

func (p *assemblyParser) text() string {
	if p.at < 0 || p.at >= len(p.tokens) {
		return ""
	}
	return tokenText(p.source, p.tokens[p.at])
}

func (p *assemblyParser) fail(code string, message string) {
	span := sourceSpan(p.source, len(p.source), len(p.source))
	if p.at >= 0 && p.at < len(p.tokens) {
		span = sourceSpan(p.source, p.tokens[p.at].Start, p.tokens[p.at].End)
	}
	p.document.Diagnostics = append(p.document.Diagnostics, Diagnostic{
		Filename: p.document.Filename, Span: span, Code: code, Message: message,
	})
}

func assemblyFail(document AssemblyDocument, span Span, code string, message string) AssemblyDocument {
	document.Diagnostics = append(document.Diagnostics, Diagnostic{
		Filename: document.Filename, Span: span, Code: code, Message: message,
	})
	return document
}

// GenerateAssemblyEvaluator emits a prepared-target source variant whose
// entry point evaluates one project assembly body. The caller compiles this
// source with the ordinary backend kernel for vm/vm32.
func GenerateAssemblyEvaluator(resolved ResolveResult, targetName string, assembly AssemblyDocument, entryIndex int) GenerateResult {
	if !resolved.Ok {
		return GenerateResult{Diagnostics: resolved.Diagnostics}
	}
	if !assembly.Ok || entryIndex < 0 || entryIndex >= len(assembly.Entries) {
		return GenerateResult{Diagnostics: []Diagnostic{{
			Filename: assembly.Filename, Span: sourceSpan(assembly.Source, 0, 0),
			Code: "RTGASM-GENERATE-001", Message: "assembly evaluator entry is invalid",
		}}}
	}
	target, ok := lookupResolvedTarget(resolved, targetName)
	if !ok {
		return GenerateResult{Diagnostics: []Diagnostic{{
			Filename: assembly.Filename, Span: sourceSpan(assembly.Source, 0, 0),
			Code: "RTGASM-GENERATE-002", Message: "assembly evaluator target is unavailable",
		}}}
	}
	entry := assembly.Entries[entryIndex]
	for step := 0; step < len(entry.Steps); step++ {
		child := entry.Steps[step]
		if len(child.Children) != 0 || len(child.Tokens) == 0 ||
			child.Tokens[0] != "call" && child.Tokens[0] != "let" &&
				child.Tokens[0] != "set" && child.Tokens[0] != "var" &&
				child.Tokens[0] != "increment" && child.Tokens[0] != "decrement" &&
				child.Tokens[0] != "return" && !sequenceBareCall(child.Tokens) {
			return GenerateResult{Diagnostics: []Diagnostic{statementDiagnostic(
				Document{Filename: assembly.Filename, Source: assembly.Source}, child,
				"RTGASM-GENERATE-003",
				"assembly body contains an unsupported bounded-sequence step")}}
		}
		if child.Tokens[0] != "return" && len(child.Tokens) < 2 {
			return GenerateResult{Diagnostics: []Diagnostic{statementDiagnostic(
				Document{Filename: assembly.Filename, Source: assembly.Source}, child,
				"RTGASM-GENERATE-004", "assembly body step is incomplete")}}
		}
		if child.Tokens[0] == "return" && (len(child.Tokens) != 1 || step+1 != len(entry.Steps)) {
			return GenerateResult{Diagnostics: []Diagnostic{statementDiagnostic(
				Document{Filename: assembly.Filename, Source: assembly.Source}, child,
				"RTGASM-GENERATE-005", "assembly return must be empty and final")}}
		}
	}
	steps := make([]Statement, len(entry.Steps))
	copy(steps, entry.Steps)
	rewriteVirtualStatements(steps, target.Arch.Package, resolved.Document.packages,
		"", true, []string{"out"})
	var wantedGo []string
	var wantedSequences []string
	goNames := embeddedGoFunctionNames(resolved.Document)
	sequences := architectureSequences(target.Arch)
	for i := 0; i < len(steps); i++ {
		for token := 0; token < len(steps[i].Tokens); token++ {
			name := steps[i].Tokens[token]
			if stringIndex(goNames, name) >= 0 && stringIndex(wantedGo, name) < 0 {
				wantedGo = append(wantedGo, name)
			}
			for sequenceIndex := 0; sequenceIndex < len(sequences); sequenceIndex++ {
				if sequences[sequenceIndex].Name == name && stringIndex(wantedSequences, name) < 0 {
					wantedSequences = append(wantedSequences, name)
				}
			}
		}
	}
	generated := generatePreparedBackendWithRoots(resolved, targetName, wantedGo, wantedSequences)
	if !generated.Ok {
		return generated
	}
	sequence := architectureSequence{
		Name:       "renvoRTGASMEntry",
		Parameters: []embeddedParameter{{Name: "out", Source: []byte("out *RTGEmitter")}},
		Steps:      steps,
	}
	generated.Source = appendAssemblyEvaluatorFunction(generated.Source, resolved.Document, target.Arch, sequence)
	generated.Source = append(generated.Source, `
func appMain(args []string, env []string) int {
	context := renvoNewCompileContext(renvoTargetRTG, true, false, false)
	var out renvoAsm
	renvoAsmInitWithContext(&out, context)
	renvoRTGASMEntry(&out)
	renvoAsmPatch(&out)
	if renvoRTGUnsupportedOperation != 0 || out.patchFailed {
		return 1
	}
	write(1, out.code, -1)
	return 0
}
`...)
	return generated
}

func appendAssemblyEvaluatorFunction(out []byte, document Document, arch Declaration, sequence architectureSequence) []byte {
	names := embeddedGoNames(document)
	prefix := "rtg" + exportedName(document.Unit)
	exports := architectureExports(arch)
	out = append(out, "\nfunc renvoRTGASMEntry(out *renvoAsm) {\n"...)
	for step := 0; step < len(sequence.Steps); step++ {
		tokens := sequence.Steps[step].Tokens
		if len(tokens) == 0 {
			continue
		}
		out = append(out, '\t')
		if tokens[0] == "call" {
			out = appendSequenceTokens(out, tokens[1:], document, names, prefix, true, exports)
		} else if sequenceBareCall(tokens) {
			out = appendSequenceTokens(out, tokens, document, names, prefix, true, exports)
		} else if tokens[0] == "let" {
			out = appendSequenceAssignment(out, tokens[1:], ":=", document, names, prefix, true, exports)
		} else if tokens[0] == "set" {
			out = appendSequenceAssignment(out, tokens[1:], "=", document, names, prefix, true, exports)
		} else if tokens[0] == "var" {
			out = append(out, "var "...)
			out = appendSequenceTokens(out, tokens[1:], document, names, prefix, true, exports)
		} else if tokens[0] == "increment" || tokens[0] == "decrement" {
			out = appendSequenceTokens(out, tokens[1:], document, names, prefix, true, exports)
			if tokens[0] == "increment" {
				out = append(out, "++"...)
			} else {
				out = append(out, "--"...)
			}
		} else if tokens[0] == "return" {
			out = append(out, "return"...)
		}
		out = append(out, '\n')
	}
	out = append(out, "}\n"...)
	return out
}
