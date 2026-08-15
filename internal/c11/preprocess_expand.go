package c11

func (p *preprocessor) expand(tokens []ppToken, depth int) []ppToken {
	if !p.ok || len(tokens) == 0 {
		return tokens
	}
	if depth >= 128 {
		p.fail(PreprocessErrDepth, p.currentPath, p.currentLine)
		return nil
	}
	capacity := len(tokens)*2 + 16
	work := make([]ppToken, len(tokens), capacity)
	copy(work, tokens)
	for i := 0; i < len(work) && p.ok; {
		tok := work[i]
		if ppTokenKind(tok) != ppIdent {
			i++
			continue
		}
		if p.tokenIs(tok, "__LINE__") {
			text := ppAppendDecimal(nil, p.currentLine)
			work[i] = p.generatedToken(ppNumber, text, ppTokenSpace(tok))
			i++
			continue
		}
		if p.tokenIs(tok, "__FILE__") {
			work[i] = p.generatedString([]byte(p.currentDisplay), ppTokenSpace(tok))
			i++
			continue
		}
		if p.tokenIs(tok, "__BASE_FILE__") {
			work[i] = p.generatedString([]byte(p.config.Path), ppTokenSpace(tok))
			i++
			continue
		}
		if p.tokenIs(tok, "__COUNTER__") {
			text := ppAppendDecimal(nil, p.counter)
			p.counter++
			work[i] = p.generatedToken(ppNumber, text, ppTokenSpace(tok))
			i++
			continue
		}
		index := p.lookupToken(tok)
		if index < 0 || p.hideContains(ppTokenHide(tok), index) {
			i++
			continue
		}
		macro := p.macros[index]
		var replacement []ppToken
		end := i
		if macro.function {
			if i+1 >= len(work) || !p.tokenIs(work[i+1], "(") {
				i++
				continue
			}
			args, close, ok := p.macroArguments(work, i+1, macro)
			if !ok {
				return p.macroFail(macro.name)
			}
			replacement = p.substitute(macro, args, depth+1)
			end = close
		} else {
			replacement = ppCloneTokens(macro.replace)
		}
		if len(replacement) > 0 {
			replacement[0] = ppWithSpace(replacement[0], ppTokenSpace(tok))
		}
		for j := 0; j < len(replacement); j++ {
			replacement[j] = p.addExpansionHide(replacement[j], ppTokenHide(tok), index)
		}
		work = ppReplaceTokens(work, i, end+1, replacement)
	}
	return work
}

func ppReplaceTokens(tokens []ppToken, start int, end int, replacement []ppToken) []ppToken {
	removed := end - start
	newLength := len(tokens) - removed + len(replacement)
	if newLength > cap(tokens) {
		capacity := cap(tokens) * 2
		if capacity < newLength {
			capacity = newLength
		}
		next := make([]ppToken, newLength, capacity)
		copy(next, tokens[:start])
		copy(next[start:], replacement)
		copy(next[start+len(replacement):], tokens[end:])
		return next
	}
	oldLength := len(tokens)
	tokens = tokens[:newLength]
	copy(tokens[start+len(replacement):], tokens[end:oldLength])
	copy(tokens[start:], replacement)
	return tokens
}

func (p *preprocessor) hideContains(hide int, macro int) bool {
	for hide > 0 && hide < len(p.hides) {
		if p.hides[hide].macro == macro {
			return true
		}
		hide = p.hides[hide].next
	}
	return false
}

func (p *preprocessor) addHide(tok ppToken, macro int) ppToken {
	hide := ppTokenHide(tok)
	if p.hideContains(hide, macro) {
		return tok
	}
	p.hides = append(p.hides, ppHide{macro: macro, next: hide})
	return ppWithHide(tok, len(p.hides)-1)
}

func (p *preprocessor) addExpansionHide(tok ppToken, invocationHide int, macro int) ppToken {
	for invocationHide > 0 && invocationHide < len(p.hides) {
		tok = p.addHide(tok, p.hides[invocationHide].macro)
		invocationHide = p.hides[invocationHide].next
	}
	return p.addHide(tok, macro)
}

func (p *preprocessor) macroArguments(tokens []ppToken, open int, macro ppMacro) ([][]ppToken, int, bool) {
	var args [][]ppToken
	depth := 0
	start := open + 1
	for i := open + 1; i < len(tokens); i++ {
		if p.tokenIs(tokens[i], "(") {
			depth++
			continue
		}
		if p.tokenIs(tokens[i], ")") {
			if depth > 0 {
				depth--
				continue
			}
			if i > start || len(args) > 0 || len(macro.params) > 0 {
				args = append(args, ppCloneTokens(tokens[start:i]))
			}
			if !macro.variadic && len(args) != len(macro.params) || macro.variadic && len(args) < len(macro.params)-1 {
				return nil, 0, false
			}
			if macro.variadic {
				fixed := len(macro.params) - 1
				if len(args) == fixed {
					args = append(args, nil)
				} else if len(args) > len(macro.params) {
					joined := ppCloneTokens(args[fixed])
					for j := fixed + 1; j < len(args); j++ {
						joined = append(joined, p.generatedToken(ppPunct, []byte(","), false))
						joined = append(joined, args[j]...)
					}
					args = append(args[:fixed], joined)
				}
			}
			return args, i, true
		}
		if p.tokenIs(tokens[i], ",") && depth == 0 {
			args = append(args, ppCloneTokens(tokens[start:i]))
			start = i + 1
		}
	}
	return nil, 0, false
}

func (p *preprocessor) substitute(macro ppMacro, args [][]ppToken, depth int) []ppToken {
	previousDetail := p.errorDetail
	p.errorDetail = macro.name
	expandedArgs := make([][]ppToken, len(args))
	expandedReady := make([]bool, len(args))
	var out []ppToken
	paste := false
	replacement := macro.replace
	for i := 0; i < len(replacement) && p.ok; i++ {
		tok := replacement[i]
		if p.tokenIs(tok, "#") {
			if i+1 >= len(replacement) {
				return p.macroFail(macro.name)
			}
			param := p.parameterIndex(macro, replacement[i+1])
			if param < 0 {
				return p.macroFail(macro.name)
			}
			piece := []ppToken{p.stringize(args[param], ppTokenSpace(tok))}
			out = p.appendReplacementPiece(out, piece, paste)
			paste = false
			i++
			continue
		}
		if p.tokenIs(tok, "##") {
			if len(out) == 0 || paste {
				return p.macroFail(macro.name)
			}
			paste = true
			continue
		}
		if p.tokenIs(tok, "__VA_OPT__") && i+1 < len(replacement) && p.tokenIs(replacement[i+1], "(") {
			inside, close, ok := p.parenthesized(replacement, i+1)
			if !ok {
				return p.macroFail(macro.name)
			}
			if macro.variadic && len(args) == len(macro.params) && len(args[len(args)-1]) > 0 {
				if len(inside) > 0 {
					inside[0] = ppWithSpace(inside[0], ppTokenSpace(tok))
				}
				replacement = ppReplaceTokens(ppCloneTokens(replacement), i, close+1, inside)
				i--
				continue
			}
			out = p.appendReplacementPiece(out, nil, paste)
			paste = false
			i = close
			continue
		}
		param := p.parameterIndex(macro, tok)
		piece := []ppToken{tok}
		if param >= 0 {
			adjacentPaste := paste || i+1 < len(replacement) && p.tokenIs(replacement[i+1], "##")
			if adjacentPaste {
				piece = args[param]
			} else {
				if !expandedReady[param] {
					expandedArgs[param] = p.expand(ppCloneTokens(args[param]), depth+1)
					expandedReady[param] = true
				}
				piece = expandedArgs[param]
			}
		}
		out = p.appendReplacementPiece(out, piece, paste)
		paste = false
	}
	if paste {
		return p.macroFail(macro.name)
	}
	if p.ok {
		p.errorDetail = previousDetail
	}
	return out
}

func (p *preprocessor) macroFail(name string) []ppToken {
	p.errorDetail = name
	p.fail(PreprocessErrMacro, p.currentPath, p.currentLine)
	return nil
}

func (p *preprocessor) parameterIndex(macro ppMacro, tok ppToken) int {
	if ppTokenKind(tok) != ppIdent {
		return -1
	}
	text := p.tokenText(tok)
	for i := 0; i < len(macro.params); i++ {
		if ppBytesStringEqual(text, macro.params[i]) {
			return i
		}
	}
	return -1
}

func (p *preprocessor) parenthesized(tokens []ppToken, open int) ([]ppToken, int, bool) {
	depth := 0
	for i := open + 1; i < len(tokens); i++ {
		if p.tokenIs(tokens[i], "(") {
			depth++
		} else if p.tokenIs(tokens[i], ")") {
			if depth == 0 {
				return ppCloneTokens(tokens[open+1 : i]), i, true
			}
			depth--
		}
	}
	return nil, 0, false
}

func (p *preprocessor) appendReplacementPiece(out []ppToken, piece []ppToken, paste bool) []ppToken {
	if !paste {
		return append(out, piece...)
	}
	if len(piece) == 0 {
		// GNU's comma-swallowing variadic extension.
		if len(out) > 0 && p.tokenIs(out[len(out)-1], ",") {
			return out[:len(out)-1]
		}
		return out
	}
	if len(out) == 0 {
		return append(out, piece...)
	}
	if p.tokenIs(out[len(out)-1], ",") {
		// GNU , ##args keeps the comma when the variadic pack is present.
		return append(out, piece...)
	}
	left := out[len(out)-1]
	right := piece[0]
	text := append([]byte{}, p.tokenText(left)...)
	text = append(text, p.tokenText(right)...)
	combined := p.generatedToken(ppPunct, text, ppTokenSpace(left))
	lexed, ok := p.lex(0, combined.start, combined.end)
	if !ok || len(lexed) != 1 || lexed[0].start != combined.start || lexed[0].end != combined.end {
		p.fail(PreprocessErrMacro, p.currentPath, p.currentLine)
		return nil
	}
	lexed[0] = ppWithSpace(lexed[0], ppTokenSpace(left))
	out[len(out)-1] = lexed[0]
	out = append(out, piece[1:]...)
	return out
}

func ppCloneTokens(tokens []ppToken) []ppToken {
	out := make([]ppToken, len(tokens))
	copy(out, tokens)
	return out
}

func (p *preprocessor) stringize(tokens []ppToken, space bool) ppToken {
	var text []byte
	text = append(text, '"')
	for i := 0; i < len(tokens); i++ {
		if i > 0 && ppTokenSpace(tokens[i]) {
			text = append(text, ' ')
		}
		spelling := p.tokenText(tokens[i])
		for j := 0; j < len(spelling); j++ {
			if spelling[j] == '\\' || spelling[j] == '"' {
				text = append(text, '\\')
			}
			text = append(text, spelling[j])
		}
	}
	text = append(text, '"')
	return p.generatedToken(ppString, text, space)
}

func (p *preprocessor) generatedString(value []byte, space bool) ppToken {
	var text []byte
	text = append(text, '"')
	for i := 0; i < len(value); i++ {
		if value[i] == '\\' || value[i] == '"' {
			text = append(text, '\\')
		}
		text = append(text, value[i])
	}
	text = append(text, '"')
	return p.generatedToken(ppString, text, space)
}
