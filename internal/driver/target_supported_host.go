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

func resolveForeignTarget(options Options, workDir string, target string, fs SourceFS) (unit.TargetBinding, bool, []string, bool) {
	if name, definition, version, ok := targetinfo.Binding(target); ok {
		return unit.TargetBinding{Target: name, Definition: definition, DescriptorVersion: version}, targetinfo.HasCapability(target, "in_place_entry"), nil, true
	}
	if options.BackendDefinition == "" || fs == nil {
		return unit.TargetBinding{}, false, nil, false
	}
	path := load.JoinPath(workDir, options.BackendDefinition)
	source, ok := fs.ReadFile(path)
	if !ok {
		return unit.TargetBinding{}, false, nil, false
	}
	resolved := backenddef.ResolveImports(source, path, target, backendDefinitionImportLoader{fs: fs})
	if !resolved.Ok {
		return unit.TargetBinding{}, false, nil, false
	}
	descriptor := resolved.Descriptor
	return unit.TargetBinding{Target: descriptor.Name, Definition: string(descriptor.Definition[:]), DescriptorVersion: descriptor.Version},
		targetDescriptorCapability(descriptor.Capabilities, "in_place_entry"), descriptor.BuildTags, true
}
