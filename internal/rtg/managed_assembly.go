package rtg

import (
	"strconv"
	"strings"
)

// ManagedBlock is an ordered physical body at a compiler-managed word boundary.
// Inputs are runtime values supplied by the expression compiler. Register data
// may use argument(N) in the serialized IR (logical parameter order, independent
// of incoming call-word pop order); Result names a physical register to
// capture into the compiler result, or is empty for a void block. The compiler
// supplies call-compatible spill/evaluation semantics, but direct uses emit the
// body inline, without CALL/RET or a user-owned function ABI.
type ManagedBlock struct {
	Block  TargetBlock
	Inputs int
	Result string
}

func EncodeManagedAssembly(v TargetVocabulary, blocks []ManagedBlock, filename string) ([]byte, []Diagnostic) {
	fail := func(message string) ([]byte, []Diagnostic) {
		return nil, []Diagnostic{{Filename: filename, Code: "RTG-MANAGED-002", Message: message}}
	}
	if !v.Ok || !v.Managed.Ok || len(blocks) == 0 {
		return fail("selected backend has no managed word boundary")
	}
	// Reuse canonical typed-body serialization; word metadata is not an encoder
	// expression and cannot expose backend implementation names.
	bodies := []TargetBlock{}
	for _, b := range blocks {
		block, message := validateManagedBlock(v, b)
		if message != "" {
			return fail(message)
		}
		bodies = append(bodies, block)
	}
	encoded, ds := EncodeTargetAssembly(v, bodies, filename)
	if len(ds) != 0 {
		return nil, ds
	}
	parsed := ParseAssembly(encoded, filename)
	var out strings.Builder
	out.WriteString("rtgasm 3\nassembly {\n")
	for i, entry := range parsed.Entries {
		b := blocks[i]
		out.WriteString(entry.Name + "(out:emitter) {\ninputs = " + strconv.Itoa(b.Inputs) + "\n")
		// The final move in the validated body was synthesized for capture.
		// Serialize the original body instead, so revalidation owns capture.
		original := bodies[i]
		if b.Result != "" {
			original.Instructions = original.Instructions[:len(original.Instructions)-1]
		}
		body, diagnostics := EncodeTargetAssembly(v, []TargetBlock{original}, filename)
		if len(diagnostics) != 0 {
			return nil, diagnostics
		}
		doc := ParseAssembly(body, filename)
		out.Write(body[doc.Entries[0].BodyStart:doc.Entries[0].BodyEnd])
		out.WriteString("\nyield(")
		if b.Result != "" {
			out.WriteString("register(" + b.Result + ")")
		}
		out.WriteString(")\n}\n")
		if out.Len() > maxDefinitionBytes {
			return fail("managed blocks exceed the source limit")
		}
	}
	out.WriteString("}\n")
	return []byte(out.String()), nil
}

func LowerManagedAssembly(resolved ResolveResult, targetName string, assembly AssemblyDocument) AssemblyDocument {
	v := FrontendOperations(resolved, targetName)
	if !v.Ok || !v.Managed.Ok {
		return assemblyFail(assembly, Span{}, "RTG-MANAGED-002", "selected backend has no managed word boundary")
	}
	for i := range assembly.Entries {
		entry := assembly.Entries[i]
		if len(entry.Steps) < 2 {
			return assemblyFail(assembly, entry.Span, "RTG-MANAGED-002", "managed block requires inputs and a final yield")
		}
		header := entry.Steps[0]
		if len(header.Children) != 0 || len(header.Tokens) != 3 || header.Tokens[0] != "inputs" || header.Tokens[1] != "=" {
			return assemblyFail(assembly, header.Span, "RTG-MANAGED-002", "managed block requires an input word count")
		}
		n, err := strconv.Atoi(header.Tokens[2])
		if err != nil {
			return assemblyFail(assembly, header.Span, "RTG-MANAGED-002", "invalid managed input count")
		}
		if len(entry.Steps[1].Tokens) != 0 && entry.Steps[1].Tokens[0] == "intrinsic" {
			steps, message := lowerManagedIntrinsic(v, entry, n)
			if message != "" {
				return assemblyFail(assembly, entry.Span, "RTG-INTRINSIC-002", message)
			}
			entry.Steps, entry.ManagedInputs, entry.ManagedOutputs = steps, n, 1
			assembly.Entries[i] = entry
			continue
		}
		block := ManagedBlock{Block: TargetBlock{Name: entry.Name}, Inputs: n}
		for step := 1; step < len(entry.Steps); step++ {
			instruction, message := decodeTargetInstruction(entry.Steps[step])
			if message != "" {
				return assemblyFail(assembly, entry.Steps[step].Span, "RTG-MANAGED-002", message)
			}
			if step+1 == len(entry.Steps) {
				if instruction.Operation != "yield" || instruction.Result != "" || len(instruction.Operands) > 1 {
					return assemblyFail(assembly, instruction.Span, "RTG-MANAGED-002", "managed block requires one final yield")
				}
				if len(instruction.Operands) == 1 {
					if instruction.Operands[0].Kind != "register" {
						return assemblyFail(assembly, instruction.Span, "RTG-MANAGED-002", "managed yield requires a register")
					}
					block.Result = instruction.Operands[0].Value
				}
				continue
			}
			for operand := range instruction.Operands {
				value := instruction.Operands[operand]
				if value.Kind == "argument" {
					index, e := strconv.Atoi(value.Value)
					if e != nil || index < 0 || index >= n || index >= len(v.Managed.Arguments) {
						return assemblyFail(assembly, instruction.Span, "RTG-MANAGED-002", "invalid managed argument reference")
					}
					instruction.Operands[operand] = TargetOperand{Kind: "register", Value: v.Managed.Arguments[n-1-index]}
				}
			}
			block.Block.Instructions = append(block.Block.Instructions, instruction)
		}
		normalized, message := validateManagedBlock(v, block)
		if message != "" {
			return assemblyFail(assembly, entry.Span, "RTG-MANAGED-002", message)
		}
		steps, message := validateTargetBlock(v, normalized)
		if message != "" {
			return assemblyFail(assembly, entry.Span, "RTG-MANAGED-002", message)
		}
		entry.Steps, entry.ManagedInputs = steps, n
		if block.Result != "" {
			entry.ManagedOutputs = 1
		}
		assembly.Entries[i] = entry
	}
	assembly.Version = 1 // internal sequence only; original Source retains mode
	return assembly
}

func validateManagedBlock(v TargetVocabulary, b ManagedBlock) (TargetBlock, string) {
	if !v.Managed.Ok || b.Inputs < 0 || b.Inputs > len(v.Managed.Arguments) {
		return b.Block, "invalid managed input word count"
	}
	block := b.Block
	// Own the slice so result capture never modifies frontend-owned data.
	block.Instructions = append([]TargetInstruction(nil), b.Block.Instructions...)
	for i := range block.Instructions {
		block.Instructions[i].Operands = append([]TargetOperand(nil), block.Instructions[i].Operands...)
		for j, operand := range block.Instructions[i].Operands {
			if operand.Kind == "argument" {
				n, err := strconv.Atoi(operand.Value)
				if err != nil || n < 0 || n >= b.Inputs {
					return block, "invalid managed argument reference"
				}
				block.Instructions[i].Operands[j] = TargetOperand{Kind: "register", Value: v.Managed.Arguments[b.Inputs-1-n]}
			}
		}
	}
	for _, instruction := range block.Instructions {
		op, ok := frontendOperation(v, instruction.Operation)
		if !ok {
			return block, "unknown managed operation " + instruction.Operation
		}
		if op.Effects.Control == "call" || op.Effects.Control == "return" {
			return block, "managed blocks cannot call or return through the user ABI"
		}
		if op.Result == "register" {
			return block, "managed register results require an allocation contract"
		}
		for _, operand := range instruction.Operands {
			if operand.Kind == "register" && stringIndex(v.Managed.Registers, operand.Value) < 0 {
				return block, "managed block references a reserved or preserved register " + operand.Value
			}
		}
		effects := append(append(append([]string(nil), op.Effects.Reads...), op.Effects.Writes...), op.Effects.Clobbers...)
		for _, effect := range effects {
			if stringIndex(v.Registers, effect) >= 0 && stringIndex(v.Managed.Registers, effect) < 0 {
				return block, "managed operation affects a reserved or preserved register " + effect
			}
			if stringIndex(v.Resources, effect) >= 0 && stringIndex(v.Managed.Resources, effect) < 0 {
				return block, "managed operation affects an unavailable resource " + effect
			}
		}
	}
	if b.Result != "" {
		if stringIndex(v.Managed.Registers, b.Result) < 0 {
			return block, "invalid managed result register"
		}
		block.Instructions = append(block.Instructions, TargetInstruction{Operation: v.Managed.Move, Operands: []TargetOperand{{Kind: "register", Value: v.Managed.Result}, {Kind: "register", Value: b.Result}}})
	}
	_, message := validateTargetBlock(v, block)
	return block, message
}
