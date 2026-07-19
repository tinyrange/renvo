package main

func appMain(args []string) int {
	s := []byte("abc")
	if !(len(s) == 3) {
		print("RENVO-0271 len_call_inside_comparison failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
