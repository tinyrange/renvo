package main

type CallbackArg = struct{ Value int }
type CallbackOther = struct{ Value int }

func aggregateCallbackApply(f func(CallbackArg) CallbackArg) int {
	return f(CallbackArg{42}).Value
}

func appMain() int {
	var call func(func(CallbackOther) CallbackOther) int = aggregateCallbackApply
	if call(func(v CallbackOther) CallbackOther { return v }) != 42 {
		return 1
	}
	print("PASS\n")
	return 0
}
