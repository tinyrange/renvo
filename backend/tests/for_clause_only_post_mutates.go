package main

func appMain(args []string) int {
	count := 0
	for i := 0; i < 4; i = i + 1 {
		count = count + 1
	}
	if count != 4 {
		print("RENVO-0420 post only failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
