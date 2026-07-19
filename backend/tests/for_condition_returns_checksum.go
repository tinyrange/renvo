package main

func renvo0399Checksum() int {
	i := 1
	sum := 0
	for i <= 5 {
		sum = sum + i
		i = i + 1
	}
	return sum
}
func appMain(args []string) int {
	if renvo0399Checksum() != 15 {
		print("RENVO-0399 checksum loop failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
