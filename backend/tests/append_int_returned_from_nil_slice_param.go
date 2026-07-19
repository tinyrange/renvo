package main

func appendInt(out []int, value int) []int {
	out = append(out, value)
	return out
}

func appMain(args []string, env []string) int {
	first := appendInt(nil, 11)
	second := appendInt(nil, 22)
	if len(first) == 1 && len(second) == 1 && first[0] == 11 && second[0] == 22 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
