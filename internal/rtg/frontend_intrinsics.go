package rtg

import (
	"strconv"
	"strings"
)

// TargetIntrinsic consumes runtime values, not physical register selections.
// Placement and result capture belong to the selected compiler boundary.
// Effects use semantic parameter names; lowering names remain private.
type TargetIntrinsic struct {
	Name       string
	Parameters []TargetParameter
	Result     string
	Effects    TargetEffects
	Requires   []string
	operation  TargetOperation
}

func frontendIntrinsics(document Document, arch Declaration, v *TargetVocabulary) {
	for _, block := range arch.Statements {
		if statementBlockName(block) != "frontend_intrinsics" {
			continue
		}
		for _, child := range block.Children {
			intrinsic, message := decodeFrontendIntrinsic(child)
			if message == "" {
				message = validateFrontendOperation(document, arch, *v, intrinsic.operation)
			}
			if message == "" {
				for _, old := range v.Intrinsics {
					if old.Name == intrinsic.Name {
						message = "duplicate frontend intrinsic " + intrinsic.Name
					}
				}
			}
			if message != "" {
				v.Diagnostics = append(v.Diagnostics, statementDiagnostic(document, child, "RTG-INTRINSIC-001", message))
			} else {
				v.Intrinsics = append(v.Intrinsics, intrinsic)
			}
		}
	}
}

func decodeFrontendIntrinsic(s Statement) (TargetIntrinsic, string) {
	result := TargetIntrinsic{}
	tokens := append([]string(nil), s.Tokens...)
	if len(tokens) < 6 || tokens[len(tokens)-3] != "-" || tokens[len(tokens)-2] != ">" || tokens[len(tokens)-1] != "word" {
		return result, "intrinsic requires one native-word result; pointer intrinsics are not supported"
	}
	result.Result = tokens[len(tokens)-1]
	tokens = tokens[:len(tokens)-3]
	if len(tokens) < 3 || tokens[1] != "(" {
		return result, "invalid intrinsic signature"
	}
	// Validate semantic kinds before adapting to the private register encoder.
	for at := 2; at < len(tokens)-1; at += 4 {
		if at+2 >= len(tokens)-1 || tokens[at+1] != ":" || tokens[at+2] != "word" || tokens[at] == "result" {
			return result, "intrinsic parameters must be runtime words"
		}
		result.Parameters = append(result.Parameters, TargetParameter{Name: tokens[at], Kind: tokens[at+2]})
		tokens[at+2] = "register"
	}
	if len(result.Parameters) > 6 {
		return result, "intrinsic exceeds the managed input word limit"
	}
	prefix := []string{tokens[0], "(", "result", ":", "register"}
	if len(result.Parameters) != 0 {
		prefix = append(prefix, ",")
	}
	transformed := s
	transformed.Tokens = append(prefix, tokens[2:]...)
	op, message := decodeFrontendOperation(transformed)
	if message != "" {
		return result, message
	}
	if len(op.Parameters) != len(result.Parameters)+1 {
		return result, "invalid intrinsic parameters"
	}
	if op.Effects.Control != "none" || len(op.Effects.Defines) != 0 || len(op.Effects.References) != 0 || len(op.Immediates) != 0 {
		return result, "intrinsic must have a straight-line runtime-value lowering"
	}
	if len(op.Effects.Writes) != 1 || op.Effects.Writes[0] != "result" || stringIndex(op.Effects.Reads, "result") >= 0 || stringIndex(op.Effects.Clobbers, "result") >= 0 {
		return result, "intrinsic must initialize its result without reading it"
	}
	for _, p := range result.Parameters {
		if stringIndex(op.Effects.Reads, p.Name) < 0 || stringIndex(op.Effects.Clobbers, p.Name) >= 0 {
			return result, "intrinsic inputs must be immutable reads"
		}
	}
	for _, name := range op.Effects.Reads {
		if frontendParameter(op, name) < 0 {
			return result, "intrinsic cannot depend on an ambient physical register or resource"
		}
	}
	result.Name, result.Effects, result.Requires, result.operation = op.Name, op.Effects, op.Requires, op
	return result, ""
}

func selectFrontendIntrinsics(v *TargetVocabulary) {
	selected := []TargetIntrinsic{}
	for _, intrinsic := range v.Intrinsics {
		allowed := v.Managed.Ok && len(intrinsic.Parameters) <= len(v.Managed.Arguments)
		for _, required := range intrinsic.Requires {
			if stringIndex(v.Target.Capabilities, required) < 0 {
				allowed = false
			}
		}
		for _, clobber := range intrinsic.Effects.Clobbers {
			// A runtime-value intrinsic cannot silently destroy a physical input.
			if stringIndex(v.Managed.Resources, clobber) < 0 {
				allowed = false
			}
		}
		if allowed {
			selected = append(selected, intrinsic)
		}
	}
	v.Intrinsics = selected
}

// IntrinsicBlock declares a compiler-managed runtime intrinsic implementation
// for a bodyless function. Function arguments are evaluated once by the normal
// expression compiler; direct calls inline the selected encoder body.
type IntrinsicBlock struct{ Name, Intrinsic string }

func EncodeIntrinsicAssembly(v TargetVocabulary, blocks []IntrinsicBlock, filename string) ([]byte, []Diagnostic) {
	fail := func(message string) ([]byte, []Diagnostic) {
		return nil, []Diagnostic{{Filename: filename, Code: "RTG-INTRINSIC-002", Message: message}}
	}
	if !v.Ok || !v.Managed.Ok || len(blocks) == 0 {
		return fail("selected backend has no runtime intrinsic boundary")
	}
	var out strings.Builder
	out.WriteString("rtgasm 3\nassembly {\n")
	names := []string{}
	for _, block := range blocks {
		if !frontendIdentifier(block.Name) || stringIndex(names, block.Name) >= 0 {
			return fail("invalid or duplicate intrinsic function name")
		}
		names = append(names, block.Name)
		intrinsic, found := targetIntrinsic(v, block.Intrinsic)
		if !found {
			return fail("unknown or unavailable intrinsic " + block.Intrinsic)
		}
		out.WriteString(block.Name + "(out:emitter) {\ninputs = ")
		// The encoder API contains no source expressions or register choices.
		out.WriteString(strconv.Itoa(len(intrinsic.Parameters)) + "\nintrinsic = " + intrinsic.Name + "\nyield()\n}\n")
		if out.Len() > maxDefinitionBytes {
			return fail("intrinsic source exceeds its limit")
		}
	}
	out.WriteString("}\n")
	return []byte(out.String()), nil
}
func targetIntrinsic(v TargetVocabulary, name string) (TargetIntrinsic, bool) {
	for _, intrinsic := range v.Intrinsics {
		if intrinsic.Name == name {
			return intrinsic, true
		}
	}
	return TargetIntrinsic{}, false
}
