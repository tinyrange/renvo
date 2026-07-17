//go:build rtg

package driver

import (
	"j5.nz/rtg/rtg/internal/build"
	"j5.nz/rtg/rtg/internal/check"
	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtg/internal/pipeline"
)

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

func (d Diagnostic) Valid() bool { return d.Code != "" }

func printRTGDiagnostic(d Diagnostic) {
	if d.Path == "" {
		print("rtg")
	} else {
		print(d.Path)
		if d.Line > 0 {
			print(":")
			rtgPrintInt(d.Line)
			print(":")
			rtgPrintInt(d.Column)
		}
	}
	print(": error ")
	print(d.Code)
	print(" (")
	print(d.Phase)
	print("): ")
	print(d.Message)
	print("\n")
}

func diagnosticForBuild(result BuildResult) Diagnostic {
	d := Diagnostic{Phase: "frontend", Code: "RTG-FRONTEND-001", Message: "frontend build failed"}
	if result.Error == BuildErrOptions {
		d.Phase, d.Code, d.Message = "options", "RTG-OPTION-001", "invalid command options"
		if result.Options.Error == ParseErrMixedFileList {
			d.Code, d.Message = "RTG-OPTION-011", "explicit source list contains a non-.go argument "+result.Options.ErrorArg
		}
		return d
	}
	if result.Error == BuildErrSource {
		d.Phase, d.Code, d.Message, d.Path = "loader", "RTG-LOAD-001", "source collection failed", result.ErrorPath
		if result.Sources.Error == SourceErrParse {
			d.Phase, d.Code, d.Message = "parser", "RTG-PARSE-001", "source syntax is invalid"
		} else if result.Sources.Error == SourceErrImport {
			d.Code, d.Message = "RTG-LOAD-008", "unresolved import "+result.ErrorPath
		} else if result.Sources.Error == SourceErrDependencyMissing {
			d.Code, d.Message = "RTG-LOAD-014", "dependency source is unavailable for "+result.ErrorPath
		} else if result.Sources.Error == SourceErrDependencyExcluded {
			d.Code, d.Message = "RTG-LOAD-015", "dependency version is excluded: "+result.ErrorPath
		} else if result.Sources.Error == SourceErrDependencyModule {
			d.Code, d.Message = "RTG-LOAD-016", "dependency has an invalid or missing go.mod: "+result.ErrorPath
		} else if result.Sources.Error == SourceErrDependencyAmbiguous {
			d.Code, d.Message = "RTG-LOAD-017", "dependency import is ambiguous: "+result.ErrorPath
		} else if result.Sources.Error == SourceErrEmbed {
			d.Code, d.Message = "RTG-LOAD-018", "invalid go:embed directive or pattern: "+result.ErrorPath
		} else if result.Sources.Error == SourceErrCgo {
			d.Code, d.Message = "RTG-LOAD-019", "cgo is not supported by RTG"
		} else if result.Sources.Error == SourceErrStandardPackage {
			d.Code, d.Message = "RTG-LOAD-020", "standard library package "+result.ErrorPath+" is not included in this RTG build"
		} else if result.Sources.Error == SourceErrFileDirectory {
			d.Code, d.Message = "RTG-LOAD-021", "named source files must all be in one directory"
		} else if result.Sources.Error == SourceErrFileListEmpty {
			d.Code, d.Message = "RTG-LOAD-022", "explicit source list contains no buildable Go files"
		}
		if result.Sources.ErrorSourcePath != "" {
			d.Path = result.Sources.ErrorSourcePath
			for i := 0; i < len(result.Sources.Files); i++ {
				if result.Sources.Files[i].Path == d.Path {
					d.Start = result.Sources.ErrorOffset
					d.Line, d.Column = 1, 1
					for j := 0; j < d.Start && j < len(result.Sources.Files[i].Src); j++ {
						if result.Sources.Files[i].Src[j] == '\n' {
							d.Line, d.Column = d.Line+1, 1
						} else {
							d.Column++
						}
					}
				}
			}
		}
		return d
	}
	if result.Error != BuildErrPipeline {
		return d
	}
	built := result.Pipeline
	if built.Error == pipeline.PipelineErrLoad {
		d.Phase, d.Code, d.Message = "loader", "RTG-LOAD-009", "workspace loading failed"
		graph := built.Workspace.Graph
		if graph.Error == load.GraphErrCycle {
			d.Code, d.Message = "RTG-LOAD-011", "import cycle detected"
			d.Path = graph.ErrorPath
			for i := 0; i < len(result.Sources.Files); i++ {
				if result.Sources.Files[i].Path == d.Path {
					d.Start = graph.ErrorOffset
					d.Line, d.Column = 1, 1
					for j := 0; j < d.Start && j < len(result.Sources.Files[i].Src); j++ {
						if result.Sources.Files[i].Src[j] == '\n' {
							d.Line, d.Column = d.Line+1, 1
						} else {
							d.Column++
						}
					}
				}
			}
			return d
		} else {
			pkg := graph.ErrorPackage
			if pkg < 0 {
				pkg = built.Workspace.ErrorFile
			}
			if pkg >= 0 && pkg < len(graph.Packages) {
				packageError := graph.Packages[pkg].Error
				if packageError == load.PackageErrParse {
					d.Phase, d.Code, d.Message = "parser", "RTG-PARSE-001", "source syntax is invalid"
				} else if packageError == load.PackageErrName {
					d.Code, d.Message = "RTG-LOAD-012", "files in one directory declare different packages"
				} else if packageError == load.PackageErrImport {
					d.Code, d.Message = "RTG-LOAD-008", "import could not be resolved"
				} else if packageError == load.PackageErrNoFiles {
					d.Code, d.Message = "RTG-LOAD-013", "package contains no selected Go files"
				}
				result.ErrorPackage = pkg
				result.ErrorFile = graph.Packages[pkg].ErrorFile
				if packageError == load.PackageErrParse && result.ErrorFile >= 0 && result.ErrorFile < len(graph.Packages[pkg].Files) {
					file := graph.Packages[pkg].Files[result.ErrorFile]
					if offset := sourceGenericsOffset(file.Src); offset >= 0 {
						d.Code, d.Message = "RTG-PARSE-002", "generics are not supported by RTG"
						for i := 0; i < len(file.File.Tokens); i++ {
							if file.File.Tokens[i].Start == offset {
								file.File.ErrorTok = i
								break
							}
						}
					}
					result.ErrorToken = file.File.ErrorTok
				}
			}
		}
	} else if built.Error == pipeline.PipelineErrBuild {
		d.Phase, d.Code, d.Message = "checker", "RTG-CHECK-001", "type checking failed"
		if built.Build.Error == build.BuildErrLower {
			d.Phase, d.Code, d.Message = "lowerer", "RTG-LOWER-001", "checked program could not be lowered"
		} else if built.Build.ErrorDetail == check.CheckErrDuplicate {
			d.Code, d.Message = "RTG-CHECK-002", "duplicate declaration"
		} else if built.Build.ErrorDetail == check.CheckErrImport {
			d.Code, d.Message = "RTG-CHECK-003", "invalid import declaration"
		} else if built.Build.ErrorDetail == check.CheckErrMethod {
			d.Code, d.Message = "RTG-CHECK-004", "invalid method declaration"
		} else if built.Build.ErrorDetail == check.CheckErrReturnCount {
			d.Code, d.Message = "RTG-CHECK-007", "return value count does not match function results"
		} else if built.Build.ErrorDetail == check.CheckErrType {
			d.Code, d.Message = "RTG-CHECK-008", "assignment value is not assignable to its destination"
		} else if built.Build.ErrorDetail == check.CheckErrBody {
			d.Code, d.Message = "RTG-CHECK-005", "invalid function body"
		} else if built.Build.ErrorDetail == check.CheckErrScope {
			d.Code, d.Message = "RTG-CHECK-006", "invalid name or scope"
		} else if built.Build.ErrorDetail == check.CheckErrExcluded {
			d.Code, d.Message = "RTG-CHECK-009", "feature is not supported by RTG"
		} else if built.Build.ErrorDetail == check.CheckErrUnusedImport {
			d.Code, d.Message = "RTG-CHECK-010", "import is not used"
		} else if built.Build.ErrorDetail == check.CheckErrCall {
			d.Code, d.Message = "RTG-CHECK-011", "called expression is not a function"
		} else if built.Build.ErrorDetail == check.CheckErrAssignTarget {
			d.Code, d.Message = "RTG-CHECK-012", "left side of assignment is not assignable"
		} else if built.Build.ErrorDetail == check.CheckErrAssignCount {
			d.Code, d.Message = "RTG-CHECK-013", "assignment count does not match"
		} else if built.Build.ErrorDetail == check.CheckErrBreak {
			d.Code, d.Message = "RTG-CHECK-014", "break is not inside a loop or switch"
		} else if built.Build.ErrorDetail == check.CheckErrContinue {
			d.Code, d.Message = "RTG-CHECK-015", "continue is not inside a loop"
		} else if built.Build.ErrorDetail == check.CheckErrCallArgument {
			d.Code, d.Message = "RTG-CHECK-016", "call argument is not assignable to its parameter"
		} else if built.Build.ErrorDetail == check.CheckErrGoroutine {
			d.Code, d.Message = "RTG-CHECK-017", "goroutines are not supported by RTG"
		} else if built.Build.ErrorDetail == check.CheckErrChannel {
			d.Code, d.Message = "RTG-CHECK-018", "channels are not supported by RTG"
		} else if built.Build.ErrorDetail == check.CheckErrSelect {
			d.Code, d.Message = "RTG-CHECK-019", "select statements are not supported by RTG"
		} else if built.Build.ErrorDetail == check.CheckErrUnusedLocal {
			d.Code, d.Message = "RTG-CHECK-020", "local variable is declared but not used"
		} else if built.Build.ErrorDetail == check.CheckErrMissingMain {
			d.Code, d.Message = "RTG-CHECK-021", "package main has no top-level func main()"
		} else if built.Build.ErrorDetail == check.CheckErrMainSignature {
			d.Code, d.Message = "RTG-CHECK-022", "func main must have no parameters or results"
		} else if built.Build.ErrorDetail == check.CheckErrMainMethod {
			d.Code, d.Message = "RTG-CHECK-023", "method main does not define the package entry point"
		} else if built.Build.ErrorDetail == check.CheckErrSliceOperand {
			d.Code, d.Message = "RTG-CHECK-024", "cannot slice an unaddressable array value"
		}
	} else {
		d.Phase, d.Code, d.Message = "linker", "RTG-LINK-001", "package linking failed"
	}
	return rtgBuildDiagnosticLocation(result, d)
}

func rtgBuildDiagnosticLocation(result BuildResult, d Diagnostic) Diagnostic {
	graph := result.Pipeline.Workspace.Graph
	pkg := result.ErrorPackage
	file := result.ErrorFile
	if pkg < 0 || pkg >= len(graph.Packages) {
		return d
	}
	if file < 0 || file >= len(graph.Packages[pkg].Files) {
		d.Path = graph.Packages[pkg].Ref.Dir
		return d
	}
	source := graph.Packages[pkg].Files[file]
	d.Path = source.Path
	tok := result.ErrorToken
	if tok < 0 || tok >= len(source.File.Tokens) {
		return d
	}
	d.Start = source.File.Tokens[tok].Start
	d.End = source.File.Tokens[tok].End
	d.Line = source.File.Tokens[tok].Line
	d.Column = 1
	for i := d.Start - 1; i >= 0 && source.Src[i] != '\n'; i-- {
		d.Column++
	}
	return d
}
