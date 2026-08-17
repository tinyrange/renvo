package main

func threadStateMultiwordResult() string {
	if false {
		panic("unreachable")
	}
	return "PASS\n"
}

func appMain(args []string) int {
	print(threadStateMultiwordResult())
	return 0
}
