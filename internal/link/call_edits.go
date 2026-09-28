package link

import (
	"renvo.dev/internal/syntax"
	"renvo.dev/internal/unit"
)

type callTokenEdit struct {
	start     int
	end       int
	offset    int
	text      []byte
	tokens    []syntax.Token
	delta     int
	line      int
	lineDelta int
}

// rewriteBuiltinCalls preserves declaration structure while replacing complete
// call expressions. Only replacement expressions and appended helpers need to
// be scanned; copying existing tokens avoids reparsing the whole linked program.
func rewriteBuiltinCalls(original *unit.Program, text []byte, edits []functionValueEdit, originalLength int, generatedStart int, transient bool) bool {
	var changes []callTokenEdit
	count := len(original.Tokens) - 1
	maxGrowth := 0
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
		startLine := original.Tokens[start].KindLine >> 8
		endLine := startLine
		if cursor > start {
			endLine = original.Tokens[cursor-1].KindLine >> 8
		}
		// Edits replace calls (ending in a closing parenthesis) or package
		// clauses (ending in an identifier), so the last token has no newline.
		newLines := 0
		for _, ch := range fragment {
			if ch == '\n' {
				newLines++
			}
		}
		changes = append(changes, callTokenEdit{start: start, end: cursor, offset: edit.start + delta, text: fragment, tokens: tokens, line: startLine, lineDelta: newLines - (endLine - startLine)})
		count += len(tokens) - 1 - (cursor - start)
		for _, tok := range tokens {
			if functionValueTokenIsEllipsis(fragment, tok) {
				count += 2
			}
		}
		if growth := count - (len(original.Tokens) - 1); growth > maxGrowth {
			maxGrowth = growth
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
	sourceTokens := original.Tokens
	if transient && cap(sourceTokens) >= count+1 && cap(sourceTokens) >= len(sourceTokens)+maxGrowth {
		// Reserve the largest cumulative expansion before writing forward. Moving
		// the input right keeps every unread token beyond the output cursor even
		// when individual replacements alternately grow and shrink.
		buffer := sourceTokens[:len(sourceTokens)+maxGrowth]
		copy(buffer[maxGrowth:], sourceTokens)
		sourceTokens = buffer[maxGrowth:]
		out.Tokens = buffer[:0]
	} else {
		out.Tokens = make([]unit.Token, 0, count+1)
	}
	out.Decls = make([]unit.Decl, 0, len(original.Decls)+len(helper.Decls))
	out.Funcs = make([]unit.Func, 0, len(original.Funcs)+len(helper.Funcs))
	cursor, delta = 0, 0
	lineDelta := 0
	originalEndLine := sourceTokens[len(sourceTokens)-1].KindLine >> 8
	for changeIndex := 0; changeIndex < len(changes); changeIndex++ {
		change := &changes[changeIndex]
		for cursor < change.start {
			tok := sourceTokens[cursor]
			tok.Start += delta
			tok.KindLine += lineDelta << 8
			out.Tokens = append(out.Tokens, tok)
			cursor++
		}
		for cursor < change.end {
			cursor++
		}
		nextDelta := delta
		if change.end > change.start {
			nextDelta = change.offset + len(change.text) - (sourceTokens[change.end-1].Start + sourceTokens[change.end-1].Size)
		}
		fragmentLine, fragmentOffset := change.line+lineDelta, 0
		for _, tok := range change.tokens[:len(change.tokens)-1] {
			for fragmentOffset < syntax.TokenStart(tok) {
				if change.text[fragmentOffset] == '\n' {
					fragmentLine++
				}
				fragmentOffset++
			}
			kind := functionValueUnitTokenKind(change.text, tok)
			if functionValueTokenIsEllipsis(change.text, tok) {
				for dot := 0; dot < 3; dot++ {
					out.Tokens = append(out.Tokens, unit.MakeToken(kind, change.offset+syntax.TokenStart(tok)+dot, 1, fragmentLine))
				}
			} else {
				out.Tokens = append(out.Tokens, unit.MakeToken(kind, change.offset+syntax.TokenStart(tok), syntax.TokenSize(tok), fragmentLine))
			}
		}
		change.delta = len(out.Tokens) - change.end
		delta = nextDelta
		lineDelta += change.lineDelta
	}
	for cursor < len(sourceTokens)-1 {
		tok := sourceTokens[cursor]
		tok.Start += delta
		tok.KindLine += lineDelta << 8
		out.Tokens = append(out.Tokens, tok)
		cursor++
	}
	for _, decl := range original.Decls {
		decl.NameStart = mapFunctionValueOffset(decl.NameStart, edits, originalLength)
		decl.NameEnd = mapFunctionValueOffset(decl.NameEnd, edits, originalLength)
		decl.StartTok = mapBuiltinCallToken(decl.StartTok, changes)
		decl.EndTok = mapBuiltinCallToken(decl.EndTok, changes)
		out.Decls = append(out.Decls, decl)
	}
	for _, fn := range original.Funcs {
		fn.NameStart = mapFunctionValueOffset(fn.NameStart, edits, originalLength)
		fn.NameEnd = mapFunctionValueOffset(fn.NameEnd, edits, originalLength)
		fn.StartTok = mapBuiltinCallToken(fn.StartTok, changes)
		fn.NameTok = mapBuiltinCallToken(fn.NameTok, changes)
		if fn.ReceiverStart != fn.ReceiverEnd {
			fn.ReceiverStart = mapBuiltinCallToken(fn.ReceiverStart, changes)
			fn.ReceiverEnd = mapBuiltinCallToken(fn.ReceiverEnd, changes)
		}
		fn.BodyStart = mapBuiltinCallToken(fn.BodyStart, changes)
		fn.BodyEnd = mapBuiltinCallToken(fn.BodyEnd, changes)
		fn.EndTok = mapBuiltinCallToken(fn.EndTok, changes)
		out.Funcs = append(out.Funcs, fn)
	}
	if len(helper.Tokens) > 0 {
		tokenBase := len(out.Tokens) - 2
		textBase := generatedStart - helperPrefix
		for _, tok := range helper.Tokens[2 : len(helper.Tokens)-1] {
			tok.Start += textBase
			tok.KindLine += (originalEndLine + lineDelta - 1) << 8
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
	endLine := originalEndLine + lineDelta
	if len(helper.Tokens) > 0 {
		endLine += (helper.Tokens[len(helper.Tokens)-1].KindLine >> 8) - 1
	}
	out.Tokens = append(out.Tokens, unit.MakeToken(unit.TokenEOF, len(text), 0, endLine))
	for i := 0; i < len(out.Funcs); i++ {
		if out.Funcs[i].ReceiverStart == out.Funcs[i].ReceiverEnd {
			out.Funcs[i].ReceiverStart = eof
			out.Funcs[i].ReceiverEnd = eof
		}
	}
	out.Packages = remapFunctionValuePackages(original, &out, edits, originalLength, generatedStart)
	replaceFunctionValueProgram(original, &out)
	return true
}

// Changes are ordered and disjoint. Only declaration/function boundaries need
// remapping, so retain one cumulative token delta per edit instead of a map
// entry for every token in the linked program.
func mapBuiltinCallToken(index int, changes []callTokenEdit) int {
	lo, hi := 0, len(changes)
	for lo < hi {
		mid := lo + (hi-lo)/2
		if changes[mid].end <= index {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	delta := 0
	if lo > 0 {
		delta = changes[lo-1].delta
	}
	if lo < len(changes) && index >= changes[lo].start {
		return changes[lo].start + delta
	}
	return index + delta
}
