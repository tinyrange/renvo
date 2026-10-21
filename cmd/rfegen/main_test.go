package main

import "testing"

func TestGeneratedX86MatchesAuthoredRFE(t *testing.T) {
	t.Chdir("../..")
	if err := generate(true); err != nil {
		t.Fatal(err)
	}
}
