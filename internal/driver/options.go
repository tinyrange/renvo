package driver

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
)

const DefaultTarget = "linux/amd64"
const ModeExecutable = "executable"
const ModeKernelModule = "kernel-module"
const DefaultModuleLicense = "Proprietary"

const HelpText = "Usage: renvo -o <file> [-t <target>] [-mode=<mode>] [-tags <list>] [-arena-size <bytes>] [-s] [-emit-unit] [-emit-image] [-windows-gui] <package | file.go...>\n       renvo run [build options] <script.go> [-- script arguments...]\n       renvo test [build options] [package]\nOptions:\n  -arena-size  set the generated program arena limit in bytes (256..1073741824)\n  -emit-unit   write the canonical linked Renvo unit without invoking a backend\n  -emit-image  write a format-neutral linked image instead of an executable\n  -mode        select executable (default) or kernel-module output\n  -script      compile one file whose top-level statements form func main\n  -windows-gui select the Windows GUI subsystem instead of the console subsystem\nSource files:\n  Explicit .go files must share one directory and package. Exactly the named files are used;\n  build constraints and OS/architecture suffixes are ignored, while _test.go files are skipped.\nTargets:\n  linux/amd64 linux/386 linux/aarch64 linux/arm\n  windows/amd64 windows/386 windows/arm64 darwin/arm64 wasi/wasm32 browser/wasm32\nUnsupported language/toolchain features:\n  generics, goroutines, channels, select, cgo\n"

const RunHelpText = "Usage: renvo run [-s] [-tags <list>] [-arena-size <bytes>] <script.go> [-- arguments...]\nTop-level statements form func main. The script is compiled for and executed on the host target.\n"

type Options struct {
	Target        string
	Output        string
	Package       string
	Files         []string
	Strip         bool
	EmitUnit      bool
	EmitImage     bool
	Script        bool
	WindowsGUI    bool
	Mode          string
	ModuleLicense string
	ArenaSize     int
	Tags          []string
	Ok            bool
	Error         int
	ErrorArg      string
	ErrorAt       int
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
	}
	i := 0
	windowsGUIAt := -1
	modeAt := -1
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
			if !IsSupportedTarget(target) {
				return parseFail(options, ParseErrUnsupportedTarget, target, i+1)
			}
			options.Target = target
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
	if options.EmitUnit && options.EmitImage {
		return parseFail(options, ParseErrConflictingEmit, "-emit-unit", len(args))
	}
	if options.Script && len(options.Files) == 0 {
		return parseFail(options, ParseErrScriptRequiresFile, options.Package, len(args))
	}
	if options.Script && len(options.Files) != 1 {
		return parseFail(options, ParseErrScriptFileCount, options.Package, len(args))
	}
	if options.WindowsGUI && options.Target != "windows/amd64" && options.Target != "windows/386" && options.Target != "windows/arm64" {
		return parseFail(options, ParseErrWindowsGUIRequiresWindows, options.Target, windowsGUIAt)
	}
	if options.Mode == ModeKernelModule && options.Target != "linux/amd64" {
		return parseFail(options, ParseErrModeRequiresLinuxAmd64, options.Target, modeAt)
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
	if target == "linux/amd64" {
		return true
	}
	if target == "linux/386" {
		return true
	}
	if target == "linux/aarch64" {
		return true
	}
	if target == "linux/arm" {
		return true
	}
	if target == "windows/amd64" {
		return true
	}
	if target == "windows/386" {
		return true
	}
	if target == "windows/arm64" {
		return true
	}
	if target == "wasi/wasm32" {
		return true
	}
	if target == "browser/wasm32" {
		return true
	}
	if target == "darwin/arm64" {
		return true
	}
	return false
}

func parseFail(options Options, err int, arg string, at int) Options {
	options.Ok = false
	options.Error = err
	options.ErrorArg = arg
	options.ErrorAt = at
	return options
}
