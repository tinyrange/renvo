package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

func invalidKnownStructSelector(pkg load.Package, info PackageInfo, fileIndex int, fn syntax.FuncDecl, body syntax.Body, literals []CompositeExpr) int {
	file := pkg.Files[fileIndex].File
	var scope CoreScope
	var bindings []scopedTypeBinding
	ready := false
	for dot := fn.BodyStart + 1; dot+1 < fn.BodyEnd; dot++ {
		if file.Tokens[dot].KindLine&255 == syntax.TokenFunc {
			dot = pointerOrderingNestedFunctionEnd(file, dot, fn.BodyEnd-1)
			continue
		}
		if !tokCharIs(&file, dot, '.') || file.Tokens[dot+1].KindLine&255 != syntax.TokenIdent {
			continue
		}
		if !ready {
			var ok bool
			scope, ok, _ = buildFuncScopeCore(file, fn)
			if !ok {
				return -1
			}
			bindings = collectScopedTypeBindings(file, fn, body)
			ready = true
		}
		var fields []Field
		known := false
		concrete := interfaceConcreteType{}
		for _, literal := range literals {
			if literal.EndTok == dot {
				fields, known = literalStructFields(pkg, info, file, literal.TypeStart, literal.TypeEnd, scope, 0)
				concrete = interfaceNamedType(pkg, info, fileIndex, scope, literal.TypeStart, literal.TypeEnd, 0)
				break
			}
		}
		if !known && file.Tokens[dot-1].KindLine&255 == syntax.TokenIdent && !tokCharIs(&file, dot-2, '.') {
			concrete = interfaceExprType(pkg, info, fileIndex, scope, bindings, dot-1, dot, dot, 0)
			if concrete.known {
				typ := info.Types[concrete.index]
				fields, known = literalStructFields(pkg, info, pkg.Files[typ.File].File, typ.TypeStart, typ.TypeEnd, CoreScope{}, 0)
			}
		}
		if !known {
			continue
		}
		name := tokenString(&file, dot+1)
		if name != "_" && LookupField(fields, name) >= 0 {
			continue
		}
		// Embedded members need full promotion/ambiguity resolution. Their
		// absence from the direct field list is not rejection evidence.
		embedded := false
		for _, field := range fields {
			if field.Name == "" {
				embedded = true
			}
		}
		if embedded || concrete.known && knownSelectorMethod(pkg, info, concrete.index, name) {
			continue
		}
		return dot + 1
	}
	return -1
}

func knownSelectorMethod(pkg load.Package, info PackageInfo, index int, name string) bool {
	for fileIndex, source := range pkg.Files {
		file := source.File
		for _, fn := range file.Funcs {
			if fn.ReceiverStart < 0 || !tokenTextIs(&file, fn.NameTok, name) {
				continue
			}
			signature := buildFuncSignature(file, fn)
			if len(signature.Receiver) != 1 {
				continue
			}
			receiver := signature.Receiver[0]
			actual := interfaceNamedType(pkg, info, fileIndex, CoreScope{}, receiver.TypeStart, receiver.TypeEnd, 0)
			if actual.known && actual.index == index {
				return true
			}
		}
	}
	return false
}
