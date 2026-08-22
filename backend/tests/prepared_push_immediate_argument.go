package main

func preparedImmediateArgumentProbe(input *byte, size uintptr, output *byte) int {
	if input == nil || output == nil || size != 3 {
		return 1
	}
	return 0
}

func appMain(args []string) int {
	var input [3]byte
	var output [32]byte
	if preparedImmediateArgumentProbe(&input[0], 3, &output[0]) != 0 {
		print("RENVO prepared_push_immediate_argument failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
