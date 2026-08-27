//go:build !renvo

package driver

import (
	"renvo.dev/internal/backenddef"
	"renvo.dev/internal/load"
	"renvo.dev/internal/targetinfo"
	"renvo.dev/internal/unit"
)

func renvoBackendTargetSupported(target string) bool {
	_ = target
	return false
}

func renvoBackendTargetBinding(target string) (string, string, int, bool) {
	_ = target
	return "", "", 0, false
}

func renvoBackendTargetHasBuildTag(target string, tag string) bool {
	_, _ = target, tag
	return false
}

func resolveForeignTarget(options *Options, workDir string, target string, fs SourceFS, result *foreignTarget) {
	if name, definition, version, ok := targetinfo.Binding(target); ok {
		result.Binding = unit.TargetBinding{Target: name, Definition: definition, DescriptorVersion: version}
		result.InPlace = targetinfo.SupportsInPlaceEntry(target)
		result.Ok = true
		return
	}
	if options.BackendDefinition == "" || fs == nil {
		return
	}
	path := load.JoinPath(workDir, options.BackendDefinition)
	source, ok := fs.ReadFile(path)
	if !ok {
		return
	}
	resolved := backenddef.ResolveImports(source, path, target, backendDefinitionImportLoader{fs: fs})
	if !resolved.Ok {
		return
	}
	descriptor := resolved.Descriptor
	result.Binding = unit.TargetBinding{Target: descriptor.Name, Definition: string(descriptor.Definition[:]), DescriptorVersion: descriptor.Version}
	result.InPlace = findString(descriptor.Capabilities, "in_place_entry") >= 0
	result.Tags = descriptor.BuildTags
	result.Ok = true
}
