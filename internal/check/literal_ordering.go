package check

import "renvo.dev/internal/syntax"

// Both operands must be known untyped numeric expressions. A zero imaginary
// constant compared with a typed real value can acquire that real type, so an
// imaginary token alone is not sufficient evidence of invalid ordering.
func invalidLiteralOrdering(file syntax.File, op int, start int, end int) bool {
	if !tokenTextIs(&file, op, "<") && !tokenTextIs(&file, op, "<=") && !tokenTextIs(&file, op, ">") && !tokenTextIs(&file, op, ">=") {
		return false
	}
	left := literalOrderingBoundary(file, op-1, start, -1) + 1
	right := literalOrderingBoundary(file, op+1, end, 1)
	leftKind := literalNumericKind(file, left, op, 0)
	rightKind := literalNumericKind(file, op+1, right, 0)
	return leftKind != 0 && rightKind != 0 && (leftKind == 2 || rightKind == 2)
}

func literalOrderingBoundary(file syntax.File, at int, limit int, direction int) int {
	depth := 0
	for ; at >= 0 && at < len(file.Tokens) && (direction < 0 && at >= limit || direction > 0 && at < limit); at += direction {
		ch := file.Tokens[at].KindLine >> syntax.TokenOperatorCharShift & syntax.TokenOperatorCharMask
		if ch == int('(') || ch == int('[') || ch == int('{') {
			if direction < 0 && depth == 0 {
				return at
			}
			if direction > 0 && ch == int('{') && depth == 0 {
				return at
			}
			depth += direction
		} else if ch == int(')') || ch == int(']') || ch == int('}') {
			if direction > 0 && depth == 0 {
				return at
			}
			depth -= direction
		} else if depth == 0 {
			kind := file.Tokens[at].KindLine & 255
			binary := exprBinaryOperatorKind(file, at)
			if ch == int(',') || ch == int(';') || ch == int(':') || isAssignOp(file, at) || binary == exprBinaryCompare || binary == exprBinaryLogical ||
				kind == syntax.TokenReturn || kind == syntax.TokenIf || kind == syntax.TokenFor || kind == syntax.TokenCase || kind == syntax.TokenSwitch {
				return at
			}
		}
	}
	return at
}

// 0 is unknown, 1 is untyped real numeric, 2 is untyped complex numeric.
// This classifies types, not values: cancellation of imaginary components
// does not change the kind of an untyped complex expression.
func literalNumericKind(file syntax.File, start int, end int, depth int) int {
	if depth > 64 {
		return 0
	}
	start, end = stripOuterParens(file, start, end)
	if start < 0 || start >= end {
		return 0
	}
	if end-start == 1 {
		kind := file.Tokens[start].KindLine & 255
		if kind == syntax.TokenChar {
			return 1
		}
		if kind == syntax.TokenNumber {
			if file.Src[int(file.Tokens[start].End)-1] == 'i' {
				return 2
			}
			return 1
		}
		return 0
	}
	for precedence := 1; precedence <= 2; precedence++ {
		op := constantIndexOperator(file, start, end, precedence)
		if op < 0 {
			continue
		}
		if !tokenTextIs(&file, op, "+") && !tokenTextIs(&file, op, "-") && !tokenTextIs(&file, op, "*") && !tokenTextIs(&file, op, "/") {
			return 0
		}
		left := literalNumericKind(file, start, op, depth+1)
		right := literalNumericKind(file, op+1, end, depth+1)
		if left == 0 || right == 0 {
			return 0
		}
		if left == 2 || right == 2 {
			return 2
		}
		return 1
	}
	if tokCharIs(&file, start, '+') || tokCharIs(&file, start, '-') {
		return literalNumericKind(file, start+1, end, depth+1)
	}
	return 0
}
