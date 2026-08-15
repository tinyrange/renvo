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

func TestTranslateGNUAttributesPreserveAggregateLayout(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef struct aligned_record {
	unsigned char tag;
} __attribute__((aligned(sizeof(void *)))) aligned_record;
typedef struct packed_record {
	unsigned char tag;
	int value;
} __attribute__((packed)) packed_record;
static __attribute__((__noinline__, __noclone__, __no_stack_protector__, nocf_check, __no_sanitize_address__, __no_profile_instrument_function__)) inline __attribute__((__gnu_inline__, __unused__)) int inspect(void) {
	aligned_record aligned;
	packed_record packed;
	aligned.tag = 3;
	packed.tag = 5;
	packed.value = 7;
	return sizeof(aligned_record) + _Alignof(aligned_record) +
		sizeof(packed_record) + _Alignof(packed_record) +
		__builtin_offsetof(packed_record, value) + aligned.tag + packed.value;
}
`), nil)
	if !result.Ok {
		t.Fatalf("GNU attribute translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("type __c_struct_aligned_record struct{__c_align uint64}"),
		[]byte("type __c_struct_packed_record struct{__c_align uint8;__c_tail [4]byte}"),
		[]byte("return 23+"),
		[]byte(".__c_ptr_value()"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("translated GNU attribute source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated GNU attribute source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateRejectsUnknownGNUAttribute(t *testing.T) {
	result := TranslateObject("main", []byte(`int value __attribute__((renvo_unknown));`), nil)
	if result.Ok || result.Error != TranslateErrUnsupported {
		t.Fatalf("unknown GNU attribute result = %#v, want explicit unsupported error", result)
	}
}

func TestCheckTransparentUnionCallingConventionAttribute(t *testing.T) {
	source := []byte(`
typedef union { int *integers; long *longs; } argument __attribute__((__transparent_union__));
void release(argument value);
`)
	checked := CheckObjectForDataModel(source, DataModelLP64)
	if !checked.Ok {
		t.Fatalf("transparent union check failed: error=%d at=%d", checked.Error, checked.ErrorAt)
	}
	lowered := TranslateObject("main", source, nil)
	if lowered.Ok || lowered.Error != TranslateErrUnsupported {
		t.Fatalf("transparent union lowering result = %#v, want explicit unsupported error", lowered)
	}
}

func TestCheckAliasSymbolAttribute(t *testing.T) {
	source := []byte(`long target(void); long alias(void) __attribute__((weak, __alias__("target")));`)
	checked := CheckObjectForDataModel(source, DataModelLP64)
	if !checked.Ok {
		t.Fatalf("alias attribute check failed: error=%d at=%d", checked.Error, checked.ErrorAt)
	}
	lowered := TranslateObject("main", source, nil)
	if lowered.Ok || lowered.Error != TranslateErrUnsupported {
		t.Fatalf("alias lowering result = %#v, want explicit unsupported error", lowered)
	}
}

func TestCheckFileScopeAssembler(t *testing.T) {
	source := []byte(`asm(".section .initcall4, \"a\"\n.long initialize - .\n.previous");`)
	checked := CheckObjectForDataModel(source, DataModelLP64)
	if !checked.Ok {
		t.Fatalf("file-scope asm check failed: error=%d at=%d", checked.Error, checked.ErrorAt)
	}
	lowered := TranslateObject("main", source, nil)
	if lowered.Ok || lowered.Error != TranslateErrUnsupported {
		t.Fatalf("file-scope asm lowering result = %#v, want explicit unsupported error", lowered)
	}
}

func TestCheckVisibilitySymbolAttribute(t *testing.T) {
	source := []byte(`extern int hidden[2] __attribute__((visibility("hidden")));`)
	checked := CheckObjectForDataModel(source, DataModelLP64)
	if !checked.Ok {
		t.Fatalf("visibility attribute check failed: error=%d at=%d", checked.Error, checked.ErrorAt)
	}
	lowered := TranslateObject("main", source, nil)
	if lowered.Ok || lowered.Error != TranslateErrUnsupported {
		t.Fatalf("visibility lowering result = %#v, want explicit unsupported error", lowered)
	}
}

func TestCheckCallingConventionAttribute(t *testing.T) {
	source := []byte(`
typedef unsigned long firmware_call(unsigned int command);
struct services { firmware_call __attribute__((ms_abi)) *call; };
`)
	checked := CheckObjectForDataModel(source, DataModelLP64)
	if !checked.Ok {
		t.Fatalf("calling-convention attribute check failed: error=%d at=%d", checked.Error, checked.ErrorAt)
	}
	lowered := TranslateObject("main", source, nil)
	if lowered.Ok || lowered.Error != TranslateErrUnsupported {
		t.Fatalf("calling-convention lowering result = %#v, want explicit unsupported error", lowered)
	}
}

func TestTranslateBuiltinVaListUsesX8664SysVLayout(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef __builtin_va_list va_list;
extern int consume(const char *, va_list);
int inspect(void) { return sizeof(va_list) + _Alignof(va_list); }
`), nil)
	if !result.Ok {
		t.Fatalf("builtin va_list translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("func consume(p0 string,p1 *uintptr) int32"),
		[]byte("return 32"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("translated builtin va_list source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated builtin va_list source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestCheckObjectAcceptsDeferredGNUAsm(t *testing.T) {
	source := []byte(`
register unsigned long stack_pointer asm("rsp");
void *adjust(void *pointer) {
	asm volatile("leaq %c1(%%rip), %0" : "=r"(pointer) : "i"(pointer));
	return pointer;
}
`)
	checked := CheckObjectForDataModel(source, DataModelLP64)
	if !checked.Ok {
		t.Fatalf("GNU asm semantic check failed: error=%d at=%d", checked.Error, checked.ErrorAt)
	}
	lowered := TranslateObjectForDataModel("main", source, nil, DataModelLP64)
	if lowered.Ok || lowered.Error != TranslateErrUnsupported {
		t.Fatalf("GNU asm lowering result = %#v, want explicit unsupported error", lowered)
	}
	malformed := CheckObjectForDataModel([]byte(`int inspect(void) { asm("broken"; }`), DataModelLP64)
	if malformed.Ok {
		t.Fatalf("malformed GNU asm check = %#v, want failure", malformed)
	}
}

func TestCheckObjectAcceptsDeferredCleanupAttribute(t *testing.T) {
	source := []byte(`
void release(int *value);
int inspect(void) { int value __attribute__((__cleanup__(release))) = 3; return value; }
`)
	checked := CheckObjectForDataModel(source, DataModelLP64)
	if !checked.Ok {
		t.Fatalf("cleanup attribute check failed: error=%d at=%d", checked.Error, checked.ErrorAt)
	}
	lowered := TranslateObjectForDataModel("main", source, nil, DataModelLP64)
	if lowered.Ok || lowered.Error != TranslateErrUnsupported {
		t.Fatalf("cleanup attribute lowering result = %#v, want explicit unsupported error", lowered)
	}
}

func TestCheckObjectModelsDeferredInt128(t *testing.T) {
	source := []byte(`
typedef __signed__ __int128 s128 __attribute__((aligned(16)));
typedef unsigned __int128 u128 __attribute__((aligned(16)));
union halves { u128 full; struct { unsigned long low, high; }; };
int widths[(sizeof(u128) == 16 && _Alignof(u128) == 16 && sizeof(union halves) == 16) ? 1 : -1];
`)
	checked := CheckObjectForDataModel(source, DataModelLP64)
	if !checked.Ok {
		t.Fatalf("int128 semantic check failed: error=%d at=%d", checked.Error, checked.ErrorAt)
	}
	lowered := TranslateObjectForDataModel("main", source, nil, DataModelLP64)
	if lowered.Ok || lowered.Error != TranslateErrUnsupported {
		t.Fatalf("int128 lowering result = %#v, want explicit unsupported error", lowered)
	}
}

func TestTranslateAutoTypeInfersInitializerType(t *testing.T) {
	result := TranslateObject("main", []byte(`
int inspect(unsigned short *source) {
	const __auto_type saved = source;
	return *saved;
}
`), nil)
	if !result.Ok || !bytes.Contains(result.Source, []byte("var saved *uint16=source")) {
		t.Fatalf("__auto_type translation = %#v\n%s", result, result.Source)
	}
	for _, source := range []string{
		"int inspect(void) { __auto_type missing; return 0; }",
		"int inspect(void) { __auto_type first = 1, second = 2; return first; }",
	} {
		invalid := TranslateObject("main", []byte(source), nil)
		if invalid.Ok || invalid.Error != TranslateErrDeclaration {
			t.Fatalf("invalid __auto_type %q result = %#v", source, invalid)
		}
	}
}

func TestTranslateTypeofExpressionAndTypeName(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct record { unsigned short value; };
int inspect(struct record *source) {
	typeof(source->value) first = 3;
	__typeof__(unsigned long) second = 4;
	return first + second;
}
`), nil)
	if !result.Ok {
		t.Fatalf("typeof translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{[]byte("var first uint16=3"), []byte("var second uint64=4")} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("translated typeof source is missing %q:\n%s", want, result.Source)
		}
	}
}

func TestTranslateGNUZeroLengthArrayLayout(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct trailer { int head; char data[20 - 2 * sizeof(long) - sizeof(int)]; };
int inspect(void) { return sizeof(struct trailer) + __builtin_offsetof(struct trailer, data); }
`), nil)
	if !result.Ok || !bytes.Contains(result.Source, []byte("return 8")) {
		t.Fatalf("zero-length array translation = %#v\n%s", result, result.Source)
	}
}

func TestCheckObjectAppliesPostfixIncrementBeforeUnaryDereference(t *testing.T) {
	checked := CheckObjectForDataModel([]byte(`
int inspect(const unsigned char *cursor) {
	while (*cursor++) { }
	return 0;
}
`), DataModelLP64)
	if !checked.Ok {
		t.Fatalf("postfix pointer increment check failed: error=%d at=%d", checked.Error, checked.ErrorAt)
	}
}

func TestCheckObjectValidatesGNUCaseRanges(t *testing.T) {
	valid := CheckObjectForDataModel([]byte(`
int classify(int value) {
	switch (value) { case '0' ... '9': return 1; default: return 0; }
}
`), DataModelLP64)
	if !valid.Ok {
		t.Fatalf("GNU case range was rejected: %#v", valid)
	}
	invalid := CheckObjectForDataModel([]byte(`
int classify(int value) {
	switch (value) { case 9 ... 0: return 1; default: return 0; }
}
`), DataModelLP64)
	if invalid.Ok || invalid.Error != TranslateErrStatement {
		t.Fatalf("reversed GNU case range result = %#v", invalid)
	}
	lowered := TranslateObject("main", []byte(`
int classify(int value) {
	switch (value) { case 0 ... 9: return 1; default: return 0; }
}
`), nil)
	if lowered.Ok || lowered.Error != TranslateErrUnsupported {
		t.Fatalf("case range lowering result = %#v", lowered)
	}
}

func TestCheckObjectSpeculativeCastDoesNotReportVLA(t *testing.T) {
	checked := CheckObjectForDataModel([]byte(`
unsigned long inspect(unsigned long *map, unsigned long index, unsigned long offset) {
	return (map[index] >> offset) & 3;
}
`), DataModelLP64)
	if !checked.Ok {
		t.Fatalf("parenthesized array expression check failed: error=%d at=%d", checked.Error, checked.ErrorAt)
	}
}

func TestTranslateStaticAssertions(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct cacheline { unsigned char data[64]; };
struct rows { int values[3]; };
_Static_assert(sizeof(struct cacheline) == 64, "cacheline layout");
_Static_assert(__builtin_offsetof(struct cacheline, data) == 0, "field offset");
_Static_assert(__builtin_offsetof(struct rows, values[2]) == 8, "array designator offset");
int inspect(void) { static_assert(_Alignof(long) == 8); return 0; }
`), nil)
	if !result.Ok {
		t.Fatalf("static assertion translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	invalid := TranslateObject("main", []byte(`_Static_assert(sizeof(long) == 4, "wrong model");`), nil)
	if invalid.Ok || invalid.Error != TranslateErrDeclaration {
		t.Fatalf("false static assertion result = %#v", invalid)
	}
}

func TestTranslateBuiltinTypeAndConstantQueries(t *testing.T) {
	result := TranslateObject("main", []byte(`
int first(int);
int second(int);
long different(long);
struct wrapper { int value; };
_Static_assert(__builtin_types_compatible_p(typeof(first), typeof(second)), "same function type");
_Static_assert(!__builtin_types_compatible_p(typeof(first), typeof(different)), "different function type");
_Static_assert(__builtin_types_compatible_p(typeof(*((struct wrapper *)0)), typeof(struct wrapper)), "cast dereference type");
_Static_assert(__builtin_types_compatible_p(typeof(((struct wrapper *)0)->value), typeof(int)), "cast member type");
_Static_assert(__builtin_constant_p(3 + 4), "constant expression");
_Static_assert(!__builtin_constant_p(first(3)), "runtime expression");
`), nil)
	if !result.Ok {
		t.Fatalf("builtin type query translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
}

func TestTranslateHexDigitsAreNotFloatingSuffixes(t *testing.T) {
	result := TranslateObject("main", []byte(`
enum flags { FIRST = 0x000e, SECOND = 0xff, BOTH = FIRST | SECOND };
_Static_assert(BOTH == 0xff, "hex digits remain integer constants");
`), nil)
	if !result.Ok {
		t.Fatalf("hex integer translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
}

func TestTranslateTypedefAndTagNamespacesAreBlockScoped(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef int item;
struct item { int item; };
enum outer_value { outer_value = 3 };
int inspect(void) {
	item before = 1;
	struct item record = { .item = 2 };
	{
		long item = 4;
		item = item + 1;
		{
			typedef short item;
			item nested = 6;
			before += nested;
		}
		before += item;
	}
	item after = outer_value;
	return before + record.item + after;
}
`), nil)
	if !result.Ok {
		t.Fatalf("namespace translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("var before int32=1"),
		[]byte("var item int64=4"),
		[]byte("item=item+1"),
		[]byte("var nested int16=6"),
		[]byte("var after int32=3"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("namespace source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated namespaces do not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
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
int alpha(void), beta(int value);
int qualified_parameter(int values[const static 3]) { return values[2]; }
int abstract_layout(void) { return sizeof(int (*)[3]) + _Alignof(int (*)(int)); }
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
		[]byte("func alpha() int32"),
		[]byte("func beta(p0 int32) int32"),
		[]byte("func qualified_parameter(values *int32) int32"),
		[]byte("func abstract_layout() int32 {return 16;"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("translated declarators are missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated declarators do not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
	if invalid := TranslateObject("main", []byte("int invalid(int values[const 3]) { values = 0; return 0; }"), nil); invalid.Ok {
		t.Fatalf("const-qualified adjusted array parameter was assignable:\n%s", invalid.Source)
	}
	for _, source := range []string{"typedef int conflict; int conflict;", "enum values { conflict }; int conflict(void);", "int conflict; int conflict(void);"} {
		if invalid := TranslateObject("main", []byte(source), nil); invalid.Ok {
			t.Fatalf("ordinary-identifier namespace conflict was accepted: %s", source)
		}
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
int converted[-1 < 1U ? 3 : 4];
int converted64[~0ULL > 0 ? 3 : 4];
int selected(void) { return mode_five + mode_mask; }
`))
	if !result.Ok {
		t.Fatalf("enum translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("var values [16]int32"),
		[]byte("var converted [4]int32"),
		[]byte("var converted64 [3]int32"),
		[]byte("return 11"),
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
	if !bytes.Contains(result.Source, []byte("return 68")) {
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
		[]byte("(*word_cursor.__c_ptr_whole())=67305985"),
		[]byte("(*word.__c_ptr_bytes())[0]"),
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
	unsigned char narrow = choice;
	switch (narrow) { case 300: total = 1000; default: break; }
	unsigned mask = ~0U;
	switch (mask) { case -1: total += 4; break; }
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
		[]byte("switch int32(narrow){case 300:"),
		[]byte("switch mask{case 4294967295:"),
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

func TestTranslateThreadStorage(t *testing.T) {
	result := TranslateObject("main", []byte(`
extern _Thread_local int value, second;
int step(int delta) { value += delta; second = value; return second; }
_Thread_local int value = 3;
_Thread_local int second;
`), nil)
	if !result.Ok {
		t.Fatalf("thread-storage translation failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("linkstatic libc,pthread_key_create"),
		[]byte("linkstatic libc,pthread_mutex_lock"),
		[]byte("func __c_tls_get_"),
		[]byte("(*__c_tls_get_"),
		[]byte("__c_calloc(1,4)"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("thread-storage source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated thread storage does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
	if external := TranslateObject("main", []byte("extern _Thread_local int value;"), nil); external.Ok || external.Error != TranslateErrUnsupported {
		t.Fatalf("unresolved external TLS result = %#v", external)
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
struct text { char label[3]; unsigned char wrapped; };
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
	struct text text = { "ok", 300 };
	struct text named = { .wrapped = 301, .label = "hi" };
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
		[]byte("label:[3]int8{111,107,0},wrapped:44"),
		[]byte("p.wrapped=v0;p.label=v1"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("aggregate initializer source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated aggregate initializer does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateVariadicPrototypeCalls(t *testing.T) {
	result := TranslateObject("main", []byte(`
int printf(const char *format, ...);
int inspect(void) {
	unsigned char small = 7;
	short negative = -3;
	return printf("values %d %d %.1f\n", small, negative, (float)2.5);
}

`), nil)
	if !result.Ok {
		t.Fatalf("variadic translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("return __c_variadic_"),
		[]byte("int32(small),int32(negative)"),
		[]byte("float64(float32(2.5))"),
		[]byte("// renvo:linkstatic libc,printf"),
		[]byte("func __c_variadic_"),
		[]byte("(p0 string,p1 int32,p2 int32,p3 float64) int32"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("variadic source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated variadic call does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateQualifierSemantics(t *testing.T) {
	result := TranslateObject("main", []byte(`
const int immutable = 3;
volatile int observed;
int inspect(const int parameter, int *restrict cursor) {
	observed = *cursor;
	return immutable + observed + parameter;
}

`), nil)
	if !result.Ok {
		t.Fatalf("qualified translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("var immutable int32=3"),
		[]byte("var observed int32"),
		[]byte("func inspect(parameter int32,cursor *int32) int32"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("qualified source is missing %q:\n%s", want, result.Source)
		}
	}
	for _, source := range []string{
		"const int value = 1; int f(void) { value = 2; return value; }",
		"struct pair { int x; }; const struct pair value = {1}; int f(void) { value.x++; return value.x; }",
		"int f(const int *value) { *value = 2; return *value; }",
		"int f(int * const value) { value = 0; return 0; }",
		"restrict int invalid;",
	} {
		invalid := TranslateObject("main", []byte(source), nil)
		if invalid.Ok {
			t.Fatalf("qualifier mutation/constraint was accepted: %s\n%s", source, invalid.Source)
		}
	}
}

func TestCheckGNUAlternateQualifierKeywords(t *testing.T) {
	result := CheckObjectForDataModel([]byte(`
static __inline__ char *copy(char *__restrict__ destination, const char *__restrict source) {
	char *__volatile__ cursor = destination;
	return cursor + (*source != 0);
}
`), DataModelLP64)
	if !result.Ok {
		t.Fatalf("GNU alternate qualifier keywords failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
}

func TestCheckFunctionParameterScope(t *testing.T) {
	valid := CheckObjectForDataModel([]byte(`
long error;
void *error_pointer(long error) { return (void *)error; }
`), DataModelLP64)
	if !valid.Ok {
		t.Fatalf("file-scope object could not be shadowed by a parameter: error=%d at=%d", valid.Error, valid.ErrorAt)
	}

	invalid := CheckObjectForDataModel([]byte(`
int duplicate(int value) { int value = 1; return value; }
`), DataModelLP64)
	if invalid.Ok || invalid.Error != TranslateErrDeclaration {
		t.Fatalf("outer function block redeclared a parameter: %#v", invalid)
	}
}

func TestCheckQualifiedIncompleteTypeTracksCompletion(t *testing.T) {
	result := CheckObjectForDataModel([]byte(`
struct device;
int inspect(const struct device *device);
struct device { int value; };
int inspect(const struct device *device);
`), DataModelLP64)
	if !result.Ok {
		t.Fatalf("qualified incomplete type lost its completion: error=%d at=%d", result.Error, result.ErrorAt)
	}
}

func TestCheckSizeofExpressionIsAnIntegerConstant(t *testing.T) {
	result := CheckObjectForDataModel([]byte(`
void format(unsigned long value) {
	char buffer[8 * sizeof(value) + 1];
	buffer[0] = 0;
}
`), DataModelLP64)
	if !result.Ok {
		t.Fatalf("sizeof expression was rejected as an array bound: error=%d at=%d", result.Error, result.ErrorAt)
	}
}

func TestCheckGNUIntegerBuiltinsInConstants(t *testing.T) {
	result := CheckObjectForDataModel([]byte(`
int leading[__builtin_clzll(1) == 63 ? 1 : -1];
int trailing[__builtin_ctz(8) == 3 ? 1 : -1];
int population[__builtin_popcount(7) == 3 ? 1 : -1];
int first[__builtin_ffs(8) == 4 ? 1 : -1];
`), DataModelLP64)
	if !result.Ok {
		t.Fatalf("GNU integer builtin constants failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
}

func TestCheckConstantConditionalSkipsRuntimeArm(t *testing.T) {
	result := CheckObjectForDataModel([]byte(`
int runtime_size(void);
int selected_left[1 ? 4 : runtime_size()];
int selected_right[0 ? runtime_size() : 4];
`), DataModelLP64)
	if !result.Ok {
		t.Fatalf("constant conditional evaluated its runtime arm: error=%d at=%d", result.Error, result.ErrorAt)
	}
}

func TestCheckGNUComposedTypeExpressions(t *testing.T) {
	result := CheckObjectForDataModel([]byte(`
struct node { int value; };
void inspect(struct node *pointer) {
	_Static_assert(__builtin_types_compatible_p(
		typeof(({ struct node *copy = pointer; copy; })), struct node *), "statement expression type");
	_Static_assert(__builtin_types_compatible_p(
		typeof(_Generic(pointer, int: 0, default: pointer)), struct node *), "generic selection type");
}
`), DataModelLP64)
	if !result.Ok {
		t.Fatalf("composed GNU type expressions failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
}

func TestCheckAddressedMemberTypeExpression(t *testing.T) {
	result := CheckObjectForDataModel([]byte(`
struct list { struct list *next; };
struct event { struct list sibling; };
void inspect(struct event *leader, struct event *event) {
	_Static_assert(__builtin_types_compatible_p(
		typeof(*((&leader->sibling)->next)),
		typeof(((typeof(*event) *)0)->sibling)), "addressed member type");
}
`), DataModelLP64)
	if !result.Ok {
		t.Fatalf("addressed member type check failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
}

func TestCheckTypeofFunctionDeclarator(t *testing.T) {
	result := CheckObjectForDataModel([]byte(`
struct operations { void (*call)(int); };
extern struct operations *operations;
extern typeof(*operations->call) call_thunk;
typedef void (*callback)(int);
void implementation(int value) { (void)value; }
extern typeof(implementation) implementation;
_Static_assert(__builtin_types_compatible_p(typeof(callback), typeof(&implementation)), "function address type");
`), DataModelLP64)
	if !result.Ok {
		t.Fatalf("typeof function declaration failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
}

func TestCheckFunctionTypeTypedefDeclarations(t *testing.T) {
	result := CheckObjectForDataModel([]byte(`
struct target;
typedef int active_fn(struct target *target, unsigned int index);
extern active_fn first_active, second_active;
`), DataModelLP64)
	if !result.Ok {
		t.Fatalf("function type typedef declarations failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
}

func TestCheckStaticForwardFunctionType(t *testing.T) {
	result := CheckObjectForDataModel([]byte(`
struct registers;
typedef unsigned int u32;
static void dispatch(struct registers *, u32);
void wrapper(struct registers *registers) {
	_Static_assert(__builtin_types_compatible_p(
		typeof(&dispatch), void (*)(struct registers *, u32)), "static forward function type");
}
static void dispatch(struct registers *registers, u32 vector) { (void)registers; (void)vector; }
`), DataModelLP64)
	if !result.Ok {
		t.Fatalf("static forward function type check failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
}

func TestTranslatePackedEnumUsesSmallestIntegerRepresentation(t *testing.T) {
	result := TranslateObject("main", []byte(`
enum rw_hint { WRITE_NONE, WRITE_SHORT, WRITE_LONG = 5 } __attribute__((__packed__));
int inspect(void) { _Static_assert(sizeof(enum rw_hint) == 1, "packed enum"); return sizeof(enum rw_hint); }
`), nil)
	if !result.Ok {
		t.Fatalf("packed enum translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("return 1")) {
		t.Fatalf("packed enum did not use a one-byte representation:\n%s", result.Source)
	}
}

func TestCheckGNUArrayRangeDesignators(t *testing.T) {
	source := []byte(`
struct page { unsigned offset; unsigned size; };
static const struct page pages[8] = {
	[1 ... 3] = { .offset = 4, .size = 8 },
	[7] = { .offset = 12, .size = 16 },
};
`)
	checked := CheckObjectForDataModel(source, DataModelLP64)
	if !checked.Ok {
		t.Fatalf("array range designator check failed: error=%d at=%d", checked.Error, checked.ErrorAt)
	}
	lowered := TranslateObject("main", source, nil)
	if lowered.Ok || lowered.Error != TranslateErrUnsupported {
		t.Fatalf("array range lowering result = %#v, want explicit unsupported error", lowered)
	}
}

func TestCheckGNUOmittedConditionalOperand(t *testing.T) {
	source := []byte(`
struct aligned { char value; } __attribute__((aligned((0) ? : 64)));
int select_value(int value) { return value ? : 7; }
`)
	checked := CheckObjectForDataModel(source, DataModelLP64)
	if !checked.Ok {
		t.Fatalf("omitted conditional operand check failed: error=%d at=%d", checked.Error, checked.ErrorAt)
	}
	lowered := TranslateObject("main", source, nil)
	if lowered.Ok || lowered.Error != TranslateErrUnsupported {
		t.Fatalf("omitted conditional lowering result = %#v, want explicit unsupported error", lowered)
	}
}

func TestTranslateGNUEmptyInferredArray(t *testing.T) {
	result := TranslateObject("main", []byte(`
static const int none[] = {};
int inspect(void) { return sizeof(none); }
`), nil)
	if !result.Ok {
		t.Fatalf("empty inferred array translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("var none [0]int32")) || !bytes.Contains(result.Source, []byte("return 0")) {
		t.Fatalf("empty inferred array representation is wrong:\n%s", result.Source)
	}
}

func TestCheckGNUEmptyLocalAggregateInitializer(t *testing.T) {
	result := CheckObjectForDataModel([]byte(`
union revision { unsigned int full; struct { unsigned char stepping; }; };
unsigned int inspect(void) { union revision value = {}; return value.full; }
`), DataModelLP64)
	if !result.Ok {
		t.Fatalf("empty local aggregate initializer check failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
}

func TestCheckExplicitZeroLengthLocalArray(t *testing.T) {
	valid := CheckObjectForDataModel([]byte(`int inspect(void) { int values[0]; return sizeof(values); }`), DataModelLP64)
	if !valid.Ok {
		t.Fatalf("explicit zero-length local array failed: error=%d at=%d", valid.Error, valid.ErrorAt)
	}
	invalid := CheckObjectForDataModel([]byte(`int inspect(void) { int values[]; return 0; }`), DataModelLP64)
	if invalid.Ok || invalid.Error != TranslateErrDeclaration {
		t.Fatalf("incomplete local array result = %#v", invalid)
	}
}

func TestCheckBracedScalarInitializer(t *testing.T) {
	result := CheckObjectForDataModel([]byte(`
struct operations { int value; };
static const struct operations implementation = { .value = 1 };
static const struct operations *selected = { &implementation };
`), DataModelLP64)
	if !result.Ok {
		t.Fatalf("braced scalar initializer check failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
}

func TestTranslateIntegerPromotionsAndConversions(t *testing.T) {
	result := TranslateObject("main", []byte(`
int inspect(void) {
	unsigned char left = 250, right = 10;
	short wide = 30000;
	signed char negative = -1;
	unsigned int one = 1;
	long large = -1;
	int sum = left + right;
	int doubled = wide + wide;
	int unsigned_compare = negative < one;
	int rank_compare = large < one;
	_Bool normalized = 260;
	int selected = negative < one ? 7 : 9;
	int shifted = one << (long)3;
	unsigned long all = ~0ULL;
	return sum == 260 && doubled == 60000 && unsigned_compare == 0 && rank_compare == 1 && normalized == 1 && selected == 9 && sizeof(one << (long)3) == 4 && shifted == 8 && all > 0;
}
`), nil)
	if !result.Ok {
		t.Fatalf("promotion translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("int32(left)+int32(right)"),
		[]byte("int32(wide)+int32(wide)"),
		[]byte("uint32(negative)<one"),
		[]byte("large<int64(one)"),
		[]byte("var shifted int32=int32(one<<3)"),
		[]byte("var all uint64=^uint64(0)"),
		[]byte("__c_bool_int("),
		[]byte("var normalized uint8=uint8(__c_bool_int((260)!=0))"),
		[]byte("__c_conditional_"),
		[]byte("func __c_conditional_1(c bool"),
		[]byte("if c{return 7};return 9"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("promotion source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated promotions do not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateCExpressionSequencingAndFunctionPointers(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef int (*binary_fn)(int, int);
static int side;
static int add(int left, int right) { return left + right; }
static int mark(int value) { side = side * 10 + value; return value; }
static int ignored_variadic(int fixed, ...) { return fixed; }
enum constants { letter = 'A', chosen = 0 ? 11 : 13, narrowed = (unsigned char)258, unsigned_compare = -1 < 1U };
int inspect(int choose) {
	binary_fn operation = add;
	unsigned char byte = 250;
	int array[3] = {1, 2, 3};
	int *cursor = array;
	int selected = choose ? mark(2) : mark(3);
	int comma = (mark(4), mark(5));
	return selected + comma + operation(7, 5) + ~byte + sizeof array + sizeof *cursor +
		letter + chosen + narrowed + unsigned_compare + ignored_variadic(9, mark(6), 2.5);
}
`), nil)
	if !result.Ok {
		t.Fatalf("expression translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("func __c_conditional_"),
		[]byte("if c{return mark(2)};return mark(3)"),
		[]byte("func __c_comma_"),
		[]byte("mark(4);return mark(5)"),
		[]byte("var operation func(int32,int32) int32=add"),
		[]byte("operation(7,5)"),
		[]byte("int32(-int32(byte)-1)"),
		[]byte("+12+4+65+13+2+0+"),
		[]byte("return ignored_variadic(p0)"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("expression source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated expressions do not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateCharacterArrayStringInitialization(t *testing.T) {
	result := TranslateObject("main", []byte(`
char inferred[] = "A" "\x42";
char escape[] = "\e";
char exact[2] = "ok";
unsigned char padded[4] = u8"x";
int inspect(void) { return sizeof inferred + sizeof "xy" + inferred[0] + inferred[1] + inferred[2] + exact[1] + padded[1] + padded[3]; }
`), nil)
	if !result.Ok {
		t.Fatalf("character-array translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("var inferred [3]int8=[3]int8{65,66,0}"),
		[]byte("var escape [2]int8=[2]int8{27,0}"),
		[]byte("var exact [2]int8=[2]int8{111,107}"),
		[]byte("var padded [4]uint8=[4]uint8{120,0}"),
		[]byte("return 3+3+int32(inferred[0])"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("character-array source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated character arrays do not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
	for _, source := range []string{
		`char too_small[1] = "no";`,
		`char unsupported[2] = L"x";`,
	} {
		invalid := TranslateObject("main", []byte(source), nil)
		if invalid.Ok {
			t.Fatalf("invalid string-array initializer %q translated successfully", source)
		}
	}
}

func TestTranslateAtomicTypesAndPreciseVLARejection(t *testing.T) {
	result := TranslateObject("main", []byte(`
_Atomic(int) first;
int _Atomic second;
int inspect(void) { int * _Atomic pointer = 0; return sizeof(first) + sizeof(second) + sizeof(pointer); }
`), nil)
	if !result.Ok {
		t.Fatalf("atomic type translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("var first int32")) || !bytes.Contains(result.Source, []byte("var second int32")) {
		t.Fatalf("atomic declarations were not retained:\n%s", result.Source)
	}
	for _, source := range []string{
		"int inspect(int count) { int values[count]; return 0; }",
		"int inspect(int count, int values[count]) { return 0; }",
		"int inspect(int values[*]);",
	} {
		invalid := TranslateObject("main", []byte(source), nil)
		if invalid.Ok || invalid.Error != TranslateErrVLA || invalid.ErrorAt < 0 {
			t.Fatalf("VLA %q result = %#v", source, invalid)
		}
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
int main(void) { puts("Hello, " "world\x21"); return 0; }
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

func TestTranslateObjectEmitsNullPointerCast(t *testing.T) {
	result := TranslateObject("main", []byte(`
const char *version(int);
int main(void) { return version(0) == ((void *)0); }
`), nil)
	if !result.Ok {
		t.Fatalf("TranslateObject failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("version(0)==(*int8)(nil)")) {
		t.Fatalf("translated null pointer cast is incorrect:\n%s", result.Source)
	}
	parsed := syntax.ParseFile(result.Source)
	if !parsed.Ok {
		t.Fatalf("translated null pointer cast does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
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
