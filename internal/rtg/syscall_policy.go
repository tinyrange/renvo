package rtg

// Raw syscall source arguments have one of two declared protocols: register
// words, or the directory-entry intrinsic selector followed by three words.
// Site-number metadata additionally requires a compile-time syscall number.
func decodeRawSyscallLayout(runtime Declaration) (bool, bool) {
	seen := false
	directory := false
	for _, statement := range runtime.Statements {
		if len(statement.Tokens) == 0 || statement.Tokens[0] != "raw_syscall_layout" {
			continue
		}
		left, right, assignment := statementAssignment(statement)
		if seen || !assignment || len(left) != 1 || len(right) != 1 || len(statement.Children) != 0 {
			return false, false
		}
		layout := valueName(right[0])
		if layout != "register_words" && layout != "directory_entries" {
			return false, false
		}
		seen = true
		directory = layout == "directory_entries"
	}
	return directory, true
}

func validateRawSyscallLayout(document Document, runtime Declaration) []Diagnostic {
	directory, ok := decodeRawSyscallLayout(runtime)
	_, custom := architectureGoHook(runtime, "emit_syscall_from_stack")
	if !ok || directory && !custom || custom && targetRuntimeSyscallField(runtime, "site_table") != "" {
		return []Diagnostic{resolveDiagnostic(document, runtime, "RTG-VALIDATE-136",
			"raw_syscall_layout must be register_words or directory_entries; directory_entries requires emit_syscall_from_stack; custom adapters cannot declare a raw instruction site table")}
	}
	return nil
}

func appendPreparedSyscallPolicy(source []byte, document Document, runtime Declaration) []byte {
	directory, _ := decodeRawSyscallLayout(runtime)
	policy := 1
	sites := targetRuntimeSyscallField(runtime, "site_table") == "address_number_pairs"
	if sites {
		policy = 2
	}
	if directory {
		policy = 3
	}
	source = append(source, "\nconst renvoRTGSyscallArgumentPolicy = "...)
	source = appendDecimalFrame(source, policy)
	hook, custom := architectureGoHook(runtime, "emit_syscall_from_stack")
	source = append(source, "\nconst renvoRTGCustomSyscall = "...)
	if custom {
		source = append(source, "true"...)
	} else {
		source = append(source, "false"...)
	}
	source = append(source, "\nfunc renvoRTGEmitCustomSyscall(out *renvoAsm, wordCount int, number int) bool {\n"...)
	if custom {
		source = append(source, "return rtg"...)
		source = append(source, exportedName(document.Unit)...)
		source = append(source, exportedName(hook)...)
		source = append(source, "(out, wordCount, number)\n"...)
	} else {
		source = append(source, "return false\n"...)
	}
	source = append(source, "}\nfunc renvoRTGRecordRawSyscall(out *renvoAsm, number int) {\n"...)
	if sites {
		source = append(source, "out.openbsdSyscalls = append(out.openbsdSyscalls, len(out.code))\nout.openbsdSyscalls = append(out.openbsdSyscalls, number)\n"...)
	}
	return append(source, "}\n"...)
}
