package driver

import (
	"renvo.dev/internal/c11"
	"renvo.dev/internal/load"
)

type CPreprocessCommandResult struct {
	Source         []byte
	Output         string
	Ok             bool
	Error          int
	ErrorPath      string
	ErrorLine      int
	Detail         string
	Option         string
	DependencyFile string
	DependencyData []byte
}

func CPreprocessCommandRequested(args []string) bool {
	for i := 1; i < len(args); i++ {
		if args[i] == "-cc-preprocess" {
			return true
		}
	}
	return false
}

func CPreprocessCommandUsesStandardInput(args []string) bool {
	for i := 1; i < len(args); i++ {
		if args[i] == "-cc-stdin" {
			return true
		}
	}
	return false
}

// PreprocessCCommand reuses the strict ordinary-driver parser after removing
// the two output-only switches. This keeps one definition of the supported C
// compiler surface without adding preprocessing state to the compact Options
// value copied through the package/backend pipeline.
func PreprocessCCommand(args []string, workDir string, fs SourceFS) CPreprocessCommandResult {
	return PreprocessCCommandWithInput(args, workDir, fs, nil)
}

func PreprocessCCommandWithInput(args []string, workDir string, fs SourceFS, input []byte) CPreprocessCommandResult {
	ordinary := make([]string, 0, len(args)+2)
	lineMarkers := true
	hasOutput := false
	standardInput := false
	nullInput := false
	macroDump := false
	for i := 1; i < len(args); i++ {
		if args[i] == "-cc-preprocess" {
			continue
		}
		if args[i] == "-cc-no-line-markers" {
			lineMarkers = false
			continue
		}
		if args[i] == "-cc-stdin" {
			ordinary = append(ordinary, "__renvo_stdin__.c")
			standardInput = true
			continue
		}
		if args[i] == "-cc-null-input" {
			ordinary = append(ordinary, "__renvo_null__.c")
			nullInput = true
			continue
		}
		if args[i] == "-cc-macro-dump" {
			macroDump = true
			continue
		}
		if args[i] == "-o" {
			hasOutput = true
		}
		ordinary = append(ordinary, args[i])
	}
	ordinary = append(ordinary, "-c")
	if !hasOutput {
		ordinary = append(ordinary, "-o", "-")
	}
	options := ParseOptions(ordinary)
	result := CPreprocessCommandResult{Output: options.Output, Ok: options.Ok, Option: options.ErrorArg}
	if !options.Ok {
		return result
	}
	options = resolveCCompilerPaths(workDir, options)
	path := load.JoinPath(workDir, options.Files[0])
	source, ok := fs.ReadFile(path)
	if standardInput {
		path, source, ok = "<stdin>", input, true
	} else if nullInput {
		path, source, ok = "/dev/null", nil, true
	}
	if !ok {
		result.Ok = false
		result.Error = c11.PreprocessErrInclude
		result.ErrorPath = path
		return result
	}
	reader := cObjectIncludeReader{fs: fs, paths: cObjectIncludePaths(workDir, options.IncludePaths, options.CNoStdIncludes, fs)}
	processed := c11.Preprocess(c11.PreprocessConfig{
		Path: path, Source: source, Reader: reader,
		Predefined: cCommandMacros(options), Undefined: cCommandUndefined(options),
		ForcedIncludes: options.CForcedInclude, EmitIncludes: true, LineMarkers: lineMarkers, MacroDump: macroDump,
	})
	result.Source = processed.Source
	result.Ok = processed.Ok
	result.Error = processed.Error
	result.ErrorPath = processed.ErrorPath
	result.ErrorLine = processed.Line
	result.Detail = processed.Detail
	result.DependencyFile = options.DependencyFile
	if processed.Ok && options.DependencyFile != "" {
		options.CDependencies = appendUniquePath(options.CDependencies, path)
		for i := 0; i < len(processed.Dependencies); i++ {
			options.CDependencies = appendUniquePath(options.CDependencies, processed.Dependencies[i])
		}
		result.DependencyData = CDependencyOutput(options)
	}
	return result
}

func CPreprocessCommandDiagnostic(result CPreprocessCommandResult) Diagnostic {
	if result.Option != "" {
		return Diagnostic{Phase: "options", Code: "RENVO-OPTION-005", Message: "unknown option " + result.Option}
	}
	return cPreprocessDiagnostic(result.Error, result.ErrorPath, result.ErrorLine, result.Detail)
}

func cPreprocessDiagnostic(preprocessError int, path string, line int, detail string) Diagnostic {
	message := "C preprocessing failed"
	code := "RENVO-CPP-002"
	if preprocessError == c11.PreprocessErrToken {
		message = "invalid preprocessing token or unterminated comment/literal"
		if detail != "" {
			message += ": " + detail
		}
	} else if preprocessError == c11.PreprocessErrDirective {
		message = "invalid or unsupported preprocessing directive"
		if detail != "" {
			message += ": " + detail
		}
	} else if preprocessError == c11.PreprocessErrExpression {
		message = "invalid preprocessor constant expression"
	} else if preprocessError == c11.PreprocessErrMacro {
		message = "invalid macro definition or expansion"
		if detail != "" {
			message += ": " + detail
		}
	} else if preprocessError == c11.PreprocessErrDepth {
		message = "preprocessor include or expansion depth exceeded"
	} else if preprocessError == c11.PreprocessErrInclude {
		message = "C include could not be read: " + path
		code = "RENVO-CPP-001"
	}
	return Diagnostic{Phase: "preprocessor", Code: code, Message: message, Path: path, Line: line, Column: 1}
}

func cCommandMacros(options Options) []c11.Macro {
	macros := make([]c11.Macro, 0, len(options.CDefines)+32)
	targetOS, targetISA, pointerBits := cCompilerTarget(options.Target)
	if pointerBits == 32 {
		macros = append(macros,
			c11.Macro{Name: "__ILP32__", Value: "1"},
			c11.Macro{Name: "_ILP32", Value: "1"},
			c11.Macro{Name: "__SIZEOF_LONG__", Value: "4"},
			c11.Macro{Name: "__SIZEOF_POINTER__", Value: "4"},
			c11.Macro{Name: "__SIZEOF_SIZE_T__", Value: "4"},
			c11.Macro{Name: "__SIZEOF_PTRDIFF_T__", Value: "4"},
			c11.Macro{Name: "__SIZE_TYPE__", Value: "unsigned int"},
			c11.Macro{Name: "__PTRDIFF_TYPE__", Value: "int"},
			c11.Macro{Name: "__INTPTR_TYPE__", Value: "int"},
			c11.Macro{Name: "__UINTPTR_TYPE__", Value: "unsigned int"},
		)
	}
	if targetOS == "windows" && pointerBits == 64 {
		macros = append(macros,
			c11.Macro{Name: "_WIN64", Value: "1"},
			c11.Macro{Name: "__SIZEOF_LONG__", Value: "4"},
			c11.Macro{Name: "__SIZEOF_POINTER__", Value: "8"},
			c11.Macro{Name: "__SIZEOF_SIZE_T__", Value: "8"},
			c11.Macro{Name: "__SIZEOF_PTRDIFF_T__", Value: "8"},
			c11.Macro{Name: "__SIZE_TYPE__", Value: "long long unsigned int"},
			c11.Macro{Name: "__PTRDIFF_TYPE__", Value: "long long int"},
			c11.Macro{Name: "__INTPTR_TYPE__", Value: "long long int"},
			c11.Macro{Name: "__UINTPTR_TYPE__", Value: "long long unsigned int"},
		)
	}
	targetMacros := cTargetPredefinedMacros(targetOS, targetISA)
	for i := 0; i < len(targetMacros); i++ {
		macros = append(macros, targetMacros[i])
	}
	if options.CCompiler && options.Mode == ModeExecutable {
		macros = append(macros, c11.Macro{Name: "__STDC_HOSTED__", Value: "1"})
	}
	if targetOS == "vm" && targetISA == "vm32" {
		// VM32 does not currently implement the binary floating-point
		// conversions required by printf's decimal formatter. This standard
		// feature-test macro lets libc retain the rest of stdio on that target.
		macros = append(macros, c11.Macro{Name: "__STDC_NO_IEC_60559_BFP__", Value: "1"})
	}
	if options.CUnsignedChar {
		macros = append(macros, c11.Macro{Name: "__CHAR_UNSIGNED__", Value: "1"})
	}
	for i := 0; i < len(options.CDefines); i++ {
		name := options.CDefines[i]
		value := "1"
		for j := 0; j < len(options.CDefines[i]); j++ {
			if options.CDefines[i][j] == '=' {
				name = options.CDefines[i][:j]
				value = options.CDefines[i][j+1:]
				break
			}
		}
		macros = append(macros, c11.Macro{Name: name, Value: value})
	}
	return macros
}

func cCommandUndefined(options Options) []string {
	undefined := make([]string, len(options.CUndefines))
	copy(undefined, options.CUndefines)
	targetOS, targetISA, pointerBits := cCompilerTarget(options.Target)
	if targetISA != "amd64" {
		undefined = append(undefined, "__x86_64__", "__x86_64", "__amd64__")
	}
	if pointerBits != 64 || targetOS == "windows" {
		undefined = append(undefined, "__LP64__", "_LP64")
	}
	if targetOS != "linux" {
		undefined = append(undefined, "__linux__", "__linux", "linux")
	}
	if targetOS != "linux" && targetOS != "freebsd" && targetOS != "openbsd" && targetOS != "netbsd" {
		undefined = append(undefined, "__ELF__")
	}
	return undefined
}

func cTargetPredefinedMacros(targetOS string, targetISA string) []c11.Macro {
	var macros []c11.Macro
	switch targetISA {
	case "386":
		macros = append(macros, c11.Macro{Name: "__i386__", Value: "1"}, c11.Macro{Name: "__i386", Value: "1"}, c11.Macro{Name: "i386", Value: "1"})
	case "aarch64":
		macros = append(macros, c11.Macro{Name: "__aarch64__", Value: "1"})
	case "arm":
		macros = append(macros, c11.Macro{Name: "__arm__", Value: "1"})
	case "wasm32":
		macros = append(macros, c11.Macro{Name: "__wasm__", Value: "1"}, c11.Macro{Name: "__wasm32__", Value: "1"})
	case "riscv32":
		macros = append(macros, c11.Macro{Name: "__riscv", Value: "1"}, c11.Macro{Name: "__riscv_xlen", Value: "32"})
	case "xtensa_lx7":
		macros = append(macros, c11.Macro{Name: "__XTENSA__", Value: "1"}, c11.Macro{Name: "__xtensa__", Value: "1"})
	}
	switch targetOS {
	case "windows":
		macros = append(macros, c11.Macro{Name: "_WIN32", Value: "1"})
	case "darwin":
		macros = append(macros, c11.Macro{Name: "__APPLE__", Value: "1"}, c11.Macro{Name: "__MACH__", Value: "1"})
	case "freebsd":
		macros = append(macros, c11.Macro{Name: "__FreeBSD__", Value: "1"})
	case "openbsd":
		macros = append(macros, c11.Macro{Name: "__OpenBSD__", Value: "1"})
	case "netbsd":
		macros = append(macros, c11.Macro{Name: "__NetBSD__", Value: "1"})
	case "wasi", "browser":
		macros = append(macros, c11.Macro{Name: "__wasi__", Value: "1"})
	}
	return macros
}

func cCompilerTarget(target string) (string, string, int) {
	os := target
	isa := ""
	for i := 0; i < len(target); i++ {
		if target[i] == '/' {
			os = target[:i]
			isa = target[i+1:]
			break
		}
	}
	pointerBits := 64
	if isa == "386" || isa == "arm" || isa == "wasm32" || isa == "vm32" || isa == "riscv32" || isa == "xtensa_lx7" {
		pointerBits = 32
	}
	return os, isa, pointerBits
}
