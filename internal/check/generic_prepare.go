package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

type genericReplacement struct {
	start int
	end   int
	text  string
}

type genericPreparedFile struct {
	changes []genericReplacement
	added   string
}

type genericPreparedPackage struct {
	files []genericPreparedFile
}

type GenericPreparation struct {
	Graph        load.Graph
	Ok           bool
	ErrorPackage int
	ErrorFile    int
	ErrorToken   int
	Message      string
}

// PrepareGenerics elaborates the source graph before ordinary concrete
// checking. The loader has already checked user import cycles. Compiler-owned
// type references may point back to a caller's package without creating an
// initialization dependency; all generated code remains in its owner's package.
func PrepareGenerics(graph load.Graph) GenericPreparation {
	out := GenericPreparation{Graph: graph, Ok: true, ErrorPackage: -1, ErrorFile: -1, ErrorToken: -1}
	needed := false
	for p := range graph.Packages {
		pkg := &graph.Packages[p]
		for f := range pkg.Files {
			file := &pkg.Files[f]
			if file.File.Generics != nil {
				needed = true
			}
		}
	}
	if !needed {
		// Type-set interfaces are declarations even without an instantiation.
		// Check them without rewriting an otherwise nongeneric source graph.
		if graphHasInterfaceElements(&graph) {
			e := newGenericEnvironment(&graph)
			e.validateInterfaceDeclarations()
			if e.errorText != "" {
				out.Ok = false
				out.ErrorPackage, out.ErrorFile, out.ErrorToken, out.Message = e.errorPkg, e.errorFile, e.errorToken, e.errorText
			}
		}
		return out
	}
	e := newGenericEnvironment(&graph)
	e.validatePackageNames()
	s := genericSpecializer{environment: e}
	prepared := make([]genericPreparedPackage, len(graph.Packages))
	for p := 0; p < len(graph.Packages); p++ {
		prepared[p].files = make([]genericPreparedFile, len(graph.Packages[p].Files))
	}
	for i := range e.decls {
		d := &e.decls[i]
		if len(d.parameters) == 0 || d.owner != 0 {
			continue
		}
		e.resolveDeclaration(i)
		s.validateDefinition(i)
		file := &graph.Packages[d.pkg].Files[d.file].File
		start, end := d.token, d.end
		if d.kind != SymbolType {
			start, end = d.function.StartTok, d.function.EndTok
		} else if start > 0 && file.Tokens[start-1].KindLine&255 == syntax.TokenType {
			start--
		}
		if end < len(file.Tokens) && tokenTextIs(file, end, ";") {
			end++
		}
		prepared[d.pkg].files[d.file].changes = append(prepared[d.pkg].files[d.file].changes, genericReplacement{start: int(file.Tokens[start].Start), end: int(file.Tokens[end-1].End)})
	}
	for i := range e.decls {
		if e.decls[i].owner != 0 {
			e.resolveDeclaration(i)
		}
	}
	e.checkInstantiationCycles()
	e.checkSizedTypes()
	e.validateInterfaceDeclarations()
	e.validateTypeUses()
	if e.errorText != "" {
		out.Ok = false
		out.ErrorPackage, out.ErrorFile, out.ErrorToken, out.Message = e.errorPkg, e.errorFile, e.errorToken, e.errorText
		return out
	}
	for p := 0; p < len(graph.Packages); p++ {
		for f := 0; f < len(graph.Packages[p].Files); f++ {
			file := &graph.Packages[p].Files[f].File
			for _, fn := range file.Funcs {
				declaration := e.functionDeclaration(p, f, fn.NameTok)
				if declaration >= 0 && len(e.decls[declaration].parameters) > 0 {
					continue
				}
				ctx := newGenericExpressionContext(&s, genericTypeScope{pkg: p, file: f}, fn, nil)
				if fn.BodyStart >= 0 {
					// Ordinary declarations can use generic types and functions,
					// even when the linker will never reach their bodies. Check
					// those bodies with the same type identities and operations
					// used for generic definitions before specializing their uses.
					ctx.scan(fn.NameTok+1, fn.BodyStart+1)
					ctx.validateBody(e.types.get(e.signature(ctx.scope, buildFuncSignature(file, &fn))))
				} else {
					ctx.scan(fn.NameTok+1, fn.EndTok)
				}
				prepared[p].files[f].changes = append(prepared[p].files[f].changes, ctx.changes...)
			}
			for _, decl := range file.Decls {
				if len(syntax.TypeParameters(file, decl.NameTok).Parameters) != 0 {
					continue
				}
				ctx := genericExpressionContext{specializer: &s, scope: genericTypeScope{pkg: p, file: f}, specialize: true}
				ctx.scan(decl.NameTok+1, decl.EndTok)
				prepared[p].files[f].changes = append(prepared[p].files[f].changes, ctx.changes...)
			}
		}
	}
	for i := 0; i < len(s.instances); i++ {
		instance := s.instances[i]
		d := &e.decls[instance.declaration]
		text := s.declarationText(i)
		if text == "" {
			e.fail(genericTypeScope{pkg: d.pkg, file: d.file}, d.token, "cannot specialize declaration")
		}
		file := &prepared[d.pkg].files[d.file]
		file.added = file.added + "\n" + text
	}
	for _, bridge := range s.bridges {
		file := &prepared[bridge.pkg].files[0]
		file.added += "\n"
		if bridge.embeddedName != "" {
			file.added += "//renvo:typename " + genericQuoted(bridge.embeddedName) + " " + genericQuoted(bridge.embeddedName) + "\n"
		}
		file.added = file.added + "type " + bridge.name + " = " + bridge.target + "\n"
	}
	e.validateTypeUses()
	if e.errorText != "" {
		out.Ok = false
		out.ErrorPackage, out.ErrorFile, out.ErrorToken, out.Message = e.errorPkg, e.errorFile, e.errorToken, e.errorText
		return out
	}
	// Clone the graph's slices so callers retain their original source and
	// token coordinates for diagnostics and incremental cache identities.
	out.Graph.Packages = make([]load.Package, len(graph.Packages))
	copy(out.Graph.Packages, graph.Packages)
	if len(s.builtins) > 0 {
		// Append the declaration-only package to preserve original package
		// indices for diagnostics. Headers are checked for the entire graph
		// before any body; these aliases have no initialization dependency.
		out.Graph.Packages = append(out.Graph.Packages, s.builtinPackage())
	}
	for p := 0; p < len(graph.Packages); p++ {
		out.Graph.Packages[p].Files = make([]load.ParsedFile, len(graph.Packages[p].Files))
		copy(out.Graph.Packages[p].Files, graph.Packages[p].Files)
		for f := 0; f < len(graph.Packages[p].Files); f++ {
			changes := prepared[p].files[f]
			changes.changes = e.appendArrayLengthChanges(p, f, 0, len(graph.Packages[p].Files[f].Src), changes.changes)
			if len(changes.changes) == 0 && changes.added == "" {
				continue
			}
			original := &graph.Packages[p].Files[f]
			source := applyGenericReplacements(original.Src, changes.changes) + changes.added
			// The inserted aliases are selected from this file's actual uses.
			parsed := syntax.ParseFile([]byte(source))
			imports := ""
			for _, imp := range s.imports {
				if imp.into != p || !genericFileUsesName(&parsed, imp.alias) {
					continue
				}
				imports += "\nimport " + imp.alias + " \"" + out.Graph.Packages[imp.from].Ref.ImportPath + "\"\n"
			}
			if imports != "" {
				offset := int(parsed.Tokens[parsed.PackageName].End)
				source = source[:offset] + "\n" + imports + source[offset:]
				parsed = syntax.ParseFile([]byte(source))
			}
			var unused []genericReplacement
			for _, imp := range original.File.Imports {
				path := tokenString(&original.File, imp.PathTok)
				alias := tokenString(&original.File, imp.NameTok)
				if imp.NameTok < 0 {
					for p := range graph.Packages {
						pkg := &graph.Packages[p]
						if path == "\""+pkg.Ref.ImportPath+"\"" {
							alias = pkg.Name
						}
					}
				}
				if alias == "" || alias == "_" || alias == "." || !genericFileUsesSelector(&original.File, alias) || genericFileUsesSelector(&parsed, alias) {
					continue
				}
				for _, next := range parsed.Imports {
					if tokenString(&parsed, next.PathTok) != path {
						continue
					}
					if next.NameTok >= 0 && tokenString(&parsed, next.NameTok) != alias {
						continue
					}
					if next.NameTok < 0 {
						unused = append(unused, genericReplacement{start: int(parsed.Tokens[next.PathTok].Start), end: int(parsed.Tokens[next.PathTok].Start), text: "_ "})
					} else {
						unused = append(unused, genericReplacement{start: int(parsed.Tokens[next.NameTok].Start), end: int(parsed.Tokens[next.NameTok].End), text: "_"})
					}
				}
			}
			if len(unused) > 0 {
				source = applyGenericReplacements([]byte(source), unused)
				parsed = syntax.ParseFile([]byte(source))
			}
			if !parsed.Ok {
				out.Ok = false
				out.ErrorPackage = p
				out.ErrorFile = f
				out.Message = "invalid specialized source"
				return out
			}
			next := &out.Graph.Packages[p].Files[f]
			next.Src, next.File = []byte(source), parsed
			next.ArenaStart, next.ArenaEnd = 0, 0
		}
	}
	return out
}

func genericFileUsesSelector(file *syntax.File, name string) bool {
	for i := 0; i+1 < len(file.Tokens); i++ {
		if file.Tokens[i].KindLine&255 == syntax.TokenIdent && tokCharIs(file, i+1, '.') && tokenString(file, i) == name {
			return true
		}
	}
	return false
}

func genericFileUsesName(file *syntax.File, name string) bool {
	for i, tok := range file.Tokens {
		if tok.KindLine&255 == syntax.TokenIdent && tokenString(file, i) == name {
			return true
		}
	}
	return false
}

func newGenericExpressionContext(s *genericSpecializer, scope genericTypeScope, fn syntax.FuncDecl, arguments []int) genericExpressionContext {
	file := &s.environment.graph.Packages[scope.pkg].Files[scope.file].File
	argSignature := buildFuncSignature(file, &fn)
	body := syntax.ParseFuncBodyStatements(*file, fn)
	context := genericExpressionContext{specializer: s, scope: scope, arguments: arguments, function: fn, bindings: collectScopedTypeBindings(file, &fn, &body, &argSignature), specialize: true}
	for i := 0; i < len(context.bindings); {
		binding := context.bindings[i]
		local := s.environment.localType(scope, tokenString(file, binding.name), binding.name)
		if local >= 0 && s.environment.decls[local].token == binding.name {
			context.bindings = append(context.bindings[:i], context.bindings[i+1:]...)
		} else {
			i++
		}
	}
	if len(argSignature.Receiver) == 1 {
		e := s.environment
		if d := e.functionDeclaration(scope.pkg, scope.file, fn.NameTok); d >= 0 && e.decls[d].receiver > 0 {
			context.receiverName = argSignature.Receiver[0].NameTok
			context.receiverType = e.receiverType(&e.decls[d], arguments)
		}
	}
	for _, stmt := range body.Stmts {
		if stmt.Kind == syntax.StmtAssign || stmt.Kind == syntax.StmtDecl || stmt.Kind == syntax.StmtCase ||
			stmt.Kind == syntax.StmtIf || stmt.Kind == syntax.StmtFor || stmt.Kind == syntax.StmtSwitch {
			start, finish := stmt.StartTok, stmt.EndTok
			if stmt.Kind == syntax.StmtDecl {
				start++
				if tokCharIs(file, start, '(') {
					close := findTypeMatching(file, start, '(', ')') - 1
					for pos := start + 1; pos < close; {
						pos = skipLocalSeparators(file, pos, close)
						if pos >= close {
							break
						}
						end := statementSpecEnd(file, pos, close)
						context.collectTupleBindings(file, pos, end)
						pos = end
					}
					continue
				}
			}
			if stmt.Kind == syntax.StmtCase {
				start, finish = stmt.ExprStart, stmt.ExprEnd
			}
			if stmt.Kind == syntax.StmtIf || stmt.Kind == syntax.StmtFor || stmt.Kind == syntax.StmtSwitch {
				start, finish = stmt.StartTok+1, stmt.BodyStart
				if semi := findTypeTopLevelChar(file, start, finish, ';'); semi >= 0 {
					finish = semi
				}
			}
			context.collectTupleBindings(file, start, finish)
		}
		if stmt.Kind != syntax.StmtFor {
			continue
		}
		if i := findTypeTopLevelKind(file, stmt.StartTok+1, stmt.BodyStart, syntax.TokenRange); i >= 0 && tokenTextIs(file, i-1, ":=") {
			names := splitExprList(file, stmt.StartTok+1, i-1)
			for n, span := range names {
				if span.EndTok == span.StartTok+1 {
					context.ranges = append(context.ranges, genericRangeBinding{name: span.StartTok, start: stmt.BodyStart, end: stmt.EndTok, expressionStart: i + 1, expressionEnd: stmt.BodyStart, index: n})
				}
			}
		}
	}
	context.switchBindings(&body)
	return context
}

func (c *genericExpressionContext) collectTupleBindings(file *syntax.File, start int, end int) {
	op := findTopLevelAssignOp(file, start, end)
	if op < 0 {
		return
	}
	names, _ := localDeclNameTokens(file, start, op)
	values := splitExprList(file, op+1, end)
	if len(names) > 1 && len(values) == 1 && file.Tokens[op+1].KindLine&255 != syntax.TokenRange {
		for i, name := range names {
			c.tuples = append(c.tuples, genericTupleBinding{name: name, start: values[0].StartTok, end: values[0].EndTok, index: i, count: len(names)})
		}
	}
}

func (c *genericExpressionContext) replace(start int, end int, text string) {
	file := &c.specializer.environment.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	if start < 0 || end <= start || end > len(file.Tokens) {
		return
	}
	change := genericReplacement{start: int(file.Tokens[start].Start), end: int(file.Tokens[end-1].End), text: text}
	for _, prior := range c.changes {
		if prior.start == change.start && prior.end == change.end {
			return
		}
	}
	c.changes = append(c.changes, change)
}

func (c *genericExpressionContext) scan(start int, end int) {
	e := c.specializer.environment
	file := &e.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	if c.specialize {
		for _, binding := range c.bindings {
			if binding.valueStart >= start && binding.valueEnd <= end && binding.valueStart < binding.valueEnd && file.Tokens[binding.valueStart].KindLine&255 != syntax.TokenRange && c.containsFunctionValue(binding.valueStart, binding.valueEnd) {
				c.expression(binding.valueStart, binding.valueEnd, binding.name)
			}
		}
		for declIndex := range e.decls {
			d := &e.decls[declIndex]
			if d.owner != 0 && !d.alias && d.pkg == c.scope.pkg && d.file == c.scope.file && start <= d.token && d.end <= end {
				id := c.typeSpan(d.token, d.token+1)
				c.replace(d.token, d.end, d.name+" = "+c.scopedTypeText(id, d.token))
			}
		}
	}
	for i := start; i < end; i++ {
		if file.Tokens[i].KindLine&255 == syntax.TokenFunc && i > c.function.BodyStart {
			fn := genericFunctionLiteral(file, i, end)
			if fn.BodyStart >= 0 && fn.BodyEnd > fn.BodyStart {
				c.scanClosure(fn)
				i = fn.BodyEnd - 1
				continue
			}
		}
		if c.replacedToken(i) {
			continue
		}
		if tokCharIs(file, i, '.') && i > start && i+1 < end && file.Tokens[i+1].KindLine&255 == syntax.TokenIdent {
			base := genericPrimaryStart(file, i-1)
			if base >= start && c.expressionType(base, i) != 0 {
				c.expression(base, i+2, base)
			}
		}
		if tokCharIs(file, i, '(') && i > c.function.BodyStart {
			close := findTypeMatching(file, i, '(', ')')
			callee := genericPrimaryStart(file, i-1)
			if callee >= start && close > i && close <= end && c.containsFunctionValue(i+1, close-1) {
				c.call(callee, i, close, i)
			}
		}
		if c.specialize && file.Tokens[i].KindLine&255 == syntax.TokenIdent && (tokCharIs(file, i-1, '.') || tokCharIs(file, i+1, ':')) {
			c.rewriteEmbeddedField(i)
		}
		if file.Tokens[i].KindLine&255 != syntax.TokenIdent || i > start && tokCharIs(file, i-1, '.') {
			continue
		}
		d, next := c.declaration(i, end)
		if d < 0 {
			name := tokenString(file, i)
			if c.specialize && (genericBasicName(name) || name == "any" || name == "error") {
				c.rewriteEmbeddedType(i)
			}
			continue
		}
		if c.specialize && e.decls[d].kind == SymbolType && c.rewriteEmbeddedType(i) {
			continue
		}
		if e.decls[d].owner != 0 {
			if c.specialize && i != e.decls[d].token && !tokCharIs(file, next, ':') && !genericTypeMemberName(file, i) {
				id := c.typeSpan(i, next)
				c.replace(i, next, c.scopedTypeText(id, i))
			}
			continue
		}
		if len(e.decls[d].parameters) == 0 {
			if e.decls[d].kind == SymbolFunc && tokCharIs(file, next, '(') {
				close := findTypeMatching(file, next, '(', ')')
				if close > next && c.containsFunctionValue(next+1, close-1) {
					c.call(i, next, close, i)
				}
			}
			continue
		}
		if tokCharIs(file, next, '[') {
			close := findTypeMatching(file, next, '[', ']')
			if close <= next || close > end {
				continue
			}
			if e.decls[d].kind == SymbolType {
				id := c.typeSpan(i, close)
				if c.specialize {
					c.replace(i, close, c.scopedTypeText(id, i))
				}
				continue
			}
			if tokCharIs(file, close, '(') {
				callEnd := findTypeMatching(file, close, '(', ')')
				if callEnd > close {
					c.call(i, close, callEnd, i)
				}
			} else {
				if expected := c.expectedValueType(i, close); expected != 0 {
					c.functionValue(d, i, close, expected)
				} else {
					c.genericCall(d, i, next, close, nil, false, false)
				}
			}
		} else if tokCharIs(file, next, '(') {
			callEnd := findTypeMatching(file, next, '(', ')')
			if callEnd > next {
				c.call(i, next, callEnd, i)
			}
		} else if expected := c.expectedValueType(i, next); expected != 0 {
			c.functionValue(d, i, next, expected)
		}
	}
}

func applyGenericReplacements(source []byte, changes []genericReplacement) string {
	changes = append([]genericReplacement(nil), changes...)
	for i := 1; i < len(changes); i++ {
		v := changes[i]
		j := i
		for j > 0 && (changes[j-1].start > v.start || changes[j-1].start == v.start && changes[j-1].end < v.end) {
			changes[j] = changes[j-1]
			j--
		}
		changes[j] = v
	}
	out := ""
	position := 0
	for _, change := range changes {
		if change.start < position || change.end > len(source) {
			continue
		}
		out += string(source[position:change.start]) + change.text
		position = change.end
	}
	return out + string(source[position:])
}
