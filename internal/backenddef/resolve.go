// Package backenddef projects an RTG definition into the public descriptor
// selected by a caller. It deliberately does not depend on the frontend driver,
// so applications that use ordinary built-in compilation do not acquire the
// definition compiler through the driver package.
package backenddef

import "renvo.dev/internal/rtg"

type Result struct {
	Descriptor rtg.TargetDescriptor
	Message    string
	Ok         bool
}

func Resolve(source []byte, filename string, targetName string) Result {
	resolved := rtg.ResolveDefinitions(rtg.Parse(source, filename))
	if !resolved.Ok {
		return Result{Message: resolved.Diagnostics[0].Message}
	}
	for i := 0; i < len(resolved.Targets); i++ {
		target := resolved.Targets[i]
		if target.Descriptor.Name == targetName || contains(target.Descriptor.Aliases, targetName) {
			return Result{Descriptor: target.Descriptor, Ok: true}
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
