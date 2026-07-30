package rtg

// validateMachineDeclarations checks the cross-declaration facts that are
// required before target composition or source emission. More specialized
// validators can grow beside these checks as each backend contract is migrated.
func validateMachineDeclarations(document Document) []Diagnostic {
	diagnostics := validateImplementsContract(document)
	goNames := embeddedGoNames(document)
	for i := 0; i < len(document.Declarations); i++ {
		declaration := document.Declarations[i]
		diagnostics = append(diagnostics, validateDeclarationFields(document, declaration)...)
		if declaration.Kind == DeclArch {
			diagnostics = append(diagnostics, validateArch(document, declaration, goNames)...)
			diagnostics = append(diagnostics, validateDirectEmitterBindings(document, declaration, goNames)...)
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
	var physicalRegisters []string
	var registerClasses []string
	var addressSpaces []string
	seenForms := []string{}
	seenInstructions := []string{}
	seenExports := []string{}
	seenExportedAlgorithms := []string{}
	for i := 0; i < len(declaration.Statements); i++ {
		statement := declaration.Statements[i]
		left, right, assignment := statementAssignment(statement)
		if assignment && len(left) == 1 && left[0] == "patch_relocations" {
			if len(right) != 2 || right[0] != "go" ||
				stringIndex(goNames, right[1]) < 0 {
				diagnostics = append(diagnostics, statementDiagnostic(document, statement,
					"RTG-VALIDATE-019",
					"architecture patch_relocations must bind an embedded Go algorithm"))
			} else if function, found := findEmbeddedFunction(document, right[1]); found &&
				!directEmitterSignatureMatches(function, directEmitterOperation{
					Name: "patch_relocations", Parameters: []string{"*RTGEmitter"},
				}) {
				diagnostics = append(diagnostics, statementDiagnostic(document, statement,
					"RTG-VALIDATE-076",
					"architecture patch_relocations has an incompatible Go signature"))
			}
		}
		if statementHead(statement, "registers") {
			values := statementListValues(statement)
			seenRegisters := []string{}
			for j := 0; j < len(values); j++ {
				if stringIndex(seenRegisters, values[j]) >= 0 {
					diagnostics = append(diagnostics, statementDiagnostic(document, statement,
						"RTG-VALIDATE-010", "duplicate register "+values[j]+" in architecture "+declaration.Name))
				}
				seenRegisters = append(seenRegisters, values[j])
				if stringIndex(physicalRegisters, values[j]) < 0 {
					physicalRegisters = append(physicalRegisters, values[j])
				}
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
		if statementHead(statement, "register_class") {
			diagnostics = append(diagnostics, validateRegisterClass(
				document, declaration, statement, physicalRegisters, &registerClasses)...)
		}
		if statementBlockName(statement) == "register_group" {
			diagnostics = append(diagnostics, validateRegisterGroup(
				document, declaration, statement, physicalRegisters, &registerClasses)...)
		}
		if statementBlockName(statement) == "address_space" {
			diagnostics = append(diagnostics, validateAddressSpace(
				document, declaration, statement, &addressSpaces)...)
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

func validateRegisterClass(document Document, declaration Declaration, statement Statement,
	physical []string, classes *[]string) []Diagnostic {
	if len(statement.Tokens) < 4 || statement.Tokens[0] != "register_class" ||
		statement.Tokens[2] != "=" {
		return []Diagnostic{statementDiagnostic(document, statement, "RTG-VALIDATE-080",
			"architecture register_class must be named and assigned a register list")}
	}
	name := statement.Tokens[1]
	if stringIndex(*classes, name) >= 0 {
		return []Diagnostic{statementDiagnostic(document, statement, "RTG-VALIDATE-081",
			"duplicate register class "+name+" in architecture "+declaration.Name)}
	}
	*classes = append(*classes, name)
	values := statementListValues(Statement{Tokens: statement.Tokens[1:]})
	var diagnostics []Diagnostic
	var seen []string
	for i := 0; i < len(values); i++ {
		if stringIndex(seen, values[i]) >= 0 {
			diagnostics = append(diagnostics, statementDiagnostic(document, statement,
				"RTG-VALIDATE-082", "duplicate register "+values[i]+" in register class "+name))
		} else if stringIndex(physical, values[i]) < 0 {
			diagnostics = append(diagnostics, statementDiagnostic(document, statement,
				"RTG-VALIDATE-083", "register class "+name+" references unknown register "+values[i]))
		}
		seen = append(seen, values[i])
	}
	if len(values) == 0 {
		diagnostics = append(diagnostics, statementDiagnostic(document, statement,
			"RTG-VALIDATE-084", "register class "+name+" is empty"))
	}
	return diagnostics
}

func validateRegisterGroup(document Document, declaration Declaration, statement Statement,
	physical []string, classes *[]string) []Diagnostic {
	if len(statement.Tokens) != 2 {
		return []Diagnostic{statementDiagnostic(document, statement, "RTG-VALIDATE-085",
			"architecture register_group must have one class name")}
	}
	name := statement.Tokens[1]
	if stringIndex(*classes, name) >= 0 {
		return []Diagnostic{statementDiagnostic(document, statement, "RTG-VALIDATE-081",
			"duplicate register class "+name+" in architecture "+declaration.Name)}
	}
	*classes = append(*classes, name)
	var diagnostics []Diagnostic
	var groups []string
	for i := 0; i < len(statement.Children); i++ {
		left, _, assignment := statementAssignment(statement.Children[i])
		values := statementListValues(statement.Children[i])
		if !assignment || len(left) != 1 || len(values) < 2 {
			diagnostics = append(diagnostics, statementDiagnostic(document, statement.Children[i],
				"RTG-VALIDATE-086", "register group member must name at least two registers"))
			continue
		}
		if stringIndex(groups, left[0]) >= 0 {
			diagnostics = append(diagnostics, statementDiagnostic(document, statement.Children[i],
				"RTG-VALIDATE-087", "duplicate register group member "+left[0]))
		}
		groups = append(groups, left[0])
		for j := 0; j < len(values); j++ {
			if stringIndex(physical, values[j]) < 0 {
				diagnostics = append(diagnostics, statementDiagnostic(document, statement.Children[i],
					"RTG-VALIDATE-088", "register group "+left[0]+" references unknown register "+values[j]))
			}
		}
	}
	if len(statement.Children) == 0 {
		diagnostics = append(diagnostics, statementDiagnostic(document, statement,
			"RTG-VALIDATE-089", "register group class "+name+" is empty"))
	}
	return diagnostics
}

func validateAddressSpace(document Document, declaration Declaration, statement Statement,
	spaces *[]string) []Diagnostic {
	if len(statement.Tokens) != 2 {
		return []Diagnostic{statementDiagnostic(document, statement, "RTG-VALIDATE-090",
			"architecture address_space must have one name")}
	}
	name := statement.Tokens[1]
	if stringIndex(*spaces, name) >= 0 {
		return []Diagnostic{statementDiagnostic(document, statement, "RTG-VALIDATE-091",
			"duplicate address space "+name+" in architecture "+declaration.Name)}
	}
	*spaces = append(*spaces, name)
	allowed := []string{"address_bits", "pointer_bits", "unit_bits", "pointer_words"}
	var diagnostics []Diagnostic
	var seen []string
	values := make([]int, len(allowed))
	present := make([]bool, len(allowed))
	for i := 0; i < len(statement.Children); i++ {
		left, right, assignment := statementAssignment(statement.Children[i])
		if !assignment || len(left) != 1 || len(right) != 1 {
			diagnostics = append(diagnostics, statementDiagnostic(document, statement.Children[i],
				"RTG-VALIDATE-092", "address-space fact must be a single integer assignment"))
			continue
		}
		field := stringIndex(allowed, left[0])
		if field < 0 {
			diagnostics = append(diagnostics, statementDiagnostic(document, statement.Children[i],
				"RTG-VALIDATE-093", "unknown address-space fact "+left[0]))
			continue
		}
		if stringIndex(seen, left[0]) >= 0 {
			diagnostics = append(diagnostics, statementDiagnostic(document, statement.Children[i],
				"RTG-VALIDATE-094", "duplicate address-space fact "+left[0]))
		}
		seen = append(seen, left[0])
		value, ok := parseInteger(right[0])
		if !ok || value <= 0 {
			diagnostics = append(diagnostics, statementDiagnostic(document, statement.Children[i],
				"RTG-VALIDATE-095", "address-space fact "+left[0]+" must be a positive integer"))
			continue
		}
		values[field] = value
		present[field] = true
	}
	if !present[0] || values[0] > 64 {
		diagnostics = append(diagnostics, statementDiagnostic(document, statement,
			"RTG-VALIDATE-096", "address space "+name+" requires address_bits between 1 and 64"))
	}
	for i := 1; i < 3; i++ {
		if !present[i] || !validDescriptorWidth(values[i]) {
			diagnostics = append(diagnostics, statementDiagnostic(document, statement,
				"RTG-VALIDATE-096", "address space "+name+" requires 8, 16, 32, or 64-bit "+
					allowed[i]))
		}
	}
	if !present[3] || values[3] > 4 {
		diagnostics = append(diagnostics, statementDiagnostic(document, statement,
			"RTG-VALIDATE-097", "address space "+name+" requires pointer_words between 1 and 4"))
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
			"stack_word_bytes", "stack_alignment", "unaligned_memory", "patch_relocations",
			"reject",
		}
	}
	if kind == DeclABI {
		return []string{
			"arch", "arguments", "result", "stack_alignment", "shadow_space",
			"saved", "clobbered", "call_words", "overflow_arguments", "aggregate_result",
			"entry_arguments", "foreign_call", "return_address", "frame_pointer",
			"overflow_stride", "scalar_result", "string_result", "slice_result",
			"hidden_result_word", "internal", "caller_saved", "callee_saved", "red_zone",
			"frame_start", "frame_finish", "push_register", "push_immediate",
			"pop_register", "frame_load", "frame_store", "frame_address",
			"store_param_word", "call_word_count",
		}
	}
	if kind == DeclRuntime {
		return []string{
			"operations", "os", "entry", "exit", "allocator", "environment", "arguments",
			"entry_state_bytes", "emit_entry_start", "emit_entry", "emit_exit",
			"emit_static_call", "emit_operation", "entry_prologue", "entry_epilogue",
			"emit_callback_address",
		}
	}
	if kind == DeclFormat {
		return []string{
			"byte_order", "address_bits", "file_alignment", "section_alignment", "page_size",
			"image_base", "machine", "kind", "cpu", "subsystem", "entry", "strip",
			"type", "headers_size", "text_rva", "sections", "code_offset", "image",
			"kernel_image",
		}
	}
	if kind == DeclTarget {
		return []string{
			"family", "arch", "abi", "runtime", "executable", "object", "aliases", "build_tags",
			"capabilities", "code_pointer_bits", "function_pointer_bits", "max_align",
			"arena_default", "subsystem", "os", "frontend_arch",
		}
	}
	if kind == DeclIR {
		return []string{"version", "operations", "capabilities"}
	}
	return nil
}

func validateABI(document Document, declaration Declaration) []Diagnostic {
	var diagnostics []Diagnostic
	current := declaration
	seen := []string{}
	arch := ""
	for {
		if stringIndex(seen, current.Name) >= 0 {
			return append(diagnostics, resolveDiagnostic(document, declaration,
				"RTG-VALIDATE-028", "ABI "+declaration.Name+" has an internal ABI inheritance cycle through "+current.Name))
		}
		seen = append(seen, current.Name)
		currentArch, hasArch := requiredNameField(document, current, "arch")
		internal, hasInternal := requiredNameField(document, current, "internal")
		if hasArch && hasInternal {
			return append(diagnostics, resolveDiagnostic(document, current,
				"RTG-VALIDATE-027", "ABI "+current.Name+" must declare exactly one of arch or internal ABI"))
		}
		if hasArch {
			arch = currentArch
			break
		}
		if !hasInternal {
			code := "RTG-VALIDATE-024"
			message := "ABI " + declaration.Name + " has no architecture through " + current.Name
			if current.Name == declaration.Name {
				code = "RTG-VALIDATE-020"
				message = "ABI " + declaration.Name + " is missing arch or internal ABI"
			}
			return append(diagnostics, resolveDiagnostic(document, current, code, message))
		}
		base, found := document.Declaration(DeclABI, internal)
		if !found {
			return append(diagnostics, resolveDiagnostic(document, current,
				"RTG-VALIDATE-023", "ABI "+current.Name+" references unknown internal ABI "+internal))
		}
		current = base
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
	hookNames := []string{
		"frame_start", "frame_finish", "push_register", "push_immediate",
		"pop_register", "frame_load", "frame_store", "frame_address",
		"store_param_word", "call_word_count",
	}
	hookParameters := make([][]string, 10)
	hookParameters[0] = []string{"*RTGEmitter"}
	hookParameters[1] = []string{"*RTGEmitter", "int", "int"}
	hookParameters[2] = []string{"*RTGEmitter", "RTGRegister"}
	hookParameters[3] = []string{"*RTGEmitter", "int"}
	hookParameters[4] = []string{"*RTGEmitter", "RTGRegister"}
	hookParameters[5] = []string{"*RTGEmitter", "RTGRegister", "int"}
	hookParameters[6] = []string{"*RTGEmitter", "int", "RTGRegister"}
	hookParameters[7] = []string{"*RTGEmitter", "RTGRegister", "int"}
	hookParameters[8] = []string{"*RTGEmitter", "int", "int"}
	hookParameters[9] = []string{"*RTGEmitter", "RTGLabel", "int"}
	for i := 0; i < len(declaration.Statements); i++ {
		left, right, assignment := statementAssignment(declaration.Statements[i])
		if !assignment || len(left) != 1 {
			continue
		}
		hook := stringIndex(hookNames, left[0])
		if hook < 0 {
			continue
		}
		if len(right) != 2 || right[0] != "go" {
			diagnostics = append(diagnostics, statementDiagnostic(document, declaration.Statements[i],
				"RTG-VALIDATE-025", "ABI "+left[0]+" must bind an embedded Go algorithm"))
			continue
		}
		function, found := findEmbeddedFunction(document, right[1])
		result := ""
		if left[0] == "frame_start" {
			result = "int"
		}
		if !found || !directEmitterSignatureMatches(function, directEmitterOperation{
			Name: left[0], Parameters: hookParameters[hook], Result: result,
		}) {
			diagnostics = append(diagnostics, statementDiagnostic(document, declaration.Statements[i],
				"RTG-VALIDATE-026", "ABI "+left[0]+" has an incompatible Go signature"))
		}
	}
	return diagnostics
}

func validateRuntime(document Document, declaration Declaration) []Diagnostic {
	hookNames := []string{
		"emit_entry_start", "emit_entry", "emit_exit", "emit_static_call",
		"emit_operation", "entry_prologue", "entry_epilogue", "emit_callback_address",
	}
	hookParameters := make([][]string, 8)
	hookParameters[0] = []string{"*RTGEmitter", "int"}
	hookParameters[1] = []string{"*RTGEmitter", "int", "int"}
	hookParameters[2] = []string{"*RTGEmitter", "RTGRegister"}
	hookParameters[3] = []string{"*RTGEmitter", "int", "int"}
	hookParameters[4] = []string{"*RTGEmitter", "int"}
	hookParameters[5] = []string{"*RTGEmitter"}
	hookParameters[6] = []string{"*RTGEmitter"}
	hookParameters[7] = []string{"*RTGEmitter", "RTGLabel"}
	hookResults := []string{"bool", "bool", "", "", "bool", "", "", ""}
	for i := 0; i < len(declaration.Statements); i++ {
		left, right, assignment := statementAssignment(declaration.Statements[i])
		if assignment && len(left) == 1 {
			hook := stringIndex(hookNames, left[0])
			if hook < 0 {
				continue
			}
			if len(right) != 2 || right[0] != "go" {
				return []Diagnostic{statementDiagnostic(document, declaration.Statements[i],
					"RTG-VALIDATE-031", "runtime "+left[0]+" must bind an embedded Go algorithm")}
			}
			function, found := findEmbeddedFunction(document, right[1])
			if !found || !directEmitterSignatureMatches(function, directEmitterOperation{
				Name: left[0], Parameters: hookParameters[hook],
				Result: hookResults[hook],
			}) {
				return []Diagnostic{statementDiagnostic(document, declaration.Statements[i],
					"RTG-VALIDATE-032", "runtime "+left[0]+" has an incompatible Go signature")}
			}
		}
	}
	allowed := []string{
		"print", "open", "close", "read", "write", "read_at", "write_at", "chmod",
		"seek", "exit",
	}
	var diagnostics []Diagnostic
	operations := listField(document, declaration, "operations")
	operationStatements := make([]Statement, len(operations))
	for i := 0; i < len(declaration.Statements); i++ {
		statement := declaration.Statements[i]
		if statementBlockName(statement) != "operation" || len(statement.Tokens) != 2 {
			continue
		}
		operations = append(operations, statement.Tokens[1])
		operationStatements = append(operationStatements, statement)
	}
	seen := []string{}
	for i := 0; i < len(operations); i++ {
		diagnostic := func(code string, message string) Diagnostic {
			if i < len(operationStatements) && len(operationStatements[i].Tokens) != 0 {
				return statementDiagnostic(document, operationStatements[i], code, message)
			}
			return resolveDiagnostic(document, declaration, code, message)
		}
		if stringIndex(allowed, operations[i]) < 0 {
			diagnostics = append(diagnostics, diagnostic("RTG-VALIDATE-030",
				"runtime "+declaration.Name+" declares unknown operation "+operations[i]))
		}
		if stringIndex(seen, operations[i]) >= 0 {
			diagnostics = append(diagnostics, diagnostic("RTG-VALIDATE-033",
				"runtime "+declaration.Name+" declares duplicate operation "+operations[i]))
		}
		seen = append(seen, operations[i])
	}
	return diagnostics
}

func validateFormat(document Document, declaration Declaration) []Diagnostic {
	var diagnostics []Diagnostic
	hookNames := []string{"image", "kernel_image"}
	hookParameters := make([][]string, 2)
	hookParameters[0] = []string{"*RTGEmitter"}
	hookParameters[1] = []string{"*RTGEmitter", "RTGLabel", "RTGLabel"}
	for i := 0; i < len(declaration.Statements); i++ {
		statement := declaration.Statements[i]
		left, right, assignment := statementAssignment(statement)
		if !assignment || len(left) != 1 {
			continue
		}
		hook := stringIndex(hookNames, left[0])
		if hook < 0 {
			continue
		}
		if len(right) != 2 || right[0] != "go" {
			diagnostics = append(diagnostics, statementDiagnostic(document, statement,
				"RTG-VALIDATE-042", "format "+left[0]+" must bind an embedded Go algorithm"))
			continue
		}
		function, found := findEmbeddedFunction(document, right[1])
		if !found || !directEmitterSignatureMatches(function, directEmitterOperation{
			Name: left[0], Parameters: hookParameters[hook], Result: "[]byte",
		}) {
			diagnostics = append(diagnostics, statementDiagnostic(document, statement,
				"RTG-VALIDATE-043", "format "+left[0]+" has an incompatible Go signature"))
		}
	}
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
		if order, found := fieldValue(document, formats[i], "byte_order"); found &&
			valueName(order) != target.Descriptor.Endian {
			diagnostics = append(diagnostics, resolveDiagnostic(document, target.Declaration,
				"RTG-VALIDATE-052", "target "+target.Descriptor.Name+" composes "+formats[i].Name+
					" byte order with a different architecture byte order"))
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
