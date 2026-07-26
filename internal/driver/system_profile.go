package driver

import (
	"strings"

	"renvo.dev/internal/load"
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
	var profile SystemProfile
	fields := strings.Fields(string(src))
	if len(fields) < 3 || fields[0] != "system" {
		return profile, "expected system declaration", false
	}
	name, ok := systemQuoted(fields[1])
	if !ok || name == "" {
		return profile, "expected non-empty quoted system name", false
	}
	if fields[2] != "{" || len(fields) < 4 || fields[len(fields)-1] != "}" {
		return profile, "expected { and } around system fields", false
	}
	profile.Name = name
	haveTarget, haveBinary, haveArena := false, false, false
	for at := 3; at < len(fields)-1; at += 3 {
		if at+2 >= len(fields)-1 || fields[at+1] != "=" {
			return profile, "expected field = value entries", false
		}
		field, value := fields[at], fields[at+2]
		if field == "target" {
			if haveTarget {
				return profile, "duplicate target field", false
			}
			profile.Target, ok = systemQuoted(value)
			if !ok || profile.Target == "" {
				return profile, "target must be a non-empty quoted string", false
			}
			haveTarget = true
		} else if field == "binary" {
			if haveBinary {
				return profile, "duplicate binary field", false
			}
			profile.BinaryLimit, ok = parseSystemSize(value)
			if !ok || profile.BinaryLimit <= 0 {
				return profile, "binary must be a positive byte size", false
			}
			haveBinary = true
		} else if field == "arena" {
			if haveArena {
				return profile, "duplicate arena field", false
			}
			profile.ArenaSize, ok = parseSystemSize(value)
			if !ok || profile.ArenaSize < 256 || profile.ArenaSize > 1073741824 {
				return profile, "arena must be between 256B and 1GiB", false
			}
			haveArena = true
		} else {
			return profile, "unknown system field " + field, false
		}
	}
	if !haveTarget {
		return profile, "missing target field", false
	}
	if !haveBinary {
		return profile, "missing binary field", false
	}
	if !haveArena {
		return profile, "missing arena field", false
	}
	return profile, "", true
}

func systemQuoted(text string) (string, bool) {
	if len(text) < 2 || text[0] != '"' || text[len(text)-1] != '"' {
		return "", false
	}
	for i := 1; i < len(text)-1; i++ {
		if text[i] == '"' || text[i] == '\\' {
			return "", false
		}
	}
	return text[1 : len(text)-1], true
}

func parseSystemSize(text string) (int, bool) {
	multiplier := 1
	digits := len(text)
	if strings.HasSuffix(text, "KiB") {
		multiplier, digits = 1024, digits-3
	} else if strings.HasSuffix(text, "MiB") {
		multiplier, digits = 1024*1024, digits-3
	} else if strings.HasSuffix(text, "GiB") {
		multiplier, digits = 1024*1024*1024, digits-3
	} else if strings.HasSuffix(text, "B") {
		digits--
	}
	if digits <= 0 {
		return 0, false
	}
	value := 0
	for i := 0; i < digits; i++ {
		if text[i] < '0' || text[i] > '9' {
			return 0, false
		}
		digit := int(text[i] - '0')
		if value > (1073741824-digit)/10 {
			return 0, false
		}
		value = value*10 + digit
	}
	if value > 1073741824/multiplier {
		return 0, false
	}
	return value * multiplier, true
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
