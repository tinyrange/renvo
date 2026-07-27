package rtg

// validateMachineDeclarations checks the cross-declaration facts that are
// required before target composition or source emission. More specialized
// validators can grow beside these checks as each backend contract is migrated.
func validateMachineDeclarations(document Document) []Diagnostic {
	var diagnostics []Diagnostic
	goNames := embeddedGoNames(document)
	for i := 0; i < len(document.Declarations); i++ {
		declaration := document.Declarations[i]
		if declaration.Kind == DeclArch {
			diagnostics = append(diagnostics, validateArch(document, declaration, goNames)...)
		} else if declaration.Kind == DeclABI {
			diagnostics = append(diagnostics, validateABI(document, declaration)...)
		} else if declaration.Kind == DeclRuntime {
			diagnostics = append(diagnostics, validateRuntime(document, declaration)...)
		} else if declaration.Kind == DeclFormat {
			diagnostics = append(diagnostics, validateFormat(document, declaration)...)
		}
	}
	return diagnostics
}

func validateArch(document Document, declaration Declaration, goNames []string) []Diagnostic {
	var diagnostics []Diagnostic
	seenRegisters := []string{}
	for i := 0; i < len(declaration.Statements); i++ {
		statement := declaration.Statements[i]
		if statementHead(statement, "registers") {
			values := statementListValues(statement)
			for j := 0; j < len(values); j++ {
				if stringIndex(seenRegisters, values[j]) >= 0 {
					diagnostics = append(diagnostics, statementDiagnostic(document, statement,
						"RTG-VALIDATE-010", "duplicate register "+values[j]+" in architecture "+declaration.Name))
				}
				seenRegisters = append(seenRegisters, values[j])
			}
		}
		if statementBlockName(statement) == "forms" || statementBlockName(statement) == "instructions" {
			for j := 0; j < len(statement.Children); j++ {
				_, value, ok := statementAssignment(statement.Children[j])
				if !ok || len(value) < 2 || value[0] != "go" {
					continue
				}
				if stringIndex(goNames, value[1]) < 0 {
					diagnostics = append(diagnostics, statementDiagnostic(document, statement.Children[j],
						"RTG-VALIDATE-011", "unknown embedded Go algorithm "+value[1]))
				}
			}
		}
	}
	return diagnostics
}

func validateABI(document Document, declaration Declaration) []Diagnostic {
	var diagnostics []Diagnostic
	arch, ok := requiredNameField(document, declaration, "arch")
	if !ok {
		return append(diagnostics, resolveDiagnostic(document, declaration,
			"RTG-VALIDATE-020", "ABI "+declaration.Name+" is missing arch"))
	}
	if _, found := document.Declaration(DeclArch, arch); !found {
		diagnostics = append(diagnostics, resolveDiagnostic(document, declaration,
			"RTG-VALIDATE-021", "ABI "+declaration.Name+" references unknown arch "+arch))
	}
	if alignment, found := integerField(document, declaration, "stack_alignment"); found &&
		(alignment <= 0 || alignment&(alignment-1) != 0) {
		diagnostics = append(diagnostics, resolveDiagnostic(document, declaration,
			"RTG-VALIDATE-022", "ABI "+declaration.Name+" has non-power-of-two stack_alignment"))
	}
	return diagnostics
}

func validateRuntime(document Document, declaration Declaration) []Diagnostic {
	operations := listField(document, declaration, "operations")
	if len(operations) == 0 {
		return nil
	}
	allowed := []string{"print", "open", "close", "read", "write", "chmod"}
	var diagnostics []Diagnostic
	for i := 0; i < len(operations); i++ {
		if stringIndex(allowed, operations[i]) < 0 {
			diagnostics = append(diagnostics, resolveDiagnostic(document, declaration,
				"RTG-VALIDATE-030", "runtime "+declaration.Name+" declares unknown operation "+operations[i]))
		}
	}
	return diagnostics
}

func validateFormat(document Document, declaration Declaration) []Diagnostic {
	var diagnostics []Diagnostic
	if bits, found := integerField(document, declaration, "address_bits"); found &&
		bits != 8 && bits != 16 && bits != 32 && bits != 64 {
		diagnostics = append(diagnostics, resolveDiagnostic(document, declaration,
			"RTG-VALIDATE-040", "format "+declaration.Name+" has invalid address_bits"))
	}
	if order, found := fieldValue(document, declaration, "byte_order"); found {
		order = valueName(order)
		if order != "little" && order != "big" {
			diagnostics = append(diagnostics, resolveDiagnostic(document, declaration,
				"RTG-VALIDATE-041", "format "+declaration.Name+" has invalid byte_order"))
		}
	}
	return diagnostics
}

func validateTargetComposition(document Document, target ResolvedTarget) []Diagnostic {
	var diagnostics []Diagnostic
	if abiArch, ok := requiredNameField(document, target.ABI, "arch"); ok &&
		abiArch != target.Arch.Name {
		diagnostics = append(diagnostics, resolveDiagnostic(document, target.Declaration,
			"RTG-VALIDATE-050", "target "+target.Descriptor.Name+" composes ABI "+target.ABI.Name+
				" for "+abiArch+" with architecture "+target.Arch.Name))
	}
	formats := []Declaration{target.Executable, target.Object}
	for i := 0; i < len(formats); i++ {
		if formats[i].Name == "" {
			continue
		}
		if bits, found := integerField(document, formats[i], "address_bits"); found &&
			bits != target.Descriptor.PointerBits {
			diagnostics = append(diagnostics, resolveDiagnostic(document, target.Declaration,
				"RTG-VALIDATE-051", "target "+target.Descriptor.Name+" composes "+formats[i].Name+
					" address width with a different architecture pointer width"))
		}
	}
	return diagnostics
}

func statementHead(statement Statement, name string) bool {
	return len(statement.Tokens) > 0 && statement.Tokens[0] == name
}

func statementBlockName(statement Statement) string {
	if len(statement.Children) == 0 || len(statement.Tokens) == 0 {
		return ""
	}
	return statement.Tokens[0]
}

func statementAssignment(statement Statement) ([]string, []string, bool) {
	for i := 0; i < len(statement.Tokens); i++ {
		if statement.Tokens[i] == "=" {
			if i == 0 || i+1 == len(statement.Tokens) {
				return nil, nil, false
			}
			return statement.Tokens[:i], statement.Tokens[i+1:], true
		}
	}
	return nil, nil, false
}

func statementListValues(statement Statement) []string {
	_, value, ok := statementAssignment(statement)
	if !ok {
		return nil
	}
	var values []string
	for i := 0; i < len(value); i++ {
		token := value[i]
		if token == "[" || token == "]" || token == "," {
			continue
		}
		values = append(values, valueName(token))
	}
	return values
}

func statementDiagnostic(document Document, statement Statement, code string, message string) Diagnostic {
	return Diagnostic{Filename: document.Filename, Span: statement.Span, Code: code, Message: message}
}
