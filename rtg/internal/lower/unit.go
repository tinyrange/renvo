//go:build rtg

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
	UnitError  int
	UnitIndex  int
	UnitDetail int
	UnitA      int
	UnitB      int
	UnitC      int
	ErrorFile  int
	ErrorToken int
}

func EmitCheckedPackageFast(pkg load.Package, info check.PackageInfo) Result {
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
	builder.reserveCheckedPackage(pkg, info)
	files := make([]fileTokens, len(pkg.Files))
	for i := 0; i < len(pkg.Files); i++ {
		if !pkg.Files[i].File.Ok {
			return emitFail(result, EmitErrPackage, i, pkg.Files[i].File.ErrorTok)
		}
		file := pkg.Files[i].File
		src := pkg.Files[i].Src
		tokens, ok := builder.addFileTokens(file, src, i, i+1 < len(pkg.Files))
		if !ok {
			return emitFail(result, builder.err, builder.errFile, builder.errToken)
		}
		files[i] = fileTokens{file: file, src: src, tokens: tokens}
	}
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
	file   syntax.File
	src    []byte
	tokens tokenMap
}

type tokenMap struct {
	base  int
	limit int
}

func (b *unitBuilder) reserveCheckedPackage(pkg load.Package, info check.PackageInfo) {
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
		callCap += len(decl.Calls)
		refCap += len(decl.CoreRefs)
		selectorCap += len(decl.CoreSelectors)
	}
	for i := 0; i < len(info.Bodies); i++ {
		body := info.Bodies[i]
		typeRefCap += len(body.CoreTypeRefs)
		callCap += len(body.Calls)
		refCap += len(body.CoreRefs)
		selectorCap += len(body.CoreSelectors)
	}
	b.program.Text = make([]byte, 0, textCap)
	b.program.Tokens = make([]unit.Token, 0, tokenCap)
	b.program.Imports = make([]unit.Import, 0, len(info.Imports))
	b.program.Symbols = make([]unit.Symbol, 0, len(info.Symbols))
	b.program.Decls = make([]unit.Decl, 0, len(info.Decls))
	b.program.Funcs = make([]unit.Func, 0, len(info.Bodies))
	b.program.TypeRefs = make([]unit.TypeRef, 0, typeRefCap)
	b.program.Calls = make([]unit.Call, 0, callCap)
	b.program.Refs = make([]unit.NameRef, 0, refCap)
	b.program.Selectors = make([]unit.Selector, 0, selectorCap)
}

func (b *unitBuilder) addFileTokens(file syntax.File, src []byte, fileIndex int, hasNext bool) (tokenMap, bool) {
	base := len(b.program.Text)
	tokenBase := len(b.program.Tokens)
	lineOffset := b.lineOffset
	for i := 0; i < len(file.Tokens); i++ {
		tok := file.Tokens[i]
		if tok.Kind == syntax.TokenEOF {
			continue
		}
		kind, ok := unitTokenKind(src, tok)
		if !ok {
			b.setErr(EmitErrToken, fileIndex, i)
			return tokenMap{}, false
		}
		b.program.Tokens = append(b.program.Tokens, unit.Token{
			Kind:  kind,
			Start: base + tok.Start,
			Size:  tok.End - tok.Start,
			Line:  lineOffset + tok.Line,
		})
	}
	b.program.Text = appendBytes(b.program.Text, src)
	b.lineOffset += countNewlines(src)
	if hasNext && (len(src) == 0 || src[len(src)-1] != '\n') {
		b.program.Text = append(b.program.Text, '\n')
		b.lineOffset++
	}
	return tokenMap{base: tokenBase, limit: len(file.Tokens)}, true
}

func appendBytes(out []byte, data []byte) []byte {
	for i := 0; i < len(data); i++ {
		out = append(out, data[i])
	}
	return out
}

func (b *unitBuilder) finishUnit() bool {
	line := b.lineOffset + 1
	b.program.Tokens = append(b.program.Tokens, unit.Token{
		Kind:  unit.TokenEOF,
		Start: len(b.program.Text),
		Size:  0,
		Line:  line,
	})
	return true
}

func (b *unitBuilder) addDecl(file syntax.File, decl syntax.TopDecl, nameTok int, mapping tokenMap, fileIndex int) bool {
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
		StartTok:  mapDeclStartToken(file, decl, mapping, b.finalEOF),
		EndTok:    mapToken(mapping, decl.EndTok, b.finalEOF),
	})
	return true
}

func mapDeclStartToken(file syntax.File, decl syntax.TopDecl, mapping tokenMap, eof int) int {
	start := decl.StartTok
	if start > 0 && start < len(file.Tokens) && file.Tokens[start-1].Kind == decl.Kind {
		start--
	}
	return mapToken(mapping, start, eof)
}

func (b *unitBuilder) addFunc(file syntax.File, fn syntax.FuncDecl, mapping tokenMap, fileIndex int) bool {
	nameTok := mapToken(mapping, fn.NameTok, b.finalEOF)
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
		StartTok:      mapToken(mapping, fn.StartTok, b.finalEOF),
		NameTok:       nameTok,
		ReceiverStart: mapToken(mapping, fn.ReceiverStart, b.finalEOF),
		ReceiverEnd:   mapToken(mapping, fn.ReceiverEnd, b.finalEOF),
		BodyStart:     mapToken(mapping, fn.BodyStart, b.finalEOF),
		BodyEnd:       mapToken(mapping, bodyEnd, b.finalEOF),
		EndTok:        mapToken(mapping, fn.EndTok, b.finalEOF),
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
		b.program.Imports = append(b.program.Imports, unit.Import{
			NameTok: mapNullableToken(decl.NameTok, files[imp.File].tokens, b.finalEOF),
			PathTok: mapToken(files[imp.File].tokens, decl.PathTok, b.finalEOF),
		})
	}
	return true
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
		nameTok := mapToken(files[declInfo.File].tokens, decl.NameTok, b.finalEOF)
		if !b.addDecl(file, decl, nameTok, files[declInfo.File].tokens, declInfo.File) {
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

func (b *unitBuilder) addCheckedTypeRefs(info check.PackageInfo, files []fileTokens) bool {
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
		if !b.addFunc(file, fn, files[body.File].tokens, body.File) {
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

func (b *unitBuilder) addCheckedSymbols(info check.PackageInfo, files []fileTokens) bool {
	symbols := info.Symbols
	for i := 0; i < len(symbols); i++ {
		symbolFile := symbols[i].File
		symbolToken := symbols[i].Token
		if symbolFile < 0 || symbolFile >= len(files) {
			b.setErr(EmitErrCheck, -1, symbolToken)
			return false
		}
		token := mapToken(files[symbolFile].tokens, symbolToken, b.finalEOF)
		if token < 0 {
			b.setErr(EmitErrCheck, symbolFile, symbolToken)
			return false
		}
		var out unit.Symbol
		out.Name = symbols[i].Name
		out.Package = symbols[i].Package
		out.Token = token
		b.program.Symbols = append(b.program.Symbols, out)
	}
	return true
}

func (b *unitBuilder) addDeclCalls(decl check.DeclInfo, mapping tokenMap, ownerIndex int) bool {
	for i := 0; i < len(decl.Calls); i++ {
		call, ok := mapCallRef(decl.Calls[i], mapping, b.finalEOF)
		if !ok || ownerIndex < 0 {
			b.setErr(EmitErrCheck, decl.File, decl.Token)
			return false
		}
		b.program.Calls = append(b.program.Calls, call)
	}
	return true
}

func (b *unitBuilder) addDeclResolution(decl check.DeclInfo, mapping tokenMap, ownerIndex int) bool {
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

func (b *unitBuilder) addBodyResolution(body check.FuncBody, mapping tokenMap, ownerIndex int) bool {
	for i := 0; i < len(body.CoreRefs); i++ {
		ref, ok := mapCoreNameRef(body.CoreRefs[i], mapping, b.finalEOF, ownerIndex)
		if !ok {
			b.setErr(EmitErrCheck, body.File, body.Body.ErrorTok)
			return false
		}
		b.program.Refs = append(b.program.Refs, ref)
	}
	for i := 0; i < len(body.CoreSelectors); i++ {
		selector, ok := mapCoreSelectorRef(body.CoreSelectors[i], mapping, b.finalEOF, ownerIndex)
		if !ok {
			b.setErr(EmitErrCheck, body.File, body.Body.ErrorTok)
			return false
		}
		b.program.Selectors = append(b.program.Selectors, selector)
	}
	return true
}

func (b *unitBuilder) addBodyCalls(body check.FuncBody, mapping tokenMap, ownerIndex int) bool {
	for i := 0; i < len(body.Calls); i++ {
		call, ok := mapCallRef(body.Calls[i], mapping, b.finalEOF)
		if !ok || ownerIndex < 0 {
			b.setErr(EmitErrCheck, body.File, body.Body.ErrorTok)
			return false
		}
		b.program.Calls = append(b.program.Calls, call)
	}
	return true
}

func (b *unitBuilder) addBodyTypeRefs(body check.FuncBody, mapping tokenMap, ownerIndex int) bool {
	for i := 0; i < len(body.CoreTypeRefs); i++ {
		if body.CoreTypeRefs[i].File != body.File {
			b.setErr(EmitErrCheck, body.File, body.Body.ErrorTok)
			return false
		}
		ref, ok := mapCoreTypeRef(body.CoreTypeRefs[i], mapping, b.finalEOF, ownerIndex)
		if !ok {
			b.setErr(EmitErrCheck, body.File, body.Body.ErrorTok)
			return false
		}
		b.program.TypeRefs = append(b.program.TypeRefs, ref)
	}
	return true
}

func mapCoreTypeRef(ref check.CoreTypeRef, mapping tokenMap, eof int, ownerIndex int) (unit.TypeRef, bool) {
	kind, ok := unitTypeRefKind(ref.Kind)
	if !ok {
		return unit.TypeRef{}, false
	}
	out := unit.TypeRef{
		Kind:    kind,
		Token:   mapToken(mapping, ref.Token, eof),
		BaseTok: mapToken(mapping, ref.BaseTok, eof),
		DotTok:  mapToken(mapping, ref.DotTok, eof),
		Package: ref.Package,
		Symbol:  ref.Symbol,
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

func mapCallRef(call check.CallRef, mapping tokenMap, eof int) (unit.Call, bool) {
	out := unit.Call{
		Kind:      call.Kind,
		CalleeTok: mapToken(mapping, call.CalleeToken, eof),
		BaseTok:   mapToken(mapping, call.BaseToken, eof),
		DotTok:    mapToken(mapping, call.DotToken, eof),
	}
	if out.CalleeTok < 0 || out.BaseTok < 0 || out.DotTok < 0 {
		return out, false
	}
	return out, true
}

func mapCoreNameRef(ref check.CoreNameRef, mapping tokenMap, eof int, ownerIndex int) (unit.NameRef, bool) {
	out := unit.NameRef{
		Kind:    ref.Kind,
		Token:   mapToken(mapping, ref.Token, eof),
		Index:   ref.Index,
		Package: ref.Package,
	}
	if ownerIndex < 0 || out.Token < 0 || out.Index < -1 || out.Package < -1 {
		return out, false
	}
	return out, true
}

func mapCoreSelectorRef(selector check.CoreSelectorRef, mapping tokenMap, eof int, ownerIndex int) (unit.Selector, bool) {
	out := unit.Selector{
		BaseTok:     mapToken(mapping, selector.BaseTok, eof),
		DotTok:      mapToken(mapping, selector.DotTok, eof),
		NameTok:     mapToken(mapping, selector.NameTok, eof),
		BaseKind:    selector.BaseKind,
		BaseIndex:   selector.BaseIndex,
		BasePackage: selector.BasePackage,
		Package:     selector.Package,
		Symbol:      selector.Symbol,
	}
	if ownerIndex < 0 || out.BaseTok < 0 || out.DotTok < 0 || out.NameTok < 0 ||
		out.BaseIndex < -1 || out.BasePackage < -1 || out.Package < -1 || out.Symbol < -1 {
		return out, false
	}
	return out, true
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

func mapToken(mapping tokenMap, tok int, eof int) int {
	if tok < 0 || tok >= mapping.limit {
		return eof
	}
	if tok == mapping.limit-1 {
		return eof
	}
	return mapping.base + tok
}

func mapNullableToken(tok int, mapping tokenMap, eof int) int {
	if tok < 0 {
		return -1
	}
	return mapToken(mapping, tok, eof)
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
	start := tok.Start
	end := tok.End
	if start < 0 {
		start = 0
	}
	if end > len(src) {
		end = len(src)
	}
	if end-start > 2 && src[start] == '0' {
		prefix := src[start+1]
		if prefix == 'x' || prefix == 'X' {
			for i := start + 2; i < end; i++ {
				c := src[i]
				if c == '.' || c == 'p' || c == 'P' {
					return true
				}
			}
			return false
		}
		if prefix == 'b' || prefix == 'B' || prefix == 'o' || prefix == 'O' {
			return false
		}
	}
	for i := start; i < end; i++ {
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
