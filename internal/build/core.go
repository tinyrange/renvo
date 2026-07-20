package build

import (
	"renvo.dev/internal/load"
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
	return buildProgramsCore(graph, false, false)
}

func BuildPrograms(graph load.Graph) Result {
	return buildProgramsCore(graph, false, false)
}

// BuildProgramsTransient releases parsed and checked package storage after
// each lowered unit has taken ownership of the data needed by the linker.
func BuildProgramsTransient(graph load.Graph) Result {
	return buildProgramsCore(graph, true, false)
}

// BuildProgramsTransientCached reuses lowered dependency packages when their
// source and graph position are unchanged. The root package is always checked
// and lowered so an editor build never conceals changes in the user's code.
func BuildProgramsTransientCached(graph load.Graph) Result {
	return buildProgramsCore(graph, true, true)
}

func buildProgramsCore(graph load.Graph, transient bool, cached bool) Result {
	session := BeginProgramsSession(graph, transient, cached)
	for !session.Step() {
	}
	return session.Result()
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
