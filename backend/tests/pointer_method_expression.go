package main

type methodExpressionValue int64

func (value *methodExpressionValue) add(delta int64) {
	*value += methodExpressionValue(delta)
}

func appMain(args []string) int {
	value := methodExpressionValue(7)
	add := (*methodExpressionValue).add
	add(&value, 5)
	if value != 12 {
		print("FAIL: pointer method expression\n")
		return 1
	}
	print("PASS\n")
	return 0
}
