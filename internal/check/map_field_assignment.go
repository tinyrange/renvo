package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

func invalidMapElementFieldWrite(pkg load.Package, info PackageInfo, file syntax.File, body syntax.Body, index IndexExpr, shape mapIndexShape) int {
	if !shape.known {
		return -1
	}
	next := index.EndTok
	for next < len(file.Tokens) && tokCharIs(&file, next, ')') {
		next++
	}
	if next >= len(file.Tokens) || !tokCharIs(&file, next, '.') {
		return -1
	}
	for _, stmt := range body.Stmts {
		if stmt.Kind != syntax.StmtAssign && stmt.Kind != syntax.StmtExpr {
			continue
		}
		start, end := trimExprSpan(file, stmt.StartTok, stmt.EndTok)
		op := findTopLevelAssignOp(file, start, end)
		if op < 0 {
			if end <= start || (!tokenTextIs(&file, end-1, "++") && !tokenTextIs(&file, end-1, "--")) {
				continue
			}
			op = end - 1
		}
		if index.StartTok < start || index.EndTok >= op {
			continue
		}
		for _, target := range splitExprList(file, start, op) {
			first, last := stripOuterParens(file, target.StartTok, target.EndTok)
			baseStart, baseEnd := index.StartTok, index.EndTok
			// Parenthesizing the map element does not make it addressable.
			for first < baseStart && tokCharIs(&file, baseStart-1, '(') && tokCharIs(&file, baseEnd, ')') && findTypeMatching(file, baseStart-1, '(', ')') == baseEnd+1 {
				baseStart--
				baseEnd++
			}
			if first != baseStart || baseEnd >= last {
				continue
			}
			var fields []string
			pos := baseEnd
			for pos+1 < last && tokCharIs(&file, pos, '.') && file.Tokens[pos+1].KindLine&255 == syntax.TokenIdent {
				fields = append(fields, tokenString(&file, pos+1))
				pos += 2
			}
			if pos == last && len(fields) > 0 && mapFieldPathUnaddressable(pkg, info, shape, fields) {
				return target.StartTok
			}
		}
	}
	return -1
}

func mapFieldPathUnaddressable(pkg load.Package, info PackageInfo, shape mapIndexShape, names []string) bool {
	fileIndex, start, end, scope := shape.file, shape.valueStart, shape.valueEnd, shape.scope
	for _, name := range names {
		// A map lookup yields a non-addressable value. Selecting through a
		// pointer changes that: m[k].Pointer.Field is a valid update.
		for depth := 0; ; depth++ {
			if depth > len(info.Types)+1 || start < 0 || start >= end {
				return false
			}
			file := pkg.Files[fileIndex].File
			start, end = stripOuterParens(file, start, end)
			if tokCharIs(&file, start, '*') {
				return false
			}
			if end-start != 1 || lookupScopeTokenNameCore(scope, &file, start) >= 0 {
				break
			}
			index := LookupType(info, tokenString(&file, start))
			if index < 0 {
				break
			}
			typ := info.Types[index]
			fileIndex, start, end, scope = typ.File, typ.TypeStart, typ.TypeEnd, CoreScope{}
		}
		file := pkg.Files[fileIndex].File
		if classifyType(file, start, end) != TypeStruct {
			return false
		}
		open := findTypeTopLevelChar(file, start, end, '{')
		if open < 0 || findTypeMatching(file, open, '{', '}') != end {
			return false
		}
		fields := parseStructFields(file, open+1, end-1)
		fieldIndex := LookupField(fields, name)
		// Promoted selectors need their complete embedding path, including any
		// embedded pointers. Unknown paths must not become rejection evidence.
		if fieldIndex < 0 {
			return false
		}
		field := fields[fieldIndex]
		start, end = field.TypeStart, field.TypeEnd
	}
	return true
}
