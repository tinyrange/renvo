package check

func arrayLiteralConstant(context constantIndexContext, start, end int, scope CoreScope) wideConstant {
	context.scope = scope
	return wideConstantExpr(&context, start, end, 0)
}
