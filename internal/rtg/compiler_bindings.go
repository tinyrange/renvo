package rtg

import (
	"renvo.dev/internal/syntax"
	"strings"
)

// compilerEmitterOperation is the shared lowering surface migrated out of the
// handwritten kernel. Bundled definitions bind it to compiler integration
// hooks; prepared definitions implement it through direct_emitter_v1 and the
// ABI adapter. It contains semantic roles, never target identities or bytes.
type compilerEmitterOperation struct {
	Name       string
	Suffix     string
	Prepared   string
	Parameters []compilerBindingParameter
	// Lowering operations may need compiler state and a typed result.
	Receiver compilerBindingParameter
	Function string
	Result   string
	Failure  string
}

type compilerBindingParameter struct {
	Name string
	Type string
}

func (op compilerEmitterOperation) signature() string {
	receiver := op.receiver()
	s := "(" + receiver.Name + " " + receiver.Type
	for _, p := range op.Parameters {
		s += ", " + p.Name + " " + p.Type
	}
	s += ")"
	if op.Result != "" {
		s += " " + op.Result
	}
	return s
}

func (op compilerEmitterOperation) arguments() string {
	s := "(" + op.receiver().Name
	for _, p := range op.Parameters {
		s += ", " + p.Name
	}
	return s + ")"
}

func (op compilerEmitterOperation) contract() directEmitterOperation {
	p := []string{op.receiver().Type}
	for _, parameter := range op.Parameters {
		p = append(p, parameter.Type)
	}
	return directEmitterOperation{Parameters: p, Result: op.Result}
}

func (op compilerEmitterOperation) receiver() compilerBindingParameter {
	if op.Receiver.Name != "" {
		return op.Receiver
	}
	return compilerBindingParameter{"a", "*renvoAsm"}
}

func (op compilerEmitterOperation) functionName() string {
	if op.Function != "" {
		return op.Function
	}
	return "renvoAsm" + op.Suffix
}

func (op compilerEmitterOperation) assembler() string {
	if op.receiver().Type == "*renvoLinearGen" {
		return op.receiver().Name + ".asm"
	}
	return op.receiver().Name
}

func (op compilerEmitterOperation) failBody() string {
	body := op.assembler() + ".patchFailed = true\n"
	if op.Result != "" {
		body += "return " + op.Failure + "\n"
	}
	return body
}

var compilerEmitterOperations = []compilerEmitterOperation{
	{Name: "string_concat_value_regs", Suffix: "StringConcatValueRegs", Function: "renvoEmitStringConcatLocationValueRegs", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"offset", "int"}}, Prepared: "renvoNonNil(g)\na := \u0026g.asm\nrenvoAsmPushStack(a, offset)\nrenvoAsmLoadSecondaryStack(a, offset-8)\nrenvoAsmPopPrimary(a)\nreturn true"},
	{Name: "append_scalar_helper", Suffix: "AppendScalarHelper", Function: "renvoEnsureAppendScalarHelper", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "int", Failure: "-1", Parameters: []compilerBindingParameter{{"elemKind", "int"}}, Prepared: "small := renvoScalarKindSize(g.c.renvoNativeIntSize, elemKind) == 1\nif small { return renvoAmd64EnsureAppend8Helper(g) }\nreturn renvoAmd64EnsureAppend64Helper(g)"},
	{Name: "append_address_helper", Suffix: "AppendAddressHelper", Function: "renvoEnsureAppendAddrHelper", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "int", Failure: "-1", Parameters: []compilerBindingParameter{}, Prepared: "return renvoAmd64EnsureAppendAddrHelper(g)"},
	{Name: "string_equal_helper", Suffix: "StringEqualHelper", Function: "renvoEnsureStringEqualHelper", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "int", Failure: "-1", Parameters: []compilerBindingParameter{}, Prepared: "return renvoRTGEnsureStringEqualHelper(g)"},
	{Name: "copy_to_fresh_arena", Suffix: "CopyToFreshArena", Function: "renvoEmitCopyToFreshArena", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"srcOff", "int"}, {"destOff", "int"}, {"byteCountOff", "int"}}, Prepared: "renvoEmitCopyBytes(g, srcOff, destOff, byteCountOff)"},
	{Name: "slice_header_addresses_secondary", Suffix: "SliceHeaderAddressesSecondary", Function: "renvoAsmSliceHeaderAddressesSecondary", Result: "", Failure: "", Parameters: []compilerBindingParameter{}, Prepared: "renvoRTGDirectMove(a, renvoRTGCallWord0, renvoRTGSecondary)\nrenvoRTGDirectMove(a, renvoRTGCallWord5, renvoRTGSecondary)\nrenvoRTGDirectMoveImmediate(a, renvoRTGScratch, 16)\nrenvoRTGDirectAdd(a, renvoRTGCallWord5, renvoRTGScratch)\nrenvoRTGDirectMove(a, renvoRTGCallWord1, renvoRTGSecondary)\nrenvoRTGDirectMoveImmediate(a, renvoRTGScratch, 8)\nrenvoRTGDirectAdd(a, renvoRTGCallWord1, renvoRTGScratch)"},
	{Name: "slice_header_addresses_bss", Suffix: "SliceHeaderAddressesBss", Function: "renvoAsmSliceHeaderAddressesBss", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"offset", "int"}}, Prepared: "renvoRTGDirectAddress(a, renvoRTGCallWord0, renvoRTGAsmBSSAddress(offset))\nrenvoRTGDirectAddress(a, renvoRTGCallWord1, renvoRTGAsmBSSAddress(offset+8))\nrenvoRTGDirectAddress(a, renvoRTGCallWord5, renvoRTGAsmBSSAddress(offset+16))"},
	{Name: "slice_header_addresses_stack", Suffix: "SliceHeaderAddressesStack", Function: "renvoAsmSliceHeaderAddressesStack", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"offset", "int"}}, Prepared: "renvoRTGAsmAddressFrame(a, renvoRTGCallWord0, offset)\nrenvoRTGAsmAddressFrame(a, renvoRTGCallWord1, offset-8)\nrenvoRTGAsmAddressFrame(a, renvoRTGCallWord5, offset-16)"},
	{Name: "store_tertiary_stack", Suffix: "StoreTertiaryStack", Function: "renvoAsmStoreTertiaryStack", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"offset", "int"}}, Prepared: "renvoRTGAsmStoreFrame(a, offset, renvoRTGTertiary)"},
	{Name: "index_address_helper_body", Suffix: "IndexAddressHelperBody", Function: "renvoEmitIndexAddressHelperBody", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"elemSize", "int"}}, Prepared: "\tnegative := renvoAsmNewLabel(\u0026g.asm)\n\tinvalid := renvoAsmNewLabel(\u0026g.asm)\n\trenvoAsmPushPrimary(\u0026g.asm)\n\trenvoAsmPushSecondary(\u0026g.asm)\n\trenvoAsmCopyTertiaryToPrimary(\u0026g.asm)\n\trenvoAsmCopyPrimaryToSecondary(\u0026g.asm)\n\trenvoAsmPrimaryImm(\u0026g.asm, 0)\n\trenvoAsmCopySecondaryToTertiary(\u0026g.asm)\n\trenvoAsmCmpTertiaryPrimarySet(\u0026g.asm, 0x9d)\n\trenvoAsmJzPrimary(\u0026g.asm, negative)\n\trenvoAsmPopPrimary(\u0026g.asm)\n\trenvoAsmCopySecondaryToTertiary(\u0026g.asm)\n\trenvoAsmCmpTertiaryPrimarySet(\u0026g.asm, 0x9c)\n\trenvoAsmJzPrimary(\u0026g.asm, invalid)\n\trenvoAsmPopPrimary(\u0026g.asm)\n\trenvoAsmCopySecondaryToTertiary(\u0026g.asm)\n\trenvoAsmAddScaledTertiary(\u0026g.asm, elemSize)\n\trenvoAsmRet(\u0026g.asm)\n\trenvoAsmMarkLabel(\u0026g.asm, negative)\n\trenvoAsmPopTertiary(\u0026g.asm)\n\trenvoAsmMarkLabel(\u0026g.asm, invalid)\n\trenvoAsmPopTertiary(\u0026g.asm)\n\trenvoEmitUncaughtFaultTransfer(g, false)"},
	{Name: "index_address_helper", Suffix: "IndexAddressHelper", Function: "renvoEmitTargetIndexAddressHelper", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "int", Failure: "-1", Parameters: []compilerBindingParameter{{"elemSize", "int"}}, Prepared: "return -1"},
	{Name: "reserved_bounds_check_helper", Suffix: "TargetBoundsCheckHelper", Function: "renvoEmitTargetBoundsCheckHelper", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "int", Failure: "-1", Parameters: []compilerBindingParameter{}, Prepared: "return -1"},
	{Name: "optimized_slice_bounds_checks", Suffix: "OptimizedSliceBoundsChecks", Function: "renvoEmitOptimizedSliceBoundsChecks", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"lowOff", "int"}, {"highOff", "int"}, {"maxOff", "int"}, {"capOff", "int"}}, Prepared: "return false"},
	{Name: "push_words", Suffix: "PushWords", Function: "renvoEmitPushWords", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"offset", "int"}, {"size", "int"}, {"wordSize", "int"}, {"mode", "int"}}, Prepared: "\trenvoNonNil(g)\n\tsize = renvoAlignValue(size, wordSize)\n\t// Keep larger arguments on the word-push path so stack growth touches each\n\t// guard page; one reservation must not skip a Windows stack guard page.\n\tfor at := size - wordSize; at \u003e= 0; at -= wordSize {\n\t\tif mode == renvoPushStack {\n\t\t\trenvoAsmLoadPrimaryStack(\u0026g.asm, offset-at)\n\t\t} else if mode == renvoPushBss {\n\t\t\trenvoAsmLoadPrimaryBss(\u0026g.asm, offset+at)\n\t\t} else {\n\t\t\trenvoAsmLoadPrimaryMemSecondaryDisp(\u0026g.asm, at)\n\t\t}\n\t\trenvoAsmPushPrimary(\u0026g.asm)\n\t}"},
	{Name: "split_word_lowering_enabled", Suffix: "SplitWordLoweringEnabled", Function: "renvoSplitWordLoweringEnabled", Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return renvoFixedTarget == 0"},
	{Name: "shift_primary_right_unsigned_immediate", Suffix: "ShiftPrimaryRightUnsignedImmediate", Function: "renvoAsmShrPrimaryImm", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"imm", "int"}}, Prepared: "if renvoFixedTarget != 0 { return }\n\trenvoRTGDirectShiftRightUnsignedImmediate(a, renvoRTGPrimary, byte(imm))\n\treturn"},
	{Name: "negate_primary", Suffix: "NegatePrimary", Function: "renvoAsmPrimaryToNegative", Result: "", Failure: "", Parameters: []compilerBindingParameter{}, Prepared: "if renvoFixedTarget != 0 { return }\n\trenvoRTGDirectMove(a, renvoRTGScratch, renvoRTGPrimary)\n\trenvoRTGDirectMoveImmediate(a, renvoRTGPrimary, 0)\n\trenvoRTGDirectSubtract(a, renvoRTGPrimary, renvoRTGScratch)\n\treturn"},
	{Name: "wide_less_stack", Suffix: "WideLessStack", Function: "renvoEmitWideLessStack", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"left", "int"}, {"right", "int"}, {"signed", "bool"}}, Prepared: "\tif renvoFixedTarget != 0 {\n\t\treturn\n\t}\n\trenvoNonNil(g)\n\thighEqual := renvoAsmNewLabel(\u0026g.asm)\n\tdone := renvoAsmNewLabel(\u0026g.asm)\n\trenvoEmitNativeCompareStack(g, left-g.c.renvoNativeIntSize, right-g.c.renvoNativeIntSize, 0x94)\n\trenvoAsmJnzPrimary(\u0026g.asm, highEqual)\n\tif signed {\n\t\trenvoEmitNativeCompareStack(g, left-g.c.renvoNativeIntSize, right-g.c.renvoNativeIntSize, 0x9c)\n\t} else {\n\t\trenvoEmitNativeUnsignedLessStack(g, left-g.c.renvoNativeIntSize, right-g.c.renvoNativeIntSize)\n\t}\n\trenvoAsmJmpMarkLabel(\u0026g.asm, done, highEqual)\n\trenvoEmitNativeUnsignedLessStack(g, left, right)\n\trenvoAsmMarkLabel(\u0026g.asm, done)"},
	{Name: "wide_add_stack", Suffix: "WideAddStack", Function: "renvoEmitWideAddStack", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"dest", "int"}, {"left", "int"}, {"right", "int"}}, Prepared: "\tif renvoFixedTarget != 0 {\n\t\treturn\n\t}\n\trenvoNonNil(g)\n\tcarry := renvoAddUnnamedLocal(g, renvoTypeInt)\n\tleftLow := renvoAddUnnamedLocal(g, renvoTypeInt)\n\trenvoAsmCopyStackSlot(\u0026g.asm, left, leftLow)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, left)\n\trenvoAsmLoadTertiaryStack(\u0026g.asm, right)\n\trenvoAsmAddPrimaryTertiary(\u0026g.asm)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, dest)\n\trenvoEmitNativeUnsignedLessStack(g, dest, leftLow)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, carry)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, left-g.c.renvoNativeIntSize)\n\trenvoAsmLoadTertiaryStack(\u0026g.asm, right-g.c.renvoNativeIntSize)\n\trenvoAsmAddPrimaryTertiary(\u0026g.asm)\n\trenvoAsmLoadTertiaryStack(\u0026g.asm, carry)\n\trenvoAsmAddPrimaryTertiary(\u0026g.asm)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, dest-g.c.renvoNativeIntSize)"},
	{Name: "wide_sub_stack", Suffix: "WideSubStack", Function: "renvoEmitWideSubStack", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"dest", "int"}, {"left", "int"}, {"right", "int"}}, Prepared: "\tif renvoFixedTarget != 0 {\n\t\treturn\n\t}\n\trenvoNonNil(g)\n\tborrow := renvoAddUnnamedLocal(g, renvoTypeInt)\n\trenvoEmitNativeUnsignedLessStack(g, left, right)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, borrow)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, left)\n\trenvoAsmLoadTertiaryStack(\u0026g.asm, right)\n\trenvoAsmSubPrimaryTertiary(\u0026g.asm)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, dest)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, left-g.c.renvoNativeIntSize)\n\trenvoAsmLoadTertiaryStack(\u0026g.asm, right-g.c.renvoNativeIntSize)\n\trenvoAsmSubPrimaryTertiary(\u0026g.asm)\n\trenvoAsmLoadTertiaryStack(\u0026g.asm, borrow)\n\trenvoAsmSubPrimaryTertiary(\u0026g.asm)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, dest-g.c.renvoNativeIntSize)"},
	{Name: "wide_shift_left_one", Suffix: "WideShiftLeftOne", Function: "renvoEmitWideShiftLeftOne", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"value", "int"}}, Prepared: "\tif renvoFixedTarget != 0 {\n\t\treturn\n\t}\n\trenvoNonNil(g)\n\tcarry := renvoAddUnnamedLocal(g, renvoTypeInt)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, value)\n\trenvoAsmShrPrimaryImm(\u0026g.asm, 31)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, carry)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, value-g.c.renvoNativeIntSize)\n\trenvoAsmShlPrimaryImm(\u0026g.asm, 1)\n\trenvoAsmLoadTertiaryStack(\u0026g.asm, carry)\n\trenvoAsmAddPrimaryTertiary(\u0026g.asm)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, value-g.c.renvoNativeIntSize)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, value)\n\trenvoAsmShlPrimaryImm(\u0026g.asm, 1)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, value)"},
	{Name: "wide_shift_right_one", Suffix: "WideShiftRightOne", Function: "renvoEmitWideShiftRightOne", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"value", "int"}, {"signed", "bool"}}, Prepared: "\tif renvoFixedTarget != 0 {\n\t\treturn\n\t}\n\trenvoNonNil(g)\n\tcarry := renvoAddUnnamedLocal(g, renvoTypeInt)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, value-g.c.renvoNativeIntSize)\n\trenvoAsmShlPrimaryImm(\u0026g.asm, 31)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, carry)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, value)\n\trenvoAsmShrPrimaryImm(\u0026g.asm, 1)\n\trenvoAsmLoadTertiaryStack(\u0026g.asm, carry)\n\trenvoAsmAddPrimaryTertiary(\u0026g.asm)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, value)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, value-g.c.renvoNativeIntSize)\n\tif signed {\n\t\trenvoAsmSarPrimaryImm(\u0026g.asm, 1)\n\t} else {\n\t\trenvoAsmShrPrimaryImm(\u0026g.asm, 1)\n\t}\n\trenvoAsmStorePrimaryStack(\u0026g.asm, value-g.c.renvoNativeIntSize)"},
	{Name: "wide_shift_stack", Suffix: "WideShiftStack", Function: "renvoEmitWideShiftStack", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"dest", "int"}, {"left", "int"}, {"count", "int"}, {"right", "bool"}, {"signed", "bool"}}, Prepared: "\tif renvoFixedTarget != 0 {\n\t\treturn\n\t}\n\trenvoNonNil(g)\n\trenvoEmitCopyStackToStack(g, left, dest, renvoBackendValueSlotSize)\n\tcounter := renvoAddUnnamedLocal(g, renvoTypeInt)\n\tlimit := renvoAddUnnamedLocal(g, renvoTypeInt)\n\trenvoAsmStoreStackImm(\u0026g.asm, limit, 64)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, count-g.c.renvoNativeIntSize)\n\tclamp := renvoAsmNewLabel(\u0026g.asm)\n\tready := renvoAsmNewLabel(\u0026g.asm)\n\tbegin := renvoAsmNewLabel(\u0026g.asm)\n\trenvoAsmJnzPrimary(\u0026g.asm, clamp)\n\trenvoEmitNativeUnsignedLessStack(g, count, limit)\n\trenvoAsmJnzPrimary(\u0026g.asm, ready)\n\trenvoAsmMarkLabel(\u0026g.asm, clamp)\n\trenvoAsmStoreStackImm(\u0026g.asm, counter, 64)\n\trenvoAsmJmpLabel(\u0026g.asm, begin)\n\trenvoAsmMarkLabel(\u0026g.asm, ready)\n\trenvoAsmCopyStackSlot(\u0026g.asm, count, counter)\n\trenvoAsmMarkLabel(\u0026g.asm, begin)\n\tdone := renvoAsmNewLabel(\u0026g.asm)\n\tloop := renvoAsmNewLabel(\u0026g.asm)\n\trenvoAsmMarkLabel(\u0026g.asm, loop)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, counter)\n\trenvoAsmJzPrimary(\u0026g.asm, done)\n\tif right {\n\t\trenvoEmitWideShiftRightOne(g, dest, signed)\n\t} else {\n\t\trenvoEmitWideShiftLeftOne(g, dest)\n\t}\n\trenvoAsmDecStack(\u0026g.asm, counter)\n\trenvoAsmJmpMarkLabel(\u0026g.asm, loop, done)"},
	{Name: "wide_multiply_stack", Suffix: "WideMultiplyStack", Function: "renvoEmitWideMulStack", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"dest", "int"}, {"left", "int"}, {"right", "int"}}, Prepared: "\tif renvoFixedTarget != 0 {\n\t\treturn\n\t}\n\trenvoNonNil(g)\n\tmultiplicand := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)\n\tmultiplier := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)\n\trenvoEmitCopyStackToStack(g, left, multiplicand, renvoBackendValueSlotSize)\n\trenvoEmitCopyStackToStack(g, right, multiplier, renvoBackendValueSlotSize)\n\trenvoZeroLocalAtOffset(g, dest)\n\tcounter := renvoAddUnnamedLocal(g, renvoTypeInt)\n\trenvoAsmStoreStackImm(\u0026g.asm, counter, 64)\n\tloop := renvoAsmNewLabel(\u0026g.asm)\n\tskipAdd := renvoAsmNewLabel(\u0026g.asm)\n\tdone := renvoAsmNewLabel(\u0026g.asm)\n\trenvoAsmMarkLabel(\u0026g.asm, loop)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, counter)\n\trenvoAsmJzPrimary(\u0026g.asm, done)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, multiplier)\n\trenvoAsmShlPrimaryImm(\u0026g.asm, 31)\n\trenvoAsmJzPrimary(\u0026g.asm, skipAdd)\n\trenvoEmitWideAddStack(g, dest, dest, multiplicand)\n\trenvoAsmMarkLabel(\u0026g.asm, skipAdd)\n\trenvoEmitWideShiftLeftOne(g, multiplicand)\n\trenvoEmitWideShiftRightOne(g, multiplier, false)\n\trenvoAsmDecStack(\u0026g.asm, counter)\n\trenvoAsmJmpMarkLabel(\u0026g.asm, loop, done)"},
	{Name: "wide_negate_stack", Suffix: "WideNegateStack", Function: "renvoEmitWideNegateInPlace", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"value", "int"}}, Prepared: "\tif renvoFixedTarget != 0 {\n\t\treturn\n\t}\n\trenvoNonNil(g)\n\tzero := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)\n\tresult := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)\n\trenvoZeroLocalAtOffset(g, zero)\n\trenvoEmitWideSubStack(g, result, zero, value)\n\trenvoEmitCopyStackToStack(g, result, value, renvoBackendValueSlotSize)"},
	{Name: "wide_unsigned_divide_stack", Suffix: "WideUnsignedDivideStack", Function: "renvoEmitWideUnsignedDivStack", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"quotient", "int"}, {"remainder", "int"}, {"dividendValue", "int"}, {"divisor", "int"}}, Prepared: "\tif renvoFixedTarget != 0 {\n\t\treturn\n\t}\n\trenvoNonNil(g)\n\tdividend := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)\n\trenvoEmitCopyStackToStack(g, dividendValue, dividend, renvoBackendValueSlotSize)\n\trenvoZeroLocalAtOffset(g, quotient)\n\trenvoZeroLocalAtOffset(g, remainder)\n\tcounter := renvoAddUnnamedLocal(g, renvoTypeInt)\n\tbit := renvoAddUnnamedLocal(g, renvoTypeInt)\n\trenvoAsmStoreStackImm(\u0026g.asm, counter, 64)\n\tloop := renvoAsmNewLabel(\u0026g.asm)\n\tskipSubtract := renvoAsmNewLabel(\u0026g.asm)\n\tdone := renvoAsmNewLabel(\u0026g.asm)\n\trenvoAsmMarkLabel(\u0026g.asm, loop)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, counter)\n\trenvoAsmJzPrimary(\u0026g.asm, done)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, dividend-g.c.renvoNativeIntSize)\n\trenvoAsmShrPrimaryImm(\u0026g.asm, 31)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, bit)\n\trenvoEmitWideShiftLeftOne(g, remainder)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, remainder)\n\trenvoAsmLoadTertiaryStack(\u0026g.asm, bit)\n\trenvoAsmAddPrimaryTertiary(\u0026g.asm)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, remainder)\n\trenvoEmitWideShiftLeftOne(g, dividend)\n\trenvoEmitWideShiftLeftOne(g, quotient)\n\trenvoEmitWideLessStack(g, remainder, divisor, false)\n\trenvoAsmJnzPrimary(\u0026g.asm, skipSubtract)\n\trenvoEmitWideSubStack(g, remainder, remainder, divisor)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, quotient)\n\trenvoAsmIncPrimary(\u0026g.asm)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, quotient)\n\trenvoAsmMarkLabel(\u0026g.asm, skipSubtract)\n\trenvoAsmDecStack(\u0026g.asm, counter)\n\trenvoAsmJmpMarkLabel(\u0026g.asm, loop, done)"},
	{Name: "unsigned_less_stack", Suffix: "UnsignedLessStack", Function: "renvoEmitNativeUnsignedLessStack", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"left", "int"}, {"right", "int"}}, Prepared: "\tif renvoFixedTarget != 0 {\n\t\treturn\n\t}\n\trenvoNonNil(g)\n\t// When the sign bits differ, the word with its sign bit clear is smaller in\n\t// unsigned order. When they match, signed subtraction cannot overflow, so\n\t// the ordinary comparison is safe even on wasm's flag emulation.\n\tzero := renvoAddUnnamedLocal(g, renvoTypeInt)\n\tleftNegative := renvoAddUnnamedLocal(g, renvoTypeInt)\n\trightNegative := renvoAddUnnamedLocal(g, renvoTypeInt)\n\trenvoAsmStoreStackImm(\u0026g.asm, zero, 0)\n\trenvoEmitNativeCompareStack(g, left, zero, 0x9c)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, leftNegative)\n\trenvoEmitNativeCompareStack(g, right, zero, 0x9c)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, rightNegative)\n\tsameSign := renvoAsmNewLabel(\u0026g.asm)\n\tdone := renvoAsmNewLabel(\u0026g.asm)\n\trenvoEmitNativeCompareStack(g, leftNegative, rightNegative, 0x94)\n\trenvoAsmJnzPrimary(\u0026g.asm, sameSign)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, rightNegative)\n\trenvoAsmJmpMarkLabel(\u0026g.asm, done, sameSign)\n\trenvoEmitNativeCompareStack(g, left, right, 0x9c)\n\trenvoAsmMarkLabel(\u0026g.asm, done)"},
	{Name: "native_wide_stack", Suffix: "NativeWideStack", Function: "renvoEmitNativeWideStack", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"dest", "int"}, {"left", "int"}, {"right", "int"}, {"mode", "int"}}, Prepared: "if renvoFixedTarget != 0 { return false }; return renvoEmitRTGWideStack(g, dest, left, right, mode)"},
	{Name: "copy_bytes", Suffix: "CopyBytes", Function: "renvoEmitCopyBytes", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"srcPtr", "int"}, {"destPtr", "int"}, {"byteCount", "int"}}, Prepared: "renvoRTGEmitCopyBytes(g, srcPtr, destPtr, byteCount)"},
	{Name: "prefer_bulk_stack_copy", Suffix: "PreferBulkStackCopy", Function: "renvoPreferBulkStackCopy", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"size", "int"}}, Prepared: "return renvoFixedTarget == 0 && size >= 64"},
	{Name: "prefer_bulk_indirect_copy", Suffix: "PreferBulkIndirectCopy", Function: "renvoPreferBulkIndirectCopy", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"size", "int"}}, Prepared: "return false"},
	{Name: "fresh_arena_zero_return", Suffix: "FreshArenaZeroReturn", Function: "renvoEmitMakeZeroFreshArenaReturn", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{}, Prepared: "\t// Object writers require relocations to name storage inside a section;\n\t// the executable fast path also references the arena's one-past end.\n\t// Keep ordinary zeroing for objects, including prepared object targets.\n\tif g.c.objectFile {\n\t\treturn\n\t}\n\trenvoStringHeapOffsets(g)\n\ta := \u0026g.asm\n\tsmall := renvoAsmNewLabel(a)\n\treusedLabel := renvoAsmNewLabel(a)\n\thighReady := renvoAsmNewLabel(a)\n\tplain := renvoAsmNewLabel(a)\n\trenvoAsmPushPrimary(a)\n\trenvoAsmPrimaryImm(a, 256)\n\trenvoAsmCmpTertiaryPrimarySet(a, 0x92)\n\trenvoAsmJnzPrimary(a, small)\n\trenvoAsmPopPrimary(a)\n\trenvoAsmPushSecondary(a)\n\trenvoAsmPushPrimary(a)\n\trenvoAsmPushTertiary(a)\n\trenvoAsmCopyPrimaryToSecondary(a)\n\trenvoAsmAddPrimaryTertiary(a)\n\trenvoAsmCopyPrimaryToTertiary(a)\n\trenvoAsmPrimaryBssAddr(a, g.stringHeapDataOff+renvoStringArenaSize(g))\n\trenvoAsmCmpTertiaryPrimarySet(a, 0x97)\n\trenvoAsmJnzPrimary(a, reusedLabel)\n\trenvoAsmLoadPrimaryBss(a, g.stringHeapOff+24)\n\trenvoAsmJzPrimary(a, highReady)\n\trenvoAsmCmpTertiaryPrimarySet(a, 0x97)\n\trenvoAsmJnzPrimary(a, reusedLabel)\n\trenvoAsmMarkLabel(a, highReady)\n\trenvoAsmCopySecondaryToPrimary(a)\n\trenvoAsmCopyPrimaryToTertiary(a)\n\trenvoAsmPrimaryBssAddr(a, g.stringHeapDataOff)\n\trenvoAsmCmpTertiaryPrimarySet(a, 0x92)\n\trenvoAsmJnzPrimary(a, reusedLabel)\n\trenvoAsmLoadPrimaryBss(a, g.stringHeapOff+16)\n\trenvoAsmCmpTertiaryPrimarySet(a, 0x92)\n\trenvoAsmJnzPrimary(a, reusedLabel)\n\trenvoAsmPopTertiary(a)\n\trenvoAsmPopPrimary(a)\n\trenvoAsmPopSecondary(a)\n\trenvoAsmPushImm(a, 0)\n\trenvoAsmPopTertiary(a)\n\trenvoAsmRet(a)\n\trenvoAsmMarkLabel(a, reusedLabel)\n\trenvoAsmPopTertiary(a)\n\trenvoAsmPopPrimary(a)\n\trenvoAsmPopSecondary(a)\n\trenvoAsmJmpLabel(a, plain)\n\trenvoAsmMarkLabel(a, small)\n\trenvoAsmPopPrimary(a)\n\trenvoAsmMarkLabel(a, plain)"},
	{Name: "make_zero_helper_body", Suffix: "MakeZeroHelperBody", Function: "renvoEmitMakeZeroHelperBody", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{}, Prepared: "\ta := \u0026g.asm\n\trenvoEmitMakeZeroFreshArenaReturn(g)\n\tloopLabel := renvoAsmNewLabel(a)\n\tdoneLabel := renvoAsmNewLabel(a)\n\trenvoAsmCopyPrimaryToSecondary(a)\n\trenvoAsmPushPrimary(a)\n\trenvoAsmMarkLabel(a, loopLabel)\n\trenvoAsmCopyTertiaryToPrimary(a)\n\trenvoAsmJzPrimary(a, doneLabel)\n\trenvoAsmPrimaryImm(a, 0)\n\trenvoAsmStorePrimaryMemSecondaryDispSize(a, 0, 1)\n\trenvoAsmAddSecondaryImm(a, 1)\n\trenvoAsmCopyTertiaryToPrimary(a)\n\trenvoAsmPushImm(a, 1)\n\trenvoAsmPopTertiary(a)\n\trenvoAsmSubPrimaryTertiary(a)\n\trenvoAsmCopyPrimaryToTertiary(a)\n\trenvoAsmJmpMarkLabel(a, loopLabel, doneLabel)\n\trenvoAsmPopPrimary(a)\n\trenvoAsmRet(a)"},
	{Name: "optimized_make_zero_helper", Suffix: "OptimizedMakeZeroHelper", Function: "renvoEmitOptimizedMakeZeroHelper", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return false"},
	{Name: "make_zero", Suffix: "MakeZero", Function: "renvoEmitMakeZero", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{}, Prepared: "\trenvoAsmCallLabel(\u0026g.asm, renvoEnsureMakeZeroHelper(g))"},
	{Name: "zero_local_storage", Suffix: "ZeroLocalStorage", Function: "renvoZeroLocalStorage", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"offset", "int"}, {"size", "int"}}, Prepared: "\ta := \u0026g.asm\n\tif g.c.renvoNativeIntSize == 8 \u0026\u0026 size \u003e= 24 {\n\t\trenvoAsmAddressPrimaryStack(a, offset)\n\t\trenvoAsmPushImm(a, size)\n\t\trenvoAsmPopTertiary(a)\n\t\trenvoEmitMakeZero(g)\n\t} else {\n\t\trenvoAsmPrimaryImm(a, 0)\n\t\tstep := g.c.renvoNativeIntSize\n\t\tfor at := 0; at \u003c size; at += step {\n\t\t\trenvoAsmStorePrimaryStack(a, offset-at)\n\t\t}\n\t}\n"},
	{Name: "primary_tertiary_operation", Suffix: "PrimaryTertiaryOperation", Function: "renvoEmitTargetPrimaryTertiaryOp", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"tok", "int"}}, Prepared: "return renvoRTGEmitPrimaryTertiaryOp(g, tok)"},
	{Name: "unsigned_shift_right", Suffix: "UnsignedShiftRight", Function: "renvoEmitTargetUnsignedShiftRight", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"tok", "int"}}, Prepared: "renvoRTGEmitBoundedVariableShift(\u0026g.asm, RTGShiftRight, false)\nreturn true"},
	{Name: "signed_division_guard", Suffix: "SignedDivisionGuard", Function: "renvoEmitTargetSignedDivisionGuard", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "int", Failure: "-1", Parameters: []compilerBindingParameter{{"mod", "bool"}}, Prepared: "return renvoEmitSignedDivisionOverflowGuard(g, mod)"},
	{Name: "unchecked_signed_division", Suffix: "UncheckedSignedDivision", Function: "renvoEmitUncheckedSignedDivision", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"mod", "bool"}}, Prepared: "renvoAsmCallLabel(&g.asm, renvoEnsureSignedDivisionHelper(g, mod))"},
	{Name: "unchecked_bounds_check", Suffix: "UncheckedBoundsCheck", Function: "renvoEmitUncheckedBoundsCheck", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{}, Prepared: "renvoAsmCallLabel(&g.asm, renvoEnsureBoundsCheckHelper(g))"},
	{Name: "bounds_success_branch", Suffix: "BoundsSuccessBranch", Function: "renvoEmitBoundsSuccessBranch", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"done", "int"}}, Prepared: "renvoAsmCallLabel(\u0026g.asm, renvoEnsureBoundsCheckHelper(g))\nrenvoAsmJnzPrimary(\u0026g.asm, done)"},
	{Name: "bounds_check_helper", Suffix: "BoundsCheckHelper", Function: "renvoEmitBoundsCheckHelperBody", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{}, Prepared: "\tinvalid := renvoAsmNewLabel(\u0026g.asm)\n\trenvoAsmCopyPrimaryToSecondary(\u0026g.asm)\n\trenvoAsmPushTertiary(\u0026g.asm)\n\trenvoAsmPrimaryImm(\u0026g.asm, 0)\n\trenvoAsmCopySecondaryToTertiary(\u0026g.asm)\n\trenvoAsmCmpTertiaryPrimarySet(\u0026g.asm, 0x9d)\n\trenvoAsmJzPrimary(\u0026g.asm, invalid)\n\trenvoAsmPopPrimary(\u0026g.asm)\n\trenvoAsmCopySecondaryToTertiary(\u0026g.asm)\n\trenvoAsmCmpTertiaryPrimarySet(\u0026g.asm, 0x9c)\n\tif !g.meta.panicEnabled {\n\t\tvalid := renvoAsmNewLabel(\u0026g.asm)\n\t\trenvoAsmJnzPrimary(\u0026g.asm, valid)\n\t\trenvoEmitUncaughtFaultTransfer(g, false)\n\t\trenvoAsmMarkLabel(\u0026g.asm, valid)\n\t\trenvoAsmRet(\u0026g.asm)\n\t\trenvoAsmMarkLabel(\u0026g.asm, invalid)\n\t\trenvoAsmPopTertiary(\u0026g.asm)\n\t\trenvoEmitUncaughtFaultTransfer(g, false)\n\t\treturn\n\t}\n\trenvoAsmRet(\u0026g.asm)\n\trenvoAsmMarkLabel(\u0026g.asm, invalid)\n\trenvoAsmPopTertiary(\u0026g.asm)\n\trenvoAsmPrimaryImm(\u0026g.asm, 0)\n\trenvoAsmRet(\u0026g.asm)"},
	{Name: "non_nil_check_helper", Suffix: "NonNilCheckHelper", Function: "renvoEmitTargetNonNilCheckHelper", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "int", Failure: "-1", Parameters: []compilerBindingParameter{{"secondary", "bool"}}, Prepared: "return -1"},
	{Name: "comparison_branch", Suffix: "ComparisonBranch", Function: "renvoEmitCompareJumpOp", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"c0", "byte"}, {"c1", "byte"}, {"label", "int"}, {"jumpIfTrue", "bool"}, {"unsigned", "bool"}}, Prepared: "\tsetcc := 0x94\n\tif c0 == '=' {\n\t\tsetcc = 0x94\n\t} else if c0 == '!' {\n\t\tsetcc = 0x95\n\t} else if c0 == '\u003c' {\n\t\tif c1 == '=' {\n\t\t\tsetcc = 0x9e\n\t\t} else {\n\t\t\tsetcc = 0x9c\n\t\t}\n\t} else if c1 == '=' {\n\t\tsetcc = 0x9d\n\t} else {\n\t\tsetcc = 0x9f\n\t}\n\tif !jumpIfTrue {\n\t\tsetcc = setcc ^ 1\n\t}\n\tif unsigned \u0026\u0026 c0 != '=' \u0026\u0026 c0 != '!' {\n\t\tsetcc = 0x92\n\t\tif c0 == '\u003e' {\n\t\t\tsetcc = 0x97\n\t\t}\n\t\tif c1 == '=' {\n\t\t\tsetcc = setcc ^ 4\n\t\t}\n\t\tif !jumpIfTrue {\n\t\t\tsetcc = setcc ^ 1\n\t\t}\n\t}\n\trenvoRTGDirectJumpCondition(a, renvoRTGConditionFromSetcc(setcc), label)\n\treturn"},
	{Name: "install_thread_state", Suffix: "InstallThreadState", Function: "renvoEmitInstallThreadState", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{}, Prepared: "return"},
	{Name: "load_primary_thread_state", Suffix: "LoadPrimaryThreadState", Function: "renvoAsmLoadPrimaryThreadState", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"stateOffset", "int"}}, Prepared: "\trenvoAsmPushSecondary(\u0026g.asm)\n\trenvoAsmLoadPrimaryBss(\u0026g.asm, g.threadStatePointerOff)\n\trenvoAsmCopyPrimaryToSecondary(\u0026g.asm)\n\trenvoAsmLoadPrimaryMemSecondaryDisp(\u0026g.asm, stateOffset)\n\trenvoAsmPopSecondary(\u0026g.asm)"},
	{Name: "store_primary_thread_state", Suffix: "StorePrimaryThreadState", Function: "renvoAsmStorePrimaryThreadState", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"stateOffset", "int"}}, Prepared: "\trenvoAsmPushSecondary(\u0026g.asm)\n\trenvoAsmPushPrimary(\u0026g.asm)\n\trenvoAsmLoadPrimaryBss(\u0026g.asm, g.threadStatePointerOff)\n\trenvoAsmCopyPrimaryToSecondary(\u0026g.asm)\n\trenvoAsmPopPrimary(\u0026g.asm)\n\trenvoAsmStorePrimaryMemSecondaryDisp(\u0026g.asm, stateOffset)\n\trenvoAsmPopSecondary(\u0026g.asm)"},
	{Name: "scalar_function", Suffix: "ScalarFunction", Function: "renvoEmitTargetScalarFunction", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"fnInfoIndex", "int"}}, Prepared: "return renvoRTGEmitScalarFunction(g, fnInfoIndex)"},
	{Name: "runtime_stack", Suffix: "RuntimeStack", Function: "renvoEmitTargetRuntimeStack", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"ep", "*renvoExprParse"}, {"e", "*renvoExpr"}, {"count", "int"}}, Prepared: "if count == 5 { renvoAsmPrimaryImm(&g.asm, 0) }; return true"},
	{Name: "unchecked_non_nil_primary", Suffix: "UncheckedNonNilPrimary", Function: "renvoEmitUncheckedNonNilPrimary", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{}, Prepared: "renvoAsmCallLabel(&g.asm, renvoEnsureNonNilCheckHelper(g, false))"},
	{Name: "unchecked_non_nil_secondary", Suffix: "UncheckedNonNilSecondary", Function: "renvoEmitUncheckedNonNilSecondary", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{}, Prepared: "renvoAsmCallLabel(&g.asm, renvoEnsureNonNilCheckHelper(g, true))"},
	{Name: "irq_stack_call", Suffix: "IRQStackCall", Function: "renvoEmitIRQStackCall", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"ep", "*renvoExprParse"}, {"e", "*renvoExpr"}, {"helper", "*renvoFuncInfo"}}, Prepared: "return false"},
	{Name: "ms_abi_call", Suffix: "MSABICall", Function: "renvoEmitMSABICall", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"ep", "*renvoExprParse"}, {"e", "*renvoExpr"}, {"helper", "*renvoFuncInfo"}}, Prepared: "return false"},
	{Name: "runtime_platform_intrinsic", Suffix: "RuntimePlatformIntrinsic", Function: "renvoEmitTargetPlatformIntrinsic", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "int", Failure: "0", Parameters: []compilerBindingParameter{{"ep", "*renvoExprParse"}, {"e", "*renvoExpr"}, {"fn", "*renvoFuncInfo"}}, Prepared: "if e.argCount == 0 \u0026\u0026\n(renvoBytesEqualText(g.prog.src, fn.nameStart, fn.nameEnd, \"renvo_runtime_CReadStackPointer\") ||\nrenvoBytesEqualText(g.prog.src, fn.nameStart, fn.nameEnd, \"renvo_runtime_CFrameAddress\")) {\nreturn 0\n}\nreturn -1"},
	{Name: "store_incoming_call_word", Suffix: "StoreIncomingCallWord", Function: "renvoStoreIncomingCallWord", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"word", "int"}, {"offset", "int"}}, Prepared: "renvoRTGStoreParamWord(g, word, offset)"},
	{Name: "call_with_word_count", Suffix: "CallWithWordCount", Function: "renvoEmitTargetCallWithWordCount", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"fnIndex", "int"}, {"wordCount", "int"}}, Prepared: "renvoRTGEmitCallWithWordCount(g, fnIndex, wordCount)"},
	{Name: "unsigned_divide_primary_tertiary", Suffix: "UnsignedDividePrimaryTertiary", Function: "renvoEmitUnsignedDividePrimaryTertiary", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"mod", "bool"}}, Prepared: "if g.c.renvoNativeIntSize == 4 {\n\tdividend := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)\n\tdivisor := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)\n\tquotient := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)\n\tremainder := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, divisor)\n\trenvoAsmStoreStackImm(\u0026g.asm, divisor-g.c.renvoNativeIntSize, 0)\n\trenvoAsmCopyTertiaryToPrimary(\u0026g.asm)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, dividend)\n\trenvoAsmStoreStackImm(\u0026g.asm, dividend-g.c.renvoNativeIntSize, 0)\n\trenvoEmitWideUnsignedDivStack(g, quotient, remainder, dividend, divisor)\n\tif mod {\n\t\trenvoAsmLoadPrimaryStack(\u0026g.asm, remainder)\n\t} else {\n\t\trenvoAsmLoadPrimaryStack(\u0026g.asm, quotient)\n\t}\n\treturn true\n}\nreturn renvoRTGEmitUnsignedDivide(\u0026g.asm, mod)"},
	{Name: "ieee_conversion_primary", Suffix: "IEEEConversionPrimary", Function: "renvoEmitIEEEFloatConversionPrimary", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"sourceKind", "int"}, {"destKind", "int"}}, Prepared: "if renvoRTGPreparedIEEEFloat == 0 { return false }\na := \u0026g.asm\n\ttemp := renvoAddUnnamedLocal(g, renvoTypeInt64)\n\trenvoAsmStorePrimaryStack(a, temp)\n\tif !renvoTypeKindIsFloat(sourceKind) \u0026\u0026 destKind == renvoTypeFloat32 {\n\t\trenvo32IEEEIntToFloatStack(g, temp, 4, 4, !renvoTypeKindIsUnsignedInteger(sourceKind))\n\t} else if sourceKind == renvoTypeFloat32 \u0026\u0026 !renvoTypeKindIsFloat(destKind) {\n\t\trenvo32IEEEFloatToIntStack(g, temp, temp, 4, 4, !renvoTypeKindIsUnsignedInteger(destKind))\n\t} else {\n\t\treturn false\n\t}\n\trenvoAsmLoadPrimaryStack(a, temp)\n\treturn true"},
	{Name: "ieee_arithmetic_primary_tertiary", Suffix: "IEEEArithmeticPrimaryTertiary", Function: "renvoEmitIEEEFloatArithmeticPrimaryTertiary", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"opChar", "byte"}, {"kind", "int"}}, Prepared: "if renvoRTGPreparedIEEEFloat == 0 { return false }\na := \u0026g.asm\n\tif kind != renvoTypeFloat32 {\n\t\treturn false\n\t}\n\tleft := renvoAddUnnamedLocal(g, renvoBuiltinTypeFloat32)\n\tright := renvoAddUnnamedLocal(g, renvoBuiltinTypeFloat32)\n\tresult := renvoAddUnnamedLocal(g, renvoBuiltinTypeFloat32)\n\trenvoAsmStorePrimaryStack(a, right)\n\trenvoAsmCopyTertiaryToPrimary(a)\n\trenvoAsmStorePrimaryStack(a, left)\n\tif !renvo32IEEEBinaryStack(g, result, left, right, opChar, 4) {\n\t\treturn false\n\t}\n\trenvoAsmLoadPrimaryStack(a, result)\n\treturn true"},
	{Name: "ieee_compare_primary_tertiary", Suffix: "IEEEComparePrimaryTertiary", Function: "renvoEmitIEEEComparePrimaryTertiary", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"c0", "byte"}, {"c1", "byte"}, {"kind", "int"}}, Prepared: "return false"},
	{Name: "ieee_compare_stack", Suffix: "IEEECompareStack", Function: "renvoEmit32IEEECompareStack", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"left", "int"}, {"right", "int"}, {"kind", "int"}, {"c0", "byte"}, {"c1", "byte"}}, Prepared: "if renvoRTGPreparedIEEEFloat == 0 {\n\t// Fixed-point prepared targets represent scalar floats in a native word.\n\t// Interface equality also visits float metadata, even in integer-only\n\t// programs; it must not fall through to the fixed x86 x87 encoder.\n\trenvoAsmLoadPrimaryTertiaryStack(\u0026g.asm, right, left)\n\tcondition := 0x94\n\tif c0 == '!' {\n\t\tcondition = 0x95\n\t} else if c0 == '\u003c' {\n\t\tcondition = 0x9c\n\t\tif c1 == '=' {\n\t\t\tcondition = 0x9e\n\t\t}\n\t} else if c0 == '\u003e' {\n\t\tcondition = 0x9f\n\t\tif c1 == '=' {\n\t\t\tcondition = 0x9d\n\t\t}\n\t}\n\trenvoAsmCmpTertiaryPrimarySet(\u0026g.asm, condition)\n\treturn true\n}\nsize := 8\nif kind == renvoTypeFloat32 { size = 4 }\n\trenvoRTGDirectMoveImmediate(\u0026g.asm, renvoRTGSyscallWord2, int64(c0)|int64(c1)\u003c\u003c8)\n\trenvoRTGDirectMoveImmediate(\u0026g.asm, renvoRTGSyscallWord3, int64(size))\n\trenvoRTGIEEEHostSyscall(g, 0x4b02, 2, left, right, 0)\n\tif renvoRTGSyscallResult.Code != renvoRTGPrimary.Code {\n\t\trenvoRTGDirectMove(\u0026g.asm, renvoRTGPrimary, renvoRTGSyscallResult)\n\t}\n\treturn true"},
	{Name: "ieee_negate_primary", Suffix: "IEEENegatePrimary", Function: "renvoEmitIEEEFloatNegatePrimary", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"kind", "int"}}, Prepared: "if renvoRTGPreparedIEEEFloat == 0 {\nrenvoAsmPrimaryToNegative(\u0026g.asm)\nreturn true\n}\nif kind == renvoTypeFloat32 { return renvoEmit32BitIEEEFloatNegatePrimary(g) }\nreturn false"},
	{Name: "global_init_frame_start", Suffix: "GlobalInitFrameStart", Function: "renvoEmitGlobalInitFrameStart", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "int", Failure: "-1", Parameters: []compilerBindingParameter{}, Prepared: "return renvoRTGFrameStart(&g.asm)"},
	{Name: "global_init_frame_end", Suffix: "GlobalInitFrameEnd", Function: "renvoEmitGlobalInitFrameEnd", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"framePatch", "int"}}, Prepared: "renvoAsmLeave(\u0026g.asm)\nrenvoRTGFrameFinish(\u0026g.asm, framePatch, g.stackPeak)"},
	{Name: "ieee_binary_stack", Suffix: "IEEEBinaryStack", Function: "renvo32IEEEBinaryStack", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"dest", "int"}, {"left", "int"}, {"right", "int"}, {"op", "byte"}, {"size", "int"}}, Prepared: "if renvoRTGPreparedIEEEFloat == 0 {\nreturn false\n}\n\trenvoRTGDirectMoveImmediate(\u0026g.asm, renvoRTGSyscallWord3, int64(op))\n\trenvoRTGDirectMoveImmediate(\u0026g.asm, renvoRTGSyscallWord4, int64(size))\n\trenvoRTGIEEEHostSyscall(g, 0x4b01, 3, dest, left, right)\n\treturn true"},
	{Name: "ieee_convert_float_stack", Suffix: "IEEEConvertFloatStack", Function: "renvo32IEEEConvertFloatStack", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"dest", "int"}, {"source", "int"}, {"sourceSize", "int"}, {"destSize", "int"}}, Prepared: "if renvoRTGPreparedIEEEFloat == 0 {\nreturn\n}\n\trenvoRTGDirectMoveImmediate(\u0026g.asm, renvoRTGSyscallWord2, int64(sourceSize))\n\trenvoRTGDirectMoveImmediate(\u0026g.asm, renvoRTGSyscallWord3, int64(destSize))\n\trenvoRTGIEEEHostSyscall(g, 0x4b03, 2, dest, source, 0)"},
	{Name: "ieee_int_to_float_stack", Suffix: "IEEEIntToFloatStack", Function: "renvo32IEEEIntToFloatStack", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"offset", "int"}, {"intSize", "int"}, {"floatSize", "int"}, {"signed", "bool"}}, Prepared: "if renvoRTGPreparedIEEEFloat == 0 {\nreturn\n}\n\trenvoRTGDirectMoveImmediate(\u0026g.asm, renvoRTGSyscallWord1, int64(intSize))\n\trenvoRTGDirectMoveImmediate(\u0026g.asm, renvoRTGSyscallWord2, int64(floatSize))\n\trenvoRTGDirectMoveImmediate(\u0026g.asm, renvoRTGSyscallWord3, int64(renvoBoolInt(signed)))\n\trenvoRTGIEEEHostSyscall(g, 0x4b04, 1, offset, 0, 0)"},
	{Name: "ieee_float_to_int_stack", Suffix: "IEEEFloatToIntStack", Function: "renvo32IEEEFloatToIntStack", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"dest", "int"}, {"source", "int"}, {"floatSize", "int"}, {"intSize", "int"}, {"signed", "bool"}}, Prepared: "if renvoRTGPreparedIEEEFloat == 0 {\nreturn\n}\n\trenvoRTGDirectMoveImmediate(\u0026g.asm, renvoRTGSyscallWord2, int64(floatSize))\n\trenvoRTGDirectMoveImmediate(\u0026g.asm, renvoRTGSyscallWord3, int64(intSize))\n\trenvoRTGDirectMoveImmediate(\u0026g.asm, renvoRTGSyscallWord4, int64(renvoBoolInt(signed)))\n\trenvoRTGIEEEHostSyscall(g, 0x4b05, 2, dest, source, 0)"},
	{Name: "ieee_negate_stack", Suffix: "IEEENegateStack", Function: "renvo32IEEENegateStack", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"offset", "int"}, {"size", "int"}}, Prepared: "if renvoRTGPreparedIEEEFloat == 0 {\nreturn\n}\n\trenvoRTGDirectMoveImmediate(\u0026g.asm, renvoRTGSyscallWord1, int64(size))\n\trenvoRTGIEEEHostSyscall(g, 0x4b06, 1, offset, 0, 0)\n\treturn"},
	{Name: "patch", Suffix: "Patch", Prepared: "renvoRTGPatchRelocations(a)\nrenvoAsmSetDataOffsets(a)", Parameters: nil},
	{Name: "mul_primary_tertiary", Suffix: "MulPrimaryTertiary", Prepared: "renvoRTGDirectMultiply(a, renvoRTGPrimary, renvoRTGTertiary)", Parameters: nil},
	{Name: "primary_imm64", Suffix: "PrimaryImm64", Prepared: "renvoRTGDirectMoveImmediate(a, renvoRTGPrimary, int64(uint32(imm)) | int64(high)<<32)", Parameters: []compilerBindingParameter{{"imm", "int"}, {"high", "int"}}},
	{Name: "primary_imm", Suffix: "PrimaryImm", Prepared: "renvoRTGDirectMoveImmediate(a, renvoRTGPrimary, int64(imm))", Parameters: []compilerBindingParameter{{"imm", "int"}}},
	{Name: "syscall", Suffix: "Syscall", Prepared: "renvoRTGDirectHostSyscall(a)", Parameters: nil},
	{Name: "push_stack_word", Suffix: "PushStackWord", Prepared: "renvoAsmPushStack(a, offset)", Parameters: []compilerBindingParameter{{"offset", "int"}}},
	{Name: "jcmp_stack_stack", Suffix: "JcmpStackStack", Prepared: "renvoAsmPushStack(a, left)\nrenvoAsmLoadPrimaryStack(a, right)\nrenvoAsmPopTertiary(a)\nrenvoAsmCmpTertiaryPrimaryJump(a, setcc, label)", Parameters: []compilerBindingParameter{{"left", "int"}, {"right", "int"}, {"label", "int"}, {"setcc", "int"}}},
	{Name: "jcmp_stack_imm", Suffix: "JcmpStackImm", Prepared: "renvoAsmPushStack(a, offset)\nrenvoAsmPrimaryImm(a, value)\nrenvoAsmPopTertiary(a)\nrenvoAsmCmpTertiaryPrimaryJump(a, setcc, label)", Parameters: []compilerBindingParameter{{"offset", "int"}, {"value", "int"}, {"label", "int"}, {"setcc", "int"}}},
	{Name: "store_byte_mem_secondary_tertiary", Suffix: "StoreByteMemSecondaryTertiary", Prepared: "renvoRTGDirectStoreU8(a,\n\trenvoRTGAsmAddress(renvoRTGSecondary, renvoRTGTertiary, 0, 1),\n\trenvoRTGPrimary)", Parameters: nil},
	{Name: "inc_tertiary", Suffix: "IncTertiary", Prepared: "renvoRTGDirectIncrement(a, renvoRTGTertiary)", Parameters: nil},
	{Name: "inc_primary", Suffix: "IncPrimary", Prepared: "renvoRTGDirectIncrement(a, renvoRTGPrimary)", Parameters: nil},
	{Name: "ret", Suffix: "Ret", Prepared: "renvoRTGDirectReturn(a)", Parameters: nil},
	{Name: "leave", Suffix: "Leave", Prepared: "renvoRTGDirectLeave(a)", Parameters: nil},
	{Name: "copy_primary_to_call_word0", Suffix: "CopyPrimaryToCallWord0", Prepared: "renvoRTGDirectMove(a, renvoRTGCallWord0, renvoRTGPrimary)", Parameters: nil},
	{Name: "copy_secondary_to_primary", Suffix: "CopySecondaryToPrimary", Prepared: "renvoRTGDirectMove(a, renvoRTGPrimary, renvoRTGSecondary)", Parameters: nil},
	{Name: "copy_primary_to_call_word1", Suffix: "CopyPrimaryToCallWord1", Prepared: "renvoRTGDirectMove(a, renvoRTGCallWord1, renvoRTGPrimary)", Parameters: nil},
	{Name: "add_secondary_tertiary", Suffix: "AddSecondaryTertiary", Prepared: "renvoRTGDirectAdd(a, renvoRTGSecondary, renvoRTGTertiary)", Parameters: nil},
	{Name: "load_byte_primary_index_tertiary", Suffix: "LoadBytePrimaryIndexTertiary", Prepared: "renvoRTGDirectLoadU8(a, renvoRTGPrimary,\n\trenvoRTGAsmAddress(renvoRTGPrimary, renvoRTGTertiary, 0, 1))", Parameters: nil},
	{Name: "store_primary_mem_secondary_tertiary8", Suffix: "StorePrimaryMemSecondaryTertiary8", Prepared: "renvoRTGDirectStoreNative(a,\n\trenvoRTGAsmAddress(renvoRTGSecondary, renvoRTGTertiary, 0, 8),\n\trenvoRTGPrimary)", Parameters: nil},
	{Name: "inc_mem_secondary", Suffix: "IncMemSecondary", Prepared: "renvoRTGAsmMemoryIncrement(a, false)", Parameters: nil},
	{Name: "dec_mem_secondary", Suffix: "DecMemSecondary", Prepared: "renvoRTGAsmMemoryIncrement(a, true)", Parameters: nil},
	{Name: "bool_not_primary", Suffix: "BoolNotPrimary", Prepared: "renvoRTGAsmBoolNot(a)", Parameters: nil},
	{Name: "bitwise_not_primary", Suffix: "BitwiseNotPrimary", Prepared: "renvoRTGDirectMoveImmediate(a, renvoRTGScratch, -1)\nrenvoRTGDirectBitXor(a, renvoRTGPrimary, renvoRTGScratch)", Parameters: nil},
	{Name: "add_primary_tertiary", Suffix: "AddPrimaryTertiary", Prepared: "renvoRTGDirectAdd(a, renvoRTGPrimary, renvoRTGTertiary)", Parameters: nil},
	{Name: "sub_primary_tertiary", Suffix: "SubPrimaryTertiary", Prepared: "renvoRTGDirectSubtract(a, renvoRTGPrimary, renvoRTGTertiary)", Parameters: nil},
	{Name: "pop_primary", Suffix: "PopPrimary", Prepared: "renvoRTGAsmPopRegister(a, renvoRTGPrimary)", Parameters: nil},
	{Name: "pop_secondary", Suffix: "PopSecondary", Prepared: "renvoRTGAsmPopRegister(a, renvoRTGSecondary)", Parameters: nil},
	{Name: "pop_tertiary", Suffix: "PopTertiary", Prepared: "renvoRTGAsmPopRegister(a, renvoRTGTertiary)", Parameters: nil},
	{Name: "copy_primary_to_secondary", Suffix: "CopyPrimaryToSecondary", Prepared: "renvoRTGDirectMove(a, renvoRTGSecondary, renvoRTGPrimary)", Parameters: nil},
	{Name: "copy_primary_to_tertiary", Suffix: "CopyPrimaryToTertiary", Prepared: "renvoRTGDirectMove(a, renvoRTGTertiary, renvoRTGPrimary)", Parameters: nil},
	{Name: "copy_secondary_to_tertiary", Suffix: "CopySecondaryToTertiary", Prepared: "renvoRTGDirectMove(a, renvoRTGTertiary, renvoRTGSecondary)", Parameters: nil},
	{Name: "copy_tertiary_to_primary", Suffix: "CopyTertiaryToPrimary", Prepared: "renvoAsmPushTertiary(a); renvoAsmPopPrimary(a)", Parameters: nil},
	{Name: "push_primary", Suffix: "PushPrimary", Prepared: "renvoRTGAsmPushRegister(a, renvoRTGPrimary)", Parameters: nil},
	{Name: "push_secondary", Suffix: "PushSecondary", Prepared: "renvoRTGAsmPushRegister(a, renvoRTGSecondary)", Parameters: nil},
	{Name: "push_tertiary", Suffix: "PushTertiary", Prepared: "renvoRTGAsmPushRegister(a, renvoRTGTertiary)", Parameters: nil},
	{Name: "push_imm", Suffix: "PushImm", Prepared: "\trenvoRTGAsmPushImmediate(a, imm)", Parameters: []compilerBindingParameter{{"imm", "int"}}},
	{Name: "store_primary_stack", Suffix: "StorePrimaryStack", Prepared: "\trenvoRTGAsmStoreFrame(a, offset, renvoRTGPrimary)", Parameters: []compilerBindingParameter{{"offset", "int"}}},
	{Name: "store_secondary_stack", Suffix: "StoreSecondaryStack", Prepared: "\trenvoRTGAsmStoreFrame(a, offset, renvoRTGSecondary)", Parameters: []compilerBindingParameter{{"offset", "int"}}},
	{Name: "load_primary_stack", Suffix: "LoadPrimaryStack", Prepared: "\trenvoRTGAsmLoadFrame(a, renvoRTGPrimary, offset)", Parameters: []compilerBindingParameter{{"offset", "int"}}},
	{Name: "inc_stack", Suffix: "IncStack", Prepared: "\trenvoAsmLoadPrimaryStack(a, offset)\n\trenvoAsmIncPrimary(a)\n\trenvoAsmStorePrimaryStack(a, offset)", Parameters: []compilerBindingParameter{{"offset", "int"}}},
	{Name: "dec_stack", Suffix: "DecStack", Prepared: "\trenvoAsmLoadPrimaryStack(a, offset)\n\trenvoAsmPushImm(a, 1)\n\trenvoAsmPopTertiary(a)\n\trenvoAsmSubPrimaryTertiary(a)\n\trenvoAsmStorePrimaryStack(a, offset)", Parameters: []compilerBindingParameter{{"offset", "int"}}},
	{Name: "address_primary_stack", Suffix: "AddressPrimaryStack", Prepared: "\trenvoRTGAsmAddressFrame(a, renvoRTGPrimary, offset)", Parameters: []compilerBindingParameter{{"offset", "int"}}},
	{Name: "address_call_word0_stack", Suffix: "AddressCallWord0Stack", Prepared: "\trenvoRTGAsmAddressFrame(a, renvoRTGCallWord0, offset)", Parameters: []compilerBindingParameter{{"offset", "int"}}},
	{Name: "address_call_word1_stack", Suffix: "AddressCallWord1Stack", Prepared: "\trenvoRTGAsmAddressFrame(a, renvoRTGCallWord1, offset)", Parameters: []compilerBindingParameter{{"offset", "int"}}},
	{Name: "load_secondary_stack", Suffix: "LoadSecondaryStack", Prepared: "\trenvoRTGAsmLoadFrame(a, renvoRTGSecondary, offset)", Parameters: []compilerBindingParameter{{"offset", "int"}}},
	{Name: "load_tertiary_stack", Suffix: "LoadTertiaryStack", Prepared: "\trenvoRTGAsmLoadFrame(a, renvoRTGTertiary, offset)", Parameters: []compilerBindingParameter{{"offset", "int"}}},
	{Name: "store_slice_stack", Suffix: "StoreSliceStack", Prepared: "renvoAsmStorePrimarySecondaryStack(a, offset, offset-8)\nrenvoRTGAsmStoreFrame(a, offset-16, renvoRTGTertiary)", Parameters: []compilerBindingParameter{{"offset", "int"}}},
	{Name: "mul_tertiary_imm", Suffix: "MulTertiaryImm", Prepared: "\tif imm == 1 {\n\t\treturn\n\t}\n\trenvoRTGDirectMoveImmediate(a, renvoRTGScratch, int64(imm))\n\trenvoRTGDirectMultiply(a, renvoRTGTertiary, renvoRTGScratch)", Parameters: []compilerBindingParameter{{"imm", "int"}}},
	{Name: "call_label", Suffix: "CallLabel", Prepared: "\trenvoRTGDirectCall(a, label)", Parameters: []compilerBindingParameter{{"label", "int"}}},
	{Name: "jmp_label", Suffix: "JmpLabel", Prepared: "\trenvoRTGDirectJump(a, label)", Parameters: []compilerBindingParameter{{"label", "int"}}},
	{Name: "jz_label", Suffix: "JzLabel", Prepared: "\trenvoRTGDirectJumpCondition(a, renvoRTGConditionFromSetcc(0x94), label)", Parameters: []compilerBindingParameter{{"label", "int"}}},
	{Name: "jnz_label", Suffix: "JnzLabel", Prepared: "\trenvoRTGDirectJumpCondition(a, renvoRTGConditionFromSetcc(0x95), label)", Parameters: []compilerBindingParameter{{"label", "int"}}},
	{Name: "secondary_imm", Suffix: "SecondaryImm", Prepared: "\trenvoRTGDirectMoveImmediate(a, renvoRTGSecondary, int64(imm))", Parameters: []compilerBindingParameter{{"imm", "int"}}},
	{Name: "primary_data_addr", Suffix: "PrimaryDataAddr", Prepared: "\trenvoRTGDirectAddress(a, renvoRTGPrimary, renvoRTGAsmDataAddress(dataOff))", Parameters: []compilerBindingParameter{{"dataOff", "int"}}},
	{Name: "primary_bss_addr", Suffix: "PrimaryBssAddr", Prepared: "\trenvoRTGDirectAddress(a, renvoRTGPrimary, renvoRTGAsmBSSAddress(bssOff))", Parameters: []compilerBindingParameter{{"bssOff", "int"}}},
	{Name: "load_primary_bss", Suffix: "LoadPrimaryBss", Prepared: "\trenvoRTGDirectLoadNative(a, renvoRTGPrimary, renvoRTGAsmBSSAddress(bssOff))", Parameters: []compilerBindingParameter{{"bssOff", "int"}}},
	{Name: "store_primary_bss", Suffix: "StorePrimaryBss", Prepared: "\trenvoRTGDirectStoreNative(a, renvoRTGAsmBSSAddress(bssOff), renvoRTGPrimary)", Parameters: []compilerBindingParameter{{"bssOff", "int"}}},
	{Name: "pop_call_word0", Suffix: "PopCallWord0", Prepared: "\trenvoRTGAsmPopRegister(a, renvoRTGCallWord0)", Parameters: nil},
	{Name: "pop_call_word1", Suffix: "PopCallWord1", Prepared: "\trenvoRTGAsmPopRegister(a, renvoRTGCallWord1)", Parameters: nil},
	{Name: "add_secondary_imm", Suffix: "AddSecondaryImm", Prepared: "\trenvoRTGDirectMoveImmediate(a, renvoRTGScratch, int64(imm))\n\trenvoRTGDirectAdd(a, renvoRTGSecondary, renvoRTGScratch)", Parameters: []compilerBindingParameter{{"imm", "int"}}},
	{Name: "load_qword_primary_index_tertiary_disp", Suffix: "LoadQwordPrimaryIndexTertiaryDisp", Prepared: "\trenvoRTGDirectLoadNative(a, renvoRTGPrimary,\n\t\trenvoRTGAsmAddress(renvoRTGPrimary, renvoRTGTertiary, disp, 1))", Parameters: []compilerBindingParameter{{"disp", "int"}}},
	{Name: "load_primary_mem_secondary_disp", Suffix: "LoadPrimaryMemSecondaryDisp", Prepared: "\trenvoRTGDirectLoadNative(a, renvoRTGPrimary,\n\t\trenvoRTGAsmAddress(renvoRTGSecondary, RTGNoRegister, disp, 1))", Parameters: []compilerBindingParameter{{"disp", "int"}}},
	{Name: "load_primary_mem_secondary_disp_size", Suffix: "LoadPrimaryMemSecondaryDispSize", Prepared: "\t// Size-only scalar loads follow the built-in backend contract: bytes are\n\t// zero-extended, while wider narrow integers are sign-extended before\n\t// typed expression lowering applies any unsigned normalization.\n\trenvoRTGAsmLoadSize(a, renvoRTGPrimary,\n\t\trenvoRTGAsmAddress(renvoRTGSecondary, RTGNoRegister, disp, 1),\n\t\tsize, size != 1)", Parameters: []compilerBindingParameter{{"disp", "int"}, {"size", "int"}}},
	{Name: "load_primary_index_tertiary_size", Suffix: "LoadPrimaryIndexTertiarySize", Prepared: "\t// Match renvoAsmLoadPrimaryMemSecondaryDispSize's scalar-load contract.\n\trenvoRTGAsmLoadSize(a, renvoRTGPrimary,\n\t\trenvoRTGAsmAddress(renvoRTGPrimary, renvoRTGTertiary, 0, size),\n\t\tsize, size != 1)", Parameters: []compilerBindingParameter{{"size", "int"}}},
	{Name: "store_primary_mem_secondary_disp", Suffix: "StorePrimaryMemSecondaryDisp", Prepared: "\trenvoRTGDirectStoreNative(a,\n\t\trenvoRTGAsmAddress(renvoRTGSecondary, RTGNoRegister, disp, 1),\n\t\trenvoRTGPrimary)", Parameters: []compilerBindingParameter{{"disp", "int"}}},
	{Name: "store_primary_mem_secondary_disp_size", Suffix: "StorePrimaryMemSecondaryDispSize", Prepared: "\trenvoRTGAsmStoreSize(a,\n\t\trenvoRTGAsmAddress(renvoRTGSecondary, RTGNoRegister, disp, 1),\n\t\trenvoRTGPrimary, size)", Parameters: []compilerBindingParameter{{"disp", "int"}, {"size", "int"}}},
	{Name: "normalize_primary_for_kind", Suffix: "NormalizePrimaryForKind", Prepared: "\trenvoRTGAsmNormalize(a, kind)", Parameters: []compilerBindingParameter{{"kind", "int"}}},
	{Name: "cmp_primary_imm8", Suffix: "CmpPrimaryImm8", Prepared: "\trenvoRTGAsmCompareImmediate(a, imm)", Parameters: []compilerBindingParameter{{"imm", "int"}}},
	{Name: "cmp_primary_imm8_discard", Suffix: "CmpPrimaryImm8Discard", Prepared: "\trenvoAsmCmpPrimaryImm8(a, imm)", Parameters: []compilerBindingParameter{{"imm", "int"}}},
	{Name: "shl_tertiary_imm", Suffix: "ShlTertiaryImm", Prepared: "\trenvoRTGDirectShiftLeftImmediate(a, renvoRTGTertiary, byte(imm))", Parameters: []compilerBindingParameter{{"imm", "int"}}},
	{Name: "shl_primary_imm", Suffix: "ShlPrimaryImm", Prepared: "\trenvoRTGDirectShiftLeftImmediate(a, renvoRTGPrimary, byte(imm))", Parameters: []compilerBindingParameter{{"imm", "int"}}},
	{Name: "sar_primary_imm", Suffix: "SarPrimaryImm", Prepared: "\trenvoRTGDirectShiftRightSignedImmediate(a, renvoRTGPrimary, byte(imm))", Parameters: []compilerBindingParameter{{"imm", "int"}}},
	{Name: "div_left_tertiary_right_primary", Suffix: "DivLeftTertiaryRightPrimary", Prepared: "\tif mod {\n\t\t// Keep the divisor while producing the quotient, then construct the\n\t\t// remainder as dividend - quotient*divisor.  This keeps prepared\n\t\t// targets on the common RTG register contract and does not depend on\n\t\t// a target-specific fused multiply/subtract operand order.\n\t\trenvoRTGDirectMove(a, renvoRTGScratch, renvoRTGPrimary)\n\t\trenvoRTGDirectSignedDivide(a, false)\n\t\trenvoRTGDirectMultiply(a, renvoRTGPrimary, renvoRTGScratch)\n\t\trenvoRTGDirectSubtract(a, renvoRTGTertiary, renvoRTGPrimary)\n\t\trenvoRTGDirectMove(a, renvoRTGPrimary, renvoRTGTertiary)\n\t} else {\n\t\trenvoRTGDirectSignedDivide(a, false)\n\t}", Parameters: []compilerBindingParameter{{"mod", "bool"}}},
	{Name: "cmp_tertiary_primary_set", Suffix: "CmpTertiaryPrimarySet", Prepared: "\trenvoRTGDirectCompare(a, renvoRTGTertiary, renvoRTGPrimary)\n\trenvoRTGDirectSetCondition(a, renvoRTGConditionFromSetcc(setcc), renvoRTGPrimary)", Parameters: []compilerBindingParameter{{"setcc", "int"}}},
	{Name: "store_primary_mem_secondary_tertiary_size", Suffix: "StorePrimaryMemSecondaryTertiarySize", Prepared: "\trenvoRTGAsmStoreSize(a,\n\t\trenvoRTGAsmAddress(renvoRTGSecondary, renvoRTGTertiary, 0, 1),\n\t\trenvoRTGPrimary, size)", Parameters: []compilerBindingParameter{{"size", "int"}}},
	{Name: "cmp_tertiary_primary_jump", Suffix: "CmpTertiaryPrimaryJump", Prepared: "\trenvoRTGDirectCompare(a, renvoRTGTertiary, renvoRTGPrimary)\n\trenvoRTGDirectJumpCondition(a, renvoRTGConditionFromSetcc(setcc), label)", Parameters: []compilerBindingParameter{{"setcc", "int"}, {"label", "int"}}},
}

func compilerEmitterOperationIndex(name string) int {
	for i := 0; i < len(compilerEmitterOperations); i++ {
		if compilerEmitterOperations[i].Name == name {
			return i
		}
	}
	return -1
}

func compilerBindingIdentifier(name string) bool {
	if len(name) == 0 {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c != '_' && !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') && !(i > 0 && c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

func compilerBindingHook(arch Declaration, name string) string {
	block, ok := declarationBlock(arch, "compiler_bindings")
	if !ok {
		return ""
	}
	for i := 0; i < len(block.Children); i++ {
		left, right, assignment := statementAssignment(block.Children[i])
		if assignment && len(left) == 1 && left[0] == name && len(right) == 1 {
			return right[0]
		}
	}
	return ""
}

func validateCompilerBindings(document Document, arch Declaration) []Diagnostic {
	block, bound := declarationBlock(arch, "compiler_bindings")
	selector, selected := fieldValue(document, arch, "compiler_selector")
	if !bound && !selected {
		return nil
	}
	var diagnostics []Diagnostic
	blocks := 0
	for i := 0; i < len(arch.Statements); i++ {
		if statementBlockName(arch.Statements[i]) == "compiler_bindings" {
			blocks++
		}
	}
	if blocks > 1 {
		return []Diagnostic{resolveDiagnostic(document, arch, "RTG-COMPILER-008", "duplicate compiler binding block")}
	}
	if !bound || !selected || !compilerBindingIdentifier(selector) {
		return []Diagnostic{resolveDiagnostic(document, arch, "RTG-COMPILER-001", "compiler bindings require a selector identifier and a complete binding block")}
	}
	var seen []string
	for i := 0; i < len(block.Children); i++ {
		child := block.Children[i]
		left, right, assignment := statementAssignment(child)
		if !assignment || len(left) != 1 || len(right) != 1 || !compilerBindingIdentifier(right[0]) {
			diagnostics = append(diagnostics, statementDiagnostic(document, child, "RTG-COMPILER-002", "compiler binding must name one compiler hook"))
			continue
		}
		operationIndex := compilerEmitterOperationIndex(left[0])
		if operationIndex < 0 || stringIndex(seen, left[0]) >= 0 {
			diagnostics = append(diagnostics, statementDiagnostic(document, child, "RTG-COMPILER-003", "unknown or duplicate compiler operation "+left[0]))
		}
		seen = append(seen, left[0])
		if operationIndex < 0 {
			continue
		}
		operation := compilerEmitterOperations[operationIndex]
		function, found := findEmbeddedFunctionKind(document, right[0], "compiler")
		if !found || !directEmitterSignatureMatches(function, operation.contract()) {
			diagnostics = append(diagnostics, statementDiagnostic(document, child, "RTG-COMPILER-004", "compiler hook "+right[0]+" must have signature func"+operation.signature()))
		}
	}
	for i := 0; i < len(compilerEmitterOperations); i++ {
		if stringIndex(seen, compilerEmitterOperations[i].Name) < 0 {
			diagnostics = append(diagnostics, resolveDiagnostic(document, arch, "RTG-COMPILER-005", "missing compiler binding "+compilerEmitterOperations[i].Name))
		}
	}
	return diagnostics
}

func appendPreparedCompilerBindings(out []byte) []byte {
	for i := 0; i < len(compilerEmitterOperations); i++ {
		operation := compilerEmitterOperations[i]
		out = append(out, "\nfunc "...)
		out = append(out, operation.functionName()...)
		out = append(out, operation.signature()...)
		out = append(out, " {\nrenvoNonNil("+operation.receiver().Name+")\n"...)
		out = append(out, operation.Prepared...)
		out = append(out, "\n}\n"...)
	}
	return out
}

// Selection is generated from the bundled definitions. There is no built-in
// architecture list here and no default ISA for an unrecognized selector.
func appendBundledCompilerBindings(out []byte, definitions []ResolveResult) GenerateResult {
	var architectures []Declaration
	var documents []Document
	var selectors []string
	for i := 0; i < len(definitions); i++ {
		definition := definitions[i]
		if !definition.Ok {
			return GenerateResult{Diagnostics: definition.Diagnostics}
		}
		found := false
		for j := 0; j < len(definition.Document.Declarations); j++ {
			arch := definition.Document.Declarations[j]
			if arch.Kind != DeclArch {
				continue
			}
			selector, ok := fieldValue(definition.Document, arch, "compiler_selector")
			if !ok {
				continue
			}
			if diagnostics := validateCompilerBindings(definition.Document, arch); len(diagnostics) != 0 {
				return GenerateResult{Diagnostics: diagnostics}
			}
			if stringIndex(selectors, selector) >= 0 {
				return GenerateResult{Diagnostics: []Diagnostic{resolveDiagnostic(definition.Document, arch, "RTG-COMPILER-006", "duplicate bundled selector "+selector)}}
			}
			found = true
			architectures = append(architectures, arch)
			documents = append(documents, definition.Document)
			selectors = append(selectors, selector)
		}
		if !found {
			return GenerateResult{Diagnostics: []Diagnostic{{Filename: definition.Document.Filename, Code: "RTG-COMPILER-007", Message: "bundled definition has no compiler bindings"}}}
		}
	}
	for i := 0; i < len(compilerEmitterOperations); i++ {
		operation := compilerEmitterOperations[i]
		out = append(out, "\nfunc "...)
		out = append(out, operation.functionName()...)
		out = append(out, operation.signature()...)
		out = append(out, " {\nrenvoNonNil("+operation.receiver().Name+")\n"...)
		// Cache the context pointer, not its selector value. Direct fact reads
		// remain visible to fixed-target specialization without duplicating
		// each condition into fixed and dynamic alternatives. The cache also
		// avoids repeated nested receiver loads in multi-target compilers.
		// Avoid capturing any identifier used by a definition-owned body.
		selectorLocal := "renvoCompilerSelector"
		for {
			collision := strings.Contains(operation.signature(), selectorLocal)
			for j := 0; j < len(documents); j++ {
				for k := 0; k < len(documents[j].Declarations); k++ {
					if strings.Contains(string(documents[j].Declarations[k].GoSource), selectorLocal) {
						collision = true
					}
				}
				if strings.Contains(selectors[j], selectorLocal) {
					collision = true
				}
			}
			if !collision {
				break
			}
			selectorLocal += "_"
		}
		out = append(out, selectorLocal+" := "+operation.receiver().Name+".c\nrenvoNonNil("+selectorLocal+")\n"...)
		// Share only byte-identical emitted bodies. Every selector remains explicit,
		// so this neither invents an ISA family nor supplies an unknown-target default.
		var bodies []string
		var conditions []string
		for j := 0; j < len(architectures); j++ {
			hook := compilerBindingHook(architectures[j], operation.Name)
			function, _ := findEmbeddedFunctionKind(documents[j], hook, "compiler")
			project := compilerBindingCanProject(function, operation)
			body := hook + operation.arguments()
			if operation.Result != "" {
				body = "return " + body
			}
			if project {
				body = string(function.Body)
			}
			// Do not create unreachable consecutive returns: the compact source
			// compiler admits a bare return only at its block termination.
			if operation.Result == "" && (!project || !function.EndsInReturn) {
				body += "\nreturn\n"
			}
			condition := selectorLocal + ".renvoTargetArch == " + selectors[j]
			group := stringIndex(bodies, body)
			if group < 0 {
				bodies = append(bodies, body)
				conditions = append(conditions, condition)
			} else {
				conditions[group] += " || " + condition
			}
		}
		out = appendCompilerBodyGroups(out, bodies, conditions)
		out = append(out, operation.failBody()...)
		out = append(out, "}\n"...)
	}
	return GenerateResult{Source: out, Ok: true}
}

// Project definition-owned bodies into their selected branch rather than add a
// second call at every emission site. Returns still leave the dispatch function.
// Noncanonical parameter names and function-scoped labels keep the call path;
// neither requires token substitution or changes the admitted hook contract.
func compilerBindingCanProject(function embeddedFunction, operation compilerEmitterOperation) bool {
	if function.HasLabels || !directEmitterSignatureMatches(function, operation.contract()) ||
		function.Parameters[0].Name != operation.receiver().Name {
		return false
	}
	for i := 0; i < len(operation.Parameters); i++ {
		if function.Parameters[i+1].Name != operation.Parameters[i].Name {
			return false
		}
	}
	return true
}

// Omit only private binding entrypoints which have no other Go references.
// Keep helpers, recursive hooks and hooks used outside the dispatch surface.
// This prevents the source bundle from carrying both a hook and its projected
// body, while retaining real cross-hook dependencies.
func compilerProjectedPrivateHooks(document Document) []string {
	var candidates []string
	var referenced []string
	for i := 0; i < len(document.Declarations); i++ {
		arch := document.Declarations[i]
		if arch.Kind != DeclArch {
			continue
		}
		for j := 0; j < len(compilerEmitterOperations); j++ {
			operation := compilerEmitterOperations[j]
			hook := compilerBindingHook(arch, operation.Name)
			function, found := findEmbeddedFunctionKind(document, hook, "compiler")
			if found && compilerBindingCanProject(function, operation) && stringIndex(candidates, hook) < 0 {
				candidates = append(candidates, hook)
			}
			if found && !compilerBindingCanProject(function, operation) && stringIndex(referenced, hook) < 0 {
				referenced = append(referenced, hook)
			}
		}
	}
	for i := 0; i < len(document.Declarations); i++ {
		declaration := document.Declarations[i]
		referenced = compilerOtherHookReferences(referenced, candidates, declaration.Statements)
		if declaration.Kind != DeclGo {
			continue
		}
		source := append([]byte("package backend\n"), declaration.GoSource...)
		file := syntax.ParseFile(source)
		if !file.Ok {
			return nil
		}
		for j := 0; j < len(file.Tokens); j++ {
			name := string(syntax.TokenText(source, file.Tokens[j]))
			if stringIndex(candidates, name) < 0 || stringIndex(referenced, name) >= 0 {
				continue
			}
			if j > 0 && string(syntax.TokenText(source, file.Tokens[j-1])) == "func" {
				continue
			}
			referenced = append(referenced, name)
		}
	}
	var private []string
	for i := 0; i < len(candidates); i++ {
		if stringIndex(referenced, candidates[i]) < 0 {
			private = append(private, candidates[i])
		}
	}
	return private
}

func appendCompilerBlockWithoutPrivateHooks(out []byte, source []byte, private []string) []byte {
	prefix := "package backend\n"
	wrapped := append([]byte(prefix), source...)
	file := syntax.ParseFile(wrapped)
	if !file.Ok {
		return append(out, source...)
	}
	start := len(prefix)
	for i := 0; i < len(file.Funcs); i++ {
		function := file.Funcs[i]
		name := string(syntax.TokenText(wrapped, file.Tokens[function.NameTok]))
		if stringIndex(private, name) < 0 || function.ReceiverStart >= 0 {
			continue
		}
		out = append(out, wrapped[start:syntax.TokenStart(file.Tokens[function.StartTok])]...)
		start = syntax.TokenEnd(file.Tokens[function.EndTok-1])
	}
	return append(out, wrapped[start:]...)
}

func compilerOtherHookReferences(referenced []string, candidates []string, statements []Statement) []string {
	for i := 0; i < len(statements); i++ {
		statement := statements[i]
		if statementBlockName(statement) == "compiler_bindings" {
			continue
		}
		for j := 0; j < len(statement.Tokens); j++ {
			name := statement.Tokens[j]
			if stringIndex(candidates, name) >= 0 && stringIndex(referenced, name) < 0 {
				referenced = append(referenced, name)
			}
		}
		referenced = compilerOtherHookReferences(referenced, candidates, statement.Children)
	}
	return referenced
}

// Share byte-identical tails only across statement boundaries whose prefixes
// cannot introduce names into the tail's scope. Calls and scoped conditionals
// retain their order and effects inside one exclusive selector chain. Declarations,
// assignments, labels and other control flow conservatively stop splitting.
func appendCompilerBodyGroups(out []byte, bodies []string, conditions []string) []byte {
	owners := make([]int, len(bodies))
	prefixes := make([]string, len(bodies))
	shared := make([]string, len(bodies))
	options := make([][]compilerBodyTail, len(bodies))
	for i := 0; i < len(bodies); i++ {
		owners[i] = -1
		shared[i] = bodies[i]
		options[i] = compilerBodyTails(bodies[i])
	}
	for i := 0; i < len(bodies); i++ {
		if owners[i] >= 0 {
			continue
		}
		best := -1
		bestSaving := 0
		for k := 0; k < len(options[i]); k++ {
			candidate := options[i][k]
			count := 0
			for j := i; j < len(bodies); j++ {
				if owners[j] < 0 && compilerBodyTailIndex(options[j], candidate.tail) >= 0 {
					count++
				}
			}
			saving := (count - 1) * len(candidate.tail)
			if count > 1 && saving > bestSaving {
				best = k
				bestSaving = saving
			}
		}
		owners[i] = i
		if best < 0 {
			continue
		}
		shared[i] = options[i][best].tail
		prefixes[i] = options[i][best].prefix
		for j := i + 1; j < len(bodies); j++ {
			k := compilerBodyTailIndex(options[j], shared[i])
			if owners[j] < 0 && k >= 0 {
				owners[j] = i
				prefixes[j] = options[j][k].prefix
			}
		}
	}
	for i := 0; i < len(bodies); i++ {
		if owners[i] != i {
			continue
		}
		condition := conditions[i]
		for j := i + 1; j < len(bodies); j++ {
			if owners[j] == i {
				condition += " || " + conditions[j]
			}
		}
		out = append(out, "if "+condition+" {\n"...)
		havePrefix := false
		for j := i; j < len(bodies); j++ {
			if owners[j] != i || prefixes[j] == "" {
				continue
			}
			if havePrefix {
				out = append(out, " else "...)
			}
			out = append(out, "if "+conditions[j]+" {\n"...)
			out = append(out, prefixes[j]...)
			out = append(out, "\n}"...)
			havePrefix = true
		}
		if havePrefix {
			out = append(out, '\n')
		}
		out = append(out, shared[i]...)
		out = append(out, "\n}\n"...)
	}
	return out
}

type compilerBodyTail struct {
	prefix string
	tail   string
}

func compilerBodyTailIndex(options []compilerBodyTail, tail string) int {
	for i := 0; i < len(options); i++ {
		if options[i].tail == tail {
			return i
		}
	}
	return -1
}

func compilerBodyTails(body string) []compilerBodyTail {
	options := []compilerBodyTail{{tail: strings.TrimSpace(body)}}
	prefix := "package backend\nfunc projected() {\n"
	source := []byte(prefix + body + "\n}")
	file := syntax.ParseFile(source)
	if !file.Ok || len(file.Funcs) != 1 {
		return options
	}
	statements := syntax.ParseFuncBodyStatements(file, file.Funcs[0])
	if !statements.Ok {
		return options
	}
	for i := 0; i < len(statements.Stmts); i++ {
		if statements.Stmts[i].Kind == syntax.StmtLabel {
			return options
		}
	}
	next := 0
	for i := 1; i < len(statements.Stmts); i++ {
		statement := statements.Stmts[i]
		if statement.StartTok < next {
			continue
		}
		if statement.Kind != syntax.StmtIf && statement.Kind != syntax.StmtExpr {
			break
		}
		next = statement.EndTok
		end := syntax.TokenEnd(file.Tokens[next-1]) - len(prefix)
		if end <= 0 || end >= len(body) {
			break
		}
		tail := strings.TrimSpace(body[end:])
		// Sharing a terminal return alone adds selector tests without removing
		// useful lowering code. It also splits call/return pairs unnecessarily.
		if tail == "" || tail == "return" || tail == "return;" || strings.HasPrefix(tail, "return ") {
			break
		}
		options = append(options, compilerBodyTail{prefix: body[:end], tail: tail})
	}
	return options
}
