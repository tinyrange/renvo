package link

import "renvo.dev/internal/unit"

// Infer a bound method's callable type from its receiver and declaration, not
// from the receiver expression alone. This is needed before captured locals
// are rewritten into environment pointers.
func functionValueInferredBoundMethod(program *unit.Program, before, start, end int) (functionValueSignature, bool) {
	start, end = functionValueUnparen(program, start, end)
	if end < start+3 || !functionValueTokenCharIs(program, end-2, '.') || program.Tokens[start].KindLine&255 != unit.TokenIdent || program.Tokens[end-1].KindLine&255 != unit.TokenIdent {
		return functionValueSignature{}, false
	}
	if !functionValueMethodNameExists(program, end-1) {
		return functionValueSignature{}, false
	}
	base := functionValueTokenText(program, start)
	owner := ""
	if end == start+3 {
		owner = functionValueEnclosingLocalType(program, before, base)
	} else {
		owner = functionValueFieldChainType(program, before, start, end-2)
	}
	if owner == "" && end == start+3 {
		owner = ordinaryGlobalType(program, base)
	}
	if owner == "" {
		return functionValueSignature{}, false
	}
	method := functionValueTokenText(program, end-1)
	for i := range program.Funcs {
		fn := &program.Funcs[i]
		if fn.ReceiverStart >= fn.ReceiverEnd || !functionValueTokenEquals(program, fn.NameTok, method) {
			continue
		}
		if !functionValueTypeEmbeds(program, owner, functionValueReceiverType(program, fn), 0) {
			continue
		}
		sig, _, ok := parseFunctionValueCallableSignature(program, fn.NameTok, "")
		return sig, ok
	}
	return functionValueSignature{}, false
}
func functionValueSignatureText(sig functionValueSignature) string {
	out := "func(" + functionValueJoin(sig.paramTypes, ",") + ")"
	if len(sig.resultTypes) == 1 {
		out += sig.resultTypes[0]
	} else if len(sig.resultTypes) > 1 {
		out += "(" + functionValueJoin(sig.resultTypes, ",") + ")"
	}
	return out
}

// Each initializer in a grouped declaration has its own callable shape.
func inferredBoundInitializers(program *unit.Program, op int) ([]int, []int, int) {
	short := functionValueTokenEquals(program, op, ":=")
	if !short && !functionValueTokenCharIs(program, op, '=') {
		return nil, nil, 0
	}
	first := op - 1
	if program.Tokens[first].KindLine&255 != unit.TokenIdent {
		return nil, nil, 0
	}
	for first >= 2 && functionValueTokenCharIs(program, first-1, ',') && program.Tokens[first-2].KindLine&255 == unit.TokenIdent {
		first -= 2
	}
	if !short && !functionValueTokenKindIs(program, first-1, unit.TokenVar) {
		return nil, nil, 0
	}
	starts, ends := functionValueCommaParts(program, op+1, mapLowerAssignmentEnd(program, op))
	if len(starts) != (op-1-first)/2+1 {
		return nil, nil, 0
	}
	return starts, ends, first
}
func discoverInferredBoundMethods(program *unit.Program, signatures []functionValueSignature) []functionValueSignature {
	for op := 1; op+3 < len(program.Tokens); op++ {
		starts, ends, first := inferredBoundInitializers(program, op)
		for i, start := range starts {
			sig, ok := functionValueInferredBoundMethod(program, first, start, ends[i])
			if !ok || functionValueSignatureByShape(signatures, sig) >= 0 {
				continue
			}
			sig.name = "__renvo_function_" + functionValueDecimal(len(signatures))
			sig.declFuncTok = -1
			sig.declEndTok = -1
			sig.anonymous = true
			signatures = append(signatures, sig)
		}
	}
	return signatures
}
func lowerInferredBoundMethods(program *unit.Program, signatures []functionValueSignature, edits []functionValueEdit) []functionValueEdit {
	for op := 1; op+3 < len(program.Tokens); op++ {
		starts, ends, first := inferredBoundInitializers(program, op)
		for i, start := range starts {
			sig, ok := functionValueInferredBoundMethod(program, first, start, ends[i])
			if !ok {
				continue
			}
			index := functionValueSignatureByShape(signatures, sig)
			if index >= 0 {
				start, _ = functionValueUnparen(program, start, ends[i])
				edits = lowerFunctionValueBoundMethod(program, first, start, index, signatures, edits)
			}
		}
	}
	return edits
}
