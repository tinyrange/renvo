package main

func appMain(args []string) int {
	bits := 8
	if bits != 8 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
