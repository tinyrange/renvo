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
		Predefined: cCommandMacros(options.CDefines), Undefined: options.CUndefines,
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

func cCommandMacros(definitions []string) []c11.Macro {
	macros := make([]c11.Macro, 0, len(definitions))
	for i := 0; i < len(definitions); i++ {
		name := definitions[i]
		value := "1"
		for j := 0; j < len(definitions[i]); j++ {
			if definitions[i][j] == '=' {
				name = definitions[i][:j]
				value = definitions[i][j+1:]
				break
			}
		}
		macros = append(macros, c11.Macro{Name: name, Value: value})
	}
	return macros
}
