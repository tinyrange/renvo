package main

func renvo0487Wide() int64 { return int64(123) }
func appMain(args []string) int {
	if renvo0487Wide() != int64(123) {
		print("RENVO-0487 int64 return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
