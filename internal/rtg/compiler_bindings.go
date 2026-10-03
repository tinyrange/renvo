package rtg

import "renvo.dev/internal/syntax"

// compilerEmitterOperation is the shared lowering surface migrated out of the
// handwritten kernel. Bundled definitions bind it to compiler integration
// hooks; prepared definitions implement it through direct_emitter_v1 and the
// ABI adapter. It contains semantic roles, never target identities or bytes.
type compilerEmitterOperation struct {
	Name       string
	Suffix     string
	Prepared   string
	Parameters []compilerBindingParameter
}

type compilerBindingParameter struct {
	Name string
	Type string
}

func (op compilerEmitterOperation) signature() string {
	s := "(a *renvoAsm"
	for _, p := range op.Parameters {
		s += ", " + p.Name + " " + p.Type
	}
	return s + ")"
}

func (op compilerEmitterOperation) arguments() string {
	s := "(a"
	for _, p := range op.Parameters {
		s += ", " + p.Name
	}
	return s + ")"
}

func (op compilerEmitterOperation) contract() directEmitterOperation {
	p := []string{"*renvoAsm"}
	for _, parameter := range op.Parameters {
		p = append(p, parameter.Type)
	}
	return directEmitterOperation{Parameters: p}
}

var compilerEmitterOperations = []compilerEmitterOperation{
	{"store_byte_mem_secondary_tertiary", "StoreByteMemSecondaryTertiary", "renvoRTGDirectStoreU8(a,\n\trenvoRTGAsmAddress(renvoRTGSecondary, renvoRTGTertiary, 0, 1),\n\trenvoRTGPrimary)", nil},
	{"inc_tertiary", "IncTertiary", "renvoRTGDirectIncrement(a, renvoRTGTertiary)", nil},
	{"inc_primary", "IncPrimary", "renvoRTGDirectIncrement(a, renvoRTGPrimary)", nil},
	{"ret", "Ret", "renvoRTGDirectReturn(a)", nil},
	{"leave", "Leave", "renvoRTGDirectLeave(a)", nil},
	{"copy_primary_to_call_word0", "CopyPrimaryToCallWord0", "renvoRTGDirectMove(a, renvoRTGCallWord0, renvoRTGPrimary)", nil},
	{"copy_secondary_to_primary", "CopySecondaryToPrimary", "renvoRTGDirectMove(a, renvoRTGPrimary, renvoRTGSecondary)", nil},
	{"copy_primary_to_call_word1", "CopyPrimaryToCallWord1", "renvoRTGDirectMove(a, renvoRTGCallWord1, renvoRTGPrimary)", nil},
	{"add_secondary_tertiary", "AddSecondaryTertiary", "renvoRTGDirectAdd(a, renvoRTGSecondary, renvoRTGTertiary)", nil},
	{"load_byte_primary_index_tertiary", "LoadBytePrimaryIndexTertiary", "renvoRTGDirectLoadU8(a, renvoRTGPrimary,\n\trenvoRTGAsmAddress(renvoRTGPrimary, renvoRTGTertiary, 0, 1))", nil},
	{"store_primary_mem_secondary_tertiary8", "StorePrimaryMemSecondaryTertiary8", "renvoRTGDirectStoreNative(a,\n\trenvoRTGAsmAddress(renvoRTGSecondary, renvoRTGTertiary, 0, 8),\n\trenvoRTGPrimary)", nil},
	{"inc_mem_secondary", "IncMemSecondary", "renvoRTGAsmMemoryIncrement(a, false)", nil},
	{"dec_mem_secondary", "DecMemSecondary", "renvoRTGAsmMemoryIncrement(a, true)", nil},
	{"bool_not_primary", "BoolNotPrimary", "renvoRTGAsmBoolNot(a)", nil},
	{"bitwise_not_primary", "BitwiseNotPrimary", "renvoRTGDirectMoveImmediate(a, renvoRTGScratch, -1)\nrenvoRTGDirectBitXor(a, renvoRTGPrimary, renvoRTGScratch)", nil},
	{"add_primary_tertiary", "AddPrimaryTertiary", "renvoRTGDirectAdd(a, renvoRTGPrimary, renvoRTGTertiary)", nil},
	{"sub_primary_tertiary", "SubPrimaryTertiary", "renvoRTGDirectSubtract(a, renvoRTGPrimary, renvoRTGTertiary)", nil},
	{"pop_primary", "PopPrimary", "renvoRTGAsmPopRegister(a, renvoRTGPrimary)", nil},
	{"pop_secondary", "PopSecondary", "renvoRTGAsmPopRegister(a, renvoRTGSecondary)", nil},
	{"pop_tertiary", "PopTertiary", "renvoRTGAsmPopRegister(a, renvoRTGTertiary)", nil},
	{"copy_primary_to_secondary", "CopyPrimaryToSecondary", "renvoRTGDirectMove(a, renvoRTGSecondary, renvoRTGPrimary)", nil},
	{"copy_primary_to_tertiary", "CopyPrimaryToTertiary", "renvoRTGDirectMove(a, renvoRTGTertiary, renvoRTGPrimary)", nil},
	{"copy_secondary_to_tertiary", "CopySecondaryToTertiary", "renvoRTGDirectMove(a, renvoRTGTertiary, renvoRTGSecondary)", nil},
	{"copy_tertiary_to_primary", "CopyTertiaryToPrimary", "renvoAsmPushTertiary(a); renvoAsmPopPrimary(a)", nil},
	{"push_primary", "PushPrimary", "renvoRTGAsmPushRegister(a, renvoRTGPrimary)", nil},
	{"push_secondary", "PushSecondary", "renvoRTGAsmPushRegister(a, renvoRTGSecondary)", nil},
	{"push_tertiary", "PushTertiary", "renvoRTGAsmPushRegister(a, renvoRTGTertiary)", nil},
	{"push_imm", "PushImm", "\trenvoRTGAsmPushImmediate(a, imm)", []compilerBindingParameter{{"imm", "int"}}},
	{"store_primary_stack", "StorePrimaryStack", "\trenvoRTGAsmStoreFrame(a, offset, renvoRTGPrimary)", []compilerBindingParameter{{"offset", "int"}}},
	{"store_secondary_stack", "StoreSecondaryStack", "\trenvoRTGAsmStoreFrame(a, offset, renvoRTGSecondary)", []compilerBindingParameter{{"offset", "int"}}},
	{"load_primary_stack", "LoadPrimaryStack", "\trenvoRTGAsmLoadFrame(a, renvoRTGPrimary, offset)", []compilerBindingParameter{{"offset", "int"}}},
	{"inc_stack", "IncStack", "\trenvoAsmLoadPrimaryStack(a, offset)\n\trenvoAsmIncPrimary(a)\n\trenvoAsmStorePrimaryStack(a, offset)", []compilerBindingParameter{{"offset", "int"}}},
	{"dec_stack", "DecStack", "\trenvoAsmLoadPrimaryStack(a, offset)\n\trenvoAsmPushImm(a, 1)\n\trenvoAsmPopTertiary(a)\n\trenvoAsmSubPrimaryTertiary(a)\n\trenvoAsmStorePrimaryStack(a, offset)", []compilerBindingParameter{{"offset", "int"}}},
	{"address_primary_stack", "AddressPrimaryStack", "\trenvoRTGAsmAddressFrame(a, renvoRTGPrimary, offset)", []compilerBindingParameter{{"offset", "int"}}},
	{"address_call_word0_stack", "AddressCallWord0Stack", "\trenvoRTGAsmAddressFrame(a, renvoRTGCallWord0, offset)", []compilerBindingParameter{{"offset", "int"}}},
	{"address_call_word1_stack", "AddressCallWord1Stack", "\trenvoRTGAsmAddressFrame(a, renvoRTGCallWord1, offset)", []compilerBindingParameter{{"offset", "int"}}},
	{"load_secondary_stack", "LoadSecondaryStack", "\trenvoRTGAsmLoadFrame(a, renvoRTGSecondary, offset)", []compilerBindingParameter{{"offset", "int"}}},
	{"load_tertiary_stack", "LoadTertiaryStack", "\trenvoRTGAsmLoadFrame(a, renvoRTGTertiary, offset)", []compilerBindingParameter{{"offset", "int"}}},
	{"store_slice_stack", "StoreSliceStack", "renvoAsmStorePrimarySecondaryStack(a, offset, offset-8)\nrenvoRTGAsmStoreFrame(a, offset-16, renvoRTGTertiary)", []compilerBindingParameter{{"offset", "int"}}},
	{"mul_tertiary_imm", "MulTertiaryImm", "\tif imm == 1 {\n\t\treturn\n\t}\n\trenvoRTGDirectMoveImmediate(a, renvoRTGScratch, int64(imm))\n\trenvoRTGDirectMultiply(a, renvoRTGTertiary, renvoRTGScratch)", []compilerBindingParameter{{"imm", "int"}}},
	{"call_label", "CallLabel", "\trenvoRTGDirectCall(a, label)", []compilerBindingParameter{{"label", "int"}}},
	{"jmp_label", "JmpLabel", "\trenvoRTGDirectJump(a, label)", []compilerBindingParameter{{"label", "int"}}},
	{"jz_label", "JzLabel", "\trenvoRTGDirectJumpCondition(a, renvoRTGConditionFromSetcc(0x94), label)", []compilerBindingParameter{{"label", "int"}}},
	{"jnz_label", "JnzLabel", "\trenvoRTGDirectJumpCondition(a, renvoRTGConditionFromSetcc(0x95), label)", []compilerBindingParameter{{"label", "int"}}},
	{"secondary_imm", "SecondaryImm", "\trenvoRTGDirectMoveImmediate(a, renvoRTGSecondary, int64(imm))", []compilerBindingParameter{{"imm", "int"}}},
	{"primary_data_addr", "PrimaryDataAddr", "\trenvoRTGDirectAddress(a, renvoRTGPrimary, renvoRTGAsmDataAddress(dataOff))", []compilerBindingParameter{{"dataOff", "int"}}},
	{"primary_bss_addr", "PrimaryBssAddr", "\trenvoRTGDirectAddress(a, renvoRTGPrimary, renvoRTGAsmBSSAddress(bssOff))", []compilerBindingParameter{{"bssOff", "int"}}},
	{"load_primary_bss", "LoadPrimaryBss", "\trenvoRTGDirectLoadNative(a, renvoRTGPrimary, renvoRTGAsmBSSAddress(bssOff))", []compilerBindingParameter{{"bssOff", "int"}}},
	{"store_primary_bss", "StorePrimaryBss", "\trenvoRTGDirectStoreNative(a, renvoRTGAsmBSSAddress(bssOff), renvoRTGPrimary)", []compilerBindingParameter{{"bssOff", "int"}}},
	{"pop_call_word0", "PopCallWord0", "\trenvoRTGAsmPopRegister(a, renvoRTGCallWord0)", nil},
	{"pop_call_word1", "PopCallWord1", "\trenvoRTGAsmPopRegister(a, renvoRTGCallWord1)", nil},
	{"add_secondary_imm", "AddSecondaryImm", "\trenvoRTGDirectMoveImmediate(a, renvoRTGScratch, int64(imm))\n\trenvoRTGDirectAdd(a, renvoRTGSecondary, renvoRTGScratch)", []compilerBindingParameter{{"imm", "int"}}},
	{"load_qword_primary_index_tertiary_disp", "LoadQwordPrimaryIndexTertiaryDisp", "\trenvoRTGDirectLoadNative(a, renvoRTGPrimary,\n\t\trenvoRTGAsmAddress(renvoRTGPrimary, renvoRTGTertiary, disp, 1))", []compilerBindingParameter{{"disp", "int"}}},
	{"load_primary_mem_secondary_disp", "LoadPrimaryMemSecondaryDisp", "\trenvoRTGDirectLoadNative(a, renvoRTGPrimary,\n\t\trenvoRTGAsmAddress(renvoRTGSecondary, RTGNoRegister, disp, 1))", []compilerBindingParameter{{"disp", "int"}}},
	{"load_primary_mem_secondary_disp_size", "LoadPrimaryMemSecondaryDispSize", "\t// Size-only scalar loads follow the built-in backend contract: bytes are\n\t// zero-extended, while wider narrow integers are sign-extended before\n\t// typed expression lowering applies any unsigned normalization.\n\trenvoRTGAsmLoadSize(a, renvoRTGPrimary,\n\t\trenvoRTGAsmAddress(renvoRTGSecondary, RTGNoRegister, disp, 1),\n\t\tsize, size != 1)", []compilerBindingParameter{{"disp", "int"}, {"size", "int"}}},
	{"load_primary_index_tertiary_size", "LoadPrimaryIndexTertiarySize", "\t// Match renvoAsmLoadPrimaryMemSecondaryDispSize's scalar-load contract.\n\trenvoRTGAsmLoadSize(a, renvoRTGPrimary,\n\t\trenvoRTGAsmAddress(renvoRTGPrimary, renvoRTGTertiary, 0, size),\n\t\tsize, size != 1)", []compilerBindingParameter{{"size", "int"}}},
	{"store_primary_mem_secondary_disp", "StorePrimaryMemSecondaryDisp", "\trenvoRTGDirectStoreNative(a,\n\t\trenvoRTGAsmAddress(renvoRTGSecondary, RTGNoRegister, disp, 1),\n\t\trenvoRTGPrimary)", []compilerBindingParameter{{"disp", "int"}}},
	{"store_primary_mem_secondary_disp_size", "StorePrimaryMemSecondaryDispSize", "\trenvoRTGAsmStoreSize(a,\n\t\trenvoRTGAsmAddress(renvoRTGSecondary, RTGNoRegister, disp, 1),\n\t\trenvoRTGPrimary, size)", []compilerBindingParameter{{"disp", "int"}, {"size", "int"}}},
	{"normalize_primary_for_kind", "NormalizePrimaryForKind", "\trenvoRTGAsmNormalize(a, kind)", []compilerBindingParameter{{"kind", "int"}}},
	{"cmp_primary_imm8", "CmpPrimaryImm8", "\trenvoRTGAsmCompareImmediate(a, imm)", []compilerBindingParameter{{"imm", "int"}}},
	{"cmp_primary_imm8_discard", "CmpPrimaryImm8Discard", "\trenvoAsmCmpPrimaryImm8(a, imm)", []compilerBindingParameter{{"imm", "int"}}},
	{"shl_tertiary_imm", "ShlTertiaryImm", "\trenvoRTGDirectShiftLeftImmediate(a, renvoRTGTertiary, byte(imm))", []compilerBindingParameter{{"imm", "int"}}},
	{"shl_primary_imm", "ShlPrimaryImm", "\trenvoRTGDirectShiftLeftImmediate(a, renvoRTGPrimary, byte(imm))", []compilerBindingParameter{{"imm", "int"}}},
	{"sar_primary_imm", "SarPrimaryImm", "\trenvoRTGDirectShiftRightSignedImmediate(a, renvoRTGPrimary, byte(imm))", []compilerBindingParameter{{"imm", "int"}}},
	{"div_left_tertiary_right_primary", "DivLeftTertiaryRightPrimary", "\tif mod {\n\t\t// Keep the divisor while producing the quotient, then construct the\n\t\t// remainder as dividend - quotient*divisor.  This keeps prepared\n\t\t// targets on the common RTG register contract and does not depend on\n\t\t// a target-specific fused multiply/subtract operand order.\n\t\trenvoRTGDirectMove(a, renvoRTGScratch, renvoRTGPrimary)\n\t\trenvoRTGDirectSignedDivide(a, false)\n\t\trenvoRTGDirectMultiply(a, renvoRTGPrimary, renvoRTGScratch)\n\t\trenvoRTGDirectSubtract(a, renvoRTGTertiary, renvoRTGPrimary)\n\t\trenvoRTGDirectMove(a, renvoRTGPrimary, renvoRTGTertiary)\n\t} else {\n\t\trenvoRTGDirectSignedDivide(a, false)\n\t}", []compilerBindingParameter{{"mod", "bool"}}},
	{"cmp_tertiary_primary_set", "CmpTertiaryPrimarySet", "\trenvoRTGDirectCompare(a, renvoRTGTertiary, renvoRTGPrimary)\n\trenvoRTGDirectSetCondition(a, renvoRTGConditionFromSetcc(setcc), renvoRTGPrimary)", []compilerBindingParameter{{"setcc", "int"}}},
	{"store_primary_mem_secondary_tertiary_size", "StorePrimaryMemSecondaryTertiarySize", "\trenvoRTGAsmStoreSize(a,\n\t\trenvoRTGAsmAddress(renvoRTGSecondary, renvoRTGTertiary, 0, 1),\n\t\trenvoRTGPrimary, size)", []compilerBindingParameter{{"size", "int"}}},
	{"cmp_tertiary_primary_jump", "CmpTertiaryPrimaryJump", "\trenvoRTGDirectCompare(a, renvoRTGTertiary, renvoRTGPrimary)\n\trenvoRTGDirectJumpCondition(a, renvoRTGConditionFromSetcc(setcc), label)", []compilerBindingParameter{{"setcc", "int"}, {"label", "int"}}},
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
		out = append(out, "\nfunc renvoAsm"...)
		out = append(out, operation.Suffix...)
		out = append(out, operation.signature()...)
		out = append(out, " {\nrenvoNonNil(a)\n"...)
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
		out = append(out, "\nfunc renvoAsm"...)
		out = append(out, operation.Suffix...)
		out = append(out, operation.signature()...)
		out = append(out, " {\nrenvoNonNil(a)\n"...)
		for j := 0; j < len(architectures); j++ {
			out = append(out, "if a.c.renvoTargetArch == "...)
			out = append(out, selectors[j]...)
			out = append(out, " {\n"...)
			hook := compilerBindingHook(architectures[j], operation.Name)
			function, _ := findEmbeddedFunctionKind(documents[j], hook, "compiler")
			if compilerBindingCanProject(function, operation) {
				out = append(out, function.Body...)
			} else {
				out = append(out, hook...)
				out = append(out, operation.arguments()...)
			}
			out = append(out, "\nreturn\n}\n"...)
		}
		out = append(out, "a.patchFailed = true\n}\n"...)
	}
	return GenerateResult{Source: out, Ok: true}
}

// Project definition-owned bodies into their selected branch rather than add a
// second call at every emission site. Returns still leave the dispatch function.
// Noncanonical parameter names and function-scoped labels keep the call path;
// neither requires token substitution or changes the admitted hook contract.
func compilerBindingCanProject(function embeddedFunction, operation compilerEmitterOperation) bool {
	if function.HasLabels || !directEmitterSignatureMatches(function, operation.contract()) ||
		function.Parameters[0].Name != "a" {
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
