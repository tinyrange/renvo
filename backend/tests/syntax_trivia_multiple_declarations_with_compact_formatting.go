package main

const renvo0823A = 2

var renvo0823B int = 5

func appMain(args []string) int {
	if renvo0823A+renvo0823B != 7 {
		print("RENVO-0823 compact declarations failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
