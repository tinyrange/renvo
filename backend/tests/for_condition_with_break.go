package main

func appMain(args []string) int {
	i := 0
	for i < 10 {
		if i == 4 {
			break
		}
		i = i + 1
	}
	if i != 4 {
		print("RENVO-0386 condition break failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
