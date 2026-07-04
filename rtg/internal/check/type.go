package check

import "j5.nz/rtg/rtg/internal/syntax"

const (
	TypeOther = iota
	TypeNamed
	TypeStruct
	TypeInterface
	TypeMap
	TypeSlice
	TypeArray
	TypePointer
	TypeFunc
)

type TypeInfo struct {
	Name             string
	Kind             int
	File             int
	Token            int
	Decl             int
	Symbol           int
	Alias            bool
	TypeStart        int
	TypeEnd          int
	Fields           []Field
	InterfaceMethods []InterfaceMethod
	InterfaceEmbeds  []InterfaceEmbed
	Methods          []int
}

func LookupType(info PackageInfo, name string) int {
	for i := 0; i < len(info.Types); i++ {
		if info.Types[i].Name == name {
			return i
		}
	}
	return -1
}

func buildTypeInfo(file syntax.File, decl DeclInfo, declIndex int) TypeInfo {
	out := TypeInfo{
		Name:      decl.Name,
		Kind:      classifyType(file, decl.TypeStart, decl.TypeEnd),
		File:      decl.File,
		Token:     decl.Token,
		Decl:      declIndex,
		Symbol:    decl.Symbol,
		Alias:     decl.Alias,
		TypeStart: decl.TypeStart,
		TypeEnd:   decl.TypeEnd,
	}
	if out.Kind == TypeStruct {
		open := findTypeTopLevelChar(file, decl.TypeStart, decl.TypeEnd, '{')
		close := findTypeMatching(file, open, '{', '}')
		if open >= 0 && close > open && close <= decl.TypeEnd {
			out.Fields = parseStructFields(file, open+1, close-1)
		}
	} else if out.Kind == TypeInterface {
		open := findTypeTopLevelChar(file, decl.TypeStart, decl.TypeEnd, '{')
		close := findTypeMatching(file, open, '{', '}')
		if open >= 0 && close > open && close <= decl.TypeEnd {
			out.InterfaceMethods, out.InterfaceEmbeds = parseInterfaceElements(file, open+1, close-1)
		}
	}
	return out
}

func classifyType(file syntax.File, start int, end int) int {
	if start < 0 || start >= end || start >= len(file.Tokens) {
		return TypeOther
	}
	if file.Tokens[start].Kind == syntax.TokenStruct {
		return TypeStruct
	}
	if file.Tokens[start].Kind == syntax.TokenInterface {
		return TypeInterface
	}
	if file.Tokens[start].Kind == syntax.TokenMap {
		return TypeMap
	}
	if file.Tokens[start].Kind == syntax.TokenFunc {
		return TypeFunc
	}
	if tokCharIs(file, start, '*') {
		return TypePointer
	}
	if tokCharIs(file, start, '[') {
		if start+1 < end && tokCharIs(file, start+1, ']') {
			return TypeSlice
		}
		return TypeArray
	}
	if file.Tokens[start].Kind == syntax.TokenIdent {
		return TypeNamed
	}
	return TypeOther
}

func parseStructFields(file syntax.File, start int, end int) []Field {
	var fields []Field
	i := start
	for i < end {
		if tokCharIs(file, i, ';') {
			i++
			continue
		}
		fieldEnd := nextStructFieldEnd(file, i, end)
		first, last := trimFieldSpan(file, i, fieldEnd)
		if first < last {
			if file.Tokens[last-1].Kind == syntax.TokenString {
				last--
			}
			fields = append(fields, parseFieldList(file, first, last)...)
		}
		if fieldEnd <= i {
			i++
		} else {
			i = fieldEnd
		}
	}
	return fields
}

func nextStructFieldEnd(file syntax.File, start int, end int) int {
	parenDepth := 0
	bracketDepth := 0
	braceDepth := 0
	i := start
	for i < end {
		if i > start && parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 && file.Tokens[i].Line != file.Tokens[i-1].Line {
			return i
		}
		if parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 && tokCharIs(file, i, ';') {
			return i
		}
		if tokCharIs(file, i, '(') {
			parenDepth++
		} else if tokCharIs(file, i, ')') {
			if parenDepth > 0 {
				parenDepth--
			}
		} else if tokCharIs(file, i, '[') {
			bracketDepth++
		} else if tokCharIs(file, i, ']') {
			if bracketDepth > 0 {
				bracketDepth--
			}
		} else if tokCharIs(file, i, '{') {
			braceDepth++
		} else if tokCharIs(file, i, '}') {
			if braceDepth > 0 {
				braceDepth--
			}
		}
		i++
	}
	return end
}

func findTypeTopLevelChar(file syntax.File, start int, end int, c byte) int {
	parenDepth := 0
	bracketDepth := 0
	for i := start; i < end; i++ {
		if parenDepth == 0 && bracketDepth == 0 && tokCharIs(file, i, c) {
			return i
		}
		if tokCharIs(file, i, '(') {
			parenDepth++
		} else if tokCharIs(file, i, ')') {
			if parenDepth > 0 {
				parenDepth--
			}
		} else if tokCharIs(file, i, '[') {
			bracketDepth++
		} else if tokCharIs(file, i, ']') {
			if bracketDepth > 0 {
				bracketDepth--
			}
		}
	}
	return -1
}

func findTypeMatching(file syntax.File, open int, left byte, right byte) int {
	if open < 0 || !tokCharIs(file, open, left) {
		return -1
	}
	depth := 0
	for i := open; i < len(file.Tokens); i++ {
		if tokCharIs(file, i, left) {
			depth++
		} else if tokCharIs(file, i, right) {
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return -1
}

func sortTypes(types []TypeInfo) {
	for i := 1; i < len(types); i++ {
		item := types[i]
		j := i - 1
		for j >= 0 && typeAfter(types[j], item) {
			types[j+1] = types[j]
			j--
		}
		types[j+1] = item
	}
}

func typeAfter(left TypeInfo, right TypeInfo) bool {
	if left.Name != right.Name {
		return left.Name > right.Name
	}
	if left.File != right.File {
		return left.File > right.File
	}
	return left.Token > right.Token
}
