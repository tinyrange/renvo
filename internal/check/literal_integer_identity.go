package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

// LiteralIntegerConstantsEqual shares the checker's exact integer arithmetic
// with type identity consumers. References require a lexical environment and
// are deliberately left unknown by this literal-only entry point.
func LiteralIntegerConstantsEqual(left []byte, right []byte) (bool, bool) {
	a := literalIntegerConstant(left)
	b := literalIntegerConstant(right)
	if !a.ok || !b.ok {
		return false, false
	}
	return a.negative == b.negative && wideMagnitudeCompare(a, b) == 0, true
}

func literalIntegerConstant(source []byte) wideConstant {
	file := syntax.File{Src: source, Tokens: syntax.Scan(source)}
	end := len(file.Tokens)
	if end > 0 && file.Tokens[end-1].KindLine&255 == syntax.TokenEOF {
		end--
	}
	for i := 0; i < end; i++ {
		if file.Tokens[i].KindLine&255 == syntax.TokenIdent {
			return wideConstant{}
		}
	}
	pkg := load.Package{Files: []load.ParsedFile{{File: file}}}
	info := PackageInfo{}
	context := constantIndexContext{pkg: &pkg, info: &info}
	return wideConstantExpr(&context, 0, end, 0)
}
