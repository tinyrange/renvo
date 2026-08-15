package c11

import "testing"

type preprocessTestReader struct {
	files map[string][]byte
}

func (r preprocessTestReader) ReadInclude(_ string, name string, _ bool) ([]byte, string, bool) {
	src, ok := r.files[name]
	return src, "/include/" + name, ok
}

func (r preprocessTestReader) ReadIncludeNext(from string, name string, angled bool) ([]byte, string, bool) {
	return r.ReadInclude(from, name, angled)
}

func TestPreprocessProbeSelectsAndExpandsObjectMacros(t *testing.T) {
	source := []byte("#if defined(__clang__)\nClang\n#elif defined(__GNUC__)\nGCC __GNUC__ __GNUC_MINOR__ __GNUC_PATCHLEVEL__\n#else\nunknown\n#endif\n")
	result := PreprocessProbe(source, []Macro{
		{Name: "__GNUC__", Value: "5"},
		{Name: "__GNUC_MINOR__", Value: "1"},
		{Name: "__GNUC_PATCHLEVEL__", Value: "0"},
	})
	if !result.Ok || string(result.Source) != "GCC 5 1 0\n" {
		t.Fatalf("preprocess result = %#v, source %q", result, result.Source)
	}
}

func TestPreprocessProbeRejectsUnsupportedActiveDirective(t *testing.T) {
	result := PreprocessProbe([]byte("#include <stdio.h>\n"), nil)
	if result.Ok || result.Line != 1 {
		t.Fatalf("unsupported directive result = %#v", result)
	}
}

func TestPreprocessExpandsFunctionVariadicStringizeAndPaste(t *testing.T) {
	source := []byte(`#define CAT_(a, b) a ## b
#define CAT(a, b) CAT_(a, b)
#define TEXT(x) #x
#define CALL(name, ...) CAT(prefix_, name)(__VA_ARGS__)
#define VALUE 7
TEXT(VALUE + 1)
CALL(run, VALUE, 9)
`)
	result := Preprocess(PreprocessConfig{Path: "main.c", Source: source, EmitIncludes: true})
	want := "\"VALUE + 1\"\nprefix_run ( 7 , 9 )\n"
	if !result.Ok || string(result.Source) != want {
		t.Fatalf("preprocess = %#v, source %q, want %q", result, result.Source, want)
	}
}

func TestPreprocessExpandsArgumentsOnceAndVAOptParameters(t *testing.T) {
	source := []byte(`#define TWICE(x) x x
#define TEXT(x) #x
#define OPTIONAL(...) __VA_OPT__(__VA_ARGS__ +) 1
#define BOTH(x, ...) x __VA_OPT__(x)
TWICE(__COUNTER__)
TEXT(__COUNTER__)
__COUNTER__
OPTIONAL()
OPTIONAL(4)
BOTH(__COUNTER__, yes)
__COUNTER__
`)
	result := Preprocess(PreprocessConfig{Path: "main.c", Source: source, EmitIncludes: true})
	want := "0 0\n\"__COUNTER__\"\n1\n1\n4 + 1\n2 2\n3\n"
	if !result.Ok || string(result.Source) != want {
		t.Fatalf("argument expansion result = %#v, source %q, want %q", result, result.Source, want)
	}
}

func TestPreprocessRescansAcrossReplacementBoundaryWithHideSets(t *testing.T) {
	source := []byte(`#define INVOKE function
#define function(x) x + 1
#define LEFT RIGHT
#define RIGHT LEFT
INVOKE(4)
LEFT
INVOKE(5) INVOKE(6)
`)
	result := Preprocess(PreprocessConfig{Path: "main.c", Source: source, EmitIncludes: true})
	want := "4 + 1\nLEFT\n5 + 1 6 + 1\n"
	if !result.Ok || string(result.Source) != want {
		t.Fatalf("preprocess = %#v, source %q, want %q", result, result.Source, want)
	}
}

func TestPreprocessIncludesConditionalsGuardsAndOnce(t *testing.T) {
	reader := preprocessTestReader{files: map[string][]byte{
		"guard.h": []byte("#ifndef GUARD_H\n#define GUARD_H\n#define PICK 3\nint guarded;\n#endif\n"),
		"once.h":  []byte("#pragma once\nint once_only;\n"),
	}}
	source := []byte(`#include <guard.h>
#include <guard.h>
#include "once.h"
#include "once.h"
#if defined(PICK) && (PICK * 4 == 12) && !defined(MISSING)
int selected = PICK;
#else
int wrong;
#endif
`)
	result := Preprocess(PreprocessConfig{Path: "main.c", Source: source, Reader: reader, EmitIncludes: true})
	want := "int guarded ;\nint once_only ;\nint selected = 3 ;\n"
	if !result.Ok || string(result.Source) != want || len(result.Dependencies) != 2 {
		t.Fatalf("preprocess = %#v, source %q", result, result.Source)
	}
}

func TestPreprocessSplicesLinesAndRemovesCommentsBeforeExpansion(t *testing.T) {
	source := []byte("#define SUM(a, b) ((a) + \\\n(b))\nSUM(1, /* middle */ 2)\nkept // removed \\\nstill removed\nnext\n")
	result := Preprocess(PreprocessConfig{Path: "main.c", Source: source, EmitIncludes: true})
	if !result.Ok || string(result.Source) != "( ( 1 ) + ( 2 ) )\nkept\nnext\n" {
		t.Fatalf("preprocess = %#v, source %q", result, result.Source)
	}
}

func TestPreprocessSuppressesIncludedTokensButRetainsMacros(t *testing.T) {
	reader := preprocessTestReader{files: map[string][]byte{
		"api.h": []byte("int declaration;\n#define API_VALUE 41\n"),
	}}
	result := Preprocess(PreprocessConfig{
		Path: "main.c", Source: []byte("#include <api.h>\nint value = API_VALUE + 1;\n"),
		Reader: reader, EmitIncludes: false,
	})
	if !result.Ok || string(result.Source) != "int value = 41 + 1 ;\n" {
		t.Fatalf("preprocess = %#v, source %q", result, result.Source)
	}
}

func TestPreprocessDoesNotEvaluateSkippedConditionalExpressions(t *testing.T) {
	source := []byte("#if 0\n#if 1 / 0\nBAD\n#endif\n#endif\n#if 1\nPASS\n#elif 1 / 0\nBAD\n#endif\n#if 0 && 1 / 0\nBAD\n#endif\n#if 1 || 1 / 0\nSHORT\n#endif\n#if 1 ? 1 : 1 / 0\nTERNARY\n#endif\n")
	result := Preprocess(PreprocessConfig{Path: "main.c", Source: source, EmitIncludes: true})
	if !result.Ok || string(result.Source) != "PASS\nSHORT\nTERNARY\n" {
		t.Fatalf("skipped conditional result = %#v, source %q", result, result.Source)
	}
}

func TestPreprocessLineMarkersTrackIncludesAndLineDirectives(t *testing.T) {
	reader := preprocessTestReader{files: map[string][]byte{"value.h": []byte("HEADER __BASE_FILE__ __FILE__\n")}}
	source := []byte("#include <value.h>\n#line 40 \"virtual.c\"\n__LINE__ __FILE__\n")
	result := Preprocess(PreprocessConfig{Path: "main.c", Source: source, Reader: reader, EmitIncludes: true, LineMarkers: true})
	want := "# 1 \"main.c\"\n# 1 \"/include/value.h\"\nHEADER \"main.c\" \"/include/value.h\"\n# 2 \"main.c\"\n# 40 \"virtual.c\"\n40 \"virtual.c\"\n"
	if !result.Ok || string(result.Source) != want {
		t.Fatalf("line marker result = %#v, source %q, want %q", result, result.Source, want)
	}
}
