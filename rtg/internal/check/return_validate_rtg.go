//go:build rtg

package check

import "j5.nz/rtg/rtg/internal/syntax"

func invalidDefiniteAssignmentType(file syntax.File, fn syntax.FuncDecl) int {
	for i := fn.BodyStart + 2; i+1 < fn.BodyEnd; i++ {
		if !tokenTextIs(&file, i, "=") || file.Tokens[i-1].Kind != syntax.TokenIdent || file.Tokens[i+1].Kind != syntax.TokenString {
			continue
		}
		for j := i - 2; j >= fn.BodyStart+1; j-- {
			if file.Tokens[j].Kind == syntax.TokenVar && j+2 < i && tokenTextIs(&file, j+1, tokenString(&file, i-1)) && tokenTextIs(&file, j+2, "int") {
				return i + 1
			}
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
