package main

func appMain(args []string) int {
	goto pass
	print("RENVO-0472 avoided diagnostic failed\n")
	return 1
pass:
	print("PASS\n")
	return 0
}
