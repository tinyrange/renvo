package check

import "renvo.dev/internal/load"
import "renvo.dev/internal/syntax"

type localRuleBinding struct {
	token    int
	start    int
	end      int
	scopeEnd int
}

func invalidLocalRules(pkg *load.Package, info *PackageInfo, file *syntax.File, fn syntax.FuncDecl, body *syntax.Body, signature *FuncSignature, scope CoreScope) (int, int) {
	var bindings []localRuleBinding
	for i := 0; i < len(signature.Params); i++ {
		field := signature.Params[i]
		if field.NameTok >= 0 {
			bindings = append(bindings, localRuleBinding{field.NameTok, field.TypeStart, field.TypeEnd, fn.BodyEnd})
		}
	}
	for i := 0; i < len(body.Stmts); i++ {
		stmt := body.Stmts[i]
		if stmt.Kind == syntax.StmtDecl && (file.Tokens[stmt.StartTok].KindLine&255 == syntax.TokenVar || file.Tokens[stmt.StartTok].KindLine&255 == syntax.TokenConst) {
			names, start := localDeclNameTokens(*file, stmt.StartTok+1, stmt.EndTok)
			op := findDeclAssign(*file, start, stmt.EndTok)
			end := stmt.EndTok
			if op >= 0 {
				end = op
			}
			start, end = trimDeclSpan(*file, start, end)
			for _, name := range names {
				bindings = append(bindings, localRuleBinding{name, start, end, localRuleScopeEnd(*body, stmt.StartTok)})
			}
			builtin := end-start == 1 && lookupScopeTokenNameCore(scope, file, start) < 0 && lookupPackageSymbol(info.Symbols, tokenString(file, start)) < 0
			if op >= 0 && builtin && literalIntegerOverflows(*file, op+1, stmt.EndTok, tokenString(file, start)) {
				return CheckErrType, op + 1
			}
			if op >= 0 && builtin {
				rightStart, rightEnd := trimDeclSpan(*file, op+1, stmt.EndTok)
				rightStart, rightEnd = stripOuterParens(file, rightStart, rightEnd)
				declared := tokenString(file, start)
				if rightEnd-rightStart == 1 && definiteBuiltinType(declared) {
					kind := definiteLiteralKind(*file, rightStart)
					if file.Tokens[rightStart].KindLine&255 == syntax.TokenIdent && (lookupScopeTokenNameCore(scope, file, rightStart) >= 0 || lookupPackageSymbol(info.Symbols, tokenString(file, rightStart)) >= 0) {
						kind = ""
					}
					if kind != "" && kind != declared {
						return CheckErrType, rightStart
					}
				}
			}
		}
		if stmt.Kind != syntax.StmtAssign {
			continue
		}
		op := findTopLevelAssignOp(*file, stmt.StartTok, stmt.EndTok)
		if op >= 0 && tokCharIs(file, stmt.StartTok+1, '[') {
			for j := len(bindings) - 1; j >= 0; j-- {
				binding := bindings[j]
				if binding.token >= stmt.StartTok || binding.scopeEnd <= stmt.StartTok || !statementTokensEqual(file, binding.token, stmt.StartTok) {
					continue
				}
				if binding.end-binding.start == 1 && (tokenTextIs(file, binding.start, "string") && lookupScopeTokenNameCore(scope, file, binding.start) < 0 && lookupPackageSymbol(info.Symbols, "string") < 0 || file.Tokens[binding.start].KindLine&255 == syntax.TokenString) {
					return CheckErrAssignTarget, stmt.StartTok
				}
				break
			}
		}
		if op < 0 || !tokenTextIs(file, op, ":=") {
			continue
		}
		endScope := localRuleScopeEnd(*body, stmt.StartTok)
		newNames := false
		for tok := stmt.StartTok; tok < op; tok++ {
			if tokCharIs(file, tok, ',') {
				continue
			}
			if file.Tokens[tok].KindLine&255 != syntax.TokenIdent {
				return CheckErrAssignTarget, tok
			}
			if tokenTextIs(file, tok, "_") {
				continue
			}
			exists := false
			for j := len(bindings) - 1; j >= 0; j-- {
				if bindings[j].scopeEnd == endScope && statementTokensEqual(file, tok, bindings[j].token) {
					exists = true
					break
				}
			}
			for prev := stmt.StartTok; prev < tok; prev++ {
				if statementTokensEqual(file, prev, tok) {
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
		if !tokenTextIs(file, tok, "==") && !tokenTextIs(file, tok, "!=") {
			continue
		}
		if tokenTextIs(file, tok-1, "nil") || tokenTextIs(file, tok+1, "nil") {
			continue
		}
		if file.Tokens[tok-1].KindLine&255 != syntax.TokenIdent || file.Tokens[tok+1].KindLine&255 != syntax.TokenIdent {
			continue
		}
		for j := len(bindings) - 1; j >= 0; j-- {
			binding := bindings[j]
			if binding.token >= tok || binding.scopeEnd <= tok || !statementTokensEqual(file, binding.token, tok-1) {
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
