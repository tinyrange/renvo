package main

import "testing"

func TestInputNameSkipsOptionValues(t *testing.T) {
	args := []string{"-t", "example/target", "-arena-size", "8192", "-s", "-o", "output.bin", "input.rtgu"}
	if got := inputName(args); got != "input.rtgu" {
		t.Fatalf("inputName = %q", got)
	}
	if got := inputName([]string{"-t", "example/target"}); got != "" {
		t.Fatalf("option-only inputName = %q", got)
	}
}

func TestCompilerReceivesUnitOnStdin(t *testing.T) {
	args := []string{"-t", "example/target", "-s", "-o", "output.bin", "input.rtgu"}
	got := stdinArguments(args, "input.rtgu")
	if got[len(got)-1] != "-" || args[len(args)-1] != "input.rtgu" {
		t.Fatalf("stdinArguments = %q; original = %q", got, args)
	}
}
