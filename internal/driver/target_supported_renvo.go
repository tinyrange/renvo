//go:build renvo

package driver

import (
	"renvo.dev/internal/backendbridge"
	"renvo.dev/internal/load"
	"renvo.dev/internal/targetinfo"
	"renvo.dev/internal/unit"
)

func renvoBackendTargetSupported(target string) bool {
	return backendbridge.TargetSupported(target)
}

func renvoBackendTargetBinding(target string) (string, string, int, bool) {
	return backendbridge.TargetBinding(target)
}

func renvoBackendTargetHasBuildTag(target string, tag string) bool {
	return backendbridge.TargetHasBuildTag(target, tag)
}

func renvoBackendTargetLayout(target string) load.TargetLayout {
	wordBits, pointerBits, _ := backendbridge.TargetLayout(target)
	scalarAlign, _ := backendbridge.TargetScalarAlignment(target)
	return load.TargetLayout{WordBits: wordBits, PointerBits: pointerBits, ScalarAlign: scalarAlign}
}

func resolveForeignTarget(options *Options, workDir string, target string, fs SourceFS, result *foreignTarget) {
	_, _, _ = options, workDir, fs
	if name, definition, version, ok := targetinfo.Binding(target); ok {
		result.Layout = frontendTargetLayout(Options{Target: target})
		result.Binding = unit.TargetBinding{Target: name, Definition: definition, DescriptorVersion: version}
		result.InPlace = targetinfo.SupportsInPlaceEntry(target)
		result.Ok = true
		return
	}
	name, definition, version, ok := backendbridge.TargetBinding(target)
	if !ok {
		return
	}
	result.Binding = unit.TargetBinding{Target: name, Definition: definition, DescriptorVersion: version}
	result.Layout = renvoBackendTargetLayout(target)
	result.InPlace = backendbridge.TargetHasCapability(target, "in_place_entry")
	result.Ok = true
}
