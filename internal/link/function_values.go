package link

import (
	"renvo.dev/internal/arena"
	"renvo.dev/internal/syntax"
	"renvo.dev/internal/unit"
)

// Function values are lowered after package linking, when every possible
// implementation is visible. The backend subset receives ordinary structs and
// direct calls: a zero tag is nil, and each non-zero tag selects a generated
// direct-call arm. A fixed-size descriptor holds the tag and an environment
// interface containing a concrete environment pointer. Dispatch restores that
// pointer's type before calling; no source-level unsafe conversion is needed.

type functionValueSignature struct {
	name          string
	params        string
	paramNames    []string
	paramTypes    []string
	result        string
	resultTypes   []string
	zeroType      string
	storage       *functionValueStorage
	anonymous     bool
	declFuncTok   int
	declEndTok    int
	sourceFuncTok int
	sourceEndTok  int
}

// Equal signatures share their representation and tag numbering, while each
// defined function type retains its own declaration and interface identity.
type functionValueStorage struct {
	impls []functionValueImpl
}

type functionValueImpl struct {
	receiverType    string
	environmentType string
	method          string
	function        string
	nilReceiver     bool
}

type functionValueClosure struct {
	envName  string
	funcName string
	fields   []string
	types    []string
	params   string
	result   string
	body     string
}

type functionValueField struct {
	owner string
	name  string
	sig   int
}

type functionValueEdit struct {
	start int
	end   int
	text  string
}

func renvo_runtime_ArenaDiscardLinkTokens(tokens []unit.Token) {}

func lowerFunctionValuesCore(program *unit.Program, transient bool) bool {
	if !lowerDiscardedRecoverCalls(program, transient) {
		return false
	}
	functions, deferred, builtins := functionValueProgramNeedsLowering(program)
	if deferred {
		if !lowerDeferredBuiltins(program, transient) {
			return false
		}
		functions = true
	}
	if builtins {
		if !lowerOrdinaryBuiltins(program, transient) {
			return false
		}
		functions = true
	}
	if !functions {
		return true
	}
	// Alias normalization reparses the complete unit. Ordinary native function
	// signatures need no tagged storage, so avoid that allocation when discovery
	// finds no values to lower. Discard speculative records before normalizing.
	discoveryMark := arena.Mark()
	signatures, fields, edits, ok := discoverFunctionValueTypes(program)
	if !ok {
		return false
	}
	if len(signatures) == 0 {
		arena.Rewind(discoveryMark)
		return true
	}
	arena.Rewind(discoveryMark)
	if !lowerFunctionSignatureAliases(program, transient) {
		return false
	}
	signatures, fields, edits, ok = discoverFunctionValueTypes(program)
	if !ok {
		return false
	}
	edits = appendFunctionValueTypeEdits(program, signatures, edits)
	edits = appendFunctionValuePackageEdits(program, edits)
	edits = lowerFunctionValueInferredDeclarations(program, signatures, edits)
	for i := 0; i < len(program.Tokens); i++ {
		text := functionValueTokenText(program, i)
		mark := arena.Mark()
		before := len(edits)
		if text == "=" {
			edits, ok = lowerFunctionValueAssignment(program, i, signatures, fields, edits)
			if !ok {
				return false
			}
		} else if text == ":" {
			edits = lowerFunctionValueCompositeField(program, i, signatures, fields, edits)
		}
		if len(edits) == before {
			arena.Rewind(mark)
		}
	}
	edits = lowerFunctionValueArrayComposites(program, signatures, edits)
	edits = lowerFunctionValueReturns(program, signatures, edits)
	edits = lowerFunctionValueCallArguments(program, signatures, edits)
	for i := 0; i < len(program.Tokens); i++ {
		text := functionValueTokenText(program, i)
		if text == "==" || text == "!=" {
			mark := arena.Mark()
			before := len(edits)
			edits = lowerFunctionValueComparison(program, i, signatures, fields, edits)
			if len(edits) == before {
				arena.Rewind(mark)
			}
		}
	}
	// Outer chained calls share a source start with their inner call. Add their
	// prefixes first so stable edit ordering preserves make()() as call(call(make)).
	for i := len(program.Tokens) - 1; i >= 0; i-- {
		if functionValueTokenEquals(program, i, "(") {
			mark := arena.Mark()
			before := len(edits)
			edits = lowerFunctionValueCall(program, i, signatures, fields, edits)
			if len(edits) == before {
				arena.Rewind(mark)
			}
		}
	}
	var closures []functionValueClosure
	signatures, closures, edits, ok = lowerFunctionValueLiterals(program, signatures, fields, closures, edits)
	if !ok {
		return false
	}
	for i := 0; i < len(signatures); i++ {
		sig := &signatures[i]
		if sig.declFuncTok >= 0 {
			replacement := functionValueStructText(*sig)
			if functionValueTokenEquals(program, sig.declFuncTok-1, "=") {
				anonymous := functionValueSignatureByShape(signatures, *sig)
				if anonymous < 0 {
					return false
				}
				replacement = signatures[anonymous].name
			}
			edits = append(edits, functionValueTokenRangeEdit(program, sig.declFuncTok, sig.declEndTok, replacement))
		}
	}
	generated := functionValueGeneratedText(signatures, closures)
	generated, ok = functionValueGeneratedTypes(generated, signatures)
	if !ok {
		return false
	}
	originalLength := len(program.Text)
	if transient {
		renvo_runtime_ArenaDiscardLinkTokens(program.Tokens)
	}
	text, ok := applyFunctionValueEditsCapacity(program.Text, edits, len(generated)+1)
	if !ok {
		return false
	}
	if len(text) > 0 && text[len(text)-1] != '\n' {
		text = append(text, '\n')
	}
	generatedStart := len(text)
	text = appendFunctionValueString(text, generated)
	if transient {
		arena.DiscardBytes(program.Text)
	}
	if !reparseFunctionValueProgramMode(program, text, edits, originalLength, generatedStart, transient) {
		return false
	}
	return lowerFunctionValueDefers(program, signatures, transient)
}

// Relocatable objects expose function values through the platform C ABI, where
// they are raw code pointers. Keep those values intact for the object backend;
// only the syntax-level builtin rewrites shared with ordinary linking apply.
func lowerObjectFunctionValuesCore(program *unit.Program, transient bool) bool {
	if !lowerDiscardedRecoverCalls(program, transient) {
		return false
	}
	_, deferred, builtins := functionValueProgramNeedsLowering(program)
	if deferred && !lowerDeferredBuiltins(program, transient) {
		return false
	}
	if builtins && !lowerOrdinaryBuiltins(program, transient) {
		return false
	}
	return true
}

func lowerDeferredBuiltins(program *unit.Program, transient bool) bool {
	var edits []functionValueEdit
	for i := 0; i+2 < len(program.Tokens); i++ {
		if !functionValueTokenEquals(program, i, "defer") || !functionValueTokenEquals(program, i+2, "(") {
			continue
		}
		name := functionValueTokenText(program, i+1)
		if name != "copy" && name != "delete" && name != "panic" && name != "print" && name != "println" && name != "recover" {
			continue
		}
		if functionValueEnclosingLocalType(program, i, name) != "" || functionValueDeclaredFunction(program, name) {
			continue
		}
		close := functionValueFindMatchingParen(program, i+2)
		if close < 0 {
			return false
		}
		var starts []int
		var ends []int
		start := i + 3
		depth := 0
		for tok := start; tok <= close; tok++ {
			text := functionValueTokenText(program, tok)
			if tok < close && (text == "(" || text == "[" || text == "{") {
				depth++
			} else if tok < close && (text == ")" || text == "]" || text == "}") {
				depth--
			}
			if tok != close && !(depth == 0 && text == ",") {
				continue
			}
			if start < tok {
				starts = append(starts, start)
				ends = append(ends, tok)
			}
			start = tok + 1
		}
		if name == "recover" {
			edits = append(edits, functionValueTokenRangeEdit(program, i, close+1, "defer func(){}()"))
			i = close
			continue
		}
		params := ""
		args := ""
		bodyArgs := ""
		for arg := 0; arg < len(starts); arg++ {
			typ := deferredBuiltinArgumentType(program, i, starts[arg], ends[arg])
			if typ == "" {
				return false
			}
			if arg > 0 {
				params += ", "
				args += ", "
				bodyArgs += ", "
			}
			argName := "__renvo_defer_" + functionValueDecimal(i) + "_" + functionValueDecimal(arg)
			params += argName + " " + typ
			args += functionValueTokensText(program, starts[arg], ends[arg])
			bodyArgs += argName
		}
		replacement := "defer func(" + params + "){" + name + "(" + bodyArgs + ")}(" + args + ")"
		edits = append(edits, functionValueTokenRangeEdit(program, i, close+1, replacement))
		i = close
	}
	if len(edits) == 0 {
		return true
	}
	originalLength := len(program.Text)
	if transient {
		renvo_runtime_ArenaDiscardLinkTokens(program.Tokens)
	}
	text, ok := applyFunctionValueEdits(program.Text, edits)
	if !ok {
		return false
	}
	if transient {
		arena.DiscardBytes(program.Text)
	}
	return ok && reparseFunctionValueProgram(program, text, edits, originalLength, -1)
}

func deferredBuiltinArgumentType(program *unit.Program, before int, start int, end int) string {
	return ordinaryBuiltinExprType(program, before, start, end)
}

func functionValueDeclaredFunction(program *unit.Program, name string) bool {
	for i := 0; i < len(program.Funcs); i++ {
		if functionValueTokenEquals(program, program.Funcs[i].NameTok, name) {
			return true
		}
	}
	return false
}

func functionValueProgramNeedsLowering(program *unit.Program) (bool, bool, bool) {
	functions := false
	deferred := false
	builtins := false
	for i := 0; i+1 < len(program.Tokens); i++ {
		token := program.Tokens[i]
		kind := token.KindLine & 255
		// Defer remains an identifier in the compact unit. Other token kinds
		// cannot introduce a function value or one of the builtin calls below.
		if kind != unit.TokenFunc && kind != unit.TokenIdent {
			continue
		}
		start := token.Start
		valid := start >= 0 && start+token.Size <= len(program.Text)
		if !functions && token.KindLine&255 == unit.TokenFunc && functionValueTokenEquals(program, i+1, "(") && !functionValueIsDeclaredFunction(program, i) {
			functions = true
		}
		if !deferred && i+2 < len(program.Tokens) && valid && token.Size == 5 && program.Text[start] == 'd' && program.Text[start+1] == 'e' && program.Text[start+2] == 'f' && program.Text[start+3] == 'e' && program.Text[start+4] == 'r' && functionValueTokenEquals(program, i+2, "(") {
			mark := arena.Mark()
			name := functionValueTokenText(program, i+1)
			if (name == "copy" || name == "delete" || name == "panic" || name == "print" || name == "println" || name == "recover") && functionValueEnclosingLocalType(program, i, name) == "" && !functionValueDeclaredFunction(program, name) {
				deferred = true
			}
			arena.Rewind(mark)
		}
		name := ""
		if !builtins && valid && token.KindLine&255 == unit.TokenIdent {
			if token.Size == 3 && program.Text[start] == 'm' {
				if program.Text[start+1] == 'i' && program.Text[start+2] == 'n' {
					name = "min"
				} else if program.Text[start+1] == 'a' && program.Text[start+2] == 'x' {
					name = "max"
				}
			} else if token.Size == 5 && program.Text[start] == 'c' && program.Text[start+1] == 'l' && program.Text[start+2] == 'e' && program.Text[start+3] == 'a' && program.Text[start+4] == 'r' {
				name = "clear"
			} else if token.Size == 6 && functionValueTokenEquals(program, i, "string") && functionValueTokenEquals(program, i+1, "(") {
				mark := arena.Mark()
				close := functionValueFindMatchingParen(program, i+1)
				if close > i+2 && ordinaryIntegerExpression(program, i, i+2, close) {
					name = "string"
				}
				arena.Rewind(mark)
			}
		}
		if name != "" && functionValueTokenEquals(program, i+1, "(") {
			mark := arena.Mark()
			if !ordinaryBuiltinShadowed(program, i, name) {
				builtins = true
			}
			arena.Rewind(mark)
		}
	}
	// Inferred global function values must keep the same representation when
	// another package later introduces callbacks or ordinary builtin lowering.
	// This also covers globals initialized by lifted function literals.
	if !functions {
		for i := 0; i < len(program.Decls); i++ {
			if functionValueGlobalInitializerFunction(program, program.Decls[i]) >= 0 {
				functions = true
				break
			}
		}
	}
	return functions, deferred, builtins
}

func discoverFunctionValueTypes(program *unit.Program) ([]functionValueSignature, []functionValueField, []functionValueEdit, bool) {
	var signatures []functionValueSignature
	var fields []functionValueField
	var edits []functionValueEdit
	for i := 0; i < len(program.Decls); i++ {
		decl := program.Decls[i]
		if decl.Kind != unit.TokenType {
			continue
		}
		nameTok := functionValueTokenAtSpan(program, decl.NameStart, decl.NameEnd)
		if nameTok < 0 {
			return signatures, fields, edits, false
		}
		owner := functionValueTokenText(program, nameTok)
		start := nameTok + 1
		if functionValueTokenEquals(program, start, "=") {
			start++
		}
		if functionValueTokenEquals(program, start, "func") {
			sig, end, ok := parseFunctionValueSignature(program, start, functionValueTokenText(program, nameTok))
			if !ok {
				return signatures, fields, edits, false
			}
			sig.declFuncTok = start
			sig.declEndTok = end
			signatures = append(signatures, sig)
			continue
		}
		if !functionValueTokenEquals(program, start, "struct") || !functionValueTokenEquals(program, start+1, "{") {
			continue
		}
		close := functionValueFindMatchingBrace(program, start+1)
		if close < 0 {
			return signatures, fields, edits, false
		}
		j := start + 2
		for j < close {
			if program.Tokens[j].KindLine&255 != unit.TokenIdent {
				j++
				continue
			}
			fieldName := functionValueTokenText(program, j)
			typeTok := j + 1
			sigIndex := functionValueSignatureByName(signatures, functionValueTokenText(program, typeTok))
			if sigIndex >= 0 {
				fields = append(fields, functionValueField{owner: owner, name: fieldName, sig: sigIndex})
				j = typeTok + 1
				continue
			}
			if functionValueTokenEquals(program, typeTok, "func") {
				sig, end, ok := parseFunctionValueSignature(program, typeTok, "")
				if !ok {
					return signatures, fields, edits, false
				}
				// Anonymous function fields with the same signature are assignment-
				// compatible. Share one tagged representation so accessors returning
				// the anonymous type dispatch every implementation of that signature.
				sigIndex = functionValueSignatureByShape(signatures, sig)
				if sigIndex < 0 || signatures[sigIndex].declFuncTok >= 0 {
					sig.name = "__renvo_function_" + functionValueDecimal(len(signatures))
					sig.declFuncTok = -1
					sig.declEndTok = -1
					sig.anonymous = true
					sigIndex = len(signatures)
					signatures = append(signatures, sig)
				}
				fields = append(fields, functionValueField{owner: owner, name: fieldName, sig: sigIndex})
				edits = append(edits, functionValueTokenRangeEdit(program, typeTok, end, signatures[sigIndex].name))
				j = end
				continue
			}
			j++
		}
	}
	// A named function type can be declared after the struct that uses it.
	for i := 0; i < len(program.Decls); i++ {
		decl := program.Decls[i]
		if decl.Kind != unit.TokenType {
			continue
		}
		nameTok := functionValueTokenAtSpan(program, decl.NameStart, decl.NameEnd)
		owner := functionValueTokenText(program, nameTok)
		start := nameTok + 1
		if !functionValueTokenEquals(program, start, "struct") || !functionValueTokenEquals(program, start+1, "{") {
			continue
		}
		close := functionValueFindMatchingBrace(program, start+1)
		for j := start + 2; j+1 < close; j++ {
			if program.Tokens[j].KindLine&255 != unit.TokenIdent {
				continue
			}
			sigIndex := functionValueSignatureByName(signatures, functionValueTokenText(program, j+1))
			if sigIndex >= 0 && functionValueFieldByOwnerAndName(fields, owner, functionValueTokenText(program, j)) < 0 {
				fields = append(fields, functionValueField{owner: owner, name: functionValueTokenText(program, j), sig: sigIndex})
			}
		}
	}
	// Inferred global function variables use the same representation as
	// anonymous callback fields, including globals initialized by lifted literals.
	for i := 0; i < len(program.Decls); i++ {
		fnIndex := functionValueGlobalInitializerFunction(program, program.Decls[i])
		if fnIndex < 0 {
			continue
		}
		candidate, _, valid := parseFunctionValueCallableSignature(program, program.Funcs[fnIndex].NameTok, "")
		if !valid || functionValueSignatureByShape(signatures, candidate) >= 0 {
			continue
		}
		candidate.name = "__renvo_function_" + functionValueDecimal(len(signatures))
		candidate.declFuncTok = -1
		candidate.declEndTok = -1
		candidate.anonymous = true
		signatures = append(signatures, candidate)
	}
	// Discover inferred closures, C pointer conversions/accessors, and anonymous
	// types whose signature also has a defined function declaration. The latter
	// need a separate type identity, with compatible storage for assignments.
	// Other ordinary function declarations retain their existing representation.
	for funcTok := 0; funcTok+1 < len(program.Tokens); funcTok++ {
		if !functionValueTokenEquals(program, funcTok, "func") || !functionValueTokenEquals(program, funcTok+1, "(") {
			continue
		}
		candidate, end, valid := parseFunctionValueSignature(program, funcTok, "")
		conversion := functionValueTokenEquals(program, funcTok-1, "(") &&
			functionValueTokenEquals(program, end, ")") && functionValueTokenEquals(program, end+1, "(")
		accessorResult := functionValueTokenEquals(program, funcTok-1, "*") && functionValueTokenInDeclaredResult(program, funcTok)
		inferredLocal := (functionValueTokenEquals(program, funcTok-1, ":=") || functionValueTokenEquals(program, funcTok-1, "=")) && functionValueTokenEquals(program, end, "{")
		nestedSignature := false
		for i := 0; i < len(signatures); i++ {
			if signatures[i].sourceFuncTok < funcTok && funcTok < signatures[i].sourceEndTok {
				nestedSignature = true
				break
			}
		}
		anonymousType := valid && !functionValueTokenEquals(program, end, "{") && !functionValueIsDeclaredFunction(program, funcTok) && (nestedSignature || functionValueNamedSignatureByShape(signatures, candidate) >= 0)
		for i := 0; anonymousType && i < len(signatures); i++ {
			if signatures[i].declFuncTok == funcTok && !functionValueTokenEquals(program, funcTok-1, "=") {
				anonymousType = false
			}
		}
		if !valid || !conversion && !accessorResult && !inferredLocal && !anonymousType {
			continue
		}
		if functionValueSignatureByShape(signatures, candidate) >= 0 {
			continue
		}
		candidate.name = "__renvo_function_" + functionValueDecimal(len(signatures))
		candidate.declFuncTok = -1
		candidate.declEndTok = -1
		candidate.anonymous = true
		signatures = append(signatures, candidate)
	}
	for i := 0; i < len(signatures); i++ {
		for j := 0; j < i; j++ {
			if functionValueSameShape(signatures[j], signatures[i]) {
				signatures[i].storage = signatures[j].storage
				break
			}
		}
	}
	return signatures, fields, edits, true
}

func functionValueGlobalInitializerFunction(program *unit.Program, decl unit.Decl) int {
	if decl.Kind != unit.TokenVar {
		return -1
	}
	nameTok := functionValueTokenAtSpan(program, decl.NameStart, decl.NameEnd)
	if nameTok < 0 {
		return -1
	}
	first, last := nameTok, nameTok
	for first >= 2 && functionValueTokenEquals(program, first-1, ",") && program.Tokens[first-2].KindLine&255 == unit.TokenIdent {
		first -= 2
	}
	for last+2 < len(program.Tokens) && functionValueTokenEquals(program, last+1, ",") && program.Tokens[last+2].KindLine&255 == unit.TokenIdent {
		last += 2
	}
	if !functionValueTokenEquals(program, last+1, "=") {
		return -1
	}
	start := last + 2
	end := decl.EndTok
	for end > start && functionValueTokenEquals(program, end-1, ";") {
		end--
	}
	if last > first {
		starts, ends := functionValueCommaParts(program, start, end)
		if len(starts) != (last-first)/2+1 {
			return -1
		}
		index := (nameTok - first) / 2
		start, end = starts[index], ends[index]
	}
	start, end = functionValueUnparen(program, start, end)
	if end != start+1 || program.Tokens[start].KindLine&255 != unit.TokenIdent {
		return -1
	}
	name := functionValueTokenText(program, start)
	for i := 0; i < len(program.Funcs); i++ {
		fn := &program.Funcs[i]
		if fn.ReceiverStart == fn.ReceiverEnd && functionValueTokenEquals(program, fn.NameTok, name) {
			return i
		}
	}
	return -1
}

func parseFunctionValueSignature(program *unit.Program, funcTok int, name string) (functionValueSignature, int, bool) {
	if !functionValueTokenEquals(program, funcTok, "func") {
		return functionValueSignature{}, funcTok, false
	}
	return parseFunctionValueCallableSignature(program, funcTok, name)
}

func parseFunctionValueCallableSignature(program *unit.Program, funcTok int, name string) (functionValueSignature, int, bool) {
	var sig functionValueSignature
	sig.name = name
	sig.declFuncTok = funcTok
	if !functionValueTokenEquals(program, funcTok+1, "(") {
		return sig, funcTok, false
	}
	close := functionValueFindMatchingParen(program, funcTok+1)
	if close < 0 {
		return sig, funcTok, false
	}
	params, names, types, ok := normalizedFunctionValueParams(program, funcTok+2, close)
	if !ok {
		return sig, funcTok, false
	}
	sig.params = params
	sig.paramNames = names
	sig.paramTypes = types
	end := close + 1
	if functionValueTokenEquals(program, end, "(") {
		resultClose := functionValueFindMatchingParen(program, end)
		if resultClose < 0 {
			return sig, funcTok, false
		}
		sig.result = functionValueTokensText(program, end, resultClose+1)
		sig.resultTypes = functionValueTupleResultTypes(program, end+1, resultClose)
		zeroType := functionValueSingleResultType(program, end+1, resultClose)
		if functionValueZero(zeroType) == "0" && !functionValueCanUseScalarZero(zeroType) {
			sig.zeroType = zeroType
		}
		end = resultClose + 1
	} else if functionValueTokenCanStartType(program, end) && program.Tokens[end].KindLine>>8 == program.Tokens[close].KindLine>>8 {
		resultEnd := functionValueTypeEnd(program, end)
		if resultEnd <= end {
			return sig, funcTok, false
		}
		sig.result = functionValueTokensText(program, end, resultEnd)
		sig.resultTypes = []string{sig.result}
		if functionValueZero(sig.result) == "0" && !functionValueCanUseScalarZero(sig.result) {
			sig.zeroType = sig.result
		}
		end = resultEnd
	}
	sig.storage = &functionValueStorage{}
	sig.sourceFuncTok, sig.sourceEndTok = funcTok, end
	// Native declarations keep their authored parameter names and aliases in
	// the body. Match their callable types against the canonical callback
	// spellings nevertheless, so an equivalent alias does not lose its arm.
	if program.Tokens[funcTok].KindLine&255 == unit.TokenIdent {
		sig.paramTypes = functionValueCanonicalCallableFields(program, funcTok+2, close)
		if functionValueTokenEquals(program, close+1, "(") {
			sig.resultTypes = functionValueCanonicalCallableFields(program, close+2, end-1)
		} else if len(sig.resultTypes) == 1 {
			sig.resultTypes[0] = functionValueCanonicalSignatureType(program, close+1, end, 0)
		}
	}
	return sig, end, true
}

func functionValueTypeEnd(program *unit.Program, start int) int {
	if start < 0 || start >= len(program.Tokens) {
		return start
	}
	text := functionValueTokenText(program, start)
	if text == "*" {
		return functionValueTypeEnd(program, start+1)
	}
	if text == "[" {
		close := functionValueFindMatching(program, start, "[", "]")
		if close < 0 {
			return start
		}
		return functionValueTypeEnd(program, close+1)
	}
	if text == "map" {
		if !functionValueTokenEquals(program, start+1, "[") {
			return start
		}
		close := functionValueFindMatching(program, start+1, "[", "]")
		if close < 0 {
			return start
		}
		return functionValueTypeEnd(program, close+1)
	}
	if text == "<-" && functionValueTokenEquals(program, start+1, "chan") {
		return functionValueTypeEnd(program, start+1)
	}
	if text == "chan" {
		element := start + 1
		if functionValueTokenEquals(program, element, "<-") {
			element++
		}
		return functionValueTypeEnd(program, element)
	}
	if text == "struct" || text == "interface" {
		if !functionValueTokenEquals(program, start+1, "{") {
			return start
		}
		close := functionValueFindMatchingBrace(program, start+1)
		if close < 0 {
			return start
		}
		return close + 1
	}
	if text == "func" {
		if !functionValueTokenEquals(program, start+1, "(") {
			return start
		}
		close := functionValueFindMatchingParen(program, start+1)
		if close < 0 {
			return start
		}
		end := close + 1
		if end >= len(program.Tokens) || program.Tokens[end].KindLine>>8 != program.Tokens[close].KindLine>>8 {
			return end
		}
		if functionValueTokenEquals(program, end, "(") {
			resultClose := functionValueFindMatchingParen(program, end)
			if resultClose < 0 {
				return start
			}
			return resultClose + 1
		}
		resultEnd := functionValueTypeEnd(program, end)
		if resultEnd > end {
			return resultEnd
		}
		return end
	}
	if text == "(" {
		close := functionValueFindMatchingParen(program, start)
		if close < 0 {
			return start
		}
		return close + 1
	}
	if program.Tokens[start].KindLine&255 == unit.TokenIdent {
		end := start + 1
		if functionValueTokenEquals(program, end, ".") && end+1 < len(program.Tokens) && program.Tokens[end+1].KindLine&255 == unit.TokenIdent {
			end += 2
		}
		return end
	}
	return start
}

func functionValueSingleResultType(program *unit.Program, start int, end int) string {
	if start >= end {
		return ""
	}
	depth := 0
	for i := start; i < end; i++ {
		text := functionValueTokenText(program, i)
		if text == "(" || text == "[" || text == "{" {
			depth++
		} else if text == ")" || text == "]" || text == "}" {
			depth--
		} else if text == "," && depth == 0 {
			return ""
		}
	}
	typeStart := start
	if start+1 < end && program.Tokens[start].KindLine&255 == unit.TokenIdent && functionValueTypeEnd(program, start) != end && functionValueTypeEnd(program, start+1) == end {
		typeStart++
	}
	if functionValueTypeEnd(program, typeStart) != end {
		return ""
	}
	return functionValueTokensText(program, typeStart, end)
}

func normalizedFunctionValueParams(program *unit.Program, start int, end int) (string, []string, []string, bool) {
	var partStarts []int
	var partEnds []int
	partStart := start
	depth := 0
	for i := start; i <= end; i++ {
		text := functionValueTokenText(program, i)
		if i < end {
			if text == "(" || text == "[" || text == "{" {
				depth++
			} else if text == ")" || text == "]" || text == "}" {
				depth--
			}
		}
		if i != end && !(depth == 0 && text == ",") {
			continue
		}
		if partStart < i {
			partStarts = append(partStarts, partStart)
			partEnds = append(partEnds, i)
		}
		partStart = i + 1
	}
	var out []byte
	var names []string
	var types []string
	for i := 0; i < len(partStarts); i++ {
		if len(out) > 0 {
			out = appendFunctionValueString(out, ", ")
		}
		partLen := partEnds[i] - partStarts[i]
		name := "arg" + functionValueDecimal(i)
		typ := functionValueTokensText(program, partStarts[i], partEnds[i])
		if partLen >= 2 && program.Tokens[partStarts[i]].KindLine&255 == unit.TokenIdent && functionValueTypeEnd(program, partStarts[i]) != partEnds[i] {
			name = functionValueTokenText(program, partStarts[i])
			typ = functionValueTokensText(program, partStarts[i]+1, partEnds[i])
		} else if partLen == 1 {
			// All names in a group take the final field's type, even when a
			// parameter name also names a declaration in the enclosing scope.
			for j := i + 1; j < len(partStarts); j++ {
				if partEnds[j]-partStarts[j] == 1 {
					continue
				}
				if program.Tokens[partStarts[j]].KindLine&255 == unit.TokenIdent && functionValueTypeEnd(program, partStarts[j]) != partEnds[j] {
					name = functionValueTokenText(program, partStarts[i])
					typ = functionValueTokensText(program, partStarts[j]+1, partEnds[j])
				}
				break
			}
		}
		out = appendFunctionValueString(out, name+" "+typ)
		names = append(names, name)
		types = append(types, typ)
	}
	return string(out), names, types, depth == 0
}

func appendFunctionValueTypeEdits(program *unit.Program, signatures []functionValueSignature, edits []functionValueEdit) []functionValueEdit {
	for funcTok := 0; funcTok+1 < len(program.Tokens); funcTok++ {
		if !functionValueTokenEquals(program, funcTok, "func") || !functionValueTokenEquals(program, funcTok+1, "(") {
			continue
		}
		declaredType := false
		for i := 0; i < len(signatures); i++ {
			if signatures[i].declFuncTok >= 0 && signatures[i].declFuncTok <= funcTok && funcTok < signatures[i].declEndTok {
				declaredType = true
				break
			}
		}
		if declaredType {
			continue
		}
		candidate, end, ok := parseFunctionValueSignature(program, funcTok, "")
		arrayElementType := funcTok > 0 && functionValueTokenEquals(program, funcTok-1, "]")
		declaredSignature := functionValueTokenInDeclaredSignature(program, funcTok)
		if !ok || functionValueTokenEquals(program, end, "{") && !arrayElementType && !declaredSignature {
			continue
		}
		sigIndex := functionValueSignatureByShape(signatures, candidate)
		if sigIndex < 0 {
			continue
		}
		conversionStart := funcTok
		conversionEnd := end
		if funcTok > 0 && functionValueTokenEquals(program, funcTok-1, "(") &&
			!functionValueTokenEquals(program, funcTok-2, "func") &&
			!functionValueTokenEquals(program, funcTok-2, ".") &&
			functionValueTokenEquals(program, end, ")") &&
			functionValueFindMatchingParen(program, funcTok-1) == end {
			conversionStart = funcTok - 1
			conversionEnd = end + 1
		}
		if functionValueTokenEquals(program, conversionEnd, "(") {
			close := functionValueFindMatchingParen(program, conversionEnd)
			if close > conversionEnd {
				argument := functionValueTokensText(program, conversionEnd+1, close)
				argumentType := ordinaryUnderlyingType(program, ordinaryBuiltinExprType(program, conversionStart, conversionEnd+1, close), 0)
				if functionValueTokenEquals(program, conversionEnd+1, "nil") && conversionEnd+2 == close {
					edits = append(edits, functionValueTokenRangeEdit(program, conversionStart, close+1, signatures[sigIndex].name+"{}"))
					funcTok = close
				} else if ordinaryBuiltinTypeName(argumentType) || len(argumentType) > 0 && argumentType[0] == '*' || argumentType == "unsafe.Pointer" {
					// C pointer casts retain their scalar tag. Go function-to-function
					// conversions copy the complete callable storage instead.
					replacement := signatures[sigIndex].name + "{kind: int(uintptr(" + argument + "))}"
					edits = append(edits, functionValueTokenRangeEdit(program, conversionStart, close+1, replacement))
					funcTok = close
				} else {
					edits = append(edits, functionValueTokenRangeEdit(program, conversionStart, conversionEnd, signatures[sigIndex].name))
					edits = lowerFunctionValueAt(program, conversionStart, conversionEnd+1, sigIndex, signatures, edits)
					funcTok = end - 1
				}
				continue
			}
		}
		edit := functionValueTokenRangeEdit(program, funcTok, end, signatures[sigIndex].name)
		duplicate := false
		for i := 0; i < len(edits); i++ {
			if edits[i].start == edit.start && edits[i].end == edit.end {
				duplicate = true
				break
			}
		}
		if !duplicate {
			edits = append(edits, edit)
		}
		funcTok = end - 1
	}
	return edits
}

func functionValueTokenInDeclaredSignature(program *unit.Program, token int) bool {
	for i := 0; i < len(program.Funcs); i++ {
		fn := &program.Funcs[i]
		if fn.StartTok < token && token < fn.BodyStart {
			return true
		}
	}
	return false
}

func functionValueTokenInDeclaredResult(program *unit.Program, token int) bool {
	for i := 0; i < len(program.Funcs); i++ {
		fn := &program.Funcs[i]
		open := fn.NameTok + 1
		if !functionValueTokenEquals(program, open, "(") {
			continue
		}
		close := functionValueFindMatchingParen(program, open)
		if close < token && token < fn.BodyStart {
			return true
		}
	}
	return false
}

func lowerFunctionValueArrayComposites(program *unit.Program, signatures []functionValueSignature, edits []functionValueEdit) []functionValueEdit {
	for open := 0; open < len(program.Tokens); open++ {
		if !functionValueTokenEquals(program, open, "{") {
			continue
		}
		funcTok := -1
		for candidate := open - 1; candidate >= 0; candidate-- {
			if functionValueTokenEquals(program, candidate, "{") || functionValueTokenEquals(program, candidate, ";") {
				break
			}
			if functionValueTokenEquals(program, candidate, "func") {
				funcTok = candidate
				break
			}
		}
		if funcTok < 1 || !functionValueTokenEquals(program, funcTok-1, "]") {
			continue
		}
		candidate, end, ok := parseFunctionValueSignature(program, funcTok, "")
		if !ok || end != open {
			continue
		}
		sigIndex := functionValueSignatureByShape(signatures, candidate)
		if sigIndex < 0 {
			continue
		}
		close := functionValueFindMatchingBrace(program, open)
		if close < 0 {
			continue
		}
		starts, ends := functionValueCommaParts(program, open+1, close)
		for i := 0; i < len(starts); i++ {
			if ends[i] == starts[i]+1 {
				edits = lowerFunctionValueAt(program, starts[i], starts[i], sigIndex, signatures, edits)
			}
		}
		open = close
	}
	return edits
}

func lowerFunctionValueReturns(program *unit.Program, signatures []functionValueSignature, edits []functionValueEdit) []functionValueEdit {
	for token := 0; token+1 < len(program.Tokens); token++ {
		if !functionValueTokenEquals(program, token, "return") {
			continue
		}
		fn, found := functionValueLexicalFunction(program, token)
		if !found {
			continue
		}
		resultType := functionValueDeclaredResultType(program, &fn)
		sigIndex := functionValueSignatureByName(signatures, functionValueBareType(resultType))
		if sigIndex < 0 {
			sigIndex = functionValueSignatureByTypeText(signatures, resultType)
		}
		if sigIndex < 0 {
			continue
		}
		rhs := token + 1
		rhsEnd := rhs + 1
		if functionValueTokenEquals(program, rhs, "(") {
			close := functionValueFindMatchingParen(program, rhs)
			if close > rhs {
				rhsEnd = close + 1
			}
		} else if functionValueTokenEquals(program, rhs+1, "(") {
			close := functionValueFindMatchingParen(program, rhs+1)
			if close > rhs {
				rhsEnd = close + 1
			}
		}
		if ordinaryBuiltinTypeName(functionValueBareType(ordinaryBuiltinExprType(program, token, rhs, rhsEnd))) {
			replacement := signatures[sigIndex].name + "{kind: int(" + functionValueTokensText(program, rhs, rhsEnd) + ")}"
			edits = append(edits, functionValueTokenRangeEdit(program, rhs, rhsEnd, replacement))
			continue
		}
		edits = lowerFunctionValueAt(program, token, rhs, sigIndex, signatures, edits)
	}
	return edits
}

func lowerFunctionValueAssignment(program *unit.Program, op int, signatures []functionValueSignature, fields []functionValueField, edits []functionValueEdit) ([]functionValueEdit, bool) {
	first := op - 1
	for first >= 2 && functionValueTokenEquals(program, first-1, ",") && program.Tokens[first-2].KindLine&255 == unit.TokenIdent {
		first -= 2
	}
	if first < op-1 && functionValueTokenEquals(program, first-1, "var") {
		// Each inferred initializer was matched to its own callable shape.
		return edits, true
	}
	sigIndex := functionValueAssignmentSignature(program, op, signatures, fields)
	if sigIndex < 0 || op+1 >= len(program.Tokens) {
		return edits, true
	}
	before := op
	depth := 0
	for token := op - 1; token >= 0; token-- {
		if depth == 0 && functionValueAssignmentLineEnds(program, token, token+1) {
			break
		}
		if functionValueTokenEquals(program, token, ")") || functionValueTokenEquals(program, token, "]") || functionValueTokenEquals(program, token, "}") {
			depth++
		} else if functionValueTokenEquals(program, token, "(") || functionValueTokenEquals(program, token, "[") || functionValueTokenEquals(program, token, "{") {
			if depth == 0 {
				break
			}
			depth--
		} else if depth == 0 {
			if functionValueTokenEquals(program, token, ";") || functionValueTokenEquals(program, token, "=") || functionValueTokenEquals(program, token, ":=") {
				break
			}
			if functionValueTokenEquals(program, token, "var") {
				before = token + 1
				break
			}
		}
	}
	return lowerFunctionValueAt(program, before, op+1, sigIndex, signatures, edits), true
}

func functionValueAssignmentLineEnds(program *unit.Program, token int, next int) bool {
	if program.Tokens[token].KindLine>>8 == program.Tokens[next].KindLine>>8 {
		return false
	}
	kind := program.Tokens[token].KindLine & 255
	return kind == unit.TokenIdent || kind == unit.TokenNumber || kind == unit.TokenFloat || kind == unit.TokenString || kind == unit.TokenChar || functionValueTokenEquals(program, token, ")") || functionValueTokenEquals(program, token, "]") || functionValueTokenEquals(program, token, "}")
}

func functionValueAssignmentSignature(program *unit.Program, op int, signatures []functionValueSignature, fields []functionValueField) int {
	// In a typed declaration the tokens before '=' denote the declared type,
	// rather than the variable being initialized. Normalize function parameter
	// names through the parsed signature before selecting its representation.
	depth := 0
	for token := op - 1; token >= 0; token-- {
		if functionValueTokenEquals(program, token, ")") || functionValueTokenEquals(program, token, "]") || functionValueTokenEquals(program, token, "}") {
			depth++
			continue
		}
		if functionValueTokenEquals(program, token, "(") || functionValueTokenEquals(program, token, "[") || functionValueTokenEquals(program, token, "{") {
			if depth == 0 {
				break
			}
			depth--
			continue
		}
		if depth != 0 {
			continue
		}
		if functionValueTokenEquals(program, token, ";") || functionValueTokenEquals(program, token, "=") || functionValueTokenEquals(program, token, ":=") {
			break
		}
		if functionValueTokenEquals(program, token, "func") {
			candidate, end, ok := parseFunctionValueSignature(program, token, "")
			if ok && end == op {
				return functionValueSignatureByShape(signatures, candidate)
			}
		}
	}
	fieldTok := functionValueSelectorFieldBefore(program, op)
	fieldIndex := functionValueFieldForSelector(program, fieldTok, fields)
	sigIndex := -1
	if fieldIndex >= 0 {
		sigIndex = fields[fieldIndex].sig
	} else if op > 0 && program.Tokens[op-1].KindLine&255 == unit.TokenIdent {
		leftName := functionValueTokenText(program, op-1)
		leftType := functionValueEnclosingLocalType(program, op, leftName)
		sigIndex = functionValueSignatureByName(signatures, functionValueBareType(leftType))
		if sigIndex < 0 {
			sigIndex = functionValueSignatureByTypeText(signatures, leftType)
		}
	}
	if sigIndex < 0 && op > 0 {
		// Indirect C structs expose function-pointer fields through an accessor,
		// so the assignment target is an expression such as (*p.callback()).
		// Infer its result type just as the call lowering pass does.
		leftStart := functionValuePrimaryStart(program, op-1)
		if functionValueTokenEquals(program, op-1, ")") {
			leftStart = functionValueFindMatchingBackward(program, op-1, "(", ")")
		}
		if leftStart >= 0 {
			leftType := ordinaryBuiltinExprType(program, op, leftStart, op)
			bareType := functionValueBareType(leftType)
			sigIndex = functionValueSignatureByName(signatures, bareType)
			if sigIndex < 0 {
				sigIndex = functionValueSignatureByTypeText(signatures, bareType)
			}
		}
	}
	return sigIndex
}

func lowerFunctionValueCompositeField(program *unit.Program, colon int, signatures []functionValueSignature, fields []functionValueField, edits []functionValueEdit) []functionValueEdit {
	if colon < 1 || colon+1 >= len(program.Tokens) || program.Tokens[colon-1].KindLine&255 != unit.TokenIdent {
		return edits
	}
	fieldName := functionValueTokenText(program, colon-1)
	fieldIndex := functionValueFieldByOwnerAndName(fields, functionValueCompositeOwner(program, colon-1), fieldName)
	if fieldIndex < 0 {
		fieldIndex = functionValueUniqueFieldByName(fields, fieldName)
	}
	if fieldIndex < 0 {
		return edits
	}
	return lowerFunctionValueAt(program, colon, colon+1, fields[fieldIndex].sig, signatures, edits)
}

func lowerFunctionValueAt(program *unit.Program, at int, rhs int, sigIndex int, signatures []functionValueSignature, edits []functionValueEdit) []functionValueEdit {
	if functionValueTokenEquals(program, rhs, "(") {
		close := functionValueFindMatchingParen(program, rhs)
		if close > rhs && !functionValueTokenEquals(program, close+1, "(") && !functionValueTokenEquals(program, close+1, ".") && !functionValueTokenEquals(program, close+1, "[") {
			rhs, _ = functionValueUnparen(program, rhs, close+1)
		}
	}
	if functionValueTokenEquals(program, rhs, "nil") {
		edits = append(edits, functionValueTokenEdit(program, rhs, signatures[sigIndex].name+"{}"))
		return edits
	}
	if program.Tokens[rhs].KindLine&255 == unit.TokenIdent {
		name := functionValueTokenText(program, rhs)
		if !functionValueTokenEquals(program, rhs+1, "(") &&
			functionValueEnclosingLocalType(program, at, name) == "" && functionValueDeclaredDirectFunction(program, name) {
			implIndex := functionValueImplIndex(signatures[sigIndex], "", "", name)
			if implIndex < 0 {
				implIndex = len(signatures[sigIndex].storage.impls)
				signatures[sigIndex].storage.impls = append(signatures[sigIndex].storage.impls, functionValueImpl{function: name})
			}
			replacement := signatures[sigIndex].name + "{kind: " + functionValueDecimal(implIndex+1) + "}"
			edits = append(edits, functionValueTokenEdit(program, rhs, replacement))
			return edits
		}
	}
	return lowerFunctionValueBoundMethod(program, at, rhs, sigIndex, signatures, edits)
}

func functionValueDeclaredDirectFunction(program *unit.Program, name string) bool {
	for i := 0; i < len(program.Funcs); i++ {
		fn := &program.Funcs[i]
		if fn.ReceiverStart == fn.ReceiverEnd && functionValueTokenEquals(program, fn.NameTok, name) {
			return true
		}
	}
	return false
}

func lowerFunctionValueBoundMethod(program *unit.Program, at int, rhs int, sigIndex int, signatures []functionValueSignature, edits []functionValueEdit) []functionValueEdit {
	if rhs+2 >= len(program.Tokens) || !functionValueTokenEquals(program, rhs+1, ".") || program.Tokens[rhs].KindLine&255 != unit.TokenIdent || program.Tokens[rhs+2].KindLine&255 != unit.TokenIdent {
		return edits
	}
	method := functionValueTokenText(program, rhs+2)
	receiverType := functionValueMethodReceiverTypeForBase(program, at, functionValueTokenText(program, rhs), method)
	if receiverType == "" {
		return edits
	}
	implIndex := functionValueImplIndex(signatures[sigIndex], receiverType, method, "")
	if implIndex < 0 {
		implIndex = len(signatures[sigIndex].storage.impls)
		fieldName := functionValueSharedMethodEnvironment(signatures[sigIndex], receiverType)
		if fieldName == "" {
			fieldName = "__renvo_method_env_" + functionValueDecimal(sigIndex) + "_" + functionValueDecimal(implIndex)
		}
		signatures[sigIndex].storage.impls = append(signatures[sigIndex].storage.impls, functionValueImpl{receiverType: receiverType, environmentType: fieldName, method: method, nilReceiver: functionValueHasPrefix(ordinaryUnderlyingType(program, receiverType, 0), "interface")})
	}
	impl := signatures[sigIndex].storage.impls[implIndex]
	receiver := functionValueTokenText(program, rhs)
	baseType := functionValueEnclosingLocalType(program, at, receiver)
	if len(receiverType) > 0 && receiverType[0] == '*' && len(baseType) > 0 && baseType[0] != '*' {
		receiver = "&" + receiver
	}
	replacement := signatures[sigIndex].name + "{kind: " + functionValueDecimal(implIndex+1) + ", data: &" + impl.environmentType + "{value: " + receiver + "}}"
	if impl.nilReceiver {
		replacement = "__renvo_bind_" + functionValueDecimal(sigIndex) + "_" + functionValueDecimal(implIndex) + "(" + receiver + ")"
	}
	edits = append(edits, functionValueTokenRangeEdit(program, rhs, rhs+3, replacement))
	return edits
}

func functionValueSharedMethodEnvironment(sig functionValueSignature, receiverType string) string {
	for i := 0; i < len(sig.storage.impls); i++ {
		impl := sig.storage.impls[i]
		if impl.method != "" && impl.receiverType == receiverType {
			return impl.environmentType
		}
	}
	return ""
}

func lowerFunctionValueCallArguments(program *unit.Program, signatures []functionValueSignature, edits []functionValueEdit) []functionValueEdit {
	indirect := false
	for _, signature := range signatures {
		for _, parameter := range signature.paramTypes {
			if functionValueSignatureByName(signatures, functionValueBareType(parameter)) >= 0 || functionValueSignatureByTypeText(signatures, parameter) >= 0 {
				indirect = true
			}
		}
	}
	if !indirect {
		// A native function variable can take an anonymous callback without
		// needing tagged storage for its own signature.
		for token := 0; token+1 < len(program.Tokens) && !indirect; token++ {
			if !functionValueTokenEquals(program, token, "func") || !functionValueTokenEquals(program, token+1, "(") {
				continue
			}
			close := functionValueFindMatchingParen(program, token+1)
			for nested := token + 2; nested < close; nested++ {
				if functionValueTokenEquals(program, nested, "func") && functionValueTokenEquals(program, nested+1, "(") {
					indirect = true
					break
				}
			}
			if close > token {
				token = close
			}
		}
	}
	// Linked units can contain thousands of functions. Index declarations once
	// so each potential call only examines functions with the same name hash.
	buckets := make([]int, len(program.Funcs)*2+1)
	next := make([]int, len(program.Funcs))
	for i := len(program.Funcs) - 1; i >= 0; i-- {
		bucket := functionValueNameHash(program, program.Funcs[i].NameTok) % len(buckets)
		next[i] = buckets[bucket]
		buckets[bucket] = i + 1
	}
	for open := 0; open < len(program.Tokens); open++ {
		if !functionValueTokenEquals(program, open, "(") || functionValueCallIsDeclaration(program, open) {
			continue
		}
		mark := arena.Mark()
		before := len(edits)
		edits = lowerFunctionValueCallArgumentsAt(program, open, signatures, edits, buckets, next, indirect)
		if len(edits) == before {
			arena.Rewind(mark)
		}
	}
	return edits
}

func lowerFunctionValueCallArgumentsAt(program *unit.Program, open int, signatures []functionValueSignature, edits []functionValueEdit, buckets []int, next []int, indirect bool) []functionValueEdit {
	fn, ok := functionValueCalledFunctionIndexed(program, open, buckets, next)
	if !ok && open > 0 && program.Tokens[open-1].KindLine&255 == unit.TokenIdent {
		if sigIndex := functionValueConversionSignature(program, open, signatures); sigIndex >= 0 {
			return lowerFunctionValueAt(program, open, open+1, sigIndex, signatures, edits)
		}
	}
	if !ok && !indirect {
		return edits
	}
	close := functionValueFindMatchingParen(program, open)
	if close <= open {
		return edits
	}
	var paramTypes []string
	if ok {
		paramTypes = functionValueFunctionParamTypes(program, fn)
	} else {
		start := functionValuePrimaryStart(program, open-1)
		if start < 0 {
			return edits
		}
		typ := ordinaryBuiltinExprType(program, open, start, open)
		index := functionValueSignatureByName(signatures, functionValueBareType(typ))
		if index < 0 {
			index = functionValueSignatureByTypeText(signatures, typ)
		}
		if index < 0 {
			parsed, valid := functionValueSignatureFromTypeText(ordinaryUnderlyingType(program, typ, 0))
			if !valid {
				return edits
			}
			paramTypes = parsed.paramTypes
		} else {
			paramTypes = signatures[index].paramTypes
		}
	}
	argStarts, argEnds := functionValueCommaParts(program, open+1, close)
	if len(argStarts) != len(paramTypes) {
		return edits
	}
	for i := 0; i < len(argStarts); i++ {
		sigIndex := -1
		if ok {
			sigIndex = functionValueParameterSignature(program, fn, i, paramTypes[i], signatures)
		} else {
			sigIndex = functionValueSignatureByName(signatures, functionValueBareType(paramTypes[i]))
			if sigIndex < 0 {
				sigIndex = functionValueSignatureByTypeText(signatures, paramTypes[i])
			}
		}
		if sigIndex < 0 {
			continue
		}
		valueStart := argStarts[i]
		valueEnd := argEnds[i]
		for valueEnd-valueStart >= 2 && functionValueTokenEquals(program, valueStart, "(") &&
			functionValueFindMatchingParen(program, valueStart) == valueEnd-1 {
			valueStart++
			valueEnd--
		}
		argType := functionValueBareType(ordinaryBuiltinExprType(program, open, valueStart, valueEnd))
		if ordinaryBuiltinTypeName(argType) {
			replacement := signatures[sigIndex].name + "{kind: int(" + functionValueTokensText(program, valueStart, valueEnd) + ")}"
			edits = append(edits, functionValueTokenRangeEdit(program, valueStart, valueEnd, replacement))
			continue
		}
		edits = lowerFunctionValueAt(program, open, valueStart, sigIndex, signatures, edits)
	}
	return edits
}

func functionValueCallIsDeclaration(program *unit.Program, open int) bool {
	if open < 2 || program.Tokens[open-1].KindLine&255 != unit.TokenIdent {
		return false
	}
	before := open - 2
	if functionValueTokenEquals(program, before, ")") {
		before = functionValueFindMatchingBackward(program, before, "(", ")") - 1
	}
	// The declaration prefix is local: func name( or func (receiver) name(.
	// Searching every function for every parenthesis makes large linked units
	// pay a quadratic cost before any callback lowering takes place.
	return before >= 0 && functionValueTokenEquals(program, before, "func")
}

func functionValueCalledFunction(program *unit.Program, open int) (*unit.Func, bool) {
	return functionValueCalledFunctionIndexed(program, open, nil, nil)
}

func functionValueNameHash(program *unit.Program, token int) int {
	name := functionValueTokenText(program, token)
	hash := 0
	for i := 0; i < len(name); i++ {
		hash = (hash*31 + int(name[i])) & 2147483647
	}
	return hash
}

func functionValueCalledFunctionIndexed(program *unit.Program, open int, buckets []int, next []int) (*unit.Func, bool) {
	var fallback *unit.Func
	fallbackOK := false
	if open < 1 || program.Tokens[open-1].KindLine&255 != unit.TokenIdent {
		return fallback, false
	}
	name := functionValueTokenText(program, open-1)
	selector := open >= 3 && functionValueTokenEquals(program, open-2, ".")
	baseType := ""
	fallbackCount := 0
	if selector {
		selectorStart := functionValueSelectorStart(program, open-1)
		if selectorStart >= 0 {
			baseType = functionValueEnclosingLocalType(program, open, functionValueTokenText(program, selectorStart))
		}
	}
	i := 0
	if len(buckets) > 0 {
		i = buckets[functionValueNameHash(program, open-1)%len(buckets)] - 1
	}
	for i >= 0 && i < len(program.Funcs) {
		fn := &program.Funcs[i]
		if len(buckets) > 0 {
			i = next[i] - 1
		} else {
			i++
		}
		if !functionValueTokenEquals(program, fn.NameTok, name) {
			continue
		}
		method := fn.ReceiverStart < fn.ReceiverEnd
		if method != selector {
			continue
		}
		if !fallbackOK {
			fallback = fn
			fallbackOK = true
		}
		fallbackCount++
		if !method || baseType != "" && functionValueTypeEmbeds(program, baseType, functionValueReceiverType(program, fn), 0) {
			return fn, true
		}
	}
	return fallback, fallbackOK && fallbackCount == 1
}

func functionValueCommaParts(program *unit.Program, start int, end int) ([]int, []int) {
	var starts []int
	var ends []int
	partStart := start
	depth := 0
	for i := start; i <= end; i++ {
		text := functionValueTokenText(program, i)
		if i < end {
			if text == "(" || text == "[" || text == "{" {
				depth++
			} else if text == ")" || text == "]" || text == "}" {
				depth--
			}
		}
		if i != end && !(depth == 0 && text == ",") {
			continue
		}
		if partStart < i {
			starts = append(starts, partStart)
			ends = append(ends, i)
		}
		partStart = i + 1
	}
	return starts, ends
}

func functionValueFunctionParamTypes(program *unit.Program, fn *unit.Func) []string {
	open := fn.NameTok + 1
	if !functionValueTokenEquals(program, open, "(") {
		return nil
	}
	close := functionValueFindMatchingParen(program, open)
	starts, ends := functionValueCommaParts(program, open+1, close)
	types := make([]string, len(starts))
	carried := ""
	for i := len(starts) - 1; i >= 0; i-- {
		start := starts[i]
		end := ends[i]
		if start+4 < end && program.Tokens[start].KindLine&255 == unit.TokenIdent && functionValueTokenEquals(program, start+1, ".") && functionValueTokenEquals(program, start+2, ".") && functionValueTokenEquals(program, start+3, ".") && functionValueTypeEnd(program, start+4) == end {
			carried = "..." + functionValueTokensText(program, start+4, end)
			types[i] = carried
		} else if start+1 < end && program.Tokens[start].KindLine&255 == unit.TokenIdent && functionValueTypeEnd(program, start+1) == end {
			carried = functionValueTokensText(program, start+1, end)
			types[i] = carried
		} else if end == start+1 && carried != "" && !functionValueNamedType(program, functionValueTokenText(program, start)) {
			types[i] = carried
		} else {
			carried = ""
			types[i] = functionValueTokensText(program, start, end)
		}
	}
	return types
}

// Parameter names within a function type do not contribute to its identity.
// Match parsed signatures so func(value *T) and func(*T) use the same wrapper.
func functionValueParameterSignature(program *unit.Program, fn *unit.Func, index int, typ string, signatures []functionValueSignature) int {
	if found := functionValueSignatureByName(signatures, functionValueBareType(typ)); found >= 0 {
		return found
	}
	open := fn.NameTok + 1
	close := functionValueFindMatchingParen(program, open)
	starts, ends := functionValueCommaParts(program, open+1, close)
	for i := index; i < len(starts); i++ {
		start := starts[i]
		end := ends[i]
		if start+1 == end && program.Tokens[start].KindLine&255 == unit.TokenIdent &&
			!functionValueNamedType(program, functionValueTokenText(program, start)) &&
			!ordinaryBuiltinTypeName(functionValueTokenText(program, start)) {
			continue // A grouped parameter obtains its type from the next part.
		}
		if !functionValueTokenEquals(program, start, "func") {
			start++ // Skip the parameter name.
		}
		candidate, signatureEnd, ok := parseFunctionValueSignature(program, start, "")
		if ok && signatureEnd == end {
			return functionValueSignatureByShape(signatures, candidate)
		}
		break
	}
	return functionValueSignatureByTypeText(signatures, typ)
}

func lowerFunctionValueComparison(program *unit.Program, op int, signatures []functionValueSignature, fields []functionValueField, edits []functionValueEdit) []functionValueEdit {
	if op+1 >= len(program.Tokens) {
		return edits
	}
	fieldTok := functionValueSelectorFieldBefore(program, op)
	fieldIndex := functionValueFieldForSelector(program, fieldTok, fields)
	lhsTok := fieldTok
	sigIndex := -1
	lhsExpression := false
	callExpression := functionValueTokenEquals(program, op-1, ")") && functionValuePrimaryStart(program, op-1) < functionValueFindMatchingBackward(program, op-1, "(", ")")
	if fieldIndex >= 0 {
		sigIndex = fields[fieldIndex].sig
	} else {
		lhsTok = functionValuePrimaryTokenBefore(program, op)
		if callExpression {
			start := functionValuePrimaryStart(program, op-1)
			typ := ordinaryBuiltinExprType(program, op, start, op)
			sigIndex = functionValueSignatureByName(signatures, functionValueBareType(typ))
			if sigIndex < 0 {
				sigIndex = functionValueSignatureByTypeText(signatures, typ)
			}
			lhsExpression = sigIndex >= 0
		} else if lhsTok >= 0 && program.Tokens[lhsTok].KindLine&255 == unit.TokenIdent {
			typ := functionValueEnclosingLocalType(program, op, functionValueTokenText(program, lhsTok))
			sigIndex = functionValueSignatureByName(signatures, functionValueBareType(typ))
			if sigIndex < 0 {
				sigIndex = functionValueSignatureByTypeText(signatures, typ)
			}
		} else if lhsTok >= 0 {
			lhsStart := functionValuePrimaryStart(program, lhsTok)
			typ := ordinaryBuiltinExprType(program, op, lhsStart, op)
			sigIndex = functionValueSignatureByName(signatures, functionValueBareType(typ))
			if sigIndex < 0 {
				sigIndex = functionValueSignatureByTypeText(signatures, typ)
			}
		}
	}
	if sigIndex < 0 && !callExpression && functionValueTokenEquals(program, op-1, ")") {
		lhsStart := functionValueFindMatchingBackward(program, op-1, "(", ")")
		if lhsStart >= 0 {
			typ := ordinaryBuiltinExprType(program, op, lhsStart, op)
			bareType := functionValueBareType(typ)
			sigIndex = functionValueSignatureByName(signatures, bareType)
			if sigIndex < 0 {
				sigIndex = functionValueSignatureByTypeText(signatures, bareType)
			}
			lhsExpression = sigIndex >= 0
		}
	}
	if sigIndex < 0 {
		return edits
	}
	rhsStart := op + 1
	rhsEnd := rhsStart + 1
	for functionValueTokenEquals(program, rhsStart, "(") {
		close := functionValueFindMatchingParen(program, rhsStart)
		if close < 0 {
			break
		}
		rhsStart++
		rhsEnd = close
	}
	replacement := ""
	replaceRHS := false
	if rhsEnd == rhsStart+1 && functionValueTokenEquals(program, rhsStart, "nil") {
		replacement = "0"
		replaceRHS = true
	} else if rhsEnd == rhsStart+1 && program.Tokens[rhsStart].KindLine&255 == unit.TokenIdent {
		name := functionValueTokenText(program, rhsStart)
		if !functionValueDeclaredDirectFunction(program, name) {
			return edits
		}
		implIndex := functionValueImplIndex(signatures[sigIndex], "", "", name)
		if implIndex < 0 {
			implIndex = len(signatures[sigIndex].storage.impls)
			signatures[sigIndex].storage.impls = append(signatures[sigIndex].storage.impls, functionValueImpl{function: name})
		}
		replacement = functionValueDecimal(implIndex + 1)
		replaceRHS = true
	} else if ordinaryBuiltinTypeName(functionValueBareType(ordinaryBuiltinExprType(program, op, rhsStart, rhsEnd))) {
		// C function-pointer sentinels such as (void(*)(void*))-1 retain
		// their scalar tag so comparisons remain word-for-word equivalent.
	} else {
		return edits
	}
	end := program.Tokens[lhsTok].Start + program.Tokens[lhsTok].Size
	if lhsExpression {
		end = program.Tokens[op-1].Start + program.Tokens[op-1].Size
		if !callExpression {
			end = program.Tokens[op].Start
		}
	}
	edits = append(edits, functionValueEdit{start: end, end: end, text: ".kind"})
	if replaceRHS {
		edits = append(edits, functionValueTokenEdit(program, rhsStart, replacement))
	}
	return edits
}

func lowerFunctionValueCall(program *unit.Program, open int, signatures []functionValueSignature, fields []functionValueField, edits []functionValueEdit) []functionValueEdit {
	if functionValueTokenInDeclaredSignature(program, open) {
		return edits
	}
	if functionValueConversionSignature(program, open, signatures) >= 0 {
		return edits
	}
	fieldTok := open - 1
	fieldIndex := -1
	if fieldTok >= 2 && functionValueTokenEquals(program, fieldTok-1, ".") {
		fieldIndex = functionValueFieldForSelector(program, fieldTok, fields)
	}
	calleeStart := functionValuePrimaryStart(program, fieldTok)
	if calleeStart < 0 {
		return edits
	}
	first, _ := functionValueUnparen(program, calleeStart, open)
	if functionValueTokenEquals(program, calleeStart-1, "defer") && functionValueTokenEquals(program, first, "func") {
		// A directly deferred literal must remain the deferred function:
		// routing it through a dispatcher changes direct recover ownership.
		return edits
	}
	sigIndex := -1
	if fieldIndex >= 0 {
		sigIndex = fields[fieldIndex].sig
	} else {
		calleeType := ordinaryBuiltinExprType(program, open, calleeStart, open)
		sigIndex = functionValueSignatureByName(signatures, functionValueBareType(calleeType))
		if sigIndex < 0 {
			sigIndex = functionValueSignatureByTypeText(signatures, calleeType)
		}
	}
	if sigIndex < 0 {
		return edits
	}
	close := functionValueFindMatchingParen(program, open)
	if close < 0 {
		return edits
	}
	start := program.Tokens[calleeStart].Start
	// A function value need not be addressable (for example, a map lookup or
	// returned struct field). Snapshot it before evaluating the call arguments.
	edits = append(edits, functionValueEdit{start: start, end: start, text: "__renvo_call_" + functionValueDecimal(sigIndex) + "("})
	if open+1 == close {
		edits = append(edits, functionValueTokenEdit(program, open, ""))
	} else {
		edits = append(edits, functionValueTokenEdit(program, open, ", "))
	}
	return edits
}

func functionValueStructText(sig functionValueSignature) string {
	out := "struct { kind int; data interface{}"
	// Function values remain noncomparable when boxed, even though their tag
	// and receiver storage may all be comparable. A zero-length array of a
	// slice has that property without storing another callable or referring
	// to a source identifier that could hide a predeclared type.
	out = out + "; _ [0][]struct{} }"
	return out
}

func functionValueGeneratedText(signatures []functionValueSignature, closures []functionValueClosure) string {
	out := ""
	var methodEnvironments []string
	for i := 0; i < len(signatures); i++ {
		for _, impl := range signatures[i].storage.impls {
			if impl.method == "" {
				continue
			}
			found := false
			for _, name := range methodEnvironments {
				if name == impl.environmentType {
					found = true
				}
			}
			if !found {
				methodEnvironments = append(methodEnvironments, impl.environmentType)
				out += "type " + impl.environmentType + " struct { value " + impl.receiverType + " }\n"
			}
		}
	}
	for i := 0; i < len(closures); i++ {
		closure := closures[i]
		out = out + "type " + closure.envName + " struct {"
		for j := 0; j < len(closure.fields); j++ {
			out = out + " " + closure.fields[j] + " " + closure.types[j] + ";"
		}
		out = out + " }\n"
		out = out + "func " + closure.funcName + "(env *" + closure.envName
		if closure.params != "" {
			out = out + ", " + closure.params
		}
		out = out + ")"
		if closure.result != "" {
			out = out + " " + closure.result
		}
		out = out + " {" + closure.body + "}\n"
	}
	for i := 0; i < len(signatures); i++ {
		sig := signatures[i]
		if sig.declFuncTok < 0 {
			out = out + "type " + sig.name + " " + functionValueStructText(sig) + "\n"
		}
		out = out + "//renvo:defer-forward call\nfunc __renvo_call_" + functionValueDecimal(i) + "(fn " + sig.name
		if sig.params != "" {
			out = out + ", " + sig.params
		}
		out = out + ")"
		if sig.result != "" {
			out = out + " " + sig.result
		}
		out = out + " {\n"
		args := functionValueJoin(sig.paramNames, ", ")
		if len(sig.paramTypes) > 0 && functionValueHasPrefix(functionValueCompactTypeText(sig.paramTypes[len(sig.paramTypes)-1]), "...") {
			args += "..."
		}
		for j := 0; j < len(sig.storage.impls); j++ {
			impl := sig.storage.impls[j]
			out = out + "if fn.kind == " + functionValueDecimal(j+1) + " { "
			if sig.result != "" {
				out = out + "return "
			}
			if impl.method != "" {
				out = out + "fn.data.(*" + impl.environmentType + ").value." + impl.method + "(" + args + ")"
			} else if impl.environmentType != "" {
				callArgs := "fn.data.(" + impl.receiverType + ")"
				if args != "" {
					callArgs = callArgs + ", " + args
				}
				out = out + impl.function + "(" + callArgs + ")"
			} else {
				out = out + impl.function + "(" + args + ")"
			}
			if sig.result == "" {
				out = out + "; return"
			}
			out = out + " }\n"
		}
		// The zero representation is a nil function, not a no-op returning the
		// result type's zero value. Retain the unreachable return below for the
		// compact backend's structural return handling.
		out = out + "panic(\"call of nil function\")\n"
		if sig.result != "" {
			if len(sig.resultTypes) > 1 {
				var names []string
				for j := 0; j < len(sig.resultTypes); j++ {
					name := "__renvo_zero_" + functionValueDecimal(j)
					out = out + "var " + name + " " + sig.resultTypes[j] + "\n"
					names = append(names, name)
				}
				out = out + "return " + functionValueJoin(names, ", ") + "\n"
			} else if len(sig.resultTypes) == 1 && functionValueSignatureByTypeText(signatures, sig.resultTypes[0]) >= 0 {
				resultSignature := functionValueSignatureByTypeText(signatures, sig.resultTypes[0])
				out = out + "var __renvo_zero " + signatures[resultSignature].name + "\nreturn __renvo_zero\n"
			} else if sig.zeroType != "" {
				out = out + "var __renvo_zero " + sig.zeroType + "\nreturn __renvo_zero\n"
			} else {
				out = out + "return " + functionValueZero(sig.result) + "\n"
			}
		} else {
			out = out + "return\n"
		}
		out = out + "}\n"
		for j, impl := range sig.storage.impls {
			if impl.nilReceiver {
				out += "func __renvo_bind_" + functionValueDecimal(i) + "_" + functionValueDecimal(j) + "(receiver " + impl.receiverType + ") " + sig.name + " { if receiver == nil { panic(\"nil interface method value\") }; return " + sig.name + "{kind: " + functionValueDecimal(j+1) + ", data: &" + impl.environmentType + "{value: receiver}} }\n"
			}
		}
	}
	return out
}

// Dispatchers are generated after the source type edits. Their nested callback
// parameters, results and zero values need the same tagged types as the source.
func functionValueGeneratedTypes(text string, signatures []functionValueSignature) (string, bool) {
	needed := false
	for i := 0; i+4 < len(text); i++ {
		if text[i:i+4] != "func" {
			continue
		}
		next := i + 4
		for next < len(text) && functionValueIsSpace(text[next]) {
			next++
		}
		if next < len(text) && text[next] == '(' {
			needed = true
			break
		}
	}
	if !needed {
		return text, true
	}
	prefix := "package main\n"
	source := []byte(prefix + text)
	program := unit.Program{Package: "main"}
	if !reparseFunctionValueProgram(&program, source, nil, len(source), -1) {
		return "", false
	}
	local := append([]functionValueSignature(nil), signatures...)
	for i := range local {
		local[i].declFuncTok = -1
	}
	edits := appendFunctionValueTypeEdits(&program, local, nil)
	if len(edits) == 0 {
		return text, true
	}
	result, ok := applyFunctionValueEdits(source, edits)
	if !ok {
		return "", false
	}
	return string(result[len(prefix):]), true
}

func functionValueTupleResultTypes(program *unit.Program, start int, end int) []string {
	starts, ends := functionValueCommaParts(program, start, end)
	types := make([]string, len(starts))
	carried := ""
	for i := len(starts) - 1; i >= 0; i-- {
		partStart := starts[i]
		partEnd := ends[i]
		if partStart+1 < partEnd && program.Tokens[partStart].KindLine&255 == unit.TokenIdent && functionValueTypeEnd(program, partStart) != partEnd && functionValueTypeEnd(program, partStart+1) == partEnd {
			carried = functionValueTokensText(program, partStart+1, partEnd)
			types[i] = carried
		} else if partEnd == partStart+1 && carried != "" {
			types[i] = carried
		} else {
			carried = ""
			types[i] = functionValueTokensText(program, partStart, partEnd)
		}
	}
	return types
}

func lowerFunctionValueLiterals(program *unit.Program, signatures []functionValueSignature, fields []functionValueField, closures []functionValueClosure, edits []functionValueEdit) ([]functionValueSignature, []functionValueClosure, []functionValueEdit, bool) {
	// Extract inner literals first; their replacements then become part of
	// the enclosing closure's body and capture initialization.
	for funcTok := len(program.Tokens) - 2; funcTok >= 0; funcTok-- {
		if !functionValueTokenEquals(program, funcTok, "func") || !functionValueTokenEquals(program, funcTok+1, "(") || functionValueIsDeclaredFunction(program, funcTok) {
			continue
		}
		literalSig, signatureEnd, ok := parseFunctionValueSignature(program, funcTok, "")
		if !ok || !functionValueTokenEquals(program, signatureEnd, "{") || functionValueLiteralTypePosition(program, funcTok, signatureEnd) {
			continue
		}
		bodyClose := functionValueFindMatchingBrace(program, signatureEnd)
		if bodyClose < 0 {
			return signatures, closures, edits, false
		}
		calleeLiteral := functionValueLiteralIsCallee(program, funcTok, bodyClose)
		if calleeLiteral && functionValueLiteralIsDirectDefer(program, funcTok, bodyClose) {
			continue
		}
		sigIndex := -1
		replacementStart := funcTok
		replacementEnd := bodyClose + 1
		if functionValueTokenEquals(program, funcTok-1, ":=") {
			sigIndex = functionValueSignatureByShape(signatures, literalSig)
		}
		if functionValueTokenEquals(program, funcTok-1, "(") && functionValueFindMatchingParen(program, funcTok-1) == bodyClose+1 {
			sigIndex = functionValueConversionSignature(program, funcTok-1, signatures)
			if sigIndex >= 0 {
				replacementStart = functionValuePrimaryStart(program, funcTok-2)
				replacementEnd = bodyClose + 2
			}
		}
		fieldTok := funcTok - 2
		if sigIndex < 0 && fieldTok >= 0 && functionValueTokenEquals(program, funcTok-1, ":") {
			fieldName := functionValueTokenText(program, fieldTok)
			fieldIndex := functionValueFieldByOwnerAndName(fields, functionValueCompositeOwner(program, fieldTok), fieldName)
			if fieldIndex < 0 {
				fieldIndex = functionValueUniqueFieldByName(fields, fieldName)
			}
			if fieldIndex >= 0 {
				sigIndex = fields[fieldIndex].sig
			}
		}
		if sigIndex < 0 {
			if !calleeLiteral && functionValueTokenEquals(program, funcTok-1, "=") {
				sigIndex = functionValueAssignmentSignature(program, funcTok-1, signatures, fields)
			}
		}
		if sigIndex < 0 {
			if !calleeLiteral && functionValueTokenEquals(program, funcTok-1, "return") {
				if fn, found := functionValueLexicalFunction(program, funcTok); found {
					resultType := functionValueDeclaredResultType(program, &fn)
					sigIndex = functionValueSignatureByName(signatures, functionValueBareType(resultType))
					if sigIndex < 0 {
						sigIndex = functionValueSignatureByTypeText(signatures, resultType)
					}
				}
			}
		}
		if sigIndex < 0 {
			sigIndex = functionValueLiteralArgumentSignature(program, funcTok, bodyClose, signatures)
		}
		if sigIndex < 0 {
			sigIndex = functionValueLiteralElementSignature(program, funcTok, bodyClose, signatures)
		}
		if sigIndex < 0 {
			// An untyped literal has the anonymous function type. Once that type
			// is lowered, literals in boxing and other expression contexts must
			// use its representation too.
			sigIndex = functionValueSignatureByShape(signatures, literalSig)
		}
		if sigIndex < 0 {
			continue
		}
		captures, captureTypes := functionValueCaptures(program, funcTok, signatureEnd, bodyClose, literalSig.paramNames)
		for i := 0; i < len(captureTypes); i++ {
			captureTypes[i] = "*" + captureTypes[i]
			pointerEnd := 0
			for pointerEnd < len(captureTypes[i]) && captureTypes[i][pointerEnd] == '*' {
				pointerEnd++
			}
			if capturedSig := functionValueSignatureByTypeText(signatures, captureTypes[i][pointerEnd:]); capturedSig >= 0 {
				captureTypes[i] = captureTypes[i][:pointerEnd] + signatures[capturedSig].name
			}
		}
		closureIndex := len(closures)
		envName := "__renvo_closure_env_" + functionValueDecimal(closureIndex)
		funcName := "__renvo_closure_" + functionValueDecimal(closureIndex)
		body, bodyOK := functionValueClosureBody(program, signatureEnd, bodyClose, captures, edits)
		if !bodyOK {
			return signatures, closures, edits, false
		}
		params, result, signatureOK := functionValueClosureSignature(program, funcTok, signatureEnd, literalSig, edits)
		if !signatureOK {
			return signatures, closures, edits, false
		}
		// The extracted signature and body own all internal rewrites, including
		// tagged function types in callback parameters and results.
		kept := edits[:0]
		for _, edit := range edits {
			callPrefix := edit.start == edit.end && edit.start == program.Tokens[replacementStart].Start
			if !callPrefix && edit.start >= program.Tokens[replacementStart].Start && edit.end <= program.Tokens[replacementEnd-1].Start+program.Tokens[replacementEnd-1].Size {
				continue
			}
			kept = append(kept, edit)
		}
		edits = kept
		closures = append(closures, functionValueClosure{envName: envName, funcName: funcName, fields: captures, types: captureTypes, params: params, result: result, body: body})
		implIndex := len(signatures[sigIndex].storage.impls)
		closureField := envName
		signatures[sigIndex].storage.impls = append(signatures[sigIndex].storage.impls, functionValueImpl{receiverType: "*" + envName, environmentType: closureField, function: funcName})
		init := "&" + envName + "{"
		for i := 0; i < len(captures); i++ {
			if i > 0 {
				init = init + ", "
			}
			init = init + captures[i] + ": &" + captures[i]
		}
		init = init + "}"
		replacement := signatures[sigIndex].name + "{kind: " + functionValueDecimal(implIndex+1) + ", data: " + init + "}"
		for token := funcTok - 1; token >= 0; token-- {
			if functionValueTokenEquals(program, token, "if") || functionValueTokenEquals(program, token, "for") || functionValueTokenEquals(program, token, "switch") {
				// Keep the aggregate's braces separate from the control body.
				replacement = "(" + replacement + ")"
				break
			}
			if functionValueTokenEquals(program, token, "{") || functionValueTokenEquals(program, token, "}") || functionValueTokenEquals(program, token, ";") {
				break
			}
		}
		edits = append(edits, functionValueTokenRangeEdit(program, replacementStart, replacementEnd, replacement))
	}
	return signatures, closures, edits, true
}

func functionValueLiteralIsCallee(program *unit.Program, start int, bodyClose int) bool {
	end := bodyClose + 1
	for start > 0 && functionValueTokenEquals(program, start-1, "(") && functionValueFindMatchingParen(program, start-1) == end {
		start--
		end++
	}
	return functionValueTokenEquals(program, end, "(")
}

func functionValueLiteralIsDirectDefer(program *unit.Program, start int, bodyClose int) bool {
	end := bodyClose + 1
	for start > 0 && functionValueTokenEquals(program, start-1, "(") && functionValueFindMatchingParen(program, start-1) == end {
		start--
		end++
	}
	return functionValueTokenEquals(program, start-1, "defer")
}

func functionValueClosureSignature(program *unit.Program, start, end int, original functionValueSignature, pending []functionValueEdit) (string, string, bool) {
	first, last := program.Tokens[start].Start, program.Tokens[end].Start
	var edits []functionValueEdit
	for _, edit := range pending {
		if edit.start >= first && edit.end <= last && !(edit.start == first && edit.end == first) {
			edits = append(edits, functionValueEdit{start: edit.start - first, end: edit.end - first, text: edit.text})
		}
	}
	if len(edits) == 0 {
		return original.params, original.result, true
	}
	header, ok := applyFunctionValueEdits(program.Text[first:last], edits)
	if !ok {
		return "", "", false
	}
	// Reuse parameter normalization, including generated names for unnamed
	// parameters, after applying nested callback type rewrites.
	source := []byte("package main\ntype ClosureSignature " + string(header) + "\n")
	parsed := unit.Program{Package: "main"}
	if !reparseFunctionValueProgram(&parsed, source, nil, len(source), -1) {
		return "", "", false
	}
	for token := 0; token < len(parsed.Tokens); token++ {
		if functionValueTokenEquals(&parsed, token, "func") {
			signature, _, valid := parseFunctionValueSignature(&parsed, token, "")
			return signature.params, signature.result, valid
		}
	}
	return "", "", false
}

func functionValueAnonymousTypeCallee(program *unit.Program, start int, end int) string {
	for end-start >= 2 && functionValueTokenEquals(program, start, "(") && functionValueFindMatchingParen(program, start) == end-1 {
		start++
		end--
	}
	if functionValueTokenEquals(program, start, "func") && functionValueTypeEnd(program, start) == end {
		return functionValueTokensText(program, start, end)
	}
	return ""
}

func functionValueConversionSignature(program *unit.Program, open int, signatures []functionValueSignature) int {
	if open < 1 {
		return -1
	}
	if program.Tokens[open-1].KindLine&255 != unit.TokenIdent {
		start := functionValuePrimaryStart(program, open-1)
		typ := functionValueAnonymousTypeCallee(program, start, open)
		return functionValueSignatureByTypeText(signatures, typ)
	}
	if functionValueTokenEquals(program, open-2, ".") {
		return -1
	}
	name := functionValueTokenText(program, open-1)
	index := functionValueSignatureByName(signatures, name)
	if index < 0 || functionValueLexicalLocalType(program, open, name) != "" {
		return -1
	}
	return index
}

func functionValueLiteralArgumentSignature(program *unit.Program, funcTok int, bodyClose int, signatures []functionValueSignature) int {
	for open := funcTok - 1; open >= 0; open-- {
		if !functionValueTokenEquals(program, open, "(") || functionValueCallIsDeclaration(program, open) {
			continue
		}
		close := functionValueFindMatchingParen(program, open)
		if close <= bodyClose {
			continue
		}
		fn, ok := functionValueCalledFunction(program, open)
		var paramTypes []string
		if ok {
			paramTypes = functionValueFunctionParamTypes(program, fn)
		} else {
			start := functionValuePrimaryStart(program, open-1)
			typ := ordinaryBuiltinExprType(program, open, start, open)
			index := functionValueSignatureByTypeText(signatures, typ)
			if index < 0 {
				index = functionValueSignatureByName(signatures, typ)
			}
			if index < 0 {
				// A native function variable can accept a callback whose own type
				// needs tagged storage. Its containing signature need not be tagged.
				parsed, valid := functionValueSignatureFromTypeText(ordinaryUnderlyingType(program, typ, 0))
				if !valid {
					continue
				}
				paramTypes = parsed.paramTypes
			} else {
				paramTypes = signatures[index].paramTypes
			}
		}
		argStarts, argEnds := functionValueCommaParts(program, open+1, close)
		if len(argStarts) != len(paramTypes) {
			continue
		}
		for i := 0; i < len(argStarts); i++ {
			start, end := argStarts[i], argEnds[i]
			for end-start >= 2 && functionValueTokenEquals(program, start, "(") && functionValueFindMatchingParen(program, start) == end-1 {
				start++
				end--
			}
			// A literal nested in an argument's body or another expression
			// does not inherit that argument's expected signature.
			if start == funcTok && end == bodyClose+1 {
				sigIndex := -1
				if ok {
					sigIndex = functionValueParameterSignature(program, fn, i, paramTypes[i], signatures)
				}
				if sigIndex < 0 {
					sigIndex = functionValueSignatureByName(signatures, paramTypes[i])
				}
				if sigIndex < 0 {
					sigIndex = functionValueSignatureByTypeText(signatures, paramTypes[i])
				}
				return sigIndex
			}
		}
	}
	return -1
}

func functionValueLiteralTypePosition(program *unit.Program, token int, end int) bool {
	if functionValueTokenInDeclaredSignature(program, token) {
		return true
	}
	for start := token - 1; start >= 0; start-- {
		if functionValueTokenEquals(program, start, "{") || functionValueTokenEquals(program, start, "}") || functionValueTokenEquals(program, start, ";") {
			break
		}
		if (functionValueTokenEquals(program, start, "func") || functionValueTokenEquals(program, start, "[") || functionValueTokenEquals(program, start, "map")) && functionValueTypeEnd(program, start) == end {
			return true
		}
	}
	return false
}

func functionValueLiteralElementSignature(program *unit.Program, start int, end int, signatures []functionValueSignature) int {
	for open := start - 1; open >= 0; open-- {
		if functionValueTokenEquals(program, open, "}") {
			open = functionValueFindMatchingBackward(program, open, "{", "}")
			continue
		}
		if !functionValueTokenEquals(program, open, "{") || functionValueFindMatchingBrace(program, open) <= end {
			continue
		}
		typeStart := -1
		for token := open - 1; token >= 0; token-- {
			if functionValueTokenEquals(program, token, "{") || functionValueTokenEquals(program, token, ";") || functionValueTokenEquals(program, token, "=") || functionValueTokenEquals(program, token, ":=") {
				break
			}
			if functionValueTypeEnd(program, token) == open {
				typeStart = token
			}
		}
		if typeStart < 0 {
			return -1
		}
		typ := ordinaryUnderlyingType(program, functionValueTokensText(program, typeStart, open), 0)
		if !functionValueHasPrefix(typ, "[") && !functionValueHasPrefix(typ, "map[") {
			return -1
		}
		source := []byte("package main\ntype Container = " + typ + "\n")
		parsed := unit.Program{Package: "main"}
		if !reparseFunctionValueProgram(&parsed, source, nil, len(source), -1) || len(parsed.Decls) != 1 {
			continue
		}
		elem := functionValueTokenAtSpan(&parsed, parsed.Decls[0].NameStart, parsed.Decls[0].NameEnd) + 2
		if functionValueTokenEquals(&parsed, elem, "map") {
			elem++
		}
		close := functionValueFindMatching(&parsed, elem, "[", "]")
		if close < elem {
			continue
		}
		elem = close + 1
		finish := functionValueTypeEnd(&parsed, elem)
		if finish <= elem {
			continue
		}
		element := functionValueTokensText(&parsed, elem, finish)
		index := functionValueSignatureByName(signatures, element)
		if index < 0 {
			index = functionValueSignatureByTypeText(signatures, element)
		}
		if index >= 0 {
			return index
		}
	}
	return -1
}

func functionValueIsDeclaredFunction(program *unit.Program, funcTok int) bool {
	for i := 0; i < len(program.Funcs); i++ {
		if program.Funcs[i].StartTok == funcTok {
			return true
		}
	}
	return false
}

func functionValueCaptures(program *unit.Program, literalStart int, bodyOpen int, bodyClose int, params []string) ([]string, []string) {
	var names []string
	var types []string
	for i := bodyOpen + 1; i < bodyClose; i++ {
		if program.Tokens[i].KindLine&255 != unit.TokenIdent {
			continue
		}
		name := functionValueTokenText(program, i)
		if name == "return" || name == "true" || name == "false" || name == "nil" || name == "bool" || ordinaryBuiltinTypeName(name) || functionValueNameInList(params, name) || functionValueNameInList(names, name) || !functionValueClosureValueReference(program, i) || functionValueClosureLocalName(program, bodyOpen, bodyClose, i) {
			continue
		}
		typ := functionValueLexicalLocalType(program, literalStart, name)
		if typ == "" {
			continue
		}
		names = append(names, name)
		types = append(types, typ)
	}
	return names, types
}

func functionValueEnclosingLocalType(program *unit.Program, before int, name string) string {
	return functionValueEnclosingLocalTypeDepth(program, before, name, 0)
}

func functionValueEnclosingLocalTypeDepth(program *unit.Program, before int, name string, depth int) string {
	return functionValueEnclosingLocalTypeDepthMode(program, before, name, depth, true)
}

func functionValueLexicalLocalType(program *unit.Program, before int, name string) string {
	return functionValueEnclosingLocalTypeDepthMode(program, before, name, 0, false)
}

func functionValueEnclosingLocalTypeDepthMode(program *unit.Program, before int, name string, depth int, allowGlobal bool) string {
	if depth > 16 {
		return ""
	}
	fnIndex := functionValueEnclosingFunc(program, before)
	if fnIndex < 0 {
		return ""
	}
	fn, _ := functionValueLexicalFunction(program, before)
	if fn.ReceiverStart < fn.ReceiverEnd {
		start := fn.ReceiverStart
		end := fn.ReceiverEnd
		if functionValueTokenEquals(program, start, "(") {
			start++
		}
		if functionValueTokenEquals(program, end-1, ")") {
			end--
		}
		if end-start >= 2 && functionValueTokenEquals(program, start, name) {
			return functionValueTokensText(program, start+1, end)
		}
	}
	// Search newest declarations first, including one immediately before use.
	for i := before - 1; i > fn.BodyStart; i-- {
		// Declarations inside a completed block cannot bind this use. Skip
		// its body, retaining control initializers before the opening brace.
		if i >= len(program.Tokens) {
			continue
		}
		item := &program.Tokens[i]
		if item.KindLine&255 == unit.TokenOp && item.Size == 1 && program.Text[item.Start] == '}' {
			open := functionValueFindMatchingBackward(program, i, "{", "}")
			if open > fn.BodyStart {
				i = open
				continue
			}
		}
		if i >= len(program.Tokens) || program.Tokens[i].KindLine&255 != unit.TokenIdent || program.Tokens[i].Size != len(name) || !functionValueTokenEquals(program, i, name) {
			continue
		}
		if functionValueLocalTypeName(program, i) && functionValueBindingInScope(program, &fn, i, before) {
			// A type in an inner block hides an outer function variable.
			return ""
		}
		short := functionValueTokenEquals(program, i+1, ":=") || functionValueTokenEquals(program, i+1, ",") || functionValueTokenEquals(program, i-1, ",")
		if !short && !functionValueTokenEquals(program, i-1, "var") {
			continue
		}
		if !functionValueBindingInScope(program, &fn, i, before) {
			continue
		}
		if short || functionValueTokenEquals(program, i+1, "=") {
			if typ := functionValueLocalCallResultType(program, i); typ != "" {
				return typ
			}
		}
		if short {
			if typ := functionRangeBindingType(program, i); typ != "" {
				return typ
			}
		}
		if functionValueTokenEquals(program, i+1, ":=") {
			if typ := functionValueTypeSwitchBindingType(program, i, before); typ != "" {
				return typ
			}
			rhs := i + 2
			rhsEnd := mapLowerAssignmentEnd(program, i+1)
			if functionValueTokenEquals(program, rhsEnd-1, ")") {
				open := functionValueFindMatchingBackward(program, rhsEnd-1, "(", ")")
				if typ := functionValueAnonymousTypeCallee(program, rhs, open); typ != "" {
					return typ
				}
			}
			// Indexing yields an element, while slicing preserves a container.
			// Infer the complete RHS before the identifier/parameter fallback.
			// Resolve its operands before this declaration so it cannot infer
			// its own binding recursively.
			if rhsEnd > rhs && (functionValueTokenEquals(program, rhsEnd-1, "]") || functionValueTokenEquals(program, rhsEnd-2, ".")) {
				if typ := ordinaryBuiltinExprType(program, i, rhs, rhsEnd); typ != "" {
					return typ
				}
			}
			if functionValueTokenEquals(program, rhs, "range") {
				end := concurrencyTopLevelToken(program, rhs+1, before, "{")
				typeEnd := functionValueTypeEnd(program, rhs+1)
				literalType := functionValueTokenEquals(program, rhs+1, "[") || functionValueTokenEquals(program, rhs+1, "map") || functionValueDeclaredType(program, functionValueTokenText(program, rhs+1))
				if literalType && typeEnd > rhs+1 && functionValueTokenEquals(program, typeEnd, "{") {
					end = functionValueFindMatchingBrace(program, typeEnd) + 1
				}
				if end > rhs+1 {
					typ := ordinaryUnderlyingType(program, ordinaryBuiltinExprType(program, rhs, rhs+1, end), 0)
					if functionValueTokenEquals(program, i-1, ",") {
						if typ == "string" {
							return "rune"
						}
						if len(typ) > 2 && typ[0] == '[' {
							for at := 1; at < len(typ); at++ {
								if typ[at] == ']' {
									return typ[at+1:]
								}
							}
						}
					} else if typ == "string" || len(typ) > 0 && typ[0] == '[' {
						return "int"
					}
				}
			}
			if (functionValueTokenEquals(program, rhs, "-") || functionValueTokenEquals(program, rhs, "+")) && rhs+1 < before {
				if program.Tokens[rhs+1].KindLine&255 == unit.TokenNumber {
					return "int"
				}
				if program.Tokens[rhs+1].KindLine&255 == unit.TokenFloat {
					return "float64"
				}
			}
			if functionValueTokenEquals(program, rhs, "&") && rhs+1 < before && program.Tokens[rhs+1].KindLine&255 == unit.TokenIdent {
				if rhsEnd == rhs+2 {
					if typ := functionValueEnclosingLocalTypeDepthMode(program, i, functionValueTokenText(program, rhs+1), depth+1, allowGlobal); typ != "" {
						return "*" + typ
					}
					if allowGlobal {
						if typ := ordinaryGlobalType(program, functionValueTokenText(program, rhs+1)); typ != "" {
							return "*" + typ
						}
					}
				}
				return "*" + functionValueTokenText(program, rhs+1)
			}
			typeEnd := functionValueTypeEnd(program, rhs)
			if typeEnd > rhs && functionValueTokenEquals(program, typeEnd, "{") {
				return functionValueTokensText(program, rhs, typeEnd)
			}
			if program.Tokens[rhs].KindLine&255 == unit.TokenNumber {
				return "int"
			}
			if program.Tokens[rhs].KindLine&255 == unit.TokenString {
				return "string"
			}
			if program.Tokens[rhs].KindLine&255 == unit.TokenIdent {
				if functionValueTokenEquals(program, rhs+1, ".") && rhs+3 < before &&
					(functionValueTokenEquals(program, rhs+3, ";") || functionValueTokenEquals(program, rhs+3, "}") || program.Tokens[rhs+3].KindLine>>8 != program.Tokens[rhs+2].KindLine>>8) {
					owner := functionValueEnclosingLocalTypeDepthMode(program, i, functionValueTokenText(program, rhs), depth+1, allowGlobal)
					if typ := functionValueStructFieldType(program, owner, functionValueTokenText(program, rhs+2)); typ != "" {
						return typ
					}
				}
				callName := ""
				if functionValueTokenEquals(program, rhs+1, "(") {
					callName = functionValueTokenText(program, rhs)
				} else if functionValueTokenEquals(program, rhs+1, ".") && rhs+3 < before && functionValueTokenEquals(program, rhs+3, "(") {
					callName = functionValueTokenText(program, rhs+2)
				}
				if callName != "" {
					if callName == "bool" && !ordinaryBuiltinShadowed(program, i, callName) {
						return "bool"
					}
					if ordinaryBuiltinTypeName(callName) {
						return callName
					}
					if callName == "make" {
						open := rhs + 1
						close := functionValueFindMatchingParen(program, open)
						starts, ends := ordinaryBuiltinArguments(program, open+1, close)
						if len(starts) > 0 {
							return ordinaryTypeArgumentText(program, starts[0], ends[0])
						}
					}
					if typ := functionValueDeclaredFunctionResultType(program, callName); typ != "" {
						return typ
					}
					if functionValueDeclaredType(program, callName) {
						return callName
					}
				}
				simpleIdent := rhs+1 >= len(program.Tokens) || functionValueTokenEquals(program, rhs+1, ";") || functionValueTokenEquals(program, rhs+1, "}") || program.Tokens[rhs+1].KindLine>>8 != program.Tokens[rhs].KindLine>>8
				if simpleIdent {
					if typ := functionValueEnclosingLocalTypeDepthMode(program, i, functionValueTokenText(program, rhs), depth+1, allowGlobal); typ != "" {
						return typ
					}
					if (functionValueTokenEquals(program, rhs, "true") || functionValueTokenEquals(program, rhs, "false")) && !ordinaryBuiltinShadowed(program, i, functionValueTokenText(program, rhs)) {
						return "bool"
					}
					if typ := functionValueDirectFunctionType(program, functionValueTokenText(program, rhs)); typ != "" {
						return typ
					}
				}
				if typ := functionValueFunctionParamType(program, &fn, functionValueTokenText(program, rhs)); typ != "" {
					return typ
				}
			}
		}
		if i > 0 && functionValueTokenEquals(program, i-1, "var") && i+1 < before {
			return functionValueTokensText(program, i+1, functionValueTypeEnd(program, i+1))
		}
	}
	if typ := functionValueFunctionParamType(program, &fn, name); typ != "" {
		return typ
	}
	if typ := functionValueNamedResultType(program, &fn, name); typ != "" {
		return typ
	}
	if fn.StartTok != program.Funcs[fnIndex].StartTok {
		return functionValueEnclosingLocalTypeDepthMode(program, fn.StartTok, name, depth+1, allowGlobal)
	}
	if allowGlobal {
		return ordinaryGlobalType(program, name)
	}
	// Package variables and declarations remain directly addressable from the
	// generated closure function. Only receiver, parameter, and enclosing-local
	// bindings belong in the persistent closure environment.
	return ""
}

func functionValueLocalTypeName(program *unit.Program, name int) bool {
	if functionValueTokenEquals(program, name-1, "type") {
		return true
	}
	if !functionValueTokenEquals(program, name+1, "=") && !functionValueTokenCanStartType(program, name+1) {
		return false
	}
	depth := 0
	for token := name - 1; token >= 0; token-- {
		item := &program.Tokens[token]
		if item.KindLine&255 != unit.TokenOp || item.Size != 1 {
			continue
		}
		c := program.Text[item.Start]
		if c == ')' || c == ']' || c == '}' {
			depth++
			continue
		}
		if c == '(' || c == '[' || c == '{' {
			if depth > 0 {
				depth--
				continue
			}
			return c == '(' && functionValueTokenEquals(program, token-1, "type")
		}
	}
	return false
}

// Resolve short declarations from the called function's parsed result list.
// This covers methods and each position of multi-result calls, without treating
// the comma-separated result list as one local's type.
func functionValueLocalCallResultType(program *unit.Program, binding int) string {
	first, last := binding, binding
	for first >= 2 && functionValueTokenEquals(program, first-1, ",") && program.Tokens[first-2].KindLine&255 == unit.TokenIdent {
		first -= 2
	}
	for last+2 < len(program.Tokens) && functionValueTokenEquals(program, last+1, ",") && program.Tokens[last+2].KindLine&255 == unit.TokenIdent {
		last += 2
	}
	if !functionValueTokenEquals(program, last+1, ":=") && !(functionValueTokenEquals(program, last+1, "=") && functionValueTokenEquals(program, first-1, "var")) {
		return ""
	}
	rhs := last + 2
	end := mapLowerAssignmentEnd(program, last+1)
	if last-first <= 2 {
		if typ := ordinaryBuiltinAssertionType(program, rhs, end); typ != "" {
			if binding == first {
				return typ
			}
			return "bool"
		}
	}
	if last > first {
		starts, ends := functionValueCommaParts(program, rhs, end)
		if len(starts) == (last-first)/2+1 {
			index := (binding - first) / 2
			start, stop := functionValueUnparen(program, starts[index], ends[index])
			typ := ordinaryBuiltinExprType(program, first, start, stop)
			if typ == "" && stop == start+1 && program.Tokens[start].KindLine&255 == unit.TokenIdent {
				typ = functionValueDirectFunctionType(program, functionValueTokenText(program, start))
			}
			return typ
		}
	}
	rhs, end = functionValueUnparen(program, rhs, end)
	if end == rhs+1 && program.Tokens[rhs].KindLine&255 == unit.TokenIdent {
		if typ := ordinaryBuiltinExprType(program, first, rhs, end); typ != "" {
			return typ
		}
		return functionValueDirectFunctionType(program, functionValueTokenText(program, rhs))
	}
	open := rhs + 1
	for functionValueTokenEquals(program, open, ".") && open+1 < len(program.Tokens) && program.Tokens[open+1].KindLine&255 == unit.TokenIdent {
		open += 2
	}
	if !functionValueTokenEquals(program, open, "(") {
		return ""
	}
	fn, ok := functionValueCalledFunction(program, open)
	if !ok {
		if open == rhs+1 && functionValueDeclaredType(program, functionValueTokenText(program, rhs)) && functionValueLexicalLocalType(program, first, functionValueTokenText(program, rhs)) == "" {
			return ""
		}
		// A callback field or variable has no function declaration of its own.
		// Its declared function type still determines every tuple result.
		typ := ordinaryUnderlyingType(program, ordinaryBuiltinExprType(program, first, rhs, open), 0)
		if !functionValueHasPrefix(functionValueCompactTypeText(typ), "func(") {
			return ""
		}
		source := []byte("package main\ntype Signature " + typ + "\n")
		signatureProgram := unit.Program{Package: "main"}
		if !reparseFunctionValueProgram(&signatureProgram, source, nil, len(source), -1) {
			return ""
		}
		for tok := 0; tok < len(signatureProgram.Tokens); tok++ {
			if !functionValueTokenEquals(&signatureProgram, tok, "func") {
				continue
			}
			sig, _, valid := parseFunctionValueSignature(&signatureProgram, tok, "")
			if valid && len(sig.resultTypes) == (last-first)/2+1 {
				return sig.resultTypes[(binding-first)/2]
			}
			return ""
		}
		return ""
	}
	paramClose := functionValueFindMatchingParen(program, fn.NameTok+1)
	start := paramClose + 1
	if start >= fn.BodyStart {
		return ""
	}
	var types []string
	if functionValueTokenEquals(program, start, "(") {
		end := functionValueFindMatchingParen(program, start)
		if end < start {
			return ""
		}
		types = functionValueTupleResultTypes(program, start+1, end)
	} else {
		end := functionValueTypeEnd(program, start)
		if end <= start {
			return ""
		}
		types = []string{functionValueTokensText(program, start, end)}
	}
	if len(types) != (last-first)/2+1 {
		return ""
	}
	return types[(binding-first)/2]
}

func functionValueBindingInScope(program *unit.Program, fn *unit.Func, binding int, before int) bool {
	// A binding cannot leave its lexical block without crossing a closing
	// brace or a new switch/select clause. Nearby uses need no whole-body scan.
	sameBlock := true
	for token := binding + 1; token < before; token++ {
		tok := &program.Tokens[token]
		kind := tok.KindLine & 255
		if kind == unit.TokenCase || kind == unit.TokenDefault || tok.Size == 1 && functionValueTokenEquals(program, token, "}") {
			sameBlock = false
			break
		}
	}
	if sameBlock {
		return true
	}
	for open := fn.BodyStart; open < binding; open++ {
		item := &program.Tokens[open]
		if item.KindLine&255 != unit.TokenOp || item.Size != 1 || program.Text[item.Start] != '{' {
			continue
		}
		close := functionValueFindMatchingBrace(program, open)
		if close > binding && before > close {
			return false
		}
		if close > binding && before <= close {
			// Each switch/select clause has its own implicit lexical block.
			bindingCase, referenceCase := -1, -1
			for scan := open + 1; scan < before; scan++ {
				if functionValueTokenEquals(program, scan, "{") {
					nested := functionValueFindMatchingBrace(program, scan)
					if nested > scan {
						scan = nested
						continue
					}
				}
				if functionValueTokenEquals(program, scan, "case") || functionValueTokenEquals(program, scan, "default") {
					referenceCase = scan
					if scan < binding {
						bindingCase = scan
					}
				}
			}
			if bindingCase >= 0 && bindingCase != referenceCase {
				return false
			}
		}
		if close < binding && close > open {
			open = close
		}
	}
	// Initializers have the control statement's implicit scope, not their
	// surrounding brace block. Skip composite literals while locating its body.
	owner := binding - 1
	if owner >= 0 && (functionValueTokenEquals(program, owner, "if") || functionValueTokenEquals(program, owner, "for") || functionValueTokenEquals(program, owner, "switch")) {
		for open := binding + 2; open < fn.BodyEnd; open++ {
			end := functionValueTypeEnd(program, open)
			if end > open && functionValueTokenEquals(program, end, "{") {
				literalType := functionValueTokenEquals(program, open, "map") || functionValueTokenEquals(program, open, "[") || functionValueTokenEquals(program, open, "struct") || functionValueDeclaredType(program, functionValueTokenText(program, open))
				if literalType {
					close := functionValueFindMatchingBrace(program, end)
					if close > end {
						open = close
						continue
					}
				}
			}
			if functionValueTokenEquals(program, open, "(") {
				close := functionValueFindMatchingParen(program, open)
				if close > open {
					open = close
					continue
				}
			}
			if !functionValueTokenEquals(program, open, "{") {
				continue
			}
			close := functionValueFindMatchingBrace(program, open)
			for close+1 < fn.BodyEnd && functionValueTokenEquals(program, close+1, "else") {
				next := concurrencyTopLevelToken(program, close+2, fn.BodyEnd, "{")
				if next < 0 {
					break
				}
				close = functionValueFindMatchingBrace(program, next)
			}
			return before <= close
		}
	}
	return true
}

func functionValueTypeSwitchBindingType(program *unit.Program, binding int, before int) string {
	if binding < 1 || binding+7 >= len(program.Tokens) || !functionValueTokenEquals(program, binding-1, "switch") {
		return ""
	}
	bodyOpen := -1
	for i := binding + 2; i < len(program.Tokens) && i < binding+16; i++ {
		if functionValueTokenEquals(program, i, "{") {
			bodyOpen = i
			break
		}
	}
	if bodyOpen < 0 || before <= bodyOpen {
		return ""
	}
	bodyClose := functionValueFindMatchingBrace(program, bodyOpen)
	if bodyClose < before {
		return ""
	}
	caseStart := -1
	depth := 0
	for i := bodyOpen + 1; i < before; i++ {
		text := functionValueTokenText(program, i)
		if text == "{" || text == "(" || text == "[" {
			depth++
			continue
		}
		if text == "}" || text == ")" || text == "]" {
			if depth > 0 {
				depth--
			}
			continue
		}
		if depth == 0 && text == "case" {
			caseStart = i + 1
		} else if depth == 0 && text == "default" {
			caseStart = -1
		}
	}
	if caseStart < 0 {
		return ""
	}
	caseEnd := caseStart
	for caseEnd < before && !functionValueTokenEquals(program, caseEnd, ":") {
		if functionValueTokenEquals(program, caseEnd, ",") {
			return ""
		}
		caseEnd++
	}
	if caseEnd >= before || functionValueTypeEnd(program, caseStart) != caseEnd {
		return ""
	}
	return functionValueTokensText(program, caseStart, caseEnd)
}

func functionValueDeclaredType(program *unit.Program, name string) bool {
	for i := 0; i < len(program.Decls); i++ {
		decl := program.Decls[i]
		if decl.Kind == unit.TokenType && ordinarySpanEquals(program.Text, decl.NameStart, decl.NameEnd, name) {
			return true
		}
	}
	return false
}

func functionValueNamedType(program *unit.Program, name string) bool {
	return functionValueDeclaredType(program, name) || ordinaryBuiltinTypeName(name) || name == "bool" || name == "error" || name == "any"
}

// Anonymous functions have lexical parameter/local scopes even though the
// compact unit's Funcs table contains only package-level declarations.
// Build the index when a linked program is created, outside lookup scratch
// marks. Lazy allocation in a resolver could outlive an arena rewind.
func indexFunctionValueLexicalScopes(program *unit.Program) {
	index := new([]unit.Func)
	// Token kinds cheaply reject ordinary source. Parse each literal once
	// instead of walking back through its enclosing body at every lookup.
	for at := 0; at < len(program.Tokens); at++ {
		if program.Tokens[at].KindLine&255 != unit.TokenFunc || !functionValueTokenEquals(program, at+1, "(") {
			continue
		}
		_, body, ok := parseFunctionValueSignature(program, at, "")
		if !ok || !functionValueTokenEquals(program, body, "{") {
			continue
		}
		close := functionValueFindMatchingBrace(program, body)
		if close >= body {
			*index = append(*index, unit.Func{StartTok: at, NameTok: at, BodyStart: body, BodyEnd: close + 1, EndTok: close + 1})
		}
	}
	program.LexicalFuncs = index
}

func functionValueLexicalFunction(program *unit.Program, token int) (unit.Func, bool) {
	index := functionValueEnclosingFunc(program, token)
	if index < 0 {
		return unit.Func{}, false
	}
	if program.LexicalFuncs == nil {
		fn := program.Funcs[index]
		for at := token - 1; at >= fn.BodyStart; at-- {
			if program.Tokens[at].KindLine&255 != unit.TokenFunc || !functionValueTokenEquals(program, at+1, "(") {
				continue
			}
			_, body, ok := parseFunctionValueSignature(program, at, "")
			if !ok || !functionValueTokenEquals(program, body, "{") || body >= token {
				continue
			}
			close := functionValueFindMatchingBrace(program, body)
			if close >= token {
				return unit.Func{StartTok: at, NameTok: at, BodyStart: body, BodyEnd: close + 1, EndTok: close + 1}, true
			}
		}
		return fn, true
	}
	literals := *program.LexicalFuncs
	low, high := 0, len(literals)
	for low < high {
		middle := low + (high-low)/2
		if literals[middle].StartTok < token {
			low = middle + 1
		} else {
			high = middle
		}
	}
	fn := program.Funcs[index]
	for at := low - 1; at >= 0; at-- {
		literal := &literals[at]
		if literal.StartTok < fn.BodyStart {
			break
		}
		if literal.BodyStart < token && token < literal.BodyEnd {
			return *literal, true
		}
	}
	return fn, true
}

func functionValueEnclosingFunc(program *unit.Program, token int) int {
	low := 0
	high := len(program.Funcs)
	for low < high {
		middle := low + (high-low)/2
		if program.Funcs[middle].BodyStart < token {
			low = middle + 1
		} else {
			high = middle
		}
	}
	if low > 0 {
		candidate := low - 1
		if token < program.Funcs[candidate].BodyEnd {
			return candidate
		}
	}
	return -1
}

func functionValueDeclaredFunctionResultType(program *unit.Program, name string) string {
	for i := 0; i < len(program.Funcs); i++ {
		fn := &program.Funcs[i]
		if fn.ReceiverStart < fn.ReceiverEnd || !functionValueTokenEquals(program, fn.NameTok, name) {
			continue
		}
		return functionValueDeclaredResultType(program, fn)
	}
	return ""
}

func functionValueDeclaredResultType(program *unit.Program, fn *unit.Func) string {
	open := fn.NameTok + 1
	if !functionValueTokenEquals(program, open, "(") {
		return ""
	}
	close := functionValueFindMatchingParen(program, open)
	resultStart := close + 1
	if resultStart >= fn.BodyStart {
		return ""
	}
	if functionValueTokenEquals(program, resultStart, "(") {
		resultEnd := functionValueFindMatchingParen(program, resultStart)
		if resultEnd <= resultStart {
			return ""
		}
		return functionValueTokensText(program, resultStart+1, resultEnd)
	}
	return functionValueTokensText(program, resultStart, fn.BodyStart)
}

func functionValueFunctionParamType(program *unit.Program, fn *unit.Func, name string) string {
	open := fn.NameTok + 1
	if !functionValueTokenEquals(program, open, "(") {
		return ""
	}
	close := functionValueFindMatchingParen(program, open)
	_, names, _, ok := normalizedFunctionValueParams(program, open+1, close)
	if !ok {
		return ""
	}
	types := functionValueFunctionParamTypes(program, fn)
	for i := 0; i < len(names) && i < len(types); i++ {
		if names[i] == name {
			if functionValueHasPrefix(types[i], "...") {
				return "[]" + types[i][3:]
			}
			return types[i]
		}
	}
	return ""
}

func functionValueNamedResultType(program *unit.Program, fn *unit.Func, name string) string {
	open := functionValueFindMatchingParen(program, fn.NameTok+1) + 1
	if !functionValueTokenEquals(program, open, "(") {
		return ""
	}
	close := functionValueFindMatchingParen(program, open)
	starts, ends := functionValueCommaParts(program, open+1, close)
	named := false
	for i := 0; i < len(starts); i++ {
		if starts[i]+1 < ends[i] && program.Tokens[starts[i]].KindLine&255 == unit.TokenIdent && functionValueTypeEnd(program, starts[i]+1) == ends[i] {
			named = true
		}
	}
	if !named {
		return ""
	}
	_, names, types, ok := normalizedFunctionValueParams(program, open+1, close)
	if ok {
		for i := 0; i < len(names); i++ {
			if names[i] == name && name != "_" {
				return types[i]
			}
		}
	}
	return ""
}

func functionValueClosureBody(program *unit.Program, bodyOpen int, bodyClose int, captures []string, pending []functionValueEdit) (string, bool) {
	if bodyOpen+1 >= bodyClose {
		return "", true
	}
	start := program.Tokens[bodyOpen].Start + program.Tokens[bodyOpen].Size
	end := program.Tokens[bodyClose].Start
	src := program.Text[start:end]
	var bodyEdits []functionValueEdit
	for _, edit := range pending {
		if edit.start >= start && edit.end <= end {
			bodyEdits = append(bodyEdits, functionValueEdit{start: edit.start - start, end: edit.end - start, text: edit.text})
		}
	}
	transformed, ok := applyFunctionValueEdits(src, bodyEdits)
	if !ok {
		return "", false
	}
	// Capture accesses must also be rewritten inside replacement expressions
	// (such as callback dispatch arguments), so tokenize the transformed body.
	prefix := "package main\nfunc body() {"
	source := []byte(prefix + string(transformed) + "}\n")
	local := unit.Program{Package: "main"}
	if !reparseFunctionValueProgram(&local, source, nil, len(source), -1) {
		return "", false
	}
	program = &local
	start = len(prefix)
	end = start + len(transformed)
	var edits []functionValueEdit
	for i := 0; i < len(program.Tokens); i++ {
		if program.Tokens[i].Start < start || program.Tokens[i].Start >= end {
			continue
		}
		name := functionValueTokenText(program, i)
		if !functionValueNameInList(captures, name) || !functionValueClosureValueReference(program, i) || functionValueClosureLocalName(program, program.Funcs[0].BodyStart, program.Funcs[0].BodyEnd-1, i) {
			continue
		}
		tok := program.Tokens[i]
		edits = append(edits, functionValueEdit{start: tok.Start - start, end: tok.Start + tok.Size - start, text: "(*env." + name + ")"})
	}
	out, ok := applyFunctionValueEdits(transformed, edits)
	if !ok {
		return "", false
	}
	return string(out), true
}

// A declaration inside the literal shadows an outer capture only after its
// initializer, and only within its lexical block/control-statement scope.
func functionValueClosureLocalName(program *unit.Program, bodyOpen int, bodyClose int, token int) bool {
	name := functionValueTokenText(program, token)
	fn := unit.Func{BodyStart: bodyOpen, BodyEnd: bodyClose + 1}
	for candidate := bodyOpen + 1; candidate <= token; candidate++ {
		if !functionValueTokenEquals(program, candidate, name) {
			continue
		}
		first, last := candidate, candidate
		for first > bodyOpen+2 && functionValueTokenEquals(program, first-1, ",") && program.Tokens[first-2].KindLine&255 == unit.TokenIdent {
			first -= 2
		}
		for last+2 < bodyClose && functionValueTokenEquals(program, last+1, ",") && program.Tokens[last+2].KindLine&255 == unit.TokenIdent {
			last += 2
		}
		short := functionValueTokenEquals(program, last+1, ":=")
		declared := functionValueTokenEquals(program, first-1, "var") || functionValueTokenEquals(program, first-1, "const")
		if !short && !declared {
			continue
		}
		if candidate == token {
			return true
		}
		activation := mapLowerAssignmentEnd(program, last+1)
		if short && functionValueTokenEquals(program, last+2, "range") {
			start := last + 3
			activation = concurrencyTopLevelToken(program, start, bodyClose, "{")
			typeEnd := functionValueTypeEnd(program, start)
			literalType := functionValueTokenEquals(program, start, "[") || functionValueTokenEquals(program, start, "map") || functionValueDeclaredType(program, functionValueTokenText(program, start))
			if literalType && typeEnd == activation {
				activation = concurrencyTopLevelToken(program, functionValueFindMatchingBrace(program, activation)+1, bodyClose, "{")
			}
		}
		if activation >= 0 && token >= activation && functionValueBindingInScope(program, &fn, first, token) {
			return true
		}
	}
	return false
}

func functionValueClosureValueReference(program *unit.Program, tok int) bool {
	if tok > 0 && functionValueTokenEquals(program, tok-1, ".") {
		return false
	}
	// A keyed composite-literal field names storage; it does not refer to an
	// enclosing local with the same spelling.
	if functionValueTokenEquals(program, tok+1, ":") {
		return false
	}
	return true
}

func appendFunctionValuePackageEdits(program *unit.Program, edits []functionValueEdit) []functionValueEdit {
	seen := false
	for i := 0; i+1 < len(program.Tokens); i++ {
		if !functionValueTokenEquals(program, i, "package") {
			continue
		}
		start := program.Tokens[i].Start
		end := program.Tokens[i+1].Start + program.Tokens[i+1].Size
		if !seen {
			edits = append(edits, functionValueEdit{start: start, end: end, text: "package main"})
			seen = true
		} else {
			edits = append(edits, functionValueEdit{start: start, end: end, text: ""})
		}
	}
	return edits
}

func reparseFunctionValueProgram(original *unit.Program, text []byte, edits []functionValueEdit, originalLength int, generatedStart int) bool {
	return reparseFunctionValueProgramMode(original, text, edits, originalLength, generatedStart, false)
}

func reparseFunctionValueProgramMode(original *unit.Program, text []byte, edits []functionValueEdit, originalLength int, generatedStart int, reuse bool) bool {
	parseMark := arena.Mark()
	file, lineStarts := syntax.ParseLinkedFile(text)
	if !file.Ok {
		return false
	}
	out := unit.Program{Package: original.Package, ImportPath: original.ImportPath, Text: text}
	count := len(file.Tokens)
	for _, tok := range file.Tokens {
		if functionValueTokenIsEllipsis(text, tok) {
			count += 2
		}
	}
	declCount, funcCount := len(file.Decls), len(file.Funcs)
	// Reserve the retained tables before temporary syntax metadata. Arena builds
	// repeat the parse after sizing, allowing all parse scratch to be reclaimed.
	if parseMark != 0 {
		arena.Reset(parseMark)
	}
	if reuse && count <= cap(original.Tokens) {
		out.Tokens = original.Tokens[:0]
	} else {
		out.Tokens = make([]unit.Token, 0, count)
	}
	out.Decls = make([]unit.Decl, 0, declCount)
	out.Funcs = make([]unit.Func, 0, funcCount)
	scratchMark := arena.Mark()
	if parseMark != 0 {
		file, lineStarts = syntax.ParseLinkedFile(text)
		if !file.Ok {
			return false
		}
	}
	tokenMap := make([]int, len(file.Tokens)+1)
	for i := 0; i < len(file.Tokens); i++ {
		tok := file.Tokens[i]
		tokenMap[i] = len(out.Tokens)
		kind := functionValueUnitTokenKind(text, tok)
		if functionValueTokenIsEllipsis(text, tok) {
			for dot := 0; dot < 3; dot++ {
				out.Tokens = append(out.Tokens, unit.MakeToken(kind, syntax.TokenStart(tok)+dot, 1, syntax.TokenLineAt(&file, i, lineStarts)))
			}
		} else {
			out.Tokens = append(out.Tokens, unit.MakeToken(kind, syntax.TokenStart(tok), syntax.TokenSize(tok), syntax.TokenLineAt(&file, i, lineStarts)))
		}
	}
	tokenMap[len(file.Tokens)] = len(out.Tokens)
	eof := len(out.Tokens) - 1
	for i := 0; i < len(file.Decls); i++ {
		decl := file.Decls[i]
		name := file.Tokens[decl.NameTok]
		nameStart := syntax.TokenStart(name)
		start := decl.StartTok
		if start > 0 && file.Tokens[start-1].KindLine&255 == decl.Kind {
			start--
		}
		out.Decls = append(out.Decls, unit.Decl{Kind: functionValueDeclKind(decl.Kind), NameStart: nameStart, NameEnd: nameStart + syntax.TokenSize(name), StartTok: tokenMap[start], EndTok: tokenMap[decl.EndTok]})
	}
	for i := 0; i < len(file.Funcs); i++ {
		fn := file.Funcs[i]
		name := file.Tokens[fn.NameTok]
		receiverStart := fn.ReceiverStart
		receiverEnd := fn.ReceiverEnd
		if receiverStart < 0 {
			receiverStart = eof
			receiverEnd = eof
		} else {
			receiverStart = tokenMap[receiverStart]
			receiverEnd = tokenMap[receiverEnd]
		}
		nameStart := syntax.TokenStart(name)
		out.Funcs = append(out.Funcs, unit.Func{NameStart: nameStart, NameEnd: nameStart + syntax.TokenSize(name), StartTok: tokenMap[fn.StartTok], NameTok: tokenMap[fn.NameTok], ReceiverStart: receiverStart, ReceiverEnd: receiverEnd, BodyStart: tokenMap[fn.BodyStart], BodyEnd: tokenMap[fn.BodyEnd], EndTok: tokenMap[fn.EndTok]})
	}
	if parseMark != 0 {
		arena.Reset(scratchMark)
	}
	out.Packages = remapFunctionValuePackages(original, &out, edits, originalLength, generatedStart)
	replaceFunctionValueProgram(original, &out)
	return true
}

func remapFunctionValuePackages(original *unit.Program, reparsed *unit.Program, edits []functionValueEdit, originalLength int, generatedStart int) []unit.PackageInfo {
	if len(original.Packages) == 0 {
		return nil
	}
	packages := make([]unit.PackageInfo, 0, len(original.Packages)+1)
	root := -1
	for i := 0; i < len(original.Packages); i++ {
		item := original.Packages[i]
		item.TextStart = mapFunctionValueOffset(item.TextStart, edits, originalLength)
		item.TextEnd = mapFunctionValueOffset(item.TextEnd, edits, originalLength)
		setFunctionValuePackageTableRanges(&item, reparsed)
		if item.ImportPath == original.ImportPath {
			root = len(packages)
		}
		packages = append(packages, item)
	}
	if generatedStart >= 0 && generatedStart < len(reparsed.Text) && root >= 0 {
		if packages[root].TextEnd == generatedStart {
			packages[root].TextEnd = len(reparsed.Text)
			setFunctionValuePackageTableRanges(&packages[root], reparsed)
		} else {
			generated := packages[root]
			generated.TextStart = generatedStart
			generated.TextEnd = len(reparsed.Text)
			setFunctionValuePackageTableRanges(&generated, reparsed)
			packages = append(packages, generated)
		}
	}
	return packages
}

func mapFunctionValueOffset(position int, edits []functionValueEdit, sourceLength int) int {
	if position < 0 {
		return 0
	}
	if position > sourceLength {
		position = sourceLength
	}
	delta := 0
	for i := 0; i < len(edits); i++ {
		edit := edits[i]
		if position < edit.start {
			break
		}
		if position < edit.end {
			return edit.start + delta
		}
		delta += len(edit.text) - (edit.end - edit.start)
	}
	return position + delta
}

func setFunctionValuePackageTableRanges(item *unit.PackageInfo, program *unit.Program) {
	item.TokenStart, item.TokenEnd = functionValueTokenRangeForText(program.Tokens, item.TextStart, item.TextEnd)
	item.DeclStart, item.DeclEnd = functionValueDeclRangeForText(program.Decls, item.TextStart, item.TextEnd)
	item.FuncStart, item.FuncEnd = functionValueFuncRangeForText(program.Funcs, item.TextStart, item.TextEnd)
}

func functionValueTokenRangeForText(items []unit.Token, start int, end int) (int, int) {
	// Token offsets are ordered. Package remapping must not scan the whole
	// linked token stream again for every package.
	lo, hi := 0, len(items)
	for lo < hi {
		mid := lo + (hi-lo)/2
		if items[mid].Start < start {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	first := lo
	if first == len(items) || items[first].Start >= end {
		return len(items), len(items)
	}
	hi = len(items)
	for lo < hi {
		mid := lo + (hi-lo)/2
		if items[mid].Start < end {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return first, lo
}

func functionValueDeclRangeForText(items []unit.Decl, start int, end int) (int, int) {
	first := len(items)
	last := len(items)
	for i := 0; i < len(items); i++ {
		if items[i].NameStart >= start && items[i].NameStart < end {
			if first == len(items) {
				first = i
			}
			last = i + 1
		}
	}
	return first, last
}

func functionValueFuncRangeForText(items []unit.Func, start int, end int) (int, int) {
	first := len(items)
	last := len(items)
	for i := 0; i < len(items); i++ {
		if items[i].NameStart >= start && items[i].NameStart < end {
			if first == len(items) {
				first = i
			}
			last = i + 1
		}
	}
	return first, last
}

func functionValueTokenIsEllipsis(src []byte, tok syntax.Token) bool {
	start := syntax.TokenStart(tok)
	end := syntax.TokenEnd(tok)
	return tok.KindLine&255 == syntax.TokenOperator && end-start == 3 && start >= 0 && end <= len(src) && src[start] == '.' && src[start+1] == '.' && src[start+2] == '.'
}

func functionValueUnitTokenKind(src []byte, tok syntax.Token) int {
	if tok.KindLine&255 == syntax.TokenEOF {
		return unit.TokenEOF
	}
	if tok.KindLine&255 == syntax.TokenIdent {
		return unit.TokenIdent
	}
	if tok.KindLine&255 == syntax.TokenNumber {
		if syntax.NumberTokenIsFloat(src, tok) {
			return unit.TokenFloat
		}
		return unit.TokenNumber
	}
	if tok.KindLine&255 == syntax.TokenString {
		return unit.TokenString
	}
	if tok.KindLine&255 == syntax.TokenChar {
		return unit.TokenChar
	}
	if tok.KindLine&255 == syntax.TokenOperator {
		return unit.TokenOp
	}
	if tok.KindLine&255 == syntax.TokenPackage {
		return unit.TokenPackage
	}
	if tok.KindLine&255 == syntax.TokenConst {
		return unit.TokenConst
	}
	if tok.KindLine&255 == syntax.TokenVar {
		return unit.TokenVar
	}
	if tok.KindLine&255 == syntax.TokenType {
		return unit.TokenType
	}
	if tok.KindLine&255 == syntax.TokenFunc {
		return unit.TokenFunc
	}
	if tok.KindLine&255 == syntax.TokenStruct {
		return unit.TokenStruct
	}
	if tok.KindLine&255 == syntax.TokenReturn {
		return unit.TokenReturn
	}
	if tok.KindLine&255 == syntax.TokenIf {
		return unit.TokenIf
	}
	if tok.KindLine&255 == syntax.TokenElse {
		return unit.TokenElse
	}
	if tok.KindLine&255 == syntax.TokenFor {
		return unit.TokenFor
	}
	if tok.KindLine&255 == syntax.TokenBreak {
		return unit.TokenBreak
	}
	if tok.KindLine&255 == syntax.TokenContinue {
		return unit.TokenContinue
	}
	if tok.KindLine&255 == syntax.TokenGoto {
		return unit.TokenGoto
	}
	if tok.KindLine&255 == syntax.TokenSwitch {
		return unit.TokenSwitch
	}
	if tok.KindLine&255 == syntax.TokenCase {
		return unit.TokenCase
	}
	if tok.KindLine&255 == syntax.TokenDefault {
		return unit.TokenDefault
	}
	return unit.TokenIdent
}

func functionValueDeclKind(kind int) int {
	if kind == syntax.TokenConst {
		return unit.TokenConst
	}
	if kind == syntax.TokenVar {
		return unit.TokenVar
	}
	return unit.TokenType
}

func functionValueMethodReceiverType(program *unit.Program, method string) string {
	for i := 0; i < len(program.Funcs); i++ {
		fn := &program.Funcs[i]
		if !functionValueTokenEquals(program, fn.NameTok, method) || fn.ReceiverStart >= fn.ReceiverEnd {
			continue
		}
		start := fn.ReceiverStart
		end := fn.ReceiverEnd
		if functionValueTokenEquals(program, start, "(") {
			start++
		}
		if functionValueTokenEquals(program, end-1, ")") {
			end--
		}
		if end-start >= 2 && program.Tokens[start].KindLine&255 == unit.TokenIdent {
			start++
		}
		return functionValueTokensText(program, start, end)
	}
	return ""
}

func functionValueMethodReceiverTypeForBase(program *unit.Program, at int, base string, method string) string {
	baseType := functionValueEnclosingLocalType(program, at, base)
	if baseType == "" {
		return functionValueMethodReceiverType(program, method)
	}
	if functionRangeInterfaceMethodType(program, baseType, method) != "" {
		return baseType
	}
	for i := 0; i < len(program.Funcs); i++ {
		fn := &program.Funcs[i]
		if !functionValueTokenEquals(program, fn.NameTok, method) || fn.ReceiverStart >= fn.ReceiverEnd {
			continue
		}
		receiverType := functionValueReceiverType(program, fn)
		if functionValueTypeEmbeds(program, baseType, receiverType, 0) {
			if functionValueBareType(baseType) == functionValueBareType(receiverType) && functionValueHasPrefix(receiverType, "*") && !functionValueHasPrefix(baseType, "*") {
				return "*" + baseType
			}
			return baseType
		}
	}
	return ""
}

func functionValueReceiverType(program *unit.Program, fn *unit.Func) string {
	start := fn.ReceiverStart
	end := fn.ReceiverEnd
	if functionValueTokenEquals(program, start, "(") {
		start++
	}
	if functionValueTokenEquals(program, end-1, ")") {
		end--
	}
	if end-start >= 2 && program.Tokens[start].KindLine&255 == unit.TokenIdent {
		start++
	}
	return functionValueTokensText(program, start, end)
}

func functionValueSelectorFieldBefore(program *unit.Program, before int) int {
	end := functionValuePrimaryTokenBefore(program, before)
	if end < 2 || !functionValueTokenEquals(program, end-1, ".") {
		return -1
	}
	return end
}

func functionValuePrimaryTokenBefore(program *unit.Program, before int) int {
	end := before - 1
	for end >= 0 && functionValueTokenEquals(program, end, ")") {
		open := functionValueFindMatchingBackward(program, end, "(", ")")
		if open < 0 {
			return -1
		}
		if open > 0 && program.Tokens[open-1].KindLine>>8 == program.Tokens[open].KindLine>>8 &&
			(program.Tokens[open-1].KindLine&255 == unit.TokenIdent ||
				functionValueTokenEquals(program, open-1, "]") || functionValueTokenEquals(program, open-1, ")")) {
			return end
		}
		end--
	}
	return end
}

func functionValueSelectorStart(program *unit.Program, end int) int {
	start := end
	for start >= 2 && functionValueTokenEquals(program, start-1, ".") {
		start = functionValuePrimaryStart(program, start-2)
		if start < 0 {
			return -1
		}
	}
	return start
}

func functionValuePrimaryStart(program *unit.Program, end int) int {
	if end < 0 || end >= len(program.Tokens) {
		return -1
	}
	start := end
	if functionValueTokenEquals(program, end, "}") {
		body := functionValueFindMatchingBackward(program, end, "{", "}")
		for candidate := body - 1; candidate >= 0; candidate-- {
			if functionValueTokenEquals(program, candidate, ";") || functionValueTokenEquals(program, candidate, "{") || functionValueTokenEquals(program, candidate, "}") {
				break
			}
			if functionValueTokenEquals(program, candidate, "func") {
				_, signatureEnd, ok := parseFunctionValueSignature(program, candidate, "")
				if ok && signatureEnd == body {
					start = candidate
					break
				}
			}
		}
	} else if functionValueTokenEquals(program, end, "]") {
		open := functionValueFindMatchingBackward(program, end, "[", "]")
		if open <= 0 {
			return -1
		}
		start = functionValuePrimaryStart(program, open-1)
	} else if functionValueTokenEquals(program, end, ")") {
		open := functionValueFindMatchingBackward(program, end, "(", ")")
		if open < 0 {
			return -1
		}
		start = open
		if open > 0 && program.Tokens[open-1].KindLine>>8 == program.Tokens[open].KindLine>>8 &&
			(program.Tokens[open-1].KindLine&255 == unit.TokenIdent || functionValueTokenEquals(program, open-1, "]") || functionValueTokenEquals(program, open-1, ")")) {
			start = functionValuePrimaryStart(program, open-1)
		}
	}
	for start >= 2 && functionValueTokenEquals(program, start-1, ".") {
		start = functionValuePrimaryStart(program, start-2)
		if start < 0 {
			return -1
		}
	}
	return start
}

func functionValueFindMatchingBackward(program *unit.Program, close int, openText string, closeText string) int {
	if len(openText) == 1 && len(closeText) == 1 {
		depth := 0
		for i := close; i >= 0; i-- {
			item := &program.Tokens[i]
			if item.KindLine&255 != unit.TokenOp || item.Size != 1 {
				continue
			}
			c := program.Text[item.Start]
			if c == closeText[0] {
				depth++
			} else if c == openText[0] {
				depth--
				if depth == 0 {
					return i
				}
			}
		}
		return -1
	}
	depth := 0
	for i := close; i >= 0; i-- {
		if functionValueTokenEquals(program, i, closeText) {
			depth++
		} else if functionValueTokenEquals(program, i, openText) {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func functionValueUniqueFieldByName(fields []functionValueField, name string) int {
	match := -1
	for i := 0; i < len(fields); i++ {
		if fields[i].name != name {
			continue
		}
		if match >= 0 {
			return -1
		}
		match = i
	}
	return match
}

func functionValueCompositeOwner(program *unit.Program, before int) string {
	depth := 0
	for i := before - 1; i >= 1; i-- {
		if functionValueTokenEquals(program, i, "}") {
			depth++
			continue
		}
		if !functionValueTokenEquals(program, i, "{") {
			continue
		}
		if depth > 0 {
			depth--
			continue
		}
		if program.Tokens[i-1].KindLine&255 == unit.TokenIdent {
			return functionValueTokenText(program, i-1)
		}
		return ""
	}
	return ""
}

func functionValueFieldByOwnerAndName(fields []functionValueField, owner string, name string) int {
	for i := 0; i < len(fields); i++ {
		if fields[i].owner == owner && fields[i].name == name {
			return i
		}
	}
	return -1
}

func functionValueFieldForSelector(program *unit.Program, fieldTok int, fields []functionValueField) int {
	if fieldTok < 2 {
		return -1
	}
	name := functionValueTokenText(program, fieldTok)
	possible := false
	for i := 0; i < len(fields); i++ {
		if fields[i].name == name {
			possible = true
			break
		}
	}
	if !possible {
		return -1
	}
	selectorStart := functionValueSelectorStart(program, fieldTok)
	if selectorStart < 0 || selectorStart >= fieldTok {
		return -1
	}
	baseType := ordinaryBuiltinExprType(program, fieldTok, selectorStart, fieldTok-1)
	pathStart := fieldTok - 1
	if baseType == "" {
		baseName := functionValueTokenText(program, selectorStart)
		pathStart = selectorStart + 1
		baseType = functionValueEnclosingLocalType(program, fieldTok, baseName)
	}
	baseType = functionValueBareType(baseType)
	if baseType == "" {
		return functionValueUniqueFieldByName(fields, name)
	}
	for i := pathStart; i < fieldTok-1; {
		if functionValueTokenEquals(program, i, "[") {
			close := functionValueFindMatching(program, i, "[", "]")
			if close < 0 || close >= fieldTok {
				return -1
			}
			baseType = functionValueElementType(baseType)
			i = close + 1
			continue
		}
		if functionValueTokenEquals(program, i, ".") && i+1 < fieldTok {
			baseType = functionValueBareType(functionValueStructFieldType(program, baseType, functionValueTokenText(program, i+1)))
			if baseType == "" {
				return -1
			}
			i += 2
			continue
		}
		return -1
	}
	// A method declared on the selected type wins over a field promoted from
	// an embedded struct at a greater selector depth. Treating the promoted
	// callback as the selected member rewrites an ordinary method call into a
	// call through the embedded function field.
	if functionValueTypeHasDirectMethod(program, baseType, name) {
		return -1
	}
	for i := 0; i < len(fields); i++ {
		if fields[i].name == name && functionValueTypeEmbeds(program, baseType, fields[i].owner, 0) {
			return i
		}
	}
	return -1
}

func functionValueTypeHasDirectMethod(program *unit.Program, typ string, name string) bool {
	typ = functionValueBareType(typ)
	for i := 0; i < len(program.Funcs); i++ {
		fn := &program.Funcs[i]
		if fn.ReceiverStart >= fn.ReceiverEnd || !functionValueTokenEquals(program, fn.NameTok, name) {
			continue
		}
		if functionValueBareType(functionValueReceiverType(program, fn)) == typ {
			return true
		}
	}
	return false
}

func functionValueElementType(typ string) string {
	for len(typ) > 0 && typ[0] == '*' {
		typ = typ[1:]
	}
	if len(typ) >= 2 && typ[0] == '[' && typ[1] == ']' {
		return functionValueBareType(typ[2:])
	}
	if len(typ) > 1 && typ[0] == '[' {
		for i := 1; i < len(typ); i++ {
			if typ[i] == ']' {
				return functionValueBareType(typ[i+1:])
			}
		}
	}
	return ""
}

func functionValueStructFieldType(program *unit.Program, owner string, fieldName string) string {
	owner = functionValueBareType(owner)
	if functionValueHasPrefix(owner, "struct") {
		// Local anonymous aggregates must retain local names and array bounds.
		// Inspect their field syntax without hoisting the type out of scope.
		source := []byte("package main\ntype __renvo_field_owner " + owner + "\n")
		parsed := unit.Program{Package: "main"}
		if reparseFunctionValueProgram(&parsed, source, nil, len(source), -1) {
			return functionValueStructFieldType(&parsed, "__renvo_field_owner", fieldName)
		}
		return ""
	}
	for i := 0; i < len(program.Decls); i++ {
		decl := program.Decls[i]
		if decl.Kind != unit.TokenType {
			continue
		}
		nameTok := functionValueTokenAtSpan(program, decl.NameStart, decl.NameEnd)
		if nameTok < 0 || !functionValueTokenEquals(program, nameTok, owner) {
			continue
		}
		start := nameTok + 1
		if !functionValueTokenEquals(program, start, "struct") || !functionValueTokenEquals(program, start+1, "{") {
			return ""
		}
		close := functionValueFindMatchingBrace(program, start+1)
		j := start + 2
		for j < close {
			for j < close && functionValueTokenEquals(program, j, ";") {
				j++
			}
			if j >= close {
				break
			}
			lineEnd := j + 1
			for lineEnd < close && !functionValueTokenEquals(program, lineEnd, ";") && program.Tokens[lineEnd].KindLine>>8 == program.Tokens[j].KindLine>>8 {
				lineEnd++
			}
			fieldEnd := lineEnd
			if fieldEnd > j && program.Tokens[fieldEnd-1].KindLine&255 == unit.TokenString {
				fieldEnd--
			}
			if functionValueTokenEquals(program, j, fieldName) && j+1 < fieldEnd {
				return functionValueTokensText(program, j+1, fieldEnd)
			}
			typeEnd := functionValueTypeEnd(program, j)
			if typeEnd == fieldEnd {
				nameTok := j
				if functionValueTokenEquals(program, nameTok, "*") {
					nameTok++
				}
				if functionValueTokenEquals(program, nameTok+1, ".") && nameTok+2 < lineEnd {
					nameTok += 2
				}
				if functionValueTokenEquals(program, nameTok, fieldName) {
					return functionValueTokensText(program, j, fieldEnd)
				}
			}
			j = lineEnd
		}
		return ""
	}
	return ""
}

func functionValueBareType(typ string) string {
	for len(typ) > 0 && typ[0] == '*' {
		typ = typ[1:]
	}
	lastDot := -1
	for i := 0; i < len(typ); i++ {
		if typ[i] == '.' {
			lastDot = i
		}
	}
	if lastDot >= 0 {
		return typ[lastDot+1:]
	}
	return typ
}

func functionValueTypeEmbeds(program *unit.Program, actual string, wanted string, depth int) bool {
	actual = functionValueBareType(actual)
	wanted = functionValueBareType(wanted)
	if actual == wanted {
		return true
	}
	if depth >= 8 {
		return false
	}
	for i := 0; i < len(program.Decls); i++ {
		decl := program.Decls[i]
		if decl.Kind != unit.TokenType {
			continue
		}
		nameTok := functionValueTokenAtSpan(program, decl.NameStart, decl.NameEnd)
		if nameTok < 0 || !functionValueTokenEquals(program, nameTok, actual) {
			continue
		}
		start := nameTok + 1
		if !functionValueTokenEquals(program, start, "struct") || !functionValueTokenEquals(program, start+1, "{") {
			return false
		}
		close := functionValueFindMatchingBrace(program, start+1)
		for j := start + 2; j < close; j++ {
			if j > start+2 && !functionValueTokenEquals(program, j-1, ";") && program.Tokens[j-1].KindLine>>8 == program.Tokens[j].KindLine>>8 {
				continue
			}
			embeddedStart := j
			if functionValueTokenEquals(program, j, "*") {
				embeddedStart++
			}
			if program.Tokens[embeddedStart].KindLine&255 != unit.TokenIdent {
				continue
			}
			embeddedEnd := embeddedStart + 1
			if functionValueTokenEquals(program, embeddedEnd, ".") && embeddedEnd+1 < close && program.Tokens[embeddedEnd+1].KindLine&255 == unit.TokenIdent {
				embeddedEnd += 2
			}
			if embeddedEnd < close && !functionValueTokenEquals(program, embeddedEnd, ";") && program.Tokens[embeddedEnd].KindLine>>8 == program.Tokens[embeddedStart].KindLine>>8 {
				continue
			}
			embeddedType := functionValueBareType(functionValueTokensText(program, j, embeddedEnd))
			if functionValueTypeEmbeds(program, embeddedType, wanted, depth+1) {
				return true
			}
		}
		return false
	}
	return false
}

func functionValueSignatureByName(signatures []functionValueSignature, name string) int {
	for i := 0; i < len(signatures); i++ {
		if signatures[i].name == name {
			return i
		}
	}
	return -1
}

func functionValueSignatureByShape(signatures []functionValueSignature, candidate functionValueSignature) int {
	for i := 0; i < len(signatures); i++ {
		sig := signatures[i]
		if !sig.anonymous || !functionValueSameShape(sig, candidate) {
			continue
		}
		return i
	}
	return -1
}

func functionValueNamedSignatureByShape(signatures []functionValueSignature, candidate functionValueSignature) int {
	for i := 0; i < len(signatures); i++ {
		if !signatures[i].anonymous && functionValueSameShape(signatures[i], candidate) {
			return i
		}
	}
	return -1
}

func functionValueSameShape(sig functionValueSignature, candidate functionValueSignature) bool {
	if len(sig.paramTypes) != len(candidate.paramTypes) || len(sig.resultTypes) != len(candidate.resultTypes) {
		return false
	}
	equal := true
	for j := 0; j < len(sig.paramTypes); j++ {
		if !functionValueSameNestedType(sig.paramTypes[j], candidate.paramTypes[j]) {
			equal = false
			break
		}
	}
	for j := 0; equal && j < len(sig.resultTypes); j++ {
		if !functionValueSameNestedType(sig.resultTypes[j], candidate.resultTypes[j]) {
			equal = false
		}
	}
	return equal
}

// Parameter names do not contribute to nested function type identity either.
// Compare parsed function signatures and aggregate members inside their type constructors;
// token boundaries keep keywords such as "chan int" distinct from identifiers.
func functionValueSameNestedType(left string, right string) bool {
	if left == right {
		return true
	}
	if !functionValueTypeContainsSemanticLiteral(left) || !functionValueTypeContainsSemanticLiteral(right) {
		return false
	}
	mark := arena.Mark()
	source := []byte("package main\ntype Left = " + left + "\ntype Right = " + right + "\n")
	program := unit.Program{Package: "main"}
	equal := false
	if reparseFunctionValueProgram(&program, source, nil, len(source), -1) && len(program.Decls) == 2 {
		l := functionValueTokenAtSpan(&program, program.Decls[0].NameStart, program.Decls[0].NameEnd) + 2
		r := functionValueTokenAtSpan(&program, program.Decls[1].NameStart, program.Decls[1].NameEnd) + 2
		lEnd, rEnd := functionValueTypeEnd(&program, l), functionValueTypeEnd(&program, r)
		equal = lEnd > l && rEnd > r && functionValueSameAggregateTokens(&program, l, lEnd, r, rEnd)
	}
	arena.Rewind(mark)
	return equal
}

func functionValueTypeContainsSemanticLiteral(typ string) bool {
	for i := 0; i+4 < len(typ); i++ {
		if typ[i:i+4] == "func" && (typ[i+4] == '(' || functionValueIsSpace(typ[i+4])) {
			return true
		}
	}
	for i := 0; i+6 < len(typ); i++ {
		if typ[i:i+6] == "struct" && (typ[i+6] == '{' || functionValueIsSpace(typ[i+6])) {
			return true
		}
		if i+9 < len(typ) && typ[i:i+9] == "interface" && (typ[i+9] == '{' || functionValueIsSpace(typ[i+9])) {
			return true
		}
	}
	return false
}

func functionValueSignatureByTypeText(signatures []functionValueSignature, typ string) int {
	mark := arena.Mark()
	for i := 0; i < len(signatures); i++ {
		sig := signatures[i]
		if !sig.anonymous {
			continue
		}
		text := "func(" + functionValueJoin(sig.paramTypes, ",") + ")"
		if len(sig.resultTypes) == 1 {
			text += sig.resultTypes[0]
		} else if len(sig.resultTypes) > 1 {
			text += "(" + functionValueJoin(sig.resultTypes, ",") + ")"
		}
		equal := functionValueCompactTypeText(text) == functionValueCompactTypeText(typ)
		arena.Rewind(mark)
		if equal {
			return i
		}
	}
	// The fast spelling comparison above handles canonical and unnamed types.
	// For named parameters/results, reuse the signature parser: names are not
	// part of function type identity. This also applies to calls through locals,
	// not just to arguments passed at the original call site.
	compact := functionValueCompactTypeText(typ)
	if len(compact) < 5 || compact[:5] != "func(" {
		arena.Rewind(mark)
		return -1
	}
	found := -1
	if candidate, ok := functionValueSignatureFromTypeText(typ); ok {
		found = functionValueSignatureByShape(signatures, candidate)
	}
	arena.Rewind(mark)
	return found
}

func functionValueSignatureFromTypeText(typ string) (functionValueSignature, bool) {
	compact := functionValueCompactTypeText(typ)
	if len(compact) < 5 || compact[:5] != "func(" {
		return functionValueSignature{}, false
	}
	source := []byte("package main\ntype Signature " + typ + "\n")
	program := unit.Program{Package: "main"}
	if reparseFunctionValueProgram(&program, source, nil, len(source), -1) {
		for tok := 0; tok < len(program.Tokens); tok++ {
			if functionValueTokenEquals(&program, tok, "func") {
				candidate, _, ok := parseFunctionValueSignature(&program, tok, "")
				return candidate, ok
			}
		}
	}
	return functionValueSignature{}, false
}

func functionValueCompactTypeText(value string) string {
	out := make([]byte, len(value))
	write := 0
	for i := 0; i < len(value); i++ {
		if !functionValueIsSpace(value[i]) {
			out[write] = value[i]
			write++
		}
	}
	return string(out[:write])
}

func functionValueImplIndex(sig functionValueSignature, receiverType string, method string, function string) int {
	for i := 0; i < len(sig.storage.impls); i++ {
		impl := sig.storage.impls[i]
		if impl.receiverType == receiverType && impl.method == method && impl.function == function {
			return i
		}
	}
	return -1
}

func functionValueTokenCanStartType(program *unit.Program, tok int) bool {
	if tok < 0 || tok >= len(program.Tokens) {
		return false
	}
	text := functionValueTokenText(program, tok)
	return program.Tokens[tok].KindLine&255 == unit.TokenIdent || text == "*" || text == "[" || text == "struct" || text == "interface" || text == "map" || text == "func" || text == "chan" || text == "<-"
}

func functionValueTokenAtSpan(program *unit.Program, start int, end int) int {
	low := 0
	high := len(program.Tokens)
	for low < high {
		middle := low + (high-low)/2
		if program.Tokens[middle].Start < start {
			low = middle + 1
		} else {
			high = middle
		}
	}
	for low < len(program.Tokens) && program.Tokens[low].Start == start {
		if program.Tokens[low].Start+program.Tokens[low].Size == end {
			return low
		}
		low++
	}
	return -1
}

func functionValueTokensText(program *unit.Program, start int, end int) string {
	if start < 0 || start >= end || end > len(program.Tokens) {
		return ""
	}
	byteStart := program.Tokens[start].Start
	last := program.Tokens[end-1]
	byteEnd := last.Start + last.Size
	if byteStart < 0 || byteEnd > len(program.Text) {
		return ""
	}
	return string(program.Text[byteStart:byteEnd])
}

func functionValueTokenEdit(program *unit.Program, tok int, replacement string) functionValueEdit {
	item := program.Tokens[tok]
	return functionValueEdit{start: item.Start, end: item.Start + item.Size, text: replacement}
}

func functionValueTokenRangeEdit(program *unit.Program, start int, end int, replacement string) functionValueEdit {
	first := program.Tokens[start]
	if end <= start {
		return functionValueEdit{start: first.Start, end: first.Start, text: replacement}
	}
	last := program.Tokens[end-1]
	return functionValueEdit{start: first.Start, end: last.Start + last.Size, text: replacement}
}

func applyFunctionValueEdits(src []byte, edits []functionValueEdit) ([]byte, bool) {
	return applyFunctionValueEditsCapacity(src, edits, 0)
}

func applyFunctionValueEditsCapacity(src []byte, edits []functionValueEdit, extra int) ([]byte, bool) {
	return applyFunctionValueEditsCapacityMode(src, edits, extra, false)
}

func applyFunctionValueEditsCapacityMode(src []byte, edits []functionValueEdit, extra int, transient bool) ([]byte, bool) {
	sortFunctionValueEdits(edits)
	size := len(src) + extra
	maxGrowth := 0
	pos := 0
	for _, edit := range edits {
		if edit.start < pos || edit.end < edit.start || edit.end > len(src) {
			return nil, false
		}
		size += len(edit.text) - (edit.end - edit.start)
		if growth := size - len(src) - extra; growth > maxGrowth {
			maxGrowth = growth
		}
		pos = edit.end
	}
	var out []byte
	if transient && cap(src) >= size && cap(src) >= len(src)+maxGrowth {
		buffer := src[:len(src)+maxGrowth]
		copy(buffer[maxGrowth:], src)
		src = buffer[maxGrowth:]
		out = buffer[:0]
	} else {
		out = make([]byte, 0, size)
	}
	pos = 0
	for i := 0; i < len(edits); i++ {
		edit := edits[i]
		if edit.start < pos || edit.end < edit.start || edit.end > len(src) {
			return nil, false
		}
		out = append(out, src[pos:edit.start]...)
		out = appendFunctionValueString(out, edit.text)
		pos = edit.end
	}
	out = append(out, src[pos:]...)
	return out, true
}

func sortFunctionValueEdits(edits []functionValueEdit) {
	sorted := true
	for i := 1; i < len(edits); i++ {
		if edits[i].start < edits[i-1].start || edits[i].start == edits[i-1].start && edits[i].end < edits[i-1].end {
			sorted = false
			break
		}
	}
	if sorted {
		return
	}
	buffer := make([]functionValueEdit, len(edits))
	for width := 1; width < len(edits); width *= 2 {
		for start := 0; start < len(edits); start += 2 * width {
			middle, end := start+width, start+2*width
			if middle > len(edits) {
				middle = len(edits)
			}
			if end > len(edits) {
				end = len(edits)
			}
			left, right := start, middle
			for out := start; out < end; out++ {
				if left < middle && (right >= end || edits[left].start < edits[right].start || edits[left].start == edits[right].start && edits[left].end <= edits[right].end) {
					buffer[out] = edits[left]
					left++
				} else {
					buffer[out] = edits[right]
					right++
				}
			}
		}
		copy(edits, buffer)
	}
}

func appendFunctionValueString(out []byte, text string) []byte {
	for i := 0; i < len(text); i++ {
		out = append(out, text[i])
	}
	return out
}

func functionValueJoin(items []string, separator string) string {
	out := ""
	for i := 0; i < len(items); i++ {
		if i > 0 {
			out = out + separator
		}
		out = out + items[i]
	}
	return out
}

func functionValueZero(result string) string {
	if result == "string" {
		return "\"\""
	}
	if result == "bool" {
		return "false"
	}
	nilable := len(result) > 0 && result[0] == '*'
	if len(result) > 0 && result[0] == '[' {
		i := 1
		for i < len(result) && functionValueIsSpace(result[i]) {
			i++
		}
		nilable = i < len(result) && result[i] == ']'
	}
	if nilable || functionValueHasPrefix(result, "map[") || functionValueHasPrefix(result, "func(") || functionValueHasPrefix(result, "interface{") {
		return "nil"
	}
	return "0"
}

func functionValueCanUseScalarZero(result string) bool {
	scalar := []string{"int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "byte", "rune", "float32", "float64", "complex64", "complex128"}
	for i := 0; i < len(scalar); i++ {
		if result == scalar[i] {
			return true
		}
	}
	return false
}

func functionValueHasPrefix(value string, prefix string) bool {
	i := 0
	for p := 0; p < len(prefix); p++ {
		for i < len(value) && functionValueIsSpace(value[i]) {
			i++
		}
		if i >= len(value) || value[i] != prefix[p] {
			return false
		}
		i++
	}
	return true
}

func functionValueIsSpace(value byte) bool {
	return value == ' ' || value == '\t' || value == '\n' || value == '\r'
}

func functionValueDecimal(value int) string {
	if value == 0 {
		return "0"
	}
	var digits []byte
	for value > 0 {
		digits = append(digits, byte('0'+value%10))
		value /= 10
	}
	var out []byte
	for i := len(digits) - 1; i >= 0; i-- {
		out = append(out, digits[i])
	}
	return string(out)
}

func functionValueNameInList(list []string, name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(list); i++ {
		if list[i] == name {
			return true
		}
	}
	return false
}

func functionValueFindMatchingParen(program *unit.Program, open int) int {
	return functionValueFindMatching(program, open, "(", ")")
}

func functionValueFindMatchingBrace(program *unit.Program, open int) int {
	return functionValueFindMatching(program, open, "{", "}")
}

func functionValueFindMatching(program *unit.Program, open int, left string, right string) int {
	if !functionValueTokenEquals(program, open, left) {
		return -1
	}
	if len(left) == 1 && len(right) == 1 {
		depth := 0
		for i := open; i < len(program.Tokens); i++ {
			item := &program.Tokens[i]
			if item.KindLine&255 != unit.TokenOp || item.Size != 1 {
				continue
			}
			c := program.Text[item.Start]
			if c == left[0] {
				depth++
			} else if c == right[0] {
				depth--
				if depth == 0 {
					return i
				}
			}
		}
		return -1
	}
	depth := 0
	for i := open; i < len(program.Tokens); i++ {
		if functionValueTokenEquals(program, i, left) {
			depth++
		} else if functionValueTokenEquals(program, i, right) {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func functionValueTokenEquals(program *unit.Program, tok int, want string) bool {
	if uint(tok) >= uint(len(program.Tokens)) {
		return false
	}
	token := &program.Tokens[tok]
	if token.Size != len(want) || token.Start < 0 || token.Start+token.Size > len(program.Text) {
		return false
	}
	if len(want) == 1 {
		return program.Text[token.Start] == want[0]
	}
	if len(want) == 2 {
		return program.Text[token.Start] == want[0] && program.Text[token.Start+1] == want[1]
	}
	if len(want) == 3 {
		return program.Text[token.Start] == want[0] && program.Text[token.Start+1] == want[1] && program.Text[token.Start+2] == want[2]
	}
	if len(want) == 4 {
		return program.Text[token.Start] == want[0] && program.Text[token.Start+1] == want[1] && program.Text[token.Start+2] == want[2] && program.Text[token.Start+3] == want[3]
	}
	for i := 0; i < len(want); i++ {
		if program.Text[token.Start+i] != want[i] {
			return false
		}
	}
	return true
}

func functionValueTokenText(program *unit.Program, tok int) string {
	if tok < 0 || tok >= len(program.Tokens) {
		return ""
	}
	token := &program.Tokens[tok]
	if token.Start < 0 || token.Start+token.Size > len(program.Text) {
		return ""
	}
	return string(program.Text[token.Start : token.Start+token.Size])
}
