package link

import (
	"renvo.dev/internal/arena"
	"renvo.dev/internal/build"
	"renvo.dev/internal/unit"
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
	var program unit.Program
	var ok bool
	if transient {
		program, ok = linkUnitsCore(result.Units, result.Root, true)
	} else {
		program, ok = LinkUnitsCore(result.Units, result.Root)
	}
	if !ok {
		out.Ok = false
		out.Error = LinkErrUnit
		return out
	}
	data, ok := unit.MarshalCore(unit.CoreProgramFrom(program))
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
	return linkUnitsCore(units, root, false)
}

func linkUnitsCore(units []build.PackageUnit, root int, transient bool) (unit.Program, bool) {
	var empty unit.Program
	if root < 0 || root >= len(units) {
		return empty, false
	}
	programs := make([]unit.Program, len(units))
	for i := 0; i < len(units); i++ {
		programs[i] = units[i].Program
	}
	return linkProgramsCore(programs, root, units[root].Name, units, transient)
}

func LinkProgramsCore(programs []unit.Program, root int, rootName string) (unit.Program, bool) {
	return linkProgramsCore(programs, root, rootName, nil, false)
}

func linkProgramsCore(programs []unit.Program, root int, rootName string, units []build.PackageUnit, transient bool) (unit.Program, bool) {
	var empty unit.Program
	if root < 0 || root >= len(programs) || rootName == "" {
		return empty, false
	}
	programs, ok := prepareProgramsCore(programs, root)
	if !ok {
		return empty, false
	}
	ensureCoreProgramSymbols(programs)
	symbolOffsets := corePackageSymbolOffsets(programs)
	aliases := corePackageSymbolAliases(programs, root, symbolOffsets)
	plusReplacement := len(aliases)
	aliases = append(aliases, "+")
	if transient {
		for i := 0; i < len(aliases); i++ {
			aliases[i] = cloneCoreLinkString(aliases[i])
		}
	}
	actionsOK := true
	finalEOF := 0
	for i := 0; i < len(programs); i++ {
		actionStart := arena.Mark()
		actions := make([]int, len(programs[i].Tokens))
		actionEnd := arena.Mark()
		if !linkedTokenActions(&programs[i], &aliases, symbolOffsets, actions, plusReplacement) {
			actionsOK = false
			arena.Discard(actionStart, actionEnd)
			break
		}
		for j := 0; j < len(actions); j++ {
			programs[i].Tokens[j].Line = actions[j]
			if programs[i].Tokens[j].Kind != unit.TokenEOF && !tokenActionSkips(actions[j]) {
				finalEOF++
				if coreLinkedTokenIsEllipsis(programs[i].Tokens[j], programs[i].Text, programs[i].Tokens[j].Start, programs[i].Tokens[j].Start+programs[i].Tokens[j].Size) {
					finalEOF += 2
				}
			}
		}
		arena.Discard(actionStart, actionEnd)
	}
	if !actionsOK {
		for i := 0; i < len(programs); i++ {
			restoreCoreTokenLines(programs[i].Text, programs[i].Tokens)
		}
		return empty, false
	}
	program := unit.Program{Package: cloneCoreLinkString(rootName), ImportPath: cloneCoreLinkString(programs[root].ImportPath)}
	reserveCompactLinkedProgram(&program, programs, finalEOF)
	line := 1
	appendOK := true
	for i := 0; i < len(programs); i++ {
		var ok bool
		ok, line = appendProgramCore(&program, programs[i], finalEOF, line, aliases, i+1 < len(programs))
		if !ok {
			appendOK = false
			break
		}
		if transient {
			arena.Discard(units[i].ArenaStart, units[i].ArenaEnd)
		}
	}
	if !transient {
		for i := 0; i < len(programs); i++ {
			restoreCoreTokenLines(programs[i].Text, programs[i].Tokens)
		}
	}
	if !appendOK {
		return empty, false
	}
	program.Tokens = append(program.Tokens, unit.Token{
		Kind:  unit.TokenEOF,
		Start: len(program.Text),
		Size:  0,
		Line:  line,
	})
	if !lowerFunctionValuesCore(&program) {
		return empty, false
	}
	return program, true
}

func cloneCoreLinkString(value string) string {
	data := make([]byte, len(value))
	copy(data, []byte(value))
	return string(data)
}

func replaceFunctionValueProgram(dst *unit.Program, src *unit.Program) {
	dst.Package = src.Package
	dst.ImportPath = src.ImportPath
	dst.Text = src.Text
	dst.Tokens = src.Tokens
	dst.Imports = src.Imports
	dst.Symbols = src.Symbols
	dst.Decls = src.Decls
	dst.Funcs = src.Funcs
	dst.TypeRefs = src.TypeRefs
	dst.Calls = src.Calls
	dst.Refs = src.Refs
	dst.Selectors = src.Selectors
}

func restoreCoreTokenLines(text []byte, tokens []unit.Token) {
	line := 1
	position := 0
	for i := 0; i < len(tokens); i++ {
		start := tokens[i].Start
		if start < position || start > len(text) {
			return
		}
		line += countCoreNewlines(text[position:start])
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

func reserveCompactLinkedProgram(program *unit.Program, programs []unit.Program, finalEOF int) {
	textCap := 0
	declCap := 0
	funcCap := 0
	for i := 0; i < len(programs); i++ {
		p := programs[i]
		textCap += len(p.Text) + 1
		declCap += len(p.Decls)
		funcCap += len(p.Funcs)
	}
	program.Text = make([]byte, 0, textCap)
	program.Tokens = make([]unit.Token, 0, finalEOF+1)
	program.Decls = make([]unit.Decl, 0, declCap)
	program.Funcs = make([]unit.Func, 0, funcCap)
}

func prepareProgramsCore(programs []unit.Program, root int) ([]unit.Program, bool) {
	out := make([]unit.Program, len(programs))
	copy(out, programs)
	initNames := coreProgramInitFunctionNames(out)
	rootProgram, ok := addRootEntrypointCore(out[root], root, programsContainCoreFunc(out, "renvo_runtime_SetProcess"), initNames)
	if !ok {
		return nil, false
	}
	out[root] = rootProgram
	return out, true
}

func addRootEntrypointCore(src unit.Program, packageIndex int, processState bool, initNames []string) (unit.Program, bool) {
	if src.Package != "main" || findCoreFuncByName(src, "appMain") >= 0 || findCoreFuncByName(src, "main") < 0 {
		return src, true
	}
	if processState {
		return addRootProcessEntrypointCore(src, packageIndex, initNames)
	}
	if len(src.Tokens) == 0 || src.Tokens[len(src.Tokens)-1].Kind != unit.TokenEOF {
		return src, false
	}
	src.Tokens = copyCoreTokens(src.Tokens, len(src.Tokens)-1)
	if len(src.Text) > 0 && src.Text[len(src.Text)-1] != '\n' {
		src.Text = append(src.Text, '\n')
	}
	start := len(src.Text)
	line := countCoreNewlines(src.Text) + 1
	src.Text = appendCoreStringBytes(src.Text, "func appMain() int { ")
	base := len(src.Tokens)
	src.Tokens = append(src.Tokens, unit.Token{Kind: unit.TokenFunc, Start: start, Size: 4, Line: line})
	src.Tokens = append(src.Tokens, unit.Token{Kind: unit.TokenIdent, Start: start + 5, Size: 7, Line: line})
	src.Tokens = append(src.Tokens, unit.Token{Kind: unit.TokenOp, Start: start + 12, Size: 1, Line: line})
	src.Tokens = append(src.Tokens, unit.Token{Kind: unit.TokenOp, Start: start + 13, Size: 1, Line: line})
	src.Tokens = append(src.Tokens, unit.Token{Kind: unit.TokenIdent, Start: start + 15, Size: 3, Line: line})
	src.Tokens = append(src.Tokens, unit.Token{Kind: unit.TokenOp, Start: start + 19, Size: 1, Line: line})
	mainTok, eof := appendRootEntrypointTailCore(&src, initNames, line)
	src.Funcs = append(src.Funcs, unit.Func{
		NameStart:     start + 5,
		NameEnd:       start + 12,
		StartTok:      base,
		NameTok:       base + 1,
		ReceiverStart: eof,
		ReceiverEnd:   eof,
		BodyStart:     base + 5,
		BodyEnd:       mainTok + 6,
		EndTok:        eof,
	})
	_ = packageIndex
	return src, true
}

func programsContainCoreFunc(programs []unit.Program, name string) bool {
	for i := 0; i < len(programs); i++ {
		if findCoreFuncByName(programs[i], name) >= 0 {
			return true
		}
	}
	return false
}

func addRootProcessEntrypointCore(src unit.Program, packageIndex int, initNames []string) (unit.Program, bool) {
	if len(src.Tokens) == 0 || src.Tokens[len(src.Tokens)-1].Kind != unit.TokenEOF {
		return src, false
	}
	src.Tokens = copyCoreTokens(src.Tokens, len(src.Tokens)-1)
	if len(src.Text) > 0 && src.Text[len(src.Text)-1] != '\n' {
		src.Text = append(src.Text, '\n')
	}
	start := len(src.Text)
	line := countCoreNewlines(src.Text) + 1
	const processPrefix = "func appMain(args []string, env []string) int { "
	const processName = "renvo_runtime_SetProcess"
	src.Text = appendCoreStringBytes(src.Text, processPrefix)
	src.Text = appendCoreStringBytes(src.Text, processName)
	src.Text = appendCoreStringBytes(src.Text, "(args, env); ")
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
	processStart := len(processPrefix)
	argsStart := processStart + len(processName) + 1
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenIdent, start, processStart, len(processName), line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenOp, start, argsStart-1, 1, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenIdent, start, argsStart, 4, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenOp, start, argsStart+4, 1, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenIdent, start, argsStart+6, 3, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenOp, start, argsStart+9, 1, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenOp, start, argsStart+10, 1, line)
	mainTok, eof := appendRootEntrypointTailCore(&src, initNames, line)
	src.Funcs = append(src.Funcs, unit.Func{
		NameStart:     start + 5,
		NameEnd:       start + 12,
		StartTok:      base,
		NameTok:       base + 1,
		ReceiverStart: eof,
		ReceiverEnd:   eof,
		BodyStart:     base + 14,
		BodyEnd:       mainTok + 6,
		EndTok:        eof,
	})
	_ = packageIndex
	return src, true
}

func coreProgramInitFunctionNames(programs []unit.Program) []string {
	var names []string
	for i := 0; i < len(programs); i++ {
		ordinal := 0
		for j := 0; j < len(programs[i].Funcs); j++ {
			if coreLinkedProgramText(programs[i], programs[i].Funcs[j].NameStart, programs[i].Funcs[j].NameEnd) != "init" {
				continue
			}
			names = append(names, coreInitFunctionAliasName(i, ordinal))
			ordinal++
		}
	}
	return names
}

func appendRootProcessTokenCore(tokens []unit.Token, kind int, base int, start int, size int, line int) []unit.Token {
	return append(tokens, unit.Token{Kind: kind, Start: base + start, Size: size, Line: line})
}

func appendRootCallCore(src *unit.Program, name string, line int) int {
	callTok := len(src.Tokens)
	callStart := len(src.Text)
	src.Text = appendCoreStringBytes(src.Text, name)
	src.Text = appendCoreStringBytes(src.Text, "(); ")
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenIdent, callStart, 0, len(name), line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenOp, callStart, len(name), 1, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenOp, callStart, len(name)+1, 1, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenOp, callStart, len(name)+2, 1, line)
	return callTok
}

func appendRootEntrypointTailCore(src *unit.Program, initNames []string, line int) (int, int) {
	for i := 0; i < len(initNames); i++ {
		appendRootCallCore(src, initNames[i], line)
	}
	mainTok := appendRootCallCore(src, "main", line)
	tailStart := len(src.Text)
	src.Text = appendCoreStringBytes(src.Text, "return 0 }\n")
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenReturn, tailStart, 0, 6, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenNumber, tailStart, 7, 1, line)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenOp, tailStart, 9, 1, line)
	eof := len(src.Tokens)
	src.Tokens = appendRootProcessTokenCore(src.Tokens, unit.TokenEOF, len(src.Text), 0, 0, countCoreNewlines(src.Text)+1)
	return mainTok, eof
}

func appendProgramCore(dst *unit.Program, src unit.Program, finalEOF int, line int, aliases []string, hasNext bool) (bool, int) {
	if src.Package == "" || len(src.Text) == 0 || len(src.Tokens) == 0 {
		return false, line
	}
	prevEnd := 0
	for i := 0; i < len(src.Tokens); i++ {
		action := src.Tokens[i].Line
		tok := src.Tokens[i]
		if tok.Kind == unit.TokenEOF {
			src.Tokens[i].Line = len(dst.Tokens)
			continue
		}
		tokStart := tok.Start
		tokEnd := tok.Start + tok.Size
		if tokenActionSkips(action) {
			if tokenActionRedirect(action) >= 0 && tok.Start > prevEnd {
				part := src.Text[prevEnd:tok.Start]
				dst.Text = appendCoreBytes(dst.Text, part)
				line += countCoreNewlines(part)
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
			dst.Text = appendCoreBytes(dst.Text, part)
			line += countCoreNewlines(part)
		}
		mappedToken := len(dst.Tokens)
		tok.Start = len(dst.Text)
		tok.Line = line
		replacementIndex := tokenActionReplacement(action)
		if replacementIndex >= 0 {
			replacement := aliases[replacementIndex]
			dst.Text = appendCoreStringBytes(dst.Text, replacement)
			tok.Kind = coreLinkedReplacementTokenKind(tok.Kind, replacement)
			tok.Size = len(replacement)
			line += countCoreStringNewlines(replacement)
		} else if coreLinkedTokenIsEllipsis(tok, src.Text, tokStart, tokEnd) {
			dst.Text = appendCoreStringBytes(dst.Text, "...")
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
			dst.Text = appendCoreBytes(dst.Text, part)
			line += countCoreNewlines(part)
		}
		dst.Tokens = append(dst.Tokens, tok)
		src.Tokens[i].Line = mappedToken
		prevEnd = tokEnd
	}
	if prevEnd < len(src.Text) {
		part := src.Text[prevEnd:]
		dst.Text = appendCoreBytes(dst.Text, part)
		line += countCoreNewlines(part)
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
		nameStart, nameEnd, ok := mapCoreTextSpanByToken(src, dst, finalEOF, decl.NameStart, decl.NameEnd)
		if !ok {
			return false, line
		}
		decl.NameStart = nameStart
		decl.NameEnd = nameEnd
		dst.Decls = append(dst.Decls, decl)
	}
	for i := 0; i < len(src.Funcs); i++ {
		fn := src.Funcs[i]
		fn.StartTok = mapLinkedToken(src.Tokens, fn.StartTok, finalEOF)
		fn.NameTok = mapLinkedToken(src.Tokens, fn.NameTok, finalEOF)
		nameStart, nameEnd, ok := mappedCoreTokenTextSpan(dst, fn.NameTok)
		if !ok {
			return false, line
		}
		fn.NameStart = nameStart
		fn.NameEnd = nameEnd
		fn.ReceiverStart = mapLinkedToken(src.Tokens, fn.ReceiverStart, finalEOF)
		fn.ReceiverEnd = mapLinkedToken(src.Tokens, fn.ReceiverEnd, finalEOF)
		normalizeCoreLinkedReceiver(&fn, finalEOF)
		fn.BodyStart = mapLinkedToken(src.Tokens, fn.BodyStart, finalEOF)
		fn.BodyEnd = mapLinkedToken(src.Tokens, fn.BodyEnd, finalEOF)
		fn.EndTok = mapLinkedFuncEndToken(src.Tokens, fn.EndTok, fn.BodyEnd, finalEOF)
		dst.Funcs = append(dst.Funcs, fn)
	}
	if hasNext && (len(src.Text) == 0 || src.Text[len(src.Text)-1] != '\n') {
		dst.Text = append(dst.Text, '\n')
		line++
	}
	return true, line
}

func linkedTokenActions(program *unit.Program, aliases *[]string, symbolOffsets []int, actions []int, plusReplacement int) bool {
	if len(actions) != len(program.Tokens) {
		return false
	}
	for i := 0; i < len(program.Imports); i++ {
		markCoreImportDeclTokens(program, actions, program.Imports[i])
	}
	for i := 0; i < len(program.Selectors); i++ {
		selector := program.Selectors[i]
		if selector.BaseKind == unit.RefImport {
			markCoreRedirectToken(actions, selector.BaseTok, selector.NameTok)
			markCoreRedirectToken(actions, selector.DotTok, selector.NameTok)
		}
	}
	for i := 0; i < len(program.TypeRefs); i++ {
		ref := program.TypeRefs[i]
		if ref.Kind == unit.TypeRefImportSelector {
			markCoreRedirectToken(actions, ref.BaseTok, ref.Token)
			markCoreRedirectToken(actions, ref.DotTok, ref.Token)
		}
	}
	if coreProgramImportsUnsafe(program) {
		markCoreUnsafeSizeofTokens(program, actions)
		markCoreUnsafePointerCallTokens(program, actions)
		markCoreUnsafePointerConversionTokens(program, actions)
	}
	markCoreEndianSelectorTokens(program, actions)
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

func markCoreImportDeclTokens(program *unit.Program, actions []int, imp unit.Import) {
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

func markCoreRedirectToken(actions []int, tok int, target int) {
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

func markCoreSkipToken(actions []int, tok int) {
	if tok < 0 || tok >= len(actions) {
		return
	}
	actions[tok] = -1
}

func markCoreUnsafePointerCallTokens(program *unit.Program, actions []int) {
	for i := 0; i < len(program.Selectors); i++ {
		selector := program.Selectors[i]
		if selector.BaseKind != unit.RefImport || !coreTokenTextEquals(program, selector.BaseTok, "unsafe") || !coreTokenTextEquals(program, selector.NameTok, "Pointer") {
			continue
		}
		open := selector.NameTok + 1
		if !coreTokenTextEquals(program, open, "(") {
			continue
		}
		close := findCoreMatchingParen(program, open)
		if close < 0 {
			continue
		}
		markCoreSkipToken(actions, selector.NameTok)
		markCoreSkipToken(actions, open)
		markCoreSkipToken(actions, close)
	}
}

func markCoreUnsafeSizeofTokens(program *unit.Program, actions []int) {
	for i := 0; i < len(program.Imports); i++ {
		imp := program.Imports[i]
		if !coreTokenTextEquals(program, imp.PathTok, "\"unsafe\"") && !coreTokenTextEquals(program, imp.PathTok, "`unsafe`") {
			continue
		}
		name := "unsafe"
		if imp.NameTok >= 0 {
			name = coreTokenText(program, imp.NameTok)
		}
		if name == "" || name == "." || name == "_" {
			continue
		}
		for tok := 0; tok+2 < len(program.Tokens); tok++ {
			if coreTokenText(program, tok) == name && coreTokenTextEquals(program, tok+1, ".") && coreTokenTextEquals(program, tok+2, "Sizeof") {
				markCoreRedirectToken(actions, tok, tok+2)
				markCoreRedirectToken(actions, tok+1, tok+2)
			}
		}
	}
}

func markCoreUnsafePointerConversionTokens(program *unit.Program, actions []int) {
	for i := 0; i+4 < len(program.Tokens); i++ {
		if !coreTokenTextEquals(program, i, "(") || !coreTokenTextEquals(program, i+1, "*") {
			continue
		}
		typeEnd := findCoreMatchingParen(program, i)
		if typeEnd <= i+2 || typeEnd+1 >= len(program.Tokens) || !coreTokenTextEquals(program, typeEnd+1, "(") {
			continue
		}
		valueEnd := findCoreMatchingParen(program, typeEnd+1)
		if valueEnd < 0 {
			continue
		}
		for j := i; j <= typeEnd; j++ {
			markCoreSkipToken(actions, j)
		}
		markCoreSkipToken(actions, typeEnd+1)
		markCoreSkipToken(actions, valueEnd)
		i = valueEnd
	}
}

func markCoreEndianSelectorTokens(program *unit.Program, actions []int) {
	for i := 0; i+2 < len(program.Tokens); i++ {
		if (coreTokenTextEquals(program, i, "LittleEndian") || coreTokenTextEquals(program, i, "BigEndian")) && coreTokenTextEquals(program, i+1, ".") {
			markCoreRedirectToken(actions, i, i+2)
			markCoreRedirectToken(actions, i+1, i+2)
		}
	}
}

func coreProgramImportsUnsafe(program *unit.Program) bool {
	for i := 0; i < len(program.Imports); i++ {
		pathTok := program.Imports[i].PathTok
		if coreTokenTextEquals(program, pathTok, "\"unsafe\"") || coreTokenTextEquals(program, pathTok, "`unsafe`") {
			return true
		}
	}
	return false
}

func coreTokenText(program *unit.Program, tok int) string {
	if tok < 0 || tok >= len(program.Tokens) {
		return ""
	}
	token := program.Tokens[tok]
	if token.Start < 0 || token.Start+token.Size > len(program.Text) {
		return ""
	}
	return string(program.Text[token.Start : token.Start+token.Size])
}

func findCoreMatchingParen(program *unit.Program, open int) int {
	if !coreTokenTextEquals(program, open, "(") {
		return -1
	}
	depth := 0
	for i := open; i < len(program.Tokens); i++ {
		if coreTokenTextEquals(program, i, "(") {
			depth++
		} else if coreTokenTextEquals(program, i, ")") {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func coreTokenTextEquals(program *unit.Program, tok int, want string) bool {
	if tok < 0 || tok >= len(program.Tokens) {
		return false
	}
	token := program.Tokens[tok]
	if token.Start < 0 || token.Size != len(want) || token.Start+token.Size > len(program.Text) {
		return false
	}
	for i := 0; i < len(want); i++ {
		if program.Text[token.Start+i] != want[i] {
			return false
		}
	}
	return true
}

func coreLinkedReplacementTokenKind(kind int, replacement string) int {
	if replacement == "return" || replacement == "return " {
		return unit.TokenReturn
	}
	if replacement == "true" || replacement == "false" {
		return unit.TokenIdent
	}
	if coreReplacementTokenIsNumber(replacement) {
		return unit.TokenNumber
	}
	if coreReplacementTokenIsString(replacement) {
		return unit.TokenString
	}
	return kind
}

func coreLinkedTokenIsEllipsis(tok unit.Token, text []byte, start int, end int) bool {
	return tok.Kind == unit.TokenOp &&
		end-start == 3 &&
		end <= len(text) &&
		text[start] == '.' &&
		text[start+1] == '.' &&
		text[start+2] == '.'
}

func coreReplacementTokenIsNumber(text string) bool {
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

func coreReplacementTokenIsString(text string) bool {
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

func corePackageSymbolAliases(programs []unit.Program, root int, symbolOffsets []int) []string {
	total := 0
	if len(programs) > 0 {
		last := len(programs) - 1
		total = symbolOffsets[last] + len(programs[last].Symbols)
	}
	out := make([]string, total)
	if total == 0 {
		return out
	}
	buckets := make([]int, total*2+1)
	for i := 0; i < len(buckets); i++ {
		buckets[i] = -1
	}
	next := make([]int, total)
	names := make([]string, total)
	duplicate := make([]bool, total)
	for i := 0; i < len(programs); i++ {
		initOrdinal := 0
		for j := 0; j < len(programs[i].Symbols); j++ {
			index := symbolOffsets[i] + j
			name := programs[i].Symbols[j].Name
			names[index] = name
			if name == "init" {
				out[index] = coreInitFunctionAliasName(i, initOrdinal)
				initOrdinal++
			}
			bucket := coreSymbolAliasHash(name) % len(buckets)
			next[index] = buckets[bucket]
			for prior := buckets[bucket]; prior >= 0; prior = next[prior] {
				if names[prior] == name {
					duplicate[index] = true
					duplicate[prior] = true
				}
			}
			buckets[bucket] = index
		}
	}
	for i := 0; i < len(programs); i++ {
		if i == root {
			continue
		}
		for j := 0; j < len(programs[i].Symbols); j++ {
			index := symbolOffsets[i] + j
			if duplicate[index] && programs[i].Symbols[j].Name != "init" && !coreSymbolKeepsRuntimeName(programs[i].Symbols[j].Name) {
				out[index] = coreSymbolAliasName(i, programs[i].Symbols[j].Name)
			}
		}
	}
	return out
}

func coreSymbolKeepsRuntimeName(name string) bool {
	switch name {
	case "renvo_runtime_Exit",
		"renvo_runtime_ArenaMark",
		"renvo_runtime_ArenaReset",
		"renvo_runtime_ArenaPersistMark",
		"renvo_runtime_ArenaPersistReset",
		"renvo_runtime_ArenaPersistString",
		"renvo_runtime_ArenaPersistBytes",
		"renvo_runtime_ArenaDiscard":
		return true
	}
	return false
}

func coreSymbolAliasHash(name string) int {
	hash := 5381
	for i := 0; i < len(name); i++ {
		hash = ((hash << 5) + hash) ^ int(name[i])
	}
	return hash & 2147483647
}

func corePackageSymbolAlias(aliases []string, symbolOffsets []int, pkg int, symbol int) string {
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

func coreSymbolAliasName(pkg int, name string) string {
	out := []byte("renvop")
	out = appendCoreInt(out, pkg)
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

func coreInitFunctionAliasName(pkg int, function int) string {
	out := []byte("renvoi")
	out = appendCoreInt(out, pkg)
	out = append(out, '_')
	out = appendCoreInt(out, function)
	return string(out)
}

func appendCoreInt(out []byte, value int) []byte {
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

func copyCoreTokens(src []unit.Token, limit int) []unit.Token {
	var out []unit.Token
	for i := 0; i < limit && i < len(src); i++ {
		out = append(out, src[i])
	}
	return out
}

func findCoreFuncByName(program unit.Program, name string) int {
	for i := 0; i < len(program.Funcs); i++ {
		fn := program.Funcs[i]
		if coreLinkedProgramText(program, fn.NameStart, fn.NameEnd) == name {
			return i
		}
	}
	return -1
}

func coreLinkedProgramText(program unit.Program, start int, end int) string {
	if start < 0 || end < start || end > len(program.Text) {
		return ""
	}
	return string(program.Text[start:end])
}

func mapCoreTextSpanByToken(src unit.Program, dst *unit.Program, eof int, start int, end int) (int, int, bool) {
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
			return mappedCoreTokenTextSpan(dst, mapLinkedToken(src.Tokens, low, eof))
		}
	}
	return 0, 0, false
}

func mappedCoreTokenTextSpan(program *unit.Program, tok int) (int, int, bool) {
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

func normalizeCoreLinkedReceiver(fn *unit.Func, eof int) {
	_ = eof
	if fn.ReceiverStart == fn.ReceiverEnd {
		fn.ReceiverStart = 0
		fn.ReceiverEnd = 0
	}
}

func corePackageSymbolOffsets(programs []unit.Program) []int {
	out := make([]int, len(programs))
	next := 0
	for i := 0; i < len(programs); i++ {
		out[i] = next
		next += len(programs[i].Symbols)
	}
	return out
}

func countCoreNewlines(text []byte) int {
	count := 0
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			count++
		}
	}
	return count
}

func countCoreStringNewlines(text string) int {
	count := 0
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			count++
		}
	}
	return count
}

func appendCoreBytes(out []byte, data []byte) []byte {
	return append(out, data...)
}

func appendCoreStringBytes(out []byte, data string) []byte {
	for i := 0; i < len(data); i++ {
		out = append(out, data[i])
	}
	return out
}
