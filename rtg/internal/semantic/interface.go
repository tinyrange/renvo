package semantic

import (
	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtg/internal/syntax"
)

type interfaceMethod struct {
	name      semanticText
	signature semanticTokens
}

type interfaceType struct {
	pkg       int
	file      int
	name      semanticText
	start     int
	end       int
	methods   []interfaceMethod
	candidate semanticText
}

type concreteMethod struct {
	pkg       int
	receiver  semanticText
	pointer   bool
	name      semanticText
	signature semanticTokens
}

type concreteType struct {
	pkg  int
	name semanticText
}

// semanticText and semanticTokens retain spans in the already-loaded source
// instead of materializing a string for every token inspected by the pass.
// That distinction matters to the self-hosted compiler: its bump arena cannot
// reclaim short-lived strings until the frontend phase finishes.
type semanticText struct {
	src   []byte
	start int
	end   int
}

type semanticTokens struct {
	src    []byte
	tokens []syntax.Token
	start  int
	end    int
}

type sourceEdit struct {
	start       int
	end         int
	literal     string
	replacement semanticText
}

// LowerInterfaces performs closed-world devirtualization only when an
// interface has exactly one concrete implementation in its package. The
// result is ordinary Go source, so normal checking still validates it.
func LowerInterfaces(graph *load.Graph) {
	if !graph.Ok {
		return
	}
	interfaces, methods, concrete := collectInterfaceTypes(graph)
	for i := 0; i < len(interfaces); i++ {
		candidate := semanticText{}
		for j := 0; j < len(concrete); j++ {
			if concrete[j].pkg != interfaces[i].pkg || !implementsInterface(methods, &concrete[j], &interfaces[i]) {
				continue
			}
			if semanticTextValid(candidate) {
				candidate = semanticText{}
				break
			}
			candidate = concrete[j].name
		}
		interfaces[i].candidate = candidate
	}
	for pkgIndex := 0; pkgIndex < len(graph.Packages); pkgIndex++ {
		for fileIndex := 0; fileIndex < len(graph.Packages[pkgIndex].Files); fileIndex++ {
			file := &graph.Packages[pkgIndex].Files[fileIndex]
			edits := interfaceFileEdits(&file.File, interfaces, pkgIndex, fileIndex)
			if len(edits) == 0 {
				continue
			}
			file.Src = applySourceEdits(file.Src, edits)
			file.File = syntax.ParseFile(file.Src)
		}
	}
}

func collectInterfaceTypes(graph *load.Graph) ([]interfaceType, []concreteMethod, []concreteType) {
	var interfaces []interfaceType
	var methods []concreteMethod
	var concrete []concreteType
	for pkgIndex := 0; pkgIndex < len(graph.Packages); pkgIndex++ {
		pkg := graph.Packages[pkgIndex]
		for fileIndex := 0; fileIndex < len(pkg.Files); fileIndex++ {
			file := pkg.Files[fileIndex].File
			for i := 0; i < len(file.Decls); i++ {
				decl := file.Decls[i]
				if decl.Kind != syntax.TokenType {
					continue
				}
				name := semanticTokenText(&file, decl.NameTok)
				typeStart := decl.NameTok + 1
				if semanticTokenEquals(&file, typeStart, "=") {
					typeStart++
				}
				if typeStart < decl.EndTok && file.Tokens[typeStart].Kind == syntax.TokenInterface && semanticTokenEquals(&file, typeStart+1, "{") {
					closeTok := semanticMatching(&file, typeStart+1, "{", "}")
					if closeTok > typeStart+1 {
						interfaces = append(interfaces, interfaceType{pkg: pkgIndex, file: fileIndex, name: name, start: file.Tokens[typeStart].Start, end: file.Tokens[closeTok].End, methods: semanticInterfaceMethods(&file, typeStart+2, closeTok)})
					}
				} else if semanticTextValid(name) {
					concrete = append(concrete, concreteType{pkg: pkgIndex, name: name})
				}
			}
			for i := 0; i < len(file.Funcs); i++ {
				fn := file.Funcs[i]
				if fn.ReceiverStart < 0 {
					continue
				}
				receiver := semanticText{}
				pointer := false
				for tok := fn.ReceiverStart; tok < fn.ReceiverEnd; tok++ {
					if semanticTokenEquals(&file, tok, "*") {
						pointer = true
					}
					if file.Tokens[tok].Kind == syntax.TokenIdent {
						receiver = semanticTokenText(&file, tok)
					}
				}
				openTok := fn.NameTok + 1
				closeTok := semanticMatching(&file, openTok, "(", ")")
				if semanticTextValid(receiver) && closeTok > openTok {
					methods = append(methods, concreteMethod{pkg: pkgIndex, receiver: receiver, pointer: pointer, name: semanticTokenText(&file, fn.NameTok), signature: semanticTokenSequence(&file, openTok, fn.BodyStart)})
				}
			}
		}
	}
	return interfaces, methods, concrete
}

func semanticInterfaceMethods(file *syntax.File, start int, end int) []interfaceMethod {
	var out []interfaceMethod
	for i := start; i+1 < end; i++ {
		if file.Tokens[i].Kind != syntax.TokenIdent || !semanticTokenEquals(file, i+1, "(") {
			continue
		}
		closeTok := semanticMatching(file, i+1, "(", ")")
		if closeTok < i+1 {
			return nil
		}
		resultEnd := closeTok + 1
		for resultEnd < end && !semanticTokenEquals(file, resultEnd, ";") && file.Tokens[resultEnd].Line == file.Tokens[closeTok].Line {
			resultEnd++
		}
		out = append(out, interfaceMethod{name: semanticTokenText(file, i), signature: semanticTokenSequence(file, i+1, resultEnd)})
		i = resultEnd
	}
	return out
}

func implementsInterface(methods []concreteMethod, concrete *concreteType, iface *interfaceType) bool {
	if len(iface.methods) == 0 {
		return false
	}
	for i := 0; i < len(iface.methods); i++ {
		found := false
		for j := 0; j < len(methods); j++ {
			method := methods[j]
			if method.pkg == concrete.pkg && semanticTextEqual(method.receiver, concrete.name) && !method.pointer && semanticTextEqual(method.name, iface.methods[i].name) && semanticTokensEqual(method.signature, iface.methods[i].signature) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func interfaceFileEdits(file *syntax.File, interfaces []interfaceType, pkgIndex int, fileIndex int) []sourceEdit {
	var edits []sourceEdit
	var names []semanticText
	var candidates []semanticText
	for i := 0; i < len(interfaces); i++ {
		iface := interfaces[i]
		if iface.pkg != pkgIndex || !semanticTextValid(iface.candidate) {
			continue
		}
		names = append(names, iface.name)
		candidates = append(candidates, iface.candidate)
		if iface.file == fileIndex {
			edits = append(edits, sourceEdit{start: iface.start, end: iface.end, literal: "= ", replacement: iface.candidate})
		}
	}
	if len(names) == 0 {
		return edits
	}
	var values []semanticText
	for i := 0; i+1 < len(file.Tokens); i++ {
		nameIndex := semanticTextIndexToken(names, file, i+1)
		if file.Tokens[i].Kind == syntax.TokenIdent && nameIndex >= 0 {
			values = semanticAppendUnique(values, semanticTokenText(file, i))
			edits = append(edits, sourceEdit{start: file.Tokens[i+1].Start, end: file.Tokens[i+1].End, replacement: candidates[nameIndex]})
		}
	}
	for i := 1; i+4 < len(file.Tokens); i++ {
		if semanticTextIndexToken(values, file, i-1) < 0 || !semanticTokenEquals(file, i, ".") || !semanticTokenEquals(file, i+1, "(") || !semanticTokenEquals(file, i+3, ")") {
			continue
		}
		if semanticTokenEquals(file, i+2, "type") {
			if i >= 2 && file.Tokens[i-2].Kind == syntax.TokenSwitch {
				closeBrace := semanticMatching(file, i+4, "{", "}")
				if closeBrace > i+4 {
					edits = append(edits, sourceEdit{start: file.Tokens[i-1].Start, end: file.Tokens[i+3].End, literal: "1"})
					edits = append(edits, semanticTypeSwitchCaseEdits(file, i+5, closeBrace, candidates)...)
				}
			}
			continue
		}
		if semanticTextIndexToken(candidates, file, i+2) < 0 {
			continue
		}
		replacement := ""
		if semanticCommaOK(file, i-1) {
			replacement = ", true"
		}
		edits = append(edits, sourceEdit{start: file.Tokens[i].Start, end: file.Tokens[i+3].End, literal: replacement})
	}
	return edits
}

func semanticTypeSwitchCaseEdits(file *syntax.File, start int, end int, candidates []semanticText) []sourceEdit {
	var edits []sourceEdit
	for i := start; i+1 < end; i++ {
		if file.Tokens[i].Kind != syntax.TokenCase || file.Tokens[i+1].Kind != syntax.TokenIdent {
			continue
		}
		value := "0"
		if semanticTextIndexToken(candidates, file, i+1) >= 0 {
			value = "1"
		}
		edits = append(edits, sourceEdit{start: file.Tokens[i+1].Start, end: file.Tokens[i+1].End, literal: value})
	}
	return edits
}

func semanticCommaOK(file *syntax.File, rhs int) bool {
	assignment := false
	for i := rhs - 1; i >= 0 && file.Tokens[i].Line == file.Tokens[rhs].Line; i-- {
		if semanticTokenEquals(file, i, ":=") || semanticTokenEquals(file, i, "=") {
			assignment = true
			continue
		}
		if assignment && semanticTokenEquals(file, i, ",") {
			return true
		}
	}
	return false
}

func applySourceEdits(src []byte, edits []sourceEdit) []byte {
	for i := 1; i < len(edits); i++ {
		item := edits[i]
		j := i - 1
		for j >= 0 && edits[j].start > item.start {
			edits[j+1] = edits[j]
			j--
		}
		edits[j+1] = item
	}
	out := make([]byte, 0, len(src))
	at := 0
	for i := 0; i < len(edits); i++ {
		edit := edits[i]
		if edit.start < at || edit.end < edit.start || edit.end > len(src) {
			continue
		}
		out = append(out, src[at:edit.start]...)
		out = append(out, edit.literal...)
		if semanticTextValid(edit.replacement) {
			out = append(out, edit.replacement.src[edit.replacement.start:edit.replacement.end]...)
		}
		at = edit.end
	}
	return append(out, src[at:]...)
}

func semanticMatching(file *syntax.File, open int, left string, right string) int {
	if !semanticTokenEquals(file, open, left) {
		return -1
	}
	depth := 0
	for i := open; i < len(file.Tokens); i++ {
		if semanticTokenEquals(file, i, left) {
			depth++
		} else if semanticTokenEquals(file, i, right) {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func semanticTokenSequence(file *syntax.File, start int, end int) semanticTokens {
	return semanticTokens{src: file.Src, tokens: file.Tokens, start: start, end: end}
}

func semanticTokenText(file *syntax.File, index int) semanticText {
	return semanticTokenTextFrom(file.Src, file.Tokens, index)
}

func semanticTokenTextFrom(src []byte, tokens []syntax.Token, index int) semanticText {
	if index < 0 || index >= len(tokens) {
		return semanticText{}
	}
	tok := tokens[index]
	if tok.Start < 0 || tok.End < tok.Start || tok.End > len(src) {
		return semanticText{}
	}
	return semanticText{src: src, start: tok.Start, end: tok.End}
}

func semanticTokenEquals(file *syntax.File, index int, value string) bool {
	return semanticTextEqualsString(semanticTokenText(file, index), value)
}

func semanticTextValid(value semanticText) bool {
	return value.src != nil && value.start >= 0 && value.end > value.start && value.end <= len(value.src)
}

func semanticTextEqual(left semanticText, right semanticText) bool {
	if !semanticTextValid(left) || !semanticTextValid(right) || left.end-left.start != right.end-right.start {
		return false
	}
	for i := 0; i < left.end-left.start; i++ {
		if left.src[left.start+i] != right.src[right.start+i] {
			return false
		}
	}
	return true
}

func semanticTextEqualsString(left semanticText, right string) bool {
	if !semanticTextValid(left) || left.end-left.start != len(right) {
		return false
	}
	for i := 0; i < len(right); i++ {
		if left.src[left.start+i] != right[i] {
			return false
		}
	}
	return true
}

func semanticTokensEqual(left semanticTokens, right semanticTokens) bool {
	leftCount := left.end - left.start
	if leftCount < 0 || leftCount != right.end-right.start {
		return false
	}
	for i := 0; i < leftCount; i++ {
		leftText := semanticTokenTextFrom(left.src, left.tokens, left.start+i)
		rightText := semanticTokenTextFrom(right.src, right.tokens, right.start+i)
		if !semanticTextEqual(leftText, rightText) {
			return false
		}
	}
	return true
}

func semanticTextIndexToken(values []semanticText, file *syntax.File, token int) int {
	value := semanticTokenText(file, token)
	return semanticTextIndex(values, value)
}

func semanticTextIndex(values []semanticText, value semanticText) int {
	for i := 0; i < len(values); i++ {
		if semanticTextEqual(values[i], value) {
			return i
		}
	}
	return -1
}

func semanticAppendUnique(values []semanticText, value semanticText) []semanticText {
	if semanticTextValid(value) && semanticTextIndex(values, value) < 0 {
		return append(values, value)
	}
	return values
}
