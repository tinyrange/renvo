package rtg

import (
	"renvo.dev/internal/check"
	"renvo.dev/internal/load"
)

const embeddedGoPrelude = `package main

type RTGRegister struct {
	Code int
	Valid bool
}

type RTGLabel struct {
	Code int
	Valid bool
}

type RTGAddress struct {
	Target RTGLabel
	Addend int
	Base RTGRegister
	Index RTGRegister
	Displacement int
	Scale int
}

type RTGCondition int
type RTGShiftDirection int

const RTGShiftLeft RTGShiftDirection = 1
const RTGShiftRight RTGShiftDirection = 2
var RTGNoRegister RTGRegister

type RTGEmitter struct{}

func (out *RTGEmitter) Byte(value byte) {}
func (out *RTGEmitter) Uint32(value uint32) {}
func (out *RTGEmitter) Uint64(value uint64) {}
func (out *RTGEmitter) Rel32(label RTGLabel) {}
func (out *RTGEmitter) Rel32Addend(label RTGLabel, addend int) {}
func (out *RTGEmitter) Patch() {}
func (out *RTGEmitter) NewLabel() RTGLabel { return RTGLabel{} }
func (out *RTGEmitter) Mark(label RTGLabel) {}

func RTGSignedFits(value int64, bits int) bool { return false }
func RTGUnsignedFits(value uint64, bits int) bool { return false }
func RTGLog2(value int) int { return 0 }
`

func validateEmbeddedGoTypes(document Document) (Diagnostic, bool) {
	var source []byte
	source = append(source, embeddedGoPrelude...)
	for i := 0; i < len(document.Declarations); i++ {
		declaration := document.Declarations[i]
		if declaration.Kind != DeclGo {
			continue
		}
		source = append(source, '\n')
		source = append(source, declaration.GoSource...)
		source = append(source, '\n')
	}
	source = append(source, "\nfunc main() {}\n"...)
	files := []load.SourceFile{
		{Path: "/rtg/go.mod", Src: []byte("module rtg.backend\n")},
		{Path: "/rtg/main.go", Src: source},
	}
	workspace := load.LoadWorkspace("/rtg", "/std", ".", files)
	if !workspace.Ok {
		return Diagnostic{
			Filename: document.Filename,
			Span:     sourceSpan(document.Source, 0, 0),
			Code:     "RTG-GO-009",
			Message:  "embedded Go could not be loaded for type checking",
		}, false
	}
	program := check.CheckGraphCore(workspace.Graph)
	if program.Ok {
		return Diagnostic{}, true
	}
	message := "embedded Go does not type-check against the RTG backend API"
	if program.Error == check.CheckErrUndefined {
		message = "embedded Go refers to an undefined name"
	} else if program.Error == check.CheckErrCallArity || program.Error == check.CheckErrCallArgument {
		message = "embedded Go call does not match its declared signature"
	} else if program.Error == check.CheckErrReturnType || program.Error == check.CheckErrReturnCount {
		message = "embedded Go return does not match its declared signature"
	}
	return Diagnostic{
		Filename: document.Filename,
		Span:     sourceSpan(document.Source, 0, 0),
		Code:     "RTG-GO-009",
		Message:  message,
	}, false
}
