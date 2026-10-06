package rtg

import (
	"strconv"
	"strings"
)

// InlineOperand describes the value boundary, not an expression to splice into
// assembler text. Constant must be checked by the caller's language frontend.
// Memory operands pass the address of the lvalue, never its current contents.
type InlineOperand struct {
	Name, Constraint string
	Constant         bool
	Bits             int
	Signed           bool
	Value            TargetOperand
}

type InlineBinding struct {
	Name, Kind, Register                  string
	Value                                 TargetOperand
	Output, Read, ReadWrite, EarlyClobber bool
	Bits                                  int
	Signed                                bool
	TiedTo                                int // -1 except matching inputs
}

type InlineConstraintPlan struct {
	Bindings            []InlineBinding // outputs first, then inputs, in source order
	Clobbers, Resources []string
	Memory              bool
	Diagnostics         []Diagnostic
	Ok                  bool
}

type inlineRegisterChoice struct {
	root      int
	registers []string
}

// BindInlineOperands implements GNU write-only/read-write, early-clobber,
// numbered and named matching constraints using only the selected vocabulary.
// A bounded search is necessary: greedily assigning a general class before a
// fixed-register input can reject otherwise satisfiable constraints.
func BindInlineOperands(v TargetVocabulary, outputs, inputs []InlineOperand, clobbers []string, filename string) InlineConstraintPlan {
	p := InlineConstraintPlan{}
	if !v.Ok {
		return inlineConstraintFail(p, filename, "selected frontend vocabulary is unavailable")
	}
	if len(outputs)+len(inputs) > 30 {
		return inlineConstraintFail(p, filename, "inline assembly exceeds the 30 operand limit")
	}
	for _, name := range clobbers {
		for _, width := range v.Widths {
			for _, alias := range width.Aliases {
				if alias.Name == name {
					name = alias.Register
				}
			}
		}
		if name == "memory" {
			if p.Memory {
				return inlineConstraintFail(p, filename, "duplicate memory clobber")
			}
			p.Memory = true
		} else if stringIndex(v.Registers, name) >= 0 {
			if stringIndex(p.Clobbers, name) >= 0 {
				return inlineConstraintFail(p, filename, "duplicate register clobber "+name)
			}
			p.Clobbers = append(p.Clobbers, name)
		} else {
			c, ok := targetConstraint(v, name)
			resource := name
			if ok && c.Kind == "resource" {
				resource = c.Resource
			}
			if stringIndex(v.Resources, resource) < 0 || stringIndex(p.Resources, resource) >= 0 {
				return inlineConstraintFail(p, filename, "unknown or duplicate resource clobber "+name)
			}
			p.Resources = append(p.Resources, resource)
		}
	}
	operands := append(append([]InlineOperand(nil), outputs...), inputs...)
	choices := []inlineRegisterChoice{}
	for index, operand := range operands {
		output := index < len(outputs)
		b := InlineBinding{Name: operand.Name, Output: output, TiedTo: -1, Bits: operand.Bits, Signed: operand.Signed}
		if operand.Name != "" {
			if !frontendIdentifier(operand.Name) {
				return inlineConstraintFail(p, filename, "invalid inline operand name")
			}
			for _, old := range p.Bindings {
				if old.Name == operand.Name {
					return inlineConstraintFail(p, filename, "duplicate inline operand name "+operand.Name)
				}
			}
		}
		constraint := operand.Constraint
		if output {
			if len(constraint) < 2 || (constraint[0] != '=' && constraint[0] != '+') {
				return inlineConstraintFail(p, filename, "output constraint requires = or +")
			}
			b.Read = constraint[0] == '+'
			b.ReadWrite = b.Read
			constraint = constraint[1:]
			if strings.HasPrefix(constraint, "&") {
				b.EarlyClobber = true
				constraint = constraint[1:]
			}
		} else {
			b.Read = true
		}
		// These modifiers carry contracts not implemented by this allocator;
		// accepting and ignoring one would silently miscompile the block.
		if constraint == "" || strings.ContainsAny(constraint, "=+&%,?!*# \t") {
			return inlineConstraintFail(p, filename, "unsupported inline constraint "+operand.Constraint)
		}
		tied := -1
		if !output && assemblerNumeric(constraint) {
			n, err := strconv.Atoi(constraint)
			if err != nil {
				return inlineConstraintFail(p, filename, "invalid numbered matching constraint")
			}
			tied = n
		} else if !output && strings.HasPrefix(constraint, "[") && strings.HasSuffix(constraint, "]") {
			name := constraint[1 : len(constraint)-1]
			for i := 0; i < len(outputs); i++ {
				if outputs[i].Name == name && name != "" {
					tied = i
				}
			}
			if tied < 0 {
				return inlineConstraintFail(p, filename, "unknown named matching constraint "+name)
			}
		}
		if tied >= 0 {
			if tied >= len(outputs) || p.Bindings[tied].Kind != "register" || p.Bindings[tied].Read {
				return inlineConstraintFail(p, filename, "matching constraint requires a write-only register output")
			}
			for _, old := range p.Bindings {
				if old.TiedTo == tied {
					return inlineConstraintFail(p, filename, "multiple input values match the same output")
				}
			}
			if b.Bits != p.Bindings[tied].Bits {
				return inlineConstraintFail(p, filename, "matching operands have different widths")
			}
			b.Kind, b.TiedTo = "register", tied
			p.Bindings = append(p.Bindings, b)
			continue
		}
		class, ok := targetConstraint(v, constraint)
		if !ok || class.Kind == "resource" {
			return inlineConstraintFail(p, filename, "unadvertised inline constraint "+constraint)
		}
		b.Kind = class.Kind
		if b.EarlyClobber && b.Kind != "register" {
			return inlineConstraintFail(p, filename, "early-clobber requires a register output")
		}
		if b.Kind == "immediate" {
			if output || !operand.Constant || !frontendIntegerKind(operand.Value.Kind) {
				return inlineConstraintFail(p, filename, "immediate constraint requires a constant integer input")
			}
			canonical, valid := targetLiteral(v, operand.Value)
			if !valid {
				return inlineConstraintFail(p, filename, "invalid immediate constraint value")
			}
			b.Value = TargetOperand{Kind: operand.Value.Kind, Value: canonical}
		} else {
			available := []string{}
			for _, reg := range class.Registers {
				_, widthOk := targetRegisterSpelling(v, reg, b.Bits)
				if stringIndex(p.Clobbers, reg) < 0 && (b.Kind == "memory" || widthOk) {
					available = append(available, reg)
				}
			}
			if len(available) == 0 || len(available) > 64 {
				return inlineConstraintFail(p, filename, "constraint has no usable register")
			}
			choices = append(choices, inlineRegisterChoice{root: index, registers: available})
			// An output memory operand always needs an incoming address even
			// when the pointed-to value is write-only.
			if b.Kind == "memory" {
				b.Read = true
			}
		}
		p.Bindings = append(p.Bindings, b)
	}
	implicit := 0
	for _, b := range p.Bindings {
		if b.ReadWrite {
			implicit++
		}
	}
	if len(p.Bindings)+implicit > 30 {
		return inlineConstraintFail(p, filename, "read-write operands exceed the 30 operand limit")
	}
	// Try the tightest classes first; restore source order in the result.
	for i := 0; i < len(choices); i++ {
		best := i
		for j := i + 1; j < len(choices); j++ {
			if len(choices[j].registers) < len(choices[best].registers) {
				best = j
			}
		}
		choices[i], choices[best] = choices[best], choices[i]
	}
	budget := 65536
	if !assignInlineRegisters(&p, choices, 0, &budget) {
		message := "inline register constraints are unsatisfiable"
		if budget == 0 {
			message = "inline constraint search limit exceeded"
		}
		return inlineConstraintFail(p, filename, message)
	}
	for i := range p.Bindings {
		if p.Bindings[i].TiedTo >= 0 {
			p.Bindings[i].Register = p.Bindings[p.Bindings[i].TiedTo].Register
		}
	}
	p.Ok = true
	return p
}

func inlineConstraintFail(p InlineConstraintPlan, filename, message string) InlineConstraintPlan {
	p.Ok = false
	p.Diagnostics = []Diagnostic{{Filename: filename, Code: "RTG-INLINE-001", Message: message}}
	return p
}
func inlineRoot(p *InlineConstraintPlan, index int) int {
	if p.Bindings[index].TiedTo >= 0 {
		return p.Bindings[index].TiedTo
	}
	return index
}
func inlineInterferes(p *InlineConstraintPlan, a, b int) bool {
	// Matching operands represent a single value/register, not interference.
	if inlineRoot(p, a) == inlineRoot(p, b) {
		return false
	}
	x, y := p.Bindings[a], p.Bindings[b]
	if x.Kind == "immediate" || y.Kind == "immediate" {
		return false
	}
	if x.Output && y.Output {
		return true
	}
	if x.Read && y.Read {
		return true
	}
	if x.Output && x.EarlyClobber && y.Read || y.Output && y.EarlyClobber && x.Read {
		return true
	}
	return false
}
func assignInlineRegisters(p *InlineConstraintPlan, choices []inlineRegisterChoice, at int, budget *int) bool {
	if at == len(choices) {
		return true
	}
	choice := choices[at]
	for _, reg := range choice.registers {
		if *budget == 0 {
			return false
		}
		*budget--
		conflict := false
		for a := range p.Bindings {
			if inlineRoot(p, a) != choice.root {
				continue
			}
			for b := range p.Bindings {
				root := inlineRoot(p, b)
				if p.Bindings[root].Register == reg && inlineInterferes(p, a, b) {
					conflict = true
				}
			}
		}
		if conflict {
			continue
		}
		p.Bindings[choice.root].Register = reg
		if assignInlineRegisters(p, choices, at+1, budget) {
			return true
		}
		p.Bindings[choice.root].Register = ""
	}
	return false
}

// ExpandInlineTemplate substitutes bound operand data, never source
// expressions. LabelNames are compiler-generated local labels in GNU operand
// numbering (outputs+inputs, with the extra implicit inputs of '+' outputs).
// The caller supplies a unique integer for %= and maps asm-goto labels to its
// own control-flow exits before materializing the returned instruction text.
type InlineLabel struct{ Name, Symbol string }

func ExpandInlineTemplate(v TargetVocabulary, plan InlineConstraintPlan, syntaxName string, template []byte, labels []InlineLabel, unique int, filename string) ([]byte, []Diagnostic) {
	fail := func(message string) ([]byte, []Diagnostic) {
		return nil, []Diagnostic{{Filename: filename, Code: "RTG-INLINE-002", Message: message}}
	}
	if !plan.Ok || unique < 0 || len(template) > maxDefinitionBytes {
		return fail("invalid inline template plan")
	}
	syntax := TargetSyntax{}
	for _, candidate := range v.Syntaxes {
		if candidate.Name == syntaxName {
			syntax = candidate
		}
	}
	if !v.Ok || syntax.Name == "" {
		return fail("inline assembler syntax is unavailable")
	}
	for _, b := range plan.Bindings {
		if b.Kind == "register" || b.Kind == "memory" {
			if !frontendIdentifier(b.Register) || stringIndex(v.Registers, b.Register) < 0 {
				return fail("invalid inline register binding")
			}
		} else if b.Kind == "immediate" {
			_, valid := targetLiteral(v, b.Value)
			if !frontendIntegerKind(b.Value.Kind) || !valid {
				return fail("invalid inline immediate binding")
			}
		} else {
			return fail("invalid inline binding kind")
		}
	}
	for i, label := range labels {
		if !frontendIdentifier(label.Name) || !assemblerSymbol(label.Symbol) || assemblerNumeric(label.Symbol) {
			return fail("invalid inline control-flow label")
		}
		for j := 0; j < i; j++ {
			if labels[j].Name == label.Name || labels[j].Symbol == label.Symbol {
				return fail("duplicate inline control-flow label")
			}
		}
	}
	var out strings.Builder
	for at := 0; at < len(template); at++ {
		if template[at] != '%' {
			out.WriteByte(template[at])
			continue
		}
		at++
		if at == len(template) {
			return fail("incomplete inline template escape")
		}
		if template[at] == '%' {
			out.WriteByte('%')
			continue
		}
		if template[at] == '=' {
			out.WriteString(strconv.Itoa(unique))
			continue
		}
		modifier := byte(0)
		widthModifier := false
		for _, width := range v.Widths {
			if width.Modifier[0] == template[at] {
				widthModifier = true
			}
		}
		if template[at] == 'c' || template[at] == 'l' || widthModifier {
			modifier = template[at]
			at++
		}
		if at == len(template) {
			return fail("missing inline operand reference")
		}
		index := -1
		if template[at] == '[' {
			end := at + 1
			for end < len(template) && template[end] != ']' {
				end++
			}
			if end == len(template) {
				return fail("unterminated named inline operand")
			}
			name := string(template[at+1 : end])
			if modifier == 'l' {
				for i, label := range labels {
					if label.Name == name {
						index = i
					}
				}
				if index < 0 {
					return fail("unknown named asm-goto label")
				}
				out.WriteString(labels[index].Symbol)
				at = end
				continue
			}
			for i, b := range plan.Bindings {
				if b.Name == name && name != "" {
					index = i
				}
			}
			at = end
		} else {
			end := at
			for end < len(template) && template[end] >= '0' && template[end] <= '9' {
				end++
			}
			if end == at {
				return fail("unsupported inline operand modifier")
			}
			n, err := strconv.Atoi(string(template[at:end]))
			if err != nil {
				return fail("invalid inline operand number")
			}
			index, at = n, end-1
		}
		if modifier == 'l' {
			base := len(plan.Bindings)
			for _, b := range plan.Bindings {
				if b.ReadWrite {
					base++
				}
			}
			index -= base
			if index < 0 || index >= len(labels) {
				return fail("invalid asm-goto label number")
			}
			out.WriteString(labels[index].Symbol)
			continue
		}
		if index < 0 || index >= len(plan.Bindings) {
			return fail("unknown inline operand reference")
		}
		b := plan.Bindings[index]
		if modifier == 'c' && b.Kind != "immediate" {
			return fail("c modifier requires an immediate operand")
		}
		if widthModifier && b.Kind != "register" {
			return fail("register width modifier requires a register operand")
		}
		switch b.Kind {
		case "register":
			if syntax.Style == "att" {
				out.WriteByte('%')
			}
			bits := b.Bits
			if widthModifier {
				for _, width := range v.Widths {
					if width.Modifier[0] == modifier {
						bits = width.Bits
					}
				}
			}
			spelling, valid := targetRegisterSpelling(v, b.Register, bits)
			if !valid {
				return fail("backend has no register spelling for operand width")
			}
			out.WriteString(spelling)
		case "memory":
			if syntax.Style == "att" {
				out.WriteString("(%" + b.Register + ")")
			} else if syntax.Style == "intel" {
				out.WriteString("[" + b.Register + "]")
			} else {
				out.WriteString("0(" + b.Register + ")")
			}
		case "immediate":
			if syntax.Style == "att" && modifier != 'c' {
				out.WriteByte('$')
			}
			out.WriteString(b.Value.Value)
		default:
			return fail("invalid inline operand binding")
		}
		if out.Len() > maxDefinitionBytes {
			return fail("expanded inline template exceeds the source limit")
		}
	}
	return []byte(out.String()), nil
}
