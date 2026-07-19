package main

func appMain(args []string) int {
	bs := []byte("orbit")
	if len(bs) == 5 {
		print("PASS\n")
		return 0
	}
	print("RENVO-0581 converted len failed\n")
	return 1
}
