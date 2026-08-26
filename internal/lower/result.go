package lower

import "renvo.dev/internal/unit"

const (
	EmitOK = iota
	EmitErrGraph
	EmitErrPackage
	EmitErrToken
	EmitErrUnit
	EmitErrCheck
	EmitErrAssembly
)

type Result struct {
	Program     unit.Program
	Ok          bool
	Error       int
	ErrorFile   int
	ErrorToken  int
	ErrorPath   string
	ErrorOffset int
}

func emitFail(result Result, err int, file int, tok int) Result {
	result.Ok = false
	result.Error = err
	result.ErrorFile = file
	result.ErrorToken = tok
	return result
}
