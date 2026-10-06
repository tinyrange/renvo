//go:build !renvo

package rtg

import (
	"strconv"
	"strings"
)

// TargetBlock is an explicit ordered machine body, not a compiler-managed SSA
// region. Result names refer to encoder handles (labels/addresses/etc.), not to
// runtime Go values. The function's caller-facing ABI remains the backend ABI.
type TargetBlock struct {
	Name         string
	Instructions []TargetInstruction
}
type TargetInstruction struct {
	Operation string
	Result    string
	Operands  []TargetOperand
	Span      Span
}

// Kind is a contract type for literals, or "value" for a preceding result.
// Value is data, never a source expression or an arbitrary helper name.
type TargetOperand struct{ Kind, Value string }

// EncodeTargetAssembly lets custom frontends produce the same source-preserved
// unit payload as the typed assembler frontend, without constructing Go text.
// Validation occurs here and again against the selected backend before emission.
func EncodeTargetAssembly(v TargetVocabulary, blocks []TargetBlock, filename string) ([]byte, []Diagnostic) {
	if !v.Ok {
		diagnostics := v.Diagnostics
		if len(diagnostics) == 0 {
			diagnostics = []Diagnostic{{Filename: filename, Code: "RTG-FRONTEND-005", Message: "frontend vocabulary is unavailable"}}
		}
		return nil, diagnostics
	}
	if len(blocks) == 0 || len(blocks) > maxStatementsPerBody {
		return nil, []Diagnostic{{Filename: filename, Code: "RTG-FRONTEND-005", Message: "invalid machine block count"}}
	}
	var out strings.Builder
	out.WriteString("rtgasm 2\nassembly {\n")
	seen := map[string]bool{}
	for _, block := range blocks {
		if !frontendIdentifier(block.Name) || seen[block.Name] {
			return nil, []Diagnostic{{Filename: filename, Code: "RTG-FRONTEND-005", Message: "invalid or duplicate machine block name"}}
		}
		seen[block.Name] = true
		if _, message := validateTargetBlock(v, block); message != "" {
			return nil, []Diagnostic{{Filename: filename, Code: "RTG-FRONTEND-005", Message: message}}
		}
		out.WriteString(block.Name + "(out:emitter) {\n")
		for _, instruction := range block.Instructions {
			if instruction.Result != "" {
				out.WriteString("let " + instruction.Result + " = ")
			}
			out.WriteString(instruction.Operation + "(")
			for i, operand := range instruction.Operands {
				if i != 0 {
					out.WriteString(", ")
				}
				if operand.Kind == "value" {
					out.WriteString(operand.Value)
				} else {
					canonical, _ := targetLiteral(v, operand)
					out.WriteString(operand.Kind + "(" + canonical + ")")
				}
			}
			out.WriteString(")\n")
		}
		out.WriteString("}\n")
		if out.Len() > maxDefinitionBytes {
			return nil, []Diagnostic{{Filename: filename, Code: "RTG-FRONTEND-005", Message: "machine blocks exceed the source limit"}}
		}
	}
	out.WriteString("}\n")
	source := []byte(out.String())
	parsed := ParseAssembly(source, filename)
	if !parsed.Ok {
		return nil, parsed.Diagnostics
	}
	return source, nil
}

// LowerTargetAssembly checks typed assembly before it can reach the legacy
// bounded-sequence evaluator. Only advertised operations, typed literal data,
// and preceding results can cross this boundary; there is no expression escape.
func LowerTargetAssembly(resolved ResolveResult, targetName string, assembly AssemblyDocument) AssemblyDocument {
	if !assembly.Ok || assembly.Version != 2 {
		return assembly
	}
	v := FrontendOperations(resolved, targetName)
	if !v.Ok {
		assembly.Ok = false
		assembly.Diagnostics = append(assembly.Diagnostics, v.Diagnostics...)
		return assembly
	}
	entries := make([]AssemblyEntry, len(assembly.Entries))
	for i, entry := range assembly.Entries {
		block := TargetBlock{Name: entry.Name}
		for _, step := range entry.Steps {
			instruction, message := decodeTargetInstruction(step)
			if message != "" {
				return assemblyFail(assembly, step.Span, "RTG-FRONTEND-005", message)
			}
			block.Instructions = append(block.Instructions, instruction)
		}
		steps, message := validateTargetBlock(v, block)
		if message != "" {
			span := entry.Span
			// validateTargetBlock annotates the offending step, including label
			// obligations diagnosed after walking the body.
			if len(steps) != 0 {
				span = steps[len(steps)-1].Span
			}
			return assemblyFail(assembly, span, "RTG-FRONTEND-005", message)
		}
		entry.Steps = steps
		entries[i] = entry
	}
	assembly.Entries = entries
	assembly.Version = 1 // internal lowered sequence, never source serialization
	return assembly
}

func decodeTargetInstruction(step Statement) (TargetInstruction, string) {
	instruction := TargetInstruction{Span: step.Span}
	t := step.Tokens
	if len(step.Children) != 0 {
		return instruction, "machine blocks cannot contain nested source"
	}
	if len(t) >= 3 && t[0] == "let" {
		if !frontendIdentifier(t[1]) || t[2] != "=" {
			return instruction, "invalid machine result binding"
		}
		instruction.Result = t[1]
		t = t[3:]
	}
	if len(t) < 3 || !frontendIdentifier(t[0]) || t[1] != "(" || t[len(t)-1] != ")" {
		return instruction, "expected a declared machine operation"
	}
	instruction.Operation = t[0]
	for at := 2; at < len(t)-1; {
		operand := TargetOperand{Kind: "value", Value: t[at]}
		if !frontendIdentifier(t[at]) {
			return instruction, "machine operands must be typed literals or preceding results"
		}
		at++
		if at < len(t)-1 && t[at] == "(" {
			operand.Kind = operand.Value
			at++
			start := at
			if at < len(t)-1 && t[at] == "-" {
				at++
			}
			if at+1 >= len(t) || t[at+1] != ")" {
				return instruction, "typed operand must contain one literal, not an expression"
			}
			operand.Value = strings.Join(t[start:at+1], "")
			at += 2
		}
		instruction.Operands = append(instruction.Operands, operand)
		if at < len(t)-1 {
			if t[at] != "," {
				return instruction, "expected machine operand comma"
			}
			at++
			if at == len(t)-1 {
				return instruction, "trailing machine operand comma"
			}
		}
	}
	return instruction, ""
}

func validateTargetBlock(v TargetVocabulary, block TargetBlock) ([]Statement, string) {
	if len(block.Instructions) == 0 || len(block.Instructions) > maxStatementsPerBody {
		return nil, "invalid machine instruction count"
	}
	locals := map[string]string{}
	lowerNames := map[string]string{}
	uses := map[string]bool{}
	definitions := map[string]bool{}
	references := map[string]Span{}
	steps := []Statement{}
	for index, instruction := range block.Instructions {
		fail := func(message string) ([]Statement, string) {
			return append(steps, Statement{Span: instruction.Span}), message
		}
		if !frontendIdentifier(instruction.Operation) || instruction.Operation == "let" {
			return fail("invalid machine operation name")
		}
		var op TargetOperation
		found := false
		for _, candidate := range v.Operations {
			if candidate.Name == instruction.Operation {
				op, found = candidate, true
				break
			}
		}
		if !found {
			return fail("unknown or unavailable machine operation " + instruction.Operation)
		}
		if len(instruction.Operands) != len(op.Parameters) {
			return fail("machine operation " + op.Name + " has the wrong operand count")
		}
		if (instruction.Result != "") != (op.Result != "") {
			return fail("machine operation " + op.Name + " has an incompatible result binding")
		}
		if instruction.Result != "" && (!frontendIdentifier(instruction.Result) || locals[instruction.Result] != "") {
			return fail("invalid or duplicate machine result " + instruction.Result)
		}
		tokens := []string{op.lowerName, "(", "out"}
		for i, operand := range instruction.Operands {
			kind := operand.Kind
			value := operand.Value
			if kind == "value" {
				uses[value] = true
				kind = locals[value]
				if kind == "" {
					return fail("unknown machine result " + value)
				}
				value = lowerNames[value]
			} else {
				canonical, ok := targetLiteral(v, operand)
				if !ok {
					return fail("invalid " + kind + " machine literal " + value)
				}
				value = canonical
			}
			if kind != op.Parameters[i].Kind {
				return fail("machine operand " + op.Parameters[i].Name + " requires " + op.Parameters[i].Kind + ", got " + kind)
			}
			limitOperand := operand
			limitOperand.Value = value
			for _, limit := range op.Immediates {
				if limit.Parameter == op.Parameters[i].Name && (operand.Kind == "value" || !targetImmediateFits(limitOperand, limit)) {
					return fail("machine immediate " + limit.Parameter + " is outside its declared range")
				}
			}
			if kind == "label" {
				if stringIndex(op.Effects.Defines, op.Parameters[i].Name) >= 0 {
					if definitions[operand.Value] {
						return fail("machine label is bound more than once: " + operand.Value)
					}
					definitions[operand.Value] = true
				}
				if stringIndex(op.Effects.References, op.Parameters[i].Name) >= 0 {
					references[operand.Value] = instruction.Span
				}
			}
			tokens = append(tokens, ",", value)
		}
		tokens = append(tokens, ")")
		if instruction.Result != "" {
			// Generated local names prevent collisions with registers, helpers,
			// keywords, and the private emitter, without reserving user names.
			name := "renvoMachineValue" + strconv.Itoa(index)
			locals[instruction.Result], lowerNames[instruction.Result] = op.Result, name
			tokens = append([]string{"let", name, "="}, tokens...)
		}
		steps = append(steps, Statement{Tokens: tokens, Span: instruction.Span})
	}
	for _, instruction := range block.Instructions { // deterministic diagnostics
		if instruction.Result != "" {
			if !uses[instruction.Result] {
				return append(steps, Statement{Span: instruction.Span}), "unused machine result " + instruction.Result
			}
			if span, referenced := references[instruction.Result]; referenced && !definitions[instruction.Result] {
				return append(steps, Statement{Span: span}), "unbound machine label " + instruction.Result
			}
		}
	}
	return steps, ""
}

func targetLiteral(v TargetVocabulary, operand TargetOperand) (string, bool) {
	if operand.Kind == "register" {
		return operand.Value, frontendIdentifier(operand.Value) && stringIndex(v.Registers, operand.Value) >= 0
	}
	if operand.Kind == "condition" {
		return operand.Value, frontendIdentifier(operand.Value) && stringIndex(v.Conditions, operand.Value) >= 0
	}
	if operand.Kind == "bool" {
		return operand.Value, operand.Value == "true" || operand.Value == "false"
	}
	// Label and address handles can only come from declared operations.
	if !frontendIntegerKind(operand.Kind) {
		return "", false
	}
	if operand.Kind == "uint64" || operand.Kind == "byte" {
		bits := 64
		if operand.Kind == "byte" {
			bits = 8
		}
		n, err := strconv.ParseUint(operand.Value, 0, bits)
		return strconv.FormatUint(n, 10), err == nil
	}
	bits := 64
	if operand.Kind == "int" {
		bits = 32
	} // evaluator's portable word
	n, err := strconv.ParseInt(operand.Value, 0, bits)
	return strconv.FormatInt(n, 10), err == nil
}

func targetImmediateFits(operand TargetOperand, limit TargetImmediate) bool {
	if limit.Bits < 1 || limit.Bits > 64 || (limit.Mode != "bits" && limit.Mode != "signed" && limit.Mode != "unsigned") {
		return false
	}
	negative := strings.HasPrefix(operand.Value, "-")
	if negative {
		n, err := strconv.ParseInt(operand.Value, 0, 64)
		if err != nil || limit.Mode == "unsigned" {
			return false
		}
		return limit.Bits == 64 || n >= -(int64(1)<<(limit.Bits-1))
	}
	n, err := strconv.ParseUint(operand.Value, 0, 64)
	if err != nil {
		return false
	}
	bits := limit.Bits
	if limit.Mode == "signed" {
		bits--
	}
	return bits == 64 || n < uint64(1)<<bits
}
