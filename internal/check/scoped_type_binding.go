package check

import "renvo.dev/internal/syntax"

// Keep visibility intervals alongside declaration spans: an inner slice binding
// must not borrow the key type of an outer map with the same name. Constants
// retain initializer spans and their specification's iota, but are not writable.
type scopedTypeBinding struct {
	name, visible, end   int
	typeStart, typeEnd   int
	valueStart, valueEnd int
	writable             bool
	constant             bool
	iotaValue            int
}

func collectScopedTypeBindings(file syntax.File, fn syntax.FuncDecl, body syntax.Body) []scopedTypeBinding {
	var bindings []scopedTypeBinding
	signature := buildFuncSignature(file, fn)
	for group := 0; group < 3; group++ {
		fields := signature.Params
		if group == 1 {
			fields = signature.Results
		}
		if group == 2 {
			fields = signature.Receiver
		}
		for _, field := range fields {
			bindings = append(bindings, scopedTypeBinding{name: field.NameTok, visible: fn.BodyStart, end: fn.BodyEnd, typeStart: field.TypeStart, typeEnd: field.TypeEnd, valueStart: -1, valueEnd: -1, writable: true})
		}
	}
	for _, stmt := range body.Stmts {
		start, end := stmt.StartTok, stmt.EndTok
		scopeEnd := localRuleScopeEnd(body, start)
		if stmt.Kind == syntax.StmtDecl {
			kind := file.Tokens[start].KindLine & 255
			start++
			constant := kind == syntax.TokenConst
			if tokCharIs(&file, start, '(') {
				ordinal, templateStart, templateEnd := 0, -1, -1
				for pos := start + 1; pos < end-1; {
					pos = skipLocalSeparators(file, pos, end-1)
					if pos >= end-1 || tokCharIs(&file, pos, ')') {
						break
					}
					finish := statementSpecEnd(file, pos, end-1)
					first := len(bindings)
					bindings = appendScopedTypeBindings(bindings, file, pos, finish, scopeEnd, kind == syntax.TokenVar, constant, false)
					if constant {
						if findTopLevelAssignOp(file, pos, finish) >= 0 {
							templateStart, templateEnd = first, len(bindings)
						} else if templateStart >= 0 && templateEnd-templateStart == len(bindings)-first {
							for i := first; i < len(bindings); i++ {
								previous := bindings[templateStart+i-first]
								bindings[i].typeStart, bindings[i].typeEnd = previous.typeStart, previous.typeEnd
								bindings[i].valueStart, bindings[i].valueEnd = previous.valueStart, previous.valueEnd
							}
						}
						for i := first; i < len(bindings); i++ {
							bindings[i].iotaValue = ordinal
						}
						ordinal++
					}
					if finish <= pos {
						break
					}
					pos = finish
				}
			} else {
				bindings = appendScopedTypeBindings(bindings, file, start, end, scopeEnd, kind == syntax.TokenVar, constant, false)
			}
			continue
		}
		if stmt.Kind == syntax.StmtIf || stmt.Kind == syntax.StmtFor || stmt.Kind == syntax.StmtSwitch {
			start++
			end = stmt.BodyStart
			scopeEnd = stmt.EndTok
			if semi := findTypeTopLevelChar(file, start, end, ';'); semi >= 0 {
				end = semi
			}
		} else if stmt.Kind == syntax.StmtCase {
			start++
		} else if stmt.Kind != syntax.StmtAssign {
			continue
		}
		op := findTopLevelAssignOp(file, start, end)
		if op >= 0 && tokenTextIs(&file, op, ":=") {
			bindings = appendScopedTypeBindings(bindings, file, start, end, scopeEnd, true, false, true)
		}
	}
	return bindings
}

func appendScopedTypeBindings(bindings []scopedTypeBinding, file syntax.File, start, end, scopeEnd int, variable, constant, short bool) []scopedTypeBinding {
	start, end = trimDeclSpan(file, start, end)
	names, namesEnd := localDeclNameTokens(file, start, end)
	op := findTopLevelAssignOp(file, start, end)
	typeStart, typeEnd := namesEnd, end
	var values []ExprSpan
	if op >= 0 {
		typeEnd = op
		values = splitExprList(file, op+1, end)
	}
	for i, name := range names {
		binding := scopedTypeBinding{name: name, visible: end, end: scopeEnd, typeStart: -1, typeEnd: -1, valueStart: -1, valueEnd: -1, writable: variable, constant: constant}
		if variable || constant {
			binding.typeStart, binding.typeEnd = typeStart, typeEnd
			if len(values) == len(names) {
				binding.valueStart, binding.valueEnd = values[i].StartTok, values[i].EndTok
			}
		}
		// A same-block short declaration reuses the existing variable.
		reused := false
		for _, old := range bindings {
			if short && old.end == scopeEnd && old.visible <= start && coreTokensEqual(&file, old.name, name) {
				reused = true
			}
		}
		if !reused {
			bindings = append(bindings, binding)
		}
	}
	return bindings
}
