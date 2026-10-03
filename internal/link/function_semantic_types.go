package link

import (
	"renvo.dev/internal/check"
	"renvo.dev/internal/unit"
)

// Resolve aliases at their declaration site. This comparison never emits an
// expanded type at a use site, so local shadows cannot capture an alias target.
func functionValueSemanticType(program *unit.Program, start int, end int, depth int) (int, int, int, string) {
	for depth <= len(program.Tokens) {
		if start < 0 || start >= end || end > len(program.Tokens) {
			return start, end, -1, ""
		}
		if functionValueTokenEquals(program, start, "(") && functionValueFindMatchingParen(program, start) == end-1 {
			start++
			end--
			depth++
			continue
		}
		if end != start+1 || program.Tokens[start].KindLine&255 != unit.TokenIdent {
			return start, end, -1, ""
		}
		name := functionValueTokenText(program, start)
		declaration := functionValueSignatureLocalType(program, start, name)
		if declaration < 0 {
			for i := 0; i < len(program.Decls); i++ {
				decl := &program.Decls[i]
				if decl.Kind == unit.TokenType && ordinarySpanEquals(program.Text, decl.NameStart, decl.NameEnd, name) {
					declaration = functionValueTokenAtSpan(program, decl.NameStart, decl.NameEnd)
					break
				}
			}
		}
		if declaration >= 0 {
			if !functionValueTokenEquals(program, declaration+1, "=") {
				return start, end, declaration, ""
			}
			start = declaration + 2
			end = functionValueTypeEnd(program, start)
			depth++
			continue
		}
		if name == "byte" {
			name = "uint8"
		} else if name == "rune" {
			name = "int32"
		}
		return start, end, -1, name
	}
	return start, start, -1, ""
}

func functionValueSameSemanticType(program *unit.Program, left int, leftEnd int, right int, rightEnd int, depth int) bool {
	if depth > len(program.Tokens) {
		return false
	}
	// Linked global names are unique. Equal identifiers need only check local
	// bindings before taking the common path without scanning every declaration.
	if left >= 0 && right >= 0 && leftEnd == left+1 && rightEnd == right+1 && leftEnd <= len(program.Tokens) && rightEnd <= len(program.Tokens) && program.Tokens[left].KindLine&255 == unit.TokenIdent && program.Tokens[right].KindLine&255 == unit.TokenIdent && functionValueTokenEquals(program, left, functionValueTokenText(program, right)) {
		name := functionValueTokenText(program, left)
		if functionValueSignatureLocalType(program, left, name) == functionValueSignatureLocalType(program, right, name) {
			return true
		}
	}
	left, leftEnd, ld, lb := functionValueSemanticType(program, left, leftEnd, depth)
	right, rightEnd, rd, rb := functionValueSemanticType(program, right, rightEnd, depth)
	if left >= leftEnd || right >= rightEnd {
		return false
	}
	if left == right && leftEnd == rightEnd {
		return true
	}
	if ld >= 0 || rd >= 0 {
		return ld >= 0 && ld == rd
	}
	if lb != "" || rb != "" {
		if lb == "any" && functionValueTokenEquals(program, right, "interface") {
			methods, ok := functionValueInterfaceMembers(program, right, rightEnd, depth+1)
			return ok && len(methods) == 0
		}
		if rb == "any" && functionValueTokenEquals(program, left, "interface") {
			methods, ok := functionValueInterfaceMembers(program, left, leftEnd, depth+1)
			return ok && len(methods) == 0
		}
		return lb != "" && lb == rb
	}
	token := functionValueTokenText(program, left)
	if token != functionValueTokenText(program, right) {
		return false
	}
	if token == "*" || token == "..." {
		return functionValueSameSemanticType(program, left+1, leftEnd, right+1, rightEnd, depth+1)
	}
	if token == "." && functionValueTokenEquals(program, left+1, ".") && functionValueTokenEquals(program, left+2, ".") && functionValueTokenEquals(program, right+1, ".") && functionValueTokenEquals(program, right+2, ".") {
		return functionValueSameSemanticType(program, left+3, leftEnd, right+3, rightEnd, depth+1)
	}
	if token == "struct" || token == "interface" {
		return functionValueSameAggregateMembersAt(program, left, leftEnd, right, rightEnd, depth+1)
	}
	if token == "func" || program.Tokens[left].KindLine&255 == unit.TokenIdent && functionValueTokenEquals(program, left+1, "(") {
		return functionValueSameCallableTypes(program, left, leftEnd, right, rightEnd, depth+1)
	}
	if token == "[" || token == "map" {
		lo, ro := left, right
		if token == "map" {
			lo++
			ro++
		}
		lc := functionValueFindMatching(program, lo, "[", "]")
		rc := functionValueFindMatching(program, ro, "[", "]")
		if lc < lo || rc < ro || lc >= leftEnd || rc >= rightEnd {
			return false
		}
		if token == "map" {
			if !functionValueSameSemanticType(program, lo+1, lc, ro+1, rc, depth+1) {
				return false
			}
		} else if !functionValueSameArrayBound(program, lo+1, lc, ro+1, rc) {
			return false
		}
		return functionValueSameSemanticType(program, lc+1, leftEnd, rc+1, rightEnd, depth+1)
	}
	if token == "chan" || token == "<-" {
		ln, rn := left+1, right+1
		if token == "<-" {
			ln++
			rn++
		} else {
			ls, rs := functionValueTokenEquals(program, ln, "<-"), functionValueTokenEquals(program, rn, "<-")
			if ls != rs {
				return false
			}
			if ls {
				ln++
				rn++
			}
		}
		return functionValueSameSemanticType(program, ln, leftEnd, rn, rightEnd, depth+1)
	}
	return functionValueSameTypeSpelling(program, left, leftEnd, right, rightEnd)
}

func functionValueSameTypeSpelling(program *unit.Program, left int, leftEnd int, right int, rightEnd int) bool {
	if leftEnd-left != rightEnd-right {
		return false
	}
	for left < leftEnd {
		if program.Tokens[left].KindLine&255 != program.Tokens[right].KindLine&255 || !functionValueTokenEquals(program, left, functionValueTokenText(program, right)) {
			return false
		}
		left++
		right++
	}
	return true
}

func functionValueSameArrayBound(program *unit.Program, left int, leftEnd int, right int, rightEnd int) bool {
	if left == leftEnd || right == rightEnd {
		return left == leftEnd && right == rightEnd
	}
	// Literal expressions have no lexical bindings. Named constants need their
	// own declaration-aware evaluation before differing spellings can be merged.
	literal := true
	for i := left; i < leftEnd; i++ {
		if program.Tokens[i].KindLine&255 == unit.TokenIdent {
			literal = false
		}
	}
	for i := right; i < rightEnd; i++ {
		if program.Tokens[i].KindLine&255 == unit.TokenIdent {
			literal = false
		}
	}
	if literal {
		if equal, known := check.LiteralIntegerConstantsEqual([]byte(functionValueTokensText(program, left, leftEnd)), []byte(functionValueTokensText(program, right, rightEnd))); known {
			return equal
		}
	}
	return functionValueSameTypeSpelling(program, left, leftEnd, right, rightEnd)
}

type functionValueSemanticRange struct {
	start int
	end   int
}

func functionValueSemanticFields(program *unit.Program, start int, end int) []functionValueSemanticRange {
	starts, ends := functionValueCommaParts(program, start, end)
	fields := make([]functionValueSemanticRange, len(starts))
	carried := functionValueSemanticRange{start: -1}
	for i := len(starts) - 1; i >= 0; i-- {
		field := functionValueSemanticRange{start: starts[i], end: ends[i]}
		if field.start+1 < field.end && program.Tokens[field.start].KindLine&255 == unit.TokenIdent && functionValueTypeEnd(program, field.start) != field.end {
			field.start++
			carried = field
		} else if field.start+1 == field.end && carried.start >= 0 {
			field = carried
		} else {
			carried.start = -1
		}
		fields[i] = field
	}
	return fields
}

func functionValueSemanticCallable(program *unit.Program, start int, end int) ([]functionValueSemanticRange, []functionValueSemanticRange, bool) {
	close := functionValueFindMatchingParen(program, start+1)
	if close < 0 || close >= end {
		return nil, nil, false
	}
	params := functionValueSemanticFields(program, start+2, close)
	var results []functionValueSemanticRange
	next := close + 1
	if next < end {
		if functionValueTokenEquals(program, next, "(") && functionValueFindMatchingParen(program, next) == end-1 {
			results = functionValueSemanticFields(program, next+1, end-1)
		} else {
			results = append(results, functionValueSemanticRange{start: next, end: end})
		}
	}
	return params, results, true
}

func functionValueSameCallableTypes(program *unit.Program, left int, leftEnd int, right int, rightEnd int, depth int) bool {
	lp, lr, lok := functionValueSemanticCallable(program, left, leftEnd)
	rp, rr, rok := functionValueSemanticCallable(program, right, rightEnd)
	if !lok || !rok || len(lp) != len(rp) || len(lr) != len(rr) {
		return false
	}
	for i := 0; i < len(lp); i++ {
		if !functionValueSameSemanticType(program, lp[i].start, lp[i].end, rp[i].start, rp[i].end, depth+1) {
			return false
		}
	}
	for i := 0; i < len(lr); i++ {
		if !functionValueSameSemanticType(program, lr[i].start, lr[i].end, rr[i].start, rr[i].end, depth+1) {
			return false
		}
	}
	return true
}

// Embedding exposes a named interface's method set; it does not make ordinary
// uses of that named type identical to its underlying interface.
func functionValueInterfaceMembers(program *unit.Program, start int, end int, depth int) ([]functionValueAggregateMember, bool) {
	if depth > len(program.Tokens) {
		return nil, false
	}
	start, end, declaration, builtin := functionValueSemanticType(program, start, end, depth)
	if builtin == "any" {
		return nil, true
	}
	if builtin == "error" {
		return []functionValueAggregateMember{{name: "Error", universeError: true}}, true
	}
	if declaration >= 0 {
		start = declaration + 1
		end = functionValueTypeEnd(program, start)
		return functionValueInterfaceMembers(program, start, end, depth+1)
	}
	if start >= end || !functionValueTokenEquals(program, start, "interface") {
		return nil, false
	}
	members, ok := functionValueAggregateMembers(program, start, end)
	if !ok {
		return nil, false
	}
	var methods []functionValueAggregateMember
	for _, member := range members {
		if member.embedded {
			nested, valid := functionValueInterfaceMembers(program, member.start, member.end, depth+1)
			if !valid {
				return nil, false
			}
			for _, method := range nested {
				var added bool
				methods, added = functionValueAppendInterfaceMember(program, methods, method, depth+1)
				if !added {
					return nil, false
				}
			}
		} else {
			var added bool
			methods, added = functionValueAppendInterfaceMember(program, methods, member, depth+1)
			if !added {
				return nil, false
			}
		}
	}
	return methods, true
}

func functionValueAppendInterfaceMember(program *unit.Program, methods []functionValueAggregateMember, member functionValueAggregateMember, depth int) ([]functionValueAggregateMember, bool) {
	for _, method := range methods {
		if method.name == member.name && method.owner == member.owner {
			return methods, functionValueSameInterfaceMethodSignature(program, method, member, depth+1)
		}
	}
	return append(methods, member), true
}

func functionValueSameInterfaceMethodSignature(program *unit.Program, left functionValueAggregateMember, right functionValueAggregateMember, depth int) bool {
	if !left.universeError && !right.universeError {
		return functionValueSameCallableTypes(program, left.start, left.end, right.start, right.end, depth+1)
	}
	if left.universeError && right.universeError {
		return true
	}
	method := left
	if left.universeError {
		method = right
	}
	params, results, ok := functionValueSemanticCallable(program, method.start, method.end)
	if !ok || len(params) != 0 || len(results) != 1 {
		return false
	}
	_, _, declaration, builtin := functionValueSemanticType(program, results[0].start, results[0].end, depth+1)
	return declaration < 0 && builtin == "string"
}
