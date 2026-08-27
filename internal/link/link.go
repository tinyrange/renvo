package link

import (
	"renvo.dev/internal/arena"
	"renvo.dev/internal/build"
	"renvo.dev/internal/syntax"
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
	return linkBuildCore(result, false, false)
}

func LinkBuildCoreTransient(result build.Result) Result {
	return linkBuildCore(result, true, false)
}

func LinkBuildObjectCore(result build.Result) Result {
	return linkBuildCore(result, false, true)
}

func linkBuildCore(result build.Result, transient bool, object bool) Result {
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
		program, ok = linkUnitsCore(result.Units, result.Root, true, object)
	} else {
		program, ok = linkUnitsCore(result.Units, result.Root, false, object)
	}
	if !ok {
		out.Ok = false
		out.Error = LinkErrUnit
		return out
	}
	var data []byte
	if transient {
		data, ok = unit.MarshalCoreTransient(unit.CoreProgramFrom(program))
	} else {
		data, ok = unit.MarshalCore(unit.CoreProgramFrom(program))
	}
	if !ok {
		out.Ok = false
		out.Error = LinkErrUnit
		return out
	}
	if !transient {
		out.Program = program
	}
	out.Data = data
	return out
}

func LinkUnitsCore(units []build.PackageUnit, root int) (unit.Program, bool) {
	return linkUnitsCore(units, root, false, false)
}

func linkUnitsCore(units []build.PackageUnit, root int, transient bool, object bool) (unit.Program, bool) {
	var empty unit.Program
	if root < 0 || root >= len(units) {
		return empty, false
	}
	programs := make([]unit.Program, len(units))
	for i := 0; i < len(units); i++ {
		programs[i] = units[i].Program
	}
	return linkProgramsCore(programs, root, units[root].Name, units, transient, object)
}

func LinkProgramsCore(programs []unit.Program, root int, rootName string) (unit.Program, bool) {
	return linkProgramsCore(programs, root, rootName, nil, false, false)
}

func linkProgramsCore(programs []unit.Program, root int, rootName string, units []build.PackageUnit, transient bool, object bool) (unit.Program, bool) {
	var empty unit.Program
	if root < 0 || root >= len(programs) || rootName == "" {
		return empty, false
	}
	c11Semantics := coreProgramsUseC11Semantics(programs)
	for i := 0; i < len(units); i++ {
		if units[i].C11 {
			c11Semantics = true
			break
		}
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
	actionCount := 0
	for i := 0; i < len(programs); i++ {
		actionCount += len(programs[i].Tokens)
	}
	actionStart := arena.Mark()
	actions := make([]tokenAction, actionCount)
	actionEnd := arena.Mark()
	// Arena allocations may align the backing array after actionStart. Derive
	// its real address from the post-allocation mark so package-at-a-time page
	// retirement never reaches into the next package's actions.
	actionDataStart := actionEnd - cap(actions)*4
	actionsOK := true
	finalEOF := 0
	actionOffset := 0
	for i := 0; i < len(programs); i++ {
		packageActions := actions[actionOffset : actionOffset+len(programs[i].Tokens)]
		actionOffset += len(packageActions)
		if !linkedTokenActions(&programs[i], &aliases, symbolOffsets, packageActions, plusReplacement) {
			actionsOK = false
			break
		}
		for j := 0; j < len(packageActions); j++ {
			tok := programs[i].Tokens[j]
			if tok.KindLine&255 != unit.TokenEOF && packageActions[j] >= 0 {
				finalEOF++
				if tok.KindLine&255 == unit.TokenOp && tok.Size == 3 && tok.Start+2 < len(programs[i].Text) && programs[i].Text[tok.Start] == '.' && programs[i].Text[tok.Start+1] == '.' && programs[i].Text[tok.Start+2] == '.' {
					finalEOF += 2
				}
			}
		}
	}
	if !actionsOK {
		arena.Discard(actionStart, actionEnd)
		return empty, false
	}
	program := unit.Program{Package: cloneCoreLinkString(rootName), ImportPath: cloneCoreLinkString(programs[root].ImportPath)}
	reserveCompactLinkedProgram(&program, programs, finalEOF)
	includePackageInfo := !transient
	line := 1
	appendOK := true
	actionOffset = 0
	for i := 0; i < len(programs); i++ {
		packageActionStart := actionDataStart + actionOffset*4
		info := -1
		if includePackageInfo {
			pkg := build.PackageUnit{ImportPath: programs[i].ImportPath, Name: programs[i].Package}
			if len(units) == len(programs) {
				pkg = units[i]
			}
			info = beginLinkedPackageInfo(&program, pkg)
		}
		var ok bool
		packageActions := actions[actionOffset : actionOffset+len(programs[i].Tokens)]
		actionOffset += len(packageActions)
		ok, line = appendProgramCore(&program, programs[i], packageActions, finalEOF, line, aliases, i+1 < len(programs), transient)
		if !ok {
			appendOK = false
			break
		}
		if transient {
			arena.Discard(packageActionStart, actionDataStart+actionOffset*4)
			arena.Discard(units[i].ArenaStart, units[i].ArenaEnd)
		}
		if includePackageInfo {
			finishLinkedPackageInfo(&program, info)
		}
	}
	if !transient {
		for i := 0; i < len(programs); i++ {
			restoreCoreTokenLines(programs[i].Text, programs[i].Tokens)
		}
	}
	if !appendOK {
		arena.Discard(actionStart, actionEnd)
		return empty, false
	}
	program.Tokens = append(program.Tokens, unit.MakeToken(unit.TokenEOF, len(program.Text), 0, line))
	if !lowerConcurrencyCore(&program, transient) {
		arena.Discard(actionStart, actionEnd)
		return empty, false
	}
	if !lowerMapsCore(&program, transient) {
		arena.Discard(actionStart, actionEnd)
		return empty, false
	}
	functionValuesOK := false
	if object {
		functionValuesOK = lowerObjectFunctionValuesCore(&program, transient)
	} else {
		functionValuesOK = lowerFunctionValuesCore(&program, transient)
	}
	if !functionValuesOK {
		arena.Discard(actionStart, actionEnd)
		return empty, false
	}
	if c11Semantics && !coreTextHasC11Directive(program.Text) {
		program.Text = appendCoreStringBytes(program.Text, "\n// renvo:c11\n")
		if len(program.Tokens) > 0 {
			last := len(program.Tokens) - 1
			if program.Tokens[last].KindLine&255 == unit.TokenEOF {
				program.Tokens[last].Start = len(program.Text)
			}
		}
	}
	arena.Discard(actionStart, actionEnd)
	compactCoreLinkedTokenLines(program.Tokens)
	return program, true
}

func coreProgramsUseC11Semantics(programs []unit.Program) bool {
	for i := 0; i < len(programs); i++ {
		if coreTextHasC11Directive(programs[i].Text) {
			return true
		}
	}
	return false
}

func coreTextHasC11Directive(text []byte) bool {
	marker := "// renvo:c11"
	for start := 0; start < len(text); {
		end := start
		for end < len(text) && text[end] != '\n' && text[end] != '\r' {
			end++
		}
		if end-start == len(marker) {
			match := true
			for i := 0; i < len(marker); i++ {
				if text[start+i] != marker[i] {
					match = false
					break
				}
			}
			if match {
				return true
			}
		}
		for end < len(text) && (text[end] == '\n' || text[end] == '\r') {
			end++
		}
		start = end
	}
	return false
}

// Linked programs no longer need gaps for comments, blank lines, or removed
// declarations. Keeping only the ordering and grouping of token lines avoids
// overflowing the version-1 unit format's 16-bit line field in large apps.
func compactCoreLinkedTokenLines(tokens []unit.Token) {
	if len(tokens) == 0 {
		return
	}
	sourceLine := tokens[0].KindLine >> 8
	denseLine := 1
	for i := 0; i < len(tokens); i++ {
		line := tokens[i].KindLine >> 8
		if line != sourceLine {
			sourceLine = line
			denseLine++
		}
		tokens[i].KindLine = tokens[i].KindLine&255 | denseLine<<8
	}
}

func cloneCoreLinkString(value string) string {
	data := make([]byte, len(value))
	copy(data, []byte(value))
	return string(data)
}

func replaceFunctionValueProgram(dst *unit.Program, src *unit.Program) {
	assembly := dst.RTGAssembly
	assemblyFuncs := dst.RTGAssemblyFuncs
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
	dst.ConcurrencySites = src.ConcurrencySites
	dst.Packages = src.Packages
	dst.RTGAssembly = assembly
	dst.RTGAssemblyFuncs = assemblyFuncs
}

func beginLinkedPackageInfo(program *unit.Program, pkg build.PackageUnit) int {
	info := unit.PackageInfo{
		Name:       cloneCoreLinkString(pkg.Name),
		ImportPath: cloneCoreLinkString(pkg.ImportPath),
		GraphKeyA:  pkg.GraphKeyA,
		GraphKeyB:  pkg.GraphKeyB,
		SourceKeyA: pkg.SourceKeyA,
		SourceKeyB: pkg.SourceKeyB,
		TextStart:  len(program.Text),
		TokenStart: len(program.Tokens),
		DeclStart:  len(program.Decls),
		FuncStart:  len(program.Funcs),
	}
	program.Packages = append(program.Packages, info)
	return len(program.Packages) - 1
}

func finishLinkedPackageInfo(program *unit.Program, index int) {
	program.Packages[index].TextEnd = len(program.Text)
	program.Packages[index].TokenEnd = len(program.Tokens)
	program.Packages[index].DeclEnd = len(program.Decls)
	program.Packages[index].FuncEnd = len(program.Funcs)
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
		tokens[i].KindLine = tokens[i].KindLine&255 | line<<8
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
	concurrencyCap := 0
	for i := 0; i < len(programs); i++ {
		p := programs[i]
		textCap += len(p.Text) + 1
		declCap += len(p.Decls)
		funcCap += len(p.Funcs)
		concurrencyCap += len(p.ConcurrencySites)
	}
	program.Text = make([]byte, 0, textCap)
	program.Tokens = make([]unit.Token, 0, finalEOF+1)
	program.Decls = make([]unit.Decl, 0, declCap)
	program.Funcs = make([]unit.Func, 0, funcCap)
	program.ConcurrencySites = make([]unit.ConcurrencySite, 0, concurrencyCap)
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
	if len(src.Tokens) == 0 || src.Tokens[len(src.Tokens)-1].KindLine&255 != unit.TokenEOF {
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
	src.Tokens = append(src.Tokens, unit.MakeToken(unit.TokenFunc, start, 4, line))
	src.Tokens = append(src.Tokens, unit.MakeToken(unit.TokenIdent, start+5, 7, line))
	src.Tokens = append(src.Tokens, unit.MakeToken(unit.TokenOp, start+12, 1, line))
	src.Tokens = append(src.Tokens, unit.MakeToken(unit.TokenOp, start+13, 1, line))
	src.Tokens = append(src.Tokens, unit.MakeToken(unit.TokenIdent, start+15, 3, line))
	src.Tokens = append(src.Tokens, unit.MakeToken(unit.TokenOp, start+19, 1, line))
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
	if len(src.Tokens) == 0 || src.Tokens[len(src.Tokens)-1].KindLine&255 != unit.TokenEOF {
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
	return append(tokens, unit.MakeToken(kind, base+start, size, line))
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

func appendProgramCore(dst *unit.Program, src unit.Program, actions []tokenAction, finalEOF int, line int, aliases []string, hasNext bool, transient bool) (bool, int) {
	if src.Package == "" || len(src.Text) == 0 || len(src.Tokens) == 0 || len(actions) != len(src.Tokens) {
		return false, line
	}
	text := src.Text
	tokens := src.Tokens
	sourceEndsNewline := text[len(text)-1] == '\n'
	if transient && !prepareTransientCoreMappings(dst, &src, actions, finalEOF) {
		return false, line
	}
	lineBase := line
	sourceEndLine := tokens[len(tokens)-1].KindLine >> 8
	if sourceEndLine < 1 {
		sourceEndLine = 1
	}
	prevEnd := 0
	pendingStart := 0
	sourceDiscardStart := 0
	nextSourceDiscard := 16384
	for i := 0; i < len(tokens); i++ {
		action := actions[i]
		tok := tokens[i]
		if tok.KindLine&255 == unit.TokenEOF {
			if !transient {
				tokens[i].KindLine = tokens[i].KindLine&255 | len(dst.Tokens)<<8
			}
			continue
		}
		tokStart := tok.Start
		tokEnd := tok.Start + tok.Size
		if action < 0 {
			flushEnd := prevEnd
			if tokenActionRedirect(action) >= 0 {
				flushEnd = tokStart
			}
			if flushEnd > pendingStart {
				dst.Text = appendCoreBytes(dst.Text, text[pendingStart:flushEnd])
			}
			pendingStart = tokEnd
			if tokEnd > prevEnd {
				prevEnd = tokEnd
			}
			if !transient && tokenActionRedirect(action) < 0 {
				tokens[i].KindLine = tokens[i].KindLine&255 | finalEOF<<8
			}
			if transient && tokEnd >= nextSourceDiscard {
				arena.DiscardBytes(text[sourceDiscardStart:tokEnd])
				sourceDiscardStart = tokEnd - 1024
				nextSourceDiscard = tokEnd + 16384
			}
			if transient && (i+1)%2048 == 0 {
				discardTransientCoreTokenChunk(tokens, i)
			}
			continue
		}
		mappedToken := len(dst.Tokens)
		line = lineBase + (tok.KindLine >> 8) - 1
		if line < lineBase {
			line = lineBase
		}
		tok.KindLine = tok.KindLine&255 | line<<8
		replacementIndex := tokenActionReplacement(action)
		if replacementIndex >= 0 {
			if tokStart > pendingStart {
				dst.Text = appendCoreBytes(dst.Text, text[pendingStart:tokStart])
			}
			tok.Start = len(dst.Text)
			replacement := aliases[replacementIndex]
			dst.Text = appendCoreStringBytes(dst.Text, replacement)
			tok.KindLine = tok.KindLine>>8<<8 | coreLinkedReplacementTokenKind(tok.KindLine&255, replacement)
			tok.Size = len(replacement)
			pendingStart = tokEnd
		} else if tok.KindLine&255 == unit.TokenOp && tok.Size == 3 && tokStart+2 < len(text) && text[tokStart] == '.' && text[tokStart+1] == '.' && text[tokStart+2] == '.' {
			if tokStart > pendingStart {
				dst.Text = appendCoreBytes(dst.Text, text[pendingStart:tokStart])
			}
			tok.Start = len(dst.Text)
			dst.Text = appendCoreStringBytes(dst.Text, "...")
			for j := 0; j < 3; j++ {
				dot := tok
				dot.Start = tok.Start + j
				dot.Size = 1
				dst.Tokens = append(dst.Tokens, dot)
			}
			if !transient {
				tokens[i].KindLine = tokens[i].KindLine&255 | mappedToken<<8
			}
			pendingStart = tokEnd
			prevEnd = tokEnd
			if transient && tokEnd >= nextSourceDiscard {
				arena.DiscardBytes(text[sourceDiscardStart:tokEnd])
				sourceDiscardStart = tokEnd - 1024
				nextSourceDiscard = tokEnd + 16384
			}
			if transient && (i+1)%2048 == 0 {
				discardTransientCoreTokenChunk(tokens, i)
			}
			continue
		} else if tok.KindLine&255 == unit.TokenString && tokStart < len(text) && text[tokStart] == '"' {
			if tokStart > pendingStart {
				dst.Text = appendCoreBytes(dst.Text, text[pendingStart:tokStart])
			}
			tok.Start = len(dst.Text)
			value, ok := syntax.StringLiteralValue(text, syntax.MakeToken(syntax.TokenString, tokStart, tokEnd, 0))
			if !ok {
				return false, line
			}
			dst.Text = appendCoreQuotedString(dst.Text, value)
			tok.Size = len(dst.Text) - tok.Start
			pendingStart = tokEnd
		} else {
			tok.Start = len(dst.Text) + tokStart - pendingStart
		}
		dst.Tokens = append(dst.Tokens, tok)
		if !transient {
			tokens[i].KindLine = tokens[i].KindLine&255 | mappedToken<<8
		}
		prevEnd = tokEnd
		if transient && tokEnd >= nextSourceDiscard {
			if pendingStart < tokEnd {
				dst.Text = appendCoreBytes(dst.Text, text[pendingStart:tokEnd])
				pendingStart = tokEnd
			}
			arena.DiscardBytes(text[sourceDiscardStart:tokEnd])
			sourceDiscardStart = tokEnd - 1024
			nextSourceDiscard = tokEnd + 16384
		}
		if transient && (i+1)%2048 == 0 {
			discardTransientCoreTokenChunk(tokens, i)
		}
	}
	if pendingStart < len(text) {
		dst.Text = appendCoreBytes(dst.Text, text[pendingStart:])
	}
	if transient {
		arena.DiscardBytes(text[sourceDiscardStart:])
		remaining := len(tokens) % 2048
		if remaining != 0 {
			start := len(tokens) - remaining
			if start >= 256 {
				start -= 256
			}
			renvo_runtime_ArenaDiscardLinkTokens(tokens[start:])
		}
	}
	if !transient {
		for i := 0; i < len(tokens); i++ {
			target := tokenActionRedirect(actions[i])
			if target >= 0 {
				tokens[i].KindLine = tokens[i].KindLine&255 | mapLinkedToken(tokens, target, finalEOF)<<8
			}
		}
	}
	for i := 0; i < len(src.Decls); i++ {
		decl := src.Decls[i]
		if transient {
			nameStart, nameEnd, ok := mappedCoreTokenTextSpan(dst, decl.NameStart)
			if !ok {
				return false, line
			}
			decl.NameStart = nameStart
			decl.NameEnd = nameEnd
		} else {
			decl.StartTok = mapLinkedToken(tokens, decl.StartTok, finalEOF)
			decl.EndTok = mapLinkedToken(tokens, decl.EndTok, finalEOF)
			nameStart, nameEnd, ok := mapCoreTextSpanByToken(src, dst, finalEOF, decl.NameStart, decl.NameEnd)
			if !ok {
				return false, line
			}
			decl.NameStart = nameStart
			decl.NameEnd = nameEnd
		}
		dst.Decls = append(dst.Decls, decl)
	}
	funcBase := len(dst.Funcs)
	for i := 0; i < len(src.Funcs); i++ {
		fn := src.Funcs[i]
		if !transient {
			fn.StartTok = mapLinkedToken(tokens, fn.StartTok, finalEOF)
			fn.NameTok = mapLinkedToken(tokens, fn.NameTok, finalEOF)
		}
		nameStart, nameEnd, ok := mappedCoreTokenTextSpan(dst, fn.NameTok)
		if !ok {
			return false, line
		}
		fn.NameStart = nameStart
		fn.NameEnd = nameEnd
		if !transient {
			fn.ReceiverStart = mapLinkedToken(tokens, fn.ReceiverStart, finalEOF)
			fn.ReceiverEnd = mapLinkedToken(tokens, fn.ReceiverEnd, finalEOF)
			normalizeCoreLinkedReceiver(&fn, finalEOF)
			fn.BodyStart = mapLinkedToken(tokens, fn.BodyStart, finalEOF)
			fn.BodyEnd = mapLinkedToken(tokens, fn.BodyEnd, finalEOF)
			fn.EndTok = mapLinkedFuncEndToken(tokens, fn.EndTok, fn.BodyEnd, finalEOF)
		}
		dst.Funcs = append(dst.Funcs, fn)
	}
	assemblyBase := len(dst.RTGAssembly)
	for i := 0; i < len(src.RTGAssembly); i++ {
		item := src.RTGAssembly[i]
		path := cloneCoreLinkString(item.Path)
		source := make([]byte, len(item.Source))
		copy(source, item.Source)
		dst.RTGAssembly = append(dst.RTGAssembly, unit.RTGAssemblySource{Path: path, Source: source})
	}
	for i := 0; i < len(src.RTGAssemblyFuncs); i++ {
		item := src.RTGAssemblyFuncs[i]
		if item.Func < 0 || item.Func >= len(src.Funcs) || item.Source < 0 || item.Source >= len(src.RTGAssembly) || item.Entry < 0 {
			return false, line
		}
		item.Func += funcBase
		item.Source += assemblyBase
		dst.RTGAssemblyFuncs = append(dst.RTGAssemblyFuncs, item)
	}
	for i := 0; i < len(src.ConcurrencySites); i++ {
		site := src.ConcurrencySites[i]
		if !transient {
			site.Token = mapLinkedToken(tokens, site.Token, finalEOF)
		}
		if site.Token < 0 || site.Token >= finalEOF {
			return false, line
		}
		dst.ConcurrencySites = append(dst.ConcurrencySites, site)
	}
	line = lineBase + sourceEndLine - 1
	if hasNext && !sourceEndsNewline {
		dst.Text = append(dst.Text, '\n')
		line++
	}
	return true, line
}

func discardTransientCoreTokenChunk(tokens []unit.Token, index int) {
	start := index + 1 - 2048
	if start >= 256 {
		start -= 256
	}
	renvo_runtime_ArenaDiscardLinkTokens(tokens[start : index+1])
}

const transientCoreMappingStride = 32

// Transient linking can retire each source-token chunk as soon as it has been
// copied. Resolve every later metadata and redirect lookup first, borrowing the
// unused tail of the already-reserved destination token array for sparse prefix
// checkpoints instead of allocating another full-token mapping.
func prepareTransientCoreMappings(dst *unit.Program, src *unit.Program, actions []tokenAction, eof int) bool {
	base := len(dst.Tokens)
	checkpointCount := (len(actions) + transientCoreMappingStride - 1) / transientCoreMappingStride
	borrowed := base+checkpointCount <= cap(dst.Tokens)
	var checkpoints []unit.Token
	if borrowed {
		dst.Tokens = dst.Tokens[:base+checkpointCount]
		checkpoints = dst.Tokens[base:]
	} else {
		checkpoints = make([]unit.Token, checkpointCount)
	}
	mapped := base
	for i := 0; i < len(actions); i++ {
		if i%transientCoreMappingStride == 0 {
			checkpoints[i/transientCoreMappingStride].Start = mapped
		}
		mapped += transientCoreTokenOutputCount(src, actions[i], i)
	}
	for i := 0; i < len(actions); i++ {
		target := tokenActionRedirect(actions[i])
		if target >= 0 {
			target = transientCoreMappedPosition(src, actions, checkpoints, target)
			if target < 0 {
				return false
			}
			actions[i] = tokenAction(-target - 2)
		}
	}
	for i := 0; i < len(src.Decls); i++ {
		decl := &src.Decls[i]
		nameTok := coreTokenIndexByTextSpan(src.Tokens, decl.NameStart, decl.NameEnd)
		nameTok = transientCoreMappedToken(src, actions, checkpoints, nameTok, eof)
		decl.StartTok = transientCoreMappedToken(src, actions, checkpoints, decl.StartTok, eof)
		decl.EndTok = transientCoreMappedToken(src, actions, checkpoints, decl.EndTok, eof)
		if nameTok < 0 || decl.StartTok < 0 || decl.EndTok < 0 {
			return false
		}
		decl.NameStart = nameTok
		decl.NameEnd = nameTok
	}
	for i := 0; i < len(src.Funcs); i++ {
		fn := &src.Funcs[i]
		fn.StartTok = transientCoreMappedToken(src, actions, checkpoints, fn.StartTok, eof)
		fn.NameTok = transientCoreMappedToken(src, actions, checkpoints, fn.NameTok, eof)
		fn.ReceiverStart = transientCoreMappedToken(src, actions, checkpoints, fn.ReceiverStart, eof)
		fn.ReceiverEnd = transientCoreMappedToken(src, actions, checkpoints, fn.ReceiverEnd, eof)
		normalizeCoreLinkedReceiver(fn, eof)
		fn.BodyStart = transientCoreMappedToken(src, actions, checkpoints, fn.BodyStart, eof)
		fn.BodyEnd = transientCoreMappedToken(src, actions, checkpoints, fn.BodyEnd, eof)
		mappedEnd := transientCoreMappedToken(src, actions, checkpoints, fn.EndTok, eof)
		if mappedEnd == eof && fn.BodyEnd >= 0 && fn.BodyEnd+1 <= eof {
			mappedEnd = fn.BodyEnd + 1
		}
		fn.EndTok = mappedEnd
		if fn.StartTok < 0 || fn.NameTok < 0 || fn.BodyStart < 0 || fn.BodyEnd < 0 || fn.EndTok < 0 {
			return false
		}
	}
	for i := 0; i < len(src.ConcurrencySites); i++ {
		site := &src.ConcurrencySites[i]
		site.Token = transientCoreMappedToken(src, actions, checkpoints, site.Token, eof)
		if site.Token < 0 || site.Token >= eof {
			return false
		}
	}
	if borrowed {
		dst.Tokens = dst.Tokens[:base]
	}
	return true
}

func transientCoreTokenOutputCount(src *unit.Program, action tokenAction, index int) int {
	if action < 0 || index < 0 || index >= len(src.Tokens) || src.Tokens[index].KindLine&255 == unit.TokenEOF {
		return 0
	}
	tok := src.Tokens[index]
	if tok.KindLine&255 == unit.TokenOp && tok.Size == 3 && tok.Start >= 0 && tok.Start+2 < len(src.Text) &&
		src.Text[tok.Start] == '.' && src.Text[tok.Start+1] == '.' && src.Text[tok.Start+2] == '.' {
		return 3
	}
	return 1
}

func transientCoreMappedToken(src *unit.Program, actions []tokenAction, checkpoints []unit.Token, tok int, eof int) int {
	if tok < 0 {
		return eof
	}
	if tok >= len(actions) {
		return -1
	}
	if actions[tok] < 0 {
		target := tokenActionRedirect(actions[tok])
		if target >= 0 {
			return target
		}
		return eof
	}
	return transientCoreMappedPosition(src, actions, checkpoints, tok)
}

func transientCoreMappedPosition(src *unit.Program, actions []tokenAction, checkpoints []unit.Token, tok int) int {
	if tok < 0 || tok >= len(actions) {
		return -1
	}
	block := tok / transientCoreMappingStride
	mapped := checkpoints[block].Start
	start := block * transientCoreMappingStride
	for i := start; i < tok; i++ {
		mapped += transientCoreTokenOutputCount(src, actions[i], i)
	}
	return mapped
}

func appendCoreQuotedString(out []byte, value string) []byte {
	const hex = "0123456789abcdef"
	out = append(out, '"')
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if ch == '"' || ch == '\\' {
			out = append(out, '\\', ch)
		} else if ch < 32 || ch >= 127 {
			out = append(out, '\\', 'x', hex[ch/16], hex[ch%16])
		} else {
			out = append(out, ch)
		}
	}
	return append(out, '"')
}

func linkedTokenActions(program *unit.Program, aliases *[]string, symbolOffsets []int, actions []tokenAction, plusReplacement int) bool {
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
		markCoreUnsafeLayoutTokens(program, actions)
		markCoreUnsafePointerCallTokens(program, actions)
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

func markCoreImportDeclTokens(program *unit.Program, actions []tokenAction, imp unit.Import) {
	if imp.PathTok < 0 || imp.PathTok >= len(program.Tokens) {
		return
	}
	line := program.Tokens[imp.PathTok].KindLine >> 8
	start := imp.PathTok
	if imp.NameTok >= 0 && imp.NameTok < start {
		start = imp.NameTok
	}
	for start > 0 && program.Tokens[start-1].KindLine>>8 == line {
		start--
	}
	end := imp.PathTok
	for end+1 < len(program.Tokens) && program.Tokens[end+1].KindLine>>8 == line {
		end++
	}
	for i := start; i <= end; i++ {
		actions[i] = -1
	}
}

func markCoreRedirectToken(actions []tokenAction, tok int, target int) {
	if tok < 0 || tok >= len(actions) || target < 0 || target >= len(actions) {
		return
	}
	actions[tok] = tokenAction(-target - 2)
}

func markReplacementToken(actions []tokenAction, tok int, replacement int) {
	if tok < 0 || tok >= len(actions) || replacement < 0 || actions[tok] < 0 {
		return
	}
	actions[tok] = tokenAction(replacement + 1)
}

func markCoreSkipToken(actions []tokenAction, tok int) {
	if tok < 0 || tok >= len(actions) {
		return
	}
	actions[tok] = -1
}

func markCoreUnsafePointerCallTokens(program *unit.Program, actions []tokenAction) {
	for i := 0; i < len(program.Selectors); i++ {
		selector := program.Selectors[i]
		if !coreSelectorIsUnsafePointer(program, selector) {
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

func markCoreUnsafeLayoutTokens(program *unit.Program, actions []tokenAction) {
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
			if coreTokenText(program, tok) == name && coreTokenTextEquals(program, tok+1, ".") && (coreTokenTextEquals(program, tok+2, "Sizeof") || coreTokenTextEquals(program, tok+2, "Offsetof")) {
				markCoreRedirectToken(actions, tok, tok+2)
				markCoreRedirectToken(actions, tok+1, tok+2)
			}
		}
	}
}

func markCoreUnsafePointerConversionTokens(program *unit.Program, actions []tokenAction) {
	for i := 0; i+4 < len(program.Tokens); i++ {
		if !coreTokenTextEquals(program, i, "(") || !coreTokenTextEquals(program, i+1, "*") {
			continue
		}
		typeEnd := findCoreMatchingParen(program, i)
		if typeEnd <= i+2 || typeEnd+1 >= len(program.Tokens) || !coreTokenTextEquals(program, typeEnd+1, "(") {
			continue
		}
		// Keep pointer-to-array conversions so the backend retains the element
		// type and array bound needed for dynamic indexing. The nested
		// unsafe.Pointer call is still erased independently.
		if coreTokenTextEquals(program, i+2, "[") {
			continue
		}
		// Keep a typed conversion that directly reinterprets an address. Erasing
		// both conversions would turn (*uint64)(unsafe.Pointer(&words32)) into
		// &words32 and lose the destination pointee type required by the backend.
		if coreUnsafePointerAddressCallAt(program, typeEnd+2) {
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

func coreUnsafePointerAddressCallAt(program *unit.Program, start int) bool {
	for i := 0; i < len(program.Selectors); i++ {
		selector := program.Selectors[i]
		if selector.BaseTok != start || !coreSelectorIsUnsafePointer(program, selector) {
			continue
		}
		open := selector.NameTok + 1
		return coreTokenTextEquals(program, open, "(") && coreTokenTextEquals(program, open+1, "&")
	}
	return false
}

func coreSelectorIsUnsafePointer(program *unit.Program, selector unit.Selector) bool {
	if selector.BaseKind != unit.RefImport || !coreTokenTextEquals(program, selector.NameTok, "Pointer") {
		return false
	}
	for i := 0; i < len(program.Imports); i++ {
		imp := program.Imports[i]
		if !coreTokenTextEquals(program, imp.PathTok, "\"unsafe\"") && !coreTokenTextEquals(program, imp.PathTok, "`unsafe`") {
			continue
		}
		name := "unsafe"
		if imp.NameTok >= 0 {
			name = coreTokenText(program, imp.NameTok)
		}
		return coreTokenText(program, selector.BaseTok) == name
	}
	return false
}

func markCoreEndianSelectorTokens(program *unit.Program, actions []tokenAction) {
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

type tokenAction int32

func tokenActionSkips(action tokenAction) bool {
	return action < 0
}

func tokenActionRedirect(action tokenAction) int {
	if action <= -2 {
		return int(-action - 2)
	}
	return -1
}

func tokenActionReplacement(action tokenAction) int {
	if action > 0 {
		return int(action - 1)
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
			// Method identity includes its receiver. Receiver types are already
			// package-aliased when needed, and method selectors inside the owning
			// package intentionally retain the authored method spelling. Aliasing a
			// colliding "Device.Read" declaration alone would disconnect d.Read().
			if coreSymbolIsMethod(programs[i], programs[i].Symbols[j]) {
				continue
			}
			directiveSize := -1
			nameTok := programs[i].Symbols[j].Token
			if nameTok > 0 && nameTok < len(programs[i].Tokens) && programs[i].Tokens[nameTok-1].KindLine&255 == unit.TokenFunc {
				end := programs[i].Tokens[nameTok-1].Start - 1
				if end >= 12 && (end == 12 || programs[i].Text[end-13] == '\n') && coreText(programs[i].Text, end-12, end) == "//renvo:load" {
					directiveSize = coreMemoryDirectiveSize(&programs[i], nameTok, true)
				} else if end >= 13 && (end == 13 || programs[i].Text[end-14] == '\n') && coreText(programs[i].Text, end-13, end) == "//renvo:store" {
					directiveSize = coreMemoryDirectiveSize(&programs[i], nameTok, false)
				}
			}
			if name == "init" {
				out[index] = coreInitFunctionAliasName(i, initOrdinal)
				initOrdinal++
			} else if len(name) > 2 && name[0] == 'C' && name[1] == '.' {
				// Explicit mixed-language packages keep C declarations in a
				// separate logical namespace. Give their emitted root-package
				// tokens a legal, collision-free backend name as well.
				out[index] = coreSymbolAliasName(i, name)
			} else if alias := coreCompilerIntrinsicAlias(programs[i].ImportPath, name); alias != "" {
				out[index] = alias
			} else if directiveSize >= 0 {
				out[index] = coreMemoryDirectiveAliasName(directiveSize, index)
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
			if out[index] == "" && duplicate[index] && programs[i].Symbols[j].Name != "init" && !coreSymbolKeepsRuntimeName(programs[i].Symbols[j].Name) {
				out[index] = coreSymbolAliasName(i, programs[i].Symbols[j].Name)
			}
		}
	}
	return out
}

func coreSymbolIsMethod(program unit.Program, symbol unit.Symbol) bool {
	for i := 0; i < len(program.Funcs); i++ {
		fn := program.Funcs[i]
		if fn.NameTok == symbol.Token && fn.ReceiverStart < fn.ReceiverEnd {
			return true
		}
	}
	return false
}

func coreMemoryDirectiveSize(program *unit.Program, nameTok int, load bool) int {
	closeTok := nameTok + 1
	depth := 0
	for closeTok < len(program.Tokens) {
		text := coreTokenText(program, closeTok)
		if text == "(" {
			depth++
		} else if text == ")" {
			depth--
			if depth == 0 {
				break
			}
		}
		closeTok++
	}
	typeTok := closeTok - 1
	if load {
		typeTok = closeTok + 1
		if coreTokenText(program, typeTok) == "(" {
			typeTok++
		}
	}
	name := coreTokenText(program, typeTok)
	if name == "byte" || name == "uint8" || name == "int8" {
		return 1
	}
	if name == "uint16" || name == "int16" {
		return 2
	}
	if name == "uint32" || name == "int32" {
		return 4
	}
	if name == "uint" || name == "int" || name == "uintptr" {
		return 0
	}
	return -1
}

// Compiler intrinsics are selected by both package identity and declaration
// name while the linker still has semantic package information. The compact
// backend unit intentionally omits that frontend-only metadata, so a reserved
// alias carries only the identity needed for safe call-site specialization.
// Calls through function values still reach the ordinary function body.
func coreCompilerIntrinsicAlias(importPath string, name string) string {
	if importPath == "fmt" && name == "Println" {
		return "renvo_runtime_FmtPrintln"
	}
	if (importPath == "os" || importPath == "renvo.dev/std/os") && name == "syscall" {
		return "renvo_runtime_Syscall"
	}
	return ""
}

func coreSymbolKeepsRuntimeName(name string) bool {
	switch name {
	case "renvo_runtime_Exit",
		"renvo_runtime_PrintMirror",
		"renvo_runtime_Call",
		"renvo_runtime_ThreadStateSwap",
		"renvo_runtime_StackRun",
		"renvo_runtime_StackSupported",
		"renvo_runtime_StackInit",
		"renvo_runtime_StackSwitch",
		"renvo_runtime_Spawn",
		"renvo_runtime_ChanCreate",
		"renvo_runtime_ChanSend",
		"renvo_runtime_ChanReceive",
		"renvo_runtime_ChanSelect",
		"renvo_runtime_ChanClose",
		"renvo_runtime_ChanLen",
		"renvo_runtime_ChanCap",
		"renvo_runtime_ChanSelectValue",
		"renvo_runtime_Channel",
		"renvo_runtime_SelectReceive",
		"renvo_runtime_SelectSend",
		"renvo_runtime_ArenaMark",
		"renvo_runtime_ArenaReset",
		"renvo_runtime_ArenaPersistMark",
		"renvo_runtime_ArenaPersistReset",
		"renvo_runtime_ArenaPersistString",
		"renvo_runtime_ArenaPersistBytes",
		"renvo_runtime_ArenaPersistCheckNameRefs",
		"renvo_runtime_ArenaPersistCheckSelectorRefs",
		"renvo_runtime_ArenaPersistCheckTypeRefs",
		"renvo_runtime_ArenaPersistCheckBools",
		"renvo_runtime_ArenaDiscard",
		"renvo_runtime_ArenaDiscardBytes",
		"renvo_runtime_ArenaDiscardDecls",
		"renvo_runtime_ArenaDiscardFuncs",
		"renvo_runtime_ArenaDiscardUnitTokens",
		"renvo_runtime_ArenaDiscardLinkTokens",
		"renvo_runtime_ArenaDiscardLowerTokens":
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

func coreMemoryDirectiveAliasName(size int, function int) string {
	out := []byte("renvo_runtime_M")
	out = append(out, byte('0'+size))
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

func coreTokenIndexByTextSpan(tokens []unit.Token, start int, end int) int {
	low := 0
	high := len(tokens)
	for low < high {
		mid := low + (high-low)/2
		if tokens[mid].Start < start {
			low = mid + 1
		} else {
			high = mid
		}
	}
	if low < len(tokens) {
		tok := tokens[low]
		if tok.Start == start && tok.Start+tok.Size == end {
			return low
		}
	}
	return -1
}

func mappedCoreTokenTextSpan(program *unit.Program, tok int) (int, int, bool) {
	if tok < 0 || tok >= len(program.Tokens) {
		return 0, 0, false
	}
	token := program.Tokens[tok]
	if token.KindLine&255 == unit.TokenEOF || token.Start < 0 || token.Start+token.Size > len(program.Text) {
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
	mapped := tokens[tok].KindLine >> 8
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
	return append(out, data...)
}
