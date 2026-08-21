package main

import "testing"

func TestProgramCachesC11SemanticsAtParseTime(t *testing.T) {
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.objectFile = true
	program := renvoParseProgramWithContext([]byte("package main\n// renvo:c11\nfunc main() {}\n"), context)
	if !program.ok || !renvoProgramUsesC11Semantics(&program) {
		t.Fatal("object program did not retain its C11 marker")
	}
	program.src = []byte("package main\nfunc main() {}\n")
	if !renvoProgramUsesC11Semantics(&program) {
		t.Fatal("C11 semantics were rescanned instead of cached")
	}
}
