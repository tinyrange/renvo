package driver

import "renvo.dev/internal/targetinfo"

const (
	ParseOK = iota
	ParseErrMissingOutput
	ParseErrMissingTarget
	ParseErrUnsupportedTarget
	ParseErrUnknownOption
	ParseErrMissingTags
	ParseErrInvalidTags
	ParseErrMissingPackage
	ParseErrExtraPackage
	ParseErrWindowsGUIRequiresWindows
	ParseErrMixedFileList
	ParseErrMissingArenaSize
	ParseErrInvalidArenaSize
	ParseErrMissingMode
	ParseErrUnsupportedMode
	ParseErrModeRequiresLinuxAmd64
	ParseErrInvalidModuleLicense
	ParseErrConflictingModuleLicense
	ParseErrScriptRequiresFile
	ParseErrScriptFileCount
	ParseErrConflictingEmit
	ParseErrMissingSystem
	ParseErrSystemRead
	ParseErrInvalidSystem
	ParseErrSystemTargetConflict
	ParseErrSystemArenaConflict
	ParseErrMissingBackend
	ParseErrBackendRead
	ParseErrInvalidBackend
	ParseErrBackendTarget
)

const ModeExecutable = "executable"
const ModeKernelModule = "kernel-module"
const DefaultModuleLicense = "Proprietary"

const RunHelpText = "Usage: renvo run [-s] [-tags <list>] [-arena-size <bytes>] <script.go> [-- arguments...]\nTop-level statements form func main. The script is compiled for and executed on the host target.\n"

type Options struct {
	Target         string
	TargetExplicit bool
	Output         string
	Package        string
	Files          []string
	Strip          bool
	EmitUnit       bool
	EmitImage      bool
	Script         bool
	WindowsGUI     bool
	Mode           string
	ModuleLicense  string
	ArenaSize      int
	BinaryLimit    int
	System         string
	SystemName     string
	SystemError    string
	SystemAt       int
	Tags           []string
	Ok             bool
	Error          int
	ErrorArg       string
	ErrorAt        int
}

func CommandHelpRequested(args []string) bool {
	return len(args) <= 1 || len(args) == 2 && args[1] == "--help"
}

func ParseOptions(args []string) Options {
	options := Options{
		Target:        DefaultTarget,
		Mode:          ModeExecutable,
		ModuleLicense: DefaultModuleLicense,
		Ok:            true,
		Error:         ParseOK,
		ErrorAt:       -1,
		SystemAt:      -1,
	}
	i := 0
	windowsGUIAt := -1
	modeAt := -1
	targetAt := -1
	arenaAt := -1
	for i < len(args) {
		arg := args[i]
		if arg == "-s" {
			options.Strip = true
			i++
			continue
		}
		if arg == "-emit-unit" {
			options.EmitUnit = true
			i++
			continue
		}
		if arg == "-emit-image" {
			options.EmitImage = true
			i++
			continue
		}
		if arg == "-script" {
			options.Script = true
			i++
			continue
		}
		if arg == "-windows-gui" {
			options.WindowsGUI = true
			windowsGUIAt = i
			i++
			continue
		}
		if len(arg) >= 6 && arg[:6] == "-mode=" {
			mode := arg[6:]
			if mode == "" {
				return parseFail(options, ParseErrMissingMode, arg, i)
			}
			if mode != ModeExecutable && mode != ModeKernelModule {
				return parseFail(options, ParseErrUnsupportedMode, mode, i)
			}
			options.Mode = mode
			modeAt = i
			i++
			continue
		}
		if arg == "-arena-size" {
			if i+1 >= len(args) {
				return parseFail(options, ParseErrMissingArenaSize, arg, i)
			}
			size, ok := parseArenaSize(args[i+1])
			if !ok {
				return parseFail(options, ParseErrInvalidArenaSize, args[i+1], i+1)
			}
			options.ArenaSize = size
			arenaAt = i
			i += 2
			continue
		}
		if arg == "-system" {
			if i+1 >= len(args) {
				return parseFail(options, ParseErrMissingSystem, arg, i)
			}
			if options.System != "" {
				options.SystemError = "system profile may only be specified once"
				return parseFail(options, ParseErrInvalidSystem, args[i+1], i)
			}
			options.System = args[i+1]
			options.SystemAt = i
			i += 2
			continue
		}
		if arg == "-o" {
			if i+1 >= len(args) {
				return parseFail(options, ParseErrMissingOutput, arg, i)
			}
			options.Output = args[i+1]
			i += 2
			continue
		}
		if arg == "-t" {
			if i+1 >= len(args) {
				return parseFail(options, ParseErrMissingTarget, arg, i)
			}
			target := args[i+1]
			options.Target = target
			options.TargetExplicit = true
			targetAt = i + 1
			i += 2
			continue
		}
		if arg == "-tags" {
			if i+1 >= len(args) {
				return parseFail(options, ParseErrMissingTags, arg, i)
			}
			tags, ok := parseBuildTags(args[i+1])
			if !ok {
				return parseFail(options, ParseErrInvalidTags, args[i+1], i+1)
			}
			for j := 0; j < len(tags); j++ {
				if findString(options.Tags, tags[j]) < 0 {
					options.Tags = append(options.Tags, tags[j])
				}
			}
			i += 2
			continue
		}
		if len(arg) > 0 && arg[0] == '-' {
			return parseFail(options, ParseErrUnknownOption, arg, i)
		}
		if options.Package == "" {
			options.Package = arg
			if optionArgIsGoFile(arg) {
				options.Files = append(options.Files, arg)
			}
			i++
			continue
		}
		if len(options.Files) == 0 {
			return parseFail(options, ParseErrExtraPackage, arg, i)
		}
		if !optionArgIsGoFile(arg) {
			return parseFail(options, ParseErrMixedFileList, arg, i)
		}
		options.Files = append(options.Files, arg)
		i++
	}
	if options.Output == "" {
		return parseFail(options, ParseErrMissingOutput, "-o", len(args))
	}
	if options.Package == "" {
		return parseFail(options, ParseErrMissingPackage, "", len(args))
	}
	if options.System != "" && arenaAt >= 0 {
		return parseFail(options, ParseErrSystemArenaConflict, options.System, arenaAt)
	}
	if options.EmitUnit && options.EmitImage {
		return parseFail(options, ParseErrConflictingEmit, "-emit-unit", len(args))
	}
	if !IsSupportedTarget(options.Target) {
		return parseFail(options, ParseErrUnsupportedTarget, options.Target, targetAt)
	}
	if options.Script && len(options.Files) == 0 {
		return parseFail(options, ParseErrScriptRequiresFile, options.Package, len(args))
	}
	if options.Script && len(options.Files) != 1 {
		return parseFail(options, ParseErrScriptFileCount, options.Package, len(args))
	}
	if options.System == "" {
		if options.WindowsGUI && options.Target != "windows/amd64" && options.Target != "windows/386" && options.Target != "windows/arm64" {
			return parseFail(options, ParseErrWindowsGUIRequiresWindows, options.Target, windowsGUIAt)
		}
		if options.Mode == ModeKernelModule && options.Target != "linux/amd64" {
			return parseFail(options, ParseErrModeRequiresLinuxAmd64, options.Target, modeAt)
		}
	}
	return options
}

func parseArenaSize(value string) (int, bool) {
	if len(value) == 0 {
		return 0, false
	}
	result := 0
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if ch < '0' || ch > '9' {
			return 0, false
		}
		digit := int(ch - '0')
		if result > (1073741824-digit)/10 {
			return 0, false
		}
		result = result*10 + digit
	}
	if result < 256 || result > 1073741824 {
		return 0, false
	}
	return result, true
}

func optionArgIsGoFile(arg string) bool {
	return len(arg) > 3 && arg[len(arg)-3:] == ".go"
}

func parseBuildTags(value string) ([]string, bool) {
	if len(value) == 0 {
		return nil, false
	}
	var tags []string
	start := 0
	for i := 0; i <= len(value); i++ {
		if i < len(value) && value[i] != ',' {
			if !isBuildTagChar(value[i]) {
				return nil, false
			}
			continue
		}
		if i == start {
			return nil, false
		}
		tags = append(tags, value[start:i])
		start = i + 1
	}
	return tags, true
}

func IsSupportedTarget(target string) bool {
	return targetinfo.IsAdvertised(target)
}

func parseFail(options Options, err int, arg string, at int) Options {
	options.Ok = false
	options.Error = err
	options.ErrorArg = arg
	options.ErrorAt = at
	return options
}
