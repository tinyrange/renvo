package main

func appMain(args []string) int {
	s := "x"
	if len(s) != 1 {
		print("strings_02 single\n")
		return 1
	}
	print("PASS\n")
	return 0
}
