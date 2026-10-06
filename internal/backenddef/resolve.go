// Package backenddef projects an RTG definition into the public descriptor
// selected by a caller. It deliberately does not depend on the frontend driver,
// so applications that use ordinary built-in compilation do not acquire the
// definition compiler through the driver package.
package backenddef

import (
	"renvo.dev/internal/rbe"
	"renvo.dev/internal/rtg"
	"renvo.dev/internal/rtgb"
)

type Result struct {
	Descriptor   rtg.TargetDescriptor
	Vocabulary   rtg.TargetVocabulary
	LibraryFiles []rbe.File
	Message      string
	Ok           bool
}

func Resolve(source []byte, filename string, targetName string) Result {
	return ResolveImports(source, filename, targetName, nil)
}

func ResolveImports(
	source []byte, filename string, targetName string, loader rtg.ImportLoader,
) Result {
	if rtgb.IsArtifact(source) {
		artifact, ok := rtgb.Decode(source)
		if !ok {
			return Result{Message: "invalid prepared backend artifact"}
		}
		descriptor := artifact.Descriptor
		if descriptor.Name == targetName || contains(descriptor.Aliases, targetName) {
			var vocabulary rtg.TargetVocabulary
			if len(artifact.DefinitionFiles) != 0 {
				root := artifact.DefinitionFiles[0]
				resolved := rtg.ResolveDefinitions(rtg.ParseImports(root.Source, root.Filename, definitionSnapshotLoader{files: artifact.DefinitionFiles}))
				if !resolved.Ok {
					return Result{Message: "invalid prepared backend definition snapshot"}
				}
				vocabulary = rtg.FrontendOperations(resolved, descriptor.Name)
				if !vocabulary.Ok {
					return Result{Message: "invalid prepared backend frontend contract"}
				}
			}
			return Result{Descriptor: descriptor, Vocabulary: vocabulary, LibraryFiles: artifact.LibraryFiles, Ok: true}
		}
		return Result{Message: "backend definition does not export target " + targetName}
	}
	bundle := rbe.Parse(source)
	if !bundle.Ok {
		return Result{Message: bundle.Message}
	}
	resolved := rtg.ResolveDefinitions(rtg.ParseImports(bundle.Definition, filename, loader))
	if !resolved.Ok {
		return Result{Message: resolved.Diagnostics[0].Message}
	}
	for i := 0; i < len(resolved.Targets); i++ {
		target := resolved.Targets[i]
		if target.Descriptor.Name == targetName || contains(target.Descriptor.Aliases, targetName) {
			vocabulary := rtg.FrontendOperations(resolved, target.Descriptor.Name)
			if !vocabulary.Ok {
				return Result{Message: vocabulary.Diagnostics[0].Message}
			}
			return Result{Descriptor: target.Descriptor, Vocabulary: vocabulary, LibraryFiles: bundle.Files, Ok: true}
		}
	}
	return Result{Message: "backend definition does not export target " + targetName}
}

func contains(values []string, value string) bool {
	for i := 0; i < len(values); i++ {
		if values[i] == value {
			return true
		}
	}
	return false
}
