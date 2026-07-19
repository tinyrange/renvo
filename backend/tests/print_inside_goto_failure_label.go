package main

func appMain(args []string) int {
	if 2+2 == 5 {
		goto fail
	}
	print("PASS\n")
	return 0
fail:
	print("RENVO-0721 goto diagnostic failed\n")
	return 1
	print("PASS\n")
	return 0
}
