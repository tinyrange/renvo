package rtg

// directEmitterOperation is one stable compiler-facing operation in the
// direct_emitter_v1 contract. Architecture definitions bind these semantic
// names to an instruction row or to an embedded Go algorithm. The operation
// table is deliberately data, not executable dispatch: validation and source
// generation consume it, and no copy survives in a fixed generated backend.
type directEmitterOperation struct {
	Name         string
	Parameters   []string
	Result       string
	Reads        []string
	Writes       []string
	Clobbers     []string
	Preserves    []string
	Memory       string
	Control      string
	Relocation   string
	Capabilities []string
	Rejectable   bool
}

const directEmitterV1Schema = `move|*RTGEmitter,RTGRegister,RTGRegister
address|*RTGEmitter,RTGRegister,RTGAddress
load.native|*RTGEmitter,RTGRegister,RTGAddress
load.i32|*RTGEmitter,RTGRegister,RTGAddress
load.u32|*RTGEmitter,RTGRegister,RTGAddress
load.i16|*RTGEmitter,RTGRegister,RTGAddress
load.u16|*RTGEmitter,RTGRegister,RTGAddress
load.i8|*RTGEmitter,RTGRegister,RTGAddress
load.u8|*RTGEmitter,RTGRegister,RTGAddress
store.native|*RTGEmitter,RTGAddress,RTGRegister
store.u32|*RTGEmitter,RTGAddress,RTGRegister
store.u16|*RTGEmitter,RTGAddress,RTGRegister
store.u8|*RTGEmitter,RTGAddress,RTGRegister
add|*RTGEmitter,RTGRegister,RTGRegister
subtract|*RTGEmitter,RTGRegister,RTGRegister
multiply|*RTGEmitter,RTGRegister,RTGRegister
bit_and|*RTGEmitter,RTGRegister,RTGRegister
bit_or|*RTGEmitter,RTGRegister,RTGRegister
bit_xor|*RTGEmitter,RTGRegister,RTGRegister
compare|*RTGEmitter,RTGRegister,RTGRegister
test|*RTGEmitter,RTGRegister,RTGRegister
increment|*RTGEmitter,RTGRegister
decrement|*RTGEmitter,RTGRegister
shift_left_immediate|*RTGEmitter,RTGRegister,byte
shift_right_unsigned_immediate|*RTGEmitter,RTGRegister,byte
shift_right_signed_immediate|*RTGEmitter,RTGRegister,byte
call|*RTGEmitter,RTGLabel
call_indirect|*RTGEmitter,RTGRegister
jump|*RTGEmitter,RTGLabel
jump_condition|*RTGEmitter,RTGCondition,RTGLabel
set_condition|*RTGEmitter,RTGCondition,RTGRegister
return|*RTGEmitter
leave|*RTGEmitter
host_syscall|*RTGEmitter
move_immediate|*RTGEmitter,RTGRegister,int64
variable_shift|*RTGEmitter,RTGShiftDirection,bool
signed_divide|*RTGEmitter,bool
copy_bytes|*RTGEmitter`

// Each effect row is:
//
//	name|reads|writes|clobbers|preserves|memory|control|relocation|capabilities|rejectable
//
// The compact schema is generator input only. Generated backends retain typed
// direct calls and reviewable effect comments, never this table.
const directEmitterV1Effects = `move|source|destination|-|all_except_writes|-|fallthrough|-|-|false
address|address|destination|-|all_except_writes|-|fallthrough|address|-|false
load.native|address|destination|-|all_except_writes|read:native:target|fallthrough|address|-|false
load.i32|address|destination|-|all_except_writes|read:32:signed:target|fallthrough|address|-|false
load.u32|address|destination|-|all_except_writes|read:32:unsigned:target|fallthrough|address|-|false
load.i16|address|destination|-|all_except_writes|read:16:signed:target|fallthrough|address|-|false
load.u16|address|destination|-|all_except_writes|read:16:unsigned:target|fallthrough|address|-|false
load.i8|address|destination|-|all_except_writes|read:8:signed:1|fallthrough|address|-|false
load.u8|address|destination|-|all_except_writes|read:8:unsigned:1|fallthrough|address|-|false
store.native|address,source|memory|-|all_locations|write:native:target|fallthrough|address|-|false
store.u32|address,source|memory|-|all_locations|write:32:target|fallthrough|address|-|false
store.u16|address,source|memory|-|all_locations|write:16:target|fallthrough|address|-|false
store.u8|address,source|memory|-|all_locations|write:8:1|fallthrough|address|-|false
add|destination,source|destination,flags|flags|all_except_writes|-|fallthrough|-|-|false
subtract|destination,source|destination,flags|flags|all_except_writes|-|fallthrough|-|-|false
multiply|destination,source|destination,flags|flags|all_except_writes|-|fallthrough|-|-|false
bit_and|destination,source|destination,flags|flags|all_except_writes|-|fallthrough|-|-|false
bit_or|destination,source|destination,flags|flags|all_except_writes|-|fallthrough|-|-|false
bit_xor|destination,source|destination,flags|flags|all_except_writes|-|fallthrough|-|-|false
compare|left,right|flags|flags|all_locations|-|fallthrough|-|-|false
test|left,right|flags|flags|all_locations|-|fallthrough|-|-|false
increment|operand|operand,flags|flags|all_except_writes|-|fallthrough|-|-|false
decrement|operand|operand,flags|flags|all_except_writes|-|fallthrough|-|-|false
shift_left_immediate|operand,value|operand,flags|flags|all_except_writes|-|fallthrough|-|-|false
shift_right_unsigned_immediate|operand,value|operand,flags|flags|all_except_writes|-|fallthrough|-|-|false
shift_right_signed_immediate|operand,value|operand,flags|flags|all_except_writes|-|fallthrough|-|-|false
call|label,call_words,stack|primary,secondary,tertiary,stack|call_clobbered,flags|callee_saved|-|call|label|-|false
call_indirect|operand,call_words,stack|primary,secondary,tertiary,stack|call_clobbered,flags|callee_saved|-|call_indirect|-|indirect_call|true
jump|label|-|-|all_locations|-|branch|label|-|false
jump_condition|condition,label|-|-|all_locations|-|conditional_branch|label|-|false
set_condition|condition,flags|destination|-|all_except_writes|-|fallthrough|-|-|false
return|stack|-|-|all_locations|-|return|-|-|false
leave|frame,stack|frame,stack|-|all_except_writes|-|fallthrough|-|-|false
host_syscall|syscall_number,syscall_words|syscall_result|syscall_clobbered,flags|callee_saved|host|call|-|host_syscall|false
move_immediate|value|destination|-|all_except_writes|-|fallthrough|-|-|false
variable_shift|primary,tertiary|primary|tertiary,flags|all_except_writes|-|fallthrough|-|-|false
signed_divide|primary,tertiary|primary|secondary,flags|all_except_writes|-|fallthrough|-|-|false
copy_bytes|copy_destination,copy_source,copy_count,memory|memory|copy_destination,copy_source,copy_count,flags|other_locations|read_write:8:1|fallthrough|-|-|false`

var directEmitterV1 []directEmitterOperation

func ensureDirectEmitterV1() {
	if len(directEmitterV1) != 0 {
		return
	}
	directEmitterV1 = decodeDirectEmitterV1()
}

func decodeDirectEmitterV1() []directEmitterOperation {
	var operations []directEmitterOperation
	lineStart := 0
	for lineStart < len(directEmitterV1Schema) {
		lineEnd := lineStart
		for lineEnd < len(directEmitterV1Schema) && directEmitterV1Schema[lineEnd] != '\n' {
			lineEnd++
		}
		nameEnd := lineStart
		for nameEnd < lineEnd && directEmitterV1Schema[nameEnd] != '|' {
			nameEnd++
		}
		operation := directEmitterOperation{Name: directEmitterV1Schema[lineStart:nameEnd]}
		parameterStart := nameEnd + 1
		for parameterStart < lineEnd {
			parameterEnd := parameterStart
			for parameterEnd < lineEnd && directEmitterV1Schema[parameterEnd] != ',' {
				parameterEnd++
			}
			operation.Parameters = append(operation.Parameters,
				directEmitterV1Schema[parameterStart:parameterEnd])
			parameterStart = parameterEnd + 1
		}
		operations = append(operations, operation)
		lineStart = lineEnd + 1
	}
	applyDirectEmitterEffects(operations)
	return operations
}

func applyDirectEmitterEffects(operations []directEmitterOperation) {
	lineStart := 0
	for lineStart < len(directEmitterV1Effects) {
		lineEnd := lineStart
		for lineEnd < len(directEmitterV1Effects) && directEmitterV1Effects[lineEnd] != '\n' {
			lineEnd++
		}
		fields := splitContractFields(directEmitterV1Effects[lineStart:lineEnd])
		if len(fields) == 10 {
			index := directEmitterOperationIndexIn(operations, fields[0])
			if index >= 0 {
				operations[index].Reads = splitContractList(fields[1])
				operations[index].Writes = splitContractList(fields[2])
				operations[index].Clobbers = splitContractList(fields[3])
				operations[index].Preserves = splitContractList(fields[4])
				operations[index].Memory = emptyContractField(fields[5])
				operations[index].Control = emptyContractField(fields[6])
				operations[index].Relocation = emptyContractField(fields[7])
				operations[index].Capabilities = splitContractList(fields[8])
				operations[index].Rejectable = fields[9] == "true"
			}
		}
		lineStart = lineEnd + 1
	}
}

func splitContractFields(line string) []string {
	var fields []string
	start := 0
	for i := 0; i <= len(line); i++ {
		if i == len(line) || line[i] == '|' {
			fields = append(fields, line[start:i])
			start = i + 1
		}
	}
	return fields
}

func splitContractList(value string) []string {
	if value == "" || value == "-" {
		return nil
	}
	var values []string
	start := 0
	for i := 0; i <= len(value); i++ {
		if i == len(value) || value[i] == ',' {
			values = append(values, value[start:i])
			start = i + 1
		}
	}
	return values
}

func emptyContractField(value string) string {
	if value == "-" {
		return ""
	}
	return value
}

func directEmitterOperationIndexIn(operations []directEmitterOperation, name string) int {
	for i := 0; i < len(operations); i++ {
		if operations[i].Name == name {
			return i
		}
	}
	return -1
}

type architectureBinding struct {
	Name        string
	Instruction string
	Algorithm   string
	Statement   Statement
}

func architectureGoHook(arch Declaration, name string) (string, bool) {
	for i := 0; i < len(arch.Statements); i++ {
		left, right, assignment := statementAssignment(arch.Statements[i])
		if assignment && len(left) == 1 && left[0] == name &&
			len(right) == 2 && (right[0] == "go" || right[0] == "sequence") {
			return right[1], true
		}
	}
	return "", false
}

func directEmitterOperationIndex(name string) int {
	ensureDirectEmitterV1()
	return directEmitterOperationIndexIn(directEmitterV1, name)
}

func architectureBindings(arch Declaration) []architectureBinding {
	block, ok := declarationBlock(arch, "bind")
	if !ok {
		return nil
	}
	var bindings []architectureBinding
	for i := 0; i < len(block.Children); i++ {
		left, right, assignment := statementAssignment(block.Children[i])
		if !assignment {
			continue
		}
		name := joinStatementTokens(left)
		binding := architectureBinding{Name: name, Statement: block.Children[i]}
		if len(right) == 2 && (right[0] == "go" || right[0] == "sequence") {
			binding.Algorithm = right[1]
		} else if len(right) == 1 {
			binding.Instruction = right[0]
		}
		bindings = append(bindings, binding)
	}
	return bindings
}

func architectureRejectedOperations(arch Declaration) []string {
	for i := 0; i < len(arch.Statements); i++ {
		left, right, assignment := statementAssignment(arch.Statements[i])
		if assignment && len(left) == 1 && left[0] == "reject" {
			var operations []string
			var operation []string
			for j := 0; j <= len(right); j++ {
				token := ","
				if j < len(right) {
					token = right[j]
				}
				if token == "[" || token == "]" {
					continue
				}
				if token == "," {
					if len(operation) != 0 {
						operations = append(operations, valueName(joinStatementTokens(operation)))
						operation = nil
					}
					continue
				}
				operation = append(operation, token)
			}
			return operations
		}
	}
	return nil
}

func joinStatementTokens(tokens []string) string {
	var out string
	for i := 0; i < len(tokens); i++ {
		out += tokens[i]
	}
	return out
}

func architectureInstructionNames(arch Declaration) []string {
	block, ok := declarationBlock(arch, "instructions")
	if !ok {
		return nil
	}
	var names []string
	for i := 0; i < len(block.Children); i++ {
		left, _, assignment := statementAssignment(block.Children[i])
		if assignment && len(left) == 1 {
			names = append(names, left[0])
		} else if len(block.Children[i].Tokens) > 0 {
			names = append(names, block.Children[i].Tokens[0])
		}
	}
	return names
}

func validateImplementsContract(document Document) []Diagnostic {
	ensureDirectEmitterV1()
	var diagnostics []Diagnostic
	for i := 0; i < len(document.Implements); i++ {
		if document.Implements[i] != "direct_emitter_v1" {
			diagnostics = append(diagnostics, Diagnostic{
				Filename: document.Filename,
				Span:     sourceSpan(document.Source, 0, 0),
				Code:     "RTG-VALIDATE-070",
				Message:  "unknown backend contract " + document.Implements[i],
			})
		}
	}
	for i := 0; i < len(directEmitterV1); i++ {
		operation := directEmitterV1[i]
		if operation.Control == "" || len(operation.Preserves) == 0 {
			diagnostics = append(diagnostics, Diagnostic{
				Filename: document.Filename,
				Span:     sourceSpan(document.Source, 0, 0),
				Code:     "RTG-VALIDATE-080",
				Message:  "incomplete direct emitter effect contract for " + operation.Name,
			})
		}
	}
	return diagnostics
}

// validateDirectEmitterBindings validates every bind block as a complete
// projection of direct_emitter_v1, except for operations the architecture
// explicitly rejects. Rejection is deliberately a closed list: it is visible
// in the definition and cannot silently become a missing implementation.
func validateDirectEmitterBindings(document Document, arch Declaration, goNames []string) []Diagnostic {
	bindings := architectureBindings(arch)
	instructions := architectureInstructionNames(arch)
	seen := make([]string, 0, len(bindings))
	var diagnostics []Diagnostic
	rejected := architectureRejectedOperations(arch)
	for i := 0; i < len(rejected); i++ {
		if directEmitterOperationIndex(rejected[i]) < 0 {
			diagnostics = append(diagnostics, resolveDiagnostic(document, arch,
				"RTG-VALIDATE-077", "architecture "+arch.Name+
					" rejects unknown direct emitter operation "+rejected[i]))
		}
		for j := 0; j < i; j++ {
			if rejected[j] == rejected[i] {
				diagnostics = append(diagnostics, resolveDiagnostic(document, arch,
					"RTG-VALIDATE-078", "architecture "+arch.Name+
						" rejects direct emitter operation more than once "+rejected[i]))
			}
		}
	}
	for i := 0; i < len(bindings); i++ {
		binding := bindings[i]
		if binding.Name == "" || binding.Algorithm == "" && binding.Instruction == "" {
			diagnostics = append(diagnostics, statementDiagnostic(document, binding.Statement,
				"RTG-VALIDATE-071", "invalid direct emitter binding"))
			continue
		}
		if directEmitterOperationIndex(binding.Name) < 0 {
			diagnostics = append(diagnostics, statementDiagnostic(document, binding.Statement,
				"RTG-VALIDATE-072", "unknown direct emitter operation "+binding.Name))
		}
		if stringIndex(seen, binding.Name) >= 0 {
			diagnostics = append(diagnostics, statementDiagnostic(document, binding.Statement,
				"RTG-VALIDATE-073", "duplicate direct emitter binding "+binding.Name))
		}
		if stringIndex(rejected, binding.Name) >= 0 {
			diagnostics = append(diagnostics, statementDiagnostic(document, binding.Statement,
				"RTG-VALIDATE-079", "direct emitter operation "+binding.Name+
					" is both bound and rejected"))
		}
		seen = append(seen, binding.Name)
		if binding.Algorithm != "" {
			_, found := findBackendFunction(document, binding.Algorithm)
			if !found {
				diagnostics = append(diagnostics, statementDiagnostic(document, binding.Statement,
					"RTG-VALIDATE-011", "unknown backend algorithm "+binding.Algorithm))
			}
		}
		if binding.Instruction != "" && stringIndex(instructions, binding.Instruction) < 0 {
			diagnostics = append(diagnostics, statementDiagnostic(document, binding.Statement,
				"RTG-VALIDATE-074", "unknown instruction "+binding.Instruction+
					" bound to "+binding.Name))
		}
		if operation := directEmitterOperationIndex(binding.Name); operation >= 0 {
			function, found := directEmitterBindingFunction(document, arch, binding)
			if found && !directEmitterSignatureMatches(function, directEmitterV1[operation]) {
				diagnostics = append(diagnostics, statementDiagnostic(document, binding.Statement,
					"RTG-VALIDATE-076", "direct emitter binding "+binding.Name+
						" has an incompatible Go signature"))
			}
		}
	}
	for i := 0; i < len(directEmitterV1); i++ {
		if stringIndex(seen, directEmitterV1[i].Name) < 0 &&
			stringIndex(rejected, directEmitterV1[i].Name) < 0 {
			diagnostics = append(diagnostics, resolveDiagnostic(document, arch,
				"RTG-VALIDATE-075", "architecture "+arch.Name+
					" does not bind direct emitter operation "+directEmitterV1[i].Name))
		}
	}
	return diagnostics
}

func directEmitterBindingFunction(document Document, arch Declaration, binding architectureBinding) (embeddedFunction, bool) {
	if binding.Algorithm != "" {
		return findBackendFunction(document, binding.Algorithm)
	}
	instructions, ok := declarationBlock(arch, "instructions")
	if !ok {
		return embeddedFunction{}, false
	}
	forms := architectureForms(arch)
	for i := 0; i < len(instructions.Children); i++ {
		child := instructions.Children[i]
		left, right, assignment := statementAssignment(child)
		name := ""
		if assignment && len(left) == 1 {
			name = left[0]
			if name == binding.Instruction && len(right) == 2 &&
				(right[0] == "go" || right[0] == "sequence") {
				return findBackendFunction(document, right[1])
			}
		} else if len(child.Tokens) >= 2 {
			name = child.Tokens[0]
		}
		if name != binding.Instruction || len(child.Tokens) < 2 {
			continue
		}
		form, found := lookupArchitectureForm(forms, child.Tokens[1])
		if !found {
			return embeddedFunction{}, false
		}
		if form.Kind == "bytes" {
			return embeddedFunction{
				Name: binding.Instruction,
				Parameters: []embeddedParameter{
					{Name: "out", Source: []byte("out *RTGEmitter")},
				},
			}, true
		}
		if form.Kind == "word32" {
			return architectureFormFunction(form, binding.Instruction)
		}
		function, found := findBackendFunction(document, form.Algorithm)
		if !found {
			return embeddedFunction{}, false
		}
		constants := instructionConstants(child.Tokens)
		filtered := function
		filtered.Parameters = nil
		for parameter := 0; parameter < len(function.Parameters); parameter++ {
			if instructionConstant(constants, function.Parameters[parameter].Name) == "" {
				filtered.Parameters = append(filtered.Parameters, function.Parameters[parameter])
			}
		}
		return filtered, true
	}
	return embeddedFunction{}, false
}

func directEmitterSignatureMatches(function embeddedFunction, operation directEmitterOperation) bool {
	if len(function.Parameters) != len(operation.Parameters) ||
		normalizeTypeText(string(function.Result)) != normalizeTypeText(operation.Result) {
		return false
	}
	for i := 0; i < len(function.Parameters); i++ {
		if embeddedParameterType(function.Parameters[i]) != normalizeTypeText(operation.Parameters[i]) {
			return false
		}
	}
	return true
}

func embeddedParameterType(parameter embeddedParameter) string {
	source := string(parameter.Source)
	if len(source) < len(parameter.Name) {
		return ""
	}
	return normalizeTypeText(source[len(parameter.Name):])
}

func normalizeTypeText(value string) string {
	out := make([]byte, 0, len(value))
	for i := 0; i < len(value); i++ {
		if value[i] != ' ' && value[i] != '\t' && value[i] != '\n' && value[i] != '\r' {
			out = append(out, value[i])
		}
	}
	return string(out)
}
