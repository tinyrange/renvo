package build

import (
	"renvo.dev/internal/arena"
	"renvo.dev/internal/check"
	"renvo.dev/internal/load"
	"renvo.dev/internal/lower"
	"renvo.dev/internal/unit"
)

const (
	BuildOK = iota
	BuildErrCheck
	BuildErrLower
	BuildErrUnit
	BuildErrRoot
)

type PackageUnit struct {
	ImportPath string
	Name       string
	Program    unit.Program
	GraphKeyA  int
	GraphKeyB  int
	SourceKeyA int
	SourceKeyB int
	ArenaStart int
	ArenaEnd   int
}

type Result struct {
	Units        []PackageUnit
	Root         int
	Ok           bool
	Error        int
	ErrorPackage int
	ErrorFile    int
	ErrorToken   int
	ErrorDetail  int
}

func BuildUnits(graph load.Graph) Result {
	return buildProgramsDirect(graph, false)
}

func BuildPrograms(graph load.Graph) Result {
	return buildProgramsCore(graph, false, false, true)
}

// BuildProgramsTransient releases parsed and checked package storage after
// each lowered unit has taken ownership of the data needed by the linker.
func BuildProgramsTransient(graph load.Graph) Result {
	return buildProgramsDirect(graph, true)
}

// BuildProgramsTransientCached reuses lowered dependency packages when their
// source and graph position are unchanged. The root package is always checked
// and lowered so an editor build never conceals changes in the user's code.
func BuildProgramsTransientCached(graph load.Graph) Result {
	return buildProgramsCore(graph, true, true, true)
}

func buildProgramsCore(graph load.Graph, transient bool, cached bool, identities bool) Result {
	if !cached && !identities {
		return buildProgramsDirect(graph, transient)
	}
	session := beginProgramsSession(graph, transient, cached, identities)
	for !session.Step() {
	}
	return session.Result()
}

// buildProgramsDirect is the compact one-shot path. Resumable editor builds
// use ProgramSession, but routing command-line builds through its cache and
// phase state adds work that cannot be reused after the process exits.
func buildProgramsDirect(graph load.Graph, transient bool) Result {
	headerStart := arena.Mark()
	checked := check.CheckGraphHeadersCore(graph)
	headerEnd := arena.Mark()
	result := Result{
		Root:         -1,
		Ok:           true,
		Error:        BuildOK,
		ErrorPackage: -1,
		ErrorFile:    -1,
		ErrorToken:   -1,
	}
	if !checked.Ok {
		result.ErrorDetail = checked.Error
		return buildFail(result, BuildErrCheck, checked.ErrorPackage, checked.ErrorFile, checked.ErrorToken)
	}
	for i := 0; i < len(graph.Packages); i++ {
		persistMark := 0
		if transient {
			persistMark = arena.PersistMark()
		}
		checked = check.CheckGraphPackageCore(graph, checked, i)
		if !checked.Ok {
			if transient {
				arena.PersistReset(persistMark)
			}
			result.ErrorDetail = checked.Error
			return buildFail(result, BuildErrCheck, checked.ErrorPackage, checked.ErrorFile, checked.ErrorToken)
		}
		pkg := graph.Packages[i]
		if pkg.Ref.ImportPath == graph.Root && pkg.Name == "main" {
			if mainErr, mainFile, mainTok := check.CheckRootMain(pkg); mainErr != check.CheckOK {
				if transient {
					arena.PersistReset(persistMark)
				}
				result.ErrorDetail = mainErr
				return buildFail(result, BuildErrCheck, i, mainFile, mainTok)
			}
		}
		unitStart := arena.Mark()
		emit := lower.EmitCheckedPackageCore(pkg, checked.Packages[i], transient)
		unitEnd := arena.Mark()
		if !emit.Ok {
			if transient {
				arena.PersistReset(persistMark)
			}
			result.ErrorDetail = emit.Error
			return buildFail(result, BuildErrLower, i, emit.ErrorFile, emit.ErrorToken)
		}
		if pkg.Ref.ImportPath == graph.Root {
			result.Root = len(result.Units)
		}
		result.Units = append(result.Units, PackageUnit{
			ImportPath: emit.Program.ImportPath,
			Name:       emit.Program.Package,
			Program:    emit.Program,
			ArenaStart: unitStart,
			ArenaEnd:   unitEnd,
		})
		if transient {
			for j := 0; j < len(pkg.Files); j++ {
				arena.Discard(pkg.Files[j].ArenaStart, pkg.Files[j].ArenaEnd)
			}
			arena.Discard(pkg.CoreArenaStart, pkg.CoreArenaEnd)
			arena.PersistReset(persistMark)
		}
	}
	if result.Root < 0 {
		return buildFail(result, BuildErrRoot, -1, -1, -1)
	}
	if transient {
		arena.Discard(headerStart, headerEnd)
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
