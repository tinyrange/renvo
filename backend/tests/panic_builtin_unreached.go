package main

var panicArgumentCalls int

func panicMessage() string {
	panicArgumentCalls++
	return "requested panic"
}

func panicIfRequested(requested bool) {
	if requested {
		panic(panicMessage())
	}
}

func appMain(args []string) int {
	panicIfRequested(len(args) == 999)
	if panicArgumentCalls != 0 {
		return 1
	}
	print("PASS\n")
	return 0
}
