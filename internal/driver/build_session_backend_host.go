//go:build !renvo

package driver

import (
	"renvo.dev/internal/rbe"
	"renvo.dev/internal/unit"
)

func resolveFSBuildSessionOptions(args []string, workDir string, fs SourceFS) (Options, unit.TargetBinding, []rbe.File) {
	resolved := resolveBackendBuildOptions(args, workDir, fs)
	var binding unit.TargetBinding
	if resolved.hasBackend && resolved.options.Ok {
		binding.Target = resolved.descriptor.Name
		binding.Definition = string(resolved.descriptor.Definition[:])
		binding.DescriptorVersion = resolved.descriptor.Version
	}
	return resolved.options, binding, resolved.libraryFiles
}
