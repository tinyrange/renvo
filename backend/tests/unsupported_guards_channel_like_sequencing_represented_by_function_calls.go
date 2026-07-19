package main

func renvo0833StepA(x int) int {
	return x + 2
}

func renvo0833StepB(x int) int {
	return x * 3
}

func appMain(args []string) int {
	x := renvo0833StepA(4)
	x = renvo0833StepB(x)
	if x != 18 {
		print("RENVO-0833 sequencing failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
