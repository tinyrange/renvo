package link

import "renvo.dev/internal/unit"

// A declaration initialized by a named function has its anonymous callable type.
// When that shape uses tagged storage elsewhere, its initializer must use the
// same storage before it can be passed through a generic identity or callback.
func lowerFunctionValueInferredDeclarations(program *unit.Program, signatures []functionValueSignature, edits []functionValueEdit) []functionValueEdit {
	buckets := make([]int, len(program.Funcs)*2+1)
	next := make([]int, len(program.Funcs))
	for i := len(program.Funcs) - 1; i >= 0; i-- {
		bucket := functionValueNameHash(program, program.Funcs[i].NameTok) % len(buckets)
		next[i] = buckets[bucket]
		buckets[bucket] = i + 1
	}
	for op := 1; op+2 < len(program.Tokens); op++ {
		short := functionValueTokenEquals(program, op, ":=")
		if !short && !functionValueTokenEquals(program, op, "=") {
			continue
		}
		first := op - 1
		if program.Tokens[first].KindLine&255 != unit.TokenIdent {
			continue
		}
		for first >= 2 && functionValueTokenEquals(program, first-1, ",") && program.Tokens[first-2].KindLine&255 == unit.TokenIdent {
			first -= 2
		}
		if !short && !functionValueTokenEquals(program, first-1, "var") {
			continue
		}
		// Ordinary '=' lowering already handles a single inferred var. This
		// pass owns short declarations and grouped inferred declarations.
		if !short && first == op-1 {
			continue
		}
		if first == op-1 {
			rhs, end := op+1, op+2
			if functionValueTokenEquals(program, rhs, "(") {
				end = functionValueFindMatchingParen(program, rhs) + 1
			}
			outerEnd := end
			rhs, end = functionValueUnparen(program, rhs, end)
			if end != rhs+1 || program.Tokens[rhs].KindLine&255 != unit.TokenIdent ||
				(!functionValueTokenEquals(program, outerEnd, ";") && !functionValueTokenEquals(program, outerEnd, "}") && program.Tokens[outerEnd].KindLine>>8 == program.Tokens[outerEnd-1].KindLine>>8) {
				continue
			}
			if _, ok := functionValueCalledFunctionIndexed(program, rhs+1, buckets, next); !ok {
				continue
			}
		}
		starts, ends := functionValueCommaParts(program, op+1, mapLowerAssignmentEnd(program, op))
		if len(starts) != (op-1-first)/2+1 {
			continue
		}
		for i, rhs := range starts {
			rhs, end := functionValueUnparen(program, rhs, ends[i])
			if end != rhs+1 || program.Tokens[rhs].KindLine&255 != unit.TokenIdent {
				continue
			}
			fn, ok := functionValueCalledFunctionIndexed(program, rhs+1, buckets, next)
			if !ok {
				continue
			}
			candidate, _, ok := parseFunctionValueCallableSignature(program, fn.NameTok, "")
			if !ok {
				continue
			}
			index := functionValueSignatureByShape(signatures, candidate)
			if index >= 0 {
				// New bindings enter scope after their initializer. Resolve a
				// same-named function using the scope before the declaration.
				edits = lowerFunctionValueAt(program, first, rhs, index, signatures, edits)
			}
		}
	}
	return edits
}

func functionValueUnparen(program *unit.Program, start int, end int) (int, int) {
	for end-start >= 2 && functionValueTokenEquals(program, start, "(") && functionValueFindMatchingParen(program, start) == end-1 {
		start++
		end--
	}
	return start, end
}

func functionValueDirectFunctionType(program *unit.Program, name string) string {
	for i := range program.Funcs {
		fn := &program.Funcs[i]
		if fn.ReceiverStart != fn.ReceiverEnd || !functionValueTokenEquals(program, fn.NameTok, name) {
			continue
		}
		sig, _, ok := parseFunctionValueCallableSignature(program, fn.NameTok, "")
		if !ok {
			return ""
		}
		text := "func(" + functionValueJoin(sig.paramTypes, ",") + ")"
		if len(sig.resultTypes) == 1 {
			text += sig.resultTypes[0]
		} else if len(sig.resultTypes) > 1 {
			text += "(" + functionValueJoin(sig.resultTypes, ",") + ")"
		}
		return text
	}
	return ""
}
