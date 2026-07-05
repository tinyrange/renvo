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

func LinkBuild(result build.Result) Result {
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
	pkg := 0
	ok := false
	program, pkg, ok = LinkUnitData(result.Units, result.Root)
	if !ok {
		out.Ok = false
		out.Error = LinkErrUnit
		out.ErrorPackage = pkg
		return out
	}
	var data []byte
	data, ok = unit.Marshal(program)
	if !ok {
		out.Ok = false
		out.Error = LinkErrUnit
		return out
	}
	out.Program = program
	out.Data = data
	return out
}

func LinkUnitData(units []build.PackageUnit, root int) (unit.Program, int, bool) {
	var empty unit.Program
	if root < 0 || root >= len(units) {
		return empty, -1, false
	}
	programs := make([]unit.Program, len(units))
	for i := 0; i < len(units); i++ {
		prog, ok := unit.Unmarshal(units[i].Data)
		if !ok {
			return empty, i, false
		}
		if units[i].Name != "" && prog.Package != units[i].Name {
			return empty, i, false
		}
		programs[i] = prog
	}
	program, ok := LinkPrograms(programs, root, units[root].Name)
	if !ok {
		return empty, -1, false
	}
	return program, -1, true
}

func LinkUnits(units []build.PackageUnit, root int) (unit.Program, bool) {
	var empty unit.Program
	if root < 0 || root >= len(units) {
		return empty, false
	}
	programs := make([]unit.Program, len(units))
	for i := 0; i < len(units); i++ {
		programs[i] = units[i].Program
	}
	return LinkPrograms(programs, root, units[root].Name)
}

func LinkPrograms(programs []unit.Program, root int, rootName string) (unit.Program, bool) {
	var empty unit.Program
	if root < 0 || root >= len(programs) || rootName == "" {
		return empty, false
	}
	programs, ok := preparePrograms(programs, root)
	if !ok {
		return empty, false
	}
	program := unit.Program{Package: rootName, ImportPath: programs[root].ImportPath}
	finalEOF := countLinkedTokens(programs)
	symbolOffsets := packageSymbolOffsets(programs)
	aliases := packageSymbolAliases(programs, root, symbolOffsets)
	lineOffset := 0
	for i := 0; i < len(programs); i++ {
		ok := appendProgram(&program, programs[i], finalEOF, lineOffset, symbolOffsets, aliases, i+1 < len(programs))
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

func preparePrograms(programs []unit.Program, root int) ([]unit.Program, bool) {
	out := make([]unit.Program, len(programs))
	copy(out, programs)
	rootProgram, ok := addRootEntrypoint(out[root], root)
	if !ok {
		return nil, false
	}
	out[root] = rootProgram
	return out, true
}

func addRootEntrypoint(src unit.Program, packageIndex int) (unit.Program, bool) {
	if src.Package != "main" || findFuncByName(src, "appMain") >= 0 || findFuncByName(src, "main") < 0 {
		return src, true
	}
	if len(src.Tokens) == 0 || src.Tokens[len(src.Tokens)-1].Kind != unit.TokenEOF {
		return src, false
	}
	var textCopy []byte
	src.Text = appendBytes(textCopy, src.Text)
	src.Tokens = copyTokens(src.Tokens, len(src.Tokens)-1)
	src.Funcs = copyFuncs(src.Funcs)
	src.Signatures = copySignatures(src.Signatures)
	src.Stmts = copyStatements(src.Stmts)
	src.Returns = copyReturns(src.Returns)
	src.Calls = copyCalls(src.Calls)
	src.Refs = copyRefs(src.Refs)
	src.Symbols = copySymbols(src.Symbols)
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
	funcIndex := len(src.Funcs)
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
	src.Signatures = append(src.Signatures, unit.FuncSignature{
		FuncIndex: funcIndex,
		Results:   []unit.Field{{NameTok: -1, TypeStart: base + 4, TypeEnd: base + 5}},
	})
	src.Stmts = append(src.Stmts, unit.Statement{FuncIndex: funcIndex, Kind: unit.StmtBlock, StartTok: base + 5, EndTok: base + 13, ExprStart: -1, ExprEnd: -1, BodyStart: base + 5, BodyEnd: base + 13, ElseStart: -1, ElseEnd: -1})
	src.Stmts = append(src.Stmts, unit.Statement{FuncIndex: funcIndex, Kind: unit.StmtExpr, StartTok: base + 6, EndTok: base + 10, ExprStart: base + 6, ExprEnd: base + 9, BodyStart: -1, BodyEnd: -1, ElseStart: -1, ElseEnd: -1})
	src.Stmts = append(src.Stmts, unit.Statement{FuncIndex: funcIndex, Kind: unit.StmtReturn, StartTok: base + 10, EndTok: base + 12, ExprStart: base + 11, ExprEnd: base + 12, BodyStart: -1, BodyEnd: -1, ElseStart: -1, ElseEnd: -1})
	src.Returns = append(src.Returns, unit.Return{
		FuncIndex: funcIndex,
		StartTok:  base + 10,
		EndTok:    base + 12,
		Values:    []unit.ExprSpan{{StartTok: base + 11, EndTok: base + 12}},
	})
	src.Calls = append(src.Calls, unit.Call{
		OwnerKind:  unit.OwnerFunc,
		OwnerIndex: funcIndex,
		Kind:       unit.CallPackage,
		CalleeTok:  base + 6,
		BaseTok:    eof,
		DotTok:     eof,
		ArgsStart:  base + 8,
		ArgsEnd:    base + 8,
	})
	mainSymbol := findFuncSymbol(src, "main")
	src.Refs = append(src.Refs, unit.NameRef{
		OwnerKind:  unit.OwnerFunc,
		OwnerIndex: funcIndex,
		Kind:       unit.RefPackage,
		Token:      base + 6,
		Index:      mainSymbol,
		Package:    packageIndex,
	})
	src.Symbols = append(src.Symbols, unit.Symbol{
		Name:       "appMain",
		Kind:       unit.SymbolFunc,
		Package:    packageIndex,
		Token:      base + 1,
		OwnerKind:  unit.OwnerFunc,
		OwnerIndex: funcIndex,
	})
	return src, true
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

func findFuncSymbol(program unit.Program, name string) int {
	for i := 0; i < len(program.Symbols); i++ {
		symbol := program.Symbols[i]
		if symbol.Kind == unit.SymbolFunc && symbol.Name == name {
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

func appendProgram(dst *unit.Program, src unit.Program, finalEOF int, lineOffset int, symbolOffsets []int, aliases []string, hasNext bool) bool {
	if src.Package == "" || len(src.Text) == 0 || len(src.Tokens) == 0 {
		return false
	}
	symbolOffset := len(dst.Symbols)
	declOffset := len(dst.Decls)
	funcOffset := len(dst.Funcs)
	typeOffset := len(dst.Types)
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
	for i := 0; i < len(src.DeclMeta); i++ {
		meta, ok := mapDeclMeta(src.DeclMeta[i], oldToNew, finalEOF, declOffset, symbolOffset)
		if !ok {
			return false
		}
		dst.DeclMeta = append(dst.DeclMeta, meta)
	}
	for i := 0; i < len(src.InitOrder); i++ {
		decl := src.InitOrder[i]
		if decl < 0 || decl >= len(src.Decls) {
			return false
		}
		dst.InitOrder = append(dst.InitOrder, declOffset+decl)
	}
	for i := 0; i < len(src.Consts); i++ {
		value, ok := mapConst(src.Consts[i], declOffset, len(src.Decls))
		if !ok {
			return false
		}
		dst.Consts = append(dst.Consts, value)
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
	for i := 0; i < len(src.Symbols); i++ {
		symbol := src.Symbols[i]
		if symbol.Token >= 0 && symbol.Token < len(replacements) && replacements[symbol.Token] != "" {
			symbol.Name = replacements[symbol.Token]
		}
		symbol, ok := mapSymbol(symbol, oldToNew, finalEOF, declOffset, funcOffset)
		if !ok {
			return false
		}
		dst.Symbols = append(dst.Symbols, symbol)
	}
	for i := 0; i < len(src.Signatures); i++ {
		sig, ok := mapSignature(src.Signatures[i], oldToNew, finalEOF, funcOffset)
		if !ok {
			return false
		}
		dst.Signatures = append(dst.Signatures, sig)
	}
	for i := 0; i < len(src.Stmts); i++ {
		stmt, ok := mapStatement(src.Stmts[i], oldToNew, finalEOF, funcOffset, len(src.Funcs))
		if !ok {
			return false
		}
		dst.Stmts = append(dst.Stmts, stmt)
	}
	for i := 0; i < len(src.Types); i++ {
		typ, ok := mapType(src, dst, src.Types[i], oldToNew, finalEOF, declOffset, symbolOffset)
		if !ok {
			return false
		}
		dst.Types = append(dst.Types, typ)
	}
	for i := 0; i < len(src.TypeFields); i++ {
		fields, ok := mapTypeFields(src.TypeFields[i], oldToNew, finalEOF, typeOffset, len(src.Types))
		if !ok {
			return false
		}
		dst.TypeFields = append(dst.TypeFields, fields)
	}
	for i := 0; i < len(src.TypeIfaces); i++ {
		iface, ok := mapTypeInterface(src.TypeIfaces[i], oldToNew, finalEOF, typeOffset, len(src.Types))
		if !ok {
			return false
		}
		dst.TypeIfaces = append(dst.TypeIfaces, iface)
	}
	for i := 0; i < len(src.TypeFuncs); i++ {
		fn, ok := mapTypeFunc(src.TypeFuncs[i], oldToNew, finalEOF, typeOffset, len(src.Types))
		if !ok {
			return false
		}
		dst.TypeFuncs = append(dst.TypeFuncs, fn)
	}
	for i := 0; i < len(src.Methods); i++ {
		method, ok := mapMethod(src.Methods[i], oldToNew, finalEOF, typeOffset, symbolOffset, funcOffset, len(src.Types), len(src.Symbols), len(src.Funcs))
		if !ok {
			return false
		}
		dst.Methods = append(dst.Methods, method)
	}
	for i := 0; i < len(src.TypeRefs); i++ {
		ref, ok := mapTypeRef(src.TypeRefs[i], oldToNew, finalEOF, declOffset, funcOffset, symbolOffsets)
		if !ok {
			return false
		}
		dst.TypeRefs = append(dst.TypeRefs, ref)
	}
	for i := 0; i < len(src.Locals); i++ {
		local, ok := mapLocal(dst, src.Locals[i], oldToNew, finalEOF, funcOffset)
		if !ok {
			return false
		}
		dst.Locals = append(dst.Locals, local)
	}
	for i := 0; i < len(src.Indexes); i++ {
		index, ok := mapIndex(src.Indexes[i], oldToNew, finalEOF, declOffset, funcOffset)
		if !ok {
			return false
		}
		dst.Indexes = append(dst.Indexes, index)
	}
	for i := 0; i < len(src.Composites); i++ {
		composite, ok := mapComposite(src.Composites[i], oldToNew, finalEOF, declOffset, funcOffset)
		if !ok {
			return false
		}
		dst.Composites = append(dst.Composites, composite)
	}
	for i := 0; i < len(src.Assigns); i++ {
		assign, ok := mapAssignment(src.Assigns[i], oldToNew, finalEOF, funcOffset)
		if !ok {
			return false
		}
		dst.Assigns = append(dst.Assigns, assign)
	}
	for i := 0; i < len(src.Returns); i++ {
		ret, ok := mapReturn(src.Returns[i], oldToNew, finalEOF, funcOffset)
		if !ok {
			return false
		}
		dst.Returns = append(dst.Returns, ret)
	}
	for i := 0; i < len(src.Calls); i++ {
		call, ok := mapCall(src.Calls[i], oldToNew, finalEOF, declOffset, funcOffset)
		if !ok {
			return false
		}
		dst.Calls = append(dst.Calls, call)
	}
	for i := 0; i < len(src.Refs); i++ {
		if src.Refs[i].Kind == unit.RefImport {
			continue
		}
		ref, ok := mapRef(src.Refs[i], oldToNew, finalEOF, declOffset, funcOffset, symbolOffsets)
		if !ok {
			return false
		}
		dst.Refs = append(dst.Refs, ref)
	}
	for i := 0; i < len(src.Selectors); i++ {
		if src.Selectors[i].BaseKind == unit.RefImport {
			continue
		}
		selector, ok := mapSelector(src.Selectors[i], oldToNew, finalEOF, declOffset, funcOffset, symbolOffsets)
		if !ok {
			return false
		}
		dst.Selectors = append(dst.Selectors, selector)
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

func copyTokens(src []unit.Token, limit int) []unit.Token {
	var out []unit.Token
	for i := 0; i < limit && i < len(src); i++ {
		out = append(out, src[i])
	}
	return out
}

func copyFuncs(src []unit.Func) []unit.Func {
	var out []unit.Func
	for i := 0; i < len(src); i++ {
		out = append(out, src[i])
	}
	return out
}

func copySignatures(src []unit.FuncSignature) []unit.FuncSignature {
	var out []unit.FuncSignature
	for i := 0; i < len(src); i++ {
		out = append(out, src[i])
	}
	return out
}

func copyStatements(src []unit.Statement) []unit.Statement {
	var out []unit.Statement
	for i := 0; i < len(src); i++ {
		out = append(out, src[i])
	}
	return out
}

func copyReturns(src []unit.Return) []unit.Return {
	var out []unit.Return
	for i := 0; i < len(src); i++ {
		out = append(out, src[i])
	}
	return out
}

func copyCalls(src []unit.Call) []unit.Call {
	var out []unit.Call
	for i := 0; i < len(src); i++ {
		out = append(out, src[i])
	}
	return out
}

func copyRefs(src []unit.NameRef) []unit.NameRef {
	var out []unit.NameRef
	for i := 0; i < len(src); i++ {
		out = append(out, src[i])
	}
	return out
}

func copySymbols(src []unit.Symbol) []unit.Symbol {
	var out []unit.Symbol
	for i := 0; i < len(src); i++ {
		out = append(out, src[i])
	}
	return out
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

func mapSymbol(symbol unit.Symbol, oldToNew []int, eof int, declOffset int, funcOffset int) (unit.Symbol, bool) {
	if len(symbol.Name) == 0 ||
		symbol.Kind < unit.SymbolConst || symbol.Kind > unit.SymbolMethod ||
		symbol.Package < -1 {
		return symbol, false
	}
	symbol.Token = mapToken(oldToNew, symbol.Token, eof)
	ownerIndex, ok := mapOwner(symbol.OwnerKind, symbol.OwnerIndex, declOffset, funcOffset)
	if !ok {
		return symbol, false
	}
	symbol.OwnerIndex = ownerIndex
	if symbol.Token < 0 {
		return symbol, false
	}
	return symbol, true
}

func mapConst(value unit.ConstValue, declOffset int, declLimit int) (unit.ConstValue, bool) {
	if value.DeclIndex < 0 || value.DeclIndex >= declLimit ||
		value.Kind < unit.ConstInt || value.Kind > unit.ConstBool {
		return value, false
	}
	value.DeclIndex += declOffset
	return value, true
}

func mapDeclMeta(meta unit.DeclMeta, oldToNew []int, eof int, declOffset int, symbolOffset int) (unit.DeclMeta, bool) {
	if meta.DeclIndex < 0 {
		return meta, false
	}
	meta.DeclIndex += declOffset
	if meta.Symbol >= 0 {
		meta.Symbol += symbolOffset
	}
	var ok bool
	meta.TypeStart, meta.TypeEnd, ok = mapNullableSpan(meta.TypeStart, meta.TypeEnd, oldToNew, eof)
	if !ok {
		return meta, false
	}
	meta.ValueStart, meta.ValueEnd, ok = mapNullableSpan(meta.ValueStart, meta.ValueEnd, oldToNew, eof)
	if !ok {
		return meta, false
	}
	for i := 0; i < len(meta.Values); i++ {
		meta.Values[i] = mapExprSpan(meta.Values[i], oldToNew, eof)
	}
	return meta, true
}

func mapSignature(sig unit.FuncSignature, oldToNew []int, eof int, funcOffset int) (unit.FuncSignature, bool) {
	if sig.FuncIndex < 0 {
		return sig, false
	}
	sig.FuncIndex += funcOffset
	var ok bool
	sig.Receiver, ok = mapFields(sig.Receiver, oldToNew, eof)
	if !ok {
		return sig, false
	}
	sig.Params, ok = mapFields(sig.Params, oldToNew, eof)
	if !ok {
		return sig, false
	}
	sig.Results, ok = mapFields(sig.Results, oldToNew, eof)
	if !ok {
		return sig, false
	}
	return sig, true
}

func mapStatement(stmt unit.Statement, oldToNew []int, eof int, funcOffset int, funcLimit int) (unit.Statement, bool) {
	if stmt.FuncIndex < 0 || stmt.FuncIndex >= funcLimit || stmt.Kind < unit.StmtOther || stmt.Kind > unit.StmtLabel {
		return stmt, false
	}
	stmt.FuncIndex += funcOffset
	stmt.StartTok = mapToken(oldToNew, stmt.StartTok, eof)
	stmt.EndTok = mapToken(oldToNew, stmt.EndTok, eof)
	var ok bool
	stmt.ExprStart, stmt.ExprEnd, ok = mapNullableSpan(stmt.ExprStart, stmt.ExprEnd, oldToNew, eof)
	if !ok {
		return stmt, false
	}
	stmt.BodyStart, stmt.BodyEnd, ok = mapNullableSpan(stmt.BodyStart, stmt.BodyEnd, oldToNew, eof)
	if !ok {
		return stmt, false
	}
	stmt.ElseStart, stmt.ElseEnd, ok = mapNullableSpan(stmt.ElseStart, stmt.ElseEnd, oldToNew, eof)
	if !ok {
		return stmt, false
	}
	if stmt.StartTok < 0 || stmt.EndTok < stmt.StartTok {
		return stmt, false
	}
	return stmt, true
}

func mapFields(fields []unit.Field, oldToNew []int, eof int) ([]unit.Field, bool) {
	for i := 0; i < len(fields); i++ {
		fields[i].NameTok = mapNullableToken(fields[i].NameTok, oldToNew, eof)
		fields[i].TypeStart = mapToken(oldToNew, fields[i].TypeStart, eof)
		fields[i].TypeEnd = mapToken(oldToNew, fields[i].TypeEnd, eof)
		if fields[i].NameTok < -1 || fields[i].TypeStart < 0 || fields[i].TypeEnd < fields[i].TypeStart {
			return fields, false
		}
	}
	return fields, true
}

func mapType(src unit.Program, dst *unit.Program, typ unit.TypeInfo, oldToNew []int, eof int, declOffset int, symbolOffset int) (unit.TypeInfo, bool) {
	if typ.NameStart < 0 || typ.NameEnd < typ.NameStart || typ.Decl < 0 {
		return typ, false
	}
	nameStart, nameEnd, ok := mapTextSpanByToken(src, dst, oldToNew, eof, typ.NameStart, typ.NameEnd)
	if !ok {
		return typ, false
	}
	typ.NameStart = nameStart
	typ.NameEnd = nameEnd
	typ.Decl += declOffset
	if typ.Symbol >= 0 {
		typ.Symbol += symbolOffset
	}
	typ.TypeStart, typ.TypeEnd, ok = mapNullableSpan(typ.TypeStart, typ.TypeEnd, oldToNew, eof)
	if !ok {
		return typ, false
	}
	typ.LenStart, typ.LenEnd, ok = mapNullableSpan(typ.LenStart, typ.LenEnd, oldToNew, eof)
	if !ok {
		return typ, false
	}
	typ.KeyStart, typ.KeyEnd, ok = mapNullableSpan(typ.KeyStart, typ.KeyEnd, oldToNew, eof)
	if !ok {
		return typ, false
	}
	typ.ElemStart, typ.ElemEnd, ok = mapNullableSpan(typ.ElemStart, typ.ElemEnd, oldToNew, eof)
	if !ok {
		return typ, false
	}
	return typ, true
}

func mapTypeFields(row unit.TypeFields, oldToNew []int, eof int, typeOffset int, typeLimit int) (unit.TypeFields, bool) {
	if row.TypeIndex < 0 || row.TypeIndex >= typeLimit {
		return row, false
	}
	row.TypeIndex += typeOffset
	var ok bool
	row.Fields, ok = mapFields(row.Fields, oldToNew, eof)
	if !ok {
		return row, false
	}
	return row, true
}

func mapTypeInterface(row unit.TypeIface, oldToNew []int, eof int, typeOffset int, typeLimit int) (unit.TypeIface, bool) {
	if row.TypeIndex < 0 || row.TypeIndex >= typeLimit {
		return row, false
	}
	row.TypeIndex += typeOffset
	for i := 0; i < len(row.Embeds); i++ {
		typeStart, typeEnd, ok := mapNullableSpan(row.Embeds[i].TypeStart, row.Embeds[i].TypeEnd, oldToNew, eof)
		if !ok || typeStart < 0 {
			return row, false
		}
		row.Embeds[i].TypeStart = typeStart
		row.Embeds[i].TypeEnd = typeEnd
	}
	for i := 0; i < len(row.Methods); i++ {
		row.Methods[i].NameTok = mapToken(oldToNew, row.Methods[i].NameTok, eof)
		if row.Methods[i].NameTok < 0 {
			return row, false
		}
		var ok bool
		row.Methods[i].Params, ok = mapFields(row.Methods[i].Params, oldToNew, eof)
		if !ok {
			return row, false
		}
		row.Methods[i].Results, ok = mapFields(row.Methods[i].Results, oldToNew, eof)
		if !ok {
			return row, false
		}
	}
	return row, true
}

func mapTypeFunc(row unit.TypeFuncSig, oldToNew []int, eof int, typeOffset int, typeLimit int) (unit.TypeFuncSig, bool) {
	if row.TypeIndex < 0 || row.TypeIndex >= typeLimit {
		return row, false
	}
	row.TypeIndex += typeOffset
	var ok bool
	row.Params, ok = mapFields(row.Params, oldToNew, eof)
	if !ok {
		return row, false
	}
	row.Results, ok = mapFields(row.Results, oldToNew, eof)
	if !ok {
		return row, false
	}
	return row, true
}

func mapMethod(method unit.MethodInfo, oldToNew []int, eof int, typeOffset int, symbolOffset int, funcOffset int, typeLimit int, symbolLimit int, funcLimit int) (unit.MethodInfo, bool) {
	if method.TypeIndex < 0 || method.TypeIndex >= typeLimit ||
		method.Symbol < 0 || method.Symbol >= symbolLimit ||
		method.FuncIndex < 0 || method.FuncIndex >= funcLimit {
		return method, false
	}
	method.NameTok = mapToken(oldToNew, method.NameTok, eof)
	if method.NameTok < 0 {
		return method, false
	}
	method.TypeIndex += typeOffset
	method.Symbol += symbolOffset
	method.FuncIndex += funcOffset
	return method, true
}

func mapTypeRef(ref unit.TypeRef, oldToNew []int, eof int, declOffset int, funcOffset int, symbolOffsets []int) (unit.TypeRef, bool) {
	ownerIndex, ok := mapOwner(ref.OwnerKind, ref.OwnerIndex, declOffset, funcOffset)
	if !ok {
		return ref, false
	}
	ref.OwnerIndex = ownerIndex
	ref.Token = mapToken(oldToNew, ref.Token, eof)
	ref.BaseTok = mapToken(oldToNew, ref.BaseTok, eof)
	ref.DotTok = mapToken(oldToNew, ref.DotTok, eof)
	if ref.Kind == unit.TypeRefImportSelector {
		ref.Kind = unit.TypeRefPackage
		ref.BaseTok = eof
		ref.DotTok = eof
	}
	symbol, ok := mapPackageSymbol(ref.Package, ref.Symbol, symbolOffsets)
	if !ok {
		return ref, false
	}
	ref.Symbol = symbol
	return ref, true
}

func mapLocal(dst *unit.Program, local unit.LocalDecl, oldToNew []int, eof int, funcOffset int) (unit.LocalDecl, bool) {
	if local.FuncIndex < 0 || local.NameStart < 0 || local.NameEnd < local.NameStart {
		return local, false
	}
	local.FuncIndex += funcOffset
	local.Token = mapToken(oldToNew, local.Token, eof)
	nameStart, nameEnd, ok := mappedTokenTextSpan(dst, local.Token)
	if !ok {
		return local, false
	}
	local.NameStart = nameStart
	local.NameEnd = nameEnd
	local.TypeStart, local.TypeEnd, ok = mapNullableSpan(local.TypeStart, local.TypeEnd, oldToNew, eof)
	if !ok {
		return local, false
	}
	local.ValueStart, local.ValueEnd, ok = mapNullableSpan(local.ValueStart, local.ValueEnd, oldToNew, eof)
	if !ok {
		return local, false
	}
	for i := 0; i < len(local.Values); i++ {
		local.Values[i] = mapExprSpan(local.Values[i], oldToNew, eof)
	}
	return local, true
}

func mapIndex(index unit.IndexExpr, oldToNew []int, eof int, declOffset int, funcOffset int) (unit.IndexExpr, bool) {
	ownerIndex, ok := mapOwner(index.OwnerKind, index.OwnerIndex, declOffset, funcOffset)
	if !ok {
		return index, false
	}
	index.OwnerIndex = ownerIndex
	index.StartTok = mapToken(oldToNew, index.StartTok, eof)
	index.EndTok = mapToken(oldToNew, index.EndTok, eof)
	index.BaseStart = mapToken(oldToNew, index.BaseStart, eof)
	index.BaseEnd = mapToken(oldToNew, index.BaseEnd, eof)
	index.OpenTok = mapToken(oldToNew, index.OpenTok, eof)
	index.CloseTok = mapToken(oldToNew, index.CloseTok, eof)
	index.IndexStart = mapToken(oldToNew, index.IndexStart, eof)
	index.IndexEnd = mapToken(oldToNew, index.IndexEnd, eof)
	return index, true
}

func mapComposite(composite unit.CompositeExpr, oldToNew []int, eof int, declOffset int, funcOffset int) (unit.CompositeExpr, bool) {
	ownerIndex, ok := mapOwner(composite.OwnerKind, composite.OwnerIndex, declOffset, funcOffset)
	if !ok {
		return composite, false
	}
	composite.OwnerIndex = ownerIndex
	composite.StartTok = mapToken(oldToNew, composite.StartTok, eof)
	composite.EndTok = mapToken(oldToNew, composite.EndTok, eof)
	composite.TypeStart = mapToken(oldToNew, composite.TypeStart, eof)
	composite.TypeEnd = mapToken(oldToNew, composite.TypeEnd, eof)
	composite.OpenTok = mapToken(oldToNew, composite.OpenTok, eof)
	composite.CloseTok = mapToken(oldToNew, composite.CloseTok, eof)
	for i := 0; i < len(composite.Elems); i++ {
		composite.Elems[i].StartTok = mapToken(oldToNew, composite.Elems[i].StartTok, eof)
		composite.Elems[i].EndTok = mapToken(oldToNew, composite.Elems[i].EndTok, eof)
	}
	return composite, true
}

func mapAssignment(assign unit.Assignment, oldToNew []int, eof int, funcOffset int) (unit.Assignment, bool) {
	if assign.FuncIndex < 0 {
		return assign, false
	}
	assign.FuncIndex += funcOffset
	assign.StartTok = mapToken(oldToNew, assign.StartTok, eof)
	assign.EndTok = mapToken(oldToNew, assign.EndTok, eof)
	assign.OpTok = mapToken(oldToNew, assign.OpTok, eof)
	assign.LeftStart = mapToken(oldToNew, assign.LeftStart, eof)
	assign.LeftEnd = mapToken(oldToNew, assign.LeftEnd, eof)
	assign.RightStart = mapToken(oldToNew, assign.RightStart, eof)
	assign.RightEnd = mapToken(oldToNew, assign.RightEnd, eof)
	for i := 0; i < len(assign.Targets); i++ {
		assign.Targets[i] = mapExprSpan(assign.Targets[i], oldToNew, eof)
	}
	for i := 0; i < len(assign.Values); i++ {
		assign.Values[i] = mapExprSpan(assign.Values[i], oldToNew, eof)
	}
	return assign, true
}

func mapReturn(ret unit.Return, oldToNew []int, eof int, funcOffset int) (unit.Return, bool) {
	if ret.FuncIndex < 0 {
		return ret, false
	}
	ret.FuncIndex += funcOffset
	ret.StartTok = mapToken(oldToNew, ret.StartTok, eof)
	ret.EndTok = mapToken(oldToNew, ret.EndTok, eof)
	for i := 0; i < len(ret.Values); i++ {
		ret.Values[i] = mapExprSpan(ret.Values[i], oldToNew, eof)
	}
	return ret, true
}

func mapCall(call unit.Call, oldToNew []int, eof int, declOffset int, funcOffset int) (unit.Call, bool) {
	ownerIndex, ok := mapOwner(call.OwnerKind, call.OwnerIndex, declOffset, funcOffset)
	if !ok {
		return call, false
	}
	call.OwnerIndex = ownerIndex
	if call.Kind == unit.CallImportSelector {
		call.Kind = unit.CallPackage
	}
	call.CalleeTok = mapToken(oldToNew, call.CalleeTok, eof)
	call.BaseTok = mapToken(oldToNew, call.BaseTok, eof)
	call.DotTok = mapToken(oldToNew, call.DotTok, eof)
	if call.Kind == unit.CallPackage {
		call.BaseTok = eof
		call.DotTok = eof
	}
	call.ArgsStart = mapToken(oldToNew, call.ArgsStart, eof)
	call.ArgsEnd = mapToken(oldToNew, call.ArgsEnd, eof)
	for i := 0; i < len(call.Args); i++ {
		call.Args[i] = mapExprSpan(call.Args[i], oldToNew, eof)
	}
	return call, true
}

func mapRef(ref unit.NameRef, oldToNew []int, eof int, declOffset int, funcOffset int, symbolOffsets []int) (unit.NameRef, bool) {
	ownerIndex, ok := mapOwner(ref.OwnerKind, ref.OwnerIndex, declOffset, funcOffset)
	if !ok {
		return ref, false
	}
	ref.OwnerIndex = ownerIndex
	ref.Token = mapToken(oldToNew, ref.Token, eof)
	if ref.Kind == unit.RefPackage && ref.Index >= 0 {
		index, ok := mapPackageSymbol(ref.Package, ref.Index, symbolOffsets)
		if !ok {
			return ref, false
		}
		ref.Index = index
	}
	return ref, true
}

func mapSelector(selector unit.Selector, oldToNew []int, eof int, declOffset int, funcOffset int, symbolOffsets []int) (unit.Selector, bool) {
	ownerIndex, ok := mapOwner(selector.OwnerKind, selector.OwnerIndex, declOffset, funcOffset)
	if !ok {
		return selector, false
	}
	selector.OwnerIndex = ownerIndex
	selector.BaseTok = mapToken(oldToNew, selector.BaseTok, eof)
	selector.DotTok = mapToken(oldToNew, selector.DotTok, eof)
	selector.NameTok = mapToken(oldToNew, selector.NameTok, eof)
	if selector.BaseKind == unit.RefPackage && selector.BaseIndex >= 0 {
		index, ok := mapPackageSymbol(selector.BasePackage, selector.BaseIndex, symbolOffsets)
		if !ok {
			return selector, false
		}
		selector.BaseIndex = index
	}
	symbol, ok := mapPackageSymbol(selector.Package, selector.Symbol, symbolOffsets)
	if !ok {
		return selector, false
	}
	selector.Symbol = symbol
	return selector, true
}

func mapExprSpan(span unit.ExprSpan, oldToNew []int, eof int) unit.ExprSpan {
	span.StartTok = mapToken(oldToNew, span.StartTok, eof)
	span.EndTok = mapToken(oldToNew, span.EndTok, eof)
	return span
}

func mapNullableSpan(start int, end int, oldToNew []int, eof int) (int, int, bool) {
	if start < 0 && end < 0 {
		return -1, -1, true
	}
	if start < 0 || end < start {
		return 0, 0, false
	}
	mappedStart := mapToken(oldToNew, start, eof)
	mappedEnd := mapToken(oldToNew, end, eof)
	if mappedStart < 0 || mappedEnd < mappedStart {
		return 0, 0, false
	}
	return mappedStart, mappedEnd, true
}

func mapOwner(kind int, index int, declOffset int, funcOffset int) (int, bool) {
	if kind == unit.OwnerDecl {
		if index < 0 {
			return 0, false
		}
		return declOffset + index, true
	}
	if kind == unit.OwnerFunc {
		if index < 0 {
			return 0, false
		}
		return funcOffset + index, true
	}
	return 0, false
}

func mapPackageSymbol(pkg int, symbol int, symbolOffsets []int) (int, bool) {
	if symbol < 0 {
		return symbol, true
	}
	if pkg < 0 || pkg >= len(symbolOffsets) {
		return symbol, false
	}
	return symbolOffsets[pkg] + symbol, true
}

func packageSymbolOffsets(programs []unit.Program) []int {
	out := make([]int, len(programs))
	offset := 0
	for i := 0; i < len(programs); i++ {
		out[i] = offset
		offset += len(programs[i].Symbols)
	}
	return out
}

func countLinkedTokens(programs []unit.Program) int {
	count := 0
	for i := 0; i < len(programs); i++ {
		tokens := programs[i].Tokens
		skip, _ := linkedTokenSkip(programs[i])
		for j := 0; j < len(tokens); j++ {
			if tokens[j].Kind != unit.TokenEOF && !skip[j] {
				count++
			}
		}
	}
	return count
}

func nextLineOffset(lineOffset int, text []byte, hasNext bool) int {
	lineOffset += countNewlines(text)
	if hasNext && (len(text) == 0 || text[len(text)-1] != '\n') {
		lineOffset++
	}
	return lineOffset
}

func mapToken(oldToNew []int, tok int, eof int) int {
	if tok < 0 || tok >= len(oldToNew) {
		return eof
	}
	return oldToNew[tok]
}

func mapNullableToken(tok int, oldToNew []int, eof int) int {
	if tok < 0 {
		return -1
	}
	return mapToken(oldToNew, tok, eof)
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

func linkFail(result Result, err int, pkg int) Result {
	result.Ok = false
	result.Error = err
	result.ErrorPackage = pkg
	return result
}
