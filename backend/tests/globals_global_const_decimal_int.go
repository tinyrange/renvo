package main

const renvo0676Value = 42

func appMain(args []string) int {
	if renvo0676Value != 42 {
		print("RENVO-0676 decimal const failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
