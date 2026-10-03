package syntax

// TypeParamList is sparse: ordinary declarations do not pay for generic
// parameter storage. All spans are token indexes, with an exclusive end.
type TypeParamList struct {
	Kind       int
	NameTok    int
	StartTok   int
	EndTok     int
	Parameters []TypeParameter
}

type TypeParameter struct {
	NameTok         int
	ConstraintStart int
	ConstraintEnd   int
}

// Keep optional generic metadata behind one pointer. File is copied throughout
// the ordinary frontend, whose declarations need no type-parameter storage.
type GenericDeclarations struct{ Lists []TypeParamList }

func TypeParameterLists(file *File) []TypeParamList {
	if file.Generics == nil {
		return nil
	}
	return file.Generics.Lists
}

func TypeParameters(file *File, nameTok int) TypeParamList {
	for _, list := range TypeParameterLists(file) {
		if list.NameTok == nameTok {
			return list
		}
	}
	return TypeParamList{}
}

func parseTypeParamList(file *File, name int, start int, kind int) (int, bool) {
	end := skipBalanced(file, start, '[', ']')
	if end <= start+2 {
		return start, false
	}
	list := TypeParamList{Kind: kind, NameTok: name, StartTok: start, EndTok: end}
	var pending []int
	for i := start + 1; i < end-1; {
		if file.Tokens[i].KindLine&255 != TokenIdent {
			return start, false
		}
		pending = append(pending, i)
		i++
		if tokCharIs(file.Tokens, i, ',') {
			i++
			continue
		}
		constraint := i
		for i < end-1 && !tokCharIs(file.Tokens, i, ',') {
			if tokCharIs(file.Tokens, i, ';') {
				return start, false
			}
			close := byte(0)
			open := byte(file.Tokens[i].KindLine >> TokenOperatorCharShift & TokenOperatorCharMask)
			if open == '[' {
				close = ']'
			}
			if open == '(' {
				close = ')'
			}
			if open == '{' {
				close = '}'
			}
			if close != 0 {
				next := skipBalanced(file, i, open, close)
				if next <= i || next > end-1 {
					return start, false
				}
				i = next
			} else {
				i++
			}
		}
		if i == constraint {
			return start, false
		}
		for j := 0; j < len(pending); j++ {
			list.Parameters = append(list.Parameters, TypeParameter{NameTok: pending[j], ConstraintStart: constraint, ConstraintEnd: i})
		}
		pending = pending[:0]
		if i < end-1 {
			i++
		}
	}
	if len(pending) != 0 || len(list.Parameters) == 0 {
		return start, false
	}
	if file.Generics == nil {
		file.Generics = &GenericDeclarations{}
	}
	file.Generics.Lists = append(file.Generics.Lists, list)
	return end, true
}

// A bracket following a type name may introduce an array length. Go resolves
// ambiguous forms such as [P * C] as an array; a trailing comma or an
// unambiguous type element makes it a parameter list instead.
func typeDeclarationHasParameters(file *File, start int, end int) bool {
	if !tokCharIs(file.Tokens, start, '[') || start+2 >= end || file.Tokens[start+1].KindLine&255 != TokenIdent {
		return false
	}
	close := skipBalanced(file, start, '[', ']')
	if close <= start+3 {
		return false
	}
	second := start + 2
	if file.Tokens[second].KindLine&255 == TokenIdent {
		return true
	}
	depth := 0
	for i := second; i < close-1; i++ {
		kind := file.Tokens[i].KindLine & 255
		if kind == TokenStruct || kind == TokenInterface || kind == TokenMap || kind == TokenChan || kind == TokenFunc || tokCharIs(file.Tokens, i, '~') {
			return true
		}
		if depth == 0 && tokCharIs(file.Tokens, i, ',') {
			return true
		}
		// An array/slice type cannot appear as an ordinary operand here.
		if tokCharIs(file.Tokens, i, '[') && (i == second || tokCharIs(file.Tokens, i-1, '*') || tokCharIs(file.Tokens, i-1, '(') || tokCharIs(file.Tokens, i-1, '|')) {
			return true
		}
		if tokCharIs(file.Tokens, i, '(') || tokCharIs(file.Tokens, i, '[') || tokCharIs(file.Tokens, i, '{') {
			depth++
		}
		if tokCharIs(file.Tokens, i, ')') || tokCharIs(file.Tokens, i, ']') || tokCharIs(file.Tokens, i, '}') {
			depth--
		}
	}
	return false
}
