package main

import "testing"

func TestCensusShellWordsAndFlagClassification(t *testing.T) {
	words := shellWords(`cc -DNAME='kernel value' -Wp,-MMD,obj.d -c source.c -o source.o`)
	if len(words) != 7 || words[1] != "-DNAME=kernel value" || words[3] != "-c" || classifyFlag(words[2]) != "-Wp," {
		t.Fatalf("words/classification = %#v/%q", words, classifyFlag(words[2]))
	}
	keys := sortedKeys(map[string]int{"z": 1, "a": 2})
	if len(keys) != 2 || keys[0] != "a" || keys[1] != "z" {
		t.Fatalf("sorted keys = %#v", keys)
	}
}
