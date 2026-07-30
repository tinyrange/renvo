//go:build !renvo

package rtg

// GenerateCheckedInArchitectureAlgorithms emits the definition-owned
// algorithms reachable from the stable names used by the checked-in compiler.
// Fixed Renvo source manifests include this projection and therefore never
// parse unrelated semantic-contract helpers.
func GenerateCheckedInArchitectureAlgorithms(resolved ResolveResult, archName string, packageName string) GenerateResult {
	ensureDirectEmitterV1()
	if !resolved.Ok {
		diagnostics := make([]Diagnostic, len(resolved.Diagnostics))
		copy(diagnostics, resolved.Diagnostics)
		return GenerateResult{Diagnostics: diagnostics}
	}
	arch, ok := resolved.Document.Declaration(DeclArch, archName)
	if !ok {
		return GenerateResult{Diagnostics: []Diagnostic{{
			Filename: resolved.Document.Filename,
			Span:     sourceSpan(resolved.Document.Source, 0, 0),
			Code:     "RTG-GENERATE-004",
			Message:  "definition does not declare architecture " + archName,
		}}}
	}
	manifest := []string{resolved.Document.Unit + " " + HashText(resolved.Document.Hash)}
	source := generateHeaderPackage(manifest, "algorithms/"+archName, packageName)
	projection := checkedInProjectionDocument(resolved.Document, arch)
	roots := architectureExportGoRoots(arch)
	roots, sequences := resolveArchitectureSequenceProjection(resolved.Document, arch,
		roots, architectureSequenceRoots(arch, true, false))
	source = appendReachableEmbeddedGo(source, projection, roots, true, architectureExports(arch))
	source = appendArchitectureSequences(source, projection, arch, true, true, false, sequences)
	if declarativeArchitectureRelocations(arch) != "" {
		source = appendCheckedInArchitectureHooks(source, projection, arch)
	}
	return GenerateResult{Source: source, Manifest: manifest, Ok: true}
}

// GenerateCheckedInArchitectureContract emits the complete typed semantic
// contract as ordinary production Go. It is compiled by normal Go-module and
// release builds, but omitted from fixed Renvo source manifests so source-level
// pruning remains effective.
func GenerateCheckedInArchitectureContract(resolved ResolveResult, archName string, packageName string) GenerateResult {
	ensureDirectEmitterV1()
	if !resolved.Ok {
		diagnostics := make([]Diagnostic, len(resolved.Diagnostics))
		copy(diagnostics, resolved.Diagnostics)
		return GenerateResult{Diagnostics: diagnostics}
	}
	arch, ok := resolved.Document.Declaration(DeclArch, archName)
	if !ok {
		return GenerateResult{Diagnostics: []Diagnostic{{
			Filename: resolved.Document.Filename,
			Span:     sourceSpan(resolved.Document.Source, 0, 0),
			Code:     "RTG-GENERATE-004",
			Message:  "definition does not declare architecture " + archName,
		}}}
	}
	if len(architectureBindings(arch)) == 0 {
		return GenerateResult{Diagnostics: []Diagnostic{{
			Filename: resolved.Document.Filename,
			Span:     arch.Span,
			Code:     "RTG-GENERATE-005",
			Message:  "architecture " + archName + " has no direct emitter bindings",
		}}}
	}
	manifest := []string{resolved.Document.Unit + " " + HashText(resolved.Document.Hash)}
	source := []byte("//go:build !renvo\n\n")
	source = append(source, generateHeaderPackage(manifest, "contract/"+archName, packageName)...)
	projection := checkedInProjectionDocument(resolved.Document, arch)
	source = appendArchitectureFacts(source, projection, arch, true)
	roots := architectureDirectGoRoots(arch)
	roots, sequences := resolveArchitectureSequenceProjection(resolved.Document, arch,
		roots, architectureSequenceRoots(arch, false, true))
	exports := architectureExports(arch)
	excluded := make([]string, 0, len(exports))
	for i := 0; i < len(exports); i++ {
		excluded = append(excluded, exports[i].Local)
	}
	source = appendReachableEmbeddedGoExcluding(
		source, projection, roots, true, exports, excluded)
	source = appendArchitectureSequences(source, projection, arch, true, false, true, sequences)
	source = appendArchitectureBindings(source, projection, arch, true, true)
	source = appendDirectEmitterBindings(source, projection, arch, true, true)
	source = appendArchitectureHooks(source, projection, arch, true)
	return GenerateResult{Source: source, Manifest: manifest, Ok: true}
}

func checkedInProjectionDocument(document Document, arch Declaration) Document {
	count := 0
	for i := 0; i < len(document.Declarations); i++ {
		if document.Declarations[i].Kind == DeclArch {
			count++
		}
	}
	if count > 1 {
		document.Unit = arch.Name
	}
	return document
}
