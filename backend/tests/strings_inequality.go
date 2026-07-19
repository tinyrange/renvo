package main

func appMain(args []string) int {
	a := "left"
	b := "right"
	if a == b {
		print("strings_18 neq\n")
		return 1
	}
	print("PASS\n")
	return 0
}
