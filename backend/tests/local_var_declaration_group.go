package main

func appMain(args []string) int {
	var (
		a int
		b = 7
	)
	a = b
	if a != 7 {
		return 1
	}
	print("PASS\n")
	return 0
}
