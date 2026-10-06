package rtg

import "strconv"

func lowerManagedIntrinsic(v TargetVocabulary, entry AssemblyEntry, inputs int) ([]Statement, string) {
	if len(entry.Steps) != 3 {
		return nil, "intrinsic block requires exactly one intrinsic and a final yield"
	}
	declaration := entry.Steps[1]
	if len(declaration.Children) != 0 || len(declaration.Tokens) != 3 || declaration.Tokens[1] != "=" {
		return nil, "invalid intrinsic selection"
	}
	intrinsic, found := targetIntrinsic(v, declaration.Tokens[2])
	if !found || inputs != len(intrinsic.Parameters) || inputs > len(v.Managed.Arguments) {
		return nil, "unknown intrinsic or incompatible runtime input count"
	}
	yield, message := decodeTargetInstruction(entry.Steps[2])
	if message != "" || yield.Operation != "yield" || yield.Result != "" || len(yield.Operands) != 0 {
		return nil, "intrinsic result placement is compiler-owned; use yield()"
	}
	result := ""
	for _, register := range v.Managed.Registers {
		if stringIndex(v.Managed.Arguments[:inputs], register) < 0 {
			result = register
			break
		}
	}
	if result == "" {
		return nil, "intrinsic result has no available managed register"
	}
	// Private encoder operation exists only while lowering this entry. It never
	// enters the public physical vocabulary, so version 2 cannot invoke it.
	op := intrinsic.operation
	op.Name = "renvoAllocatedIntrinsic"
	for suffix := 0; ; suffix++ {
		if _, exists := frontendOperation(v, op.Name); !exists {
			break
		}
		op.Name = "renvoAllocatedIntrinsic" + strconv.Itoa(suffix)
	}
	v.Operations = append(append([]TargetOperation(nil), v.Operations...), op)
	instruction := TargetInstruction{Operation: op.Name, Span: declaration.Span, Operands: []TargetOperand{{Kind: "register", Value: result}}}
	for i := 0; i < inputs; i++ {
		instruction.Operands = append(instruction.Operands, TargetOperand{Kind: "register", Value: v.Managed.Arguments[inputs-1-i]})
	}
	body, message := validateManagedBlock(v, ManagedBlock{Block: TargetBlock{Name: entry.Name, Instructions: []TargetInstruction{instruction}}, Inputs: inputs, Result: result})
	if message != "" {
		return nil, message
	}
	return validateTargetBlock(v, body)
}
