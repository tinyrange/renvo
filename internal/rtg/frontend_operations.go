package rtg

// TargetOperation is the backend-owned vocabulary for explicit physical machine
// blocks. It does not name compiler helpers or prescribe assembler spelling.
// Every operation is ordered; these effects are a contract, not permission to
// reorder a block or to replace it with a runtime helper call.
type TargetOperation struct {
	Name       string
	Parameters []TargetParameter
	Result     string
	Effects    TargetEffects
	Requires   []string
	Immediates []TargetImmediate
	lowerKind  string
	lowerName  string
	statement  Statement
}

type TargetParameter struct{ Name, Kind string }
type TargetEffects struct {
	Reads, Writes, Clobbers   []string
	Memory, Control, Ordering string
	Defines, References       []string
}

// TargetImmediate limits an operand beyond its transport type. bits permits
// signed or unsigned bit patterns; signed and unsigned have their usual ranges.
type TargetImmediate struct {
	Parameter, Mode string
	Bits            int
}

type TargetVocabulary struct {
	Target                           TargetDescriptor
	Operations                       []TargetOperation
	Syntaxes                         []TargetSyntax
	Widths                           []TargetWidth
	Constraints                      []TargetConstraint
	Managed                          TargetManaged
	Intrinsics                       []TargetIntrinsic
	Registers, Conditions, Resources []string
	Diagnostics                      []Diagnostic
	Ok                               bool
}

// FrontendOperations advertises the selected backend's public machine-block
// contract. Private lowering names never become callable frontend operations.
func FrontendOperations(resolved ResolveResult, targetName string) TargetVocabulary {
	if !resolved.Ok {
		return TargetVocabulary{Diagnostics: resolved.Diagnostics}
	}
	target, ok := lookupResolvedTarget(resolved, targetName)
	if !ok {
		return TargetVocabulary{Diagnostics: []Diagnostic{{Code: "RTG-FRONTEND-001", Message: "frontend operation target is unavailable"}}}
	}
	v := frontendVocabulary(resolved.Document, target.Arch)
	if len(v.Diagnostics) != 0 {
		return v
	}
	available := []TargetOperation{}
	for i := 0; i < len(v.Operations); i++ {
		op := v.Operations[i]
		allowed := true
		for j := 0; j < len(op.Requires); j++ {
			if stringIndex(target.Descriptor.Capabilities, op.Requires[j]) < 0 {
				allowed = false
			}
		}
		if allowed {
			available = append(available, op)
		}
	}
	v.Operations = available
	v.Target = target.Descriptor
	selectFrontendManaged(resolved, target, &v)
	selectFrontendIntrinsics(&v)
	v.Ok = true
	return v
}

func frontendVocabulary(document Document, arch Declaration) TargetVocabulary {
	v := TargetVocabulary{}
	for i := 0; i < len(arch.Statements); i++ {
		s := arch.Statements[i]
		if statementHead(s, "registers") {
			v.Registers = append(v.Registers, statementListValues(s)...)
		}
		if statementBlockName(s) == "conditions" {
			for j := 0; j < len(s.Children); j++ {
				left, _, ok := statementAssignment(s.Children[j])
				if !ok && len(s.Children[j].Tokens) >= 2 {
					left = s.Children[j].Tokens[:1]
					ok = true
				}
				if ok && len(left) == 1 {
					v.Conditions = append(v.Conditions, left[0])
				}
			}
		}
	}
	// Extensions may contribute more than one block; do not silently ignore a
	// second one or allow it to override a public operation.
	for i := 0; i < len(arch.Statements); i++ {
		block := arch.Statements[i]
		if statementBlockName(block) != "frontend_operations" {
			continue
		}
		for j := 0; j < len(block.Children); j++ {
			child := block.Children[j]
			left, _, assignment := statementAssignment(child)
			if assignment && len(left) == 1 && left[0] == "resources" {
				values, ok := frontendList(child)
				if !ok {
					v.Diagnostics = append(v.Diagnostics, statementDiagnostic(document, child, "RTG-FRONTEND-002", "resources must be an identifier list"))
				}
				for k := 0; k < len(values); k++ {
					if stringIndex(v.Resources, values[k]) >= 0 || stringIndex(v.Registers, values[k]) >= 0 {
						v.Diagnostics = append(v.Diagnostics, statementDiagnostic(document, child, "RTG-FRONTEND-002", "duplicate frontend resource "+values[k]))
					}
					v.Resources = append(v.Resources, values[k])
				}
				continue
			}
			op, message := decodeFrontendOperation(child)
			if message != "" {
				v.Diagnostics = append(v.Diagnostics, statementDiagnostic(document, child, "RTG-FRONTEND-002", message))
				continue
			}
			for k := 0; k < len(v.Operations); k++ {
				if v.Operations[k].Name == op.Name {
					v.Diagnostics = append(v.Diagnostics, statementDiagnostic(document, child, "RTG-FRONTEND-003", "duplicate frontend operation "+op.Name))
				}
			}
			v.Operations = append(v.Operations, op)
		}
	}
	for i := 0; i < len(v.Operations); i++ {
		op := v.Operations[i]
		message := validateFrontendOperation(document, arch, v, op)
		if message != "" {
			v.Diagnostics = append(v.Diagnostics, statementDiagnostic(document, op.statement, "RTG-FRONTEND-004", message))
		}
	}
	frontendWidths(document, arch, &v)
	frontendSyntaxes(document, arch, &v)
	frontendConstraints(document, arch, &v)
	frontendManaged(document, arch, &v)
	frontendIntrinsics(document, arch, &v)
	v.Ok = len(v.Diagnostics) == 0
	return v
}

func decodeFrontendOperation(s Statement) (TargetOperation, string) {
	op := TargetOperation{statement: s}
	t := s.Tokens
	if len(t) < 3 || !frontendIdentifier(t[0]) || t[1] != "(" {
		return op, "invalid frontend operation signature"
	}
	op.Name = t[0]
	if op.Name == "let" {
		return op, "let is reserved for machine result bindings"
	}
	at := 2
	for at < len(t) && t[at] != ")" {
		if at+2 >= len(t) || !frontendIdentifier(t[at]) || t[at+1] != ":" || !frontendKind(t[at+2]) {
			return op, "invalid frontend operand declaration"
		}
		if t[at] == "out" {
			return op, "frontend operands cannot name the private emitter"
		}
		for i := 0; i < len(op.Parameters); i++ {
			if op.Parameters[i].Name == t[at] {
				return op, "duplicate frontend operand " + t[at]
			}
		}
		op.Parameters = append(op.Parameters, TargetParameter{Name: t[at], Kind: t[at+2]})
		at += 3
		if at < len(t) && t[at] == "," {
			at++
			if at >= len(t) || t[at] == ")" {
				return op, "trailing operand comma"
			}
		} else if at < len(t) && t[at] != ")" {
			return op, "expected operand comma"
		}
	}
	if at >= len(t) || t[at] != ")" {
		return op, "unterminated frontend signature"
	}
	at++
	if at < len(t) {
		if at+3 != len(t) || t[at] != "-" || t[at+1] != ">" || !frontendKind(t[at+2]) {
			return op, "invalid frontend result declaration"
		}
		op.Result = t[at+2]
	}
	seen := []string{}
	for i := 0; i < len(s.Children); i++ {
		child := s.Children[i]
		if statementBlockName(child) == "immediates" {
			if stringIndex(seen, "immediates") >= 0 {
				return op, "duplicate immediates contract"
			}
			seen = append(seen, "immediates")
			for j := 0; j < len(child.Children); j++ {
				left, right, ok := statementAssignment(child.Children[j])
				if !ok || len(left) != 1 || len(right) != 2 || (right[0] != "bits" && right[0] != "signed" && right[0] != "unsigned") {
					return op, "invalid immediate range contract"
				}
				bits, ok := parseInteger(right[1])
				if !ok || bits < 1 || bits > 64 {
					return op, "immediate width must be between 1 and 64"
				}
				for k := 0; k < len(op.Immediates); k++ {
					if op.Immediates[k].Parameter == left[0] {
						return op, "duplicate immediate range"
					}
				}
				op.Immediates = append(op.Immediates, TargetImmediate{Parameter: left[0], Mode: right[0], Bits: bits})
			}
			continue
		}
		left, right, ok := statementAssignment(child)
		if !ok || len(left) != 1 || len(child.Children) != 0 {
			return op, "invalid frontend operation property"
		}
		name := left[0]
		if stringIndex(seen, name) >= 0 {
			return op, "duplicate frontend property " + name
		}
		seen = append(seen, name)
		if name == "lower" {
			if len(right) != 2 || (right[0] != "go" && right[0] != "sequence") || !frontendIdentifier(right[1]) {
				return op, "lower must bind a backend function or sequence"
			}
			op.lowerKind, op.lowerName = right[0], right[1]
		} else if name == "memory" || name == "control" || name == "ordering" {
			if len(right) != 1 {
				return op, "invalid frontend effect " + name
			}
			if name == "memory" {
				op.Effects.Memory = right[0]
			}
			if name == "control" {
				op.Effects.Control = right[0]
			}
			if name == "ordering" {
				op.Effects.Ordering = right[0]
			}
		} else {
			values, ok := frontendList(child)
			if !ok {
				return op, "frontend property must be an identifier list: " + name
			}
			if name == "reads" {
				op.Effects.Reads = values
			} else if name == "writes" {
				op.Effects.Writes = values
			} else if name == "clobbers" {
				op.Effects.Clobbers = values
			} else if name == "requires" {
				op.Requires = values
			} else if name == "defines" {
				op.Effects.Defines = values
			} else if name == "references" {
				op.Effects.References = values
			} else {
				return op, "unknown frontend property " + name
			}
		}
	}
	required := []string{"lower", "reads", "writes", "clobbers", "memory", "control", "ordering", "requires"}
	for i := 0; i < len(required); i++ {
		if stringIndex(seen, required[i]) < 0 {
			return op, "missing frontend property " + required[i]
		}
	}
	if stringIndex([]string{"none", "read", "write", "read_write", "unknown"}, op.Effects.Memory) < 0 {
		return op, "unknown memory effect"
	}
	if stringIndex([]string{"none", "label", "jump", "branch", "call", "return"}, op.Effects.Control) < 0 {
		return op, "unknown control effect"
	}
	if op.Effects.Ordering != "ordered" {
		return op, "physical operations must be ordered"
	}
	return op, ""
}

func validateFrontendOperation(document Document, arch Declaration, v TargetVocabulary, op TargetOperation) string {
	parameters := []string{"*RTGEmitter"}
	for i := 0; i < len(op.Parameters); i++ {
		goType, _ := sequenceGoType(op.Parameters[i].Kind)
		parameters = append(parameters, goType)
	}
	result := ""
	if op.Result != "" {
		result, _ = sequenceGoType(op.Result)
	}
	var function embeddedFunction
	found := false
	if op.lowerKind == "sequence" {
		sequence, ok := findArchitectureSequence(arch, op.lowerName)
		found = ok
		function = architectureSequenceFunction(sequence)
	} else {
		function, found = findEmbeddedFunction(document, op.lowerName)
	}
	if !found || !directEmitterSignatureMatches(function, directEmitterOperation{Parameters: parameters, Result: result}) {
		return "frontend operation " + op.Name + " has an incompatible lowering signature"
	}
	lists := [][]string{op.Effects.Reads, op.Effects.Writes, op.Effects.Clobbers}
	for i := 0; i < len(lists); i++ {
		for j := 0; j < len(lists[i]); j++ {
			name := lists[i][j]
			parameter := frontendParameter(op, name)
			if parameter >= 0 {
				if op.Parameters[parameter].Kind != "register" && op.Parameters[parameter].Kind != "address" {
					return "register effect references a non-location operand " + name
				}
			} else if stringIndex(v.Registers, name) < 0 && stringIndex(v.Resources, name) < 0 {
				return "unknown frontend effect resource " + name
			}
		}
	}
	labels := [][]string{op.Effects.Defines, op.Effects.References}
	for i := 0; i < len(labels); i++ {
		for j := 0; j < len(labels[i]); j++ {
			p := frontendParameter(op, labels[i][j])
			if p < 0 || op.Parameters[p].Kind != "label" {
				return "label effect must reference a label operand"
			}
		}
	}
	if (op.Effects.Control == "jump" || op.Effects.Control == "branch") && len(op.Effects.References) == 0 {
		return "branch operation requires a label reference"
	}
	if (len(op.Effects.Defines) != 0) != (op.Effects.Control == "label") {
		return "label definitions require label control effects"
	}
	for i := 0; i < len(op.Parameters); i++ {
		p := op.Parameters[i]
		if op.Result == "label" && p.Kind == "label" {
			return "label results must be fresh, not aliases of label operands"
		}
		if stringIndex(op.Effects.Defines, p.Name) >= 0 && stringIndex(op.Effects.References, p.Name) >= 0 {
			return "label operand cannot be defined and referenced by one operation"
		}
		if p.Kind == "register" && stringIndex(op.Effects.Reads, p.Name) < 0 && stringIndex(op.Effects.Writes, p.Name) < 0 && stringIndex(op.Effects.Clobbers, p.Name) < 0 {
			return "register operand requires an explicit effect: " + p.Name
		}
		if op.Parameters[i].Kind == "label" && stringIndex(op.Effects.Defines, op.Parameters[i].Name) < 0 && stringIndex(op.Effects.References, op.Parameters[i].Name) < 0 {
			return "label operand requires a defines or references effect"
		}
	}
	for i := 0; i < len(op.Immediates); i++ {
		p := frontendParameter(op, op.Immediates[i].Parameter)
		if p < 0 || !frontendIntegerKind(op.Parameters[p].Kind) {
			return "immediate range must reference an integer operand"
		}
	}
	return ""
}

func frontendParameter(op TargetOperation, name string) int {
	for i := 0; i < len(op.Parameters); i++ {
		if op.Parameters[i].Name == name {
			return i
		}
	}
	return -1
}
func frontendKind(kind string) bool {
	return frontendIntegerKind(kind) || kind == "register" || kind == "condition" || kind == "address" || kind == "label" || kind == "bool"
}
func frontendIntegerKind(kind string) bool {
	return kind == "int" || kind == "int64" || kind == "uint64" || kind == "byte"
}
func frontendIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_' || i > 0 && c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}
func frontendList(s Statement) ([]string, bool) {
	_, right, ok := statementAssignment(s)
	if !ok || len(right) < 2 || right[0] != "[" || right[len(right)-1] != "]" {
		return nil, false
	}
	values := []string{}
	for at := 1; at < len(right)-1; {
		if !frontendIdentifier(right[at]) || stringIndex(values, right[at]) >= 0 {
			return nil, false
		}
		values = append(values, right[at])
		at++
		if at < len(right)-1 {
			if right[at] != "," {
				return nil, false
			}
			at++
			if at == len(right)-1 {
				return nil, false
			}
		}
	}
	return values, true
}
