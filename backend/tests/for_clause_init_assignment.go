package main

func appMain(args []string) int {
	i := 9
	count := 0
	for i = 0; i < 3; i = i + 1 {
		count = count + 1
	}
	if i != 3 || count != 3 {
		print("RENVO-0404 init assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
