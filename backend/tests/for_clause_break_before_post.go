package main

func appMain(args []string) int {
	i := 0
	for i = 0; i < 5; i = i + 1 {
		if i == 2 {
			break
		}
	}
	if i != 2 {
		print("RENVO-0415 break post failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
