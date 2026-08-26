package driver

import (
	"strings"
	"testing"

	frontendbuild "renvo.dev/internal/build"
	"renvo.dev/internal/c11"
	"renvo.dev/internal/check"
	"renvo.dev/internal/link"
	"renvo.dev/internal/load"
	"renvo.dev/internal/lower"
	"renvo.dev/internal/pipeline"
	"renvo.dev/internal/syntax"
)

func TestEveryOptionErrorHasSpecificDiagnostic(t *testing.T) {
	for detail := ParseErrMissingOutput; detail <= ParseErrMissingIncludePath; detail++ {
		result := BuildResult{Error: BuildErrOptions, Options: Options{Error: detail, ErrorArg: "detail"}}
		diagnostic := diagnosticForBuild(result)
		if !diagnostic.Valid() || diagnostic.Code == "RENVO-OPTION-001" || strings.HasPrefix(diagnostic.Code, "RENVO-BUG-") || diagnostic.Message == "invalid command options" {
			t.Errorf("option error %d has generic diagnostic %#v", detail, diagnostic)
		}
	}
}

func TestEveryParserErrorHasSpecificDiagnostic(t *testing.T) {
	for detail := syntax.ParseErrScan; detail <= syntax.ParseErrTopLevel; detail++ {
		diagnostic := syntaxErrorDiagnostic(Diagnostic{}, detail)
		if !diagnostic.Valid() || strings.HasPrefix(diagnostic.Code, "RENVO-BUG-") || strings.Contains(diagnostic.Message, "syntax is invalid") {
			t.Errorf("parser error %d has generic diagnostic %#v", detail, diagnostic)
		}
	}
}

func TestEveryCTranslatorErrorHasSpecificDiagnostic(t *testing.T) {
	for detail := c11.TranslateErrScan; detail <= c11.TranslateErrVLA; detail++ {
		diagnostic := c11ErrorDiagnostic(Diagnostic{}, detail)
		if !diagnostic.Valid() || strings.HasPrefix(diagnostic.Code, "RENVO-BUG-") || strings.Contains(diagnostic.Message, "is invalid") {
			t.Errorf("C translator error %d has generic diagnostic %#v", detail, diagnostic)
		}
	}
}

func TestEveryCPreprocessorErrorHasSpecificDiagnostic(t *testing.T) {
	for detail := c11.PreprocessErrToken; detail <= c11.PreprocessErrDepth; detail++ {
		diagnostic := cPreprocessDiagnostic(detail, "source.c", 3, "detail")
		if !diagnostic.Valid() || strings.HasPrefix(diagnostic.Code, "RENVO-BUG-") || diagnostic.Line != 3 {
			t.Errorf("C preprocessor error %d has generic diagnostic %#v", detail, diagnostic)
		}
	}
}

func TestEverySourceErrorHasSpecificDiagnostic(t *testing.T) {
	for detail := SourceErrMissingModule; detail <= SourceErrNoSelectedFiles; detail++ {
		source := SourceResult{Error: detail, ErrorPath: "detail"}
		if detail == SourceErrParse {
			source.ErrorSourcePath = "detail"
			source.Files = []load.SourceFile{{Path: "detail", Src: []byte("package\n")}}
		}
		if detail == SourceErrCPreprocess {
			source.CPreprocessError = c11.PreprocessErrToken
		}
		result := BuildResult{Error: BuildErrSource, ErrorPath: "detail", Sources: source}
		diagnostic := diagnosticForBuild(result)
		if !diagnostic.Valid() || diagnostic.Code == "RENVO-LOAD-001" || strings.HasPrefix(diagnostic.Code, "RENVO-BUG-") || diagnostic.Message == "source collection failed" {
			t.Errorf("source error %d has generic diagnostic %#v", detail, diagnostic)
		}
	}
}

func TestEveryWorkspaceAndGraphErrorHasSpecificDiagnostic(t *testing.T) {
	for detail := load.WorkspaceErrDuplicateFile; detail <= load.WorkspaceErrGraph; detail++ {
		workspace := load.Workspace{Error: detail, ErrorFile: -1}
		if detail == load.WorkspaceErrGraph {
			workspace.Graph = load.Graph{Error: load.GraphErrRoot, ErrorPackage: -1}
		}
		diagnostic := diagnosticForBuild(BuildResult{Error: BuildErrPipeline, Pipeline: pipeline.Result{Error: pipeline.PipelineErrLoad, Workspace: workspace}})
		if !diagnostic.Valid() || diagnostic.Code == "RENVO-LOAD-009" || strings.HasPrefix(diagnostic.Code, "RENVO-BUG-") {
			t.Errorf("workspace error %d has generic diagnostic %#v", detail, diagnostic)
		}
	}
	for detail := load.GraphErrRoot; detail <= load.GraphErrCycle; detail++ {
		graph := load.Graph{Error: detail, ErrorPackage: -1}
		if detail == load.GraphErrPackage {
			graph.ErrorPackage = 0
			graph.Packages = []load.Package{{Error: load.PackageErrRef, ErrorFile: -1}}
		}
		workspace := load.Workspace{Error: load.WorkspaceErrGraph, ErrorFile: -1, Graph: graph}
		diagnostic := diagnosticForBuild(BuildResult{Error: BuildErrPipeline, Pipeline: pipeline.Result{Error: pipeline.PipelineErrLoad, Workspace: workspace}})
		if !diagnostic.Valid() || diagnostic.Code == "RENVO-LOAD-009" || strings.HasPrefix(diagnostic.Code, "RENVO-BUG-") {
			t.Errorf("graph error %d has generic diagnostic %#v", detail, diagnostic)
		}
	}
}

func TestEveryPackageLoaderErrorHasSpecificDiagnostic(t *testing.T) {
	badSource := []byte("package\n")
	parsed := load.ParsedFile{Path: "/repo/main.go", Src: badSource, File: syntax.ParseFile(badSource)}
	for detail := load.PackageErrRef; detail <= load.PackageErrC11; detail++ {
		pkg := load.Package{Error: detail, ErrorFile: -1}
		if detail >= load.PackageErrParse {
			pkg.Files = []load.ParsedFile{parsed}
			pkg.ErrorFile = 0
		}
		if detail == load.PackageErrC11 {
			pkg.C11Error = c11.TranslateErrScan
		}
		graph := load.Graph{Error: load.GraphErrPackage, ErrorPackage: 0, Packages: []load.Package{pkg}}
		workspace := load.Workspace{Error: load.WorkspaceErrGraph, ErrorFile: -1, Graph: graph}
		diagnostic := diagnosticForBuild(BuildResult{Error: BuildErrPipeline, Pipeline: pipeline.Result{Error: pipeline.PipelineErrLoad, Workspace: workspace}})
		if !diagnostic.Valid() || diagnostic.Code == "RENVO-LOAD-009" || strings.HasPrefix(diagnostic.Code, "RENVO-BUG-") {
			t.Errorf("package error %d has generic diagnostic %#v", detail, diagnostic)
		}
	}
}

func TestEveryLinkerErrorHasSpecificDiagnostic(t *testing.T) {
	for detail := link.LinkErrBuild; detail <= link.LinkErrUnit; detail++ {
		result := BuildResult{Error: BuildErrPipeline, Pipeline: pipeline.Result{
			Error: pipeline.PipelineErrLink, Link: link.Result{Error: detail},
		}, ErrorPackage: -1, ErrorFile: -1, ErrorToken: -1}
		diagnostic := diagnosticForBuild(result)
		if !diagnostic.Valid() || strings.HasPrefix(diagnostic.Code, "RENVO-BUG-") {
			t.Errorf("linker error %d has generic diagnostic %#v", detail, diagnostic)
		}
	}
}

func TestEveryCheckerErrorHasSpecificDiagnostic(t *testing.T) {
	for detail := check.CheckErrGraph; detail <= check.CheckErrTypeAssertion; detail++ {
		result := BuildResult{
			Error: BuildErrPipeline,
			Pipeline: pipeline.Result{
				Error: pipeline.PipelineErrBuild,
				Build: frontendbuild.Result{Error: frontendbuild.BuildErrCheck, ErrorDetail: detail},
			},
			ErrorPackage: -1, ErrorFile: -1, ErrorToken: -1,
		}
		diagnostic := diagnosticForBuild(result)
		if !diagnostic.Valid() || strings.HasPrefix(diagnostic.Code, "RENVO-BUG-") || diagnostic.Message == "type checking failed" {
			t.Errorf("checker error %d has generic diagnostic %#v", detail, diagnostic)
		}
	}
}

func TestEveryLowererErrorHasSpecificDiagnostic(t *testing.T) {
	for detail := lower.EmitErrGraph; detail <= lower.EmitErrAssembly; detail++ {
		result := BuildResult{
			Error: BuildErrPipeline,
			Pipeline: pipeline.Result{
				Error: pipeline.PipelineErrBuild,
				Build: frontendbuild.Result{Error: frontendbuild.BuildErrLower, ErrorDetail: detail},
			},
			ErrorPackage: -1, ErrorFile: -1, ErrorToken: -1,
		}
		diagnostic := diagnosticForBuild(result)
		if !diagnostic.Valid() || strings.HasPrefix(diagnostic.Code, "RENVO-BUG-") || diagnostic.Message == "checked program could not be lowered" {
			t.Errorf("lowerer error %d has generic diagnostic %#v", detail, diagnostic)
		}
	}
}

func TestUndeclaredDiagnosticStatesAreReportedAsCompilerBugs(t *testing.T) {
	buildDiagnostic := diagnosticForBuild(BuildResult{Error: 99})
	if buildDiagnostic.Code != "RENVO-BUG-001" || !strings.Contains(buildDiagnostic.Message, "undeclared error code 99") {
		t.Fatalf("undeclared build diagnostic = %#v", buildDiagnostic)
	}
	pipelineDiagnostic := diagnosticForBuild(BuildResult{
		Error: BuildErrPipeline, Pipeline: pipeline.Result{Error: 99},
	})
	if pipelineDiagnostic.Code != "RENVO-BUG-002" || !strings.Contains(pipelineDiagnostic.Message, "undeclared error code 99") {
		t.Fatalf("undeclared pipeline diagnostic = %#v", pipelineDiagnostic)
	}
	formatted := FormatDiagnostic(Diagnostic{})
	if !strings.Contains(formatted, "RENVO-BUG-003") || !strings.Contains(formatted, "compiler bug") || strings.Contains(formatted, "compilation failed") {
		t.Fatalf("missing diagnostic fallback = %q", formatted)
	}
}
