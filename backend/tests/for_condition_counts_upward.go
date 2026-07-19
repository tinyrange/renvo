package main

func appMain(args []string) int {
	i := 0
	for i < 5 {
		i = i + 1
	}
	if i != 5 {
		print("RENVO-0376 count up failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
