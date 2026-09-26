package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

func invalidMapLiteralTypes(pkg *load.Package, info *PackageInfo, file *syntax.File, literals []CompositeExpr, scope CoreScope) int {
	for _, literal := range literals {
		key, value := mapLiteralPrimitiveTypes(pkg, info, file, literal.TypeStart, literal.TypeEnd, scope, 0)
		if key == "" && value == "" {
			continue
		}
		for _, element := range literal.Elems {
			colon := findTypeTopLevelChar(file, element.StartTok, element.EndTok, ':')
			if colon < 0 {
				continue
			}
			if mapLiteralPrimitiveMismatch(file, element.StartTok, colon, key) {
				return element.StartTok
			}
			if mapLiteralPrimitiveMismatch(file, colon+1, element.EndTok, value) {
				return colon + 1
			}
		}
	}
	return -1
}
