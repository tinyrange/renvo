package main

func appMain(args []string) int {
	x := 6
	if x != 6 {
		goto fail
	}
	print("PASS\n")
	return 0
fail:
	print("RENVO-0454 failure label\n")
	return 1
}
