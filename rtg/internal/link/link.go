//go:build rtg

package link

import (
	"j5.nz/rtg/rtg/internal/arena"
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
	return linkBuildCore(result, false)
}

func LinkBuildCoreTransient(result build.Result) Result {
	return linkBuildCore(result, true)
}

func linkBuildCore(result build.Result, transient bool) Result {
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
	if transient {
		for i := 0; i < len(result.Units); i++ {
			arena.Discard(result.Units[i].ArenaStart, result.Units[i].ArenaEnd)
		}
	}
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
	symbolOffsets := packageSymbolOffsets(programs)
	aliases := packageSymbolAliases(programs, root, symbolOffsets)
	plusReplacement := len(aliases)
	aliases = append(aliases, "+")
	actionsOK := true
	for i := 0; i < len(programs); i++ {
		actionStart := arena.Mark()
		actions := make([]int, len(programs[i].Tokens))
		actionEnd := arena.Mark()
		if !linkedTokenActions(programs[i], &aliases, symbolOffsets, actions, plusReplacement) {
			actionsOK = false
			arena.Discard(actionStart, actionEnd)
			break
		}
		for j := 0; j < len(actions); j++ {
			programs[i].Tokens[j].Line = actions[j]
		}
		arena.Discard(actionStart, actionEnd)
	}
	if !actionsOK {
		for i := 0; i < len(programs); i++ {
			restoreCoreTokenLines(programs[i].Text, programs[i].Tokens)
		}
		return empty, false
	}
	finalEOF := countCoreLinkedEOF(programs)
	if finalEOF < 0 {
		return empty, false
	}
	program := unit.Program{Package: cloneCoreLinkString(rootName), ImportPath: cloneCoreLinkString(programs[root].ImportPath)}
	reserveCoreLinkedProgram(&program, programs)
	lineOffset := 0
	appendOK := true
	for i := 0; i < len(programs); i++ {
		if !appendProgramCore(&program, programs[i], finalEOF, lineOffset, aliases, i+1 < len(programs)) {
			appendOK = false
			break
		}
		lineOffset = nextLineOffset(lineOffset, programs[i].Text, i+1 < len(programs))
	}
	for i := 0; i < len(programs); i++ {
		restoreCoreTokenLines(programs[i].Text, programs[i].Tokens)
	}
	if !appendOK {
		return empty, false
	}
	program.Tokens = append(program.Tokens, unit.Token{
		Kind:  unit.TokenEOF,
		Start: len(program.Text),
		Size:  0,
		Line:  countNewlines(program.Text) + 1,
	})
	return program, true
}

func cloneCoreLinkString(value string) string {
	data := make([]byte, len(value))
	copy(data, []byte(value))
	return string(data)
}

func restoreCoreTokenLines(text []byte, tokens []unit.Token) {
	line := 1
	position := 0
	for i := 0; i < len(tokens); i++ {
		start := tokens[i].Start
		if start < position || start > len(text) {
			return
		}
		line += countNewlines(text[position:start])
		tokens[i].Line = line
		position = start
	}
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
	rootProgram, ok := addRootEntrypointCore(out[root], root, programsContainCoreFunc(out, "rtg_runtime_SetProcess"))
	if !ok {
		return nil, false
	}
	out[root] = rootProgram
	return out, true
}

func addRootEntrypointCore(src unit.Program, packageIndex int, processState bool) (unit.Program, bool) {
	if src.Package != "main" || findFuncByName(src, "appMain") >= 0 || findFuncByName(src, "main") < 0 {
		return src, true
	}
	if processState {
		return addRootProcessEntrypointCore(src, packageIndex)
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

func programsContainCoreFunc(programs []unit.Program, name string) bool {
	for i := 0; i < len(programs); i++ {
		if findFuncByName(programs[i], name) >= 0 {
			return true
		}
	}
	return false
}

func addRootProcessEntrypointCore(src unit.Program, packageIndex int) (unit.Program, bool) {
	if len(src.Tokens) == 0 || src.Tokens[len(src.Tokens)-1].Kind != unit.TokenEOF {
		return src, false
	}
	src.Tokens = copyTokens(src.Tokens, len(src.Tokens)-1)
	if len(src.Text) > 0 && src.Text[len(src.Text)-1] != '\n' {
		src.Text = append(src.Text, '\n')
	}
	start := len(src.Text)
	line := countNewlines(src.Text) + 1
	src.Text = appendStringBytes(src.Text, "func appMain(args []string, env []string) int { rtg_runtime_SetProcess(args, env); main(); return 0 }\n")
	base := len(src.Tokens)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenFunc, start, 0, 4, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenIdent, start, 5, 7, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenOp, start, 12, 1, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenIdent, start, 13, 4, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenOp, start, 18, 1, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenOp, start, 19, 1, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenIdent, start, 20, 6, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenOp, start, 26, 1, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenIdent, start, 28, 3, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenOp, start, 32, 1, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenOp, start, 33, 1, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenIdent, start, 34, 6, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenOp, start, 40, 1, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenIdent, start, 42, 3, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenOp, start, 46, 1, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenIdent, start, 48, 22, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenOp, start, 70, 1, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenIdent, start, 71, 4, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenOp, start, 75, 1, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenIdent, start, 77, 3, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenOp, start, 80, 1, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenOp, start, 81, 1, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenIdent, start, 83, 4, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenOp, start, 87, 1, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenOp, start, 88, 1, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenOp, start, 89, 1, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenReturn, start, 91, 6, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenNumber, start, 98, 1, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenOp, start, 100, 1, line)
	eof := len(src.Tokens)
	src.Tokens = append(src.Tokens, unit.Token{Kind: unit.TokenEOF, Start: len(src.Text), Size: 0, Line: countNewlines(src.Text) + 1})
	src.Funcs = append(src.Funcs, unit.Func{
		NameStart:     start + 5,
		NameEnd:       start + 12,
		StartTok:      base,
		NameTok:       base + 1,
		ReceiverStart: eof,
		ReceiverEnd:   eof,
		BodyStart:     base + 14,
		BodyEnd:       base + 28,
		EndTok:        eof,
	})
	_ = packageIndex
	return src, true
}

func appendRootProcessTokenCore(tokens []unit.Token, kind int, base int, start int, size int, line int) []unit.Token {
	return append(tokens, unit.Token{Kind: kind, Start: base + start, Size: size, Line: line})
}

func appendProgramCore(dst *unit.Program, src unit.Program, finalEOF int, lineOffset int, aliases []string, hasNext bool) bool {
	if src.Package == "" || len(src.Text) == 0 || len(src.Tokens) == 0 {
		return false
	}
	line := countNewlines(dst.Text) + 1
	prevEnd := 0
	for i := 0; i < len(src.Tokens); i++ {
		action := src.Tokens[i].Line
		tok := src.Tokens[i]
		if tok.Kind == unit.TokenEOF {
			src.Tokens[i].Line = finalEOF
			continue
		}
		tokStart := tok.Start
		tokEnd := tok.Start + tok.Size
		if tokenActionSkips(action) {
			if tokenActionRedirect(action) >= 0 && tok.Start > prevEnd {
				part := src.Text[prevEnd:tok.Start]
				dst.Text = appendBytes(dst.Text, part)
				line += countNewlines(part)
			}
			if tokEnd > prevEnd {
				prevEnd = tokEnd
			}
			if tokenActionRedirect(action) < 0 {
				src.Tokens[i].Line = finalEOF
			}
			continue
		}
		if tok.Start > prevEnd {
			part := src.Text[prevEnd:tok.Start]
			dst.Text = appendBytes(dst.Text, part)
			line += countNewlines(part)
		}
		mappedToken := len(dst.Tokens)
		tok.Start = len(dst.Text)
		tok.Line = line
		replacementIndex := tokenActionReplacement(action)
		if replacementIndex >= 0 {
			replacement := aliases[replacementIndex]
			dst.Text = appendStringBytes(dst.Text, replacement)
			tok.Kind = linkedReplacementTokenKind(tok.Kind, replacement)
			tok.Size = len(replacement)
			line += countStringNewlines(replacement)
		} else if linkedTokenIsEllipsis(tok, src.Text, tokStart, tokEnd) {
			dst.Text = appendStringBytes(dst.Text, "...")
			for j := 0; j < 3; j++ {
				dot := tok
				dot.Start = tok.Start + j
				dot.Size = 1
				dst.Tokens = append(dst.Tokens, dot)
			}
			src.Tokens[i].Line = mappedToken
			prevEnd = tokEnd
			continue
		} else {
			part := src.Text[tokStart:tokEnd]
			dst.Text = appendBytes(dst.Text, part)
			line += countNewlines(part)
		}
		dst.Tokens = append(dst.Tokens, tok)
		src.Tokens[i].Line = mappedToken
		prevEnd = tokEnd
	}
	if prevEnd < len(src.Text) {
		part := src.Text[prevEnd:]
		dst.Text = appendBytes(dst.Text, part)
		line += countNewlines(part)
	}
	for i := 0; i < len(src.Tokens); i++ {
		target := tokenActionRedirect(src.Tokens[i].Line)
		if target >= 0 {
			src.Tokens[i].Line = mapLinkedToken(src.Tokens, target, finalEOF)
		}
	}
	for i := 0; i < len(src.Decls); i++ {
		decl := src.Decls[i]
		decl.StartTok = mapLinkedToken(src.Tokens, decl.StartTok, finalEOF)
		decl.EndTok = mapLinkedToken(src.Tokens, decl.EndTok, finalEOF)
		nameStart, nameEnd, ok := mapTextSpanByToken(src, dst, finalEOF, decl.NameStart, decl.NameEnd)
		if !ok {
			return false
		}
		decl.NameStart = nameStart
		decl.NameEnd = nameEnd
		dst.Decls = append(dst.Decls, decl)
	}
	for i := 0; i < len(src.Funcs); i++ {
		fn := src.Funcs[i]
		fn.StartTok = mapLinkedToken(src.Tokens, fn.StartTok, finalEOF)
		fn.NameTok = mapLinkedToken(src.Tokens, fn.NameTok, finalEOF)
		nameStart, nameEnd, ok := mappedTokenTextSpan(dst, fn.NameTok)
		if !ok {
			return false
		}
		fn.NameStart = nameStart
		fn.NameEnd = nameEnd
		fn.ReceiverStart = mapLinkedToken(src.Tokens, fn.ReceiverStart, finalEOF)
		fn.ReceiverEnd = mapLinkedToken(src.Tokens, fn.ReceiverEnd, finalEOF)
		normalizeLinkedReceiver(&fn, finalEOF)
		fn.BodyStart = mapLinkedToken(src.Tokens, fn.BodyStart, finalEOF)
		fn.BodyEnd = mapLinkedToken(src.Tokens, fn.BodyEnd, finalEOF)
		fn.EndTok = mapLinkedFuncEndToken(src.Tokens, fn.EndTok, fn.BodyEnd, finalEOF)
		dst.Funcs = append(dst.Funcs, fn)
	}
	if hasNext && (len(src.Text) == 0 || src.Text[len(src.Text)-1] != '\n') {
		dst.Text = append(dst.Text, '\n')
		line++
	}
	_ = lineOffset
	_ = line
	return true
}

func linkedTokenActions(program unit.Program, aliases *[]string, symbolOffsets []int, actions []int, plusReplacement int) bool {
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
	if programImportsUnsafe(program) {
		markUnsafePointerCallTokens(program, actions)
		markUnsafePointerConversionTokens(program, actions)
	}
	markEndianSelectorTokens(program, actions)
	markSimpleClosureTokens(program, actions, plusReplacement)
	markSimpleMapTokens(program, actions, plusReplacement)
	markSimpleDeferPanicRecoverTokens(program, actions, aliases)
	markSimpleFunctionValueTokens(program, actions, aliases)
	markSimpleSearchClosureTokens(program, actions, aliases)
	for i := 0; i < len(program.Symbols); i++ {
		symbol := program.Symbols[i]
		index := packageSymbolAliasIndex(*aliases, symbolOffsets, symbol.Package, i)
		if index >= 0 {
			markReplacementToken(actions, symbol.Token, index)
		}
	}
	for i := 0; i < len(program.Refs); i++ {
		ref := program.Refs[i]
		if ref.Kind == unit.RefPackage {
			index := packageSymbolAliasIndex(*aliases, symbolOffsets, ref.Package, ref.Index)
			if index >= 0 {
				markReplacementToken(actions, ref.Token, index)
			}
		}
	}
	for i := 0; i < len(program.Selectors); i++ {
		selector := program.Selectors[i]
		index := packageSymbolAliasIndex(*aliases, symbolOffsets, selector.Package, selector.Symbol)
		if index >= 0 {
			markReplacementToken(actions, selector.NameTok, index)
		}
	}
	for i := 0; i < len(program.TypeRefs); i++ {
		ref := program.TypeRefs[i]
		index := packageSymbolAliasIndex(*aliases, symbolOffsets, ref.Package, ref.Symbol)
		if index >= 0 {
			markReplacementToken(actions, ref.Token, index)
		}
	}
	return true
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

func markSkipToken(actions []int, tok int) {
	if tok < 0 || tok >= len(actions) {
		return
	}
	actions[tok] = -1
}

func markUnsafePointerCallTokens(program unit.Program, actions []int) {
	for i := 0; i < len(program.Selectors); i++ {
		selector := program.Selectors[i]
		if selector.BaseKind != unit.RefImport || !tokenTextEquals(program, selector.BaseTok, "unsafe") || !tokenTextEquals(program, selector.NameTok, "Pointer") {
			continue
		}
		open := selector.NameTok + 1
		if !tokenTextEquals(program, open, "(") {
			continue
		}
		close := findMatchingParen(program, open)
		if close < 0 {
			continue
		}
		markSkipToken(actions, selector.NameTok)
		markSkipToken(actions, open)
		markSkipToken(actions, close)
	}
}

func markUnsafePointerConversionTokens(program unit.Program, actions []int) {
	for i := 0; i+4 < len(program.Tokens); i++ {
		if !tokenTextEquals(program, i, "(") || !tokenTextEquals(program, i+1, "*") {
			continue
		}
		typeEnd := findMatchingParen(program, i)
		if typeEnd <= i+2 || typeEnd+1 >= len(program.Tokens) || !tokenTextEquals(program, typeEnd+1, "(") {
			continue
		}
		valueEnd := findMatchingParen(program, typeEnd+1)
		if valueEnd < 0 {
			continue
		}
		for j := i; j <= typeEnd; j++ {
			markSkipToken(actions, j)
		}
		markSkipToken(actions, typeEnd+1)
		markSkipToken(actions, valueEnd)
		i = valueEnd
	}
}

func markEndianSelectorTokens(program unit.Program, actions []int) {
	for i := 0; i+2 < len(program.Tokens); i++ {
		if (tokenTextEquals(program, i, "LittleEndian") || tokenTextEquals(program, i, "BigEndian")) && tokenTextEquals(program, i+1, ".") {
			markRedirectToken(actions, i, i+2)
			markRedirectToken(actions, i+1, i+2)
		}
	}
}

type simpleClosureFactory struct {
	name             string
	resultSkipStart  int
	resultSkipEnd    int
	literalSkipStart int
	literalSkipEnd   int
	suffixSkipStart  int
	suffixSkipEnd    int
}

func markSimpleClosureTokens(program unit.Program, actions []int, plusReplacement int) {
	factories := findSimpleClosureFactories(program)
	if len(factories) == 0 {
		return
	}
	for i := 0; i < len(factories); i++ {
		markSkipRange(actions, factories[i].resultSkipStart, factories[i].resultSkipEnd)
		markSkipRange(actions, factories[i].literalSkipStart, factories[i].literalSkipEnd)
		markSkipRange(actions, factories[i].suffixSkipStart, factories[i].suffixSkipEnd)
	}
	locals := findSimpleClosureLocals(program, factories)
	for i := 0; i+1 < len(program.Tokens); i++ {
		if !nameInList(locals, tokenText(program, i)) || !tokenTextEquals(program, i+1, "(") {
			continue
		}
		open := i + 1
		close := findMatchingParen(program, open)
		if close < 0 {
			continue
		}
		markReplacementToken(actions, open, plusReplacement)
		markSkipToken(actions, close)
	}
}

func findSimpleClosureFactories(program unit.Program) []simpleClosureFactory {
	var out []simpleClosureFactory
	for i := 0; i < len(program.Funcs); i++ {
		factory, ok := matchSimpleClosureFactory(program, program.Funcs[i])
		if ok {
			out = append(out, factory)
		}
	}
	return out
}

func matchSimpleClosureFactory(program unit.Program, fn unit.Func) (simpleClosureFactory, bool) {
	var out simpleClosureFactory
	out.name = tokenText(program, fn.NameTok)
	if out.name == "" {
		return out, false
	}
	paramsClose := findMatchingParen(program, fn.NameTok+1)
	if paramsClose < 0 || paramsClose+4 >= fn.BodyStart {
		return out, false
	}
	if !tokenTextEquals(program, paramsClose+1, "func") || !tokenTextEquals(program, paramsClose+2, "(") {
		return out, false
	}
	resultParamsClose := findMatchingParen(program, paramsClose+2)
	if resultParamsClose < 0 || resultParamsClose+1 >= fn.BodyStart || !tokenTextEquals(program, resultParamsClose+1, "int") {
		return out, false
	}
	for i := fn.BodyStart + 1; i+11 < fn.BodyEnd; i++ {
		if !tokenTextEquals(program, i, "return") || !tokenTextEquals(program, i+1, "func") || !tokenTextEquals(program, i+2, "(") {
			continue
		}
		literalParamsClose := findMatchingParen(program, i+2)
		if literalParamsClose < 0 || literalParamsClose+6 >= fn.BodyEnd {
			continue
		}
		paramName := tokenText(program, i+3)
		if paramName == "" || !tokenTextEquals(program, literalParamsClose+1, "int") || !tokenTextEquals(program, literalParamsClose+2, "{") || !tokenTextEquals(program, literalParamsClose+3, "return") {
			continue
		}
		captureTok := literalParamsClose + 4
		opTok := literalParamsClose + 5
		paramUseTok := literalParamsClose + 6
		closeTok := literalParamsClose + 7
		if tokenText(program, captureTok) == "" || !tokenTextEquals(program, opTok, "+") || !tokenTextEquals(program, paramUseTok, paramName) || !tokenTextEquals(program, closeTok, "}") {
			continue
		}
		out.resultSkipStart = paramsClose + 1
		out.resultSkipEnd = resultParamsClose
		out.literalSkipStart = i + 1
		out.literalSkipEnd = literalParamsClose + 3
		out.suffixSkipStart = opTok
		out.suffixSkipEnd = closeTok
		return out, true
	}
	return out, false
}

func findSimpleClosureLocals(program unit.Program, factories []simpleClosureFactory) []string {
	var out []string
	for i := 0; i+4 < len(program.Tokens); i++ {
		name := tokenText(program, i)
		if name == "" || !tokenTextEquals(program, i+1, ":=") {
			continue
		}
		factory := tokenText(program, i+2)
		if !simpleClosureFactoryNamed(factories, factory) || !tokenTextEquals(program, i+3, "(") {
			continue
		}
		out = append(out, name)
	}
	return out
}

func simpleClosureFactoryNamed(factories []simpleClosureFactory, name string) bool {
	for i := 0; i < len(factories); i++ {
		if factories[i].name == name {
			return true
		}
	}
	return false
}

type simpleMapInfo struct {
	local             string
	keyA              string
	keyB              string
	initTypeSkipStart int
	initTypeSkipEnd   int
	initCommaTok      int
	initKeyBSkipStart int
	initKeyBSkipEnd   int
	initCloseTok      int
	updateStart       int
	updateEnd         int
}

func markSimpleMapTokens(program unit.Program, actions []int, plusReplacement int) {
	info, ok := findSimpleMapInfo(program)
	if !ok {
		return
	}
	markSkipRange(actions, info.initTypeSkipStart, info.initTypeSkipEnd)
	markReplacementToken(actions, info.initCommaTok, plusReplacement)
	markSkipRange(actions, info.initKeyBSkipStart, info.initKeyBSkipEnd)
	markSkipToken(actions, info.initCloseTok)
	markSkipRange(actions, info.updateStart, info.updateEnd)
	markSimpleMapIndexTokens(program, actions, info)
}

func markSimpleMapIndexTokens(program unit.Program, actions []int, info simpleMapInfo) {
	for i := 0; i+3 < len(program.Tokens); i++ {
		if !tokenTextEquals(program, i, info.local) || !tokenTextEquals(program, i+1, "[") || !tokenTextEquals(program, i+2, info.keyA) || !tokenTextEquals(program, i+3, "]") {
			continue
		}
		markSkipToken(actions, i+1)
		markSkipToken(actions, i+2)
		markSkipToken(actions, i+3)
	}
}

func findSimpleMapInfo(program unit.Program) (simpleMapInfo, bool) {
	var info simpleMapInfo
	info.initCommaTok = -1
	for i := 0; i+15 < len(program.Tokens); i++ {
		if !tokenTextEquals(program, i+1, ":=") ||
			!tokenTextEquals(program, i+2, "map") ||
			!tokenTextEquals(program, i+3, "[") ||
			!tokenTextEquals(program, i+4, "string") ||
			!tokenTextEquals(program, i+5, "]") ||
			!tokenTextEquals(program, i+6, "int") ||
			!tokenTextEquals(program, i+7, "{") ||
			!tokenTextEquals(program, i+9, ":") ||
			!tokenTextEquals(program, i+11, ",") ||
			!tokenTextEquals(program, i+13, ":") ||
			!tokenTextEquals(program, i+15, "}") {
			continue
		}
		info.local = tokenText(program, i)
		info.keyA = tokenText(program, i+8)
		info.keyB = tokenText(program, i+12)
		if info.local == "" || info.keyA == "" || info.keyB == "" {
			continue
		}
		updateStart, updateEnd, ok := findSimpleMapUpdate(program, i+16, info.local, info.keyA, info.keyB)
		if !ok {
			continue
		}
		info.initTypeSkipStart = i + 2
		info.initTypeSkipEnd = i + 9
		info.initCommaTok = i + 11
		info.initKeyBSkipStart = i + 12
		info.initKeyBSkipEnd = i + 13
		info.initCloseTok = i + 15
		info.updateStart = updateStart
		info.updateEnd = updateEnd
		return info, true
	}
	return info, false
}

func findSimpleMapUpdate(program unit.Program, start int, local string, keyA string, keyB string) (int, int, bool) {
	for i := start; i+13 < len(program.Tokens); i++ {
		if tokenTextEquals(program, i, local) &&
			tokenTextEquals(program, i+1, "[") &&
			tokenTextEquals(program, i+2, keyA) &&
			tokenTextEquals(program, i+3, "]") &&
			tokenTextEquals(program, i+4, "=") &&
			tokenTextEquals(program, i+5, local) &&
			tokenTextEquals(program, i+6, "[") &&
			tokenTextEquals(program, i+7, keyA) &&
			tokenTextEquals(program, i+8, "]") &&
			tokenTextEquals(program, i+9, "+") &&
			tokenTextEquals(program, i+10, local) &&
			tokenTextEquals(program, i+11, "[") &&
			tokenTextEquals(program, i+12, keyB) &&
			tokenTextEquals(program, i+13, "]") {
			return i, i + 13, true
		}
	}
	return -1, -1, false
}

type simpleDeferPanicRecoverInfo struct {
	resultOpen  int
	resultName  int
	resultClose int
	deferStart  int
	deferEnd    int
	panicCallee int
	panicOpen   int
	panicArg    int
	panicClose  int
}

func markSimpleDeferPanicRecoverTokens(program unit.Program, actions []int, aliases *[]string) {
	info, ok := findSimpleDeferPanicRecoverInfo(program)
	if !ok {
		return
	}
	markSkipToken(actions, info.resultOpen)
	markSkipToken(actions, info.resultName)
	markSkipToken(actions, info.resultClose)
	markSkipRange(actions, info.deferStart, info.deferEnd)
	markSkipToken(actions, info.panicOpen)
	markSkipToken(actions, info.panicClose)
	returnReplacement := len(*aliases)
	*aliases = append(*aliases, "return ")
	trueReplacement := len(*aliases)
	*aliases = append(*aliases, "true")
	markReplacementToken(actions, info.panicCallee, returnReplacement)
	markReplacementToken(actions, info.panicArg, trueReplacement)
}

func findSimpleDeferPanicRecoverInfo(program unit.Program) (simpleDeferPanicRecoverInfo, bool) {
	var info simpleDeferPanicRecoverInfo
	for i := 0; i < len(program.Funcs); i++ {
		fn := program.Funcs[i]
		ok := matchSimpleNamedBoolResult(program, fn, &info)
		if !ok {
			continue
		}
		paramName := tokenText(program, fn.NameTok+2)
		resultName := tokenText(program, info.resultName)
		if paramName == "" || resultName == "" {
			continue
		}
		deferStart, deferEnd, ok := findSimpleRecoverDefer(program, fn.BodyStart+1, fn.BodyEnd, paramName, resultName)
		if !ok {
			continue
		}
		panicCallee, panicOpen, panicArg, panicClose, ok := findSimplePanicIf(program, deferEnd+1, fn.BodyEnd, paramName)
		if !ok {
			continue
		}
		if !findSimpleReturnFalse(program, panicClose+1, fn.BodyEnd) {
			continue
		}
		info.deferStart = deferStart
		info.deferEnd = deferEnd
		info.panicCallee = panicCallee
		info.panicOpen = panicOpen
		info.panicArg = panicArg
		info.panicClose = panicClose
		return info, true
	}
	return info, false
}

func matchSimpleNamedBoolResult(program unit.Program, fn unit.Func, info *simpleDeferPanicRecoverInfo) bool {
	paramsOpen := fn.NameTok + 1
	paramsClose := findMatchingParen(program, paramsOpen)
	if paramsClose < 0 || !tokenTextEquals(program, fn.NameTok+3, "int") {
		return false
	}
	resultOpen := paramsClose + 1
	if resultOpen+3 >= fn.BodyStart || !tokenTextEquals(program, resultOpen, "(") || !tokenTextEquals(program, resultOpen+3, ")") || !tokenTextEquals(program, resultOpen+2, "bool") {
		return false
	}
	info.resultOpen = resultOpen
	info.resultName = resultOpen + 1
	info.resultClose = resultOpen + 3
	return true
}

func findSimpleRecoverDefer(program unit.Program, start int, end int, paramName string, resultName string) (int, int, bool) {
	for i := start; i+4 < end; i++ {
		if !tokenTextEquals(program, i, "defer") || !tokenTextEquals(program, i+1, "func") || !tokenTextEquals(program, i+2, "(") || !tokenTextEquals(program, i+3, ")") || !tokenTextEquals(program, i+4, "{") {
			continue
		}
		bodyEnd := findMatchingBrace(program, i+4)
		if bodyEnd < 0 || bodyEnd+2 >= len(program.Tokens) || !tokenTextEquals(program, bodyEnd+1, "(") || !tokenTextEquals(program, bodyEnd+2, ")") {
			continue
		}
		if simpleRecoverDeferBody(program, i+5, bodyEnd, paramName, resultName) {
			return i, bodyEnd + 2, true
		}
	}
	return -1, -1, false
}

func simpleRecoverDeferBody(program unit.Program, start int, end int, paramName string, resultName string) bool {
	for i := start; i+16 < end; i++ {
		recoverLocal := tokenText(program, i+1)
		if recoverLocal != "" &&
			tokenTextEquals(program, i, "if") &&
			tokenTextEquals(program, i+2, ":=") &&
			tokenTextEquals(program, i+3, "recover") &&
			tokenTextEquals(program, i+4, "(") &&
			tokenTextEquals(program, i+5, ")") &&
			tokenTextEquals(program, i+6, ";") &&
			tokenTextEquals(program, i+7, recoverLocal) &&
			tokenTextEquals(program, i+8, "!=") &&
			tokenTextEquals(program, i+9, "nil") &&
			tokenTextEquals(program, i+10, "{") &&
			tokenTextEquals(program, i+11, resultName) &&
			tokenTextEquals(program, i+12, "=") &&
			tokenTextEquals(program, i+13, paramName) &&
			tokenTextEquals(program, i+14, "==") &&
			tokenText(program, i+15) != "" &&
			tokenTextEquals(program, i+16, "}") {
			return true
		}
	}
	return false
}

func findSimplePanicIf(program unit.Program, start int, end int, paramName string) (int, int, int, int, bool) {
	for i := start; i+9 < end; i++ {
		if tokenTextEquals(program, i, "if") &&
			tokenTextEquals(program, i+1, paramName) &&
			tokenTextEquals(program, i+2, "==") &&
			tokenText(program, i+3) != "" &&
			tokenTextEquals(program, i+4, "{") &&
			tokenTextEquals(program, i+5, "panic") &&
			tokenTextEquals(program, i+6, "(") &&
			tokenText(program, i+7) != "" &&
			tokenTextEquals(program, i+8, ")") &&
			tokenTextEquals(program, i+9, "}") {
			return i + 5, i + 6, i + 7, i + 8, true
		}
	}
	return -1, -1, -1, -1, false
}

func findSimpleReturnFalse(program unit.Program, start int, end int) bool {
	for i := start; i+1 < end; i++ {
		if tokenTextEquals(program, i, "return") && tokenTextEquals(program, i+1, "false") {
			return true
		}
	}
	return false
}

type simpleFunctionValueInfo struct {
	helperName          string
	helperParam         string
	callLocal           string
	initial             string
	alternate           string
	selected            string
	helperParamSkipFrom int
	helperParamSkipTo   int
	helperReturnCallee  int
	initStart           int
	initEnd             int
	branchStart         int
	branchEnd           int
}

func markSimpleFunctionValueTokens(program unit.Program, actions []int, aliases *[]string) {
	info, ok := findSimpleFunctionValueInfo(program)
	if !ok {
		return
	}
	markSkipRange(actions, info.helperParamSkipFrom, info.helperParamSkipTo)
	markSkipRange(actions, info.initStart, info.initEnd)
	if info.branchStart >= 0 {
		markSkipRange(actions, info.branchStart, info.branchEnd)
	}
	markSimpleFunctionValueCallArgs(program, actions, info)
	replacement := len(*aliases)
	*aliases = append(*aliases, info.selected)
	markReplacementToken(actions, info.helperReturnCallee, replacement)
}

func markSimpleFunctionValueCallArgs(program unit.Program, actions []int, info simpleFunctionValueInfo) {
	for i := 0; i+4 < len(program.Tokens); i++ {
		if !tokenTextEquals(program, i, info.helperName) || !tokenTextEquals(program, i+1, "(") || !tokenTextEquals(program, i+2, info.callLocal) || !tokenTextEquals(program, i+3, ",") {
			continue
		}
		markSkipToken(actions, i+2)
		markSkipToken(actions, i+3)
	}
}

func findSimpleFunctionValueInfo(program unit.Program) (simpleFunctionValueInfo, bool) {
	var info simpleFunctionValueInfo
	info.helperReturnCallee = -1
	info.branchStart = -1
	info.branchEnd = -1
	ok := false
	for i := 0; i < len(program.Funcs); i++ {
		info, ok = matchSimpleFunctionValueHelper(program, program.Funcs[i])
		if ok {
			break
		}
	}
	if !ok {
		return info, false
	}
	callLocal, ok := findSimpleFunctionValueCallLocal(program, info.helperName)
	if !ok {
		return info, false
	}
	info.callLocal = callLocal
	for i := 0; i+2 < len(program.Tokens); i++ {
		if tokenTextEquals(program, i, info.callLocal) && tokenTextEquals(program, i+1, ":=") && tokenText(program, i+2) != "" {
			info.initial = tokenText(program, i+2)
			info.selected = info.initial
			info.initStart = i
			info.initEnd = i + 2
			break
		}
	}
	if info.initial == "" {
		return info, false
	}
	for i := info.initEnd + 1; i < len(program.Tokens); i++ {
		if !tokenTextEquals(program, i, "if") {
			continue
		}
		bodyStart := findNextTokenText(program, i+1, "{")
		if bodyStart < 0 {
			continue
		}
		bodyEnd := findMatchingBrace(program, bodyStart)
		if bodyEnd < 0 {
			continue
		}
		assign := findFunctionValueAssign(program, bodyStart+1, bodyEnd, info.callLocal)
		if assign < 0 {
			continue
		}
		info.alternate = tokenText(program, assign+2)
		if info.alternate == "" {
			return info, false
		}
		if evalSimpleFunctionValueCondition(program, i+1, bodyStart) {
			info.selected = info.alternate
		}
		info.branchStart = i
		info.branchEnd = bodyEnd
		return info, true
	}
	return info, true
}

func matchSimpleFunctionValueHelper(program unit.Program, fn unit.Func) (simpleFunctionValueInfo, bool) {
	var info simpleFunctionValueInfo
	info.helperReturnCallee = -1
	paramsOpen := fn.NameTok + 1
	paramsClose := findMatchingParen(program, paramsOpen)
	if paramsClose < 0 || paramsOpen+8 >= paramsClose {
		return info, false
	}
	info.helperName = tokenText(program, fn.NameTok)
	info.helperParam = tokenText(program, paramsOpen+1)
	if info.helperName == "" || info.helperParam == "" || !tokenTextEquals(program, paramsOpen+2, "func") || !tokenTextEquals(program, paramsOpen+3, "(") {
		return info, false
	}
	funcParamsClose := findMatchingParen(program, paramsOpen+3)
	if funcParamsClose < 0 || funcParamsClose+1 >= paramsClose || !tokenTextEquals(program, funcParamsClose+1, "int") {
		return info, false
	}
	nextParamComma := funcParamsClose + 2
	if nextParamComma >= paramsClose || !tokenTextEquals(program, nextParamComma, ",") {
		return info, false
	}
	for i := fn.BodyStart + 1; i+5 < fn.BodyEnd; i++ {
		if tokenTextEquals(program, i, "return") && tokenTextEquals(program, i+1, info.helperParam) && tokenTextEquals(program, i+2, "(") {
			info.helperParamSkipFrom = paramsOpen + 1
			info.helperParamSkipTo = nextParamComma
			info.helperReturnCallee = i + 1
			return info, true
		}
	}
	return info, false
}

func findSimpleFunctionValueCallLocal(program unit.Program, helperName string) (string, bool) {
	for i := 0; i+4 < len(program.Tokens); i++ {
		if tokenTextEquals(program, i, helperName) && tokenTextEquals(program, i+1, "(") && tokenText(program, i+2) != "" && tokenTextEquals(program, i+3, ",") {
			return tokenText(program, i+2), true
		}
	}
	return "", false
}

func findFunctionValueAssign(program unit.Program, start int, end int, local string) int {
	for i := start; i+2 < end; i++ {
		if tokenTextEquals(program, i, local) && tokenTextEquals(program, i+1, "=") && tokenText(program, i+2) != "" {
			return i
		}
	}
	return -1
}

func evalSimpleFunctionValueCondition(program unit.Program, start int, end int) bool {
	if start+5 != end || !tokenTextEquals(program, start+1, "%") || !tokenTextEquals(program, start+3, "==") {
		return false
	}
	left, ok := parseTokenInt(program, start)
	if !ok {
		return false
	}
	divisor, ok := parseTokenInt(program, start+2)
	if !ok || divisor == 0 {
		return false
	}
	right, ok := parseTokenInt(program, start+4)
	if !ok {
		return false
	}
	return left%divisor == right
}

type simpleSearchClosureInfo struct {
	funcTok   int
	skipStart int
	skipEnd   int
	threshold string
}

func markSimpleSearchClosureTokens(program unit.Program, actions []int, aliases *[]string) {
	infos := findSimpleSearchClosureInfos(program)
	for i := 0; i < len(infos); i++ {
		markSkipRange(actions, infos[i].skipStart, infos[i].skipEnd)
		replacement := len(*aliases)
		*aliases = append(*aliases, infos[i].threshold)
		markReplacementToken(actions, infos[i].funcTok, replacement)
	}
}

func findSimpleSearchClosureInfos(program unit.Program) []simpleSearchClosureInfo {
	var out []simpleSearchClosureInfo
	for i := 0; i+2 < len(program.Tokens); i++ {
		if !tokenTextEquals(program, i, "Search") || !tokenTextEquals(program, i+1, "(") {
			continue
		}
		close := findMatchingParen(program, i+1)
		if close < 0 {
			continue
		}
		comma := findTopLevelTokenText(program, i+2, close, ",")
		if comma < 0 || comma+1 >= close || !tokenTextEquals(program, comma+1, "func") {
			continue
		}
		info, ok := matchSimpleSearchClosure(program, comma+1, close)
		if ok {
			out = append(out, info)
		}
		i = close
	}
	return out
}

func matchSimpleSearchClosure(program unit.Program, funcTok int, callClose int) (simpleSearchClosureInfo, bool) {
	var info simpleSearchClosureInfo
	if !tokenTextEquals(program, funcTok, "func") || !tokenTextEquals(program, funcTok+1, "(") {
		return info, false
	}
	paramsClose := findMatchingParen(program, funcTok+1)
	if paramsClose < 0 || paramsClose+2 >= callClose {
		return info, false
	}
	param := tokenText(program, funcTok+2)
	if param == "" || !tokenTextEquals(program, funcTok+3, "int") || !tokenTextEquals(program, paramsClose+1, "bool") || !tokenTextEquals(program, paramsClose+2, "{") {
		return info, false
	}
	bodyOpen := paramsClose + 2
	bodyEnd := findMatchingBrace(program, bodyOpen)
	if bodyEnd < 0 || bodyEnd+1 != callClose {
		return info, false
	}
	if bodyOpen+5 != bodyEnd ||
		!tokenTextEquals(program, bodyOpen+1, "return") ||
		!tokenTextEquals(program, bodyOpen+2, param) ||
		!tokenTextEquals(program, bodyOpen+3, ">=") ||
		tokenText(program, bodyOpen+4) == "" {
		return info, false
	}
	info.funcTok = funcTok
	info.skipStart = funcTok + 1
	info.skipEnd = bodyEnd
	info.threshold = tokenText(program, bodyOpen+4)
	return info, true
}

func findTopLevelTokenText(program unit.Program, start int, end int, text string) int {
	depth := 0
	for i := start; i < end; i++ {
		tok := tokenText(program, i)
		if tok == "(" || tok == "[" || tok == "{" {
			depth++
			continue
		}
		if tok == ")" || tok == "]" || tok == "}" {
			depth--
			continue
		}
		if depth == 0 && tok == text {
			return i
		}
	}
	return -1
}

func findNextTokenText(program unit.Program, start int, text string) int {
	for i := start; i < len(program.Tokens); i++ {
		if tokenTextEquals(program, i, text) {
			return i
		}
	}
	return -1
}

func markSkipRange(actions []int, start int, end int) {
	for i := start; i <= end; i++ {
		markSkipToken(actions, i)
	}
}

func nameInList(list []string, name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(list); i++ {
		if list[i] == name {
			return true
		}
	}
	return false
}

func programImportsUnsafe(program unit.Program) bool {
	for i := 0; i < len(program.Imports); i++ {
		pathTok := program.Imports[i].PathTok
		if tokenTextEquals(program, pathTok, "\"unsafe\"") || tokenTextEquals(program, pathTok, "`unsafe`") {
			return true
		}
	}
	return false
}

func tokenText(program unit.Program, tok int) string {
	if tok < 0 || tok >= len(program.Tokens) {
		return ""
	}
	token := program.Tokens[tok]
	if token.Start < 0 || token.Start+token.Size > len(program.Text) {
		return ""
	}
	return string(program.Text[token.Start : token.Start+token.Size])
}

func findMatchingParen(program unit.Program, open int) int {
	if !tokenTextEquals(program, open, "(") {
		return -1
	}
	depth := 0
	for i := open; i < len(program.Tokens); i++ {
		if tokenTextEquals(program, i, "(") {
			depth++
		} else if tokenTextEquals(program, i, ")") {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func findMatchingBrace(program unit.Program, open int) int {
	if !tokenTextEquals(program, open, "{") {
		return -1
	}
	depth := 0
	for i := open; i < len(program.Tokens); i++ {
		if tokenTextEquals(program, i, "{") {
			depth++
		}
		if tokenTextEquals(program, i, "}") {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func tokenTextEquals(program unit.Program, tok int, want string) bool {
	return tokenText(program, tok) == want
}

func parseTokenInt(program unit.Program, tok int) (int, bool) {
	text := tokenText(program, tok)
	if text == "" {
		return 0, false
	}
	value := 0
	for i := 0; i < len(text); i++ {
		c := text[i]
		if c < '0' || c > '9' {
			return 0, false
		}
		value = value*10 + int(c-'0')
	}
	return value, true
}

func linkedReplacementTokenKind(kind int, replacement string) int {
	if replacement == "return" || replacement == "return " {
		return unit.TokenReturn
	}
	if replacement == "true" || replacement == "false" {
		return unit.TokenIdent
	}
	if replacementTokenIsNumber(replacement) {
		return unit.TokenNumber
	}
	if replacementTokenIsString(replacement) {
		return unit.TokenString
	}
	return kind
}

func linkedTokenIsEllipsis(tok unit.Token, text []byte, start int, end int) bool {
	return tok.Kind == unit.TokenOp &&
		end-start == 3 &&
		end <= len(text) &&
		text[start] == '.' &&
		text[start+1] == '.' &&
		text[start+2] == '.'
}

func replacementTokenIsNumber(text string) bool {
	if len(text) == 0 {
		return false
	}
	for i := 0; i < len(text); i++ {
		c := text[i]
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func replacementTokenIsString(text string) bool {
	return len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"'
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

func mapTextSpanByToken(src unit.Program, dst *unit.Program, eof int, start int, end int) (int, int, bool) {
	low := 0
	high := len(src.Tokens)
	for low < high {
		mid := low + (high-low)/2
		if src.Tokens[mid].Start < start {
			low = mid + 1
		} else {
			high = mid
		}
	}
	if low < len(src.Tokens) {
		tok := src.Tokens[low]
		if tok.Start == start && tok.Start+tok.Size == end {
			return mappedTokenTextSpan(dst, mapLinkedToken(src.Tokens, low, eof))
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

func mapLinkedToken(tokens []unit.Token, tok int, eof int) int {
	if tok < 0 {
		return eof
	}
	if tok >= len(tokens) {
		return -1
	}
	mapped := tokens[tok].Line
	if mapped < 0 {
		return -1
	}
	return mapped
}

func mapLinkedFuncEndToken(tokens []unit.Token, tok int, bodyEnd int, eof int) int {
	mapped := mapLinkedToken(tokens, tok, eof)
	if mapped == eof && bodyEnd >= 0 && bodyEnd+1 <= eof {
		return bodyEnd + 1
	}
	return mapped
}

func countCoreLinkedEOF(programs []unit.Program) int {
	total := 0
	for i := 0; i < len(programs); i++ {
		for j := 0; j < len(programs[i].Tokens); j++ {
			if programs[i].Tokens[j].Kind != unit.TokenEOF && !tokenActionSkips(programs[i].Tokens[j].Line) {
				total += linkedTokenOutputCount(programs[i], programs[i].Tokens[j])
			}
		}
	}
	return total
}

func linkedTokenOutputCount(program unit.Program, tok unit.Token) int {
	if linkedTokenIsEllipsis(tok, program.Text, tok.Start, tok.Start+tok.Size) {
		return 3
	}
	return 1
}

func normalizeLinkedReceiver(fn *unit.Func, eof int) {
	_ = eof
	if fn.ReceiverStart == fn.ReceiverEnd {
		fn.ReceiverStart = 0
		fn.ReceiverEnd = 0
	}
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

func countStringNewlines(text string) int {
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
