//go:build rtg

package build

import (
	"j5.nz/rtg/rtg/internal/arena"
	"j5.nz/rtg/rtg/internal/check"
	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtg/internal/lower"
	"j5.nz/rtg/rtg/internal/unit"
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
	ErrorDetail  int
}

func BuildUnits(graph load.Graph) Result {
	return buildProgramsCore(graph, false)
}

func BuildPrograms(graph load.Graph) Result {
	return buildProgramsCore(graph, false)
}

// BuildProgramsTransient allows the command pipeline to release parsed and
// checked package storage after each lowered unit has taken ownership of the
// data needed by the linker.
func BuildProgramsTransient(graph load.Graph) Result {
	return buildProgramsCore(graph, true)
}

func markCoreBuildArena() int {
	return arena.Mark()
}

func makeCorePackageUnit(program unit.Program, arenaStart int, arenaEnd int) PackageUnit {
	return PackageUnit{
		ImportPath: program.ImportPath,
		Name:       program.Package,
		Program:    program,
		ArenaStart: arenaStart,
		ArenaEnd:   arenaEnd,
	}
}

func discardCoreBuildArena(start int, end int) {
	arena.Discard(start, end)
}

func discardCorePackageSources(pkg load.Package) {
	for i := 0; i < len(pkg.Files); i++ {
		arena.Discard(pkg.Files[i].ArenaStart, pkg.Files[i].ArenaEnd)
	}
	arena.Discard(pkg.CoreArenaStart, pkg.CoreArenaEnd)
}

func retainCoreCheckResult(result Result, prog check.Program) Result {
	return result
}

func retainCoreLowerError(result Result, emit lower.Result) Result {
	return result
}
