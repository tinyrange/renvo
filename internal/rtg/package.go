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
	for i := 0; i < len(document.Declarations); i++ {
		declaration := document.Declarations[i]
		if declaration.Kind == DeclGo {
			addVirtualGoSymbols(document, declaration)
		} else if declaration.Kind == DeclArch {
			addVirtualArchitectureStatementSymbols(document, declaration)
			addVirtualSequenceSymbols(document, declaration)
		}
	}
	if len(document.Diagnostics) != 0 {
		return
	}
	for i := 0; i < len(document.Declarations); i++ {
		declaration := &document.Declarations[i]
		if declaration.Kind == DeclGo {
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
			if document.packages[pkg].Name == declaration.Package {
				document.packages[pkg].StatementSymbols = append(
					document.packages[pkg].StatementSymbols,
					virtualSymbol{Local: local, Qualified: symbols[i].Local})
				break
			}
		}
	}
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
	locals := make([]string, 0, len(sequence.Parameters)+len(sequence.Steps))
	for i := 0; i < len(sequence.Parameters); i++ {
		locals = append(locals, sequence.Parameters[i].Name)
	}
	for i := 0; i < len(sequence.Steps); i++ {
		tokens := sequence.Steps[i].Tokens
		if len(tokens) >= 2 && (tokens[0] == "let" || tokens[0] == "var") &&
			stringIndex(locals, tokens[1]) < 0 {
			locals = append(locals, tokens[1])
		}
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
			if qualified, ok := virtualQualified(packages, packageName, text); ok {
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
