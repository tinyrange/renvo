//go:build !renvo

package backendbuiltin

import (
	"path/filepath"

	"renvo.dev/internal/rtg"
)

// Resolve loads one built-in target from the embedded RTG source catalog.
func Resolve(target string) rtg.ResolveResult {
	root, ok := definitionRoot(target)
	if !ok {
		return rtg.ResolveResult{Diagnostics: []rtg.Diagnostic{{
			Code: "RTG-BUILTIN-001", Message: "unknown built-in target " + target,
		}}}
	}
	source, ok := definitionSource(root)
	if !ok {
		return rtg.ResolveResult{Diagnostics: []rtg.Diagnostic{{
			Code: "RTG-BUILTIN-002", Message: "missing built-in definition " + root,
		}}}
	}
	return rtg.ResolveDefinitions(rtg.ParseImports(source, root, catalogLoader{}))
}

type catalogLoader struct{}

func (catalogLoader) LoadImport(importingFilename string, importPath string) rtg.ImportSource {
	name := filepath.Clean(filepath.Join(filepath.Dir(importingFilename), importPath))
	source, ok := definitionSource(name)
	return rtg.ImportSource{Source: source, Filename: name, Ok: ok}
}
