package main

func appMain(args []string) int {
	x := 0
	for {
		x = 6
		goto exit
	}
exit:
	if x != 6 {
		print("RENVO-0443 goto shared exit failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
