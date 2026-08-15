package c11

import "testing"

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
