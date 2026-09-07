package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

type interfaceConcreteType struct {
	index   int
	pointer bool
	known   bool
}

func invalidDefiniteInterfaceCompatibility(pkg load.Package, info PackageInfo, fileIndex int, fn syntax.FuncDecl, body syntax.Body) int {
	hasInterface := false
	for _, typ := range info.Types {
		if typ.Kind == TypeInterface {
			hasInterface = true
		}
	}
	if !hasInterface {
		return -1
	}
	file := pkg.Files[fileIndex].File
	scope, ok, _ := buildFuncScopeCore(file, fn)
	if !ok {
		return -1
	}
	bindings := collectScopedTypeBindings(file, fn, body)
	for _, binding := range bindings {
		want := interfaceNamedType(pkg, info, fileIndex, scope, binding.typeStart, binding.typeEnd, 0)
		if !want.known || want.pointer || info.Types[want.index].Kind != TypeInterface || binding.valueStart < 0 {
			continue
		}
		got := interfaceExprType(pkg, info, fileIndex, scope, bindings, binding.valueStart, binding.valueEnd, binding.name, 0)
		if definiteInterfaceMismatch(pkg, info, want.index, got) {
			return binding.valueStart
		}
	}
	for _, stmt := range body.Stmts {
		if stmt.Kind != syntax.StmtAssign {
			continue
		}
		op := findTopLevelAssignOp(file, stmt.StartTok, stmt.EndTok)
		if op < 0 || !tokenTextIs(&file, op, "=") {
			continue
		}
		left := splitExprList(file, stmt.StartTok, op)
		right := splitExprList(file, op+1, stmt.EndTok)
		if len(left) != len(right) {
			continue
		}
		for i, target := range left {
			want := interfaceExprType(pkg, info, fileIndex, scope, bindings, target.StartTok, target.EndTok, op, 0)
			if !want.known || want.pointer || info.Types[want.index].Kind != TypeInterface {
				continue
			}
			value := right[i]
			got := interfaceExprType(pkg, info, fileIndex, scope, bindings, value.StartTok, value.EndTok, op, 0)
			if definiteInterfaceMismatch(pkg, info, want.index, got) {
				return value.StartTok
			}
		}
	}
	for tok := fn.BodyStart + 1; tok+3 < fn.BodyEnd; tok++ {
		if file.Tokens[tok].KindLine&255 == syntax.TokenFunc {
			tok = pointerOrderingNestedFunctionEnd(file, tok, fn.BodyEnd-1)
			continue
		}
		if !tokCharIs(&file, tok, '.') || !tokCharIs(&file, tok+1, '(') {
			continue
		}
		if tok >= 2 && tokCharIs(&file, tok-2, '.') {
			continue
		}
		close := findTypeMatching(file, tok+1, '(', ')')
		if close <= tok+2 {
			continue
		}
		want := interfaceExprType(pkg, info, fileIndex, scope, bindings, tok-1, tok, tok, 0)
		if !want.known || want.pointer || info.Types[want.index].Kind != TypeInterface {
			continue
		}
		got := interfaceNamedType(pkg, info, fileIndex, scope, tok+2, close-1, 0)
		if definiteInterfaceMismatch(pkg, info, want.index, got) {
			return tok + 2
		}
	}
	return -1
}

func interfaceNamedType(pkg load.Package, info PackageInfo, fileIndex int, scope CoreScope, start, end, depth int) interfaceConcreteType {
	if depth > len(info.Types)+1 {
		return interfaceConcreteType{}
	}
	file := pkg.Files[fileIndex].File
	start, end = stripOuterParens(file, start, end)
	if start < 0 || start >= end {
		return interfaceConcreteType{}
	}
	pointer := false
	if tokCharIs(&file, start, '*') {
		pointer = true
		start++
	}
	if end-start != 1 || lookupScopeTokenNameCore(scope, &file, start) >= 0 {
		return interfaceConcreteType{}
	}
	index := LookupType(info, tokenString(&file, start))
	if index < 0 {
		return interfaceConcreteType{}
	}
	typ := info.Types[index]
	if typ.Alias {
		resolved := interfaceNamedType(pkg, info, typ.File, CoreScope{}, typ.TypeStart, typ.TypeEnd, depth+1)
		if pointer && resolved.pointer {
			return interfaceConcreteType{}
		}
		resolved.pointer = pointer || resolved.pointer
		return resolved
	}
	return interfaceConcreteType{index, pointer, true}
}

func interfaceExprType(pkg load.Package, info PackageInfo, fileIndex int, scope CoreScope, bindings []scopedTypeBinding, start, end, before, depth int) interfaceConcreteType {
	if depth > 32 {
		return interfaceConcreteType{}
	}
	file := pkg.Files[fileIndex].File
	start, end = stripOuterParens(file, start, end)
	if start < 0 || start >= end {
		return interfaceConcreteType{}
	}
	if tokCharIs(&file, start, '&') {
		typ := interfaceExprType(pkg, info, fileIndex, scope, bindings, start+1, end, before, depth+1)
		if typ.pointer {
			return interfaceConcreteType{}
		}
		typ.pointer = true
		return typ
	}
	if end-start >= 3 && tokCharIs(&file, start+1, '(') && findTypeMatching(file, start+1, '(', ')') == end {
		return interfaceNamedType(pkg, info, fileIndex, scope, start, start+1, 0)
	}
	if end-start >= 3 && tokCharIs(&file, start+1, '{') && findTypeMatching(file, start+1, '{', '}') == end {
		return interfaceNamedType(pkg, info, fileIndex, scope, start, start+1, 0)
	}
	if end-start != 1 {
		return interfaceConcreteType{}
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
			return interfaceNamedType(pkg, info, fileIndex, scope, binding.typeStart, binding.typeEnd, 0)
		}
		return interfaceExprType(pkg, info, fileIndex, scope, bindings, binding.valueStart, binding.valueEnd, binding.name, depth+1)
	}
	name := tokenString(&file, start)
	for _, decl := range info.Decls {
		if decl.Name != name || decl.Kind != SymbolVar {
			continue
		}
		return interfaceNamedType(pkg, info, decl.File, CoreScope{}, decl.TypeStart, decl.TypeEnd, 0)
	}
	return interfaceConcreteType{}
}

func definiteInterfaceMismatch(pkg load.Package, info PackageInfo, want int, got interfaceConcreteType) bool {
	if !got.known || info.Types[got.index].Kind == TypeInterface {
		return false
	}
	target := info.Types[want]
	concrete := info.Types[got.index]
	for _, required := range target.InterfaceMethods {
		found := false
		for fileIndex, source := range pkg.Files {
			file := source.File
			for _, fn := range file.Funcs {
				if fn.ReceiverStart < 0 || fn.ReceiverEnd <= fn.ReceiverStart || !tokenTextIs(&file, fn.NameTok, required.Name) {
					continue
				}
				signature := buildFuncSignature(file, fn)
				if len(signature.Receiver) != 1 {
					continue
				}
				receiver := signature.Receiver[0]
				actual := interfaceNamedType(pkg, info, fileIndex, CoreScope{}, receiver.TypeStart, receiver.TypeEnd, 0)
				if !actual.known || actual.index != got.index {
					continue
				}
				found = true
				if actual.pointer && !got.pointer {
					return true
				}
				if definiteMethodSignatureMismatch(pkg, info, target.File, required.Signature, fileIndex, signature) {
					return true
				}
			}
		}
		if !found {
			// A struct may acquire promoted methods from embedded fields.
			// Resolve that path before using an absent direct method as proof.
			if concrete.Kind == TypeStruct {
				for _, field := range concrete.Fields {
					if field.Name == "" {
						return false
					}
				}
			} else if concrete.Kind == TypeNamed {
				file := pkg.Files[concrete.File].File
				if concrete.TypeEnd-concrete.TypeStart == 1 && LookupType(info, tokenString(&file, concrete.TypeStart)) >= 0 {
					return false
				}
			}
			return true
		}
	}
	return false
}

func definiteMethodSignatureMismatch(pkg load.Package, info PackageInfo, wantFile int, want FuncSignature, gotFile int, got FuncSignature) bool {
	if len(want.Params) != len(got.Params) || len(want.Results) != len(got.Results) {
		return true
	}
	for group := 0; group < 2; group++ {
		left, right := want.Params, got.Params
		if group == 1 {
			left, right = want.Results, got.Results
		}
		for i, field := range left {
			other := right[i]
			if field.Variadic != other.Variadic {
				return true
			}
			a := interfaceSignatureTypeIdentity(pkg, info, wantFile, field.TypeStart, field.TypeEnd, 0)
			b := interfaceSignatureTypeIdentity(pkg, info, gotFile, other.TypeStart, other.TypeEnd, 0)
			if a != "" && b != "" && a != b {
				return true
			}
		}
	}
	return false
}

func interfaceSignatureTypeIdentity(pkg load.Package, info PackageInfo, fileIndex, start, end, depth int) string {
	if depth > len(info.Types)+16 || start < 0 || start >= end {
		return ""
	}
	file := pkg.Files[fileIndex].File
	if tokenTextIs(&file, start, "...") {
		start++
	}
	if tokCharIs(&file, start, '*') {
		inner := interfaceSignatureTypeIdentity(pkg, info, fileIndex, start+1, end, depth+1)
		if inner != "" {
			return "*" + inner
		}
		return ""
	}
	if end-start != 1 {
		return ""
	}
	name := tokenString(&file, start)
	index := LookupType(info, name)
	if index >= 0 {
		typ := info.Types[index]
		if typ.Alias {
			return interfaceSignatureTypeIdentity(pkg, info, typ.File, typ.TypeStart, typ.TypeEnd, depth+1)
		}
		return "named:" + name
	}
	if name == "byte" {
		return "uint8"
	}
	if name == "rune" {
		return "int32"
	}
	if definiteBuiltinType(name) || name == "float32" || name == "float64" || name == "complex64" || name == "complex128" {
		return name
	}
	return ""
}
