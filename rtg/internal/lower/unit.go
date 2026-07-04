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
	if !builder.addCheckedDecls(info, files) {
		return emitFail(result, builder.err, builder.errFile, builder.errToken)
	}
	if !builder.addCheckedFuncs(info, files) {
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

func (b *unitBuilder) addCheckedDecls(info check.PackageInfo, files []fileTokens) bool {
	if len(info.DeclOrder) != len(info.Decls) {
		b.setErr(EmitErrCheck, -1, -1)
		return false
	}
	seen := make([]bool, len(info.Decls))
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
		if !b.addDeclShapes(declInfo, files[declInfo.File].oldToNew, ownerIndex) {
			return false
		}
		if !b.addDeclCalls(declInfo, files[declInfo.File].oldToNew, ownerIndex) {
			return false
		}
	}
	return true
}

func (b *unitBuilder) addCheckedFuncs(info check.PackageInfo, files []fileTokens) bool {
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
		if !b.addBodyShapes(body, files[body.File].oldToNew, ownerIndex) {
			return false
		}
		if !b.addBodyFlow(body, files[body.File].oldToNew, ownerIndex) {
			return false
		}
		if !b.addBodyCalls(body, files[body.File].oldToNew, ownerIndex) {
			return false
		}
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

func findFileDecl(file syntax.File, nameTok int) syntax.TopDecl {
	for i := 0; i < len(file.Decls); i++ {
		if file.Decls[i].NameTok == nameTok {
			return file.Decls[i]
		}
	}
	return syntax.TopDecl{NameTok: -1}
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

func mapToken(oldToNew []int, tok int, eof int) int {
	if tok < 0 || tok >= len(oldToNew) {
		return eof
	}
	return oldToNew[tok]
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
