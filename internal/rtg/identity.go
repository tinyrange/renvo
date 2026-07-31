package rtg

// targetSemanticIdentity identifies the resolved implementation of one target.
// It deliberately excludes unrelated declarations in the same catalog so
// adding another architecture does not invalidate existing RTGU bindings or
// prepared-backend cache entries.
func targetSemanticIdentity(document Document, target ResolvedTarget) [32]byte {
	var canonical []byte
	canonical = appendFramed(canonical, "rtg-target")
	canonical = appendDecimalFrame(canonical, DescriptorVersion)
	canonical = appendDecimalFrame(canonical, GeneratorVersion)
	canonical = appendFramed(canonical, document.Unit)
	canonical = appendFramed(canonical, target.Descriptor.Family)

	implements := make([]string, len(document.Implements))
	copy(implements, document.Implements)
	sortStrings(implements)
	for i := 0; i < len(implements); i++ {
		canonical = appendFramed(canonical, "implements")
		canonical = appendFramed(canonical, implements[i])
	}

	declarations := targetSemanticDeclarations(document, target)
	hasPackages := hasSemanticVirtualPackages(document)
	for i := 1; i < len(declarations); i++ {
		value := declarations[i]
		at := i
		for at > 0 && canonicalDeclarationLess(value, declarations[at-1]) {
			declarations[at] = declarations[at-1]
			at--
		}
		declarations[at] = value
	}
	for i := 0; i < len(declarations); i++ {
		canonical = appendFramed(canonical, declarations[i].kind)
		canonical = appendFramed(canonical, declarations[i].name)
		if hasPackages {
			canonical = appendFramed(canonical, declarations[i].pkg)
		}
		canonical = appendFramedBytes(canonical, declarations[i].body)
	}

	goRoots := baseTargetGoRoots(document, target)
	goRoots, sequenceRoots := resolveArchitectureSequenceProjection(
		document, target.Arch, goRoots, targetSequenceRoots(document, target))
	sequences := targetSemanticSequences(target.Arch, sequenceRoots)
	for i := 0; i < len(sequences); i++ {
		canonical = appendFramed(canonical, "sequence")
		canonical = appendFramed(canonical, sequences[i].name)
		canonical = appendFramedBytes(canonical, sequences[i].body)
	}

	parts := reachableEmbeddedGoParts(document, goRoots, nil)
	for i := 0; i < len(parts); i++ {
		parts[i].source = canonicalTokenStream(parts[i].source)
	}
	for i := 1; i < len(parts); i++ {
		value := parts[i]
		at := i
		for at > 0 && embeddedGoPartLess(value, parts[at-1]) {
			parts[at] = parts[at-1]
			at--
		}
		parts[at] = value
	}
	for i := 0; i < len(parts); i++ {
		canonical = appendFramed(canonical, "go")
		canonical = appendFramed(canonical, parts[i].name)
		canonical = appendFramedBytes(canonical, parts[i].source)
	}
	return sha256Bytes(canonical)
}

type semanticSequence struct {
	name string
	body []byte
}

func targetSemanticSequences(
	arch Declaration, selected []string,
) []semanticSequence {
	var sequences []semanticSequence
	available := architectureSequences(arch)
	for i := 0; i < len(available); i++ {
		if stringIndex(selected, available[i].Name) < 0 {
			continue
		}
		sequences = append(sequences, semanticSequence{
			name: available[i].Name,
			body: canonicalSequenceStatement(available[i].Statement),
		})
	}
	for i := 1; i < len(sequences); i++ {
		value := sequences[i]
		at := i
		for at > 0 && stringLess(value.name, sequences[at-1].name) {
			sequences[at] = sequences[at-1]
			at--
		}
		sequences[at] = value
	}
	return sequences
}

func canonicalSequenceStatement(statement Statement) []byte {
	var canonical []byte
	canonical = appendDecimalFrame(canonical, len(statement.Tokens))
	for i := 0; i < len(statement.Tokens); i++ {
		text := statement.Tokens[i]
		if len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"' {
			text = unquoteSimple(text)
		} else if len(text) != 0 && text[0] >= '0' && text[0] <= '9' {
			text = normalizeNumber(text)
		}
		canonical = appendFramed(canonical, text)
	}
	canonical = appendDecimalFrame(canonical, len(statement.Children))
	for i := 0; i < len(statement.Children); i++ {
		canonical = appendFramedBytes(
			canonical, canonicalSequenceStatement(statement.Children[i]))
	}
	return canonical
}

func targetSemanticDeclarations(document Document, target ResolvedTarget) []canonicalDeclaration {
	var declarations []canonicalDeclaration
	appendDeclaration := func(declaration Declaration) {
		if declaration.Name == "" {
			return
		}
		for i := 0; i < len(declarations); i++ {
			if declarations[i].kind == declaration.Kind &&
				declarations[i].name == declaration.Name {
				return
			}
		}
		declarations = append(declarations, canonicalDeclaration{
			kind: declaration.Kind,
			name: declaration.Name,
			pkg:  semanticVirtualPackage(document, declaration),
			body: canonicalRange(document, declaration.BodyStart, declaration.BodyEnd),
		})
	}
	appendDeclaration(target.Declaration)
	appendDeclaration(target.Arch)
	abi := target.ABI
	for abi.Name != "" {
		appendDeclaration(abi)
		internal, ok := fieldValue(document, abi, "internal")
		if !ok {
			break
		}
		abi, _ = document.Declaration(DeclABI, valueName(internal))
	}
	appendDeclaration(target.Runtime)
	appendDeclaration(target.Executable)
	appendDeclaration(target.Object)
	return declarations
}

func targetGoRoots(document Document, target ResolvedTarget) []string {
	roots := baseTargetGoRoots(document, target)
	roots, _ = resolveArchitectureSequenceProjection(
		document, target.Arch, roots, targetSequenceRoots(document, target))
	return roots
}

func baseTargetGoRoots(document Document, target ResolvedTarget) []string {
	roots := architectureGoRoots(target.Arch)
	roots = appendDeclarationGoRoots(roots, target.ABI)
	abi := target.ABI
	for {
		internal, ok := fieldValue(document, abi, "internal")
		if !ok {
			break
		}
		base, ok := document.Declaration(DeclABI, valueName(internal))
		if !ok {
			break
		}
		roots = appendDeclarationGoRoots(roots, base)
		abi = base
	}
	roots = appendDeclarationGoRoots(roots, target.Runtime)
	roots = appendDeclarationGoRoots(roots, target.Executable)
	roots = appendDeclarationGoRoots(roots, target.Object)
	roots = appendDeclarationGoRoots(roots, target.Declaration)
	if len(roots) == 0 {
		roots = embeddedGoNames(document)
	}
	sortStrings(roots)
	return roots
}

func canonicalTokenStream(source []byte) []byte {
	tokens, diagnostics := scan(source, "")
	if len(diagnostics) != 0 {
		return source
	}
	var canonical []byte
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		if token.Kind == TokenEOF {
			continue
		}
		text := tokenText(source, token)
		if token.Kind == TokenNumber {
			text = normalizeNumber(text)
		}
		canonical = appendDecimalFrame(canonical, token.Kind)
		canonical = appendFramed(canonical, text)
	}
	return canonical
}

func embeddedGoPartLess(left embeddedGoPart, right embeddedGoPart) bool {
	if left.name != right.name {
		return stringLess(left.name, right.name)
	}
	return bytesLess(left.source, right.source)
}
