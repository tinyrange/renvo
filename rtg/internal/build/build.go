package build

import (
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
	Data       []byte
}

type Result struct {
	Check           check.Program
	Units           []PackageUnit
	Root            int
	Ok              bool
	Error           int
	LowerError      int
	LowerUnitError  int
	LowerUnitIndex  int
	LowerUnitDetail int
	LowerUnitA      int
	LowerUnitB      int
	LowerUnitC      int
	ErrorPackage    int
	ErrorFile       int
	ErrorToken      int
}

func BuildUnits(graph load.Graph) Result {
	prog := check.CheckGraph(graph)
	result := Result{
		Check:        prog,
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
		pkg := graph.Packages[i]
		emit := lower.EmitCheckedPackage(pkg, prog.Packages[i])
		if !emit.Ok {
			result.LowerError = emit.Error
			result.LowerUnitError = emit.UnitError
			result.LowerUnitIndex = emit.UnitIndex
			result.LowerUnitDetail = emit.UnitDetail
			result.LowerUnitA = emit.UnitA
			result.LowerUnitB = emit.UnitB
			result.LowerUnitC = emit.UnitC
			return buildFail(result, BuildErrLower, i, emit.ErrorFile, emit.ErrorToken)
		}
		data, ok := unit.Marshal(emit.Program)
		if !ok {
			return buildFail(result, BuildErrUnit, i, -1, -1)
		}
		if pkg.Ref.ImportPath == graph.Root {
			result.Root = len(result.Units)
		}
		result.Units = append(result.Units, PackageUnit{
			ImportPath: pkg.Ref.ImportPath,
			Name:       pkg.Name,
			Program:    emit.Program,
			Data:       data,
		})
	}
	if result.Root < 0 {
		return buildFail(result, BuildErrRoot, -1, -1, -1)
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
