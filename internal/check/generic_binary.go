package check

// Binary operands are checked before evaluating constants. In particular an
// untyped operand must fit its typed peer even if the final result would fit.
func (e *genericEnvironment) binaryOperation(left genericArgument, right genericArgument, op string) (genericArgument, bool) {
	result := genericArgument{}
	if op == "<<" || op == ">>" {
		leftOK := e.allowsOperation(left.typ, op)
		rightOK := e.allowsOperation(right.typ, op)
		if left.untyped != 0 {
			leftOK = genericConstantInteger(left.constant).ok || len(left.shifted) != 0
			if left.constant != nil && right.constant != nil && left.untyped != genericUntypedRune {
				left.untyped = genericUntypedInt
			}
		}
		if right.untyped != 0 {
			rightOK = e.argumentAssignable(right, e.types.basic("uint64"))
		}
		if right.constant != nil && right.constant.real.numerator.negative {
			rightOK = false
		}
		result.typ, result.untyped = left.typ, left.untyped
		result.shifted = left.shifted
		if left.untyped != 0 && right.constant == nil && left.constant != nil {
			result.shifted = append(result.shifted, left.constant)
		}
		result.constant = genericConstantBinary(left.constant, right.constant, op, true)
		return result, leftOK && rightOK && (left.constant == nil || right.constant == nil || result.constant != nil)
	}
	equality := op == "==" || op == "!="
	comparison := equality || op == "<" || op == "<=" || op == ">" || op == ">="
	if left.untyped == genericUntypedNil || right.untyped == genericUntypedNil {
		typ := left.typ
		if typ == 0 {
			typ = right.typ
		}
		return genericArgument{untyped: genericUntypedBool}, equality && typ != 0 && e.allowsOperation(typ, "nil")
	}
	valid := true
	if left.typ == 0 && right.typ == 0 {
		result.untyped, valid = genericMergeUntyped(left.untyped, right.untyped)
		if len(left.shifted) != 0 || len(right.shifted) != 0 {
			result.shifted = append(append([]*genericConstant(nil), left.shifted...), right.shifted...)
			if left.constant != nil {
				result.shifted = append(result.shifted, left.constant)
			}
			if right.constant != nil {
				result.shifted = append(result.shifted, right.constant)
			}
		}
	} else {
		result.typ = left.typ
		if result.typ == 0 {
			result.typ = right.typ
		}
		if left.untyped != 0 {
			valid = e.argumentAssignable(left, result.typ)
			left.constant = e.roundConstant(left.constant, result.typ)
		} else if right.untyped != 0 {
			valid = e.argumentAssignable(right, result.typ)
			right.constant = e.roundConstant(right.constant, result.typ)
		} else if comparison {
			valid = e.argumentAssignable(left, right.typ) || e.argumentAssignable(right, left.typ)
			valid = valid && e.allowsOperation(right.typ, op)
		} else {
			valid = left.typ == right.typ
		}
	}
	typ := result.typ
	if typ == 0 {
		typ = e.types.defaultType(result.untyped)
	}
	valid = valid && e.allowsOperation(typ, op)
	if comparison {
		if len(result.shifted) != 0 {
			valid = valid && e.argumentAssignable(left, typ) && e.argumentAssignable(right, typ)
		}
		return genericArgument{untyped: genericUntypedBool, constant: genericConstantComparison(left.constant, right.constant, op)}, valid
	}
	integer := genericInteger(e.types.get(e.types.underlying(typ)).name)
	if (op == "/" || op == "%") && right.constant != nil && len(right.constant.real.numerator.words) == 0 && len(right.constant.imaginary.numerator.words) == 0 && e.allowsOperation(typ, "%") {
		valid = false
	}
	result.constant = genericConstantBinary(left.constant, right.constant, op, integer)
	return result, valid && (left.constant == nil || right.constant == nil || result.constant != nil)
}
