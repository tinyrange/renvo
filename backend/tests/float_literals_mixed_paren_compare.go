package main

func appMain(args []string) int {
	x := (2.0 + 3.0) * (4.0 - 1.0)
	if !(x == 15.0 && x > 14.5) {
		print("float_literals_25 mixed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
