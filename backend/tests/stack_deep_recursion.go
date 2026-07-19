package main

var compilerDefaultTarget int

const stackTestTargetWasiWasm32 = 7

func stackDeepRecurse(n int) int {
	if n == 0 {
		return 0
	}
	return stackDeepRecurse(n-1) + 1
}

func appMain() int {
	// WASI uses a virtual value/frame stack and the embedding engine owns the
	// native call-stack limit. This regression covers native frame emitters.
	if compilerDefaultTarget == stackTestTargetWasiWasm32 {
		print("PASS\n")
		return 0
	}
	if stackDeepRecurse(6000) != 6000 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
