package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

type numericBuiltinValue struct {
	kind, identity string
	typed          bool
}

func numericBuiltinInNestedFunction(file syntax.File, fn syntax.FuncDecl, callee int) bool {
	for tok := fn.BodyStart + 1; tok < callee; tok++ {
		if file.Tokens[tok].KindLine&255 != syntax.TokenFunc {
			continue
		}
		end := pointerOrderingNestedFunctionEnd(file, tok, fn.BodyEnd-1)
		if end >= callee {
			return true
		}
		tok = end
	}
	return false
}

func invalidNumericBuiltinCall(pkg load.Package, info PackageInfo, fileIndex int, scope CoreScope, bindings []scopedTypeBinding, name string, callee, close int, args []ExprSpan) (int, int) {
	file := pkg.Files[fileIndex].File
	count := 1
	if name == "complex" {
		count = 2
	}
	if len(args) != count || tokenTextIs(&file, close-2, "...") {
		return CheckErrBuiltinArity, callee
	}
	identity := ""
	for _, arg := range args {
		value := numericBuiltinExprValue(pkg, info, fileIndex, scope, bindings, arg.StartTok, arg.EndTok, callee, 0)
		if value.kind == "string" || value.kind == "bool" || value.kind == "other" {
			return CheckErrBuiltinOperand, arg.StartTok
		}
		if !value.typed || value.kind == "" {
			continue
		}
		if name != "complex" {
			if value.kind != "complex" {
				return CheckErrBuiltinOperand, arg.StartTok
			}
		} else {
			if value.kind != "float" || identity != "" && value.identity != "" && identity != value.identity {
				return CheckErrBuiltinOperand, arg.StartTok
			}
			identity = value.identity
		}
	}
	return CheckOK, -1
}

func numericBuiltinExprValue(pkg load.Package, info PackageInfo, fileIndex int, scope CoreScope, bindings []scopedTypeBinding, start, end, before, depth int) numericBuiltinValue {
	if depth > 32 {
		return numericBuiltinValue{}
	}
	file := pkg.Files[fileIndex].File
	start, end = stripOuterParens(file, start, end)
	if start < 0 || start >= end {
		return numericBuiltinValue{}
	}
	if end-start == 1 {
		kind := file.Tokens[start].KindLine & 255
		if kind == syntax.TokenString {
			return numericBuiltinValue{kind: "string"}
		}
		if kind == syntax.TokenChar {
			return numericBuiltinValue{kind: "int", identity: "int32"}
		}
		if kind == syntax.TokenNumber {
			text := tokenString(&file, start)
			if len(text) > 0 && text[len(text)-1] == 'i' {
				return numericBuiltinValue{kind: "complex", identity: "complex128"}
			}
			hex := len(text) > 1 && text[0] == '0' && (text[1] == 'x' || text[1] == 'X')
			for i := 0; i < len(text); i++ {
				if text[i] == '.' || text[i] == 'p' || text[i] == 'P' || !hex && (text[i] == 'e' || text[i] == 'E') {
					return numericBuiltinValue{kind: "float", identity: "float64"}
				}
			}
			return numericBuiltinValue{kind: "int", identity: "int"}
		}
	}
	if (tokCharIs(&file, start, '+') || tokCharIs(&file, start, '-')) && start+1 < end {
		return numericBuiltinExprValue(pkg, info, fileIndex, scope, bindings, start+1, end, before, depth+1)
	}
	if start+1 < end && tokCharIs(&file, start+1, '(') && findTypeMatching(file, start+1, '(', ')') == end {
		return numericBuiltinTypeValue(pkg, info, fileIndex, scope, start, start+1, 0)
	}
	if end-start != 1 || file.Tokens[start].KindLine&255 != syntax.TokenIdent {
		return numericBuiltinValue{}
	}
	chosen := -1
	for i, binding := range bindings {
		if binding.visible <= before && before < binding.end && coreTokensEqual(&file, binding.name, start) && (chosen < 0 || binding.visible > bindings[chosen].visible) {
			chosen = i
		}
	}
	if chosen >= 0 {
		binding := bindings[chosen]
		if binding.typeEnd > binding.typeStart {
			return numericBuiltinTypeValue(pkg, info, fileIndex, scope, binding.typeStart, binding.typeEnd, 0)
		}
		value := numericBuiltinExprValue(pkg, info, fileIndex, scope, bindings, binding.valueStart, binding.valueEnd, binding.name, depth+1)
		if binding.writable {
			value.typed = true
		}
		return value
	}
	name := tokenString(&file, start)
	if (name == "true" || name == "false" || name == "nil") && lookupScopeTokenNameCore(scope, &file, start) < 0 && LookupPackageSymbol(info, name) < 0 {
		if name != "nil" {
			return numericBuiltinValue{kind: "bool"}
		}
		return numericBuiltinValue{kind: "other"}
	}
	for _, decl := range info.Decls {
		if decl.Name != name || (decl.Kind != SymbolVar && decl.Kind != SymbolConst) {
			continue
		}
		if decl.TypeEnd > decl.TypeStart {
			return numericBuiltinTypeValue(pkg, info, decl.File, CoreScope{}, decl.TypeStart, decl.TypeEnd, 0)
		}
		values := splitExprList(pkg.Files[decl.File].File, decl.ValueStart, decl.ValueEnd)
		if decl.ValueIndex >= 0 && decl.ValueIndex < len(values) {
			span := values[decl.ValueIndex]
			value := numericBuiltinExprValue(pkg, info, decl.File, CoreScope{}, nil, span.StartTok, span.EndTok, decl.Token, depth+1)
			if decl.Kind == SymbolVar {
				value.typed = true
			}
			return value
		}
	}
	return numericBuiltinValue{}
}

func numericBuiltinTypeValue(pkg load.Package, info PackageInfo, fileIndex int, scope CoreScope, start, end, depth int) numericBuiltinValue {
	if depth > len(info.Types)+1 || start < 0 || start >= end {
		return numericBuiltinValue{}
	}
	file := pkg.Files[fileIndex].File
	start, end = stripOuterParens(file, start, end)
	if end-start != 1 {
		return numericBuiltinValue{}
	}
	if lookupScopeTokenNameCore(scope, &file, start) >= 0 {
		return numericBuiltinValue{}
	}
	name := tokenString(&file, start)
	index := LookupType(info, name)
	if index >= 0 {
		typ := info.Types[index]
		value := numericBuiltinTypeValue(pkg, info, typ.File, CoreScope{}, typ.TypeStart, typ.TypeEnd, depth+1)
		if !typ.Alias {
			value.identity = "named:" + name
		}
		return value
	}
	if LookupPackageSymbol(info, name) >= 0 {
		return numericBuiltinValue{}
	}
	value := numericBuiltinValue{identity: name, typed: true}
	if name == "string" || name == "bool" {
		value.kind = name
	} else if name == "float32" || name == "float64" {
		value.kind = "float"
	} else if name == "complex64" || name == "complex128" {
		value.kind = "complex"
	} else if definitePrimitiveTypeCode(name) == definitePrimitiveInt || name == "byte" || name == "rune" {
		value.kind = "int"
	}
	return value
}
