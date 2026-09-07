package check

import "renvo.dev/internal/syntax"

// Validate literal operands without interpreting a shadowable identifier as a
// predeclared constant or type. Unknown expressions need ordinary type checking.
func invalidLiteralUnary(file syntax.File, op int, end int) bool {
	operator := file.Tokens[op]
	receive := tokenTextIs(&file, op, "<-")
	if operator.End-operator.Start != 1 && !receive {
		return false
	}
	ch := file.Src[int(operator.Start)]
	if !receive && ch != '!' && ch != '*' && ch != '&' && ch != '+' && ch != '-' && ch != '^' {
		return false
	}
	if op > 0 {
		previous := file.Tokens[op-1].KindLine & 255
		if previous == syntax.TokenIdent || previous == syntax.TokenNumber || previous == syntax.TokenString || previous == syntax.TokenChar ||
			tokCharIs(&file, op-1, ')') || tokCharIs(&file, op-1, ']') || tokCharIs(&file, op-1, '}') {
			return false // binary operator or channel send
		}
	}
	start := op + 1
	finish := start + 1
	for start < end && tokCharIs(&file, start, '(') {
		close := findTypeMatching(file, start, '(', ')')
		if close <= start || close > end {
			return false
		}
		if start != op+1 && close != end {
			return false
		}
		if start == op+1 {
			finish = close
		}
		start++
		end = close - 1
	}
	if start >= end {
		return false
	}
	// Parentheses must enclose only this literal, not an operation such as
	// !(1 == 2). A suffix can also change the type: +"x"[0] is valid.
	if finish > op+2 && start+1 != end {
		return false
	}
	if finish < len(file.Tokens) && (tokCharIs(&file, finish, '[') || tokCharIs(&file, finish, '(') || tokCharIs(&file, finish, '.')) {
		return false
	}
	kind := file.Tokens[start].KindLine & 255
	if kind != syntax.TokenNumber && kind != syntax.TokenString && kind != syntax.TokenChar {
		return false
	}
	if receive || ch == '!' || ch == '*' || ch == '&' {
		return true
	}
	if kind == syntax.TokenString {
		return true
	}
	return ch == '^' && unsafeAddFractionalDecimal(file, start, start+1)
}
