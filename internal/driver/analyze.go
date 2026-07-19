package driver

import (
	"renvo.dev/internal/check"
	"renvo.dev/internal/load"
)

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
	diagnostic := Diagnostic{Phase: "loader", Code: "RENVO-LOAD-009", Message: "workspace loading failed"}
	if workspace.Error == load.WorkspaceErrMissingModule {
		diagnostic.Code, diagnostic.Message = "RENVO-LOAD-002", "go.mod was not found"
		return diagnostic
	}
	if workspace.Error == load.WorkspaceErrModule {
		diagnostic.Code, diagnostic.Message = "RENVO-LOAD-003", "invalid module declaration"
		if workspace.ErrorFile >= 0 && workspace.ErrorFile < len(workspace.Files) {
			diagnostic.Path = workspace.Files[workspace.ErrorFile].Path
		}
		return diagnostic
	}
	graph := workspace.Graph
	pkgIndex := graph.ErrorPackage
	if pkgIndex < 0 {
		pkgIndex = workspace.ErrorFile
	}
	if pkgIndex < 0 || pkgIndex >= len(graph.Packages) {
		return diagnostic
	}
	pkg := graph.Packages[pkgIndex]
	if pkg.Error == load.PackageErrParse {
		diagnostic.Phase, diagnostic.Code, diagnostic.Message = "parser", "RENVO-PARSE-001", "source syntax is invalid"
	} else if pkg.Error == load.PackageErrName {
		diagnostic.Code, diagnostic.Message = "RENVO-LOAD-012", "files in one directory declare different packages"
	} else if pkg.Error == load.PackageErrImport {
		diagnostic.Code, diagnostic.Message = "RENVO-LOAD-008", "import could not be resolved"
	}
	if pkg.ErrorFile >= 0 && pkg.ErrorFile < len(pkg.Files) {
		file := pkg.Files[pkg.ErrorFile]
		return analysisDiagnosticAtToken(diagnostic, file, file.File.ErrorTok)
	}
	return diagnostic
}

func analysisCheckDiagnostic(graph load.Graph, program check.Program) Diagnostic {
	code, message := "RENVO-CHECK-001", "type checking failed"
	switch program.Error {
	case check.CheckErrDuplicate:
		code, message = "RENVO-CHECK-002", "duplicate declaration"
	case check.CheckErrImport:
		code, message = "RENVO-CHECK-003", "invalid import declaration"
	case check.CheckErrMethod:
		code, message = "RENVO-CHECK-004", "invalid method declaration"
	case check.CheckErrBody:
		code, message = "RENVO-CHECK-005", "invalid function body"
	case check.CheckErrScope:
		code, message = "RENVO-CHECK-006", "invalid name or scope"
	case check.CheckErrReturnCount:
		code, message = "RENVO-CHECK-007", "return value count does not match function results"
	case check.CheckErrType:
		code, message = "RENVO-CHECK-008", "assignment value is not assignable to its destination"
	case check.CheckErrExcluded:
		code, message = "RENVO-CHECK-009", "feature is excluded from the current frontend scope"
	case check.CheckErrUnusedImport:
		code, message = "RENVO-CHECK-010", "import is not used"
	case check.CheckErrCall:
		code, message = "RENVO-CHECK-011", "called expression is not a function"
	case check.CheckErrAssignTarget:
		code, message = "RENVO-CHECK-012", "left side of assignment is not assignable"
	case check.CheckErrAssignCount:
		code, message = "RENVO-CHECK-013", "assignment count does not match"
	case check.CheckErrBreak:
		code, message = "RENVO-CHECK-014", "break is not inside a loop or switch"
	case check.CheckErrContinue:
		code, message = "RENVO-CHECK-015", "continue is not inside a loop"
	case check.CheckErrCallArgument:
		code, message = "RENVO-CHECK-016", "call argument is not assignable to its parameter"
	case check.CheckErrArrayIndex:
		code, message = "RENVO-CHECK-025", "constant array index is out of bounds"
	case check.CheckErrDeferBuiltin:
		code, message = "RENVO-CHECK-026", "deferred builtin call discards a result"
	case check.CheckErrBuiltinArity:
		code, message = "RENVO-CHECK-027", "invalid number of arguments to builtin"
	case check.CheckErrBuiltinOperand:
		code, message = "RENVO-CHECK-028", "invalid operand type for builtin"
	}
	diagnostic := Diagnostic{Phase: "checker", Code: code, Message: message}
	if program.ErrorPackage < 0 || program.ErrorPackage >= len(graph.Packages) {
		return diagnostic
	}
	pkg := graph.Packages[program.ErrorPackage]
	if program.ErrorFile < 0 || program.ErrorFile >= len(pkg.Files) {
		return diagnostic
	}
	return analysisDiagnosticAtToken(diagnostic, pkg.Files[program.ErrorFile], program.ErrorToken)
}

func analysisDiagnosticAtToken(diagnostic Diagnostic, file load.ParsedFile, tokenIndex int) Diagnostic {
	diagnostic.Path = file.Path
	start, end, line := len(file.Src), len(file.Src), 1
	if tokenIndex >= 0 && tokenIndex < len(file.File.Tokens) {
		token := file.File.Tokens[tokenIndex]
		start, end, line = token.Start, token.End, token.Line
	}
	diagnostic.Start, diagnostic.End, diagnostic.Line = start, end, line
	diagnostic.Column = 1
	for i := start - 1; i >= 0 && i < len(file.Src) && file.Src[i] != '\n'; i-- {
		diagnostic.Column++
	}
	return diagnostic
}
