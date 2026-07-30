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
		canonical = appendFramedBytes(canonical, declarations[i].body)
	}

	roots := targetGoRoots(document, target)
	parts := reachableEmbeddedGoParts(document, roots, nil)
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
