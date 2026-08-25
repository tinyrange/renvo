//go:build !renvo

package driver

import "renvo.dev/internal/load"

const (
	CompileOK = iota
	CompileErrBuild
	CompileErrBackend
	CompileErrSystem
)

type Backend interface {
	CompileUnit(unit []byte, target string, strip bool, windowsGUI bool) BackendResult
}

type ArenaBackend interface {
	CompileUnitWithArena(unit []byte, target string, strip bool, windowsGUI bool, arenaSize int) BackendResult
}

type BackendCompileOptions struct {
	Target        string
	Mode          string
	Output        string
	Strip         bool
	WindowsGUI    bool
	EmitImage     bool
	ArenaSize     int
	ModuleLicense string
	ObjectFile    bool
	Code16        bool
	RegParm       int
}

type OptionsBackend interface {
	CompileUnitWithOptions(unit []byte, options BackendCompileOptions) BackendResult
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
	built := BuildFromFSWithBackendModuleCache(args, workDir, stdRoot, moduleCache, fs)
	return CompileBuildResult(built, backend)
}

// CompileBuildResult runs the backend for a completed frontend build. GUI
// clients use it with BeginFSBuildSession so the frontend can advance in
// bounded steps and retain its incremental package caches between builds.
func CompileBuildResult(built BuildResult, backend Backend) CompileResult {
	result := CompileResult{Ok: true, Error: CompileOK}
	result.Build = built
	if !built.Ok {
		result.Diagnostic = built.Diagnostic
		return compileFail(result, CompileErrBuild)
	}
	if built.CacheHit {
		return result
	}
	return compileBuiltUnit(result, built, backend)
}

func compileBuiltUnit(result CompileResult, built BuildResult, backend Backend) CompileResult {
	if built.Options.EmitUnit {
		result.Binary = built.Unit
		return result
	}
	if built.Options.CAssemblyOutput {
		result.Diagnostic = Diagnostic{Phase: "backend", Code: "RENVO-BACKEND-007", Message: "C assembly output is not implemented"}
		return compileFail(result, CompileErrBackend)
	}
	if backend == nil {
		return compileFail(result, CompileErrBackend)
	}
	var backendResult BackendResult
	optionsBackend, acceptsOptions := backend.(OptionsBackend)
	arenaBackend, acceptsArena := backend.(ArenaBackend)
	arenaSize := backendArenaSize(built.Options.Target, built.Options.Tags, built.Options.ArenaSize, built.Options.Mode)
	if acceptsOptions && (built.Options.Mode != ModeExecutable || built.Options.EmitImage) {
		backendResult = optionsBackend.CompileUnitWithOptions(built.Unit, BackendCompileOptions{Target: built.Options.Target, Mode: built.Options.Mode, Output: built.Options.Output, Strip: built.Options.Strip, WindowsGUI: built.Options.WindowsGUI, EmitImage: built.Options.EmitImage, ArenaSize: arenaSize, ModuleLicense: built.Options.ModuleLicense, ObjectFile: built.Options.Mode == ModeObject, Code16: built.Options.CCode16, RegParm: built.Options.CRegParm})
	} else if built.Options.Mode != ModeExecutable {
		backendResult.Diagnostic = Diagnostic{Phase: "backend", Code: "RENVO-BACKEND-006", Message: "backend does not accept output modes"}
	} else if acceptsArena {
		backendResult = arenaBackend.CompileUnitWithArena(built.Unit, backendTargetForOptions(built.Options.Target, built.Options.Mode), built.Options.Strip, built.Options.WindowsGUI, arenaSize)
	} else if built.Options.ArenaSize != 0 {
		backendResult.Diagnostic = Diagnostic{Phase: "backend", Code: "RENVO-BACKEND-005", Message: "backend does not accept an arena policy"}
	} else {
		backendResult = backend.CompileUnit(built.Unit, backendTargetForOptions(built.Options.Target, built.Options.Mode), built.Options.Strip, built.Options.WindowsGUI)
	}
	if !backendResult.Ok || len(backendResult.Binary) == 0 {
		result.Diagnostic = backendResult.Diagnostic
		if !result.Diagnostic.Valid() {
			result.Diagnostic = Diagnostic{Phase: "backend", Code: "RENVO-BACKEND-001", Message: "backend compilation failed"}
		}
		return compileFail(result, CompileErrBackend)
	}
	result.Binary = backendResult.Binary
	if built.Options.Target == "browser/wasm32" && !built.Options.EmitImage {
		result.Binary = PackageBrowserHTML(result.Binary)
	}
	if built.Options.BinaryLimit > 0 && len(result.Binary) > built.Options.BinaryLimit {
		result.Diagnostic = systemBinaryLimitDiagnostic(built.Options.SystemName, len(result.Binary), built.Options.BinaryLimit)
		result.Binary = nil
		return compileFail(result, CompileErrSystem)
	}
	return result
}

func compileFail(result CompileResult, err int) CompileResult {
	result.Ok = false
	result.Error = err
	return result
}
