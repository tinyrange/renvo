package check

import "renvo.dev/internal/syntax"

type definiteChannelBinding struct {
	name      string
	direction int
	element   string
	token     int
}

func functionMayNeedChannelCheck(file syntax.File, fn syntax.FuncDecl) bool {
	start := fn.BodyStart + 1
	end := fn.BodyEnd
	if start < 0 {
		start = 0
	}
	if end > len(file.Tokens) {
		end = len(file.Tokens)
	}
	for i := start; i < end; i++ {
		token := file.Tokens[i]
		kind := token.KindLine & 255
		if kind == syntax.TokenChan {
			return true
		}
		size := int(token.End - token.Start)
		start := int(token.Start)
		if kind == syntax.TokenOperator && size == 2 && file.Src[start] == '<' && file.Src[start+1] == '-' {
			return true
		}
		if kind == syntax.TokenIdent && size == 5 && file.Src[start] == 'c' && tokenTextIs(&file, i, "close") {
			return true
		}
	}
	return false
}

func invalidDefiniteAssignmentType(file syntax.File, fn syntax.FuncDecl) (int, int) {
	for i := fn.BodyStart + 2; i+1 < fn.BodyEnd; i++ {
		if file.Tokens[i].KindLine&255 != syntax.TokenOperator {
			continue
		}
		operator := file.Src[int(file.Tokens[i].Start)]
		if operator == '+' || operator == '-' || operator == '*' || operator == '/' || operator == '%' || operator == '&' || operator == '|' || operator == '^' || operator == '<' || operator == '>' || operator == '!' || operator == '=' {
			leftToken := file.Tokens[i-1]
			rightToken := file.Tokens[i+1]
			leftSize := leftToken.End - leftToken.Start
			rightSize := rightToken.End - rightToken.Start
			leftPossible := leftToken.KindLine&255 == syntax.TokenNumber || leftToken.KindLine&255 == syntax.TokenString || leftToken.KindLine&255 == syntax.TokenIdent && (leftSize == 4 || leftSize == 5)
			rightPossible := rightToken.KindLine&255 == syntax.TokenNumber || rightToken.KindLine&255 == syntax.TokenString || rightToken.KindLine&255 == syntax.TokenIdent && (rightSize == 4 || rightSize == 5)
			if leftPossible && rightPossible {
				leftKind := definiteLiteralKind(file, i-1)
				rightKind := definiteLiteralKind(file, i+1)
				if exprBinaryOperatorKind(file, i) == exprBinaryLogical {
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
		if operator != '=' || file.Tokens[i].End-file.Tokens[i].Start != 1 || file.Tokens[i-1].KindLine&255 != syntax.TokenIdent {
			continue
		}
		valueKind := definiteLiteralKind(file, i+1)
		if valueKind == "" && file.Tokens[i+1].KindLine&255 == syntax.TokenIdent {
			valueKind = definiteShortValueKind(file, fn, i+1, i)
		}
		if valueKind == "" {
			continue
		}
		name := tokenString(&file, i-1)
		for j := i - 2; j >= fn.BodyStart+1; j-- {
			if file.Tokens[j].KindLine&255 != syntax.TokenVar || j+2 >= i || file.Tokens[j+1].KindLine&255 != syntax.TokenIdent || tokenString(&file, j+1) != name {
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
		if i+2 >= before || file.Tokens[i].KindLine&255 != syntax.TokenIdent || !tokenTextIs(&file, i+1, ":=") {
			continue
		}
		if statementTokensEqual(&file, i, nameTok) {
			return definiteLiteralKind(file, i+2)
		}
	}
	return ""
}

func excludedFileFeature(file syntax.File) (int, int) {
	return CheckOK, -1
}

func invalidDefiniteChannelOperation(file syntax.File, fn syntax.FuncDecl) int {
	var channels []definiteChannelBinding
	maybeChannelOperation := false
	for i := 0; i < len(file.Decls); i++ {
		decl := file.Decls[i]
		if decl.Kind != syntax.TokenVar {
			continue
		}
		typeStart := definiteGlobalDeclTypeStart(file, decl)
		if direction, element, ok := definiteChannelShape(file, typeStart, decl.EndTok); ok {
			channels = append(channels, definiteChannelBinding{name: tokenString(&file, decl.NameTok), direction: direction, element: element, token: decl.NameTok})
		}
	}
	for i := fn.StartTok; i+1 < fn.BodyEnd && i+1 < len(file.Tokens); i++ {
		kind := file.Tokens[i].KindLine & 255
		if kind == syntax.TokenChan {
			maybeChannelOperation = true
		}
		if kind == syntax.TokenOperator && file.Tokens[i].End-file.Tokens[i].Start == 2 &&
			file.Src[int(file.Tokens[i].Start)] == '<' && file.Src[int(file.Tokens[i].Start)+1] == '-' {
			maybeChannelOperation = true
		}
		if kind != syntax.TokenIdent {
			continue
		}
		size := int(file.Tokens[i].End - file.Tokens[i].Start)
		start := int(file.Tokens[i].Start)
		if size == 4 && file.Src[start] == 'm' && tokenTextIs(&file, i, "make") ||
			size == 5 && file.Src[start] == 'c' && tokenTextIs(&file, i, "close") {
			maybeChannelOperation = true
		}
		typeStart := i + 1
		directChannel := typeStart < len(file.Tokens) && (file.Tokens[typeStart].KindLine&255 == syntax.TokenChan || tokenTextIs(&file, typeStart, "<-"))
		namedDeclaration := typeStart < len(file.Tokens) && file.Tokens[typeStart].KindLine&255 == syntax.TokenIdent && (i < fn.BodyStart || i > fn.BodyStart && file.Tokens[i-1].KindLine&255 == syntax.TokenVar)
		if directChannel || namedDeclaration {
			if direction, element, ok := definiteChannelShape(file, typeStart, len(file.Tokens)); ok {
				channels = append(channels, definiteChannelBinding{name: tokenString(&file, i), direction: direction, element: element, token: i})
				continue
			}
		}
		if i+4 < len(file.Tokens) && tokenTextIs(&file, i+1, ":=") && tokenTextIs(&file, i+2, "make") && tokCharIs(&file, i+3, '(') {
			if direction, element, ok := definiteChannelShape(file, i+4, len(file.Tokens)); ok {
				channels = append(channels, definiteChannelBinding{name: tokenString(&file, i), direction: direction, element: element, token: i})
				continue
			}
		}
	}
	if len(channels) == 0 && !maybeChannelOperation {
		return -1
	}
	for i := fn.BodyStart + 1; i < fn.BodyEnd && i < len(file.Tokens); i++ {
		if tokenTextIs(&file, i, "=") && i > fn.BodyStart+1 && i+1 < fn.BodyEnd && file.Tokens[i-1].KindLine&255 == syntax.TokenIdent && file.Tokens[i+1].KindLine&255 == syntax.TokenIdent {
			left, leftFound := definiteChannelBindingAt(channels, tokenString(&file, i-1), i)
			right, rightFound := definiteChannelBindingAt(channels, tokenString(&file, i+1), i)
			if leftFound && rightFound {
				if left.element != "" && right.element != "" && left.element != right.element || left.direction == ChanBoth && right.direction != ChanBoth || left.direction == ChanSendOnly && right.direction == ChanReceiveOnly || left.direction == ChanReceiveOnly && right.direction == ChanSendOnly {
					return i
				}
			} else if leftFound && definiteNonChannelBindingKind(file, fn, tokenString(&file, i+1), i) != "" || rightFound && definiteNonChannelBindingKind(file, fn, tokenString(&file, i-1), i) != "" {
				return i
			}
		}
		if tokenTextIs(&file, i, "make") && i+2 < fn.BodyEnd && tokCharIs(&file, i+1, '(') {
			close := findTypeMatching(file, i+1, '(', ')')
			if close > i+1 {
				args := splitExprList(file, i+2, close-1)
				if len(args) > 0 {
					if _, _, channel := definiteChannelShape(file, args[0].StartTok, args[0].EndTok); channel {
						if len(args) > 2 {
							return i
						}
						if len(args) == 2 && invalidDefiniteChannelCapacity(file, args[1]) {
							return args[1].StartTok
						}
					}
				}
			}
		}
		if file.Tokens[i].KindLine&255 == syntax.TokenFor {
			rangeTok := -1
			for j := i + 1; j < fn.BodyEnd && !tokCharIs(&file, j, '{'); j++ {
				if file.Tokens[j].KindLine&255 == syntax.TokenRange {
					rangeTok = j
					break
				}
			}
			channelRange := false
			if rangeTok > i && rangeTok+1 < fn.BodyEnd && file.Tokens[rangeTok+1].KindLine&255 == syntax.TokenIdent {
				_, found := definiteChannelBindingAt(channels, tokenString(&file, rangeTok+1), rangeTok)
				channelRange = found
			}
			if channelRange {
				for j := i + 1; j < rangeTok; j++ {
					if tokCharIs(&file, j, ',') {
						return j
					}
				}
			}
		}
		if tokenTextIs(&file, i, "<-") {
			if i > 0 && file.Tokens[i-1].KindLine&255 == syntax.TokenChan || i+1 < len(file.Tokens) && file.Tokens[i+1].KindLine&255 == syntax.TokenChan {
				continue
			}
			if i > 0 && file.Tokens[i-1].KindLine&255 == syntax.TokenIdent {
				if binding, found := definiteChannelBindingAt(channels, tokenString(&file, i-1), i); found {
					if binding.direction == ChanReceiveOnly {
						return i
					}
					if i+1 < fn.BodyEnd {
						kind := definiteLiteralKind(file, i+1)
						if kind != "" && binding.element != "" && definiteChannelBuiltinElement(binding.element) && kind != binding.element {
							return i + 1
						}
					}
					continue
				}
				if definiteNonChannelBindingKind(file, fn, tokenString(&file, i-1), i) != "" {
					return i
				}
			}
			if i+1 < len(file.Tokens) && file.Tokens[i+1].KindLine&255 == syntax.TokenIdent {
				if binding, found := definiteChannelBindingAt(channels, tokenString(&file, i+1), i); found {
					if binding.direction == ChanSendOnly {
						return i
					}
				} else if definiteNonChannelBindingKind(file, fn, tokenString(&file, i+1), i) != "" {
					return i
				}
			}
		}
		if tokenTextIs(&file, i, "close") && i+1 < fn.BodyEnd && tokCharIs(&file, i+1, '(') {
			if i > fn.BodyStart+1 && tokCharIs(&file, i-1, '.') {
				continue
			}
			close := findTypeMatching(file, i+1, '(', ')')
			args := splitExprList(file, i+2, close-1)
			if len(args) != 1 {
				return i
			}
			if args[0].EndTok-args[0].StartTok == 1 && file.Tokens[args[0].StartTok].KindLine&255 == syntax.TokenIdent {
				name := tokenString(&file, args[0].StartTok)
				if binding, found := definiteChannelBindingAt(channels, name, i); found {
					// Renvo's syscall surface predates channel support and uses
					// close(int) for file descriptors. Only the channel form has a
					// direction restriction; other definite operand types remain
					// invalid.
					if binding.direction == ChanReceiveOnly {
						return i
					}
				} else if kind := definiteNonChannelBindingKind(file, fn, name, i); kind != "" && kind != "int" {
					return i
				}
			}
		}
	}
	return -1
}

func definiteNonChannelBindingKind(file syntax.File, fn syntax.FuncDecl, name string, before int) string {
	for i := before - 1; i >= fn.StartTok; i-- {
		if file.Tokens[i].KindLine&255 != syntax.TokenIdent || tokenString(&file, i) != name {
			continue
		}
		if i+2 < before && tokenTextIs(&file, i+1, ":=") {
			return definiteLiteralKind(file, i+2)
		}
		if i > fn.StartTok && file.Tokens[i-1].KindLine&255 == syntax.TokenVar && i+1 < before {
			return tokenString(&file, i+1)
		}
	}
	return ""
}

func definiteGlobalDeclTypeStart(file syntax.File, decl syntax.TopDecl) int {
	i := decl.StartTok
	for i < decl.EndTok && file.Tokens[i].KindLine&255 == syntax.TokenIdent {
		i++
		if i < decl.EndTok && tokCharIs(&file, i, ',') {
			i++
			continue
		}
		break
	}
	return i
}

func invalidDefiniteChannelCapacity(file syntax.File, span ExprSpan) bool {
	start, end := trimExprSpan(file, span.StartTok, span.EndTok)
	if start >= end {
		return true
	}
	if tokenTextIs(&file, start, "-") && start+1 < end && file.Tokens[start+1].KindLine&255 == syntax.TokenNumber {
		return true
	}
	kind := definiteLiteralKind(file, start)
	return kind == "string" || kind == "bool"
}

func definiteChannelBuiltinElement(element string) bool {
	return element == "bool" || element == "string" || definiteBuiltinType(element)
}

func definiteChannelBindingAt(channels []definiteChannelBinding, name string, before int) (definiteChannelBinding, bool) {
	for i := len(channels) - 1; i >= 0; i-- {
		if channels[i].token <= before && channels[i].name == name {
			return channels[i], true
		}
	}
	return definiteChannelBinding{}, false
}

func definiteChannelBindingDirection(channels []definiteChannelBinding, name string, before int) int {
	if binding, ok := definiteChannelBindingAt(channels, name, before); ok {
		return binding.direction
	}
	return ChanBoth
}

func definiteChannelDirection(file syntax.File, start int, end int) (int, bool) {
	direction, _, ok := definiteChannelShape(file, start, end)
	return direction, ok
}

func definiteChannelShape(file syntax.File, start int, end int) (int, string, bool) {
	return definiteChannelShapeDepth(file, start, end, 0)
}

func definiteChannelDirectionDepth(file syntax.File, start int, end int, depth int) (int, bool) {
	direction, _, ok := definiteChannelShapeDepth(file, start, end, depth)
	return direction, ok
}

func definiteChannelShapeDepth(file syntax.File, start int, end int, depth int) (int, string, bool) {
	if depth > len(file.Decls) {
		return ChanBoth, "", false
	}
	if start < 0 || start >= end || start >= len(file.Tokens) {
		return ChanBoth, "", false
	}
	if file.Tokens[start].KindLine&255 == syntax.TokenChan {
		if start+1 < end && tokenTextIs(&file, start+1, "<-") {
			if start+2 < end {
				return ChanSendOnly, tokenString(&file, start+2), true
			}
			return ChanSendOnly, "", true
		}
		if start+1 < end {
			return ChanBoth, tokenString(&file, start+1), true
		}
		return ChanBoth, "", true
	}
	if tokenTextIs(&file, start, "<-") && start+1 < end && file.Tokens[start+1].KindLine&255 == syntax.TokenChan {
		if start+2 < end {
			return ChanReceiveOnly, tokenString(&file, start+2), true
		}
		return ChanReceiveOnly, "", true
	}
	if file.Tokens[start].KindLine&255 == syntax.TokenIdent {
		name := tokenString(&file, start)
		for i := 0; i < len(file.Decls); i++ {
			decl := file.Decls[i]
			if decl.Kind != syntax.TokenType || tokenString(&file, decl.NameTok) != name {
				continue
			}
			typeStart := decl.NameTok + 1
			if tokenTextIs(&file, typeStart, "=") {
				typeStart++
			}
			if typeStart != start {
				return definiteChannelShapeDepth(file, typeStart, decl.EndTok, depth+1)
			}
		}
	}
	return ChanBoth, "", false
}

func definiteLiteralKind(file syntax.File, tok int) string {
	if tok < 0 || tok >= len(file.Tokens) {
		return ""
	}
	if file.Tokens[tok].KindLine&255 == syntax.TokenString {
		return "string"
	}
	if file.Tokens[tok].KindLine&255 == syntax.TokenNumber {
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
