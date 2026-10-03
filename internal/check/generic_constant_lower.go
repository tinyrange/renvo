package check

func (c *genericExpressionContext) constantNameShadowed(text string, before int) bool {
	e := c.specializer.environment
	file := &e.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	shadowed := e.localType(c.scope, text, before) >= 0 || e.packageValueName(c.scope.pkg, text) || e.imported(c.scope, text) >= 0
	if declaration := e.lookupInScope(c.scope, text); declaration >= 0 && e.decls[declaration].kind == SymbolFunc {
		shadowed = true
	}
	for _, parameter := range c.scope.parameters {
		if e.types.get(parameter).name == text {
			shadowed = true
		}
	}
	for _, binding := range c.bindings {
		if binding.name >= 0 && binding.visible <= before && before < binding.end && tokenStringEquals(file, binding.name, text) {
			shadowed = true
		}
	}
	return shadowed
}

func (c *genericExpressionContext) constantTypeText(typ int, before int) string {
	return c.scopedTypeText(typ, before)
}

// Emit the exact checked value before target-width arithmetic reaches the
// backend. Keep the constant's type category, including untyped rune values.
func (c *genericExpressionContext) constantUsesVariable(start int, end int) bool {
	// A constant len/cap can still be the syntactic use of a local array.
	// Leave those expressions intact so lowering does not make it unused.
	for token := start; token < end; token++ {
		if binding := c.binding(token, start); binding >= 0 && c.bindings[binding].writable {
			return true
		}
	}
	return false
}

func genericIntegerHex(integer wideConstant) string {
	if !integer.ok {
		return ""
	}
	text := ""
	for bit := ((len(integer.words)*15+3)/4 - 1) * 4; bit >= 0; bit -= 4 {
		word, shift := bit/15, bit%15
		digit := integer.words[word] >> shift
		if shift > 11 && word+1 < len(integer.words) {
			digit |= integer.words[word+1] << (15 - shift)
		}
		digit &= 15
		if digit != 0 || text != "" {
			text += string("0123456789abcdef"[digit])
		}
	}
	if text == "" {
		text = "0"
	}
	text = "0x" + text
	if integer.negative {
		text = "-" + text
	}
	return text
}

// Rounded binary floats have a power-of-two denominator. Emit their exact
// dyadic value as one literal so the backend cannot re-evaluate the arithmetic.
func genericRationalFloatText(value genericRational) string {
	if !value.numerator.ok || !value.denominator.ok {
		return ""
	}
	if len(value.numerator.words) == 0 {
		return "0x0p0"
	}
	power := -1
	for i, word := range value.denominator.words {
		if word == 0 {
			continue
		}
		if power >= 0 || word&(word-1) != 0 {
			return ""
		}
		power = i * 15
		for word > 1 {
			power++
			word >>= 1
		}
	}
	if power < 0 {
		return ""
	}
	return genericIntegerHex(value.numerator) + "p-" + genericDecimal(power)
}

func (c *genericExpressionContext) constantReplacementText(value genericArgument) string {
	if value.constant != nil && value.constant.boolean != nil {
		text := "false"
		if *value.constant.boolean {
			text = "true"
		}
		if c.constantNameShadowed(text, value.start) {
			text = "(0!=0)"
			if *value.constant.boolean {
				text = "(0==0)"
			}
		}
		if value.typ != 0 {
			text = c.constantTypeText(value.typ, value.start) + "(" + text + ")"
		}
		return text
	}
	if value.typ != 0 && value.constant != nil {
		e := c.specializer.environment
		name := e.types.get(e.types.underlying(value.typ)).name
		if name == "float32" || name == "float64" || name == "complex64" || name == "complex128" {
			real := genericRationalFloatText(value.constant.real)
			if real == "" {
				return ""
			}
			if name == "complex64" || name == "complex128" {
				imaginary := genericRationalFloatText(value.constant.imaginary)
				if imaginary == "" {
					return ""
				}
				if c.constantNameShadowed("complex", value.start) {
					real = "(" + real + "+" + imaginary + "i)"
				} else {
					real = "complex(" + real + "," + imaginary + ")"
				}
			}
			return c.constantTypeText(value.typ, value.start) + "(" + real + ")"
		}
	}
	integer := genericConstantInteger(value.constant)
	if !integer.ok {
		return ""
	}
	text := genericIntegerHex(integer)
	if value.typ != 0 {
		text = c.constantTypeText(value.typ, value.start) + "(" + text + ")"
	} else if value.untyped == genericUntypedFloat {
		text += "p0"
	} else if value.untyped == genericUntypedComplex {
		text = "(" + text + "+0i)"
	} else if value.untyped == genericUntypedRune {
		text = "('\\x00'+" + text + ")"
	}
	return text
}

func (c *genericExpressionContext) lowerIntegerConstant(start int, end int, value genericArgument) {
	if c.constantUsesVariable(start, end) {
		return
	}
	if text := c.constantReplacementText(value); text != "" {
		c.replace(start, end, text)
	}
}

// The destination supplies the rounding context for an untyped constant.
// Typed constants retain their own type when assigned or boxed in an interface.
func (c *genericExpressionContext) lowerExpectedConstant(value genericArgument, target int) {
	if !c.specialize || value.constant == nil || value.end <= value.start || target == 0 {
		return
	}
	e := c.specializer.environment
	if value.untyped == 0 {
		target = value.typ
	}
	if e.types.get(e.types.underlying(target)).kind == genericInterface {
		target = e.types.defaultType(value.untyped)
	}
	if !e.argumentAssignable(value, target) || c.constantUsesVariable(value.start, value.end) {
		return
	}
	value.constant = e.roundConstant(value.constant, target)
	value.typ, value.untyped = target, 0
	text := c.constantReplacementText(value)
	if text == "" {
		return
	}
	file := &e.graph.Packages[c.scope.pkg].Files[c.scope.file].File
	start, end := int(file.Tokens[value.start].Start), int(file.Tokens[value.end-1].End)
	// Contextual folding takes precedence over an earlier context-free fold
	// of the same source expression. Later scans preserve this replacement.
	for i := range c.changes {
		if c.changes[i].start == start && c.changes[i].end == end {
			c.changes[i].text = text
			return
		}
	}
	c.replace(value.start, value.end, text)
}
