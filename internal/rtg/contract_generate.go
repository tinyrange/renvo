package rtg

import "strings"

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
		out = append(out, '\n')
		if !nativeEmitter || arch.Name == "vm32" {
			out = append(out, "// Generated direct_emitter_v1 binding for "...)
			out = append(out, binding.Name...)
			out = append(out, ".\n"...)
			out = appendDirectEmitterEffectComment(out, directEmitterV1[operation])
		}
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
	return appendArchitectureHooksNamed(out, document, arch, nativeEmitter,
		"rtg"+exportedName(document.Unit)+"PatchRelocations")
}

func appendArchitectureHooksNamed(out []byte, document Document, arch Declaration,
	nativeEmitter bool, functionName string) []byte {
	return appendArchitectureHooksNamedMode(out, document, arch, nativeEmitter,
		functionName, false)
}

func appendCheckedInArchitectureHooks(out []byte, document Document,
	arch Declaration) []byte {
	return appendArchitectureHooksNamedMode(out, document, arch, true,
		"rtg"+exportedName(document.Unit)+"PatchRelocations", true)
}

func appendArchitectureHooksNamedMode(out []byte, document Document, arch Declaration,
	nativeEmitter bool, functionName string, compactNativeLabels bool) []byte {
	algorithm, found := architectureGoHook(arch, "patch_relocations")
	encoding := declarativeArchitectureRelocations(arch)
	if !found && encoding == "" {
		return out
	}
	prefix := "rtg" + exportedName(document.Unit)
	out = append(out, "\n// Generated definition-owned relocation finalizer.\nfunc "...)
	out = append(out, functionName...)
	out = append(out, "(out "...)
	if nativeEmitter {
		out = append(out, "*renvoAsm"...)
	} else {
		out = append(out, "*RTGEmitter"...)
	}
	out = append(out, ") {\n"...)
	if encoding != "" {
		out = appendDeclarativeArchitectureRelocationBody(
			out, encoding, nativeEmitter, compactNativeLabels)
		out = append(out, "}\n"...)
		return out
	}
	out = append(out, '\t')
	if external, found := embeddedExternalName(architectureExports(arch), algorithm); found {
		out = append(out, external...)
	} else {
		out = append(out, prefix...)
		out = append(out, exportedName(algorithm)...)
	}
	out = append(out, "(out)\n}\n"...)
	return out
}

func declarativeArchitectureRelocations(arch Declaration) string {
	for i := 0; i < len(arch.Statements); i++ {
		left, right, assignment := statementAssignment(arch.Statements[i])
		if !assignment || len(left) != 1 || left[0] != "patch_relocations" || len(right) != 1 {
			continue
		}
		if right[0] == "relative32_le" || right[0] == "aarch64_labels" ||
			right[0] == "arm32_labels" {
			return right[0]
		}
	}
	return ""
}

func appendDeclarativeArchitectureRelocationBody(out []byte, encoding string,
	nativeEmitter bool, compactNativeLabels bool) []byte {
	get32 := "RTGGet32At"
	code := "out.Code()"
	patch32 := "out.PatchUint32("
	if nativeEmitter {
		get32 = "renvoGet32At"
		code = "out.code"
		patch32 = "renvoPut32At(out.code, "
		out = append(out, "\tfor i := 0; i+1 < len(out.relocs); i += 2 {\n"...)
		out = append(out, "\t\tat := int(renvo_runtime_UnsafeInt32At(out.relocs, i)) & 2147483647\n"...)
		out = append(out, "\t\tlabel := int(renvo_runtime_UnsafeInt32At(out.relocs, i+1)) & 2147483647\n"...)
		out = append(out, "\t\ttarget := renvoAsmLabelPosition(out, label)\n"...)
		out = append(out, "\t\tif at < 0 || at+4 > len(out.code) {\n"...)
		out = append(out, "\t\t\tout.patchFailed = true\n"...)
		out = append(out, "\t\t\tcontinue\n"...)
		out = append(out, "\t\t}\n"...)
		out = append(out, "\t\tif target < 0 { continue }\n"...)
	} else {
		out = append(out, "\tfor i := 0; i < out.RelocationCount(); i++ {\n"...)
		out = append(out, "\t\tat := out.RelocationOffset(i)\n"...)
		out = append(out, "\t\tlabel := out.RelocationLabel(i)\n"...)
		out = append(out, "\t\ttarget := out.LabelPosition(label)\n"...)
		out = append(out, "\t\tif at < 0 || at+4 > len(out.Code()) || target < 0 { continue }\n"...)
	}
	if encoding == "relative32_le" {
		out = append(out, "\t\taddend := "...)
		out = append(out, get32...)
		out = append(out, '(')
		out = append(out, code...)
		out = append(out, ", at)\n\t\t"...)
		out = append(out, patch32...)
		out = append(out, "at, target+addend-(at+4))\n"...)
	} else if encoding == "aarch64_labels" {
		out = append(out, "\t\tinsn := "...)
		out = append(out, get32...)
		out = append(out, '(')
		out = append(out, code...)
		out = append(out, ", at)\n"...)
		body := declarativeAArch64LabelRelocationBody
		if compactNativeLabels {
			// Checked-in lowering emits only branch labels. Data and BSS
			// addresses use the separate absolute-relocation stream. Keep ADR
			// label support in prepared backends, whose typed hooks may create
			// label-backed RTGAddress values directly.
			body = declarativeAArch64NativeLabelRelocationBody
		}
		body = strings.ReplaceAll(body, "PATCH32(", patch32)
		out = append(out, body...)
	} else if encoding == "arm32_labels" {
		out = append(out, "\t\tinsn := "...)
		out = append(out, get32...)
		out = append(out, '(')
		out = append(out, code...)
		out = append(out, ", at)\n"...)
		body := declarativeARM32LabelRelocationBody
		if nativeEmitter {
			body = declarativeARM32NativeLabelRelocationBody
		}
		if compactNativeLabels {
			body = declarativeARM32CompactNativeLabelRelocationBody
		}
		body = strings.ReplaceAll(body, "RTGGET32", get32)
		body = strings.ReplaceAll(body, "CODE", code)
		body = strings.ReplaceAll(body, "PATCH32(", patch32)
		out = append(out, body...)
	}
	return append(out, "\t}\n"...)
}

const declarativeAArch64LabelRelocationBody = `
		displacement := target - at
		if (insn & 0xfc000000) == 0x94000000 {
			PATCH32(at, 0x94000000|((displacement/4)&0x03ffffff))
		} else if (insn & 0xfc000000) == 0x14000000 {
			PATCH32(at, 0x14000000|((displacement/4)&0x03ffffff))
		} else if (insn & 0xff000010) == 0x54000000 {
			PATCH32(at, (insn&0xff00001f)|(((displacement/4)&0x7ffff)<<5))
		} else if (insn & 0x9f000000) == 0x10000000 {
			addend := ((insn >> 29) & 3) | (((insn >> 5) & 0x7ffff) << 2)
			if (addend & 0x100000) != 0 {
				addend = addend - 0x200000
			}
			displacement = displacement + addend
			PATCH32(at, 0x10000000|(insn&31)|
				((displacement&3)<<29)|(((displacement>>2)&0x7ffff)<<5))
	}
`

const declarativeAArch64NativeLabelRelocationBody = `
		displacement := target - at
		if (insn & 0x7c000000) == 0x14000000 {
			PATCH32(at, (insn&0xfc000000)|((displacement/4)&0x03ffffff))
		} else if (insn & 0xff000010) == 0x54000000 {
			PATCH32(at, (insn&0xff00001f)|(((displacement/4)&0x7ffff)<<5))
		}
`

const declarativeARM32LabelRelocationBody = `
		if (insn & 0x0e000000) == 0x0a000000 {
			displacement := target - (at + 8)
			PATCH32(at, (insn&0xff000000)|((displacement/4)&0x00ffffff))
		} else if (insn & 0xfff00000) == 0xe3000000 {
			highInsn := RTGGET32(CODE, at+4)
			addend := insn & 0x0fff
			addend = addend | ((insn >> 4) & 0xf000)
			addend = addend | ((highInsn & 0x0fff) << 16)
			addend = addend | ((highInsn & 0x000f0000) << 12)
			if (addend & 0x80000000) != 0 {
				addend = addend - 4294967296
			}
			value := target + addend - (at + 16)
			low := value & 65535
			high := (value >> 16) & 65535
			PATCH32(at, (insn&0xfff0f000)|((low&0xf000)<<4)|(low&0x0fff))
			PATCH32(at+4,
				(highInsn&0xfff0f000)|((high&0xf000)<<4)|(high&0x0fff))
		}
`

const declarativeARM32NativeLabelRelocationBody = `
		if (insn & 0x0e000000) == 0x0a000000 {
			displacement := target - (at + 8)
			PATCH32(at, (insn&0xff000000)|((displacement/4)&0x00ffffff))
		} else if (insn & 0xfff00000) == 0xe3000000 {
			highInsn := RTGGET32(CODE, at+4)
			addend := (insn & 0x0fff) | ((insn >> 4) & 0xf000)
			addend = addend | ((highInsn & 0x0fff) << 16)
			addend = addend | ((highInsn & 0x000f0000) << 12)
			if (addend & 0x80000000) != 0 {
				addend = addend - 4294967296
			}
			reg := (insn >> 12) & 15
			renvoArmAsmPatchMovRegImmAt(out, at, reg, target+addend-(at+16))
		}
`

const declarativeARM32CompactNativeLabelRelocationBody = `
		if (insn & 0x0e000000) == 0x0a000000 {
			displacement := target - (at + 8)
			PATCH32(at, (insn&0xff000000)|((displacement/4)&0x00ffffff))
		}
`

// Prepared backends expose one stable, target-neutral adapter surface to the
// shared lowering kernel. The adapter is generated once for the selected
// architecture and every call is a direct Go call to that definition's
// specialized binding.
func appendPreparedDirectEmitterAdapters(out []byte, document Document, target ResolvedTarget) []byte {
	prefix := "rtg" + exportedName(document.Unit)
	bindings := architectureBindings(target.Arch)
	out = append(out, "\nvar renvoRTGUnsupportedOperation int\n"...)
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
	if declarativeArchitectureRelocations(target.Arch) != "" {
		out = append(out, prefix...)
		out = append(out, "PatchRelocations(out)\n"...)
	} else if algorithm, found := architectureGoHook(target.Arch, "patch_relocations"); found {
		out = append(out, prefix...)
		out = append(out, exportedName(algorithm)...)
		out = append(out, "(out)\n"...)
	}
	out = append(out, "}\n"...)
	return out
}

func appendDirectEmitterKernelAdapters(out []byte) []byte {
	out = append(out, "\nvar renvoRTGUnsupportedOperation int\n"...)
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
func renvoRTGMarkLabel(out *renvoAsm, label int) {}
func renvoRTGFunctionStart(out *renvoAsm, label int) {}
func renvoRTGFunctionFinish(out *renvoAsm) {}
func renvoRTGEmitJITCall(out *renvoAsm, entry RTGRegister, stackTop RTGRegister, argsData RTGRegister, argsLen RTGRegister, envData RTGRegister, envLen RTGRegister) bool { return false }
func renvoRTGEmitUnsignedDivide(out *renvoAsm, remainder bool) bool { return false }
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
	out = append(out, "func renvoRTGMarkLabel(out *renvoAsm, label int) {\n"...)
	if algorithm, found := targetABIGoHook(document, target.ABI, "mark_label"); found {
		out = append(out, prefix...)
		out = append(out, exportedName(algorithm)...)
		out = append(out, "(out, label)\n"...)
	}
	out = append(out, "}\n"...)
	structured, _ := integerField(document, target.ABI, "structured_functions")
	out = append(out, "const renvoRTGStructuredFunctions = "...)
	if structured != 0 {
		out = append(out, '1')
	} else {
		out = append(out, '0')
	}
	out = append(out, "\nfunc renvoRTGFunctionStart(out *renvoAsm, label int) {\n"...)
	if algorithm, found := targetABIGoHook(document, target.ABI, "function_start"); found {
		out = append(out, prefix...)
		out = append(out, exportedName(algorithm)...)
		out = append(out, "(out, label)\n"...)
	}
	out = append(out, "}\nfunc renvoRTGFunctionFinish(out *renvoAsm) {\n"...)
	if algorithm, found := targetABIGoHook(document, target.ABI, "function_finish"); found {
		out = append(out, prefix...)
		out = append(out, exportedName(algorithm)...)
		out = append(out, "(out)\n"...)
	}
	out = append(out, "}\n"...)
	out = appendPreparedABIHook(out, document, target.ABI, prefix,
		"renvoRTGEmitJITCall", "jit_call",
		"out *renvoAsm, entry RTGRegister, stackTop RTGRegister, argsData RTGRegister, argsLen RTGRegister, envData RTGRegister, envLen RTGRegister",
		"out, entry, stackTop, argsData, argsLen, envData, envLen")
	out = appendPreparedABIHook(out, document, target.ABI, prefix,
		"renvoRTGEmitUnsignedDivide", "unsigned_divide",
		"out *renvoAsm, remainder bool", "out, remainder")
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
	out = append(out, "\nfunc renvoRTGRejectImageSize(memory bool, needed int, limit int) {\n"...)
	out = append(out, "\trenvoRTGImageLimitMemory = memory\n\trenvoRTGImageLimitNeeded = needed\n\trenvoRTGImageLimit = limit\n}\n"...)
	out = append(out, "\nconst renvoRTGCodeOffset = "...)
	out = appendDecimalFrame(out, codeOffset)
	out = append(out, "\nfunc renvoRTGImage(out *renvoAsm) []byte {\n"...)
	if declarativeFormatImage(outputFormat) != "" {
		out = append(out, "\treturn "...)
		out = append(out, declarativeFormatImageName(document, outputFormat)...)
		out = append(out, "(out)\n"...)
	} else if algorithm, hasImage := architectureGoHook(outputFormat, "image"); hasImage {
		out = append(out, "\treturn rtg"...)
		out = append(out, exportedName(document.Unit)...)
		out = append(out, exportedName(algorithm)...)
		out = append(out, "(out)\n"...)
	} else {
		out = append(out, "\treturn nil\n"...)
	}
	out = append(out, "}\n"...)
	out = append(out, "func renvoRTGKernelImage(out *renvoAsm, initLabel int, exitLabel int) []byte {\n"...)
	if declarativeFormatKernelImage(outputFormat) != "" {
		out = append(out, "\treturn "...)
		out = append(out, declarativeFormatKernelImageName(document, outputFormat)...)
		out = append(out, "(out, initLabel, exitLabel)\n"...)
	} else if algorithm, hasAlgorithm := architectureGoHook(outputFormat, "kernel_image"); hasAlgorithm {
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
	if algorithm, hasAlgorithm := architectureGoHook(target.Runtime, "emit_entry_start_simple"); hasAlgorithm {
		out = append(out, "\t_ = stateOffset\n\treturn rtg"...)
		out = append(out, exportedName(document.Unit)...)
		out = append(out, exportedName(algorithm)...)
		out = append(out, "(out)\n"...)
	} else if algorithm, hasAlgorithm := architectureGoHook(target.Runtime, "emit_entry_start"); hasAlgorithm {
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
	} else if template, found := decodeRuntimeEntryTemplate(target.Runtime); found {
		out = append(out, "\t_ = stateOffset\n\tif parameterCount == 0 { return true }\n"...)
		out = append(out, "\tif parameterCount < 0 || parameterCount > "...)
		out = appendDecimalFrame(out, template.maxParameters)
		if template.requiresState {
			out = append(out, " || stateOffset < 0"...)
		}
		out = append(out, " { return false }\n"...)
		for i := 0; i < len(template.bss); i++ {
			out = append(out, "\t"...)
			out = append(out, template.bss[i].name...)
			out = append(out, "Offset := out.ReserveBSS("...)
			out = appendDecimalFrame(out, template.bss[i].size)
			out = append(out, ", "...)
			out = appendDecimalFrame(out, template.bss[i].alignment)
			out = append(out, ")\n"...)
		}
		if template.algorithm != "" {
			out = append(out, "\trtg"...)
			out = append(out, exportedName(document.Unit)...)
			out = append(out, exportedName(template.algorithm)...)
			out = append(out, "(out"...)
			for i := 0; i < len(template.bss); i++ {
				out = append(out, ", "...)
				out = append(out, template.bss[i].name...)
				out = append(out, "Offset"...)
			}
			if template.requiresState {
				out = append(out, ", stateOffset"...)
			}
			out = append(out, ")\n"...)
		} else {
			out = append(out, "\tbase := len(out.code)\n\trenvoAsmEmitText(out, "...)
			out = append(out, template.code...)
			out = append(out, ")\n"...)
			for i := 0; i < len(template.relocations); i++ {
				relocation := template.relocations[i]
				out = append(out, "\trenvoAsmAddAbsReloc(out, base+"...)
				out = appendDecimalFrame(out, relocation.offset)
				out = append(out, ", "...)
				if relocation.kind == "bss" {
					out = append(out, relocation.target...)
					out = append(out, "Offset"...)
					if relocation.addend != 0 {
						out = append(out, '+')
						out = appendDecimalFrame(out, relocation.addend)
					}
					out = append(out, ", RTGRelocationAbsoluteBSS)\n"...)
				} else {
					out = append(out, relocation.target...)
					out = append(out, ", RTGRelocationImport)\n"...)
				}
			}
		}
		out = append(out, "\treturn true\n"...)
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

type runtimeEntryBSS struct {
	name      string
	size      int
	alignment int
}

type runtimeEntryRelocation struct {
	offset int
	kind   string
	target string
	addend int
}

type runtimeEntryTemplate struct {
	maxParameters int
	code          string
	algorithm     string
	requiresState bool
	bss           []runtimeEntryBSS
	relocations   []runtimeEntryRelocation
}

func decodeRuntimeEntryTemplate(runtime Declaration) (runtimeEntryTemplate, bool) {
	var result runtimeEntryTemplate
	for i := 0; i < len(runtime.Statements); i++ {
		statement := runtime.Statements[i]
		if len(statement.Tokens) < 2 || len(statement.Tokens) > 3 ||
			statement.Tokens[0] != "entry" ||
			statement.Tokens[1] != "template" && statement.Tokens[1] != "sequence" {
			continue
		}
		if statement.Tokens[1] == "sequence" {
			if len(statement.Tokens) != 3 {
				continue
			}
			result.algorithm = statement.Tokens[2]
		}
		for j := 0; j < len(statement.Children); j++ {
			child := statement.Children[j]
			left, right, assignment := statementAssignment(child)
			if assignment && len(left) == 1 && left[0] == "max_parameters" &&
				len(right) == 1 {
				result.maxParameters, _ = parseInteger(right[0])
			} else if assignment && len(left) == 1 && left[0] == "code" &&
				len(right) == 1 {
				result.code = right[0]
			} else if assignment && len(left) == 1 && left[0] == "requires_state" &&
				len(right) == 1 {
				result.requiresState = right[0] == "true"
			} else if len(child.Tokens) == 4 && child.Tokens[0] == "bss" {
				size, sizeOK := parseInteger(child.Tokens[2])
				alignment, alignmentOK := parseInteger(child.Tokens[3])
				if sizeOK && alignmentOK {
					result.bss = append(result.bss, runtimeEntryBSS{
						name: child.Tokens[1], size: size, alignment: alignment,
					})
				}
			} else if len(child.Tokens) == 4 && child.Tokens[0] == "relocation" &&
				(child.Tokens[2] == "bss" || child.Tokens[2] == "import") {
				offset, ok := parseInteger(child.Tokens[1])
				if ok {
					result.relocations = append(result.relocations, runtimeEntryRelocation{
						offset: offset, kind: child.Tokens[2], target: child.Tokens[3],
					})
				}
			}
		}
		return result, result.maxParameters > 0 &&
			(result.code != "" || result.algorithm != "")
	}
	return result, false
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
	ioTemplate, hasIOTemplate := decodeRuntimeIOTemplate(target.Runtime)
	if hasIOTemplate {
		positionRegister := ""
		callWords := targetABICallWords(document, target.ABI)
		if len(callWords) > 3 {
			positionRegister, _ = architectureRegisterOutput(document, target.Arch, callWords[3])
		}
		out = append(out, "func renvoRTGEmitRuntimeIO(out *renvoAsm, write bool, positional bool) {\n"...)
		for i := 0; i < len(ioTemplate.bss); i++ {
			out = append(out, "\t"...)
			out = append(out, ioTemplate.bss[i].name...)
			out = append(out, "Offset := out.ReserveBSS("...)
			out = appendDecimalFrame(out, ioTemplate.bss[i].size)
			out = append(out, ", "...)
			out = appendDecimalFrame(out, ioTemplate.bss[i].alignment)
			out = append(out, ")\n"...)
		}
		out = append(out, "\thelper := out.NewLabel()\n\tafter := out.NewLabel()\n"...)
		out = append(out, "\trenvoRTGDirectJump(out, after)\n\tout.Mark(helper)\n\tbase := len(out.code)\n"...)
		out = append(out, "\tif write {\n\t\trenvoAsmEmitText(out, "...)
		out = append(out, ioTemplate.writeCode...)
		out = append(out, ")\n"...)
		out = appendRuntimeTemplateRelocations(out, ioTemplate.writeRelocations)
		out = append(out, "\t} else {\n\t\trenvoAsmEmitText(out, "...)
		out = append(out, ioTemplate.readCode...)
		out = append(out, ")\n"...)
		out = appendRuntimeTemplateRelocations(out, ioTemplate.readRelocations)
		out = append(out, "\t}\n\tout.Mark(after)\n"...)
		if positionRegister != "" {
			out = append(out, "\tif !positional { renvoRTGDirectMoveImmediate(out, "...)
			out = append(out, positionRegister...)
			out = append(out, ", -1) }\n"...)
		}
		out = append(out, "\trenvoRTGDirectCall(out, helper)\n}\n"...)
	}
	out = append(out, "func renvoRTGEmitRuntimeOperation(out *renvoAsm, operation int) bool {\n"...)
	if algorithm, found := architectureGoHook(target.Runtime, "emit_operation"); found {
		out = append(out, "\treturn rtg"...)
		out = append(out, exportedName(document.Unit)...)
		out = append(out, exportedName(algorithm)...)
		out = append(out, "(out, operation)\n}\n"...)
		return out
	}
	operationHooks := []runtimeOperationTemplate{
		{"read", 1, nil},
		{"write", 2, nil},
		{"read_at", 3, nil},
		{"write_at", 4, nil},
		{"open", 5, nil},
		{"close", 6, nil},
		{"chmod", 7, nil},
	}
	hasOperationHooks := false
	if hasIOTemplate {
		out = append(out, "\tif operation == 1 { renvoRTGEmitRuntimeIO(out, false, false); return true }\n"...)
		out = append(out, "\tif operation == 2 { renvoRTGEmitRuntimeIO(out, true, false); return true }\n"...)
		out = append(out, "\tif operation == 3 { renvoRTGEmitRuntimeIO(out, false, true); return true }\n"...)
		out = append(out, "\tif operation == 4 { renvoRTGEmitRuntimeIO(out, true, true); return true }\n"...)
		hasOperationHooks = true
	}
	for i := 0; i < len(operationHooks); i++ {
		algorithm, found := runtimeOperationHook(target.Runtime, operationHooks[i].name)
		if !found {
			continue
		}
		hasOperationHooks = true
		out = append(out, "\tif operation == "...)
		out = appendDecimalFrame(out, operationHooks[i].code)
		out = append(out, " {\n\t\trtg"...)
		out = append(out, exportedName(document.Unit)...)
		out = append(out, exportedName(algorithm)...)
		out = append(out, "(out)\n\t\treturn true\n\t}\n"...)
	}
	if hasOperationHooks {
		out = append(out, "\treturn false\n}\n"...)
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
		valid := resultOK
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
			out = append(out, ")\n"...)
			if target.Descriptor.OS == "openbsd" {
				out = append(out, "\t\tout.openbsdSyscalls = append(out.openbsdSyscalls, len(out.code))\n"...)
				out = append(out, "\t\tout.openbsdSyscalls = append(out.openbsdSyscalls, "...)
				out = appendDecimalFrame(out, number)
				out = append(out, ")\n"...)
			}
			out = append(out, "\t\trenvoRTGDirectHostSyscall(out)\n"...)
			if resultOK && targetRuntimeSyscallField(target.Runtime, "error") == "carry" {
				out = append(out, "\t\tsyscallOK := out.NewLabel()\n"...)
				out = append(out, "\t\trenvoRTGDirectJumpCondition(out, renvoRTGConditionFromSetcc(147), syscallOK)\n"...)
				out = append(out, "\t\trenvoRTGDirectMove(out, renvoRTGScratch, "...)
				out = append(out, resultRegister...)
				out = append(out, ")\n\t\trenvoRTGDirectMoveImmediate(out, "...)
				out = append(out, resultRegister...)
				out = append(out, ", 0)\n\t\trenvoRTGDirectSubtract(out, "...)
				out = append(out, resultRegister...)
				out = append(out, ", renvoRTGScratch)\n\t\tout.Mark(syscallOK)\n"...)
			}
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

type runtimeIOTemplate struct {
	readCode         string
	writeCode        string
	bss              []runtimeEntryBSS
	readRelocations  []runtimeEntryRelocation
	writeRelocations []runtimeEntryRelocation
}

func decodeRuntimeIOTemplate(runtime Declaration) (runtimeIOTemplate, bool) {
	var result runtimeIOTemplate
	for i := 0; i < len(runtime.Statements); i++ {
		statement := runtime.Statements[i]
		if len(statement.Tokens) != 1 || statement.Tokens[0] != "io_template" {
			continue
		}
		for j := 0; j < len(statement.Children); j++ {
			child := statement.Children[j]
			left, right, assignment := statementAssignment(child)
			if assignment && len(left) == 1 && len(right) == 1 {
				if left[0] == "read_code" {
					result.readCode = right[0]
				} else if left[0] == "write_code" {
					result.writeCode = right[0]
				}
			} else if len(child.Tokens) == 4 && child.Tokens[0] == "bss" {
				size, sizeOK := parseInteger(child.Tokens[2])
				alignment, alignmentOK := parseInteger(child.Tokens[3])
				if sizeOK && alignmentOK {
					result.bss = append(result.bss, runtimeEntryBSS{
						name: child.Tokens[1], size: size, alignment: alignment,
					})
				}
			} else if len(child.Tokens) >= 4 && len(child.Tokens) <= 5 &&
				(child.Tokens[0] == "read_relocation" ||
					child.Tokens[0] == "write_relocation") {
				offset, offsetOK := parseInteger(child.Tokens[1])
				addend := 0
				addendOK := true
				if len(child.Tokens) == 5 {
					addend, addendOK = parseInteger(child.Tokens[4])
				}
				if offsetOK && addendOK {
					relocation := runtimeEntryRelocation{
						offset: offset, kind: child.Tokens[2], target: child.Tokens[3],
						addend: addend,
					}
					if child.Tokens[0] == "read_relocation" {
						result.readRelocations = append(result.readRelocations, relocation)
					} else {
						result.writeRelocations = append(result.writeRelocations, relocation)
					}
				}
			}
		}
		return result, result.readCode != "" && result.writeCode != ""
	}
	return result, false
}

func appendRuntimeTemplateRelocations(out []byte, relocations []runtimeEntryRelocation) []byte {
	for i := 0; i < len(relocations); i++ {
		relocation := relocations[i]
		out = append(out, "\t\trenvoAsmAddAbsReloc(out, base+"...)
		out = appendDecimalFrame(out, relocation.offset)
		out = append(out, ", "...)
		out = append(out, relocation.target...)
		if relocation.kind == "bss" {
			out = append(out, "Offset"...)
			if relocation.addend != 0 {
				out = append(out, '+')
				out = appendDecimalFrame(out, relocation.addend)
			}
			out = append(out, ", RTGRelocationAbsoluteBSS)\n"...)
		} else {
			out = append(out, ", RTGRelocationImport)\n"...)
		}
	}
	return out
}

func runtimeOperationHook(runtime Declaration, operation string) (string, bool) {
	for i := 0; i < len(runtime.Statements); i++ {
		statement := runtime.Statements[i]
		if len(statement.Tokens) < 2 || statement.Tokens[0] != "operation" ||
			statement.Tokens[1] != operation {
			continue
		}
		declaration := Declaration{Statements: statement.Children}
		return architectureGoHook(declaration, "emit")
	}
	return "", false
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

// RuntimeOperationInteger exposes one resolved runtime field to generators
// which project target metadata outside the RTG package.
func RuntimeOperationInteger(runtime Declaration, operation string, field string) (int, bool) {
	return runtimeOperationInteger(runtime, operation, field)
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
			tokens := statement.Children[j].Tokens
			for at := 0; at+2 < len(tokens); at++ {
				if tokens[at] != field || tokens[at+1] != "=" || tokens[at+2] != "[" {
					continue
				}
				var values []string
				for at += 3; at < len(tokens) && tokens[at] != "]"; at++ {
					if tokens[at] != "," {
						values = append(values, valueName(tokens[at]))
					}
				}
				return values
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
		out = append(out, "\tif renvoRTGUnsupportedOperation == 0 { renvoRTGUnsupportedOperation = "...)
		out = appendDecimalFrame(out, directEmitterOperationIndex(operation.Name)+1)
		out = append(out, " }\n"...)
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

func targetRuntimeSyscallField(runtime Declaration, field string) string {
	block, ok := declarationBlock(runtime, "syscall")
	if !ok {
		return ""
	}
	for i := 0; i < len(block.Children); i++ {
		left, right, assignment := statementAssignment(block.Children[i])
		if assignment && len(left) == 1 && left[0] == field && len(right) == 1 {
			return valueName(right[0])
		}
	}
	return ""
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
	out = append(out, "if renvoRTGUnsupportedOperation == 0 { renvoRTGUnsupportedOperation = 1000 + setcc }\n"...)
	out = append(out, "return RTGCondition{}\n}\n"...)
	return out
}
