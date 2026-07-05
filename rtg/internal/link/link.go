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
	program := unit.Program{Package: rootName, ImportPath: programs[root].ImportPath}
	reserveCoreLinkedProgram(&program, programs)
	finalEOF := countCoreLinkedEOF(programs)
	symbolOffsets := packageSymbolOffsets(programs)
	aliases := packageSymbolAliases(programs, root, symbolOffsets)
	lineOffset := 0
	for i := 0; i < len(programs); i++ {
		ok := appendProgramCore(&program, programs[i], finalEOF, lineOffset, symbolOffsets, aliases, i+1 < len(programs))
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

func appendProgramCore(dst *unit.Program, src unit.Program, finalEOF int, lineOffset int, symbolOffsets []int, aliases []string, hasNext bool) bool {
	if src.Package == "" || len(src.Text) == 0 || len(src.Tokens) == 0 {
		return false
	}
	oldToNew := make([]int, len(src.Tokens))
	skip, redirect := linkedTokenSkip(src)
	replacements := linkedTokenReplacements(src, aliases, symbolOffsets)
	prevEnd := 0
	for i := 0; i < len(src.Tokens); i++ {
		tok := src.Tokens[i]
		if tok.Kind == unit.TokenEOF {
			oldToNew[i] = finalEOF
			continue
		}
		tokStart := tok.Start
		tokEnd := tok.Start + tok.Size
		if skip[i] {
			oldToNew[i] = finalEOF
			if tokEnd > prevEnd {
				prevEnd = tokEnd
			}
			continue
		}
		if tok.Start > prevEnd {
			dst.Text = appendBytes(dst.Text, src.Text[prevEnd:tok.Start])
		}
		oldToNew[i] = len(dst.Tokens)
		tok.Start = len(dst.Text)
		if replacements[i] != "" {
			dst.Text = appendStringBytes(dst.Text, replacements[i])
			tok.Size = len(replacements[i])
		} else {
			dst.Text = appendBytes(dst.Text, src.Text[tokStart:tokEnd])
		}
		tok.Line += lineOffset
		dst.Tokens = append(dst.Tokens, tok)
		prevEnd = tokEnd
	}
	if prevEnd < len(src.Text) {
		dst.Text = appendBytes(dst.Text, src.Text[prevEnd:])
	}
	for i := 0; i < len(redirect); i++ {
		if skip[i] && redirect[i] >= 0 {
			oldToNew[i] = mapToken(oldToNew, redirect[i], finalEOF)
		}
	}
	for i := 0; i < len(src.Decls); i++ {
		decl := src.Decls[i]
		decl.StartTok = mapToken(oldToNew, decl.StartTok, finalEOF)
		decl.EndTok = mapToken(oldToNew, decl.EndTok, finalEOF)
		nameStart, nameEnd, ok := mapTextSpanByToken(src, dst, oldToNew, finalEOF, decl.NameStart, decl.NameEnd)
		if !ok {
			return false
		}
		decl.NameStart = nameStart
		decl.NameEnd = nameEnd
		dst.Decls = append(dst.Decls, decl)
	}
	for i := 0; i < len(src.Funcs); i++ {
		fn := src.Funcs[i]
		fn.StartTok = mapToken(oldToNew, fn.StartTok, finalEOF)
		fn.NameTok = mapToken(oldToNew, fn.NameTok, finalEOF)
		nameStart, nameEnd, ok := mappedTokenTextSpan(dst, fn.NameTok)
		if !ok {
			return false
		}
		fn.NameStart = nameStart
		fn.NameEnd = nameEnd
		fn.ReceiverStart = mapToken(oldToNew, fn.ReceiverStart, finalEOF)
		fn.ReceiverEnd = mapToken(oldToNew, fn.ReceiverEnd, finalEOF)
		fn.BodyStart = mapToken(oldToNew, fn.BodyStart, finalEOF)
		fn.BodyEnd = mapToken(oldToNew, fn.BodyEnd, finalEOF)
		fn.EndTok = mapToken(oldToNew, fn.EndTok, finalEOF)
		dst.Funcs = append(dst.Funcs, fn)
	}
	if hasNext && (len(src.Text) == 0 || src.Text[len(src.Text)-1] != '\n') {
		dst.Text = append(dst.Text, '\n')
	}
	return true
}

func linkedTokenSkip(program unit.Program) ([]bool, []int) {
	skip := make([]bool, len(program.Tokens))
	redirect := make([]int, len(program.Tokens))
	for i := 0; i < len(redirect); i++ {
		redirect[i] = -1
	}
	for i := 0; i < len(program.Imports); i++ {
		markImportDeclTokens(program, skip, program.Imports[i])
	}
	for i := 0; i < len(program.Selectors); i++ {
		selector := program.Selectors[i]
		if selector.BaseKind == unit.RefImport {
			markRedirectToken(skip, redirect, selector.BaseTok, selector.NameTok)
			markRedirectToken(skip, redirect, selector.DotTok, selector.NameTok)
		}
	}
	for i := 0; i < len(program.TypeRefs); i++ {
		ref := program.TypeRefs[i]
		if ref.Kind == unit.TypeRefImportSelector {
			markRedirectToken(skip, redirect, ref.BaseTok, ref.Token)
			markRedirectToken(skip, redirect, ref.DotTok, ref.Token)
		}
	}
	for i := 0; i < len(program.Calls); i++ {
		call := program.Calls[i]
		if call.Kind == unit.CallImportSelector {
			markRedirectToken(skip, redirect, call.BaseTok, call.CalleeTok)
			markRedirectToken(skip, redirect, call.DotTok, call.CalleeTok)
		}
	}
	return skip, redirect
}

func markImportDeclTokens(program unit.Program, skip []bool, imp unit.Import) {
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
		skip[i] = true
	}
}

func markRedirectToken(skip []bool, redirect []int, tok int, target int) {
	if tok < 0 || tok >= len(skip) || target < 0 || target >= len(skip) {
		return
	}
	skip[tok] = true
	redirect[tok] = target
}

func linkedTokenReplacements(program unit.Program, aliases []string, symbolOffsets []int) []string {
	out := make([]string, len(program.Tokens))
	for i := 0; i < len(program.Symbols); i++ {
		symbol := program.Symbols[i]
		name := packageSymbolAlias(aliases, symbolOffsets, symbol.Package, i)
		if name != "" && symbol.Token >= 0 && symbol.Token < len(out) {
			out[symbol.Token] = name
		}
	}
	for i := 0; i < len(program.Refs); i++ {
		ref := program.Refs[i]
		if ref.Kind == unit.RefPackage {
			name := packageSymbolAlias(aliases, symbolOffsets, ref.Package, ref.Index)
			if name != "" && ref.Token >= 0 && ref.Token < len(out) {
				out[ref.Token] = name
			}
		}
	}
	for i := 0; i < len(program.Selectors); i++ {
		selector := program.Selectors[i]
		name := packageSymbolAlias(aliases, symbolOffsets, selector.Package, selector.Symbol)
		if name != "" && selector.NameTok >= 0 && selector.NameTok < len(out) {
			out[selector.NameTok] = name
		}
	}
	for i := 0; i < len(program.TypeRefs); i++ {
		ref := program.TypeRefs[i]
		name := packageSymbolAlias(aliases, symbolOffsets, ref.Package, ref.Symbol)
		if name != "" && ref.Token >= 0 && ref.Token < len(out) {
			out[ref.Token] = name
		}
	}
	return out
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
	if pkg < 0 || pkg >= len(symbolOffsets) || symbol < 0 {
		return ""
	}
	index := symbolOffsets[pkg] + symbol
	if index < 0 || index >= len(aliases) {
		return ""
	}
	return aliases[index]
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

func mapTextSpanByToken(src unit.Program, dst *unit.Program, oldToNew []int, eof int, start int, end int) (int, int, bool) {
	for i := 0; i < len(src.Tokens); i++ {
		tok := src.Tokens[i]
		if tok.Start == start && tok.Start+tok.Size == end {
			mapped := mapToken(oldToNew, i, eof)
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

func mapToken(oldToNew []int, tok int, eof int) int {
	if tok < 0 {
		return eof
	}
	if tok >= len(oldToNew) {
		return -1
	}
	mapped := oldToNew[tok]
	if mapped < 0 {
		return -1
	}
	return mapped
}

func countCoreLinkedEOF(programs []unit.Program) int {
	total := 0
	for i := 0; i < len(programs); i++ {
		skip, _ := linkedTokenSkip(programs[i])
		for j := 0; j < len(programs[i].Tokens); j++ {
			if programs[i].Tokens[j].Kind != unit.TokenEOF && !skip[j] {
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
