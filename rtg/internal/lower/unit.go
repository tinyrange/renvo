package lower

import (
	"j5.nz/rtg/rtg/internal/arena"
	"j5.nz/rtg/rtg/internal/check"
	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtg/internal/syntax"
	"j5.nz/rtg/rtg/internal/unit"
)

// EmitCheckedPackageCore lowers the compact checked-package metadata used by
// both host-built and self-hosted frontend pipelines.
func EmitCheckedPackageCore(pkg load.Package, info check.PackageInfo, transient bool) Result {
	result := Result{Ok: true, Error: EmitOK, ErrorFile: -1, ErrorToken: -1}
	if !pkg.Ok || pkg.Name == "" || len(pkg.Files) == 0 {
		return emitFail(result, EmitErrPackage, -1, -1)
	}
	if info.Name != pkg.Name {
		return emitFail(result, EmitErrCheck, -1, -1)
	}
	var builder coreUnitBuilder
	builder.program.Package = cloneCoreString(pkg.Name)
	builder.program.ImportPath = cloneCoreString(pkg.Ref.ImportPath)
	files, ok := builder.prepareFileTokens(pkg)
	if !ok {
		return emitFail(result, builder.err, builder.errFile, builder.errToken)
	}
	builder.reserveCheckedPackage(pkg, info)
	if !builder.addCheckedImports(info, files) {
		return emitFail(result, builder.err, builder.errFile, builder.errToken)
	}
	if !builder.addCheckedDecls(info, files) {
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
	if transient {
		arena.Discard(info.CoreArenaStart, info.CoreArenaEnd)
	}
	for i := 0; i < len(pkg.Files); i++ {
		file := pkg.Files[i].File
		if _, ok := builder.addFileTokens(file, pkg.Files[i].Src, i, i+1 < len(pkg.Files)); !ok {
			return emitFail(result, builder.err, builder.errFile, builder.errToken)
		}
	}
	if !builder.finishUnit() {
		return emitFail(result, builder.err, builder.errFile, builder.errToken)
	}
	result.Program = builder.program
	return result
}

type coreUnitBuilder struct {
	program    unit.Program
	lineOffset int
	finalEOF   int
	err        int
	errFile    int
	errToken   int
	declRows   []int
	funcRows   []int
}

type coreFileTokens struct {
	file     syntax.File
	tokens   coreTokenMap
	textBase int
}

type coreTokenMap struct {
	base  int
	limit int
}

func (b *coreUnitBuilder) prepareFileTokens(pkg load.Package) ([]coreFileTokens, bool) {
	files := make([]coreFileTokens, len(pkg.Files))
	textBase := 0
	tokenBase := 0
	for i := 0; i < len(pkg.Files); i++ {
		file := pkg.Files[i].File
		if !file.Ok {
			b.setErr(EmitErrPackage, i, file.ErrorTok)
			return files, false
		}
		files[i] = coreFileTokens{
			file:     file,
			tokens:   coreTokenMap{base: tokenBase, limit: len(file.Tokens)},
			textBase: textBase,
		}
		for j := 0; j < len(file.Tokens); j++ {
			if file.Tokens[j].Kind != syntax.TokenEOF {
				tokenBase++
			}
		}
		src := pkg.Files[i].Src
		textBase += len(src)
		if i+1 < len(pkg.Files) && (len(src) == 0 || src[len(src)-1] != '\n') {
			textBase++
		}
	}
	b.finalEOF = tokenBase
	return files, true
}

func (b *coreUnitBuilder) reserveCheckedPackage(pkg load.Package, info check.PackageInfo) {
	textCap := 0
	tokenCap := 1
	for i := 0; i < len(pkg.Files); i++ {
		src := pkg.Files[i].Src
		textCap += len(src)
		tokenCap += len(pkg.Files[i].File.Tokens)
		if i+1 < len(pkg.Files) && (len(src) == 0 || src[len(src)-1] != '\n') {
			textCap++
		}
	}
	typeRefCap := len(info.CoreTypeRefs)
	callCap := 0
	refCap := 0
	selectorCap := 0
	for i := 0; i < len(info.Decls); i++ {
		decl := info.Decls[i]
		refCap += len(decl.CoreRefs)
		selectorCap += len(decl.CoreSelectors)
	}
	for i := 0; i < len(info.CoreBodies); i++ {
		body := info.CoreBodies[i]
		typeRefCap += len(body.CoreTypeRefs)
		refCap += len(body.CoreRefs)
		selectorCap += len(body.CoreSelectors)
	}
	b.program.Text = make([]byte, 0, textCap)
	b.program.Tokens = make([]unit.Token, 0, tokenCap)
	b.program.Imports = make([]unit.Import, 0, len(info.Imports))
	b.program.Symbols = make([]unit.Symbol, 0, len(info.Symbols))
	b.program.Decls = make([]unit.Decl, 0, len(info.Decls))
	b.program.Funcs = make([]unit.Func, 0, len(info.CoreBodies))
	b.program.TypeRefs = make([]unit.TypeRef, 0, typeRefCap)
	b.program.Calls = make([]unit.Call, 0, callCap)
	b.program.Refs = make([]unit.NameRef, 0, refCap)
	b.program.Selectors = make([]unit.Selector, 0, selectorCap)
}

func (b *coreUnitBuilder) addFileTokens(file syntax.File, src []byte, fileIndex int, hasNext bool) (coreTokenMap, bool) {
	base := len(b.program.Text)
	tokenBase := len(b.program.Tokens)
	lineOffset := b.lineOffset
	for i := 0; i < len(file.Tokens); i++ {
		tok := file.Tokens[i]
		if tok.Kind == syntax.TokenEOF {
			continue
		}
		kind, ok := coreUnitTokenKind(src, tok)
		if !ok {
			b.setErr(EmitErrToken, fileIndex, i)
			return coreTokenMap{}, false
		}
		b.program.Tokens = append(b.program.Tokens, unit.Token{
			Kind:  kind,
			Start: base + tok.Start,
			Size:  tok.End - tok.Start,
			Line:  lineOffset + tok.Line,
		})
	}
	b.program.Text = appendCoreBytes(b.program.Text, src)
	b.lineOffset += countCoreNewlines(src)
	if hasNext && (len(src) == 0 || src[len(src)-1] != '\n') {
		b.program.Text = append(b.program.Text, '\n')
		b.lineOffset++
	}
	return coreTokenMap{base: tokenBase, limit: len(file.Tokens)}, true
}

func appendCoreBytes(out []byte, data []byte) []byte {
	return append(out, data...)
}

func (b *coreUnitBuilder) finishUnit() bool {
	line := b.lineOffset + 1
	b.program.Tokens = append(b.program.Tokens, unit.Token{
		Kind:  unit.TokenEOF,
		Start: len(b.program.Text),
		Size:  0,
		Line:  line,
	})
	return true
}

func (b *coreUnitBuilder) addDecl(file syntax.File, decl syntax.TopDecl, mapping coreTokenMap, textBase int, fileIndex int) bool {
	if decl.NameTok < 0 || decl.NameTok >= len(file.Tokens) {
		b.setErr(EmitErrToken, fileIndex, decl.NameTok)
		return false
	}
	kind, ok := coreUnitDeclKind(decl.Kind)
	if !ok {
		b.setErr(EmitErrToken, fileIndex, decl.NameTok)
		return false
	}
	name := file.Tokens[decl.NameTok]
	if name.Start < 0 || name.End < name.Start || name.End > len(file.Src) {
		b.setErr(EmitErrToken, fileIndex, decl.NameTok)
		return false
	}
	b.program.Decls = append(b.program.Decls, unit.Decl{
		Kind:      kind,
		NameStart: textBase + name.Start,
		NameEnd:   textBase + name.End,
		StartTok:  mapCoreDeclStartToken(file, decl, mapping, b.finalEOF),
		EndTok:    mapCoreToken(mapping, decl.EndTok, b.finalEOF),
	})
	return true
}

func mapCoreDeclStartToken(file syntax.File, decl syntax.TopDecl, mapping coreTokenMap, eof int) int {
	start := decl.StartTok
	if start > 0 && start < len(file.Tokens) && file.Tokens[start-1].Kind == decl.Kind {
		start--
	}
	return mapCoreToken(mapping, start, eof)
}

func (b *coreUnitBuilder) addFunc(file syntax.File, fn syntax.FuncDecl, mapping coreTokenMap, textBase int, fileIndex int) bool {
	if fn.NameTok < 0 || fn.NameTok >= len(file.Tokens) {
		b.setErr(EmitErrToken, fileIndex, fn.NameTok)
		return false
	}
	nameTok := mapCoreToken(mapping, fn.NameTok, b.finalEOF)
	bodyEnd := fn.BodyEnd - 1
	if bodyEnd < fn.BodyStart {
		b.setErr(EmitErrToken, fileIndex, fn.BodyEnd)
		return false
	}
	name := file.Tokens[fn.NameTok]
	if name.Start < 0 || name.End < name.Start || name.End > len(file.Src) {
		b.setErr(EmitErrToken, fileIndex, fn.NameTok)
		return false
	}
	b.program.Funcs = append(b.program.Funcs, unit.Func{
		NameStart:     textBase + name.Start,
		NameEnd:       textBase + name.End,
		StartTok:      mapCoreToken(mapping, fn.StartTok, b.finalEOF),
		NameTok:       nameTok,
		ReceiverStart: mapCoreToken(mapping, fn.ReceiverStart, b.finalEOF),
		ReceiverEnd:   mapCoreToken(mapping, fn.ReceiverEnd, b.finalEOF),
		BodyStart:     mapCoreToken(mapping, fn.BodyStart, b.finalEOF),
		BodyEnd:       mapCoreToken(mapping, bodyEnd, b.finalEOF),
		EndTok:        mapCoreToken(mapping, fn.EndTok, b.finalEOF),
	})
	return true
}

func (b *coreUnitBuilder) addCheckedImports(info check.PackageInfo, files []coreFileTokens) bool {
	for i := 0; i < len(info.Imports); i++ {
		imp := info.Imports[i]
		if imp.File < 0 || imp.File >= len(files) {
			b.setErr(EmitErrCheck, -1, imp.Token)
			return false
		}
		decl := findCoreFileImport(files[imp.File].file, imp.Token)
		if decl.PathTok < 0 {
			b.setErr(EmitErrCheck, imp.File, imp.Token)
			return false
		}
		b.program.Imports = append(b.program.Imports, unit.Import{
			NameTok: mapNullableCoreToken(decl.NameTok, files[imp.File].tokens, b.finalEOF),
			PathTok: mapCoreToken(files[imp.File].tokens, decl.PathTok, b.finalEOF),
		})
	}
	return true
}

func (b *coreUnitBuilder) addCheckedDecls(info check.PackageInfo, files []coreFileTokens) bool {
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
		decl := findCoreFileDecl(file, declInfo.Token)
		if decl.NameTok < 0 {
			b.setErr(EmitErrCheck, declInfo.File, declInfo.Token)
			return false
		}
		if !b.addDecl(file, decl, files[declInfo.File].tokens, files[declInfo.File].textBase, declInfo.File) {
			return false
		}
		ownerIndex := len(b.program.Decls) - 1
		b.declRows[index] = ownerIndex
		if !b.addDeclCalls(declInfo, files[declInfo.File].tokens, ownerIndex) {
			return false
		}
		if !b.addDeclResolution(declInfo, files[declInfo.File].tokens, ownerIndex) {
			return false
		}
	}
	return true
}

func (b *coreUnitBuilder) addCheckedTypeRefs(info check.PackageInfo, files []coreFileTokens) bool {
	for i := 0; i < len(info.CoreTypeRefs); i++ {
		ref := info.CoreTypeRefs[i]
		if ref.File < 0 || ref.File >= len(files) {
			b.setErr(EmitErrCheck, -1, ref.Token)
			return false
		}
		if ref.OwnerDecl < 0 || ref.OwnerDecl >= len(b.declRows) || b.declRows[ref.OwnerDecl] < 0 {
			b.setErr(EmitErrCheck, ref.File, ref.Token)
			return false
		}
		mapped, ok := mapCoreTypeRef(ref, files[ref.File].tokens, b.finalEOF, b.declRows[ref.OwnerDecl])
		if !ok {
			b.setErr(EmitErrCheck, ref.File, ref.Token)
			return false
		}
		b.program.TypeRefs = append(b.program.TypeRefs, mapped)
	}
	return true
}

func (b *coreUnitBuilder) addCheckedFuncs(info check.PackageInfo, files []coreFileTokens) bool {
	b.funcRows = make([]int, len(info.CoreBodies))
	for i := 0; i < len(b.funcRows); i++ {
		b.funcRows[i] = -1
	}
	for i := 0; i < len(info.CoreBodies); i++ {
		body := info.CoreBodies[i]
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
		if !b.addFunc(file, fn, files[body.File].tokens, files[body.File].textBase, body.File) {
			return false
		}
		ownerIndex := len(b.program.Funcs) - 1
		b.funcRows[i] = ownerIndex
		if !b.addBodyCalls(body, files[body.File].tokens, ownerIndex) {
			return false
		}
		if !b.addBodyResolution(body, files[body.File].tokens, ownerIndex) {
			return false
		}
		if !b.addBodyTypeRefs(body, files[body.File].tokens, ownerIndex) {
			return false
		}
	}
	return true
}

func (b *coreUnitBuilder) addCheckedSymbols(info check.PackageInfo, files []coreFileTokens) bool {
	symbols := info.Symbols
	for i := 0; i < len(symbols); i++ {
		symbolFile := symbols[i].File
		symbolToken := symbols[i].Token
		if symbolFile < 0 || symbolFile >= len(files) {
			b.setErr(EmitErrCheck, -1, symbolToken)
			return false
		}
		token := mapCoreToken(files[symbolFile].tokens, symbolToken, b.finalEOF)
		if token < 0 {
			b.setErr(EmitErrCheck, symbolFile, symbolToken)
			return false
		}
		var out unit.Symbol
		out.Name = cloneCoreString(symbols[i].Name)
		out.Package = symbols[i].Package
		out.Token = token
		b.program.Symbols = append(b.program.Symbols, out)
	}
	return true
}

func cloneCoreString(value string) string {
	data := make([]byte, len(value))
	copy(data, []byte(value))
	return string(data)
}

func (b *coreUnitBuilder) addDeclCalls(decl check.DeclInfo, mapping coreTokenMap, ownerIndex int) bool {
	return true
}

func (b *coreUnitBuilder) addDeclResolution(decl check.DeclInfo, mapping coreTokenMap, ownerIndex int) bool {
	for i := 0; i < len(decl.CoreRefs); i++ {
		ref, ok := mapCoreNameRef(decl.CoreRefs[i], mapping, b.finalEOF, ownerIndex)
		if !ok {
			b.setErr(EmitErrCheck, decl.File, decl.Token)
			return false
		}
		b.program.Refs = append(b.program.Refs, ref)
	}
	for i := 0; i < len(decl.CoreSelectors); i++ {
		selector, ok := mapCoreSelectorRef(decl.CoreSelectors[i], mapping, b.finalEOF, ownerIndex)
		if !ok {
			b.setErr(EmitErrCheck, decl.File, decl.Token)
			return false
		}
		b.program.Selectors = append(b.program.Selectors, selector)
	}
	return true
}

func (b *coreUnitBuilder) addBodyResolution(body check.CoreFuncBody, mapping coreTokenMap, ownerIndex int) bool {
	for i := 0; i < len(body.CoreRefs); i++ {
		ref, ok := mapCoreNameRef(body.CoreRefs[i], mapping, b.finalEOF, ownerIndex)
		if !ok {
			b.setErr(EmitErrCheck, body.File, body.ErrorToken)
			return false
		}
		b.program.Refs = append(b.program.Refs, ref)
	}
	for i := 0; i < len(body.CoreSelectors); i++ {
		selector, ok := mapCoreSelectorRef(body.CoreSelectors[i], mapping, b.finalEOF, ownerIndex)
		if !ok {
			b.setErr(EmitErrCheck, body.File, body.ErrorToken)
			return false
		}
		b.program.Selectors = append(b.program.Selectors, selector)
	}
	return true
}

func (b *coreUnitBuilder) addBodyCalls(body check.CoreFuncBody, mapping coreTokenMap, ownerIndex int) bool {
	return true
}

func (b *coreUnitBuilder) addBodyTypeRefs(body check.CoreFuncBody, mapping coreTokenMap, ownerIndex int) bool {
	for i := 0; i < len(body.CoreTypeRefs); i++ {
		if body.CoreTypeRefs[i].File != body.File {
			b.setErr(EmitErrCheck, body.File, body.ErrorToken)
			return false
		}
		ref, ok := mapCoreTypeRef(body.CoreTypeRefs[i], mapping, b.finalEOF, ownerIndex)
		if !ok {
			b.setErr(EmitErrCheck, body.File, body.ErrorToken)
			return false
		}
		b.program.TypeRefs = append(b.program.TypeRefs, ref)
	}
	return true
}

func mapCoreTypeRef(ref check.CoreTypeRef, mapping coreTokenMap, eof int, ownerIndex int) (unit.TypeRef, bool) {
	kind, ok := coreUnitTypeRefKind(ref.Kind)
	if !ok {
		return unit.TypeRef{}, false
	}
	out := unit.TypeRef{
		Kind:    kind,
		Token:   mapCoreToken(mapping, ref.Token, eof),
		BaseTok: mapCoreToken(mapping, ref.BaseTok, eof),
		DotTok:  mapCoreToken(mapping, ref.DotTok, eof),
		Package: ref.Package,
		Symbol:  ref.Symbol,
	}
	if ownerIndex < 0 || out.Token < 0 || out.BaseTok < 0 || out.DotTok < 0 || out.Package < -1 || out.Symbol < -1 {
		return out, false
	}
	return out, true
}

func coreUnitTypeRefKind(kind int) (int, bool) {
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

func mapCoreCallRef(call check.CallRef, mapping coreTokenMap, eof int) (unit.Call, bool) {
	out := unit.Call{
		Kind:      call.Kind,
		CalleeTok: mapCoreToken(mapping, call.CalleeToken, eof),
		BaseTok:   mapCoreToken(mapping, call.BaseToken, eof),
		DotTok:    mapCoreToken(mapping, call.DotToken, eof),
	}
	if out.CalleeTok < 0 || out.BaseTok < 0 || out.DotTok < 0 {
		return out, false
	}
	return out, true
}

func mapCoreNameRef(ref check.CoreNameRef, mapping coreTokenMap, eof int, ownerIndex int) (unit.NameRef, bool) {
	out := unit.NameRef{
		Kind:    unit.RefPackage,
		Token:   mapCoreToken(mapping, ref.Token, eof),
		Index:   ref.Index,
		Package: ref.Package,
	}
	if ownerIndex < 0 || out.Token < 0 || out.Index < -1 || out.Package < -1 {
		return out, false
	}
	return out, true
}

func mapCoreSelectorRef(selector check.CoreSelectorRef, mapping coreTokenMap, eof int, ownerIndex int) (unit.Selector, bool) {
	out := unit.Selector{
		BaseTok:     mapCoreToken(mapping, selector.BaseTok, eof),
		DotTok:      mapCoreToken(mapping, selector.DotTok, eof),
		NameTok:     mapCoreToken(mapping, selector.NameTok, eof),
		BaseKind:    unit.RefImport,
		BaseIndex:   selector.BaseIndex,
		BasePackage: selector.BasePackage,
		Package:     selector.BasePackage,
		Symbol:      selector.Symbol,
	}
	if ownerIndex < 0 || out.BaseTok < 0 || out.DotTok < 0 || out.NameTok < 0 ||
		out.BaseIndex < -1 || out.BasePackage < -1 || out.Package < -1 || out.Symbol < -1 {
		return out, false
	}
	return out, true
}

func findCoreFileDecl(file syntax.File, nameTok int) syntax.TopDecl {
	for i := 0; i < len(file.Decls); i++ {
		if file.Decls[i].NameTok == nameTok {
			return file.Decls[i]
		}
	}
	return syntax.TopDecl{NameTok: -1}
}

func findCoreFileImport(file syntax.File, tok int) syntax.ImportDecl {
	for i := 0; i < len(file.Imports); i++ {
		imp := file.Imports[i]
		if imp.NameTok == tok || imp.PathTok == tok {
			return imp
		}
	}
	return syntax.ImportDecl{PathTok: -1}
}

func (b *coreUnitBuilder) setErr(err int, file int, tok int) {
	b.err = err
	b.errFile = file
	b.errToken = tok
}

func coreUnitTokenKind(src []byte, tok syntax.Token) (int, bool) {
	kind := tok.Kind
	if kind >= syntax.TokenEOF && kind <= syntax.TokenIdent {
		return kind, true
	}
	if kind == syntax.TokenNumber {
		if syntax.NumberTokenIsFloat(src, tok) {
			return unit.TokenFloat, true
		}
		return unit.TokenNumber, true
	}
	if kind >= syntax.TokenString && kind <= syntax.TokenChar {
		return kind + 1, true
	}
	if kind == syntax.TokenOperator {
		return unit.TokenOp, true
	}
	if kind == syntax.TokenPackage {
		return unit.TokenPackage, true
	}
	if kind >= syntax.TokenConst && kind <= syntax.TokenStruct {
		return kind - 1, true
	}
	if kind >= syntax.TokenReturn && kind <= syntax.TokenFor {
		return kind - 3, true
	}
	if kind >= syntax.TokenSwitch && kind <= syntax.TokenDefault {
		return kind - 1, true
	}
	if kind >= syntax.TokenBreak && kind <= syntax.TokenGoto {
		return kind - 7, true
	}
	return unit.TokenIdent, true
}

func coreUnitDeclKind(kind int) (int, bool) {
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

func mapCoreToken(mapping coreTokenMap, tok int, eof int) int {
	if tok < 0 || tok >= mapping.limit {
		return eof
	}
	return mapping.base + tok
}

func mapNullableCoreToken(tok int, mapping coreTokenMap, eof int) int {
	if tok < 0 {
		return -1
	}
	return mapCoreToken(mapping, tok, eof)
}

func countCoreNewlines(src []byte) int {
	count := 0
	for i := 0; i < len(src); i++ {
		if src[i] == '\n' {
			count++
		}
	}
	return count
}
