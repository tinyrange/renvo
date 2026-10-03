package check

// Concrete type text is normally emitted at package scope. When inserted in a
// body or closure signature, an original lexical declaration may capture any
// identifier inside it, including a container element or method result type.
// A package alias resolves the complete checked type outside that local scope.
func (s *genericSpecializer) scopedTypeText(id int, pkg int, hidden []string) string {
	text := s.typeText(id, pkg)
	for _, name := range hidden {
		if name != "" && name != "_" && genericTextUsesIdentifier(text, name) {
			return s.bridge(pkg, text)
		}
	}
	return text
}

func genericSignatureNames(signature FuncSignature) []string {
	var names []string
	for _, field := range signature.Receiver {
		names = append(names, field.Name)
	}
	for _, field := range signature.Params {
		names = append(names, field.Name)
	}
	for _, field := range signature.Results {
		names = append(names, field.Name)
	}
	return names
}

func (c *genericExpressionContext) typeNamesHiddenAt(before int) []string {
	e := c.specializer.environment
	file := &e.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	var names []string
	for _, parameter := range c.scope.parameters {
		names = append(names, e.types.get(parameter).name)
	}
	for _, binding := range c.bindings {
		if binding.name >= 0 && binding.visible <= before && before < binding.end {
			names = append(names, tokenString(file, binding.name))
		}
	}
	for _, decl := range e.decls {
		if decl.owner != 0 && decl.pkg == c.scope.pkg && decl.file == c.scope.file && decl.token <= before && before < decl.scopeEnd {
			names = append(names, decl.name)
		}
	}
	return names
}

func (c *genericExpressionContext) scopedTypeText(id int, before int) string {
	return c.specializer.scopedTypeText(id, c.scope.pkg, c.typeNamesHiddenAt(before))
}
