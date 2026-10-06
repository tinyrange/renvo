package rtg

// TargetManaged advertises registers and resources that the backend's ordinary
// expression/call boundary may destroy. The backend, not a caller's assembler
// spelling, owns argument and result placement. Arguments lists incoming
// call words in pop order; logical scalar parameter i uses word inputs-1-i.
type TargetManaged struct {
	Registers, Resources, Arguments                            []string
	Result, Move                                               string
	Address, Load, Store, Immediate, Jump, NewLabel, BindLabel string
	Ok                                                         bool
}

func frontendManaged(document Document, arch Declaration, v *TargetVocabulary) {
	seen := false
	for _, block := range arch.Statements {
		if statementBlockName(block) != "frontend_managed" {
			continue
		}
		message := ""
		if seen {
			message = "duplicate managed frontend policy"
		}
		seen = true
		policy := TargetManaged{}
		fields := []string{}
		for _, child := range block.Children {
			left, right, ok := statementAssignment(child)
			if !ok || len(left) != 1 || len(child.Children) != 0 || stringIndex(fields, left[0]) >= 0 {
				message = "invalid managed frontend property"
				continue
			}
			fields = append(fields, left[0])
			switch left[0] {
			case "registers", "resources":
				values := statementListValues(child)
				if len(right) < 2 || right[0] != "[" || right[len(right)-1] != "]" {
					message = "managed locations require a list"
					continue
				}
				allowed := v.Registers
				if left[0] == "resources" {
					allowed = v.Resources
				}
				checked := []string{}
				for _, name := range values {
					if stringIndex(allowed, name) < 0 || stringIndex(checked, name) >= 0 {
						message = "unknown or duplicate managed location " + name
					}
					checked = append(checked, name)
				}
				if left[0] == "registers" {
					policy.Registers = checked
				} else {
					policy.Resources = checked
				}
			case "move", "address", "load", "store", "immediate", "jump", "new_label", "bind_label":
				if len(right) != 1 {
					message = "invalid managed move operation"
				} else {
					switch left[0] {
					case "move":
						policy.Move = right[0]
					case "address":
						policy.Address = right[0]
					case "load":
						policy.Load = right[0]
					case "store":
						policy.Store = right[0]
					case "immediate":
						policy.Immediate = right[0]
					case "jump":
						policy.Jump = right[0]
					case "new_label":
						policy.NewLabel = right[0]
					case "bind_label":
						policy.BindLabel = right[0]
					}
				}
			default:
				message = "unknown managed frontend property " + left[0]
			}
		}
		if len(policy.Registers) == 0 || stringIndex(fields, "resources") < 0 {
			message = "managed policy requires registers and resources"
		}
		move, ok := frontendOperation(*v, policy.Move)
		if !ok || len(move.Parameters) != 2 || move.Parameters[0].Kind != "register" || move.Parameters[1].Kind != "register" || move.Result != "" || move.Effects.Control != "none" || move.Effects.Memory != "none" {
			message = "managed move has an incompatible signature"
		}
		for _, s := range arch.Statements {
			if statementBlockName(s) != "locations" {
				continue
			}
			for _, child := range s.Children {
				left, right, ok := statementAssignment(child)
				if !ok || len(left) != 1 {
					continue
				}
				if left[0] == "primary" && len(right) == 1 {
					policy.Result = right[0]
				}
				if left[0] == "frame" || left[0] == "stack" {
					for _, word := range right {
						if stringIndex(policy.Registers, word) >= 0 {
							message = "managed blocks cannot own the compiler stack or frame"
						}
					}
				}
			}
		}
		if stringIndex(policy.Registers, policy.Result) < 0 {
			message = "managed policy must permit the compiler primary result register"
		}
		if message != "" {
			v.Diagnostics = append(v.Diagnostics, statementDiagnostic(document, block, "RTG-MANAGED-001", message))
		} else {
			policy.Ok = true
			v.Managed = policy
		}
	}
}

func selectFrontendManaged(resolved ResolveResult, target ResolvedTarget, v *TargetVocabulary) {
	if !v.Managed.Ok {
		return
	}
	arguments := targetABICallWords(resolved.Document, target.ABI)
	if len(arguments) == 0 || len(arguments) > 6 {
		v.Managed.Ok = false
		return
	}
	// Specialized stack/register call adapters need a separately advertised
	// managed boundary. Do not apply the generic six-word adapter to them.
	if _, ok := targetABIGoHook(resolved.Document, target.ABI, "call_word_count"); ok {
		v.Managed.Ok = false
		return
	}
	for _, reg := range arguments {
		if stringIndex(v.Managed.Registers, reg) < 0 {
			v.Managed.Ok = false
			return
		}
	}
	v.Managed.Arguments = arguments
}
