package main

func appMain(args []string) int {
	value := 5
	if value*value != 25 {
		print("program_shape_25 ignore\n")
		return 1
	}
	print("PASS\n")
	return 0
}
