package rtg

// TargetWidth maps language operand widths and spelling modifiers to backend
// operations and canonical register identities. Aliases do not add physical
// registers or grant access to preserved compiler state.
type TargetWidth struct {
	Bits                              int
	Modifier, Load, LoadSigned, Store string
	Operations                        []string
	Aliases                           []TargetRegisterAlias
}
type TargetRegisterAlias struct{ Register, Name string }

func frontendWidths(document Document, arch Declaration, v *TargetVocabulary) {
	for _, block := range arch.Statements {
		if statementBlockName(block) != "frontend_widths" {
			continue
		}
		for _, child := range block.Children {
			width, message := decodeFrontendWidth(child, *v)
			if message == "" {
				for _, old := range v.Widths {
					if old.Bits == width.Bits || old.Modifier == width.Modifier {
						message = "duplicate frontend width or modifier"
					}
					for _, alias := range old.Aliases {
						for _, next := range width.Aliases {
							if alias.Name == next.Name {
								message = "duplicate frontend register alias"
							}
						}
					}
					for _, op := range old.Operations {
						if stringIndex(width.Operations, op) >= 0 {
							message = "operation has multiple operand widths"
						}
					}
				}
			}
			if message != "" {
				v.Diagnostics = append(v.Diagnostics, statementDiagnostic(document, child, "RTG-WIDTH-001", message))
			} else {
				v.Widths = append(v.Widths, width)
			}
		}
	}
}
func decodeFrontendWidth(s Statement, v TargetVocabulary) (TargetWidth, string) {
	w := TargetWidth{}
	seen := []string{}
	for _, child := range s.Children {
		if statementBlockName(child) == "aliases" {
			if stringIndex(seen, "aliases") >= 0 {
				return w, "duplicate register aliases"
			}
			seen = append(seen, "aliases")
			for _, entry := range child.Children {
				left, right, ok := statementAssignment(entry)
				if !ok || len(left) != 1 || len(right) != 1 || len(entry.Children) != 0 || stringIndex(v.Registers, left[0]) < 0 || !frontendIdentifier(right[0]) {
					return w, "invalid register alias"
				}
				if stringIndex(v.Registers, right[0]) >= 0 && right[0] != left[0] {
					return w, "register alias changes physical identity"
				}
				for _, old := range w.Aliases {
					if old.Register == left[0] || old.Name == right[0] {
						return w, "duplicate register alias"
					}
				}
				w.Aliases = append(w.Aliases, TargetRegisterAlias{Register: left[0], Name: right[0]})
			}
			continue
		}
		left, right, ok := statementAssignment(child)
		if !ok || len(left) != 1 || len(child.Children) != 0 || stringIndex(seen, left[0]) >= 0 {
			return w, "invalid frontend width property"
		}
		seen = append(seen, left[0])
		if left[0] == "operations" {
			list, valid := frontendList(child)
			if !valid {
				return w, "width operations require a list"
			}
			for _, name := range list {
				op, found := frontendOperation(v, name)
				if !found || op.Result != "" || stringIndex(w.Operations, name) >= 0 {
					return w, "invalid width operation"
				}
				w.Operations = append(w.Operations, name)
			}
			continue
		}
		if len(right) != 1 {
			return w, "invalid frontend width value"
		}
		switch left[0] {
		case "bits":
			bits, valid := parseInteger(right[0])
			if !valid || (bits != 8 && bits != 16 && bits != 32 && bits != 64) {
				return w, "unsupported operand width"
			}
			w.Bits = bits
		case "modifier":
			w.Modifier = right[0]
		case "load":
			w.Load = right[0]
		case "load_signed":
			w.LoadSigned = right[0]
		case "store":
			w.Store = right[0]
		default:
			return w, "unknown frontend width property"
		}
	}
	if w.Bits == 0 || len(w.Modifier) != 1 || w.Modifier == "c" || w.Modifier == "l" || !frontendIdentifier(w.Modifier) || len(w.Aliases) == 0 {
		return w, "width requires bits, a register modifier, and aliases"
	}
	for _, name := range []string{w.Load, w.LoadSigned, w.Store} {
		op, ok := frontendOperation(v, name)
		memory, first, second := "read", "register", "address"
		if name == w.Store {
			memory, first, second = "write", "address", "register"
		}
		if !ok || op.Result != "" || len(op.Parameters) != 2 || op.Parameters[0].Kind != first || op.Parameters[1].Kind != second || op.Effects.Memory != memory || op.Effects.Control != "none" {
			return w, "width helper has an incompatible signature"
		}
	}
	return w, ""
}
func targetWidth(v TargetVocabulary, bits int) (TargetWidth, bool) {
	for _, w := range v.Widths {
		if w.Bits == bits {
			return w, true
		}
	}
	return TargetWidth{}, false
}
func targetRegisterSpelling(v TargetVocabulary, register string, bits int) (string, bool) {
	if bits == 0 {
		bits = v.Target.WordBits
	}
	if w, ok := targetWidth(v, bits); ok {
		for _, alias := range w.Aliases {
			if alias.Register == register {
				return alias.Name, true
			}
		}
		return "", false
	}
	return register, bits == v.Target.WordBits && stringIndex(v.Registers, register) >= 0
}
func targetSpellingRegister(v TargetVocabulary, spelling string, bits int) (string, bool) {
	if bits == 0 {
		bits = v.Target.WordBits
	}
	for _, w := range v.Widths {
		for _, alias := range w.Aliases {
			if alias.Name == spelling {
				return alias.Register, w.Bits == bits
			}
		}
	}
	return spelling, bits == v.Target.WordBits && stringIndex(v.Registers, spelling) >= 0
}
func targetOperationWidth(v TargetVocabulary, name string) int {
	for _, w := range v.Widths {
		if stringIndex(w.Operations, name) >= 0 {
			return w.Bits
		}
	}
	return v.Target.WordBits
}
