package rtg

// architectureSequence is a bounded declarative function: its body is a
// straight-line list of calls, assignments, declarations, increments, and one
// final return. It is intentionally sufficient for regular encoder/ABI/runtime
// glue without becoming a second control-flow language.
type architectureSequence struct {
	Name       string
	Parameters []embeddedParameter
	Result     []byte
	HasResult  bool
	Steps      []Statement
	Statement  Statement
}

func architectureSequences(arch Declaration) []architectureSequence {
	block, ok := declarationBlock(arch, "sequences")
	if !ok {
		return nil
	}
	var sequences []architectureSequence
	for i := 0; i < len(block.Children); i++ {
		statement := block.Children[i]
		sequence, ok := decodeArchitectureSequence(statement)
		if ok {
			sequences = append(sequences, sequence)
		}
	}
	return sequences
}

func decodeArchitectureSequence(statement Statement) (architectureSequence, bool) {
	tokens := statement.Tokens
	if len(tokens) < 3 || tokens[1] != "(" {
		return architectureSequence{}, false
	}
	close := -1
	depth := 0
	for i := 1; i < len(tokens); i++ {
		if tokens[i] == "(" {
			depth++
		} else if tokens[i] == ")" {
			depth--
			if depth == 0 {
				close = i
				break
			}
		}
	}
	if close < 0 {
		return architectureSequence{}, false
	}
	sequence := architectureSequence{Name: tokens[0], Steps: statement.Children, Statement: statement}
	for at := 2; at < close; {
		if tokens[at] == "," {
			at++
			continue
		}
		if at+2 >= close || tokens[at+1] != ":" {
			return architectureSequence{}, false
		}
		name := tokens[at]
		kind := tokens[at+2]
		goType, ok := sequenceGoType(kind)
		if !ok {
			return architectureSequence{}, false
		}
		sequence.Parameters = append(sequence.Parameters, embeddedParameter{
			Name: name, Source: []byte(name + " " + goType),
		})
		at += 3
	}
	if close+2 < len(tokens) && tokens[close+1] == "-" && tokens[close+2] == ">" {
		if close+3 >= len(tokens) {
			return architectureSequence{}, false
		}
		goType, ok := sequenceGoType(tokens[close+3])
		if !ok || close+4 != len(tokens) {
			return architectureSequence{}, false
		}
		sequence.Result = []byte(goType)
		sequence.HasResult = true
	} else if close+1 != len(tokens) {
		return architectureSequence{}, false
	}
	return sequence, true
}

func sequenceGoType(kind string) (string, bool) {
	if kind == "emitter" {
		return "*RTGEmitter", true
	}
	if kind == "register" {
		return "RTGRegister", true
	}
	if kind == "address" {
		return "RTGAddress", true
	}
	if kind == "condition" {
		return "RTGCondition", true
	}
	if kind == "label" {
		return "RTGLabel", true
	}
	if kind == "symbol" {
		return "*RTGSymbol", true
	}
	if kind == "shift_direction" {
		return "RTGShiftDirection", true
	}
	if kind == "bytes" {
		return "[]byte", true
	}
	if kind == "ints" {
		return "[]int", true
	}
	if kind == "int" || kind == "int64" || kind == "uint64" ||
		kind == "byte" || kind == "bool" || kind == "string" {
		return kind, true
	}
	return "", false
}

func architectureSequenceFunction(sequence architectureSequence) embeddedFunction {
	return embeddedFunction{
		Name:       sequence.Name,
		Parameters: sequence.Parameters,
		Result:     sequence.Result,
		HasResult:  sequence.HasResult,
	}
}

func findArchitectureSequence(arch Declaration, name string) (architectureSequence, bool) {
	sequences := architectureSequences(arch)
	for i := 0; i < len(sequences); i++ {
		if sequences[i].Name == name {
			return sequences[i], true
		}
	}
	return architectureSequence{}, false
}

func appendArchitectureSequences(out []byte, document Document, arch Declaration,
	nativeEmitter bool, exposeExports bool, excludeExports bool, selected []string) []byte {
	sequences := architectureSequences(arch)
	exports := architectureExports(arch)
	names := embeddedGoNames(document)
	prefix := "rtg" + exportedName(document.Unit)
	for i := 0; i < len(sequences); i++ {
		sequence := sequences[i]
		if selected != nil && stringIndex(selected, sequence.Name) < 0 {
			continue
		}
		external, exported := embeddedExternalName(exports, sequence.Name)
		if excludeExports && exported {
			continue
		}
		out = append(out, "\n// Generated from bounded sequence "...)
		out = append(out, sequence.Name...)
		out = append(out, ".\nfunc "...)
		if exposeExports && exported {
			out = append(out, external...)
		} else {
			out = append(out, prefix...)
			out = append(out, exportedName(sequence.Name)...)
		}
		out = append(out, '(')
		for parameter := 0; parameter < len(sequence.Parameters); parameter++ {
			if parameter != 0 {
				out = append(out, ", "...)
			}
			if nativeEmitter {
				out = appendRewrittenGoMode(out, sequence.Parameters[parameter].Source,
					names, prefix, true, &document)
			} else {
				out = append(out, sequence.Parameters[parameter].Source...)
			}
		}
		out = append(out, ')')
		if sequence.HasResult {
			out = append(out, ' ')
			if nativeEmitter {
				out = appendRewrittenGoMode(out, sequence.Result, names, prefix, true, &document)
			} else {
				out = append(out, sequence.Result...)
			}
		}
		out = append(out, " {\n"...)
		for step := 0; step < len(sequence.Steps); step++ {
			tokens := sequence.Steps[step].Tokens
			if len(tokens) == 0 {
				continue
			}
			out = append(out, '\t')
			if tokens[0] == "call" {
				out = appendSequenceTokens(out, tokens[1:], document, names, prefix,
					nativeEmitter, exports)
			} else if tokens[0] == "let" {
				out = appendSequenceAssignment(out, tokens[1:], ":=", document, names,
					prefix, nativeEmitter, exports)
			} else if tokens[0] == "set" {
				out = appendSequenceAssignment(out, tokens[1:], "=", document, names,
					prefix, nativeEmitter, exports)
			} else if tokens[0] == "var" {
				out = append(out, "var "...)
				out = appendSequenceTokens(out, tokens[1:], document, names, prefix,
					nativeEmitter, exports)
			} else if tokens[0] == "increment" || tokens[0] == "decrement" {
				out = appendSequenceTokens(out, tokens[1:], document, names, prefix,
					nativeEmitter, exports)
				if tokens[0] == "increment" {
					out = append(out, "++"...)
				} else {
					out = append(out, "--"...)
				}
			} else if tokens[0] == "return" {
				out = append(out, "return"...)
				if len(tokens) > 1 {
					out = append(out, ' ')
					out = appendSequenceTokens(out, tokens[1:], document, names, prefix,
						nativeEmitter, exports)
				}
			}
			out = append(out, '\n')
		}
		out = append(out, "}\n"...)
	}
	return out
}

func declarationSequenceRoots(declaration Declaration) []string {
	var roots []string
	var visit func([]Statement)
	visit = func(statements []Statement) {
		for i := 0; i < len(statements); i++ {
			tokens := statements[i].Tokens
			for j := 0; j+1 < len(tokens); j++ {
				if tokens[j] == "sequence" && stringIndex(roots, tokens[j+1]) < 0 {
					roots = append(roots, tokens[j+1])
				}
			}
			visit(statements[i].Children)
		}
	}
	visit(declaration.Statements)
	sortStrings(roots)
	return roots
}

func architectureSequenceRoots(arch Declaration, exportsOnly bool, excludeExports bool) []string {
	var roots []string
	for i := 0; i < len(arch.Statements); i++ {
		isExports := statementBlockName(arch.Statements[i]) == "exports"
		if exportsOnly && !isExports || excludeExports && isExports {
			continue
		}
		part := Declaration{Statements: []Statement{arch.Statements[i]}}
		roots = append(roots, declarationSequenceRoots(part)...)
	}
	sortStrings(roots)
	return roots
}

func targetSequenceRoots(document Document, target ResolvedTarget) []string {
	roots := architectureSequenceRoots(target.Arch, false, false)
	roots = append(roots, declarationSequenceRoots(target.ABI)...)
	abi := target.ABI
	for {
		internal, ok := fieldValue(document, abi, "internal")
		if !ok {
			break
		}
		base, ok := document.Declaration(DeclABI, valueName(internal))
		if !ok {
			break
		}
		roots = append(roots, declarationSequenceRoots(base)...)
		abi = base
	}
	roots = append(roots, declarationSequenceRoots(target.Runtime)...)
	roots = append(roots, declarationSequenceRoots(target.Executable)...)
	roots = append(roots, declarationSequenceRoots(target.Object)...)
	roots = append(roots, declarationSequenceRoots(target.Declaration)...)
	sortStrings(roots)
	return roots
}

func resolveArchitectureSequenceProjection(document Document, arch Declaration,
	goRoots []string, sequenceRoots []string) ([]string, []string) {
	sequences := architectureSequences(arch)
	sequenceNames := make([]string, 0, len(sequences))
	for i := 0; i < len(sequences); i++ {
		sequenceNames = append(sequenceNames, sequences[i].Name)
	}
	selected := append([]string(nil), sequenceRoots...)
	goSelected := append([]string(nil), goRoots...)
	for {
		changed := false
		body := reachableEmbeddedGo(document, goSelected)
		tokens, _ := scan(body, "")
		for i := 0; i < len(tokens); i++ {
			name := tokenText(body, tokens[i])
			if stringIndex(sequenceNames, name) >= 0 && stringIndex(selected, name) < 0 {
				selected = append(selected, name)
				changed = true
			}
		}
		for i := 0; i < len(sequences); i++ {
			if stringIndex(selected, sequences[i].Name) < 0 {
				continue
			}
			for step := 0; step < len(sequences[i].Steps); step++ {
				tokens := sequences[i].Steps[step].Tokens
				for token := 0; token < len(tokens); token++ {
					name := tokens[token]
					if stringIndex(sequenceNames, name) >= 0 &&
						stringIndex(selected, name) < 0 {
						selected = append(selected, name)
						changed = true
					}
					if _, found := findEmbeddedFunction(document, name); found &&
						stringIndex(goSelected, name) < 0 {
						goSelected = append(goSelected, name)
						changed = true
					}
				}
			}
		}
		if !changed {
			break
		}
		sortStrings(selected)
		sortStrings(goSelected)
	}
	sortStrings(selected)
	sortStrings(goSelected)
	return goSelected, selected
}

func appendSequenceAssignment(out []byte, tokens []string, operator string,
	document Document, names []string, prefix string, nativeEmitter bool,
	exports []embeddedExport) []byte {
	equals := -1
	for i := 0; i < len(tokens); i++ {
		if tokens[i] == "=" {
			equals = i
			break
		}
	}
	if equals < 0 {
		return out
	}
	out = appendSequenceTokens(out, tokens[:equals], document, names, prefix,
		nativeEmitter, exports)
	out = append(out, operator...)
	return appendSequenceTokens(out, tokens[equals+1:], document, names, prefix,
		nativeEmitter, exports)
}

func appendSequenceTokens(out []byte, tokens []string, document Document, names []string,
	prefix string, nativeEmitter bool, exports []embeddedExport) []byte {
	var compact []byte
	translated := translateSequenceRecordTokens(tokens)
	for i := 0; i < len(translated); i++ {
		compact = append(compact, translated[i]...)
	}
	return appendRewrittenGoModeExports(out, compact, names, prefix, nativeEmitter, &document, exports)
}

func translateSequenceRecordTokens(tokens []string) []string {
	out := append([]string(nil), tokens...)
	var recordDepths []int
	depth := 0
	for i := 0; i < len(out); i++ {
		record := ""
		if out[i] == "record_register" {
			record = "RTGRegister"
		} else if out[i] == "record_address" {
			record = "RTGAddress"
		} else if out[i] == "record_condition" {
			record = "RTGCondition"
		} else if out[i] == "record_ints" {
			record = "[]int"
		} else if out[i] == "record_bytes" {
			record = "[]byte"
		}
		if record != "" && i+1 < len(out) && out[i+1] == "(" {
			out[i] = record
			out[i+1] = "{"
			depth++
			recordDepths = append(recordDepths, depth)
			i++
			continue
		}
		if out[i] == "(" {
			depth++
		} else if out[i] == ")" {
			if len(recordDepths) != 0 && recordDepths[len(recordDepths)-1] == depth {
				out[i] = "}"
				recordDepths = recordDepths[:len(recordDepths)-1]
			}
			depth--
		}
	}
	return out
}

func appendStatefulArchitectureSequences(out []byte, arch Declaration) []byte {
	sequences := architectureSequences(arch)
	for i := 0; i < len(sequences); i++ {
		sequence := sequences[i]
		out = append(out, "\nfunc "...)
		out = append(out, sequence.Name...)
		out = append(out, '(')
		for parameter := 0; parameter < len(sequence.Parameters); parameter++ {
			if parameter != 0 {
				out = append(out, ", "...)
			}
			out = append(out, sequence.Parameters[parameter].Source...)
		}
		out = append(out, ')')
		if sequence.HasResult {
			out = append(out, ' ')
			out = append(out, sequence.Result...)
		}
		out = append(out, " {\n"...)
		for step := 0; step < len(sequence.Steps); step++ {
			tokens := sequence.Steps[step].Tokens
			if len(tokens) == 0 {
				continue
			}
			out = append(out, '\t')
			if tokens[0] == "call" {
				translated := translateSequenceRecordTokens(tokens[1:])
				for token := 0; token < len(translated); token++ {
					out = append(out, translated[token]...)
				}
			} else if tokens[0] == "let" || tokens[0] == "set" {
				operator := "="
				if tokens[0] == "let" {
					operator = ":="
				}
				translated := translateSequenceRecordTokens(tokens[1:])
				for token := 0; token < len(translated); token++ {
					if translated[token] == "=" {
						out = append(out, operator...)
					} else {
						out = append(out, translated[token]...)
					}
				}
			} else if tokens[0] == "var" {
				out = append(out, "var "...)
				for token := 1; token < len(tokens); token++ {
					out = append(out, tokens[token]...)
				}
			} else if tokens[0] == "increment" || tokens[0] == "decrement" {
				for token := 1; token < len(tokens); token++ {
					out = append(out, tokens[token]...)
				}
				if tokens[0] == "increment" {
					out = append(out, "++"...)
				} else {
					out = append(out, "--"...)
				}
			} else if tokens[0] == "return" {
				out = append(out, "return"...)
				if len(tokens) > 1 {
					out = append(out, ' ')
				}
				translated := translateSequenceRecordTokens(tokens[1:])
				for token := 0; token < len(translated); token++ {
					out = append(out, translated[token]...)
				}
			}
			out = append(out, '\n')
		}
		out = append(out, "}\n"...)
	}
	return out
}

func sequenceGoRoots(arch Declaration, document Document) []string {
	var roots []string
	goNames := embeddedGoNames(document)
	sequences := architectureSequences(arch)
	for i := 0; i < len(sequences); i++ {
		for j := 0; j < len(sequences[i].Steps); j++ {
			tokens := sequences[i].Steps[j].Tokens
			for k := 0; k < len(tokens); k++ {
				if stringIndex(goNames, tokens[k]) >= 0 && stringIndex(roots, tokens[k]) < 0 {
					roots = append(roots, tokens[k])
				}
			}
		}
	}
	sortStrings(roots)
	return roots
}
