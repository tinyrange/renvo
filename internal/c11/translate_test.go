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

func TestTranslateDeclarationAfterControlFlow(t *testing.T) {
	result := Translate("main", []byte(`
int main(void) {
	if (1) { int first = 1; }
	int second = 2;
	return second;
}
`))
	if !result.Ok || !bytes.Contains(result.Source, []byte("}\nvar second int32=2")) {
		t.Fatalf("declaration after control flow translation = %#v\n%s", result, result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated declaration boundary does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateWrapsUnbracedControlledStatements(t *testing.T) {
	result := Translate("main", []byte(`
int main(void) {
	int value = 0;
	if (value == 0) value = 1; else value = 2;
	while (value < 3) value++;
	for (value = 3; value < 4; value++) value += 1;
	return value;
}
`))
	for _, want := range [][]byte{
		[]byte("if value==0 {value=1;} else {value=2;}"),
		[]byte("for value<3 {value++;}"),
		[]byte("for value=3;value<4;value++ {value+=1;}"),
	} {
		if !result.Ok || !bytes.Contains(result.Source, want) {
			t.Fatalf("controlled statement translation is missing %q: %#v\n%s", want, result, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated controlled statements do not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
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

func TestTranslateTypedefStructLayoutToSharedSyntax(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef struct record {
	unsigned char tag;
	int values[3];
	long tail;
} record;

int inspect(void) {
	record value;
	value.tag = 7;
	value.values[2] = 11;
	value.tail = 13;
	if (sizeof(record) != 24 || _Alignof(record) != 8 || __builtin_offsetof(record, values) != 4) {
		return 0;
	}
	return value.tag + value.values[2] + value.tail;
}
`), nil)
	if !result.Ok {
		t.Fatalf("TranslateObject failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("type __c_struct_record struct{tag uint8;values [3]int32;tail int64;}"),
		[]byte("var value __c_struct_record"),
		[]byte("if 24!=24||8!=8||4!=4"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("translated struct source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated struct source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateLinuxSignedIntegerSpellings(t *testing.T) {
	result := Translate("main", []byte(`
typedef __signed__ char signed_byte;
__extension__ typedef __signed__ long long signed_quad;
signed_byte first(signed_byte value) { return value; }
signed_quad second(signed_quad value) { return value; }
`))
	if !result.Ok || !bytes.Contains(result.Source, []byte("func first(value int8) int8")) ||
		!bytes.Contains(result.Source, []byte("func second(value int64) int64")) {
		t.Fatalf("Linux integer spelling translation = %#v\n%s", result, result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated Linux integer spellings do not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
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
		[]byte("func puts(p0 string) int32{return 0}"),
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

func TestTranslateObjectMapsSystemTypedefsAndOpaquePointers(t *testing.T) {
	result := TranslateObject("main", []byte(`
int main(void) {
	EVP_MD_CTX *context = EVP_MD_CTX_new();
	size_t count = 0;
	return 0;
}
`), []byte(`
EVP_MD_CTX *EVP_MD_CTX_new(void);
int EVP_DigestUpdate(EVP_MD_CTX *ctx, const void *data, size_t count);
int EVP_DigestFinal_ex(EVP_MD_CTX *ctx, unsigned char *md, uint32_t *size);
`))
	if !result.Ok {
		t.Fatalf("TranslateObject failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("var context *byte=EVP_MD_CTX_new()"),
		[]byte("var count uintptr=0"),
		[]byte("func EVP_DigestUpdate(p0 *byte,p1 *byte,p2 uintptr) int32"),
		[]byte("func EVP_DigestFinal_ex(p0 *byte,p1 *uint8,p2 *uint32) int32"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("translated object source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated system declarations do not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
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
