package check

import "renvo.dev/internal/load"
import "renvo.dev/internal/syntax"

type localRuleBinding struct {
	token    int
	start    int
	end      int
	scopeEnd int
}

// Each switch/select clause introduces an implicit block, independent of its
// siblings. Use its boundary as well as the explicit brace block boundary.
func localRuleScopeEnd(body syntax.Body, tok int) int {
	end := definiteStatementScopeEnd(body, tok)
	for i := 0; i < len(body.Stmts); i++ {
		owner := body.Stmts[i]
		if (owner.Kind != syntax.StmtSwitch && owner.Kind != syntax.StmtSelect) || tok <= owner.BodyStart || tok >= owner.BodyEnd {
			continue
		}
		for j := i + 1; j < len(body.Stmts); j++ {
			clause := body.Stmts[j]
			if (clause.Kind != syntax.StmtCase && clause.Kind != syntax.StmtDefault) || clause.StartTok <= tok || clause.StartTok >= end || clause.StartTok >= owner.BodyEnd {
				continue
			}
			nested := false
			for k := i + 1; k < j; k++ {
				inner := body.Stmts[k]
				if (inner.Kind == syntax.StmtSwitch || inner.Kind == syntax.StmtSelect) && inner.BodyStart < clause.StartTok && clause.StartTok < inner.BodyEnd {
					nested = true
				}
			}
			if !nested {
				end = clause.StartTok
			}
		}
	}
	return end
}

func invalidLocalRules(pkg load.Package, info PackageInfo, file syntax.File, fn syntax.FuncDecl, body syntax.Body) (int, int) {
	var bindings []localRuleBinding
	signature := buildFuncSignature(file, fn)
	for i := 0; i < len(signature.Params); i++ {
		field := signature.Params[i]
		if field.NameTok >= 0 {
			bindings = append(bindings, localRuleBinding{field.NameTok, field.TypeStart, field.TypeEnd, fn.BodyEnd})
		}
	}
	for i := 0; i < len(body.Stmts); i++ {
		stmt := body.Stmts[i]
		if stmt.Kind == syntax.StmtDecl && (file.Tokens[stmt.StartTok].KindLine&255 == syntax.TokenVar || file.Tokens[stmt.StartTok].KindLine&255 == syntax.TokenConst) {
			names, start := localDeclNameTokens(file, stmt.StartTok+1, stmt.EndTok)
			op := findDeclAssign(file, start, stmt.EndTok)
			end := stmt.EndTok
			if op >= 0 {
				end = op
			}
			start, end = trimDeclSpan(file, start, end)
			for _, name := range names {
				bindings = append(bindings, localRuleBinding{name, start, end, localRuleScopeEnd(body, stmt.StartTok)})
			}
			if op >= 0 && end-start == 1 && literalIntegerOverflows(file, op+1, stmt.EndTok, tokenString(&file, start)) {
				return CheckErrType, op + 1
			}
			if op >= 0 && end-start == 1 {
				rightStart, rightEnd := trimDeclSpan(file, op+1, stmt.EndTok)
				rightStart, rightEnd = stripOuterParens(file, rightStart, rightEnd)
				declared := tokenString(&file, start)
				if rightEnd-rightStart == 1 && definiteBuiltinType(declared) {
					kind := definiteLiteralKind(file, rightStart)
					if kind != "" && kind != declared {
						return CheckErrType, rightStart
					}
				}
			}
		}
		if stmt.Kind != syntax.StmtAssign {
			continue
		}
		op := findTopLevelAssignOp(file, stmt.StartTok, stmt.EndTok)
		if op >= 0 && tokCharIs(&file, stmt.StartTok+1, '[') {
			for j := len(bindings) - 1; j >= 0; j-- {
				binding := bindings[j]
				if binding.token >= stmt.StartTok || binding.scopeEnd <= stmt.StartTok || !statementTokensEqual(&file, binding.token, stmt.StartTok) {
					continue
				}
				if binding.end-binding.start == 1 && (tokenTextIs(&file, binding.start, "string") || file.Tokens[binding.start].KindLine&255 == syntax.TokenString) {
					return CheckErrAssignTarget, stmt.StartTok
				}
				break
			}
		}
		if op < 0 || !tokenTextIs(&file, op, ":=") {
			continue
		}
		endScope := localRuleScopeEnd(body, stmt.StartTok)
		newNames := false
		for tok := stmt.StartTok; tok < op; tok++ {
			if tokCharIs(&file, tok, ',') {
				continue
			}
			if file.Tokens[tok].KindLine&255 != syntax.TokenIdent {
				return CheckErrAssignTarget, tok
			}
			if tokenTextIs(&file, tok, "_") {
				continue
			}
			exists := false
			for j := len(bindings) - 1; j >= 0; j-- {
				if bindings[j].scopeEnd == endScope && statementTokensEqual(&file, tok, bindings[j].token) {
					exists = true
					break
				}
			}
			for prev := stmt.StartTok; prev < tok; prev++ {
				if statementTokensEqual(&file, prev, tok) {
					return CheckErrScope, tok
				}
			}
			if !exists {
				newNames = true
			}
			start, end := op+1, op+1
			if op-stmt.StartTok == 1 {
				if file.Tokens[start].KindLine&255 == syntax.TokenString {
					end = start + 1
				}
				if open := findTypeTopLevelChar(file, start, stmt.EndTok, '{'); open >= 0 {
					end = open
				}
			}
			bindings = append(bindings, localRuleBinding{tok, start, end, endScope})
		}
		if !newNames {
			return CheckErrScope, op
		}
	}
	for tok := fn.BodyStart + 2; tok+1 < fn.BodyEnd; tok++ {
		if !tokenTextIs(&file, tok, "==") && !tokenTextIs(&file, tok, "!=") {
			continue
		}
		if tokenTextIs(&file, tok-1, "nil") || tokenTextIs(&file, tok+1, "nil") {
			continue
		}
		if file.Tokens[tok-1].KindLine&255 != syntax.TokenIdent || file.Tokens[tok+1].KindLine&255 != syntax.TokenIdent {
			continue
		}
		for j := len(bindings) - 1; j >= 0; j-- {
			binding := bindings[j]
			if binding.token >= tok || binding.scopeEnd <= tok || !statementTokensEqual(&file, binding.token, tok-1) {
				continue
			}
			if nonComparableTypeSpan(pkg, info, file, binding.start, binding.end, 0) {
				return CheckErrOperand, tok
			}
			break
		}
	}
	return CheckOK, -1
}

// Prove representability of integer literals without first truncating them to
// the host integer width. Non-literal constant expressions need full constant
// evaluation and are deliberately not guessed here.
func literalIntegerOverflows(file syntax.File, start int, end int, typ string) bool {
	start, end = trimDeclSpan(file, start, end)
	start, end = stripOuterParens(file, start, end)
	negative := false
	if start < end && (tokenTextIs(&file, start, "-") || tokenTextIs(&file, start, "+")) {
		negative = tokenTextIs(&file, start, "-")
		start++
	}
	bits := 0
	signed := true
	if typ == "byte" || typ == "uint8" {
		bits = 8
		signed = false
	} else if typ == "int8" {
		bits = 8
	} else if typ == "uint16" {
		bits = 16
		signed = false
	} else if typ == "int16" {
		bits = 16
	} else if typ == "uint32" {
		bits = 32
		signed = false
	} else if typ == "int32" || typ == "rune" {
		bits = 32
	} else if typ == "uint64" {
		bits = 64
		signed = false
	} else if typ == "int64" {
		bits = 64
	}
	if bits == 0 || end-start != 1 || file.Tokens[start].KindLine&255 != syntax.TokenNumber {
		return false
	}
	text := tokenString(&file, start)
	base := uint64(10)
	i := 0
	if len(text) > 1 && text[0] == '0' {
		base = 8
		i = 1
		if text[1] == 'x' || text[1] == 'X' {
			base = 16
			i = 2
		} else if text[1] == 'b' || text[1] == 'B' {
			base = 2
			i = 2
		} else if text[1] == 'o' || text[1] == 'O' {
			i = 2
		}
	}
	limit := ^uint64(0)
	if bits < 64 {
		limit = uint64(1)<<uint(bits) - 1
	}
	if signed {
		limit >>= 1
		if negative {
			limit++
		}
	}
	value := uint64(0)
	overflow := false
	for ; i < len(text); i++ {
		c := text[i]
		if c == '_' {
			continue
		}
		digit := uint64(99)
		if c >= '0' && c <= '9' {
			digit = uint64(c - '0')
		} else if c >= 'a' && c <= 'f' {
			digit = uint64(c-'a') + 10
		} else if c >= 'A' && c <= 'F' {
			digit = uint64(c-'A') + 10
		}
		if digit >= base {
			return false
		}
		if value > limit/base || value == limit/base && digit > limit%base {
			overflow = true
		} else if !overflow {
			value = value*base + digit
		}
	}
	return overflow || negative && !signed && value != 0
}
