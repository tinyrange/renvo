package main

func appMain(args []string) int {
	var xs []int
	if len(xs) != 0 {
		print("RENVO-0551 zero int slice length failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
