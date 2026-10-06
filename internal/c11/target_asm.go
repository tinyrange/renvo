package c11

func (t *translator) emitTargetAsm(operation cAsm) bool {
	fail := func() bool { t.ok = false; t.err = TranslateErrUnsupported; t.errorAt = operation.offset; return false }
	compiler := t.asmCompiler
	if compiler == nil || compiler.WordBits() == 0 {
		return fail()
	}
	name := "__c_target_asm_" + t.asmNamespace + "_" + decimalString(len(t.asmSources))
	outputs := []AssemblyOperand{}
	inputs := []AssemblyOperand{}
	operands := append(append([]cAsmOperand(nil), operation.outputs...), operation.inputs...)
	for i, operand := range operands {
		value := AssemblyOperand{Name: operand.name, Constraint: operand.constraint}
		if i < len(operation.outputs) {
			typ := t.typeInfo(t.lvalueType(operand.expression))
			value.Bits, value.Signed = typ.size*8, typ.kind == cTypeInt
			if typ.kind != cTypeInt && typ.kind != cTypeUint && typ.kind != cTypePointer || typ.size <= 0 || typ.size > compiler.WordBits()/8 || typ.qualifiers&cQualifierConst != 0 {
				return fail()
			}
			outputs = append(outputs, value)
		} else {
			typ := t.typeInfo(t.expressionType(operand.expression))
			value.Bits, value.Signed = typ.size*8, typ.kind == cTypeInt
			if operand.constraint == "i" || operand.constraint == "n" {
				constant, ok := t.constantExpression(operand.expression)
				if !ok {
					return fail()
				}
				value.Constant = true
				value.Value = decimalString(constant)
			} else if typ.kind != cTypeInt && typ.kind != cTypeUint && typ.kind != cTypePointer || typ.size <= 0 || typ.size > compiler.WordBits()/8 {
				return fail()
			}
			inputs = append(inputs, value)
		}
	}
	lowered := compiler.CompileInline(AssemblyRequest{Name: name, Template: operation.template, Outputs: outputs, Inputs: inputs, Clobbers: operation.clobbers, Labels: operation.labels, Unique: len(t.asmSources)})
	if !lowered.Ok {
		t.asmError = lowered.Message
		return fail()
	}
	// All expression evaluation stays in ordinary frontend code. Only their
	// runtime words enter the managed body; assembly never splices C source.
	t.asmSources = append(t.asmSources, lowered.Source)
	declaration := "func " + name + "("
	for i := range lowered.Words {
		if i != 0 {
			declaration += ","
		}
		declaration += "word" + decimalString(i) + " uintptr"
	}
	declaration += ")"
	if operation.gotoAsm {
		declaration += " uintptr"
	}
	t.staticOut = append(t.staticOut, []byte(declaration+"\n")...)
	t.appendText("{")
	if operation.gotoAsm {
		t.appendText("__c_asm_exit:=")
	}
	t.appendText(name + "(")
	for i, word := range lowered.Words {
		if i != 0 {
			t.appendText(",")
		}
		operand := operands[word.Operand]
		if word.Address {
			if t.lvalueType(operand.expression) == cTypeVoidID {
				return fail()
			}
			t.appendText("uintptr(__c_unsafe.Pointer(&(")
			t.emitExpression(operand.expression)
			t.appendText(")))")
			t.usesUnsafe = true
		} else if t.typeInfo(t.expressionType(operand.expression)).kind == cTypePointer {
			t.appendText("uintptr(__c_unsafe.Pointer(")
			t.emitExpression(operand.expression)
			t.appendText("))")
			t.usesUnsafe = true
		} else {
			t.appendText("uintptr(")
			t.emitExpression(operand.expression)
			t.appendText(")")
		}
	}
	t.appendText(");")
	for i, label := range operation.labels {
		t.appendText("if __c_asm_exit==" + decimalString(i+1) + " {goto " + t.localLabelGoName(label) + "};")
	}
	t.appendText("};")
	return t.ok
}
