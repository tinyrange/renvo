package rtg

// This is a closed foreign-call classification protocol, not an ABI display
// name. The runtime must separately provide an emitter for outbound calls.
func decodeObjectCallLayout(abi Declaration) (bool, bool) {
	seen := false
	for _, statement := range abi.Statements {
		if len(statement.Tokens) == 0 || statement.Tokens[0] != "object_call_layout" {
			continue
		}
		left, right, assignment := statementAssignment(statement)
		if seen || !assignment || len(left) != 1 || len(right) != 1 ||
			len(statement.Children) != 0 || valueName(right[0]) != "sysv_eightbyte" {
			return false, false
		}
		seen = true
	}
	return seen, true
}

func appendPreparedObjectCallPolicy(source []byte, target ResolvedTarget) []byte {
	layout, _ := decodeObjectCallLayout(target.ABI) // validated before generation
	source = append(source, "\nconst renvoRTGObjectAggregateRegisterBytes = "...)
	if layout {
		source = append(source, "16"...)
	} else {
		source = append(source, '0')
	}
	source = append(source, "\nconst renvoRTGObjectCallABI = "...)
	_, emitsCall := architectureGoHook(target.Runtime, "emit_static_call")
	if layout && emitsCall && stringIndex(target.Descriptor.Capabilities, "kernel_module") < 0 {
		source = append(source, "renvoObjectABISysV"...)
	} else {
		source = append(source, "renvoObjectABIUnavailable"...)
	}
	return append(source, '\n')
}
