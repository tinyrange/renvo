package driver

import (
	"renvo.dev/internal/c11"
	"renvo.dev/internal/load"
)

type CAssemblyCommandResult struct {
	Source         []byte
	Output         string
	DependencyFile string
	DependencyData []byte
	Ok             bool
	Error          int
	ErrorAt        int
	ErrorPath      string
	Option         string
}

func CAssemblyCommandRequested(args []string) bool {
	for i := 1; i < len(args); i++ {
		if args[i] == "-cc-assembly-output" {
			return true
		}
	}
	return false
}

// CompileCAssemblyCommand implements the narrow, truthful -S surface needed
// by C layout generators. The C frontend evaluates .ascii "->..." metadata
// records and rejects translation units which contain no such records.
func CompileCAssemblyCommand(args []string, workDir string, fs SourceFS) CAssemblyCommandResult {
	options := ParseOptions(args[1:])
	result := CAssemblyCommandResult{Ok: options.Ok, ErrorAt: -1, Option: options.ErrorArg}
	if !options.Ok {
		return result
	}
	options = resolveCCompilerPaths(workDir, options)
	result.Output = options.Output
	if len(options.Files) != 1 {
		result.Ok = false
		result.Error = c11.TranslateErrUnsupported
		return result
	}
	path := load.JoinPath(workDir, options.Files[0])
	source, ok := fs.ReadFile(path)
	if !ok {
		result.Ok = false
		result.ErrorPath = path
		return result
	}
	reader := cObjectIncludeReader{fs: fs, paths: cObjectIncludePaths(workDir, options.IncludePaths, options.CNoStdIncludes, fs)}
	header := c11.HeaderResult{Ok: true, ErrorAt: -1}
	if !options.CNoStdIncludes {
		headerSource := source
		if len(options.CForcedInclude) > 0 {
			prefix := make([]byte, 0, len(options.CForcedInclude)*32+len(source))
			for i := 0; i < len(options.CForcedInclude); i++ {
				prefix = append(prefix, "#include \""...)
				prefix = append(prefix, options.CForcedInclude[i]...)
				prefix = append(prefix, '"', '\n')
			}
			prefix = append(prefix, source...)
			headerSource = prefix
		}
		header = c11.BuildObjectPrelude(path, headerSource, reader)
		if !header.Ok {
			result.Ok = false
			result.ErrorPath = header.ErrorPath
			result.ErrorAt = header.ErrorAt
			return result
		}
	}
	processed := c11.Preprocess(c11.PreprocessConfig{
		Path: path, Source: source, Reader: reader,
		Predefined: cCommandMacros(options.CDefines), Undefined: options.CUndefines,
		ForcedIncludes: options.CForcedInclude, EmitIncludes: options.CNoStdIncludes, EmitQuotedIncludes: true,
		SuppressForcedIncludes: !options.CNoStdIncludes,
	})
	if !processed.Ok {
		result.Ok = false
		result.Error = processed.Error
		result.ErrorPath = processed.ErrorPath
		return result
	}
	dataModel := c11.DataModelLP64
	if options.Target == "linux/386" {
		dataModel = c11.DataModelILP32
	}
	translated := c11.TranslateAssemblyMetadata(processed.Source, header.Prelude, c11.ObjectConfig{
		DataModel: dataModel, ShortWChar: options.CShortWChar, KernelCodeModel: options.CKernelCodeModel,
	})
	if !translated.Ok {
		result.Ok = false
		result.Error = translated.Error
		result.ErrorAt = translated.ErrorAt
		result.ErrorPath = path
		return result
	}
	options.CDependencies = appendUniquePath(options.CDependencies, path)
	for i := 0; i < len(header.Dependencies); i++ {
		options.CDependencies = appendUniquePath(options.CDependencies, header.Dependencies[i])
	}
	for i := 0; i < len(processed.Dependencies); i++ {
		options.CDependencies = appendUniquePath(options.CDependencies, processed.Dependencies[i])
	}
	options = finalizeCDependencyOptions(options)
	result.Source = translated.Source
	result.DependencyFile = options.DependencyFile
	result.DependencyData = CDependencyOutput(options)
	return result
}

func CAssemblyCommandDiagnostic(result CAssemblyCommandResult) Diagnostic {
	if result.Option != "" {
		return Diagnostic{Phase: "options", Code: "RENVO-OPTION-005", Message: "unknown option " + result.Option}
	}
	if result.ErrorPath != "" && result.Error == 0 {
		return Diagnostic{Phase: "preprocessor", Code: "RENVO-CPP-001", Message: "C include could not be read: " + result.ErrorPath, Path: result.ErrorPath, Line: 1, Column: 1}
	}
	return Diagnostic{Phase: "c11", Code: "RENVO-C11-001", Message: "C assembly metadata source is not supported or is invalid", Path: result.ErrorPath, Line: 1, Column: 1}
}
