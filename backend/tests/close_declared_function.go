package main

func close(value int32) int32 { return value + 7 }
func appMain(args []string) int {
	var descriptor int32 = 5
	if close(descriptor) != 12 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
