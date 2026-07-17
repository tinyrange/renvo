package lower

import "j5.nz/rtg/rtg/internal/unit"

const (
	EmitOK = iota
	EmitErrGraph
	EmitErrPackage
	EmitErrToken
	EmitErrUnit
	EmitErrCheck
)

type Result struct {
	Program    unit.Program
	Ok         bool
	Error      int
	UnitError  int
	UnitIndex  int
	UnitDetail int
	UnitA      int
	UnitB      int
	UnitC      int
	ErrorFile  int
	ErrorToken int
}

func emitFail(result Result, err int, file int, tok int) Result {
	result.Ok = false
	result.Error = err
	result.ErrorFile = file
	result.ErrorToken = tok
	return result
}
