package c11

import (
	"bytes"
	"testing"
)

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

func TestPreprocessReportsImplementedDiagnosticErrorAttribute(t *testing.T) {
	result := Preprocess(PreprocessConfig{Path: "probe.c", Source: []byte(`
#if __has_attribute(__error__)
extern void invalid(void) __attribute__((__error__("invalid")));
#else
extern void invalid(void);
#endif
#if __has_attribute(renvo_not_implemented)
INVALID_FEATURE_REPORT
#endif
void use(int value) { if (value) invalid(); }
`)})
	if !result.Ok {
		t.Fatalf("Preprocess failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("__attribute__")) || !bytes.Contains(result.Source, []byte("__error__")) {
		t.Fatalf("implemented error attribute was hidden from the source:\n%s", result.Source)
	}
	if bytes.Contains(result.Source, []byte("INVALID_FEATURE_REPORT")) {
		t.Fatalf("unknown attribute was reported as implemented:\n%s", result.Source)
	}
	translated := TranslateObject("main", result.Source, nil)
	if !translated.Ok {
		t.Fatalf("TranslateObject failed: error=%d at=%d", translated.Error, translated.ErrorAt)
	}
	if !bytes.Contains(translated.Source, []byte("renvo_runtime_CUndefinedInstruction()")) ||
		bytes.Contains(translated.Source, []byte("invalid();")) {
		t.Fatalf("diagnostic error call was not lowered locally:\n%s", translated.Source)
	}
}

func TestPreprocessDefinesTargetInt128Width(t *testing.T) {
	result := PreprocessProbe([]byte("#ifdef __SIZEOF_INT128__\n__SIZEOF_INT128__\n#endif\n"), nil)
	if !result.Ok || string(result.Source) != "16\n" {
		t.Fatalf("int128 target predefine = %#v, source %q", result, result.Source)
	}
}

func TestPreprocessAdvertisesSupportedAsmFlagOutputs(t *testing.T) {
	result := PreprocessProbe([]byte("#ifdef __GCC_ASM_FLAG_OUTPUTS__\nflag_outputs\n#endif\n"), nil)
	if !result.Ok || string(result.Source) != "flag_outputs\n" {
		t.Fatalf("asm flag-output predefine = %#v, source %q", result, result.Source)
	}
}

func TestPreprocessMacroDumpIncludesActiveBuiltinsAndSourceDefinitions(t *testing.T) {
	result := Preprocess(PreprocessConfig{
		Path: "main.c", Source: []byte("#define PROJECT_VALUE 42\n#undef __GNUC_MINOR__\n"),
		EmitIncludes: true, MacroDump: true,
	})
	if !result.Ok || !bytes.Contains(result.Source, []byte("#define __GNUC__ 5\n")) ||
		!bytes.Contains(result.Source, []byte("#define PROJECT_VALUE 42\n")) ||
		bytes.Contains(result.Source, []byte("#define __GNUC_MINOR__")) {
		t.Fatalf("macro dump = %#v\n%s", result, result.Source)
	}
}

func TestPreprocessGNUVariadicStructGroupWithCommaMembers(t *testing.T) {
	source := []byte(`#define __struct_group_tag(TAG) TAG
#define __struct_group(TAG, NAME, ATTRS, MEMBERS...) union { struct { MEMBERS } ATTRS; struct __struct_group_tag(TAG) { MEMBERS } ATTRS NAME; } ATTRS
#define struct_group(NAME, MEMBERS...) __struct_group(, NAME, , MEMBERS)
struct packet {
	struct_group(headers,
		unsigned char first:1, second:1;
#if defined(KEEP_UNION)
		union { unsigned int left, right; };
#endif
	);
};
`)
	result := Preprocess(PreprocessConfig{
		Path: "main.c", Source: source, EmitIncludes: true,
		Predefined: []Macro{{Name: "KEEP_UNION", Value: "1"}},
	})
	if !result.Ok || !bytes.Contains(result.Source, []byte("union { struct")) ||
		!bytes.Contains(result.Source, []byte("unsigned int left , right")) {
		t.Fatalf("struct-group expansion = %#v\n%s", result, result.Source)
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

func TestPreprocessPreservesReplacementWhitespaceThroughNestedStringize(t *testing.T) {
	source := []byte(`#define ASM_REF(symbol) .quad symbol
#define STRINGIFY_INNER(value...) #value
#define STRINGIFY(value...) STRINGIFY_INNER(value)
STRINGIFY(ASM_REF(exported))
`)
	result := Preprocess(PreprocessConfig{Path: "main.c", Source: source, EmitIncludes: true})
	if !result.Ok || string(result.Source) != "\".quad exported\"\n" {
		t.Fatalf("nested stringize expansion = %#v, source %q", result, result.Source)
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

func TestPreprocessEmitsQuotedProjectHeadersOnly(t *testing.T) {
	reader := preprocessTestReader{files: map[string][]byte{
		"project.h": []byte("typedef struct item { int value; } item;\n#include <system.h>\n#define PROJECT_VALUE 42\n"),
		"system.h":  []byte("typedef int unrelated_system_type;\n#define SYSTEM_VALUE 7\n"),
		"forced.h":  []byte("typedef int forced_environment_type;\n#define FORCED_VALUE 1\n"),
	}}
	result := Preprocess(PreprocessConfig{
		Path: "main.c", Source: []byte("#include \"project.h\"\nint value = PROJECT_VALUE + SYSTEM_VALUE + FORCED_VALUE;\n"), Reader: reader,
		ForcedIncludes: []string{"forced.h"}, EmitQuotedIncludes: true,
	})
	if !result.Ok {
		t.Fatalf("Preprocess failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("typedef struct item")) ||
		bytes.Contains(result.Source, []byte("unrelated_system_type")) ||
		bytes.Contains(result.Source, []byte("forced_environment_type")) ||
		!bytes.Contains(result.Source, []byte("int value = 42 + 7 + 1")) {
		t.Fatalf("project/system include boundary was not preserved:\n%s", result.Source)
	}
}

func TestPreprocessCanSuppressForcedDeclarationsWhileEmittingIncludes(t *testing.T) {
	reader := preprocessTestReader{files: map[string][]byte{
		"forced.h": []byte("typedef int forced_type;\n#define FORCED_VALUE 9\n"),
	}}
	result := Preprocess(PreprocessConfig{
		Path: "main.c", Source: []byte("int value = FORCED_VALUE;\n"), Reader: reader,
		ForcedIncludes: []string{"forced.h"}, EmitIncludes: true, SuppressForcedIncludes: true,
	})
	if !result.Ok || bytes.Contains(result.Source, []byte("forced_type")) ||
		!bytes.Contains(result.Source, []byte("int value = 9")) {
		t.Fatalf("forced declaration boundary was not preserved: %#v\n%s", result, result.Source)
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
