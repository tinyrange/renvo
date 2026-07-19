package main

type arith17Box struct{ value int }

func appMain(args []string) int {
	x := arith17Box{value: 6 * 7}
	if x.value != 42 {
		print("arithmetic_17 struct\n")
		return 1
	}
	print("PASS\n")
	return 0
}
