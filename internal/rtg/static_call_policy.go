package rtg

// The split-register protocol carries semantic argument kinds and permits
// integer-only overflow on the stack. Its physical limits belong to this
// prepared recipe and the corresponding definition-owned bundled recipe.
func decodeStaticCallLayout(runtime Declaration) (bool, bool) {
	seen := false
	for _, statement := range runtime.Statements {
		if len(statement.Tokens) == 0 || statement.Tokens[0] != "static_call_layout" {
			continue
		}
		left, right, assignment := statementAssignment(statement)
		if seen || !assignment || len(left) != 1 || len(right) != 1 ||
			len(statement.Children) != 0 || valueName(right[0]) != "split_register_words8" {
			return false, false
		}
		seen = true
	}
	return seen, true
}

func appendPreparedStaticCallPolicy(source []byte, target ResolvedTarget) []byte {
	split, _ := decodeStaticCallLayout(target.Runtime)
	_, emitsCall := architectureGoHook(target.Runtime, "emit_static_call")
	source = append(source, "\nconst renvoRTGStaticCallPolicy = "...)
	policy := "renvoStaticCallUnavailable"
	if emitsCall {
		policy = "renvoStaticCallWords"
		if split {
			policy = "renvoStaticCallSplitRegisters"
		}
	}
	source = append(source, policy...)
	return append(source, '\n')
}
