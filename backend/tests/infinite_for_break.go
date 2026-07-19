package main

func appMain(args []string) int {
	n := 0
	for {
		n = 3
		break
	}
	if n != 3 {
		print("RENVO-0426 break failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
