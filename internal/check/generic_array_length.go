package check

import "renvo.dev/internal/syntax"

type genericArrayLengthEvaluation struct{ pkg, file, token int }

// The ordinary wide evaluator is the fast path for literal and simple constant
// lengths. Generic expression checking also understands unevaluated layout
// operands inside constant len/cap and preserves the enclosing lexical scope.
func (e *genericEnvironment) genericArrayLength(scope genericTypeScope, start int, end int, fn syntax.FuncDecl, bindings []scopedTypeBinding) wideConstant {
	key := genericArrayLengthEvaluation{pkg: scope.pkg, file: scope.file, token: start}
	for _, active := range e.arrayEvaluations {
		if active == key {
			e.fail(scope, start, "cyclic array length expression")
			return wideConstant{}
		}
	}
	e.arrayEvaluations = append(e.arrayEvaluations, key)
	s := genericSpecializer{environment: e}
	ctx := genericExpressionContext{specializer: &s, scope: scope}
	if fn.BodyStart >= 0 {
		ctx = newGenericExpressionContext(&s, scope, fn, nil)
	}
	ctx.bindings, ctx.specialize = bindings, false
	value := ctx.expression(start, end, start)
	e.arrayEvaluations = e.arrayEvaluations[:len(e.arrayEvaluations)-1]
	if value.typ != 0 {
		v := e.types.get(e.types.underlying(value.typ))
		if v.kind != genericBasic || !genericInteger(v.name) {
			return wideConstant{}
		}
	}
	// Array type checking uses a separate expression context. Retain its typed
	// layout operands when a local variable prevents folding the whole bound.
	if genericConstantInteger(value.constant).ok {
		for _, change := range ctx.changes {
			e.recordArrayLengthChange(scope, change)
		}
	}
	return genericConstantInteger(value.constant)
}
