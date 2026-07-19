package main

func appMain(args []string) int {
	x := 0
	p := &x
	*p = 10
	goto check
check:
	if x != 10 {
		print("RENVO-0467 pointer goto failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
