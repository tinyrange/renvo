package driver

import (
	"renvo.dev/internal/arena"
	"renvo.dev/internal/c11"
	"renvo.dev/internal/load"
)

type CSyntaxCommandResult struct {
	Ok             bool
	Error          int
	ErrorAt        int
	ErrorLine      int
	ErrorColumn    int
	ErrorPath      string
	Preprocess     bool
	Detail         string
	Option         string
	DependencyFile string
	DependencyData []byte
}

func CSyntaxCommandRequested(args []string) bool {
	for i := 1; i < len(args); i++ {
		if args[i] == "-cc-syntax-only" {
			return true
		}
	}
	return false
}

// CheckCCommand preprocesses and semantically checks one C translation unit.
// Operations deliberately deferred past M4 (notably GNU inline assembly) are
// parsed and validated without pretending that object lowering exists.
func CheckCCommand(args []string, workDir string, fs SourceFS) CSyntaxCommandResult {
	ordinary := make([]string, 0, len(args)+3)
	for i := 1; i < len(args); i++ {
		if args[i] != "-cc-syntax-only" {
			ordinary = append(ordinary, args[i])
		}
	}
	ordinary = append(ordinary, "-c")
	hasOutput := false
	for i := 0; i < len(ordinary); i++ {
		if ordinary[i] == "-o" {
			hasOutput = true
			break
		}
	}
	if !hasOutput {
		ordinary = append(ordinary, "-o", "-")
	}
	options := ParseOptions(ordinary)
	result := CSyntaxCommandResult{Ok: options.Ok, ErrorAt: -1, Option: options.ErrorArg}
	if !options.Ok {
		return result
	}
	options = resolveCCompilerPaths(workDir, options)
	path := load.JoinPath(workDir, options.Files[0])
	source, ok := fs.ReadFile(path)
	if !ok {
		result.Ok = false
		result.Error = c11.PreprocessErrInclude
		result.ErrorPath = path
		result.ErrorLine = 1
		result.ErrorColumn = 1
		result.Preprocess = true
		return result
	}
	if optionArgIsPreprocessedCFile(path) {
		checkMark := arena.Mark()
		checkPersistMark := arena.PersistMark()
		checked := c11.CheckObjectForDataModel(source, c11.DataModelLP64)
		result.Ok = checked.Ok
		result.Error = checked.Error
		result.ErrorAt = checked.ErrorAt
		result.ErrorPath = path
		result.ErrorLine, result.ErrorColumn = cSourcePosition(source, checked.ErrorAt)
		arena.PersistReset(checkPersistMark)
		arena.Reset(checkMark)
		return result
	}
	reader := cObjectIncludeReader{fs: fs, paths: cObjectIncludePaths(workDir, options.IncludePaths, options.CNoStdIncludes, fs)}
	preprocessMark := arena.Mark()
	processed := c11.Preprocess(c11.PreprocessConfig{
		Path: path, Source: source, Reader: reader,
		Predefined: cCommandMacros(options.CDefines), Undefined: options.CUndefines,
		ForcedIncludes: options.CForcedInclude, EmitIncludes: options.CNoStdIncludes, EmitQuotedIncludes: true,
		SuppressForcedIncludes: true,
	})
	if !processed.Ok {
		result.Ok = false
		result.Error = processed.Error
		result.ErrorPath = processed.ErrorPath
		result.ErrorLine = processed.Line
		result.ErrorColumn = 1
		result.Preprocess = true
		result.Detail = processed.Detail
		return result
	}
	result.DependencyFile = options.DependencyFile
	if options.DependencyFile != "" {
		options.CDependencies = appendUniquePath(options.CDependencies, path)
		for i := 0; i < len(processed.Dependencies); i++ {
			options.CDependencies = appendUniquePath(options.CDependencies, processed.Dependencies[i])
		}
		result.DependencyData = arena.PersistBytes(CDependencyOutput(options))
	}
	// Only the normalized token stream survives into semantic checking. Moving
	// it to the opposite arena before resetting the preprocessing phase keeps a
	// self-hosted compiler bounded by the larger phase instead of their sum.
	processedPersistMark := arena.PersistMark()
	processedSource := arena.PersistBytes(processed.Source)
	arena.Reset(preprocessMark)
	checkMark := arena.Mark()
	checked := c11.CheckObjectForDataModel(processedSource, c11.DataModelLP64)
	result.Ok = checked.Ok
	result.Error = checked.Error
	result.ErrorAt = checked.ErrorAt
	result.ErrorPath = path
	result.ErrorLine, result.ErrorColumn = cSourcePosition(processedSource, checked.ErrorAt)
	arena.Reset(checkMark)
	arena.PersistReset(processedPersistMark)
	return result
}

func CSyntaxCommandDiagnostic(result CSyntaxCommandResult) Diagnostic {
	if result.Option != "" {
		return Diagnostic{Phase: "options", Code: "RENVO-OPTION-005", Message: "unknown option " + result.Option}
	}
	if result.Preprocess || result.Error == c11.PreprocessErrInclude {
		return cPreprocessDiagnostic(result.Error, result.ErrorPath, result.ErrorLine, result.Detail)
	}
	return Diagnostic{Phase: "c11", Code: "RENVO-C11-001", Message: "C11 source is not supported or is invalid", Path: result.ErrorPath, Line: result.ErrorLine, Column: result.ErrorColumn}
}

func cSourcePosition(source []byte, offset int) (int, int) {
	line, column := 1, 1
	for i := 0; i < offset && i < len(source); i++ {
		if source[i] == '\n' {
			line, column = line+1, 1
		} else {
			column++
		}
	}
	return line, column
}
