package rtg

type generatedSymbol struct {
	Local  string
	Output string
	Kind   string
	Code   int
	Value  string
	Extra  string
}

func architectureLocalPrefix(name string) string {
	var out []byte
	for i := 0; i < len(name); i++ {
		if isIdentPart(name[i]) {
			out = append(out, name[i])
		}
	}
	if len(out) == 0 {
		return "arch"
	}
	return string(out)
}

func upperIdentifier(name string) string {
	var out []byte
	for i := 0; i < len(name); i++ {
		ch := name[i]
		if ch >= 'a' && ch <= 'z' {
			ch -= 'a' - 'A'
		}
		if isIdentPart(ch) {
			out = append(out, ch)
		}
	}
	return string(out)
}

func instructionGoSuffix(name string) string {
	var out []byte
	start := 0
	for start < len(name) {
		end := start
		for end < len(name) && name[end] != '_' && name[end] != '-' {
			end++
		}
		part := name[start:end]
		replacement := ""
		if abbreviationWithNumericSuffix(part, "cl") {
			replacement = "CL" + optionalExportedName(part[2:])
		} else if abbreviationWithNumericSuffix(part, "imm") {
			replacement = "Immediate" + optionalExportedName(part[3:])
		} else if abbreviationWithNumericSuffix(part, "reg") {
			replacement = "Register" + optionalExportedName(part[3:])
		} else if abbreviationWithNumericSuffix(part, "mem") {
			replacement = "Memory" + optionalExportedName(part[3:])
		} else {
			replacement = exportedName(part)
		}
		out = append(out, replacement...)
		start = end + 1
	}
	return string(out)
}

func optionalExportedName(value string) string {
	if value == "" {
		return ""
	}
	return exportedName(value)
}

func hasPrefixText(value string, prefix string) bool {
	return len(value) >= len(prefix) && value[:len(prefix)] == prefix
}

func abbreviationWithNumericSuffix(value string, prefix string) bool {
	if !hasPrefixText(value, prefix) {
		return false
	}
	for i := len(prefix); i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return true
}

func architectureSymbols(document Document, arch Declaration) []generatedSymbol {
	localPrefix := architectureLocalPrefix(arch.Name)
	outputPrefix := "rtg" + exportedName(document.Unit)
	var symbols []generatedSymbol
	for i := 0; i < len(arch.Statements); i++ {
		statement := arch.Statements[i]
		if statementHead(statement, "registers") {
			registers := statementListValues(statement)
			for code := 0; code < len(registers); code++ {
				suffix := upperIdentifier(registers[code])
				symbols = append(symbols, generatedSymbol{
					Local: localPrefix + suffix, Output: outputPrefix + suffix,
					Kind: "register", Code: code,
				})
			}
		}
	}
	locations, hasLocations := declarationBlock(arch, "locations")
	if hasLocations {
		for i := 0; i < len(locations.Children); i++ {
			left, right, ok := statementAssignment(locations.Children[i])
			if !ok || len(left) != 1 || len(right) != 1 {
				continue
			}
			suffix := exportedName(left[0])
			symbols = append(symbols, generatedSymbol{
				Local: localPrefix + suffix, Output: outputPrefix + suffix,
				Kind: "location", Value: localPrefix + upperIdentifier(right[0]),
			})
		}
	}
	conditions, hasConditions := declarationBlock(arch, "conditions")
	if hasConditions {
		for i := 0; i < len(conditions.Children); i++ {
			child := conditions.Children[i]
			tokens := child.Tokens
			name := ""
			value := ""
			extra := ""
			left, right, assignment := statementAssignment(child)
			if assignment && len(left) == 1 && len(right) == 1 {
				name = left[0]
				value = right[0]
			} else if len(tokens) >= 2 {
				name = tokens[0]
				value = tokens[1]
				if len(tokens) > 2 {
					extra = tokens[2]
				}
			}
			if name == "" || value == "" {
				continue
			}
			suffix := upperIdentifier(name)
			symbols = append(symbols, generatedSymbol{
				Local: localPrefix + suffix, Output: outputPrefix + suffix,
				Kind: "condition", Value: value, Extra: extra,
			})
		}
	}
	instructions, hasInstructions := declarationBlock(arch, "instructions")
	if hasInstructions {
		for i := 0; i < len(instructions.Children); i++ {
			child := instructions.Children[i]
			left, _, assignment := statementAssignment(child)
			name := ""
			if assignment && len(left) == 1 {
				name = left[0]
			} else if len(child.Tokens) > 0 {
				name = child.Tokens[0]
			}
			if name == "" {
				continue
			}
			suffix := instructionGoSuffix(name)
			symbols = append(symbols, generatedSymbol{
				Local: localPrefix + suffix, Output: outputPrefix + suffix,
				Kind: "instruction", Value: name,
			})
		}
	}
	return symbols
}

func generatedArchitectureOutput(document Document, local string) (string, bool) {
	for i := 0; i < len(document.Declarations); i++ {
		if document.Declarations[i].Kind != DeclArch {
			continue
		}
		symbols := architectureSymbols(document, document.Declarations[i])
		for j := 0; j < len(symbols); j++ {
			if symbols[j].Local == local {
				return symbols[j].Output, true
			}
		}
	}
	return "", false
}

func generatedArchitectureCode(document Document, local string) (int, bool) {
	for i := 0; i < len(document.Declarations); i++ {
		if document.Declarations[i].Kind != DeclArch {
			continue
		}
		symbols := architectureSymbols(document, document.Declarations[i])
		name := local
		for depth := 0; depth <= len(symbols); depth++ {
			found := false
			for j := 0; j < len(symbols); j++ {
				if symbols[j].Local != name {
					continue
				}
				if symbols[j].Kind == "register" {
					return symbols[j].Code, true
				}
				if symbols[j].Kind == "location" {
					name = symbols[j].Value
					if len(name) >= 3 && name[len(name)-3:] == "XZR" {
						return 31, true
					}
					found = true
				}
				break
			}
			if !found {
				break
			}
		}
	}
	return 0, false
}

func appendArchitectureFacts(out []byte, document Document, arch Declaration, mangle bool) []byte {
	symbols := architectureSymbols(document, arch)
	var declared []string
	for i := 0; i < len(symbols); i++ {
		symbol := symbols[i]
		if symbol.Kind == "instruction" {
			continue
		}
		name := symbol.Local
		if mangle {
			name = symbol.Output
		}
		if stringIndex(declared, name) >= 0 {
			continue
		}
		if symbol.Kind == "register" {
			out = append(out, "\nvar "...)
			out = append(out, name...)
			out = append(out, " = RTGRegister{Code:"...)
			out = appendDecimalFrame(out, symbol.Code)
			out = append(out, ", Valid:true}\n"...)
			declared = append(declared, name)
		} else if symbol.Kind == "location" {
			value := symbol.Value
			if mangle {
				if replacement, found := generatedArchitectureOutput(document, value); found {
					value = replacement
				}
			}
			if stringIndex(declared, value) < 0 {
				code := 0
				if len(symbol.Value) >= 2 && symbol.Value[len(symbol.Value)-2:] == "ZR" {
					code = 31
				}
				out = append(out, "\nvar "...)
				out = append(out, value...)
				out = append(out, " = RTGRegister{Code:"...)
				out = appendDecimalFrame(out, code)
				out = append(out, ", Valid:true}\n"...)
				declared = append(declared, value)
			}
			out = append(out, "var "...)
			out = append(out, name...)
			out = append(out, " = "...)
			out = append(out, value...)
			out = append(out, '\n')
			declared = append(declared, name)
		} else if symbol.Kind == "condition" {
			out = append(out, "\nvar "...)
			out = append(out, name...)
			out = append(out, " = RTGCondition{Code:"...)
			out = append(out, symbol.Value...)
			if symbol.Extra != "" {
				out = append(out, ", SetOpcode:"...)
				out = append(out, symbol.Value...)
				out = append(out, ", JumpOpcode:"...)
				out = append(out, symbol.Extra...)
			}
			out = append(out, "}\n"...)
			declared = append(declared, name)
		}
	}
	return out
}

func appendGeneratedSymbolPrelude(out []byte, document Document) []byte {
	for i := 0; i < len(document.Declarations); i++ {
		arch := document.Declarations[i]
		if arch.Kind != DeclArch {
			continue
		}
		symbols := architectureSymbols(document, arch)
		var declared []string
		for j := 0; j < len(symbols); j++ {
			symbol := symbols[j]
			if stringIndex(declared, symbol.Local) >= 0 {
				continue
			}
			if symbol.Kind == "register" {
				out = append(out, "var "...)
				out = append(out, symbol.Local...)
				out = append(out, " = RTGRegister{Code:"...)
				out = appendDecimalFrame(out, symbol.Code)
				out = append(out, ",Valid:true}\n"...)
				declared = append(declared, symbol.Local)
			} else if symbol.Kind == "location" {
				if stringIndex(declared, symbol.Value) < 0 {
					out = append(out, "var "...)
					out = append(out, symbol.Value...)
					out = append(out, " RTGRegister\n"...)
					declared = append(declared, symbol.Value)
				}
				out = append(out, "var "...)
				out = append(out, symbol.Local...)
				out = append(out, " = "...)
				out = append(out, symbol.Value...)
				out = append(out, '\n')
				declared = append(declared, symbol.Local)
			} else if symbol.Kind == "condition" {
				out = append(out, "var "...)
				out = append(out, symbol.Local...)
				out = append(out, " = RTGCondition{Code:"...)
				out = append(out, symbol.Value...)
				if symbol.Extra != "" {
					out = append(out, ",SetOpcode:"...)
					out = append(out, symbol.Value...)
					out = append(out, ",JumpOpcode:"...)
					out = append(out, symbol.Extra...)
				}
				out = append(out, "}\n"...)
				declared = append(declared, symbol.Local)
			}
		}
		out = appendInstructionSymbolPrelude(out, document, arch)
	}
	return out
}

func appendInstructionSymbolPrelude(out []byte, document Document, arch Declaration) []byte {
	instructions, ok := declarationBlock(arch, "instructions")
	if !ok {
		return out
	}
	localPrefix := architectureLocalPrefix(arch.Name)
	forms := architectureForms(arch)
	for i := 0; i < len(instructions.Children); i++ {
		child := instructions.Children[i]
		left, right, assignment := statementAssignment(child)
		name := ""
		var function embeddedFunction
		found := false
		var constants []string
		if assignment && len(left) == 1 && len(right) == 2 && right[0] == "go" {
			name = left[0]
			function, found = findEmbeddedFunction(document, right[1])
		} else if len(child.Tokens) >= 2 {
			name = child.Tokens[0]
			form, formFound := lookupArchitectureForm(forms, child.Tokens[1])
			if formFound && form.Kind == "go" {
				function, found = findEmbeddedFunction(document, form.Algorithm)
				constants = instructionConstants(child.Tokens)
			} else if formFound && form.Kind == "bytes" {
				out = append(out, "var "...)
				out = append(out, localPrefix...)
				out = append(out, instructionGoSuffix(name)...)
				out = append(out, " func(out *RTGEmitter)\n"...)
			}
		}
		if name == "" || !found {
			continue
		}
		out = append(out, "var "...)
		out = append(out, localPrefix...)
		out = append(out, instructionGoSuffix(name)...)
		out = append(out, " func("...)
		first := true
		for parameter := 0; parameter < len(function.Parameters); parameter++ {
			if instructionConstant(constants, function.Parameters[parameter].Name) != "" {
				continue
			}
			if !first {
				out = append(out, ", "...)
			}
			first = false
			out = append(out, function.Parameters[parameter].Source...)
		}
		out = append(out, ')')
		if len(function.Result) != 0 {
			out = append(out, ' ')
			out = append(out, function.Result...)
		}
		out = append(out, '\n')
	}
	return out
}
