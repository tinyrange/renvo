//go:build !rtg

package check

import "j5.nz/rtg/rtg/internal/syntax"

func invalidDefiniteAssignmentType(file syntax.File, fn syntax.FuncDecl) int {
	for i := fn.BodyStart + 2; i+1 < fn.BodyEnd; i++ {
		if !tokenTextIs(file, i, "=") || file.Tokens[i-1].Kind != syntax.TokenIdent {
			continue
		}
		valueKind := definiteLiteralKind(file, i+1)
		if valueKind == "" {
			continue
		}
		name := tokenString(file, i-1)
		for j := i - 2; j >= fn.BodyStart+1; j-- {
			if file.Tokens[j].Kind != syntax.TokenVar || j+2 >= i || file.Tokens[j+1].Kind != syntax.TokenIdent || tokenString(file, j+1) != name {
				continue
			}
			declared := tokenString(file, j+2)
			if definiteBuiltinType(declared) && declared != valueKind {
				return i + 1
			}
			break
		}
	}
	return -1
}

func excludedFeatureToken(file syntax.File, fn syntax.FuncDecl) int {
	for i := fn.StartTok; i < fn.EndTok && i < len(file.Tokens); i++ {
		kind := file.Tokens[i].Kind
		if kind == syntax.TokenGo || kind == syntax.TokenChan || kind == syntax.TokenSelect {
			return i
		}
	}
	return -1
}

func definiteLiteralKind(file syntax.File, tok int) string {
	if tok < 0 || tok >= len(file.Tokens) {
		return ""
	}
	if file.Tokens[tok].Kind == syntax.TokenString {
		return "string"
	}
	if file.Tokens[tok].Kind == syntax.TokenNumber {
		return "int"
	}
	if tokenTextIs(file, tok, "true") || tokenTextIs(file, tok, "false") {
		return "bool"
	}
	return ""
}

func definiteBuiltinType(name string) bool {
	return name == "int" || name == "string" || name == "bool"
}
