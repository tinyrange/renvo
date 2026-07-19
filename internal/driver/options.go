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
)

const DefaultTarget = "linux/amd64"

const HelpText = "Usage: renvo -o <file> [-t <target>] [-tags <list>] [-arena-size <bytes>] [-s] [-emit-unit] [-windows-gui] <package | file.go...>\nOptions:\n  -arena-size   set the generated program arena limit in bytes (256..1073741824)\n  -emit-unit    write the canonical linked Renvo unit without invoking a backend\n  -windows-gui  select the Windows GUI subsystem instead of the console subsystem\nSource files:\n  Explicit .go files must share one directory and package. Exactly the named files are used;\n  build constraints and OS/architecture suffixes are ignored, while _test.go files are skipped.\nTargets:\n  linux/amd64 linux/386 linux/aarch64 linux/arm\n  windows/amd64 windows/386 windows/arm64 darwin/arm64 wasi/wasm32\nUnsupported language/toolchain features:\n  generics, goroutines, channels, select, cgo\n"

type Options struct {
	Target     string
	Output     string
	Package    string
	Files      []string
	Strip      bool
	EmitUnit   bool
	WindowsGUI bool
	ArenaSize  int
	Tags       []string
	Ok         bool
	Error      int
	ErrorArg   string
	ErrorAt    int
}

func CommandHelpRequested(args []string) bool {
	return len(args) <= 1 || len(args) == 2 && args[1] == "--help"
}

func ParseOptions(args []string) Options {
	options := Options{
		Target:  DefaultTarget,
		Ok:      true,
		Error:   ParseOK,
		ErrorAt: -1,
	}
	i := 0
	windowsGUIAt := -1
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
		if arg == "-windows-gui" {
			options.WindowsGUI = true
			windowsGUIAt = i
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
	if options.WindowsGUI && options.Target != "windows/amd64" && options.Target != "windows/386" && options.Target != "windows/arm64" {
		return parseFail(options, ParseErrWindowsGUIRequiresWindows, options.Target, windowsGUIAt)
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
