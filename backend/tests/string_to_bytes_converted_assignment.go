package main

func appMain(args []string) int {
	first := []byte("copy")
	second := first
	second[0] += 1
	if first[0] != 'd' || second[0] != 'd' {
		print("RENVO-0594 converted assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
