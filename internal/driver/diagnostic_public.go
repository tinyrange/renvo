package driver

import (
	frontendbuild "renvo.dev/internal/build"
	"renvo.dev/internal/check"
	"renvo.dev/internal/load"
	"renvo.dev/internal/pipeline"
)

// WorkspaceDiagnostic converts a declared workspace-loader failure into the
// same source-facing diagnostic used by compilation commands.
func WorkspaceDiagnostic(workspace load.Workspace) Diagnostic {
	return diagnosticForBuild(BuildResult{
		Error:        BuildErrPipeline,
		Sources:      SourceResult{Files: workspace.Files},
		Pipeline:     pipeline.Result{Error: pipeline.PipelineErrLoad, Workspace: workspace},
		ErrorPackage: workspace.Graph.ErrorPackage,
		ErrorFile:    workspace.ErrorFile,
		ErrorToken:   -1,
	})
}

// SourceDiagnostic converts a source-collection failure into the shared
// source-facing diagnostic contract.
func SourceDiagnostic(source SourceResult) Diagnostic {
	return diagnosticForBuild(BuildResult{
		Error:        BuildErrSource,
		ErrorPath:    source.ErrorPath,
		ErrorAt:      source.ErrorOffset,
		Sources:      source,
		ErrorPackage: -1,
		ErrorFile:    -1,
		ErrorToken:   -1,
	})
}

// CTranslatorDiagnostic identifies one declared C translator failure.
func CTranslatorDiagnostic(detail int, path string, start int, line int, column int) Diagnostic {
	diagnostic := c11ErrorDiagnostic(Diagnostic{}, detail)
	diagnostic.Path, diagnostic.Start, diagnostic.End = path, start, start+1
	diagnostic.Line, diagnostic.Column = line, column
	return diagnostic
}

// CPreprocessorDiagnostic identifies one declared C preprocessor failure.
func CPreprocessorDiagnostic(detail int, path string, line int, messageDetail string) Diagnostic {
	return cPreprocessDiagnostic(detail, path, line, messageDetail)
}

// CheckerDiagnostic converts a declared checker failure into the shared
// source-facing diagnostic contract.
func CheckerDiagnostic(graph load.Graph, program check.Program) Diagnostic {
	return diagnosticForBuild(BuildResult{
		Error: BuildErrPipeline,
		Pipeline: pipeline.Result{
			Error:        pipeline.PipelineErrBuild,
			ErrorPackage: program.ErrorPackage,
			ErrorFile:    program.ErrorFile,
			ErrorToken:   program.ErrorToken,
			Workspace:    load.Workspace{Graph: graph},
			Build: frontendbuild.Result{
				Error:        frontendbuild.BuildErrCheck,
				ErrorDetail:  program.Error,
				ErrorPackage: program.ErrorPackage,
				ErrorFile:    program.ErrorFile,
				ErrorToken:   program.ErrorToken,
			},
		},
		ErrorPackage: program.ErrorPackage,
		ErrorFile:    program.ErrorFile,
		ErrorToken:   program.ErrorToken,
	})
}
