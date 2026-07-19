package main

var shape15Bytes []byte

func appMain(args []string) int {
	shape15Bytes = append(shape15Bytes, byte(80))
	if len(shape15Bytes) != 1 || shape15Bytes[0] != 'P' {
		print("program_shape_15 slice\n")
		return 1
	}
	print("PASS\n")
	return 0
}
