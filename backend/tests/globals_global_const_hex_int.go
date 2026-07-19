package main

const renvo0677Value = 0x2a

func appMain(args []string) int {
	if renvo0677Value != 42 {
		print("RENVO-0677 hex const failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
