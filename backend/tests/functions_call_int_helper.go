package main

func renvo0477Value() int { return 21 }
func appMain(args []string) int {
	if renvo0477Value() != 21 {
		print("RENVO-0477 int helper failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
