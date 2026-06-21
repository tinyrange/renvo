package main

func appMain(args []string) int {
	a, b, c := 9, 2, 6
	if a-b-c != 1 {
		print("RTG-1037 three value short decl failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
