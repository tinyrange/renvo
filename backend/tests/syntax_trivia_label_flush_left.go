package main

func appMain(args []string) int {
	goto done
	print("RENVO-0816 skipped label failed\n")
	return 1
done:
	print("PASS\n")
	return 0
}
