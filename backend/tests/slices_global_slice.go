package main

var renvo0569Global []int

func appMain(args []string) int {
	renvo0569Global = append(renvo0569Global, 2)
	renvo0569Global = append(renvo0569Global, 3)
	renvo0569Global[0] += 5
	if renvo0569Global[0]+renvo0569Global[1] != 10 {
		print("RENVO-0569 global slice failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
