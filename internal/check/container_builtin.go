package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

func invalidAppendOperands(pkg load.Package, info PackageInfo, fileIndex int, scope CoreScope, bindings []scopedTypeBinding, before int, args []ExprSpan, expanded bool) int {
	file := pkg.Files[fileIndex].File
	count := 1
	if expanded {
		count = 2
	}
	for i := 0; i < count; i++ {
		arg := args[i]
		end := arg.EndTok
		if expanded && i == 1 {
			end-- // exclude the ellipsis, which is not part of the source value
		}
		value := numericBuiltinExprValue(pkg, info, fileIndex, scope, bindings, arg.StartTok, end, before, 0)
		start, finish := stripOuterParens(file, arg.StartTok, end)
		if i == 1 && finish-start == 1 && tokenTextIs(&file, start, "nil") && value.kind == "other" {
			continue
		}
		if i == 1 && value.kind == "string" {
			continue // element compatibility is checked below
		}
		if value.kind != "" {
			return arg.StartTok
		}
		kind := containerBuiltinExprKind(pkg, info, fileIndex, scope, bindings, arg.StartTok, end, before, 0)
		if kind != 0 && kind != TypeSlice {
			return arg.StartTok
		}
	}
	if expanded {
		return invalidSliceTransferElements(pkg, info, fileIndex, scope, bindings, before, args[0], args[1], args[1].EndTok-1)
	}
	return -1
}

func invalidCopyDeleteOperands(pkg load.Package, info PackageInfo, fileIndex int, scope CoreScope, bindings []scopedTypeBinding, name string, before int, args []ExprSpan) int {
	file := pkg.Files[fileIndex].File
	count := 2
	if name == "delete" {
		count = 1
	}
	for i := 0; i < count; i++ {
		arg := args[i]
		value := numericBuiltinExprValue(pkg, info, fileIndex, scope, bindings, arg.StartTok, arg.EndTok, before, 0)
		if name == "copy" && i == 1 && value.kind == "string" {
			continue
		}
		if value.kind != "" {
			return arg.StartTok
		}
		kind := containerBuiltinExprKind(pkg, info, fileIndex, scope, bindings, arg.StartTok, arg.EndTok, before, 0)
		want := TypeSlice
		if name == "delete" {
			want = TypeMap
		}
		if kind != 0 && kind != want {
			return arg.StartTok
		}
	}
	if name == "delete" {
		shape := mapIndexExprShape(pkg, info, fileIndex, scope, bindings, args[0].StartTok, args[0].EndTok, before, 0)
		if mapLiteralPrimitiveMismatch(file, args[1].StartTok, args[1].EndTok, shape.key) {
			return args[1].StartTok
		}
	}
	if name == "copy" {
		return invalidSliceTransferElements(pkg, info, fileIndex, scope, bindings, before, args[0], args[1], args[1].EndTok)
	}
	return -1
}

// Reuse allocation type families to distinguish slices/maps/channels from
// definite non-container operands. Zero remains unknown, not rejection proof.
func containerBuiltinExprKind(pkg load.Package, info PackageInfo, fileIndex int, scope CoreScope, bindings []scopedTypeBinding, start, end, before, depth int) int {
	if depth > 32 || start < 0 || start >= end {
		return 0
	}
	file := pkg.Files[fileIndex].File
	start, end = stripOuterParens(file, start, end)
	if tokCharIs(&file, end-1, '}') {
		for open := end - 2; open >= start; open-- {
			if tokCharIs(&file, open, '{') && findTypeMatching(file, open, '{', '}') == end {
				return makeAllocationType(pkg, info, fileIndex, start, open, scope, 0)
			}
		}
	}
	if tokenTextIs(&file, start, "make") && start+1 < end && tokCharIs(&file, start+1, '(') && findTypeMatching(file, start+1, '(', ')') == end && lookupScopeTokenNameCore(scope, &file, start) < 0 && LookupPackageSymbol(info, "make") < 0 {
		return makeAllocationType(pkg, info, fileIndex, start+2, nextTopLevelComma(file, start+2, end-1), scope, 0)
	}
	if end-start != 1 || file.Tokens[start].KindLine&255 != syntax.TokenIdent {
		return 0
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
			return makeAllocationType(pkg, info, fileIndex, binding.typeStart, binding.typeEnd, scope, 0)
		}
		return containerBuiltinExprKind(pkg, info, fileIndex, scope, bindings, binding.valueStart, binding.valueEnd, binding.name, depth+1)
	}
	for _, decl := range info.Decls {
		if decl.Name != tokenString(&file, start) || decl.Kind != SymbolVar {
			continue
		}
		if decl.TypeEnd > decl.TypeStart {
			return makeAllocationType(pkg, info, decl.File, decl.TypeStart, decl.TypeEnd, CoreScope{}, 0)
		}
		values := splitExprList(pkg.Files[decl.File].File, decl.ValueStart, decl.ValueEnd)
		if decl.ValueIndex >= 0 && decl.ValueIndex < len(values) {
			value := values[decl.ValueIndex]
			return containerBuiltinExprKind(pkg, info, decl.File, CoreScope{}, nil, value.StartTok, value.EndTok, decl.Token, depth+1)
		}
	}
	return 0
}
