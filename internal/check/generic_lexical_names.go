package check

import "renvo.dev/internal/syntax"

type genericLexicalBindings struct {
	pkg, file, function int
	bindings            []scopedTypeBinding
}

func (e *genericEnvironment) packageValueName(pkg int, name string) bool {
	for _, source := range e.graph.Packages[pkg].Files {
		for _, decl := range source.File.Decls {
			if (decl.Kind == syntax.TokenVar || decl.Kind == syntax.TokenConst) && tokenStringEquals(&source.File, decl.NameTok, name) {
				return true
			}
		}
	}
	return false
}

func (e *genericEnvironment) dotImportedValuePackage(scope genericTypeScope, name string) int {
	if !syntax.IdentifierExported([]byte(name), 0) {
		return -1
	}
	file := &e.graph.Packages[scope.pkg].Files[scope.file].File
	found := -1
	for _, imp := range file.Imports {
		if imp.NameTok < 0 || !tokCharIs(file, imp.NameTok, '.') {
			continue
		}
		path, _ := syntax.StringLiteralValue(file.Src, file.Tokens[imp.PathTok])
		for pkg := range e.graph.Packages {
			if e.graph.Packages[pkg].Ref.ImportPath != path || !e.packageValueName(pkg, name) {
				continue
			}
			if found >= 0 {
				e.fail(scope, imp.NameTok, "ambiguous dot-imported name")
				return -1
			}
			found = pkg
		}
	}
	return found
}

// Type syntax shares the lexical namespace with values. Cache each function's
// bindings, including nested literal scopes, so recursively parsed container
// types and constraints obey the same visibility intervals as expressions.
func (e *genericEnvironment) lexicalBindings(scope genericTypeScope, token int) []scopedTypeBinding {
	if scope.bindings != nil {
		return scope.bindings
	}
	file := &e.graph.Packages[scope.pkg].Files[scope.file].File
	for _, fn := range file.Funcs {
		if fn.BodyStart < 0 || token <= fn.BodyStart || token >= fn.BodyEnd {
			continue
		}
		for _, cached := range e.lexicalNames {
			if cached.pkg == scope.pkg && cached.file == scope.file && cached.function == fn.StartTok {
				return cached.bindings
			}
		}
		body := syntax.ParseFuncBodyStatements(*file, fn)
		signature := buildFuncSignature(file, &fn)
		bindings := collectScopedTypeBindings(file, &fn, &body, &signature)
		for start := fn.BodyStart + 1; start < fn.BodyEnd-1; start++ {
			if file.Tokens[start].KindLine&255 != syntax.TokenFunc {
				continue
			}
			inner := genericFunctionLiteral(file, start, fn.BodyEnd-1)
			if inner.BodyStart < 0 {
				continue
			}
			body = syntax.ParseFuncBodyStatements(*file, inner)
			signature = buildFuncSignature(file, &inner)
			bindings = append(bindings, collectScopedTypeBindings(file, &inner, &body, &signature)...)
		}
		e.lexicalNames = append(e.lexicalNames, genericLexicalBindings{pkg: scope.pkg, file: scope.file, function: fn.StartTok, bindings: bindings})
		return bindings
	}
	return nil
}

func (e *genericEnvironment) typeNameIsValue(scope genericTypeScope, token int) bool {
	file := &e.graph.Packages[scope.pkg].Files[scope.file].File
	name := tokenString(file, token)
	chosen, visible := -1, -1
	for _, binding := range e.lexicalBindings(scope, token) {
		if binding.name < 0 || binding.visible > token || token >= binding.end || !coreTokensEqual(file, binding.name, token) {
			continue
		}
		if local := e.localType(scope, name, binding.name); local >= 0 && e.decls[local].token == binding.name {
			continue // Type declarations are also represented by the binding scanner.
		}
		if binding.visible > visible {
			chosen, visible = binding.name, binding.visible
		}
	}
	if chosen < 0 {
		return false
	}
	local := e.localType(scope, name, token)
	return local < 0 || e.decls[local].token < chosen
}
