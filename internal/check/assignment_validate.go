package check

import "renvo.dev/internal/syntax"

func invalidDefiniteAssignmentType(file syntax.File, fn syntax.FuncDecl) (int, int) {
	for i := fn.BodyStart + 2; i+1 < fn.BodyEnd; i++ {
		if file.Tokens[i].Kind != syntax.TokenOperator {
			continue
		}
		operator := file.Src[file.Tokens[i].Start]
		if operator == '+' || operator == '-' || operator == '*' || operator == '/' || operator == '%' || operator == '&' || operator == '|' || operator == '^' || operator == '<' || operator == '>' || operator == '!' || operator == '=' {
			leftToken := file.Tokens[i-1]
			rightToken := file.Tokens[i+1]
			leftPossible := leftToken.Kind == syntax.TokenNumber || leftToken.Kind == syntax.TokenString || leftToken.Kind == syntax.TokenIdent && (leftToken.End-leftToken.Start == 4 || leftToken.End-leftToken.Start == 5)
			rightPossible := rightToken.Kind == syntax.TokenNumber || rightToken.Kind == syntax.TokenString || rightToken.Kind == syntax.TokenIdent && (rightToken.End-rightToken.Start == 4 || rightToken.End-rightToken.Start == 5)
			if leftPossible && rightPossible {
				leftKind := definiteLiteralKind(file, i-1)
				rightKind := definiteLiteralKind(file, i+1)
				if tokenTextIs(&file, i, "&&") || tokenTextIs(&file, i, "||") {
					if leftKind != "bool" && i >= 2 && isExprBinaryOp(file, i-2) {
						continue
					}
					if rightKind != "bool" && i+2 < fn.BodyEnd && isExprBinaryOp(file, i+2) {
						continue
					}
				}
				if leftKind != "" && rightKind != "" && isExprBinaryOp(file, i) && invalidDefiniteLiteralBinary(file, i, leftKind, rightKind) {
					return CheckErrOperand, i
				}
			}
		}
		if !tokenTextIs(&file, i, "=") || file.Tokens[i-1].Kind != syntax.TokenIdent {
			continue
		}
		valueKind := definiteLiteralKind(file, i+1)
		if valueKind == "" && file.Tokens[i+1].Kind == syntax.TokenIdent {
			valueKind = definiteShortValueKind(file, fn, i+1, i)
		}
		if valueKind == "" {
			continue
		}
		name := tokenString(&file, i-1)
		for j := i - 2; j >= fn.BodyStart+1; j-- {
			if file.Tokens[j].Kind != syntax.TokenVar || j+2 >= i || file.Tokens[j+1].Kind != syntax.TokenIdent || tokenString(&file, j+1) != name {
				continue
			}
			declared := tokenString(&file, j+2)
			if definiteBuiltinType(declared) && declared != valueKind {
				return CheckErrType, i + 1
			}
			break
		}
	}
	return CheckOK, -1
}

func definiteShortValueKind(file syntax.File, fn syntax.FuncDecl, nameTok int, before int) string {
	limit := before - 64
	if limit < fn.BodyStart {
		limit = fn.BodyStart
	}
	for i := before - 1; i > limit; i-- {
		if statementTokensEqual(&file, i, nameTok) && i+2 < before && tokenTextIs(&file, i+1, ":=") {
			return definiteLiteralKind(file, i+2)
		}
	}
	return ""
}

func excludedFileFeature(file syntax.File) (int, int) {
	for i := 0; i < len(file.Tokens); i++ {
		if file.Tokens[i].Kind == syntax.TokenSelect {
			return CheckErrSelect, i
		}
	}
	for i := 0; i < len(file.Tokens); i++ {
		kind := file.Tokens[i].Kind
		if kind == syntax.TokenGo {
			return CheckErrGoroutine, i
		}
		if kind == syntax.TokenChan || tokenTextIs(&file, i, "<-") {
			return CheckErrChannel, i
		}
	}
	return CheckOK, -1
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
	if tokenTextIs(&file, tok, "true") || tokenTextIs(&file, tok, "false") {
		return "bool"
	}
	return ""
}

func definiteBuiltinType(name string) bool {
	return name == "int" || name == "string" || name == "bool"
}
