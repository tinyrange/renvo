package main

const renvo0394Limit = 4

func appMain(args []string) int {
	i := 0
	for i < renvo0394Limit {
		i = i + 1
	}
	if i != 4 {
		print("RENVO-0394 const condition failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
