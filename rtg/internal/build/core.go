package build

import (
	"j5.nz/rtg/rtg/internal/check"
	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtg/internal/lower"
	"j5.nz/rtg/rtg/internal/semantic"
)

const (
	BuildOK = iota
	BuildErrCheck
	BuildErrLower
	BuildErrUnit
	BuildErrRoot
)

func buildProgramsCore(graph load.Graph, transient bool) Result {
	semantic.LowerInterfaces(&graph)
	headerStart := markCoreBuildArena()
	prog := check.CheckGraphHeadersCore(graph)
	headerEnd := markCoreBuildArena()
	result := Result{
		Root:         -1,
		Ok:           true,
		Error:        BuildOK,
		ErrorPackage: -1,
		ErrorFile:    -1,
		ErrorToken:   -1,
	}
	result = retainCoreCheckResult(result, prog)
	if !prog.Ok {
		result.ErrorDetail = prog.Error
		return buildFail(result, BuildErrCheck, prog.ErrorPackage, prog.ErrorFile, prog.ErrorToken)
	}
	for i := 0; i < len(graph.Packages); i++ {
		prog = check.CheckGraphPackageCore(graph, prog, i)
		result = retainCoreCheckResult(result, prog)
		if !prog.Ok {
			result.ErrorDetail = prog.Error
			return buildFail(result, BuildErrCheck, prog.ErrorPackage, prog.ErrorFile, prog.ErrorToken)
		}
		pkg := graph.Packages[i]
		unitStart := markCoreBuildArena()
		emit := lower.EmitCheckedPackageCore(pkg, prog.Packages[i], transient)
		unitEnd := markCoreBuildArena()
		if !emit.Ok {
			result.ErrorDetail = emit.Error
			result = retainCoreLowerError(result, emit)
			return buildFail(result, BuildErrLower, i, emit.ErrorFile, emit.ErrorToken)
		}
		if pkg.Ref.ImportPath == graph.Root {
			result.Root = len(result.Units)
		}
		result.Units = append(result.Units, makeCorePackageUnit(emit.Program, unitStart, unitEnd))
		if transient {
			discardCorePackageSources(pkg)
		}
	}
	if result.Root < 0 {
		return buildFail(result, BuildErrRoot, -1, -1, -1)
	}
	if transient {
		discardCoreBuildArena(headerStart, headerEnd)
	}
	return result
}

func RootUnit(result Result) PackageUnit {
	if !result.Ok || result.Root < 0 || result.Root >= len(result.Units) {
		return PackageUnit{}
	}
	return result.Units[result.Root]
}

func buildFail(result Result, err int, pkg int, file int, tok int) Result {
	result.Ok = false
	result.Error = err
	result.ErrorPackage = pkg
	result.ErrorFile = file
	result.ErrorToken = tok
	return result
}
