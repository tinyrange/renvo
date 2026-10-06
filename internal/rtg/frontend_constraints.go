package rtg

// TargetConstraint is a backend-owned GNU constraint class. Register and memory
// classes enumerate usable physical registers; an immediate class requires a
// compile-time integer. Resource classes are clobber names, not value operands.
type TargetConstraint struct {
	Name, Kind string
	Registers  []string
	Resource   string
}

func frontendConstraints(document Document, arch Declaration, v *TargetVocabulary) {
	for _, block := range arch.Statements {
		if statementBlockName(block) != "frontend_constraints" {
			continue
		}
		for _, s := range block.Children {
			left, right, ok := statementAssignment(s)
			c := TargetConstraint{}
			message := ""
			if !ok || len(left) != 1 || !frontendIdentifier(left[0]) || len(s.Children) != 0 {
				message = "invalid frontend constraint declaration"
			} else {
				c.Name = left[0]
				if len(right) == 1 && right[0] == "immediate" {
					c.Kind = "immediate"
				} else if len(right) == 2 && right[0] == "resource" && stringIndex(v.Resources, right[1]) >= 0 {
					c.Kind, c.Resource = "resource", right[1]
				} else {
					list := right
					c.Kind = "register"
					if len(list) != 0 && list[0] == "memory" {
						c.Kind = "memory"
						list = list[1:]
					}
					if len(list) < 3 || list[0] != "[" || list[len(list)-1] != "]" {
						message = "constraint must declare registers, memory registers, an immediate, or a resource"
					} else {
						for at := 1; at < len(list)-1; at++ {
							if at%2 == 0 {
								if list[at] != "," {
									message = "invalid constraint register separator"
								}
							} else {
								reg := list[at]
								if stringIndex(v.Registers, reg) < 0 || stringIndex(c.Registers, reg) >= 0 {
									message = "unknown or duplicate constraint register " + reg
								}
								c.Registers = append(c.Registers, reg)
							}
						}
						if len(list)%2 == 0 {
							message = "trailing constraint register comma"
						}
					}
				}
				for _, old := range v.Constraints {
					if old.Name == c.Name {
						message = "duplicate frontend constraint " + c.Name
					}
				}
			}
			if message != "" {
				v.Diagnostics = append(v.Diagnostics, statementDiagnostic(document, s, "RTG-CONSTRAINT-001", message))
			} else {
				v.Constraints = append(v.Constraints, c)
			}
		}
	}
}

func targetConstraint(v TargetVocabulary, name string) (TargetConstraint, bool) {
	for _, c := range v.Constraints {
		if c.Name == name {
			return c, true
		}
	}
	return TargetConstraint{}, false
}
