package rtg

import "renvo.dev/internal/syntax"

// Imported RTG fragments are virtual packages. The package name is derived
// from the file's base name and exists only while resolving authored helper
// names. Machine declaration identities and generated public export names stay
// global and stable.
type virtualPackage struct {
	Name             string
	Filename         string
	Symbols          []virtualSymbol
	StatementSymbols []virtualSymbol
}

type virtualSymbol struct {
	Local     string
	Qualified string
}

func applyVirtualPackages(document *Document) {
	if len(document.sourceMap) == 0 {
		// A standalone definition has one implicit namespace, so no symbol
		// rewriting is needed. Architecture extensions still compose normally.
		mergeArchitectureExtensions(document)
		return
	}
	for i := 0; i < len(document.Declarations); i++ {
		filename, _ := documentPosition(*document, document.Declarations[i].Start)
		document.Declarations[i].Package = virtualPackageName(filename)
		addVirtualPackage(document, filename, document.Declarations[i].Package)
	}
	if len(document.Diagnostics) != 0 {
		return
	}
	// Collect every package's own symbols before applying extensions. RTG
	// declaration order is not semantic, so an extension must inherit the same
	// helpers whether its import appears before or after it in the entrypoint.
	for i := 0; i < len(document.Declarations); i++ {
		declaration := document.Declarations[i]
		if declaration.Kind == DeclGo && declaration.Name == "backend" {
			addVirtualGoSymbols(document, declaration)
		} else if declaration.Kind == DeclArch {
			addVirtualArchitectureStatementSymbols(document, declaration)
			addVirtualSequenceSymbols(document, declaration)
		}
	}
	for i := 0; i < len(document.Declarations); i++ {
		declaration := document.Declarations[i]
		if declaration.Kind == DeclArchExtension {
			base, ok := document.Declaration(DeclArch, declaration.Name)
			if !ok {
				document.Diagnostics = append(document.Diagnostics,
					resolveDiagnostic(*document, declaration, "RTG-EXTEND-001",
						"extended architecture "+declaration.Name+" is not declared"))
				continue
			}
			addVirtualArchitectureStatementSymbolsForPackage(
				document, base, declaration.Package)
			inheritVirtualPackageSymbols(
				document, base.Package, declaration.Package)
			addVirtualSequenceSymbols(document, declaration)
		}
	}
	if len(document.Diagnostics) != 0 {
		return
	}
	for i := 0; i < len(document.Declarations); i++ {
		declaration := &document.Declarations[i]
		if declaration.Kind == DeclGo && declaration.Name == "backend" {
			declaration.GoSource = rewriteVirtualSource(
				declaration.GoSource, declaration.Package, document.packages)
		}
		rewriteVirtualStatements(
			declaration.Statements, declaration.Package, document.packages, "", false, nil)
		for field := 0; field < len(declaration.Fields); field++ {
			value := compactSource(document.Source[declaration.Fields[field].ValueStart:declaration.Fields[field].ValueEnd])
			declaration.Fields[field].Value = rewriteVirtualField(
				value, declaration.Package, document.packages)
		}
	}
	mergeArchitectureExtensions(document)
}

func addVirtualPackage(document *Document, filename string, name string) {
	for i := 0; i < len(document.packages); i++ {
		if document.packages[i].Filename == filename {
			return
		}
		if document.packages[i].Name == name {
			document.Diagnostics = append(document.Diagnostics, Diagnostic{
				Filename: filename,
				Span:     sourceSpan(nil, 0, 0),
				Code:     "RTG-IMPORT-007",
				Message: "virtual package name " + name + " is also used by " +
					document.packages[i].Filename,
			})
			return
		}
	}
	document.packages = append(document.packages, virtualPackage{
		Name: name, Filename: filename,
	})
}

func addVirtualGoSymbols(document *Document, declaration Declaration) {
	wrapped := append([]byte("package backend\n"), declaration.GoSource...)
	file := syntax.ParseFile(wrapped)
	if !file.Ok {
		return
	}
	for i := 0; i < len(file.Decls); i++ {
		addVirtualSymbol(document, declaration,
			string(syntax.TokenText(wrapped, file.Tokens[file.Decls[i].NameTok])))
	}
	for i := 0; i < len(file.Funcs); i++ {
		if file.Funcs[i].ReceiverStart >= 0 {
			continue
		}
		addVirtualSymbol(document, declaration,
			string(syntax.TokenText(wrapped, file.Tokens[file.Funcs[i].NameTok])))
	}
}

func addVirtualSequenceSymbols(document *Document, declaration Declaration) {
	sequences := architectureSequences(declaration)
	for i := 0; i < len(sequences); i++ {
		addVirtualSymbol(document, declaration, sequences[i].Name)
	}
}

func addVirtualArchitectureStatementSymbols(document *Document, declaration Declaration) {
	addVirtualArchitectureStatementSymbolsForPackage(
		document, declaration, declaration.Package)
}

func addVirtualArchitectureStatementSymbolsForPackage(
	document *Document, declaration Declaration, packageName string,
) {
	prefix := architectureLocalPrefix(declaration.Name)
	symbols := architectureSymbols(*document, declaration)
	for i := 0; i < len(symbols); i++ {
		if symbols[i].Kind == "sequence" ||
			!hasPrefixText(symbols[i].Local, prefix) ||
			len(symbols[i].Local) == len(prefix) {
			continue
		}
		local := symbols[i].Local[len(prefix):]
		allUpper := true
		for at := 0; at < len(local); at++ {
			if local[at] >= 'a' && local[at] <= 'z' {
				allUpper = false
				break
			}
		}
		data := []byte(local)
		if allUpper {
			for at := 0; at < len(data); at++ {
				if data[at] >= 'A' && data[at] <= 'Z' {
					data[at] += 'a' - 'A'
				}
			}
		} else if local[0] >= 'A' && local[0] <= 'Z' {
			data[0] += 'a' - 'A'
		}
		local = string(data)
		for pkg := 0; pkg < len(document.packages); pkg++ {
			if document.packages[pkg].Name == packageName {
				document.packages[pkg].StatementSymbols = append(
					document.packages[pkg].StatementSymbols,
					virtualSymbol{Local: local, Qualified: symbols[i].Local})
				if symbols[i].Kind == "register" ||
					symbols[i].Kind == "location" ||
					symbols[i].Kind == "condition" ||
					symbols[i].Kind == "instruction" {
					addVirtualArchitectureGoSymbol(
						&document.packages[pkg], local, symbols[i].Local)
				}
				break
			}
		}
	}
}

func inheritVirtualPackageSymbols(
	document *Document, sourcePackage string, destinationPackage string,
) {
	source := -1
	destination := -1
	for i := 0; i < len(document.packages); i++ {
		if document.packages[i].Name == sourcePackage {
			source = i
		}
		if document.packages[i].Name == destinationPackage {
			destination = i
		}
	}
	if source < 0 || destination < 0 || source == destination {
		return
	}
	for i := 0; i < len(document.packages[source].Symbols); i++ {
		symbol := document.packages[source].Symbols[i]
		if !virtualSymbolLocalExists(document.packages[destination].Symbols, symbol.Local) {
			document.packages[destination].Symbols = append(
				document.packages[destination].Symbols, symbol)
		}
	}
	for i := 0; i < len(document.packages[source].StatementSymbols); i++ {
		symbol := document.packages[source].StatementSymbols[i]
		if !virtualSymbolLocalExists(
			document.packages[destination].StatementSymbols, symbol.Local,
		) {
			document.packages[destination].StatementSymbols = append(
				document.packages[destination].StatementSymbols, symbol)
		}
	}
}

func virtualSymbolLocalExists(symbols []virtualSymbol, local string) bool {
	for i := 0; i < len(symbols); i++ {
		if symbols[i].Local == local {
			return true
		}
	}
	return false
}

func mergeArchitectureExtensions(document *Document) {
	for extensionIndex := 0; extensionIndex < len(document.Declarations); extensionIndex++ {
		extension := document.Declarations[extensionIndex]
		if extension.Kind != DeclArchExtension {
			continue
		}
		sequenceBlock := -1
		valid := len(extension.Fields) == 0
		for statement := 0; statement < len(extension.Statements); statement++ {
			if statementBlockName(extension.Statements[statement]) != "sequences" ||
				sequenceBlock >= 0 {
				valid = false
				break
			}
			sequenceBlock = statement
		}
		if !valid || sequenceBlock < 0 {
			document.Diagnostics = append(document.Diagnostics,
				resolveDiagnostic(*document, extension, "RTG-EXTEND-002",
					"architecture extensions require exactly one sequences block"))
			continue
		}
		baseIndex := -1
		for declaration := 0; declaration < len(document.Declarations); declaration++ {
			if document.Declarations[declaration].Kind == DeclArch &&
				document.Declarations[declaration].Name == extension.Name {
				baseIndex = declaration
				break
			}
		}
		if baseIndex < 0 {
			document.Diagnostics = append(document.Diagnostics,
				resolveDiagnostic(*document, extension, "RTG-EXTEND-001",
					"extended architecture "+extension.Name+" is not declared"))
			continue
		}
		baseBlock := -1
		for statement := 0; statement < len(document.Declarations[baseIndex].Statements); statement++ {
			if statementBlockName(document.Declarations[baseIndex].Statements[statement]) == "sequences" {
				baseBlock = statement
				break
			}
		}
		if baseBlock < 0 {
			document.Declarations[baseIndex].Statements = append(
				document.Declarations[baseIndex].Statements,
				Statement{Tokens: []string{"sequences"}, Span: extension.Span})
			baseBlock = len(document.Declarations[baseIndex].Statements) - 1
		}
		document.Declarations[baseIndex].Statements[baseBlock].Children = append(
			document.Declarations[baseIndex].Statements[baseBlock].Children,
			extension.Statements[sequenceBlock].Children...)
	}
	out := document.Declarations[:0]
	for i := 0; i < len(document.Declarations); i++ {
		if document.Declarations[i].Kind != DeclArchExtension {
			out = append(out, document.Declarations[i])
		}
	}
	document.Declarations = out
}

func addVirtualArchitectureGoSymbol(pkg *virtualPackage, local string, qualified string) {
	for i := 0; i < len(pkg.Symbols); i++ {
		if pkg.Symbols[i].Local == local {
			return
		}
	}
	pkg.Symbols = append(pkg.Symbols, virtualSymbol{
		Local: local, Qualified: qualified,
	})
}

func addVirtualSymbol(document *Document, declaration Declaration, local string) {
	addVirtualSymbolQualified(
		document, declaration, local,
		declaration.Package+"Package"+exportedName(local))
}

func addVirtualSymbolQualified(document *Document, declaration Declaration,
	local string, qualified string) {
	for i := 0; i < len(document.packages); i++ {
		if document.packages[i].Name != declaration.Package {
			continue
		}
		for j := 0; j < len(document.packages[i].Symbols); j++ {
			if document.packages[i].Symbols[j].Local == local {
				return
			}
		}
		for j := 0; j < len(document.packages[i].Symbols); j++ {
			if document.packages[i].Symbols[j].Qualified == qualified {
				document.Diagnostics = append(document.Diagnostics,
					resolveDiagnostic(*document, declaration, "RTG-IMPORT-008",
						"local names collide after virtual package qualification: "+
							local+" and "+document.packages[i].Symbols[j].Local))
				return
			}
		}
		document.packages[i].Symbols = append(document.packages[i].Symbols, virtualSymbol{
			Local: local, Qualified: qualified,
		})
		return
	}
}

func rewriteVirtualStatements(statements []Statement, packageName string,
	packages []virtualPackage, parent string, sequenceBody bool, locals []string) {
	for i := 0; i < len(statements); i++ {
		block := statementBlockName(statements[i])
		childLocals := locals
		if sequenceBody {
			statements[i].Tokens = rewriteVirtualSequenceStep(
				statements[i].Tokens, packageName, packages, locals)
			tokens := statements[i].Tokens
			if len(tokens) >= 2 && (tokens[0] == "let" || tokens[0] == "var") &&
				stringIndex(locals, tokens[1]) < 0 {
				locals = append(locals, tokens[1])
			}
		} else if parent == "sequences" {
			childLocals = virtualSequenceLocals(statements[i])
			if len(statements[i].Tokens) != 0 {
				if qualified, ok := virtualQualified(
					packages, packageName, statements[i].Tokens[0]); ok {
					statements[i].Tokens[0] = qualified
				}
			}
		} else {
			for token := 0; token+1 < len(statements[i].Tokens); token++ {
				if statements[i].Tokens[token] != "go" &&
					statements[i].Tokens[token] != "sequence" {
					continue
				}
				name := token + 1
				if name+2 < len(statements[i].Tokens) &&
					statements[i].Tokens[name+1] == "." {
					if qualified, ok := virtualStatementQualified(packages,
						statements[i].Tokens[name], statements[i].Tokens[name+2]); ok {
						statements[i].Tokens[name] = qualified
						statements[i].Tokens = append(
							statements[i].Tokens[:name+1],
							statements[i].Tokens[name+3:]...)
					}
				} else if qualified, ok := virtualStatementQualified(
					packages, packageName, statements[i].Tokens[name]); ok {
					statements[i].Tokens[name] = qualified
				}
			}
		}
		rewriteVirtualStatements(
			statements[i].Children, packageName, packages, block,
			parent == "sequences", childLocals)
	}
}

func rewriteVirtualSequenceStep(tokens []string, packageName string,
	packages []virtualPackage, locals []string) []string {
	if len(tokens) == 0 {
		return tokens
	}
	start := 1
	if sequenceBareCall(tokens) {
		start = 0
	} else if tokens[0] == "let" || tokens[0] == "set" {
		for start < len(tokens) && tokens[start] != "=" {
			start++
		}
		if start < len(tokens) {
			start++
		}
	} else if tokens[0] == "var" || tokens[0] == "increment" ||
		tokens[0] == "decrement" {
		return tokens
	}
	return rewriteVirtualTokens(tokens, start, packageName, packages, locals)
}

func rewriteVirtualTokens(tokens []string, start int, packageName string,
	packages []virtualPackage, locals []string) []string {
	var out []string
	out = append(out, tokens[:start]...)
	for i := start; i < len(tokens); i++ {
		if stringIndex(locals, tokens[i]) >= 0 {
			out = append(out, tokens[i])
			continue
		}
		if i+2 < len(tokens) && tokens[i+1] == "." {
			if qualified, ok := virtualStatementQualified(
				packages, tokens[i], tokens[i+2]); ok {
				out = append(out, qualified)
				i += 2
				continue
			}
		}
		if qualified, ok := virtualStatementQualified(
			packages, packageName, tokens[i]); ok {
			out = append(out, qualified)
		} else {
			out = append(out, tokens[i])
		}
	}
	return out
}

func virtualSequenceLocals(statement Statement) []string {
	sequence, ok := decodeArchitectureSequence(statement)
	if !ok {
		return nil
	}
	locals := make([]string, 0, len(sequence.Parameters))
	for i := 0; i < len(sequence.Parameters); i++ {
		locals = append(locals, sequence.Parameters[i].Name)
	}
	return locals
}

func virtualStatementQualified(packages []virtualPackage, packageName string,
	local string) (string, bool) {
	if qualified, ok := virtualQualified(packages, packageName, local); ok {
		return qualified, true
	}
	for i := 0; i < len(packages); i++ {
		if packages[i].Name != packageName {
			continue
		}
		for j := 0; j < len(packages[i].StatementSymbols); j++ {
			if packages[i].StatementSymbols[j].Local == local {
				return packages[i].StatementSymbols[j].Qualified, true
			}
		}
	}
	return "", false
}

func rewriteVirtualSource(source []byte, packageName string,
	packages []virtualPackage) []byte {
	tokens, diagnostics := scan(source, "")
	if len(diagnostics) != 0 {
		return source
	}
	protected := virtualGoProtectedIdentifiers(source, tokens)
	var out []byte
	last := 0
	for i := 0; i < len(tokens) && tokens[i].Kind != TokenEOF; i++ {
		token := tokens[i]
		out = append(out, source[last:token.Start]...)
		text := tokenText(source, token)
		if token.Kind == TokenIdent && i+2 < len(tokens) &&
			tokenText(source, tokens[i+1]) == "." && tokens[i+2].Kind == TokenIdent {
			other := tokenText(source, tokens[i+2])
			if qualified, ok := virtualQualified(packages, text, other); ok {
				out = append(out, qualified...)
				last = tokens[i+2].End
				i += 2
				continue
			}
		}
		if token.Kind == TokenIdent {
			selector := i > 0 && tokenText(source, tokens[i-1]) == "."
			member := i < len(protected) && protected[i]
			if qualified, ok := virtualQualified(packages, packageName, text); ok &&
				!selector && !member && !virtualGoKeyword(text) {
				out = append(out, qualified...)
			} else {
				out = append(out, source[token.Start:token.End]...)
			}
		} else {
			out = append(out, source[token.Start:token.End]...)
		}
		last = token.End
	}
	return append(out, source[last:]...)
}

func virtualGoKeyword(value string) bool {
	for _, keyword := range []string{
		"break", "default", "func", "interface", "select",
		"case", "defer", "go", "map", "struct",
		"chan", "else", "goto", "package", "switch",
		"const", "fallthrough", "if", "range", "type",
		"continue", "for", "import", "return", "var",
	} {
		if value == keyword {
			return true
		}
	}
	return false
}

// virtualGoProtectedIdentifiers marks identifiers that are members rather than
// package names. The RTG package resolver intentionally stays a token rewrite,
// but it must not turn a struct field, interface method, composite-literal key,
// label, or method declaration into a package helper merely because the names
// happen to match.
func virtualGoProtectedIdentifiers(source []byte, tokens []Token) []bool {
	protected := make([]bool, len(tokens))
	protectVirtualFunctionBindings(source, tokens, protected)
	protectVirtualMembers(source, tokens, protected)
	return protected
}

// protectVirtualFunctionBindings keeps Go's lexical names lexical. Package
// qualification applies to top-level RTG helpers, never to receiver, parameter,
// or named-result bindings that happen to use the same spelling.
func protectVirtualFunctionBindings(source []byte, tokens []Token, protected []bool) {
	const prefix = "package backend\n"
	wrapped := make([]byte, 0, len(prefix)+len(source))
	wrapped = append(wrapped, prefix...)
	wrapped = append(wrapped, source...)
	file := syntax.ParseFile(wrapped)
	if !file.Ok {
		return
	}
	for i := 0; i < len(file.Funcs); i++ {
		fn := file.Funcs[i]
		var bindings []string
		if fn.ReceiverStart >= 0 {
			bindings = appendVirtualBindings(
				bindings, file, fn.ReceiverStart, fn.ReceiverEnd,
				tokens, protected, len(prefix))
			protectVirtualSyntaxToken(
				tokens, protected, syntax.TokenStart(file.Tokens[fn.NameTok])-len(prefix))
		}
		bindings = appendVirtualBindings(
			bindings, file, fn.ParamsStart+1, fn.ParamsEnd-1,
			tokens, protected, len(prefix))
		if fn.ResultStart < fn.ResultEnd &&
			string(syntax.TokenText(wrapped, file.Tokens[fn.ResultStart])) == "(" {
			bindings = appendVirtualBindings(
				bindings, file, fn.ResultStart+1, fn.ResultEnd-1,
				tokens, protected, len(prefix))
		}
		bodyStart := syntax.TokenStart(file.Tokens[fn.BodyStart]) - len(prefix)
		bodyEnd := syntax.TokenEnd(file.Tokens[fn.BodyEnd-1]) - len(prefix)
		protectVirtualLocalBindings(source, tokens, protected, bodyStart, bodyEnd)
		for at := 0; at < len(tokens) && tokens[at].Kind != TokenEOF; at++ {
			if tokens[at].Start < bodyStart || tokens[at].End > bodyEnd {
				continue
			}
			if stringIndex(bindings, tokenText(source, tokens[at])) >= 0 {
				protected[at] = true
			}
		}
	}
}

func protectVirtualLocalBindings(source []byte, tokens []Token, protected []bool,
	bodyStart int, bodyEnd int) {
	blockEnds := virtualBlockEnds(source, tokens)
	for at := 0; at < len(tokens) && tokens[at].Kind != TokenEOF; at++ {
		if tokens[at].Start < bodyStart || tokens[at].End > bodyEnd {
			continue
		}
		text := tokenText(source, tokens[at])
		if text == "var" || text == "const" {
			name := at + 1
			if name < len(tokens) && tokens[name].Kind == TokenIdent {
				protectVirtualLocalScope(
					source, tokens, protected, name, name, blockEnds)
			}
			continue
		}
		if text != ":" || at+1 >= len(tokens) ||
			tokenText(source, tokens[at+1]) != "=" {
			continue
		}
		line := tokens[at].Line
		first := at - 1
		for first > 0 && tokens[first-1].Line == line {
			previous := tokenText(source, tokens[first-1])
			if tokens[first-1].Kind != TokenIdent && previous != "," {
				break
			}
			first--
		}
		for name := first; name < at; name++ {
			if tokens[name].Kind == TokenIdent &&
				tokenText(source, tokens[name]) != "_" {
				protectVirtualLocalScope(
					source, tokens, protected, name, at+1, blockEnds)
			}
		}
	}
}

func virtualBlockEnds(source []byte, tokens []Token) []int {
	ends := make([]int, len(tokens))
	var stack []int
	for i := 0; i < len(tokens) && tokens[i].Kind != TokenEOF; i++ {
		text := tokenText(source, tokens[i])
		if text == "{" {
			stack = append(stack, i)
		} else if text == "}" && len(stack) != 0 {
			ends[stack[len(stack)-1]] = i
			stack = stack[:len(stack)-1]
		}
	}
	return ends
}

func protectVirtualLocalScope(source []byte, tokens []Token, protected []bool,
	nameToken int, scopeStart int, blockEnds []int) {
	name := tokenText(source, tokens[nameToken])
	scopeEnd := len(tokens)
	for open := nameToken; open >= 0; open-- {
		if tokenText(source, tokens[open]) != "{" {
			continue
		}
		if blockEnds[open] > nameToken {
			scopeEnd = blockEnds[open]
			break
		}
	}
	protected[nameToken] = true
	for at := scopeStart; at < scopeEnd; at++ {
		if tokens[at].Kind == TokenIdent && tokenText(source, tokens[at]) == name {
			protected[at] = true
		}
	}
}

func appendVirtualBindings(bindings []string, file syntax.File, start int, end int,
	tokens []Token, protected []bool, prefix int) []string {
	itemStart := start
	depth := 0
	for at := start; at <= end; at++ {
		text := ""
		if at < end {
			text = string(syntax.TokenText(file.Src, file.Tokens[at]))
		}
		if text == "(" || text == "[" {
			depth++
		} else if text == ")" || text == "]" {
			depth--
		}
		if at != end && (text != "," || depth != 0) {
			continue
		}
		// A single-token final item is an unnamed type. Earlier single-token
		// items belong to a grouped declaration such as "left, right int".
		if itemStart < at && (itemStart+1 < at || at < end) {
			first := string(syntax.TokenText(file.Src, file.Tokens[itemStart]))
			if first != "..." && stringIndex(bindings, first) < 0 {
				bindings = append(bindings, first)
			}
			protectVirtualSyntaxToken(
				tokens, protected, syntax.TokenStart(file.Tokens[itemStart])-prefix)
		}
		itemStart = at + 1
	}
	return bindings
}

func protectVirtualSyntaxToken(tokens []Token, protected []bool, start int) {
	for i := 0; i < len(tokens) && tokens[i].Kind != TokenEOF; i++ {
		if tokens[i].Start == start {
			protected[i] = true
			return
		}
	}
}

func protectVirtualMembers(source []byte, tokens []Token, protected []bool) {
	type block struct {
		kind  string
		depth int
	}
	var blocks []block
	depth := 0
	fieldStart := false
	fieldLine := 0
	for i := 0; i < len(tokens) && tokens[i].Kind != TokenEOF; i++ {
		text := tokenText(source, tokens[i])
		if text == "{" {
			kind := ""
			if i > 0 {
				previous := tokenText(source, tokens[i-1])
				if previous == "struct" || previous == "interface" {
					kind = previous
				}
			}
			depth++
			if kind != "" {
				blocks = append(blocks, block{kind: kind, depth: depth})
				fieldStart = true
				fieldLine = 0
			}
			continue
		}
		if text == "}" {
			if len(blocks) != 0 && blocks[len(blocks)-1].depth == depth {
				blocks = blocks[:len(blocks)-1]
			}
			depth--
			fieldStart = false
			continue
		}
		if i+1 < len(tokens) && tokenText(source, tokens[i+1]) == ":" {
			protected[i] = true
		}
		if len(blocks) == 0 || blocks[len(blocks)-1].depth != depth {
			continue
		}
		if text == ";" {
			fieldStart = true
			fieldLine = 0
			continue
		}
		if fieldLine != 0 && tokens[i].Line != fieldLine {
			fieldStart = true
		}
		if !fieldStart {
			continue
		}
		fieldLine = tokens[i].Line
		if tokens[i].Kind != TokenIdent || i+1 >= len(tokens) {
			fieldStart = false
			continue
		}
		next := tokenText(source, tokens[i+1])
		if tokens[i+1].Line == tokens[i].Line &&
			next != "}" && next != ";" && next != "." {
			protected[i] = true
		}
		fieldStart = false
	}
}

func rewriteVirtualField(value string, packageName string,
	packages []virtualPackage) string {
	tokens, diagnostics := scan([]byte(value), "")
	if len(diagnostics) != 0 || len(tokens) < 2 {
		return value
	}
	first := tokenText([]byte(value), tokens[0])
	if first != "go" && first != "sequence" {
		return value
	}
	return string(rewriteVirtualSource([]byte(value), packageName, packages))
}

func virtualQualified(packages []virtualPackage, packageName string,
	local string) (string, bool) {
	for i := 0; i < len(packages); i++ {
		if packages[i].Name != packageName {
			continue
		}
		for j := 0; j < len(packages[i].Symbols); j++ {
			if packages[i].Symbols[j].Local == local {
				return packages[i].Symbols[j].Qualified, true
			}
		}
	}
	return "", false
}

func authoredVirtualSource(document Document, source []byte) []byte {
	if len(document.packages) == 0 {
		return source
	}
	tokens, diagnostics := scan(source, "")
	if len(diagnostics) != 0 {
		return source
	}
	var out []byte
	last := 0
	for i := 0; i < len(tokens) && tokens[i].Kind != TokenEOF; i++ {
		token := tokens[i]
		out = append(out, source[last:token.Start]...)
		text := tokenText(source, token)
		replaced := false
		if token.Kind == TokenIdent {
			for pkg := 0; pkg < len(document.packages) && !replaced; pkg++ {
				for symbol := 0; symbol < len(document.packages[pkg].Symbols); symbol++ {
					item := document.packages[pkg].Symbols[symbol]
					if item.Qualified == text {
						out = append(out, item.Local...)
						replaced = true
						break
					}
				}
			}
		}
		if !replaced {
			out = append(out, source[token.Start:token.End]...)
		}
		last = token.End
	}
	return append(out, source[last:]...)
}

func semanticVirtualPackage(document Document, declaration Declaration) string {
	for i := 0; i < len(document.packages); i++ {
		if document.packages[i].Name == declaration.Package &&
			len(document.packages[i].Symbols) != 0 {
			return declaration.Package
		}
	}
	return ""
}

func hasSemanticVirtualPackages(document Document) bool {
	for i := 0; i < len(document.packages); i++ {
		if len(document.packages[i].Symbols) != 0 {
			return true
		}
	}
	return false
}

func virtualPackageName(filename string) string {
	start := len(filename)
	for start > 0 && filename[start-1] != '/' && filename[start-1] != '\\' {
		start--
	}
	end := len(filename)
	for i := end - 1; i >= start; i-- {
		if filename[i] == '.' {
			end = i
			break
		}
	}
	var out []byte
	for i := start; i < end; i++ {
		ch := filename[i]
		if isIdentPart(ch) {
			out = append(out, ch)
		} else if len(out) == 0 || out[len(out)-1] != '_' {
			out = append(out, '_')
		}
	}
	if len(out) == 0 {
		return "definition"
	}
	if out[0] >= '0' && out[0] <= '9' {
		out = append([]byte("definition_"), out...)
	}
	return string(out)
}
