package driver

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckCCommandUsesGNU11SemanticPath(t *testing.T) {
	dir := t.TempDir()
	include := filepath.Join(dir, "include")
	if err := os.Mkdir(include, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(include, "types.h"), []byte(`
#ifdef __SIZEOF_INT128__
typedef unsigned __int128 u128 __attribute__((aligned(16)));
#endif
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.c"), []byte(`
#include <types.h>
register unsigned long stack_pointer asm("rsp");
int inspect(unsigned short *source) {
	const __auto_type saved = source;
	asm volatile("" : "+r"(saved));
	return sizeof(u128) + *saved;
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	args := NormalizeCCompilerCommand([]string{
		"renvo", "cc", "-fsyntax-only", "-nostdinc", "-Iinclude", "-Wp,-MMD,main.d", "main.c",
	})
	if !CSyntaxCommandRequested(args) {
		t.Fatalf("normalized syntax command was not recognized: %#v", args)
	}
	result := CheckCCommand(args, dir, OSFS{})
	if !result.Ok || result.DependencyFile != "main.d" || len(result.DependencyData) == 0 {
		t.Fatalf("syntax command result = %#v", result)
	}
}

func TestCheckCCommandRejectsMalformedSource(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.c"), []byte(`int inspect(void) { asm("broken"; }`), 0o644); err != nil {
		t.Fatal(err)
	}
	args := NormalizeCCompilerCommand([]string{"renvo", "cc", "-fsyntax-only", "main.c"})
	result := CheckCCommand(args, dir, OSFS{})
	if result.Ok || result.ErrorAt < 0 {
		t.Fatalf("malformed syntax command result = %#v", result)
	}
}

func TestCheckCCommandAcceptsPreprocessedCInput(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.i"), []byte(`
typedef unsigned long size_t;
static inline size_t inspect(const char *text) { return sizeof(*text); }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	args := NormalizeCCompilerCommand([]string{"renvo", "cc", "-fsyntax-only", "main.i"})
	result := CheckCCommand(args, dir, OSFS{})
	if !result.Ok {
		t.Fatalf("preprocessed syntax command result = %#v", result)
	}
}
