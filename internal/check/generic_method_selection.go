package check

import "renvo.dev/internal/syntax"

// A type-parameter method value keeps the dictionary receiver. Merely
// substituting a concrete receiver can bind an embedded interface field too
// early. An ordinary method-only interface preserves that selection at call
// time and is fully concrete before the backend boundary.
func (c *genericExpressionContext) lowerGenericMethodSelection(start, end, baseType, methodType int) {
	if !c.specialize {
		return
	}
	e := c.specializer.environment
	file := &e.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	name := tokenString(file, end-1)
	if baseType != 0 {
		method := *e.types.get(methodType)
		method.params = append([]int{baseType}, method.params...)
		parameters := make([]Field, len(method.params))
		arguments := ""
		for i := range parameters {
			parameters[i].Name = "__renvo_method_argument_" + genericDecimal(i)
			if i > 0 {
				if i > 1 {
					arguments += ", "
				}
				arguments += parameters[i].Name
			}
		}
		if method.variadic {
			arguments += "..."
		}
		body := parameters[0].Name + "." + name + "(" + arguments + ")"
		if len(method.results) > 0 {
			body = "return " + body
		}
		c.replace(start, end, "(func"+c.specializer.signatureText(&method, c.scope.pkg, parameters, nil)+" { "+body+" })")
		return
	}
	if len(c.arguments) == 0 {
		return
	}
	original := *c
	original.arguments, original.specialize = nil, false
	original.changes = nil
	if original.receiverType != 0 {
		if declaration := e.functionDeclaration(c.scope.pkg, c.scope.file, c.function.NameTok); declaration >= 0 {
			original.receiverType = e.receiverType(&e.decls[declaration], nil)
		}
	}
	value := original.expression(start, end-2, start)
	if e.types.get(value.typ).kind != genericParameter {
		return
	}
	method := genericMethod{name: name, typ: methodType}
	if !syntax.IdentifierExported([]byte(name), 0) {
		method.pkg = e.graph.Packages[c.scope.pkg].Ref.ImportPath
	}
	iface := e.types.intern(genericType{kind: genericInterface, methods: []genericMethod{method}})
	first, last := int(file.Tokens[start].Start), int(file.Tokens[end-3].End)
	var changes []genericReplacement
	for _, change := range c.changes {
		if change.start >= first && change.end <= last {
			change.start -= first
			change.end -= first
			changes = append(changes, change)
		}
	}
	receiver := applyGenericReplacements(file.Src[first:last], changes)
	c.replace(start, end, "("+c.scopedTypeText(iface, start)+"("+receiver+"))."+name)
}
