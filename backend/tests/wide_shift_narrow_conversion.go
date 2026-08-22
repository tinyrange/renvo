package main

func appMain(args []string) int {
	wide := uint64(17)<<32 | 5
	if uint32(wide>>32) != 17 || uint16(wide>>32) != 17 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
