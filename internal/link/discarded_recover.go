package link

import "renvo.dev/internal/syntax"
import "renvo.dev/internal/unit"

// Recover may appear as an expression statement. Store its result in the blank
// target in the same frame: calling a wrapper would change direct recovery.
// Parse only bodies containing the builtin spelling, and select actual simple
// statements rather than similar text in interface declarations or composites.
func lowerDiscardedRecoverCalls(program *unit.Program, transient bool) bool {
	var seen []int
	var edits []functionValueEdit
	topLevel := ordinaryBuiltinTopLevelObject(program, "recover")
	if topLevel {
		return true
	}
	for tok := 0; tok+2 < len(program.Tokens); tok++ {
		if !functionValueTokenEquals(program, tok, "recover") || !functionValueTokenEquals(program, tok+1, "(") || !functionValueTokenEquals(program, tok+2, ")") {
			continue
		}
		fn, ok := functionRangeDeferLexicalOwner(program, tok)
		if !ok {
			continue
		}
		found := false
		for _, open := range seen {
			if open == fn.BodyStart {
				found = true
			}
		}
		if found {
			continue
		}
		seen = append(seen, fn.BodyStart)
		base := program.Tokens[fn.BodyStart].Start
		last := program.Tokens[fn.BodyEnd-1]
		source := program.Text[base : last.Start+last.Size]
		file := syntax.File{Src: source, Tokens: syntax.Scan(source)}
		body := syntax.ParseFuncBodyStatements(file, syntax.FuncDecl{BodyStart: 0, BodyEnd: len(file.Tokens) - 1})
		if !body.Ok {
			return false
		}
		for _, stmt := range body.Stmts {
			start, end := stmt.ExprStart, stmt.ExprEnd
			if stmt.Kind == syntax.StmtExpr {
				edits = appendDiscardedRecoverEdit(program, &file, base, start, end, edits)
			} else if stmt.Kind == syntax.StmtIf || stmt.Kind == syntax.StmtFor || stmt.Kind == syntax.StmtSwitch {
				// Initializers and a for-clause post are simple statements too.
				initEnd := discardedRecoverInitializerEnd(&file, start, end)
				edits = appendDiscardedRecoverEdit(program, &file, base, start, initEnd, edits)
				if stmt.Kind == syntax.StmtFor && initEnd >= 0 {
					conditionEnd := discardedRecoverInitializerEnd(&file, initEnd+1, end)
					if conditionEnd >= 0 {
						edits = appendDiscardedRecoverEdit(program, &file, base, conditionEnd+1, end, edits)
					}
				}
			}
		}
	}
	if len(edits) == 0 {
		return true
	}
	return functionDeferApplyEdits(program, edits, transient)
}

func appendDiscardedRecoverEdit(program *unit.Program, file *syntax.File, base int, start int, end int, edits []functionValueEdit) []functionValueEdit {
	call := discardedRecoverExpression(file, start, end)
	if call < 0 {
		return edits
	}
	offset := base + syntax.TokenStart(file.Tokens[call])
	original := discardedRecoverTokenAt(program, offset)
	if original < 0 || ordinaryBuiltinShadowedWithTopLevel(program, original, "recover", false) {
		return edits
	}
	first := base + syntax.TokenStart(file.Tokens[start])
	last := base + syntax.TokenEnd(file.Tokens[end-1])
	return append(edits, functionValueEdit{start: first, end: last, text: "_ = " + string(program.Text[first:last])})
}

func discardedRecoverExpression(file *syntax.File, start int, end int) int {
	if start < 0 || end > len(file.Tokens) {
		return -1
	}
	for end-start > 3 && discardedRecoverTokenEquals(file, start, "(") && discardedRecoverTokenEquals(file, end-1, ")") {
		depth := 0
		for at := start; at < end; at++ {
			if discardedRecoverTokenEquals(file, at, "(") {
				depth++
			}
			if discardedRecoverTokenEquals(file, at, ")") {
				depth--
			}
			if depth == 0 && at != end-1 {
				return -1
			}
		}
		start++
		end--
	}
	if end-start == 3 && discardedRecoverTokenEquals(file, start, "recover") && discardedRecoverTokenEquals(file, start+1, "(") && discardedRecoverTokenEquals(file, start+2, ")") {
		return start
	}
	return -1
}

func discardedRecoverInitializerEnd(file *syntax.File, start int, end int) int {
	if start < 0 {
		return -1
	}
	depth := 0
	for at := start; at < end; at++ {
		text := string(syntax.TokenText(file.Src, file.Tokens[at]))
		if text == "(" || text == "[" || text == "{" {
			depth++
		}
		if text == ")" || text == "]" || text == "}" {
			depth--
		}
		if depth == 0 && text == ";" {
			return at
		}
	}
	return -1
}

func discardedRecoverTokenEquals(file *syntax.File, tok int, text string) bool {
	return tok >= 0 && tok < len(file.Tokens) && string(syntax.TokenText(file.Src, file.Tokens[tok])) == text
}

func discardedRecoverTokenAt(program *unit.Program, offset int) int {
	low, high := 0, len(program.Tokens)
	for low < high {
		mid := low + (high-low)/2
		if program.Tokens[mid].Start < offset {
			low = mid + 1
		} else {
			high = mid
		}
	}
	if low < len(program.Tokens) && program.Tokens[low].Start == offset {
		return low
	}
	return -1
}
