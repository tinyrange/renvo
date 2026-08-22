package driver

import (
	"strings"
	"testing"

	"renvo.dev/internal/load"
)

func TestCompileCAssemblyCommandEmitsFreestandingLayoutMetadata(t *testing.T) {
	fs := memorySourceFS{files: []load.SourceFile{
		{Path: "/repo/case/offsets.c", Src: []byte(`#include <layout.h>
int main(void) {
	asm volatile("\n.ascii \"->SIZE_record %0 sizeof(struct record)\"" : : "i"(sizeof(struct record)));
	return 0;
}
`)},
		{Path: "/repo/include/layout.h", Src: []byte("struct record { unsigned long value; char tag; };\n")},
	}}
	args := NormalizeCCompilerCommand([]string{
		"renvo", "cc", "-Wp,-MMD,.offsets.s.d", "-nostdinc", "-I/repo/include",
		"-fshort-wchar", "-mcmodel=kernel", "-S", "offsets.c", "-o", "offsets.s",
	})
	if !CAssemblyCommandRequested(args) {
		t.Fatalf("normalized -S command was not recognized: %#v", args)
	}
	result := CompileCAssemblyCommand(args, "/repo/case", fs)
	if !result.Ok {
		t.Fatalf("assembly command failed: %#v", result)
	}
	if string(result.Source) != ".ascii \"->SIZE_record $16 sizeof(struct record)\"\n" {
		t.Fatalf("assembly output = %q", result.Source)
	}
	if result.Output != "offsets.s" || result.DependencyFile != ".offsets.s.d" ||
		!strings.Contains(string(result.DependencyData), "offsets.c") ||
		!strings.Contains(string(result.DependencyData), "/repo/include/layout.h") {
		t.Fatalf("assembly artifacts = %#v", result)
	}
}

func TestCompileCAssemblyCommandRejectsOrdinaryAssemblyOutput(t *testing.T) {
	fs := memorySourceFS{files: []load.SourceFile{{Path: "/repo/case/main.c", Src: []byte("int main(void) { return 0; }\n")}}}
	args := NormalizeCCompilerCommand([]string{"renvo", "cc", "-S", "main.c", "-o", "main.s"})
	result := CompileCAssemblyCommand(args, "/repo/case", fs)
	if result.Ok {
		t.Fatalf("ordinary -S unexpectedly succeeded: %#v", result)
	}
}
