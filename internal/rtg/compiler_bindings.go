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
	// Optional conservative reachability fact, derived from the bound bodies.
	// False means every definition and the unknown-selector path return false.
	ReachabilityGuard string
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

// Context receivers are read-only target queries, not emission operations.
// An unknown selector returns their explicit unavailable result; there is no
// assembler on which to record an emission failure.
func (op compilerEmitterOperation) failBody() string {
	if op.receiver().Type == "*renvoCompileContext" {
		return "return " + op.Failure + "\n"
	}
	body := op.assembler() + ".patchFailed = true\n"
	if op.Result != "" {
		body += "return " + op.Failure + "\n"
	}
	return body
}

// Representation policies are separate: a definition may support IEEE stack
// values while retaining scaled untyped atom literals, and may require wide
// argument reconstruction only when the source compiler uses narrow integers.
// Function-address layout describes object ABI storage, not function dispatch.
// Shared comparisons use semantic predicates. Legacy setcc operations remain
// private compatibility inputs for physical recipes, never shared lowering.
var compilerEmitterOperations = []compilerEmitterOperation{
	{Name: "repl_globals_supported", Suffix: "ReplGlobalsSupported", Function: "renvoReplGlobalsSupported", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return true"},
	{Name: "profile_float_model", Suffix: "ProfileFloatModel", Function: "renvoCompilerProfileFloatModel", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "int", Failure: "0", Parameters: []compilerBindingParameter{}, Prepared: "if renvoRTGTargetHasCapability(c.renvoTarget, \"ieee_float\") { return renvoFloatIEEESoft }\nreturn renvoFloatScaledInteger"},
	{Name: "syscall_argument_policy", Suffix: "SyscallArgumentPolicy", Function: "renvoSyscallArgumentPolicy", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "int", Failure: "0", Parameters: []compilerBindingParameter{}, Prepared: "return renvoRTGSyscallArgumentPolicy"},
	{Name: "jit_call_supported", Suffix: "JITCallSupported", Function: "renvoJITCallSupported", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return renvoRTGJITCallSupported"},
	{Name: "open_flag", Suffix: "OpenFlag", Function: "renvoTargetOpenFlag", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "int", Failure: "-1", Parameters: []compilerBindingParameter{{"flag", "int"}}, Prepared: "if flag == renvoOpenReadOnly { return 0 }\nif flag == renvoOpenWriteOnly { return 1 }\nif flag == renvoOpenReadWrite { return 2 }\nif flag == renvoOpenCreate { return renvoRTGOpenCreate }\nif flag == renvoOpenTruncate { return renvoRTGOpenTruncate }\nreturn -1"},
	{Name: "file_offset_sentinel", Suffix: "FileOffsetSentinel", Function: "renvoFileOffsetSentinel", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return false"},
	{Name: "static_call_binding", Suffix: "StaticCallBinding", Function: "renvoTargetStaticCallBinding", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "int", Failure: "0", Parameters: []compilerBindingParameter{{"src", "[]byte"}, {"libraryStart", "int"}, {"libraryEnd", "int"}}, Prepared: "return 1"},
	{Name: "linked_static_import", Suffix: "LinkedStaticImport", Function: "renvoAsmAddLinkedStaticImport", Result: "int", Failure: "-1", Parameters: []compilerBindingParameter{{"libraryStart", "int"}, {"libraryEnd", "int"}, {"nameStart", "int"}, {"nameEnd", "int"}, {"src", "[]byte"}}, Prepared: "if targetIsKernelModule(a.c) {\nif !renvoBytesEqualText(src, libraryStart, libraryEnd, \"kernel\") { return -1 }\nreturn renvoAsmAddKernelImport(a, src, nameStart, nameEnd)\n}\nreturn renvoAsmAddPreparedStaticImport(a, libraryStart, libraryEnd, nameStart, nameEnd, src)"},
	{Name: "finish_static_call_shape", Suffix: "FinishStaticCallShape", Function: "renvoAsmFinishStaticCallShape", Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"integerCount", "int"}, {"floatCount", "int"}, {"allInteger", "bool"}, {"resultRegister", "int"}, {"resultKind", "int"}}, Prepared: "if renvoRTGStaticCallPolicy != renvoStaticCallSplitRegisters { return false }\nif !allInteger \u0026\u0026 (integerCount \u003e 8 || floatCount \u003e 8) { return false }\nif resultKind != renvoStaticCallInteger \u0026\u0026 (resultRegister \u003c 0 || resultRegister \u003e= 8) { return false }\nif resultKind == renvoStaticCallFloat32 { resultRegister += 8 }\na.staticCallResultFloat = resultRegister\nreturn true"},
	{Name: "static_call_policy", Suffix: "StaticCallPolicy", Function: "renvoTargetStaticCallPolicy", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "int", Failure: "renvoStaticCallUnavailable", Parameters: []compilerBindingParameter{}, Prepared: "return renvoRTGStaticCallPolicy"},
	{Name: "compact_c_value_helpers", Suffix: "CompactCValueHelpers", Function: "renvoCompactCValueHelpers", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return false"},
	{Name: "condition_branch", Suffix: "ConditionBranch", Function: "renvoAsmConditionBranch", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"condition", "int"}, {"label", "int"}}, Prepared: "renvoRTGDirectJumpCondition(a, renvoRTGConditionFromSemantic(condition), label)"},
	{Name: "compare_stack_immediate_jump", Suffix: "CompareStackImmediateJump", Function: "renvoAsmCompareStackImmediateJump", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"offset", "int"}, {"value", "int"}, {"label", "int"}, {"condition", "int"}}, Prepared: "renvoAsmPushStack(a, offset)\nrenvoAsmPrimaryImm(a, value)\nrenvoAsmPopTertiary(a)\nrenvoAsmCompareJump(a, condition, label)"},
	{Name: "compare_stack_stack_jump", Suffix: "CompareStackStackJump", Function: "renvoAsmCompareStackStackJump", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"left", "int"}, {"right", "int"}, {"label", "int"}, {"condition", "int"}}, Prepared: "renvoAsmPushStack(a, left)\nrenvoAsmLoadPrimaryStack(a, right)\nrenvoAsmPopTertiary(a)\nrenvoAsmCompareJump(a, condition, label)"},
	{Name: "compare_jump", Suffix: "CompareJump", Function: "renvoAsmCompareJump", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"condition", "int"}, {"label", "int"}}, Prepared: "renvoRTGDirectCompare(a, renvoRTGTertiary, renvoRTGPrimary)\nrenvoRTGDirectJumpCondition(a, renvoRTGConditionFromSemantic(condition), label)"},
	{Name: "compare_set", Suffix: "CompareSet", Function: "renvoAsmCompareSet", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"condition", "int"}}, Prepared: "renvoRTGDirectCompare(a, renvoRTGTertiary, renvoRTGPrimary)\nrenvoRTGDirectSetCondition(a, renvoRTGConditionFromSemantic(condition), renvoRTGPrimary)"},
	// Allows shared symbolic pointer analysis to request absolute symbol-address emission.
	{Name: "object_absolute_symbols", Suffix: "ObjectAbsoluteSymbols", Function: "renvoObjectAbsoluteSymbols", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return renvoIsSysVObject(c)"},
	// Initializes persistent object arena bounds at first use when no process entry initializes them.
	{Name: "object_lazy_arena", Suffix: "ObjectLazyArena", Function: "renvoObjectLazyArena", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return renvoIsSysVObject(c)"},
	// Uses a bounded trap helper instead of a process-oriented uncaught-fault runtime.
	{Name: "object_faults_trap", Suffix: "ObjectFaultsTrap", Function: "renvoObjectFaultsTrap", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return renvoIsSysVObject(c)"},
	// Requests local function symbols around separately emitted runtime helper bodies.
	{Name: "object_helper_symbols", Suffix: "ObjectHelperSymbols", Function: "renvoObjectHelperSymbols", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return renvoIsSysVObject(c)"},
	// Allows a small integer aggregate result in the primary/secondary object word pair.
	{Name: "object_pair_result", Suffix: "ObjectPairResult", Function: "renvoObjectPairResult", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return renvoIsSysVObject(c)"},
	// Selects shared integer register/overflow-stack foreign-call planning; argument locations and calls are physical operations.
	{Name: "object_register_scalar_abi", Suffix: "ObjectRegisterScalarABI", Function: "renvoObjectRegisterScalarABI", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return renvoIsSysVObject(c)"},
	// Selects the shared stack-word scalar adapter: one word per parameter and
	// an optional two-word scalar result. Physical locations remain target hooks.
	{Name: "object_stack_scalar_abi", Suffix: "ObjectStackScalarABI", Function: "renvoObjectStackScalarABI", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return renvoIsCdeclObject(c)"},
	// These optimization policies are queried only for object compilation. Keep
	// the cheap mode rejection at shared call sites: executable compilation does
	// not need target dispatch or object-specific analysis bookkeeping.
	// Enables shared direct-use counting and propagation of constants into singly-called private functions.
	{Name: "single_call_constants", Suffix: "SingleCallConstants", Function: "renvoSingleCallConstants", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return renvoIsSysVObject(c)"},
	// Allows omission of empty calls only after shared argument side-effect and function-body checks.
	{Name: "elide_empty_calls", Suffix: "ElideEmptyCalls", Function: "renvoElideEmptyCalls", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return renvoIsSysVObject(c)"},
	// Allows the shared evaluator to fold eligible zero-argument calls; source purity and recursion checks remain in the core.
	{Name: "pure_call_constants", Suffix: "PureCallConstants", Function: "renvoPureCallConstants", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return renvoIsSysVObject(c)"},
	// Controls local scalar constant tracking and flow-sensitive evaluation; does not select a calling convention.
	{Name: "flow_constant_propagation", Suffix: "FlowConstantPropagation", Function: "renvoFlowConstantPropagation", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return renvoIsSysVObject(c)"},
	{Name: "reset_program_emission", Suffix: "ResetProgramEmission", Function: "renvoResetProgramEmission", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{}, Prepared: "renvoRTGUnsupportedOperation = 0\nrenvoRTGFailureDetail = -1\nrenvoRTGImageLimitMemory = false\nrenvoRTGImageLimitNeeded = 0\nrenvoRTGImageLimit = 0"},
	{Name: "function_address_layout", Suffix: "FunctionAddressLayout", Function: "renvoFunctionAddressLayout", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return (renvoFixedTarget == 0 || renvoRTGPreparedObject != 0) && c.objectFile"},
	{Name: "reconstruct_wide_argument", ReachabilityGuard: "renvoMayReconstructWideArgument", Suffix: "ReconstructWideArgument", Function: "renvoReconstructWideArgument", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return c.renvoNativeIntSize == 8"},
	{Name: "scaled_atom_literals", ReachabilityGuard: "renvoMayUseScaledAtomLiterals", Suffix: "ScaledAtomLiterals", Function: "renvoScaledAtomLiterals", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return true"},
	{Name: "wide_float_locals", ReachabilityGuard: "renvoMayUseWideFloatLocals", Suffix: "WideFloatLocals", Function: "renvoWideFloatLocals", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return renvoRTGPreparedIEEEFloat != 0"},
	{Name: "scaled_float64_values", ReachabilityGuard: "renvoMayUseScaledFloat64Values", Suffix: "ScaledFloat64Values", Function: "renvoScaledFloat64Values", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return renvoRTGPreparedIEEEFloat == 0"},
	{Name: "result_copy_via_frame", ReachabilityGuard: "renvoMayCopyResultViaFrame", Suffix: "ResultCopyViaFrame", Function: "renvoResultCopyViaFrame", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return true"},
	{Name: "reserve_function_labels", Suffix: "ReserveFunctionLabels", Function: "renvoReserveFunctionLabels", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"mode", "int"}}, Prepared: "return false"},
	{Name: "optimize_program_runtime", Suffix: "OptimizeProgramRuntime", Function: "renvoOptimizeProgramRuntime", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return true"},
	{Name: "fixed_object_cabi", ReachabilityGuard: "renvoMayCompileFixedObjectCABI", Suffix: "FixedObjectCABI", Function: "renvoFixedObjectCABI", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return true"},
	{Name: "save_slice_slot_addresses", Suffix: "SaveSliceSlotAddresses", Function: "renvoAsmSaveSliceSlotAddresses", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"dataSlot", "int"}, {"lenSlot", "int"}, {"capSlot", "int"}}, Prepared: "renvoRTGSaveSliceSlotAddresses(a, dataSlot, lenSlot, capSlot)"},
	{Name: "inline_append", ReachabilityGuard: "renvoMayInlineAppend", Suffix: "InlineAppend", Function: "renvoInlineAppend", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return true"},
	{Name: "string_arguments_from_frame", ReachabilityGuard: "renvoMayLoadStringArgumentsFromFrame", Suffix: "StringArgumentsFromFrame", Function: "renvoStringArgumentsFromFrame", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return true"},
	{Name: "tuple_parameter_layout", ReachabilityGuard: "renvoMayUseTupleParameterLayout", Suffix: "TupleParameterLayout", Function: "renvoTupleParameterLayout", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return true"},
	{Name: "hosted_object", Suffix: "HostedObject", Function: "renvoTargetHostedObject", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "if renvoRTGPreparedObject != 0 {\n\treturn c != nil \u0026\u0026 c.objectFile \u0026\u0026 !targetIsKernelModule(c)\n}\nreturn renvoIsSysVObject(c) || renvoIsCdeclObject(c)"},
	{Name: "retain_panic_runtime", Suffix: "RetainPanicRuntime", Function: "renvoRetainPanicRuntime", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return true"},
	{Name: "stable_function_order", ReachabilityGuard: "renvoMayRequireStableFunctionOrder", Suffix: "StableFunctionOrder", Function: "renvoStableFunctionOrder", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return true"},
	{Name: "program_failure_exit_code", Suffix: "ProgramFailureExitCode", Function: "renvoProgramFailureExitCode", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "int", Failure: "1", Parameters: []compilerBindingParameter{}, Prepared: "// Preserve the image-size rejection status for command-line embedders.\nif renvoRTGUnsupportedOperation == 5001 {\n\treturn 125\n}\nreturn 1"},
	{Name: "program_result_valid", Suffix: "ProgramResultValid", Function: "renvoProgramResultValid", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"result", "*renvoCompileResult"}}, Prepared: "a := \u0026g.asm\nrenvoRTGValidateRelocations(a)\nif renvoRTGUnsupportedOperation != 0 {\n\trenvoRTGReportFailure(g)\n\treturn false\n}\nif len(result.data) == 0 \u0026\u0026 !renvoObjectProgram(g.c) \u0026\u0026 !renvoKernelProgram(g.c) {\n\tif renvoRTGImageLimit \u003e 0 {\n\t\trenvoRTGReportImageSize(g)\n\t} else {\n\t\trenvoPrintErr(\"renvo: error RENVO-BUG-020 (backend): target image encoder returned no output or diagnostic\\n\")\n\t}\n\trenvoRTGUnsupportedOperation = 5001\n\treturn false\n}\nreturn true"},
	{Name: "program_emission_valid", Suffix: "ProgramEmissionValid", Function: "renvoProgramEmissionValid", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "if renvoRTGUnsupportedOperation != 0 {\n\trenvoRTGReportFailure(g)\n\treturn false\n}\nreturn true"},
	{Name: "label_notifications", ReachabilityGuard: "renvoMayNotifyLabels", Suffix: "LabelNotifications", Function: "renvoLabelNotifications", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return true"},
	{Name: "unsupported_operation", Suffix: "UnsupportedOperation", Function: "renvoAsmUnsupportedOperation", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"code", "int"}}, Prepared: "if renvoRTGUnsupportedOperation == 0 {\n\trenvoRTGUnsupportedOperation = code\n}"},
	{Name: "label_boundary", Suffix: "LabelBoundary", Function: "renvoAsmLabelBoundary", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"label", "int"}}, Prepared: "renvoRTGMarkLabel(a, label)"},
	{Name: "structured_string_equal_body", Suffix: "StructuredStringEqualBody", Function: "renvoEmitStructuredStringEqualBody", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{}, Prepared: "renvoRTGEmitStringEqualHelperBody(g)"},
	{Name: "helper_function_boundary", Suffix: "HelperFunctionBoundary", Function: "renvoAsmHelperFunctionBoundary", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"label", "int"}, {"start", "bool"}}, Prepared: "if start {\n\trenvoRTGFunctionStart(a, label)\n} else {\n\trenvoRTGFunctionFinish(a)\n}"},
	{Name: "structured_functions", ReachabilityGuard: "renvoMayUseStructuredFunctions", Suffix: "StructuredFunctions", Function: "renvoUsesStructuredFunctions", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return renvoRTGStructuredFunctions != 0"},
	{Name: "object_variadic_word_limit", Suffix: "ObjectVariadicWordLimit", Function: "renvoObjectVariadicWordLimit", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "int", Failure: "0", Parameters: []compilerBindingParameter{}, Prepared: "return 0"},
	{Name: "object_stack_arguments", Suffix: "ObjectStackArguments", Function: "renvoObjectStackArguments", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return false"},
	{Name: "string_compare_arguments", Suffix: "StringCompareArguments", Function: "renvoAsmStringCompareArguments", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"left", "int"}, {"leftLength", "int"}, {"right", "int"}, {"rightLength", "int"}}, Prepared: "renvoRTGAsmLoadFrame(a, renvoRTGCallWord0, left)\nrenvoRTGAsmLoadFrame(a, renvoRTGCallWord1, leftLength)\nrenvoRTGAsmLoadFrame(a, renvoRTGCallWord2, right)\nrenvoRTGAsmLoadFrame(a, renvoRTGCallWord3, rightLength)"},
	{Name: "object_absolute_address", Suffix: "ObjectAbsoluteAddress", Function: "renvoAsmObjectAbsoluteAddress", Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"targetStart", "int"}, {"targetEnd", "int"}, {"addend", "int"}}, Prepared: "// Absolute symbol value is required before the final kernel mapping is active.\nrenvoAsmEmit16(a, 0xb848)\nsourceLabel := renvoAsmNewLabel(a)\nrenvoAsmMarkLabel(a, sourceLabel)\nrenvoAsmEmit64(a, 0)\na.objectDataRelocs = append(a.objectDataRelocs, renvoObjectDataRelocation{\n\toffset: -sourceLabel - 1, targetStart: targetStart, targetEnd: targetEnd, typ: 1, addend: addend})\nreturn true"},
	{Name: "object_fault_trap", Suffix: "ObjectFaultTrap", Function: "renvoAsmObjectFaultTrap", Result: "", Failure: "", Parameters: []compilerBindingParameter{}, Prepared: "renvoAsmEmit16(a, 0x0b0f)"},
	{Name: "object_indirect_aggregate", Suffix: "ObjectIndirectAggregate", Function: "renvoObjectIndirectAggregate", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return false"},
	{Name: "object_word_limit", Suffix: "ObjectWordLimit", Function: "renvoObjectWordLimit", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "int", Failure: "0", Parameters: []compilerBindingParameter{{"export", "bool"}}, Prepared: "if export { return 20 }\nlimit := renvoRTGObjectRegisterCount()\nif limit \u003e 20 { limit = 20 }\nreturn limit"},
	{Name: "object_argument_word_bytes", Suffix: "ObjectArgumentWordBytes", Function: "renvoObjectArgumentWordBytes", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "int", Failure: "0", Parameters: []compilerBindingParameter{}, Prepared: "return renvoRTGStackWordBytes"},
	{Name: "source_token_capacity", Suffix: "SourceTokenCapacity", Function: "renvoSourceTokenCapacity", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "int", Failure: "0", Parameters: []compilerBindingParameter{{"length", "int"}}, Prepared: "return length/4 + 8192"},
	{Name: "source_soft_float", Suffix: "SourceSoftFloat", Function: "renvoSourceSoftFloat", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return false"},
	{Name: "source_scratch", Suffix: "SourceScratch", Function: "renvoSourceScratch", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return false"},
	{Name: "source_capacity", Suffix: "SourceCapacity", Function: "renvoSourceCapacity", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "int", Failure: "0", Parameters: []compilerBindingParameter{}, Prepared: "return 0"},
	{Name: "empty_function", Suffix: "EmptyFunction", Function: "renvoEmitEmptyFunction", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"label", "int"}}, Prepared: "renvoRTGFunctionStart(a, label)\nrenvoAsmMarkLabel(a, label)\nrenvoAsmRet(a)\nrenvoRTGFunctionFinish(a)"},
	{Name: "resolve_unemitted_closures", Suffix: "ResolveUnemittedClosures", Function: "renvoResolveUnemittedClosures", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return true"},
	{Name: "object_program", Suffix: "ObjectProgram", Function: "renvoObjectProgram", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return renvoRTGPreparedObject != 0"},
	{Name: "program_image_entry", Suffix: "ProgramImageEntry", Function: "renvoProgramImageEntry", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return false"},
	{Name: "program_cache_supported", Suffix: "ProgramCacheSupported", Function: "renvoProgramCacheSupported", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return false"},
	{Name: "program_target_mode", Suffix: "ProgramTargetMode", Function: "renvoProgramTargetMode", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "int", Failure: "0", Parameters: []compilerBindingParameter{}, Prepared: "return 0"},
	{Name: "kernel_program", Suffix: "KernelProgram", Function: "renvoKernelProgram", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return renvoRTGPreparedKernelModule != 0 && !c.objectFile"},
	{Name: "program_image", Suffix: "ProgramImage", Function: "renvoBuildProgramImage", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"initLabel", "int"}, {"exitLabel", "int"}, {"result", "*renvoCompileResult"}}, Prepared: "if renvoRTGPreparedObject == 0 {\n\trenvoAsmPatch(a)\n}\nif renvoRTGPreparedKernelModule != 0 \u0026\u0026 renvoRTGPreparedObject == 0 {\n\tresult.data = renvoRTGKernelImage(a, initLabel, exitLabel)\n} else {\n\tresult.data = renvoRTGImage(a)\n\tif renvoRTGPreparedObject == 0 \u0026\u0026 renvoFixedTarget == 0 \u0026\u0026 a.c.emitImage {\n\t\tresult.data = renvoAppendReplLinkTable(result.data, a)\n\t}\n}"},
	{Name: "finalize_object_code", Suffix: "FinalizeObjectCode", Function: "renvoFinalizeObjectCode", Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return true"},
	{Name: "release_program_scratch", Suffix: "ReleaseProgramScratch", Function: "renvoReleaseProgramScratch", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return false"},
	{Name: "release_program_declarations", Suffix: "ReleaseProgramDeclarations", Function: "renvoReleaseProgramDeclarations", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return false"},
	{Name: "program_layout", Suffix: "ProgramLayout", Function: "renvoSetupProgramLayout", Result: "int", Failure: "-2", Parameters: []compilerBindingParameter{{"image", "bool"}, {"functionCount", "int"}}, Prepared: "a.codeOffset = renvoRTGCodeOffset\nif renvoRTGPreparedKernelModule != 0 {\n\treturn -1\n}\noffset := -1\nif renvoRTGEntryStateBytes \u003e 0 {\n\toffset = a.ReserveBSS(renvoRTGEntryStateBytes, renvoRTGStackWordBytes)\n}\nif !renvoRTGEmitEntryStart(a, offset) {\n\treturn -2\n}\nreturn offset"},
	{Name: "compact_c_value_helper", Suffix: "CompactCValueHelper", Function: "renvoEmitCompactCValueHelperBody", Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"label", "int"}, {"size", "int"}, {"signedValue", "bool"}, {"postDec", "bool"}}, Prepared: "return false"},
	{Name: "function_override", Suffix: "FunctionOverride", Function: "renvoEmitFunctionOverride", Result: "int", Failure: "-1", Parameters: []compilerBindingParameter{{"declIndex", "int"}, {"label", "int"}}, Prepared: "return renvoRTGEmitAssemblyFunction(a, declIndex, label)"},
	{Name: "zero_void_return", Suffix: "ZeroVoidReturn", Function: "renvoZeroVoidReturn", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return false"},
	{Name: "function_frame_finish", Suffix: "FunctionFrameFinish", Function: "renvoFunctionFrameFinish", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"framePatch", "int"}}, Prepared: "renvoRTGFrameFinish(\u0026g.asm, framePatch, g.stackPeak)\nrenvoRTGFunctionFinish(\u0026g.asm)"},
	{Name: "function_frame_start", Suffix: "FunctionFrameStart", Function: "renvoFunctionFrameStart", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "int", Failure: "-1", Parameters: []compilerBindingParameter{{"label", "int"}}, Prepared: "a := &g.asm\nrenvoRTGFunctionStart(a, label)\nrenvoAsmMarkLabel(a, label)\nreturn renvoRTGFrameStart(a)"},
	{Name: "store_hidden_result", Suffix: "StoreHiddenResult", Function: "renvoStoreHiddenResult", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"offset", "int"}}, Prepared: "renvoRTGStoreParamWord(g, 0, offset)"},
	{Name: "struct_argument_by_reference", ReachabilityGuard: "renvoMayPassStructPointers", Suffix: "StructArgumentByReference", Function: "renvoTargetStructArgumentByReference", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return c.renvoNativeIntSize == 4 || c.renvoNativeIntSize == 2"},
	{Name: "resolves_static_import", Suffix: "ResolvesStaticImport", Function: "renvoTargetResolvesStaticImport", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"absoluteLibrary", "bool"}}, Prepared: "return true"},
	{Name: "entry_runtime_registers", Suffix: "EntryRuntimeRegisters", Function: "renvoEmitEntryRuntimeRegisters", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{}, Prepared: "return"},
	{Name: "program_exit", Suffix: "ProgramExit", Function: "renvoEmitProgramExit", Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"image", "bool"}}, Prepared: "if image { return false }; return renvoRTGEmitExit(a, renvoRTGPrimary)"},
	{Name: "program_entry_frame", Suffix: "ProgramEntryFrame", Function: "renvoEmitProgramEntryFrame", Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"image", "bool"}}, Prepared: "return !image"},
	{Name: "image_entry_words", Suffix: "ImageEntryWords", Function: "renvoEmitImageEntryWords", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"paramCount", "int"}}, Prepared: "return false"},
	{Name: "process_entry_words", Suffix: "ProcessEntryWords", Function: "renvoEmitProcessEntryWords", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"paramCount", "int"}, {"entryStateOffset", "int"}}, Prepared: "return renvoRTGEmitEntry(&g.asm, paramCount, entryStateOffset)"},
	{Name: "object_reverse_register_call", Suffix: "ObjectReverseRegisterCall", Function: "renvoAsmObjectReverseRegisterCall", Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"importID", "int"}, {"wordCount", "int"}}, Prepared: "for i := 0; i \u003c wordCount/2; i++ {\n\tleft := renvoRTGAsmAddress(renvoRTGStack, RTGNoRegister, i*renvoRTGStackWordBytes, 1)\n\tright := renvoRTGAsmAddress(renvoRTGStack, RTGNoRegister, (wordCount-1-i)*renvoRTGStackWordBytes, 1)\n\trenvoRTGDirectLoadNative(a, renvoRTGPrimary, left)\n\trenvoRTGDirectLoadNative(a, renvoRTGTertiary, right)\n\trenvoRTGDirectStoreNative(a, left, renvoRTGTertiary)\n\trenvoRTGDirectStoreNative(a, right, renvoRTGPrimary)\n}\nreturn renvoRTGEmitStaticCall(a, importID, wordCount)"},
	{Name: "local_storage_unit", Suffix: "LocalStorageUnit", Function: "renvoLocalStorageUnit", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "int", Failure: "renvoBackendValueSlotSize", Parameters: []compilerBindingParameter{{"compactScalar", "bool"}}, Prepared: "return renvoBackendValueSlotSize"},
	{Name: "kernel_callback_address", Suffix: "KernelCallbackAddress", Function: "renvoAsmKernelCallbackAddress", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"label", "int"}}, Prepared: "renvoRTGKernelCallbackAddress(a, label)"},
	{Name: "kernel_entry_return", Suffix: "KernelEntryReturn", Function: "renvoAsmKernelEntryReturn", Result: "", Failure: "", Parameters: []compilerBindingParameter{}, Prepared: "renvoRTGKernelEntryEpilogue(a)"},
	{Name: "kernel_entry_frame", Suffix: "KernelEntryFrame", Function: "renvoEmitKernelEntryFrame", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{}, Prepared: "renvoRTGKernelEntryPrologue(&g.asm)"},
	{Name: "object_aggregate_register_bytes", Suffix: "ObjectAggregateRegisterBytes", Function: "renvoObjectAggregateRegisterBytes", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "int", Failure: "0", Parameters: []compilerBindingParameter{}, Prepared: "return renvoRTGObjectAggregateRegisterBytes"},
	{Name: "hosted_static_call", Suffix: "HostedStaticCall", Function: "renvoAsmHostedStaticCall", Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"importID", "int"}, {"wordCount", "int"}}, Prepared: "return renvoRTGEmitStaticCall(a, importID, wordCount)"},
	{Name: "object_register_call", Suffix: "ObjectRegisterCall", Function: "renvoAsmObjectRegisterCall", Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"importID", "int"}, {"wordCount", "int"}, {"vectorMask", "int"}}, Prepared: "return renvoRTGEmitStaticCall(a, importID, wordCount|vectorMask<<8)"},
	{Name: "cdecl_object_call", Suffix: "CdeclObjectCall", Function: "renvoAsmCdeclObjectCall", Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"importID", "int"}, {"wordCount", "int"}, {"variadic", "bool"}}, Prepared: "return false"},
	{Name: "discard_arena_pages", Suffix: "DiscardArenaPages", Function: "renvoAsmDiscardArenaPages", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"startOff", "int"}, {"endOff", "int"}, {"lenOff", "int"}}, Prepared: "if renvoRTGDiscardPageSize == 0 { renvoAsmUnsupportedOperation(a, 1930); return }\ndoneLabel := renvoAsmNewLabel(a)\n\trenvoAsmLoadPrimaryStack(a, startOff)\n\trenvoRTGDirectMoveImmediate(a, renvoRTGScratch, renvoRTGDiscardPageSize-1)\nrenvoRTGDirectAdd(a, renvoRTGPrimary, renvoRTGScratch)\n\trenvoRTGDirectMoveImmediate(a, renvoRTGScratch, -renvoRTGDiscardPageSize)\nrenvoRTGDirectBitAnd(a, renvoRTGPrimary, renvoRTGScratch)\n\trenvoAsmStorePrimaryStack(a, startOff)\n\trenvoAsmLoadPrimaryStack(a, endOff)\n\trenvoRTGDirectMoveImmediate(a, renvoRTGScratch, -renvoRTGDiscardPageSize)\nrenvoRTGDirectBitAnd(a, renvoRTGPrimary, renvoRTGScratch)\n\trenvoAsmLoadTertiaryStack(a, startOff)\n\trenvoAsmSubPrimaryTertiary(a)\n\trenvoAsmStorePrimaryStack(a, lenOff)\n\trenvoAsmCmpPrimaryImm8(a, 0)\n\trenvoRTGDirectJumpCondition(a, renvoRTGConditionFromSetcc(0x9e), doneLabel)\n\trenvoAsmLoadPrimaryStack(a, startOff)\n\trenvoRTGDirectMove(a, renvoRTGSyscallWord0, renvoRTGPrimary)\n\trenvoAsmLoadPrimaryStack(a, lenOff)\n\trenvoRTGDirectMove(a, renvoRTGSyscallWord1, renvoRTGPrimary)\n\trenvoRTGDirectMoveImmediate(a, renvoRTGSyscallWord2, renvoRTGDiscardAdvice)\n\trenvoRTGDirectMoveImmediate(a, renvoRTGSyscallNumber, renvoRTGDiscardNumber)\n\trenvoRTGRecordDiscardSyscall(a)\nrenvoRTGDirectHostSyscall(a)\n\trenvoAsmMarkLabel(a, doneLabel)"},
	{Name: "arena_discard_supported", Suffix: "ArenaDiscardSupported", Function: "renvoArenaDiscardSupported", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return renvoRTGDiscardPageSize != 0"},
	{Name: "write_value_regs", Suffix: "WriteValueRegs", Function: "renvoEmitTargetWriteValueRegs", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"fd", "int"}}, Prepared: "a := \u0026g.asm\nrenvoRTGDirectMove(a, renvoRTGCallWord2, renvoRTGSecondary)\nrenvoRTGDirectMove(a, renvoRTGCallWord1, renvoRTGPrimary)\nrenvoRTGDirectMoveImmediate(a, renvoRTGCallWord0, int64(fd))\nreturn renvoRTGEmitRuntimeOperation(a, RTGRuntimeWrite)"},
	{Name: "exit_status", Suffix: "ExitStatus", Function: "renvoAsmExitStatus", Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return renvoRTGEmitExit(a, renvoRTGPrimary)"},
	{Name: "syscall_from_stack", Suffix: "SyscallFromStack", Function: "renvoAsmSyscallFromStack", Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"wordCount", "int"}, {"syscallNumber", "int"}}, Prepared: "if wordCount \u003c 1 || wordCount \u003e 7 || renvoRTGSyscallArgumentPolicy == 2 \u0026\u0026 syscallNumber \u003c 0 {\n\treturn false\n}\nif renvoRTGCustomSyscall { return renvoRTGEmitCustomSyscall(a, wordCount, syscallNumber) }\nregisters := []RTGRegister{\n\trenvoRTGSyscallNumber,\n\trenvoRTGSyscallWord0, renvoRTGSyscallWord1, renvoRTGSyscallWord2,\n\trenvoRTGSyscallWord3, renvoRTGSyscallWord4, renvoRTGSyscallWord5,\n}\nfor i := 0; i \u003c wordCount; i++ {\n\tif !registers[i].Valid {\n\t\treturn false\n\t}\n}\nfor i := 0; i \u003c wordCount; i++ {\n\trenvoRTGAsmPopRegister(a, registers[i])\n}\nrenvoRTGRecordRawSyscall(a, syscallNumber)\nrenvoRTGDirectHostSyscall(a)\nif renvoRTGSyscallResult.Valid \u0026\u0026\n\trenvoRTGSyscallResult.Code != renvoRTGPrimary.Code {\n\trenvoRTGDirectMove(a, renvoRTGPrimary, renvoRTGSyscallResult)\n}\nreturn true"},
	{Name: "jit_call_from_stack", Suffix: "JITCallFromStack", Function: "renvoAsmJITCallFromStack", Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "entry := renvoRTGScratch\nstackTop := renvoRTGPrimary\nargsData := renvoRTGCallWord0\nargsLen := renvoRTGCallWord1\nenvData := renvoRTGTertiary\nenvLen := renvoRTGSecondary\nrenvoRTGAsmPopRegister(a, entry)\nrenvoRTGAsmPopRegister(a, stackTop)\nrenvoRTGAsmPopRegister(a, argsData)\nrenvoRTGAsmPopRegister(a, argsLen)\nrenvoRTGAsmPopRegister(a, envData)\nrenvoRTGAsmPopRegister(a, envLen)\nreturn renvoRTGEmitJITCall(a, entry, stackTop, argsData, argsLen, envData, envLen)"},
	{Name: "runtime_stack_helpers", Suffix: "RuntimeStackHelpers", Function: "renvoAsmRuntimeStackHelpers", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"init", "int"}, {"switchStack", "int"}, {"fnLabel", "int"}}, Prepared: "\ta.patchFailed = true"},
	{Name: "object_indirect_register_call", Suffix: "ObjectIndirectRegisterCall", Function: "renvoAsmObjectIndirectRegisterCall", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"handleOffset", "int"}}, Prepared: "renvoRTGAsmLoadFrame(a, renvoRTGScratch, handleOffset)\nrenvoRTGDirectCallIndirect(a, renvoRTGScratch)"},
	{Name: "load_object_argument_word", Suffix: "LoadObjectArgumentWord", Function: "renvoAsmLoadObjectArgumentWord", Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"word", "int"}, {"offset", "int"}}, Prepared: "registers := renvoRTGObjectRegisters()\nif word \u003c 0 || word \u003e= len(registers) {\n\treturn false\n}\nrenvoRTGAsmLoadFrame(a, registers[word], offset)\nreturn true"},
	{Name: "object_indirect_stack_call", Suffix: "ObjectIndirectStackCall", Function: "renvoAsmObjectIndirectStackCall", Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"handleOffset", "int"}, {"argOffsets", "[]int"}}, Prepared: "\treturn false"},
	{Name: "object_integer_stack_call", Suffix: "ObjectIntegerStackCall", Function: "renvoAsmObjectIntegerStackCall", Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"importID", "int"}, {"wordCount", "int"}}, Prepared: "\treturn false"},
	{Name: "finish_object_variadic_args", Suffix: "FinishObjectVariadicArgs", Function: "renvoFinishObjectVariadicArgs", Result: "", Failure: "", Parameters: []compilerBindingParameter{}, Prepared: "\ta.patchFailed = true"},
	{Name: "push_object_variadic_args", Suffix: "PushObjectVariadicArgs", Function: "renvoPushObjectVariadicArgs", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"fixedWords", "int"}}, Prepared: "\ta.patchFailed = true"},
	{Name: "reserve_object_variadic_args", Suffix: "ReserveObjectVariadicArgs", Function: "renvoReserveObjectVariadicArgs", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"fixedCount", "int"}}, Prepared: "\tg.asm.patchFailed = true"},
	{Name: "object_call_with_word_count", Suffix: "ObjectCallWithWordCount", Function: "renvoObjectCallWithWordCount", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"fnIndex", "int"}, {"wordCount", "int"}}, Prepared: "\trenvoRTGEmitCallWithWordCount(g, fnIndex, wordCount)"},
	{Name: "finish_object_aggregate_result", Suffix: "FinishObjectAggregateResult", Function: "renvoFinishObjectAggregateResult", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"resultWords", "int"}}, Prepared: "\trenvoRTGFinishObjectAggregateResult(a, resultWords)"},
	{Name: "push_object_private_result", Suffix: "PushObjectPrivateResult", Function: "renvoPushObjectPrivateResult", Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"wordCount", "int"}}, Prepared: "\treturn renvoRTGPushObjectPrivateResult(a, wordCount)"},
	{Name: "push_object_sret_pointer", Suffix: "PushObjectSRetPointer", Function: "renvoPushObjectSRetPointer", Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "\treturn renvoRTGPushObjectSRetPointer(a)"},
	{Name: "begin_object_aggregate_result", Suffix: "BeginObjectAggregateResult", Function: "renvoBeginObjectAggregateResult", Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"sret", "bool"}}, Prepared: "\treturn renvoRTGBeginObjectAggregateResult(a, sret)"},
	{Name: "begin_object_stack_args", Suffix: "BeginObjectStackArgs", Function: "renvoBeginObjectStackArgs", Result: "", Failure: "", Parameters: []compilerBindingParameter{}, Prepared: "\ta.patchFailed = true"},
	{Name: "push_object_register_word", Suffix: "PushObjectRegisterWord", Function: "renvoAsmPushObjectRegisterWordKind", Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"register", "int"}, {"kind", "int"}}, Prepared: "if !renvoRTGPushObjectCallWord(a, register) {\n\treturn false\n}\nif kind != 0 {\n\trenvoAsmPopPrimary(a)\n\trenvoAsmNormalizePrimaryForKind(a, kind)\n\trenvoAsmPushPrimary(a)\n}\nreturn true"},
	{Name: "push_object_stack_word", Suffix: "PushObjectStackWord", Function: "renvoAsmPushObjectStackWordKind", Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"word", "int"}, {"kind", "int"}}, Prepared: "return false"},
	{Name: "object_export_frame", Suffix: "ObjectExportFrame", Function: "renvoObjectExportFrame", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"reserve", "bool"}}, Prepared: "renvoRTGObjectExportFrame(&g.asm, reserve)"},
	{Name: "object_argument_register_count", Suffix: "ObjectArgumentRegisterCount", Function: "renvoObjectArgumentRegisterCount", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "int", Failure: "0", Parameters: []compilerBindingParameter{}, Prepared: "return renvoRTGObjectRegisterCount()"},
	{Name: "object_call_abi", Suffix: "ObjectCallABI", Function: "renvoTargetObjectCallABI", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "int", Failure: "0", Parameters: []compilerBindingParameter{}, Prepared: "return renvoRTGObjectCallABI"},
	{Name: "word_call_intrinsic", Suffix: "WordCallIntrinsic", Function: "renvoEmitWordCallIntrinsic", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "int", Failure: "0", Parameters: []compilerBindingParameter{{"ep", "*renvoExprParse"}, {"idx", "int"}}, Prepared: "if renvoExprIsIdentText(g.prog, ep, ep.exprs[idx].left, \"renvo_runtime_CKernelLinkAddress\") {\n\treturn renvoBoolInt(renvoEmitKernelLinkAddressCall(g, ep, idx))\n}\nif renvoFixedTarget == 0 {\n\treturn renvoEmitCNativeIntCall(g, ep, idx, \u0026ep.exprs[idx])\n}\nreturn -1"},
	{Name: "unsigned_word_order_result", Suffix: "UnsignedWordOrderResult", Function: "renvoEmitUnsignedWordOrderResult", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"op0", "byte"}, {"op1", "byte"}, {"opLen", "int"}, {"kind", "int"}}, Prepared: "if g.c.renvoNativeIntSize != 8 \u0026\u0026 g.c.renvoNativeIntSize != 4 {\n\treturn false\n}\nif !renvoEmitUnsignedPrimaryTertiaryCompare(g, op0, op1, opLen) {\n\treturn false\n}\nrenvoAsmNormalizePrimaryForKind(\u0026g.asm, kind)\nreturn true"},
	{Name: "word_constant_immediate", Suffix: "WordConstantImmediate", Function: "renvoAsmWordConstantImmediate", Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"kind", "int"}, {"value", "int"}}, Prepared: "if kind == renvoTypeInt64 || kind == renvoTypeUint64 {\n\treturn false\n}\nrenvoAsmPrimaryImm(a, value)\nreturn true"},
	{Name: "bounded_word_shift", Suffix: "BoundedWordShift", Function: "renvoEmitBoundedWordShift", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"tok", "int"}, {"right", "bool"}, {"leftUnsigned", "bool"}, {"resultUnsigned", "bool"}}, Prepared: "if right {\n\trenvoRTGEmitBoundedVariableShift(\u0026g.asm, RTGShiftRight, !leftUnsigned)\n} else {\n\trenvoRTGEmitBoundedVariableShift(\u0026g.asm, RTGShiftLeft, false)\n}\nreturn true"},
	{Name: "read_write_file", Suffix: "ReadWriteFile", Function: "renvoFinishFileReadWrite", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"operation", "int"}, {"hasOffset", "bool"}}, Prepared: "a := \u0026g.asm\nrenvoAsmPrepareReadWriteBuf(a)\nif hasOffset {\n\trenvoAsmPopReadWriteOffset(a)\n}\nrenvoAsmPopCallWord0(a)\nif hasOffset {\n\toperation += RTGRuntimeReadAt - RTGRuntimeRead\n}\nreturn renvoRTGEmitRuntimeOperation(a, operation)"},
	{Name: "pop_read_write_offset", Suffix: "PopReadWriteOffset", Function: "renvoAsmPopReadWriteOffset", Result: "", Failure: "", Parameters: []compilerBindingParameter{}, Prepared: "renvoRTGAsmPopRegister(a, renvoRTGCallWord3)"},
	{Name: "chmod_file", Suffix: "ChmodFile", Function: "renvoAsmChmodFile", Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return renvoRTGEmitRuntimeOperation(a, RTGRuntimeChmod)"},
	{Name: "close_file", Suffix: "CloseFile", Function: "renvoAsmCloseFile", Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return renvoRTGEmitRuntimeOperation(a, RTGRuntimeClose)"},
	{Name: "open_path_length", Suffix: "OpenPathLength", Function: "renvoOpenPathNeedsLength", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return false"},
	{Name: "open_file", Suffix: "OpenFile", Function: "renvoAsmOpenFile", Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "renvoRTGDirectMove(a, renvoRTGCallWord0, renvoRTGPrimary)\nrenvoRTGAsmPopRegister(a, renvoRTGCallWord1)\nrenvoRTGDirectMoveImmediate(a, renvoRTGCallWord2, 493)\nreturn renvoRTGEmitRuntimeOperation(a, RTGRuntimeOpen)"},
	{Name: "function_word_conversion", Suffix: "FunctionWordConversion", Function: "renvoSupportsFunctionWordConversion", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return false"},
	{Name: "named_function_address", Suffix: "NamedFunctionAddress", Function: "renvoCanTakeNamedFunctionAddress", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return true"},
	{Name: "address_taken_local", Suffix: "AddressTakenLocal", Function: "renvoAsmAddressTakenLocal", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"offset", "int"}}, Prepared: "renvoAsmAddressPrimaryStack(a, offset)"},
	{Name: "address_result_buffer", Suffix: "AddressResultBuffer", Function: "renvoAsmAddressResultBuffer", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"offset", "int"}}, Prepared: "renvoAsmAddressPrimaryStack(a, offset)"},
	{Name: "load_indirect_field_value", Suffix: "LoadIndirectFieldValue", Function: "renvoAsmLoadIndirectFieldValue", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"size", "int"}, {"nativeABI", "bool"}}, Prepared: "if nativeABI {\nrenvoAsmLoadPrimaryMemSecondaryDispSize(a, 0, size)\n} else {\nrenvoAsmLoadPrimaryMemSecondaryDisp(a, 0)\n}"},
	{Name: "load_frame_field_value", Suffix: "LoadFrameFieldValue", Function: "renvoAsmLoadFrameFieldValue", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"offset", "int"}, {"size", "int"}, {"nativeABI", "bool"}}, Prepared: "if nativeABI {\nrenvoAsmAddressPrimaryStack(a, offset)\nrenvoAsmCopyPrimaryToSecondary(a)\nrenvoAsmLoadPrimaryMemSecondaryDispSize(a, 0, size)\n} else {\nrenvoAsmLoadPrimaryStack(a, offset)\n}"},
	{Name: "direct_slice_count_selector", Suffix: "DirectSliceCountSelector", Function: "renvoCanLoadDirectSliceCountSelector", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return true"},
	{Name: "slice_count_result", Suffix: "SliceCountResult", Function: "renvoAsmSliceCountResult", Result: "", Failure: "", Parameters: []compilerBindingParameter{}, Prepared: "renvoAsmCopyTertiaryToPrimary(a)"},
	{Name: "move_read_write_offset", Suffix: "MoveReadWriteOffset", Function: "renvoAsmMoveOffsetArg", Result: "", Failure: "", Parameters: []compilerBindingParameter{}, Prepared: "renvoRTGDirectMove(a, renvoRTGCallWord3, renvoRTGPrimary)"},
	{Name: "prepare_read_write_buffer", Suffix: "PrepareReadWriteBuffer", Function: "renvoAsmPrepareReadWriteBuf", Result: "", Failure: "", Parameters: []compilerBindingParameter{}, Prepared: "renvoRTGDirectMove(a, renvoRTGCallWord1, renvoRTGPrimary)\nrenvoRTGDirectMove(a, renvoRTGCallWord2, renvoRTGTertiary)"},
	{Name: "compare_word_immediate_kind", Suffix: "CompareWordImmediateKind", Function: "renvoAsmCompareWordImmediateKind", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"imm", "int"}, {"kind", "int"}}, Prepared: "renvoAsmNormalizePrimaryForKind(a, kind)\nrenvoAsmCmpPrimaryImm8Discard(a, imm)"},
	{Name: "unsigned_pointer_ordering", Suffix: "UnsignedPointerOrdering", Function: "renvoUsesUnsignedPointerOrdering", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return false"},
	{Name: "logical_shift_primary_word_immediate", Suffix: "LogicalShiftPrimaryWordImmediate", Function: "renvoAsmLogicalShiftPrimaryWordImm", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"imm", "int"}}, Prepared: "renvoRTGDirectShiftRightUnsignedImmediate(a, renvoRTGPrimary, byte(imm))"},
	{Name: "bitwise_primary_tertiary", Suffix: "BitwisePrimaryTertiary", Function: "renvoAsmBitwisePrimaryTertiary", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"op", "byte"}}, Prepared: "if op == '\u0026' {\nrenvoRTGDirectBitAnd(a, renvoRTGPrimary, renvoRTGTertiary)\n} else if op == '|' {\nrenvoRTGDirectBitOr(a, renvoRTGPrimary, renvoRTGTertiary)\n} else if op == '^' {\nrenvoRTGDirectBitXor(a, renvoRTGPrimary, renvoRTGTertiary)\n} else {\na.patchFailed = true\n}"},
	{Name: "ensure_append_bytes_helper", Suffix: "EnsureAppendBytesHelper", Function: "renvoEnsureAppendBytesHelper", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "int", Failure: "0", Parameters: []compilerBindingParameter{}, Prepared: "g.asm.patchFailed = true\nreturn 0"},
	{Name: "append_bytes_helper", Suffix: "AppendBytesHelper", Function: "renvoHasAppendBytesHelper", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return false"},
	{Name: "unsigned_word_comparison", Suffix: "UnsignedWordComparison", Function: "renvoCanCompareUnsignedWord", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return true"},
	{Name: "direct_scalar_deref", Suffix: "DirectScalarDeref", Function: "renvoCanDirectScalarDeref", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return false"},
	{Name: "compound_local_assign", Suffix: "CompoundLocalAssign", Function: "renvoEmitCompoundLocalAssignPeephole", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "int", Failure: "-1", Parameters: []compilerBindingParameter{{"ep", "*renvoExprParse"}, {"idx", "int"}, {"offset", "int"}, {"tok", "int"}, {"op", "byte"}, {"kind", "int"}, {"size", "int"}}, Prepared: "return -1"},
	{Name: "self_binary_local_assign", Suffix: "SelfBinaryLocalAssign", Function: "renvoEmitSelfBinaryLocalAssignPeephole", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"ep", "*renvoExprParse"}, {"idx", "int"}, {"offset", "int"}}, Prepared: "return false"},
	{Name: "scalar_preserves_secondary", Suffix: "ScalarPreservesSecondary", Function: "renvoScalarPreservesSecondary", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"ep", "*renvoExprParse"}, {"idx", "int"}}, Prepared: "return false"},
	{Name: "direct_scalar_store", Suffix: "DirectScalarStore", Function: "renvoCanDirectScalarStore", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"size", "int"}}, Prepared: "return false"},
	{Name: "compound_pointer_memory", Suffix: "CompoundPointerMemory", Function: "renvoEmitCompoundPointerMemoryPeephole", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"ep", "*renvoExprParse"}, {"idx", "int"}, {"kind", "int"}, {"tok", "int"}}, Prepared: "return false"},
	{Name: "indexed_pointer_address", Suffix: "IndexedPointerAddress", Function: "renvoEmitIndexedPointerAddressPeephole", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "int", Failure: "-1", Parameters: []compilerBindingParameter{{"ep", "*renvoExprParse"}, {"idx", "int"}}, Prepared: "return -1"},
	{Name: "switch_case_peephole", Suffix: "SwitchCasePeephole", Function: "renvoEmitSwitchCasePeephole", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "int", Failure: "-1", Parameters: []compilerBindingParameter{{"ep", "*renvoExprParse"}, {"idx", "int"}, {"valueOffset", "int"}, {"matchLabel", "int"}, {"known", "bool"}, {"value", "int"}}, Prepared: "return -1"},
	{Name: "register_switch", Suffix: "RegisterSwitch", Function: "renvoCanKeepSwitchPrimary", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return false"},
	{Name: "string_equal_left_length", Suffix: "StringEqualLeftLength", Function: "renvoAsmStringEqualLeftLength", Result: "", Failure: "", Parameters: []compilerBindingParameter{}, Prepared: "renvoAsmCopyPrimaryToCallWord1(a)"},
	{Name: "folded_indexed_scalar_load", Suffix: "FoldedIndexedScalarLoad", Function: "renvoAsmFoldedIndexedScalarLoad", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"elementSize", "int"}, {"size", "int"}, {"signed", "bool"}}, Prepared: "a.patchFailed = true"},
	{Name: "can_fold_indexed_scalar_load", Suffix: "CanFoldIndexedScalarLoad", Function: "renvoCanFoldIndexedScalarLoad", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"ep", "*renvoExprParse"}, {"idx", "int"}, {"elementSize", "int"}, {"size", "int"}}, Prepared: "return false"},
	{Name: "local_immediate_compare_jump", Suffix: "LocalImmediateCompareJump", Function: "renvoEmitLocalImmediateCompareJump", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"offset", "int"}, {"value", "int"}, {"c0", "byte"}, {"c1", "byte"}, {"label", "int"}, {"jumpIfTrue", "bool"}, {"unsigned", "bool"}}, Prepared: "return false"},
	{Name: "deref_compare_jump", Suffix: "DerefCompareJump", Function: "renvoEmitDerefCompareJump", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"ep", "*renvoExprParse"}, {"idx", "int"}, {"value", "int"}, {"c0", "byte"}, {"c1", "byte"}, {"label", "int"}, {"jumpIfTrue", "bool"}, {"unsigned", "bool"}}, Prepared: "return false"},
	{Name: "local_bit_test_jump", Suffix: "LocalBitTestJump", Function: "renvoEmitLocalBitTestJump", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"ep", "*renvoExprParse"}, {"idx", "int"}, {"c0", "byte"}, {"label", "int"}, {"jumpIfTrue", "bool"}}, Prepared: "return false"},
	{Name: "negate_primary_word", Suffix: "NegatePrimaryWord", Function: "renvoAsmNegatePrimaryWord", Result: "", Failure: "", Parameters: []compilerBindingParameter{}, Prepared: "renvoAsmPrimaryToNegative(a)"},
	{Name: "primary_address_offset", Suffix: "PrimaryAddressOffset", Function: "renvoAsmPrimaryAddressOffset", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"offset", "int"}}, Prepared: "renvoAsmPushImm(a, offset)\nrenvoAsmPopTertiary(a)\nrenvoAsmAddPrimaryTertiary(a)"},
	{Name: "word_expression_peephole", Suffix: "WordExpressionPeephole", Function: "renvoEmitWordExpressionPeephole", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "int", Failure: "-1", Parameters: []compilerBindingParameter{{"ep", "*renvoExprParse"}, {"idx", "int"}}, Prepared: "return -1"},
	{Name: "wide_ident_to_local", Suffix: "WideIdentToLocal", Function: "renvoEmitWideIdentToLocal", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"e", "*renvoExpr"}, {"offset", "int"}}, Prepared: "return false"},
	{Name: "constant_pointer_step", Suffix: "ConstantPointerStep", Function: "renvoEmitConstantPointerStep", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"delta", "int"}}, Prepared: "return false"},
	{Name: "c_update_intrinsic", Suffix: "CUpdateIntrinsic", Function: "renvoEmitCUpdateIntrinsic", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "int", Failure: "-1", Parameters: []compilerBindingParameter{{"ep", "*renvoExprParse"}, {"idx", "int"}, {"e", "*renvoExpr"}, {"callee", "*renvoExpr"}, {"discard", "bool"}}, Prepared: "return -1"},
	{Name: "c_inline_return", Suffix: "CInlineReturn", Function: "renvoEmitCInlineReturn", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "int", Failure: "-1", Parameters: []compilerBindingParameter{{"ep", "*renvoExprParse"}, {"idx", "int"}, {"e", "*renvoExpr"}, {"callee", "*renvoExpr"}, {"label", "int"}, {"jumpIfTrue", "bool"}}, Prepared: "return -1"},
	{Name: "slice_backing_size", Suffix: "SliceBackingSize", Function: "renvoAsmSliceBackingSize", Result: "int", Failure: "0", Parameters: []compilerBindingParameter{{"elemSize", "int"}}, Prepared: "\tbackingSize := 0\n\tif renvoFixedTarget != 0 {\n\t\tcount := 4096\n\t\tif elemSize == 1 {\n\t\t\tcount = 65536\n\t\t}\n\t\tbackingSize = elemSize * count\n\t\tif backingSize \u003c 8192 {\n\t\t\tbackingSize = 8192\n\t\t}\n\t\tif backingSize \u003e 65536 {\n\t\t\tbackingSize = 65536\n\t\t}\n\t\tif backingSize \u003c elemSize {\n\t\t\tbackingSize = elemSize\n\t\t}\n\t} else {\n\t\tbackingSize = elemSize * 8192\n\t\tif backingSize \u003c 4096 {\n\t\t\tbackingSize = 4096\n\t\t}\n\t\tif backingSize \u003e 65536 {\n\t\t\tbackingSize = 65536\n\t\t}\n\t}\n\treturn backingSize"},
	{Name: "folded_field_addressing", Suffix: "FoldedFieldAddressing", Function: "renvoAsmFoldedFieldAddressing", Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return false"},
	{Name: "can_store_scalar", Suffix: "CanStoreScalar", Function: "renvoAsmCanStoreScalar", Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"size", "int"}}, Prepared: "return true"},
	{Name: "store_primary_mem_tertiary_disp", Suffix: "StorePrimaryMemTertiaryDisp", Function: "renvoAsmStorePrimaryMemTertiaryDisp", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"disp", "int"}}, Prepared: "renvoRTGDirectStoreNative(a, renvoRTGAsmAddress(renvoRTGTertiary, RTGNoRegister, disp, 1), renvoRTGPrimary)"},
	{Name: "load_tertiary_mem_secondary_disp", Suffix: "LoadTertiaryMemSecondaryDisp", Function: "renvoAsmLoadTertiaryMemSecondaryDisp", Result: "", Failure: "", Parameters: []compilerBindingParameter{{"disp", "int"}}, Prepared: "renvoRTGDirectLoadNative(a, renvoRTGTertiary, renvoRTGAsmAddress(renvoRTGSecondary, RTGNoRegister, disp, 1))"},
	{Name: "needs_function_symbols", Suffix: "NeedsFunctionSymbols", Function: "renvoAsmNeedsFunctionSymbols", Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "if renvoFixedTarget == 0 \u0026\u0026 renvoIsHostedObject(a.c) { return true }\nreturn renvoRTGPreparedFunctionSymbols != 0"},
	{Name: "assembler_reserves", Suffix: "AssemblerReserves", Function: "renvoAssemblerReserves", Result: "renvoAsmReserves", Failure: "renvoAsmReserves{}", Parameters: []compilerBindingParameter{}, Prepared: "r := renvoAsmReserves{code: 2097152, labels: 32768, relocs: 65536, absRelocs: 49152, data: 65536, kernelImports: true, openbsdSyscalls: true}\nif !a.c.stripSymbols || renvoAsmNeedsFunctionSymbols(a) {\n r.symbols = 1024\n}\nreturn r"},
	{Name: "dereference_secondary", Suffix: "DereferenceSecondary", Function: "renvoEmitDereferenceSecondary", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{}, Prepared: "renvoAsmLoadPrimaryMemSecondaryDisp(\u0026g.asm, 0)\nrenvoAsmCopyPrimaryToSecondary(\u0026g.asm)"},
	{Name: "secondary_frame_address", Suffix: "SecondaryFrameAddress", Function: "renvoEmitSecondaryFrameAddress", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"offset", "int"}}, Prepared: "renvoAsmAddressPrimaryStack(\u0026g.asm, offset)\nrenvoAsmCopyPrimaryToSecondary(\u0026g.asm)"},
	{Name: "local_word_compare_jump", Suffix: "LocalWordCompareJump", Function: "renvoEmitLocalWordCompareJump", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"left", "int"}, {"right", "int"}, {"c0", "byte"}, {"c1", "byte"}, {"label", "int"}, {"jumpIfTrue", "bool"}, {"unsigned", "bool"}, {"leftKind", "int"}, {"rightKind", "int"}, {"leftSize", "int"}, {"rightSize", "int"}}, Prepared: "return false"},
	{Name: "register_ieee_comparison", Suffix: "RegisterIEEEComparison", Function: "renvoUsesRegisterIEEEComparison", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{}, Prepared: "return false"},
	{Name: "immediate_word_comparison", Suffix: "ImmediateWordComparison", Function: "renvoCanCompareWordImmediate", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"unsigned", "bool"}}, Prepared: "return true"},
	{Name: "compare_word_operands", Suffix: "CompareWordOperands", Function: "renvoEmitCompareWordOperands", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"unsigned", "bool"}}, Prepared: "renvoRTGDirectCompare(\u0026g.asm, renvoRTGTertiary, renvoRTGPrimary)\nreturn unsigned"},
	{Name: "object_external_call", Suffix: "ObjectExternalCall", Function: "renvoEmitObjectExternalCall", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"fn", "*renvoFuncInfo"}}, Prepared: "return false"},
	{Name: "checked_index_address", Suffix: "CheckedIndexAddress", Function: "renvoEmitCheckedIndexAddress", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"elemSize", "int"}}, Prepared: "renvoAsmCallLabel(&g.asm, renvoEnsureIndexAddressHelper(g, elemSize))"},
	{Name: "increment_global_word", Suffix: "IncrementGlobalWord", Function: "renvoEmitIncrementGlobalWord", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"globalOffset", "int"}, {"inc", "bool"}}, Prepared: "a := \u0026g.asm\nrenvoAsmLoadPrimaryBss(a, globalOffset)\nrenvoAsmPushImm(a, 1)\nrenvoAsmPopTertiary(a)\nif inc {\n\trenvoAsmAddPrimaryTertiary(a)\n} else {\n\trenvoAsmSubPrimaryTertiary(a)\n}\nrenvoAsmStorePrimaryBss(a, globalOffset)\nreturn true"},
	{Name: "increment_local_word", Suffix: "IncrementLocalWord", Function: "renvoEmitIncrementLocalWord", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"localOffset", "int"}, {"inc", "bool"}}, Prepared: "a := \u0026g.asm\nrenvoAsmLoadPrimaryStack(a, localOffset)\nrenvoAsmPushImm(a, 1)\nrenvoAsmPopTertiary(a)\nif inc {\n\trenvoAsmAddPrimaryTertiary(a)\n} else {\n\trenvoAsmSubPrimaryTertiary(a)\n}\nrenvoAsmStorePrimaryStack(a, localOffset)\nreturn true"},
	{Name: "bounded_narrow_unsigned_shift", Suffix: "BoundedNarrowUnsignedShift", Function: "renvoEmitBoundedNarrowUnsignedShift", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"tok", "int"}}, Prepared: "renvoRTGEmitBoundedVariableShift(&g.asm, RTGShiftRight, false)"},
	{Name: "bounded_wide_word_shift", Suffix: "BoundedWideWordShift", Function: "renvoEmitBoundedWideWordShift", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"mode", "int"}}, Prepared: "if mode == 0 {\n\trenvoRTGEmitBoundedVariableShift(\u0026g.asm, RTGShiftLeft, false)\n} else {\n\trenvoRTGEmitBoundedVariableShift(\u0026g.asm, RTGShiftRight, mode == 1)\n}"},
	{Name: "optimized_native_binary_expr", Suffix: "OptimizedNativeBinaryExpr", Function: "renvoEmitOptimizedNativeBinaryExpr", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "int", Failure: "-1", Parameters: []compilerBindingParameter{{"ep", "*renvoExprParse"}, {"idx", "int"}}, Prepared: "return -1"},
	{Name: "thread_state_register_capability", Suffix: "ThreadStateRegisterCapability", Function: "renvoEmitThreadStateRegisterCapability", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{}, Prepared: "renvoAsmPrimaryImm(&g.asm, 0)"},
	{Name: "swap_thread_state", Suffix: "SwapThreadState", Function: "renvoEmitSwapThreadState", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{}, Prepared: "renvoAsmPushPrimary(\u0026g.asm)\nrenvoAsmLoadPrimaryBss(\u0026g.asm, g.threadStatePointerOff)\nrenvoAsmPopSecondary(\u0026g.asm)\nrenvoAsmPushPrimary(\u0026g.asm)\nrenvoAsmCopySecondaryToPrimary(\u0026g.asm)\nrenvoAsmStorePrimaryBss(\u0026g.asm, g.threadStatePointerOff)\nrenvoAsmPopPrimary(\u0026g.asm)"},
	{Name: "record_local_storage", Suffix: "RecordLocalStorage", Function: "renvoRecordLocalStorage", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"offset", "int"}, {"size", "int"}, {"captureOff", "int"}, {"typ", "int"}}, Prepared: ""},
	{Name: "wide_compare_value", Suffix: "WideCompareValue", Function: "renvoEmitWideCompareValue", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"left", "int"}, {"right", "int"}, {"tok", "int"}, {"signed", "bool"}}, Prepared: "return renvoEmitNativeWideStack(g, 0, left, right, renvoWideBinaryMode(g, tok, signed))"},
	{Name: "wide_binary_value", Suffix: "WideBinaryValue", Function: "renvoEmitWideBinaryValue", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"dest", "int"}, {"left", "int"}, {"right", "int"}, {"tok", "int"}, {"signed", "bool"}}, Prepared: "return renvoEmitNativeWideStack(g, dest, left, right, renvoWideBinaryMode(g, tok, signed))"},
	{Name: "stack_ieee_float", Suffix: "StackIEEEFloat", Function: "renvoUsesStackIEEEFloat", Result: "bool", Failure: "false", Prepared: "return renvoRTGPreparedIEEEFloat != 0"},
	{Name: "scaled_float", Suffix: "ScaledFloat", Function: "renvoUsesScaledFloat", Result: "bool", Failure: "false", Prepared: "return renvoRTGPreparedIEEEFloat == 0"},
	{Name: "direct_float64_operands", Suffix: "DirectFloat64Operands", Function: "renvoUsesDirectFloat64Operands", Result: "bool", Failure: "false", Prepared: "return renvoRTGPreparedIEEEFloat != 0"},
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
	{Name: "wide_less_stack", Suffix: "WideLessStack", Function: "renvoEmitWideLessStack", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"left", "int"}, {"right", "int"}, {"signed", "bool"}}, Prepared: "\tif renvoFixedTarget != 0 {\n\t\treturn\n\t}\n\trenvoNonNil(g)\n\thighEqual := renvoAsmNewLabel(\u0026g.asm)\n\tdone := renvoAsmNewLabel(\u0026g.asm)\n\trenvoEmitNativeCompareStack(g, left-g.c.renvoNativeIntSize, right-g.c.renvoNativeIntSize, renvoConditionEqual)\n\trenvoAsmJnzPrimary(\u0026g.asm, highEqual)\n\tif signed {\n\t\trenvoEmitNativeCompareStack(g, left-g.c.renvoNativeIntSize, right-g.c.renvoNativeIntSize, renvoConditionSignedLess)\n\t} else {\n\t\trenvoEmitNativeUnsignedLessStack(g, left-g.c.renvoNativeIntSize, right-g.c.renvoNativeIntSize)\n\t}\n\trenvoAsmJmpMarkLabel(\u0026g.asm, done, highEqual)\n\trenvoEmitNativeUnsignedLessStack(g, left, right)\n\trenvoAsmMarkLabel(\u0026g.asm, done)"},
	{Name: "wide_add_stack", Suffix: "WideAddStack", Function: "renvoEmitWideAddStack", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"dest", "int"}, {"left", "int"}, {"right", "int"}}, Prepared: "\tif renvoFixedTarget != 0 {\n\t\treturn\n\t}\n\trenvoNonNil(g)\n\tcarry := renvoAddUnnamedLocal(g, renvoTypeInt)\n\tleftLow := renvoAddUnnamedLocal(g, renvoTypeInt)\n\trenvoAsmCopyStackSlot(\u0026g.asm, left, leftLow)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, left)\n\trenvoAsmLoadTertiaryStack(\u0026g.asm, right)\n\trenvoAsmAddPrimaryTertiary(\u0026g.asm)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, dest)\n\trenvoEmitNativeUnsignedLessStack(g, dest, leftLow)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, carry)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, left-g.c.renvoNativeIntSize)\n\trenvoAsmLoadTertiaryStack(\u0026g.asm, right-g.c.renvoNativeIntSize)\n\trenvoAsmAddPrimaryTertiary(\u0026g.asm)\n\trenvoAsmLoadTertiaryStack(\u0026g.asm, carry)\n\trenvoAsmAddPrimaryTertiary(\u0026g.asm)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, dest-g.c.renvoNativeIntSize)"},
	{Name: "wide_sub_stack", Suffix: "WideSubStack", Function: "renvoEmitWideSubStack", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"dest", "int"}, {"left", "int"}, {"right", "int"}}, Prepared: "\tif renvoFixedTarget != 0 {\n\t\treturn\n\t}\n\trenvoNonNil(g)\n\tborrow := renvoAddUnnamedLocal(g, renvoTypeInt)\n\trenvoEmitNativeUnsignedLessStack(g, left, right)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, borrow)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, left)\n\trenvoAsmLoadTertiaryStack(\u0026g.asm, right)\n\trenvoAsmSubPrimaryTertiary(\u0026g.asm)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, dest)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, left-g.c.renvoNativeIntSize)\n\trenvoAsmLoadTertiaryStack(\u0026g.asm, right-g.c.renvoNativeIntSize)\n\trenvoAsmSubPrimaryTertiary(\u0026g.asm)\n\trenvoAsmLoadTertiaryStack(\u0026g.asm, borrow)\n\trenvoAsmSubPrimaryTertiary(\u0026g.asm)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, dest-g.c.renvoNativeIntSize)"},
	{Name: "wide_shift_left_one", Suffix: "WideShiftLeftOne", Function: "renvoEmitWideShiftLeftOne", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"value", "int"}}, Prepared: "\tif renvoFixedTarget != 0 {\n\t\treturn\n\t}\n\trenvoNonNil(g)\n\tcarry := renvoAddUnnamedLocal(g, renvoTypeInt)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, value)\n\trenvoAsmShrPrimaryImm(\u0026g.asm, 31)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, carry)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, value-g.c.renvoNativeIntSize)\n\trenvoAsmShlPrimaryImm(\u0026g.asm, 1)\n\trenvoAsmLoadTertiaryStack(\u0026g.asm, carry)\n\trenvoAsmAddPrimaryTertiary(\u0026g.asm)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, value-g.c.renvoNativeIntSize)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, value)\n\trenvoAsmShlPrimaryImm(\u0026g.asm, 1)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, value)"},
	{Name: "wide_shift_right_one", Suffix: "WideShiftRightOne", Function: "renvoEmitWideShiftRightOne", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"value", "int"}, {"signed", "bool"}}, Prepared: "\tif renvoFixedTarget != 0 {\n\t\treturn\n\t}\n\trenvoNonNil(g)\n\tcarry := renvoAddUnnamedLocal(g, renvoTypeInt)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, value-g.c.renvoNativeIntSize)\n\trenvoAsmShlPrimaryImm(\u0026g.asm, 31)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, carry)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, value)\n\trenvoAsmShrPrimaryImm(\u0026g.asm, 1)\n\trenvoAsmLoadTertiaryStack(\u0026g.asm, carry)\n\trenvoAsmAddPrimaryTertiary(\u0026g.asm)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, value)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, value-g.c.renvoNativeIntSize)\n\tif signed {\n\t\trenvoAsmSarPrimaryImm(\u0026g.asm, 1)\n\t} else {\n\t\trenvoAsmShrPrimaryImm(\u0026g.asm, 1)\n\t}\n\trenvoAsmStorePrimaryStack(\u0026g.asm, value-g.c.renvoNativeIntSize)"},
	{Name: "wide_shift_stack", Suffix: "WideShiftStack", Function: "renvoEmitWideShiftStack", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"dest", "int"}, {"left", "int"}, {"count", "int"}, {"right", "bool"}, {"signed", "bool"}}, Prepared: "\tif renvoFixedTarget != 0 {\n\t\treturn\n\t}\n\trenvoNonNil(g)\n\trenvoEmitCopyStackToStack(g, left, dest, renvoBackendValueSlotSize)\n\tcounter := renvoAddUnnamedLocal(g, renvoTypeInt)\n\tlimit := renvoAddUnnamedLocal(g, renvoTypeInt)\n\trenvoAsmStoreStackImm(\u0026g.asm, limit, 64)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, count-g.c.renvoNativeIntSize)\n\tclamp := renvoAsmNewLabel(\u0026g.asm)\n\tready := renvoAsmNewLabel(\u0026g.asm)\n\tbegin := renvoAsmNewLabel(\u0026g.asm)\n\trenvoAsmJnzPrimary(\u0026g.asm, clamp)\n\trenvoEmitNativeUnsignedLessStack(g, count, limit)\n\trenvoAsmJnzPrimary(\u0026g.asm, ready)\n\trenvoAsmMarkLabel(\u0026g.asm, clamp)\n\trenvoAsmStoreStackImm(\u0026g.asm, counter, 64)\n\trenvoAsmJmpLabel(\u0026g.asm, begin)\n\trenvoAsmMarkLabel(\u0026g.asm, ready)\n\trenvoAsmCopyStackSlot(\u0026g.asm, count, counter)\n\trenvoAsmMarkLabel(\u0026g.asm, begin)\n\tdone := renvoAsmNewLabel(\u0026g.asm)\n\tloop := renvoAsmNewLabel(\u0026g.asm)\n\trenvoAsmMarkLabel(\u0026g.asm, loop)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, counter)\n\trenvoAsmJzPrimary(\u0026g.asm, done)\n\tif right {\n\t\trenvoEmitWideShiftRightOne(g, dest, signed)\n\t} else {\n\t\trenvoEmitWideShiftLeftOne(g, dest)\n\t}\n\trenvoAsmDecStack(\u0026g.asm, counter)\n\trenvoAsmJmpMarkLabel(\u0026g.asm, loop, done)"},
	{Name: "wide_multiply_stack", Suffix: "WideMultiplyStack", Function: "renvoEmitWideMulStack", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"dest", "int"}, {"left", "int"}, {"right", "int"}}, Prepared: "\tif renvoFixedTarget != 0 {\n\t\treturn\n\t}\n\trenvoNonNil(g)\n\tmultiplicand := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)\n\tmultiplier := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)\n\trenvoEmitCopyStackToStack(g, left, multiplicand, renvoBackendValueSlotSize)\n\trenvoEmitCopyStackToStack(g, right, multiplier, renvoBackendValueSlotSize)\n\trenvoZeroLocalAtOffset(g, dest)\n\tcounter := renvoAddUnnamedLocal(g, renvoTypeInt)\n\trenvoAsmStoreStackImm(\u0026g.asm, counter, 64)\n\tloop := renvoAsmNewLabel(\u0026g.asm)\n\tskipAdd := renvoAsmNewLabel(\u0026g.asm)\n\tdone := renvoAsmNewLabel(\u0026g.asm)\n\trenvoAsmMarkLabel(\u0026g.asm, loop)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, counter)\n\trenvoAsmJzPrimary(\u0026g.asm, done)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, multiplier)\n\trenvoAsmShlPrimaryImm(\u0026g.asm, 31)\n\trenvoAsmJzPrimary(\u0026g.asm, skipAdd)\n\trenvoEmitWideAddStack(g, dest, dest, multiplicand)\n\trenvoAsmMarkLabel(\u0026g.asm, skipAdd)\n\trenvoEmitWideShiftLeftOne(g, multiplicand)\n\trenvoEmitWideShiftRightOne(g, multiplier, false)\n\trenvoAsmDecStack(\u0026g.asm, counter)\n\trenvoAsmJmpMarkLabel(\u0026g.asm, loop, done)"},
	{Name: "wide_negate_stack", Suffix: "WideNegateStack", Function: "renvoEmitWideNegateInPlace", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"value", "int"}}, Prepared: "\tif renvoFixedTarget != 0 {\n\t\treturn\n\t}\n\trenvoNonNil(g)\n\tzero := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)\n\tresult := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)\n\trenvoZeroLocalAtOffset(g, zero)\n\trenvoEmitWideSubStack(g, result, zero, value)\n\trenvoEmitCopyStackToStack(g, result, value, renvoBackendValueSlotSize)"},
	{Name: "wide_unsigned_divide_stack", Suffix: "WideUnsignedDivideStack", Function: "renvoEmitWideUnsignedDivStack", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"quotient", "int"}, {"remainder", "int"}, {"dividendValue", "int"}, {"divisor", "int"}}, Prepared: "\tif renvoFixedTarget != 0 {\n\t\treturn\n\t}\n\trenvoNonNil(g)\n\tdividend := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)\n\trenvoEmitCopyStackToStack(g, dividendValue, dividend, renvoBackendValueSlotSize)\n\trenvoZeroLocalAtOffset(g, quotient)\n\trenvoZeroLocalAtOffset(g, remainder)\n\tcounter := renvoAddUnnamedLocal(g, renvoTypeInt)\n\tbit := renvoAddUnnamedLocal(g, renvoTypeInt)\n\trenvoAsmStoreStackImm(\u0026g.asm, counter, 64)\n\tloop := renvoAsmNewLabel(\u0026g.asm)\n\tskipSubtract := renvoAsmNewLabel(\u0026g.asm)\n\tdone := renvoAsmNewLabel(\u0026g.asm)\n\trenvoAsmMarkLabel(\u0026g.asm, loop)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, counter)\n\trenvoAsmJzPrimary(\u0026g.asm, done)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, dividend-g.c.renvoNativeIntSize)\n\trenvoAsmShrPrimaryImm(\u0026g.asm, 31)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, bit)\n\trenvoEmitWideShiftLeftOne(g, remainder)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, remainder)\n\trenvoAsmLoadTertiaryStack(\u0026g.asm, bit)\n\trenvoAsmAddPrimaryTertiary(\u0026g.asm)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, remainder)\n\trenvoEmitWideShiftLeftOne(g, dividend)\n\trenvoEmitWideShiftLeftOne(g, quotient)\n\trenvoEmitWideLessStack(g, remainder, divisor, false)\n\trenvoAsmJnzPrimary(\u0026g.asm, skipSubtract)\n\trenvoEmitWideSubStack(g, remainder, remainder, divisor)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, quotient)\n\trenvoAsmIncPrimary(\u0026g.asm)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, quotient)\n\trenvoAsmMarkLabel(\u0026g.asm, skipSubtract)\n\trenvoAsmDecStack(\u0026g.asm, counter)\n\trenvoAsmJmpMarkLabel(\u0026g.asm, loop, done)"},
	{Name: "unsigned_less_stack", Suffix: "UnsignedLessStack", Function: "renvoEmitNativeUnsignedLessStack", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"left", "int"}, {"right", "int"}}, Prepared: "\tif renvoFixedTarget != 0 {\n\t\treturn\n\t}\n\trenvoNonNil(g)\n\t// When the sign bits differ, the word with its sign bit clear is smaller in\n\t// unsigned order. When they match, signed subtraction cannot overflow, so\n\t// the ordinary comparison is safe even on wasm's flag emulation.\n\tzero := renvoAddUnnamedLocal(g, renvoTypeInt)\n\tleftNegative := renvoAddUnnamedLocal(g, renvoTypeInt)\n\trightNegative := renvoAddUnnamedLocal(g, renvoTypeInt)\n\trenvoAsmStoreStackImm(\u0026g.asm, zero, 0)\n\trenvoEmitNativeCompareStack(g, left, zero, renvoConditionSignedLess)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, leftNegative)\n\trenvoEmitNativeCompareStack(g, right, zero, renvoConditionSignedLess)\n\trenvoAsmStorePrimaryStack(\u0026g.asm, rightNegative)\n\tsameSign := renvoAsmNewLabel(\u0026g.asm)\n\tdone := renvoAsmNewLabel(\u0026g.asm)\n\trenvoEmitNativeCompareStack(g, leftNegative, rightNegative, renvoConditionEqual)\n\trenvoAsmJnzPrimary(\u0026g.asm, sameSign)\n\trenvoAsmLoadPrimaryStack(\u0026g.asm, rightNegative)\n\trenvoAsmJmpMarkLabel(\u0026g.asm, done, sameSign)\n\trenvoEmitNativeCompareStack(g, left, right, renvoConditionSignedLess)\n\trenvoAsmMarkLabel(\u0026g.asm, done)"},
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
	{Name: "install_thread_state", Suffix: "InstallThreadState", Function: "renvoEmitInstallThreadState", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{}, Prepared: "return"},
	{Name: "load_primary_thread_state", Suffix: "LoadPrimaryThreadState", Function: "renvoAsmLoadPrimaryThreadState", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"stateOffset", "int"}}, Prepared: "\trenvoAsmPushSecondary(\u0026g.asm)\n\trenvoAsmLoadPrimaryBss(\u0026g.asm, g.threadStatePointerOff)\n\trenvoAsmCopyPrimaryToSecondary(\u0026g.asm)\n\trenvoAsmLoadPrimaryMemSecondaryDisp(\u0026g.asm, stateOffset)\n\trenvoAsmPopSecondary(\u0026g.asm)"},
	{Name: "store_primary_thread_state", Suffix: "StorePrimaryThreadState", Function: "renvoAsmStorePrimaryThreadState", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"stateOffset", "int"}}, Prepared: "\trenvoAsmPushSecondary(\u0026g.asm)\n\trenvoAsmPushPrimary(\u0026g.asm)\n\trenvoAsmLoadPrimaryBss(\u0026g.asm, g.threadStatePointerOff)\n\trenvoAsmCopyPrimaryToSecondary(\u0026g.asm)\n\trenvoAsmPopPrimary(\u0026g.asm)\n\trenvoAsmStorePrimaryMemSecondaryDisp(\u0026g.asm, stateOffset)\n\trenvoAsmPopSecondary(\u0026g.asm)"},
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
	{Name: "emit_byte_slice_conversion_regs", Suffix: "EmitByteSliceConversionRegs", Function: "renvoEmitByteSliceConversionRegs", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"ep", "*renvoExprParse"}, {"idx", "int"}}, Prepared: "renvoNonNil(g, ep)\na := &g.asm\ne := &ep.exprs[idx]\nif e.argCount != 1 {\n\treturn false\n}\nif renvoPreparedBackendActive == 0 && (g.c.renvoTargetArch == renvoArchAmd64 || g.c.renvoTargetArch == renvoArch386 && !g.c.code16 || g.c.renvoTargetArch == renvoArchWasm32 || g.c.renvoTargetArch == renvoArchArm || g.c.renvoTargetArch == renvoArchAarch64) {\n\tlabel := 0\n\tif g.c.renvoTargetArch == renvoArchWasm32 {\n\t\tlabel = renvoEnsureStringCopyWasm32(g)\n\t} else if g.c.renvoTargetArch == renvoArchArm || g.c.renvoTargetArch == renvoArchAarch64 {\n\t\tlabel = renvoEnsureStringStorageArm(g, false)\n\t} else {\n\t\tlabel = renvoEnsureStringCopyX86(g)\n\t}\n\tif !renvoEmitStringValueRegs(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)) {\n\t\treturn false\n\t}\n\trenvoAsmCopySecondaryToTertiary(a)\n\trenvoAsmCallLabel(a, label)\n\trenvoEmitArenaAllocationCheck(g)\n\trenvoAsmCopySecondaryToTertiary(a)\n\treturn true\n}\nsrcOff := renvoAddUnnamedLocal(g, renvoTypeInt)\nlenOff := renvoAddUnnamedLocal(g, renvoTypeInt)\ndestOff := renvoAddUnnamedLocal(g, renvoTypeInt)\nargIndex := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)\nif !renvoEmitStringValueRegs(g, ep, argIndex) {\n\treturn false\n}\nrenvoAsmStorePrimarySecondaryStack(a, srcOff, lenOff)\nrenvoEmitArenaAllocStackPrimary(g, lenOff)\nrenvoAsmStorePrimaryStack(a, destOff)\n// The destination is fresh arena storage, so a bulk copy preserves the\n// conversion's ownership while avoiding a scalar loop at every call site.\nrenvoEmitCopyBytes(g, srcOff, destOff, lenOff)\nrenvoAsmLoadPrimarySecondaryStack(a, destOff, lenOff)\nrenvoAsmCopySecondaryToTertiary(a)\nreturn true"},
	{Name: "emit_byte_slice_string_copy_value_regs", Suffix: "EmitByteSliceStringCopyValueRegs", Function: "renvoEmitByteSliceStringCopyValueRegs", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"ep", "*renvoExprParse"}, {"argIndex", "int"}}, Prepared: "renvoNonNil(g, ep)\na := &g.asm\nif !renvoEmitSlicePtrLen(g, ep, argIndex) {\n\treturn false\n}\nif renvoPreparedBackendActive == 0 && (g.c.renvoTargetArch == renvoArchAmd64 || g.c.renvoTargetArch == renvoArch386 && !g.c.code16 || g.c.renvoTargetArch == renvoArchWasm32 || g.c.renvoTargetArch == renvoArchArm || g.c.renvoTargetArch == renvoArchAarch64) {\n\tlabel := 0\n\tif g.c.renvoTargetArch == renvoArchWasm32 {\n\t\tlabel = renvoEnsureStringCopyWasm32(g)\n\t} else if g.c.renvoTargetArch == renvoArchArm || g.c.renvoTargetArch == renvoArchAarch64 {\n\t\tlabel = renvoEnsureStringStorageArm(g, false)\n\t} else {\n\t\tlabel = renvoEnsureStringCopyX86(g)\n\t}\n\trenvoAsmCallLabel(a, label)\n\trenvoEmitArenaAllocationCheck(g)\n\treturn true\n}\nsrcOff := renvoAddUnnamedLocal(g, renvoTypeInt)\nlenOff := renvoAddUnnamedLocal(g, renvoTypeInt)\ndestOff := renvoAddUnnamedLocal(g, renvoTypeInt)\nrenvoAsmStorePrimaryStack(a, srcOff)\nrenvoAsmCopyTertiaryToPrimary(a)\nrenvoAsmStorePrimaryStack(a, lenOff)\nrenvoEmitArenaAllocStackPrimary(g, lenOff)\nrenvoAsmStorePrimaryStack(a, destOff)\nrenvoEmitCopyToFreshArena(g, srcOff, destOff, lenOff)\nrenvoAsmLoadPrimarySecondaryStack(a, destOff, lenOff)\nreturn true"},
	{Name: "emit_copy_native", Suffix: "EmitCopyNative", Function: "renvoEmitCopyNative", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"srcOffset", "int"}, {"destOffset", "int"}, {"size", "int"}, {"mode", "int"}}, Prepared: "renvoNonNil(g)\nif renvoPreparedBackendActive == 0 && g.c.renvoTargetArch == renvoArchWasm32 && g.c.renvoTarget != renvoTargetVM32 && size >= 16 && (mode == renvoNativeCopyStackToMem || mode == renvoNativeCopyMemToStack) {\n\trenvoAsmEmit8(&g.asm, renvoWasm32OpCopyFrameBlock)\n\trenvoAsmEmit8(&g.asm, mode)\n\trenvoAsmEmit32(&g.asm, srcOffset)\n\trenvoAsmEmit32(&g.asm, destOffset)\n\trenvoAsmEmit32(&g.asm, size)\n\tg.asm.lastPrimaryLoad = 0\n\treturn\n}\nif renvoPreparedBackendActive == 0 && g.c.renvoTarget == renvoTargetVM32 && size >= 32 {\n\trenvoVM32CopyFixed(g, srcOffset, destOffset, size, mode)\n\treturn\n}\nif renvoPreparedBackendActive == 0 && g.c.renvoTargetArch == renvoArch386 && !g.c.code16 && size >= 16 {\n\trenvo386CopyFixed(g, srcOffset, destOffset, size, mode)\n\treturn\n}\nif renvoPreparedBackendActive == 0 && g.c.renvoTargetArch == renvoArchAmd64 && size >= 16 {\n\trenvoAmd64CopyFixed(g, srcOffset, destOffset, size, mode)\n\treturn\n}\nif renvoPreparedBackendActive == 0 && g.c.renvoTargetArch == renvoArchArm && size >= 16 {\n\trenvoArmCopyFixed(g, srcOffset, destOffset, size, mode)\n\treturn\n}\nif renvoPreparedBackendActive == 0 && g.c.renvoTargetArch == renvoArchAarch64 && size >= 16 {\n\trenvoAarch64CopyFixed(g, srcOffset, destOffset, size, mode)\n\treturn\n}\n// Large aggregate loads and stores use the existing overlap-safe copy\n// operation instead of expanding a load/store pair for every word.\nif renvoPreferBulkIndirectCopy(g, size) &&\n\t(mode == renvoNativeCopyMemToStack || mode == renvoNativeCopyStackToMem) {\n\tsource := renvoAddUnnamedLocal(g, renvoTypeInt)\n\tdestination := renvoAddUnnamedLocal(g, renvoTypeInt)\n\tcount := renvoAddUnnamedLocal(g, renvoTypeInt)\n\tif mode == renvoNativeCopyMemToStack {\n\t\trenvoAsmStoreSecondaryStack(&g.asm, source)\n\t\trenvoAsmAddressPrimaryStack(&g.asm, destOffset)\n\t\trenvoAsmStorePrimaryStack(&g.asm, destination)\n\t} else {\n\t\trenvoAsmAddSecondaryImm(&g.asm, destOffset)\n\t\trenvoAsmStoreSecondaryStack(&g.asm, destination)\n\t\trenvoAsmAddressPrimaryStack(&g.asm, srcOffset)\n\t\trenvoAsmStorePrimaryStack(&g.asm, source)\n\t}\n\trenvoAsmStoreStackImm(&g.asm, count, size)\n\trenvoEmitCopyBytes(g, source, destination, count)\n\treturn\n}\na := &g.asm\nfor at := 0; at < size; {\n\tchunkSize := g.c.renvoNativeIntSize\n\tif size-at < chunkSize {\n\t\tchunkSize = size - at\n\t}\n\t// Scalar load/store emitters accept power-of-two widths. Split an\n\t// aggregate tail such as a three-byte array into 2+1 bytes instead of\n\t// accidentally selecting the native-width fallback and overwriting the\n\t// object immediately following it.\n\tif chunkSize > 4 && chunkSize < 8 {\n\t\tchunkSize = 4\n\t} else if chunkSize == 3 {\n\t\tchunkSize = 2\n\t}\n\tif mode == renvoNativeCopyMemToStack {\n\t\trenvoAsmLoadPrimaryMemSecondaryDispSize(a, at, chunkSize)\n\t} else if mode == renvoNativeCopyBSSToStack {\n\t\trenvoAsmLoadPrimaryBss(a, srcOffset+at)\n\t} else {\n\t\trenvoAsmLoadPrimaryStack(a, srcOffset-at)\n\t}\n\tif mode == renvoNativeCopyStackToMem {\n\t\trenvoAsmStorePrimaryMemSecondaryDispSize(a, destOffset+at, chunkSize)\n\t} else if mode == renvoNativeCopyStackToBSS {\n\t\trenvoAsmStorePrimaryBss(a, destOffset+at)\n\t} else {\n\t\t// A narrow frame store uses secondary to address its destination.\n\t\t// Keep the source base across that store when a split tail still\n\t\t// needs another memory load (for example, a three-byte RGB value).\n\t\tpreserveSource := mode == renvoNativeCopyMemToStack && chunkSize < g.c.renvoNativeIntSize && at+chunkSize < size\n\t\tif preserveSource {\n\t\t\trenvoAsmPushSecondary(a)\n\t\t}\n\t\trenvoAsmStorePrimaryStackSize(a, destOffset-at, chunkSize)\n\t\tif preserveSource {\n\t\t\trenvoAsmPopSecondary(a)\n\t\t}\n\t}\n\tat += chunkSize\n}"},
	{Name: "emit_direct_selector_words", Suffix: "EmitDirectSelectorWords", Function: "renvoEmitDirectSelectorWords", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"ep", "*renvoExprParse"}, {"idx", "int"}, {"primaryDisp", "int"}, {"tertiaryDisp", "int"}, {"size", "int"}}, Prepared: "renvoNonNil(g, ep)\nif !renvoAsmFoldedFieldAddressing(&g.asm) {\n\treturn false\n}\ne := &ep.exprs[idx]\nif e.kind != renvoExprSelector {\n\treturn false\n}\nbase := &ep.exprs[e.left]\nif base.kind != renvoExprIdent {\n\treturn false\n}\nbaseType := renvoInferParsedExprType(g, ep, e.left)\nif !renvoLoadStructFieldPath(g, baseType, e.nameStart, e.nameEnd) || g.fieldPointerIndex >= 0 {\n\treturn false\n}\nfieldOffset := g.fieldOffset\nlocalIndex := renvoFindLocalIndex(g, base.nameStart, base.nameEnd)\nif localIndex < 0 || g.c.renvoTargetArch != renvoArchAmd64 && g.locals[localIndex].captureOff != 0 ||\n\trenvoResolveType(g.meta, g.locals[localIndex].typ).kind != renvoTypePointer {\n\treturn false\n}\nrenvoAsmLoadSecondaryStack(&g.asm, g.locals[localIndex].offset)\nneedCheck := renvoRuntimeNonNilLocalNeeded(g, localIndex)\nprimaryOffset := fieldOffset + primaryDisp\ntertiaryOffset := -1\nif tertiaryDisp >= 0 {\n\ttertiaryOffset = fieldOffset + tertiaryDisp\n}\nif needCheck {\n\trenvoEmitRuntimeNonNilSecondary(g)\n}\nrenvoAsmLoadPrimaryMemSecondaryDispSize(&g.asm, primaryOffset, size)\nif tertiaryOffset >= 0 {\n\trenvoAsmLoadTertiaryMemSecondaryDisp(&g.asm, tertiaryOffset)\n}\nreturn true"},
	{Name: "emit_index_expr", Suffix: "EmitIndexExpr", Function: "renvoEmitIndexExpr", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"ep", "*renvoExprParse"}, {"idx", "int"}}, Prepared: "renvoNonNil(g, ep)\nmeta := g.meta\na := &g.asm\ne := &ep.exprs[idx]\nbaseResolved := renvoResolveType(meta, renvoInferParsedExprType(g, ep, e.left))\nrenvoNonNil(baseResolved)\nif baseResolved.kind == renvoTypePointer {\n\tbaseResolved = renvoResolveType(meta, baseResolved.elem)\n}\nif baseResolved.kind == renvoTypeString {\n\tif !renvoEmitStringValueRegs(g, ep, e.left) {\n\t\treturn false\n\t}\n\tif renvoPreparedBackendActive == 0 && g.c.renvoTargetArch == renvoArchAmd64 && !g.meta.panicEnabled {\n\t\t// Put the descriptor length in the frame-index evaluator's input register.\n\t\trenvoAsmEmitText(a, \"\\x48\\x89\\xd1\") // MOV RCX, RDX\n\t\tif renvoEmitFrameIndexTertiary(g, ep, e.right, 0) {\n\t\t\tfault := renvoEnsureUncaughtFaultHelper(g, false)\n\t\t\trenvoAsmEmitText(a, \"\\x48\\x39\\xd1\") // CMP RCX, RDX\n\t\t\trenvoAmd64AsmJccLabel(a, 0x83, fault)\n\t\t\trenvoAsmLoadBytePrimaryIndexTertiary(a)\n\t\t\treturn true\n\t\t}\n\t}\n\trenvoAsmPushPrimary(a)\n\trenvoAsmPushSecondary(a)\n\tif !renvoEmitIntExpr(g, ep, e.right) {\n\t\treturn false\n\t}\n\trenvoAsmPopTertiary(a)\n\tif renvoPreparedBackendActive == 0 && g.c.renvoTargetArch == renvoArchArm && !g.meta.panicEnabled {\n\t\t// The length is nonnegative. One unsigned comparison rejects a\n\t\t// negative or excessive index without a successful helper return.\n\t\tfault := renvoEnsureUncaughtFaultHelper(g, false)\n\t\trenvoArmAsmCmpRegReg(a, 0, 2)\n\t\trenvoArmAsmBCondLabel(a, fault, 2) // CS\n\t\trenvoArmAsmMovRegReg(a, 2, 0)\n\t} else {\n\t\trenvoEmitRuntimeBoundsCheck(g)\n\t\trenvoAsmCopySecondaryToTertiary(a)\n\t}\n\trenvoAsmPopPrimary(a)\n\trenvoAsmLoadBytePrimaryIndexTertiary(a)\n\treturn true\n}\nif baseResolved.kind == renvoTypeArray || baseResolved.kind == renvoTypeSlice {\n\telem := renvoResolveType(meta, baseResolved.elem)\n\trenvoNonNil(elem)\n\tif !renvoTypeKindIsScalarValue(elem.kind) && elem.kind != renvoTypePointer && elem.kind != renvoTypeFunc {\n\t\treturn false\n\t}\n\tif !renvoEmitIndexAddressPrimary(g, ep, idx) {\n\t\treturn false\n\t}\n\trenvoAsmCopyPrimaryToSecondary(a)\n\trenvoAsmLoadPrimaryMemSecondaryDispSize(a, 0, renvoNativeScalarStorageSize(meta, baseResolved.elem))\n\t// Size-only loads sign-extend halfwords. Restore the parsed element's\n\t// signedness before its value reaches comparisons or wider arithmetic.\n\trenvoAsmNormalizePrimaryForKind(a, elem.kind)\n\treturn true\n}\nreturn false"},
	{Name: "emit_index_primary", Suffix: "EmitIndexPrimary", Function: "renvoEmitIndexPrimary", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"ep", "*renvoExprParse"}, {"indexIdx", "int"}, {"loadKind", "int"}}, Prepared: "renvoNonNil(g, ep)\nmeta := g.meta\nrenvoNonNil(meta)\na := &g.asm\nindexExpr := &ep.exprs[indexIdx]\nsliceType := renvoResolveType(meta, renvoInferParsedExprType(g, ep, indexExpr.left))\nrenvoNonNil(sliceType)\nif sliceType.kind == renvoTypePointer {\n\telem := renvoResolveType(meta, sliceType.elem)\n\tif elem.kind != renvoTypeArray && elem.kind != renvoTypeSlice {\n\t\tif !renvoEmitCPointerIndexAddressPrimary(g, ep, indexIdx, sliceType) {\n\t\t\treturn false\n\t\t}\n\t\trenvoEmitIndexLoadFromAddress(g, loadKind)\n\t\treturn true\n\t}\n}\npointerArray := sliceType.kind == renvoTypePointer\nif pointerArray {\n\tsliceType = renvoResolveType(meta, sliceType.elem)\n}\nif sliceType.kind != renvoTypeArray && sliceType.kind != renvoTypeSlice {\n\treturn false\n}\nelemSize := renvoTypeSize(meta, sliceType.elem)\ncheckedSize := elemSize == 1 || elemSize == 8 || elemSize == 72 || elemSize == 4 &&\n\trenvoPreparedBackendActive == 0 && (g.c.renvoTargetArch == renvoArchArm || g.c.renvoTargetArch == renvoArchAarch64 ||\n\tg.c.renvoTargetArch == renvoArch386 && !g.c.code16 || g.c.renvoTarget == renvoTargetVM32)\nbaseExpr := &ep.exprs[indexExpr.left]\nif pointerArray {\n\tif !renvoEmitIntExpr(g, ep, indexExpr.left) {\n\t\treturn false\n\t}\n\trenvoEmitRuntimeNonNilPrimary(g)\n\trenvoAsmPushPrimary(a)\n\trenvoAsmPrimaryArrayLength(a, sliceType.arrayLength)\n\trenvoAsmPopPrimaryToTertiary(a)\n} else if sliceType.kind == renvoTypeArray {\n\tbase := baseExpr\n\tif base.kind == renvoExprIdent {\n\t\tlocalIndex := renvoFindLocalIndex(g, base.nameStart, base.nameEnd)\n\t\tif localIndex < 0 {\n\t\t\tglobalOffset := renvoFindGlobalOffset(g, base.nameStart, base.nameEnd)\n\t\t\tglobalType := renvoResolveType(g.meta, renvoFindGlobalType(g, base.nameStart, base.nameEnd))\n\t\t\trenvoNonNil(globalType)\n\t\t\tif globalOffset < 0 || globalType.kind != renvoTypeArray {\n\t\t\t\treturn false\n\t\t\t}\n\t\t\trenvoAsmPrimaryBssAddr(a, globalOffset)\n\t\t} else {\n\t\t\trenvoAsmAddressPrimaryStack(a, g.locals[localIndex].offset)\n\t\t}\n\t} else if base.kind == renvoExprIndex {\n\t\tif !renvoEmitIndexAddressPrimary(g, ep, indexExpr.left) {\n\t\t\treturn false\n\t\t}\n\t} else if base.kind == renvoExprSelector {\n\t\tif !renvoEmitSelectorAddressSecondary(g, ep, indexExpr.left) {\n\t\t\treturn false\n\t\t}\n\t\trenvoAsmCopySecondaryToPrimary(a)\n\t} else if base.kind == renvoExprUnary && renvoTokCharIs(g.prog, base.tok, '*') {\n\t\tif !renvoEmitAddressPrimary(g, ep, indexExpr.left) {\n\t\t\treturn false\n\t\t}\n\t} else if base.kind == renvoExprCall || base.kind == renvoExprComposite {\n\t\tbaseType := renvoInferParsedExprType(g, ep, indexExpr.left)\n\t\ttempOffset := renvoAddUnnamedLocal(g, baseType)\n\t\tif !renvoEmitTypedAssign(g, ep, indexExpr.left, tempOffset) {\n\t\t\treturn false\n\t\t}\n\t\trenvoAsmAddressPrimaryStack(a, tempOffset)\n\t} else {\n\t\treturn false\n\t}\n\trenvoAsmPushPrimary(a)\n\trenvoAsmPrimaryArrayLength(a, sliceType.arrayLength)\n\trenvoAsmPopPrimaryToTertiary(a)\n} else {\n\tif !renvoEmitSlicePtrLen(g, ep, indexExpr.left) {\n\t\treturn false\n\t}\n}\n// A frame-local index has no evaluation side effects. Load it directly\n// after the base, keeping pointer, length and index in registers instead\n// of spilling two operands to the expression stack. Bounds checks remain.\nif renvoPreparedBackendActive == 0 && g.c.renvoTargetArch == renvoArchAmd64 &&\n\t!g.meta.panicEnabled && elemSize >= 0 && elemSize <= 2147483647 &&\n\t(ep.exprs[indexExpr.right].kind == renvoExprBinary || ep.exprs[indexExpr.right].kind == renvoExprInt) &&\n\trenvoEmitFrameIndexTertiary(g, ep, indexExpr.right, 0) {\n\tfault := renvoEnsureUncaughtFaultHelper(g, false)\n\trenvoAsmEmitText(a, \"\\x48\\x39\\xd1\") // CMP RCX, RDX\n\trenvoAmd64AsmJccLabel(a, 0x83, fault)\n\trenvoEmitScaledIndexPrimary(g, elemSize, loadKind)\n\treturn true\n}\nif renvoPreparedBackendActive == 0 && !g.meta.panicEnabled &&\n\t(g.c.renvoTarget == renvoTargetVM32 || g.c.renvoTargetArch == renvoArchAmd64 ||\n\t\tg.c.renvoTargetArch == renvoArchArm || g.c.renvoTargetArch == renvoArchAarch64 ||\n\t\tg.c.renvoTargetArch == renvoArch386 && !g.c.code16) &&\n\t(checkedSize || g.c.renvoTargetArch == renvoArchAmd64 && elemSize >= 0 && elemSize <= 2147483647) {\n\tindex := &ep.exprs[indexExpr.right]\n\tif index.kind == renvoExprIdent {\n\t\tlocal := renvoFindLocalIndex(g, index.nameStart, index.nameEnd)\n\t\tif local >= 0 && g.locals[local].captureOff == 0 && renvoTypeIsNativeInt(g.meta, g.locals[local].typ) {\n\t\t\tif g.c.renvoTargetArch == renvoArchAmd64 {\n\t\t\t\trenvoAsmEmitText(a, \"\\x48\\x89\\xca\") // MOV RDX, RCX (length)\n\t\t\t\trenvoAsmLoadTertiaryStack(a, g.locals[local].offset)\n\t\t\t\tif !(sliceType.kind == renvoTypeSlice && !pointerArray && baseExpr.kind == renvoExprIdent &&\n\t\t\t\t\tlocal+1 == g.boundedIndexLocal && renvoFindLocalIndex(g, baseExpr.nameStart, baseExpr.nameEnd)+1 == g.boundedSliceLocal) {\n\t\t\t\t\tfault := renvoEnsureUncaughtFaultHelper(g, false)\n\t\t\t\t\trenvoAsmEmitText(a, \"\\x48\\x39\\xd1\") // CMP RCX, RDX\n\t\t\t\t\trenvoAmd64AsmJccLabel(a, 0x83, fault)\n\t\t\t\t}\n\t\t\t\trenvoEmitScaledIndexPrimary(g, elemSize, loadKind)\n\t\t\t\treturn true\n\t\t\t} else if g.c.renvoTargetArch == renvoArch386 {\n\t\t\t\trenvoAsmEmitText(a, \"\\x89\\xca\")\n\t\t\t} else if g.c.renvoTargetArch == renvoArchArm {\n\t\t\t\trenvoArmAsmMovRegReg(a, 1, 2)\n\t\t\t} else if g.c.renvoTargetArch == renvoArchAarch64 {\n\t\t\t\trenvoAarch64AsmMovRegReg(a, 1, 2)\n\t\t\t} else {\n\t\t\t\trenvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRdx, renvoWasm32RegRcx)\n\t\t\t}\n\t\t\trenvoAsmLoadTertiaryStack(a, g.locals[local].offset)\n\t\t\tif sliceType.kind == renvoTypeSlice && !pointerArray && baseExpr.kind == renvoExprIdent &&\n\t\t\t\tlocal+1 == g.boundedIndexLocal && renvoFindLocalIndex(g, baseExpr.nameStart, baseExpr.nameEnd)+1 == g.boundedSliceLocal {\n\t\t\t\trenvoAsmAddScaledTertiary(a, elemSize)\n\t\t\t} else if g.c.renvoTargetArch == renvoArchArm {\n\t\t\t\trenvoArmEmitCheckedIndexAddress(g, elemSize)\n\t\t\t} else {\n\t\t\t\trenvoAsmCallLabel(a, renvoEnsureIndexAddressHelper(g, elemSize))\n\t\t\t}\n\t\t\trenvoEmitIndexLoadFromAddress(g, loadKind)\n\t\t\treturn true\n\t\t}\n\t}\n}\nrenvoAsmPushPrimary(a)\nrenvoAsmPushTertiary(a)\nif !renvoEmitIntExpr(g, ep, indexExpr.right) {\n\treturn false\n}\nif g.boundedIndexLocal > 0 && baseExpr.kind == renvoExprIdent &&\n\tep.exprs[indexExpr.right].kind == renvoExprIdent && sliceType.kind == renvoTypeSlice && !pointerArray {\n\tindex := &ep.exprs[indexExpr.right]\n\tif renvoFindLocalIndex(g, index.nameStart, index.nameEnd)+1 == g.boundedIndexLocal &&\n\t\trenvoFindLocalIndex(g, baseExpr.nameStart, baseExpr.nameEnd)+1 == g.boundedSliceLocal {\n\t\t// The enclosing strict length guard dominates this access, and neither\n\t\t// descriptor nor counter can change in the body or through an alias.\n\t\trenvoAsmCopyPrimaryToSecondary(a)\n\t\trenvoAsmPopTertiary(a)\n\t\trenvoAsmCopySecondaryToTertiary(a)\n\t\trenvoAsmPopPrimary(a)\n\t\trenvoEmitScaledIndexPrimary(g, elemSize, loadKind)\n\t\treturn true\n\t}\n}\nif !g.meta.panicEnabled && checkedSize {\n\trenvoAsmCopyPrimaryToTertiary(a)\n\trenvoAsmPopSecondary(a)\n\trenvoAsmPopPrimary(a)\n\trenvoEmitCheckedIndexAddress(g, elemSize)\n\treturn true\n}\nrenvoAsmPopTertiary(a)\nrenvoEmitRuntimeBoundsCheck(g)\nrenvoAsmCopySecondaryToTertiary(a)\nrenvoAsmPopPrimary(a)\nrenvoEmitScaledIndexPrimary(g, elemSize, loadKind)\nreturn true"},
	{Name: "emit_slice_bounds_checks", Suffix: "EmitSliceBoundsChecks", Function: "renvoEmitSliceBoundsChecks", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"lowOff", "int"}, {"highOff", "int"}, {"maxOff", "int"}, {"capOff", "int"}}, Prepared: "renvoNonNil(g)\na := &g.asm\nif renvoEmitOptimizedSliceBoundsChecks(g, lowOff, highOff, maxOff, capOff) {\n\treturn\n}\nif renvoPreparedBackendActive == 0 && (g.c.renvoTargetArch == renvoArchArm || g.c.renvoTargetArch == renvoArchAarch64) {\n\tinvalid := renvoAsmNewLabel(a)\n\tdone := renvoAsmNewLabel(a)\n\t// With a nonnegative capacity, the unsigned chain low <= high <=\n\t// max <= cap rejects every negative bound as well as reversed bounds.\n\t// Read each saved operand once; no expression is reevaluated here.\n\tif g.c.renvoTargetArch == renvoArchArm {\n\t\trenvoArmAsmLoadRegStack(a, 3, capOff)\n\t\trenvoArmAsmLoadRegStack(a, 2, maxOff)\n\t\trenvoArmAsmLoadRegStack(a, 1, highOff)\n\t\trenvoArmAsmLoadRegStack(a, 0, lowOff)\n\t\trenvoArmAsmCmpRegImm(a, 3, 0)\n\t\trenvoArmAsmBCondLabel(a, invalid, 4) // MI\n\t\tfor reg := 0; reg < 3; reg++ {\n\t\t\trenvoArmAsmCmpRegReg(a, reg+1, reg)\n\t\t\trenvoArmAsmBCondLabel(a, invalid, 3) // CC\n\t\t}\n\t} else {\n\t\trenvoAarch64AsmLoadRegStack(a, 3, capOff)\n\t\trenvoAarch64AsmLoadRegStack(a, 2, maxOff)\n\t\trenvoAarch64AsmLoadRegStack(a, 1, highOff)\n\t\trenvoAarch64AsmLoadRegStack(a, 0, lowOff)\n\t\trenvoAarch64AsmCmpRegImm(a, 3, 0)\n\t\trenvoAarch64AsmBCondLabel(a, invalid, 4) // MI\n\t\tfor reg := 0; reg < 3; reg++ {\n\t\t\trenvoAarch64AsmCmpRegReg(a, reg+1, reg)\n\t\t\trenvoAarch64AsmBCondLabel(a, invalid, 3) // LO\n\t\t}\n\t}\n\trenvoAsmJmpMarkLabel(a, done, invalid)\n\trenvoEmitRuntimeFault(g)\n\trenvoAsmMarkLabel(a, done)\n\treturn\n}\ninvalid := renvoAsmNewLabel(a)\ndone := renvoAsmNewLabel(a)\nrenvoEmitStackLessImmJump(g, lowOff, 0, invalid)\nrenvoAsmJltStackStack(a, highOff, lowOff, invalid)\nrenvoAsmJltStackStack(a, maxOff, highOff, invalid)\nrenvoAsmJltStackStack(a, capOff, maxOff, invalid)\nrenvoAsmJmpMarkLabel(a, done, invalid)\nrenvoEmitRuntimeFault(g)\nrenvoAsmMarkLabel(a, done)"},
	{Name: "emit_string_concat_pair_value_regs", Suffix: "EmitStringConcatPairValueRegs", Function: "renvoEmitStringConcatPairValueRegs", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"left", "*renvoExprParse"}, {"leftIndex", "int"}, {"right", "*renvoExprParse"}, {"rightIndex", "int"}}, Prepared: "renvoNonNil(g, left, right)\na := &g.asm\nif g.c.renvoTarget == renvoTargetVM32 && (renvoStringConcatIsBinary(g, left, leftIndex) || renvoStringConcatIsBinary(g, right, rightIndex)) {\n\treturn renvoEmitStringConcatFlatValueRegs(g, left, leftIndex, right, rightIndex)\n}\nif renvoPreparedBackendActive == 0 && (g.c.renvoTargetArch == renvoArchAmd64 || g.c.renvoTargetArch == renvoArch386 && !g.c.code16 || g.c.renvoTargetArch == renvoArchWasm32 || g.c.renvoTargetArch == renvoArchArm || g.c.renvoTargetArch == renvoArchAarch64) {\n\tlabel := 0\n\tif g.c.renvoTargetArch == renvoArchWasm32 {\n\t\tlabel = renvoEnsureStringConcatWasm32(g)\n\t} else if g.c.renvoTargetArch == renvoArchArm || g.c.renvoTargetArch == renvoArchAarch64 {\n\t\tlabel = renvoEnsureStringStorageArm(g, true)\n\t} else {\n\t\tlabel = renvoEnsureStringConcatX86(g)\n\t}\n\tif !renvoEmitStringValueRegs(g, left, leftIndex) {\n\t\treturn false\n\t}\n\trenvoAsmPushStringRegs(a)\n\tif !renvoEmitStringValueRegs(g, right, rightIndex) {\n\t\treturn false\n\t}\n\trenvoAsmCopySecondaryToTertiary(a)\n\trenvoAsmCopyPrimaryToSecondary(a)\n\trenvoAsmPopCallWord0(a)\n\trenvoAsmPopCallWord1(a)\n\trenvoAsmCallLabel(a, label)\n\trenvoEmitArenaAllocationCheck(g)\n\treturn true\n}\nleftPtr := renvoAddUnnamedLocal(g, renvoTypeInt)\nleftLen := renvoAddUnnamedLocal(g, renvoTypeInt)\nrightPtr := renvoAddUnnamedLocal(g, renvoTypeInt)\nrightLen := renvoAddUnnamedLocal(g, renvoTypeInt)\ntotalLen := renvoAddUnnamedLocal(g, renvoTypeInt)\nresultPtr := renvoAddUnnamedLocal(g, renvoTypeInt)\nrightDest := renvoAddUnnamedLocal(g, renvoTypeInt)\n// Evaluate both authored operands in source order before copying either.\n// One exact-sized allocation replaces the per-byte append/growth loop.\nif !renvoEmitStringValueRegs(g, left, leftIndex) {\n\treturn false\n}\nrenvoAsmStorePrimarySecondaryStack(a, leftPtr, leftLen)\nif !renvoEmitStringValueRegs(g, right, rightIndex) {\n\treturn false\n}\nrenvoAsmStorePrimarySecondaryStack(a, rightPtr, rightLen)\nrenvoAsmLoadPrimaryTertiaryStack(a, leftLen, rightLen)\nrenvoAsmAddPrimaryTertiary(a)\nrenvoAsmStorePrimaryStack(a, totalLen)\nlengthOK := renvoAsmNewLabel(a)\nrenvoAsmJgeStackStack(a, totalLen, leftLen, lengthOK)\nrenvoEmitUncaughtFaultTransfer(g, true)\nrenvoAsmMarkLabel(a, lengthOK)\nrenvoEmitArenaAllocStackPrimary(g, totalLen)\nrenvoAsmStorePrimaryStack(a, resultPtr)\nrenvoAsmLoadTertiaryStack(a, leftLen)\nrenvoAsmAddPrimaryTertiary(a)\nrenvoAsmStorePrimaryStack(a, rightDest)\nrenvoEmitCopyBytes(g, leftPtr, resultPtr, leftLen)\nrenvoEmitCopyBytes(g, rightPtr, rightDest, rightLen)\nrenvoAsmLoadPrimarySecondaryStack(a, resultPtr, totalLen)\nreturn true"},
	{Name: "emit_string_ordering", Suffix: "EmitStringOrdering", Function: "renvoEmitStringOrdering", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"ep", "*renvoExprParse"}, {"e", "*renvoExpr"}}, Prepared: "a := &g.asm\nif renvoPreparedBackendActive == 0 && (g.c.renvoTarget == renvoTargetVM32 || g.c.renvoTargetArch == renvoArchAmd64) {\n\tlabel := renvoEnsureStringOrderHelper(g)\n\tif !renvoEmitStringCompareValueRegs(g, ep, e.left, renvoStringCompareOperandIsReadOnly(g, ep, e.right)) {\n\t\treturn false\n\t}\n\trenvoAsmPushStringRegs(a)\n\tif !renvoEmitStringCompareValueRegs(g, ep, e.right, true) {\n\t\treturn false\n\t}\n\trenvoAsmCopySecondaryToTertiary(a)\n\trenvoAsmCopyPrimaryToSecondary(a)\n\trenvoAsmPopCallWord0(a)\n\trenvoAsmPopCallWord1(a)\n\trenvoAsmCallLabel(a, label)\n\tcond, setcc := renvoWasm32CondLt, 0x9c\n\tif renvoTokCharIs(g.prog, e.tok, '>') || renvoTok2Is(g.prog, e.tok, '>', '=') {\n\t\tcond, setcc = renvoWasm32CondGt, 0x9f\n\t}\n\tif renvoTok2Is(g.prog, e.tok, '<', '=') {\n\t\tcond, setcc = renvoWasm32CondLe, 0x9e\n\t} else if renvoTok2Is(g.prog, e.tok, '>', '=') {\n\t\tcond, setcc = renvoWasm32CondGe, 0x9d\n\t}\n\tif g.c.renvoTarget == renvoTargetVM32 {\n\t\trenvoAsmEmit8(a, renvoWasm32OpSetCond)\n\t\trenvoAsmEmit8(a, cond)\n\t} else {\n\t\trenvoAsmEmit3(a, 0x0f, setcc, 0xc0)\n\t\trenvoAsmEmitText(a, \"\\x48\\x0f\\xb6\\xc0\")\n\t}\n\treturn true\n}\nleft := renvoAddUnnamedLocal(g, renvoTypeString)\nright := renvoAddUnnamedLocal(g, renvoTypeString)\nif !renvoEmitStringValueRegs(g, ep, e.left) {\n\treturn false\n}\nrenvoAsmStorePrimarySecondaryStack(a, left, left-8)\nif !renvoEmitStringValueRegs(g, ep, e.right) {\n\treturn false\n}\nrenvoAsmStorePrimarySecondaryStack(a, right, right-8)\nindex := renvoAddUnnamedLocal(g, renvoTypeInt)\nlbyte := renvoAddUnnamedLocal(g, renvoTypeInt)\nrbyte := renvoAddUnnamedLocal(g, renvoTypeInt)\nloop := renvoAsmNewLabel(a)\nlengths := renvoAsmNewLabel(a)\nless := renvoAsmNewLabel(a)\ngreater := renvoAsmNewLabel(a)\ndone := renvoAsmNewLabel(a)\nrenvoAsmStoreStackImm(a, index, 0)\nrenvoAsmMarkLabel(a, loop)\nrenvoAsmJgeStackStack(a, index, left-8, lengths)\nrenvoAsmJgeStackStack(a, index, right-8, lengths)\nrenvoAsmLoadPrimaryTertiaryStack(a, left, index)\nrenvoAsmLoadPrimaryIndexTertiarySize(a, 1)\nrenvoAsmStorePrimaryStack(a, lbyte)\nrenvoAsmLoadPrimaryTertiaryStack(a, right, index)\nrenvoAsmLoadPrimaryIndexTertiarySize(a, 1)\nrenvoAsmStorePrimaryStack(a, rbyte)\nrenvoAsmCompareStackStackJump(a, lbyte, rbyte, less, renvoConditionSignedLess)\nrenvoAsmCompareStackStackJump(a, lbyte, rbyte, greater, renvoConditionSignedGreater)\nrenvoAsmIncStack(a, index)\nrenvoAsmJmpMarkLabel(a, loop, lengths)\nrenvoAsmCompareStackStackJump(a, left-8, right-8, less, renvoConditionSignedLess)\nrenvoAsmCompareStackStackJump(a, left-8, right-8, greater, renvoConditionSignedGreater)\nequalValue := 0\nif renvoTok2Is(g.prog, e.tok, '<', '=') || renvoTok2Is(g.prog, e.tok, '>', '=') {\n\tequalValue = 1\n}\nlessValue := 0\nif renvo_runtime_UnsafeByteAt(g.prog.src, int(renvoTokStart(g.prog, e.tok))) == '<' {\n\tlessValue = 1\n}\nrenvoAsmPrimaryImm(a, equalValue)\nrenvoAsmJmpMarkLabel(a, done, less)\nrenvoAsmPrimaryImm(a, lessValue)\nrenvoAsmJmpMarkLabel(a, done, greater)\nrenvoAsmPrimaryImm(a, 1-lessValue)\nrenvoAsmMarkLabel(a, done)\nreturn true"},
	{Name: "emit_word_compare_jump", Suffix: "EmitWordCompareJump", Function: "renvoEmitWordCompareJump", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"ep", "*renvoExprParse"}, {"e", "*renvoExpr"}, {"label", "int"}, {"jumpIfTrue", "bool"}}, Prepared: "renvoNonNil(g, ep, e)\np := g.prog\nif e.tok < 0 || e.tok >= renvoTokCount(p) {\n\treturn false\n}\nstart := renvoTokStart(p, e.tok)\nend := renvoTokEnd(p, e.tok)\nif start >= end {\n\treturn false\n}\nc0 := renvo_runtime_UnsafeByteAt(p.src, start)\nvar c1 byte\nif start+1 < end {\n\tc1 = renvo_runtime_UnsafeByteAt(p.src, start+1)\n}\nif !renvoIsComparisonChars(c0, c1) {\n\treturn false\n}\n// The immediate compare fast path operates on raw integer bits. Floating\n// operands must use IEEE comparison so NaNs remain unordered.\nusesFloat := renvoBinaryUsesFloat(g, ep, e)\nfloatKind := 0\nif usesFloat {\n\tfloatKind = renvoBinaryFloatKind(g, ep, e)\n}\nleftIndex := e.left\nrightIndex := e.right\nif (c0 == '=' || c0 == '!') && renvoComparisonCompositeType(g, ep, e, renvoInferParsedExprType(g, ep, e.left), renvoInferParsedExprType(g, ep, e.right)) != 0 {\n\treturn false\n}\nunsigned := (c0 == '<' || c0 == '>') &&\n\t(renvoExprHasUnsignedIntType(g, ep, e.left) ||\n\t\trenvoExprHasUnsignedIntType(g, ep, e.right))\nright := &ep.exprs[rightIndex]\nif !usesFloat && (c0 == '<' || c0 == '>') && renvoPreparedBackendActive == 0 && g.c.renvoTarget == renvoTargetVM32 {\n\tvalue := renvoEvalConstExpr(g, ep, rightIndex)\n\tif value.ok && value.value >= -2147483648 && value.value <= 2147483647 && (!unsigned || value.value >= 0) {\n\t\tif !renvoEmitIntExpr(g, ep, leftIndex) {\n\t\t\treturn false\n\t\t}\n\t\trenvoNormalizeNativeExprPrimary(g, ep, leftIndex)\n\t\tif value.value == 0 && !unsigned {\n\t\t\trenvoWasm32EmitRegImm(&g.asm, renvoWasm32OpCmpRegImm, renvoWasm32RegRax, 0)\n\t\t\trenvoEmitCompareJumpOp(&g.asm, c0, c1, label, jumpIfTrue, false)\n\t\t\treturn true\n\t\t}\n\t\tnonnegative := renvoVMExprIsNonnegative(g, ep, leftIndex)\n\t\tif !nonnegative || value.value < 0 {\n\t\t\t// Comparing opposite signs has a known result, independent of the\n\t\t\t// subtraction flag. Same-sign subtraction cannot overflow.\n\t\t\tknownGreater := unsigned || value.value < 0\n\t\t\tknownTakesJump := (c0 == '>') == knownGreater\n\t\t\tif !jumpIfTrue {\n\t\t\t\tknownTakesJump = !knownTakesJump\n\t\t\t}\n\t\t\tdone := label\n\t\t\tif !knownTakesJump {\n\t\t\t\tdone = renvoAsmNewLabel(&g.asm)\n\t\t\t}\n\t\t\trenvoWasm32EmitRegImm(&g.asm, renvoWasm32OpCmpRegImm, renvoWasm32RegRax, 0)\n\t\t\tcond := renvoWasm32CondLt\n\t\t\tif value.value < 0 {\n\t\t\t\tcond = renvoWasm32CondGe\n\t\t\t}\n\t\t\trenvoWasm32EmitCondBranch(&g.asm, cond, done)\n\t\t\trenvoWasm32EmitRegImm(&g.asm, renvoWasm32OpCmpRegImm, renvoWasm32RegRax, value.value)\n\t\t\trenvoEmitCompareJumpOp(&g.asm, c0, c1, label, jumpIfTrue, false)\n\t\t\tif !knownTakesJump {\n\t\t\t\trenvoAsmMarkLabel(&g.asm, done)\n\t\t\t}\n\t\t} else {\n\t\t\trenvoWasm32EmitRegImm(&g.asm, renvoWasm32OpCmpRegImm, renvoWasm32RegRax, value.value)\n\t\t\trenvoEmitCompareJumpOp(&g.asm, c0, c1, label, jumpIfTrue, false)\n\t\t}\n\t\treturn true\n\t}\n}\nif !usesFloat && !(unsigned && g.c.renvoTargetArch == renvoArchWasm32) {\n\trightConst := renvoEvalConstExpr(g, ep, rightIndex)\n\tleft := &ep.exprs[leftIndex]\n\tif rightConst.ok && renvoPreparedBackendActive == 0 && g.c.renvoTargetArch == renvoArchArm &&\n\t\trenvoTypeSize(g.meta, renvoInferParsedExprType(g, ep, leftIndex)) <= 4 {\n\t\tif !renvoEmitIntExpr(g, ep, leftIndex) {\n\t\t\treturn false\n\t\t}\n\t\trenvoNormalizeNativeExprPrimary(g, ep, leftIndex)\n\t\trenvoArmAsmCmpRegImm(&g.asm, renvoArmRegRax, rightConst.value)\n\t\trenvoEmitCompareJumpOp(&g.asm, c0, c1, label, jumpIfTrue, unsigned)\n\t\treturn true\n\t}\n\tif rightConst.ok && rightConst.value >= -2147483648 && rightConst.value <= 2147483647 && renvoPreparedBackendActive == 0 && g.c.renvoTargetArch == renvoArchAmd64 && left.kind == renvoExprIdent {\n\t\tlocal := renvoFindLocalIndex(g, left.nameStart, left.nameEnd)\n\t\tif local >= 0 && g.locals[local].captureOff == 0 {\n\t\t\tkind := renvoResolveType(g.meta, g.locals[local].typ).kind\n\t\t\tif kind == renvoTypeInt || kind == renvoTypeUint64 || kind == renvoTypeInt32 || kind == renvoTypeUint32 {\n\t\t\t\topcode := 0x81\n\t\t\t\tif renvoAsmImmFits8Signed(rightConst.value) {\n\t\t\t\t\topcode = 0x83\n\t\t\t\t}\n\t\t\t\tif kind == renvoTypeInt || kind == renvoTypeUint64 {\n\t\t\t\t\topcode = opcode<<8 | 0x48\n\t\t\t\t}\n\t\t\t\tif opcode > 255 {\n\t\t\t\t\trenvoAmd64AsmStackMem(&g.asm, g.locals[local].offset, opcode, 0x7d, 0xbd)\n\t\t\t\t} else {\n\t\t\t\t\trenvoAsmEmit8(&g.asm, opcode)\n\t\t\t\t\toffset := g.locals[local].offset\n\t\t\t\t\tif offset >= 0 && offset <= 128 {\n\t\t\t\t\t\trenvoAsmEmit8(&g.asm, 0x7d)\n\t\t\t\t\t\trenvoAsmEmit8(&g.asm, -offset)\n\t\t\t\t\t} else {\n\t\t\t\t\t\trenvoAsmEmit8(&g.asm, 0xbd)\n\t\t\t\t\t\trenvoAsmEmit32(&g.asm, -offset)\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t\tif renvoAsmImmFits8Signed(rightConst.value) {\n\t\t\t\t\trenvoAsmEmit8(&g.asm, rightConst.value)\n\t\t\t\t} else {\n\t\t\t\t\trenvoAsmEmit32(&g.asm, rightConst.value)\n\t\t\t\t}\n\t\t\t\trenvoEmitCompareJumpOp(&g.asm, c0, c1, label, jumpIfTrue, unsigned)\n\t\t\t\treturn true\n\t\t\t}\n\t\t}\n\t}\n\tif rightConst.ok && renvoAsmImmFits8Signed(rightConst.value) &&\n\t\t(g.c.renvoTargetArch != renvoArchWasm32 || rightConst.value == 0 || c0 == '=' || c0 == '!') {\n\t\tif !renvoEmitIntExpr(g, ep, leftIndex) {\n\t\t\treturn false\n\t\t}\n\t\t// Locals occupy a native-sized backend slot even when their language\n\t\t// type is narrower. A pointer write may update only the low byte, word,\n\t\t// or dword, so normalize the loaded operand before an immediate branch.\n\t\trenvoNormalizeNativeExprPrimary(g, ep, leftIndex)\n\t\trenvoAsmCmpPrimaryImm8Discard(&g.asm, rightConst.value)\n\t\trenvoEmitCompareJumpOp(&g.asm, c0, c1, label, jumpIfTrue, unsigned)\n\t\treturn true\n\t}\n}\nif !usesFloat && renvoPreparedBackendActive == 0 && g.c.renvoTargetArch == renvoArchAmd64 {\n\tleft := &ep.exprs[leftIndex]\n\trightOffset := renvoNativeCompareFrameOffset(g, ep, rightIndex)\n\tif rightOffset >= 0 {\n\t\tleftOffset := -1\n\t\tif left.kind == renvoExprIdent {\n\t\t\tleftOffset = renvoNativeCompareFrameOffset(g, ep, leftIndex)\n\t\t}\n\t\tif leftOffset >= 0 {\n\t\t\trenvoAsmLoadPrimaryStack(&g.asm, rightOffset)\n\t\t\trenvoAmd64AsmStackMem(&g.asm, leftOffset, 0x3948, 0x45, 0x85)\n\t\t\trenvoEmitCompareJumpOp(&g.asm, c0, c1, label, jumpIfTrue, unsigned)\n\t\t\treturn true\n\t\t}\n\t\tif left.kind == renvoExprSelector && ep.exprs[left.left].kind == renvoExprIdent {\n\t\t\tkind := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, leftIndex)).kind\n\t\t\tif kind == renvoTypeInt || kind == renvoTypeUint64 {\n\t\t\t\tif !renvoEmitSelectorAddressSecondary(g, ep, leftIndex) {\n\t\t\t\t\treturn false\n\t\t\t\t}\n\t\t\t\trenvoAsmLoadPrimaryStack(&g.asm, rightOffset)\n\t\t\t\trenvoAsmEmitText(&g.asm, \"\\x48\\x39\\x02\") // CMP [RDX], RAX\n\t\t\t\trenvoEmitCompareJumpOp(&g.asm, c0, c1, label, jumpIfTrue, unsigned)\n\t\t\t\treturn true\n\t\t\t}\n\t\t}\n\t\tkind := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, leftIndex)).kind\n\t\tif kind == renvoTypeInt || kind == renvoTypeUint64 {\n\t\t\tif !renvoEmitIntExpr(g, ep, leftIndex) {\n\t\t\t\treturn false\n\t\t\t}\n\t\t\trenvoAmd64AsmStackMem(&g.asm, rightOffset, 0x3b48, 0x45, 0x85)\n\t\t\trenvoEmitCompareJumpOp(&g.asm, c0, c1, label, jumpIfTrue, unsigned)\n\t\t\treturn true\n\t\t}\n\t}\n\tif right.kind == renvoExprIdent {\n\t\tlocal := renvoFindLocalIndex(g, right.nameStart, right.nameEnd)\n\t\tif local >= 0 && g.locals[local].captureOff == 0 {\n\t\t\tkind := renvoResolveType(g.meta, g.locals[local].typ).kind\n\t\t\tleftKind := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, leftIndex)).kind\n\t\t\tif leftKind == kind && renvoTypeKindIsScalarInt(kind) {\n\t\t\t\tif !renvoEmitIntExpr(g, ep, leftIndex) {\n\t\t\t\t\treturn false\n\t\t\t\t}\n\t\t\t\t// Compare at the language width: pointer writes may leave the\n\t\t\t\t// upper bytes of a narrow local's value slot unchanged.\n\t\t\t\tsize := renvoScalarKindSize(g.c.renvoNativeIntSize, kind)\n\t\t\t\tif size == 8 {\n\t\t\t\t\trenvoAsmEmit16(&g.asm, 0x3b48)\n\t\t\t\t} else if size == 2 {\n\t\t\t\t\trenvoAsmEmit16(&g.asm, 0x3b66)\n\t\t\t\t} else if size == 1 {\n\t\t\t\t\trenvoAsmEmit8(&g.asm, 0x3a)\n\t\t\t\t} else {\n\t\t\t\t\trenvoAsmEmit8(&g.asm, 0x3b)\n\t\t\t\t}\n\t\t\t\toffset := g.locals[local].offset\n\t\t\t\tif offset >= 0 && offset <= 128 {\n\t\t\t\t\trenvoAsmEmit2(&g.asm, 0x45, -offset)\n\t\t\t\t} else {\n\t\t\t\t\trenvoAsmEmit8(&g.asm, 0x85)\n\t\t\t\t\trenvoAsmEmit32(&g.asm, -offset)\n\t\t\t\t}\n\t\t\t\trenvoEmitCompareJumpOp(&g.asm, c0, c1, label, jumpIfTrue, unsigned)\n\t\t\t\treturn true\n\t\t\t}\n\t\t}\n\t}\n}\nif g.c.renvoTargetArch == renvoArchAmd64 {\n\tleft := &ep.exprs[leftIndex]\n\tif left.kind == renvoExprIdent && right.kind == renvoExprIdent {\n\t\tleftLocal := renvoFindLocalIndex(g, left.nameStart, left.nameEnd)\n\t\trightLocal := renvoFindLocalIndex(g, right.nameStart, right.nameEnd)\n\t\tif leftLocal >= 0 && rightLocal >= 0 && renvoTypeIsNativeInt(g.meta, g.locals[leftLocal].typ) && renvoTypeIsNativeInt(g.meta, g.locals[rightLocal].typ) {\n\t\t\trenvoAsmLoadPrimaryStack(&g.asm, g.locals[rightLocal].offset)\n\t\t\trenvoAmd64AsmStackMem(&g.asm, g.locals[leftLocal].offset, 0x3948, 0x45, 0x85)\n\t\t\trenvoEmitCompareJumpOp(&g.asm, c0, c1, label, jumpIfTrue, unsigned)\n\t\t\treturn true\n\t\t}\n\t}\n}\nif c0 == '=' || c0 == '!' {\n\tleftType := renvoInferParsedExprType(g, ep, leftIndex)\n\trightType := renvoInferParsedExprType(g, ep, rightIndex)\n\tleftResolved := renvoResolveType(g.meta, leftType)\n\trenvoNonNil(leftResolved)\n\tif leftResolved.kind == renvoTypeArray || leftResolved.kind == renvoTypeStruct || renvoTypeKindIsComplex(leftResolved.kind) {\n\t\treturn false\n\t}\n\tif renvoTypeIsString(g.meta, leftType) || renvoTypeIsString(g.meta, rightType) {\n\t\treturn false\n\t}\n\tif right.kind == renvoExprString {\n\t\treturn false\n\t}\n\tif right.kind == renvoExprIdent {\n\t\tlocalIndex := renvoFindLocalIndex(g, right.nameStart, right.nameEnd)\n\t\tif localIndex >= 0 && renvoTypeIsString(g.meta, g.locals[localIndex].typ) {\n\t\t\treturn false\n\t\t}\n\t}\n}\nif !usesFloat && renvoPreparedBackendActive == 0 &&\n\t(g.c.renvoTargetArch == renvoArchArm || g.c.renvoTargetArch == renvoArchAarch64 || g.c.renvoTarget == renvoTargetVM32) {\n\tleftOffset := renvoNativeCompareFrameOffset(g, ep, leftIndex)\n\trightOffset := renvoNativeCompareFrameOffset(g, ep, rightIndex)\n\tif leftOffset >= 0 && rightOffset >= 0 {\n\t\trenvoAsmLoadPrimaryStack(&g.asm, rightOffset)\n\t\trenvoAsmLoadTertiaryStack(&g.asm, leftOffset)\n\t\tif g.c.renvoTargetArch == renvoArchArm {\n\t\t\trenvoArmAsmCmpRegReg(&g.asm, renvoArmRegRcx, renvoArmRegRax)\n\t\t} else if g.c.renvoTargetArch == renvoArchAarch64 {\n\t\t\trenvoAarch64AsmCmpRegReg(&g.asm, renvoAarch64RegRcx, renvoAarch64RegRax)\n\t\t} else {\n\t\t\trenvoEmitWasmParsedCompareFlags(g, ep, leftIndex, rightIndex, c0, unsigned)\n\t\t\tunsigned = false\n\t\t}\n\t\trenvoEmitCompareJumpOp(&g.asm, c0, c1, label, jumpIfTrue, unsigned)\n\t\treturn true\n\t}\n}\nif !renvoEmitWideCompareOperand(g, ep, leftIndex, floatKind) {\n\treturn false\n}\nrenvoAsmPushPrimary(&g.asm)\nif !renvoEmitWideCompareOperand(g, ep, rightIndex, floatKind) {\n\treturn false\n}\nrenvoAsmPopTertiary(&g.asm)\nif usesFloat && (g.c.renvoTargetArch == renvoArchAmd64 || g.c.renvoTargetArch == renvoArchAarch64) {\n\tif !renvoEmitIEEEFloatPrimaryTertiaryOp(g, e.tok, floatKind) {\n\t\treturn false\n\t}\n\tif jumpIfTrue {\n\t\trenvoAsmJnzPrimary(&g.asm, label)\n\t} else {\n\t\trenvoAsmJzPrimary(&g.asm, label)\n\t}\n\treturn true\n}\n// Evaluation leaves the left operand in tertiary and the right in primary.\n// Compare them directly, preserving source order without a register swap.\nif renvoPreparedBackendActive != 0 {\n\trenvoRTGDirectCompare(&g.asm, renvoRTGTertiary, renvoRTGPrimary)\n} else if g.c.renvoTargetArch == renvoArchAarch64 {\n\trenvoAarch64AsmCmpRegReg(&g.asm, renvoAarch64RegRcx, renvoAarch64RegRax)\n} else if g.c.renvoTargetArch == renvoArchArm {\n\trenvoArmAsmCmpRegReg(&g.asm, renvoArmRegRcx, renvoArmRegRax)\n} else if g.c.renvoTargetArch == renvoArchWasm32 {\n\trenvoEmitWasmParsedCompareFlags(g, ep, leftIndex, rightIndex, c0, unsigned)\n\tunsigned = false\n} else {\n\tunsigned = renvoEmitCompareWordOperands(g, unsigned)\n}\nrenvoEmitCompareJumpOp(&g.asm, c0, c1, label, jumpIfTrue, unsigned)\nreturn true"},
	{Name: "emit_zero_struct_result", Suffix: "EmitZeroStructResult", Function: "renvoEmitZeroStructResult", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"size", "int"}}, Prepared: "a := &g.asm\nif renvoPreparedBackendActive == 0 && g.c.renvoTarget == renvoTargetVM32 && size >= 32 {\n\trenvoAsmPushSecondary(a)\n\trenvoAsmPushTertiary(a)\n\trenvoAsmCopySecondaryToPrimary(a)\n\trenvoAsmPushImm(a, size)\n\trenvoAsmPopTertiary(a)\n\trenvoEmitMakeZero(g)\n\trenvoAsmPopTertiary(a)\n\trenvoAsmPopSecondary(a)\n\trenvoAsmPrimaryImm(a, 0)\n\treturn\n}\nif renvoPreparedBackendActive == 0 && g.c.renvoTarget != renvoTargetVM32 &&\n\t(size >= 64 || size >= 32 && (g.c.renvoTargetArch == renvoArchAmd64 || g.c.renvoTargetArch == renvoArch386 && !g.c.code16)) {\n\trenvoAsmCopySecondaryToPrimary(a)\n\trenvoAsmPushImm(a, size)\n\trenvoAsmPopTertiary(a)\n\trenvoEmitMakeZero(g)\n\treturn\n}\nrenvoAsmPrimaryImm(a, 0)\nfor at := 0; at < size; at += g.c.renvoNativeIntSize {\n\trenvoAsmStorePrimaryMemSecondaryDisp(a, at)\n}"},
	{Name: "zero_local_at_offset", Suffix: "ZeroLocalAtOffset", Function: "renvoZeroLocalAtOffset", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"offset", "int"}}, Prepared: "renvoNonNil(g)\n\t a := &g.asm\nsize := 8\ntyp := renvoTypeInt\nfor i := g.localCount - 1; i >= 0; i-- {\n\tlocal := &g.locals[i]\n\tif local.offset == offset {\n\t\tsize = local.size\n\t\ttyp = local.typ\n\t\tbreak\n\t}\n}\nt := renvoResolveType(g.meta, typ)\nrenvoNonNil(t)\nif renvoPreparedBackendActive == 0 && g.c.renvoTargetArch == renvoArchAmd64 && size == 16 {\n\t// A single unaligned vector store clears exactly one string descriptor\n\t// or two-word value while retaining the scalar zero-result contract.\n\trenvoAsmPrimaryImm(a, 0)\n\trenvoAsmEmitText(a, \"\\x0f\\x57\\xc0\\xf3\\x0f\\x7f\")\n\tif offset <= 128 {\n\t\trenvoAsmEmit8(a, 0x45)\n\t\trenvoAsmEmit8(a, -offset)\n\t} else {\n\t\trenvoAsmEmit8(a, 0x85)\n\t\trenvoAsmEmit32(a, -offset)\n\t}\n\treturn\n}\nif renvoPreparedBackendActive == 0 && g.c.renvoTargetArch == renvoArchWasm32 && g.c.renvoTarget != renvoTargetVM32 && size >= 16 {\n\trenvoAsmEmit8(a, renvoWasm32OpZeroFrameBlock)\n\trenvoAsmEmit32(a, offset)\n\trenvoAsmEmit32(a, size)\n\ta.lastPrimaryLoad = 0\n\treturn\n}\nif renvoPreparedBackendActive == 0 && g.c.renvoTarget == renvoTargetVM32 && size >= 32 {\n\trenvoAsmPushSecondary(a)\n\trenvoAsmPushTertiary(a)\n\trenvoAsmAddressPrimaryStack(a, offset)\n\trenvoAsmPushImm(a, size)\n\trenvoAsmPopTertiary(a)\n\trenvoEmitMakeZero(g)\n\trenvoAsmPopTertiary(a)\n\trenvoAsmPopSecondary(a)\n\trenvoAsmPrimaryImm(a, 0)\n\treturn\n}\nif t.kind == renvoTypeSlice && g.c.renvoTargetArch != renvoArchAmd64 {\n\trenvoInitEmptySliceStack(g, offset)\n\treturn\n}\nrenvoZeroLocalStorage(g, offset, size)\nif t.kind == renvoTypeStruct {\n\trenvoInitStructSliceFields(g, typ, offset)\n}"},
	{Name: "tracks_index_range_facts", Suffix: "TracksIndexRangeFacts", Function: "renvoTracksIndexRangeFacts", Receiver: compilerBindingParameter{"c", "*renvoCompileContext"}, Result: "bool", Failure: "false", Prepared: "return renvoPreparedBackendActive == 0 && c.renvoTarget == renvoTargetVM32"},
	// Native helper bodies are supplied by definitions. Prepared adapters never
	// invoke them: their callers use the portable fallback when preparation is active.
	{Name: "aarch64_copy_fixed", Suffix: "Aarch64CopyFixed", Function: "renvoAarch64CopyFixed", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"srcOffset", "int"}, {"destOffset", "int"}, {"size", "int"}, {"mode", "int"}}, Prepared: "g.asm.patchFailed = true"},
	{Name: "amd64_emit_fixed_vector_copy", Suffix: "Amd64EmitFixedVectorCopy", Function: "renvoAmd64EmitFixedVectorCopy", Receiver: compilerBindingParameter{"a", "*renvoAsm"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"size", "int"}}, Prepared: "a.patchFailed = true"},
	{Name: "amd64_ensure_fixed_vector_copy", Suffix: "Amd64EnsureFixedVectorCopy", Function: "renvoAmd64EnsureFixedVectorCopy", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "int", Failure: "0", Parameters: []compilerBindingParameter{{"size", "int"}}, Prepared: "g.asm.patchFailed = true\nreturn -1"},
	{Name: "arm_copy_fixed", Suffix: "ArmCopyFixed", Function: "renvoArmCopyFixed", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"srcOffset", "int"}, {"destOffset", "int"}, {"size", "int"}, {"mode", "int"}}, Prepared: "g.asm.patchFailed = true"},
	{Name: "arm_emit_checked_index_address", Suffix: "ArmEmitCheckedIndexAddress", Function: "renvoArmEmitCheckedIndexAddress", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"elemSize", "int"}}, Prepared: "g.asm.patchFailed = true"},
	{Name: "emit_frame_index_tertiary", Suffix: "EmitFrameIndexTertiary", Function: "renvoEmitFrameIndexTertiary", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "bool", Failure: "false", Parameters: []compilerBindingParameter{{"ep", "*renvoExprParse"}, {"idx", "int"}, {"depth", "int"}}, Prepared: "return false"},
	{Name: "emit_scaled_index_primary", Suffix: "EmitScaledIndexPrimary", Function: "renvoEmitScaledIndexPrimary", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"size", "int"}, {"kind", "int"}}, Prepared: "renvoNonNil(g)\nif kind != 0 && renvoPreparedBackendActive == 0 && g.c.renvoTargetArch == renvoArchAmd64 &&\n\t(size == 1 || size == 2 || size == 4 || size == 8) {\n\trenvoAmd64AsmLoadRaxIndexRcxSize(&g.asm, size)\n\trenvoAsmNormalizePrimaryForKind(&g.asm, kind)\n\treturn\n}\nrenvoAsmAddScaledTertiary(&g.asm, size)\nrenvoEmitIndexLoadFromAddress(g, kind)"},
	{Name: "emit_string_storage_arm_return", Suffix: "EmitStringStorageArmReturn", Function: "renvoEmitStringStorageArmReturn", Receiver: compilerBindingParameter{"a", "*renvoAsm"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{}, Prepared: "a.patchFailed = true"},
	{Name: "emit_wasm_parsed_compare_flags", Suffix: "EmitWasmParsedCompareFlags", Function: "renvoEmitWasmParsedCompareFlags", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"ep", "*renvoExprParse"}, {"leftIndex", "int"}, {"rightIndex", "int"}, {"c0", "byte"}, {"unsigned", "bool"}}, Prepared: "g.asm.patchFailed = true"},
	{Name: "ensure_copy_bytes_aarch64", Suffix: "EnsureCopyBytesAarch64", Function: "renvoEnsureCopyBytesAarch64", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "int", Failure: "0", Parameters: []compilerBindingParameter{}, Prepared: "g.asm.patchFailed = true\nreturn -1"},
	{Name: "ensure_copy_bytes_arm", Suffix: "EnsureCopyBytesArm", Function: "renvoEnsureCopyBytesArm", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "int", Failure: "0", Parameters: []compilerBindingParameter{}, Prepared: "g.asm.patchFailed = true\nreturn -1"},
	{Name: "ensure_copy_bytes_vm32", Suffix: "EnsureCopyBytesVM32", Function: "renvoEnsureCopyBytesVM32", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "int", Failure: "0", Parameters: []compilerBindingParameter{}, Prepared: "g.asm.patchFailed = true\nreturn -1"},
	{Name: "ensure_slice_bounds_vm32", Suffix: "EnsureSliceBoundsVM32", Function: "renvoEnsureSliceBoundsVM32", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "int", Failure: "0", Parameters: []compilerBindingParameter{}, Prepared: "g.asm.patchFailed = true\nreturn -1"},
	{Name: "ensure_small_risc_copy", Suffix: "EnsureSmallRiscCopy", Function: "renvoEnsureSmallRiscCopy", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "int", Failure: "0", Parameters: []compilerBindingParameter{{"size", "int"}}, Prepared: "g.asm.patchFailed = true\nreturn -1"},
	{Name: "ensure_string_concat_wasm32", Suffix: "EnsureStringConcatWasm32", Function: "renvoEnsureStringConcatWasm32", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "int", Failure: "0", Parameters: []compilerBindingParameter{}, Prepared: "g.asm.patchFailed = true\nreturn -1"},
	{Name: "ensure_string_concat_x86", Suffix: "EnsureStringConcatX86", Function: "renvoEnsureStringConcatX86", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "int", Failure: "0", Parameters: []compilerBindingParameter{}, Prepared: "g.asm.patchFailed = true\nreturn -1"},
	{Name: "ensure_string_copy_wasm32", Suffix: "EnsureStringCopyWasm32", Function: "renvoEnsureStringCopyWasm32", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "int", Failure: "0", Parameters: []compilerBindingParameter{}, Prepared: "g.asm.patchFailed = true\nreturn -1"},
	{Name: "ensure_string_copy_x86", Suffix: "EnsureStringCopyX86", Function: "renvoEnsureStringCopyX86", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "int", Failure: "0", Parameters: []compilerBindingParameter{}, Prepared: "g.asm.patchFailed = true\nreturn -1"},
	{Name: "ensure_string_order_helper", Suffix: "EnsureStringOrderHelper", Function: "renvoEnsureStringOrderHelper", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "int", Failure: "0", Parameters: []compilerBindingParameter{}, Prepared: "g.asm.patchFailed = true\nreturn -1"},
	{Name: "ensure_string_storage_arm", Suffix: "EnsureStringStorageArm", Function: "renvoEnsureStringStorageArm", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "int", Failure: "0", Parameters: []compilerBindingParameter{{"concat", "bool"}}, Prepared: "g.asm.patchFailed = true\nreturn -1"},
	{Name: "vm32_copy_fixed", Suffix: "VM32CopyFixed", Function: "renvoVM32CopyFixed", Receiver: compilerBindingParameter{"g", "*renvoLinearGen"}, Result: "", Failure: "", Parameters: []compilerBindingParameter{{"srcOffset", "int"}, {"destOffset", "int"}, {"size", "int"}, {"mode", "int"}}, Prepared: "g.asm.patchFailed = true"},
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
	return validateCompilerBindingsIndexed(document, arch, indexEmbeddedFunctions(document, "compiler"))
}

func validateCompilerBindingsIndexed(document Document, arch Declaration, functions []embeddedFunction) []Diagnostic {
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
		function, found := indexedEmbeddedFunction(functions, right[0])
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
		if operation.ReachabilityGuard != "" {
			// Prepared adapters retain their query, including target contract facts.
			out = append(out, "\nconst "+operation.ReachabilityGuard+" = true\n"...)
		}
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
	var functions [][]embeddedFunction
	var selectors []string
	for i := 0; i < len(definitions); i++ {
		definition := definitions[i]
		if !definition.Ok {
			return GenerateResult{Diagnostics: definition.Diagnostics}
		}
		indexed := indexEmbeddedFunctions(definition.Document, "compiler")
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
			if diagnostics := validateCompilerBindingsIndexed(definition.Document, arch, indexed); len(diagnostics) != 0 {
				return GenerateResult{Diagnostics: diagnostics}
			}
			if stringIndex(selectors, selector) >= 0 {
				return GenerateResult{Diagnostics: []Diagnostic{resolveDiagnostic(definition.Document, arch, "RTG-COMPILER-006", "duplicate bundled selector "+selector)}}
			}
			found = true
			architectures = append(architectures, arch)
			documents = append(documents, definition.Document)
			functions = append(functions, indexed)
			selectors = append(selectors, selector)
		}
		if !found {
			return GenerateResult{Diagnostics: []Diagnostic{{Filename: definition.Document.Filename, Code: "RTG-COMPILER-007", Message: "bundled definition has no compiler bindings"}}}
		}
	}
	// Choose a capture-free local once for the entire bundle. Definition bodies
	// and selectors are shared across operations; rescanning them for every
	// operation adds no safety. Include every signature in this shared choice.
	selectorLocal := "renvoCompilerSelector"
	for {
		collision := false
		for i := 0; i < len(compilerEmitterOperations); i++ {
			if strings.Contains(compilerEmitterOperations[i].signature(), selectorLocal) {
				collision = true
			}
		}
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
		context := operation.receiver().Name
		if operation.receiver().Type != "*renvoCompileContext" {
			context += ".c"
		}
		out = append(out, selectorLocal+" := "+context+"\nrenvoNonNil("+selectorLocal+")\n"...)
		// Share only byte-identical emitted bodies. Every selector remains explicit,
		// so this neither invents an ISA family nor supplies an unknown-target default.
		var bodies []string
		var conditions []string
		for j := 0; j < len(architectures); j++ {
			hook := compilerBindingHook(architectures[j], operation.Name)
			function, _ := indexedEmbeddedFunction(functions[j], hook)
			project := compilerBindingCanProject(function, operation)
			body := hook + operation.arguments()
			if operation.Result != "" {
				body = "return " + body
			}
			if project {
				body = strings.TrimSpace(string(function.Body))
			}
			// Do not create unreachable consecutive returns: the compact source
			// compiler admits a bare return only at its block termination.
			if operation.Result == "" && (!project || !function.EndsInReturn) {
				body += "\nreturn\n"
			}
			// A read-only query may share its explicit unavailable result with
			// known definitions. Omitting that identical branch preserves both
			// known and unknown selectors without testing common defaults at
			// every call site. Never do this for emission operations: even an
			// unavailable result must still record an unknown-selector failure.
			if operation.receiver().Type == "*renvoCompileContext" &&
				strings.TrimSpace(body) == strings.TrimSpace(operation.failBody()) {
				continue
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
		if operation.ReachabilityGuard != "" {
			// Only the existing exact false-body elimination proves unreachability.
			// Unknown selectors keep the same false result; arbitrary hook bodies
			// remain reachable even if they happen to return false at runtime.
			value := "true"
			if operation.receiver().Type == "*renvoCompileContext" &&
				operation.Result == "bool" && operation.Failure == "false" && len(bodies) == 0 {
				value = "false"
			}
			out = append(out, "\nconst "+operation.ReachabilityGuard+" = "+value+"\n"...)
		}
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
	functions := indexEmbeddedFunctions(document, "compiler")
	if len(functions) == 0 {
		return nil
	}
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
			function, found := indexedEmbeddedFunction(functions, hook)
			if found && compilerBindingCanProject(function, operation) && stringIndex(candidates, hook) < 0 {
				candidates = append(candidates, hook)
			}
			if found && !compilerBindingCanProject(function, operation) && stringIndex(referenced, hook) < 0 {
				referenced = append(referenced, hook)
			}
		}
	}
	// With no projectable hook there is nothing to omit, even if other Go
	// blocks are malformed. This helper is not the definition validator.
	if len(candidates) == 0 {
		return nil
	}
	// Candidate membership is queried for every source token. Keep a sorted
	// local index instead of scanning every operation for punctuation, literals,
	// and unrelated identifiers. Retain the original candidate order below.
	candidateIndex := append([]string(nil), candidates...)
	sortStrings(candidateIndex)
	for i := 0; i < len(document.Declarations); i++ {
		declaration := document.Declarations[i]
		referenced = compilerOtherHookReferences(referenced, candidateIndex, declaration.Statements)
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
			if !compilerHookIndexContains(candidateIndex, name) || stringIndex(referenced, name) >= 0 {
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
	if len(private) == 0 {
		return append(out, source...)
	}
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

// The input is a sorted local membership index, not a declaration-order list.
func compilerHookIndexContains(names []string, name string) bool {
	low := 0
	high := len(names)
	for low < high {
		middle := low + (high-low)/2
		if names[middle] < name {
			low = middle + 1
		} else {
			high = middle
		}
	}
	return low < len(names) && names[low] == name
}

func compilerOtherHookReferences(referenced []string, candidates []string, statements []Statement) []string {
	for i := 0; i < len(statements); i++ {
		statement := statements[i]
		if statementBlockName(statement) == "compiler_bindings" {
			continue
		}
		for j := 0; j < len(statement.Tokens); j++ {
			name := statement.Tokens[j]
			if compilerHookIndexContains(candidates, name) && stringIndex(referenced, name) < 0 {
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
		options[i] = []compilerBodyTail{{tail: strings.TrimSpace(bodies[i])}}
		// Most emitters share only their terminal return. Avoid parsing their
		// bodies when a cheap suffix comparison proves there is no useful tail.
		for j := 0; j < len(bodies); j++ {
			if j != i && compilerBodiesMayShareTail(bodies[i], bodies[j]) {
				options[i] = compilerBodyTails(bodies[i])
				break
			}
		}
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

// This is only a rejection filter: every accepted candidate still passes the
// statement/scope validation above. Keep full shorter bodies and otherwise
// discard the first (possibly partial) matching line of the common suffix.
func compilerBodiesMayShareTail(left string, right string) bool {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	n := 0
	for n < len(left) && n < len(right) && left[len(left)-1-n] == right[len(right)-1-n] {
		n++
	}
	tail := left[len(left)-n:]
	if n < len(left) && n < len(right) {
		newline := strings.Index(tail, "\n")
		if newline < 0 {
			return false
		}
		tail = tail[newline+1:]
	}
	tail = strings.TrimSpace(tail)
	// A common suffix may begin inside matching nested returns, followed by
	// the real shared tail. Only a single-line return proves rejection.
	if strings.Contains(tail, "\n") {
		return true
	}
	return tail != "" && tail != "return" && tail != "return;" &&
		!strings.HasPrefix(tail, "return ") && !strings.HasPrefix(tail, "return\n")
}
