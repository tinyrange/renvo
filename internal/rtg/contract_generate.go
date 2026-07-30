package rtg

type runtimeOperationTemplate struct {
	name string
	code int
	args []string
}

// appendDirectEmitterBindings emits one ordinary direct wrapper for every
// semantic operation bound by an architecture. The wrappers retain no
// contract table or dynamic lookup. Their signatures are validated before
// generation, so the shared compiler can call the same typed surface for every
// architecture.
func appendDirectEmitterBindings(out []byte, document Document, arch Declaration, nativeEmitter bool, exposeExports bool) []byte {
	bindings := architectureBindings(arch)
	if len(bindings) == 0 {
		return out
	}
	prefix := "rtg" + exportedName(document.Unit)
	exports := architectureExports(arch)
	for operation := 0; operation < len(directEmitterV1); operation++ {
		var binding architectureBinding
		found := false
		for i := 0; i < len(bindings); i++ {
			if bindings[i].Name == directEmitterV1[operation].Name {
				binding = bindings[i]
				found = true
				break
			}
		}
		if !found {
			continue
		}
		function, found := directEmitterBindingFunction(document, arch, binding)
		if !found {
			continue
		}
		out = append(out, "\n// Generated direct_emitter_v1 binding for "...)
		out = append(out, binding.Name...)
		out = append(out, ".\n"...)
		out = appendDirectEmitterEffectComment(out, directEmitterV1[operation])
		out = append(out, "func "...)
		out = append(out, prefix...)
		out = append(out, "Direct"...)
		out = append(out, operationGoSuffix(binding.Name)...)
		out = append(out, '(')
		for i := 0; i < len(function.Parameters); i++ {
			if i != 0 {
				out = append(out, ", "...)
			}
			if nativeEmitter {
				out = appendRewrittenGoMode(out, function.Parameters[i].Source,
					embeddedGoNames(document), prefix, true, &document)
			} else {
				out = append(out, function.Parameters[i].Source...)
			}
		}
		out = append(out, ')')
		if len(function.Result) != 0 {
			out = append(out, ' ')
			if nativeEmitter {
				out = appendRewrittenGoMode(out, function.Result,
					embeddedGoNames(document), prefix, true, &document)
			} else {
				out = append(out, function.Result...)
			}
		}
		out = append(out, " {\n\t"...)
		if function.HasResult {
			out = append(out, "return "...)
		}
		if exposeExports {
			out = appendDirectEmitterTarget(out, document, arch, binding, prefix, exports)
		} else {
			out = appendDirectEmitterTarget(out, document, arch, binding, prefix, nil)
		}
		out = append(out, '(')
		for i := 0; i < len(function.Parameters); i++ {
			if i != 0 {
				out = append(out, ", "...)
			}
			out = append(out, function.Parameters[i].Name...)
		}
		out = append(out, ")\n}\n"...)
	}
	return out
}

func appendDirectEmitterEffectComment(out []byte, operation directEmitterOperation) []byte {
	out = append(out, "// Effects: reads="...)
	out = appendContractList(out, operation.Reads)
	out = append(out, "; writes="...)
	out = appendContractList(out, operation.Writes)
	out = append(out, "; clobbers="...)
	out = appendContractList(out, operation.Clobbers)
	out = append(out, "; preserves="...)
	out = appendContractList(out, operation.Preserves)
	if operation.Memory != "" {
		out = append(out, "; memory="...)
		out = append(out, operation.Memory...)
	}
	out = append(out, "; control="...)
	out = append(out, operation.Control...)
	if operation.Relocation != "" {
		out = append(out, "; relocation="...)
		out = append(out, operation.Relocation...)
	}
	if len(operation.Capabilities) != 0 {
		out = append(out, "; requires="...)
		out = appendContractList(out, operation.Capabilities)
	}
	if operation.Rejectable {
		out = append(out, "; rejection=permitted"...)
	}
	return append(out, '\n')
}

func appendContractList(out []byte, values []string) []byte {
	if len(values) == 0 {
		return append(out, '-')
	}
	for i := 0; i < len(values); i++ {
		if i != 0 {
			out = append(out, ',')
		}
		out = append(out, values[i]...)
	}
	return out
}

func appendDirectEmitterTarget(out []byte, document Document, arch Declaration,
	binding architectureBinding, prefix string, exports []embeddedExport) []byte {
	if binding.Instruction != "" {
		out = append(out, prefix...)
		out = append(out, instructionGoSuffix(binding.Instruction)...)
		return out
	}
	if external, found := embeddedExternalName(exports, binding.Algorithm); found {
		return append(out, external...)
	}
	out = append(out, prefix...)
	out = append(out, exportedName(binding.Algorithm)...)
	return out
}

func operationGoSuffix(name string) string {
	normalized := make([]byte, len(name))
	for i := 0; i < len(name); i++ {
		if name[i] == '.' {
			normalized[i] = '_'
		} else {
			normalized[i] = name[i]
		}
	}
	return instructionGoSuffix(string(normalized))
}

func appendArchitectureHooks(out []byte, document Document, arch Declaration, nativeEmitter bool) []byte {
	algorithm, found := architectureGoHook(arch, "patch_relocations")
	if !found {
		return out
	}
	prefix := "rtg" + exportedName(document.Unit)
	out = append(out, "\n// Generated definition-owned relocation finalizer.\nfunc "...)
	out = append(out, prefix...)
	out = append(out, "PatchRelocations(out "...)
	if nativeEmitter {
		out = append(out, "*renvoAsm"...)
	} else {
		out = append(out, "*RTGEmitter"...)
	}
	out = append(out, ") {\n\t"...)
	if external, found := embeddedExternalName(architectureExports(arch), algorithm); found {
		out = append(out, external...)
	} else {
		out = append(out, prefix...)
		out = append(out, exportedName(algorithm)...)
	}
	out = append(out, "(out)\n}\n"...)
	return out
}

// Prepared backends expose one stable, target-neutral adapter surface to the
// shared lowering kernel. The adapter is generated once for the selected
// architecture and every call is a direct Go call to that definition's
// specialized binding.
func appendPreparedDirectEmitterAdapters(out []byte, document Document, target ResolvedTarget) []byte {
	prefix := "rtg" + exportedName(document.Unit)
	bindings := architectureBindings(target.Arch)
	out = append(out, "\nvar renvoRTGUnsupportedOperation bool\n"...)
	for operation := 0; operation < len(directEmitterV1); operation++ {
		found := false
		for i := 0; i < len(bindings); i++ {
			if bindings[i].Name == directEmitterV1[operation].Name {
				found = true
				break
			}
		}
		out = appendDirectEmitterAdapterFunction(out, directEmitterV1[operation], prefix, found)
	}
	out = appendPreparedLocationAdapters(out, document, target)
	out = append(out, "\nfunc renvoRTGPatchRelocations(out *renvoAsm) {\n"...)
	if algorithm, found := architectureGoHook(target.Arch, "patch_relocations"); found {
		out = append(out, prefix...)
		out = append(out, exportedName(algorithm)...)
		out = append(out, "(out)\n"...)
	}
	out = append(out, "}\n"...)
	return out
}

func appendDirectEmitterKernelAdapters(out []byte) []byte {
	out = append(out, "\nvar renvoRTGUnsupportedOperation bool\n"...)
	for i := 0; i < len(directEmitterV1); i++ {
		out = appendDirectEmitterAdapterFunction(out, directEmitterV1[i], "", false)
	}
	out = append(out, `
var renvoRTGPrimary = RTGNoRegister
var renvoRTGSecondary = RTGNoRegister
var renvoRTGTertiary = RTGNoRegister
var renvoRTGScratch = RTGNoRegister
var renvoRTGCopyDestination = RTGNoRegister
var renvoRTGCopySource = RTGNoRegister
var renvoRTGCopyCount = RTGNoRegister
var renvoRTGStack = RTGNoRegister
var renvoRTGFrame = RTGNoRegister
var renvoRTGCallWord0 = RTGNoRegister
var renvoRTGCallWord1 = RTGNoRegister
var renvoRTGCallWord2 = RTGNoRegister
var renvoRTGCallWord3 = RTGNoRegister
var renvoRTGCallWord4 = RTGNoRegister
var renvoRTGCallWord5 = RTGNoRegister
var renvoRTGSyscallNumber = RTGNoRegister
var renvoRTGSyscallWord0 = RTGNoRegister
var renvoRTGSyscallWord1 = RTGNoRegister
var renvoRTGSyscallWord2 = RTGNoRegister
var renvoRTGSyscallWord3 = RTGNoRegister
var renvoRTGSyscallWord4 = RTGNoRegister
var renvoRTGSyscallWord5 = RTGNoRegister
var renvoRTGSyscallResult = RTGNoRegister
const renvoRTGStackWordBytes = 0
func renvoRTGConditionFromSetcc(setcc int) RTGCondition { return RTGCondition{} }
func renvoRTGPatchRelocations(out *renvoAsm) {}
func renvoRTGFrameStart(out *renvoAsm) int { return -1 }
func renvoRTGFrameFinish(out *renvoAsm, framePatch int, stackUsed int) {}
func renvoRTGABIPushRegister(out *renvoAsm, source RTGRegister) bool { return false }
func renvoRTGABIPushImmediate(out *renvoAsm, value int) bool { return false }
func renvoRTGABIPopRegister(out *renvoAsm, destination RTGRegister) bool { return false }
func renvoRTGABILoadFrame(out *renvoAsm, destination RTGRegister, offset int) bool { return false }
func renvoRTGABIStoreFrame(out *renvoAsm, offset int, source RTGRegister) bool { return false }
func renvoRTGABIAddressFrame(out *renvoAsm, destination RTGRegister, offset int) bool { return false }
func renvoRTGABIStoreParamWord(out *renvoAsm, word int, offset int) bool { return false }
func renvoRTGABICallWordCount(out *renvoAsm, label int, wordCount int) bool { return false }
const renvoRTGCodeOffset = 0
func renvoRTGImage(out *renvoAsm) []byte { return nil }
func renvoRTGKernelImage(out *renvoAsm, initLabel int, exitLabel int) []byte { return nil }
const renvoRTGEntryStateBytes = 0
func renvoRTGEmitEntryStart(out *renvoAsm, stateOffset int) bool { return true }
func renvoRTGEmitEntry(out *renvoAsm, parameterCount int, stateOffset int) bool { return parameterCount == 0 }
func renvoRTGKernelEntryPrologue(out *renvoAsm) {}
func renvoRTGKernelEntryEpilogue(out *renvoAsm) {}
func renvoRTGKernelCallbackAddress(out *renvoAsm, label int) {}
func renvoRTGEmitExit(out *renvoAsm, status RTGRegister) bool { return false }
func renvoRTGEmitStaticCall(out *renvoAsm, importID int, wordCount int) bool { return false }
func renvoRTGEmitRuntimeOperation(out *renvoAsm, operation int) bool { return false }
`...)
	return out
}

func appendPreparedABIAdapters(out []byte, document Document, target ResolvedTarget) []byte {
	prefix := "rtg" + exportedName(document.Unit)
	start, hasStart := targetABIGoHook(document, target.ABI, "frame_start")
	finish, hasFinish := targetABIGoHook(document, target.ABI, "frame_finish")
	out = append(out, "\nfunc renvoRTGFrameStart(out *renvoAsm) int {\n"...)
	if hasStart {
		out = append(out, "return "...)
		out = append(out, prefix...)
		out = append(out, exportedName(start)...)
		out = append(out, "(out)\n"...)
	} else {
		out = append(out, "return -1\n"...)
	}
	out = append(out, "}\nfunc renvoRTGFrameFinish(out *renvoAsm, framePatch int, stackUsed int) {\n"...)
	if hasFinish {
		out = append(out, prefix...)
		out = append(out, exportedName(finish)...)
		out = append(out, "(out, framePatch, stackUsed)\n"...)
	}
	out = append(out, "}\n"...)
	out = appendPreparedABIHook(out, document, target.ABI, prefix,
		"renvoRTGABIPushRegister", "push_register",
		"out *renvoAsm, source RTGRegister", "out, source")
	out = appendPreparedABIHook(out, document, target.ABI, prefix,
		"renvoRTGABIPushImmediate", "push_immediate",
		"out *renvoAsm, value int", "out, value")
	out = appendPreparedABIHook(out, document, target.ABI, prefix,
		"renvoRTGABIPopRegister", "pop_register",
		"out *renvoAsm, destination RTGRegister", "out, destination")
	out = appendPreparedABIHook(out, document, target.ABI, prefix,
		"renvoRTGABILoadFrame", "frame_load",
		"out *renvoAsm, destination RTGRegister, offset int", "out, destination, offset")
	out = appendPreparedABIHook(out, document, target.ABI, prefix,
		"renvoRTGABIStoreFrame", "frame_store",
		"out *renvoAsm, offset int, source RTGRegister", "out, offset, source")
	out = appendPreparedABIHook(out, document, target.ABI, prefix,
		"renvoRTGABIAddressFrame", "frame_address",
		"out *renvoAsm, destination RTGRegister, offset int", "out, destination, offset")
	out = appendPreparedABIHook(out, document, target.ABI, prefix,
		"renvoRTGABIStoreParamWord", "store_param_word",
		"out *renvoAsm, word int, offset int", "out, word, offset")
	out = appendPreparedABIHook(out, document, target.ABI, prefix,
		"renvoRTGABICallWordCount", "call_word_count",
		"out *renvoAsm, label int, wordCount int", "out, label, wordCount")
	return out
}

func appendPreparedABIHook(out []byte, document Document, abi Declaration, prefix string,
	wrapper string, field string, parameters string, arguments string) []byte {
	algorithm, found := targetABIGoHook(document, abi, field)
	out = append(out, "func "...)
	out = append(out, wrapper...)
	out = append(out, '(')
	out = append(out, parameters...)
	out = append(out, ") bool {\n"...)
	if found {
		out = append(out, prefix...)
		out = append(out, exportedName(algorithm)...)
		out = append(out, '(')
		out = append(out, arguments...)
		out = append(out, ")\nreturn true\n"...)
	}
	out = append(out, "return false\n}\n"...)
	return out
}

func appendPreparedFormatAdapters(out []byte, document Document, target ResolvedTarget) []byte {
	outputFormat := target.Executable
	if outputFormat.Name == "" {
		outputFormat = target.Object
	}
	codeOffset, ok := integerField(document, outputFormat, "code_offset")
	if !ok {
		codeOffset = 0
	}
	out = append(out, "\nconst renvoRTGCodeOffset = "...)
	out = appendDecimalFrame(out, codeOffset)
	out = append(out, "\nfunc renvoRTGImage(out *renvoAsm) []byte {\n"...)
	algorithm, hasImage := architectureGoHook(outputFormat, "image")
	if hasImage {
		out = append(out, "\treturn rtg"...)
		out = append(out, exportedName(document.Unit)...)
		out = append(out, exportedName(algorithm)...)
		out = append(out, "(out)\n"...)
	} else {
		out = append(out, "\treturn nil\n"...)
	}
	out = append(out, "}\n"...)
	out = append(out, "func renvoRTGKernelImage(out *renvoAsm, initLabel int, exitLabel int) []byte {\n"...)
	if algorithm, hasAlgorithm := architectureGoHook(outputFormat, "kernel_image"); hasAlgorithm {
		out = append(out, "\treturn rtg"...)
		out = append(out, exportedName(document.Unit)...)
		out = append(out, exportedName(algorithm)...)
		out = append(out, "(out, initLabel, exitLabel)\n"...)
	} else {
		out = append(out, "\treturn nil\n"...)
	}
	out = append(out, "}\n"...)

	entryStateBytes, hasEntryStateBytes := integerField(document, target.Runtime, "entry_state_bytes")
	if !hasEntryStateBytes {
		entryStateBytes = 0
	}
	out = append(out, "const renvoRTGEntryStateBytes = "...)
	out = appendDecimalFrame(out, entryStateBytes)
	out = append(out, "\nfunc renvoRTGEmitEntryStart(out *renvoAsm, stateOffset int) bool {\n"...)
	if algorithm, hasAlgorithm := architectureGoHook(target.Runtime, "emit_entry_start"); hasAlgorithm {
		out = append(out, "\treturn rtg"...)
		out = append(out, exportedName(document.Unit)...)
		out = append(out, exportedName(algorithm)...)
		out = append(out, "(out, stateOffset)\n"...)
	} else {
		out = append(out, "\treturn true\n"...)
	}
	out = append(out, "}\nfunc renvoRTGEmitEntry(out *renvoAsm, parameterCount int, stateOffset int) bool {\n"...)
	if algorithm, hasAlgorithm := architectureGoHook(target.Runtime, "emit_entry"); hasAlgorithm {
		out = append(out, "\treturn rtg"...)
		out = append(out, exportedName(document.Unit)...)
		out = append(out, exportedName(algorithm)...)
		out = append(out, "(out, parameterCount, stateOffset)\n"...)
	} else {
		out = append(out, "\treturn parameterCount == 0\n"...)
	}
	out = append(out, "}\n"...)
	out = appendPreparedRuntimeVoidHook(out, document, target.Runtime,
		"renvoRTGKernelEntryPrologue", "entry_prologue", "out *renvoAsm", "out")
	out = appendPreparedRuntimeVoidHook(out, document, target.Runtime,
		"renvoRTGKernelEntryEpilogue", "entry_epilogue", "out *renvoAsm", "out")
	out = appendPreparedRuntimeVoidHook(out, document, target.Runtime,
		"renvoRTGKernelCallbackAddress", "emit_callback_address",
		"out *renvoAsm, label int", "out, label")

	exitNumber, hasExit := runtimeOperationInteger(target.Runtime, "exit", "number")
	syscallNumber := targetRuntimeRegisterList(target.Runtime, "number")
	syscallWords := targetRuntimeRegisterList(target.Runtime, "arguments")
	out = append(out, "func renvoRTGEmitExit(out *renvoAsm, status RTGRegister) bool {\n"...)
	if algorithm, hasAlgorithm := architectureGoHook(target.Runtime, "emit_exit"); hasAlgorithm {
		out = append(out, "\trtg"...)
		out = append(out, exportedName(document.Unit)...)
		out = append(out, exportedName(algorithm)...)
		out = append(out, "(out, status)\n\treturn true\n"...)
	} else if hasExit && len(syscallNumber) != 0 && len(syscallWords) != 0 {
		numberRegister, numberOK := architectureRegisterOutput(document, target.Arch, syscallNumber[0])
		statusRegister, statusOK := architectureRegisterOutput(document, target.Arch, syscallWords[0])
		if numberOK && statusOK {
			out = append(out, "\trenvoRTGDirectMove(out, "...)
			out = append(out, statusRegister...)
			out = append(out, ", status)\n\trenvoRTGDirectMoveImmediate(out, "...)
			out = append(out, numberRegister...)
			out = append(out, ", "...)
			out = appendDecimalFrame(out, exitNumber)
			out = append(out, ")\n\trenvoRTGDirectHostSyscall(out)\n\treturn true\n"...)
		}
	}
	out = append(out, "\treturn false\n}\n"...)
	out = append(out, "func renvoRTGEmitStaticCall(out *renvoAsm, importID int, wordCount int) bool {\n"...)
	if algorithm, hasAlgorithm := architectureGoHook(target.Runtime, "emit_static_call"); hasAlgorithm {
		out = append(out, "\trtg"...)
		out = append(out, exportedName(document.Unit)...)
		out = append(out, exportedName(algorithm)...)
		out = append(out, "(out, importID, wordCount)\n\treturn true\n"...)
	}
	out = append(out, "\treturn false\n}\n"...)
	out = appendPreparedRuntimeOperationAdapter(out, document, target)
	return out
}

func appendPreparedRuntimeVoidHook(
	out []byte, document Document, runtime Declaration,
	wrapper string, field string, parameters string, arguments string,
) []byte {
	out = append(out, "func "...)
	out = append(out, wrapper...)
	out = append(out, '(')
	out = append(out, parameters...)
	out = append(out, ") {\n"...)
	if algorithm, found := architectureGoHook(runtime, field); found {
		out = append(out, "\trtg"...)
		out = append(out, exportedName(document.Unit)...)
		out = append(out, exportedName(algorithm)...)
		out = append(out, '(')
		out = append(out, arguments...)
		out = append(out, ")\n"...)
	}
	out = append(out, "}\n"...)
	return out
}

func appendPreparedRuntimeOperationAdapter(
	out []byte, document Document, target ResolvedTarget,
) []byte {
	out = append(out, "func renvoRTGEmitRuntimeOperation(out *renvoAsm, operation int) bool {\n"...)
	if algorithm, found := architectureGoHook(target.Runtime, "emit_operation"); found {
		out = append(out, "\treturn rtg"...)
		out = append(out, exportedName(document.Unit)...)
		out = append(out, exportedName(algorithm)...)
		out = append(out, "(out, operation)\n}\n"...)
		return out
	}
	numberRegisters := targetRuntimeRegisterList(target.Runtime, "number")
	argumentRegisters := targetRuntimeRegisterList(target.Runtime, "arguments")
	resultRegisters := targetRuntimeRegisterList(target.Runtime, "result")
	if len(numberRegisters) == 0 || len(argumentRegisters) == 0 {
		out = append(out, "\treturn false\n}\n"...)
		return out
	}
	numberRegister, numberOK := architectureRegisterOutput(
		document, target.Arch, numberRegisters[0])
	resultRegister := ""
	resultOK := false
	if len(resultRegisters) != 0 {
		resultRegister, resultOK = architectureRegisterOutput(
			document, target.Arch, resultRegisters[0])
	}
	operations := []runtimeOperationTemplate{
		{"read", 1, []string{"fd", "buffer", "count"}},
		{"write", 2, []string{"fd", "buffer", "count"}},
		{"read_at", 3, []string{"fd", "buffer", "count", "offset"}},
		{"write_at", 4, []string{"fd", "buffer", "count", "offset"}},
		{"open", 5, []string{"path", "flags", "mode"}},
		{"close", 6, []string{"fd"}},
		{"chmod", 7, []string{"fd", "mode"}},
	}
	callWords := targetABICallWords(document, target.ABI)
	for i := 0; i < len(operations); i++ {
		number, found := runtimeOperationInteger(
			target.Runtime, operations[i].name, "number")
		arguments := runtimeOperationList(
			target.Runtime, operations[i].name, "args")
		if !found || !numberOK || len(arguments) > len(argumentRegisters) {
			continue
		}
		out = append(out, "\tif operation == "...)
		out = appendDecimalFrame(out, operations[i].code)
		out = append(out, " {\n"...)
		valid := true
		sources := make([]string, len(arguments))
		destinations := make([]string, len(arguments))
		sourceIndices := make([]int, len(arguments))
		fixed := make([]bool, len(arguments))
		for j := 0; j < len(arguments); j++ {
			sourceIndices[j] = stringIndex(operations[i].args, arguments[j])
			if sourceIndices[j] < 0 {
				if operations[i].name == "open" && arguments[j] == "dirfd" {
					fixed[j] = true
				} else {
					valid = false
					break
				}
			} else if sourceIndices[j] >= len(callWords) {
				valid = false
				break
			} else {
				sources[j], valid = architectureRegisterOutput(
					document, target.Arch, callWords[sourceIndices[j]])
				if !valid {
					break
				}
			}
			destinations[j], valid = architectureRegisterOutput(
				document, target.Arch, argumentRegisters[j])
			if !valid {
				break
			}
		}
		if valid {
			for j := 0; j < len(sources); j++ {
				if fixed[j] {
					continue
				}
				out = append(out, "\t\trenvoRTGAsmPushRegister(out, "...)
				out = append(out, sources[j]...)
				out = append(out, ")\n"...)
			}
			for j := len(destinations) - 1; j >= 0; j-- {
				if fixed[j] {
					continue
				}
				out = append(out, "\t\trenvoRTGAsmPopRegister(out, "...)
				out = append(out, destinations[j]...)
				out = append(out, ")\n"...)
			}
			for j := 0; j < len(destinations); j++ {
				if !fixed[j] {
					continue
				}
				out = append(out, "\t\trenvoRTGDirectMoveImmediate(out, "...)
				out = append(out, destinations[j]...)
				out = append(out, ", -100)\n"...)
			}
			out = append(out, "\t\trenvoRTGDirectMoveImmediate(out, "...)
			out = append(out, numberRegister...)
			out = append(out, ", "...)
			out = appendDecimalFrame(out, number)
			out = append(out, ")\n\t\trenvoRTGDirectHostSyscall(out)\n"...)
			if resultOK {
				out = append(out, "\t\trenvoRTGDirectMove(out, renvoRTGPrimary, "...)
				out = append(out, resultRegister...)
				out = append(out, ")\n"...)
			}
			out = append(out, "\t\treturn true\n"...)
		}
		out = append(out, "\t}\n"...)
	}
	out = append(out, "\treturn false\n}\n"...)
	return out
}

func runtimeOperationInteger(runtime Declaration, operation string, field string) (int, bool) {
	for i := 0; i < len(runtime.Statements); i++ {
		statement := runtime.Statements[i]
		if len(statement.Tokens) < 2 || statement.Tokens[0] != "operation" ||
			statement.Tokens[1] != operation {
			continue
		}
		for j := 0; j < len(statement.Children); j++ {
			tokens := statement.Children[j].Tokens
			for at := 0; at+2 < len(tokens); at++ {
				if tokens[at] == field && tokens[at+1] == "=" {
					return parseInteger(tokens[at+2])
				}
			}
		}
	}
	return 0, false
}

func runtimeOperationList(
	runtime Declaration, operation string, field string,
) []string {
	for i := 0; i < len(runtime.Statements); i++ {
		statement := runtime.Statements[i]
		if len(statement.Tokens) < 2 || statement.Tokens[0] != "operation" ||
			statement.Tokens[1] != operation {
			continue
		}
		for j := 0; j < len(statement.Children); j++ {
			left, _, assignment := statementAssignment(statement.Children[j])
			if assignment && len(left) == 1 && left[0] == field {
				return statementListValues(statement.Children[j])
			}
		}
	}
	return nil
}

func targetABIGoHook(document Document, abi Declaration, name string) (string, bool) {
	if algorithm, ok := architectureGoHook(abi, name); ok {
		return algorithm, true
	}
	internal, ok := fieldValue(document, abi, "internal")
	if !ok {
		return "", false
	}
	base, ok := document.Declaration(DeclABI, internal)
	if !ok {
		return "", false
	}
	return targetABIGoHook(document, base, name)
}

func appendDirectEmitterAdapterFunction(out []byte, operation directEmitterOperation, prefix string, active bool) []byte {
	out = append(out, "\nfunc renvoRTGDirect"...)
	out = append(out, operationGoSuffix(operation.Name)...)
	out = append(out, '(')
	var used []string
	out, used = appendDirectEmitterAdapterParameters(out, operation)
	out = append(out, ") {\n"...)
	if active {
		out = append(out, prefix...)
		out = append(out, "Direct"...)
		out = append(out, operationGoSuffix(operation.Name)...)
		out = append(out, '(')
		for i := 0; i < len(used); i++ {
			if i != 0 {
				out = append(out, ", "...)
			}
			out = append(out, used[i]...)
		}
		out = append(out, ")\n"...)
	} else {
		out = append(out, "\trenvoRTGUnsupportedOperation = true\n"...)
	}
	out = append(out, "}\n"...)
	return out
}

func appendDirectEmitterAdapterParameters(out []byte, operation directEmitterOperation) ([]byte, []string) {
	names := []string{"out", "destination", "source", "address", "operand", "value", "label", "condition", "direction", "signed", "remainder"}
	used := make([]string, 0, len(operation.Parameters))
	for i := 0; i < len(operation.Parameters); i++ {
		if i != 0 {
			out = append(out, ", "...)
		}
		name := names[i]
		if i == 0 {
			name = "out"
		} else {
			// Pick a readable stable name by type and position.
			typ := operation.Parameters[i]
			if typ == "RTGAddress" {
				name = "address"
			} else if typ == "RTGLabel" {
				name = "label"
			} else if typ == "RTGCondition" {
				name = "condition"
			} else if typ == "RTGShiftDirection" {
				name = "direction"
			} else if typ == "bool" {
				if operation.Name == "signed_divide" {
					name = "remainder"
				} else {
					name = "signed"
				}
			} else if typ == "byte" || typ == "int64" {
				name = "value"
			} else if typ == "RTGRegister" {
				if i == 1 && len(operation.Parameters) > 2 {
					name = "destination"
				} else if i == 2 {
					name = "source"
				} else {
					name = "operand"
				}
			}
		}
		used = append(used, name)
		out = append(out, name...)
		out = append(out, ' ')
		out = append(out, nativeDirectEmitterType(operation.Parameters[i])...)
	}
	return out, used
}

func nativeDirectEmitterType(typ string) string {
	if typ == "*RTGEmitter" {
		return "*renvoAsm"
	}
	if typ == "RTGAddress" {
		return "renvoRTGAddress"
	}
	if typ == "RTGLabel" {
		return "int"
	}
	return typ
}

func appendPreparedLocationAdapters(out []byte, document Document, target ResolvedTarget) []byte {
	locations := []string{
		"primary", "secondary", "tertiary", "scratch", "copy_destination",
		"copy_source", "copy_count", "stack", "frame",
	}
	for i := 0; i < len(locations); i++ {
		out = append(out, "\nvar renvoRTG"...)
		out = append(out, exportedName(locations[i])...)
		out = append(out, " = "...)
		if symbol, ok := architectureLocationOutput(document, target.Arch, locations[i]); ok {
			out = append(out, symbol...)
		} else {
			out = append(out, "RTGNoRegister"...)
		}
		out = append(out, '\n')
	}
	callWords := targetABICallWords(document, target.ABI)
	for i := 0; i < 6; i++ {
		out = append(out, "var renvoRTGCallWord"...)
		out = appendDecimalFrame(out, i)
		out = append(out, " = "...)
		if i < len(callWords) {
			if symbol, ok := architectureRegisterOutput(document, target.Arch, callWords[i]); ok {
				out = append(out, symbol...)
			} else {
				out = append(out, "RTGNoRegister"...)
			}
		} else {
			out = append(out, "RTGNoRegister"...)
		}
		out = append(out, '\n')
	}
	syscallNumber := targetRuntimeRegisterList(target.Runtime, "number")
	syscallWords := targetRuntimeRegisterList(target.Runtime, "arguments")
	syscallResult := targetRuntimeRegisterList(target.Runtime, "result")
	out = appendPreparedRegisterAdapter(out, document, target.Arch,
		"renvoRTGSyscallNumber", syscallNumber, 0)
	for i := 0; i < 6; i++ {
		out = appendPreparedRegisterAdapter(out, document, target.Arch,
			"renvoRTGSyscallWord"+decimalText(i), syscallWords, i)
	}
	out = appendPreparedRegisterAdapter(out, document, target.Arch,
		"renvoRTGSyscallResult", syscallResult, 0)
	wordBytes, ok := integerField(document, target.Arch, "stack_word_bytes")
	if !ok || wordBytes <= 0 {
		wordBytes = target.Descriptor.WordBits / 8
	}
	out = append(out, "const renvoRTGStackWordBytes = "...)
	out = appendDecimalFrame(out, wordBytes)
	out = append(out, '\n')
	out = appendPreparedConditionAdapter(out, document, target.Arch)
	return out
}

func appendPreparedRegisterAdapter(out []byte, document Document, arch Declaration,
	name string, registers []string, index int) []byte {
	out = append(out, "var "...)
	out = append(out, name...)
	out = append(out, " = "...)
	if index < len(registers) {
		if symbol, ok := architectureRegisterOutput(document, arch, registers[index]); ok {
			out = append(out, symbol...)
		} else {
			out = append(out, "RTGNoRegister"...)
		}
	} else {
		out = append(out, "RTGNoRegister"...)
	}
	out = append(out, '\n')
	return out
}

func targetRuntimeRegisterList(runtime Declaration, field string) []string {
	block, ok := declarationBlock(runtime, "syscall")
	if !ok {
		return nil
	}
	for i := 0; i < len(block.Children); i++ {
		left, _, assignment := statementAssignment(block.Children[i])
		if assignment && len(left) == 1 && left[0] == field {
			return statementListValues(block.Children[i])
		}
	}
	return nil
}

func decimalText(value int) string {
	var out []byte
	out = appendDecimalFrame(out, value)
	return string(out)
}

func architectureLocationOutput(document Document, arch Declaration, location string) (string, bool) {
	local := architectureLocalPrefix(arch.Name) + exportedName(location)
	return generatedArchitectureOutput(document, local)
}

func architectureRegisterOutput(document Document, arch Declaration, register string) (string, bool) {
	local := architectureLocalPrefix(arch.Name) + upperIdentifier(register)
	return generatedArchitectureOutput(document, local)
}

func targetABICallWords(document Document, abi Declaration) []string {
	words := listField(document, abi, "call_words")
	if len(words) != 0 {
		return words
	}
	internal, ok := fieldValue(document, abi, "internal")
	if ok {
		base, found := document.Declaration(DeclABI, internal)
		if found {
			return targetABICallWords(document, base)
		}
	}
	words = listField(document, abi, "arguments")
	if len(words) != 0 {
		return words
	}
	return nil
}

func appendPreparedConditionAdapter(out []byte, document Document, arch Declaration) []byte {
	names := []string{"eq", "ne", "slt", "sge", "sle", "sgt", "ult", "uge", "ule", "ugt"}
	setcc := []int{0x94, 0x95, 0x9c, 0x9d, 0x9e, 0x9f, 0x92, 0x93, 0x96, 0x97}
	out = append(out, "func renvoRTGConditionFromSetcc(setcc int) RTGCondition {\n"...)
	for i := 0; i < len(names); i++ {
		local := architectureLocalPrefix(arch.Name) + upperIdentifier(names[i])
		symbol, ok := generatedArchitectureOutput(document, local)
		if !ok {
			continue
		}
		out = append(out, "if setcc == "...)
		out = appendDecimalFrame(out, setcc[i])
		out = append(out, " { return "...)
		out = append(out, symbol...)
		out = append(out, " }\n"...)
	}
	out = append(out, "return RTGCondition{}\n}\n"...)
	return out
}
