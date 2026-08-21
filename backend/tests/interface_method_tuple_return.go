package main

type tupleReturnPairer interface {
	pair() (int, bool)
}

type tupleReturnValue struct{}

func (tupleReturnValue) pair() (int, bool) { return 42, true }

func tupleReturnForward(input tupleReturnPairer) (int, bool) {
	return input.pair()
}

func appMain() int {
	got, ok := tupleReturnForward(tupleReturnValue{})
	if got == 42 && ok {
		print("PASS\n")
		return 0
	}
	return 1
}
