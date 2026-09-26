package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

func invalidMapIndexType(pkg *load.Package, info *PackageInfo, fileIndex int, fn syntax.FuncDecl, body *syntax.Body, signature *FuncSignature, scope CoreScope, cachedBindings *[]scopedTypeBinding, indexes []IndexExpr) (int, int) {
	file := &pkg.Files[fileIndex].File
	if len(indexes) == 0 {
		return CheckOK, -1
	}
	bindings := *cachedBindings
	if bindings == nil {
		bindings = collectScopedTypeBindings(file, fn, body, signature)
		*cachedBindings = bindings
	}
	hasNested := false
	for tok := fn.BodyStart + 1; tok < fn.BodyEnd; tok++ {
		if file.Tokens[tok].KindLine&255 == syntax.TokenFunc {
			hasNested = true
			break
		}
	}
	for indexPosition := 0; indexPosition < len(indexes); indexPosition++ {
		index := &indexes[indexPosition]
		start, end := stripOuterParens(file, index.BaseStart, index.BaseEnd)
		// The current definite resolvers do not infer selector result types.
		if end-start == 3 && tokCharIs(file, start+1, '.') {
			continue
		}
		if mapIndexExplicitSequence(file, bindings, index) {
			continue
		}
		if !hasNested || !numericBuiltinInNestedFunction(*file, fn, index.OpenTok) {
			value := numericBuiltinExprValue(pkg, info, fileIndex, scope, bindings, index.BaseStart, index.BaseEnd, index.OpenTok, 0)
			if value.kind != "" && value.kind != "string" || definiteStructExpr(pkg, info, fileIndex, scope, bindings, index.BaseStart, index.BaseEnd, index.OpenTok, 0) || containerBuiltinExprKind(pkg, info, fileIndex, scope, bindings, index.BaseStart, index.BaseEnd, index.OpenTok, 0) == TypeChan {
				return CheckErrOperand, index.OpenTok
			}
		}
		shape := mapIndexExprShape(pkg, info, fileIndex, scope, bindings, index.BaseStart, index.BaseEnd, index.OpenTok, 0)
		if mapLiteralPrimitiveMismatch(file, index.IndexStart, index.IndexEnd, shape.key) {
			return CheckErrType, index.IndexStart
		}
		if tok := invalidMapElementFieldWrite(pkg, info, file, body, index, shape); tok >= 0 {
			return CheckErrAssignTarget, tok
		}
	}
	return CheckOK, -1
}

// Explicit array and slice bindings cannot be scalar operands or map elements.
// Resolve the closest visible binding once before the more general resolvers.
func mapIndexExplicitSequence(file *syntax.File, bindings []scopedTypeBinding, index *IndexExpr) bool {
	start, end := stripOuterParens(file, index.BaseStart, index.BaseEnd)
	if end-start != 1 || file.Tokens[start].KindLine&255 != syntax.TokenIdent {
		return false
	}
	chosen := -1
	for i := 0; i < len(bindings); i++ {
		binding := &bindings[i]
		if binding.visible <= index.OpenTok && index.OpenTok < binding.end && coreTokensEqual(file, binding.name, start) && (chosen < 0 || binding.visible > bindings[chosen].visible) {
			chosen = i
		}
	}
	if chosen < 0 {
		return false
	}
	binding := &bindings[chosen]
	return binding.typeEnd > binding.typeStart && tokCharIs(file, binding.typeStart, '[')
}
