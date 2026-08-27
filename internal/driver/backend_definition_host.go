//go:build !renvo

package driver

import (
	"renvo.dev/internal/backenddef"
	"renvo.dev/internal/load"
	"renvo.dev/internal/rbe"
	"renvo.dev/internal/rtg"
	"renvo.dev/internal/rtgprofile"
	"renvo.dev/internal/unit"
)

type backendBuildOptions struct {
	options      Options
	descriptor   rtg.TargetDescriptor
	libraryFiles []rbe.File
	hasBackend   bool
}

func resolveBackendBuildOptions(args []string, workDir string, fs SourceFS) backendBuildOptions {
	backendPath := ""
	systemPath := ""
	targetName := ""
	backendAt := -1
	targetExplicit := false
	arenaExplicit := false
	for i := 0; i < len(args); i++ {
		if args[i] == "-backend" {
			backendAt = i
			if i+1 < len(args) {
				backendPath = args[i+1]
				i++
			}
		} else if args[i] == "-system" && i+1 < len(args) {
			systemPath = args[i+1]
			i++
		} else if args[i] == "-t" && i+1 < len(args) {
			targetName = args[i+1]
			targetExplicit = true
			i++
		} else if args[i] == "-arena-size" {
			arenaExplicit = true
			if i+1 < len(args) {
				i++
			}
		}
	}
	if backendAt < 0 {
		return backendBuildOptions{options: parseFSOptions(args, workDir, fs)}
	}
	var failed Options
	failed.Ok = false
	failed.ErrorAt = backendAt
	if backendPath == "" {
		failed.Error = ParseErrMissingBackend
		failed.ErrorArg = "-backend"
		return backendBuildOptions{options: failed, hasBackend: true}
	}
	var profile rtgprofile.Profile
	if systemPath != "" {
		source, ok := fs.ReadFile(load.JoinPath(workDir, systemPath))
		if !ok {
			failed.Error = ParseErrSystemRead
			failed.ErrorArg = systemPath
			failed.SystemError = "could not read " + systemPath
			return backendBuildOptions{options: failed, hasBackend: true}
		}
		var diagnostic rtgprofile.Diagnostic
		profile, diagnostic, ok = rtgprofile.Parse(source)
		if !ok {
			failed.Error = ParseErrInvalidSystem
			failed.ErrorArg = systemPath
			failed.SystemError = diagnostic.Message
			return backendBuildOptions{options: failed, hasBackend: true}
		}
		if !targetExplicit {
			targetName = profile.Target
		}
		if arenaExplicit {
			failed.Error = ParseErrSystemArenaConflict
			failed.ErrorArg = systemPath
			return backendBuildOptions{options: failed, hasBackend: true}
		}
	}
	if targetName == "" {
		failed.Error = ParseErrMissingTarget
		failed.ErrorArg = "-t"
		return backendBuildOptions{options: failed, hasBackend: true}
	}
	source, ok := fs.ReadFile(load.JoinPath(workDir, backendPath))
	if !ok {
		failed.Error = ParseErrBackendRead
		failed.ErrorArg = backendPath
		return backendBuildOptions{options: failed, hasBackend: true}
	}
	rootPath := load.JoinPath(workDir, backendPath)
	resolved := backenddef.ResolveImports(source, rootPath, targetName,
		backendDefinitionImportLoader{fs: fs})
	if !resolved.Ok {
		failed.Error = ParseErrInvalidBackend
		failed.ErrorArg = resolved.Message
		if resolved.Message == "backend definition does not export target "+targetName {
			failed.Error = ParseErrBackendTarget
			failed.ErrorArg = targetName
		}
		return backendBuildOptions{options: failed, hasBackend: true}
	}
	clean := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "-backend" || args[i] == "-system" {
			i++
			continue
		}
		clean = append(clean, args[i])
	}
	if !targetExplicit {
		clean = append(clean, "-t", resolved.Descriptor.Name)
	}
	options := parseOptions(clean, false)
	if !options.Ok {
		return backendBuildOptions{options: options, hasBackend: true}
	}
	options.Target = resolved.Descriptor.Name
	options.BackendDefinition = backendPath
	options.BackendBuildTags = append(options.BackendBuildTags, resolved.Descriptor.BuildTags...)
	options.TargetExplicit = targetExplicit
	options.System = systemPath
	if options.ArenaSize == 0 {
		options.ArenaSize = resolved.Descriptor.ArenaDefault
	}
	if systemPath != "" {
		options.SystemName = profile.Name
		options.BinaryLimit = profile.BinaryLimit
		options.ArenaSize = profile.ArenaSize
	}
	for i := 0; i < len(resolved.Descriptor.BuildTags); i++ {
		tag := resolved.Descriptor.BuildTags[i]
		if findString(options.Tags, tag) < 0 {
			options.Tags = append(options.Tags, tag)
		}
	}
	if options.WindowsGUI && resolved.Descriptor.OS != "windows" {
		options = parseFail(options, ParseErrWindowsGUIRequiresWindows, options.Target, backendAt)
	}
	if options.Mode == ModeKernelModule && !hostContains(resolved.Descriptor.Capabilities, "kernel_module") {
		options = parseFail(options, ParseErrModeRequiresLinuxAmd64, options.Target, backendAt)
	}
	return backendBuildOptions{options: options, descriptor: resolved.Descriptor,
		libraryFiles: resolved.LibraryFiles, hasBackend: true}
}

type backendDefinitionImportLoader struct {
	fs SourceFS
}

func (loader backendDefinitionImportLoader) LoadImport(
	importingFilename string, importPath string,
) rtg.ImportSource {
	path := load.JoinPath(load.DirPath(importingFilename), importPath)
	imported, found := loader.fs.ReadFile(path)
	return rtg.ImportSource{Source: imported, Filename: path, Ok: found}
}

func BuildFromFSWithBackend(args []string, workDir string, stdRoot string, fs SourceFS) BuildResult {
	return BuildFromFSWithBackendModuleCache(args, workDir, stdRoot, "", fs)
}

func BuildFromFSWithBackendModuleCache(args []string, workDir string, stdRoot string, moduleCache string, fs SourceFS) BuildResult {
	resolved := resolveBackendBuildOptions(args, workDir, fs)
	if len(resolved.libraryFiles) != 0 {
		fs = backendEnablementFS{base: fs, stdRoot: load.CleanPath(stdRoot), files: resolved.libraryFiles}
	}
	result := buildFromFSOptions(resolved.options, workDir, stdRoot, moduleCache, fs, false)
	if !result.Ok || !resolved.hasBackend {
		return result
	}
	var binding unit.TargetBinding
	binding.Target = resolved.descriptor.Name
	binding.Definition = string(resolved.descriptor.Definition[:])
	binding.DescriptorVersion = resolved.descriptor.Version
	bound, ok := unit.BindTarget(result.Unit, binding)
	if ok {
		result.Unit = bound
	}
	return result
}

func hostContains(values []string, value string) bool {
	for i := 0; i < len(values); i++ {
		if values[i] == value {
			return true
		}
	}
	return false
}
