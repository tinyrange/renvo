package link

import "renvo.dev/internal/unit"

// Parallel assignments bind each RHS callable to its own destination type.
// Using the last selector's signature for the first RHS silently changes arity.
func lowerFunctionValueParallelAssignment(program *unit.Program, op int, signatures []functionValueSignature, fields []functionValueField, edits []functionValueEdit) ([]functionValueEdit, bool) {
	end := op
	var left []int
	for {
		start := functionValuePrimaryStart(program, end-1)
		if start < 0 {
			return edits, false
		}
		left = append(left, end-1)
		if start == 0 || !functionValueTokenCharIs(program, start-1, ',') {
			break
		}
		end = start - 1
	}
	if len(left) < 2 {
		return edits, false
	}
	rhsEnd := mapLowerAssignmentEnd(program, op)
	var right []int
	start, depth := op+1, 0
	for token := start; token < rhsEnd; token++ {
		if functionValueTokenCharIs(program, token, '(') || functionValueTokenCharIs(program, token, '[') || functionValueTokenCharIs(program, token, '{') {
			depth++
		}
		if functionValueTokenCharIs(program, token, ')') || functionValueTokenCharIs(program, token, ']') || functionValueTokenCharIs(program, token, '}') {
			depth--
		}
		if depth == 0 && functionValueTokenCharIs(program, token, ',') {
			right = append(right, start)
			start = token + 1
		}
	}
	right = append(right, start)
	// A single multi-result call is not a list of independently bound methods.
	if len(left) != len(right) {
		return edits, true
	}
	for i, rhs := range right {
		target := left[len(left)-1-i]
		field := functionValueFieldForSelector(program, target, fields)
		sig := -1
		if field >= 0 {
			sig = fields[field].sig
		} else if program.Tokens[target].KindLine&255 == unit.TokenIdent {
			typ := functionValueEnclosingLocalType(program, op, functionValueTokenText(program, target))
			sig = functionValueSignatureByTypeText(signatures, typ)
			if sig < 0 {
				sig = functionValueSignatureByName(signatures, functionValueBareType(typ))
			}
		}
		if sig >= 0 {
			edits = lowerFunctionValueAt(program, op, rhs, sig, signatures, edits)
		}
	}
	return edits, true
}
