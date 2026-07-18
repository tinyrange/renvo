package check

import (
	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtg/internal/syntax"
)

// SignatureHelpGraph describes the innermost call containing offset. The
// result shares completion's semantic resolution so editor presentation never
// needs to infer Go method sets or package symbols itself.
func SignatureHelpGraph(graph load.Graph, path string, offset int) SignatureHelp {
	return SignatureHelpProgram(graph, completionProgram(graph), path, offset)
}

func SignatureHelpProgram(graph load.Graph, program Program, path string, offset int) SignatureHelp {
	pkgIndex, fileIndex := completionFile(graph, path)
	if pkgIndex < 0 || fileIndex < 0 {
		return SignatureHelp{}
	}
	file := graph.Packages[pkgIndex].Files[fileIndex].File
	if offset < 0 {
		offset = 0
	}
	if offset > len(file.Src) {
		offset = len(file.Src)
	}
	open := signatureCallOpen(file, offset)
	if open <= 0 || file.Tokens[open-1].Kind != syntax.TokenIdent {
		return SignatureHelp{}
	}
	nameTok := open - 1
	name := tokenString(&file, nameTok)
	queryOffset := file.Tokens[nameTok].End
	if queryOffset > file.Tokens[nameTok].Start {
		queryOffset--
	}
	items := CompleteProgram(graph, program, path, queryOffset)
	for i := 0; i < len(items); i++ {
		if items[i].Name != name || items[i].Signature == "" {
			continue
		}
		active := signatureActiveParameter(file, open, offset)
		if len(items[i].Parameters) > 0 && active >= len(items[i].Parameters) {
			active = len(items[i].Parameters) - 1
		}
		return SignatureHelp{Ok: true, Label: items[i].Signature, Parameters: items[i].Parameters, ActiveParameter: active}
	}
	return SignatureHelp{}
}

func signatureCallOpen(file syntax.File, offset int) int {
	paren, bracket, brace := 0, 0, 0
	for i := len(file.Tokens) - 1; i >= 0; i-- {
		tok := file.Tokens[i]
		if tok.Start >= offset || tok.Kind == syntax.TokenEOF {
			continue
		}
		text := syntax.TokenText(file.Src, tok)
		if len(text) != 1 {
			continue
		}
		switch text[0] {
		case ')':
			paren++
		case ']':
			bracket++
		case '}':
			brace++
		case '(':
			if paren > 0 {
				paren--
			} else if bracket == 0 && brace == 0 {
				return i
			}
		case '[':
			if bracket > 0 {
				bracket--
			}
		case '{':
			if brace > 0 {
				brace--
			}
		}
	}
	return -1
}

func signatureActiveParameter(file syntax.File, open int, offset int) int {
	active := 0
	paren, bracket, brace := 0, 0, 0
	for i := open + 1; i < len(file.Tokens); i++ {
		tok := file.Tokens[i]
		if tok.Start >= offset || tok.Kind == syntax.TokenEOF {
			break
		}
		text := syntax.TokenText(file.Src, tok)
		if len(text) != 1 {
			continue
		}
		switch text[0] {
		case '(':
			paren++
		case ')':
			if paren > 0 {
				paren--
			}
		case '[':
			bracket++
		case ']':
			if bracket > 0 {
				bracket--
			}
		case '{':
			brace++
		case '}':
			if brace > 0 {
				brace--
			}
		case ',':
			if paren == 0 && bracket == 0 && brace == 0 {
				active++
			}
		}
	}
	return active
}
