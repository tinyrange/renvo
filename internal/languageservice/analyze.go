package languageservice

import (
	"renvo.dev/internal/check"
	"renvo.dev/internal/driver"
	"renvo.dev/internal/load"
)

type Diagnostic = driver.Diagnostic

// AnalysisResult is the frontend-only result used by interactive tools. It
// stops after checking and therefore never lowers, links, or invokes a backend.
type AnalysisResult struct {
	Workspace  load.Workspace
	Program    check.Program
	Diagnostic Diagnostic
	Ok         bool
}

func AnalyzeWorkspace(workDir string, stdRoot string, arg string, files []load.SourceFile) AnalysisResult {
	result := AnalysisResult{Ok: true}
	result.Workspace = load.LoadWorkspace(workDir, stdRoot, arg, files)
	if !result.Workspace.Ok {
		result.Ok = false
		result.Diagnostic = analysisLoadDiagnostic(result.Workspace)
		return result
	}
	result.Program = check.CheckGraph(result.Workspace.Graph)
	if !result.Program.Ok {
		result.Ok = false
		result.Diagnostic = analysisCheckDiagnostic(result.Workspace.Graph, result.Program)
	}
	return result
}

func analysisLoadDiagnostic(workspace load.Workspace) Diagnostic {
	return driver.WorkspaceDiagnostic(workspace)
}

func analysisCheckDiagnostic(graph load.Graph, program check.Program) Diagnostic {
	return driver.CheckerDiagnostic(graph, program)
}
