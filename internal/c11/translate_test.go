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

func TestTranslateObjectDiagnosticErrorCallTrapsLocally(t *testing.T) {
	result := TranslateObject("main", []byte(`
extern void invalid_size(void) __attribute__((error("invalid size")));
void check(unsigned long size) {
	if (size == 3)
		invalid_size();
}
`), nil)
	if !result.Ok {
		t.Fatalf("diagnostic error call failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("renvo_runtime_CUndefinedInstruction()")) ||
		bytes.Contains(result.Source, []byte("invalid_size();")) {
		t.Fatalf("diagnostic error call was not lowered to a local trap:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectDiagnosticErrorCallInStatementExpressionTrapsLocally(t *testing.T) {
	result := TranslateObject("main", []byte(`
int check(int value) {
	return ({
		extern void invalid_type(void) __attribute__((error("invalid type")));
		if (value)
			invalid_type();
		1;
	});
}
`), nil)
	if !result.Ok {
		t.Fatalf("statement-expression diagnostic call failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("renvo_runtime_CUndefinedInstruction()")) ||
		bytes.Contains(result.Source, []byte("invalid_type();")) {
		t.Fatalf("statement-expression diagnostic call was not lowered to a local trap:\n%s", result.Source)
	}
}

func TestTranslateObjectAcceptsCMainSignature(t *testing.T) {
	result := TranslateObject("main", []byte(`void main(void) {}`), nil)
	if !result.Ok {
		t.Fatalf("TranslateObject failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("// renvo:object function main"),
		[]byte("//export main"),
		[]byte("func appMain()"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("translated object source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated object source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectLowersIRQStackCallAsm(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct regs { unsigned long ip; };
extern void irq_enter_rcu(void);
extern void irq_exit_rcu(void);
static void dispatch(struct regs *regs, unsigned int vector);
static void run_softirq(void);
void invoke(void *tos, void *current_stack_pointer, struct regs *regs, unsigned int vector) {
	asm volatile(
		"movq %%rsp, (%[tos])\n"
		"movq %[tos], %%rsp\n"
		"call irq_enter_rcu\n"
		"movq %[arg2], %%rsi\n"
		"movq %[arg1], %%rdi\n"
		"call %c[__func]\n"
		"998:\n\t.pushsection .discard.reachable\n\t.long 998b\n\t.popsection\n\t"
		"call irq_exit_rcu\n"
		"popq %%rsp\n"
		: "+r" (tos), "+r" (current_stack_pointer)
		: [__func] "i" (dispatch), [tos] "r" (tos), [arg1] "r" (regs), [arg2] "r" ((unsigned long)vector)
		: "cc", "rax", "rcx", "rdx", "rsi", "rdi", "r8", "r9", "r10", "memory");
}
void invoke_softirq(void *tos, void *current_stack_pointer) {
	asm volatile(
		"movq %%rsp, (%[tos])\n"
		"movq %[tos], %%rsp\n"
		"call %c[__func]\n"
		"998:\n\t.pushsection .discard.reachable\n\t.long 998b\n\t.popsection\n\t"
		"popq %%rsp\n"
		: "+r" (tos), "+r" (current_stack_pointer)
		: [__func] "i" (run_softirq), [tos] "r" (tos)
		: "cc", "rax", "rcx", "rdx", "rsi", "rdi", "r8", "r9", "r10", "memory");
}
static void dispatch(struct regs *regs, unsigned int vector) { (void)regs; (void)vector; }
static void run_softirq(void) {}
`), nil)
	if !result.Ok {
		t.Fatalf("IRQ stack-call lowering failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("func renvo_runtime_CIRQStackCall2_dispatch(tos *byte,arg1 *__c_struct_regs,arg2 uint64){}"),
		[]byte("renvo_runtime_CIRQStackCall2_dispatch(tos,regs,uint32(uint64(vector)));"),
		[]byte("func renvo_runtime_CIRQStackCall0_run_softirq(tos *byte){}"),
		[]byte("renvo_runtime_CIRQStackCall0_run_softirq(tos);"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("IRQ stack-call source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("IRQ stack-call source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectPacksNestedConstantBitfieldInitializer(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct bits {
	unsigned short ist : 3;
	unsigned short zero : 5;
	unsigned short type : 5;
	unsigned short dpl : 2;
	unsigned short present : 1;
} __attribute__((packed));
struct entry { unsigned int vector; struct bits bits; void *address; };
extern void handler(void);
static const struct entry entries[] = {
	{ .vector = 0xec, .bits.ist = 0, .bits.type = 14, .bits.dpl = 0, .bits.present = 1, .address = handler },
};
`), nil)
	if !result.Ok {
		t.Fatalf("nested bitfield initializer failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("bits:__c_struct_bits{__c_align:0,__c_tail:[1]byte{142}}")) ||
		bytes.Contains(result.Source, []byte("__c_aggregate_init_")) {
		t.Fatalf("nested bitfield initializer was not packed into constant object bytes:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("nested bitfield initializer source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectPacksIndirectConstantStructInitializer(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct descriptor {
	unsigned short limit;
	unsigned short base;
	unsigned char type : 4;
	unsigned char system : 1;
	unsigned char privilege : 2;
	unsigned char present : 1;
} __attribute__((packed));
static struct descriptor value = {
	.limit = 0xffff,
	.base = 0x1234,
	.type = 11,
	.system = 1,
	.privilege = 3,
	.present = 1,
};
`), nil)
	if !result.Ok {
		t.Fatalf("indirect constant struct initializer failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("__c_struct_descriptor{__c_align:255,__c_tail:[4]byte{255,52,18,251}}")) ||
		bytes.Contains(result.Source, []byte("__c_aggregate_init_")) {
		t.Fatalf("indirect constant struct initializer was not packed into C-layout bytes:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("indirect constant struct source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateEmptyUnionInitializer(t *testing.T) {
	result := Translate("main", []byte(`
union value { unsigned int word; unsigned char bytes[4]; };
unsigned int zero(void) {
	union value value = {};
	return value.word;
}
`))
	if !result.Ok {
		t.Fatalf("empty union initializer failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("var value __c_union_value=__c_union_value{}")) {
		t.Fatalf("empty union initializer did not become a zero carrier:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("empty union initializer source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectPointerUnionInitializer(t *testing.T) {
	result := TranslateObject("main", []byte(`
static int target;
union value { void *pointer; unsigned long word; };
static union value value = { &target };
static union value empty = { .pointer = (void *)0 };
`), nil)
	if !result.Ok {
		t.Fatalf("pointer union initializer failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("__c_union_value{__c_align:uint64(&(target))}")) ||
		!bytes.Contains(result.Source, []byte("var empty __c_union_value=__c_union_value{__c_align:0}")) ||
		bytes.Contains(result.Source, []byte("__c_union_init_")) || bytes.Contains(result.Source, []byte("uint64(nil)")) {
		t.Fatalf("pointer union initializer did not become a relocatable scalar carrier:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("pointer union initializer source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectEncodesPackedAggregateFieldOffsets(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct descriptor { unsigned short size; unsigned long address; } __attribute__((packed));
static unsigned char table[1];
static struct descriptor value = { .size = 4095, .address = (unsigned long)table };
`), nil)
	if !result.Ok {
		t.Fatalf("packed aggregate initializer failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("=__c_object_aggregate_0_2_x")) {
		t.Fatalf("packed aggregate initializer does not encode its C field offsets:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("packed aggregate source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectPredefinedFunctionName(t *testing.T) {
	result := TranslateObject("main", []byte(`
extern void observe(const char *);
void sample(void) { observe(__func__); }
void unused(void) {}
`), nil)
	if !result.Ok {
		t.Fatalf("__func__ lowering failed: %#v", result)
	}
	if bytes.Count(result.Source, []byte("var __c_func_name_")) != 1 ||
		!bytes.Contains(result.Source, []byte("=[7]int8{115,97,109,112,108,101,0}")) {
		t.Fatalf("__func__ did not become one per-function string array:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("__func__ source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
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
	for (value = 4; value < 5;) value++;
	for (; value < 6; value++) {}
	return value;
}
`))
	for _, want := range [][]byte{
		[]byte("if value==0 {value=1;} else {value=2;}"),
		[]byte("for value<3 {value++;}"),
		[]byte("for value=3;value<4;value++"),
		[]byte("for value=4;value<5; {value++;}"),
		[]byte("for ;value<6;value++"),
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
		[]byte("value.values[2]=11"),
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
		[]byte("return int32(23+"),
		[]byte(".__c_ptr_1_value()"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("translated GNU attribute source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated GNU attribute source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateGNUExternInlineDoesNotExportDefinition(t *testing.T) {
	result := TranslateObject("main", []byte(`
extern inline __attribute__((__gnu_inline__)) int header_add(int value) {
	return value + 1;
}
int exported_add(int value) { return header_add(value); }
`), nil)
	if !result.Ok {
		t.Fatalf("GNU extern-inline translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if bytes.Contains(result.Source, []byte("//export header_add")) {
		t.Fatalf("GNU extern-inline definition was exported:\n%s", result.Source)
	}
	for _, want := range [][]byte{
		[]byte("func header_add(value int32) int32"),
		[]byte("//export exported_add\nfunc exported_add(value int32) int32"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("GNU extern-inline source is missing %q:\n%s", want, result.Source)
		}
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
void release(argument);
long *select_longs(argument value) { return value.longs; }
`)
	checked := CheckObjectForDataModel(source, DataModelLP64)
	if !checked.Ok {
		t.Fatalf("transparent union check failed: error=%d at=%d", checked.Error, checked.ErrorAt)
	}
	lowered := TranslateObject("main", source, nil)
	if !lowered.Ok {
		t.Fatalf("transparent union lowering failed: error=%d at=%d", lowered.Error, lowered.ErrorAt)
	}
	if !bytes.Contains(lowered.Source, []byte("func release(p0 *int32)")) {
		t.Fatalf("transparent union did not use its first member ABI:\n%s", lowered.Source)
	}
	if !bytes.Contains(lowered.Source, []byte("func select_longs(value *int32) *int64 {return (*int64)(value);")) ||
		bytes.Contains(lowered.Source, []byte("value.longs")) {
		t.Fatalf("transparent union member did not lower from its first-member ABI value:\n%s", lowered.Source)
	}
}

func TestCheckAliasSymbolAttribute(t *testing.T) {
	source := []byte(`long target(void) { return 7; }
long alias(void) __attribute__((weak, __alias__("target")));
long call_alias(void) { return alias(); }`)
	checked := CheckObjectForDataModel(source, DataModelLP64)
	if !checked.Ok {
		t.Fatalf("alias attribute check failed: error=%d at=%d", checked.Error, checked.ErrorAt)
	}
	lowered := TranslateObject("main", source, nil)
	if !lowered.Ok {
		t.Fatalf("alias lowering failed: %#v", lowered)
	}
	if !bytes.Contains(lowered.Source, []byte("// renvo:object function-alias alias - 0 2 0 target 8 0 - 0")) {
		t.Fatalf("alias lowering is missing object metadata:\n%s", lowered.Source)
	}
	if !bytes.Contains(lowered.Source, []byte("// renvo:linkstatic libc,alias")) {
		t.Fatalf("alias forward references are missing their relocatable declaration:\n%s", lowered.Source)
	}
	if !bytes.Contains(lowered.Source, []byte("func call_alias() int64 {return alias();")) {
		t.Fatalf("call through alias lost the aliased symbol name:\n%s", lowered.Source)
	}
}

func TestCheckFileScopeAssembler(t *testing.T) {
	source := []byte(`int initialize(void) { return 0; }
asm(".section \".initcall4.init\", \"a\"\n"
    "__initcall_initialize4:\n"
    ".long initialize - .\n"
    ".previous\n");`)
	checked := CheckObjectForDataModel(source, DataModelLP64)
	if !checked.Ok {
		t.Fatalf("file-scope asm check failed: error=%d at=%d", checked.Error, checked.ErrorAt)
	}
	lowered := TranslateObject("main", source, nil)
	if !lowered.Ok {
		t.Fatalf("file-scope asm lowering failed: %#v", lowered)
	}
	if !bytes.Contains(lowered.Source, []byte("// renvo:object variable __initcall_initialize4 .initcall4.init 4 0 0 - 4 2 initialize 0")) {
		t.Fatalf("file-scope relative entry metadata is missing:\n%s", lowered.Source)
	}
	if parsed := syntax.ParseFile(lowered.Source); !parsed.Ok {
		t.Fatalf("file-scope asm source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, lowered.Source)
	}
}

func TestTranslateFileScopeAssemblerWeakAlias(t *testing.T) {
	lowered := TranslateObject("main", []byte(`
long target(void) { return 7; }
asm(".weak alias\n\t.set alias,target");
`), nil)
	if !lowered.Ok {
		t.Fatalf("file-scope weak alias lowering failed: %#v", lowered)
	}
	if !bytes.Contains(lowered.Source, []byte("// renvo:object function-alias alias - 0 2 0 target 8 0 - 0")) {
		t.Fatalf("file-scope weak alias metadata is missing:\n%s", lowered.Source)
	}
	if parsed := syntax.ParseFile(lowered.Source); !parsed.Ok {
		t.Fatalf("file-scope weak alias source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, lowered.Source)
	}
}

func TestTranslateFileScopeAssemblerDataAndAbsoluteRelocation(t *testing.T) {
	source := []byte(`void exported(void);
asm(".section \".export_symbol\",\"a\" ; export_record: ; .asciz \"GPL\" ; .ascii \"\" \"\\0\" ; .balign 8 ; .quad exported ; .previous");`)
	lowered := TranslateObject("main", source, nil)
	if !lowered.Ok {
		t.Fatalf("file-scope data assembler lowering failed: %#v", lowered)
	}
	for _, want := range [][]byte{
		[]byte("// renvo:object variable export_record .export_symbol 1 0 0 - 8 0 - 0"),
		[]byte("[8]uint8{71,80,76,0,0,0,0,0}"),
		[]byte(".export_symbol 1 0 0 - 8 1 exported 0"),
	} {
		if !bytes.Contains(lowered.Source, want) {
			t.Fatalf("file-scope data assembler source is missing %q:\n%s", want, lowered.Source)
		}
	}
	if parsed := syntax.ParseFile(lowered.Source); !parsed.Ok {
		t.Fatalf("file-scope data assembler source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, lowered.Source)
	}
}

func TestTranslateFileScopeAssemblerNumericPCIEntry(t *testing.T) {
	result := TranslateObject("main", []byte(`
void fixup(void);
asm(".section .pci_fixup_header, \"a\"\n"
    ".balign 16\n"
    ".short 0x8086, 0x2670\n"
    ".long (~0), 0\n"
    ".long fixup - .\n"
    ".previous\n");
`), nil)
	if !result.Ok {
		t.Fatalf("numeric PCI assembler entry failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte(".pci_fixup_header 1 0 0 - 12 0 - 0"),
		[]byte("[12]uint8{134,128,112,38,255,255,255,255,0,0,0,0}"),
		[]byte(".pci_fixup_header 1 0 0 - 4 2 fixup 0"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("numeric PCI assembler source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("numeric PCI assembler source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateFileScopeStaticCallTemplate(t *testing.T) {
	source := []byte(`extern void __SCT__target(void);
asm(".pushsection .static_call.text, \"ax\"\n"
    ".globl __SCT__target\n"
    "__SCT__target:\n"
    "ret; int3; nop; nop; nop\n"
    ".byte 0x0f, 0xb9, 0xcc\n"
    ".popsection\n");`)
	lowered := TranslateObject("main", source, nil)
	if !lowered.Ok {
		t.Fatalf("file-scope static-call template failed: %#v", lowered)
	}
	for _, want := range [][]byte{
		[]byte("// renvo:linkstatic libc,__SCT__target"),
		[]byte("// renvo:object static-call __SCT__target .static_call.text 4 1 0 - 8 0 - 0"),
		[]byte("[8]uint8{195,204,144,144,144,15,185,204}"),
	} {
		if !bytes.Contains(lowered.Source, want) {
			t.Fatalf("static-call source is missing %q:\n%s", want, lowered.Source)
		}
	}
	if parsed := syntax.ParseFile(lowered.Source); !parsed.Ok {
		t.Fatalf("static-call source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, lowered.Source)
	}
}

func TestTranslateFileScopeStaticCallJumpTemplate(t *testing.T) {
	result := TranslateObject("main", []byte(`
extern void target(void);
extern void __SCT__target(void);
asm(".pushsection .static_call.text, \"ax\"\n"
    ".align 4\n"
    ".globl __SCT__target\n"
    "__SCT__target:\n"
    ".byte 0xe9; .long target - (. + 4)\n"
    ".byte 0x0f, 0xb9, 0xcc\n"
    ".type __SCT__target, @function\n"
    ".size __SCT__target, . - __SCT__target\n"
    ".popsection\n");
`), nil)
	if !result.Ok {
		t.Fatalf("static-call jump template failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("// renvo:object static-call __SCT__target .static_call.text 4 1 0 - 8 2 target 0"),
		[]byte("[8]uint8{233,0,0,0,0,15,185,204}"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("static-call jump source is missing %q:\n%s", want, result.Source)
		}
	}
}

func TestTranslateFileScopeStaticCallTrampKey(t *testing.T) {
	result := TranslateObject("main", []byte(`
extern void __SCT__target(void);
extern int __SCK__target;
asm(".pushsection .static_call_tramp_key, \"a\"\n"
    ".long __SCT__target - .\n"
    ".long __SCK__target - .\n"
    ".popsection\n");
`), nil)
	if !result.Ok {
		t.Fatalf("static-call trampoline key lowering failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte(".static_call_tramp_key 4 0 0 - 4 2 __SCT__target 0"),
		[]byte(".static_call_tramp_key 4 0 0 - 4 2 __SCK__target 0"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("static-call trampoline key source is missing %q:\n%s", want, result.Source)
		}
	}
}

func TestTranslateFileScopeNopFunctionTemplate(t *testing.T) {
	source := []byte(`
asm(".pushsection .entry.text, \"ax\"\n"
    ".global nop_func\n"
    ".type nop_func, @function\n"
    ".balign 16, 0x90;;\n"
    "nop_func:\n"
    "ret\n"
    ".size nop_func, . - nop_func\n"
    ".popsection");
extern typeof(nop_func) nop_func;
void *address(void) { return (void *)&nop_func; }
`)
	lowered := TranslateObject("main", source, nil)
	if !lowered.Ok {
		t.Fatalf("file-scope nop function template failed: %#v", lowered)
	}
	for _, want := range [][]byte{
		[]byte("// renvo:object function nop_func .entry.text 16 1 0 - 8 0 - 0"),
		[]byte("//export nop_func\nfunc nop_func(){}"),
		[]byte("return (*byte)(&(nop_func))"),
	} {
		if !bytes.Contains(lowered.Source, want) {
			t.Fatalf("file-scope nop function source is missing %q:\n%s", want, lowered.Source)
		}
	}
	if parsed := syntax.ParseFile(lowered.Source); !parsed.Ok {
		t.Fatalf("file-scope nop function source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, lowered.Source)
	}
}

func TestTranslateFileScopeReturnInt3FunctionTemplate(t *testing.T) {
	result := TranslateObject("main", []byte(`
extern void __static_call_return(void);
asm(".global __static_call_return\n\t"
    ".type __static_call_return, @function\n\t"
    ".balign 16, 0x90;;\n\t"
    "__static_call_return:\n\t"
    "998:\n\t.pushsection .discard.reachable\n\t.long 998b\n\t.popsection\n\t"
    "ret; int3\n\t"
    ".size __static_call_return, . - __static_call_return\n\t");
`), nil)
	if !result.Ok {
		t.Fatalf("file-scope return/int3 function lowering failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("// renvo:object function __static_call_return - 16 1 0 - 8 0 - 0"),
		[]byte("//export __static_call_return\nfunc __static_call_return(){}"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("return/int3 function source is missing %q:\n%s", want, result.Source)
		}
	}
}

func TestTranslateFileScopeInt3MagicFunctionTemplate(t *testing.T) {
	source := []byte(`
extern void int3_magic(unsigned int *ptr);
asm(".pushsection .init.text, \"ax\", @progbits\n"
    ".type int3_magic, @function\n"
    "int3_magic:\n"
    "movl $1, (%rdi)\n"
    "ret\n"
    ".size int3_magic, .-int3_magic\n"
    ".popsection");
void call_magic(unsigned int *ptr) { int3_magic(ptr); }
`)
	lowered := TranslateObject("main", source, nil)
	if !lowered.Ok {
		t.Fatalf("file-scope int3 helper template failed: %#v", lowered)
	}
	for _, want := range [][]byte{
		[]byte("// renvo:object function int3_magic .init.text 0 0 0 - 8 0 - 0"),
		[]byte("func int3_magic(ptr *uint32){*ptr=1}"),
		[]byte("func call_magic(ptr *uint32) {int3_magic(ptr);"),
	} {
		if !bytes.Contains(lowered.Source, want) {
			t.Fatalf("file-scope int3 helper source is missing %q:\n%s", want, lowered.Source)
		}
	}
	if parsed := syntax.ParseFile(lowered.Source); !parsed.Ok {
		t.Fatalf("file-scope int3 helper source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, lowered.Source)
	}
}

func TestCheckVisibilitySymbolAttribute(t *testing.T) {
	source := []byte(`int hidden __attribute__((visibility("hidden")));`)
	checked := CheckObjectForDataModel(source, DataModelLP64)
	if !checked.Ok {
		t.Fatalf("visibility attribute check failed: error=%d at=%d", checked.Error, checked.ErrorAt)
	}
	lowered := TranslateObject("main", source, nil)
	if !lowered.Ok {
		t.Fatalf("visibility lowering failed: %#v", lowered)
	}
	if !bytes.Contains(lowered.Source, []byte("// renvo:object variable hidden - 4 1 2 - 4 0 - 0")) {
		t.Fatalf("visibility lowering is missing object metadata:\n%s", lowered.Source)
	}
}

func TestTranslateObjectDefinitionInheritsWeakTentativeDeclaration(t *testing.T) {
	source := []byte(`
struct value { long member; };
struct value shared __attribute__((weak));
struct value shared = { 7 };
`)
	lowered := TranslateObject("main", source, nil)
	if !lowered.Ok {
		t.Fatalf("weak tentative declaration lowering failed: %#v", lowered)
	}
	if !bytes.Contains(lowered.Source, []byte("// renvo:object variable shared - 8 2 0 - 8 0 - 0")) {
		t.Fatalf("definition did not inherit weak binding:\n%s", lowered.Source)
	}
}

func TestTranslateObjectPlacementPolicy(t *testing.T) {
	source := []byte(`
static int local;
const int constant = 3;
int leaf(void) { return local + constant; }
`)
	plain := TranslateObjectWithConfig("main", source, nil, ObjectConfig{DataModel: DataModelLP64})
	if !plain.Ok {
		t.Fatalf("plain object lowering failed: %#v", plain)
	}
	for _, unwanted := range [][]byte{[]byte(".text.leaf"), []byte(".bss.local"), []byte(".rodata.constant")} {
		if bytes.Contains(plain.Source, unwanted) {
			t.Fatalf("plain object unexpectedly split %q:\n%s", unwanted, plain.Source)
		}
	}
	split := TranslateObjectWithConfig("main", source, nil, ObjectConfig{
		DataModel: DataModelLP64, FunctionSections: true, DataSections: true,
	})
	if !split.Ok {
		t.Fatalf("split object lowering failed: %#v", split)
	}
	for _, wanted := range [][]byte{[]byte(".text.leaf"), []byte(".bss.local"), []byte(".rodata.constant")} {
		if !bytes.Contains(split.Source, wanted) {
			t.Fatalf("split object is missing %q:\n%s", wanted, split.Source)
		}
	}
}

func TestTranslateObjectRetainsUnusedInt128Types(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef __signed__ __int128 s128 __attribute__((aligned(16)));
typedef unsigned __int128 u128 __attribute__((aligned(16)));
unsigned long leaf(unsigned long value) { return value + 1; }
`), nil)
	if !result.Ok {
		t.Fatalf("unused int128 carrier lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("func leaf(value uint64) uint64")) {
		t.Fatalf("unexpected int128 carrier unit:\n%s", result.Source)
	}
}

func TestCheckCallingConventionAttribute(t *testing.T) {
	source := []byte(`
typedef unsigned long firmware_call(unsigned int command);
struct services { firmware_call __attribute__((ms_abi)) *call; };
unsigned long invoke(struct services *services, unsigned int command) {
	return services->call(command);
}
`)
	checked := CheckObjectForDataModel(source, DataModelLP64)
	if !checked.Ok {
		t.Fatalf("calling-convention attribute check failed: error=%d at=%d", checked.Error, checked.ErrorAt)
	}
	lowered := TranslateObject("main", source, nil)
	if !lowered.Ok {
		t.Fatalf("calling-convention lowering failed: error=%d at=%d", lowered.Error, lowered.ErrorAt)
	}
	if !bytes.Contains(lowered.Source, []byte("return renvo_runtime_CMSABICall_")) {
		t.Fatalf("calling-convention lowering did not retain the Microsoft ABI call:\n%s", lowered.Source)
	}
	parsed := syntax.ParseFile(lowered.Source)
	if !parsed.Ok {
		t.Fatalf("translated calling-convention source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, lowered.Source)
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
	if !lowered.Ok {
		t.Fatalf("GNU asm lowering failed: error=%d at=%d", lowered.Error, lowered.ErrorAt)
	}
	if !bytes.Contains(lowered.Source, []byte("return pointer")) {
		t.Fatalf("GNU asm identity did not preserve its operand:\n%s", lowered.Source)
	}
	malformed := CheckObjectForDataModel([]byte(`int inspect(void) { asm("broken"; }`), DataModelLP64)
	if malformed.Ok {
		t.Fatalf("malformed GNU asm check = %#v, want failure", malformed)
	}
}

func TestTranslateObjectAsmClobberOperations(t *testing.T) {
	result := TranslateObject("main", []byte(`
void barrier(void) { asm volatile("" : : : "memory", "cc"); }
`), nil)
	if !result.Ok {
		t.Fatalf("asm clobber lowering failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("renvo_runtime_CMemoryBarrier();"),
		[]byte("renvo_runtime_CConditionClobber();"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("asm clobber source is missing %q:\n%s", want, result.Source)
		}
	}
}

func TestTranslateObjectStackRegisterDeclaration(t *testing.T) {
	result := TranslateObject("main", []byte(`
register unsigned long current_stack_pointer asm("rsp");
unsigned long stack(void) { return current_stack_pointer; }
`), nil)
	if !result.Ok {
		t.Fatalf("stack-register lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("return renvo_runtime_CReadStackPointer()")) {
		t.Fatalf("stack-register read is not explicit in the shared stream:\n%s", result.Source)
	}
}

func TestTranslateObjectAsmConditionCodeBitTest(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef unsigned int u32;
typedef _Bool bool;
bool variable_test_bit(int nr, const void *addr) {
	bool value;
	const u32 *words = addr;
	asm("btl %2,%1\n\t/* output condition code c*/\n" : "=@ccc"(value) : "m"(*words), "Ir"(nr));
	asm("btl %[nr], %[var]\n\t/* output condition code " "c" "*/\n" : "=@cc" "c"(value) : [var] "m"(*words), [nr] "rI"(nr));
	u32 array[16];
	asm("btl %[nr], %[var]\n\t/* output condition code " "c" "*/\n" : "=@cc" "c"(value) : [var] "m"((*(typeof(*(&array)) *)(unsigned long)(&array))), [nr] "rI"(nr));
	return value;
}
`), nil)
	if !result.Ok {
		t.Fatalf("condition-code asm lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("(*__c_pointer_index_")) ||
		!bytes.Contains(result.Source, []byte("words,uintptr(int64(nr)>>5)")) {
		t.Fatalf("condition-code asm did not become a typed bit test:\n%s", result.Source)
	}
}

func TestTranslateObjectAsmSMAPAccessAndFlags(t *testing.T) {
	result := TranslateObject("main", []byte(`
void disable_access(void) { asm volatile("# alternative\n .byte 0x0f,0x01,0xca" : : : "memory"); }
void enable_access(void) { asm volatile("# alternative\n .byte 0x0f,0x01,0xcb" : : : "memory"); }
unsigned long save_access(void) {
	unsigned long flags;
	asm volatile("pushf; pop %0; .byte 0x0f,0x01,0xca" : "=rm"(flags) : : "memory", "cc");
	return flags;
}
void restore_access(unsigned long flags) { asm volatile("push %0; popf" : : "g"(flags) : "memory", "cc"); }
`), nil)
	if !result.Ok {
		t.Fatalf("SMAP assembler translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("renvo_runtime_CDisableUserAccess();"),
		[]byte("renvo_runtime_CEnableUserAccess();"),
		[]byte("flags=renvo_runtime_CReadFlags();renvo_runtime_CDisableUserAccess();"),
		[]byte("renvo_runtime_CWriteFlags(flags);"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("SMAP assembler source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("SMAP assembler source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectAsmInvalidateProcessContext(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct descriptor { unsigned long words[2]; };
void invalidate(unsigned long type) {
	struct descriptor desc = {{3, 4}};
	asm volatile("invpcid %[desc], %[type]" :: [desc] "m"(desc), [type] "r"(type) : "memory");
}
`), nil)
	if !result.Ok {
		t.Fatalf("INVPCID assembler translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("renvo_runtime_CInvalidateProcessContext(uintptr(&(desc)),uintptr(__c_name_type));")) {
		t.Fatalf("INVPCID assembler was not lowered with its descriptor and type operands:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("INVPCID assembler source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectAsmInvalidatePage(t *testing.T) {
	result := TranslateObject("main", []byte(`
void invalidate_page(unsigned long address) {
	asm volatile("invlpg (%0)" :: "r" (address) : "memory");
}
`), nil)
	if !result.Ok {
		t.Fatalf("INVLPG assembler translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("renvo_runtime_CInvalidatePage(uintptr(address));")) {
		t.Fatalf("INVLPG assembler lost its address operand:\n%s", result.Source)
	}
}

func TestTranslateObjectAsmClampAddress(t *testing.T) {
	result := TranslateObject("main", []byte(`
void *clamp(void *pointer, unsigned long limit) {
	void *result;
	asm("cmp %1,%0\n\tcmova %1,%0" : "=r"(result) : "r"(limit), "0"(pointer));
	return result;
}
`), nil)
	if !result.Ok {
		t.Fatalf("conditional-move clamp translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("func __c_clamp_address_"),
		[]byte("if uintptr(__c_unsafe.Pointer(value))>limit"),
		[]byte("pointer,uintptr(limit)"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("conditional-move clamp source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("conditional-move clamp source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectAsmRuntimePointer(t *testing.T) {
	result := TranslateObject("main", []byte(`
extern unsigned long USER_PTR_MAX;
unsigned long runtime_limit(void) {
	unsigned long result;
	asm("mov %1,%0\n1:\n.pushsection runtime_ptr_USER_PTR_MAX,\"a\"\n\t.long 1b - %c2 - .\n.popsection"
		: "=r"(result) : "i"((unsigned long)0x12345678), "i"(sizeof(long)));
	return result;
}
unsigned long runtime_hash(unsigned long value) {
	asm("shrl $12,%k0\n1:\n.pushsection runtime_shift_d_hash_shift,\"a\"\n\t.long 1b - 1 - .\n.popsection" : "+r"(value));
	return value;
}
struct item;
struct item *runtime_table(void) {
	struct item *result;
	asm("mov %1,%0\n1:\n.pushsection runtime_ptr_table,\"a\"\n\t.long 1b - %c2 - .\n.popsection"
		: "=r"(result) : "i"((unsigned long)0x12345678), "i"(sizeof(long)));
	return result;
}
`), nil)
	if !result.Ok {
		t.Fatalf("runtime-pointer assembler translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("func renvo_runtime_CRuntimePointer_runtime_ptr_USER_PTR_MAX"),
		[]byte("result=uint64(renvo_runtime_CRuntimePointer_runtime_ptr_USER_PTR_MAX(uintptr(uint64(0x12345678))))"),
		[]byte("func renvo_runtime_CRuntimeShift_runtime_shift_d_hash_shift"),
		[]byte("value=renvo_runtime_CRuntimeShift_runtime_shift_d_hash_shift(value)"),
		[]byte("result=(*__c_struct_item)(__c_unsafe.Pointer(renvo_runtime_CRuntimePointer_runtime_ptr_table"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("runtime-pointer assembler source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("runtime-pointer assembler source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectAsmVideoTextFill(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef unsigned short u16;
typedef unsigned int u32;
void fill(u32 dst, int count, u16 segment) {
	asm volatile("pushw %%es ; movw %2,%%es ; shrw %%cx ; jnc 1f ; stosw \n\t1: rep;stosl ; popw %%es"
		: "+D" (dst), "+c" (count) : "bdS" (segment), "a" (0x07200720));
}
`), nil)
	if !result.Ok {
		t.Fatalf("video text fill asm lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("renvo_runtime_CWriteSegmentfs16")) ||
		!bytes.Contains(result.Source, []byte("for count>0")) {
		t.Fatalf("video text fill asm was not lowered to typed segment writes:\n%s", result.Source)
	}
}

func TestTranslateObjectAsmRepCopy(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef unsigned long size_t;
void copy(void *dest, const void *src, size_t n) {
	long d0, d1, d2;
	asm volatile("rep ; movsq\n\tmovq %4,%%rcx\n\trep ; movsb\n\t"
		: "=&c" (d0), "=&D" (d1), "=&S" (d2)
		: "0" (n >> 3), "g" (n & 7), "1" (dest), "2" (src) : "memory");
}
`), nil)
	if !result.Ok {
		t.Fatalf("rep copy asm lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("renvo_runtime_CCopyBytes")) ||
		!bytes.Contains(result.Source, []byte("__c_asm_count_")) {
		t.Fatalf("rep copy asm was not lowered to a typed copy operation:\n%s", result.Source)
	}
}

func TestTranslateObjectAsmRepCopyWithTail(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef unsigned long size_t;
void copy(void *to, const void *from, size_t n) {
	unsigned long d0, d1, d2;
	asm volatile("rep ; movsl\n\t"
	             "testb $2,%b4\n\t"
	             "je 1f\n\t"
	             "movsw\n"
	             "1:\ttestb $1,%b4\n\t"
	             "je 2f\n\t"
	             "movsb\n"
	             "2:"
	             : "=&c"(d0), "=&D"(d1), "=&S"(d2)
	             : "0"(n / 4), "q"(n), "1"((long)to), "2"((long)from)
	             : "memory");
}
`), nil)
	if !result.Ok {
		t.Fatalf("rep copy with byte tail failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("renvo_runtime_CCopyBytes"),
		[]byte("__c_asm_count_1:=uintptr(n)"),
		[]byte("d1=uint64(__c_asm_dest_1+__c_asm_count_1)"),
		[]byte("d2=uint64(__c_asm_src_1+__c_asm_count_1)"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("rep copy with byte tail is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("rep-copy-tail source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectAsmMoveString(t *testing.T) {
	result := TranslateObject("main", []byte(`
void copy_byte(void *to, const void *from) {
	asm volatile("movsb" : "=&D"(to), "=&S"(from) : "0"(to), "1"(from) : "memory");
}
void copy_word(void *to, const void *from) {
	asm volatile("movsw" : "=&D"(to), "=&S"(from) : "0"(to), "1"(from) : "memory");
}
`), nil)
	if !result.Ok {
		t.Fatalf("move-string lowering failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("renvo_runtime_CCopyBytes((*byte)(__c_unsafe.Pointer(__c_asm_dest_1)),(*byte)(__c_unsafe.Pointer(__c_asm_src_1)),1)"),
		[]byte("renvo_runtime_CCopyBytes((*byte)(__c_unsafe.Pointer(__c_asm_dest_2)),(*byte)(__c_unsafe.Pointer(__c_asm_src_2)),2)"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("move-string source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("move-string source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectAsmClearUser(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned long clear_user(void *address, unsigned long size, unsigned long stack) {
	asm volatile("1:\n\trep stosb\n2:\n"
	             ".pushsection \"__ex_table\",\"a\"\n"
	             ".long (1b) - .\n.long (2b) - .\n.long 3\n.popsection\n"
	             : "+c"(size), "+D"(address), "+r"(stack) : "a"(0));
	return size;
}
`), nil)
	if !result.Ok {
		t.Fatalf("clear-user lowering failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("renvo_runtime_CClearUserBytes___ex_table"),
		[]byte("address=(*byte)(__c_unsafe.Pointer(uintptr(__c_unsafe.Pointer(__c_clear_dest_1))+__c_clear_done_1));"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("clear-user source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("clear-user source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectAsmByteCompare(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef _Bool bool;
typedef unsigned long size_t;
int compare(const void *left, const void *right, size_t count) {
	bool different;
	asm("repe; cmpsb\n\t/* output condition code nz*/\n"
		: "=@ccnz"(different), "+D"(left), "+S"(right), "+c"(count));
	return different;
}
`), nil)
	if !result.Ok {
		t.Fatalf("byte-compare asm lowering failed: %#v", result)
	}
	for _, want := range [][]byte{[]byte("for __c_asm_n_"), []byte("__c_asm_diff"), []byte("__c_unsafe.Pointer")} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("byte-compare asm source is missing %q:\n%s", want, result.Source)
		}
	}
}

func TestTranslateObjectAsmDivideAndBitSet(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef unsigned int u32;
void divide(u32 divisor, u32 upper, u32 low, u32 *remainder) {
	asm("divl %2" : "=a"(low), "=d"(*remainder) : "rm"(divisor), "0"(low), "1"(upper));
}
void set_bit(int bit, void *address) {
	asm("btsl %1,%0" : "+m"(*(u32 *)address) : "Ir"(bit));
}
`), nil)
	if !result.Ok {
		t.Fatalf("divide/bit-set asm lowering failed: %#v", result)
	}
	for _, want := range [][]byte{[]byte("func renvo_runtime_CDivide32"), []byte("low=renvo_runtime_CDivide32(divisor,low,upper,remainder)"), []byte("renvo_runtime_CBitSet32")} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("divide/bit-set asm source is missing %q:\n%s", want, result.Source)
		}
	}
}

func TestTranslateObjectAsmByteAndBitOperations64(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef _Bool bool;
void set_constant(char *address, int mask) {
	asm volatile("orb %b1,%0" : "+m"(*address) : "iq"(mask) : "memory");
}
bool test_and_clear(unsigned long *address, long bit) {
	bool old;
	asm volatile("btrq %2,%1\n\t/* output condition code c*/\n"
		: "=@ccc"(old) : "m"(*address), "Ir"(bit) : "memory");
	return old;
}
bool xor_negative(char *address, int mask) {
	bool negative;
	asm volatile("xorb %2,%1\n\t/* output condition code s*/\n"
		: "=@ccs"(negative), "+m"(*address) : "iq"(mask) : "memory");
	return negative;
}
bool test_byte(unsigned char *address, int mask) {
	bool nonzero;
	asm volatile("testb %2,%1\n\t/* output condition code nz*/\n"
		: "=@ccnz"(nonzero) : "m"(*address), "i"(mask) : "memory");
	return nonzero;
}
`), nil)
	if !result.Ok {
		t.Fatalf("byte/64-bit asm lowering failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("renvo_runtime_COrByte"),
		[]byte("negative=renvo_runtime_CXorByteNegative"),
		[]byte("nonzero=renvo_runtime_CTestByte"),
		[]byte("old=renvo_runtime_CBitReset64"),
		[]byte("renvo_runtime_CMemoryBarrier();"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("byte/64-bit asm source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("byte/64-bit asm source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectAsmBitScan(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned long first(unsigned long word) {
	asm("rep; bsf %1,%0" : "=r"(word) : "rm"(word));
	return word;
}
int last(unsigned int word) {
	int result;
	asm("bsrl %1,%0" : "=r"(result) : "rm"(word), "0"(-1));
	return result + 1;
}
`), nil)
	if !result.Ok {
		t.Fatalf("bit-scan asm lowering failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("renvo_runtime_CBitScanForward64"),
		[]byte("renvo_runtime_CBitScanReverse32"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("bit-scan asm source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("bit-scan asm source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectRuntimeBitBuiltins(t *testing.T) {
	result := TranslateObject("main", []byte(`
int inspect(unsigned long value) {
	return __builtin_clzl(value) + __builtin_ctzl(value) + __builtin_popcountl(value) + __builtin_ffsl(value) +
		__builtin_bswap64(value);
}
`), nil)
	if !result.Ok {
		t.Fatalf("runtime bit builtin lowering failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("__c_builtin_leading_64"),
		[]byte("__c_builtin_trailing_64"),
		[]byte("__c_builtin_population_64"),
		[]byte("__c_builtin_first_64"),
		[]byte("__c_byte_swap_64"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("runtime bit builtin source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("runtime bit builtin source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectBuiltinIsDigit(t *testing.T) {
	result := TranslateObject("main", []byte(`
int classify(const char *text) {
	return __builtin_isdigit(*text++) + __builtin_isdigit(*text);
}
`), nil)
	if !result.Ok {
		t.Fatalf("isdigit builtin lowering failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("func __c_builtin_isdigit(value int32) int32{if value>=48&&value<=57{return 1};return 0}"),
		[]byte("__c_builtin_isdigit("),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("isdigit builtin source is missing %q:\n%s", want, result.Source)
		}
	}
	if bytes.Count(result.Source, []byte("func __c_builtin_isdigit")) != 1 {
		t.Fatalf("isdigit builtin helper was not shared:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("isdigit builtin source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectExpandsInitializedFlexibleArrayMember(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct namespace { long marker; };
struct upid { int nr; struct namespace *ns; };
struct pid { int level; struct upid numbers[]; };
extern struct pid initial_pid;
struct namespace initial_namespace;
struct pid initial_pid = {
	.level = 2,
	.numbers = { { .nr = 7, .ns = &initial_namespace }, },
};
`), nil)
	if !result.Ok {
		t.Fatalf("flexible-array object initializer failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("type __c_struct_flexible_"),
		[]byte("// renvo:object variable initial_pid - 8 1 0 - 24 0 - 0"),
		[]byte("__c_object_aggregate_0_8_x"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("flexible-array object source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("flexible-array object source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectInitializesPositionalAnonymousUnion(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct extent { unsigned first; unsigned count; };
struct map {
	union {
		struct { struct extent extent[2]; unsigned nr_extents; };
		struct { struct extent *forward; struct extent *reverse; };
	};
};
struct holder { struct map map; };
struct holder initial = {
	.map = { { .extent[0] = { .first = 4, .count = 8 }, .nr_extents = 1 } },
};
`), nil)
	if !result.Ok {
		t.Fatalf("anonymous-union object initializer failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("__c_object_aggregate_0_16_x")) {
		t.Fatalf("anonymous-union initializer did not retain promoted field offsets:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("anonymous-union object source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectInitializesLocalAnonymousStructUnion(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef unsigned __int128 full_t;
typedef union {
	struct { void *freelist; unsigned long counter; };
	full_t full;
} aba_t;
unsigned long inspect(void *freelist, unsigned long counter) {
	aba_t value = { .freelist = freelist, .counter = counter };
	return value.counter;
}
`), nil)
	if !result.Ok {
		t.Fatalf("local anonymous-struct union initializer failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("__c_object_aggregate_0_8_x")) {
		t.Fatalf("local anonymous-struct union initializer did not retain promoted offsets:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("local anonymous-struct union source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectUsesIndirectLayoutForZeroSizedField(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct empty {};
struct inner { struct empty lock; int value; };
struct outer { int head; struct inner inner; int tail; };
struct outer initial = { .head = 1, .inner = { .value = 2 }, .tail = 3 };
`), nil)
	if !result.Ok {
		t.Fatalf("zero-sized-field object initializer failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("type __c_struct_inner struct{__c_align uint32}"),
		[]byte("// renvo:object variable initial - 4 1 0 - 12 0 - 0"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("zero-sized-field source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("zero-sized-field source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectSizeBuiltins(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned long unknown(void *pointer) { return __builtin_object_size(pointer, 0); }
unsigned long minimum(void *pointer) { return __builtin_dynamic_object_size(pointer, 2); }
unsigned long known(void) { char bytes[12]; return __builtin_object_size(&bytes[3], 1); }
`), nil)
	if !result.Ok {
		t.Fatalf("object-size builtin lowering failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("return uint64(^uint64(0));"),
		[]byte("return uint64(0);"),
		[]byte("return uint64(9);"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("object-size builtin source is missing %q:\n%s", want, result.Source)
		}
	}
	if bytes.Contains(result.Source, []byte("__builtin_object_size")) || bytes.Contains(result.Source, []byte("__builtin_dynamic_object_size")) {
		t.Fatalf("object-size builtin leaked into shared syntax:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("object-size builtin source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectAsmAlternativePopulationCount(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned int weight(unsigned int value) {
	unsigned int result;
	asm("call fallback\n.popcntl %1, %0" : "=a"(result) : "D"(value));
	return result;
}
`), nil)
	if !result.Ok {
		t.Fatalf("population-count asm lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("__c_population_count_32")) ||
		!bytes.Contains(result.Source, []byte("for value!=0{count+=value&1;value>>=1}")) {
		t.Fatalf("population-count asm did not become a baseline-safe helper:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("population-count asm source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectAsmAlternativeCallUsesBaseline(t *testing.T) {
	result := TranslateObject("main", []byte(`
register unsigned long stack_pointer asm("rsp");
void baseline(void *page);
void replacement(void *page);
void clear(void *page) {
	asm volatile("call %c[old]\n replacement: call %c[new]"
		: "+r"(stack_pointer), "=D"(page)
		: [old] "i"(baseline), [new] "i"(replacement), "D"(page)
		: "cc", "memory", "rax", "rcx");
}
`), nil)
	if !result.Ok {
		t.Fatalf("alternative-call asm lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("baseline(page);")) || bytes.Contains(result.Source, []byte("replacement(page)")) {
		t.Fatalf("alternative-call asm did not select the architectural baseline:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("alternative-call asm source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectAsmGotoCPUFeature(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef _Bool bool;
typedef unsigned short u16;
unsigned char capability[4];
bool static_cpu_has(u16 bit) {
	asm goto("jmp 6f\n.pushsection .altinstr_aux,\"ax\"\n6:\n"
		"testb %[bitnum], %a[cap_byte]\n"
		"jnz %l[t_yes]\n"
		"jmp %l[t_no]\n.popsection"
		: : [feature] "i"(bit), [bitnum] "i"(1 << (bit & 7)),
		[cap_byte] "i"(&capability[bit >> 3]) : : t_yes, t_no);
t_yes:
	return 1;
t_no:
	return 0;
}
`), nil)
	if !result.Ok {
		t.Fatalf("CPU-feature asm goto lowering failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("_=bit;"),
		[]byte("if (uint8(*(&(capability[int32(bit)>>3])))&uint8(1<<(int32(bit)&7)))!=0{goto t_yes};goto t_no;"),
		[]byte("t_yes:"),
		[]byte("t_no:"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("CPU-feature asm goto source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("CPU-feature asm goto source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectAsmInstructionPointer(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned long caller_ip(void) {
	unsigned long here;
	asm("lea 0(%%rip), %0" : "=r"(here));
	return here;
}
`), nil)
	if !result.Ok {
		t.Fatalf("instruction-pointer asm lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("here=renvo_runtime_CReadInstructionPointer();")) {
		t.Fatalf("instruction-pointer asm did not become a typed backend operation:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("instruction-pointer asm source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectAsmByteSwap(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned int swap32(unsigned int value) { asm("bswapl %0" : "=r"(value) : "0"(value)); return value; }
unsigned long swap64(unsigned long value) { asm("bswapq %0" : "=r"(value) : "0"(value)); return value; }
`), nil)
	if !result.Ok {
		t.Fatalf("byte-swap asm lowering failed: %#v", result)
	}
	for _, want := range [][]byte{[]byte("__c_byte_swap_32"), []byte("__c_byte_swap_64")} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("byte-swap asm source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("byte-swap asm source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectAsmMultiplyDivide64(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned long scale(unsigned long value, unsigned long multiplier, unsigned long divisor) {
	unsigned long quotient;
	asm("mulq %2; divq %3" : "=a"(quotient) : "a"(value), "rm"(multiplier), "rm"(divisor) : "rdx");
	return quotient;
}
`), nil)
	if !result.Ok {
		t.Fatalf("multiply/divide asm lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("renvo_runtime_CMultiplyDivide64")) {
		t.Fatalf("multiply/divide asm did not become a typed backend operation:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("multiply/divide asm source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectAsmMultiplyShift32(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef unsigned long u64;
typedef unsigned int u32;
u64 scale(u64 delta, u32 fraction) {
	u64 product, high;
	asm("mulq %[mul_frac] ; shrd $32, %[hi], %[lo]"
		: [lo] "=a"(product), [hi] "=d"(high)
		: "0"(delta), [mul_frac] "rm"((u64)fraction));
	return product + (high == 0);
}
`), nil)
	if !result.Ok {
		t.Fatalf("multiply/shift asm lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("product=renvo_runtime_CMultiplyShift32")) ||
		!bytes.Contains(result.Source, []byte(",&(high))")) {
		t.Fatalf("multiply/shift asm did not preserve both outputs:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("multiply/shift asm source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectAsmCompareExchange128(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef unsigned __int128 u128;
typedef unsigned long u64;
typedef _Bool bool;
union halves { u128 full; struct { u64 low, high; }; };
bool compare_exchange(volatile u128 *address, u128 old, u128 replacement) {
	union halves observed = { .full = old }, next = { .full = replacement };
	bool equal;
	asm volatile("cmpxchg16b %[ptr]\n\t/* output condition code e*/\n"
		: "=@cce"(equal), [ptr] "+m"(*address), "+a"(observed.low), "+d"(observed.high)
		: "b"(next.low), "c"(next.high) : "memory");
	return equal;
}
`), nil)
	if !result.Ok {
		t.Fatalf("cmpxchg128 asm lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("equal=renvo_runtime_CCompareExchange128")) ||
		!bytes.Contains(result.Source, []byte("__c_ptr_0_low")) || !bytes.Contains(result.Source, []byte("__c_ptr_8_high")) {
		t.Fatalf("cmpxchg128 asm did not preserve outputs:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("cmpxchg128 asm source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectAsmAtomicArithmetic(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef _Bool bool;
void add(int *value, int change) { asm volatile("addl %1,%0" : "+m"(*value) : "ir"(change) : "memory"); }
bool subtract_zero(int *value, int change) {
	bool zero;
	asm volatile("subl %[change],%[value]\n/* output condition code e*/"
		: [value] "+m"(*value), "=@cce"(zero) : [change] "er"(change) : "memory");
	return zero;
}
bool increment_zero(long *value) {
	bool zero;
	asm volatile("incq %[value]\n/* output condition code e*/"
		: [value] "+m"(*value), "=@cce"(zero) : : "memory");
	return zero;
}
`), nil)
	if !result.Ok {
		t.Fatalf("atomic arithmetic asm lowering failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("renvo_runtime_CAtomicAdd32"),
		[]byte("zero=uint8((renvo_runtime_CAtomicSub32"),
		[]byte("zero=uint8((renvo_runtime_CAtomicInc64"),
		[]byte("renvo_runtime_CMemoryBarrier();"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("atomic arithmetic asm source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("atomic arithmetic asm source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectAsmPerCPUArithmeticWidths(t *testing.T) {
	result := TranslateObject("main", []byte(`
void add_byte(unsigned long *value, unsigned char change) { asm volatile("addb %[val], %[var]" : [var] "+m"(*value) : [val] "qi"(change)); }
void add_word(unsigned long *value, unsigned short change) { asm volatile("addw %[val], %[var]" : [var] "+m"(*value) : [val] "ri"(change)); }
void increment_byte(unsigned long *value) { asm volatile("incb %[var]" : [var] "+m"(*value)); }
void decrement_word(unsigned long *value) { asm volatile("decw %[var]" : [var] "+m"(*value)); }
`), nil)
	if !result.Ok {
		t.Fatalf("per-CPU arithmetic lowering failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("renvo_runtime_CAtomicAdd8"), []byte("renvo_runtime_CAtomicAdd16"),
		[]byte("renvo_runtime_CAtomicInc8"), []byte("renvo_runtime_CAtomicDec16"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("per-CPU arithmetic source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("per-CPU arithmetic source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectAsmPositionalScalarMemory(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned char readb(const volatile void *address) {
	unsigned char value;
	asm volatile("movb %1,%0" : "=q"(value) : "m"(*(volatile unsigned char *)address) : "memory");
	return value;
}
void writeq(unsigned long value, volatile void *address) {
	asm volatile("movq %0,%1" : : "r"(value), "m"(*(volatile unsigned long *)address) : "memory");
}
`), nil)
	if !result.Ok {
		t.Fatalf("positional scalar memory lowering failed: %#v", result)
	}
	if bytes.Contains(result.Source, []byte("asm")) || bytes.Count(result.Source, []byte("renvo_runtime_CMemoryBarrier")) < 2 {
		t.Fatalf("positional scalar memory operations were not lowered:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("positional scalar memory source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectAsmStringPortIO(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef unsigned short u16;
typedef unsigned long size_t;
void output_bytes(u16 port, const void *address, size_t count) {
	asm volatile("rep; outsb" : "+S"(address), "+c"(count) : "d"(port) : "memory");
}
void input_words(u16 port, void *address, size_t count) {
	asm volatile("rep; insw" : "+D"(address), "+c"(count) : "d"(port) : "memory");
}
`), nil)
	if !result.Ok {
		t.Fatalf("string port I/O lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("renvo_runtime_CPortOutString8")) ||
		!bytes.Contains(result.Source, []byte("renvo_runtime_CPortInString16")) {
		t.Fatalf("string port I/O operations were not lowered:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("string port I/O source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectAsmVDSoSyscall(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef long clockid_t;
struct timespec { long seconds; long nanos; };
long fallback(clockid_t clock, struct timespec *result) {
	long ret;
	asm("syscall" : "=a"(ret), "=m"(*result) : "0"(228), "D"(clock), "S"(result) : "rcx", "r11");
	return ret;
}
`), nil)
	if !result.Ok {
		t.Fatalf("vDSO syscall lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("ret=int64(renvo_runtime_CVDSOSyscall")) {
		t.Fatalf("vDSO syscall was not lowered:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("vDSO syscall source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectAsmAtomicExchangeWidths(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned char fetch_add_byte(unsigned char *pointer, unsigned char value) {
	asm volatile("xaddb %b0, %1" : "+q"(value), "+m"(*pointer) : : "memory", "cc");
	return value;
}
unsigned short fetch_add_word(unsigned short *pointer, unsigned short value) {
	asm volatile("xaddw %w0, %1" : "+r"(value), "+m"(*pointer) : : "memory", "cc");
	return value;
}
unsigned int exchange_long(unsigned int *pointer, unsigned int value) {
	asm volatile("xchgl %0, %1" : "+r"(value), "+m"(*pointer) : : "memory", "cc");
	return value;
}
void *exchange_pointer(void **pointer, void *value) {
	asm volatile("xchgq %q0, %1" : "+r"(value), "+m"(*pointer) : : "memory", "cc");
	return value;
}
`), nil)
	if !result.Ok {
		t.Fatalf("atomic exchange lowering failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("renvo_runtime_CAtomicFetchAdd8"), []byte("renvo_runtime_CAtomicFetchAdd16"),
		[]byte("renvo_runtime_CAtomicExchange32"), []byte("renvo_runtime_CAtomicExchange64"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("atomic exchange source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("atomic exchange source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectAsmRepeatedStore(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef unsigned long size_t;
void fill(unsigned int *pointer, unsigned int value, size_t count) {
	asm volatile("rep stosl" : "+D"(pointer), "+c"(count) : "a"(value) : "memory");
}
`), nil)
	if !result.Ok {
		t.Fatalf("repeated-store asm lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("for count>0{*(pointer)=value")) ||
		!bytes.Contains(result.Source, []byte("__c_pointer_step_inc_")) {
		t.Fatalf("repeated-store asm did not preserve pointer/count updates:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("repeated-store asm source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectAsmRepeatedCopy(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef unsigned long size_t;
void copy_words(void *to, const void *from, size_t count) {
	asm volatile("rep ; movsl" : "=&c"(count), "=&D"(to), "=&S"(from) : "0"(count), "1"(to), "2"(from) : "memory");
}
`), nil)
	if !result.Ok {
		t.Fatalf("repeated copy lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("renvo_runtime_CCopyBytes")) ||
		!bytes.Contains(result.Source, []byte("__c_asm_count_1:=uintptr(count)*4")) {
		t.Fatalf("repeated copy did not preserve byte count and tied outputs:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("repeated copy source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectAsmNonTemporalStore(t *testing.T) {
	result := TranslateObject("main", []byte(`
void store32(unsigned int *address, unsigned int value) { asm("movntil %1,%0" : "=m"(*address) : "r"(value)); }
void store64(unsigned long *address, unsigned long value) { asm("movntiq %1,%0" : "=m"(*address) : "r"(value)); }
`), nil)
	if !result.Ok {
		t.Fatalf("non-temporal store asm lowering failed: %#v", result)
	}
	for _, want := range [][]byte{[]byte("renvo_runtime_CNonTemporalStore32"), []byte("renvo_runtime_CNonTemporalStore64")} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("non-temporal store asm source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("non-temporal store asm source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectAsmNonTemporalCopy(t *testing.T) {
	result := TranslateObject("main", []byte(`
void copy32(unsigned long source, unsigned long destination) {
	asm("movq (%0), %%r8\nmovq 8(%0), %%r9\nmovq 16(%0), %%r10\nmovq 24(%0), %%r11\n"
	    "movnti %%r8, (%1)\nmovnti %%r9, 8(%1)\nmovnti %%r10, 16(%1)\nmovnti %%r11, 24(%1)"
	    :: "r"(source), "r"(destination) : "memory", "r8", "r9", "r10", "r11");
}
void copy8(unsigned long source, unsigned long destination) {
	asm("movq (%0), %%r8\nmovnti %%r8, (%1)" :: "r"(source), "r"(destination) : "memory", "r8");
}
void copy4(unsigned long source, unsigned long destination) {
	asm("movl (%0), %%r8d\nmovnti %%r8d, (%1)" :: "r"(source), "r"(destination) : "memory", "r8");
}
`), nil)
	if !result.Ok {
		t.Fatalf("non-temporal copy lowering failed: %#v", result)
	}
	if bytes.Count(result.Source, []byte("renvo_runtime_CNonTemporalStore64(")) < 6 ||
		!bytes.Contains(result.Source, []byte("renvo_runtime_CNonTemporalStore32(uintptr(destination),uint64(*(*uint32)")) ||
		!bytes.Contains(result.Source, []byte("uintptr(source)+24")) {
		t.Fatalf("non-temporal copy did not retain its fixed-width loads and stores:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("non-temporal-copy source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectEmptyAsmPreservesInputsAndClobbers(t *testing.T) {
	result := TranslateObject("main", []byte(`
void retain(void *pointer) { asm volatile("" : : "r"(pointer) : "memory"); }
`), nil)
	if !result.Ok {
		t.Fatalf("empty asm lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("_=pointer;renvo_runtime_CMemoryBarrier();")) {
		t.Fatalf("empty asm did not preserve its input and memory clobber:\n%s", result.Source)
	}
}

func TestTranslateObjectEmptyAsmPreservesReadWriteOperands(t *testing.T) {
	result := TranslateObject("main", []byte(`unsigned long retain(unsigned long value) { asm("" : "+rm"(value)); return value; }`), nil)
	if !result.Ok {
		t.Fatalf("empty read/write asm lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("_=value;")) {
		t.Fatalf("empty asm did not preserve its read/write operand:\n%s", result.Source)
	}
}

func TestTranslateObjectAsmScalarLoad(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned int global;
unsigned int load_memory(void) {
	unsigned int value;
	asm("movl %[var], %[val]" : [val] "=r"(value) : [var] "m"(global));
	return value;
}
unsigned int load_address(void) {
	unsigned int value;
	asm("movl %a[var], %[val]" : [val] "=r"(value) : [var] "i"(&global));
	return value;
}
`), nil)
	if !result.Ok {
		t.Fatalf("scalar-load asm lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("value=global;")) || !bytes.Contains(result.Source, []byte("value=*(&(global));")) {
		t.Fatalf("scalar-load asm did not preserve memory/address semantics:\n%s", result.Source)
	}
}

func TestTranslateObjectAsmScalarStore(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned long global;
void store(unsigned long value) { asm volatile("movq %[val], %[var]" : [var] "=m"(global) : [val] "re"(value)); }
`), nil)
	if !result.Ok {
		t.Fatalf("scalar-store asm lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("global=value;")) {
		t.Fatalf("scalar-store asm did not become a typed assignment:\n%s", result.Source)
	}
}

func TestTranslateObjectAsmFixedAccumulatorMMIO(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned char readb(void *position) {
	unsigned char value;
	asm volatile("movb (%1),%%al" : "=a"(value) : "r"(position));
	return value;
}
void writeb(void *position, unsigned char value) {
	asm volatile("movb %%al,(%1)" : : "a"(value), "r"(position) : "memory");
}
`), nil)
	if !result.Ok {
		t.Fatalf("fixed-accumulator MMIO lowering failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("value=*(*uint8)(__c_unsafe.Pointer(position));"),
		[]byte("*(*uint8)(__c_unsafe.Pointer(position))=value;"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("fixed-accumulator MMIO source is missing %q:\n%s", want, result.Source)
		}
	}
}

func TestTranslateObjectAlternativeMSRWrite(t *testing.T) {
	result := TranslateObject("main", []byte(`
void write(unsigned int msr, unsigned long value, int feature) {
	asm volatile("old: ds wrmsr\n replacement: wrmsr"
		: : "c"(msr), "a"((unsigned int)value), "d"((unsigned int)(value >> 32)), [feature] "i"(feature) : "memory");
}
`), nil)
	if !result.Ok {
		t.Fatalf("alternative MSR asm lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("renvo_runtime_CWriteMSR(msr,uint32(value),uint32((value>>32)))")) {
		t.Fatalf("alternative MSR asm did not become a typed write:\n%s", result.Source)
	}
}

func TestTranslateObjectWrappedMSRReadWithWideOutputs(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned long long read(unsigned int msr) {
	unsigned long low, high;
	asm volatile("1: rdmsr\n2:\n.pushsection \"__ex_table\",\"a\"\n.popsection"
		: "=a"(low), "=d"(high) : "c"(msr));
	return low | (high << 32);
}
`), nil)
	if !result.Ok {
		t.Fatalf("wrapped MSR read lowering failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("renvo_runtime_CReadMSR(msr,&__c_msr_word_"),
		[]byte("low=uint64(__c_msr_word_"),
		[]byte("high=uint64(__c_msr_word_"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("wrapped MSR read source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("wrapped MSR read source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectSafeMSROperations(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned long long read_safe(unsigned int msr, int *err) {
	unsigned long low, high;
	asm volatile("1: rdmsr ; xor %[err],%[err]\n.pushsection \"__ex_table\",\"a\"\n.popsection"
		: [err] "=r"(*err), "=a"(low), "=d"(high) : "c"(msr));
	return low | (high << 32);
}
int write_safe(unsigned int msr, unsigned int low, unsigned int high) {
	int err;
	asm volatile("1: wrmsr ; xor %[err],%[err]\n.pushsection \"__ex_table\",\"a\"\n.popsection"
		: [err] "=a"(err) : "c"(msr), "0"(low), "d"(high) : "memory");
	return err;
}
`), nil)
	if !result.Ok {
		t.Fatalf("safe MSR lowering failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("renvo_runtime_CReadMSRSafe(msr,err,&__c_msr_safe_word_"),
		[]byte("renvo_runtime_CWriteMSRSafe(msr,low,high)"),
		[]byte("renvo_runtime_CMemoryBarrier();"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("safe MSR source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("safe MSR source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectCounterAsm(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned long long timestamp(void) {
	unsigned long low, high;
	asm volatile("rdtsc" : "=a"(low), "=d"(high));
	return low | (high << 32);
}
unsigned long long ordered_timestamp(void) {
	unsigned long low, high;
	asm volatile("old: rdtsc\nreplacement: lfence; rdtsc\nreplacement2: rdtscp"
		: "=a"(low), "=d"(high) : : "ecx");
	return low | (high << 32);
}
unsigned long long performance_counter(int counter) {
	unsigned long low, high;
	asm volatile("rdpmc" : "=a"(low), "=d"(high) : "c"(counter));
	return low | (high << 32);
}
`), nil)
	if !result.Ok {
		t.Fatalf("counter asm lowering failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("renvo_runtime_CReadTimestampCounter(&__c_counter_word_"),
		[]byte("renvo_runtime_CReadTimestampCounterOrdered(&__c_counter_word_"),
		[]byte("renvo_runtime_CReadPerformanceCounter(counter,&__c_counter_word_"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("counter asm source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("counter asm source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectAsmVerifySegment(t *testing.T) {
	result := TranslateObject("main", []byte(`
void clear_buffers(void) { static const unsigned short ds = 3 * 8; asm volatile("verw %[ds]" : : [ds] "m"(ds) : "cc"); }
`), nil)
	if !result.Ok {
		t.Fatalf("verify-segment asm lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("renvo_runtime_CVerifySegment(uintptr(__c_unsafe.Pointer(&(")) ||
		!bytes.Contains(result.Source, []byte("renvo_runtime_CConditionClobber();")) {
		t.Fatalf("verify-segment asm did not preserve the operation/clobber:\n%s", result.Source)
	}
}

func TestTranslateObjectAsmFlagsAndHalt(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned long flags(void) { unsigned long value; asm volatile("pushf ; pop %0" : "=rm"(value) : : "memory"); return value; }
void halt(void) { asm volatile("hlt" : : : "memory"); }
void enable_and_halt(void) { asm volatile("sti; hlt" : : : "memory"); }
void disable(void) { asm volatile("cli" : : : "memory"); }
void flush_cache(void) { asm volatile("wbinvd" : : : "memory"); }
void swap_segments(void) { asm volatile("swapgs" : : : "memory"); }
`), nil)
	if !result.Ok {
		t.Fatalf("flags/halt asm lowering failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("value=renvo_runtime_CReadFlags();"),
		[]byte("renvo_runtime_CHalt();"),
		[]byte("renvo_runtime_CEnableInterruptsAndHalt();"),
		[]byte("renvo_runtime_CWriteBackInvalidateCache();"),
		[]byte("renvo_runtime_CSwapGS();"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("flags/halt asm source is missing %q:\n%s", want, result.Source)
		}
	}
}

func TestTranslateObjectAsmCacheFlush(t *testing.T) {
	result := TranslateObject("main", []byte(`
void flush(void *address) { asm volatile("clflush %0" : "+m"(*(volatile char *)address)); }
void alternative_flush(void *address) {
	asm volatile("# ALT: oldinstr\n.pushsection .altinstr_replacement, \"ax\"\nclflush (%[addr])\n.popsection"
		: : "i"(0), [addr] "a"(address));
}
`), nil)
	if !result.Ok {
		t.Fatalf("cache-flush asm lowering failed: %#v", result)
	}
	if bytes.Count(result.Source, []byte("renvo_runtime_CCacheLineFlush(uintptr(__c_unsafe.Pointer(")) != 2 {
		t.Fatalf("cache-flush asm did not become a typed backend operation:\n%s", result.Source)
	}
}

func TestTranslateObjectAsmPrefetch(t *testing.T) {
	result := TranslateObject("main", []byte(`
void prefetch(const void *address) {
	asm volatile("prefetcht0 %1" : : "i"(0), "m"(*(const char *)address));
	__builtin_prefetch(address);
	__builtin_prefetch((const char *)address + 64, 0, 3);
}
`), nil)
	if !result.Ok {
		t.Fatalf("prefetch asm lowering failed: %#v", result)
	}
	if bytes.Count(result.Source, []byte("renvo_runtime_CPrefetch(uintptr(__c_unsafe.Pointer(")) != 3 ||
		bytes.Contains(result.Source, []byte("__builtin_prefetch")) {
		t.Fatalf("prefetch asm did not become a typed backend operation:\n%s", result.Source)
	}
}

func TestTranslateObjectAsmHardwareFences(t *testing.T) {
	result := TranslateObject("main", []byte(`
void both(void) { asm volatile("mfence; lfence" ::: "memory"); }
void memory(void) { asm volatile("mfence" ::: "memory"); }
void locked_stack(void) { asm volatile("lock; addl $0,-4(%%rsp)" ::: "memory", "cc"); }
void read(void) { asm volatile("lfence" ::: "memory"); }
void write(void) { asm volatile("sfence" ::: "memory"); }
`), nil)
	if !result.Ok {
		t.Fatalf("hardware-fence asm lowering failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("renvo_runtime_CMemoryReadFence();"),
		[]byte("renvo_runtime_CMemoryFence();"),
		[]byte("renvo_runtime_CReadFence();"),
		[]byte("renvo_runtime_CWriteFence();"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("hardware-fence asm source is missing %q:\n%s", want, result.Source)
		}
	}
}

func TestTranslateObjectAsmDirectInstructions(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef _Bool bool;
struct block { char bytes[64]; };
void serialize(void) { asm volatile(".byte 0xf, 0x1, 0xe8" ::: "memory"); }
void tile_release(void) { asm volatile(".byte 0xc4, 0xe2, 0x78, 0x49, 0xc0"); }
void move(struct block *destination, const struct block *source) {
	asm volatile(".byte 0x66, 0x0f, 0x38, 0xf8, 0x02" : "+m"(*destination) : "m"(*source), "a"(destination), "d"(source));
}
bool enqueue(struct block *destination, const struct block *source) {
	bool zero;
	asm volatile(".byte 0xf3, 0x0f, 0x38, 0xf8, 0x02\n/* output condition code z*/"
		: "=@ccz"(zero), "+m"(*destination) : "m"(*source), "a"(destination), "d"(source));
	return zero;
}
`), nil)
	if !result.Ok {
		t.Fatalf("direct instruction asm lowering failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("renvo_runtime_CSerialize();"),
		[]byte("renvo_runtime_CTileRelease();"),
		[]byte("renvo_runtime_CDirectMove64"),
		[]byte("zero=renvo_runtime_CEnqueueCommand"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("direct instruction asm source is missing %q:\n%s", want, result.Source)
		}
	}
}

func TestTranslateObjectAsmUndefinedInstruction(t *testing.T) {
	result := TranslateObject("main", []byte(`void trap(void) { asm volatile(".byte 0x0f, 0x0b"); __builtin_unreachable(); }`), nil)
	if !result.Ok {
		t.Fatalf("undefined-instruction asm lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("renvo_runtime_CUndefinedInstruction();")) {
		t.Fatalf("undefined-instruction asm did not become a backend operation:\n%s", result.Source)
	}
	if bytes.Contains(result.Source, []byte("__builtin_unreachable")) ||
		bytes.Count(result.Source, []byte("renvo_runtime_CUndefinedInstruction();")) != 2 {
		t.Fatalf("builtin unreachable did not become a terminal backend operation:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("undefined-instruction asm source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectAsmBreakpoint(t *testing.T) {
	for _, instruction := range []string{`asm volatile("int3")`, `asm("   int $3")`} {
		result := TranslateObject("main", []byte(`void trap(void) { `+instruction+`; }`), nil)
		if !result.Ok || !bytes.Contains(result.Source, []byte("renvo_runtime_CBreakpoint();")) {
			t.Fatalf("breakpoint asm lowering failed for %q: %#v\n%s", instruction, result, result.Source)
		}
	}
}

func TestTranslateObjectAsmRandom(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef unsigned long u64;
_Bool random_long(u64 *value) {
	_Bool ok;
	asm volatile("rdrand %[out]\n\t/* output condition code c*/\n" : "=@ccc"(ok), [out] "=r"(*value));
	return ok;
}
_Bool seed_long(u64 *value) {
	_Bool ok;
	asm volatile("rdseed %[out]\n\t/* output condition code c*/\n" : "=@ccc"(ok), [out] "=r"(*value));
	return ok;
}
`), nil)
	if !result.Ok {
		t.Fatalf("random-instruction asm lowering failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("ok=renvo_runtime_CRDRAND64(value);"),
		[]byte("ok=renvo_runtime_CRDSEED64(value);"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("random-instruction asm source is missing %q:\n%s", want, result.Source)
		}
	}
}

func TestTranslateObjectAsmFDIVBug(t *testing.T) {
	result := TranslateObject("main", []byte(`
void inspect(int *result, double *x, double *y) {
	asm("fninit\n\t"
	    "fldl %1\n\t"
	    "fdivl %2\n\t"
	    "fmull %2\n\t"
	    "fldl %1\n\t"
	    "fsubp %%st,%%st(1)\n\t"
	    "fistpl %0\n\t"
	    "fwait\n\t"
	    "fninit" : "=m"(*result) : "m"(*x), "m"(*y));
}
`), nil)
	if !result.Ok || !bytes.Contains(result.Source, []byte("renvo_runtime_CFDIVBug(result,x,y);")) {
		t.Fatalf("FDIV-bug asm lowering failed: %#v\n%s", result, result.Source)
	}
}

func TestTranslateObjectAsmFPUState(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct fxstate { unsigned char data[512]; };
int save_user(struct fxstate *fx) {
	int err;
	asm volatile("1: fxsaveq %[fx]\n2:\n.pushsection \"__ex_table\",\"a\"\n.long (1b) - .\n.long (2b) - .\n.long 15\n.popsection\n"
	             : [err] "=a"(err), [fx] "=m"(*fx) : "0"(0), "m"(*fx));
	return err;
}
void restore(struct fxstate *fx) {
	asm volatile("1:fxrstorq %[fx]\n2:\n.pushsection \"__ex_table\",\"a\"\n.long (1b) - .\n.long (2b) - .\n.long 6\n.popsection\n"
	             : "=m"(*fx) : [fx] "m"(*fx));
}
int restore_safe(struct fxstate *fx) {
	int err;
	asm volatile("1:fxrstorq %[fx]\n2:\n.pushsection \"__ex_table\",\"a\"\n.long (1b) - .\n.long (2b) - .\nextable_type_reg reg=%[err], type=(17 | ((-14) << 16))\n.popsection\n"
	             : [err] "=r"(err), "=m"(*fx) : "0"(0), [fx] "m"(*fx));
	return err;
}
void load_control(unsigned int value) { asm volatile("ldmxcsr %0" :: "m"(value)); }
unsigned long read_xcontrol(unsigned int index) {
	unsigned int low, high;
	asm volatile("xgetbv" : "=a"(low), "=d"(high) : "c"(index));
	return low + ((unsigned long)high << 32);
}
void write_xcontrol(unsigned int index, unsigned long value) {
	unsigned int low = value, high = value >> 32;
	asm volatile("xsetbv" :: "a"(low), "d"(high), "c"(index));
}
void save_legacy(struct fxstate *state) {
	asm volatile("fnsave %[fp]; fwait" : [fp] "=m"(*state));
}
void prepare_legacy(int *state) {
	asm volatile("fnclex\n\temms\n\tfildl %[addr]" : : [addr] "m"(*state));
}
void wait_legacy(void) {
	asm volatile("1: fwait\n2:\n.pushsection \"__ex_table\",\"a\"\n.long (1b) - .\n.long (2b) - .\n.long 1\n.popsection\n");
}
`), nil)
	if !result.Ok {
		t.Fatalf("FPU-state asm lowering failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("err=renvo_runtime_CFPUState_FXSAVEQ_User15"),
		[]byte("renvo_runtime_CFPUState_FXRSTORQ_Fault6"),
		[]byte("err=renvo_runtime_CFPUState_FXRSTORQ_Safe17"),
		[]byte("renvo_runtime_CFPUState_LDMXCSR_Plain"),
		[]byte("renvo_runtime_CReadXControl"),
		[]byte("renvo_runtime_CWriteXControl"),
		[]byte("renvo_runtime_CFPUState_FNSAVE_Plain"),
		[]byte("renvo_runtime_CFPUPrepare(uintptr(__c_unsafe.Pointer(state)))"),
		[]byte("renvo_runtime_CFPUWait___ex_table()"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("FPU-state source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("FPU-state source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectAsmFarJump32(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct restart { unsigned int offset; unsigned int segment; };
extern struct restart *header;
void restart(unsigned int type) {
	asm volatile("ljmpl *%0" : : "m" (header->offset), "D" (type));
	__builtin_unreachable();
}
`), nil)
	if !result.Ok {
		t.Fatalf("far-jump asm lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("renvo_runtime_CFarJump32(uintptr(&(header.offset)),uintptr(__c_name_type));")) {
		t.Fatalf("far-jump asm did not retain its memory address and RDI operand:\n%s", result.Source)
	}
}

func TestTranslateObjectAsmLoadAccessRights(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned int inspect(unsigned short selector) {
	unsigned int ar;
	asm volatile("lar %[old_ss], %[ar]\n\t"
	             "jz 1f\n\t"
	             "xorl %[ar], %[ar]\n\t"
	             "1:"
	             : [ar] "=r" (ar)
	             : [old_ss] "rm" (selector));
	return ar;
}
`), nil)
	if !result.Ok || !bytes.Contains(result.Source, []byte("ar=renvo_runtime_CLoadAccessRights(uint16(selector));")) {
		t.Fatalf("access-rights asm lowering failed: %#v\n%s", result, result.Source)
	}
}

func TestTranslateObjectAsmGotoUserStore(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct large { unsigned long value; };
int store(unsigned long *pointer, unsigned long value) {
	asm goto("1: movq %0,%1\n"
	         ".pushsection \"__ex_table\",\"a\"\n"
	         ".balign 4\n"
	         ".long (1b) - .\n"
	         ".long (%l2) - .\n"
	         ".long 3\n"
	         ".popsection\n"
	         : : "er"(value), "m"((*(struct large *)(pointer))) : : Efault);
	return 0;
Efault:
	return -14;
}
`), nil)
	if !result.Ok {
		t.Fatalf("user-store asm-goto lowering failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("renvo_runtime_CUserStore64_Efault"),
		[]byte("uintptr(__c_unsafe.Pointer((*__c_struct_large)((pointer))))"),
		[]byte("if false{goto Efault}"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("user-store asm-goto source is missing %q:\n%s", want, result.Source)
		}
	}
}

func TestTranslateObjectAsmExceptionLoad(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned long load_unaligned_zeropad(const void *addr) {
	unsigned long ret;
	asm volatile("1: mov %[mem], %[ret]\n2:\n"
	             ".pushsection \"__ex_table\",\"a\"\n"
	             ".balign 4\n"
	             ".long (1b) - .\n"
	             ".long (2b) - .\n"
	             ".long 20\n"
	             ".popsection\n"
	             : [ret] "=r" (ret)
	             : [mem] "m" (*(unsigned long *)addr));
	return ret;
}
`), nil)
	if !result.Ok {
		t.Fatalf("exception load translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, fragment := range [][]byte{
		[]byte("ret=uint64(renvo_runtime_CExceptionLoad64(uintptr(__c_unsafe.Pointer("),
		[]byte("func renvo_runtime_CExceptionLoad64(address uintptr) uint64"),
	} {
		if !bytes.Contains(result.Source, fragment) {
			t.Fatalf("exception load translation missing %q:\n%s", fragment, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("exception load source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectAsmIRETToSelf(t *testing.T) {
	result := TranslateObject("main", []byte(`
register unsigned long current_stack_pointer asm("rsp");
void serialize(void) {
	unsigned int tmp;
	asm volatile("mov %%ss, %0\n\tpushq %q0\n\tpushq %%rsp\n\taddq $8, (%%rsp)\n\tpushfq\n\tmov %%cs, %0\n\tpushq %q0\n\tpushq $1f\n\tiretq\n1:"
		: "=&r"(tmp), "+r"(current_stack_pointer) : : "cc", "memory");
}
`), nil)
	if !result.Ok {
		t.Fatalf("IRET-to-self asm lowering failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("renvo_runtime_CIRETToSelf();"),
		[]byte("renvo_runtime_CConditionClobber();"),
		[]byte("renvo_runtime_CMemoryBarrier();"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("IRET-to-self source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("IRET-to-self source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectConcatenatedSectionAttribute(t *testing.T) {
	result := TranslateObject("main", []byte(`extern __attribute__((section(".data" "..read_mostly"))) unsigned long value;`), nil)
	if !result.Ok {
		t.Fatalf("concatenated section attribute failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte(".data..read_mostly")) {
		t.Fatalf("concatenated section name was not retained:\n%s", result.Source)
	}
}

func TestTranslateObjectNonDotSectionAttribute(t *testing.T) {
	result := TranslateObject("main", []byte(`static unsigned long value __attribute__((section("__param"))) = 1;`), nil)
	if !result.Ok {
		t.Fatalf("non-dot section attribute failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("// renvo:object variable value __param")) {
		t.Fatalf("non-dot section name was not retained:\n%s", result.Source)
	}
}

func TestTranslateObjectAggregatePointerReturnSectionAttribute(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct page;
struct page *allocate(unsigned long size);
struct page __attribute__((section(".init.text"))) *allocate(unsigned long size) {
	return (struct page *)0;
}
`), nil)
	if !result.Ok {
		t.Fatalf("aggregate pointer return section attribute failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("// renvo:object function allocate .init.text")) {
		t.Fatalf("aggregate pointer return section was not retained:\n%s", result.Source)
	}
}

func TestTranslateObjectExpressionHelpersInheritFunctionSection(t *testing.T) {
	result := TranslateObject("main", []byte(`
static int __attribute__((section(".init.text"))) initialize(int *value) {
	int result = ({ int nested = ({ *value = 1; *value; }); nested; });
	return (result, *value);
}
`), nil)
	if !result.Ok {
		t.Fatalf("sectioned expression helper translation failed: %#v", result)
	}
	for _, helper := range [][]byte{
		[]byte("// renvo:object function __c_statement_expr_"),
		[]byte("// renvo:object function __c_comma_"),
	} {
		at := bytes.Index(result.Source, helper)
		if at < 0 || !bytes.Contains(result.Source[at:at+min(len(result.Source)-at, 160)], []byte(".init.text")) {
			t.Fatalf("helper did not inherit .init.text for %q:\n%s", helper, result.Source)
		}
	}
	if bytes.Count(result.Source, []byte("// renvo:object function __c_statement_expr_")) < 2 ||
		bytes.Contains(result.Source, []byte("}// renvo:object")) {
		t.Fatalf("nested helper metadata was not line-delimited:\n%s", result.Source)
	}
}

func TestTranslateObjectAsmCompareExchange(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef _Bool bool;
unsigned compare_exchange(unsigned *address, unsigned expected, unsigned replacement) {
	unsigned observed;
	asm volatile("cmpxchgl %2,%1" : "=a"(observed), "+m"(*address) : "r"(replacement), "0"(expected) : "memory");
	return observed;
}
bool try_compare_exchange(unsigned *address, unsigned *expected, unsigned replacement) {
	bool equal;
	asm volatile("cmpxchgl %[new],%[ptr]\n/* output condition code z*/"
		: "=@ccz"(equal), [ptr] "+m"(*address), [old] "+a"(*expected) : [new] "r"(replacement) : "memory");
	return equal;
}
`), nil)
	if !result.Ok {
		t.Fatalf("cmpxchg asm lowering failed: %#v", result)
	}
	if bytes.Count(result.Source, []byte("renvo_runtime_CCompareExchange32")) < 3 ||
		!bytes.Contains(result.Source, []byte("equal=renvo_runtime_CCompareExchange32")) {
		t.Fatalf("cmpxchg asm did not preserve observed/condition outputs:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("cmpxchg asm source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectAsmControlRegisters(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned long read_cr0(void) {
	unsigned long value;
	asm volatile("mov %%cr0,%0" : "=r"(value));
	return value;
}
void write_cr0(unsigned long value) {
	asm volatile("mov %0,%%cr0" : : "r"(value) : "memory");
}
void write_cr4_readwrite(unsigned long value) {
	asm volatile("mov %0,%%cr4" : "+r"(value) : : "memory");
}
unsigned long read_db7(void) {
	unsigned long value;
	asm volatile("mov %%db7,%0" : "=r"(value) : "m"(*(unsigned int *)4096));
	return value;
}
void write_db7(unsigned long value) {
	asm volatile("mov %0,%%db7" : : "r"(value), "m"(*(unsigned int *)4096));
}
`), nil)
	if !result.Ok {
		t.Fatalf("control-register asm lowering failed: %#v", result)
	}
	for _, want := range [][]byte{[]byte("renvo_runtime_CReadControl0()"), []byte("renvo_runtime_CWriteControl0(value)"), []byte("renvo_runtime_CWriteControl4(value)"), []byte("renvo_runtime_CReadDebug7()"), []byte("renvo_runtime_CWriteDebug7(value)"), []byte("renvo_runtime_CMemoryBarrier()")} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("control-register asm source is missing %q:\n%s", want, result.Source)
		}
	}
}

func TestTranslateObjectAsmFPUAndFlags(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef unsigned short u16;
int fpu(void) {
	u16 status = -1, control = -1;
	asm volatile("fninit ; fnstsw %0 ; fnstcw %1" : "+m"(status), "+m"(control));
	return status == 0 && (control & 0x103f) == 0x003f;
}
int toggle(unsigned long mask) {
	unsigned long before, after;
	asm volatile("pushfq\n\tpushfq\n\tpop %0\n\tmov %0,%1\n\txor %2,%1\n\tpush %1\n\tpopfq\n\tpushfq\n\tpop %1\n\tpopfq"
		: "=&r"(before), "=&r"(after) : "ri"(mask));
	return before != after;
}
`), nil)
	if !result.Ok {
		t.Fatalf("FPU/flags asm lowering failed: %#v", result)
	}
	for _, want := range [][]byte{[]byte("renvo_runtime_CInitFPU()"), []byte("renvo_runtime_CReadFlags()"), []byte("renvo_runtime_CWriteFlags(")} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("FPU/flags asm source is missing %q:\n%s", want, result.Source)
		}
	}
}

func TestTranslateObjectAsmCPUID(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef unsigned int u32;
void cpuid_count(u32 leaf, u32 count, u32 *a, u32 *b, u32 *c, u32 *d) {
	asm volatile("cpuid" : "=a"(*a), "=b"(*b), "=c"(*c), "=d"(*d) : "0"(leaf), "2"(count));
}
`), nil)
	if !result.Ok {
		t.Fatalf("CPUID asm lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("renvo_runtime_CCPUID(leaf,count,a,b,c,d)")) {
		t.Fatalf("CPUID asm did not become a typed backend operation:\n%s", result.Source)
	}
}

func TestTranslateObjectAsmPortIO(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef unsigned char u8;
typedef unsigned short u16;
typedef unsigned int u32;
void outb(u8 value, u16 port) { asm volatile("outb %b0, %w1" : : "a"(value), "Nd"(port)); }
u8 inb(u16 port) { u8 value; asm volatile("inb %w1, %b0" : "=a"(value) : "Nd"(port)); return value; }
void outw(u16 value, u16 port) { asm volatile("outw %w0, %w1" : : "a"(value), "Nd"(port)); }
u32 inl(u16 port) { u32 value; asm volatile("inl %w1, %0" : "=a"(value) : "Nd"(port)); return value; }
void delay80(void) { asm volatile("outb %al, $0x80"); }
void delayed(void) { asm volatile("outb %al, $237"); }
`), nil)
	if !result.Ok {
		t.Fatalf("port-I/O asm lowering failed: %#v", result)
	}
	for _, want := range [][]byte{[]byte("renvo_runtime_COut8"), []byte("renvo_runtime_CIn8"), []byte("renvo_runtime_COut16"), []byte("renvo_runtime_CIn32")} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("port-I/O asm source is missing %q:\n%s", want, result.Source)
		}
	}
	if !bytes.Contains(result.Source, []byte("renvo_runtime_COut8(0,128)")) ||
		!bytes.Contains(result.Source, []byte("renvo_runtime_COut8(0,237)")) {
		t.Fatalf("direct immediate port output was not lowered:\n%s", result.Source)
	}
}

func TestTranslateObjectAsmSegmentRegisters(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef unsigned short u16;
typedef unsigned int u32;
u16 read_ds(void) { u16 value; asm("movw %%ds,%0" : "=rm"(value)); return value; }
u16 read_fs(void) { u16 value; asm("movw %%fs,%0" : "=rm"(value)); return value; }
u16 read_ss(void) { u16 value; asm("mov %%ss,%0" : "=r"(value) : : "memory"); return value; }
u16 read_es_inferred(void) { u16 value; asm("mov %%es,%0" : "=r"(value)); return value; }
u16 read_ds_inferred(void) { u16 value; asm("mov %%ds,%0" : "=r"(value)); return value; }
u32 read_es32(void) { u32 value; asm("movl %%es,%0" : "=r"(value)); return value; }
u32 read_ds32(void) { u32 value; asm("movl %%ds,%0" : "=r"(value)); return value; }
u32 read_fs32(void) { u32 value; asm("movl %%fs,%0" : "=r"(value)); return value; }
u32 read_gs32(void) { u32 value; asm("movl %%gs,%0" : "=r"(value)); return value; }
void write_fs(u16 value) { asm volatile("movw %0,%%fs" : : "rm"(value)); }
void clear_fs(void) { asm volatile("mov %0,%%fs" : : "rm"(0)); }
void write_fs_checked(u16 value) {
	asm volatile("1: movw %0, %%fs\n2:\n .pushsection \"__ex_table\",\"a\"\n .long 1b, 2b\n .popsection"
		: : "rm"(value) : "memory");
}
void write_startup_segments(u32 value) {
	asm volatile("movl %%eax, %%ds\nmovl %%eax, %%ss\nmovl %%eax, %%es" : : "a"(value) : "memory");
}
`), nil)
	if !result.Ok {
		t.Fatalf("segment-register asm lowering failed: %#v", result)
	}
	for _, want := range [][]byte{[]byte("renvo_runtime_CReadSegmentes"), []byte("renvo_runtime_CReadSegmentds"), []byte("renvo_runtime_CReadSegmentss"), []byte("renvo_runtime_CReadSegmentfs"), []byte("renvo_runtime_CReadSegmentgs"), []byte("renvo_runtime_CWriteSegmentfs")} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("segment-register asm source is missing %q:\n%s", want, result.Source)
		}
	}
	if !bytes.Contains(result.Source, []byte("renvo_runtime_CWriteStartupSegments(uint16(value))")) {
		t.Fatalf("startup segment-register sequence was not preserved:\n%s", result.Source)
	}
}

func TestTranslateObjectAsmSafeSegmentRegister(t *testing.T) {
	result := TranslateObject("main", []byte(`
void load(unsigned short selector) {
	asm volatile("1: movl %k0,%%es\n"
	             ".pushsection \"__ex_table\",\"a\"\n"
	             ".long (1b) - .\n"
	             ".long (1b) - .\n"
	             ".macro extable_type_reg type:req reg:req\n"
	             ".long \\type\n"
	             ".endm\n"
	             "extable_type_reg reg=%k0, type=(17 | ((0) << 16))\n"
	             ".purgem extable_type_reg\n"
	             ".popsection\n"
	             : "+r" (selector) : : "memory");
}
`), nil)
	if !result.Ok || !bytes.Contains(result.Source, []byte("renvo_runtime_CWriteSegmentSafees(uint16(selector));")) {
		t.Fatalf("safe segment-register asm lowering failed: %#v\n%s", result, result.Source)
	}
}

func TestTranslateObjectAsmBaseRegisters(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned long read_fs(void) { unsigned long value; asm volatile("rdfsbase %0" : "=r"(value) :: "memory"); return value; }
unsigned long read_gs(void) { unsigned long value; asm volatile("rdgsbase %0" : "=r"(value) :: "memory"); return value; }
void write_fs(unsigned long value) { asm volatile("wrfsbase %0" :: "r"(value) : "memory"); }
void write_gs(unsigned long value) { asm volatile("wrgsbase %0" :: "r"(value) : "memory"); }
`), nil)
	if !result.Ok {
		t.Fatalf("base-register asm lowering failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("renvo_runtime_CReadFSBase()"), []byte("renvo_runtime_CReadGSBase()"),
		[]byte("renvo_runtime_CWriteFSBase(uint64(value))"), []byte("renvo_runtime_CWriteGSBase(uint64(value))"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("base-register asm source is missing %q:\n%s", want, result.Source)
		}
	}
	if bytes.Count(result.Source, []byte("renvo_runtime_CMemoryBarrier();")) != 4 {
		t.Fatalf("base-register asm did not retain memory clobbers:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("base-register asm source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectAsmSegmentMemory(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef unsigned char u8;
typedef unsigned short u16;
typedef unsigned int u32;
u8 read_fs8(u8 *pointer) { u8 value; asm volatile("movb %%fs:%1,%0" : "=q"(value) : "m"(*pointer)); return value; }
u16 read_gs16(u16 *pointer) { u16 value; asm volatile("movw %%gs:%1,%0" : "=r"(value) : "m"(*pointer)); return value; }
void write_fs32(u32 *pointer, u32 value) { asm volatile("movl %1,%%fs:%0" : "+m"(*pointer) : "ri"(value)); }
`), nil)
	if !result.Ok {
		t.Fatalf("segment-memory asm lowering failed: %#v", result)
	}
	for _, want := range [][]byte{[]byte("renvo_runtime_CReadSegmentfs8"), []byte("renvo_runtime_CReadSegmentgs16"), []byte("renvo_runtime_CWriteSegmentfs32")} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("segment-memory asm source is missing %q:\n%s", want, result.Source)
		}
	}
}

func TestTranslateObjectAsmSegmentCompare(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef _Bool bool;
typedef unsigned int addr_t;
typedef unsigned int size_t;
bool compare_fs(const void *left, addr_t right, size_t count) {
	bool different;
	asm volatile("fs; repe; cmpsb\n\t/* output condition code nz*/\n"
		: "=@ccnz"(different), "+D"(left), "+S"(right), "+c"(count));
	return different;
}
`), nil)
	if !result.Ok {
		t.Fatalf("segment-compare asm lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("renvo_runtime_CSegmentComparefs(&left,&right,&count)")) {
		t.Fatalf("segment-compare asm did not become a typed backend operation:\n%s", result.Source)
	}
}

func TestTranslateObjectNestedTypeofCast(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned long read_once(const void *address) {
	return *(const volatile typeof(_Generic((*(unsigned long *)address),
		unsigned long: (unsigned long)0, default: (*(unsigned long *)address))) *)
		&(*(unsigned long *)address);
}
struct item;
struct item *read_next(struct item **cursor) {
	return *(const volatile typeof(_Generic((*cursor++),
		default: (*cursor++))) *)&(*cursor++);
}
`), nil)
	if !result.Ok {
		t.Fatalf("nested typeof cast lowering failed: %#v", result)
	}
	if bytes.Contains(result.Source, []byte("typeof")) {
		t.Fatalf("nested typeof escaped into shared syntax:\n%s", result.Source)
	}
	if !bytes.Contains(result.Source, []byte("__c_post_deref_inc_")) ||
		bytes.Contains(result.Source, []byte("&((__c_post_deref_inc_")) {
		t.Fatalf("qualified postfix lvalue read was not canonicalized:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("nested typeof shared source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateVoidDiscardCast(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct key { int value; };
void discard(struct key *key) {
	(void)&key->value;
	(void)"mapping.invalidate_lock";
}
`), nil)
	if !result.Ok {
		t.Fatalf("void discard cast failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("__c_discard_")) || bytes.Contains(result.Source, []byte("byte((")) {
		t.Fatalf("void casts were not lowered to typed discards:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("void discard source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateStatementExpressionFunctionPointerCall(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct sk_buff;
struct neighbour {
	int (*output)(struct neighbour *, struct sk_buff *);
};
int call_output(struct neighbour *n, struct sk_buff *skb) {
	return ({
		do { } while (0);
		(*(const volatile typeof(_Generic((n->output),
			char: (char)0,
			default: (n->output))) *)&(n->output));
	})(n, skb);
}
`), nil)
	if !result.Ok {
		t.Fatalf("statement-expression function pointer call failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("__c_statement_expr_")) || bytes.Contains(result.Source, []byte("typeof")) {
		t.Fatalf("statement-expression function pointer callee was not lowered:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("statement-expression function pointer source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectHoistsLocalUnionHelpers(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned long divide(unsigned long value) {
	union { unsigned long full; unsigned int words[2]; } parts = { value };
	return parts.words[0];
}
`), nil)
	if !result.Ok {
		t.Fatalf("local union lowering failed: %#v", result)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("local union helpers were not hoisted: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectCompileTimeBuiltins(t *testing.T) {
	result := TranslateObject("main", []byte(`
int inspect(int value) {
	return __builtin_expect(value, 0) +
		__builtin_constant_p(4 + 5) +
		__builtin_choose_expr(sizeof(long) == 8, 3, 7) +
		__builtin_types_compatible_p(unsigned long, unsigned long);
}
`), nil)
	if !result.Ok {
		t.Fatalf("compile-time builtin lowering failed: %#v", result)
	}
	for _, unwanted := range [][]byte{[]byte("__builtin_expect"), []byte("__builtin_constant_p"), []byte("__builtin_choose_expr"), []byte("__builtin_types_compatible_p")} {
		if bytes.Contains(result.Source, unwanted) {
			t.Fatalf("compile-time builtin %q escaped into shared syntax:\n%s", unwanted, result.Source)
		}
	}
}

func TestTranslateGNUSizeofDereferencedVoidPointer(t *testing.T) {
	result := TranslateObject("main", []byte(`
int choose_size(void) {
	return __builtin_choose_expr(
		sizeof(int) == sizeof(*(8 ? (void *)0 : (int *)8)),
		3, 7);
}
`), nil)
	if !result.Ok {
		t.Fatalf("GNU sizeof dereferenced void pointer failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("return 7")) || bytes.Contains(result.Source, []byte("__builtin_choose_expr")) {
		t.Fatalf("GNU void-pointer sizeof was not folded:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("GNU void-pointer sizeof source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectEscapesGoKeywordNames(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct callback { void (*func)(struct callback *); int type; };
void inspect(struct callback *value, int type) {
	value->type = type;
}
`), nil)
	if !result.Ok {
		t.Fatalf("Go-keyword identifier lowering failed: %#v", result)
	}
	for _, want := range [][]byte{[]byte("__c_name_func"), []byte("__c_name_type")} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("Go-keyword identifier source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("Go-keyword identifier shared source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectCanonicalizesPostfixDereference(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned char next(const unsigned char *pointer) { return *pointer++; }
unsigned char nested(unsigned char **pointer) { return *((*pointer)++); }
unsigned char cast_next(const char *pointer) { return *(const unsigned char *)pointer++; }
`), nil)
	if !result.Ok {
		t.Fatalf("postfix dereference lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("__c_post_deref_inc_")) ||
		!bytes.Contains(result.Source, []byte("__c_post_cast_deref_inc_")) ||
		!bytes.Contains(result.Source, []byte("uintptr(1)")) ||
		bytes.Contains(result.Source, []byte("&((*uint8)(pointer))")) {
		t.Fatalf("postfix dereference was not canonicalized:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("postfix dereference shared source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectDecaysLocalArrayForArrowMember(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct item { unsigned long min; unsigned long max; };
int inside(int n) {
	struct item items[3];
	return items[n].min >= items->min && items[n].max <= items->max;
}
`), nil)
	if !result.Ok {
		t.Fatalf("local array arrow-member lowering failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if bytes.Count(result.Source, []byte("(items)[0].")) != 2 ||
		bytes.Contains(result.Source, []byte("(items).min")) || bytes.Contains(result.Source, []byte("(items).max")) {
		t.Fatalf("local array did not decay to its first element for arrow-member access:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("local array arrow-member source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectCastedIncompleteArrayElementAddress(t *testing.T) {
	result := TranslateObject("main", []byte(`
extern const int values[];
struct table { void *extra; };
struct table entry = { .extra = (void *)&values[8] };
`), nil)
	if !result.Ok {
		t.Fatalf("casted incomplete-array element address failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("index*uintptr(4)")) ||
		bytes.Contains(result.Source, []byte("__c_pointer_index_16(pointer *byte")) {
		t.Fatalf("cast was applied before incomplete-array address arithmetic:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("casted incomplete-array address source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectCanonicalizesParenthesizedPostfixDereferenceAssignment(t *testing.T) {
	result := TranslateObject("main", []byte(`
void store(unsigned long *cursor, unsigned long value) { ((*cursor++) = value); }
void store_operand_parenthesized(unsigned long *cursor, unsigned long value) { *(cursor++) = value; }
void store_cancelled(unsigned long *cursor, unsigned long value) { (*&*cursor++ = value); }
void store_once(unsigned long *cursor, unsigned long value) {
	*(volatile typeof(*&*cursor++) *)&(*&*cursor++) = value;
}
`), nil)
	if !result.Ok {
		t.Fatalf("parenthesized postfix assignment lowering failed: %#v", result)
	}
	if bytes.Count(result.Source, []byte("__c_post_assign_inc_")) < 5 ||
		bytes.Contains(result.Source, []byte("&((__c_post_deref_inc_")) ||
		bytes.Contains(result.Source, []byte("__c_post_deref_inc_2(&(cursor))=")) {
		t.Fatalf("parenthesized postfix assignment was not canonicalized:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("parenthesized postfix assignment source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectPostfixDereferenceAssignmentValue(t *testing.T) {
	result := TranslateObject("main", []byte(`
char *copy(char *destination, const char *source) {
	while ((*destination++ = *source++) != 0) {}
	return destination;
}
`), nil)
	if !result.Ok {
		t.Fatalf("postfix dereference assignment value failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("__c_post_assign_inc_")) ||
		bytes.Contains(result.Source, []byte("&((__c_post_deref_inc_")) {
		t.Fatalf("postfix dereference assignment value was not canonicalized:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("postfix dereference assignment value source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectCanonicalizesScalarPostfixValue(t *testing.T) {
	result := TranslateObject("main", []byte(`
int countdown(int loops) { int count = 0; while (loops--) count++; return count; }
`), nil)
	if !result.Ok {
		t.Fatalf("scalar postfix lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("__c_scalar_post_dec_")) || bytes.Contains(result.Source, []byte("for loops--")) {
		t.Fatalf("scalar postfix value was not canonicalized:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("scalar postfix shared source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateScalarPrefixMutationThroughDereference(t *testing.T) {
	result := TranslateObject("main", []byte(`
int release(int *count) {
	if (!--*count)
		return 1;
	return ++*count;
}
`), nil)
	if !result.Ok {
		t.Fatalf("scalar prefix dereference mutation failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("__c_scalar_pre_dec_")) ||
		!bytes.Contains(result.Source, []byte("__c_scalar_pre_inc_")) ||
		bytes.Contains(result.Source, []byte("((--)")) {
		t.Fatalf("scalar prefix dereference mutation was not canonicalized:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("scalar prefix dereference source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectCanonicalizesNestedScalarMutation(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct counters { unsigned long compressed; };
struct stream { struct counters block; };
unsigned long step(struct stream *stream) {
	++stream->block.compressed;
	return stream->block.compressed--;
}
`), nil)
	if !result.Ok {
		t.Fatalf("nested scalar mutation lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("__c_scalar_pre_inc_")) ||
		!bytes.Contains(result.Source, []byte("__c_scalar_post_dec_")) ||
		bytes.Contains(result.Source, []byte("++stream.block.compressed")) {
		t.Fatalf("nested scalar mutation was not canonicalized:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("nested scalar mutation source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectCanonicalizesChainedAssignment(t *testing.T) {
	result := TranslateObject("main", []byte(`
int copy(int value) { int first, second; first = second = value; return first + second; }
`), nil)
	if !result.Ok {
		t.Fatalf("chained assignment lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("first=__c_assignment_value_")) ||
		!bytes.Contains(result.Source, []byte("(&(second),value)")) {
		t.Fatalf("chained assignment was not canonicalized:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("chained assignment shared source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectCanonicalizesParenthesizedAssignmentValue(t *testing.T) {
	result := TranslateObject("main", []byte(`
extern unsigned int read_seq(void);
unsigned int wait_seq(void) {
	unsigned int seq;
	while (__builtin_expect(!!((seq = read_seq()) & 1), 0)) {}
	return seq;
}
`), nil)
	if !result.Ok {
		t.Fatalf("parenthesized assignment-value lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("__c_assignment_value_")) ||
		bytes.Contains(result.Source, []byte("seq=read_seq()&")) {
		t.Fatalf("parenthesized assignment escaped expression-value lowering:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("parenthesized assignment-value source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectPromotesParenthesizedAssignmentValue(t *testing.T) {
	result := TranslateObject("main", []byte(`
char consume(const char *cursor) {
	char value;
	while ((value = *cursor++) != '\0') {}
	return value;
}
`), nil)
	if !result.Ok {
		t.Fatalf("promoted assignment-value lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("__c_assignment_value_")) ||
		bytes.Contains(result.Source, []byte("for value=")) {
		t.Fatalf("promoted assignment escaped expression-value lowering:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("promoted assignment-value source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectPreservesPointerAssignmentValueType(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct node { struct node *next; };
int count(struct node *node) {
	struct node *next;
	int total = 0;
	while ((next = node->next) != (void *)0) { total++; node = next; }
	return total;
}
`), nil)
	if !result.Ok {
		t.Fatalf("pointer assignment-value lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("__c_assignment_value_")) ||
		!bytes.Contains(result.Source, []byte(")!=nil")) ||
		bytes.Contains(result.Source, []byte("int32(nil)")) {
		t.Fatalf("pointer assignment lost its expression type:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("pointer assignment-value source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectSequencesCommaAssignmentsBeforeForLoop(t *testing.T) {
	result := TranslateObject("main", []byte(`
int count(int limit) {
	int index, total;
	for (index = 0, total = 0; index < limit;) { total++; index++; }
	return total;
}
`), nil)
	if !result.Ok {
		t.Fatalf("comma assignment lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("for __c_comma_")) ||
		bytes.Contains(result.Source, []byte("for index=__c_comma_")) {
		t.Fatalf("comma operator lost precedence to assignment:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("comma assignment source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectSequencesCompoundAssignmentValueInCommaExpression(t *testing.T) {
	result := TranslateObject("main", []byte(`
int advance(int delta) {
	int value = 0;
	return (value = 1, value += delta);
}
`), nil)
	if !result.Ok {
		t.Fatalf("compound assignment comma lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("__c_comma_")) ||
		!bytes.Contains(result.Source, []byte("return __c_compound_1_")) ||
		bytes.Contains(result.Source, []byte("return (*p")) && bytes.Contains(result.Source, []byte("+=")) {
		t.Fatalf("compound assignment escaped expression-value lowering:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("compound assignment comma source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectSequencesPointerCompoundAssignmentValue(t *testing.T) {
	result := TranslateObject("main", []byte(`
char *advance(char *pointer, unsigned long *count) {
	return ((*count) += 4096, pointer += 4096);
}
`), nil)
	if !result.Ok {
		t.Fatalf("pointer compound assignment comma lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("return __c_pointer_assignment_inc_value_")) ||
		bytes.Contains(result.Source, []byte("return (*p1)=")) {
		t.Fatalf("pointer compound assignment escaped expression-value lowering:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("pointer compound assignment source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectDiscardsEmptyStatementExpressionInComma(t *testing.T) {
	result := TranslateObject("main", []byte(`
int update(int *value) { return (({ ; }), *value = 7); }
int update_without_null_statement(int *value) { return (({}), *value = 9); }
`), nil)
	if !result.Ok {
		t.Fatalf("empty statement-expression comma lowering failed: %#v", result)
	}
	if bytes.Count(result.Source, []byte("{_=0;return __c_assignment_value_")) != 2 ||
		bytes.Contains(result.Source, []byte("({;})")) {
		t.Fatalf("empty statement expression escaped comma lowering:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("empty statement-expression source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectLowersCleanupGuardLoop(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct guard { int *pointer; };
struct guard construct(void);
void destruct(struct guard *);
int *lock_pointer(struct guard *);
int conditional;
void disable(void);
void test(void) {
	for (struct guard scope __attribute__((cleanup(destruct))) = construct();
	     lock_pointer(&scope) || !conditional;
	     ({ goto done; }))
		if (0) { done: break; } else disable();
}
int test_conditional(void) {
	for (struct guard scope __attribute__((cleanup(destruct))) = construct();
	     1;
	     ({ goto conditional_done; }))
		if (!lock_pointer(&scope)) { return -1; conditional_done: break; } else disable();
	return 0;
}
`), nil)
	if !result.Ok {
		t.Fatalf("cleanup guard loop lowering failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("defer destruct(&scope)"),
		[]byte("for ;"),
		[]byte("disable()"),
		[]byte(";break;}"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("cleanup guard loop is missing %q:\n%s", want, result.Source)
		}
	}
	if bytes.Contains(result.Source, []byte("goto done")) || bytes.Contains(result.Source, []byte("goto conditional_done")) {
		t.Fatalf("cleanup guard loop retained its synthetic C goto:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("cleanup guard loop source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectCanonicalizesCompoundAssignment(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned long clear(unsigned long value, unsigned long mask) { value &= ~mask; return value; }
unsigned long merge(unsigned long value, unsigned int high) { value |= (unsigned long)high << 32; return value; }
struct flags { unsigned long value; };
void clear_member(struct flags *flags, unsigned long mask) { flags->value &= ~mask; }
void clear_indirect(unsigned long *value, unsigned long mask) { *value &= ~mask; }
void update_bool(_Bool *nx, unsigned long flags) { *nx |= flags & (1UL << 63); *nx &= flags & 2; }
const char *advance(const char *pointer) { pointer += 2; return pointer; }
`), nil)
	if !result.Ok {
		t.Fatalf("compound assignment lowering failed: %#v", result)
	}
	if bytes.Contains(result.Source, []byte("&=")) || bytes.Contains(result.Source, []byte("|=")) ||
		!bytes.Contains(result.Source, []byte("value=value&")) || !bytes.Contains(result.Source, []byte("value=value|uint64(high)<<32")) ||
		!bytes.Contains(result.Source, []byte("flags.value=flags.value&")) || !bytes.Contains(result.Source, []byte("__c_compound_8_")) ||
		bytes.Contains(result.Source, []byte("uint64()")) || bytes.Count(result.Source, []byte("__c_compound_bool_")) < 2 ||
		!bytes.Contains(result.Source, []byte("__c_pointer_step_inc_")) {
		t.Fatalf("compound assignment was not canonicalized:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("compound assignment shared source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslatePrefixPointerMutationDereferenceAssignment(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct cursor { char *buffer; };
void prepend(struct cursor cursor, char value) { *--cursor.buffer = value; }
`), nil)
	if !result.Ok {
		t.Fatalf("prefix pointer dereference assignment failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("__c_pre_deref_assign_dec_")) || bytes.Contains(result.Source, []byte("*(--cursor.buffer)")) {
		t.Fatalf("prefix pointer dereference assignment was not canonicalized:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("prefix pointer dereference assignment source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslatePrefixPointerMutationMemberRead(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct cursor { unsigned long length; };
unsigned long previous_length(struct cursor *cursor) { return (--cursor)->length; }
`), nil)
	if !result.Ok {
		t.Fatalf("prefix pointer member read failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("__c_pointer_pre_dec_")) ||
		bytes.Contains(result.Source, []byte("(--cursor).length")) {
		t.Fatalf("prefix pointer member read was not canonicalized:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("prefix pointer member shared source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectLowersStatementExpressionWithTiedAsm(t *testing.T) {
	result := TranslateObject("main", []byte(`
void *real_mode_pointer(unsigned int address) {
	return ({
		unsigned long pointer;
		__asm__("" : "=r"(pointer) : "0"((void *)address));
		(typeof((void *)address))(pointer + 0);
	});
}
`), nil)
	if !result.Ok {
		t.Fatalf("statement-expression lowering failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("__c_statement_expr_"),
		[]byte("pointer=uint64((*byte)"),
		[]byte("return (*byte)((pointer+0))"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("statement-expression lowering is missing %q:\n%s", want, result.Source)
		}
	}
	if bytes.Contains(result.Source, []byte("__asm__")) || bytes.Contains(result.Source, []byte("typeof")) {
		t.Fatalf("statement-expression lowering leaked C syntax:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("statement-expression shared source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectLowersMSRAsm(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct msr { unsigned int low, high; };
void read_msr(unsigned int reg, struct msr *value) {
	asm volatile("rdmsr" : "=a"(value->low), "=d"(value->high) : "c"(reg));
}
void write_msr(unsigned int reg, const struct msr *value) {
	asm volatile("wrmsr" : : "c"(reg), "a"(value->low), "d"(value->high) : "memory");
}
`), nil)
	if !result.Ok {
		t.Fatalf("MSR asm lowering failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("renvo_runtime_CReadMSR(reg,&(value.low),&(value.high))"),
		[]byte("renvo_runtime_CWriteMSR(reg,value.low,value.high)"),
		[]byte("renvo_runtime_CMemoryBarrier()"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("MSR asm lowering is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("MSR shared source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectIsolatesComplexCondition(t *testing.T) {
	result := TranslateObject("main", []byte(`
int inspect(const char *pointer, unsigned int base) {
	if (base == 16 && pointer[0] == '0' && (pointer[1] | 32) == 'x')
		return 1;
	return 0;
}
`), nil)
	if !result.Ok {
		t.Fatalf("complex condition lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("if __c_condition_")) {
		t.Fatalf("complex condition was not isolated:\n%s", result.Source)
	}
}

func TestTranslateObjectConsumesFallthroughAttributeStatement(t *testing.T) {
	result := TranslateObject("main", []byte(`
int select_value(int value) {
	do {
		__attribute__((__noreturn__)) extern void fail(void);
		if (value < 0) fail();
	} while (0);
	switch (value) {
	case 1:
		value++;
		__attribute__((__fallthrough__));
	case 2:
		return value;
	default:
		return 0;
	}
}
`), nil)
	if !result.Ok {
		t.Fatalf("fallthrough attribute lowering failed: %#v", result)
	}
	if bytes.Contains(result.Source, []byte("__attribute__")) || !bytes.Contains(result.Source, []byte("fallthrough;case 2:")) {
		t.Fatalf("fallthrough attribute was not consumed:\n%s", result.Source)
	}
}

func TestTranslateObjectExpandsSmallCaseRange(t *testing.T) {
	result := TranslateObject("main", []byte(`
int digit(int value) {
	switch (value) {
	case '0' ... '3': return 1;
	default: return 0;
	}
}
`), nil)
	if !result.Ok || !bytes.Contains(result.Source, []byte("case 48,49,50,51:")) {
		t.Fatalf("case-range lowering failed: %#v\n%s", result, result.Source)
	}
}

func TestTranslateObjectCanonicalizesPointerLoopStep(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned long length(const char *pointer) {
	const char *start = pointer;
	for (; *pointer; pointer++) { }
	return pointer - start;
}
struct cursor { const char *entry; };
const char *take(struct cursor *cursor) {
	return cursor->entry++;
}
`), nil)
	if !result.Ok {
		t.Fatalf("pointer loop-step lowering failed: %#v", result)
	}
	if bytes.Count(result.Source, []byte("__c_pointer_post_inc_")) < 3 || !bytes.Contains(result.Source, []byte("__c_pointer_diff_")) || bytes.Contains(result.Source, []byte(".entry++")) {
		t.Fatalf("pointer loop step was not canonicalized:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("pointer loop-step shared source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectOverflowBuiltins(t *testing.T) {
	result := TranslateObject("main", []byte(`
int arithmetic(unsigned long a, unsigned long b, unsigned long *result) {
	return __builtin_add_overflow(a, b, result) |
		__builtin_sub_overflow(a, b, result) |
		__builtin_mul_overflow(a, b, result);
}
int signed_multiply(long a, long b, long *result) {
	return __builtin_mul_overflow(a, b, result);
}
`), nil)
	if !result.Ok {
		t.Fatalf("overflow builtin lowering failed: %#v", result)
	}
	for _, want := range [][]byte{[]byte("__c_add_overflow_u64"), []byte("__c_sub_overflow_u64"), []byte("__c_mul_overflow_u64"), []byte("__c_mul_overflow_i64")} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("overflow builtin source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("overflow builtin shared source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectLowersArrayIndexStatementExpression(t *testing.T) {
	result := TranslateObject("main", []byte(`
int select_digit(unsigned long value, unsigned int base) {
	static const char digits[] = "0123456789";
	return digits[({ int result; result = value % base; value = value / base; result; })];
}
`), nil)
	if !result.Ok || bytes.Count(result.Source, []byte("__c_statement_expr_")) < 2 || bytes.Contains(result.Source, []byte("({")) {
		t.Fatalf("array-index statement-expression lowering failed: %#v\n%s", result, result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("array-index statement-expression shared source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectRecognizesCastBeforeUnaryAddress(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef unsigned long size_t;
unsigned short address(void) { static int value; return (size_t)&value; }
`), nil)
	if !result.Ok || bytes.Contains(result.Source, []byte("size_t")) || !bytes.Contains(result.Source, []byte("uint16(uint64(&(")) {
		t.Fatalf("cast-before-unary lowering failed: %#v\n%s", result, result.Source)
	}
}

func TestTranslateObjectCanonicalizesBuiltinMemoryCalls(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef unsigned long size_t;
void *memcpy(void *, const void *, size_t);
void *memmove(void *, const void *, size_t);
void *memset(void *, int, size_t);
int memcmp(const void *, const void *, size_t);
void clear(void *pointer, size_t size) { __builtin_memset(pointer, 0, size); }
void copy(void *to, const void *from, size_t size) { __builtin_memcpy(to, from, size); }
`), nil)
	if !result.Ok {
		t.Fatalf("builtin memory-call lowering failed: %#v", result)
	}
	if bytes.Contains(result.Source, []byte("__builtin_mem")) || !bytes.Contains(result.Source, []byte("memset(pointer,0,size)")) ||
		!bytes.Contains(result.Source, []byte("memcpy(to,from,size)")) {
		t.Fatalf("builtin memory calls were not canonicalized:\n%s", result.Source)
	}
}

func TestTranslateObjectRejectsUnknownAsmTemplate(t *testing.T) {
	result := TranslateObject("main", []byte(`void mystery(void) { asm volatile("renvo-mystery"); }`), nil)
	if result.Ok || result.Error != TranslateErrUnsupported {
		t.Fatalf("unknown asm template result = %#v, want explicit unsupported error", result)
	}
}

func TestTranslateObjectLowersPauseAsm(t *testing.T) {
	result := TranslateObject("main", []byte(`
void first(void) { asm volatile("pause"); }
void second(void) { asm volatile("rep; nop" ::: "memory"); }
`), nil)
	if !result.Ok || bytes.Count(result.Source, []byte("renvo_runtime_CPause();")) != 2 ||
		!bytes.Contains(result.Source, []byte("renvo_runtime_CMemoryBarrier();")) {
		t.Fatalf("pause asm lowering failed: %#v\n%s", result, result.Source)
	}
}

func TestTranslateObjectLowersChecksumFoldAsm(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned short fold(unsigned int sum) {
	asm("addl %1,%0\n\tadcl $0xffff,%0"
		: "=r"(sum) : "r"(sum << 16), "0"(sum & 0xffff0000));
	return (unsigned short)(~sum >> 16);
}
`), nil)
	if !result.Ok {
		t.Fatalf("checksum-fold asm lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("sum=renvo_runtime_CChecksumFold(uint32(sum&4294901760),uint32(sum<<16));")) ||
		!bytes.Contains(result.Source, []byte("wide:=uint64(initial)+uint64(addend)")) {
		t.Fatalf("checksum-fold asm did not preserve add-with-carry semantics:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("checksum-fold source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectLowersIPChecksumAsm(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned short checksum(const void *data, unsigned int words) {
	unsigned int sum;
	asm("movl (%1), %0\n"
		"subl $4, %2\n"
		"jbe 2f\n"
		"addl 4(%1), %0\n"
		"adcl 8(%1), %0\n"
		"adcl 12(%1), %0\n"
		"1: adcl 16(%1), %0\n"
		"lea 4(%1), %1\n"
		"decl %2\n"
		"jne 1b\n"
		"adcl $0, %0\n"
		"movl %0, %2\n"
		"shrl $16, %0\n"
		"addw %w2, %w0\n"
		"adcl $0, %0\n"
		"notl %0\n"
		"2:"
		: "=r"(sum), "=r"(data), "=r"(words) : "1"(data), "2"(words) : "memory");
	return (unsigned short)sum;
}
`), nil)
	if !result.Ok {
		t.Fatalf("IP-checksum asm lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("sum=renvo_runtime_CIPChecksum(&(data),&(words));")) ||
		!bytes.Contains(result.Source, []byte("for i:=uint32(1);i<words;i++")) ||
		!bytes.Contains(result.Source, []byte("renvo_runtime_CMemoryBarrier();")) {
		t.Fatalf("IP-checksum asm did not preserve memory and carry semantics:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("IP-checksum source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectLowersChecksumAddAsm(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned int add(unsigned int a, unsigned int b) {
	asm("addl %2,%0\n\tadcl $0,%0" : "=r"(a) : "0"(a), "rm"(b));
	return a;
}
unsigned int pseudo(unsigned int source, unsigned int destination, unsigned int length_protocol, unsigned int sum) {
	asm("addl %1,%0\n\tadcl %2,%0\n\tadcl %3,%0\n\tadcl $0,%0"
		: "=r"(sum) : "g"(destination), "g"(source), "g"(length_protocol), "0"(sum));
	return sum;
}
`), nil)
	if !result.Ok {
		t.Fatalf("checksum-add asm lowering failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("a=renvo_runtime_CChecksumAdd(uint32(a),uint32(b));"),
		[]byte("sum=renvo_runtime_CChecksumAdd3(uint32(sum),uint32(destination),uint32(source),uint32(length_protocol));"),
		[]byte("return uint32(wide)+uint32(wide>>32)"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("checksum-add source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("checksum-add source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectLowersChecksumAdd64Asm(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned long add_five(unsigned long sum, const unsigned long values[5]) {
	asm("addq %1,%0\n\tadcq %2,%0\n\tadcq %3,%0\n\tadcq %4,%0\n\tadcq %5,%0\n\tadcq $0,%0"
	    : "+r"(sum) : "m"(values[0]), "m"(values[1]), "m"(values[2]), "m"(values[3]), "m"(values[4]));
	return sum;
}
unsigned long add_one(unsigned long sum, unsigned long value) {
	asm("addq %1,%0\n\tadcq $0,%0" : "+r"(sum) : "r"(value));
	return sum;
}
unsigned long add_memory(unsigned long sum, const void *source) {
	asm("addq 0*8(%[src]),%[res]\n\tadcq 1*8(%[src]),%[res]\n\tadcq $0,%[res]"
	    : [res] "+r"(sum) : [src] "r"(source), "m"(*(const char (*)[16])source));
	return sum;
}
unsigned long add_two_addresses(unsigned long rest, const unsigned long *source, const unsigned long *destination) {
	unsigned long sum;
	asm("addq (%[saddr]),%[sum]\n\tadcq 8(%[saddr]),%[sum]\n\tadcq (%[daddr]),%[sum]\n\tadcq 8(%[daddr]),%[sum]\n\tadcq $0,%[sum]"
	    : [sum] "=r"(sum) : "[sum]"(rest), [saddr] "r"(source), [daddr] "r"(destination));
	return sum;
}
`), nil)
	if !result.Ok {
		t.Fatalf("64-bit checksum-add asm lowering failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("sum=renvo_runtime_CChecksumAdd64Five(uint64(sum),uint64("),
		[]byte("sum=renvo_runtime_CChecksumAdd64(uint64(sum),uint64(value));"),
		[]byte("sum=renvo_runtime_CChecksumAdd64Memory(uint64(sum),(*uint64)(__c_unsafe.Pointer(source)),2);"),
		[]byte("sum=renvo_runtime_CChecksumAdd64Five(uint64(rest),uint64(*(*uint64)(__c_unsafe.Pointer(source)))"),
		[]byte("uintptr(__c_unsafe.Pointer(destination))+8"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("64-bit checksum-add source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("64-bit checksum-add source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectLowersMonitorWaitAsm(t *testing.T) {
	result := TranslateObject("main", []byte(`
void monitor(const void *address, unsigned long extensions, unsigned long hints) {
	asm volatile(".byte 0x0f, 0x01, 0xc8;" :: "a"(address), "c"(extensions), "d"(hints));
}
void monitorx(const void *address, unsigned long extensions, unsigned long hints) {
	asm volatile(".byte 0x0f, 0x01, 0xfa;" :: "a"(address), "c"(extensions), "d"(hints));
}
void wait(unsigned long extensions, unsigned long hints) {
	asm volatile(".byte 0x0f, 0x01, 0xc9;" :: "a"(extensions), "c"(hints));
}
void waitx(unsigned long extensions, unsigned long counter, unsigned long hints) {
	asm volatile(".byte 0x0f, 0x01, 0xfb;" :: "a"(extensions), "b"(counter), "c"(hints));
}
void sti_wait(unsigned long extensions, unsigned long hints) {
	asm volatile("sti; .byte 0x0f, 0x01, 0xc9;" :: "a"(extensions), "c"(hints));
}
void timed_pause(unsigned int control, unsigned int high, unsigned int low) {
	asm volatile(".byte 0x66, 0x0f, 0xae, 0xf1" : : "c"(control), "d"(high), "a"(low));
}
`), nil)
	if !result.Ok {
		t.Fatalf("monitor/wait asm lowering failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("renvo_runtime_CMonitor(uintptr(__c_unsafe.Pointer(address)),uint64(extensions),uint64(hints));"),
		[]byte("renvo_runtime_CMonitorExtended(uintptr(__c_unsafe.Pointer(address)),uint64(extensions),uint64(hints));"),
		[]byte("renvo_runtime_CWait(uint64(extensions),uint64(hints));"),
		[]byte("renvo_runtime_CWaitExtended(uint64(extensions),uint64(counter),uint64(hints));"),
		[]byte("renvo_runtime_CEnableInterruptsAndWait(uint64(extensions),uint64(hints));"),
		[]byte("renvo_runtime_CTimedPause(uint64(control),uint64(high),uint64(low));"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("monitor/wait source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("monitor/wait source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectLowersInt3SelftestAsm(t *testing.T) {
	result := TranslateObject("main", []byte(`
register unsigned long current_stack_pointer asm("rsp");
void test(void) {
	unsigned int value = 0;
	asm volatile("int3_selftest_ip:\n\tint3; nop; nop; nop; nop\n\t"
		: "+r"(current_stack_pointer) : "D"(&value) : "memory");
}
`), nil)
	if !result.Ok {
		t.Fatalf("int3 selftest asm lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("renvo_runtime_CInt3Selftest(&(value));")) ||
		!bytes.Contains(result.Source, []byte("renvo_runtime_CMemoryBarrier();")) {
		t.Fatalf("int3 selftest asm did not become a typed operation:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("int3 selftest source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectLowersAlternativeRelocationSelftest(t *testing.T) {
	result := TranslateObject("main", []byte(`
register unsigned long current_stack_pointer asm("rsp");
static int target;
static void __alt_reloc_selftest(void *pointer) { if (pointer != &target) __builtin_unreachable(); }
void test(void) {
	asm volatile("# ALT: oldinstr\n.pushsection .altinstructions,\"a\"\n"
		".pushsection .altinstr_replacement, \"ax\"\n"
		"lea %[mem], %%rdi; call __alt_reloc_selftest;\n.popsection\n"
		: "+r"(current_stack_pointer) : [mem] "m"(target) : "rdi");
}
`), nil)
	if !result.Ok {
		t.Fatalf("alternative relocation selftest lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("__alt_reloc_selftest((*byte)(__c_unsafe.Pointer(&(target))));")) {
		t.Fatalf("alternative relocation selftest did not retain its relocated call:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("alternative relocation selftest source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectLowersAlternativeCPUNODEAsm(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned int read_node(void) {
	unsigned int value;
	asm volatile("# ALT: oldinstr\n lsl %[seg],%[p]\n# replacement rdpid"
		: [p] "=a"(value) : "i"(0), [seg] "r"(123));
	return value;
}
`), nil)
	if !result.Ok || !bytes.Contains(result.Source, []byte("value=renvo_runtime_CReadCPUNODE(uint32(123))")) {
		t.Fatalf("alternative CPUNODE asm lowering failed: %#v\n%s", result, result.Source)
	}
}

func TestTranslateObjectLowersReadWriteCPUIDAsm(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct cpu { unsigned int flags[1]; };
void inspect(struct cpu *cpu) {
	unsigned int level = 1;
	asm("cpuid" : "+a"(level), "=d"(cpu->flags[0]) : : "ecx", "ebx");
}
`), nil)
	if !result.Ok || !bytes.Contains(result.Source, []byte("renvo_runtime_CCPUID(level,0,&(level),&__c_cpuid_b_")) {
		t.Fatalf("read/write CPUID asm lowering failed: %#v\n%s", result, result.Source)
	}
}

func TestTranslateObjectLowersRealModeTransitionAsm(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct table_pointer { unsigned short limit; unsigned int address; };
void transition(unsigned int far_pointer, struct table_pointer *gdt, struct table_pointer *idt) {
	asm volatile("lcallw *%0" : : "m"(far_pointer) : "eax", "ebx", "ecx", "edx");
	asm volatile("cli");
	asm volatile("sti");
	asm volatile("lgdtl %0" : : "m"(*gdt));
	asm volatile("lidtl %0" : : "m"(*idt));
	asm volatile("lgdt %0" : : "m"(*gdt));
	asm volatile("lidt %0" : : "m"(*idt));
	asm volatile("lldt %w0" : : "q"(0));
	asm volatile("lldt %0" : : "rm"(0));
	asm volatile("sgdt %0" : "=m"(*gdt));
	asm volatile("sidt %0" : "=m"(*idt));
	asm volatile("ltr %w0" : : "q"(64));
	unsigned long task;
	asm volatile("str %0" : "=r"(task));
	unsigned short local_table;
	asm volatile("sldt %0" : "=g"(local_table));
	asm volatile("sldt %0" : "=m"(local_table));
}
`), nil)
	if !result.Ok {
		t.Fatalf("real-mode transition asm lowering failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("renvo_runtime_CFarCall16"),
		[]byte("renvo_runtime_CDisableInterrupts"),
		[]byte("renvo_runtime_CEnableInterrupts"),
		[]byte("renvo_runtime_CLoadGDT"),
		[]byte("renvo_runtime_CLoadIDT"),
		[]byte("renvo_runtime_CLoadLDT"),
		[]byte("renvo_runtime_CStoreGDT"),
		[]byte("renvo_runtime_CStoreIDT"),
		[]byte("renvo_runtime_CLoadTR"),
		[]byte("renvo_runtime_CStoreTR"),
		[]byte("renvo_runtime_CStoreLDT"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("real-mode transition asm lowering is missing %q:\n%s", want, result.Source)
		}
	}
}

func TestTranslateObjectCleanupAttribute(t *testing.T) {
	source := []byte(`
int cleaned;
void release(int *value);
int inspect(void) { int value __attribute__((__cleanup__(release))) = 3; return value; }
`)
	checked := CheckObjectForDataModel(source, DataModelLP64)
	if !checked.Ok {
		t.Fatalf("cleanup attribute check failed: error=%d at=%d", checked.Error, checked.ErrorAt)
	}
	lowered := TranslateObjectForDataModel("main", source, nil, DataModelLP64)
	if !lowered.Ok {
		t.Fatalf("cleanup attribute lowering failed: %#v", lowered)
	}
	if !bytes.Contains(lowered.Source, []byte("var value int32=3;\n_=&value;defer release(&value);")) {
		t.Fatalf("cleanup attribute did not become a scoped defer:\n%s", lowered.Source)
	}
	if parsed := syntax.ParseFile(lowered.Source); !parsed.Ok {
		t.Fatalf("cleanup attribute source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, lowered.Source)
	}
}

func TestCheckObjectModelsInt128CarrierTypes(t *testing.T) {
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
	if !lowered.Ok {
		t.Fatalf("int128 carrier lowering failed: %#v", lowered)
	}
	if parsed := syntax.ParseFile(lowered.Source); !parsed.Ok {
		t.Fatalf("int128 carrier unit does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, lowered.Source)
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

func TestTranslateObjectPreservesTypeofStringLiteralArray(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct note_header { unsigned int namesz, descsz, type; };
struct packed_note {
	struct note_header header;
	unsigned char name[8];
	typeof("") desc;
} __attribute__((packed));
static const struct packed_note note = { { 8, sizeof(""), 256 }, "Linux", "" };
`), nil)
	if !result.Ok {
		t.Fatalf("typeof string-literal array lowering failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("descsz:1"),
		[]byte("[1]int8{0}"),
		[]byte("=__c_object_aggregate_0_12_20_x"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("typeof string-literal array source is missing %q:\n%s", want, result.Source)
		}
	}
	if bytes.Contains(result.Source, []byte("typeof")) {
		t.Fatalf("typeof escaped into shared source:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("typeof string-literal array source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateSignedMinimumConstant(t *testing.T) {
	result := TranslateObject("main", []byte(`
long minimum(void) {
	return (-((long long)~((unsigned long long)1 << 63)) - 1);
}
`), nil)
	if !result.Ok || !bytes.Contains(result.Source, []byte("return -9223372036854775808")) {
		t.Fatalf("signed-minimum constant lowering failed: %#v\n%s", result, result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("signed-minimum source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateSelfReferentialLocalAggregateInitializer(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct pair { int *self; int value; };
int build(void) {
	struct pair pair = {
		&pair.value,
		*({ pair.value = 7; &pair.value; }),
	};
	return pair.value;
}
`), nil)
	if !result.Ok {
		t.Fatalf("self-referential aggregate initializer failed: %#v", result)
	}
	for _, want := range [][]byte{
		[]byte("var pair __c_struct_pair;"),
		[]byte("_=&pair;pair=__c_struct_pair{"),
		[]byte("func __c_statement_expr_"),
		[]byte("*__c_struct_pair"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("self-referential aggregate source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("self-referential aggregate source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
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
	if !lowered.Ok || !bytes.Contains(lowered.Source, []byte("case 0,1,2,3,4,5,6,7,8,9:")) {
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

func TestTranslateDynamicOffsetofArrayDesignator(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct item { long value; };
struct collection { struct item node[2]; };
unsigned long inspect(struct item *node, int index) {
	(void)node;
	return __builtin_offsetof(struct collection, node[index]);
}
`), nil)
	if !result.Ok {
		t.Fatalf("dynamic offsetof translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if bytes.Contains(result.Source, []byte("__builtin_offsetof")) || !bytes.Contains(result.Source, []byte("uint64(index)")) {
		t.Fatalf("dynamic offsetof designator was not lowered independently of the colliding local name:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("dynamic offsetof source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
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

func TestTranslateCastedPointerMemberAccess(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct thread_info { unsigned long flags; };
struct task { unsigned long unrelated; };
struct task *get_current(void);
unsigned long inspect(void) { return ((struct thread_info *)get_current())->flags; }
`), nil)
	if !result.Ok {
		t.Fatalf("casted pointer member translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("return ((*__c_struct_thread_info)(get_current())).flags;")) ||
		bytes.Contains(result.Source, []byte("struct thread_info")) {
		t.Fatalf("casted pointer member retained C-only syntax:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("casted pointer member source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
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

func TestTranslateNestedMemberAfterIndirectAggregateUsesAddress(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct inner { unsigned value; };
struct outer { struct inner inner; } __attribute__((aligned(16)));
void set(struct outer *outer) { outer->inner.value = 7; }
`), nil)
	if !result.Ok {
		t.Fatalf("nested indirect member translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte(".__c_nested_ptr_")) ||
		!bytes.Contains(result.Source, []byte("outer.__c_ptr_0_inner()")) {
		t.Fatalf("nested indirect member did not preserve an addressable receiver:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("nested indirect member source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateCallMemberAfterIndirectAggregateUsesAccessor(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct context { void *image; };
struct mm { union { struct { struct context context; }; }; };
struct task { struct mm *mm; };
extern struct task *current(void);
void *image(void) { return current()->mm->context.image; }
`), nil)
	if !result.Ok {
		t.Fatalf("call member translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("current()).__c_nested_ptr_")) ||
		!bytes.Contains(result.Source, []byte("__c_ptr_0_context()")) ||
		!bytes.Contains(result.Source, []byte("__c_nested_ptr_13_0_image()")) ||
		bytes.Contains(result.Source, []byte("current().mm.context")) {
		t.Fatalf("call-rooted indirect member bypassed its accessor:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("call member source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateStatementExpressionPointerMemberAfterIndirectAggregate(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct indirect { unsigned long flags; } __attribute__((aligned(16)));
unsigned long *flags(struct indirect *value) {
	return &({ (void)sizeof(*value); value; })->flags;
}
`), nil)
	if !result.Ok {
		t.Fatalf("statement-expression indirect member translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte(").__c_ptr_0_flags())")) {
		t.Fatalf("statement-expression indirect member did not use its pointer accessor:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("statement-expression indirect member source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateElidesConstantShortCircuitBranch(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct value { long member; };
struct value first, second;
long selected(void) {
	if (0 && ((void *)&first - (void *)&second))
		return first.member;
	else
		return 7;
}
long literal(void) {
	if (0) first.member = 99;
	else return 8;
}
`), nil)
	if !result.Ok {
		t.Fatalf("constant branch translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if bytes.Contains(result.Source, []byte("__c_pointer_diff_")) || bytes.Contains(result.Source, []byte("=99")) ||
		!bytes.Contains(result.Source, []byte("return 7;")) || !bytes.Contains(result.Source, []byte("return 8;")) {
		t.Fatalf("constant-false branch was not eliminated:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("constant branch source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateSeparatesIfFromFollowingLoweredDoBlock(t *testing.T) {
	result := TranslateObject("main", []byte(`
void update(int active) {
	if (active) active = 0;
	do { active = 1; } while (0);
}
`), nil)
	if !result.Ok {
		t.Fatalf("if/do translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("};\n{var __c_do_first_")) {
		t.Fatalf("if and following lowered do block were not explicitly separated:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("if/do source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateVoidFunctionReturnsVoidExpression(t *testing.T) {
	result := TranslateObject("main", []byte(`
void initialize(int value) { (void)value; }
void initialize_default(void) { return initialize(7); }
`), nil)
	if !result.Ok {
		t.Fatalf("void-expression return translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("func initialize_default() {initialize(7);return;")) {
		t.Fatalf("void-expression return was not sequenced before a bare return:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("void-expression return source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateStaticIndirectZeroInitializerUsesZeroValue(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct descriptor { unsigned short limit; unsigned int address; } __attribute__((packed));
void load(void) { static const struct descriptor empty = {0, 0}; (void)&empty; }
`), nil)
	if !result.Ok {
		t.Fatalf("static indirect zero initializer translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if bytes.Contains(result.Source, []byte("_empty __c_struct_descriptor=")) {
		t.Fatalf("static indirect zero initializer was not represented by its zero value:\n%s", result.Source)
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
		[]byte("func(p *__c_union_word)__c_ptr_0_whole()*uint32"),
		[]byte("func(p *__c_struct_flags)__c_get_delta() int32"),
		[]byte("flags.__c_set_low(flags.__c_get_low()+1)"),
		[]byte("(*word_cursor.__c_ptr_0_whole())=67305985"),
		[]byte("(*word.__c_ptr_0_bytes())[0]"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("translated union/bitfield source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated union/bitfield source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectScalarUnionCastPreservesBits(t *testing.T) {
	result := TranslateObject("main", []byte(`
union control {
	unsigned long value;
	struct { unsigned long count : 16; unsigned long enabled : 1; };
};
unsigned long count(unsigned long config) {
	union control control = (union control)config;
	return control.count;
}
`), nil)
	if !result.Ok {
		t.Fatalf("scalar union cast translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("control=__c_union_control{__c_align:config}")) {
		t.Fatalf("scalar union cast did not use raw storage bits:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("scalar union cast source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslatePromotedAnonymousBitfieldUsesOffsetStorage(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct controls {
	union {
		unsigned long capabilities;
		struct { unsigned long enabled : 1; unsigned long ready : 1; };
	};
};
void enable(struct controls *controls) { controls->enabled = 1; }
`), nil)
	if !result.Ok {
		t.Fatalf("promoted bitfield translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("func(p *__c_struct_controls)__c_set_enabled(v uint64){q:=(*uint64)")) ||
		bytes.Contains(result.Source, []byte("func(p *__c_struct_controls)__c_set_enabled(v uint64){p.__c_bits_")) {
		t.Fatalf("promoted bitfield did not use its containing offset:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("promoted bitfield source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateIndexedBitfieldMutationUsesSetter(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct branch { unsigned long mispred : 1; unsigned long predicted : 1; };
struct branch_table { struct branch entries[4]; };
void update(struct branch *branches, int index, unsigned long value) {
	branches[index].mispred = value;
	branches[index].predicted ^= 1;
}
void update_nested(struct branch_table *table, int index) {
	table->entries[index].mispred = 1;
}
`), nil)
	if !result.Ok {
		t.Fatalf("indexed bitfield mutation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte(")).__c_set_mispred(value)")) ||
		!bytes.Contains(result.Source, []byte(")).__c_set_predicted(")) ||
		!bytes.Contains(result.Source, []byte("table.entries[index].__c_set_mispred(1)")) ||
		bytes.Contains(result.Source, []byte(".__c_get_mispred()=value")) {
		t.Fatalf("indexed bitfield mutation did not use setters:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("indexed bitfield source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateConditionalPointerBitfieldMutationUsesSetter(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct entry { unsigned base : 12; unsigned end : 1; };
void update(struct entry *first, struct entry *second, int select) {
	(select ? first : second)->base = 7;
	(select ? first : second)->end = 1;
}
`), nil)
	if !result.Ok {
		t.Fatalf("conditional pointer bitfield mutation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte(".__c_set_base(7)")) ||
		!bytes.Contains(result.Source, []byte(".__c_set_end(1)")) ||
		bytes.Contains(result.Source, []byte(".__c_get_base()=7")) ||
		bytes.Contains(result.Source, []byte(".__c_get_end()=1")) {
		t.Fatalf("conditional pointer bitfield mutation did not use setters:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("conditional pointer bitfield source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateUnaryOperatorMemberPrecedence(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct masks { unsigned long guest; };
unsigned long invert_guest(struct masks *masks) { return ~masks->guest; }
struct callbacks { void (*run)(void); };
int callback_ready(struct callbacks *callbacks) { return !!callbacks->run; }
`), nil)
	if !result.Ok {
		t.Fatalf("unary member translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if bytes.Contains(result.Source, []byte("__c_struct_masks(-masks-1)")) ||
		!bytes.Contains(result.Source, []byte("uint64(-masks.guest-1)")) {
		t.Fatalf("unary operator did not retain the complete member operand:\n%s", result.Source)
	}
	if !bytes.Contains(result.Source, []byte("callbacks.run)!=nil")) {
		t.Fatalf("function-pointer condition did not compare with nil:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated unary member source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateArrayLvalueDecayForArrowMember(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct pgd { unsigned long value; };
struct pgd table[2];
struct pgd (*tables)[2];
unsigned int next;
void *relative_pointer(void *pointer);
unsigned long *first_value(void) {
	return &(*(typeof(&(table)))relative_pointer(&(table)))->value;
}
unsigned long *indexed_value(void) {
	return &tables[(*(typeof(&(next)))relative_pointer(&(next)))++]->value;
}
`), nil)
	if !result.Ok {
		t.Fatalf("array-lvalue arrow member translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if bytes.Count(result.Source, []byte(")[0].value")) != 2 ||
		!bytes.Contains(result.Source, []byte("__c_scalar_post_inc_")) || bytes.Contains(result.Source, []byte("typeof")) {
		t.Fatalf("array lvalue did not decay to its first element for arrow member access:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("array-lvalue arrow member source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslatePostfixMutationThroughTypeofPointerCast(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned int next;
void *relative_pointer(void *pointer);
unsigned int take_next(void) {
	return (*(typeof(&(next)))relative_pointer(&(next)))++;
}
`), nil)
	if !result.Ok {
		t.Fatalf("typeof pointer-cast postfix mutation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("__c_scalar_post_inc_")) || bytes.Contains(result.Source, []byte("typeof")) {
		t.Fatalf("typeof pointer-cast postfix mutation was not lowered:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("typeof pointer-cast postfix source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateForPostCommaMutations(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct cursor { unsigned long position; };
unsigned long scan(struct cursor *cursor, char *data) {
	unsigned long offset = 0;
	for (; offset < 4096 && data[offset]; offset++, cursor->position++)
		;
	return offset + cursor->position;
}
`), nil)
	if !result.Ok {
		t.Fatalf("for post comma translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("__c_comma_")) ||
		!bytes.Contains(result.Source, []byte("__c_scalar_post_inc_")) ||
		!bytes.Contains(result.Source, []byte(") uint64{")) ||
		bytes.Contains(result.Source, []byte("&(__c_comma_")) {
		t.Fatalf("for post comma expression did not retain its separate mutations:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated for post comma source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateIncompleteExternalArraySubscriptUsesPointerIndex(t *testing.T) {
	result := TranslateObject("main", []byte(`
extern unsigned long values[];
unsigned long read_value(void) { return values[511]; }
void write_value(unsigned long value) { values[511] = value; }
`), nil)
	if !result.Ok {
		t.Fatalf("incomplete external array subscript failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if bytes.Count(result.Source, []byte("__c_pointer_index_")) < 3 ||
		bytes.Contains(result.Source, []byte("values[511]")) {
		t.Fatalf("incomplete external array retained bounded indexing:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("incomplete external array source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
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
		[]byte("case 0:total=total+1;"),
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
		[]byte("var sparse [5]int32=[5]int32{3:9,4:10}"),
		[]byte("var inferred [4]int32=[4]int32{2:8,3:9}"),
		[]byte("p[0]=v0;p[0]=v1"),
		[]byte("func __c_object_union_init_"),
		[]byte("__c_union_word{__c_align:67305985}"),
		[]byte("(*p.__c_ptr_0_bytes())=v"),
		[]byte("__c_bits_5:(17&((uint32(1)<<5)-1))<<3"),
		[]byte("__c_bits_5:(6&((uint32(1)<<3)-1))"),
		[]byte("value:2"),
		[]byte("label:[3]int8{111,107,0},wrapped:44"),
		[]byte("__c_struct_text{wrapped:45,label:[3]int8{104,105,0}}"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("aggregate initializer source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated aggregate initializer does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectMarksNestedUnionConstantHelpers(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct pair { unsigned int first; unsigned int second; };
union holder { struct pair pair; unsigned long word; };
struct root { union holder holder; unsigned int tail; };
static struct root root = { .holder = { .pair = { 7, 8 } }, .tail = 9 };
`), nil)
	if !result.Ok {
		t.Fatalf("nested union constant translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("func __c_object_union_init_")) {
		t.Fatalf("nested union constant helper lacks its object serialization marker:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("nested union constant source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslatePromotedUnionAggregateHelper(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef _Bool bool;
struct iterator {
	unsigned char type;
	bool nofault : 1;
	bool source : 1;
	unsigned long offset;
	union { void *ubuf; unsigned long word; };
	union {
		struct {
			unsigned long count;
			union { unsigned long nr_segs; long position; };
		};
		unsigned long alignment[2];
	};
};
struct iterator make_iterator(void *buffer, unsigned long count) {
	return (struct iterator){ .type = 1, .source = 1, .ubuf = buffer, .count = count, .nr_segs = 1 };
}
`), nil)
	if !result.Ok {
		t.Fatalf("promoted union aggregate helper failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("__c_aggregate_init_")) ||
		!bytes.Contains(result.Source, []byte(".__c_ptr_")) ||
		bytes.Contains(result.Source, []byte("p.ubuf=")) {
		t.Fatalf("promoted union initializer bypassed its offset accessor:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("promoted union aggregate source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslatePositionalAnonymousUnionCompoundLiteral(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef unsigned int u32;
typedef unsigned long u64;
typedef unsigned long size_t;
size_t strlen(const char *name);
struct qstr {
	union { struct { u32 hash; u32 len; }; u64 hash_len; };
	const unsigned char *name;
};
int consume(struct qstr *name);
int inspect(const char *name) {
	return consume(&(struct qstr){ { { .len = strlen(name) } }, .name = (const unsigned char *)name });
}
`), nil)
	if !result.Ok {
		t.Fatalf("anonymous-union compound literal failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("__c_object_union_init_4_x")) || bytes.Contains(result.Source, []byte("__c_struct_qstr({")) {
		t.Fatalf("anonymous-union compound literal did not retain its union initializer:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("anonymous-union compound literal source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
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

func TestTranslateVariadicFixedArrayArgumentDecays(t *testing.T) {
	result := TranslateObject("main", []byte(`
int sprintf(char *buffer, const char *format, ...);
int inspect(int value) { static char buffer[16]; return sprintf(buffer, "%s %d", buffer, value); }
`), nil)
	if !result.Ok {
		t.Fatalf("variadic array argument translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("__c_variadic_")) ||
		bytes.Count(result.Source, []byte("&(__c_static_")) < 2 {
		t.Fatalf("variadic fixed array argument did not decay:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("variadic fixed array source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
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

func TestTranslateObjectConstantConditionalSkipsRuntimeArm(t *testing.T) {
	result := TranslateObject("main", []byte(`
extern int missing(void);
int selected(void) {
	return (__builtin_constant_p(100) && 100 <= 5) ? missing() : 42;
}
`), nil)
	if !result.Ok {
		t.Fatalf("constant conditional translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if bytes.Count(result.Source, []byte("missing()")) != 1 {
		t.Fatalf("constant conditional retained its runtime arm:\n%s", result.Source)
	}
}

func TestTranslateObjectConstantLoopConditionSkipsRuntimeOperand(t *testing.T) {
	result := TranslateObject("main", []byte(`
extern int missing(void);
void check(void) {
	do { } while (0 && missing());
	while (0 && missing()) { }
}
`), nil)
	if !result.Ok {
		t.Fatalf("constant loop condition translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if bytes.Count(result.Source, []byte("missing()")) != 1 {
		t.Fatalf("constant loop retained a runtime operand:\n%s", result.Source)
	}
}

func TestTranslateObjectInlinesSideEffectFreeConstantReturnFunction(t *testing.T) {
	result := TranslateObject("main", []byte(`
static inline _Bool feature_enabled(void *value) { return 0; }
static inline _Bool wrapped_feature(void *value) { return __builtin_expect(!!feature_enabled(value), 0); }
struct feature_state { unsigned int flags; };
static inline _Bool masked_feature(struct feature_state *state) { return state->flags & 0; }
extern int unavailable(void);
int choose(void *value) {
	if (wrapped_feature(value))
		return unavailable();
	return 7;
}
extern int observe(void);
int preserve_condition_effect(void *value) {
	if (observe() && wrapped_feature(value))
		return unavailable();
	return 9;
}
int choose_masked(struct feature_state *state) {
	if (masked_feature(state))
		return unavailable();
	return 11;
}
`), nil)
	if !result.Ok {
		t.Fatalf("TranslateObject failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if bytes.Contains(result.Source, []byte("if (wrapped_feature(value))")) || bytes.Contains(result.Source, []byte("if wrapped_feature(value)")) {
		t.Fatalf("constant inline call was retained:\n%s", result.Source)
	}
	if bytes.Contains(result.Source, []byte("if (uint8(0))!=0")) || bytes.Count(result.Source, []byte("unavailable()")) != 1 {
		t.Fatalf("constant inline false branch was not removed:\n%s", result.Source)
	}
	if bytes.Count(result.Source, []byte("observe()")) != 2 {
		t.Fatalf("constant suffix pruning did not preserve the evaluated prefix:\n%s", result.Source)
	}
}

func TestCheckHashedNamespacesAndTypeCachesAcrossScopes(t *testing.T) {
	source := []byte(`
typedef unsigned long word;
struct word { int value; };
int target(const volatile word *value);

int first(const volatile word *value) {
	word target = *value;
	{
		typedef int word;
		word inner = 1;
		target += inner;
	}
	return target;
}

int second(const volatile word *value) {
	struct word local = { (int)*value };
	return local.value;
}
`)
	result := CheckObjectForDataModel(source, DataModelLP64)
	if !result.Ok {
		t.Fatalf("hashed semantic check failed: error=%d at=%d", result.Error, result.ErrorAt)
	}

	invalid := CheckObjectForDataModel([]byte(`typedef int value; int value;`), DataModelLP64)
	if invalid.Ok || invalid.Error != TranslateErrDeclaration {
		t.Fatalf("same-scope ordinary namespace collision = %#v, want declaration error", invalid)
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

func TestTranslateGenericSelectionExpression(t *testing.T) {
	result := TranslateObject("main", []byte(`
int side;
int inspect(unsigned long value) {
	return _Generic(value, int: (side = 1), unsigned long: 7, default: (side = 2));
}
`), nil)
	if !result.Ok {
		t.Fatalf("generic selection lowering failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("return 7;")) || bytes.Contains(result.Source, []byte("_Generic")) ||
		bytes.Contains(result.Source, []byte("side=1")) || bytes.Contains(result.Source, []byte("side=2")) {
		t.Fatalf("generic selection did not emit only its selected association:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("generic selection source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateFoldedNegativeBinaryOperand(t *testing.T) {
	result := TranslateObject("main", []byte(`
enum kind { sentinel = 0xff };
int inspect(int value) { return value < (sentinel << 24); }
`), nil)
	if !result.Ok {
		t.Fatalf("negative binary operand lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("value<(-16777216)")) || bytes.Contains(result.Source, []byte("value<-")) {
		t.Fatalf("negative operand merged into a Go channel token:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("negative binary operand source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateUnaryMinusCallKeepsCallPrecedence(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned long long source(void);
long inspect(void) { return -source(); }
`), nil)
	if !result.Ok {
		t.Fatalf("unary call lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("return int64(uint64(-(source())));")) {
		t.Fatalf("unary minus did not remain outside the call:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("unary call source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateFixedRegisterMoveAsm(t *testing.T) {
	result := TranslateObject("main", []byte(`
void load_r9(unsigned long *address) {
	asm volatile("mov %[address], %%r9" :: [address] "g" (*address) : "r9", "memory");
}
`), nil)
	if !result.Ok {
		t.Fatalf("fixed-register asm lowering failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("func renvo_runtime_CLoadR9(value uint64){}"),
		[]byte("renvo_runtime_CLoadR9(uint64(*(address)))"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("fixed-register asm source is missing %q:\n%s", want, result.Source)
		}
	}
}

func TestTranslateLoadGSAsm(t *testing.T) {
	result := TranslateObject("main", []byte(`
void load_gs(unsigned selector) {
	asm volatile(".byte 0xf2,0x0f,0x00,0xf7" : "+D" (selector));
}
`), nil)
	if !result.Ok {
		t.Fatalf("load-gs asm translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("renvo_runtime_CLoadGS(uint32(selector));")) {
		t.Fatalf("load-gs asm did not use its backend intrinsic:\n%s", result.Source)
	}
}

func TestTranslateUint128MultiplyAddShift(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned long scale(unsigned long value, unsigned multiplier, unsigned long addend, unsigned shift) {
	return ((unsigned __int128)value * multiplier + addend) >> shift;
}
unsigned long scale_without_addend(unsigned long value, unsigned multiplier, unsigned shift) {
	return ((unsigned __int128)value * multiplier) >> shift;
}
`), nil)
	if !result.Ok {
		t.Fatalf("uint128 multiply-add-shift failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("renvo_runtime_CMultiplyAddShift64(uint64(value),uint64(multiplier),uint64(addend),uint32(shift))")) {
		t.Fatalf("uint128 multiply-add-shift did not use its backend primitive:\n%s", result.Source)
	}
	if !bytes.Contains(result.Source, []byte("renvo_runtime_CMultiplyAddShift64(uint64(value),uint64(multiplier),uint64(0),uint32(shift))")) {
		t.Fatalf("uint128 multiply-shift did not use a zero addend:\n%s", result.Source)
	}
}

func TestTranslateGenericSelectionCallee(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct sequence { unsigned value; };
unsigned get(struct sequence *pointer);
unsigned inspect(struct sequence *pointer) {
	return _Generic(*pointer, struct sequence: get)(pointer);
}
`), nil)
	if !result.Ok {
		t.Fatalf("generic callee lowering failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("return get(pointer);")) || bytes.Contains(result.Source, []byte("_Generic")) {
		t.Fatalf("generic callee did not select its function:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("generic callee source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateStatementExpressionAfterSwitch(t *testing.T) {
	result := TranslateObject("main", []byte(`
int inspect(int value) {
	return ({ int result = value; switch (sizeof(value)) { case 4: result += 2; break; default: result = 0; } result; });
}
`), nil)
	if !result.Ok {
		t.Fatalf("statement expression after switch failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("switch 4{case 4:result=result+2;")) ||
		!bytes.Contains(result.Source, []byte("return result}")) || bytes.Contains(result.Source, []byte("return switch")) {
		t.Fatalf("statement expression confused its switch with the final value:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("statement expression after switch does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateStatementExpressionHoistsNonlocalGoto(t *testing.T) {
	result := TranslateObject("main", []byte(`
int consume(int);
int read_value(int *cursor, int *end) {
	int value;
	consume(({ if (cursor + 1 > end) goto err_out; value = *cursor++; value; }));
	return 0;
err_out:
	return -1;
}
`), nil)
	if !result.Ok {
		t.Fatalf("nonlocal statement-expression goto failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	guard := []byte("if __c_pointer_step_inc_")
	if !bytes.Contains(result.Source, guard) || !bytes.Contains(result.Source, []byte("{goto err_out}")) {
		t.Fatalf("statement-expression goto guard was not hoisted:\n%s", result.Source)
	}
	helper := bytes.Index(result.Source, []byte("func __c_statement_expr_"))
	if helper < 0 || bytes.Contains(result.Source[helper:], []byte("goto err_out")) {
		t.Fatalf("statement-expression helper retained its nonlocal goto:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("nonlocal-goto source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateDiscardedVoidStatementExpression(t *testing.T) {
	result := TranslateObject("main", []byte(`
void increment(int *value) {
	({ asm volatile("incl %[var]" : [var] "+m"(*value)); });
}
`), nil)
	if !result.Ok {
		t.Fatalf("discarded statement expression lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("renvo_runtime_CAtomicInc32")) || bytes.Contains(result.Source, []byte("return asm")) {
		t.Fatalf("discarded statement expression did not lower its asm statement:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("discarded statement expression source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateVoidStatementExpressionEndingInWhile(t *testing.T) {
	result := TranslateObject("main", []byte(`
void delay(void);
void wait(int quick) {
	quick ? ({ if (quick) delay(); else delay(); }) : ({ unsigned long remaining = 2; while (remaining--) ({ delay(); }); });
}
`), nil)
	if !result.Ok {
		t.Fatalf("void statement expression ending in while failed: %#v", result)
	}
	if bytes.Contains(result.Source, []byte("return while")) || !bytes.Contains(result.Source, []byte("for (__c_scalar_post_dec_")) {
		t.Fatalf("void statement expression treated while as a value:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("void while statement-expression source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateNestedInitializerDoesNotCaptureItself(t *testing.T) {
	result := TranslateObject("main", []byte(`
int inspect(int seed) {
	int next = ({ int value = ({ seed; }); asm volatile("" : : : "memory"); value; });
	return next;
}
`), nil)
	if !result.Ok {
		t.Fatalf("nested initializer lowering failed: %#v", result)
	}
	if bytes.Contains(result.Source, []byte("__c_statement_expr_1(&seed,&next)")) {
		t.Fatalf("nested initializer captured the variable being initialized:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("nested initializer source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateStatementExpressionCapturesOnlyReferencedLocals(t *testing.T) {
	result := TranslateObject("main", []byte(`
int inspect(int seed) {
	int unrelated = 0;
	int used = seed;
	int result = ({ used++; used; });
	return result + unrelated;
}
`), nil)
	if !result.Ok {
		t.Fatalf("selective statement-expression capture failed: %#v", result)
	}
	call := bytes.Index(result.Source, []byte("__c_statement_expr_"))
	if call < 0 {
		t.Fatalf("statement-expression helper was not emitted:\n%s", result.Source)
	}
	close := bytes.IndexByte(result.Source[call:], ')')
	if close < 0 || !bytes.Contains(result.Source[call:call+close], []byte("(&used")) ||
		bytes.Contains(result.Source[call:call+close], []byte("seed")) ||
		bytes.Contains(result.Source[call:call+close], []byte("unrelated")) {
		t.Fatalf("statement expression captured unrelated locals:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("selective-capture source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
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

func TestTranslateGNUArrayRangeDesignators(t *testing.T) {
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
	if !lowered.Ok {
		t.Fatalf("array range lowering failed: %#v", lowered)
	}
	for _, want := range [][]byte{[]byte("1:"), []byte("2:"), []byte("3:"), []byte("7:")} {
		if !bytes.Contains(lowered.Source, want) {
			t.Fatalf("array range lowering is missing index %q:\n%s", want, lowered.Source)
		}
	}
	if parsed := syntax.ParseFile(lowered.Source); !parsed.Ok {
		t.Fatalf("array range source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, lowered.Source)
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
	if !lowered.Ok || !bytes.Contains(lowered.Source, []byte("__c_conditional_value_")) {
		t.Fatalf("omitted conditional lowering result = %#v\n%s", lowered, lowered.Source)
	}
	if parsed := syntax.ParseFile(lowered.Source); !parsed.Ok {
		t.Fatalf("omitted conditional source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, lowered.Source)
	}
}

func TestTranslateAtomicAsmThroughTypeofPointerCast(t *testing.T) {
	source := []byte(`
typedef unsigned long uintptr_t;
typedef int atomic_long_t;
void increment(atomic_long_t *value) {
	asm volatile ("inc" "q " "%" "[var]" : [var] "+m" ((*(typeof(*value) *)(uintptr_t)(value))));
}
`)
	result := TranslateObjectForDataModel("main", source, nil, DataModelLP64)
	if !result.Ok {
		t.Fatalf("atomic asm translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("renvo_runtime_CAtomicInc64")) {
		t.Fatalf("atomic asm helper missing from translation:\n%s", result.Source)
	}
}

func TestTranslateGetUserAsm(t *testing.T) {
	source := []byte(`
typedef unsigned long uintptr_t;
typedef unsigned long u64;
int get_byte(const unsigned char *address, unsigned char *output) {
	int result;
	u64 value;
	uintptr_t stack;
	asm volatile("call __get_user_%c[size]"
		: "=a" (result), "=r" (value), "+r" (stack)
		: "0" (address), [size] "i" (sizeof(*address)));
	*output = (unsigned char)value;
	return result;
}
int get_word_nocheck(const unsigned long *address, unsigned long *output) {
	int result;
	u64 value;
	uintptr_t stack;
	asm volatile("call __get_user_nocheck_%c[size]"
		: "=a" (result), "=r" (value), "+r" (stack)
		: "0" (address), [size] "i" (sizeof(*address)));
	*output = value;
	return result;
}
`)
	result := TranslateObjectForDataModel("main", source, nil, DataModelLP64)
	if !result.Ok {
		t.Fatalf("get_user asm translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("renvo_runtime_CGetUser1")) ||
		!bytes.Contains(result.Source, []byte("renvo_runtime_CGetUser8")) {
		t.Fatalf("get_user helper missing from translation:\n%s", result.Source)
	}
}

func TestTranslateInlineGetUserAsm(t *testing.T) {
	source := []byte(`
struct __large_struct { unsigned long buf[100]; };
int get_word(unsigned long *address, unsigned long *output) {
	int error;
	unsigned long value;
	asm volatile("1: movq %[umem],%[output]\n2:\n"
		".pushsection \"__ex_table\",\"a\"\n"
		".long (1b) - .\n.long (2b) - .\n"
		"extable_type_reg reg=%[errout], type=(17 | ((-14) << 16))\n"
		".popsection\n"
		: [errout] "=r" (error), [output] "=a" (value)
		: [umem] "m" (*(struct __large_struct *)address), "0" (error));
	*output = value;
	return error;
}
`)
	result := TranslateObjectForDataModel("main", source, nil, DataModelLP64)
	if !result.Ok {
		t.Fatalf("inline get_user asm translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("renvo_runtime_CGetUser8")) || bytes.Contains(result.Source, []byte("asm")) {
		t.Fatalf("inline get_user helper missing from translation:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("inline get_user source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslatePutUserAsm(t *testing.T) {
	source := []byte(`
typedef unsigned long uintptr_t;
typedef unsigned long u64;
int put_word(unsigned int *address, unsigned int value) {
	int result;
	uintptr_t stack;
	asm volatile("call __put_user_%c[size]"
		: "=c" (result), "+r" (stack)
		: "0" (address), "r" (value), [size] "i" (sizeof(*address)) : "ebx");
	return result;
}
`)
	result := TranslateObjectForDataModel("main", source, nil, DataModelLP64)
	if !result.Ok {
		t.Fatalf("put_user asm translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("result=int32(renvo_runtime_CPutUser4(uintptr(__c_unsafe.Pointer(address)),uint64(value)))")) {
		t.Fatalf("put_user helper missing from translation:\n%s", result.Source)
	}
}

func TestTranslateAlternativeMMIOStore(t *testing.T) {
	source := []byte(`
void write_register(volatile unsigned int *address, unsigned int value) {
	asm volatile("# ALT: oldinstr\n771:\n\tmovl %0, %1\n"
		".pushsection .altinstructions,\"a\"\n.popsection\n"
		".pushsection .altinstr_replacement,\"ax\"\n774:\n\txchgl %0, %1\n775:\n.popsection\n"
		: "=r" (value), "=m" (*address) : "i" (0), "0" (value), "m" (*address));
}
`)
	result := TranslateObjectForDataModel("main", source, nil, DataModelLP64)
	if !result.Ok {
		t.Fatalf("alternative store translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("*(address)=uint32(value)")) {
		t.Fatalf("alternative baseline store missing from translation:\n%s", result.Source)
	}
}

func TestTranslateAsmBoundsMask(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned long array_index_mask(unsigned long index, unsigned long size) {
	unsigned long mask;
	asm volatile("cmp %1,%2; sbb %0,%0" : "=r"(mask) : "g"(size), "r"(index) : "cc");
	return mask;
}
`), nil)
	if !result.Ok || !bytes.Contains(result.Source, []byte("renvo_runtime_CBoundsMask(uint64(index),uint64(size))")) {
		t.Fatalf("bounds-mask asm lowering failed: %#v\n%s", result, result.Source)
	}
}

func TestCheckBlockExternRedeclaredAtFileScope(t *testing.T) {
	result := CheckObjectForDataModel([]byte(`
void release(void) { extern char begin[], end[]; }
int (*factory(void))(int);
extern char begin[], end[];
`), DataModelLP64)
	if !result.Ok {
		t.Fatalf("compatible block/file extern redeclaration failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
}

func TestTranslateMergesBlockScopeFunctionDeclarations(t *testing.T) {
	result := TranslateObject("main", []byte(`
void first(void) { extern void failure(void); }
void second(void) { extern void failure(void); }
`), nil)
	if !result.Ok {
		t.Fatalf("block-scope function declarations failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if bytes.Count(result.Source, []byte("func failure()")) != 1 {
		t.Fatalf("external function was emitted more than once:\n%s", result.Source)
	}
}

func TestTranslateUsesStaticForwardPrototypeTypes(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef unsigned long word_t;
static word_t select_word(word_t value);
word_t use_word(void) { return select_word((word_t)1); }
static word_t select_word(word_t value) { return value; }
`), nil)
	if !result.Ok {
		t.Fatalf("static forward prototype translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if bytes.Contains(result.Source, []byte("word_t")) || !bytes.Contains(result.Source, []byte("select_word(1)")) {
		t.Fatalf("static forward call did not use its prototype:\n%s", result.Source)
	}
}

func TestTranslateFunctionPointerCallArguments(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef int s32;
void apply(void (*fn)(void *, unsigned long), unsigned long value, s32 *entry) {
	fn(*entry + (void *)entry, value);
}
void invoke(void (*release)(void *, void *), void *device, void *resource) {
	(*(release))(device, resource);
}
`), nil)
	if !result.Ok {
		t.Fatalf("function-pointer call translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if bytes.Contains(result.Source, []byte("void")) || !bytes.Contains(result.Source, []byte("__c_pointer_step_inc_")) {
		t.Fatalf("function-pointer arguments were not lowered:\n%s", result.Source)
	}
	if bytes.Contains(result.Source, []byte("(*(release))")) || !bytes.Contains(result.Source, []byte("release(device,resource)")) {
		t.Fatalf("parenthesized function-pointer dereference was not normalized:\n%s", result.Source)
	}
}

func TestTranslateParenthesizedCompoundLiteralCallArgument(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef struct { unsigned long value; } word_t;
static word_t drop(word_t left, word_t right) { left.value &= ~right.value; return left; }
word_t apply(word_t value) { return drop(value, ((word_t){ 7 })); }
`), nil)
	if !result.Ok {
		t.Fatalf("compound literal call translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if bytes.Contains(result.Source, []byte("word_t")) || !bytes.Contains(result.Source, []byte("__c_struct_anon_")) {
		t.Fatalf("compound literal call argument was not lowered:\n%s", result.Source)
	}
}

func TestTranslateBuiltinAndDeclaredStringLength(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef unsigned long size_t;
size_t strlen(const char *value);
size_t lengths(const char *value) { return __builtin_strlen(value) + strlen(value); }
size_t literal(void) { return strlen("port@"); }
`), nil)
	if !result.Ok {
		t.Fatalf("string length translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if bytes.Count(result.Source, []byte("renvo_runtime_CStringLength((*int8)(value))")) != 2 {
		t.Fatalf("string lengths did not use the typed helper:\n%s", result.Source)
	}
	if !bytes.Contains(result.Source, []byte("renvo_runtime_CStringLength(renvo_runtime_CStringPointer(\"\\x70\\x6f\\x72\\x74\\x40\\x00\"))")) {
		t.Fatalf("string-literal length did not use relocatable string storage:\n%s", result.Source)
	}
}

func TestTranslateBuiltinFrameAddresses(t *testing.T) {
	result := TranslateObject("main", []byte(`
void *frame(void) { return __builtin_frame_address(0); }
void *caller(void) { return __builtin_return_address(0); }
`), nil)
	if !result.Ok {
		t.Fatalf("builtin address translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, helper := range [][]byte{[]byte("renvo_runtime_CFrameAddress()"), []byte("renvo_runtime_CReturnAddress()")} {
		if !bytes.Contains(result.Source, helper) {
			t.Fatalf("builtin address helper %q is missing:\n%s", helper, result.Source)
		}
	}
}

func TestTranslateIndirectMemberThroughStatementExpression(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct page { int entries[16]; } __attribute__((aligned(4096)));
extern struct page page;
int *entries(void) { return (*({ (struct page *)&page; })).entries; }
`), nil)
	if !result.Ok {
		t.Fatalf("indirect statement-expression member failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if bytes.Contains(result.Source, []byte("({")) || !bytes.Contains(result.Source, []byte("__c_ptr_0_entries")) {
		t.Fatalf("indirect member was not lowered through its accessor:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("indirect member source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateNestedStatementExpressionMemberType(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct page { int value; };
struct folio { struct page page; };
struct tag;
extern struct folio *allocate(void);
struct page *inspect(void) {
	return &({ ; ({ struct tag *old = (void *)0; typeof(allocate()) result = allocate(); do {} while (0); result; }); })->page;
}
`), nil)
	if !result.Ok {
		t.Fatalf("nested statement-expression member failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if bytes.Contains(result.Source, []byte("struct tag")) || !bytes.Contains(result.Source, []byte("func __c_statement_expr_")) ||
		!bytes.Contains(result.Source, []byte(").page")) {
		t.Fatalf("nested statement-expression member was not semantically lowered:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("nested statement-expression source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateStatementExpressionCapturedPointerSubscriptMember(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct timestamp { long sec; };
struct data { int prefix; union { struct timestamp basetime[2]; }; };
unsigned long inspect(struct data *value) {
	return ({ *(unsigned long *)&(value[0].basetime[0].sec); });
}
`), nil)
	if !result.Ok {
		t.Fatalf("captured pointer-subscript member failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("__c_pointer_index_")) ||
		!bytes.Contains(result.Source, []byte(".__c_ptr_8_basetime()")) ||
		bytes.Contains(result.Source, []byte("(*p0)[0].basetime")) {
		t.Fatalf("captured pointer-subscript member bypassed its typed accessors:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("captured pointer-subscript source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateGNULocalLabels(t *testing.T) {
	result := TranslateObject("main", []byte(`
int choose(int value) {
	int first = ({ __label__ out; if (value) goto out; value = 1; out: value; });
	int second = ({ __label__ out; if (value) goto out; value = 2; out: value; });
	return first + second;
}
`), nil)
	if !result.Ok {
		t.Fatalf("GNU local-label translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("goto __c_label_")) || bytes.Count(result.Source, []byte("__c_label_")) < 4 {
		t.Fatalf("GNU local labels were not uniquely lowered:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("GNU local-label source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
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

func TestTranslateCompletesIncompleteArrayTypedefInitializer(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct match_token { int token; const char *pattern; };
typedef struct match_token match_table_t[];
static const match_table_t tokens = {
	{ 1, "first" },
	{ 2, "second" },
};
`), nil)
	if !result.Ok {
		t.Fatalf("incomplete-array typedef initializer failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("var tokens [2]__c_struct_match_token")) {
		t.Fatalf("incomplete-array typedef was not completed from its initializer:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("completed typedef array source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateUniquifiesShadowedAggregateTags(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct event { int outer; };
void first(void) { struct event { int value; } local = { 1 }; (void)local; }
void second(void) { struct event { long value; } local = { 2 }; (void)local; }
void introduce(void) { (void)sizeof(struct delayed *); }
struct delayed *declare_delayed(void);
struct delayed { int value; };
`), nil)
	if !result.Ok {
		t.Fatalf("shadowed aggregate tags failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if bytes.Count(result.Source, []byte("type __c_struct_event")) != 3 ||
		!bytes.Contains(result.Source, []byte("type __c_struct_event_")) {
		t.Fatalf("shadowed aggregate tags did not receive unique carrier names:\n%s", result.Source)
	}
	if bytes.Count(result.Source, []byte("type __c_struct_delayed")) != 2 ||
		!bytes.Contains(result.Source, []byte("type __c_struct_delayed_")) {
		t.Fatalf("scoped incomplete tag and later completed tag did not retain distinct carriers:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("shadowed aggregate source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
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
	source := []byte(`
struct operations { int value; };
static const struct operations implementation = { .value = 1 };
static const struct operations *selected = { &implementation };
`)
	result := CheckObjectForDataModel(source, DataModelLP64)
	if !result.Ok {
		t.Fatalf("braced scalar initializer check failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	lowered := TranslateObject("main", source, nil)
	if !lowered.Ok || !bytes.Contains(lowered.Source, []byte("variable selected - 8 0 0 - 8 1 implementation 0")) ||
		!bytes.Contains(lowered.Source, []byte("var selected *__c_struct_operations=nil")) {
		t.Fatalf("braced scalar initializer lowering failed: %#v\n%s", lowered, lowered.Source)
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
	void *raw = (void *)operation;
	operation = raw;
	unsigned char byte = 250;
	int array[3] = {1, 2, 3};
	int *cursor = array;
	int selected = choose ? mark(2) : mark(3);
	int comma = (mark(4), mark(5));
	return selected + comma + operation(7, 5) + ~byte + sizeof array + sizeof *cursor +
		letter + chosen + narrowed + unsigned_compare + ignored_variadic(9, mark(6), 25);
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
		[]byte("operation=raw"),
		[]byte("operation(7,5)"),
		[]byte("int32(-int32(byte)-1)"),
		[]byte("+12+4+65+13+2+0+"),
		[]byte("__c_va_words[0]=uintptr(p1)"),
		[]byte("return ignored_variadic(p0,&__c_va)"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("expression source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated expressions do not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectFunctionPointerCastPrecedence(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef void (*callback)(char *);
void adjust(callback value, unsigned long offset) {
	*(unsigned int *)((unsigned char *)value + offset) += (unsigned long)value;
}
`), nil)
	if !result.Ok {
		t.Fatalf("function pointer arithmetic translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("__c_pointer_step_inc_"),
		[]byte("pointer *uint8,count uintptr) *uint8"),
		[]byte("count*uintptr(1)"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("function pointer arithmetic source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated function pointer arithmetic does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectKVMHypercallAsm(t *testing.T) {
	result := TranslateObject("main", []byte(`
long vm(unsigned int nr, unsigned long value) {
	long result;
	asm volatile("vmcall" : "=a"(result) : "a"(nr), "b"(value) : "memory");
	return result;
}
long vmm(unsigned int nr) {
	long result;
	asm volatile("vmmcall" : "=a"(result) : "a"(nr) : "memory");
	return result;
}
`), nil)
	if !result.Ok {
		t.Fatalf("KVM hypercall translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("result=renvo_runtime_CVMCall2(uint64(nr),uint64(value))"),
		[]byte("func renvo_runtime_CVMCall2(p0 uint64,p1 uint64) int64"),
		[]byte("result=renvo_runtime_CVMMCall1(uint64(nr))"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("KVM hypercall source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated KVM hypercalls do not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectLowersIntegerAndPointerVarargs(t *testing.T) {
	result := TranslateObject("main", []byte(`
typedef __builtin_va_list va_list;
static unsigned long consume(va_list args) {
	int number = __builtin_va_arg(args, int);
	char *pointer = __builtin_va_arg(args, char *);
	return (unsigned long)number + (pointer != 0);
}
static unsigned long copy_first(va_list source) {
	va_list destination;
	__builtin_va_copy(destination, source);
	return __builtin_va_arg(destination, unsigned long);
}
unsigned long inspect(int fixed, ...) {
	va_list args;
	unsigned long result;
	__builtin_va_start(args, fixed);
	result = consume(args);
	__builtin_va_end(args);
	return result;
}
unsigned long call_inspect(char *pointer) { return inspect(1, 42, pointer); }
`), nil)
	if !result.Ok {
		t.Fatalf("varargs lowering failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("// renvo:object function inspect - 0 1 0 - 8 0 - 1"),
		[]byte("func inspect(fixed int32,__c_va *[3]uintptr)"),
		[]byte("args=*__c_va"),
		[]byte("renvo_runtime_CVAArg(args)"),
		[]byte("__c_va_copy(&(destination)[0],source)"),
		[]byte("consume(&(args)[0])"),
		[]byte("__c_va_words[0]=uintptr(p1)"),
		[]byte("__c_va_words[1]=uintptr(__c_unsafe.Pointer(p2))"),
		[]byte("return inspect(p0,&__c_va)"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("varargs source is missing %q:\n%s", want, result.Source)
		}
	}
	if bytes.Contains(result.Source, []byte("_=args=*__c_va")) {
		t.Fatalf("discarded va_start was wrapped in a scalar sink:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("varargs shared source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectLowersVariadicStackWords(t *testing.T) {
	result := TranslateObject("main", []byte(`
static unsigned long inspect(int fixed, ...) { return fixed; }
unsigned long call(void) { return inspect(1, 2, 3, 4, 5, 6, 7, 8); }
`), nil)
	if !result.Ok {
		t.Fatalf("variadic stack-word lowering failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("var __c_va_words [7]uintptr"),
		[]byte("__c_va_words[6]=uintptr(p7)"),
		[]byte("__c_va[2]=uintptr(__c_unsafe.Pointer(&__c_va_words[6]))"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("variadic stack-word source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("variadic stack-word source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectStaticDesignatedArrayHelperDoesNotAliasOutput(t *testing.T) {
	result := TranslateObject("main", []byte(`
void setup(void) {
	static const unsigned long table[] = { [2] = 11, [2] = 12 };
}
`), nil)
	if !result.Ok {
		t.Fatalf("static designated array lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("return p}")) || !bytes.Contains(result.Source, []byte("var __c_static_")) {
		t.Fatalf("static designated array helper was truncated:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("static designated array shared source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectFlatDesignatedStructUsesKeyedComposite(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct callbacks { const char *name; int (*probe)(void); unsigned enabled : 1; };
static int probe(void) { return 1; }
struct callbacks callbacks = { .name = ("test"), .probe = probe, .enabled = 1 };
`), nil)
	if !result.Ok {
		t.Fatalf("flat designated struct lowering failed: %#v", result)
	}
	if !bytes.Contains(result.Source, []byte("__c_struct_callbacks{name:renvo_runtime_CStringPointer(")) ||
		!bytes.Contains(result.Source, []byte("__c_bits_")) ||
		bytes.Contains(result.Source, []byte("var callbacks __c_struct_callbacks=__c_aggregate_init_")) {
		t.Fatalf("flat designated struct with a bitfield did not use a constant keyed composite:\n%s", result.Source)
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

func TestTranslateShortWideStringPointer(t *testing.T) {
	result := TranslateObjectWithConfig("main", []byte(`
typedef unsigned short char16;
extern int consume(const char16 *value);
int invoke(void) { return consume(L"EFI"); }
`), nil, ObjectConfig{DataModel: DataModelLP64, ShortWChar: true})
	if !result.Ok {
		t.Fatalf("short wide-string translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("func renvo_runtime_CWideStringPointer2(value string) *uint16"),
		[]byte(`consume(renvo_runtime_CWideStringPointer2("\x45\x00\x46\x00\x49\x00\x00\x00"))`),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("short wide-string source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated short wide string does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectMergesFunctionPointerObjectRedeclarations(t *testing.T) {
	result := TranslateObject("main", []byte(`
extern void (*pm_power_off)(void);
void (*pm_power_off)(void);
extern typeof(pm_power_off) pm_power_off;
static void *addressable_pm_power_off = (void *)(unsigned long)&pm_power_off;
`), nil)
	if !result.Ok {
		t.Fatalf("function-pointer object redeclarations failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("variable addressable_pm_power_off"),
		[]byte(" 1 pm_power_off 0"),
		[]byte("var pm_power_off func()"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("function-pointer redeclaration source is missing %q:\n%s", want, result.Source)
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

func TestTranslateObjectForeignAggregateReturnUsesZeroComposite(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct pair { long first; long second; };
extern struct pair read_pair(int index);
long inspect(void) { return read_pair(3).second; }
`), nil)
	if !result.Ok {
		t.Fatalf("foreign aggregate return translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("func read_pair(p0 int32) __c_struct_pair{return __c_struct_pair{}}")) ||
		bytes.Contains(result.Source, []byte("__c_struct_pair{return 0}")) {
		t.Fatalf("foreign aggregate stub has an invalid zero value:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("foreign aggregate source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectEmitsUndefinedObjectDeclaration(t *testing.T) {
	result := TranslateObject("main", []byte(`
extern unsigned long max_pfn_mapped;
unsigned long read_max(void) { return max_pfn_mapped + 3; }
`), nil)
	if !result.Ok {
		t.Fatalf("TranslateObject failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("// renvo:object variable-extern max_pfn_mapped - 8 1 0 - 8 0 - 0"),
		[]byte("var max_pfn_mapped uint64;"),
		[]byte("return max_pfn_mapped+3"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("translated object source is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated object source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateObjectEmitsArrayDecayRelocation(t *testing.T) {
	result := TranslateObject("main", []byte(`
extern char image_end[];
char *heap = image_end;
`), nil)
	if !result.Ok {
		t.Fatalf("array-decay relocation translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("// renvo:object variable heap - 8 1 0 - 8 1 image_end 0"),
		[]byte("var heap *int8=nil"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("array-decay relocation source is missing %q:\n%s", want, result.Source)
		}
	}
}

func TestTranslateObjectEmitsStaticLocalRelocation(t *testing.T) {
	result := TranslateObject("main", []byte(`
extern struct key target;
void use(void) {
	static void *addressable __attribute__((used, section(".discard.addressable"))) = (void *)(uintptr_t)&target;
}
`), nil)
	if !result.Ok {
		t.Fatalf("static-local relocation translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("// renvo:object variable __c_static_2_addressable .discard.addressable 8 0 0 - 8 1 target 0"),
		[]byte("var __c_static_2_addressable *byte=nil"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("static-local relocation source is missing %q:\n%s", want, result.Source)
		}
	}
}

func TestTranslateDiscardedScalarExpressionUsesSink(t *testing.T) {
	result := TranslateObject("main", []byte(`int inspect(int value) { value + 1; return value; }`), nil)
	if !result.Ok {
		t.Fatalf("discarded scalar translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("_=value+1;")) {
		t.Fatalf("discarded scalar expression is not a valid Go statement:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("discarded scalar source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateArrayDecayWithPointerReinterpretation(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct cpu_info { unsigned int capability[24]; };
extern struct cpu_info boot_cpu_data;
extern unsigned char arch_test_bit(unsigned long, const unsigned long *);
unsigned char inspect(void) { return arch_test_bit(272, (const unsigned long *)boot_cpu_data.capability); }
`), nil)
	if !result.Ok {
		t.Fatalf("array pointer reinterpretation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("(*uint64)(__c_unsafe.Pointer(&(boot_cpu_data.capability)))")) {
		t.Fatalf("array pointer reinterpretation did not preserve its address:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("array pointer reinterpretation source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslatePromotedAnonymousUnionArrayReinterpretation(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct cpu_info { union { unsigned int capability[24]; unsigned long alignment; }; };
extern struct cpu_info boot_cpu_data;
extern unsigned char arch_test_bit(unsigned long, const unsigned long *);
unsigned char inspect(void) { return arch_test_bit(272, (const unsigned long *)(&boot_cpu_data)->capability); }
`), nil)
	if !result.Ok {
		t.Fatalf("promoted array reinterpretation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte(".__c_ptr_0_capability()")) {
		t.Fatalf("promoted array reinterpretation bypassed its offset accessor:\n%s", result.Source)
	}
	if bytes.Contains(result.Source, []byte("(&boot_cpu_data).capability")) {
		t.Fatalf("promoted anonymous member became a nonexistent direct field:\n%s", result.Source)
	}
}

func TestTranslateLogicalNotOfInteger(t *testing.T) {
	result := TranslateObject("main", []byte(`int empty(unsigned long value) { if (!value) return 1; return 0; }`), nil)
	if !result.Ok {
		t.Fatalf("TranslateObject failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("if !((value)!=0)")) {
		t.Fatalf("translated logical not is incorrect:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated logical not does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslatePointerSubscriptUsesTypedAddress(t *testing.T) {
	result := TranslateObject("main", []byte(`
int digit(const char *text) { return text[2]; }
int literal_digit(void) { return "\004\002\006\006"[2]; }
`), nil)
	if !result.Ok {
		t.Fatalf("TranslateObject failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("func __c_pointer_index_"),
		[]byte("__c_unsafe.Pointer(uintptr(__c_unsafe.Pointer(pointer))+index*uintptr(1))"),
		[]byte("return int32((*__c_pointer_index_"),
		[]byte("text,uintptr(2)"),
		[]byte("renvo_runtime_CStringPointer"),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("translated pointer subscript is missing %q:\n%s", want, result.Source)
		}
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated pointer subscript does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateArrayOnePastAddressUsesPointerArithmetic(t *testing.T) {
	result := TranslateObject("main", []byte(`
int values[4];
int *end(void) { return &values[4]; }
`), nil)
	if !result.Ok {
		t.Fatalf("one-past array address failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("__c_pointer_index_")) ||
		!bytes.Contains(result.Source, []byte("&(values)[0]")) ||
		bytes.Contains(result.Source, []byte("&(values[4])")) {
		t.Fatalf("one-past array address used a bounds-checked index:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("one-past array source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslatePointerSubscriptMemberUsesTypedAddress(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct record { unsigned long value; int (*test)(int, void *); };
unsigned long inspect(struct record *records, unsigned int index, void *data) {
	return records[index].value + records[index].test(index, data);
}
`), nil)
	if !result.Ok {
		t.Fatalf("pointer-subscript member translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("(*__c_pointer_index_")) || !bytes.Contains(result.Source, []byte(")).value")) {
		t.Fatalf("pointer-subscript member did not use typed pointer arithmetic:\n%s", result.Source)
	}
	if bytes.Contains(result.Source, []byte("records[index].test")) || !bytes.Contains(result.Source, []byte("int32(index)")) {
		t.Fatalf("pointer-subscript callback call was not lowered with C argument conversion:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("pointer-subscript member source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateUnaryPrefixBindsAfterSubscript(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct flags { unsigned values[4]; };
unsigned invert(struct flags *flags, int index) { return ~flags->values[index]; }
`), nil)
	if !result.Ok {
		t.Fatalf("unary subscript translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if bytes.Contains(result.Source, []byte("[4]uint32(-")) ||
		!bytes.Contains(result.Source, []byte("uint32(-flags.values[index]-1)")) {
		t.Fatalf("unary prefix bound before the subscript:\n%s", result.Source)
	}
}

func TestTranslateUnaryPrefixBindsAfterSubscriptMember(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct mask { unsigned long valid; };
unsigned long invert(struct mask *masks, int index) { return ~masks[index].valid; }
`), nil)
	if !result.Ok {
		t.Fatalf("unary subscript-member translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if bytes.Contains(result.Source, []byte("-masks-1")) ||
		!bytes.Contains(result.Source, []byte("uint64(-(*__c_pointer_index_")) {
		t.Fatalf("unary prefix bound into the subscript pointer base:\n%s", result.Source)
	}
}

func TestTranslateConditionalDecaysArrayOperand(t *testing.T) {
	result := TranslateObject("main", []byte(`
unsigned values[4];
unsigned *select_values(int choose) { return choose ? values : ((void *)0); }
`), nil)
	if !result.Ok {
		t.Fatalf("conditional array translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte(") *uint32{if c{return &(values)[0]}")) {
		t.Fatalf("conditional array operand did not decay:\n%s", result.Source)
	}
}

func TestTranslateConditionalInfersFunctionPointerMemberCallResult(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct callbacks { int (*map)(int, unsigned char); };
extern struct callbacks callbacks;
int select_value(int choose, int value) { return choose ? callbacks.map(value, 1) : value; }
`), nil)
	if !result.Ok {
		t.Fatalf("conditional callback translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte(") int32{if c{return callbacks.")) {
		t.Fatalf("conditional callback result did not retain its scalar type:\n%s", result.Source)
	}
}

func TestTranslateObjectConditionalVoidCallsWithSubscriptArguments(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct entry { unsigned long bits[1]; };
extern void first(unsigned long, unsigned long *);
extern void second(unsigned long, unsigned long *);
void choose(struct entry *entries, int index) {
	((index) ? first(index, entries[index].bits) : second(index, entries[index].bits));
}
`), nil)
	if !result.Ok {
		t.Fatalf("conditional void-call translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	for _, want := range [][]byte{
		[]byte("func __c_conditional_"),
		[]byte("{if c{first("),
		[]byte(";return};second("),
	} {
		if !bytes.Contains(result.Source, want) {
			t.Fatalf("conditional void-call source is missing %q:\n%s", want, result.Source)
		}
	}
	if bytes.Contains(result.Source, []byte(") byte{if c{return first(")) ||
		bytes.Contains(result.Source, []byte(")(&((*__c_pointer_index_")) {
		t.Fatalf("conditional void call was misparsed as a call through its subscript argument:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("conditional void-call source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateFunctionPointerStatementExpressionUsesDestinationType(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct callbacks { void (*run)(void); };
int callback_set(struct callbacks *callbacks) {
	void (*run)(void) = ({ do { } while (0); callbacks->run; });
	return run != 0;
}
`), nil)
	if !result.Ok {
		t.Fatalf("function-pointer statement expression failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if bytes.Contains(result.Source, []byte("=({do")) ||
		!bytes.Contains(result.Source, []byte(") func(){")) {
		t.Fatalf("function-pointer statement expression was not lowered through a typed helper:\n%s", result.Source)
	}
}

func TestTranslateConditionalCapturedPointerMemberUsesAddress(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct item { int value; };
int select_value(int choose, struct item *item) { return choose ? item->value : 0; }
`), nil)
	if !result.Ok {
		t.Fatalf("captured pointer member translation failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("(*p1).__c_nested_ptr_")) {
		t.Fatalf("captured pointer member did not retain an addressable receiver:\n%s", result.Source)
	}
}

func TestTranslateBuiltinExpectPreservesBooleanType(t *testing.T) {
	result := TranslateObject("main", []byte(`int likely(unsigned long value) { if (__builtin_expect(!!(value & 8), 0)) return 1; return 0; }`), nil)
	if !result.Ok {
		t.Fatalf("TranslateObject failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("if !(!(((value&8))!=0)) {")) {
		t.Fatalf("translated builtin boolean is incorrect:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("translated builtin boolean does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
	}
}

func TestTranslateGNUAlignofExpression(t *testing.T) {
	result := TranslateObject("main", []byte(`
struct item { char prefix; long value; };
unsigned long inspect(struct item *item) { return __alignof__(*item); }
`), nil)
	if !result.Ok {
		t.Fatalf("GNU expression alignof failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("return uint64(8);")) {
		t.Fatalf("GNU expression alignof did not use the operand type alignment:\n%s", result.Source)
	}
	if bytes.Contains(result.Source, []byte("*item")) {
		t.Fatalf("GNU expression alignof evaluated its operand:\n%s", result.Source)
	}
}

func TestTranslateObjectExtractReturnAddress(t *testing.T) {
	result := TranslateObject("main", []byte(`
void *extract(void *address) { return __builtin_extract_return_addr(address); }
void *frob(void *address) { return __builtin_frob_return_addr(address); }
`), nil)
	if !result.Ok {
		t.Fatalf("return-address identity builtins failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if bytes.Contains(result.Source, []byte("__builtin_")) ||
		bytes.Count(result.Source, []byte("return address;")) != 2 {
		t.Fatalf("return-address identity builtins were not lowered once:\n%s", result.Source)
	}
	if parsed := syntax.ParseFile(result.Source); !parsed.Ok {
		t.Fatalf("return-address identity source does not parse: error=%d token=%d\n%s", parsed.Error, parsed.ErrorTok, result.Source)
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
	if !bytes.Contains(result.Source, []byte("version(0)==nil")) {
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
