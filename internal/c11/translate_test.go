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

func TestTranslateParenthesizedAndFunctionDeclarators(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef int (*binary_op)(int, int);
struct callbacks {
	binary_op apply;
	int (*direct)(int);
	int *values[3];
	int (*matrix)[3];
};
int *select_value(int *value);
int consume(int (*matrix)[3], int values[3]);
`), nil)
	if !result.Ok {
		t.Fatalf("declarator translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("apply func(int32,int32) int32"),
		[]byte("direct func(int32) int32"),
		[]byte("values [3]*int32"),
		[]byte("matrix *[3]int32"),
		[]byte("func select_value(p0 *int32) *int32"),
		[]byte("func consume(p0 *[3]int32,p1 *int32) int32"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("translated declarators are missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated declarators do not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateEnumConstantsAndConstantArrayBounds(t *testing.T) {
	result := Translate("main", []byte(`
enum mode {
	mode_zero,
	mode_four = 1 << 2,
	mode_five,
	mode_mask = mode_four | 2
};
int values[1024 / (8 * sizeof(long))];
int selected(void) { return mode_five + mode_mask; }
`))
	if !result.Ok {
		t.Fatalf("enum translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("var values [16]int32"),
		[]byte("return 5+6"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("translated enum source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated enum source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateAnonymousAggregateMemberLayout(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct envelope {
	char prefix;
	union {
		struct { int low; short high; };
		long both;
	};
	char tail;
};
int layout(void) {
	return sizeof(struct envelope) +
		__builtin_offsetof(struct envelope, low) +
		__builtin_offsetof(struct envelope, high) +
		__builtin_offsetof(struct envelope, both) +
		__builtin_offsetof(struct envelope, tail);
}
`), nil)
	if !result.Ok {
		t.Fatalf("anonymous aggregate translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("return 24+8+12+8+16")) {
		t.Fatalf("anonymous aggregate offsets were not preserved:\n%s", result.Source)
	}
	if !bytes.Contains(result.Source, []byte("struct{__c_align uint64}")) {
		t.Fatalf("anonymous union carrier does not preserve alignment:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated anonymous aggregate does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateUnionOverlapAndBitfieldAccess(t *testing.T) {
	result := TranslateObject("main", []byte(`
union word {
	unsigned whole;
	unsigned char bytes[4];
};
struct flags {
	unsigned low : 3;
	unsigned high : 5;
	signed delta : 6;
	unsigned : 0;
	_Bool ready : 1;
	unsigned tail : 7;
};
int inspect(void) {
	union word word;
	union word *word_cursor = &word;
	struct flags flags;
	struct flags *flag_cursor = &flags;
	word_cursor->whole = 0x04030201;
	flags.low = 5;
	flags.high = 17;
	flags.delta = -3;
	flags.ready = 1;
	flag_cursor->tail = 65;
	flags.low += 1;
	return sizeof(union word) + sizeof(struct flags) + word.bytes[0] +
		flags.low + flags.high + flags.delta + flags.ready + flags.tail;
}
`), nil)
	if !result.Ok {
		t.Fatalf("union/bitfield translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("import __c_unsafe \"unsafe\""),
		[]byte("func(p *__c_union_word)__c_ptr_whole()*uint32"),
		[]byte("func(p *__c_struct_flags)__c_get_delta() int32"),
		[]byte("flags.__c_set_low(flags.__c_get_low()+1)"),
		[]byte("(*word_cursor.__c_ptr_whole())=0x04030201"),
		[]byte("return 4+12+(*word.__c_ptr_bytes())[0]+flags.__c_get_low()"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("translated union/bitfield source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated union/bitfield source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateCompleteISOControlFlow(t *testing.T) {
	result := TranslateObject("main", []byte(`
int control(int choice) {
	int i = 0;
	int total = 0;
	do {
		i++;
		if (i < 3) continue;
		total += i;
	} while (i < 4);
	for (int j = 0; j < 3; j++) total += j;
	switch (choice) {
	case 0:
		total += 1;
	case 1:
		total += 2;
		break;
	default:
		total += 9;
	}
	goto done;
	total = 1000;
done:
	return total;
}
`), nil)
	if !result.Ok {
		t.Fatalf("control-flow translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("bool=true;for __c_do_first_"),
		[]byte("var j int32=0;"),
		[]byte("for ;j<3;j++"),
		[]byte("case 0:total+=1;"),
		[]byte("fallthrough;case 1:"),
		[]byte("goto done;"),
		[]byte("done:"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("translated control flow is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated control flow does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateCoalescesTentativeDefinitionsAndPrototypes(t *testing.T) {
	result := TranslateObject("main", []byte(`
int accumulate(int value);
int shared;
extern int shared;
int shared;
static int hidden;
int fallback[];
int bounded[];
int bounded[3];
int selected;
int selected = 7;
int selected;
int accumulate(int value) {
	extern int shared;
	static int calls;
	calls++;
	hidden++;
	shared += value;
	return shared + hidden + calls + selected;
}
int accumulate(int value);
`), nil)
	if !result.Ok {
		t.Fatalf("linkage translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, declaration := range [][]byte{
		[]byte("var shared int32;"),
		[]byte("var hidden int32;"),
		[]byte("var fallback [1]int32;"),
		[]byte("var bounded [3]int32;"),
		[]byte("var selected int32=7;"),
		[]byte("var __c_static_1_calls int32;"),
		[]byte("func accumulate(value int32) int32"),
	} {
		if bytes.Count(result.Source, declaration) != 1 {
			t.Fatalf("declaration %q was not coalesced:\n%s", declaration, result.Source)
		}
	}
	if bytes.Contains(result.Source, []byte("linkstatic libc,accumulate")) {
		t.Fatalf("defined prototype became a foreign stub:\n%s", result.Source)
	}
	if !bytes.Contains(result.Source, []byte("__c_static_1_calls++;")) {
		t.Fatalf("static local references were not hoisted consistently:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated linkage source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateRejectsConflictingTentativeDefinition(t *testing.T) {
	for _, source := range []string{"int shared; long shared;", "int values[2]; int values[3];"} {
		result := TranslateObject("main", []byte(source), nil)
		if result.Ok || result.Error != TranslateErrDeclaration {
			t.Fatalf("conflicting tentative definition %q result = %#v", source, result)
		}
	}
}

func TestTranslateRejectsThreadStorageUntilTLSLoweringExists(t *testing.T) {
	for _, source := range []string{
		"_Thread_local int value;",
		"_Thread_local static int value;",
		"extern _Thread_local int value;",
	} {
		result := TranslateObject("main", []byte(source), nil)
		if result.Ok || result.Error != TranslateErrUnsupported {
			t.Fatalf("thread-storage %q result = %#v", source, result)
		}
	}
}

func TestTranslateDesignatedInitializersAndCompoundLiterals(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct inner { int x; int y; };
struct record {
	int head;
	struct inner nested;
	int values[3];
	int tail;
};
union word { unsigned whole; unsigned char bytes[4]; };
struct packed { unsigned low : 3; unsigned high : 5; int value; };
int inspect(void) {
	struct record selected = {
		.tail = 5,
		.nested.y = 3,
		.nested.x = 2,
		.values[2] = 7,
		.values[0] = 1
	};
	struct record positional = { 1, { 2, 3 }, { 4, 5, 6 }, 7 };
	int sparse[5] = { [3] = 9, 10 };
	int inferred[] = { [2] = 8, 9 };
	int overwritten[2] = { [0] = 1, [0] = 4 };
	union word word = { .whole = 0x04030201 };
	union word bytes = (union word){ .bytes = { 9, 8, 7, 6 } };
	struct packed packed = { .high = 17, .low = 5, .value = 4 };
	struct packed sequence = { 6, 3, 2 };
	return selected.tail + selected.nested.x + selected.values[2] +
		positional.nested.y + sparse[4] + inferred[3] + ((struct inner){ .x = 4, .y = 6 }).y +
		word.bytes[0] + bytes.bytes[1] + packed.high + packed.low + packed.value +
		sequence.low + sequence.high + sequence.value + overwritten[0];
}
`), nil)
	if !result.Ok {
		t.Fatalf("aggregate initializer translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("p.nested.y=v1;p.nested.x=v2;p.values[2]=v3;p.values[0]=v4"),
		[]byte("__c_struct_record{head:1,nested:__c_struct_inner{x:2,y:3},values:[3]int32{4,5,6},tail:7}"),
		[]byte("p[3]=v0;p[4]=v1"),
		[]byte("var inferred [4]int32=__c_aggregate_init_"),
		[]byte("p[0]=v0;p[0]=v1"),
		[]byte("func __c_union_init_"),
		[]byte("(*p.__c_ptr_whole())=v"),
		[]byte("(*p.__c_ptr_bytes())=v"),
		[]byte("func __c_aggregate_init_"),
		[]byte("p.__c_set_high(v0)"),
		[]byte("p.__c_set_low(v1)"),
		[]byte("p.value=v2"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("aggregate initializer source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated aggregate initializer does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
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
