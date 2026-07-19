package main

const renvo0807Value = 7

/* block comment between declarations */

func appMain(args []string) int {
	if renvo0807Value != 7 {
		print("RENVO-0807 block comment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
