//go:build rtg

package link

import (
	"j5.nz/rtg/rtg/internal/build"
	"j5.nz/rtg/rtg/internal/unit"
)

const (
	LinkOK = iota
	LinkErrBuild
	LinkErrRoot
	LinkErrUnit
)

type Result struct {
	Program      unit.Program
	Data         []byte
	Ok           bool
	Error        int
	ErrorPackage int
}

func LinkBuildCore(result build.Result) Result {
	out := Result{Ok: true, Error: LinkOK, ErrorPackage: -1}
	if !result.Ok {
		out.Ok = false
		out.Error = LinkErrBuild
		out.ErrorPackage = result.ErrorPackage
		return out
	}
	if result.Root < 0 || result.Root >= len(result.Units) {
		out.Ok = false
		out.Error = LinkErrRoot
		return out
	}
	program, ok := LinkUnitsCore(result.Units, result.Root)
	if !ok {
		out.Ok = false
		out.Error = LinkErrUnit
		return out
	}
	data, ok := unit.Marshal(program)
	if !ok {
		out.Ok = false
		out.Error = LinkErrUnit
		return out
	}
	out.Program = program
	out.Data = data
	return out
}

func LinkUnitsCore(units []build.PackageUnit, root int) (unit.Program, bool) {
	var empty unit.Program
	if root < 0 || root >= len(units) {
		return empty, false
	}
	programs := make([]unit.Program, len(units))
	for i := 0; i < len(units); i++ {
		programs[i] = units[i].Program
	}
	return LinkProgramsCore(programs, root, units[root].Name)
}

func LinkProgramsCore(programs []unit.Program, root int, rootName string) (unit.Program, bool) {
	var empty unit.Program
	if root < 0 || root >= len(programs) || rootName == "" {
		return empty, false
	}
	programs, ok := prepareProgramsCore(programs, root)
	if !ok {
		return empty, false
	}
	ensureCoreProgramSymbols(programs)
	program := unit.Program{Package: rootName, ImportPath: programs[root].ImportPath}
	reserveCoreLinkedProgram(&program, programs)
	symbolOffsets := packageSymbolOffsets(programs)
	aliases := packageSymbolAliases(programs, root, symbolOffsets)
	actions, actionOffsets := linkedProgramActions(programs, aliases, symbolOffsets)
	finalEOF := countCoreLinkedEOF(programs, actions, actionOffsets)
	if finalEOF < 0 {
		return empty, false
	}
	lineOffset := 0
	for i := 0; i < len(programs); i++ {
		ok := appendProgramCore(&program, programs[i], finalEOF, lineOffset, actions, actionOffsets[i], aliases, i+1 < len(programs))
		if !ok {
			return empty, false
		}
		lineOffset = nextLineOffset(lineOffset, programs[i].Text, i+1 < len(programs))
	}
	program.Tokens = append(program.Tokens, unit.Token{
		Kind:  unit.TokenEOF,
		Start: len(program.Text),
		Size:  0,
		Line:  lineOffset + 1,
	})
	return program, true
}

func ensureCoreProgramSymbols(programs []unit.Program) {
	for i := 0; i < len(programs); i++ {
		if len(programs[i].Symbols) == 0 {
			programs[i].Symbols = synthesizeCoreSymbols(programs[i], i)
		}
	}
}

func synthesizeCoreSymbols(program unit.Program, pkg int) []unit.Symbol {
	out := make([]unit.Symbol, 0, len(program.Decls)+len(program.Funcs))
	for i := 0; i < len(program.Decls); i++ {
		decl := program.Decls[i]
		var symbol unit.Symbol
		symbol.Name = coreText(program.Text, decl.NameStart, decl.NameEnd)
		symbol.Package = pkg
		symbol.Token = coreTokenAt(program, decl.NameStart, decl.NameEnd)
		out = append(out, symbol)
	}
	for i := 0; i < len(program.Funcs); i++ {
		fn := program.Funcs[i]
		var symbol unit.Symbol
		symbol.Name = coreText(program.Text, fn.NameStart, fn.NameEnd)
		symbol.Package = pkg
		symbol.Token = fn.NameTok
		out = append(out, symbol)
	}
	return out
}

func coreText(text []byte, start int, end int) string {
	if start < 0 || end < start || end > len(text) {
		return ""
	}
	return string(text[start:end])
}

func coreTokenAt(program unit.Program, start int, end int) int {
	for i := 0; i < len(program.Tokens); i++ {
		tok := program.Tokens[i]
		if tok.Start == start && tok.Start+tok.Size == end {
			return i
		}
	}
	return -1
}

func reserveCoreLinkedProgram(program *unit.Program, programs []unit.Program) {
	textCap := 0
	tokenCap := 1
	declCap := 0
	funcCap := 0
	for i := 0; i < len(programs); i++ {
		p := programs[i]
		textCap += len(p.Text) + 1
		tokenCap += len(p.Tokens)
		declCap += len(p.Decls)
		funcCap += len(p.Funcs)
	}
	program.Text = make([]byte, 0, textCap)
	program.Tokens = make([]unit.Token, 0, tokenCap)
	program.Decls = make([]unit.Decl, 0, declCap)
	program.Funcs = make([]unit.Func, 0, funcCap)
}

func prepareProgramsCore(programs []unit.Program, root int) ([]unit.Program, bool) {
	out := make([]unit.Program, len(programs))
	copy(out, programs)
	rootProgram, ok := addRootEntrypointCore(out[root], root)
	if !ok {
		return nil, false
	}
	out[root] = rootProgram
	return out, true
}

func addRootEntrypointCore(src unit.Program, packageIndex int) (unit.Program, bool) {
	if src.Package != "main" || findFuncByName(src, "appMain") >= 0 || findFuncByName(src, "main") < 0 {
		return src, true
	}
	if len(src.Tokens) == 0 || src.Tokens[len(src.Tokens)-1].Kind != unit.TokenEOF {
		return src, false
	}
	src.Tokens = copyTokens(src.Tokens, len(src.Tokens)-1)
	if len(src.Text) > 0 && src.Text[len(src.Text)-1] != '\n' {
		src.Text = append(src.Text, '\n')
	}
	start := len(src.Text)
	line := countNewlines(src.Text) + 1
	src.Text = appendStringBytes(src.Text, "func appMain() int { main(); return 0 }\n")
	base := len(src.Tokens)
	src.Tokens = append(src.Tokens, unit.Token{Kind: unit.TokenFunc, Start: start, Size: 4, Line: line})
	src.Tokens = append(src.Tokens, unit.Token{Kind: unit.TokenIdent, Start: start + 5, Size: 7, Line: line})
	src.Tokens = append(src.Tokens, unit.Token{Kind: unit.TokenOp, Start: start + 12, Size: 1, Line: line})
	src.Tokens = append(src.Tokens, unit.Token{Kind: unit.TokenOp, Start: start + 13, Size: 1, Line: line})
	src.Tokens = append(src.Tokens, unit.Token{Kind: unit.TokenIdent, Start: start + 15, Size: 3, Line: line})
	src.Tokens = append(src.Tokens, unit.Token{Kind: unit.TokenOp, Start: start + 19, Size: 1, Line: line})
	src.Tokens = append(src.Tokens, unit.Token{Kind: unit.TokenIdent, Start: start + 21, Size: 4, Line: line})
	src.Tokens = append(src.Tokens, unit.Token{Kind: unit.TokenOp, Start: start + 25, Size: 1, Line: line})
	src.Tokens = append(src.Tokens, unit.Token{Kind: unit.TokenOp, Start: start + 26, Size: 1, Line: line})
	src.Tokens = append(src.Tokens, unit.Token{Kind: unit.TokenOp, Start: start + 27, Size: 1, Line: line})
	src.Tokens = append(src.Tokens, unit.Token{Kind: unit.TokenReturn, Start: start + 29, Size: 6, Line: line})
	src.Tokens = append(src.Tokens, unit.Token{Kind: unit.TokenNumber, Start: start + 36, Size: 1, Line: line})
	src.Tokens = append(src.Tokens, unit.Token{Kind: unit.TokenOp, Start: start + 38, Size: 1, Line: line})
	eof := len(src.Tokens)
	src.Tokens = append(src.Tokens, unit.Token{Kind: unit.TokenEOF, Start: len(src.Text), Size: 0, Line: countNewlines(src.Text) + 1})
	src.Funcs = append(src.Funcs, unit.Func{
		NameStart:     start + 5,
		NameEnd:       start + 12,
		StartTok:      base,
		NameTok:       base + 1,
		ReceiverStart: eof,
		ReceiverEnd:   eof,
		BodyStart:     base + 5,
		BodyEnd:       base + 12,
		EndTok:        base + 13,
	})
	_ = packageIndex
	return src, true
}

func appendProgramCore(dst *unit.Program, src unit.Program, finalEOF int, lineOffset int, actions []int, actionStart int, aliases []string, hasNext bool) bool {
	if src.Package == "" || len(src.Text) == 0 || len(src.Tokens) == 0 {
		return false
	}
	if actionStart < 0 || actionStart+len(src.Tokens) > len(actions) {
		return false
	}
	prevEnd := 0
	for i := 0; i < len(src.Tokens); i++ {
		actionIndex := actionStart + i
		action := actions[actionStart+i]
		tok := src.Tokens[i]
		if tok.Kind == unit.TokenEOF {
			actions[actionIndex] = finalEOF
			continue
		}
		tokStart := tok.Start
		tokEnd := tok.Start + tok.Size
		if tokenActionSkips(action) {
			if tokenActionRedirect(action) >= 0 && tok.Start > prevEnd {
				dst.Text = appendBytes(dst.Text, src.Text[prevEnd:tok.Start])
			}
			if tokEnd > prevEnd {
				prevEnd = tokEnd
			}
			if tokenActionRedirect(action) < 0 {
				actions[actionIndex] = finalEOF
			}
			continue
		}
		if tok.Start > prevEnd {
			dst.Text = appendBytes(dst.Text, src.Text[prevEnd:tok.Start])
		}
		mappedToken := len(dst.Tokens)
		tok.Start = len(dst.Text)
		replacementIndex := tokenActionReplacement(action)
		if replacementIndex >= 0 {
			replacement := aliases[replacementIndex]
			dst.Text = appendStringBytes(dst.Text, replacement)
			tok.Size = len(replacement)
		} else {
			dst.Text = appendBytes(dst.Text, src.Text[tokStart:tokEnd])
		}
		tok.Line += lineOffset
		dst.Tokens = append(dst.Tokens, tok)
		actions[actionIndex] = mappedToken
		prevEnd = tokEnd
	}
	if prevEnd < len(src.Text) {
		dst.Text = appendBytes(dst.Text, src.Text[prevEnd:])
	}
	for i := 0; i < len(src.Tokens); i++ {
		actionIndex := actionStart + i
		target := tokenActionRedirect(actions[actionIndex])
		if target >= 0 {
			actions[actionIndex] = mapLinkedToken(actions, actionStart, len(src.Tokens), target, finalEOF)
		}
	}
	for i := 0; i < len(src.Decls); i++ {
		decl := src.Decls[i]
		decl.StartTok = mapLinkedToken(actions, actionStart, len(src.Tokens), decl.StartTok, finalEOF)
		decl.EndTok = mapLinkedToken(actions, actionStart, len(src.Tokens), decl.EndTok, finalEOF)
		nameStart, nameEnd, ok := mapTextSpanByToken(src, dst, actions, actionStart, finalEOF, decl.NameStart, decl.NameEnd)
		if !ok {
			return false
		}
		decl.NameStart = nameStart
		decl.NameEnd = nameEnd
		dst.Decls = append(dst.Decls, decl)
	}
	for i := 0; i < len(src.Funcs); i++ {
		fn := src.Funcs[i]
		fn.StartTok = mapLinkedToken(actions, actionStart, len(src.Tokens), fn.StartTok, finalEOF)
		fn.NameTok = mapLinkedToken(actions, actionStart, len(src.Tokens), fn.NameTok, finalEOF)
		nameStart, nameEnd, ok := mappedTokenTextSpan(dst, fn.NameTok)
		if !ok {
			return false
		}
		fn.NameStart = nameStart
		fn.NameEnd = nameEnd
		fn.ReceiverStart = mapLinkedToken(actions, actionStart, len(src.Tokens), fn.ReceiverStart, finalEOF)
		fn.ReceiverEnd = mapLinkedToken(actions, actionStart, len(src.Tokens), fn.ReceiverEnd, finalEOF)
		fn.BodyStart = mapLinkedToken(actions, actionStart, len(src.Tokens), fn.BodyStart, finalEOF)
		fn.BodyEnd = mapLinkedToken(actions, actionStart, len(src.Tokens), fn.BodyEnd, finalEOF)
		fn.EndTok = mapLinkedToken(actions, actionStart, len(src.Tokens), fn.EndTok, finalEOF)
		dst.Funcs = append(dst.Funcs, fn)
	}
	if hasNext && (len(src.Text) == 0 || src.Text[len(src.Text)-1] != '\n') {
		dst.Text = append(dst.Text, '\n')
	}
	return true
}

func linkedTokenActions(program unit.Program, aliases []string, symbolOffsets []int, actions []int) bool {
	if len(actions) != len(program.Tokens) {
		return false
	}
	for i := 0; i < len(program.Imports); i++ {
		markImportDeclTokens(program, actions, program.Imports[i])
	}
	for i := 0; i < len(program.Selectors); i++ {
		selector := program.Selectors[i]
		if selector.BaseKind == unit.RefImport {
			markRedirectToken(actions, selector.BaseTok, selector.NameTok)
			markRedirectToken(actions, selector.DotTok, selector.NameTok)
		}
	}
	for i := 0; i < len(program.TypeRefs); i++ {
		ref := program.TypeRefs[i]
		if ref.Kind == unit.TypeRefImportSelector {
			markRedirectToken(actions, ref.BaseTok, ref.Token)
			markRedirectToken(actions, ref.DotTok, ref.Token)
		}
	}
	for i := 0; i < len(program.Calls); i++ {
		call := program.Calls[i]
		if call.Kind == unit.CallImportSelector {
			markRedirectToken(actions, call.BaseTok, call.CalleeTok)
			markRedirectToken(actions, call.DotTok, call.CalleeTok)
		}
	}
	for i := 0; i < len(program.Symbols); i++ {
		symbol := program.Symbols[i]
		index := packageSymbolAliasIndex(aliases, symbolOffsets, symbol.Package, i)
		if index >= 0 {
			markReplacementToken(actions, symbol.Token, index)
		}
	}
	for i := 0; i < len(program.Refs); i++ {
		ref := program.Refs[i]
		if ref.Kind == unit.RefPackage {
			index := packageSymbolAliasIndex(aliases, symbolOffsets, ref.Package, ref.Index)
			if index >= 0 {
				markReplacementToken(actions, ref.Token, index)
			}
		}
	}
	for i := 0; i < len(program.Selectors); i++ {
		selector := program.Selectors[i]
		index := packageSymbolAliasIndex(aliases, symbolOffsets, selector.Package, selector.Symbol)
		if index >= 0 {
			markReplacementToken(actions, selector.NameTok, index)
		}
	}
	for i := 0; i < len(program.TypeRefs); i++ {
		ref := program.TypeRefs[i]
		index := packageSymbolAliasIndex(aliases, symbolOffsets, ref.Package, ref.Symbol)
		if index >= 0 {
			markReplacementToken(actions, ref.Token, index)
		}
	}
	return true
}

func linkedProgramActions(programs []unit.Program, aliases []string, symbolOffsets []int) ([]int, []int) {
	offsets := make([]int, len(programs)+1)
	total := 0
	for i := 0; i < len(programs); i++ {
		offsets[i] = total
		total += len(programs[i].Tokens)
	}
	offsets[len(programs)] = total
	actions := make([]int, total)
	for i := 0; i < len(programs); i++ {
		if !linkedTokenActions(programs[i], aliases, symbolOffsets, actions[offsets[i]:offsets[i+1]]) {
			return nil, nil
		}
	}
	return actions, offsets
}

func markImportDeclTokens(program unit.Program, actions []int, imp unit.Import) {
	if imp.PathTok < 0 || imp.PathTok >= len(program.Tokens) {
		return
	}
	line := program.Tokens[imp.PathTok].Line
	start := imp.PathTok
	if imp.NameTok >= 0 && imp.NameTok < start {
		start = imp.NameTok
	}
	for start > 0 && program.Tokens[start-1].Line == line {
		start--
	}
	end := imp.PathTok
	for end+1 < len(program.Tokens) && program.Tokens[end+1].Line == line {
		end++
	}
	for i := start; i <= end; i++ {
		actions[i] = -1
	}
}

func markRedirectToken(actions []int, tok int, target int) {
	if tok < 0 || tok >= len(actions) || target < 0 || target >= len(actions) {
		return
	}
	actions[tok] = -target - 2
}

func markReplacementToken(actions []int, tok int, replacement int) {
	if tok < 0 || tok >= len(actions) || replacement < 0 || actions[tok] < 0 {
		return
	}
	actions[tok] = replacement + 1
}

func tokenActionSkips(action int) bool {
	return action < 0
}

func tokenActionRedirect(action int) int {
	if action <= -2 {
		return -action - 2
	}
	return -1
}

func tokenActionReplacement(action int) int {
	if action > 0 {
		return action - 1
	}
	return -1
}

func packageSymbolAliases(programs []unit.Program, root int, symbolOffsets []int) []string {
	total := 0
	if len(programs) > 0 {
		last := len(programs) - 1
		total = symbolOffsets[last] + len(programs[last].Symbols)
	}
	out := make([]string, total)
	for i := 0; i < len(programs); i++ {
		if i == root {
			continue
		}
		for j := 0; j < len(programs[i].Symbols); j++ {
			if symbolNeedsAlias(programs, i, j) {
				out[symbolOffsets[i]+j] = symbolAliasName(i, programs[i].Symbols[j].Name)
			}
		}
	}
	return out
}

func symbolNeedsAlias(programs []unit.Program, pkg int, symbol int) bool {
	name := programs[pkg].Symbols[symbol].Name
	for i := 0; i < len(programs); i++ {
		for j := 0; j < len(programs[i].Symbols); j++ {
			if i == pkg && j == symbol {
				continue
			}
			if programs[i].Symbols[j].Name == name {
				return true
			}
		}
	}
	return false
}

func packageSymbolAlias(aliases []string, symbolOffsets []int, pkg int, symbol int) string {
	index := packageSymbolAliasIndex(aliases, symbolOffsets, pkg, symbol)
	if index < 0 {
		return ""
	}
	return aliases[index]
}

func packageSymbolAliasIndex(aliases []string, symbolOffsets []int, pkg int, symbol int) int {
	if pkg < 0 || pkg >= len(symbolOffsets) || symbol < 0 {
		return -1
	}
	index := symbolOffsets[pkg] + symbol
	if index < 0 || index >= len(aliases) {
		return -1
	}
	if aliases[index] == "" {
		return -1
	}
	return index
}

func symbolAliasName(pkg int, name string) string {
	out := []byte("rtgp")
	out = appendInt(out, pkg)
	out = append(out, '_')
	for i := 0; i < len(name); i++ {
		c := name[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			out = append(out, c)
		} else {
			out = append(out, '_')
		}
	}
	return string(out)
}

func appendInt(out []byte, value int) []byte {
	if value == 0 {
		return append(out, '0')
	}
	var digits []byte
	for value > 0 {
		digits = append(digits, byte('0'+value%10))
		value = value / 10
	}
	for i := len(digits) - 1; i >= 0; i-- {
		out = append(out, digits[i])
	}
	return out
}

func copyTokens(src []unit.Token, limit int) []unit.Token {
	var out []unit.Token
	for i := 0; i < limit && i < len(src); i++ {
		out = append(out, src[i])
	}
	return out
}

func findFuncByName(program unit.Program, name string) int {
	for i := 0; i < len(program.Funcs); i++ {
		fn := program.Funcs[i]
		if linkedProgramText(program, fn.NameStart, fn.NameEnd) == name {
			return i
		}
	}
	return -1
}

func linkedProgramText(program unit.Program, start int, end int) string {
	if start < 0 || end < start || end > len(program.Text) {
		return ""
	}
	return string(program.Text[start:end])
}

func mapTextSpanByToken(src unit.Program, dst *unit.Program, tokenMap []int, tokenStart int, eof int, start int, end int) (int, int, bool) {
	for i := 0; i < len(src.Tokens); i++ {
		tok := src.Tokens[i]
		if tok.Start == start && tok.Start+tok.Size == end {
			mapped := mapLinkedToken(tokenMap, tokenStart, len(src.Tokens), i, eof)
			return mappedTokenTextSpan(dst, mapped)
		}
	}
	return 0, 0, false
}

func mappedTokenTextSpan(program *unit.Program, tok int) (int, int, bool) {
	if tok < 0 || tok >= len(program.Tokens) {
		return 0, 0, false
	}
	token := program.Tokens[tok]
	if token.Kind == unit.TokenEOF || token.Start < 0 || token.Start+token.Size > len(program.Text) {
		return 0, 0, false
	}
	return token.Start, token.Start + token.Size, true
}

func mapLinkedToken(tokenMap []int, tokenStart int, tokenCount int, tok int, eof int) int {
	if tok < 0 {
		return eof
	}
	if tok >= tokenCount || tokenStart < 0 || tokenStart+tok >= len(tokenMap) {
		return -1
	}
	mapped := tokenMap[tokenStart+tok]
	if mapped < 0 {
		return -1
	}
	return mapped
}

func countCoreLinkedEOF(programs []unit.Program, actions []int, actionOffsets []int) int {
	total := 0
	for i := 0; i < len(programs); i++ {
		if i+1 >= len(actionOffsets) || actionOffsets[i] < 0 || actionOffsets[i+1] > len(actions) || actionOffsets[i+1]-actionOffsets[i] != len(programs[i].Tokens) {
			return -1
		}
		for j := 0; j < len(programs[i].Tokens); j++ {
			if programs[i].Tokens[j].Kind != unit.TokenEOF && !tokenActionSkips(actions[actionOffsets[i]+j]) {
				total++
			}
		}
	}
	return total
}

func packageSymbolOffsets(programs []unit.Program) []int {
	out := make([]int, len(programs))
	next := 0
	for i := 0; i < len(programs); i++ {
		out[i] = next
		next += len(programs[i].Symbols)
	}
	return out
}

func nextLineOffset(lineOffset int, text []byte, hasNext bool) int {
	lineOffset += countNewlines(text)
	if hasNext && (len(text) == 0 || text[len(text)-1] != '\n') {
		lineOffset++
	}
	return lineOffset
}

func countNewlines(text []byte) int {
	count := 0
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			count++
		}
	}
	return count
}

func appendBytes(out []byte, data []byte) []byte {
	for i := 0; i < len(data); i++ {
		out = append(out, data[i])
	}
	return out
}

func appendStringBytes(out []byte, data string) []byte {
	for i := 0; i < len(data); i++ {
		out = append(out, data[i])
	}
	return out
}
