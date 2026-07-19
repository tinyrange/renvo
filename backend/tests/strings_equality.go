package main

func appMain(args []string) int {
	a := "same"
	b := "same"
	if a != b {
		print("strings_17 eq\n")
		return 1
	}
	print("PASS\n")
	return 0
}
