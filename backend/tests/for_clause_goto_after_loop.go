package main

func appMain(args []string) int {
	sum := 0
	for i := 0; i < 3; i = i + 1 {
		sum = sum + i
	}
	goto done
	print("RENVO-0424 skipped\n")
	return 1
done:
	if sum != 3 {
		print("RENVO-0424 goto after for failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
