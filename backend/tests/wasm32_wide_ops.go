package main

func appMain(args []string, env []string) int {
	value := uint64(1) << 40
	if value+3 != 1099511627779 {
		return 1
	}
	print("PASS\n")
	return 0
}
