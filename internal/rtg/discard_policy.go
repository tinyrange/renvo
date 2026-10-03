package rtg

// Runtime page reclamation is an optional three-argument syscall contract:
// address, byte count, advice. No OS or ABI display name enables it.
type runtimeDiscardPolicy struct {
	pageSize int
	number   int
	advice   int
}

func decodeRuntimeDiscardPolicy(runtime Declaration) (runtimeDiscardPolicy, bool) {
	var policy runtimeDiscardPolicy
	blocks := 0
	for _, statement := range runtime.Statements {
		if len(statement.Tokens) == 0 || statement.Tokens[0] != "discard_pages" {
			continue
		}
		if len(statement.Tokens) != 1 {
			return policy, false
		}
		blocks++
		if blocks != 1 {
			return policy, false
		}
		seen := []bool{false, false, false}
		for _, child := range statement.Children {
			left, right, assignment := statementAssignment(child)
			if !assignment || len(left) != 1 || len(right) != 1 || len(child.Children) != 0 {
				return policy, false
			}
			index := stringIndex([]string{"page_size", "number", "advice"}, left[0])
			if index < 0 || seen[index] {
				return policy, false
			}
			seen[index] = true
			value, ok := parseInteger(valueName(right[0]))
			if !ok || value < 0 {
				return policy, false
			}
			if index == 0 {
				policy.pageSize = value
			}
			if index == 1 {
				policy.number = value
			}
			if index == 2 {
				policy.advice = value
			}
		}
		if !seen[0] || !seen[1] || !seen[2] || policy.pageSize < 1 || policy.pageSize > 1073741824 || policy.pageSize&(policy.pageSize-1) != 0 {
			return policy, false
		}
	}
	return policy, true
}

func appendPreparedDiscardPolicy(source []byte, runtime Declaration) []byte {
	policy, _ := decodeRuntimeDiscardPolicy(runtime) // validated before generation
	source = append(source, "\nconst renvoRTGDiscardPageSize = "...)
	source = appendDecimalFrame(source, policy.pageSize)
	source = append(source, "\nconst renvoRTGDiscardNumber = "...)
	source = appendDecimalFrame(source, policy.number)
	source = append(source, "\nconst renvoRTGDiscardAdvice = "...)
	source = appendDecimalFrame(source, policy.advice)
	source = append(source, "\nfunc renvoRTGRecordDiscardSyscall(a *renvoAsm) {\n"...)
	if policy.pageSize != 0 && targetRuntimeSyscallField(runtime, "site_table") == "address_number_pairs" {
		source = append(source, "a.openbsdSyscalls = append(a.openbsdSyscalls, len(a.code))\na.openbsdSyscalls = append(a.openbsdSyscalls, renvoRTGDiscardNumber)\n"...)
	}
	return append(source, "}\n"...)
}
