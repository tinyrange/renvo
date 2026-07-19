package check

import "renvo.dev/internal/syntax"

type InterfaceMethod struct {
	Name      string
	NameTok   int
	Signature FuncSignature
}

type InterfaceEmbed struct {
	TypeStart int
	TypeEnd   int
}

func LookupInterfaceMethod(methods []InterfaceMethod, name string) int {
	for i := 0; i < len(methods); i++ {
		if methods[i].Name == name {
			return i
		}
	}
	return -1
}

func parseInterfaceElements(file syntax.File, start int, end int) ([]InterfaceMethod, []InterfaceEmbed) {
	var methods []InterfaceMethod
	var embeds []InterfaceEmbed
	i := start
	for i < end {
		if tokCharIs(&file, i, ';') {
			i++
			continue
		}
		elemEnd := nextInterfaceElementEnd(file, i, end)
		first, last := trimFieldSpan(file, i, elemEnd)
		if first < last {
			if isInterfaceMethodSpec(file, first, last) {
				methods = append(methods, parseInterfaceMethod(file, first, last))
			} else {
				embeds = append(embeds, InterfaceEmbed{TypeStart: first, TypeEnd: last})
			}
		}
		if elemEnd <= i {
			i++
		} else {
			i = elemEnd
		}
	}
	sortInterfaceMethods(methods)
	return methods, embeds
}

func isInterfaceMethodSpec(file syntax.File, start int, end int) bool {
	return start+1 < end && file.Tokens[start].KindLine&255 == syntax.TokenIdent && tokCharIs(&file, start+1, '(')
}

func parseInterfaceMethod(file syntax.File, start int, end int) InterfaceMethod {
	paramsStart := start + 1
	paramsEnd := findTypeMatching(file, paramsStart, '(', ')')
	if paramsEnd < 0 || paramsEnd > end {
		paramsEnd = paramsStart + 1
	}
	return InterfaceMethod{
		Name:      tokenString(&file, start),
		NameTok:   start,
		Signature: buildSignatureFromParts(file, -1, -1, paramsStart, paramsEnd, paramsEnd, end),
	}
}

func nextInterfaceElementEnd(file syntax.File, start int, end int) int {
	parenDepth := 0
	bracketDepth := 0
	braceDepth := 0
	i := start
	for i < end {
		if i > start && parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 && file.Tokens[i].KindLine>>8 != file.Tokens[i-1].KindLine>>8 {
			return i
		}
		if parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 && tokCharIs(&file, i, ';') {
			return i
		}
		if tokCharIs(&file, i, '(') {
			parenDepth++
		} else if tokCharIs(&file, i, ')') {
			if parenDepth > 0 {
				parenDepth--
			}
		} else if tokCharIs(&file, i, '[') {
			bracketDepth++
		} else if tokCharIs(&file, i, ']') {
			if bracketDepth > 0 {
				bracketDepth--
			}
		} else if tokCharIs(&file, i, '{') {
			braceDepth++
		} else if tokCharIs(&file, i, '}') {
			if braceDepth > 0 {
				braceDepth--
			}
		}
		i++
	}
	return end
}

func sortInterfaceMethods(methods []InterfaceMethod) {
	for i := 1; i < len(methods); i++ {
		item := methods[i]
		j := i - 1
		for j >= 0 && checkStringAfter(methods[j].Name, item.Name) {
			methods[j+1] = methods[j]
			j--
		}
		methods[j+1] = item
	}
}
