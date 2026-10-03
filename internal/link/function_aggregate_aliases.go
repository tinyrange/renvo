package link

import (
	"renvo.dev/internal/syntax"
	"renvo.dev/internal/unit"
)

// Use one spelling for matching aggregate aliases in callback signatures. Keep
// their declarations intact: expanding an imported struct/interface at a use
// site can change the package identity of its unexported members.
func functionValueMatchingAggregateAlias(program *unit.Program, target int, end int, use int) string {
	for i := 0; i < len(program.Decls); i++ {
		decl := &program.Decls[i]
		if decl.Kind != unit.TokenType {
			continue
		}
		name := functionValueTokenAtSpan(program, decl.NameStart, decl.NameEnd)
		start := name + 2
		if !functionValueTokenCharIs(program, name+1, '=') || !functionValueTokenEquals(program, start, functionValueTokenText(program, target)) {
			continue
		}
		finish := functionValueTypeEnd(program, start)
		if finish <= start || !functionValueSameAggregateTokens(program, start, finish, target, end) {
			continue
		}
		candidate := functionValueTokenText(program, name)
		if functionValueSignatureLocalType(program, use, candidate) < 0 && functionValueLexicalLocalType(program, use, candidate) == "" {
			return candidate
		}
	}
	return ""
}

func functionValueTypeOwner(program *unit.Program, token int) string {
	for _, pkg := range program.Packages {
		if program.Tokens[token].Start >= pkg.TextStart && program.Tokens[token].Start < pkg.TextEnd {
			return pkg.ImportPath
		}
	}
	return program.ImportPath
}

func functionValueSameAggregateTokens(program *unit.Program, left int, leftEnd int, right int, rightEnd int) bool {
	return functionValueSameSemanticType(program, left, leftEnd, right, rightEnd, 0)
}

type functionValueAggregateMember struct {
	name          string
	owner         string
	tag           string
	start         int
	end           int
	embedded      bool
	universeError bool
}

// Struct identity uses the ordered fields after expanding grouped names.
// Explicit interface methods are compared without regard to declaration order.
func functionValueSameAggregateMembersAt(program *unit.Program, left int, leftEnd int, right int, rightEnd int, depth int) bool {
	structure := functionValueTokenKindIs(program, left, unit.TokenStruct)
	ls, lok := functionValueAggregateMembers(program, left, leftEnd)
	rs, rok := functionValueAggregateMembers(program, right, rightEnd)
	if !structure {
		ls, lok = functionValueInterfaceMembers(program, left, leftEnd, depth+1)
		rs, rok = functionValueInterfaceMembers(program, right, rightEnd, depth+1)
	}
	if !lok || !rok || len(ls) != len(rs) {
		return false
	}
	used := make([]bool, len(rs))
	for i := 0; i < len(ls); i++ {
		match := i
		if !structure {
			match = -1
			for j := 0; j < len(rs); j++ {
				if !used[j] && ls[i].name == rs[j].name && ls[i].owner == rs[j].owner && functionValueSameInterfaceMethodSignature(program, ls[i], rs[j], depth+1) {
					match = j
					break
				}
			}
		}
		if match < 0 || ls[i].name != rs[match].name || ls[i].owner != rs[match].owner || ls[i].tag != rs[match].tag || ls[i].embedded != rs[match].embedded {
			return false
		}
		if structure {
			if !functionValueSameSemanticType(program, ls[i].start, ls[i].end, rs[match].start, rs[match].end, depth+1) {
				return false
			}
		} else if !functionValueSameInterfaceMethodSignature(program, ls[i], rs[match], depth+1) {
			return false
		}
		used[match] = true
	}
	return true
}

func functionValueAggregateMembers(program *unit.Program, start int, end int) ([]functionValueAggregateMember, bool) {
	var members []functionValueAggregateMember
	structure := functionValueTokenKindIs(program, start, unit.TokenStruct)
	for field := start + 2; field < end-1; {
		if functionValueTokenCharIs(program, field, ';') {
			field++
			continue
		}
		typ := field
		finish := functionValueTypeEnd(program, field)
		embedded := false
		var names []int
		if !structure && functionValueTokenCharIs(program, field+1, '(') {
			_, next, ok := parseFunctionValueCallableSignature(program, field, "")
			if !ok {
				return nil, false
			}
			finish = next
			names = append(names, field)
		} else if structure {
			embedded = finish > field && (finish == end-1 || functionValueTokenCharIs(program, finish, ';') || program.Tokens[finish].KindLine&255 == unit.TokenString)
			if embedded {
				names = append(names, finish-1)
			} else {
				for {
					if typ >= end-1 || program.Tokens[typ].KindLine&255 != unit.TokenIdent {
						return nil, false
					}
					names = append(names, typ)
					typ++
					if !functionValueTokenCharIs(program, typ, ',') {
						break
					}
					typ++
				}
				finish = functionValueTypeEnd(program, typ)
			}
		} else {
			embedded = true
			names = append(names, -1)
		}
		if finish <= typ || finish > end-1 {
			return nil, false
		}
		next := finish
		tag := ""
		if next < end-1 && program.Tokens[next].KindLine&255 == unit.TokenString {
			tok := program.Tokens[next]
			value, ok := syntax.StringLiteralValue(program.Text, syntax.MakeToken(syntax.TokenString, tok.Start, tok.Start+tok.Size, tok.KindLine>>8))
			if !ok {
				return nil, false
			}
			tag = value
			next++
		}
		for _, nameToken := range names {
			name, owner := "", ""
			if nameToken >= 0 {
				name = functionValueTokenText(program, nameToken)
				if !syntax.IdentifierExported(program.Text, program.Tokens[nameToken].Start) {
					owner = functionValueTypeOwner(program, nameToken)
				}
			}
			members = append(members, functionValueAggregateMember{name: name, owner: owner, tag: tag, start: typ, end: finish, embedded: embedded})
		}
		field = next
	}
	return members, true
}
