package rtg

import (
	"strconv"
	"strings"
)

// InlineWord identifies one runtime word required by an inline block. Operand
// indexes the outputs-then-inputs array. Address passes the lvalue address; an
// output register's address is retained privately for single-evaluation capture.
type InlineWord struct {
	Operand int
	Address bool
}
type InlineAssembly struct {
	Block       ManagedBlock
	Plan        InlineConstraintPlan
	Words       []InlineWord
	Diagnostics []Diagnostic
	Ok          bool
}

// BuildInlineAssembly lowers a GNU operand/template contract at the selected
// compiler boundary. It never evaluates frontend expressions or emits opcodes.
// Exit selectors are zero for fallthrough and one-based for asm-goto labels.
func BuildInlineAssembly(v TargetVocabulary, syntaxName string, template []byte, outputs, inputs []InlineOperand, clobbers []string, labels []InlineLabel, name, filename string, unique int) InlineAssembly {
	r := InlineAssembly{}
	fail := func(message string) InlineAssembly {
		r.Diagnostics = []Diagnostic{{Filename: filename, Code: "RTG-INLINE-001", Message: message}}
		return r
	}
	if !v.Ok || !v.Managed.Ok {
		return fail("selected backend has no managed inline boundary")
	}
	policy := v.Managed
	for _, operand := range append(append([]InlineOperand(nil), outputs...), inputs...) {
		if operand.Constant || operand.Bits == 0 || operand.Bits == v.Target.WordBits {
			continue
		}
		if _, ok := targetWidth(v, operand.Bits); !ok || operand.Bits > v.Target.WordBits {
			return fail("selected backend does not support the C operand width")
		}
	}
	// These operations are backend-selected, and their shapes are checked here
	// before any operand indexing. No architecture or mnemonic switch exists.
	contracts := []struct {
		name, result, memory, control string
		kinds                         []string
	}{
		{policy.Address, "address", "none", "none", []string{"register", "int"}},
		{policy.Load, "", "read", "none", []string{"register", "address"}},
		{policy.Store, "", "write", "none", []string{"address", "register"}},
		{policy.Immediate, "", "none", "none", []string{"register", "int64"}},
		{policy.Jump, "", "none", "jump", []string{"label"}},
		{policy.NewLabel, "label", "none", "none", nil},
		{policy.BindLabel, "", "none", "label", []string{"label"}},
	}
	for _, c := range contracts {
		op, ok := frontendOperation(v, c.name)
		if !ok || op.Result != c.result || op.Effects.Memory != c.memory || op.Effects.Control != c.control || len(op.Parameters) != len(c.kinds) {
			return fail("backend inline boundary has an incompatible operation " + c.name)
		}
		for i, kind := range c.kinds {
			if op.Parameters[i].Kind != kind {
				return fail("backend inline boundary has an incompatible operand " + c.name)
			}
		}
	}
	// Allocate user operands first, including fixed classes. Only then choose
	// private capture locations from unused caller-owned registers. Reserving
	// registers greedily before constraint solving can steal a fixed input.
	registers := append([]string(nil), policy.Registers...)
	constrained := v
	constrained.Constraints = append([]TargetConstraint(nil), v.Constraints...)
	for i, c := range constrained.Constraints {
		c.Registers = nil
		for _, register := range v.Constraints[i].Registers {
			if stringIndex(registers, register) >= 0 {
				c.Registers = append(c.Registers, register)
			}
		}
		constrained.Constraints[i] = c
	}
	r.Plan = BindInlineOperands(constrained, outputs, inputs, clobbers, filename)
	if !r.Plan.Ok {
		r.Diagnostics = r.Plan.Diagnostics
		return r
	}
	outputRegisters := 0
	used := append([]string(nil), r.Plan.Clobbers...)
	for _, b := range r.Plan.Bindings {
		if b.Output && b.Kind == "register" {
			outputRegisters++
		}
		if b.Register != "" {
			used = append(used, b.Register)
		}
	}
	reserved := []string{}
	for i := len(registers) - 1; i >= 0 && len(reserved) < outputRegisters; i-- {
		if stringIndex(used, registers[i]) < 0 {
			reserved = append(reserved, registers[i])
		}
	}
	if len(reserved) != outputRegisters {
		return fail("no registers remain for output-address capture")
	}
	for _, register := range r.Plan.Clobbers {
		if stringIndex(policy.Registers, register) < 0 {
			return fail("inline clobber references a preserved compiler register")
		}
	}
	for _, resource := range r.Plan.Resources {
		if stringIndex(policy.Resources, resource) < 0 {
			return fail("inline clobber references an unavailable compiler resource")
		}
	}
	type copyWord struct{ destination, source string }
	copies := []copyWord{}
	pointers := make([]string, len(outputs))
	cursor := 0
	// Addresses are always evaluated once, even for +r outputs. Initial loads
	// happen after the parallel runtime-word placement, through retained pointers.
	for i, b := range r.Plan.Bindings {
		if b.Output && b.Kind == "register" {
			pointers[i] = reserved[cursor]
			cursor++
			copies = append(copies, copyWord{destination: pointers[i]})
			r.Words = append(r.Words, InlineWord{Operand: i, Address: true})
		} else if b.Kind == "memory" || (!b.Output && b.Kind == "register") {
			copies = append(copies, copyWord{destination: b.Register})
			r.Words = append(r.Words, InlineWord{Operand: i, Address: b.Kind == "memory"})
		}
	}
	if len(r.Words) > len(policy.Arguments) {
		return fail("inline block exceeds the managed runtime-word limit")
	}
	copyDestinations := []string{}
	for i := range copies {
		copies[i].source = policy.Arguments[len(copies)-1-i]
		copyDestinations = append(copyDestinations, copies[i].destination)
	}
	block := TargetBlock{Name: name}
	reg := func(name string) TargetOperand { return TargetOperand{Kind: "register", Value: name} }
	emit := func(op string, operands ...TargetOperand) {
		block.Instructions = append(block.Instructions, TargetInstruction{Operation: op, Operands: operands})
	}
	// Parallel copy scheduling prevents ABI/constraint register cycles from
	// losing an input. A scratch is needed only for an actual cycle.
	for len(copies) > 0 {
		ready := -1
		for i, c := range copies {
			used := false
			for j, other := range copies {
				if i != j && other.source == c.destination {
					used = true
				}
			}
			if !used || c.destination == c.source {
				ready = i
				break
			}
		}
		if ready >= 0 {
			c := copies[ready]
			if c.destination != c.source {
				emit(policy.Move, reg(c.destination), reg(c.source))
			}
			copies = append(copies[:ready], copies[ready+1:]...)
			continue
		}
		scratch := ""
		for _, register := range registers {
			busy := stringIndex(copyDestinations, register) >= 0
			for _, c := range copies {
				if c.destination == register || c.source == register {
					busy = true
				}
			}
			if !busy {
				scratch = register
				break
			}
		}
		if scratch == "" {
			return fail("inline parallel-copy cycle has no scratch register")
		}
		source := copies[0].source
		emit(policy.Move, reg(scratch), reg(source))
		for i := range copies {
			if copies[i].source == source {
				copies[i].source = scratch
			}
		}
	}
	for i, b := range r.Plan.Bindings {
		if b.Output && b.Kind == "register" && b.ReadWrite {
			handle := "renvoInlineInitial" + strconv.Itoa(i)
			block.Instructions = append(block.Instructions, TargetInstruction{Operation: policy.Address, Result: handle, Operands: []TargetOperand{reg(pointers[i]), {Kind: "int", Value: "0"}}})
			load := policy.Load
			if width, ok := targetWidth(v, b.Bits); ok {
				load = width.Load
				if b.Signed {
					load = width.LoadSigned
				}
			}
			emit(load, reg(b.Register), TargetOperand{Kind: "value", Value: handle})
		}
	}
	expanded, diagnostics := ExpandInlineTemplate(v, r.Plan, syntaxName, template, labels, unique, filename)
	if len(diagnostics) != 0 {
		r.Diagnostics = diagnostics
		return r
	}
	originalEnd := len(expanded)
	stubOffsets := []int{}
	for _, label := range labels {
		expanded = append(expanded, '\n')
		stubOffsets = append(stubOffsets, len(expanded))
		expanded = append(expanded, []byte(label.Symbol+":\n")...)
	}
	parsed := TargetBlock{Name: name}
	if strings.TrimSpace(string(expanded)) != "" {
		parsed, diagnostics = ParseTargetInstructionBlock(v, syntaxName, expanded, filename, name)
		if len(diagnostics) != 0 {
			r.Diagnostics = diagnostics
			return r
		}
	}
	// Verify declared destruction before adding compiler-owned captures/stubs.
	allowed := append([]string(nil), r.Plan.Clobbers...)
	available := append([]string(nil), allowed...)
	for _, b := range r.Plan.Bindings {
		if b.Register != "" {
			available = append(available, b.Register)
		}
		if b.Output && b.Kind == "register" {
			allowed = append(allowed, b.Register)
		}
	}
	for _, instruction := range parsed.Instructions {
		if instruction.Span.Start.Offset >= originalEnd && instruction.Operation == policy.BindLabel {
			continue
		}
		op, _ := frontendOperation(v, instruction.Operation)
		if op.Effects.Control == "call" || op.Effects.Control == "return" {
			return fail("inline templates cannot call or return through a user ABI")
		}
		for _, operand := range instruction.Operands {
			if operand.Kind == "register" && stringIndex(available, operand.Value) < 0 {
				return fail("inline template references an undeclared register " + operand.Value)
			}
		}
		for _, effect := range op.Effects.Reads {
			actual := effect
			if p := frontendParameter(op, effect); p >= 0 && instruction.Operands[p].Kind == "register" {
				actual = instruction.Operands[p].Value
			}
			if stringIndex(v.Registers, actual) >= 0 && stringIndex(available, actual) < 0 {
				return fail("inline template reads an undeclared register " + actual)
			}
		}
		for _, list := range [][]string{op.Effects.Writes, op.Effects.Clobbers} {
			for _, effect := range list {
				actual := effect
				if p := frontendParameter(op, effect); p >= 0 && instruction.Operands[p].Kind == "register" {
					actual = instruction.Operands[p].Value
				}
				if stringIndex(v.Registers, actual) >= 0 && stringIndex(allowed, actual) < 0 {
					return fail("inline template destroys an undeclared register " + actual)
				}
				if stringIndex(v.Resources, actual) >= 0 && stringIndex(r.Plan.Resources, actual) < 0 {
					return fail("inline template destroys an undeclared resource " + actual)
				}
			}
		}
	}
	join := "renvoInlineCaptureJoin"
	if len(labels) != 0 {
		block.Instructions = append(block.Instructions, TargetInstruction{Operation: policy.NewLabel, Result: join})
	}
	// Keep the exit selector in a reserved output pointer register only after
	// captures; use a free register so capture itself cannot overwrite it.
	selector := ""
	if len(labels) != 0 {
		for _, register := range registers {
			if stringIndex(available, register) < 0 && stringIndex(reserved, register) < 0 {
				selector = register
				break
			}
		}
		if selector == "" {
			return fail("asm goto has no register for its exit selector")
		}
	}
	capture := func(exit int) {
		for i, b := range r.Plan.Bindings {
			if b.Output && b.Kind == "register" {
				handle := "renvoInlineCapture" + strconv.Itoa(exit) + "_" + strconv.Itoa(i)
				block.Instructions = append(block.Instructions, TargetInstruction{Operation: policy.Address, Result: handle, Operands: []TargetOperand{reg(pointers[i]), {Kind: "int", Value: "0"}}})
				store := policy.Store
				if width, ok := targetWidth(v, b.Bits); ok {
					store = width.Store
				}
				emit(store, TargetOperand{Kind: "value", Value: handle}, reg(b.Register))
			}
		}
		if len(labels) != 0 {
			emit(policy.Immediate, reg(selector), TargetOperand{Kind: "int64", Value: strconv.Itoa(exit)})
			emit(policy.Jump, TargetOperand{Kind: "value", Value: join})
		}
	}
	stub := 0
	captured := false
	for _, instruction := range parsed.Instructions {
		if stub < len(stubOffsets) && instruction.Operation == policy.BindLabel && instruction.Span.Start.Offset == stubOffsets[stub] {
			if !captured {
				capture(0)
				captured = true
			}
			block.Instructions = append(block.Instructions, instruction)
			capture(stub + 1)
			stub++
		} else {
			block.Instructions = append(block.Instructions, instruction)
		}
	}
	if !captured {
		capture(0)
	}
	if len(labels) != 0 {
		emit(policy.BindLabel, TargetOperand{Kind: "value", Value: join})
	}
	if len(block.Instructions) == 0 {
		emit(policy.Move, reg(policy.Result), reg(policy.Result))
	}
	r.Block = ManagedBlock{Block: block, Inputs: len(r.Words), Result: selector}
	if _, message := validateManagedBlock(v, r.Block); message != "" {
		return fail(message)
	}
	r.Ok = true
	return r
}
