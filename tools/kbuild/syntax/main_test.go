package main

import (
	"strings"
	"testing"
)

func TestPreprocessingCommandRewritesOnlyCompileSuffix(t *testing.T) {
	command := "cc -nostdinc -Iinclude -DNAME='quoted value' -c -o build/main.o source/main.c"
	source, rewritten, ok := preprocessingCommand(command, "/tmp/work dir/unit.i")
	if !ok {
		t.Fatal("preprocessing command was rejected")
	}
	if source != "source/main.c" {
		t.Fatalf("source = %q", source)
	}
	if !strings.HasPrefix(rewritten, "cc -nostdinc -Iinclude -DNAME='quoted value'") {
		t.Fatalf("flags changed: %q", rewritten)
	}
	if !strings.HasSuffix(rewritten, " -E -P source/main.c -o '/tmp/work dir/unit.i'") {
		t.Fatalf("compile suffix = %q", rewritten)
	}
}

func TestPreprocessingCommandRejectsUnexpectedSuffix(t *testing.T) {
	if _, _, ok := preprocessingCommand("cc -c source/main.c -o build/main.o", "unit.i"); ok {
		t.Fatal("accepted command whose Kbuild compile suffix is not supported")
	}
}
