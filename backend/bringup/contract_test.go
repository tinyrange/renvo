package bringup

import "testing"

func TestStandardStagesAreCumulativeAndStable(t *testing.T) {
	stages := StandardStages("renvo_omnibus")
	if len(stages) != 7 {
		t.Fatalf("stage count = %d", len(stages))
	}
	for i, stage := range stages {
		if int(stage.Milestone) != i+1 || stage.Milestone.String() == "unknown" || stage.Description == "" {
			t.Fatalf("stage %d = %+v", i, stage)
		}
	}
	if stages[0].RequiredExports[0] != "renvo_omnibus_stage0" || stages[4].RequiredExports[0] != "renvo_omnibus_run_all" {
		t.Fatalf("unexpected exported roots: %+v", stages)
	}
}

func TestToolchainRequiresReproducibleContract(t *testing.T) {
	valid := Toolchain{CCompiler: "cc", Linker: "ld", ObjectFormat: "elf32-littleriscv", ABI: "ilp32e", TargetShell: "ch32v003"}
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	valid.ABI = ""
	if err := valid.Validate(); err == nil {
		t.Fatal("toolchain without ABI was accepted")
	}
}
