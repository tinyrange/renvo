//go:build !renvo

package rtg

// GenerateProductionArchitectureBackend emits only definition-owned
// production algorithms under their explicit exported names. Declarative
// facts are folded into those algorithms so fixed-target source does not
// retain unused descriptors, register objects, or comparison bindings.
func GenerateProductionArchitectureBackend(resolved ResolveResult, archName string, packageName string) GenerateResult {
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
	source := generateHeaderPackage(manifest, "arch/"+archName, packageName)
	source = appendArchitectureEmbeddedGo(source, resolved.Document, arch, true)
	return GenerateResult{Source: source, Manifest: manifest, Ok: true}
}
