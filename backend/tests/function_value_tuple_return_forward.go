package main

type functionValueTupleCallback func(int) (int, int)

func functionValueTupleMake(value int) (int, int) {
	return value, value + 4
}

func functionValueTupleForward(callback functionValueTupleCallback, value int) (int, int) {
	return callback(value)
}

func appMain() int {
	left, right := functionValueTupleForward(functionValueTupleMake, 19)
	if left+right == 42 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
