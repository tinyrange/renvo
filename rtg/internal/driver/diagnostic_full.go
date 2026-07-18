//go:build !rtg

package driver

import (
	"j5.nz/rtg/rtg/internal/build"
	"j5.nz/rtg/rtg/internal/check"
	"j5.nz/rtg/rtg/internal/link"
	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtg/internal/pipeline"
	"j5.nz/rtg/rtg/internal/syntax"
)

// Diagnostic is the stable source-facing error contract shared by the host
// and self-hosted command paths and by external backend adapters.
type Diagnostic struct {
	Phase   string
	Code    string
	Message string
	Path    string
	Start   int
	End     int
	Line    int
	Column  int
}

func (d Diagnostic) Valid() bool {
	return d.Code != ""
}

// FormatDiagnostic produces the common CLI representation without relying on
// fmt, so the self-hosted frontend uses exactly the same implementation.
func FormatDiagnostic(d Diagnostic) string {
	if !d.Valid() {
		return "rtg: error RTG-UNKNOWN-001 (unknown): compilation failed\n"
	}
	out := ""
	if d.Path != "" {
		out = d.Path
		if d.Line > 0 {
			out = out + ":" + diagnosticIntText(d.Line)
			if d.Column > 0 {
				out = out + ":" + diagnosticIntText(d.Column)
			}
		}
		out = out + ": "
	} else {
		out = "rtg: "
	}
	out = out + "error " + d.Code
	if d.Phase != "" {
		out = out + " (" + d.Phase + ")"
	}
	out = out + ": " + d.Message + "\n"
	return out
}

func diagnosticForBuild(result BuildResult) Diagnostic {
	if result.Error == BuildErrOptions {
		return optionDiagnostic(result.Options)
	}
	if result.Error == BuildErrSource {
		return sourceDiagnostic(result)
	}
	if result.Error == BuildErrPipeline {
		return pipelineDiagnostic(result)
	}
	return Diagnostic{Phase: "frontend", Code: "RTG-FRONTEND-001", Message: "frontend build failed"}
}

func optionDiagnostic(options Options) Diagnostic {
	code := "RTG-OPTION-001"
	message := "invalid command options"
	switch options.Error {
	case ParseErrMissingOutput:
		code, message = "RTG-OPTION-002", "missing output after -o"
	case ParseErrMissingTarget:
		code, message = "RTG-OPTION-003", "missing target after -t"
	case ParseErrUnsupportedTarget:
		code, message = "RTG-OPTION-004", "unsupported target "+options.ErrorArg
	case ParseErrUnknownOption:
		code, message = "RTG-OPTION-005", "unknown option "+options.ErrorArg
	case ParseErrMissingTags:
		code, message = "RTG-OPTION-006", "missing tags after -tags"
	case ParseErrInvalidTags:
		code, message = "RTG-OPTION-007", "invalid build tags "+options.ErrorArg
	case ParseErrMissingPackage:
		code, message = "RTG-OPTION-008", "missing package path"
	case ParseErrExtraPackage:
		code, message = "RTG-OPTION-009", "extra package path "+options.ErrorArg
	case ParseErrWindowsGUIRequiresWindows:
		code, message = "RTG-OPTION-010", "-windows-gui requires a Windows target"
	case ParseErrMixedFileList:
		code, message = "RTG-OPTION-011", "explicit source list contains a non-.go argument "+options.ErrorArg
	case ParseErrMissingArenaSize:
		code, message = "RTG-OPTION-012", "missing arena size after -arena-size"
	case ParseErrInvalidArenaSize:
		code, message = "RTG-OPTION-013", "invalid arena size "+options.ErrorArg
	}
	return Diagnostic{Phase: "options", Code: code, Message: message}
}

func sourceDiagnostic(result BuildResult) Diagnostic {
	code := "RTG-LOAD-001"
	message := "source collection failed"
	phase := "loader"
	switch result.Sources.Error {
	case SourceErrMissingModule:
		code, message = "RTG-LOAD-002", "go.mod was not found"
	case SourceErrModule:
		code, message = "RTG-LOAD-003", "invalid module declaration"
	case SourceErrPackageArg:
		code, message = "RTG-LOAD-004", "package path is outside the main module"
	case SourceErrReadDir:
		code, message = "RTG-LOAD-005", "package directory could not be read"
	case SourceErrReadFile:
		code, message = "RTG-LOAD-006", "source file could not be read"
	case SourceErrBuildConstraint:
		code, message = "RTG-LOAD-007", "invalid build constraint"
	case SourceErrParse:
		phase, code, message = "parser", "RTG-PARSE-001", "source syntax is invalid"
	case SourceErrImport:
		code, message = "RTG-LOAD-008", "unresolved import "+result.Sources.ErrorPath
	case SourceErrDependencyMissing:
		code, message = "RTG-LOAD-014", "dependency source is unavailable for "+result.Sources.ErrorPath
	case SourceErrDependencyExcluded:
		code, message = "RTG-LOAD-015", "dependency version is excluded: "+result.Sources.ErrorPath
	case SourceErrDependencyModule:
		code, message = "RTG-LOAD-016", "dependency has an invalid or missing go.mod: "+result.Sources.ErrorPath
	case SourceErrDependencyAmbiguous:
		code, message = "RTG-LOAD-017", "dependency import is ambiguous: "+result.Sources.ErrorPath
	case SourceErrEmbed:
		code, message = "RTG-LOAD-018", "invalid go:embed directive or pattern: "+result.Sources.ErrorPath
	case SourceErrCgo:
		code, message = "RTG-LOAD-019", "cgo is not supported by RTG"
	case SourceErrStandardPackage:
		code, message = "RTG-LOAD-020", "standard library package "+result.Sources.ErrorPath+" is not included in this RTG build"
	case SourceErrFileDirectory:
		code, message = "RTG-LOAD-021", "named source files must all be in one directory"
	case SourceErrFileListEmpty:
		code, message = "RTG-LOAD-022", "explicit source list contains no buildable Go files"
	}
	path := result.ErrorPath
	if result.Sources.ErrorSourcePath != "" {
		path = result.Sources.ErrorSourcePath
	}
	diagnostic := Diagnostic{Phase: phase, Code: code, Message: message, Path: path}
	if result.Sources.ErrorSourcePath != "" {
		if source, ok := findSource(result.Sources.Files, result.Sources.ErrorSourcePath); ok {
			return diagnosticAtOffset(diagnostic, source, result.Sources.ErrorOffset)
		}
	}
	if result.Sources.Error == SourceErrParse {
		if source, ok := findSource(result.Sources.Files, result.ErrorPath); ok {
			parsed := syntax.ParseFile(source.Src)
			diagnostic = diagnosticAtToken(diagnostic, source, parsed.Tokens, parsed.ErrorTok)
		}
	}
	return diagnostic
}

func diagnosticAtOffset(diagnostic Diagnostic, source load.SourceFile, offset int) Diagnostic {
	if offset < 0 {
		offset = 0
	}
	if offset > len(source.Src) {
		offset = len(source.Src)
	}
	diagnostic.Path = source.Path
	diagnostic.Start = offset
	diagnostic.End = offset
	diagnostic.Line = lineAtOffset(source.Src, offset)
	diagnostic.Column = columnAtOffset(source.Src, offset)
	return diagnostic
}

func pipelineDiagnostic(result BuildResult) Diagnostic {
	built := result.Pipeline
	if built.Error == pipeline.PipelineErrLoad {
		return loadDiagnostic(result, built)
	}
	if built.Error == pipeline.PipelineErrBuild {
		return buildPhaseDiagnostic(result, built)
	}
	if built.Error == pipeline.PipelineErrLink {
		code := "RTG-LINK-001"
		message := "package linking failed"
		if built.Link.Error == link.LinkErrRoot {
			code, message = "RTG-LINK-002", "root package is missing"
		} else if built.Link.Error == link.LinkErrUnit {
			code, message = "RTG-LINK-003", "linked unit is invalid"
		}
		return diagnosticAtPipeline(result, Diagnostic{Phase: "linker", Code: code, Message: message}, built.ErrorPackage, built.ErrorFile, built.ErrorToken)
	}
	return Diagnostic{Phase: "frontend", Code: "RTG-FRONTEND-002", Message: "frontend pipeline failed"}
}

func loadDiagnostic(result BuildResult, built pipeline.Result) Diagnostic {
	workspace := built.Workspace
	diagnostic := Diagnostic{Phase: "loader", Code: "RTG-LOAD-009", Message: "workspace loading failed"}
	if workspace.Error == load.WorkspaceErrDuplicateFile {
		diagnostic.Code, diagnostic.Message = "RTG-LOAD-010", "duplicate source file"
		if workspace.ErrorFile >= 0 && workspace.ErrorFile < len(workspace.Files) {
			diagnostic.Path = workspace.Files[workspace.ErrorFile].Path
		}
		return diagnostic
	}
	if workspace.Error == load.WorkspaceErrMissingModule {
		diagnostic.Code, diagnostic.Message = "RTG-LOAD-002", "go.mod was not found"
		return diagnostic
	}
	if workspace.Error == load.WorkspaceErrModule {
		diagnostic.Code, diagnostic.Message = "RTG-LOAD-003", "invalid module declaration"
		if workspace.ErrorFile >= 0 && workspace.ErrorFile < len(workspace.Files) {
			diagnostic.Path = workspace.Files[workspace.ErrorFile].Path
		}
		return diagnostic
	}
	graph := workspace.Graph
	if graph.Error == load.GraphErrCycle {
		diagnostic.Code, diagnostic.Message = "RTG-LOAD-011", "import cycle detected"
		if source, ok := findSource(result.Sources.Files, graph.ErrorPath); ok {
			return diagnosticAtOffset(diagnostic, source, graph.ErrorOffset)
		}
	}
	if graph.Error == load.GraphErrRoot {
		diagnostic.Code, diagnostic.Message = "RTG-LOAD-004", "root package could not be resolved"
	}
	packageIndex := graph.ErrorPackage
	if packageIndex < 0 && workspace.ErrorFile >= 0 && workspace.ErrorFile < len(graph.Packages) {
		packageIndex = workspace.ErrorFile
	}
	if packageIndex < 0 || packageIndex >= len(graph.Packages) {
		return diagnostic
	}
	pkg := graph.Packages[packageIndex]
	if pkg.Error == load.PackageErrParse {
		diagnostic.Phase, diagnostic.Code, diagnostic.Message = "parser", "RTG-PARSE-001", "source syntax is invalid"
	} else if pkg.Error == load.PackageErrName {
		diagnostic.Code, diagnostic.Message = "RTG-LOAD-012", "files in one directory declare different packages"
	} else if pkg.Error == load.PackageErrImport {
		diagnostic.Code, diagnostic.Message = "RTG-LOAD-008", "import could not be resolved"
	} else if pkg.Error == load.PackageErrNoFiles {
		diagnostic.Code, diagnostic.Message = "RTG-LOAD-013", "package contains no selected Go files"
	}
	if pkg.ErrorFile >= 0 && pkg.ErrorFile < len(pkg.Files) {
		file := pkg.Files[pkg.ErrorFile]
		diagnostic.Path = file.Path
		if pkg.Error == load.PackageErrParse {
			if offset := sourceGenericsOffset(file.Src); offset >= 0 {
				diagnostic.Code, diagnostic.Message = "RTG-PARSE-002", "generics are not supported by RTG"
				return diagnosticAtOffset(diagnostic, load.SourceFile{Path: file.Path, Src: file.Src}, offset)
			}
			diagnostic = diagnosticAtToken(diagnostic, load.SourceFile{Path: file.Path, Src: file.Src}, file.File.Tokens, file.File.ErrorTok)
		}
	} else {
		diagnostic.Path = pkg.Ref.Dir
	}
	return diagnostic
}

func buildPhaseDiagnostic(result BuildResult, built pipeline.Result) Diagnostic {
	phase := "checker"
	code := "RTG-CHECK-001"
	message := "type checking failed"
	if built.Build.Error == build.BuildErrCheck {
		switch built.Build.ErrorDetail {
		case check.CheckErrDuplicate:
			code, message = "RTG-CHECK-002", "duplicate declaration"
		case check.CheckErrImport:
			code, message = "RTG-CHECK-003", "invalid import declaration"
		case check.CheckErrMethod:
			code, message = "RTG-CHECK-004", "invalid method declaration"
		case check.CheckErrBody:
			code, message = "RTG-CHECK-005", "invalid function body"
		case check.CheckErrScope:
			code, message = "RTG-CHECK-006", "invalid name or scope"
		case check.CheckErrReturnCount:
			code, message = "RTG-CHECK-007", "return value count does not match function results"
		case check.CheckErrType:
			code, message = "RTG-CHECK-008", "assignment value is not assignable to its destination"
		case check.CheckErrExcluded:
			code, message = "RTG-CHECK-009", "feature is not supported by RTG"
		case check.CheckErrUnusedImport:
			code, message = "RTG-CHECK-010", "import is not used"
		case check.CheckErrCall:
			code, message = "RTG-CHECK-011", "called expression is not a function"
		case check.CheckErrAssignTarget:
			code, message = "RTG-CHECK-012", "left side of assignment is not assignable"
		case check.CheckErrAssignCount:
			code, message = "RTG-CHECK-013", "assignment count does not match"
		case check.CheckErrBreak:
			code, message = "RTG-CHECK-014", "break is not inside a loop or switch"
		case check.CheckErrContinue:
			code, message = "RTG-CHECK-015", "continue is not inside a loop"
		case check.CheckErrCallArgument:
			code, message = "RTG-CHECK-016", "call argument is not assignable to its parameter"
		case check.CheckErrGoroutine:
			code, message = "RTG-CHECK-017", "goroutines are not supported by RTG"
		case check.CheckErrChannel:
			code, message = "RTG-CHECK-018", "channels are not supported by RTG"
		case check.CheckErrSelect:
			code, message = "RTG-CHECK-019", "select statements are not supported by RTG"
		case check.CheckErrUnusedLocal:
			code, message = "RTG-CHECK-020", "local variable is declared but not used"
		case check.CheckErrMissingMain:
			code, message = "RTG-CHECK-021", "package main has no top-level func main()"
		case check.CheckErrMainSignature:
			code, message = "RTG-CHECK-022", "func main must have no parameters or results"
		case check.CheckErrMainMethod:
			code, message = "RTG-CHECK-023", "method main does not define the package entry point"
		case check.CheckErrSliceOperand:
			code, message = "RTG-CHECK-024", "cannot slice an unaddressable array value"
		}
	} else if built.Build.Error == build.BuildErrLower {
		phase, code, message = "lowerer", "RTG-LOWER-001", "checked program could not be lowered"
	} else if built.Build.Error == build.BuildErrUnit {
		phase, code, message = "unit", "RTG-UNIT-001", "lowered package unit is invalid"
	} else if built.Build.Error == build.BuildErrRoot {
		phase, code, message = "linker", "RTG-LINK-002", "root package is missing"
	}
	return diagnosticAtPipeline(result, Diagnostic{Phase: phase, Code: code, Message: message}, built.ErrorPackage, built.ErrorFile, built.ErrorToken)
}

func diagnosticAtPipeline(result BuildResult, diagnostic Diagnostic, packageIndex int, fileIndex int, tokenIndex int) Diagnostic {
	graph := result.Pipeline.Workspace.Graph
	if packageIndex < 0 || packageIndex >= len(graph.Packages) {
		return diagnostic
	}
	pkg := graph.Packages[packageIndex]
	if fileIndex < 0 || fileIndex >= len(pkg.Files) {
		diagnostic.Path = pkg.Ref.Dir
		return diagnostic
	}
	file := pkg.Files[fileIndex]
	return diagnosticAtToken(diagnostic, load.SourceFile{Path: file.Path, Src: file.Src}, file.File.Tokens, tokenIndex)
}

func diagnosticAtToken(diagnostic Diagnostic, source load.SourceFile, tokens []syntax.Token, tokenIndex int) Diagnostic {
	diagnostic.Path = source.Path
	start := 0
	end := 0
	line := 1
	if tokenIndex >= 0 && tokenIndex < len(tokens) {
		token := tokens[tokenIndex]
		start, end, line = token.Start, token.End, token.Line
	} else if len(source.Src) > 0 {
		start, end = len(source.Src), len(source.Src)
		line = lineAtOffset(source.Src, start)
	}
	diagnostic.Start = start
	diagnostic.End = end
	diagnostic.Line = line
	diagnostic.Column = columnAtOffset(source.Src, start)
	return diagnostic
}

func findSource(files []load.SourceFile, path string) (load.SourceFile, bool) {
	for i := 0; i < len(files); i++ {
		if files[i].Path == path {
			return files[i], true
		}
	}
	return load.SourceFile{}, false
}

func lineAtOffset(src []byte, offset int) int {
	line := 1
	if offset > len(src) {
		offset = len(src)
	}
	for i := 0; i < offset; i++ {
		if src[i] == '\n' {
			line++
		}
	}
	return line
}

func columnAtOffset(src []byte, offset int) int {
	if offset > len(src) {
		offset = len(src)
	}
	column := 1
	for i := offset - 1; i >= 0 && src[i] != '\n'; i-- {
		column++
	}
	return column
}

func diagnosticIntText(value int) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	var digits []byte
	for value > 0 {
		digits = append(digits, byte('0'+value%10))
		value /= 10
	}
	out := ""
	if negative {
		out = "-"
	}
	for i := len(digits) - 1; i >= 0; i-- {
		out = out + string(digits[i:i+1])
	}
	return out
}
