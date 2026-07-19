package main

func appMain(args []string) int {
	x := 10 + 40 - 8
	if x != 42 {
		print("arithmetic_08 chain\n")
		return 1
	}
	print("PASS\n")
	return 0
}
