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
		return linkFail(out, LinkErrBuild, result.ErrorPackage)
	}
	if result.Root < 0 || result.Root >= len(result.Units) {
		return linkFail(out, LinkErrRoot, -1)
	}
	program, pkg, ok := LinkUnitData(result.Units, result.Root)
	if !ok {
		return linkFail(out, LinkErrUnit, pkg)
	}
	data, ok := unit.Marshal(program)
	if !ok {
		return linkFail(out, LinkErrUnit, -1)
	}
	out.Program = program
	out.Data = data
	return out
}

func LinkUnitData(units []build.PackageUnit, root int) (unit.Program, int, bool) {
	if root < 0 || root >= len(units) {
		return unit.Program{}, -1, false
	}
	programs := make([]unit.Program, len(units))
	for i := 0; i < len(units); i++ {
		prog, ok := unit.Unmarshal(units[i].Data)
		if !ok {
			return unit.Program{}, i, false
		}
		if units[i].Name != "" && prog.Package != units[i].Name {
			return unit.Program{}, i, false
		}
		programs[i] = prog
	}
	program, ok := LinkPrograms(programs, root, units[root].Name)
	if !ok {
		return unit.Program{}, -1, false
	}
	return program, -1, true
}

func LinkUnits(units []build.PackageUnit, root int) (unit.Program, bool) {
	if root < 0 || root >= len(units) {
		return unit.Program{}, false
	}
	programs := make([]unit.Program, len(units))
	for i := 0; i < len(units); i++ {
		programs[i] = units[i].Program
	}
	return LinkPrograms(programs, root, units[root].Name)
}

func LinkPrograms(programs []unit.Program, root int, rootName string) (unit.Program, bool) {
	if root < 0 || root >= len(programs) || rootName == "" {
		return unit.Program{}, false
	}
	program := unit.Program{Package: rootName}
	finalEOF := countLinkedTokens(programs)
	lineOffset := 0
	for i := 0; i < len(programs); i++ {
		ok := appendProgram(&program, programs[i], finalEOF, lineOffset, i+1 < len(programs))
		if !ok {
			return unit.Program{}, false
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

func appendProgram(dst *unit.Program, src unit.Program, finalEOF int, lineOffset int, hasNext bool) bool {
	if src.Package == "" || len(src.Text) == 0 || len(src.Tokens) == 0 {
		return false
	}
	textOffset := len(dst.Text)
	declOffset := len(dst.Decls)
	funcOffset := len(dst.Funcs)
	oldToNew := make([]int, len(src.Tokens))
	for i := 0; i < len(src.Tokens); i++ {
		tok := src.Tokens[i]
		if tok.Kind == unit.TokenEOF {
			oldToNew[i] = finalEOF
			continue
		}
		oldToNew[i] = len(dst.Tokens)
		tok.Start += textOffset
		tok.Line += lineOffset
		dst.Tokens = append(dst.Tokens, tok)
	}
	dst.Text = append(dst.Text, src.Text...)
	for i := 0; i < len(src.Decls); i++ {
		decl := src.Decls[i]
		decl.NameStart += textOffset
		decl.NameEnd += textOffset
		decl.StartTok = mapToken(oldToNew, decl.StartTok, finalEOF)
		decl.EndTok = mapToken(oldToNew, decl.EndTok, finalEOF)
		dst.Decls = append(dst.Decls, decl)
	}
	for i := 0; i < len(src.Funcs); i++ {
		fn := src.Funcs[i]
		fn.NameStart += textOffset
		fn.NameEnd += textOffset
		fn.StartTok = mapToken(oldToNew, fn.StartTok, finalEOF)
		fn.NameTok = mapToken(oldToNew, fn.NameTok, finalEOF)
		fn.ReceiverStart = mapToken(oldToNew, fn.ReceiverStart, finalEOF)
		fn.ReceiverEnd = mapToken(oldToNew, fn.ReceiverEnd, finalEOF)
		fn.BodyStart = mapToken(oldToNew, fn.BodyStart, finalEOF)
		fn.BodyEnd = mapToken(oldToNew, fn.BodyEnd, finalEOF)
		fn.EndTok = mapToken(oldToNew, fn.EndTok, finalEOF)
		dst.Funcs = append(dst.Funcs, fn)
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
	if hasNext && (len(src.Text) == 0 || src.Text[len(src.Text)-1] != '\n') {
		dst.Text = append(dst.Text, '\n')
	}
	return true
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

func mapExprSpan(span unit.ExprSpan, oldToNew []int, eof int) unit.ExprSpan {
	span.StartTok = mapToken(oldToNew, span.StartTok, eof)
	span.EndTok = mapToken(oldToNew, span.EndTok, eof)
	return span
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

func countLinkedTokens(programs []unit.Program) int {
	count := 0
	for i := 0; i < len(programs); i++ {
		tokens := programs[i].Tokens
		for j := 0; j < len(tokens); j++ {
			if tokens[j].Kind != unit.TokenEOF {
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
