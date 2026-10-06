package rtg

// TargetSyntax describes a spelling frontend, separately from the machine
// operation contract. Forms and operand permutations remain backend-owned.
type TargetSyntax struct {
	Name, Style                  string
	NewLabel, BindLabel, Address string
	Forms                        []TargetSyntaxForm
}
type TargetSyntaxForm struct {
	Mnemonic   string
	Parameters []TargetParameter
	Operation  string
	Arguments  []TargetSyntaxArgument
}
type TargetSyntaxArgument struct {
	Parameter                 int // -1 for a typed literal supplied by the definition
	LiteralKind, LiteralValue string
}

func frontendSyntaxes(document Document, arch Declaration, v *TargetVocabulary) {
	for i := 0; i < len(arch.Statements); i++ {
		block := arch.Statements[i]
		if statementBlockName(block) != "frontend_syntax" {
			continue
		}
		for j := 0; j < len(block.Children); j++ {
			s := block.Children[j]
			syntax, message := decodeFrontendSyntax(s, *v)
			if message == "" {
				for k := 0; k < len(v.Syntaxes); k++ {
					if v.Syntaxes[k].Name == syntax.Name {
						message = "duplicate assembler syntax " + syntax.Name
					}
				}
			}
			if message != "" {
				v.Diagnostics = append(v.Diagnostics, statementDiagnostic(document, s, "RTG-SYNTAX-001", message))
			} else {
				v.Syntaxes = append(v.Syntaxes, syntax)
			}
		}
	}
}
func frontendOperation(v TargetVocabulary, name string) (TargetOperation, bool) {
	for i := 0; i < len(v.Operations); i++ {
		if v.Operations[i].Name == name {
			return v.Operations[i], true
		}
	}
	return TargetOperation{}, false
}
func decodeFrontendSyntax(s Statement, v TargetVocabulary) (TargetSyntax, string) {
	syntax := TargetSyntax{}
	if len(s.Tokens) != 1 || !frontendIdentifier(s.Tokens[0]) {
		return syntax, "invalid assembler syntax name"
	}
	syntax.Name = s.Tokens[0]
	seen := []string{}
	for i := 0; i < len(s.Children); i++ {
		child := s.Children[i]
		left, right, assignment := statementAssignment(child)
		if assignment && len(left) == 1 {
			if len(right) != 1 || !frontendIdentifier(right[0]) || len(child.Children) != 0 || stringIndex(seen, left[0]) >= 0 {
				return syntax, "invalid or duplicate assembler syntax property"
			}
			seen = append(seen, left[0])
			switch left[0] {
			case "style":
				syntax.Style = right[0]
			case "new_label":
				syntax.NewLabel = right[0]
			case "bind_label":
				syntax.BindLabel = right[0]
			case "address":
				syntax.Address = right[0]
			default:
				return syntax, "unknown assembler syntax property " + left[0]
			}
			continue
		}
		form, message := decodeFrontendSyntaxForm(child, v)
		if message != "" {
			return syntax, message
		}
		for j := 0; j < len(syntax.Forms); j++ {
			old := syntax.Forms[j]
			if old.Mnemonic != form.Mnemonic || len(old.Parameters) != len(form.Parameters) {
				continue
			}
			same := true
			for k := 0; k < len(old.Parameters); k++ {
				if old.Parameters[k].Kind != form.Parameters[k].Kind {
					same = false
				}
			}
			if same && targetOperationWidth(v, old.Operation) == targetOperationWidth(v, form.Operation) {
				return syntax, "ambiguous assembler form " + form.Mnemonic
			}
		}
		syntax.Forms = append(syntax.Forms, form)
	}
	if syntax.Style != "att" && syntax.Style != "intel" && syntax.Style != "native" {
		return syntax, "assembler style must be att, intel, or native"
	}
	if len(syntax.Forms) == 0 {
		return syntax, "assembler syntax contains no forms"
	}
	if syntax.NewLabel != "" || syntax.BindLabel != "" {
		create, c := frontendOperation(v, syntax.NewLabel)
		bind, b := frontendOperation(v, syntax.BindLabel)
		if !c || !b || create.Result != "label" || len(create.Parameters) != 0 || len(bind.Parameters) != 1 || bind.Parameters[0].Kind != "label" || bind.Result != "" || bind.Effects.Control != "label" {
			return syntax, "assembler label operations have incompatible signatures"
		}
	}
	if syntax.Address != "" {
		address, ok := frontendOperation(v, syntax.Address)
		if !ok || address.Result != "address" || len(address.Parameters) != 2 || address.Parameters[0].Kind != "register" || address.Parameters[1].Kind != "int" {
			return syntax, "assembler address operation has an incompatible signature"
		}
	}
	return syntax, ""
}
func decodeFrontendSyntaxForm(s Statement, v TargetVocabulary) (TargetSyntaxForm, string) {
	form := TargetSyntaxForm{}
	t := s.Tokens
	if len(s.Children) != 0 || len(t) < 7 || !frontendIdentifier(t[0]) || t[1] != "(" {
		return form, "invalid assembler form signature"
	}
	form.Mnemonic = t[0]
	at := 2
	for at < len(t) && t[at] != ")" {
		if at+2 >= len(t) || !frontendIdentifier(t[at]) || t[at+1] != ":" || !frontendKind(t[at+2]) || t[at+2] == "condition" || t[at+2] == "bool" {
			return form, "invalid assembler form operand"
		}
		for i := 0; i < len(form.Parameters); i++ {
			if form.Parameters[i].Name == t[at] {
				return form, "duplicate assembler operand name"
			}
		}
		form.Parameters = append(form.Parameters, TargetParameter{Name: t[at], Kind: t[at+2]})
		at += 3
		if at < len(t) && t[at] == "," {
			at++
			if at >= len(t) || t[at] == ")" {
				return form, "trailing assembler operand comma"
			}
		} else if at < len(t) && t[at] != ")" {
			return form, "expected assembler operand comma"
		}
	}
	if at+4 >= len(t) || t[at] != ")" || t[at+1] != "=" || !frontendIdentifier(t[at+2]) || t[at+3] != "(" || t[len(t)-1] != ")" {
		return form, "assembler form must map to a declared operation"
	}
	form.Operation = t[at+2]
	op, ok := frontendOperation(v, form.Operation)
	if !ok || op.Result != "" {
		return form, "assembler form requires a declared result-free operation"
	}
	at += 4
	for at < len(t)-1 {
		argument := TargetSyntaxArgument{Parameter: -1}
		word := t[at]
		for i := 0; i < len(form.Parameters); i++ {
			if form.Parameters[i].Name == word {
				argument.Parameter = i
			}
		}
		index := len(form.Arguments)
		if index >= len(op.Parameters) {
			return form, "assembler lowering has too many operands"
		}
		if argument.Parameter >= 0 {
			if form.Parameters[argument.Parameter].Kind != op.Parameters[index].Kind {
				return form, "assembler operand type differs from the operation"
			}
		} else {
			kind := op.Parameters[index].Kind
			valid := kind == "condition" && stringIndex(v.Conditions, word) >= 0 || kind == "register" && stringIndex(v.Registers, word) >= 0 || kind == "bool" && (word == "true" || word == "false")
			if !valid {
				return form, "assembler form contains an unknown operand or invalid fixed literal"
			}
			argument.LiteralKind, argument.LiteralValue = kind, word
		}
		form.Arguments = append(form.Arguments, argument)
		at++
		if at < len(t)-1 {
			if t[at] != "," {
				return form, "expected assembler lowering comma"
			}
			at++
			if at == len(t)-1 {
				return form, "trailing assembler lowering comma"
			}
		}
	}
	if len(form.Arguments) != len(op.Parameters) {
		return form, "assembler lowering has the wrong operand count"
	}
	for i := 0; i < len(form.Parameters); i++ {
		used := false
		for j := 0; j < len(form.Arguments); j++ {
			if form.Arguments[j].Parameter == i {
				used = true
			}
		}
		if !used {
			return form, "assembler form discards a source operand"
		}
	}
	return form, ""
}
