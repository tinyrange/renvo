package c11

import "testing"

func TestLanguageServiceQueriesUseOriginalCSpans(t *testing.T) {
	source := []byte(`
/** One sample in three dimensions. */
struct Vec3 {
	int x;
	int y;
	int z;
};

/** Read the next sample. */
extern struct Vec3 read_accel(int rate);

int main(void) {
	struct Vec3 value = read_accel(60);
	return value.x;
}
`)
	path := "/workspace/main.c"
	analysis := AnalyzeLanguage([]LanguageFile{{Path: path, Source: source}})
	if len(analysis.Diagnostics()) != 0 {
		t.Fatalf("valid C produced diagnostics: %#v", analysis.Diagnostics())
	}

	readUse := languageTestOffset(source, "read_accel(60)") + 2
	hover := analysis.Hover(path, readUse)
	if !hover.Ok || hover.Signature != "struct Vec3 read_accel(int rate)" || hover.Documentation != "Read the next sample." {
		t.Fatalf("unexpected hover: %#v", hover)
	}
	navigation := analysis.Navigate(path, readUse)
	declaration := languageTestOffset(source, "read_accel(int rate)")
	if !navigation.Ok || navigation.Definition.Start != declaration || len(navigation.References) != 2 {
		t.Fatalf("unexpected navigation: %#v", navigation)
	}

	signature := analysis.Signature(path, languageTestOffset(source, "60)")+1)
	if !signature.Ok || signature.Label != "struct Vec3 read_accel(int rate)" || signature.ActiveParameter != 0 || len(signature.Parameters) != 1 || signature.Parameters[0].Type != "int" {
		t.Fatalf("unexpected signature help: %#v", signature)
	}

	memberAt := languageTestOffset(source, "value.x") + len("value.")
	items := analysis.Complete(path, memberAt)
	if !languageTestCompletion(items, "x", LanguageField) || !languageTestCompletion(items, "y", LanguageField) || !languageTestCompletion(items, "z", LanguageField) {
		t.Fatalf("missing aggregate member completions: %#v", items)
	}
	fieldNavigation := analysis.Navigate(path, memberAt)
	fieldDefinition := languageTestOffset(source, "int x") + len("int ")
	if !fieldNavigation.Ok || fieldNavigation.Definition.Start != fieldDefinition {
		t.Fatalf("unexpected field navigation: %#v", fieldNavigation)
	}
}

func TestLanguageServiceResolvesScopesHeadersAndMacros(t *testing.T) {
	header := []byte("/** Maximum sample rate. */\n#define MAX_RATE 60\nint sample(int rate);\n")
	source := []byte(`#include "sensor.h"
int sample(int rate) { return rate; }
int main(void) {
	int rate = MAX_RATE;
	return sample(rate);
}

func TestLanguageServicePrefersStandardLibraryImplementation(t *testing.T) {
	header := []byte("int printf(const char *format, ...);\n")
	implementation := []byte("int printf(const char *format, ...) { return *format != 0; }\n")
	source := []byte("int main(void) { return printf(\"hello\\n\"); }\n")
	analysis := AnalyzeLanguage([]LanguageFile{
		{Path: "main.c", Source: source},
		{Path: "libc/include/stdio.h", Source: header, SuppressDiagnostics: true},
		{Path: "libc/src/stdio.c", Source: implementation, SuppressDiagnostics: true},
	})
	navigation := analysis.Navigate("main.c", languageTestOffset(source, "printf")+1)
	if !navigation.Ok || navigation.Definition.Path != "libc/src/stdio.c" ||
		navigation.Definition.Start != languageTestOffset(implementation, "printf") || navigation.DefinitionIsDeclaration {
		t.Fatalf("printf did not resolve to its bundled implementation: %#v", navigation)
	}
}

func TestLanguageServiceRecognizesIncludeNavigation(t *testing.T) {
	source := []byte("  # include <stdint.h>\n#include \"local/value.h\"\nint value;\n")
	analysis := AnalyzeLanguage([]LanguageFile{{Path: "main.c", Source: source}})
	standard := analysis.IncludeAt("main.c", languageTestOffset(source, "stdint")+2)
	if !standard.Ok || standard.Name != "stdint.h" || !standard.Angled ||
		standard.Start != languageTestOffset(source, "stdint.h") || standard.End != standard.Start+len("stdint.h") {
		t.Fatalf("angle include = %#v", standard)
	}
	quoted := analysis.IncludeAt("main.c", languageTestOffset(source, "local/value.h")+2)
	if !quoted.Ok || quoted.Name != "local/value.h" || quoted.Angled {
		t.Fatalf("quoted include = %#v", quoted)
	}
	if include := analysis.IncludeAt("main.c", languageTestOffset(source, "value;")+1); include.Ok {
		t.Fatalf("ordinary identifier treated as include: %#v", include)
	}
}
`)
	analysis := AnalyzeLanguage([]LanguageFile{
		{Path: "/workspace/main.c", Source: source},
		{Path: "/workspace/sensor.h", Source: header},
	})

	macroUse := languageTestOffset(source, "MAX_RATE") + 1
	macro := analysis.Navigate("/workspace/main.c", macroUse)
	if !macro.Ok || macro.Definition.Path != "/workspace/sensor.h" || macro.Definition.Start != languageTestOffset(header, "MAX_RATE") {
		t.Fatalf("unexpected macro navigation: %#v", macro)
	}

	rateUse := languageTestOffset(source, "sample(rate)") + len("sample(") + 1
	rate := analysis.Navigate("/workspace/main.c", rateUse)
	localDefinition := languageTestOffset(source, "int rate =") + len("int ")
	if !rate.Ok || rate.Definition.Start != localDefinition {
		t.Fatalf("local did not shadow function parameter: %#v", rate)
	}

	sampleUse := languageTestOffset(source, "sample(rate)") + 1
	sample := analysis.Navigate("/workspace/main.c", sampleUse)
	definition := languageTestOffset(source, "sample(int rate)")
	if !sample.Ok || sample.Definition.Path != "/workspace/main.c" || sample.Definition.Start != definition {
		t.Fatalf("definition did not prefer implementation: %#v", sample)
	}
}

func TestLanguageServiceReturnsPartialResultsForIncompleteSource(t *testing.T) {
	source := []byte("struct Point { int x; int y; };\nint main(void) { struct Point point; point.\n")
	analysis := AnalyzeLanguage([]LanguageFile{{Path: "main.c", Source: source}})
	items := analysis.Complete("main.c", languageTestOffset(source, "point.")+len("point."))
	if !languageTestCompletion(items, "x", LanguageField) || !languageTestCompletion(items, "y", LanguageField) {
		t.Fatalf("incomplete buffer lost member completion: %#v", items)
	}
	if len(analysis.Diagnostics()) == 0 || analysis.Diagnostics()[0].Code != "RENVO-C-PARSE-001" {
		t.Fatalf("missing incomplete-buffer diagnostic: %#v", analysis.Diagnostics())
	}
}

func TestLanguageServiceNavigatesTagsAndInfersAutoTypes(t *testing.T) {
	source := []byte(`struct Sample { int value; };
int read_sample(void);
int main(void) {
	struct Sample sample;
	__auto_type count = read_sample();
	__auto_type ratio = 1.5;
	return sample.value + count + (int)ratio;
}
`)
	analysis := AnalyzeLanguage([]LanguageFile{{Path: "main.c", Source: source}})
	tagUse := languageTestOffset(source, "struct Sample sample") + len("struct ") + 1
	tag := analysis.Navigate("main.c", tagUse)
	if !tag.Ok || tag.Definition.Start != languageTestOffset(source, "Sample {") {
		t.Fatalf("unexpected tag navigation: %#v", tag)
	}
	countUse := languageTestOffset(source, "count +") + 1
	count := analysis.Hover("main.c", countUse)
	if !count.Ok || count.Signature != "int count" {
		t.Fatalf("function result was not inferred: %#v", count)
	}
	ratioUse := languageTestOffset(source, "ratio;") + 1
	ratio := analysis.Hover("main.c", ratioUse)
	if !ratio.Ok || ratio.Signature != "double ratio" {
		t.Fatalf("floating literal was not inferred: %#v", ratio)
	}
}

func TestLanguageServiceHandlesEnumsAndChainedMembers(t *testing.T) {
	source := []byte(`enum Mode { MODE_OFF, MODE_ON = 2 };
struct Inner { int value; };
struct Outer { struct Inner inner; };
int main(void) {
	struct Outer outer;
	return MODE_ON + outer.inner.value;
}
`)
	analysis := AnalyzeLanguage([]LanguageFile{{Path: "main.c", Source: source}})
	if len(analysis.Diagnostics()) != 0 {
		t.Fatalf("valid enum/member source produced diagnostics: %#v", analysis.Diagnostics())
	}
	enumUse := languageTestOffset(source, "MODE_ON +") + 1
	enumNavigation := analysis.Navigate("main.c", enumUse)
	if !enumNavigation.Ok || enumNavigation.Definition.Start != languageTestOffset(source, "MODE_ON =") {
		t.Fatalf("unexpected enumerator navigation: %#v", enumNavigation)
	}
	memberAt := languageTestOffset(source, "outer.inner.value") + len("outer.inner.")
	items := analysis.Complete("main.c", memberAt)
	if !languageTestCompletion(items, "value", LanguageField) {
		t.Fatalf("missing chained member completion: %#v", items)
	}
}

func TestLanguageServiceReportsUndefinedIdentifiers(t *testing.T) {
	source := []byte("int main(void) { return missing; }\n")
	analysis := AnalyzeLanguage([]LanguageFile{{Path: "main.c", Source: source}})
	if len(analysis.Diagnostics()) != 1 || analysis.Diagnostics()[0].Code != "RENVO-C-CHECK-002" ||
		analysis.Diagnostics()[0].Start != languageTestOffset(source, "missing") {
		t.Fatalf("undefined identifier diagnostic = %#v", analysis.Diagnostics())
	}
}

func TestLanguageServiceKeepsStaticSymbolsInTheirTranslationUnit(t *testing.T) {
	left := []byte("static int helper(void) { return 1; }\nint left(void) { return helper(); }\n")
	right := []byte("static int helper(void) { return 2; }\nint right(void) { return helper(); }\n")
	analysis := AnalyzeLanguage([]LanguageFile{{Path: "left.c", Source: left}, {Path: "right.c", Source: right}})
	leftUse := languageTestOffset(left, "helper();") + 1
	rightUse := languageTestOffset(right, "helper();") + 1
	leftNavigation := analysis.Navigate("left.c", leftUse)
	rightNavigation := analysis.Navigate("right.c", rightUse)
	if !leftNavigation.Ok || leftNavigation.Definition.Path != "left.c" || len(leftNavigation.References) != 2 {
		t.Fatalf("left static navigation = %#v", leftNavigation)
	}
	if !rightNavigation.Ok || rightNavigation.Definition.Path != "right.c" || len(rightNavigation.References) != 2 {
		t.Fatalf("right static navigation = %#v", rightNavigation)
	}
}

func TestLanguageServiceCompletesAndDescribesFunctionMacros(t *testing.T) {
	source := []byte("/** Select the larger value. */\n#define MAX(a, b) ((a) > (b) ? (a) : (b))\nint main(void) { return MAX(1, 2); }\n")
	analysis := AnalyzeLanguage([]LanguageFile{{Path: "main.c", Source: source}})
	use := languageTestOffset(source, "MAX(1") + 1
	hover := analysis.Hover("main.c", use)
	if !hover.Ok || hover.Documentation != "Select the larger value." || hover.Signature != "#define MAX(a, b) ((a) > (b) ? (a) : (b))" {
		t.Fatalf("macro hover = %#v", hover)
	}
	signature := analysis.Signature("main.c", languageTestOffset(source, "2);")+1)
	if !signature.Ok || signature.ActiveParameter != 1 || len(signature.Parameters) != 2 || signature.Parameters[1].Name != "b" {
		t.Fatalf("macro signature = %#v", signature)
	}
}

func languageTestOffset(source []byte, text string) int {
	for i := 0; i+len(text) <= len(source); i++ {
		if string(source[i:i+len(text)]) == text {
			return i
		}
	}
	return -1
}

func languageTestCompletion(items []LanguageCompletion, name string, kind int) bool {
	for i := 0; i < len(items); i++ {
		if items[i].Name == name && items[i].Kind == kind {
			return true
		}
	}
	return false
}
