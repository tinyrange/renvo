package main

type widePayloadOperation interface {
	apply(int64) int64
}

type widePayloadValue int64

func (value widePayloadValue) apply(input int64) int64 {
	return int64(value) * input
}

func appMain(args []string) int {
	const expected widePayloadValue = -0x100000001
	var operation widePayloadOperation = expected
	if operation.apply(3) != -0x300000003 {
		print("FAIL: direct wide interface method\n")
		return 1
	}
	apply := operation.apply
	if apply(5) != -0x500000005 {
		print("FAIL: wide interface method value\n")
		return 1
	}
	var boxed interface{} = expected
	asserted, ok := boxed.(widePayloadValue)
	if !ok || asserted != expected {
		print("FAIL: wide interface assertion\n")
		return 1
	}
	print("PASS\n")
	return 0
}
