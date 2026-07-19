package main

func appMain(args []string) int {
	x := 1
	i := 0
	for i < 4 {
		x = x | (1 << i)
		i = i + 1
	}
	if !(x == 15) {
		print("RENVO-0216 bitwise_on_variable_modified_in_loop failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
