package main

func bool05(ok bool) bool { return ok }
func appMain(args []string) int {
	if !bool05(true) {
		print("booleans_05 param\n")
		return 1
	}
	print("PASS\n")
	return 0
}
