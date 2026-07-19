package main

const renvo0696Limit = 4

func appMain(args []string) int {
	var values []int
	i := 0
	for i < renvo0696Limit {
		values = append(values, i+1)
		i = i + 1
	}
	if len(values) != 4 {
		print("RENVO-0696 global const loop length failed\n")
		return 1
	}
	if values[3] != 4 {
		print("RENVO-0696 global const loop value failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
