package main

func appMain(args []string) int {
	var s []int
	s = append(s, 1)
	s = append(s, 2)
	sum := 0
	i := 0
	for i < len(s) {
		sum += s[i]
		i = i + 1
	}
	if sum != 3 {
		print("RENVO-0826 no-range loop failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
