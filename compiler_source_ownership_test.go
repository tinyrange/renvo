package main

import (
	"os"
	"strings"
	"testing"
)

func TestTargetCompositionDoesNotLiveInLinuxRuntimeFile(t *testing.T) {
	linuxSource, err := os.ReadFile("compiler_linux_impl.go")
	if err != nil {
		t.Fatal(err)
	}
	targetSource, err := os.ReadFile("compiler_target_impl.go")
	if err != nil {
		t.Fatal(err)
	}

	targetOwned := []string{
		"func compileTarget(",
		"func RtgCompileSourceToBytes(",
		"func RtgCompileSourceToOutputStrip(",
		"func RtgCompileUnitToOutputStrip(",
		"func rtgCompileParsedProgram(",
	}
	for _, declaration := range targetOwned {
		if strings.Contains(string(linuxSource), declaration) {
			t.Errorf("compiler_linux_impl.go still owns %s", declaration)
		}
		if !strings.Contains(string(targetSource), declaration) {
			t.Errorf("compiler_target_impl.go does not own %s", declaration)
		}
	}
	if strings.Contains(string(targetSource), "func rtgLinuxSys") {
		t.Error("compiler_target_impl.go contains Linux runtime lowering")
	}
}
