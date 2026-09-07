package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

// Shifts share multiplicative precedence and associate left-to-right. A
// lower-precedence expression on the left and any binary expression on the
// right are outside the operands, unless enclosed in parentheses.
func shiftOperandBounds(file syntax.File, left, op, right int) (int, int) {
	if split := constantIndexOperator(file, left, op, 1); split >= 0 {
		left = split + 1
	}
	for {
		split := constantIndexOperator(file, op+1, right, 1)
		other := constantIndexOperator(file, op+1, right, 2)
		if split < 0 || other >= 0 && other < split {
			split = other
		}
		if split < 0 {
			break
		}
		right = split
	}
	return left, right
}

func invalidKnownShiftOperand(pkg load.Package, info PackageInfo, fileIndex int, scope CoreScope, bindings []scopedTypeBinding, start, end, before int) bool {
	file := pkg.Files[fileIndex].File
	start, end = stripOuterParens(file, start, end)
	value := numericBuiltinExprValue(pkg, info, fileIndex, scope, bindings, start, end, before, 0)
	if value.kind == "string" || value.kind == "bool" || value.kind == "other" || value.typed && (value.kind == "float" || value.kind == "complex") {
		return true
	}
	if unsafeAddFractionalDecimal(file, start, end) {
		return true
	}
	return false
}
