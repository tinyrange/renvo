package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

// Check resolved scalar and slice conversions. Unknown expression types are
// not evidence of an invalid conversion; they still require general typing.
func invalidKnownConversion(pkg load.Package, info PackageInfo, fileIndex int, fn syntax.FuncDecl, scope CoreScope) int {
	file := pkg.Files[fileIndex].File
	var bindings []scopedTypeBinding
	ready := false
	for name := fn.BodyStart + 1; name+1 < fn.BodyEnd; name++ {
		if file.Tokens[name].KindLine&255 == syntax.TokenFunc {
			name = pointerOrderingNestedFunctionEnd(file, name, fn.BodyEnd-1)
			continue
		}
		if file.Tokens[name].KindLine&255 != syntax.TokenIdent || !tokCharIs(&file, name+1, '(') {
			continue
		}
		start := name
		if name > fn.BodyStart+1 && tokCharIs(&file, name-1, ']') {
			if name < fn.BodyStart+3 || !tokCharIs(&file, name-2, '[') {
				continue // array types need their own conversion rules
			}
			start -= 2
		}
		if start > fn.BodyStart+1 && (tokCharIs(&file, start-1, '.') || tokCharIs(&file, start-1, '*') || tokCharIs(&file, start-1, ']') || file.Tokens[start-1].KindLine&255 == syntax.TokenChan || tokenTextIs(&file, start-1, "<-")) {
			continue
		}
		target := conversionUnderlyingType(pkg, info, fileIndex, scope, start, name+1, 0)
		if target == "" {
			continue
		}
		close := findTypeMatching(file, name+1, '(', ')')
		if close <= name+1 || close > fn.BodyEnd {
			continue
		}
		args := splitExprList(file, name+2, close-1)
		if len(args) != 1 || tokenTextIs(&file, close-2, "...") {
			return start
		}
		if !ready {
			bindings = collectScopedTypeBindings(file, fn, syntax.ParseFuncBodyStatements(file, fn))
			ready = true
		}
		arg := args[0]
		value := numericBuiltinExprValue(pkg, info, fileIndex, scope, bindings, arg.StartTok, arg.EndTok, start, 0)
		slice := len(target) > 6 && target[:6] == "slice:"
		if slice {
			if value.kind == "string" {
				element := target[6:]
				if element != "byte" && element != "uint8" && element != "rune" && element != "int32" {
					return start
				}
			} else if value.kind != "" && value.kind != "other" {
				return start
			}
			continue
		}
		if definiteStructExpr(pkg, info, fileIndex, scope, bindings, arg.StartTok, arg.EndTok, start, 0) {
			return start
		}
		if target == "string" {
			if value.kind == "float" || value.kind == "complex" || value.kind == "bool" || value.kind == "other" {
				return start
			}
		} else if target == "bool" {
			if value.kind != "" && value.kind != "bool" {
				return start
			}
		} else if value.kind == "string" || value.kind == "bool" || value.kind == "other" {
			return start
		}
	}
	return -1
}

// Follow underlying types (not just aliases): named byte/rune slice elements
// participate in string conversion, while named int elements do not.
func conversionUnderlyingType(pkg load.Package, info PackageInfo, fileIndex int, scope CoreScope, start, end, depth int) string {
	if depth > len(info.Types)+2 || start < 0 || start >= end {
		return ""
	}
	file := pkg.Files[fileIndex].File
	if end-start >= 3 && tokCharIs(&file, start, '[') && tokCharIs(&file, start+1, ']') {
		element := conversionUnderlyingType(pkg, info, fileIndex, scope, start+2, end, depth+1)
		if element != "" {
			return "slice:" + element
		}
		return ""
	}
	if end-start != 1 || lookupScopeTokenNameCore(scope, &file, start) >= 0 {
		return ""
	}
	name := tokenString(&file, start)
	if index := LookupType(info, name); index >= 0 {
		typ := info.Types[index]
		return conversionUnderlyingType(pkg, info, typ.File, CoreScope{}, typ.TypeStart, typ.TypeEnd, depth+1)
	}
	if LookupPackageSymbol(info, name) >= 0 {
		return ""
	}
	if name == "string" || name == "bool" || name == "int" || name == "int8" || name == "int16" || name == "int32" || name == "int64" || name == "uint" || name == "uint8" || name == "uint16" || name == "uint32" || name == "uint64" || name == "uintptr" || name == "byte" || name == "rune" || name == "float32" || name == "float64" || name == "complex64" || name == "complex128" {
		return name
	}
	return ""
}
