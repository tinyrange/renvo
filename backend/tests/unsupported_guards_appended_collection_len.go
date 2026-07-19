package main

func appMain(args []string) int {
	var values []int
	values = append(values, 2)
	values = append(values, 4)
	values = append(values, 8)
	/* spaced like a compact literal without using arrays */
	if len(values) != 3 {
		print("RENVO-0849 appended collection len failed\n")
		return 1
	}
	if values[2] != 8 {
		print("RENVO-0849 appended collection value failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
