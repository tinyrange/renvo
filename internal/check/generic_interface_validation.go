package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

func graphHasInterfaceElements(graph *load.Graph) bool {
	for pkg := range graph.Packages {
		for f := range graph.Packages[pkg].Files {
			file := &graph.Packages[pkg].Files[f].File
			for _, token := range file.InterfaceCandidates {
				if tokenTextIs(file, token, "comparable") && !tokCharIs(file, token-1, '.') && !tokCharIs(file, token+1, ':') && file.Tokens[token+1].KindLine&255 != syntax.TokenIdent {
					declarationName := false
					for _, fn := range file.Funcs {
						if fn.NameTok == token {
							declarationName = true
						}
					}
					if !declarationName {
						return true
					}
				}
				if file.Tokens[token].KindLine&255 != syntax.TokenInterface || !tokCharIs(file, token+1, '{') {
					continue
				}
				end := findTypeMatching(file, token+1, '{', '}')
				if end > token+1 {
					_, embeds := parseInterfaceElements(file, token+2, end-1)
					if len(embeds) != 0 {
						return true
					}
				}
			}
		}
	}
	return false
}

func (e *genericEnvironment) validateInterfaceDeclarations() {
	for index := range e.decls {
		e.resolveDeclaration(index)
	}
	s := genericSpecializer{environment: e}
	for pkg := range e.graph.Packages {
		for f := range e.graph.Packages[pkg].Files {
			file := &e.graph.Packages[pkg].Files[f].File
			scope := genericTypeScope{pkg: pkg, file: f}
			for _, decl := range file.Decls {
				if decl.Kind != syntax.TokenVar && decl.Kind != syntax.TokenConst {
					continue
				}
				info := buildDeclInfo(file, f, PackageInfo{}, nil, decl)
				if info.TypeStart >= 0 && info.TypeEnd > info.TypeStart && !e.valueType(e.parseType(scope, info.TypeStart, info.TypeEnd)) {
					e.fail(scope, info.TypeStart, "constraint interface cannot be used as a value type")
				}
				ctx := genericExpressionContext{specializer: &s, scope: scope}
				ctx.validateValueTypeExpression(info.ValueStart, info.ValueEnd)
			}
			for _, fn := range file.Funcs {
				if d := e.functionDeclaration(pkg, f, fn.NameTok); d >= 0 && len(e.decls[d].parameters) > 0 {
					continue
				}
				ctx := newGenericExpressionContext(&s, scope, fn, nil)
				ctx.specialize = false
				for _, binding := range ctx.bindings {
					start := binding.typeStart
					if start >= 0 && tokenTextIs(file, start, "...") {
						start++
					}
					if start >= 0 && binding.typeEnd > start && !e.valueType(ctx.typeSpan(start, binding.typeEnd)) {
						e.fail(scope, binding.typeStart, "constraint interface cannot be used as a value type")
					}
				}
				ctx.validateValueTypeBody()
			}
		}
	}
	e.validateTypeUses()
}
