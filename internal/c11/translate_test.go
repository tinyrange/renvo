package c11

import (
	"bytes"
	"testing"

	"renvo.dev/internal/syntax"
)

func TestTranslateScalarFunctionsToSharedSyntax(t *testing.T) {
	source := []byte(`
#include <stdint.h>
extern int goDouble(int value);

int add(int left, int right) {
	int sum = left + right;
	return sum;
}

int call_go(int value) {
	return goDouble(value);
}
`)
	result := Translate("main", source)
	if !result.Ok {
		t.Fatalf("Translate failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("func add(left int32,right int32) int32")) ||
		!bytes.Contains(result.Source, []byte("var sum int32=left+right")) ||
		!bytes.Contains(result.Source, []byte("return goDouble(value)")) {
		t.Fatalf("unexpected shared source:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateCMainAndControlFlow(t *testing.T) {
	source := []byte(`
int main(void) {
	int value = 40;
	value += 2;
	if (value == 42) {
		print("PASS\n");
	}
	return 0;
}
`)
	result := Translate("main", source)
	if !result.Ok {
		t.Fatalf("Translate failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("func appMain() int32")) {
		t.Fatalf("C entrypoint was not adapted:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateFixedArray(t *testing.T) {
	result := Translate("main", []byte(`int first(void) {
	int values[3] = {42, 1, 2};
	return values[0];
}`))
	if !result.Ok || !bytes.Contains(result.Source, []byte("var values [3]int32=[3]int32{42,1,2}")) {
		t.Fatalf("fixed array translation = %#v\n%s", result, result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated array does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateReportsOriginalOffset(t *testing.T) {
	source := []byte("int main(void) { int value = 1")
	result := Translate("main", source)
	if result.Ok || result.ErrorAt <= 0 || result.ErrorAt > len(source) {
		t.Fatalf("invalid failure result: %#v", result)
	}
}

func TestTranslateObjectEmitsForeignDeclarationAndCStringData(t *testing.T) {
	result := TranslateObject("main", []byte(`
#include <stdio.h>
int main(void) { puts("Hello, world!"); return 0; }
`), []byte("extern int puts (const char *__s);\n"))
	if !result.Ok {
		t.Fatalf("TranslateObject failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("//export main"),
		[]byte("// renvo:linkstatic libc,puts"),
		[]byte("func puts(__s string) int32{return 0}"),
		[]byte(`puts("\x48\x65\x6c\x6c\x6f\x2c\x20\x77\x6f\x72\x6c\x64\x21\x00")`),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("translated object source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated object source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateRejectsSemanticsNotYetPreserved(t *testing.T) {
	for _, source := range []string{
		"#if 0\nint hidden(void) { return 1; }\n#endif\n",
		"int main(int argc, char **argv) { return argc; }\n",
		"int value(void) { switch (1) { case 1: return 1; default: return 0; } }\n",
		"int value(void) { goto done; done: return 1; }\n",
	} {
		result := Translate("main", []byte(source))
		if result.Ok || result.Error != TranslateErrUnsupported {
			t.Fatalf("Translate(%q) = %#v, want explicit unsupported error", source, result)
		}
	}
}

func BenchmarkTranslateScalarFunctions(b *testing.B) {
	source := []byte(`int add(int left, int right) {
	int sum = left + right;
	return sum;
}
int call(int value) { return add(value, 2); }
`)
	b.ReportAllocs()
	b.SetBytes(int64(len(source)))
	for i := 0; i < b.N; i++ {
		result := Translate("main", source)
		if !result.Ok {
			b.Fatal(result.Error)
		}
	}
}
