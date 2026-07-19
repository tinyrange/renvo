package main

func appMain(args []string) int {
	total := 0
	j := 1
	for j <= 4 {
		total = total + j*3 - 1
		j = j + 1
	}
	if !(total == 26) {
		print("RENVO-0173 multi_step_checksum_over_ints failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
