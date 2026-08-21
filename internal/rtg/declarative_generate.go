package rtg

import "renvo.dev/internal/syntax"

type embeddedFunction struct {
	Name       string
	Signature  []byte
	Parameters []embeddedParameter
	Result     []byte
	HasResult  bool
}

type embeddedParameter struct {
	Name   string
	Source []byte
}

type architectureForm struct {
	Name      string
	Algorithm string
	Kind      string
	Operands  []architectureFormOperand
}

type architectureFormOperand struct {
	Name       string
	Kind       string
	Shifts     []string
	Relocation bool
}

// appendArchitectureBindings specializes declarative instruction rows into
// ordinary direct Go wrappers. Neither the table nor an algorithm registry is
// retained in generated code.
func appendArchitectureBindings(out []byte, document Document, arch Declaration, rewriteAlgorithms bool, nativeEmitter bool) []byte {
	instructions, ok := declarationBlock(arch, "instructions")
	if !ok {
		return out
	}
	names := embeddedGoNames(document)
	prefix := "rtg" + exportedName(document.Unit)
	if !rewriteAlgorithms {
		prefix = architectureLocalPrefix(arch.Name)
	}
	forms := architectureForms(arch)
	for i := 0; i < len(instructions.Children); i++ {
		left, right, assignment := statementAssignment(instructions.Children[i])
		if assignment && len(left) == 1 && len(right) == 2 &&
			(right[0] == "go" || right[0] == "sequence") {
			function, found := findBackendFunction(document, right[1])
			if !found {
				continue
			}
			out = appendDirectInstructionWrapper(out, left[0], function, names, prefix, rewriteAlgorithms, nativeEmitter)
			continue
		}
		row := instructions.Children[i].Tokens
		if len(row) < 2 {
			continue
		}
		form, found := lookupArchitectureForm(forms, row[1])
		if !found {
			continue
		}
		if form.Kind == "bytes" {
			out = appendFixedInstructionWrapper(out, row, prefix, nativeEmitter)
			continue
		}
		if bits := typedWordFormBits(form.Kind); bits != 0 {
			bigEndian := false
			if endian, found := fieldValue(document, arch, "endian"); found {
				bigEndian = valueName(endian) == "big"
			}
			out = appendTypedWordInstructionWrapper(
				out, row, form, prefix, nativeEmitter, bits, bigEndian)
			continue
		}
		function, found := findBackendFunction(document, form.Algorithm)
		if !found {
			continue
		}
		out = appendFormInstructionWrapper(out, row, function, names, prefix, rewriteAlgorithms, nativeEmitter)
	}
	return out
}

func appendTypedWordInstructionWrapper(out []byte, row []string, form architectureForm,
	prefix string, nativeEmitter bool, bits int, bigEndian bool) []byte {
	if len(row) < 3 {
		return out
	}
	out = append(out, "\n// Generated from declarative "...)
	out = append(out, form.Kind...)
	out = append(out, " instruction "...)
	out = append(out, row[0]...)
	out = append(out, ".\nfunc "...)
	out = append(out, prefix...)
	out = append(out, instructionGoSuffix(row[0])...)
	out = append(out, "(out "...)
	if nativeEmitter {
		out = append(out, "*renvoAsm"...)
	} else {
		out = append(out, "*RTGEmitter"...)
	}
	for i := 0; i < len(form.Operands); i++ {
		out = append(out, ", "...)
		out = append(out, form.Operands[i].Name...)
		out = append(out, ' ')
		out = appendArchitectureFormOperandType(out, form.Operands[i].Kind, nativeEmitter)
	}
	out = append(out, ") {\n\t"...)
	if bigEndian {
		out = append(out, "value := "...)
	} else {
		out = appendTypedWordEmitter(out, nativeEmitter, bits)
	}
	out = appendTypedWordValue(out, row, form)
	if bigEndian {
		out = append(out, '\n')
		out = appendBigEndianWord(out, nativeEmitter, bits)
	} else {
		out = append(out, ")\n"...)
	}
	for i := 0; i < len(form.Operands); i++ {
		if !form.Operands[i].Relocation {
			continue
		}
		out = append(out, '\t')
		if nativeEmitter {
			out = append(out, "renvoRTGReloc(out, "...)
		} else {
			out = append(out, "RTGReloc(out, "...)
		}
		out = append(out, form.Operands[i].Name...)
		out = append(out, ")\n"...)
	}
	return append(out, "}\n"...)
}

func appendTypedWordValue(out []byte, row []string, form architectureForm) []byte {
	out = append(out, row[2]...)
	for i := 0; i < len(form.Operands); i++ {
		operand := form.Operands[i]
		for j := 0; j < len(operand.Shifts); j++ {
			out = append(out, "|("...)
			out = append(out, operand.Name...)
			if operand.Kind == "reg" || operand.Kind == "condition" {
				out = append(out, ".Code"...)
			} else if operand.Kind == "address_base" {
				out = append(out, ".Base.Code"...)
			}
			if operand.Shifts[j] != "0" {
				out = append(out, "<<"...)
				out = append(out, operand.Shifts[j]...)
			}
			out = append(out, ')')
		}
	}
	return out
}

func appendTypedWordEmitter(out []byte, nativeEmitter bool, bits int) []byte {
	if bits == 32 {
		if nativeEmitter {
			return append(out, "renvoRTGUint32(out, "...)
		}
		return append(out, "RTGUint32(out, "...)
	}
	if bits == 16 {
		if nativeEmitter {
			return append(out, "renvoAsmEmit16(out, "...)
		}
		return append(out, "out.Uint16("...)
	}
	return out
}

func appendBigEndianWord(out []byte, nativeEmitter bool, bits int) []byte {
	for shift := bits - 8; shift >= 0; shift -= 8 {
		out = append(out, '\t')
		if nativeEmitter {
			out = append(out, "renvoRTGByte(out, byte(value>>"...)
		} else {
			out = append(out, "RTGByte(out, byte(value>>"...)
		}
		out = append(out, decimalText(shift)...)
		out = append(out, "))\n"...)
	}
	return out
}

func appendArchitectureFormOperandType(out []byte, kind string, nativeEmitter bool) []byte {
	if kind == "reg" {
		return append(out, "RTGRegister"...)
	}
	if kind == "condition" {
		return append(out, "RTGCondition"...)
	}
	if kind == "label" {
		if nativeEmitter {
			return append(out, "int"...)
		}
		return append(out, "RTGLabel"...)
	}
	if kind == "address_base" {
		if nativeEmitter {
			return append(out, "renvoRTGAddress"...)
		}
		return append(out, "RTGAddress"...)
	}
	if kind == "byte" {
		return append(out, "byte"...)
	}
	return append(out, "int"...)
}

func appendFixedInstructionWrapper(out []byte, row []string, prefix string, nativeEmitter bool) []byte {
	values := instructionByteValues(row)
	if len(values) == 0 {
		return out
	}
	out = append(out, "\n// Generated from instruction "...)
	out = append(out, row[0]...)
	out = append(out, ".\nfunc "...)
	out = append(out, prefix...)
	out = append(out, instructionGoSuffix(row[0])...)
	if nativeEmitter {
		out = append(out, "(out *renvoAsm) {\n"...)
	} else {
		out = append(out, "(out *RTGEmitter) {\n"...)
	}
	if len(values) == 4 {
		if nativeEmitter {
			out = append(out, "\trenvoRTGUint32(out, "...)
		} else {
			out = append(out, "\tRTGUint32(out, "...)
		}
		out = appendPackedBytes(out, values)
		out = append(out, ")\n"...)
	} else if len(values) == 8 {
		if nativeEmitter {
			out = append(out, "\trenvoRTGUint64(out, "...)
		} else {
			out = append(out, "\tRTGUint64(out, "...)
		}
		out = appendPackedBytes(out, values)
		out = append(out, ")\n"...)
	} else {
		for i := 0; i < len(values); i++ {
			if nativeEmitter {
				out = append(out, "\trenvoRTGByte(out, "...)
			} else {
				out = append(out, "\tRTGByte(out, "...)
			}
			out = append(out, values[i]...)
			out = append(out, ")\n"...)
		}
	}
	return append(out, "}\n"...)
}

func instructionByteValues(row []string) []string {
	var values []string
	inList := false
	for i := 2; i < len(row); i++ {
		if row[i] == "[" {
			inList = true
			continue
		}
		if row[i] == "]" {
			break
		}
		if inList && row[i] != "," {
			values = append(values, row[i])
		}
	}
	return values
}

func appendPackedBytes(out []byte, values []string) []byte {
	for i := 0; i < len(values); i++ {
		if i != 0 {
			out = append(out, '|')
		}
		out = append(out, values[i]...)
		if i != 0 {
			out = append(out, "<<"...)
			out = appendDecimalFrame(out, i*8)
		}
	}
	return out
}

func appendDirectInstructionWrapper(
	out []byte,
	instruction string,
	function embeddedFunction,
	names []string,
	prefix string,
	rewriteAlgorithms bool,
	nativeEmitter bool,
) []byte {
	out = append(out, "\n// Generated from instruction "...)
	out = append(out, instruction...)
	out = append(out, ".\nfunc "...)
	out = append(out, prefix...)
	out = append(out, instructionGoSuffix(instruction)...)
	if rewriteAlgorithms {
		out = appendRewrittenGoMode(out, function.Signature, names, prefix, nativeEmitter, nil)
	} else {
		out = append(out, function.Signature...)
	}
	out = append(out, " {\n"...)
	out = appendWrapperCall(out, function, nil, prefix, rewriteAlgorithms)
	return append(out, "}\n"...)
}

func appendFormInstructionWrapper(
	out []byte,
	row []string,
	function embeddedFunction,
	names []string,
	prefix string,
	rewriteAlgorithms bool,
	nativeEmitter bool,
) []byte {
	constants := instructionConstants(row)
	out = append(out, "\n// Generated from instruction "...)
	out = append(out, row[0]...)
	out = append(out, ".\nfunc "...)
	out = append(out, prefix...)
	out = append(out, instructionGoSuffix(row[0])...)
	out = append(out, '(')
	first := true
	for i := 0; i < len(function.Parameters); i++ {
		if instructionConstant(constants, function.Parameters[i].Name) != "" {
			continue
		}
		if !first {
			out = append(out, ", "...)
		}
		first = false
		if rewriteAlgorithms {
			out = appendRewrittenGoMode(out, function.Parameters[i].Source, names, prefix, nativeEmitter, nil)
		} else {
			out = append(out, function.Parameters[i].Source...)
		}
	}
	out = append(out, ')')
	if len(function.Result) != 0 {
		out = append(out, ' ')
		if rewriteAlgorithms {
			out = appendRewrittenGoMode(out, function.Result, names, prefix, nativeEmitter, nil)
		} else {
			out = append(out, function.Result...)
		}
	}
	out = append(out, " {\n"...)
	out = appendWrapperCall(out, function, constants, prefix, rewriteAlgorithms)
	return append(out, "}\n"...)
}

func appendWrapperCall(
	out []byte,
	function embeddedFunction,
	constants []string,
	prefix string,
	rewriteAlgorithms bool,
) []byte {
	if function.HasResult {
		out = append(out, "\tresult := "...)
	} else {
		out = append(out, '\t')
	}
	if rewriteAlgorithms {
		out = append(out, prefix...)
		out = append(out, exportedName(function.Name)...)
	} else {
		out = append(out, function.Name...)
	}
	out = append(out, '(')
	for i := 0; i < len(function.Parameters); i++ {
		if i != 0 {
			out = append(out, ", "...)
		}
		value := instructionConstant(constants, function.Parameters[i].Name)
		if value == "" {
			value = function.Parameters[i].Name
		}
		out = append(out, value...)
	}
	out = append(out, ")\n"...)
	if function.HasResult {
		out = append(out, "\treturn result\n"...)
	}
	return out
}

func architectureForms(arch Declaration) []architectureForm {
	block, ok := declarationBlock(arch, "forms")
	if !ok {
		return nil
	}
	var forms []architectureForm
	for i := 0; i < len(block.Children); i++ {
		left, right, assignment := statementAssignment(block.Children[i])
		if assignment && len(left) == 1 && len(right) == 2 &&
			(right[0] == "go" || right[0] == "sequence") {
			forms = append(forms, architectureForm{Name: left[0], Algorithm: right[1], Kind: right[0]})
		} else if assignment && len(left) == 1 && len(right) == 1 && right[0] == "bytes" {
			forms = append(forms, architectureForm{Name: left[0], Kind: "bytes"})
		} else if assignment && len(left) == 1 && len(right) >= 3 &&
			typedWordFormBits(right[0]) != 0 {
			form := architectureForm{Name: left[0], Kind: right[0]}
			form.Operands = decodeArchitectureFormOperands(right[1:])
			forms = append(forms, form)
		}
	}
	return forms
}

func decodeArchitectureFormOperands(tokens []string) []architectureFormOperand {
	var operands []architectureFormOperand
	for at := 0; at < len(tokens); {
		if tokens[at] == "(" || tokens[at] == ")" || tokens[at] == "," {
			at++
			continue
		}
		if at+3 >= len(tokens) || tokens[at+1] != ":" {
			break
		}
		operand := architectureFormOperand{Name: tokens[at], Kind: tokens[at+2]}
		at += 3
		if at >= len(tokens) || tokens[at] != "@" {
			break
		}
		at++
		for at < len(tokens) {
			if tokens[at] == "reloc" {
				operand.Relocation = true
				at++
			} else {
				operand.Shifts = append(operand.Shifts, tokens[at])
				at++
			}
			if at >= len(tokens) || tokens[at] != "+" {
				break
			}
			at++
		}
		operands = append(operands, operand)
	}
	return operands
}

func architectureFormFunction(form architectureForm, name string) (embeddedFunction, bool) {
	if typedWordFormBits(form.Kind) == 0 {
		return embeddedFunction{}, false
	}
	function := embeddedFunction{
		Name: name,
		Parameters: []embeddedParameter{{
			Name: "out", Source: []byte("out *RTGEmitter"),
		}},
	}
	for i := 0; i < len(form.Operands); i++ {
		operand := form.Operands[i]
		source := append([]byte(operand.Name+" "), architectureFormOperandType(operand.Kind)...)
		function.Parameters = append(function.Parameters, embeddedParameter{
			Name: operand.Name, Source: source,
		})
	}
	return function, true
}

func architectureFormOperandType(kind string) string {
	if kind == "reg" {
		return "RTGRegister"
	}
	if kind == "condition" {
		return "RTGCondition"
	}
	if kind == "label" {
		return "RTGLabel"
	}
	if kind == "address_base" {
		return "RTGAddress"
	}
	if kind == "byte" {
		return "byte"
	}
	return "int"
}

func typedWordFormBits(kind string) int {
	if kind == "word16" {
		return 16
	}
	if kind == "word32" {
		return 32
	}
	return 0
}

func lookupArchitectureForm(forms []architectureForm, name string) (architectureForm, bool) {
	for i := 0; i < len(forms); i++ {
		if forms[i].Name == name {
			return forms[i], true
		}
	}
	return architectureForm{}, false
}

// instructionConstants returns alternating name/value entries.
func instructionConstants(row []string) []string {
	var constants []string
	if len(row) > 2 && row[2] != "[" {
		constants = append(constants, "opcode", row[2])
	}
	for i := 2; i < len(row); i++ {
		if row[i] == "/" && i+1 < len(row) {
			constants = append(constants, "selector", row[i+1])
			i++
			continue
		}
		if i+2 < len(row) && row[i+1] == "=" {
			constants = append(constants, row[i], row[i+2])
			i += 2
		}
	}
	if instructionConstant(constants, "width") == "" {
		if width := instructionNameWidth(row[0]); width != "" {
			constants = append(constants, "width", width)
		}
	}
	return constants
}

func instructionConstant(constants []string, name string) string {
	for i := 0; i+1 < len(constants); i += 2 {
		if constants[i] == name {
			return constants[i+1]
		}
	}
	return ""
}

func instructionNameWidth(name string) string {
	digits := ""
	for i := len(name) - 1; i >= 0 && name[i] >= '0' && name[i] <= '9'; i-- {
		digits = name[i:]
	}
	if digits == "8" || digits == "16" || digits == "32" || digits == "64" {
		return digits
	}
	return ""
}

func declarationBlock(declaration Declaration, name string) (Statement, bool) {
	for i := 0; i < len(declaration.Statements); i++ {
		if statementBlockName(declaration.Statements[i]) == name {
			return declaration.Statements[i], true
		}
	}
	return Statement{}, false
}

func findEmbeddedFunction(document Document, name string) (embeddedFunction, bool) {
	for i := 0; i < len(document.Declarations); i++ {
		declaration := document.Declarations[i]
		if declaration.Kind != DeclGo {
			continue
		}
		wrapped := make([]byte, 0, len(declaration.GoSource)+16)
		wrapped = append(wrapped, "package backend\n"...)
		wrapped = append(wrapped, declaration.GoSource...)
		file := syntax.ParseFile(wrapped)
		if !file.Ok {
			continue
		}
		for j := 0; j < len(file.Funcs); j++ {
			fn := file.Funcs[j]
			fnName := string(syntax.TokenText(wrapped, file.Tokens[fn.NameTok]))
			if fnName != name {
				continue
			}
			start := syntax.TokenStart(file.Tokens[fn.ParamsStart])
			end := syntax.TokenStart(file.Tokens[fn.ResultEnd])
			signature := make([]byte, end-start)
			copy(signature, wrapped[start:end])
			for len(signature) > 0 && (signature[len(signature)-1] == ' ' ||
				signature[len(signature)-1] == '\t' || signature[len(signature)-1] == '\n' ||
				signature[len(signature)-1] == '\r') {
				signature = signature[:len(signature)-1]
			}
			return embeddedFunction{
				Name:       name,
				Signature:  signature,
				Parameters: functionParameters(file, fn),
				Result:     functionResult(file, fn),
				HasResult:  fn.ResultStart != fn.ResultEnd,
			}, true
		}
	}
	return embeddedFunction{}, false
}

func findBackendFunction(document Document, name string) (embeddedFunction, bool) {
	if function, found := findEmbeddedFunction(document, name); found {
		return function, true
	}
	for i := 0; i < len(document.Declarations); i++ {
		if document.Declarations[i].Kind != DeclArch {
			continue
		}
		if sequence, found := findArchitectureSequence(document.Declarations[i], name); found {
			return architectureSequenceFunction(sequence), true
		}
	}
	return embeddedFunction{}, false
}

func functionParameters(file syntax.File, fn syntax.FuncDecl) []embeddedParameter {
	var parameters []embeddedParameter
	start := fn.ParamsStart + 1
	depth := 0
	for i := start; i < fn.ParamsEnd-1; i++ {
		text := string(syntax.TokenText(file.Src, file.Tokens[i]))
		if text == "(" || text == "[" {
			depth++
		} else if text == ")" || text == "]" {
			depth--
		}
		if text == "," && depth == 0 {
			if start < i {
				parameters = append(parameters, embeddedFunctionParameter(file, start, i))
			}
			start = i + 1
		}
	}
	if start < fn.ParamsEnd-1 {
		parameters = append(parameters, embeddedFunctionParameter(file, start, fn.ParamsEnd-1))
	}
	return parameters
}

func embeddedFunctionParameter(file syntax.File, start int, end int) embeddedParameter {
	sourceStart := syntax.TokenStart(file.Tokens[start])
	sourceEnd := syntax.TokenEnd(file.Tokens[end-1])
	source := make([]byte, sourceEnd-sourceStart)
	copy(source, file.Src[sourceStart:sourceEnd])
	return embeddedParameter{
		Name:   string(syntax.TokenText(file.Src, file.Tokens[start])),
		Source: source,
	}
}

func functionResult(file syntax.File, fn syntax.FuncDecl) []byte {
	if fn.ResultStart == fn.ResultEnd {
		return nil
	}
	start := syntax.TokenStart(file.Tokens[fn.ResultStart])
	end := syntax.TokenStart(file.Tokens[fn.ResultEnd])
	for end > start && (file.Src[end-1] == ' ' || file.Src[end-1] == '\t' ||
		file.Src[end-1] == '\n' || file.Src[end-1] == '\r') {
		end--
	}
	result := make([]byte, end-start)
	copy(result, file.Src[start:end])
	return result
}
