//go:build !rtg

package link

import (
	"strconv"

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
	return linkBuild(result, false)
}

func LinkBuildCore(result build.Result) Result {
	return linkBuild(result, true)
}

func LinkBuildCoreTransient(result build.Result) Result {
	return LinkBuildCore(result)
}

func linkBuild(result build.Result, coreOnly bool) Result {
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
	if coreOnly {
		if linkCoreNeedsInitFold(result) {
			program, ok = LinkUnits(result.Units, result.Root)
			if ok {
				lowerLinkedInitConstDecls(&program)
			}
		} else {
			program, ok = LinkUnitsCore(result.Units, result.Root)
		}
	} else {
		program, ok = LinkUnits(result.Units, result.Root)
	}
	if !ok {
		program, pkg, ok = LinkUnitData(result.Units, result.Root)
	}
	if !ok {
		out.Ok = false
		out.Error = LinkErrUnit
		out.ErrorPackage = pkg
		return out
	}
	var data []byte
	if coreOnly {
		data, ok = marshalCoreProgram(program)
	} else {
		data, ok = unit.Marshal(program)
	}
	if !ok {
		out.Ok = false
		out.Error = LinkErrUnit
		return out
	}
	out.Program = program
	out.Data = data
	return out
}

func linkCoreNeedsInitFold(result build.Result) bool {
	for i := 0; i < len(result.Units); i++ {
		if len(result.Units[i].Program.InitOrder) > 0 && len(result.Units[i].Program.DeclMeta) > 0 {
			return true
		}
	}
	return false
}

func marshalCoreProgram(program unit.Program) ([]byte, bool) {
	program.Imports = nil
	program.Symbols = nil
	program.DeclMeta = nil
	program.InitOrder = nil
	program.Consts = nil
	program.Signatures = nil
	program.Stmts = nil
	program.Types = nil
	program.TypeFields = nil
	program.TypeIfaces = nil
	program.TypeFuncs = nil
	program.Methods = nil
	program.TypeRefs = nil
	program.Locals = nil
	program.Indexes = nil
	program.Composites = nil
	program.Assigns = nil
	program.Returns = nil
	program.Calls = nil
	program.Refs = nil
	program.Selectors = nil
	return unit.Marshal(program)
}

func lowerLinkedInitConstDecls(program *unit.Program) {
	var names []string
	var values []int
	for i := 0; i < len(program.InitOrder); i++ {
		declIndex := program.InitOrder[i]
		if declIndex < 0 || declIndex >= len(program.Decls) || program.Decls[declIndex].Kind != unit.TokenVar {
			continue
		}
		metaIndex := findDeclMeta(program, declIndex)
		if metaIndex < 0 {
			continue
		}
		meta := &program.DeclMeta[metaIndex]
		if meta.ValueStart < 0 || meta.ValueEnd <= meta.ValueStart {
			continue
		}
		value, ok := evalLinkedInitIntExpr(program, meta.ValueStart, meta.ValueEnd, names, values)
		if !ok {
			continue
		}
		name := linkedDeclName(program, declIndex)
		if name != "" {
			names = append(names, name)
			values = append(values, value)
		}
		if !writeLinkedInitIntLiteral(program, declIndex, meta, value) {
			continue
		}
	}
}

func findDeclMeta(program *unit.Program, declIndex int) int {
	for i := 0; i < len(program.DeclMeta); i++ {
		if program.DeclMeta[i].DeclIndex == declIndex {
			return i
		}
	}
	return -1
}

func linkedDeclName(program *unit.Program, declIndex int) string {
	decl := program.Decls[declIndex]
	if decl.NameStart < 0 || decl.NameEnd < decl.NameStart || decl.NameEnd > len(program.Text) {
		return ""
	}
	return string(program.Text[decl.NameStart:decl.NameEnd])
}

func evalLinkedInitIntExpr(program *unit.Program, start int, end int, names []string, values []int) (int, bool) {
	if start < 0 || end <= start || end > len(program.Tokens) {
		return 0, false
	}
	if start+1 == end {
		tok := program.Tokens[start]
		text := tokenText(*program, start)
		if tok.Kind == unit.TokenNumber {
			value, err := strconv.Atoi(text)
			return value, err == nil
		}
		if tok.Kind == unit.TokenIdent {
			return lookupLinkedInitValue(names, values, text)
		}
		return 0, false
	}
	op := findLinkedInitBinaryOp(program, start, end, "+-")
	if op < 0 {
		op = findLinkedInitBinaryOp(program, start, end, "*/")
	}
	if op < 0 {
		return 0, false
	}
	left, ok := evalLinkedInitIntExpr(program, start, op, names, values)
	if !ok {
		return 0, false
	}
	right, ok := evalLinkedInitIntExpr(program, op+1, end, names, values)
	if !ok {
		return 0, false
	}
	opText := tokenText(*program, op)
	if opText == "+" {
		return left + right, true
	}
	if opText == "-" {
		return left - right, true
	}
	if opText == "*" {
		return left * right, true
	}
	if opText == "/" && right != 0 {
		return left / right, true
	}
	return 0, false
}

func findLinkedInitBinaryOp(program *unit.Program, start int, end int, ops string) int {
	depth := 0
	for i := end - 1; i >= start; i-- {
		text := tokenText(*program, i)
		if text == ")" || text == "]" || text == "}" {
			depth++
			continue
		}
		if text == "(" || text == "[" || text == "{" {
			depth--
			continue
		}
		if depth == 0 && program.Tokens[i].Kind == unit.TokenOp && stringHasByte(ops, text) {
			return i
		}
	}
	return -1
}

func stringHasByte(chars string, text string) bool {
	if len(text) != 1 {
		return false
	}
	for i := 0; i < len(chars); i++ {
		if chars[i] == text[0] {
			return true
		}
	}
	return false
}

func lookupLinkedInitValue(names []string, values []int, name string) (int, bool) {
	for i := len(names) - 1; i >= 0; i-- {
		if names[i] == name {
			return values[i], true
		}
	}
	return 0, false
}

func writeLinkedInitIntLiteral(program *unit.Program, declIndex int, meta *unit.DeclMeta, value int) bool {
	lit := strconv.Itoa(value)
	tokIndex := meta.ValueStart
	if tokIndex < 0 || tokIndex >= len(program.Tokens) {
		return false
	}
	tok := &program.Tokens[tokIndex]
	if tok.Start < 0 || tok.Start+tok.Size > len(program.Text) || len(lit) > tok.Size {
		return false
	}
	for i := 0; i < tok.Size; i++ {
		program.Text[tok.Start+i] = ' '
	}
	for i := 0; i < len(lit); i++ {
		program.Text[tok.Start+i] = lit[i]
	}
	tok.Kind = unit.TokenNumber
	tok.Size = len(lit)
	program.Decls[declIndex].EndTok = tokIndex + 1
	meta.ValueEnd = tokIndex + 1
	meta.Values = []unit.ExprSpan{{StartTok: tokIndex, EndTok: tokIndex + 1}}
	return true
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

func LinkPrograms(programs []unit.Program, root int, rootName string) (unit.Program, bool) {
	return linkPrograms(programs, root, rootName, false)
}

func LinkProgramsCore(programs []unit.Program, root int, rootName string) (unit.Program, bool) {
	return linkPrograms(programs, root, rootName, true)
}

func linkPrograms(programs []unit.Program, root int, rootName string, coreOnly bool) (unit.Program, bool) {
	var empty unit.Program
	if root < 0 || root >= len(programs) || rootName == "" {
		return empty, false
	}
	programs, ok := preparePrograms(programs, root)
	if !ok {
		return empty, false
	}
	program := unit.Program{Package: rootName, ImportPath: programs[root].ImportPath}
	if coreOnly {
		reserveCoreLinkedProgram(&program, programs)
	} else {
		reserveLinkedProgram(&program, programs)
	}
	finalEOF := countLinkedTokens(programs, coreOnly)
	symbolOffsets := packageSymbolOffsets(programs)
	aliases := packageSymbolAliases(programs, root, symbolOffsets)
	lineOffset := 0
	for i := 0; i < len(programs); i++ {
		ok := appendProgram(&program, programs[i], finalEOF, lineOffset, symbolOffsets, aliases, i+1 < len(programs), coreOnly)
		if !ok {
			return empty, false
		}
		lineOffset = nextLineOffset(lineOffset, programs[i].Text, i+1 < len(programs))
	}
	program.Tokens = append(program.Tokens, unit.Token{
		Kind:  unit.TokenEOF,
		Start: len(program.Text),
		Size:  0,
		Line:  countNewlines(program.Text) + 1,
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

func reserveLinkedProgram(program *unit.Program, programs []unit.Program) {
	textCap := 0
	tokenCap := 1
	importCap := 0
	symbolCap := 0
	declCap := 0
	declMetaCap := 0
	initOrderCap := 0
	constCap := 0
	funcCap := 0
	signatureCap := 0
	stmtCap := 0
	typeCap := 0
	typeFieldCap := 0
	typeIfaceCap := 0
	typeFuncCap := 0
	methodCap := 0
	typeRefCap := 0
	localCap := 0
	indexCap := 0
	compositeCap := 0
	assignCap := 0
	returnCap := 0
	callCap := 0
	refCap := 0
	selectorCap := 0
	for i := 0; i < len(programs); i++ {
		p := programs[i]
		textCap += len(p.Text) + 1
		tokenCap += len(p.Tokens)
		importCap += len(p.Imports)
		symbolCap += len(p.Symbols)
		declCap += len(p.Decls)
		declMetaCap += len(p.DeclMeta)
		initOrderCap += len(p.InitOrder)
		constCap += len(p.Consts)
		funcCap += len(p.Funcs)
		signatureCap += len(p.Signatures)
		stmtCap += len(p.Stmts)
		typeCap += len(p.Types)
		typeFieldCap += len(p.TypeFields)
		typeIfaceCap += len(p.TypeIfaces)
		typeFuncCap += len(p.TypeFuncs)
		methodCap += len(p.Methods)
		typeRefCap += len(p.TypeRefs)
		localCap += len(p.Locals)
		indexCap += len(p.Indexes)
		compositeCap += len(p.Composites)
		assignCap += len(p.Assigns)
		returnCap += len(p.Returns)
		callCap += len(p.Calls)
		refCap += len(p.Refs)
		selectorCap += len(p.Selectors)
	}
	program.Text = make([]byte, 0, textCap)
	program.Tokens = make([]unit.Token, 0, tokenCap)
	program.Imports = make([]unit.Import, 0, importCap)
	program.Symbols = make([]unit.Symbol, 0, symbolCap)
	program.Decls = make([]unit.Decl, 0, declCap)
	program.DeclMeta = make([]unit.DeclMeta, 0, declMetaCap)
	program.InitOrder = make([]int, 0, initOrderCap)
	program.Consts = make([]unit.ConstValue, 0, constCap)
	program.Funcs = make([]unit.Func, 0, funcCap)
	program.Signatures = make([]unit.FuncSignature, 0, signatureCap)
	program.Stmts = make([]unit.Statement, 0, stmtCap)
	program.Types = make([]unit.TypeInfo, 0, typeCap)
	program.TypeFields = make([]unit.TypeFields, 0, typeFieldCap)
	program.TypeIfaces = make([]unit.TypeIface, 0, typeIfaceCap)
	program.TypeFuncs = make([]unit.TypeFuncSig, 0, typeFuncCap)
	program.Methods = make([]unit.MethodInfo, 0, methodCap)
	program.TypeRefs = make([]unit.TypeRef, 0, typeRefCap)
	program.Locals = make([]unit.LocalDecl, 0, localCap)
	program.Indexes = make([]unit.IndexExpr, 0, indexCap)
	program.Composites = make([]unit.CompositeExpr, 0, compositeCap)
	program.Assigns = make([]unit.Assignment, 0, assignCap)
	program.Returns = make([]unit.Return, 0, returnCap)
	program.Calls = make([]unit.Call, 0, callCap)
	program.Refs = make([]unit.NameRef, 0, refCap)
	program.Selectors = make([]unit.Selector, 0, selectorCap)
}

func preparePrograms(programs []unit.Program, root int) ([]unit.Program, bool) {
	out := make([]unit.Program, len(programs))
	copy(out, programs)
	processPackage, processSymbol := findProgramSymbol(out, "rtg_runtime_SetProcess")
	rootProgram, ok := addRootEntrypoint(out[root], root, processPackage, processSymbol)
	if !ok {
		return nil, false
	}
	out[root] = rootProgram
	return out, true
}

func addRootEntrypoint(src unit.Program, packageIndex int, processPackage int, processSymbol int) (unit.Program, bool) {
	if src.Package != "main" || findFuncByName(src, "appMain") >= 0 || findFuncByName(src, "main") < 0 {
		return src, true
	}
	if processPackage >= 0 && processSymbol >= 0 {
		return addRootProcessEntrypoint(src, packageIndex, processPackage, processSymbol)
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

func findProgramSymbol(programs []unit.Program, name string) (int, int) {
	for i := 0; i < len(programs); i++ {
		for j := 0; j < len(programs[i].Symbols); j++ {
			if programs[i].Symbols[j].Name == name && programs[i].Symbols[j].Kind == unit.SymbolFunc {
				return i, j
			}
		}
	}
	return -1, -1
}

func addRootProcessEntrypoint(src unit.Program, packageIndex int, processPackage int, processSymbol int) (unit.Program, bool) {
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
	src.Text = appendStringBytes(src.Text, "func appMain(args []string, env []string) int { rtg_runtime_SetProcess(args, env); main(); return 0 }\n")
	base := len(src.Tokens)
	items := []struct {
		kind  int
		start int
		size  int
	}{
		{unit.TokenFunc, 0, 4}, {unit.TokenIdent, 5, 7}, {unit.TokenOp, 12, 1},
		{unit.TokenIdent, 13, 4}, {unit.TokenOp, 18, 1}, {unit.TokenOp, 19, 1}, {unit.TokenIdent, 20, 6}, {unit.TokenOp, 26, 1},
		{unit.TokenIdent, 28, 3}, {unit.TokenOp, 32, 1}, {unit.TokenOp, 33, 1}, {unit.TokenIdent, 34, 6}, {unit.TokenOp, 40, 1},
		{unit.TokenIdent, 42, 3}, {unit.TokenOp, 46, 1},
		{unit.TokenIdent, 48, 22}, {unit.TokenOp, 70, 1}, {unit.TokenIdent, 71, 4}, {unit.TokenOp, 75, 1}, {unit.TokenIdent, 77, 3}, {unit.TokenOp, 80, 1}, {unit.TokenOp, 81, 1},
		{unit.TokenIdent, 83, 4}, {unit.TokenOp, 87, 1}, {unit.TokenOp, 88, 1}, {unit.TokenOp, 89, 1},
		{unit.TokenReturn, 91, 6}, {unit.TokenNumber, 98, 1}, {unit.TokenOp, 100, 1},
	}
	for i := 0; i < len(items); i++ {
		item := items[i]
		src.Tokens = append(src.Tokens, unit.Token{Kind: item.kind, Start: start + item.start, Size: item.size, Line: line})
	}
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
		BodyStart:     base + 14,
		BodyEnd:       base + 28,
		EndTok:        eof,
	})
	src.Signatures = append(src.Signatures, unit.FuncSignature{
		FuncIndex: funcIndex,
		Params: []unit.Field{
			{NameTok: base + 3, TypeStart: base + 4, TypeEnd: base + 7},
			{NameTok: base + 8, TypeStart: base + 9, TypeEnd: base + 12},
		},
		Results: []unit.Field{{NameTok: -1, TypeStart: base + 13, TypeEnd: base + 14}},
	})
	src.Stmts = append(src.Stmts,
		unit.Statement{FuncIndex: funcIndex, Kind: unit.StmtBlock, StartTok: base + 14, EndTok: eof, ExprStart: -1, ExprEnd: -1, BodyStart: base + 14, BodyEnd: eof, ElseStart: -1, ElseEnd: -1},
		unit.Statement{FuncIndex: funcIndex, Kind: unit.StmtExpr, StartTok: base + 15, EndTok: base + 22, ExprStart: base + 15, ExprEnd: base + 21, BodyStart: -1, BodyEnd: -1, ElseStart: -1, ElseEnd: -1},
		unit.Statement{FuncIndex: funcIndex, Kind: unit.StmtExpr, StartTok: base + 22, EndTok: base + 26, ExprStart: base + 22, ExprEnd: base + 25, BodyStart: -1, BodyEnd: -1, ElseStart: -1, ElseEnd: -1},
		unit.Statement{FuncIndex: funcIndex, Kind: unit.StmtReturn, StartTok: base + 26, EndTok: base + 28, ExprStart: base + 27, ExprEnd: base + 28, BodyStart: -1, BodyEnd: -1, ElseStart: -1, ElseEnd: -1},
	)
	src.Returns = append(src.Returns, unit.Return{
		FuncIndex: funcIndex,
		StartTok:  base + 26,
		EndTok:    base + 28,
		Values:    []unit.ExprSpan{{StartTok: base + 27, EndTok: base + 28}},
	})
	src.Calls = append(src.Calls,
		unit.Call{
			OwnerKind:  unit.OwnerFunc,
			OwnerIndex: funcIndex,
			Kind:       unit.CallPackage,
			CalleeTok:  base + 15,
			BaseTok:    eof,
			DotTok:     eof,
			ArgsStart:  base + 17,
			ArgsEnd:    base + 20,
			Args: []unit.ExprSpan{
				{StartTok: base + 17, EndTok: base + 18},
				{StartTok: base + 19, EndTok: base + 20},
			},
		},
		unit.Call{
			OwnerKind:  unit.OwnerFunc,
			OwnerIndex: funcIndex,
			Kind:       unit.CallPackage,
			CalleeTok:  base + 22,
			BaseTok:    eof,
			DotTok:     eof,
			ArgsStart:  base + 24,
			ArgsEnd:    base + 24,
		},
	)
	mainSymbol := findFuncSymbol(src, "main")
	src.Refs = append(src.Refs,
		unit.NameRef{OwnerKind: unit.OwnerFunc, OwnerIndex: funcIndex, Kind: unit.RefPackage, Token: base + 15, Index: processSymbol, Package: processPackage},
		unit.NameRef{OwnerKind: unit.OwnerFunc, OwnerIndex: funcIndex, Kind: unit.RefScope, Token: base + 17, Index: 0, Package: -1},
		unit.NameRef{OwnerKind: unit.OwnerFunc, OwnerIndex: funcIndex, Kind: unit.RefScope, Token: base + 19, Index: 1, Package: -1},
		unit.NameRef{OwnerKind: unit.OwnerFunc, OwnerIndex: funcIndex, Kind: unit.RefPackage, Token: base + 22, Index: mainSymbol, Package: packageIndex},
	)
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

func appendProgram(dst *unit.Program, src unit.Program, finalEOF int, lineOffset int, symbolOffsets []int, aliases []string, hasNext bool, coreOnly bool) bool {
	if src.Package == "" || len(src.Text) == 0 || len(src.Tokens) == 0 {
		return false
	}
	symbolOffset := len(dst.Symbols)
	declOffset := len(dst.Decls)
	funcOffset := len(dst.Funcs)
	typeOffset := len(dst.Types)
	oldToNew := make([]int, len(src.Tokens))
	skip, redirect := linkedTokenSkip(src, coreOnly)
	replacements := linkedTokenReplacements(src, aliases, symbolOffsets, coreOnly)
	prevEnd := 0
	line := countNewlines(dst.Text) + 1
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
			if redirect[i] >= 0 && tok.Start > prevEnd {
				part := src.Text[prevEnd:tok.Start]
				dst.Text = appendBytes(dst.Text, part)
				line += countNewlines(part)
			}
			if tokEnd > prevEnd {
				prevEnd = tokEnd
			}
			continue
		}
		if tok.Start > prevEnd {
			part := src.Text[prevEnd:tok.Start]
			dst.Text = appendBytes(dst.Text, part)
			line += countNewlines(part)
		}
		oldToNew[i] = len(dst.Tokens)
		tok.Start = len(dst.Text)
		tok.Line = line
		if replacements[i] != "" {
			dst.Text = appendStringBytes(dst.Text, replacements[i])
			tok.Kind = linkedReplacementTokenKind(tok.Kind, replacements[i])
			tok.Size = len(replacements[i])
			line += countStringNewlines(replacements[i])
		} else if linkedTokenIsEllipsis(tok, src.Text, tokStart, tokEnd) {
			dst.Text = appendStringBytes(dst.Text, "...")
			for j := 0; j < 3; j++ {
				dot := tok
				dot.Start = tok.Start + j
				dot.Size = 1
				dst.Tokens = append(dst.Tokens, dot)
			}
			prevEnd = tokEnd
			continue
		} else {
			part := src.Text[tokStart:tokEnd]
			dst.Text = appendBytes(dst.Text, part)
			line += countNewlines(part)
		}
		dst.Tokens = append(dst.Tokens, tok)
		prevEnd = tokEnd
	}
	if prevEnd < len(src.Text) {
		part := src.Text[prevEnd:]
		dst.Text = appendBytes(dst.Text, part)
		line += countNewlines(part)
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
	if !coreOnly {
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
	}
	for i := 0; i < len(src.Funcs); i++ {
		fn := src.Funcs[i]
		if tokenSkipped(skip, fn.NameTok) {
			continue
		}
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
		normalizeLinkedReceiver(&fn, finalEOF)
		fn.BodyStart = mapToken(oldToNew, fn.BodyStart, finalEOF)
		fn.BodyEnd = mapToken(oldToNew, fn.BodyEnd, finalEOF)
		fn.EndTok = mapFuncEndToken(oldToNew, fn.EndTok, fn.BodyEnd, finalEOF)
		dst.Funcs = append(dst.Funcs, fn)
	}
	if coreOnly {
		if hasNext && (len(src.Text) == 0 || src.Text[len(src.Text)-1] != '\n') {
			dst.Text = append(dst.Text, '\n')
		}
		return true
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
		if tokenSkipped(skip, src.TypeRefs[i].Token) {
			continue
		}
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
		if tokenSkipped(skip, src.Indexes[i].StartTok) {
			continue
		}
		index, ok := mapIndex(src.Indexes[i], oldToNew, finalEOF, declOffset, funcOffset)
		if !ok {
			return false
		}
		dst.Indexes = append(dst.Indexes, index)
	}
	for i := 0; i < len(src.Composites); i++ {
		if tokenSkipped(skip, src.Composites[i].StartTok) || tokenSkipped(skip, src.Composites[i].TypeStart) {
			continue
		}
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
		if tokenSkipped(skip, src.Calls[i].CalleeTok) || isSimpleClosureCall(src, src.Calls[i]) {
			continue
		}
		call, ok := mapCall(src.Calls[i], oldToNew, finalEOF, declOffset, funcOffset)
		if !ok {
			return false
		}
		dst.Calls = append(dst.Calls, call)
	}
	for i := 0; i < len(src.Refs); i++ {
		if src.Refs[i].Kind == unit.RefImport || tokenSkipped(skip, src.Refs[i].Token) {
			continue
		}
		ref, ok := mapRef(src.Refs[i], oldToNew, finalEOF, declOffset, funcOffset, symbolOffsets)
		if !ok {
			return false
		}
		dst.Refs = append(dst.Refs, ref)
	}
	for i := 0; i < len(src.Selectors); i++ {
		if src.Selectors[i].BaseKind == unit.RefImport || tokenSkipped(skip, src.Selectors[i].NameTok) {
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
		line++
	}
	_ = lineOffset
	_ = line
	return true
}

func linkedTokenSkip(program unit.Program, coreOnly bool) ([]bool, []int) {
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
			markUnsafePointerCallTokens(program, skip, call)
		}
	}
	if programImportsUnsafe(program) {
		markUnsafePointerConversionTokens(program, skip)
	}
	markEndianSelectorTokens(program, skip, redirect)
	markSimpleClosureSkipTokens(program, skip)
	if coreOnly {
		markSimpleDeferPanicRecoverSkipTokens(program, skip)
		markSimpleFunctionValueSkipTokens(program, skip)
		markSimpleSearchClosureSkipTokens(program, skip)
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

func tokenSkipped(skip []bool, tok int) bool {
	return tok >= 0 && tok < len(skip) && skip[tok]
}

func markUnsafePointerCallTokens(program unit.Program, skip []bool, call unit.Call) {
	if !tokenTextEquals(program, call.BaseTok, "unsafe") || !tokenTextEquals(program, call.CalleeTok, "Pointer") {
		return
	}
	open := call.CalleeTok + 1
	close := findMatchingParen(program, open)
	if close < 0 {
		return
	}
	markSkipToken(skip, call.CalleeTok)
	markSkipToken(skip, open)
	markSkipToken(skip, close)
}

func markUnsafePointerConversionTokens(program unit.Program, skip []bool) {
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
			markSkipToken(skip, j)
		}
		markSkipToken(skip, typeEnd+1)
		markSkipToken(skip, valueEnd)
		i = valueEnd
	}
}

func markEndianSelectorTokens(program unit.Program, skip []bool, redirect []int) {
	for i := 0; i+2 < len(program.Tokens); i++ {
		if (tokenTextEquals(program, i, "LittleEndian") || tokenTextEquals(program, i, "BigEndian")) && tokenTextEquals(program, i+1, ".") {
			markRedirectToken(skip, redirect, i, i+2)
			markRedirectToken(skip, redirect, i+1, i+2)
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

func markSimpleClosureSkipTokens(program unit.Program, skip []bool) {
	factories := findSimpleClosureFactories(program)
	for i := 0; i < len(factories); i++ {
		markSkipRange(skip, factories[i].resultSkipStart, factories[i].resultSkipEnd)
		markSkipRange(skip, factories[i].literalSkipStart, factories[i].literalSkipEnd)
		markSkipRange(skip, factories[i].suffixSkipStart, factories[i].suffixSkipEnd)
	}
	locals := findSimpleClosureLocals(program, factories)
	for i := 0; i < len(program.Calls); i++ {
		call := program.Calls[i]
		if !nameInList(locals, tokenText(program, call.CalleeTok)) {
			continue
		}
		close := findMatchingParen(program, call.CalleeTok+1)
		if close >= 0 {
			markSkipToken(skip, close)
		}
	}
}

func markSimpleClosureReplacementTokens(program unit.Program, replacements []string) {
	factories := findSimpleClosureFactories(program)
	locals := findSimpleClosureLocals(program, factories)
	for i := 0; i < len(program.Calls); i++ {
		call := program.Calls[i]
		if !nameInList(locals, tokenText(program, call.CalleeTok)) {
			continue
		}
		open := call.CalleeTok + 1
		if open >= 0 && open < len(replacements) && tokenTextEquals(program, open, "(") {
			replacements[open] = "+"
		}
	}
}

func isSimpleClosureCall(program unit.Program, call unit.Call) bool {
	factories := findSimpleClosureFactories(program)
	locals := findSimpleClosureLocals(program, factories)
	return nameInList(locals, tokenText(program, call.CalleeTok))
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

func markSimpleDeferPanicRecoverSkipTokens(program unit.Program, skip []bool) {
	info, ok := findSimpleDeferPanicRecoverInfo(program)
	if !ok {
		return
	}
	markSkipToken(skip, info.resultOpen)
	markSkipToken(skip, info.resultName)
	markSkipToken(skip, info.resultClose)
	markSkipRange(skip, info.deferStart, info.deferEnd)
	markSkipToken(skip, info.panicOpen)
	markSkipToken(skip, info.panicClose)
}

func markSimpleDeferPanicRecoverReplacementTokens(program unit.Program, replacements []string) {
	info, ok := findSimpleDeferPanicRecoverInfo(program)
	if !ok {
		return
	}
	if info.panicCallee >= 0 && info.panicCallee < len(replacements) {
		replacements[info.panicCallee] = "return "
	}
	if info.panicArg >= 0 && info.panicArg < len(replacements) {
		replacements[info.panicArg] = "true"
	}
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

func markSimpleFunctionValueSkipTokens(program unit.Program, skip []bool) {
	info, ok := findSimpleFunctionValueInfo(program)
	if !ok {
		return
	}
	markSkipRange(skip, info.helperParamSkipFrom, info.helperParamSkipTo)
	markSkipRange(skip, info.initStart, info.initEnd)
	if info.branchStart >= 0 {
		markSkipRange(skip, info.branchStart, info.branchEnd)
	}
	markSimpleFunctionValueCallArgs(program, skip, info)
}

func markSimpleFunctionValueCallArgs(program unit.Program, skip []bool, info simpleFunctionValueInfo) {
	for i := 0; i+4 < len(program.Tokens); i++ {
		if !tokenTextEquals(program, i, info.helperName) || !tokenTextEquals(program, i+1, "(") || !tokenTextEquals(program, i+2, info.callLocal) || !tokenTextEquals(program, i+3, ",") {
			continue
		}
		markSkipToken(skip, i+2)
		markSkipToken(skip, i+3)
	}
}

func markSimpleFunctionValueReplacementTokens(program unit.Program, replacements []string) {
	info, ok := findSimpleFunctionValueInfo(program)
	if !ok {
		return
	}
	if info.helperReturnCallee >= 0 && info.helperReturnCallee < len(replacements) {
		replacements[info.helperReturnCallee] = info.selected
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

func markSimpleSearchClosureSkipTokens(program unit.Program, skip []bool) {
	infos := findSimpleSearchClosureInfos(program)
	for i := 0; i < len(infos); i++ {
		markSkipRange(skip, infos[i].skipStart, infos[i].skipEnd)
	}
}

func markSimpleSearchClosureReplacementTokens(program unit.Program, replacements []string) {
	infos := findSimpleSearchClosureInfos(program)
	for i := 0; i < len(infos); i++ {
		if infos[i].funcTok >= 0 && infos[i].funcTok < len(replacements) {
			replacements[infos[i].funcTok] = infos[i].threshold
		}
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

func markSkipRange(skip []bool, start int, end int) {
	for i := start; i <= end; i++ {
		markSkipToken(skip, i)
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

func markSkipToken(skip []bool, tok int) {
	if tok < 0 || tok >= len(skip) {
		return
	}
	skip[tok] = true
}

func programImportsUnsafe(program unit.Program) bool {
	for i := 0; i < len(program.Imports); i++ {
		imp := program.Imports[i]
		if imp.ImportPath == "unsafe" || tokenTextEquals(program, imp.PathTok, "\"unsafe\"") || tokenTextEquals(program, imp.PathTok, "`unsafe`") {
			return true
		}
	}
	return false
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

func tokenTextEquals(program unit.Program, tok int, want string) bool {
	return tokenText(program, tok) == want
}

func parseTokenInt(program unit.Program, tok int) (int, bool) {
	text := tokenText(program, tok)
	if text == "" {
		return 0, false
	}
	value, err := strconv.Atoi(text)
	return value, err == nil
}

func linkedTokenReplacements(program unit.Program, aliases []string, symbolOffsets []int, coreOnly bool) []string {
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
	markSimpleClosureReplacementTokens(program, out)
	if coreOnly {
		markSimpleDeferPanicRecoverReplacementTokens(program, out)
		markSimpleFunctionValueReplacementTokens(program, out)
		markSimpleSearchClosureReplacementTokens(program, out)
	}
	return out
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
		var ok bool
		fields[i].TypeStart, fields[i].TypeEnd, ok = mapTokenSpan(oldToNew, fields[i].TypeStart, fields[i].TypeEnd, eof)
		if fields[i].NameTok < -1 || !ok || fields[i].TypeStart < 0 || fields[i].TypeEnd < fields[i].TypeStart {
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
	var ok bool
	assign.StartTok, assign.EndTok, ok = mapTokenSpan(oldToNew, assign.StartTok, assign.EndTok, eof)
	if !ok {
		return assign, false
	}
	assign.OpTok = mapToken(oldToNew, assign.OpTok, eof)
	assign.LeftStart, assign.LeftEnd, ok = mapTokenSpan(oldToNew, assign.LeftStart, assign.LeftEnd, eof)
	if !ok {
		return assign, false
	}
	assign.RightStart, assign.RightEnd, ok = mapTokenSpan(oldToNew, assign.RightStart, assign.RightEnd, eof)
	if !ok {
		return assign, false
	}
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
	call.ArgsStart, call.ArgsEnd, ok = mapTokenSpan(oldToNew, call.ArgsStart, call.ArgsEnd, eof)
	if !ok {
		return call, false
	}
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
	start, end, ok := mapTokenSpan(oldToNew, span.StartTok, span.EndTok, eof)
	if !ok {
		start = eof
		end = eof
	}
	span.StartTok = start
	span.EndTok = end
	return span
}

func mapNullableSpan(start int, end int, oldToNew []int, eof int) (int, int, bool) {
	if start < 0 && end < 0 {
		return -1, -1, true
	}
	if start < 0 || end < start {
		return 0, 0, false
	}
	return mapTokenSpan(oldToNew, start, end, eof)
}

func mapTokenSpan(oldToNew []int, start int, end int, eof int) (int, int, bool) {
	if start < 0 || end < start {
		return 0, 0, false
	}
	if start == end {
		mapped := mapToken(oldToNew, start, eof)
		return mapped, mapped, mapped >= 0
	}
	mappedStart := mapToken(oldToNew, start, eof)
	mappedEnd := mapToken(oldToNew, end, eof)
	if mappedStart >= 0 && mappedEnd >= mappedStart {
		return mappedStart, mappedEnd, true
	}
	first := -1
	last := -1
	limit := end
	if limit > len(oldToNew) {
		limit = len(oldToNew)
	}
	for i := start; i < limit; i++ {
		mapped := mapToken(oldToNew, i, eof)
		if mapped < 0 || mapped == eof {
			continue
		}
		if first < 0 {
			first = mapped
		}
		last = mapped
	}
	if first < 0 {
		return eof, eof, true
	}
	return first, last + 1, true
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

func countLinkedTokens(programs []unit.Program, coreOnly bool) int {
	count := 0
	for i := 0; i < len(programs); i++ {
		tokens := programs[i].Tokens
		skip, _ := linkedTokenSkip(programs[i], coreOnly)
		for j := 0; j < len(tokens); j++ {
			if tokens[j].Kind != unit.TokenEOF && !skip[j] {
				count += linkedTokenOutputCount(programs[i], tokens[j])
			}
		}
	}
	return count
}

func linkedTokenOutputCount(program unit.Program, tok unit.Token) int {
	if linkedTokenIsEllipsis(tok, program.Text, tok.Start, tok.Start+tok.Size) {
		return 3
	}
	return 1
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

func mapFuncEndToken(oldToNew []int, tok int, bodyEnd int, eof int) int {
	mapped := mapToken(oldToNew, tok, eof)
	if mapped == eof && bodyEnd >= 0 && bodyEnd+1 <= eof {
		return bodyEnd + 1
	}
	return mapped
}

func normalizeLinkedReceiver(fn *unit.Func, eof int) {
	_ = eof
	if fn.ReceiverStart == fn.ReceiverEnd {
		fn.ReceiverStart = 0
		fn.ReceiverEnd = 0
	}
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

func countStringNewlines(text string) int {
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
