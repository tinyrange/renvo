package main

func appMain(args []string) int {
	b := byte(67)
	goto check
check:
	if int(b) != 67 {
		print("RENVO-0662 byte int conversion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
