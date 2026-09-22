package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

func invalidStructLiterals(pkg load.Package, info PackageInfo, file syntax.File, literals []CompositeExpr, scope CoreScope) int {
	for _, literal := range literals {
		fields, known := literalStructFields(pkg, info, file, literal.TypeStart, literal.TypeEnd, scope, 0)
		if !known || len(literal.Elems) == 0 {
			continue
		}
		keyed, positional := false, false
		seen := make([]bool, len(fields))
		for _, element := range literal.Elems {
			colon := findTypeTopLevelChar(file, element.StartTok, element.EndTok, ':')
			if colon < 0 {
				positional = true
			} else {
				keyed = true
				if colon != element.StartTok+1 || file.Tokens[element.StartTok].KindLine&255 != syntax.TokenIdent {
					return element.StartTok
				}
				name := tokenString(&file, element.StartTok)
				index := LookupField(fields, name)
				if name == "_" || index < 0 || seen[index] {
					return element.StartTok
				}
				seen[index] = true
			}
			if keyed && positional {
				return element.StartTok
			}
		}
		if positional && len(literal.Elems) != len(fields) {
			return literal.OpenTok
		}
	}
	return -1
}

func literalStructFields(pkg load.Package, info PackageInfo, file syntax.File, start int, end int, scope CoreScope, depth int) ([]Field, bool) {
	if depth > len(info.Types)+1 || start < 0 || start >= end {
		return nil, false
	}
	start, end = stripOuterParens(file, start, end)
	if classifyType(file, start, end) == TypeStruct {
		open := findTypeTopLevelChar(file, start, end, '{')
		if open >= 0 && findTypeMatching(file, open, '{', '}') == end {
			return parseStructFields(file, open+1, end-1), true
		}
	}
	if end-start != 1 || file.Tokens[start].KindLine&255 != syntax.TokenIdent {
		return nil, false
	}
	// Do not mistake a shadowed package type for a local map or array type.
	if lookupScopeTokenNameCore(scope, &file, start) >= 0 {
		return nil, false
	}
	index := LookupType(info, tokenString(&file, start))
	if index < 0 {
		return nil, false
	}
	typ := info.Types[index]
	if typ.Kind == TypeStruct {
		return typ.Fields, true
	}
	if typ.Kind == TypeNamed {
		return literalStructFields(pkg, info, pkg.Files[typ.File].File, typ.TypeStart, typ.TypeEnd, CoreScope{}, depth+1)
	}
	return nil, false
}
