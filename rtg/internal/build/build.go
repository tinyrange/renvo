//go:build rtg

package build

import (
	"j5.nz/rtg/rtg/internal/arena"
	"j5.nz/rtg/rtg/internal/check"
	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtg/internal/lower"
	"j5.nz/rtg/rtg/internal/unit"
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
}

func BuildUnits(graph load.Graph) Result {
	return BuildPrograms(graph)
}

func BuildPrograms(graph load.Graph) Result {
	return buildPrograms(graph, false)
}

// BuildProgramsTransient allows the command pipeline to release parsed and
// checked package storage after each lowered unit has taken ownership of the
// data needed by the linker.
func BuildProgramsTransient(graph load.Graph) Result {
	return buildPrograms(graph, true)
}

func buildPrograms(graph load.Graph, transient bool) Result {
	headerStart := arena.Mark()
	prog := check.CheckGraphHeadersCore(graph)
	headerEnd := arena.Mark()
	result := Result{
		Root:         -1,
		Ok:           true,
		Error:        BuildOK,
		ErrorPackage: -1,
		ErrorFile:    -1,
		ErrorToken:   -1,
	}
	if !prog.Ok {
		return buildFail(result, BuildErrCheck, prog.ErrorPackage, prog.ErrorFile, prog.ErrorToken)
	}
	for i := 0; i < len(graph.Packages); i++ {
		prog = check.CheckGraphPackageCore(graph, prog, i)
		if !prog.Ok {
			return buildFail(result, BuildErrCheck, prog.ErrorPackage, prog.ErrorFile, prog.ErrorToken)
		}
		pkg := graph.Packages[i]
		unitStart := arena.Mark()
		emit := lower.EmitCheckedPackageFast(pkg, prog.Packages[i])
		unitEnd := arena.Mark()
		if !emit.Ok {
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
			arena.Discard(prog.Packages[i].CoreArenaStart, prog.Packages[i].CoreArenaEnd)
			arena.Discard(pkg.CoreArenaStart, pkg.CoreArenaEnd)
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
