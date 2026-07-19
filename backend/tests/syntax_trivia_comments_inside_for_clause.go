package main

func appMain(args []string) int {
	sum := 0
	for i := 0; /* init done */ i < 4; i = i + 1 {
		sum += i
	}
	if sum != 6 {
		print("RENVO-0810 for comment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
