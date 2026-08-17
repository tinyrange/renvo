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
	ParseErrScriptRequiresGo
	ParseErrObjectRequiresLinuxAmd64
	ParseErrObjectFileCount
	ParseErrMissingIncludePath
)

const ModeExecutable = "executable"
const ModeKernelModule = "kernel-module"
const ModeObject = "object"
const DefaultModuleLicense = "Proprietary"

const RunHelpText = "Usage: renvo run [-s] [-tags <list>] [-arena-size <bytes>] <script.go> [-- arguments...]\nTop-level statements form func main. The script is compiled for and executed on the host target.\n"

type Options struct {
	Target               string
	TargetExplicit       bool
	Output               string
	Package              string
	Files                []string
	Strip                bool
	EmitUnit             bool
	EmitImage            bool
	Script               bool
	WindowsGUI           bool
	Mode                 string
	ModuleLicense        string
	ArenaSize            int
	BinaryLimit          int
	System               string
	SystemName           string
	SystemError          string
	SystemAt             int
	Tags                 []string
	IncludePaths         []string
	CCompiler            bool
	CLanguage            string
	CDefines             []string
	CUndefines           []string
	CForcedInclude       []string
	CNoStdIncludes       bool
	CAssemblyOutput      bool
	CFunctionSections    bool
	CDataSections        bool
	CShortWChar          bool
	DependencyFile       string
	DependencyTarget     string
	CDependencyRoot      string
	CDependencyRequested bool
	CDependencies        []string
	CDependencyData      []byte
	Ok                   bool
	Error                int
	ErrorArg             string
	ErrorAt              int
}

func CommandHelpRequested(args []string) bool {
	return len(args) <= 1 || len(args) == 2 && args[1] == "--help"
}

// NormalizeCCompilerCommand translates the explicitly supported GCC-driver
// surface into private ordinary-driver options. Unrecognized arguments are
// retained so the shared parser rejects them and cc-option probes stay honest.
func NormalizeCCompilerCommand(args []string) []string {
	if len(args) < 2 || args[1] != "cc" {
		return args
	}
	out := make([]string, 1, len(args))
	out[0] = args[0]
	out = append(out, "-cc")
	for i := 2; i < len(args); i++ {
		arg := args[i]
		if arg == "-MMD" || arg == "-MD" {
			out = append(out, "-cc-dependencies")
			continue
		}
		if arg == "-fshort-wchar" {
			out = append(out, "-cc-short-wchar")
			continue
		}
		if cCompilerInertOption(arg) {
			continue
		}
		option, width := cCompilerOptionIndex("-nostdinc|-E|-P|-S|-fsyntax-only", arg, false)
		if option >= 0 {
			switch option {
			case 0:
				out = append(out, "-cc-nostdinc")
			case 1:
				out = append(out, "-cc-preprocess")
			case 2:
				out = append(out, "-cc-no-line-markers")
			case 3:
				out = append(out, "-c", "-cc-assembly-output")
			case 4:
				out = append(out, "-cc-syntax-only")
			}
			continue
		}
		if arg == "-ffunction-sections" {
			out = append(out, "-cc-function-sections")
			continue
		}
		if arg == "-fdata-sections" {
			out = append(out, "-cc-data-sections")
			continue
		}
		option, _ = cCompilerOptionIndex("-x|-include|-D|-U|-MF|-MT", arg, false)
		if option >= 0 {
			if i+1 >= len(args) {
				out = append(out, arg)
				continue
			}
			i++
			prefix := "-cc-language="
			switch option {
			case 1:
				prefix = "-cc-include="
			case 2:
				prefix = "-cc-define="
			case 3:
				prefix = "-cc-undefine="
			case 4:
				prefix = "-cc-depfile="
			case 5:
				prefix = "-cc-deptarget="
			}
			out = append(out, prefix+args[i])
			continue
		}
		option, width = cCompilerOptionIndex("-D|-U|-Wp,-MMD,|-Wp,-MD,|-Wp,-MT,", arg, true)
		if option >= 0 {
			prefix := "-cc-depfile="
			if option == 0 {
				prefix = "-cc-define="
			} else if option == 1 {
				prefix = "-cc-undefine="
			} else if option == 4 {
				prefix = "-cc-deptarget="
			}
			out = append(out, prefix+arg[width:])
			continue
		}
		out = append(out, arg)
	}
	return out
}

const cCompilerInertOptions = "-MMD|-MD|-MP|-pipe|-std=gnu11|-std=c11|-m64|-Os|-O0|-O1|-O2|-O3|-funsigned-char|-fno-common|-fno-PIE|-fno-pie|-fno-strict-aliasing|-fno-asynchronous-unwind-tables|-fno-delete-null-pointer-checks|-fno-stack-protector|-fomit-frame-pointer|-fno-strict-overflow|-fno-stack-check|-fconserve-stack|-fno-builtin-wcslen|-falign-functions=16|-fverbose-asm|-mno-sse|-mno-mmx|-mno-sse2|-mno-3dnow|-mno-avx|-mno-80387|-mtune=generic|-mno-red-zone|-mcmodel=kernel|-Wall|-Wextra|-Wundef|-Werror|-Werror=implicit-function-declaration|-Werror=implicit-int|-Werror=return-type|-Werror=strict-prototypes|-Wno-format-security|-Wno-trigraphs|-Wmissing-declarations|-Wmissing-prototypes|-Wframe-larger-than=2048|-Wno-main|-Wvla|-Wno-pointer-sign|-Werror=date-time|-Wunused|-Wno-override-init|-Wno-missing-field-initializers|-Wno-type-limits|-Wno-shift-negative-value|-Wno-maybe-uninitialized|-Wno-sign-compare|-Wno-unused-parameter"

func cCompilerInertOption(arg string) bool {
	option, _ := cCompilerOptionIndex(cCompilerInertOptions, arg, false)
	return option >= 0
}

func cCompilerOptionIndex(options string, arg string, prefix bool) (int, int) {
	option := 0
	for start := 0; start < len(options); {
		end := start
		for end < len(options) && options[end] != '|' {
			end++
		}
		width := end - start
		if width == len(arg) || prefix && width < len(arg) {
			match := true
			for i := 0; i < width; i++ {
				if arg[i] != options[start+i] {
					match = false
					break
				}
			}
			if match {
				return option, width
			}
		}
		start = end + 1
		option++
	}
	return -1, 0
}

func ParseOptions(args []string) Options {
	return parseOptions(args, true)
}

// parseOptions separates command-line syntax from release-target membership.
// External backend definitions resolve their descriptor before this function
// is called and therefore do not need to masquerade as an advertised target.
func parseOptions(args []string, requireAdvertisedTarget bool) Options {
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
		if arg == "-cc" {
			options.CCompiler = true
			i++
			continue
		}
		if len(arg) >= 13 && arg[:13] == "-cc-language=" {
			options.CLanguage = arg[13:]
			if options.CLanguage != "c" {
				return parseFail(options, ParseErrUnknownOption, "-x "+options.CLanguage, i)
			}
			i++
			continue
		}
		if len(arg) >= 11 && arg[:11] == "-cc-define=" {
			options.CDefines = append(options.CDefines, arg[11:])
			i++
			continue
		}
		if len(arg) >= 13 && arg[:13] == "-cc-undefine=" {
			options.CUndefines = append(options.CUndefines, arg[13:])
			i++
			continue
		}
		if len(arg) >= 12 && arg[:12] == "-cc-include=" {
			options.CForcedInclude = append(options.CForcedInclude, arg[12:])
			i++
			continue
		}
		if arg == "-cc-nostdinc" {
			options.CNoStdIncludes = true
			i++
			continue
		}
		if arg == "-cc-assembly-output" {
			options.CAssemblyOutput = true
			i++
			continue
		}
		if arg == "-cc-function-sections" {
			options.CFunctionSections = true
			i++
			continue
		}
		if arg == "-cc-data-sections" {
			options.CDataSections = true
			i++
			continue
		}
		if arg == "-cc-short-wchar" {
			options.CShortWChar = true
			i++
			continue
		}
		if arg == "-cc-dependencies" {
			options.CDependencyRequested = true
			i++
			continue
		}
		if len(arg) >= 12 && arg[:12] == "-cc-depfile=" {
			options.DependencyFile = arg[12:]
			i++
			continue
		}
		if len(arg) >= 14 && arg[:14] == "-cc-deptarget=" {
			options.DependencyTarget = arg[14:]
			i++
			continue
		}
		if arg == "-s" {
			options.Strip = true
			i++
			continue
		}
		if arg == "-c" {
			options.Mode = ModeObject
			modeAt = i
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
			if mode != ModeExecutable && mode != ModeKernelModule && mode != ModeObject {
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
		if arg == "-I" || arg == "-isystem" {
			if i+1 >= len(args) {
				return parseFail(options, ParseErrMissingIncludePath, arg, i)
			}
			options.IncludePaths = append(options.IncludePaths, args[i+1])
			i += 2
			continue
		}
		if len(arg) > 2 && arg[0] == '-' && arg[1] == 'I' {
			options.IncludePaths = append(options.IncludePaths, arg[2:])
			i++
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
			if optionArgIsSourceFile(arg) {
				options.Files = append(options.Files, arg)
			}
			i++
			continue
		}
		if len(options.Files) == 0 {
			return parseFail(options, ParseErrExtraPackage, arg, i)
		}
		if !optionArgIsSourceFile(arg) {
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
	if requireAdvertisedTarget && !IsSupportedTarget(options.Target) {
		return parseFail(options, ParseErrUnsupportedTarget, options.Target, targetAt)
	}
	if options.Script && len(options.Files) == 0 {
		return parseFail(options, ParseErrScriptRequiresFile, options.Package, len(args))
	}
	if options.Script && len(options.Files) != 1 {
		return parseFail(options, ParseErrScriptFileCount, options.Package, len(args))
	}
	if options.Script && !optionArgIsGoFile(options.Files[0]) {
		return parseFail(options, ParseErrScriptRequiresGo, options.Files[0], len(args))
	}
	if options.Mode == ModeObject && len(options.Files) != 1 {
		return parseFail(options, ParseErrObjectFileCount, options.Package, len(args))
	}
	if requireAdvertisedTarget && options.System == "" {
		if options.WindowsGUI && options.Target != "windows/amd64" && options.Target != "windows/386" && options.Target != "windows/arm64" {
			return parseFail(options, ParseErrWindowsGUIRequiresWindows, options.Target, windowsGUIAt)
		}
		if options.Mode == ModeKernelModule && options.Target != "linux/amd64" {
			return parseFail(options, ParseErrModeRequiresLinuxAmd64, options.Target, modeAt)
		}
		if options.Mode == ModeObject && options.Target != "linux/amd64" {
			return parseFail(options, ParseErrObjectRequiresLinuxAmd64, options.Target, modeAt)
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

func optionArgIsCFile(arg string) bool {
	return len(arg) > 2 && arg[len(arg)-2:] == ".c"
}

func optionArgIsPreprocessedCFile(arg string) bool {
	return len(arg) > 2 && arg[len(arg)-2:] == ".i"
}

func optionArgIsSourceFile(arg string) bool {
	return optionArgIsGoFile(arg) || optionArgIsCFile(arg) || optionArgIsPreprocessedCFile(arg)
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
