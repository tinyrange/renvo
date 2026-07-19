package main

func appMain(args []string) int {
	i := 5
	for i > 0 {
		i = i - 1
	}
	if i != 0 {
		print("RENVO-0377 count down failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
