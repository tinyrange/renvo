//go:build !rtg

package driver

import "j5.nz/rtg/rtg/internal/load"

const (
	CompileOK = iota
	CompileErrBuild
	CompileErrBackend
)

type Backend interface {
	CompileUnit(unit []byte, target string, strip bool, windowsGUI bool) BackendResult
}

type BackendResult struct {
	Binary     []byte
	Ok         bool
	Diagnostic Diagnostic
}

type CompileResult struct {
	Build      BuildResult
	Binary     []byte
	Ok         bool
	Error      int
	Diagnostic Diagnostic
}

func CompileUnit(args []string, workDir string, stdRoot string, files []load.SourceFile, backend Backend) CompileResult {
	result := CompileResult{Ok: true, Error: CompileOK}
	built := BuildUnit(args, workDir, stdRoot, files)
	result.Build = built
	if !built.Ok {
		result.Diagnostic = built.Diagnostic
		return compileFail(result, CompileErrBuild)
	}
	return compileBuiltUnit(result, built, backend)
}

func CompileFromFS(args []string, workDir string, stdRoot string, fs SourceFS, backend Backend) CompileResult {
	return CompileFromFSWithModuleCache(args, workDir, stdRoot, "", fs, backend)
}

func CompileFromFSWithModuleCache(args []string, workDir string, stdRoot string, moduleCache string, fs SourceFS, backend Backend) CompileResult {
	result := CompileResult{Ok: true, Error: CompileOK}
	built := BuildFromFSWithModuleCache(args, workDir, stdRoot, moduleCache, fs)
	result.Build = built
	if !built.Ok {
		result.Diagnostic = built.Diagnostic
		return compileFail(result, CompileErrBuild)
	}
	return compileBuiltUnit(result, built, backend)
}

func compileBuiltUnit(result CompileResult, built BuildResult, backend Backend) CompileResult {
	if built.Options.EmitUnit {
		result.Binary = built.Unit
		return result
	}
	if backend == nil {
		return compileFail(result, CompileErrBackend)
	}
	backendResult := backend.CompileUnit(built.Unit, built.Options.Target, built.Options.Strip, built.Options.WindowsGUI)
	if !backendResult.Ok || len(backendResult.Binary) == 0 {
		result.Diagnostic = backendResult.Diagnostic
		if !result.Diagnostic.Valid() {
			result.Diagnostic = Diagnostic{Phase: "backend", Code: "RTG-BACKEND-001", Message: "backend compilation failed"}
		}
		return compileFail(result, CompileErrBackend)
	}
	result.Binary = backendResult.Binary
	return result
}

func compileFail(result CompileResult, err int) CompileResult {
	result.Ok = false
	result.Error = err
	return result
}
