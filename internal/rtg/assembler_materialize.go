package rtg

// LowerAssemblerFile shares operation validation and evaluator generation with
// typed assembly while retaining original source and source spans for diagnostics.
func LowerAssemblerFile(resolved ResolveResult, targetName string, source []byte, filename string) AssemblyDocument {
	v := FrontendOperations(resolved, targetName)
	blocks, diagnostics := ParseTargetAssembler(v, "", source, filename)
	document := AssemblyDocument{Filename: filename, Source: source, Version: 1, Diagnostics: diagnostics, Ok: len(diagnostics) == 0}
	if !document.Ok {
		return document
	}
	for _, block := range blocks {
		steps, message := validateTargetBlock(v, block)
		if message != "" {
			document.Ok = false
			document.Diagnostics = assemblerDiagnostic(filename, Span{}, message)
			return document
		}
		span := Span{}
		if len(steps) != 0 {
			span = steps[0].Span
		}
		document.Entries = append(document.Entries, AssemblyEntry{Name: block.Name, Steps: steps, Parameters: []AssemblyParameter{{Name: "out", Kind: "emitter"}}, Span: span})
	}
	return document
}
