package main

func appMain(args []string) int {
	goto done
	print("RENVO-0817 skipped label failed\n")
	return 1
done:
	print("PASS\n")
	return 0
}
