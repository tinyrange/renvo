package main

type functionValueReceiverName int64

func (value *functionValueReceiverName) unused(delta int64) {}

func functionValueReceiverNameResult() int64 {
	value := func(first, second int64) int64 { return second }
	return value(1, 0)
}

func appMain(args []string) int {
	if functionValueReceiverNameResult() != 0 {
		print("FAIL: receiver name shadowed function value\n")
		return 1
	}
	print("PASS\n")
	return 0
}
