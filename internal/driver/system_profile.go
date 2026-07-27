package driver

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/rtg"
)

// SystemProfile is the deliberately small hosted resource contract loaded by
// -system. ArenaSize limits the generated program's managed allocator, while
// BinaryLimit limits the final artifact written by the frontend.
type SystemProfile struct {
	Name        string
	Target      string
	BinaryLimit int
	ArenaSize   int
}

func parseFSOptions(args []string, workDir string, fs SourceFS) Options {
	return resolveSystemOptions(ParseOptions(args), workDir, fs)
}

func resolveSystemOptions(options Options, workDir string, fs SourceFS) Options {
	if !options.Ok || options.System == "" {
		return options
	}
	src, ok := fs.ReadFile(load.JoinPath(workDir, options.System))
	if !ok {
		options.SystemError = "could not read " + options.System
		return parseFail(options, ParseErrSystemRead, options.System, options.SystemAt)
	}
	profile, message, ok := parseSystemProfile(src)
	if !ok {
		options.SystemError = message
		return parseFail(options, ParseErrInvalidSystem, options.System, options.SystemAt)
	}
	if !IsSupportedTarget(profile.Target) {
		options.SystemError = "unsupported target " + profile.Target
		return parseFail(options, ParseErrInvalidSystem, options.System, options.SystemAt)
	}
	options.SystemName = profile.Name
	options.Target = profile.Target
	options.BinaryLimit = profile.BinaryLimit
	options.ArenaSize = profile.ArenaSize
	if options.WindowsGUI && options.Target != "windows/amd64" && options.Target != "windows/386" && options.Target != "windows/arm64" {
		return parseFail(options, ParseErrWindowsGUIRequiresWindows, options.Target, options.SystemAt)
	}
	if options.Mode == ModeKernelModule && options.Target != "linux/amd64" {
		return parseFail(options, ParseErrModeRequiresLinuxAmd64, options.Target, options.SystemAt)
	}
	return options
}

func parseSystemProfile(src []byte) (SystemProfile, string, bool) {
	parsed, diagnostic, ok := rtg.ParseSystem(src, "<system>")
	if !ok {
		return SystemProfile{}, diagnostic.Message, false
	}
	return SystemProfile{
		Name:        parsed.Name,
		Target:      parsed.Target,
		BinaryLimit: parsed.BinaryLimit,
		ArenaSize:   parsed.ArenaSize,
	}, "", true
}

func systemBinaryLimitDiagnostic(name string, actual int, limit int) Diagnostic {
	return Diagnostic{
		Phase: "system",
		Code:  "RENVO-SYSTEM-002",
		Message: "binary exceeds system " + name + " limit: output " +
			diagnosticIntText(actual) + " bytes, limit " + diagnosticIntText(limit) +
			" bytes, over " + diagnosticIntText(actual-limit) + " bytes",
	}
}
