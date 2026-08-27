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
			return Result{Descriptor: descriptor, LibraryFiles: artifact.LibraryFiles, Ok: true}
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
			return Result{Descriptor: target.Descriptor, LibraryFiles: bundle.Files, Ok: true}
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
