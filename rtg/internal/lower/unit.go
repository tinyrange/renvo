package lower

import (
	"j5.nz/rtg/rtg/internal/check"
	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtg/internal/syntax"
	"j5.nz/rtg/rtg/internal/unit"
)

const (
	EmitOK = iota
	EmitErrGraph
	EmitErrPackage
	EmitErrToken
	EmitErrUnit
	EmitErrCheck
)

type Result struct {
	Program    unit.Program
	Ok         bool
	Error      int
	ErrorFile  int
	ErrorToken int
}

func EmitRoot(graph load.Graph) Result {
	if !graph.Ok {
		return emitFail(Result{}, EmitErrGraph, -1, -1)
	}
	for i := 0; i < len(graph.Packages); i++ {
		if graph.Packages[i].Ref.ImportPath == graph.Root {
			return EmitPackage(graph.Packages[i])
		}
	}
	return emitFail(Result{}, EmitErrGraph, -1, -1)
}

func EmitPackage(pkg load.Package) Result {
	result := Result{Ok: true, Error: EmitOK, ErrorFile: -1, ErrorToken: -1}
	if !pkg.Ok || pkg.Name == "" || len(pkg.Files) == 0 {
		return emitFail(result, EmitErrPackage, -1, -1)
	}
	var builder unitBuilder
	builder.program.Package = pkg.Name
	builder.program.ImportPath = pkg.Ref.ImportPath
	builder.finalEOF = countPackageTokens(pkg)
	for i := 0; i < len(pkg.Files); i++ {
		if !pkg.Files[i].File.Ok {
			return emitFail(result, EmitErrPackage, i, pkg.Files[i].File.ErrorTok)
		}
		if !builder.addFile(pkg.Files[i].File, i, i+1 < len(pkg.Files)) {
			return emitFail(result, builder.err, builder.errFile, builder.errToken)
		}
	}
	line := builder.lineOffset + 1
	builder.program.Tokens = append(builder.program.Tokens, unit.Token{
		Kind:  unit.TokenEOF,
		Start: len(builder.program.Text),
		Size:  0,
		Line:  line,
	})
	if _, ok := unit.Marshal(builder.program); !ok {
		return emitFail(result, EmitErrUnit, -1, -1)
	}
	result.Program = builder.program
	return result
}

func EmitRootChecked(graph load.Graph, prog check.Program) Result {
	if !graph.Ok || !prog.Ok || len(graph.Packages) != len(prog.Packages) {
		return emitFail(Result{}, EmitErrGraph, -1, -1)
	}
	for i := 0; i < len(graph.Packages); i++ {
		if graph.Packages[i].Ref.ImportPath == graph.Root {
			return EmitCheckedPackage(graph.Packages[i], prog.Packages[i])
		}
	}
	return emitFail(Result{}, EmitErrGraph, -1, -1)
}

func EmitCheckedPackage(pkg load.Package, info check.PackageInfo) Result {
	result := Result{Ok: true, Error: EmitOK, ErrorFile: -1, ErrorToken: -1}
	if !pkg.Ok || pkg.Name == "" || len(pkg.Files) == 0 {
		return emitFail(result, EmitErrPackage, -1, -1)
	}
	if info.Name != pkg.Name {
		return emitFail(result, EmitErrCheck, -1, -1)
	}
	var builder unitBuilder
	builder.program.Package = pkg.Name
	builder.program.ImportPath = pkg.Ref.ImportPath
	builder.finalEOF = countPackageTokens(pkg)
	files := make([]fileTokens, len(pkg.Files))
	for i := 0; i < len(pkg.Files); i++ {
		if !pkg.Files[i].File.Ok {
			return emitFail(result, EmitErrPackage, i, pkg.Files[i].File.ErrorTok)
		}
		oldToNew, ok := builder.addFileTokens(pkg.Files[i].File, i, i+1 < len(pkg.Files))
		if !ok {
			return emitFail(result, builder.err, builder.errFile, builder.errToken)
		}
		files[i] = fileTokens{file: pkg.Files[i].File, oldToNew: oldToNew}
	}
	if !builder.addCheckedImports(info, files) {
		return emitFail(result, builder.err, builder.errFile, builder.errToken)
	}
	if !builder.addCheckedDecls(info, files) {
		return emitFail(result, builder.err, builder.errFile, builder.errToken)
	}
	if !builder.addCheckedInitOrder(info) {
		return emitFail(result, builder.err, builder.errFile, builder.errToken)
	}
	if !builder.addCheckedTypes(info, files) {
		return emitFail(result, builder.err, builder.errFile, builder.errToken)
	}
	if !builder.addCheckedTypeRefs(info, files) {
		return emitFail(result, builder.err, builder.errFile, builder.errToken)
	}
	if !builder.addCheckedFuncs(info, files) {
		return emitFail(result, builder.err, builder.errFile, builder.errToken)
	}
	if !builder.addCheckedSymbols(info, files) {
		return emitFail(result, builder.err, builder.errFile, builder.errToken)
	}
	if !builder.finishUnit() {
		return emitFail(result, builder.err, builder.errFile, builder.errToken)
	}
	result.Program = builder.program
	return result
}

type unitBuilder struct {
	program    unit.Program
	lineOffset int
	finalEOF   int
	err        int
	errFile    int
	errToken   int
	declRows   []int
	funcRows   []int
}

type fileTokens struct {
	file     syntax.File
	oldToNew []int
}

func (b *unitBuilder) addFile(file syntax.File, fileIndex int, hasNext bool) bool {
	oldToNew, ok := b.addFileTokens(file, fileIndex, hasNext)
	if !ok {
		return false
	}
	for i := 0; i < len(file.Decls); i++ {
		decl := file.Decls[i]
		nameTok := mapToken(oldToNew, decl.NameTok, b.finalEOF)
		if !b.addDecl(file, decl, nameTok, oldToNew, fileIndex) {
			return false
		}
	}
	for i := 0; i < len(file.Funcs); i++ {
		if !b.addFunc(file, file.Funcs[i], oldToNew, fileIndex) {
			return false
		}
	}
	return true
}

func (b *unitBuilder) addFileTokens(file syntax.File, fileIndex int, hasNext bool) ([]int, bool) {
	base := len(b.program.Text)
	lineOffset := b.lineOffset
	oldToNew := make([]int, len(file.Tokens))
	for i := 0; i < len(file.Tokens); i++ {
		tok := file.Tokens[i]
		if tok.Kind == syntax.TokenEOF {
			oldToNew[i] = b.finalEOF
			continue
		}
		kind, ok := unitTokenKind(file.Src, tok)
		if !ok {
			b.setErr(EmitErrToken, fileIndex, i)
			return nil, false
		}
		oldToNew[i] = len(b.program.Tokens)
		b.program.Tokens = append(b.program.Tokens, unit.Token{
			Kind:  kind,
			Start: base + tok.Start,
			Size:  tok.End - tok.Start,
			Line:  lineOffset + tok.Line,
		})
	}
	b.program.Text = append(b.program.Text, file.Src...)
	b.lineOffset += countNewlines(file.Src)
	if hasNext && (len(file.Src) == 0 || file.Src[len(file.Src)-1] != '\n') {
		b.program.Text = append(b.program.Text, '\n')
		b.lineOffset++
	}
	return oldToNew, true
}

func (b *unitBuilder) finishUnit() bool {
	line := b.lineOffset + 1
	b.program.Tokens = append(b.program.Tokens, unit.Token{
		Kind:  unit.TokenEOF,
		Start: len(b.program.Text),
		Size:  0,
		Line:  line,
	})
	if _, ok := unit.Marshal(b.program); !ok {
		b.setErr(EmitErrUnit, -1, -1)
		return false
	}
	return true
}

func (b *unitBuilder) addDecl(file syntax.File, decl syntax.TopDecl, nameTok int, oldToNew []int, fileIndex int) bool {
	if nameTok < 0 || nameTok >= len(b.program.Tokens) {
		b.setErr(EmitErrToken, fileIndex, decl.NameTok)
		return false
	}
	kind, ok := unitDeclKind(decl.Kind)
	if !ok {
		b.setErr(EmitErrToken, fileIndex, decl.NameTok)
		return false
	}
	name := b.program.Tokens[nameTok]
	b.program.Decls = append(b.program.Decls, unit.Decl{
		Kind:      kind,
		NameStart: name.Start,
		NameEnd:   name.Start + name.Size,
		StartTok:  mapToken(oldToNew, decl.StartTok, b.finalEOF),
		EndTok:    mapToken(oldToNew, decl.EndTok, b.finalEOF),
	})
	return true
}

func (b *unitBuilder) addFunc(file syntax.File, fn syntax.FuncDecl, oldToNew []int, fileIndex int) bool {
	nameTok := mapToken(oldToNew, fn.NameTok, b.finalEOF)
	if nameTok < 0 || nameTok >= len(b.program.Tokens) {
		b.setErr(EmitErrToken, fileIndex, fn.NameTok)
		return false
	}
	bodyEnd := fn.BodyEnd - 1
	if bodyEnd < fn.BodyStart {
		b.setErr(EmitErrToken, fileIndex, fn.BodyEnd)
		return false
	}
	name := b.program.Tokens[nameTok]
	b.program.Funcs = append(b.program.Funcs, unit.Func{
		NameStart:     name.Start,
		NameEnd:       name.Start + name.Size,
		StartTok:      mapToken(oldToNew, fn.StartTok, b.finalEOF),
		NameTok:       nameTok,
		ReceiverStart: mapToken(oldToNew, fn.ReceiverStart, b.finalEOF),
		ReceiverEnd:   mapToken(oldToNew, fn.ReceiverEnd, b.finalEOF),
		BodyStart:     mapToken(oldToNew, fn.BodyStart, b.finalEOF),
		BodyEnd:       mapToken(oldToNew, bodyEnd, b.finalEOF),
		EndTok:        mapToken(oldToNew, fn.EndTok, b.finalEOF),
	})
	return true
}

func (b *unitBuilder) addCheckedImports(info check.PackageInfo, files []fileTokens) bool {
	for i := 0; i < len(info.Imports); i++ {
		imp := info.Imports[i]
		if imp.File < 0 || imp.File >= len(files) {
			b.setErr(EmitErrCheck, -1, imp.Token)
			return false
		}
		decl := findFileImport(files[imp.File].file, imp.Token)
		if decl.PathTok < 0 {
			b.setErr(EmitErrCheck, imp.File, imp.Token)
			return false
		}
		mapped := unit.Import{
			Name:       imp.Name,
			ImportPath: imp.ImportPath,
			Package:    imp.Package,
			NameTok:    mapNullableToken(decl.NameTok, files[imp.File].oldToNew, b.finalEOF),
			PathTok:    mapToken(files[imp.File].oldToNew, decl.PathTok, b.finalEOF),
			Dot:        imp.Dot,
			Blank:      imp.Blank,
		}
		if len(mapped.Name) == 0 || len(mapped.ImportPath) == 0 ||
			mapped.Package < -1 ||
			mapped.NameTok < -1 ||
			mapped.PathTok < 0 ||
			(mapped.Dot && mapped.Blank) {
			b.setErr(EmitErrCheck, imp.File, imp.Token)
			return false
		}
		b.program.Imports = append(b.program.Imports, mapped)
	}
	return true
}

func (b *unitBuilder) addCheckedSymbols(info check.PackageInfo, files []fileTokens) bool {
	for i := 0; i < len(info.Symbols); i++ {
		symbol := info.Symbols[i]
		if symbol.File < 0 || symbol.File >= len(files) {
			b.setErr(EmitErrCheck, -1, symbol.Token)
			return false
		}
		kind, ok := unitSymbolKind(symbol.Kind)
		if !ok {
			b.setErr(EmitErrCheck, symbol.File, symbol.Token)
			return false
		}
		token := mapToken(files[symbol.File].oldToNew, symbol.Token, b.finalEOF)
		if token < 0 {
			b.setErr(EmitErrCheck, symbol.File, symbol.Token)
			return false
		}
		ownerKind, ownerIndex := b.symbolOwner(info, files, i)
		if ownerIndex < 0 {
			b.setErr(EmitErrCheck, symbol.File, symbol.Token)
			return false
		}
		b.program.Symbols = append(b.program.Symbols, unit.Symbol{
			Name:       symbol.Name,
			Kind:       kind,
			Package:    symbol.Package,
			Token:      token,
			OwnerKind:  ownerKind,
			OwnerIndex: ownerIndex,
		})
	}
	return true
}

func (b *unitBuilder) symbolOwner(info check.PackageInfo, files []fileTokens, symbolIndex int) (int, int) {
	symbol := info.Symbols[symbolIndex]
	if symbol.Kind == check.SymbolConst || symbol.Kind == check.SymbolVar || symbol.Kind == check.SymbolType {
		for i := 0; i < len(info.Decls); i++ {
			if info.Decls[i].Symbol == symbolIndex && i < len(b.declRows) {
				return unit.OwnerDecl, b.declRows[i]
			}
		}
		return 0, -1
	}
	if symbol.Kind == check.SymbolFunc || symbol.Kind == check.SymbolMethod {
		for i := 0; i < len(info.Bodies); i++ {
			body := info.Bodies[i]
			if body.File < 0 || body.File >= len(files) || body.File != symbol.File || i >= len(b.funcRows) {
				continue
			}
			if body.Func < 0 || body.Func >= len(files[body.File].file.Funcs) {
				continue
			}
			if files[body.File].file.Funcs[body.Func].NameTok == symbol.Token {
				return unit.OwnerFunc, b.funcRows[i]
			}
		}
	}
	return 0, -1
}

func (b *unitBuilder) addCheckedDecls(info check.PackageInfo, files []fileTokens) bool {
	if len(info.DeclOrder) != len(info.Decls) {
		b.setErr(EmitErrCheck, -1, -1)
		return false
	}
	seen := make([]bool, len(info.Decls))
	b.declRows = make([]int, len(info.Decls))
	for i := 0; i < len(b.declRows); i++ {
		b.declRows[i] = -1
	}
	for i := 0; i < len(info.DeclOrder); i++ {
		index := info.DeclOrder[i]
		if index < 0 || index >= len(info.Decls) || seen[index] {
			b.setErr(EmitErrCheck, -1, -1)
			return false
		}
		seen[index] = true
		declInfo := info.Decls[index]
		if declInfo.File < 0 || declInfo.File >= len(files) {
			b.setErr(EmitErrCheck, -1, declInfo.Token)
			return false
		}
		file := files[declInfo.File].file
		decl := findFileDecl(file, declInfo.Token)
		if decl.NameTok < 0 {
			b.setErr(EmitErrCheck, declInfo.File, declInfo.Token)
			return false
		}
		nameTok := mapToken(files[declInfo.File].oldToNew, decl.NameTok, b.finalEOF)
		if !b.addDecl(file, decl, nameTok, files[declInfo.File].oldToNew, declInfo.File) {
			return false
		}
		ownerIndex := len(b.program.Decls) - 1
		b.declRows[index] = ownerIndex
		meta, ok := b.mapDeclMeta(declInfo, files[declInfo.File].oldToNew, ownerIndex)
		if !ok {
			b.setErr(EmitErrCheck, declInfo.File, declInfo.Token)
			return false
		}
		b.program.DeclMeta = append(b.program.DeclMeta, meta)
		if !b.addDeclShapes(declInfo, files[declInfo.File].oldToNew, ownerIndex) {
			return false
		}
		if !b.addDeclCalls(declInfo, files[declInfo.File].oldToNew, ownerIndex) {
			return false
		}
		if !b.addDeclResolution(declInfo, files[declInfo.File].oldToNew, ownerIndex) {
			return false
		}
	}
	return true
}

func (b *unitBuilder) addCheckedInitOrder(info check.PackageInfo) bool {
	seen := make([]bool, len(info.Decls))
	for i := 0; i < len(info.InitOrder); i++ {
		index := info.InitOrder[i]
		if index < 0 || index >= len(info.Decls) || seen[index] || info.Decls[index].Kind != check.SymbolVar || index >= len(b.declRows) || b.declRows[index] < 0 {
			b.setErr(EmitErrCheck, -1, -1)
			return false
		}
		seen[index] = true
		b.program.InitOrder = append(b.program.InitOrder, b.declRows[index])
	}
	return true
}

func (b *unitBuilder) mapDeclMeta(decl check.DeclInfo, oldToNew []int, declIndex int) (unit.DeclMeta, bool) {
	typeStart, typeEnd, ok := mapNullableTokenSpan(decl.TypeStart, decl.TypeEnd, oldToNew, b.finalEOF)
	if !ok {
		return unit.DeclMeta{}, false
	}
	valueStart, valueEnd, ok := mapNullableTokenSpan(decl.ValueStart, decl.ValueEnd, oldToNew, b.finalEOF)
	if !ok {
		return unit.DeclMeta{}, false
	}
	out := unit.DeclMeta{
		DeclIndex:  declIndex,
		Symbol:     decl.Symbol,
		ValueIndex: decl.ValueIndex,
		TypeStart:  typeStart,
		TypeEnd:    typeEnd,
		ValueStart: valueStart,
		ValueEnd:   valueEnd,
		Values:     make([]unit.ExprSpan, 0, len(decl.Values)),
		Alias:      decl.Alias,
	}
	if declIndex < 0 || out.Symbol < -1 || out.ValueIndex < 0 {
		return out, false
	}
	for i := 0; i < len(decl.Values); i++ {
		span, ok := mapExprSpan(decl.Values[i], oldToNew, b.finalEOF)
		if !ok {
			return out, false
		}
		out.Values = append(out.Values, span)
	}
	return out, true
}

func (b *unitBuilder) addCheckedTypes(info check.PackageInfo, files []fileTokens) bool {
	for i := 0; i < len(info.Types); i++ {
		typ := info.Types[i]
		if typ.File < 0 || typ.File >= len(files) {
			b.setErr(EmitErrCheck, -1, typ.Token)
			return false
		}
		mapped, ok := b.mapTypeInfo(typ, files[typ.File].oldToNew)
		if !ok {
			b.setErr(EmitErrCheck, typ.File, typ.Token)
			return false
		}
		b.program.Types = append(b.program.Types, mapped)
	}
	return true
}

func (b *unitBuilder) addCheckedTypeRefs(info check.PackageInfo, files []fileTokens) bool {
	for i := 0; i < len(info.TypeRefs); i++ {
		ref := info.TypeRefs[i]
		if ref.File < 0 || ref.File >= len(files) {
			b.setErr(EmitErrCheck, -1, ref.Token)
			return false
		}
		if ref.OwnerDecl < 0 || ref.OwnerDecl >= len(b.declRows) || b.declRows[ref.OwnerDecl] < 0 {
			b.setErr(EmitErrCheck, ref.File, ref.Token)
			return false
		}
		mapped, ok := mapTypeRef(ref, files[ref.File].oldToNew, b.finalEOF, unit.OwnerDecl, b.declRows[ref.OwnerDecl])
		if !ok {
			b.setErr(EmitErrCheck, ref.File, ref.Token)
			return false
		}
		b.program.TypeRefs = append(b.program.TypeRefs, mapped)
	}
	return true
}

func (b *unitBuilder) addCheckedFuncs(info check.PackageInfo, files []fileTokens) bool {
	b.funcRows = make([]int, len(info.Bodies))
	for i := 0; i < len(b.funcRows); i++ {
		b.funcRows[i] = -1
	}
	for i := 0; i < len(info.Bodies); i++ {
		body := info.Bodies[i]
		if body.File < 0 || body.File >= len(files) {
			b.setErr(EmitErrCheck, -1, -1)
			return false
		}
		file := files[body.File].file
		if body.Func < 0 || body.Func >= len(file.Funcs) {
			b.setErr(EmitErrCheck, body.File, -1)
			return false
		}
		fn := file.Funcs[body.Func]
		if fn.NameTok < 0 || fn.NameTok >= len(file.Tokens) {
			b.setErr(EmitErrCheck, body.File, fn.NameTok)
			return false
		}
		if !b.addFunc(file, fn, files[body.File].oldToNew, body.File) {
			return false
		}
		ownerIndex := len(b.program.Funcs) - 1
		b.funcRows[i] = ownerIndex
		if !b.addBodySignature(body, files[body.File].oldToNew, ownerIndex) {
			return false
		}
		if !b.addBodyLocals(body, files[body.File].oldToNew, ownerIndex) {
			return false
		}
		if !b.addBodyShapes(body, files[body.File].oldToNew, ownerIndex) {
			return false
		}
		if !b.addBodyFlow(body, files[body.File].oldToNew, ownerIndex) {
			return false
		}
		if !b.addBodyCalls(body, files[body.File].oldToNew, ownerIndex) {
			return false
		}
		if !b.addBodyResolution(body, files[body.File].oldToNew, ownerIndex) {
			return false
		}
		if !b.addBodyTypeRefs(body, files[body.File].oldToNew, ownerIndex) {
			return false
		}
	}
	return true
}

func (b *unitBuilder) addBodySignature(body check.FuncBody, oldToNew []int, funcIndex int) bool {
	sig, ok := mapSignature(body.Signature, oldToNew, b.finalEOF, funcIndex)
	if !ok {
		b.setErr(EmitErrCheck, body.File, body.Body.ErrorTok)
		return false
	}
	b.program.Signatures = append(b.program.Signatures, sig)
	return true
}

func (b *unitBuilder) addBodyLocals(body check.FuncBody, oldToNew []int, funcIndex int) bool {
	for i := 0; i < len(body.Locals); i++ {
		local := body.Locals[i]
		if local.File != body.File {
			b.setErr(EmitErrCheck, body.File, body.Body.ErrorTok)
			return false
		}
		mapped, ok := b.mapLocalDecl(local, oldToNew, funcIndex)
		if !ok {
			b.setErr(EmitErrCheck, body.File, local.Token)
			return false
		}
		b.program.Locals = append(b.program.Locals, mapped)
	}
	return true
}

func (b *unitBuilder) addDeclShapes(decl check.DeclInfo, oldToNew []int, ownerIndex int) bool {
	for i := 0; i < len(decl.Indexes); i++ {
		index, ok := mapIndexExpr(decl.Indexes[i], oldToNew, b.finalEOF, unit.OwnerDecl, ownerIndex)
		if !ok {
			b.setErr(EmitErrCheck, decl.File, decl.Token)
			return false
		}
		b.program.Indexes = append(b.program.Indexes, index)
	}
	for i := 0; i < len(decl.Composites); i++ {
		composite, ok := mapCompositeExpr(decl.Composites[i], oldToNew, b.finalEOF, unit.OwnerDecl, ownerIndex)
		if !ok {
			b.setErr(EmitErrCheck, decl.File, decl.Token)
			return false
		}
		b.program.Composites = append(b.program.Composites, composite)
	}
	return true
}

func (b *unitBuilder) addBodyShapes(body check.FuncBody, oldToNew []int, ownerIndex int) bool {
	for i := 0; i < len(body.Indexes); i++ {
		index, ok := mapIndexExpr(body.Indexes[i], oldToNew, b.finalEOF, unit.OwnerFunc, ownerIndex)
		if !ok {
			b.setErr(EmitErrCheck, body.File, body.Body.ErrorTok)
			return false
		}
		b.program.Indexes = append(b.program.Indexes, index)
	}
	for i := 0; i < len(body.Composites); i++ {
		composite, ok := mapCompositeExpr(body.Composites[i], oldToNew, b.finalEOF, unit.OwnerFunc, ownerIndex)
		if !ok {
			b.setErr(EmitErrCheck, body.File, body.Body.ErrorTok)
			return false
		}
		b.program.Composites = append(b.program.Composites, composite)
	}
	return true
}

func (b *unitBuilder) addDeclCalls(decl check.DeclInfo, oldToNew []int, ownerIndex int) bool {
	for i := 0; i < len(decl.Calls); i++ {
		call, ok := mapCallRef(decl.Calls[i], oldToNew, b.finalEOF, unit.OwnerDecl, ownerIndex)
		if !ok {
			b.setErr(EmitErrCheck, decl.File, decl.Token)
			return false
		}
		b.program.Calls = append(b.program.Calls, call)
	}
	return true
}

func (b *unitBuilder) addDeclResolution(decl check.DeclInfo, oldToNew []int, ownerIndex int) bool {
	for i := 0; i < len(decl.Refs); i++ {
		ref, ok := mapNameRef(decl.Refs[i], oldToNew, b.finalEOF, unit.OwnerDecl, ownerIndex)
		if !ok {
			b.setErr(EmitErrCheck, decl.File, decl.Token)
			return false
		}
		b.program.Refs = append(b.program.Refs, ref)
	}
	for i := 0; i < len(decl.Selectors); i++ {
		selector, ok := mapSelectorRef(decl.Selectors[i], oldToNew, b.finalEOF, unit.OwnerDecl, ownerIndex)
		if !ok {
			b.setErr(EmitErrCheck, decl.File, decl.Token)
			return false
		}
		b.program.Selectors = append(b.program.Selectors, selector)
	}
	return true
}

func (b *unitBuilder) addBodyFlow(body check.FuncBody, oldToNew []int, funcIndex int) bool {
	for i := 0; i < len(body.Assigns); i++ {
		assign, ok := mapAssignment(body.Assigns[i], oldToNew, b.finalEOF, funcIndex)
		if !ok {
			b.setErr(EmitErrCheck, body.File, body.Body.ErrorTok)
			return false
		}
		b.program.Assigns = append(b.program.Assigns, assign)
	}
	for i := 0; i < len(body.Returns); i++ {
		ret, ok := mapReturn(body.Returns[i], oldToNew, b.finalEOF, funcIndex)
		if !ok {
			b.setErr(EmitErrCheck, body.File, body.Body.ErrorTok)
			return false
		}
		b.program.Returns = append(b.program.Returns, ret)
	}
	return true
}

func (b *unitBuilder) addBodyResolution(body check.FuncBody, oldToNew []int, ownerIndex int) bool {
	for i := 0; i < len(body.Refs); i++ {
		ref, ok := mapNameRef(body.Refs[i], oldToNew, b.finalEOF, unit.OwnerFunc, ownerIndex)
		if !ok {
			b.setErr(EmitErrCheck, body.File, body.Body.ErrorTok)
			return false
		}
		b.program.Refs = append(b.program.Refs, ref)
	}
	for i := 0; i < len(body.Selectors); i++ {
		selector, ok := mapSelectorRef(body.Selectors[i], oldToNew, b.finalEOF, unit.OwnerFunc, ownerIndex)
		if !ok {
			b.setErr(EmitErrCheck, body.File, body.Body.ErrorTok)
			return false
		}
		b.program.Selectors = append(b.program.Selectors, selector)
	}
	return true
}

func (b *unitBuilder) addBodyCalls(body check.FuncBody, oldToNew []int, ownerIndex int) bool {
	for i := 0; i < len(body.Calls); i++ {
		call, ok := mapCallRef(body.Calls[i], oldToNew, b.finalEOF, unit.OwnerFunc, ownerIndex)
		if !ok {
			b.setErr(EmitErrCheck, body.File, body.Body.ErrorTok)
			return false
		}
		b.program.Calls = append(b.program.Calls, call)
	}
	return true
}

func (b *unitBuilder) addBodyTypeRefs(body check.FuncBody, oldToNew []int, ownerIndex int) bool {
	for i := 0; i < len(body.TypeRefs); i++ {
		if body.TypeRefs[i].File != body.File {
			b.setErr(EmitErrCheck, body.File, body.Body.ErrorTok)
			return false
		}
		ref, ok := mapTypeRef(body.TypeRefs[i], oldToNew, b.finalEOF, unit.OwnerFunc, ownerIndex)
		if !ok {
			b.setErr(EmitErrCheck, body.File, body.Body.ErrorTok)
			return false
		}
		b.program.TypeRefs = append(b.program.TypeRefs, ref)
	}
	return true
}

func mapSignature(sig check.FuncSignature, oldToNew []int, eof int, funcIndex int) (unit.FuncSignature, bool) {
	out := unit.FuncSignature{FuncIndex: funcIndex}
	var ok bool
	out.Receiver, ok = mapFields(sig.Receiver, oldToNew, eof)
	if !ok {
		return out, false
	}
	out.Params, ok = mapFields(sig.Params, oldToNew, eof)
	if !ok {
		return out, false
	}
	out.Results, ok = mapFields(sig.Results, oldToNew, eof)
	if !ok {
		return out, false
	}
	return out, funcIndex >= 0
}

func mapFields(fields []check.Field, oldToNew []int, eof int) ([]unit.Field, bool) {
	out := make([]unit.Field, 0, len(fields))
	for i := 0; i < len(fields); i++ {
		field := fields[i]
		mapped := unit.Field{
			NameTok:   mapNullableToken(field.NameTok, oldToNew, eof),
			TypeStart: mapToken(oldToNew, field.TypeStart, eof),
			TypeEnd:   mapToken(oldToNew, field.TypeEnd, eof),
			Variadic:  field.Variadic,
		}
		if mapped.NameTok < -1 || mapped.TypeStart < 0 || mapped.TypeEnd < mapped.TypeStart {
			return nil, false
		}
		out = append(out, mapped)
	}
	return out, true
}

func (b *unitBuilder) mapLocalDecl(local check.LocalDeclInfo, oldToNew []int, funcIndex int) (unit.LocalDecl, bool) {
	kind, ok := unitDeclKindFromSymbol(local.Kind)
	if !ok {
		return unit.LocalDecl{}, false
	}
	nameTok := mapToken(oldToNew, local.Token, b.finalEOF)
	if nameTok < 0 || nameTok >= len(b.program.Tokens) {
		return unit.LocalDecl{}, false
	}
	typeStart, typeEnd, ok := mapNullableTokenSpan(local.TypeStart, local.TypeEnd, oldToNew, b.finalEOF)
	if !ok {
		return unit.LocalDecl{}, false
	}
	valueStart, valueEnd, ok := mapNullableTokenSpan(local.ValueStart, local.ValueEnd, oldToNew, b.finalEOF)
	if !ok {
		return unit.LocalDecl{}, false
	}
	out := unit.LocalDecl{
		FuncIndex:  funcIndex,
		Kind:       kind,
		Token:      nameTok,
		Scope:      local.Scope,
		ValueIndex: local.ValueIndex,
		TypeStart:  typeStart,
		TypeEnd:    typeEnd,
		ValueStart: valueStart,
		ValueEnd:   valueEnd,
		Values:     make([]unit.ExprSpan, 0, len(local.Values)),
		Alias:      local.Alias,
	}
	name := b.program.Tokens[nameTok]
	out.NameStart = name.Start
	out.NameEnd = name.Start + name.Size
	if funcIndex < 0 || out.Scope < -1 || out.ValueIndex < 0 {
		return out, false
	}
	for i := 0; i < len(local.Values); i++ {
		span, ok := mapExprSpan(local.Values[i], oldToNew, b.finalEOF)
		if !ok {
			return out, false
		}
		out.Values = append(out.Values, span)
	}
	return out, true
}

func (b *unitBuilder) mapTypeInfo(typ check.TypeInfo, oldToNew []int) (unit.TypeInfo, bool) {
	kind, ok := unitTypeKind(typ.Kind)
	if !ok {
		return unit.TypeInfo{}, false
	}
	if typ.Decl < 0 || typ.Decl >= len(b.declRows) || b.declRows[typ.Decl] < 0 {
		return unit.TypeInfo{}, false
	}
	nameTok := mapToken(oldToNew, typ.Token, b.finalEOF)
	if nameTok < 0 || nameTok >= len(b.program.Tokens) {
		return unit.TypeInfo{}, false
	}
	typeStart, typeEnd, ok := mapNullableTokenSpan(typ.TypeStart, typ.TypeEnd, oldToNew, b.finalEOF)
	if !ok {
		return unit.TypeInfo{}, false
	}
	lenStart, lenEnd, ok := mapNullableTokenSpan(typ.LenStart, typ.LenEnd, oldToNew, b.finalEOF)
	if !ok {
		return unit.TypeInfo{}, false
	}
	keyStart, keyEnd, ok := mapNullableTokenSpan(typ.KeyStart, typ.KeyEnd, oldToNew, b.finalEOF)
	if !ok {
		return unit.TypeInfo{}, false
	}
	elemStart, elemEnd, ok := mapNullableTokenSpan(typ.ElemStart, typ.ElemEnd, oldToNew, b.finalEOF)
	if !ok {
		return unit.TypeInfo{}, false
	}
	name := b.program.Tokens[nameTok]
	return unit.TypeInfo{
		NameStart: name.Start,
		NameEnd:   name.Start + name.Size,
		Kind:      kind,
		Decl:      b.declRows[typ.Decl],
		Symbol:    typ.Symbol,
		Alias:     typ.Alias,
		TypeStart: typeStart,
		TypeEnd:   typeEnd,
		LenStart:  lenStart,
		LenEnd:    lenEnd,
		KeyStart:  keyStart,
		KeyEnd:    keyEnd,
		ElemStart: elemStart,
		ElemEnd:   elemEnd,
	}, true
}

func unitTypeKind(kind int) (int, bool) {
	if kind == check.TypeOther {
		return unit.TypeOther, true
	}
	if kind == check.TypeNamed {
		return unit.TypeNamed, true
	}
	if kind == check.TypeStruct {
		return unit.TypeStruct, true
	}
	if kind == check.TypeInterface {
		return unit.TypeInterface, true
	}
	if kind == check.TypeMap {
		return unit.TypeMap, true
	}
	if kind == check.TypeSlice {
		return unit.TypeSlice, true
	}
	if kind == check.TypeArray {
		return unit.TypeArray, true
	}
	if kind == check.TypePointer {
		return unit.TypePointer, true
	}
	if kind == check.TypeFunc {
		return unit.TypeFunc, true
	}
	return 0, false
}

func mapTypeRef(ref check.TypeRef, oldToNew []int, eof int, ownerKind int, ownerIndex int) (unit.TypeRef, bool) {
	kind, ok := unitTypeRefKind(ref.Kind)
	if !ok {
		return unit.TypeRef{}, false
	}
	out := unit.TypeRef{
		OwnerKind:  ownerKind,
		OwnerIndex: ownerIndex,
		Kind:       kind,
		Token:      mapToken(oldToNew, ref.Token, eof),
		BaseTok:    mapToken(oldToNew, ref.BaseToken, eof),
		DotTok:     mapToken(oldToNew, ref.DotToken, eof),
		Package:    ref.Package,
		Symbol:     ref.Symbol,
	}
	if ownerIndex < 0 || out.Token < 0 || out.BaseTok < 0 || out.DotTok < 0 || out.Package < -1 || out.Symbol < -1 {
		return out, false
	}
	return out, true
}

func unitTypeRefKind(kind int) (int, bool) {
	if kind == check.TypeRefUnknown {
		return unit.TypeRefUnknown, true
	}
	if kind == check.TypeRefScope {
		return unit.TypeRefScope, true
	}
	if kind == check.TypeRefPackage {
		return unit.TypeRefPackage, true
	}
	if kind == check.TypeRefImportSelector {
		return unit.TypeRefImportSelector, true
	}
	if kind == check.TypeRefBuiltin {
		return unit.TypeRefBuiltin, true
	}
	return 0, false
}

func mapIndexExpr(index check.IndexExpr, oldToNew []int, eof int, ownerKind int, ownerIndex int) (unit.IndexExpr, bool) {
	out := unit.IndexExpr{
		OwnerKind:  ownerKind,
		OwnerIndex: ownerIndex,
		StartTok:   mapToken(oldToNew, index.StartTok, eof),
		EndTok:     mapToken(oldToNew, index.EndTok, eof),
		BaseStart:  mapToken(oldToNew, index.BaseStart, eof),
		BaseEnd:    mapToken(oldToNew, index.BaseEnd, eof),
		OpenTok:    mapToken(oldToNew, index.OpenTok, eof),
		CloseTok:   mapToken(oldToNew, index.CloseTok, eof),
		IndexStart: mapToken(oldToNew, index.IndexStart, eof),
		IndexEnd:   mapToken(oldToNew, index.IndexEnd, eof),
	}
	if out.StartTok < 0 || out.EndTok < out.StartTok || out.BaseStart < 0 || out.BaseEnd < out.BaseStart || out.IndexStart < 0 || out.IndexEnd < out.IndexStart {
		return out, false
	}
	if out.OpenTok < 0 || out.CloseTok < 0 || ownerIndex < 0 {
		return out, false
	}
	return out, true
}

func mapCompositeExpr(composite check.CompositeExpr, oldToNew []int, eof int, ownerKind int, ownerIndex int) (unit.CompositeExpr, bool) {
	out := unit.CompositeExpr{
		OwnerKind:  ownerKind,
		OwnerIndex: ownerIndex,
		StartTok:   mapToken(oldToNew, composite.StartTok, eof),
		EndTok:     mapToken(oldToNew, composite.EndTok, eof),
		TypeStart:  mapToken(oldToNew, composite.TypeStart, eof),
		TypeEnd:    mapToken(oldToNew, composite.TypeEnd, eof),
		OpenTok:    mapToken(oldToNew, composite.OpenTok, eof),
		CloseTok:   mapToken(oldToNew, composite.CloseTok, eof),
		Elems:      make([]unit.ExprSpan, 0, len(composite.Elems)),
	}
	if out.StartTok < 0 || out.EndTok < out.StartTok || out.TypeStart < 0 || out.TypeEnd < out.TypeStart {
		return out, false
	}
	if out.OpenTok < 0 || out.CloseTok < 0 || ownerIndex < 0 {
		return out, false
	}
	for i := 0; i < len(composite.Elems); i++ {
		elem := composite.Elems[i]
		mapped := unit.ExprSpan{
			StartTok: mapToken(oldToNew, elem.StartTok, eof),
			EndTok:   mapToken(oldToNew, elem.EndTok, eof),
		}
		if mapped.StartTok < 0 || mapped.EndTok < mapped.StartTok {
			return out, false
		}
		out.Elems = append(out.Elems, mapped)
	}
	return out, true
}

func mapNameRef(ref check.NameRef, oldToNew []int, eof int, ownerKind int, ownerIndex int) (unit.NameRef, bool) {
	out := unit.NameRef{
		OwnerKind:  ownerKind,
		OwnerIndex: ownerIndex,
		Kind:       ref.Kind,
		Token:      mapToken(oldToNew, ref.Token, eof),
		Index:      ref.Index,
		Package:    ref.Package,
	}
	if ownerIndex < 0 || out.Token < 0 || out.Index < -1 || out.Package < -1 {
		return out, false
	}
	return out, true
}

func mapSelectorRef(selector check.SelectorRef, oldToNew []int, eof int, ownerKind int, ownerIndex int) (unit.Selector, bool) {
	out := unit.Selector{
		OwnerKind:   ownerKind,
		OwnerIndex:  ownerIndex,
		Kind:        selector.Kind,
		BaseTok:     mapToken(oldToNew, selector.BaseToken, eof),
		DotTok:      mapToken(oldToNew, selector.DotToken, eof),
		NameTok:     mapToken(oldToNew, selector.NameToken, eof),
		BaseKind:    selector.BaseRef.Kind,
		BaseIndex:   selector.BaseRef.Index,
		BasePackage: selector.BaseRef.Package,
		Package:     selector.Package,
		Symbol:      selector.Symbol,
	}
	if ownerIndex < 0 || out.BaseTok < 0 || out.DotTok < 0 || out.NameTok < 0 ||
		out.BaseIndex < -1 || out.BasePackage < -1 || out.Package < -1 || out.Symbol < -1 {
		return out, false
	}
	return out, true
}

func mapCallRef(call check.CallRef, oldToNew []int, eof int, ownerKind int, ownerIndex int) (unit.Call, bool) {
	out := unit.Call{
		OwnerKind:  ownerKind,
		OwnerIndex: ownerIndex,
		Kind:       call.Kind,
		CalleeTok:  mapToken(oldToNew, call.CalleeToken, eof),
		BaseTok:    mapToken(oldToNew, call.BaseToken, eof),
		DotTok:     mapToken(oldToNew, call.DotToken, eof),
		ArgsStart:  mapToken(oldToNew, call.ArgsStart, eof),
		ArgsEnd:    mapToken(oldToNew, call.ArgsEnd, eof),
		Args:       make([]unit.ExprSpan, 0, len(call.Args)),
	}
	if ownerIndex < 0 || out.CalleeTok < 0 || out.BaseTok < 0 || out.DotTok < 0 || out.ArgsStart < 0 || out.ArgsEnd < out.ArgsStart {
		return out, false
	}
	for i := 0; i < len(call.Args); i++ {
		span, ok := mapExprSpan(call.Args[i], oldToNew, eof)
		if !ok {
			return out, false
		}
		out.Args = append(out.Args, span)
	}
	return out, true
}

func mapAssignment(assign check.AssignInfo, oldToNew []int, eof int, funcIndex int) (unit.Assignment, bool) {
	out := unit.Assignment{
		FuncIndex:  funcIndex,
		Kind:       assign.Kind,
		StartTok:   mapToken(oldToNew, assign.StartTok, eof),
		EndTok:     mapToken(oldToNew, assign.EndTok, eof),
		OpTok:      mapToken(oldToNew, assign.OpTok, eof),
		LeftStart:  mapToken(oldToNew, assign.LeftStart, eof),
		LeftEnd:    mapToken(oldToNew, assign.LeftEnd, eof),
		RightStart: mapToken(oldToNew, assign.RightStart, eof),
		RightEnd:   mapToken(oldToNew, assign.RightEnd, eof),
		Targets:    make([]unit.ExprSpan, 0, len(assign.Targets)),
		Values:     make([]unit.ExprSpan, 0, len(assign.Values)),
	}
	if funcIndex < 0 || out.StartTok < 0 || out.EndTok < out.StartTok || out.OpTok < 0 ||
		out.LeftStart < 0 || out.LeftEnd < out.LeftStart || out.RightStart < 0 || out.RightEnd < out.RightStart {
		return out, false
	}
	for i := 0; i < len(assign.Targets); i++ {
		span, ok := mapExprSpan(assign.Targets[i].Span, oldToNew, eof)
		if !ok {
			return out, false
		}
		out.Targets = append(out.Targets, span)
	}
	for i := 0; i < len(assign.Values); i++ {
		span, ok := mapExprSpan(assign.Values[i], oldToNew, eof)
		if !ok {
			return out, false
		}
		out.Values = append(out.Values, span)
	}
	return out, true
}

func mapReturn(ret check.ReturnInfo, oldToNew []int, eof int, funcIndex int) (unit.Return, bool) {
	out := unit.Return{
		FuncIndex: funcIndex,
		StartTok:  mapToken(oldToNew, ret.StartTok, eof),
		EndTok:    mapToken(oldToNew, ret.EndTok, eof),
		Values:    make([]unit.ExprSpan, 0, len(ret.Values)),
	}
	if funcIndex < 0 || out.StartTok < 0 || out.EndTok < out.StartTok {
		return out, false
	}
	for i := 0; i < len(ret.Values); i++ {
		span, ok := mapExprSpan(ret.Values[i], oldToNew, eof)
		if !ok {
			return out, false
		}
		out.Values = append(out.Values, span)
	}
	return out, true
}

func mapExprSpan(span check.ExprSpan, oldToNew []int, eof int) (unit.ExprSpan, bool) {
	out := unit.ExprSpan{
		StartTok: mapToken(oldToNew, span.StartTok, eof),
		EndTok:   mapToken(oldToNew, span.EndTok, eof),
	}
	if out.StartTok < 0 || out.EndTok < out.StartTok {
		return out, false
	}
	return out, true
}

func mapNullableTokenSpan(start int, end int, oldToNew []int, eof int) (int, int, bool) {
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

func findFileDecl(file syntax.File, nameTok int) syntax.TopDecl {
	for i := 0; i < len(file.Decls); i++ {
		if file.Decls[i].NameTok == nameTok {
			return file.Decls[i]
		}
	}
	return syntax.TopDecl{NameTok: -1}
}

func findFileImport(file syntax.File, tok int) syntax.ImportDecl {
	for i := 0; i < len(file.Imports); i++ {
		imp := file.Imports[i]
		if imp.NameTok == tok || imp.PathTok == tok {
			return imp
		}
	}
	return syntax.ImportDecl{PathTok: -1}
}

func (b *unitBuilder) setErr(err int, file int, tok int) {
	b.err = err
	b.errFile = file
	b.errToken = tok
}

func countPackageTokens(pkg load.Package) int {
	count := 0
	for i := 0; i < len(pkg.Files); i++ {
		file := pkg.Files[i].File
		for j := 0; j < len(file.Tokens); j++ {
			if file.Tokens[j].Kind != syntax.TokenEOF {
				count++
			}
		}
	}
	return count
}

func unitTokenKind(src []byte, tok syntax.Token) (int, bool) {
	if tok.Kind == syntax.TokenEOF {
		return unit.TokenEOF, true
	}
	if tok.Kind == syntax.TokenIdent {
		return unit.TokenIdent, true
	}
	if tok.Kind == syntax.TokenNumber {
		if isFloatNumber(src, tok) {
			return unit.TokenFloat, true
		}
		return unit.TokenNumber, true
	}
	if tok.Kind == syntax.TokenString {
		return unit.TokenString, true
	}
	if tok.Kind == syntax.TokenChar {
		return unit.TokenChar, true
	}
	if tok.Kind == syntax.TokenOperator {
		return unit.TokenOp, true
	}
	if tok.Kind == syntax.TokenPackage {
		return unit.TokenPackage, true
	}
	if tok.Kind == syntax.TokenConst {
		return unit.TokenConst, true
	}
	if tok.Kind == syntax.TokenVar {
		return unit.TokenVar, true
	}
	if tok.Kind == syntax.TokenType {
		return unit.TokenType, true
	}
	if tok.Kind == syntax.TokenFunc {
		return unit.TokenFunc, true
	}
	if tok.Kind == syntax.TokenStruct {
		return unit.TokenStruct, true
	}
	if tok.Kind == syntax.TokenReturn {
		return unit.TokenReturn, true
	}
	if tok.Kind == syntax.TokenIf {
		return unit.TokenIf, true
	}
	if tok.Kind == syntax.TokenElse {
		return unit.TokenElse, true
	}
	if tok.Kind == syntax.TokenFor {
		return unit.TokenFor, true
	}
	if tok.Kind == syntax.TokenBreak {
		return unit.TokenBreak, true
	}
	if tok.Kind == syntax.TokenContinue {
		return unit.TokenContinue, true
	}
	if tok.Kind == syntax.TokenGoto {
		return unit.TokenGoto, true
	}
	if tok.Kind == syntax.TokenSwitch {
		return unit.TokenSwitch, true
	}
	if tok.Kind == syntax.TokenCase {
		return unit.TokenCase, true
	}
	if tok.Kind == syntax.TokenDefault {
		return unit.TokenDefault, true
	}
	return unit.TokenIdent, true
}

func unitDeclKind(kind int) (int, bool) {
	if kind == syntax.TokenConst {
		return unit.TokenConst, true
	}
	if kind == syntax.TokenVar {
		return unit.TokenVar, true
	}
	if kind == syntax.TokenType {
		return unit.TokenType, true
	}
	return 0, false
}

func unitDeclKindFromSymbol(kind int) (int, bool) {
	if kind == check.SymbolConst {
		return unit.TokenConst, true
	}
	if kind == check.SymbolVar {
		return unit.TokenVar, true
	}
	if kind == check.SymbolType {
		return unit.TokenType, true
	}
	return 0, false
}

func unitSymbolKind(kind int) (int, bool) {
	if kind == check.SymbolConst {
		return unit.SymbolConst, true
	}
	if kind == check.SymbolVar {
		return unit.SymbolVar, true
	}
	if kind == check.SymbolType {
		return unit.SymbolType, true
	}
	if kind == check.SymbolFunc {
		return unit.SymbolFunc, true
	}
	if kind == check.SymbolMethod {
		return unit.SymbolMethod, true
	}
	return 0, false
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

func countNewlines(src []byte) int {
	count := 0
	for i := 0; i < len(src); i++ {
		if src[i] == '\n' {
			count++
		}
	}
	return count
}

func isFloatNumber(src []byte, tok syntax.Token) bool {
	for i := tok.Start; i < tok.End && i < len(src); i++ {
		c := src[i]
		if c == '.' || c == 'e' || c == 'E' || c == 'p' || c == 'P' {
			return true
		}
	}
	return false
}

func emitFail(result Result, err int, file int, tok int) Result {
	result.Ok = false
	result.Error = err
	result.ErrorFile = file
	result.ErrorToken = tok
	return result
}
