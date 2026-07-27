package rtg

// validateMachineDeclarations checks the cross-declaration facts that are
// required before target composition or source emission. More specialized
// validators can grow beside these checks as each backend contract is migrated.
func validateMachineDeclarations(document Document) []Diagnostic {
	var diagnostics []Diagnostic
	goNames := embeddedGoNames(document)
	for i := 0; i < len(document.Declarations); i++ {
		declaration := document.Declarations[i]
		diagnostics = append(diagnostics, validateDeclarationFields(document, declaration)...)
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
	seenForms := []string{}
	seenInstructions := []string{}
	seenExports := []string{}
	seenExportedAlgorithms := []string{}
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
		if statementBlockName(statement) == "forms" {
			for j := 0; j < len(statement.Children); j++ {
				name, value, ok := statementAssignment(statement.Children[j])
				if !ok || len(name) != 1 {
					continue
				}
				if stringIndex(seenForms, name[0]) >= 0 {
					diagnostics = append(diagnostics, statementDiagnostic(document, statement.Children[j],
						"RTG-VALIDATE-012", "duplicate instruction form "+name[0]))
				}
				seenForms = append(seenForms, name[0])
				if len(value) == 1 && value[0] == "bytes" {
					continue
				}
				if len(value) != 2 || value[0] != "go" {
					diagnostics = append(diagnostics, statementDiagnostic(document, statement.Children[j],
						"RTG-VALIDATE-015", "instruction form "+name[0]+" must bind go algorithm or bytes"))
					continue
				}
				if stringIndex(goNames, value[1]) < 0 {
					diagnostics = append(diagnostics, statementDiagnostic(document, statement.Children[j],
						"RTG-VALIDATE-011", "unknown embedded Go algorithm "+value[1]))
				}
			}
		}
		if statementBlockName(statement) == "exports" {
			for j := 0; j < len(statement.Children); j++ {
				left, right, assignment := statementAssignment(statement.Children[j])
				if !assignment || len(left) != 1 || len(right) != 2 || right[0] != "go" {
					diagnostics = append(diagnostics, statementDiagnostic(document, statement.Children[j],
						"RTG-VALIDATE-016", "backend export must be name = go algorithm"))
					continue
				}
				if stringIndex(seenExports, left[0]) >= 0 {
					diagnostics = append(diagnostics, statementDiagnostic(document, statement.Children[j],
						"RTG-VALIDATE-017", "duplicate backend export "+left[0]))
				}
				if stringIndex(seenExportedAlgorithms, right[1]) >= 0 {
					diagnostics = append(diagnostics, statementDiagnostic(document, statement.Children[j],
						"RTG-VALIDATE-018", "embedded Go algorithm exported more than once "+right[1]))
				}
				if stringIndex(goNames, right[1]) < 0 {
					diagnostics = append(diagnostics, statementDiagnostic(document, statement.Children[j],
						"RTG-VALIDATE-011", "unknown embedded Go algorithm "+right[1]))
				}
				seenExports = append(seenExports, left[0])
				seenExportedAlgorithms = append(seenExportedAlgorithms, right[1])
			}
		}
	}
	for i := 0; i < len(declaration.Statements); i++ {
		statement := declaration.Statements[i]
		if statementBlockName(statement) != "instructions" {
			continue
		}
		for j := 0; j < len(statement.Children); j++ {
			child := statement.Children[j]
			left, value, assignment := statementAssignment(child)
			name := ""
			if assignment && len(left) == 1 {
				name = left[0]
				if len(value) == 2 && value[0] == "go" && stringIndex(goNames, value[1]) < 0 {
					diagnostics = append(diagnostics, statementDiagnostic(document, child,
						"RTG-VALIDATE-011", "unknown embedded Go algorithm "+value[1]))
				}
			} else if len(child.Tokens) >= 2 {
				name = child.Tokens[0]
				if stringIndex(seenForms, child.Tokens[1]) < 0 && child.Tokens[1] != "fixed" {
					diagnostics = append(diagnostics, statementDiagnostic(document, child,
						"RTG-VALIDATE-013", "instruction "+name+" references unknown form "+child.Tokens[1]))
				}
			}
			if name == "" {
				continue
			}
			if stringIndex(seenInstructions, name) >= 0 {
				diagnostics = append(diagnostics, statementDiagnostic(document, child,
					"RTG-VALIDATE-014", "duplicate instruction "+name))
			}
			seenInstructions = append(seenInstructions, name)
		}
	}
	return diagnostics
}

func validateDeclarationFields(document Document, declaration Declaration) []Diagnostic {
	allowed := declarationAllowedFields(declaration.Kind)
	if len(allowed) == 0 {
		return nil
	}
	var diagnostics []Diagnostic
	var seen []string
	for i := 0; i < len(declaration.Fields); i++ {
		field := declaration.Fields[i]
		if stringIndex(seen, field.Name) >= 0 {
			diagnostics = append(diagnostics, Diagnostic{
				Filename: document.Filename,
				Span:     field.Span,
				Code:     "RTG-VALIDATE-060",
				Message:  "duplicate field " + field.Name + " in " + declaration.Kind + " " + declaration.Name,
			})
		}
		seen = append(seen, field.Name)
		if stringIndex(allowed, field.Name) < 0 && !declarationOwnsNamedAssignment(declaration, field.Name) {
			diagnostics = append(diagnostics, Diagnostic{
				Filename: document.Filename,
				Span:     field.Span,
				Code:     "RTG-VALIDATE-061",
				Message:  "unknown field " + field.Name + " in " + declaration.Kind + " " + declaration.Name,
			})
		}
	}
	return diagnostics
}

func declarationOwnsNamedAssignment(declaration Declaration, name string) bool {
	for i := 0; i < len(declaration.Statements); i++ {
		tokens := declaration.Statements[i].Tokens
		if len(tokens) >= 3 && tokens[0] == "registers" && tokens[1] == name && tokens[2] == "=" {
			return true
		}
		for j := 1; j+1 < len(tokens); j++ {
			if tokens[j] == name && tokens[j+1] == "=" {
				return true
			}
		}
	}
	return false
}

func declarationAllowedFields(kind string) []string {
	if kind == DeclArch {
		return []string{
			"alias", "endian", "word_bits", "pointer_bits", "instruction_alignment",
			"stack_word_bytes", "stack_alignment", "unaligned_memory",
		}
	}
	if kind == DeclABI {
		return []string{
			"arch", "arguments", "result", "stack_alignment", "shadow_space",
			"saved", "clobbered", "call_words", "overflow_arguments", "aggregate_result",
			"entry_arguments", "foreign_call", "return_address", "frame_pointer",
			"overflow_stride", "scalar_result", "string_result", "slice_result",
			"hidden_result_word", "internal", "caller_saved", "callee_saved", "red_zone",
		}
	}
	if kind == DeclRuntime {
		return []string{"operations", "os", "entry", "exit", "allocator", "environment", "arguments"}
	}
	if kind == DeclFormat {
		return []string{
			"byte_order", "address_bits", "file_alignment", "section_alignment", "page_size",
			"image_base", "machine", "kind", "cpu", "subsystem", "entry", "strip",
			"type", "headers_size", "text_rva", "sections",
		}
	}
	if kind == DeclTarget {
		return []string{
			"arch", "abi", "runtime", "executable", "object", "aliases", "build_tags",
			"capabilities", "code_pointer_bits", "function_pointer_bits", "max_align",
			"arena_default", "subsystem",
		}
	}
	if kind == DeclIR {
		return []string{"version", "operations", "capabilities"}
	}
	return nil
}

func validateABI(document Document, declaration Declaration) []Diagnostic {
	var diagnostics []Diagnostic
	arch, ok := requiredNameField(document, declaration, "arch")
	if !ok {
		internal, hasInternal := requiredNameField(document, declaration, "internal")
		if !hasInternal {
			return append(diagnostics, resolveDiagnostic(document, declaration,
				"RTG-VALIDATE-020", "ABI "+declaration.Name+" is missing arch or internal ABI"))
		}
		base, found := document.Declaration(DeclABI, internal)
		if !found {
			return append(diagnostics, resolveDiagnostic(document, declaration,
				"RTG-VALIDATE-023", "ABI "+declaration.Name+" references unknown internal ABI "+internal))
		}
		arch, ok = requiredNameField(document, base, "arch")
		if !ok {
			return append(diagnostics, resolveDiagnostic(document, declaration,
				"RTG-VALIDATE-024", "ABI "+declaration.Name+" has no architecture through "+internal))
		}
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
	if abiArch, ok := abiArchitecture(document, target.ABI); ok &&
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

func abiArchitecture(document Document, declaration Declaration) (string, bool) {
	seen := []string{}
	for declaration.Name != "" {
		if stringIndex(seen, declaration.Name) >= 0 {
			return "", false
		}
		seen = append(seen, declaration.Name)
		if arch, ok := requiredNameField(document, declaration, "arch"); ok {
			return arch, true
		}
		internal, ok := requiredNameField(document, declaration, "internal")
		if !ok {
			return "", false
		}
		declaration, ok = document.Declaration(DeclABI, internal)
		if !ok {
			return "", false
		}
	}
	return "", false
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
