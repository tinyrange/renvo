// Package repl assembles successive linked-image generations for the Renvo
// REPL.
package repl

import (
	"renvo.dev/internal/syntax"
	"renvo.dev/std/strconv"
)

const (
	SubmissionInvalid = iota
	SubmissionImport
	SubmissionDeclaration
	SubmissionExpression
	SubmissionStatement
)

type replImport struct {
	source  string
	literal string
	name    string
	always  bool
}

type replBinding struct {
	input       string
	names       []string
	ids         []int
	typ         string
	rhs         string
	initialized bool
}

// State is the source and symbol context for an interactive session. Submitted
// statements are never replayed. Interactive variables are package globals
// backed by stable renvo_repl_value_N slots that the in-process linker migrates
// from one executable image to the next.
type State struct {
	imports      []replImport
	declarations []string
	bindingNames []string
	bindingTypes []string
	bindingRHS   []string
	bindingInit  []bool
	bindingStart []int
	bindingCount []int
	history      []string
	nextSymbol   int
}

// Submission describes one or two compilation attempts. A normal expression
// is tried through fmt.Println first and then as a statement, allowing both
// value expressions and calls with no result to feel natural at the prompt.
type Submission struct {
	Input  string
	Kind   int
	First  []byte
	Second []byte
	Error  string

	imports     []replImport
	binding     *replBinding
	declaration string
}

// Prepare classifies input and assembles complete Go source for its compilation
// attempts. Import declarations are accepted into session state without
// compiling until a later submission uses them.
func (s *State) Prepare(input string) Submission {
	input = CleanInput(input)
	prepared := Submission{Input: input}
	if input == "" {
		return prepared
	}
	tokens := syntax.Scan([]byte(input))
	first := replFirstToken(tokens)
	if first < 0 {
		prepared.Error = "input contains no tokens"
		return prepared
	}
	kind := tokens[first].KindLine & 255
	if kind == syntax.TokenImport {
		imports := replParseImports(input, tokens)
		if len(imports) == 0 {
			prepared.Error = "invalid import declaration"
			return prepared
		}
		prepared.Kind = SubmissionImport
		prepared.imports = imports
		prepared.First = replImportValidationSource(imports)
		return prepared
	}
	if kind == syntax.TokenVar {
		binding, ok := s.newVarBinding(input)
		if !ok {
			prepared.Error = "REPL var declarations must use a single declaration such as `var name [type] [= value]`"
			return prepared
		}
		prepared.Kind = SubmissionStatement
		prepared.binding = &binding
		prepared.First = s.buildSource("", false, false, prepared.binding)
		return prepared
	}
	if kind == syntax.TokenConst || kind == syntax.TokenType || kind == syntax.TokenFunc {
		prepared.Kind = SubmissionDeclaration
		linked := s.linkIdentifiers(input)
		prepared.First = s.buildSource(linked, false, true, nil)
		prepared.declaration = linked
		return prepared
	}
	if binding, ok := s.newShortBinding(input); ok {
		prepared.Kind = SubmissionStatement
		prepared.binding = &binding
		prepared.First = s.buildSource("", false, false, prepared.binding)
		return prepared
	}
	if replLikelyStatement(input, tokens, kind) {
		prepared.Kind = SubmissionStatement
		prepared.First = s.buildSource(s.linkIdentifiers(input), false, false, nil)
		return prepared
	}
	prepared.Kind = SubmissionExpression
	linked := s.linkIdentifiers(input)
	prepared.First = s.buildSource(linked, true, false, nil)
	prepared.Second = s.buildSource(linked, false, false, nil)
	return prepared
}

// Accept records a successfully executed submission. attempt is zero for
// First and one for Second.
func (s *State) Accept(prepared Submission, attempt int) {
	if prepared.Input == "" || prepared.Kind == SubmissionInvalid {
		return
	}
	s.history = append(s.history, prepared.Input)
	if prepared.Kind == SubmissionImport {
		imports := replParseImports(prepared.Input, syntax.Scan([]byte(prepared.Input)))
		s.imports = append(s.imports, imports...)
		return
	}
	if prepared.Kind == SubmissionDeclaration {
		s.declarations = append(s.declarations, s.linkIdentifiers(prepared.Input))
		return
	}
	if prepared.binding != nil {
		// Compilation reclaims its transient arena. Re-parse the accepted
		// binding now so no session state retains slices owned by the prepared
		// compilation attempt.
		binding, ok := s.newBinding(prepared.Input)
		if !ok {
			return
		}
		s.bindingStart = append(s.bindingStart, len(s.bindingNames))
		s.bindingCount = append(s.bindingCount, len(binding.names))
		s.bindingTypes = append(s.bindingTypes, binding.typ)
		s.bindingRHS = append(s.bindingRHS, binding.rhs)
		s.bindingInit = append(s.bindingInit, binding.initialized)
		for i := 0; i < len(binding.names); i++ {
			s.bindingNames = append(s.bindingNames, binding.names[i])
		}
		s.nextSymbol += len(binding.ids)
	}
}

// Reset removes all retained session source.
func (s *State) Reset() {
	s.imports = nil
	s.declarations = nil
	s.bindingNames = nil
	s.bindingTypes = nil
	s.bindingRHS = nil
	s.bindingInit = nil
	s.bindingStart = nil
	s.bindingCount = nil
	s.history = nil
	s.nextSymbol = 0
}

// History returns a copy of the successfully accepted submissions.
func (s *State) History() []string {
	out := make([]string, len(s.history))
	copy(out, s.history)
	return out
}

// Source returns the current linked-generation source for inspection.
func (s *State) Source() string {
	return string(s.buildSource("", false, true, nil))
}

func (s *State) buildSource(input string, expression bool, declaration bool, candidate *replBinding) []byte {
	out, _ := s.buildSourcePosition(input, expression, declaration, candidate, false)
	return out
}

func (s *State) buildSourcePosition(input string, expression bool, declaration bool, candidate *replBinding, completionAliases bool) ([]byte, int) {
	var packageBody []byte
	packageInput := -1
	for i := 0; i < len(s.declarations); i++ {
		packageBody = append(packageBody, s.declarations[i]...)
		packageBody = append(packageBody, '\n')
	}
	if declaration {
		packageInput = len(packageBody)
		packageBody = append(packageBody, input...)
		packageBody = append(packageBody, '\n')
	}
	bindingCount := len(s.bindingRHS)
	if candidate != nil {
		bindingCount++
	}
	for bindingIndex := 0; bindingIndex < bindingCount; bindingIndex++ {
		var names []string
		firstID := 0
		typ := ""
		rhs := ""
		initialized := false
		if bindingIndex < len(s.bindingRHS) {
			start := s.bindingStart[bindingIndex]
			end := start + s.bindingCount[bindingIndex]
			names = s.bindingNames[start:end]
			firstID = start
			typ = s.bindingTypes[bindingIndex]
			rhs = s.bindingRHS[bindingIndex]
			initialized = s.bindingInit[bindingIndex]
		} else {
			names = candidate.names
			firstID = candidate.ids[0]
			typ = candidate.typ
			rhs = candidate.rhs
			initialized = candidate.initialized
		}
		packageBody = append(packageBody, "var "...)
		for i := 0; i < len(names); i++ {
			if i > 0 {
				packageBody = append(packageBody, ',', ' ')
			}
			packageBody = append(packageBody, "renvo_repl_storage_"...)
			packageBody = append(packageBody, strconv.Itoa(firstID+i)...)
		}
		if typ != "" {
			packageBody = append(packageBody, ' ')
			packageBody = append(packageBody, typ...)
		}
		if initialized {
			packageBody = append(packageBody, " = "...)
			packageBody = append(packageBody, rhs...)
		}
		packageBody = append(packageBody, '\n')
		for i := 0; i < len(names); i++ {
			if names[i] == "_" {
				continue
			}
			packageBody = append(packageBody, "var renvo_repl_value_"...)
			packageBody = append(packageBody, strconv.Itoa(firstID+i)...)
			packageBody = append(packageBody, " = &renvo_repl_storage_"...)
			packageBody = append(packageBody, strconv.Itoa(firstID+i)...)
			packageBody = append(packageBody, '\n')
		}
	}
	if completionAliases {
		for i := 0; i < len(s.bindingNames); i++ {
			if s.bindingNames[i] == "_" {
				continue
			}
			packageBody = append(packageBody, "var "...)
			packageBody = append(packageBody, s.bindingNames[i]...)
			packageBody = append(packageBody, " = *renvo_repl_value_"...)
			packageBody = append(packageBody, strconv.Itoa(i)...)
			packageBody = append(packageBody, '\n')
		}
	}
	var mainBody []byte
	mainInput := -1
	if !declaration && candidate == nil {
		if expression {
			mainBody = append(mainBody, "renvoreplfmt.Println("...)
			mainInput = len(mainBody)
			mainBody = append(mainBody, input...)
			mainBody = append(mainBody, ')', '\n')
		} else {
			mainInput = len(mainBody)
			mainBody = append(mainBody, input...)
			mainBody = append(mainBody, '\n')
		}
	}
	uses := make([]byte, 0, len(packageBody)+len(mainBody))
	uses = append(uses, packageBody...)
	uses = append(uses, mainBody...)

	var out []byte
	out = append(out, "package main\n"...)
	if expression {
		out = append(out, "import renvoreplfmt \"fmt\"\n"...)
	}
	for i := 0; i < len(s.imports); i++ {
		item := s.imports[i]
		if item.always || replUsesIdentifier(uses, item.name) {
			out = append(out, item.source...)
			out = append(out, '\n')
		}
	}
	inputPosition := -1
	if packageInput >= 0 {
		inputPosition = len(out) + packageInput
	}
	out = append(out, packageBody...)
	out = append(out, "func main() {\n"...)
	if mainInput >= 0 {
		inputPosition = len(out) + mainInput
	}
	out = append(out, mainBody...)
	out = append(out, "}\n"...)
	return out, inputPosition
}

func (s *State) newShortBinding(input string) (replBinding, bool) {
	names, rhs := replShortDeclaration(input)
	if len(names) == 0 || rhs == "" {
		return replBinding{}, false
	}
	binding := replBinding{input: input, names: names, rhs: s.linkIdentifiers(rhs), initialized: true}
	return s.assignBindingIDs(binding)
}

func (s *State) newVarBinding(input string) (replBinding, bool) {
	names, typ, rhs, initialized := replVarDeclaration(input)
	if len(names) == 0 || typ == "" && !initialized {
		return replBinding{}, false
	}
	binding := replBinding{
		input:       input,
		names:       names,
		typ:         typ,
		rhs:         s.linkIdentifiers(rhs),
		initialized: initialized,
	}
	return s.assignBindingIDs(binding)
}

func (s *State) newBinding(input string) (replBinding, bool) {
	tokens := syntax.Scan([]byte(input))
	first := replFirstToken(tokens)
	if first >= 0 && tokens[first].KindLine&255 == syntax.TokenVar {
		return s.newVarBinding(input)
	}
	return s.newShortBinding(input)
}

func (s *State) assignBindingIDs(binding replBinding) (replBinding, bool) {
	names := binding.names
	for i := 0; i < len(names); i++ {
		if names[i] != "_" && s.hasBinding(names[i]) {
			return replBinding{}, false
		}
		binding.ids = append(binding.ids, s.nextSymbol+i)
	}
	return binding, true
}

func (s *State) linkIdentifiers(input string) string {
	src := []byte(input)
	tokens := syntax.Scan(src)
	shadowed := s.shadowedBindings(src, tokens)
	var out []byte
	at := 0
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		if token.KindLine&255 == syntax.TokenEOF || syntax.TokenStart(token) < at {
			continue
		}
		id := -1
		if token.KindLine&255 == syntax.TokenIdent &&
			(i == 0 || string(syntax.TokenText(src, tokens[i-1])) != ".") {
			name := string(syntax.TokenText(src, token))
			if !replStringContains(shadowed, name) {
				id = s.bindingID(name)
			}
		}
		if id < 0 {
			continue
		}
		out = append(out, src[at:syntax.TokenStart(token)]...)
		out = append(out, "(*renvo_repl_value_"...)
		out = append(out, strconv.Itoa(id)...)
		out = append(out, ')')
		at = syntax.TokenEnd(token)
	}
	if at == 0 {
		return input
	}
	out = append(out, src[at:]...)
	return string(out)
}

func (s *State) shadowedBindings(src []byte, tokens []syntax.Token) []string {
	var shadowed []string
	for i := 0; i < len(tokens); i++ {
		if tokens[i].KindLine&255 != syntax.TokenFunc {
			continue
		}
		for j := i + 1; j < len(tokens); j++ {
			text := string(syntax.TokenText(src, tokens[j]))
			if text == "{" {
				break
			}
			if tokens[j].KindLine&255 == syntax.TokenIdent {
				name := string(syntax.TokenText(src, tokens[j]))
				if s.hasBinding(name) && !replStringContains(shadowed, name) {
					shadowed = append(shadowed, name)
				}
			}
		}
	}
	for i := 0; i < len(tokens); i++ {
		if string(syntax.TokenText(src, tokens[i])) != ":=" {
			continue
		}
		for j := i - 1; j >= 0; j-- {
			text := string(syntax.TokenText(src, tokens[j]))
			if text == "{" || text == "}" || text == ";" {
				break
			}
			if tokens[j].KindLine&255 == syntax.TokenIdent {
				name := string(syntax.TokenText(src, tokens[j]))
				if s.hasBinding(name) && !replStringContains(shadowed, name) {
					shadowed = append(shadowed, name)
				}
			} else if text != "," {
				break
			}
		}
	}
	return shadowed
}

func replStringContains(values []string, value string) bool {
	for i := 0; i < len(values); i++ {
		if values[i] == value {
			return true
		}
	}
	return false
}

func (s *State) bindingID(name string) int {
	for i := 0; i < len(s.bindingNames); i++ {
		if s.bindingNames[i] == name {
			return i
		}
	}
	return -1
}

func (s *State) hasBinding(name string) bool {
	for i := 0; i < len(s.bindingNames); i++ {
		if s.bindingNames[i] == name {
			return true
		}
	}
	return false
}

func replShortDeclaration(input string) ([]string, string) {
	src := []byte(input)
	tokens := syntax.Scan(src)
	assignment := -1
	for i := 0; i < len(tokens); i++ {
		if tokens[i].KindLine&255 == syntax.TokenOperator &&
			string(syntax.TokenText(src, tokens[i])) == ":=" {
			assignment = i
			break
		}
	}
	if assignment < 0 {
		return nil, ""
	}
	names := replShortDeclarationNames(input)
	if len(names) == 0 {
		return nil, ""
	}
	at := syntax.TokenEnd(tokens[assignment])
	for at < len(src) && replSpace(src[at]) {
		at++
	}
	return names, string(src[at:])
}

func replVarDeclaration(input string) ([]string, string, string, bool) {
	src := []byte(input)
	tokens := syntax.Scan(src)
	first := replFirstToken(tokens)
	if first < 0 || tokens[first].KindLine&255 != syntax.TokenVar || first+1 >= len(tokens) {
		return nil, "", "", false
	}
	if string(syntax.TokenText(src, tokens[first+1])) == "(" {
		return nil, "", "", false
	}
	assign := -1
	end := len(tokens)
	for i := first + 1; i < len(tokens); i++ {
		if tokens[i].KindLine&255 == syntax.TokenEOF {
			end = i
			break
		}
		if string(syntax.TokenText(src, tokens[i])) == "=" {
			assign = i
			end = i
			break
		}
	}
	var names []string
	i := first + 1
	for i < end {
		if tokens[i].KindLine&255 != syntax.TokenIdent {
			break
		}
		names = append(names, string(syntax.TokenText(src, tokens[i])))
		i++
		if i >= end || string(syntax.TokenText(src, tokens[i])) != "," {
			break
		}
		i++
	}
	if len(names) == 0 {
		return nil, "", "", false
	}
	typ := ""
	if i < end {
		typ = CleanInput(string(src[syntax.TokenStart(tokens[i]):syntax.TokenEnd(tokens[end-1])]))
	}
	initialized := assign >= 0
	rhs := ""
	if initialized {
		at := syntax.TokenEnd(tokens[assign])
		for at < len(src) && replSpace(src[at]) {
			at++
		}
		rhs = CleanInput(string(src[at:]))
		if rhs == "" {
			return nil, "", "", false
		}
	}
	if typ == "" && !initialized {
		return nil, "", "", false
	}
	return names, typ, rhs, initialized
}

func replShortDeclarationNames(input string) []string {
	src := []byte(input)
	tokens := syntax.Scan(src)
	assignment := -1
	for i := 0; i < len(tokens); i++ {
		if tokens[i].KindLine&255 == syntax.TokenOperator &&
			string(syntax.TokenText(src, tokens[i])) == ":=" {
			assignment = i
			break
		}
	}
	if assignment < 0 {
		return nil
	}
	var out []string
	for i := 0; i < assignment; i++ {
		if tokens[i].KindLine&255 == syntax.TokenIdent {
			out = append(out, string(syntax.TokenText(src, tokens[i])))
		} else if tokens[i].KindLine&255 != syntax.TokenOperator ||
			string(syntax.TokenText(src, tokens[i])) != "," {
			return nil
		}
	}
	return out
}

func replFirstToken(tokens []syntax.Token) int {
	for i := 0; i < len(tokens); i++ {
		if tokens[i].KindLine&255 != syntax.TokenEOF {
			return i
		}
	}
	return -1
}

func replParseImports(input string, tokens []syntax.Token) []replImport {
	src := []byte(input)
	var out []replImport
	for i := 0; i < len(tokens); i++ {
		if tokens[i].KindLine&255 != syntax.TokenString {
			continue
		}
		literal := string(syntax.TokenText(src, tokens[i]))
		path, err := strconv.Unquote(literal)
		if err != nil || path == "" {
			return nil
		}
		alias := ""
		explicit := false
		if i > 0 {
			previous := tokens[i-1]
			previousKind := previous.KindLine & 255
			previousText := string(syntax.TokenText(src, previous))
			if previousKind == syntax.TokenIdent {
				alias = previousText
				explicit = true
			} else if previousKind == syntax.TokenOperator && previousText == "." {
				alias = "."
				explicit = true
			}
		}
		name := alias
		if !explicit {
			name = replImportBase(path)
		}
		var source []byte
		source = append(source, "import "...)
		if explicit {
			source = append(source, alias...)
			source = append(source, ' ')
		}
		source = append(source, literal...)
		out = append(out, replImport{
			source:  string(source),
			literal: literal,
			name:    name,
			always:  alias == "_" || alias == ".",
		})
	}
	return out
}

func replImportValidationSource(imports []replImport) []byte {
	var out []byte
	out = append(out, "package main\n"...)
	for i := 0; i < len(imports); i++ {
		out = append(out, "import _ "...)
		out = append(out, imports[i].literal...)
		out = append(out, '\n')
	}
	out = append(out, "func main() {}\n"...)
	return out
}

func replImportBase(path string) string {
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			start = i + 1
		}
	}
	return path[start:]
}

func replUsesIdentifier(src []byte, name string) bool {
	if name == "" {
		return false
	}
	tokens := syntax.Scan(src)
	for i := 0; i < len(tokens); i++ {
		if tokens[i].KindLine&255 == syntax.TokenIdent &&
			string(syntax.TokenText(src, tokens[i])) == name {
			return true
		}
	}
	return false
}

func replHasAssignment(input string, tokens []syntax.Token) bool {
	src := []byte(input)
	for i := 0; i < len(tokens); i++ {
		if tokens[i].KindLine&255 != syntax.TokenOperator {
			continue
		}
		text := string(syntax.TokenText(src, tokens[i]))
		if text == "=" || text == ":=" || text == "+=" || text == "-=" ||
			text == "*=" || text == "/=" || text == "%=" || text == "&=" ||
			text == "|=" || text == "^=" || text == "<<=" || text == ">>=" ||
			text == "&^=" || text == "++" || text == "--" {
			return true
		}
	}
	return false
}

func replPersistentAssignment(input string, tokens []syntax.Token) bool {
	src := []byte(input)
	seenName := false
	for i := 0; i < len(tokens); i++ {
		kind := tokens[i].KindLine & 255
		if kind == syntax.TokenEOF {
			return false
		}
		if kind == syntax.TokenIdent {
			seenName = true
			continue
		}
		if kind != syntax.TokenOperator {
			return false
		}
		text := string(syntax.TokenText(src, tokens[i]))
		if text == "," {
			continue
		}
		return seenName && (text == "=" || text == ":=" || text == "+=" || text == "-=" || text == "*=" || text == "/=" || text == "%=" || text == "&=" || text == "|=" || text == "^=" || text == "<<=" || text == ">>=" || text == "&^=" || text == "++" || text == "--")
	}
	return false
}

func replLikelyStatement(input string, tokens []syntax.Token, firstKind int) bool {
	if replHasAssignment(input, tokens) {
		return true
	}
	src := []byte(input)
	for i := 0; i < len(tokens); i++ {
		if tokens[i].KindLine&255 != syntax.TokenIdent {
			continue
		}
		name := string(syntax.TokenText(src, tokens[i]))
		known := name == "print" || name == "println" || name == "panic" || name == "Print" || name == "Printf" || name == "Println" || name == "Fprint" || name == "Fprintf" || name == "Fprintln"
		if known {
			if i+1 < len(tokens) && string(syntax.TokenText(src, tokens[i+1])) == "(" {
				return true
			}
		}
	}
	return firstKind == syntax.TokenReturn || firstKind == syntax.TokenIf || firstKind == syntax.TokenFor || firstKind == syntax.TokenSwitch || firstKind == syntax.TokenDefer || firstKind == syntax.TokenGo || firstKind == syntax.TokenSelect || firstKind == syntax.TokenGoto || firstKind == syntax.TokenBreak || firstKind == syntax.TokenContinue || firstKind == syntax.TokenFallthrough
}

// CleanInput removes surrounding ASCII whitespace while preserving all source
// inside the submission.
func CleanInput(input string) string {
	start := 0
	for start < len(input) && replSpace(input[start]) {
		start++
	}
	end := len(input)
	for end > start && replSpace(input[end-1]) {
		end--
	}
	return input[start:end]
}

func replSpace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n'
}

// InputComplete reports whether delimiters, raw strings, and block comments
// are balanced. Ordinary quoted strings terminate at a newline and are left to
// the compiler diagnostic rather than trapping the prompt in continuation
// mode.
func InputComplete(input string) bool {
	paren, bracket, brace := 0, 0, 0
	quote := byte(0)
	escaped := false
	lineComment := false
	blockComment := false
	for i := 0; i < len(input); i++ {
		ch := input[i]
		next := byte(0)
		if i+1 < len(input) {
			next = input[i+1]
		}
		if lineComment {
			if ch == '\n' {
				lineComment = false
			}
			continue
		}
		if blockComment {
			if ch == '*' && next == '/' {
				blockComment = false
				i++
			}
			continue
		}
		if quote != 0 {
			if quote == '`' {
				if ch == '`' {
					quote = 0
				}
				continue
			}
			if ch == '\n' {
				quote = 0
				escaped = false
				continue
			}
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
			} else if ch == quote {
				quote = 0
			}
			continue
		}
		if ch == '/' && next == '/' {
			lineComment = true
			i++
			continue
		}
		if ch == '/' && next == '*' {
			blockComment = true
			i++
			continue
		}
		if ch == '"' || ch == '\'' || ch == '`' {
			quote = ch
			continue
		}
		if ch == '(' {
			paren++
		} else if ch == ')' {
			paren--
		} else if ch == '[' {
			bracket++
		} else if ch == ']' {
			bracket--
		} else if ch == '{' {
			brace++
		} else if ch == '}' {
			brace--
		}
	}
	return paren <= 0 && bracket <= 0 && brace <= 0 && quote != '`' && !blockComment
}
