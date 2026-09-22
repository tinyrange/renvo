package link

import (
	"renvo.dev/internal/syntax"
	"renvo.dev/internal/unit"
)

type callTokenEdit struct {
	start  int
	end    int
	offset int
	text   []byte
	tokens []syntax.Token
}

// rewriteBuiltinCalls preserves declaration structure while replacing complete
// call expressions. Only replacement expressions and appended helpers need to
// be scanned; copying existing tokens avoids reparsing the whole linked program.
func rewriteBuiltinCalls(original *unit.Program, text []byte, edits []functionValueEdit, originalLength int, generatedStart int) bool {
	var changes []callTokenEdit
	count := len(original.Tokens) - 1
	cursor, delta := 0, 0
	for _, edit := range edits {
		for cursor < len(original.Tokens)-1 && original.Tokens[cursor].Start < edit.start {
			cursor++
		}
		start := cursor
		for cursor < len(original.Tokens)-1 && original.Tokens[cursor].Start < edit.end {
			cursor++
		}
		fragment := []byte(edit.text)
		tokens := syntax.Scan(fragment)
		if len(tokens) == 0 || tokens[len(tokens)-1].KindLine&255 != syntax.TokenEOF {
			return false
		}
		if start < cursor && (original.Tokens[start].Start != edit.start || original.Tokens[cursor-1].Start+original.Tokens[cursor-1].Size != edit.end) {
			return false
		}
		changes = append(changes, callTokenEdit{start: start, end: cursor, offset: edit.start + delta, text: fragment, tokens: tokens})
		count += len(tokens) - 1 - (cursor - start)
		for _, tok := range tokens {
			if functionValueTokenIsEllipsis(fragment, tok) {
				count += 2
			}
		}
		delta += len(edit.text) - (edit.end - edit.start)
	}
	var helper unit.Program
	helperPrefix := len("package main\n")
	if generatedStart >= 0 && generatedStart < len(text) {
		source := make([]byte, 0, helperPrefix+len(text)-generatedStart)
		source = append(source, "package main\n"...)
		source = append(source, text[generatedStart:]...)
		if !reparseFunctionValueProgram(&helper, source, nil, len(source), -1) {
			return false
		}
		count += len(helper.Tokens) - 3
	}
	out := unit.Program{Package: original.Package, ImportPath: original.ImportPath, Text: text}
	out.Tokens = make([]unit.Token, 0, count+1)
	out.Decls = make([]unit.Decl, 0, len(original.Decls)+len(helper.Decls))
	out.Funcs = make([]unit.Func, 0, len(original.Funcs)+len(helper.Funcs))
	tokenMap := make([]int, len(original.Tokens)+1)
	cursor, delta = 0, 0
	for _, change := range changes {
		for cursor < change.start {
			tokenMap[cursor] = len(out.Tokens)
			tok := original.Tokens[cursor]
			tok.Start += delta
			out.Tokens = append(out.Tokens, tok)
			cursor++
		}
		for cursor < change.end {
			tokenMap[cursor] = len(out.Tokens)
			cursor++
		}
		for _, tok := range change.tokens[:len(change.tokens)-1] {
			kind := functionValueUnitTokenKind(change.text, tok)
			if functionValueTokenIsEllipsis(change.text, tok) {
				for dot := 0; dot < 3; dot++ {
					out.Tokens = append(out.Tokens, unit.MakeToken(kind, change.offset+syntax.TokenStart(tok)+dot, 1, 1))
				}
			} else {
				out.Tokens = append(out.Tokens, unit.MakeToken(kind, change.offset+syntax.TokenStart(tok), syntax.TokenSize(tok), 1))
			}
		}
		if change.end > change.start {
			delta = change.offset + len(change.text) - (original.Tokens[change.end-1].Start + original.Tokens[change.end-1].Size)
		}
	}
	for cursor < len(original.Tokens)-1 {
		tokenMap[cursor] = len(out.Tokens)
		tok := original.Tokens[cursor]
		tok.Start += delta
		out.Tokens = append(out.Tokens, tok)
		cursor++
	}
	tokenMap[cursor] = len(out.Tokens)
	tokenMap[cursor+1] = len(out.Tokens)
	for _, decl := range original.Decls {
		decl.NameStart = mapFunctionValueOffset(decl.NameStart, edits, originalLength)
		decl.NameEnd = mapFunctionValueOffset(decl.NameEnd, edits, originalLength)
		decl.StartTok = tokenMap[decl.StartTok]
		decl.EndTok = tokenMap[decl.EndTok]
		out.Decls = append(out.Decls, decl)
	}
	for _, fn := range original.Funcs {
		fn.NameStart = mapFunctionValueOffset(fn.NameStart, edits, originalLength)
		fn.NameEnd = mapFunctionValueOffset(fn.NameEnd, edits, originalLength)
		fn.StartTok = tokenMap[fn.StartTok]
		fn.NameTok = tokenMap[fn.NameTok]
		if fn.ReceiverStart != fn.ReceiverEnd {
			fn.ReceiverStart = tokenMap[fn.ReceiverStart]
			fn.ReceiverEnd = tokenMap[fn.ReceiverEnd]
		}
		fn.BodyStart = tokenMap[fn.BodyStart]
		fn.BodyEnd = tokenMap[fn.BodyEnd]
		fn.EndTok = tokenMap[fn.EndTok]
		out.Funcs = append(out.Funcs, fn)
	}
	if len(helper.Tokens) > 0 {
		tokenBase := len(out.Tokens) - 2
		textBase := generatedStart - helperPrefix
		for _, tok := range helper.Tokens[2 : len(helper.Tokens)-1] {
			tok.Start += textBase
			out.Tokens = append(out.Tokens, tok)
		}
		for _, decl := range helper.Decls {
			decl.NameStart += textBase
			decl.NameEnd += textBase
			decl.StartTok += tokenBase
			decl.EndTok += tokenBase
			out.Decls = append(out.Decls, decl)
		}
		for _, fn := range helper.Funcs {
			fn.NameStart += textBase
			fn.NameEnd += textBase
			fn.StartTok += tokenBase
			fn.NameTok += tokenBase
			if fn.ReceiverStart != fn.ReceiverEnd {
				fn.ReceiverStart += tokenBase
				fn.ReceiverEnd += tokenBase
			}
			fn.BodyStart += tokenBase
			fn.BodyEnd += tokenBase
			fn.EndTok += tokenBase
			out.Funcs = append(out.Funcs, fn)
		}
	}
	eof := len(out.Tokens)
	out.Tokens = append(out.Tokens, unit.MakeToken(unit.TokenEOF, len(text), 0, 1))
	for i := 0; i < len(out.Funcs); i++ {
		if out.Funcs[i].ReceiverStart == out.Funcs[i].ReceiverEnd {
			out.Funcs[i].ReceiverStart = eof
			out.Funcs[i].ReceiverEnd = eof
		}
	}
	line, offset := 1, 0
	for i := 0; i < len(out.Tokens); i++ {
		for offset < out.Tokens[i].Start {
			if text[offset] == '\n' {
				line++
			}
			offset++
		}
		out.Tokens[i].KindLine = out.Tokens[i].KindLine&255 | line<<8
	}
	out.Packages = remapFunctionValuePackages(original, &out, edits, originalLength, generatedStart)
	replaceFunctionValueProgram(original, &out)
	return true
}
