package main

func appMain(args []string) int {
	a, b := 2, 5
	if a+b != 7 {
		print("RTG-1036 two int short decl failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
