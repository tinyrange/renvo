// Package bringup defines host-side contracts for incrementally replacing a
// trusted C object with objects emitted by a new Renvo backend.
package bringup

import "fmt"

type Milestone int

const (
	MilestoneConstantFunction Milestone = iota + 1
	MilestoneArithmetic
	MilestoneCalls
	MilestoneStaticData
	MilestoneCompleteOmnibus
	MilestoneProgramObjects
	MilestoneStandaloneImage
)

func (m Milestone) String() string {
	switch m {
	case MilestoneConstantFunction:
		return "constant-function"
	case MilestoneArithmetic:
		return "arithmetic-control-memory"
	case MilestoneCalls:
		return "calls-frames-spills"
	case MilestoneStaticData:
		return "static-data-relocations"
	case MilestoneCompleteOmnibus:
		return "complete-omnibus-object"
	case MilestoneProgramObjects:
		return "ordinary-program-objects"
	case MilestoneStandaloneImage:
		return "standalone-image"
	default:
		return "unknown"
	}
}

type Toolchain struct {
	CCompiler    string
	Assembler    string
	Linker       string
	ObjectFormat string
	ABI          string
	TargetShell  string
	Runner       []string
}

func (t Toolchain) Validate() error {
	if t.CCompiler == "" || t.Linker == "" || t.ObjectFormat == "" || t.ABI == "" || t.TargetShell == "" {
		return fmt.Errorf("C compiler, linker, object format, ABI, and target shell are required")
	}
	return nil
}

type Stage struct {
	Milestone       Milestone
	RequiredExports []string
	Description     string
}

func StandardStages(entry string) []Stage {
	return []Stage{
		{Milestone: MilestoneConstantFunction, RequiredExports: []string{entry + "_stage0"}, Description: "one C-ABI function returning a constant"},
		{Milestone: MilestoneArithmetic, RequiredExports: []string{entry + "_stage1"}, Description: "arithmetic, control flow, and memory access"},
		{Milestone: MilestoneCalls, RequiredExports: []string{entry + "_stage2"}, Description: "calls, frames, spills, and callee-saved state"},
		{Milestone: MilestoneStaticData, RequiredExports: []string{entry + "_stage3"}, Description: "data, rodata, BSS, symbols, and relocations"},
		{Milestone: MilestoneCompleteOmnibus, RequiredExports: []string{entry + "_run_all"}, Description: "complete omnibus object with trusted startup only"},
		{Milestone: MilestoneProgramObjects, RequiredExports: nil, Description: "all ordinary program objects"},
		{Milestone: MilestoneStandaloneImage, RequiredExports: nil, Description: "backend-owned startup and final image"},
	}
}
