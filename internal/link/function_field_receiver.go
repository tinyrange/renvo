package link

import "renvo.dev/internal/unit"

// Resolve a side-effect-free named field chain without running the general
// expression/type inference machinery for every ordinary struct selector.
func functionValueFieldChainType(program *unit.Program, before, start, end int) string {
	if start >= end || (end-start)&1 == 0 || program.Tokens[start].KindLine&255 != unit.TokenIdent {
		return ""
	}
	for tok := start + 1; tok < end; tok += 2 {
		if !functionValueTokenCharIs(program, tok, '.') || program.Tokens[tok+1].KindLine&255 != unit.TokenIdent {
			return ""
		}
	}
	name := functionValueTokenText(program, start)
	typ := functionValueEnclosingLocalType(program, before, name)
	if typ == "" {
		typ = ordinaryGlobalType(program, name)
	}
	for tok := start + 2; typ != "" && tok < end; tok += 2 {
		typ = functionValueStructFieldType(program, typ, functionValueTokenText(program, tok))
	}
	return typ
}
func functionValueMethodNameExists(program *unit.Program, tok int) bool {
	for i := range program.Funcs {
		fn := &program.Funcs[i]
		if fn.ReceiverStart < fn.ReceiverEnd && functionValueTokenEquals(program, fn.NameTok, functionValueTokenText(program, tok)) {
			return true
		}
	}
	return false
}
